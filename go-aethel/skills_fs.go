package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// --- 2. SKILL: READ FILE ---

type ReadFileSkill struct{}

type ReadFileArgs struct {
	Path string `json:"path"`
}

func (s *ReadFileSkill) Name() string { return "fs_read_file" }
func (s *ReadFileSkill) Description() string {
	return "Liest den Inhalt einer Textdatei aus dem VGT Workspace."
}
func (s *ReadFileSkill) RiskLevel() RiskLevel { return RiskSafe }

func (s *ReadFileSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string", "description": "Relativer Pfad zur Datei (z.B. 'notes.txt')"},
		},
		"required": []string{"path"},
	}
}

func (s *ReadFileSkill) Execute(args json.RawMessage) (string, error) {
	var input ReadFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	safePath, err := validatePath(input.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		LogKernelActivity("READ_FAILED", input.Path, "ERROR")
		return "", err
	}
	LogKernelActivity("READ", input.Path, "SUCCESS")
	return string(data), nil
}

// --- 3. SKILL: WRITE FILE ---

type WriteFileSkill struct{}

type WriteFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (s *WriteFileSkill) Name() string { return "fs_write_file" }
func (s *WriteFileSkill) Description() string {
	return "Schreibt Text in eine Datei im VGT Workspace. Überschreibt existierende Dateien."
}
func (s *WriteFileSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *WriteFileSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":    map[string]interface{}{"type": "string", "description": "Relativer Zielpfad"},
			"content": map[string]interface{}{"type": "string", "description": "Vollständiger Textinhalt"},
		},
		"required": []string{"path", "content"},
	}
}

func (s *WriteFileSkill) Execute(args json.RawMessage) (string, error) {
	var input WriteFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	safePath, err := validateWritePath(input.Path)
	if err != nil {
		return "", err
	}
	snapshotID, err := defaultFileSnapshots.Create(safePath)
	if err != nil {
		return "", err
	}

	// Read old content if exists
	var oldContent string
	if data, err := os.ReadFile(safePath); err == nil {
		oldContent = string(data)
	}

	// Ensure parent directories exist
	if err := os.MkdirAll(filepath.Dir(safePath), 0700); err != nil {
		return "", err
	}

	if err := os.WriteFile(safePath, []byte(input.Content), 0600); err != nil {
		LogKernelActivity("WRITE_FAILED", input.Path, "ERROR")
		return "", err
	}
	LogKernelActivity("WRITE", input.Path, "SUCCESS")

	// Compute and record line changes
	added, removed := computeLineChanges(oldContent, input.Content)
	recordFileChange(safePath, added, removed)

	return fmt.Sprintf("Datei erfolgreich geschrieben: %s (+%d, -%d Zeilen) Snapshot: %s", input.Path, added, removed, snapshotID), nil
}

// computeLineChanges matches line counts to calculate added/removed lines for diff panels.
func computeLineChanges(oldContent, newContent string) (added, removed int) {
	if oldContent == newContent {
		return 0, 0
	}
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	oldLineMap := make(map[string]int)
	for _, l := range oldLines {
		oldLineMap[strings.TrimSpace(l)]++
	}

	for _, l := range newLines {
		trimmed := strings.TrimSpace(l)
		if count, exists := oldLineMap[trimmed]; exists && count > 0 {
			oldLineMap[trimmed]--
		} else {
			added++
		}
	}

	for _, count := range oldLineMap {
		removed += count
	}
	return
}

// --- 3b. SKILL: REPLACE FILE CONTENT ---

type ReplaceFileContentSkill struct{}

type ReplaceFileContentArgs struct {
	Path               string `json:"path"`
	StartLine          int    `json:"start_line"`
	EndLine            int    `json:"end_line"`
	TargetContent      string `json:"target_content"`
	ReplacementContent string `json:"replacement_content"`
}

func (s *ReplaceFileContentSkill) Name() string { return "fs_replace_file_content" }
func (s *ReplaceFileContentSkill) Description() string {
	return "Ersetzt einen bestimmten, exakten Textblock in einer Datei innerhalb eines Zeilenbereichs."
}
func (s *ReplaceFileContentSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *ReplaceFileContentSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":                map[string]interface{}{"type": "string", "description": "Relativer oder absoluter Pfad zur Datei"},
			"start_line":          map[string]interface{}{"type": "integer", "description": "Startzeile (1-basiert) für den Suchbereich"},
			"end_line":            map[string]interface{}{"type": "integer", "description": "Endzeile (1-basiert) für den Suchbereich"},
			"target_content":      map[string]interface{}{"type": "string", "description": "Der exakte Textblock, der ersetzt werden soll"},
			"replacement_content": map[string]interface{}{"type": "string", "description": "Der neue Textblock, der als Ersatz dient"},
		},
		"required": []string{"path", "start_line", "end_line", "target_content", "replacement_content"},
	}
}

func (s *ReplaceFileContentSkill) Execute(args json.RawMessage) (string, error) {
	var input ReplaceFileContentArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	safePath, err := validateWritePath(input.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", err
	}
	oldContent := string(data)
	snapshotID, err := defaultFileSnapshots.Create(safePath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(oldContent, "\n")
	if input.StartLine < 1 || input.EndLine > len(lines) || input.StartLine > input.EndLine {
		return "", fmt.Errorf("ungültiger Zeilenbereich: %d bis %d (Datei hat %d Zeilen)", input.StartLine, input.EndLine, len(lines))
	}

	// Extract lines within range and verify target_content
	rangeContent := strings.Join(lines[input.StartLine-1:input.EndLine], "\n")
	if !strings.Contains(rangeContent, input.TargetContent) {
		return "", fmt.Errorf("der Zieltext konnte im angegebenen Zeilenbereich [%d - %d] nicht gefunden werden", input.StartLine, input.EndLine)
	}

	// Perform replacement
	newRangeContent := strings.Replace(rangeContent, input.TargetContent, input.ReplacementContent, 1)

	// Rebuild content
	var newLines []string
	newLines = append(newLines, lines[:input.StartLine-1]...)
	newLines = append(newLines, strings.Split(newRangeContent, "\n")...)
	newLines = append(newLines, lines[input.EndLine:]...)
	newContent := strings.Join(newLines, "\n")

	if err := os.WriteFile(safePath, []byte(newContent), 0600); err != nil {
		LogKernelActivity("WRITE_FAILED", input.Path, "ERROR")
		return "", err
	}
	LogKernelActivity("WRITE", input.Path, "SUCCESS")

	// Compute and record line changes
	added, removed := computeLineChanges(oldContent, newContent)
	recordFileChange(safePath, added, removed)

	return fmt.Sprintf("Erfolgreich geändert in %s (+%d, -%d Zeilen) Snapshot: %s", filepath.Base(safePath), added, removed, snapshotID), nil
}

// --- 7. SKILL: LIST DIRECTORY ---

type ListDirSkill struct{}

type ListDirArgs struct {
	Path string `json:"path"`
}

func (s *ListDirSkill) Name() string { return "fs_list_dir" }
func (s *ListDirSkill) Description() string {
	return "Listet alle Dateien und Ordner in einem Verzeichnis im Workspace oder einem eingehängten Ordner auf."
}
func (s *ListDirSkill) RiskLevel() RiskLevel { return RiskSafe }

func (s *ListDirSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string", "description": "Relativer Pfad im Workspace (z.B. '.') oder ein absoluter Pfad zu einem eingehängten Ordner"},
		},
		"required": []string{"path"},
	}
}

func (s *ListDirSkill) Execute(args json.RawMessage) (string, error) {
	var input ListDirArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	safePath, err := validatePath(input.Path)
	if err != nil {
		return "", err
	}

	files, err := os.ReadDir(safePath)
	if err != nil {
		LogKernelActivity("LIST_DIR_FAILED", input.Path, "ERROR")
		return "", err
	}

	var fileInfos []string
	for _, f := range files {
		info, err := f.Info()
		var sizeStr string
		if err == nil {
			sizeStr = fmt.Sprintf("%d Bytes", info.Size())
		} else {
			sizeStr = "Größe unbekannt"
		}

		typeStr := "Datei"
		if f.IsDir() {
			typeStr = "Ordner"
			sizeStr = "-"
		}

		fileInfos = append(fileInfos, fmt.Sprintf("- [%s] %s (Größe: %s)", typeStr, f.Name(), sizeStr))
	}

	LogKernelActivity("LIST_DIR", input.Path, "SUCCESS")
	if len(fileInfos) == 0 {
		return "Verzeichnis ist leer.", nil
	}
	return strings.Join(fileInfos, "\n"), nil
}

// --- 8. SKILL: MOUNT FOLDER ---

type MountFolderSkill struct{}

type MountFolderArgs struct {
	Path            string `json:"path"`
	Access          string `json:"access"`
	DurationMinutes int    `json:"duration_minutes"`
}

func (s *MountFolderSkill) Name() string { return "fs_mount_folder" }
func (s *MountFolderSkill) Description() string {
	return "Hängt einen Ordner vom Host-PC in das System ein, damit Aethel lesend/schreibend darauf zugreifen darf."
}
func (s *MountFolderSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *MountFolderSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"access":           map[string]interface{}{"type": "string", "enum": []string{"read", "write"}},
			"duration_minutes": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 1440},
			"path":             map[string]interface{}{"type": "string", "description": "Absoluter Pfad des Ordners, der eingehängt werden soll (z.B. 'C:\\Users\\Masterboard\\Documents')"},
		},
		"required": []string{"path"},
	}
}

func (s *MountFolderSkill) Execute(args json.RawMessage) (string, error) {
	var input MountFolderArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	// Verify if directory actually exists on the host
	info, err := os.Stat(input.Path)
	if err != nil {
		return "", fmt.Errorf("Ordner existiert nicht oder Zugriff verweigert: %v", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("Pfad '%s' ist kein Ordner", input.Path)
	}

	access := MountAccess(strings.ToLower(strings.TrimSpace(input.Access)))
	if access == "" {
		access = MountRead
	}
	duration := input.DurationMinutes
	if duration == 0 {
		duration = 60
	}
	err = state.AddMount(input.Path, access, time.Duration(duration)*time.Minute)
	if err != nil {
		LogKernelActivity("MOUNT_FAILED", input.Path, "ERROR")
		return "", fmt.Errorf("Fehler beim Speichern der Einhängung: %v", err)
	}

	LogKernelActivity("MOUNT_FOLDER", input.Path, "SUCCESS")

	// List currently mounted dirs
	mounts := state.GetMounts()
	var mountsList []string
	for _, m := range mounts {
		mountsList = append(mountsList, fmt.Sprintf("- %s (%s until %s)", m.Path, m.Access, m.ExpiresAt.Local().Format(time.RFC3339)))
	}

	return fmt.Sprintf("Ordner '%s' erfolgreich eingehängt.\n\nAktuell eingehängte Ordner:\n%s", input.Path, strings.Join(mountsList, "\n")), nil
}
