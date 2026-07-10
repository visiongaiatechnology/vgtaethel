package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	cartographyDefaultOutput = "code_cartography.md"
	cartographyMaxFiles      = 500
	cartographyMaxFileBytes  = 512 << 10
)

var (
	cartographyImportPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*import\s+(?:.+?\s+from\s+)?["']([^"']+)["']`),
		regexp.MustCompile(`(?m)^\s*(?:from|import)\s+([A-Za-z0-9_./-]+)`),
		regexp.MustCompile(`(?m)^\s*use\s+([A-Za-z0-9_:]+)`),
		regexp.MustCompile(`(?m)^\s*#include\s*[<"]([^>"]+)[>"]`),
	}
	cartographyPackagePattern = regexp.MustCompile(`(?m)^\s*(?:package|namespace|module)\s+([A-Za-z0-9_./:-]+)`)
	cartographySymbolPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:async\s+)?(?:func|function|class|interface|type|struct|enum)\s+([A-Za-z_][A-Za-z0-9_]*)`),
		regexp.MustCompile(`(?m)^\s*(?:pub\s+)?(?:fn|struct|enum|trait)\s+([A-Za-z_][A-Za-z0-9_]*)`),
	}
)

type CodeCartographySkill struct{}

type CodeCartographyArgs struct {
	Path       string `json:"path"`
	OutputPath string `json:"output_path,omitempty"`
	MaxFiles   int    `json:"max_files,omitempty"`
}

type cartographyFile struct {
	Path     string
	Language string
	Lines    int
	Bytes    int64
	Package  string
	Imports  []string
	Symbols  []string
	Role     string
}

func (s *CodeCartographySkill) Name() string { return "code_cartography" }
func (s *CodeCartographySkill) Description() string {
	return "Erstellt eine lokale Markdown-Codekartografie: rekursive Dateikarte, Module, Imports, Symbole und Architektur-Verbindungen eines freigegebenen Code-Ordners."
}
func (s *CodeCartographySkill) RiskLevel() RiskLevel { return RiskModerate }

func (s *CodeCartographySkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":        map[string]interface{}{"type": "string", "description": "Code-Ordner im Workspace oder ein lesend eingehängter absoluter Ordner"},
			"output_path": map[string]interface{}{"type": "string", "description": "Optionaler Markdown-Zielpfad im VGT Workspace; Standard: code_cartography.md"},
			"max_files":   map[string]interface{}{"type": "integer", "minimum": 1, "maximum": cartographyMaxFiles},
		},
		"required": []string{"path"},
	}
}

func (s *CodeCartographySkill) Execute(args json.RawMessage) (string, error) {
	var input CodeCartographyArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	root, err := validatePath(input.Path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("code cartography requires a directory")
	}
	output := input.OutputPath
	if strings.TrimSpace(output) == "" {
		output = cartographyDefaultOutput
	}
	safeOutput, err := validateWritePath(output)
	if err != nil {
		return "", err
	}
	maxFiles := input.MaxFiles
	if maxFiles <= 0 || maxFiles > cartographyMaxFiles {
		maxFiles = cartographyMaxFiles
	}

	files, skipped, err := collectCartographyFiles(root, maxFiles)
	if err != nil {
		return "", err
	}
	report := renderCartographyReport(root, files, skipped, maxFiles)
	snapshotID, err := defaultFileSnapshots.Create(safeOutput)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(safeOutput), 0700); err != nil {
		return "", err
	}
	if err := os.WriteFile(safeOutput, []byte(report), 0600); err != nil {
		return "", err
	}
	recordFileChange(safeOutput, len(strings.Split(report, "\n")), 0)
	LogKernelActivity("CODE_CARTOGRAPHY", root, "SUCCESS")
	return fmt.Sprintf("Codekartografie erstellt: %s. Analysierte Dateien: %d, ausgelassen: %d. Snapshot: %s", safeOutput, len(files), skipped, snapshotID), nil
}

func collectCartographyFiles(root string, maxFiles int) ([]cartographyFile, int, error) {
	files := make([]cartographyFile, 0, maxFiles)
	skipped := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && isCartographyIgnoredDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(files) >= maxFiles {
			skipped++
			return nil
		}
		language := cartographyLanguage(path)
		if language == "" || isCartographySensitiveFile(entry.Name()) {
			skipped++
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() > cartographyMaxFileBytes {
			skipped++
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || !isLikelyText(data) {
			skipped++
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, analyzeCartographyFile(filepath.ToSlash(rel), language, data, info.Size()))
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, skipped, err
}

func analyzeCartographyFile(path, language string, data []byte, size int64) cartographyFile {
	content := string(data)
	file := cartographyFile{Path: path, Language: language, Bytes: size, Lines: lineCount(content), Role: inferCartographyRole(path)}
	if match := cartographyPackagePattern.FindStringSubmatch(content); len(match) == 2 {
		file.Package = match[1]
	}
	file.Imports = findCartographyMatches(content, cartographyImportPatterns, 24)
	file.Symbols = findCartographyMatches(content, cartographySymbolPatterns, 24)
	return file
}

func renderCartographyReport(root string, files []cartographyFile, skipped, maxFiles int) string {
	var out strings.Builder
	out.WriteString("# Code Kartografie\n\n")
	out.WriteString(fmt.Sprintf("- **Quelle:** `%s`\n- **Erstellt:** %s\n- **Analysierte Dateien:** %d\n- **Ausgelassen:** %d (Binär-, Secret-, Build- oder Größenlimit)\n- **Grenze:** %d Dateien\n\n", filepath.ToSlash(root), time.Now().UTC().Format(time.RFC3339), len(files), skipped, maxFiles))
	out.WriteString("## Architekturüberblick\n\n")
	roles := make(map[string]int)
	for _, file := range files {
		roles[file.Role]++
	}
	roleNames := make([]string, 0, len(roles))
	for role := range roles {
		roleNames = append(roleNames, role)
	}
	sort.Strings(roleNames)
	for _, role := range roleNames {
		out.WriteString(fmt.Sprintf("- **%s:** %d Dateien\n", role, roles[role]))
	}
	out.WriteString("\n## Modul- und Importkanten\n\n")
	edges := make([]string, 0)
	for _, file := range files {
		for _, dependency := range file.Imports {
			edges = append(edges, fmt.Sprintf("- `%s` → `%s`\n", file.Path, dependency))
		}
	}
	if len(edges) == 0 {
		out.WriteString("- Keine statischen Imports erkannt.\n")
	} else {
		for _, edge := range edges {
			out.WriteString(edge)
		}
	}
	out.WriteString("\n## Datei für Datei\n\n")
	for _, file := range files {
		out.WriteString(fmt.Sprintf("### `%s`\n\n", file.Path))
		out.WriteString(fmt.Sprintf("- **Rolle:** %s\n- **Sprache:** %s\n- **Größe:** %d Bytes · %d Zeilen\n", file.Role, file.Language, file.Bytes, file.Lines))
		if file.Package != "" {
			out.WriteString(fmt.Sprintf("- **Modul/Package:** `%s`\n", file.Package))
		}
		writeCartographyList(&out, "Imports", file.Imports)
		writeCartographyList(&out, "Erkannte Symbole", file.Symbols)
		out.WriteString("\n")
	}
	out.WriteString("## Nächster Architektur-Schritt\n\nLies bei Änderungen zuerst die betroffene Datei und ihre eingehenden/ausgehenden Importkanten. Die Kartografie ist eine statische Orientierung; semantische Entscheidungen werden danach im Chat begründet und verifiziert.\n")
	return out.String()
}

func writeCartographyList(out *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		return
	}
	out.WriteString(fmt.Sprintf("- **%s:** %s\n", label, inlineCodeList(values)))
}

func inlineCodeList(values []string) string {
	items := make([]string, len(values))
	for i, value := range values {
		items[i] = "`" + strings.ReplaceAll(value, "`", "'") + "`"
	}
	return strings.Join(items, ", ")
}

func findCartographyMatches(content string, patterns []*regexp.Regexp, limit int) []string {
	seen := make(map[string]struct{})
	values := make([]string, 0, limit)
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatch(content, -1) {
			if len(match) < 2 || match[1] == "" {
				continue
			}
			value := strings.TrimSpace(match[1])
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			values = append(values, value)
			if len(values) >= limit {
				return values
			}
		}
	}
	return values
}

func lineCount(content string) int {
	if content == "" {
		return 0
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	count := 0
	for scanner.Scan() {
		count++
	}
	return count
}

func isLikelyText(data []byte) bool { return !strings.ContainsRune(string(data), 0) }

func cartographyLanguage(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "Go"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".js", ".jsx", ".mjs":
		return "JavaScript"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".java":
		return "Java"
	case ".cs":
		return "C#"
	case ".c", ".h":
		return "C"
	case ".cpp", ".cc", ".hpp":
		return "C++"
	case ".php":
		return "PHP"
	case ".html", ".css", ".scss":
		return "Web"
	case ".json", ".yaml", ".yml", ".toml", ".xml":
		return "Configuration"
	case ".sql":
		return "SQL"
	case ".md":
		return "Documentation"
	default:
		return ""
	}
}

func isCartographyIgnoredDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".svn", "node_modules", "vendor", "dist", "build", "bin", "obj", "coverage", ".next", ".cache":
		return true
	default:
		return false
	}
}

func isCartographySensitiveFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, ".env") || strings.Contains(lower, "secret") || strings.Contains(lower, "credential") || strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key")
}

func inferCartographyRole(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "handler"), strings.Contains(lower, "controller"), strings.Contains(lower, "route"):
		return "Interface/API"
	case strings.Contains(lower, "store"), strings.Contains(lower, "repo"), strings.Contains(lower, "database"), strings.Contains(lower, "memory"):
		return "Storage/State"
	case strings.Contains(lower, "test"), strings.Contains(lower, "spec"):
		return "Verification"
	case strings.Contains(lower, "config"), strings.Contains(lower, "setting"):
		return "Configuration"
	case strings.Contains(lower, "main."), strings.Contains(lower, "app."):
		return "Entrypoint"
	case strings.Contains(lower, "ui"), strings.Contains(lower, "view"), strings.Contains(lower, "frontend"):
		return "User Interface"
	default:
		return "Module"
	}
}
