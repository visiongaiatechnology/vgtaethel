// STATUS: DIAMANT VGT SUPREME
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"go-aethel/security"
	"go-aethel/system"
)

var (
	codeWorkspaceMu   sync.RWMutex
	codeWorkspaceRoot string
)

var errNoCodeWorkspace = errors.New("no code workspace selected")

const (
	codeMaxFileBytes = 2 << 20
	codeMaxTreeItems = 1200
	codeMaxTreeDepth = 5
)

type codeTreeEntry struct {
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	Kind     string          `json:"kind"`
	Size     int64           `json:"size_bytes,omitempty"`
	Children []codeTreeEntry `json:"children,omitempty"`
}

type codeFilePayload struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	Size       int    `json:"size_bytes"`
	SnapshotID string `json:"snapshot_id,omitempty"`
}

// SetCodeWorkspaceRoot activates exactly one operator-selected coding project.
// Selection authority originates from the native Wails directory dialog.
func SetCodeWorkspaceRoot(path string) error {
	canonical, err := security.CanonicalDir(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		return errors.New("selected code workspace is not a directory")
	}
	codeWorkspaceMu.Lock()
	codeWorkspaceRoot = canonical
	codeWorkspaceMu.Unlock()
	return nil
}

func ClearCodeWorkspaceRoot() {
	codeWorkspaceMu.Lock()
	codeWorkspaceRoot = ""
	codeWorkspaceMu.Unlock()
}

func currentCodeWorkspaceRoot() (string, error) {
	codeWorkspaceMu.RLock()
	root := codeWorkspaceRoot
	codeWorkspaceMu.RUnlock()
	if root == "" {
		return "", errNoCodeWorkspace
	}
	return root, nil
}

func CurrentCodeWorkspaceRoot() (string, error) {
	return currentCodeWorkspaceRoot()
}

func HandleCodeWorkspaceTree(w http.ResponseWriter, r *http.Request) {
	setCodeJSONHeaders(w)
	if r.Method != http.MethodGet {
		codeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is supported.")
		return
	}
	root, target, relative, err := resolveCodeWorkspacePath(r.URL.Query().Get("path"), false)
	if err != nil {
		if errors.Is(err, errNoCodeWorkspace) {
			codeError(w, http.StatusConflict, "no_project", "Select a coding project first.")
			return
		}
		codeError(w, http.StatusBadRequest, "invalid_path", "Workspace path rejected.")
		return
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		codeError(w, http.StatusNotFound, "directory_not_found", "Workspace directory not found.")
		return
	}
	remaining := codeMaxTreeItems
	entries, truncated, err := readCodeTree(root, target, relative, 0, &remaining)
	if err != nil {
		codeError(w, http.StatusInternalServerError, "tree_unavailable", "Workspace tree unavailable.")
		return
	}
	writeCodeJSON(w, http.StatusOK, map[string]interface{}{
		"root": filepath.Base(root), "path": relative, "entries": entries, "truncated": truncated,
	})
}

func HandleCodeWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	setCodeJSONHeaders(w)
	switch r.Method {
	case http.MethodGet:
		handleCodeFileRead(w, r)
	case http.MethodPut:
		handleCodeFileWrite(w, r)
	default:
		codeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET and PUT are supported.")
	}
}

func handleCodeFileRead(w http.ResponseWriter, r *http.Request) {
	_, target, relative, err := resolveCodeWorkspacePath(r.URL.Query().Get("path"), false)
	if err != nil {
		if errors.Is(err, errNoCodeWorkspace) {
			codeError(w, http.StatusConflict, "no_project", "Select a coding project first.")
			return
		}
		codeError(w, http.StatusBadRequest, "invalid_path", "Workspace path rejected.")
		return
	}
	file, err := os.Open(target) // #nosec G304 -- target is canonical and workspace-jailed above.
	if err != nil {
		codeError(w, http.StatusNotFound, "file_not_found", "Workspace file not found.")
		return
	}
	defer file.Close()
	limited := io.LimitReader(file, codeMaxFileBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		codeError(w, http.StatusInternalServerError, "read_failed", "Workspace file could not be read.")
		return
	}
	if len(data) > codeMaxFileBytes {
		codeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "File exceeds the 2 MiB editor boundary.")
		return
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		codeError(w, http.StatusUnsupportedMediaType, "binary_file", "Binary files are not available in the text editor.")
		return
	}
	writeCodeJSON(w, http.StatusOK, codeFilePayload{Path: relative, Content: string(data), Size: len(data)})
}

func handleCodeFileWrite(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, codeMaxFileBytes+4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decoder.Decode(&request); err != nil {
		codeError(w, http.StatusBadRequest, "invalid_request", "Invalid or oversized editor payload.")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		codeError(w, http.StatusBadRequest, "invalid_request", "Only one editor payload is accepted.")
		return
	}
	if len(request.Content) > codeMaxFileBytes || !utf8.ValidString(request.Content) {
		codeError(w, http.StatusRequestEntityTooLarge, "invalid_content", "Editor content exceeds the UTF-8 file boundary.")
		return
	}
	_, target, relative, err := resolveCodeWorkspacePath(request.Path, true)
	if err != nil {
		if errors.Is(err, errNoCodeWorkspace) {
			codeError(w, http.StatusConflict, "no_project", "Select a coding project first.")
			return
		}
		codeError(w, http.StatusBadRequest, "invalid_path", "Workspace path rejected.")
		return
	}
	snapshotID, err := system.DefaultFileSnapshots.Create(target)
	if err != nil {
		codeError(w, http.StatusInternalServerError, "snapshot_failed", "Safety snapshot could not be created.")
		return
	}
	root, name, safeTarget, err := security.OpenAuthorizedFile(target, security.MountWrite)
	if err != nil || !codePathsEqual(safeTarget, target) {
		if root != nil {
			_ = root.Close()
		}
		codeError(w, http.StatusForbidden, "write_rejected", "Workspace write rejected.")
		return
	}
	defer root.Close()
	if err := root.WriteFile(name, []byte(request.Content), 0600); err != nil {
		codeError(w, http.StatusInternalServerError, "write_failed", "Workspace file could not be saved.")
		return
	}
	writeCodeJSON(w, http.StatusOK, codeFilePayload{Path: relative, Content: request.Content, Size: len(request.Content), SnapshotID: snapshotID})
}

func codePathsEqual(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func resolveCodeWorkspacePath(input string, allowMissing bool) (string, string, string, error) {
	if strings.ContainsRune(input, 0) || len(input) > 1024 || filepath.IsAbs(input) {
		return "", "", "", errors.New("invalid workspace path")
	}
	clean := filepath.Clean(strings.TrimSpace(input))
	if clean == "." {
		clean = ""
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", "", errors.New("workspace traversal rejected")
	}
	root, err := currentCodeWorkspaceRoot()
	if err != nil {
		return "", "", "", err
	}
	target, err := security.CanonicalTarget(filepath.Join(root, clean))
	if err != nil || !security.IsPathInside(root, target) {
		return "", "", "", errors.New("workspace jail escaped")
	}
	authorized, err := security.ValidatePathForAccess(target, security.MountRead)
	if err != nil || !codePathsEqual(authorized, target) {
		return "", "", "", errors.New("workspace authorization expired")
	}
	if !allowMissing {
		if _, err := os.Stat(target); err != nil {
			return "", "", "", err
		}
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", "", errors.New("workspace relative path rejected")
	}
	if relative == "." {
		relative = ""
	}
	return root, target, filepath.ToSlash(relative), nil
}

func readCodeTree(root, dir, relative string, depth int, remaining *int) ([]codeTreeEntry, bool, error) {
	if depth >= codeMaxTreeDepth || *remaining <= 0 {
		return []codeTreeEntry{}, true, nil
	}
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false, err
	}
	sort.Slice(dirEntries, func(i, j int) bool {
		if dirEntries[i].IsDir() != dirEntries[j].IsDir() {
			return dirEntries[i].IsDir()
		}
		return strings.ToLower(dirEntries[i].Name()) < strings.ToLower(dirEntries[j].Name())
	})
	result := make([]codeTreeEntry, 0, len(dirEntries))
	truncated := false
	for _, entry := range dirEntries {
		if *remaining <= 0 {
			truncated = true
			break
		}
		if shouldSkipCodeTreeEntry(entry.Name()) {
			continue
		}
		candidate := filepath.Join(dir, entry.Name())
		canonical, canonicalErr := security.CanonicalTarget(candidate)
		if canonicalErr != nil || !security.IsPathInside(root, canonical) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		*remaining--
		path := filepath.ToSlash(filepath.Join(relative, entry.Name()))
		item := codeTreeEntry{Name: entry.Name(), Path: path, Kind: "file", Size: info.Size()}
		if entry.IsDir() {
			item.Kind = "directory"
			item.Size = 0
			children, childTruncated, childErr := readCodeTree(root, canonical, path, depth+1, remaining)
			if childErr == nil {
				item.Children = children
			}
			truncated = truncated || childTruncated
		}
		result = append(result, item)
	}
	return result, truncated, nil
}

func shouldSkipCodeTreeEntry(name string) bool {
	switch strings.ToLower(name) {
	case ".git", "node_modules", "vendor", ".idea", ".vscode", "file_snapshots":
		return true
	default:
		return false
	}
}

func setCodeJSONHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func codeError(w http.ResponseWriter, status int, code, message string) {
	writeCodeJSON(w, status, map[string]string{"error": code, "message": message})
}

func writeCodeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
