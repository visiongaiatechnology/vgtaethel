package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const (
	maximumImportBytes   = 8 << 20
	maximumExtractedText = 2 << 20
	maximumImportRecords = 1000
	maximumRecordRunes   = 32 << 10
)

type ImportedDocument struct {
	ID               string            `json:"id"`
	SourceID         string            `json:"source_id"`
	Filename         string            `json:"filename"`
	Format           string            `json:"format"`
	RawSHA256        string            `json:"raw_sha256"`
	RawBytes         int               `json:"raw_bytes"`
	ObservationIDs   []string          `json:"observation_ids"`
	InstructionFlags []string          `json:"instruction_flags,omitempty"`
	Quarantined      bool              `json:"quarantined"`
	ImportedAt       time.Time         `json:"imported_at"`
	ParserVersion    string            `json:"parser_version"`
	SnapshotID       string            `json:"snapshot_id"`
	CaptureScope     string            `json:"capture_scope"`
	OriginalURL      string            `json:"original_url,omitempty"`
	FinalURL         string            `json:"final_url,omitempty"`
	MIMEType         string            `json:"mime_type,omitempty"`
	ResponseHeaders  map[string]string `json:"response_headers,omitempty"`
	FetchedAt        time.Time         `json:"fetched_at,omitempty"`
}

func (s *Store) ImportDocument(raw []byte, format, sourceID, filename, domain string) (ImportedDocument, error) {
	return s.importDocument(raw, format, sourceID, filename, domain, AcquisitionMetadata{})
}

type AcquisitionMetadata struct {
	OriginalURL     string
	FinalURL        string
	MIMEType        string
	ResponseHeaders map[string]string
	FetchedAt       time.Time
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func (s *Store) ImportAcquiredDocument(raw []byte, format, sourceID, filename, domain string, metadata AcquisitionMetadata) (ImportedDocument, error) {
	if metadata.OriginalURL == "" || metadata.FinalURL == "" || metadata.FetchedAt.IsZero() {
		return ImportedDocument{}, errors.New("acquisition provenance is incomplete")
	}
	return s.importDocument(raw, format, sourceID, filename, domain, metadata)
}

func (s *Store) importDocument(raw []byte, format, sourceID, filename, domain string, metadata AcquisitionMetadata) (ImportedDocument, error) {
	if len(raw) == 0 || len(raw) > maximumImportBytes {
		return ImportedDocument{}, errors.New("document size boundary violation")
	}
	format, sourceID, filename, domain = strings.ToLower(strings.TrimSpace(format)), strings.TrimSpace(sourceID), strings.TrimSpace(filename), strings.TrimSpace(domain)
	if (format != "json" && format != "html" && format != "text") || sourceID == "" || len([]rune(sourceID)) > 160 || filename == "" || len([]rune(filename)) > 240 || strings.ContainsAny(filename, "/\\\x00") || len([]rune(domain)) > 80 {
		return ImportedDocument{}, errors.New("document metadata is invalid")
	}
	records, err := extractImportRecords(raw, format)
	if err != nil {
		return ImportedDocument{}, err
	}
	digest := sha256.Sum256(raw)
	rawHash := hex.EncodeToString(digest[:])
	importedAt := time.Now().UTC()
	if s.vault == nil {
		return ImportedDocument{}, errors.New("evidence vault unavailable")
	}
	localURL := "local-import://" + "import-" + rawHash[:24] + "/" + filename
	originalURL, finalURL, mimeType, fetchedAt := metadata.OriginalURL, metadata.FinalURL, metadata.MIMEType, metadata.FetchedAt
	headers := cloneStringMap(metadata.ResponseHeaders)
	if originalURL == "" {
		originalURL, finalURL, fetchedAt = localURL, localURL, importedAt
	}
	if finalURL == "" {
		finalURL = originalURL
	}
	if mimeType == "" {
		mimeType = importMIME(format)
	}
	if fetchedAt.IsZero() {
		fetchedAt = importedAt
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["content-length"] = strconv.Itoa(len(raw))
	snapshot, err := s.vault.Capture(sourceID, originalURL, finalURL, mimeType, headers, fetchedAt, raw)
	if err != nil {
		return ImportedDocument{}, fmt.Errorf("capture immutable import snapshot: %w", err)
	}
	record := ImportedDocument{
		ID: "import-" + rawHash[:24], SourceID: sourceID, Filename: filename, Format: format,
		RawSHA256: rawHash, RawBytes: len(raw), ImportedAt: importedAt, ParserVersion: "document-import-v1",
		SnapshotID: snapshot.ID, CaptureScope: "original",
		OriginalURL: originalURL, FinalURL: finalURL, MIMEType: mimeType, ResponseHeaders: headers, FetchedAt: fetchedAt,
	}
	combinedFlags := make(map[string]bool)
	observations := make([]Observation, 0, len(records))
	for index, text := range records {
		id := fmt.Sprintf("%s-%04d", record.ID, index+1)
		flags := DetectInstructionSignals(text)
		for _, flag := range flags {
			combinedFlags[flag] = true
		}
		record.ObservationIDs = append(record.ObservationIDs, id)
		observations = append(observations, Observation{
			ID: id, SourceID: sourceID, RawText: text, ObservedAt: importedAt, FetchedAt: importedAt,
			Domain: domain, MIMEType: importMIME(format), ParserVersion: record.ParserVersion,
			OriginalURL: originalURL, FinalURL: finalURL, SnapshotID: snapshot.ID,
			ResponseHeaders: cloneStringMap(headers),
		})
	}
	for flag := range combinedFlags {
		record.InstructionFlags = append(record.InstructionFlags, flag)
	}
	sortStrings(record.InstructionFlags)
	record.Quarantined = len(record.InstructionFlags) > 0
	s.IngestObservationsBatch(observations)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.state.ImportedDocuments {
		if existing.ID == record.ID {
			return existing, nil
		}
	}
	s.state.ImportedDocuments = append(s.state.ImportedDocuments, record)
	s.state.Audits = append(s.state.Audits, AuditEvent{At: importedAt, Action: "document.imported", Actor: "operator", Detail: record.ID})
	if err := s.save(); err != nil {
		return ImportedDocument{}, err
	}
	return record, nil
}

func extractImportRecords(raw []byte, format string) ([]string, error) {
	var records []string
	switch format {
	case "text":
		if !utf8.Valid(raw) {
			return nil, errors.New("text import must be valid UTF-8")
		}
		records = chunkImportText(string(raw))
	case "html":
		document, err := html.Parse(io.LimitReader(bytes.NewReader(raw), maximumImportBytes+1))
		if err != nil {
			return nil, errors.New("HTML document cannot be parsed")
		}
		var builder strings.Builder
		var walk func(*html.Node, bool)
		walk = func(node *html.Node, suppressed bool) {
			if node.Type == html.ElementNode && (node.Data == "script" || node.Data == "style" || node.Data == "noscript") {
				suppressed = true
			}
			if node.Type == html.TextNode && !suppressed {
				builder.WriteString(node.Data)
				builder.WriteByte(' ')
			}
			if builder.Len() <= maximumExtractedText {
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					walk(child, suppressed)
				}
			}
		}
		walk(document, false)
		if builder.Len() > maximumExtractedText {
			return nil, errors.New("HTML extracted text boundary violation")
		}
		records = chunkImportText(builder.String())
	case "json":
		decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(raw), maximumImportBytes+1))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, errors.New("JSON export cannot be parsed")
		}
		if decoder.Decode(&struct{}{}) != io.EOF {
			return nil, errors.New("JSON export contains trailing data")
		}
		records = extractJSONRecords(value)
	}
	if len(records) == 0 {
		return nil, errors.New("document contains no extractable text")
	}
	if len(records) > maximumImportRecords {
		return nil, errors.New("document record boundary violation")
	}
	total := 0
	for _, record := range records {
		total += len(record)
	}
	if total > maximumExtractedText {
		return nil, errors.New("document extracted text boundary violation")
	}
	return records, nil
}

func extractJSONRecords(value any) []string {
	items, ok := value.([]any)
	if !ok {
		items = []any{value}
	}
	records := make([]string, 0, len(items))
	for _, item := range items {
		parts := make([]string, 0, 8)
		collectJSONText(item, 0, &parts)
		if text := normalizeImportText(strings.Join(parts, " ")); text != "" {
			records = append(records, truncateRunes(text, maximumRecordRunes))
		}
		if len(records) >= maximumImportRecords+1 {
			break
		}
	}
	return records
}

func collectJSONText(value any, depth int, output *[]string) {
	if depth > 12 || len(*output) > 256 {
		return
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) != "" {
			*output = append(*output, typed)
		}
	case []any:
		for _, item := range typed {
			collectJSONText(item, depth+1, output)
		}
	case map[string]any:
		for _, key := range []string{"title", "name", "text", "message", "content", "caption", "description", "body", "url", "timestamp", "created_at"} {
			if item, found := typed[key]; found {
				collectJSONText(item, depth+1, output)
			}
		}
	}
}

func chunkImportText(text string) []string {
	text = normalizeImportText(text)
	if text == "" {
		return nil
	}
	runes := []rune(text)
	result := make([]string, 0, (len(runes)/maximumRecordRunes)+1)
	for len(runes) > 0 && len(result) <= maximumImportRecords {
		end := maximumRecordRunes
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[:end]))
		runes = runes[end:]
	}
	return result
}

func normalizeImportText(text string) string {
	return strings.Join(strings.Fields(strings.ToValidUTF8(text, "�")), " ")
}
func truncateRunes(text string, maximum int) string {
	runes := []rune(text)
	if len(runes) > maximum {
		return string(runes[:maximum])
	}
	return text
}
func importMIME(format string) string {
	return map[string]string{"json": "application/json", "html": "text/html", "text": "text/plain"}[format]
}
func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
