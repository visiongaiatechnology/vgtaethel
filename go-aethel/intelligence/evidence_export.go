package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"go-aethel/security"
)

const evidenceExportFormat = "vgt-aethel-evidence-export-v1"

type evidenceExportSigner struct {
	private ed25519.PrivateKey
	public  ed25519.PublicKey
}

type EvidenceExportItem struct {
	EvidenceID       string `json:"evidence_id"`
	RawSHA256        string `json:"raw_sha256"`
	NormalizedSHA256 string `json:"normalized_sha256,omitempty"`
	SnapshotID       string `json:"snapshot_id,omitempty"`
	CaptureScope     string `json:"capture_scope"`
	ValidationStatus string `json:"validation_status"`
	ArchivePath      string `json:"archive_path,omitempty"`
}

type EvidenceExportManifest struct {
	Format            string               `json:"format"`
	CaseID            string               `json:"case_id"`
	CreatedAt         time.Time            `json:"created_at"`
	StoreRevision     uint64               `json:"store_revision"`
	CustodyChainValid bool                 `json:"custody_chain_valid"`
	GlobalCustodyHead string               `json:"global_custody_head,omitempty"`
	Evidence          []EvidenceExportItem `json:"evidence"`
	CaseSHA256        string               `json:"case_sha256"`
	ClaimsSHA256      string               `json:"claims_sha256"`
	CustodySHA256     string               `json:"custody_sha256"`
	PassagesSHA256    string               `json:"passages_sha256"`
	ReportSHA256      string               `json:"report_sha256"`
	SigningAlgorithm  string               `json:"signing_algorithm"`
	SigningPublicKey  string               `json:"signing_public_key"`
	ManifestSignature string               `json:"manifest_signature"`
}

func newEvidenceExportSigner(path string) (*evidenceExportSigner, error) {
	payload, err := security.ReadAuthorityFile(path)
	if err == nil {
		if len(payload) != ed25519.PrivateKeySize {
			return nil, errors.New("evidence signing key length is invalid")
		}
		private := append(ed25519.PrivateKey(nil), payload...)
		public := append(ed25519.PublicKey(nil), private.Public().(ed25519.PublicKey)...)
		return &evidenceExportSigner{private: private, public: public}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := security.WriteSealedFile(path, private); err != nil {
		return nil, err
	}
	return &evidenceExportSigner{private: private, public: public}, nil
}

func (s *Store) ExportCaseEvidence(caseID string) ([]byte, EvidenceExportManifest, error) {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" || s == nil {
		return nil, EvidenceExportManifest{}, errors.New("case identifier is required")
	}
	if s.setupErr != nil || s.vault == nil || s.signer == nil {
		return nil, EvidenceExportManifest{}, errors.New("forensic export services unavailable")
	}

	s.mu.RLock()
	state := s.state
	s.mu.RUnlock()
	var caseRecord Case
	found := false
	for _, candidate := range state.Cases {
		if candidate.ID == caseID {
			caseRecord = candidate
			found = true
			break
		}
	}
	if !found {
		return nil, EvidenceExportManifest{}, errors.New("case not found")
	}
	claims := make([]Claim, 0)
	for _, claim := range state.Claims {
		if claim.CaseID == caseID {
			claims = append(claims, claim)
		}
	}
	sort.Slice(claims, func(i, j int) bool { return claims[i].ID < claims[j].ID })
	passageIDs := make(map[string]bool)
	for _, claim := range claims {
		for _, id := range claim.PassageIDs {
			passageIDs[id] = true
		}
	}
	passages := make([]Passage, 0, len(passageIDs))
	for _, passage := range state.Passages {
		if passageIDs[passage.ID] {
			passages = append(passages, passage)
		}
	}
	sort.Slice(passages, func(i, j int) bool { return passages[i].ID < passages[j].ID })
	custody := custodyForEvidence(state.CustodyEvents, caseRecord.Evidence)
	caseJSON, err := json.MarshalIndent(caseRecord, "", "  ")
	if err != nil {
		return nil, EvidenceExportManifest{}, err
	}
	claimsJSON, err := json.MarshalIndent(claims, "", "  ")
	if err != nil {
		return nil, EvidenceExportManifest{}, err
	}
	custodyJSON, err := json.MarshalIndent(custody, "", "  ")
	if err != nil {
		return nil, EvidenceExportManifest{}, err
	}
	passagesJSON, err := json.MarshalIndent(passages, "", "  ")
	if err != nil {
		return nil, EvidenceExportManifest{}, err
	}
	createdAt := time.Now().UTC()
	report := buildCaseEvidenceReport(caseRecord, claims, passages, createdAt)

	manifest := EvidenceExportManifest{
		Format: evidenceExportFormat, CaseID: caseID, CreatedAt: createdAt, StoreRevision: state.Revision,
		CustodyChainValid: VerifyCustodyChain(state.CustodyEvents), CaseSHA256: bytesSHA256(caseJSON),
		ClaimsSHA256: bytesSHA256(claimsJSON), CustodySHA256: bytesSHA256(custodyJSON),
		PassagesSHA256: bytesSHA256(passagesJSON), ReportSHA256: bytesSHA256(report),
		SigningAlgorithm: "ed25519", SigningPublicKey: base64.StdEncoding.EncodeToString(s.signer.public),
	}
	if len(state.CustodyEvents) > 0 {
		manifest.GlobalCustodyHead = state.CustodyEvents[len(state.CustodyEvents)-1].EventHash
	}

	type archiveObject struct {
		path string
		raw  []byte
	}
	objects := make([]archiveObject, 0, len(caseRecord.Evidence))
	for _, evidence := range caseRecord.Evidence {
		item := EvidenceExportItem{
			EvidenceID: evidence.ID, RawSHA256: firstNonEmpty(evidence.RawSHA256, evidence.SHA256),
			NormalizedSHA256: evidence.NormalizedSHA256, SnapshotID: evidence.SnapshotID,
			CaptureScope: firstNonEmpty(evidence.CaptureScope, "excerpt"), ValidationStatus: evidence.ValidationStatus,
		}
		if evidence.SnapshotID != "" {
			raw, record, readErr := s.vault.Read(evidence.SnapshotID)
			if readErr != nil {
				return nil, EvidenceExportManifest{}, fmt.Errorf("read evidence snapshot %s: %w", evidence.ID, readErr)
			}
			if record.RawSHA256 != evidence.SnapshotID {
				return nil, EvidenceExportManifest{}, errors.New("snapshot manifest digest mismatch")
			}
			item.RawSHA256 = record.RawSHA256
			item.ArchivePath = "snapshots/" + evidence.SnapshotID + ".bin"
			item.CaptureScope = "original"
			objects = append(objects, archiveObject{path: item.ArchivePath, raw: raw})
		}
		manifest.Evidence = append(manifest.Evidence, item)
	}
	sort.Slice(manifest.Evidence, func(i, j int) bool { return manifest.Evidence[i].EvidenceID < manifest.Evidence[j].EvidenceID })

	unsignedManifest, err := marshalUnsignedEvidenceManifest(manifest)
	if err != nil {
		return nil, EvidenceExportManifest{}, err
	}
	manifest.ManifestSignature = base64.StdEncoding.EncodeToString(ed25519.Sign(s.signer.private, unsignedManifest))
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, EvidenceExportManifest{}, err
	}

	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, file := range []archiveObject{{path: "manifest.json", raw: manifestJSON}, {path: "report.md", raw: report}, {path: "case.json", raw: caseJSON}, {path: "claims.json", raw: claimsJSON}, {path: "passages.json", raw: passagesJSON}, {path: "custody.json", raw: custodyJSON}} {
		if err := writeEvidenceArchiveFile(archive, file.path, file.raw); err != nil {
			_ = archive.Close()
			return nil, EvidenceExportManifest{}, err
		}
	}
	for _, object := range objects {
		if err := writeEvidenceArchiveFile(archive, object.path, object.raw); err != nil {
			_ = archive.Close()
			return nil, EvidenceExportManifest{}, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, EvidenceExportManifest{}, err
	}
	return output.Bytes(), manifest, nil
}

func buildCaseEvidenceReport(caseRecord Case, claims []Claim, passages []Passage, createdAt time.Time) []byte {
	var report strings.Builder
	report.WriteString("# " + markdownText(caseRecord.Title) + "\n\n")
	report.WriteString("- Case ID: `" + caseRecord.ID + "`\n- Classification: " + markdownText(caseRecord.Classification) + "\n- Generated: " + createdAt.Format(time.RFC3339Nano) + "\n\n")
	report.WriteString("## Analytic claims\n\n")
	if len(claims) == 0 {
		report.WriteString("No reviewed claims are attached to this case.\n\n")
	}
	for index, claim := range claims {
		report.WriteString(fmt.Sprintf("### C%d — %s\n\n", index+1, markdownText(claim.Statement)))
		report.WriteString(fmt.Sprintf("Claim ID: `%s` · Source: `%s` (%s) · Confidence: %d%% · Status: %s\n\n", claim.ID, claim.AssertingSourceID, markdownText(claim.SourceNature), claim.Confidence, markdownText(claim.Status)))
		report.WriteString("Passages: " + markdownReferences(claim.PassageIDs) + "  \nSupporting evidence: " + markdownReferences(claim.SupportingEvidenceIDs) + "  \nContradicting evidence: " + markdownReferences(claim.ContradictingEvidenceIDs) + "\n\n")
		if claim.CalibrationBasis != "" {
			report.WriteString("Calibration: " + markdownText(claim.CalibrationBasis) + "\n\n")
		}
	}
	report.WriteString("## Referenced passages\n\n")
	for _, passage := range passages {
		report.WriteString("- `" + passage.ID + "` → `" + passage.DocumentID + "` offsets " + fmt.Sprintf("%d–%d", passage.StartOffset, passage.EndOffset) + ": " + markdownText(passage.Text) + "\n")
	}
	report.WriteString("\n## Evidence catalog\n\n")
	for _, evidence := range caseRecord.Evidence {
		report.WriteString("- `" + evidence.ID + "` · SHA-256 `" + firstNonEmpty(evidence.RawSHA256, evidence.SHA256) + "` · " + markdownText(firstNonEmpty(evidence.CaptureScope, "excerpt")) + " · " + markdownText(evidence.ValidationStatus) + "\n")
	}
	return []byte(report.String())
}

func markdownReferences(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, "`"+strings.ReplaceAll(value, "`", "")+"`")
	}
	return strings.Join(result, ", ")
}

func markdownText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	replacer := strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(value)
}

func VerifyEvidenceExportManifest(manifest EvidenceExportManifest) error {
	if manifest.Format != evidenceExportFormat || manifest.SigningAlgorithm != "ed25519" {
		return errors.New("unsupported evidence export manifest")
	}
	public, err := base64.StdEncoding.DecodeString(manifest.SigningPublicKey)
	if err != nil || len(public) != ed25519.PublicKeySize {
		return errors.New("evidence export public key is invalid")
	}
	signature, err := base64.StdEncoding.DecodeString(manifest.ManifestSignature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return errors.New("evidence export signature is invalid")
	}
	unsigned, err := marshalUnsignedEvidenceManifest(manifest)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(public), unsigned, signature) {
		return errors.New("evidence export signature verification failed")
	}
	return nil
}

func VerifyEvidenceExportArchive(payload []byte) (EvidenceExportManifest, error) {
	if len(payload) == 0 || len(payload) > 512<<20 {
		return EvidenceExportManifest{}, errors.New("evidence archive size boundary violation")
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return EvidenceExportManifest{}, errors.New("evidence archive is invalid")
	}
	files := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		if file.UncompressedSize64 > 128<<20 || files[file.Name] != nil {
			return EvidenceExportManifest{}, errors.New("evidence archive contains invalid entries")
		}
		stream, openErr := file.Open()
		if openErr != nil {
			return EvidenceExportManifest{}, errors.New("evidence archive entry cannot be opened")
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, (128<<20)+1))
		_ = stream.Close()
		if readErr != nil || len(content) > 128<<20 {
			return EvidenceExportManifest{}, errors.New("evidence archive entry violates size boundary")
		}
		files[file.Name] = content
	}
	var manifest EvidenceExportManifest
	if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
		return EvidenceExportManifest{}, errors.New("evidence manifest is invalid")
	}
	if err := VerifyEvidenceExportManifest(manifest); err != nil {
		return EvidenceExportManifest{}, err
	}
	expected := map[string]string{"case.json": manifest.CaseSHA256, "claims.json": manifest.ClaimsSHA256, "custody.json": manifest.CustodySHA256, "passages.json": manifest.PassagesSHA256, "report.md": manifest.ReportSHA256}
	allowed := map[string]bool{"manifest.json": true}
	for name, digest := range expected {
		allowed[name] = true
		if digest == "" || bytesSHA256(files[name]) != digest {
			return EvidenceExportManifest{}, errors.New("evidence archive content digest mismatch")
		}
	}
	for _, item := range manifest.Evidence {
		if item.ArchivePath == "" {
			continue
		}
		allowed[item.ArchivePath] = true
		if bytesSHA256(files[item.ArchivePath]) != item.RawSHA256 {
			return EvidenceExportManifest{}, errors.New("evidence snapshot digest mismatch")
		}
	}
	for name := range files {
		if !allowed[name] {
			return EvidenceExportManifest{}, errors.New("evidence archive contains an unmanifested entry")
		}
	}
	return manifest, nil
}

func marshalUnsignedEvidenceManifest(manifest EvidenceExportManifest) ([]byte, error) {
	manifest.ManifestSignature = ""
	return json.Marshal(manifest)
}

func custodyForEvidence(events []CustodyEvent, evidence []Evidence) []CustodyEvent {
	ids := make(map[string]bool, len(evidence))
	for _, item := range evidence {
		ids[item.ID] = true
	}
	result := make([]CustodyEvent, 0)
	for _, event := range events {
		if ids[event.EvidenceID] {
			result = append(result, event)
		}
	}
	return result
}

func writeEvidenceArchiveFile(archive *zip.Writer, name string, payload []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o600)
	header.SetModTime(time.Unix(0, 0).UTC())
	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = writer.Write(payload)
	return err
}

func bytesSHA256(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func errorsJoin(left, right error) error {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	return errors.Join(left, right)
}
