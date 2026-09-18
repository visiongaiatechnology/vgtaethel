// STATUS: DIAMANT VGT SUPREME
package coder

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go-aethel/security"
)

const (
	maximumGitOutput = 4 << 20
	maximumDiffBytes = 2 << 20
	maximumScanFiles = 20000
)

type RepositoryTracker struct{}

func (RepositoryTracker) InspectRoot(projectRoot string) (string, string, string) {
	root, err := security.CanonicalDir(projectRoot)
	if err != nil {
		return projectRoot, "", "FILESYSTEM_FALLBACK"
	}
	output, err := runGit(root, 5*time.Second, "rev-parse", "--show-toplevel")
	if err != nil {
		return root, "", "FILESYSTEM_FALLBACK"
	}
	repository, err := security.CanonicalDir(strings.TrimSpace(output))
	if err != nil || !security.IsPathInside(root, repository) {
		return root, "", "FILESYSTEM_FALLBACK"
	}
	branch, _ := runGit(repository, 4*time.Second, "branch", "--show-current")
	return repository, strings.TrimSpace(branch), "GIT"
}

func (RepositoryTracker) CaptureBaseline(root, mode string) (map[string]string, map[string]FileFingerprint, error) {
	if mode == "GIT" {
		files, err := gitWorktreeChanges(root)
		if err == nil {
			return signatures(files), nil, nil
		}
	}
	files, err := scanFingerprints(root)
	return map[string]string{}, files, err
}

func (RepositoryTracker) Changes(session CoderSession, scope string) ([]CoderChangedFile, error) {
	if session.DiffMode == "GIT" {
		if strings.EqualFold(scope, "BRANCH") {
			return gitBranchChanges(session.RepositoryRoot, session.BaseBranch)
		}
		current, err := gitWorktreeChanges(session.RepositoryRoot)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(scope, "AGENT") {
			return changesSinceBaseline(current, session.BaselineChanges), nil
		}
		return current, nil
	}
	current, err := scanFingerprints(session.ProjectRoot)
	if err != nil {
		return nil, err
	}
	return fingerprintChanges(session.BaselineFiles, current), nil
}

func (RepositoryTracker) Diff(session CoderSession, relativePath, scope string) (CoderDiff, error) {
	pathRoot := session.ProjectRoot
	if session.DiffMode == "GIT" {
		pathRoot = session.RepositoryRoot
	}
	relative, absolute, err := safeRepositoryPath(pathRoot, relativePath)
	if err != nil {
		return CoderDiff{}, err
	}
	changes := session.WorktreeFiles
	if strings.EqualFold(scope, "AGENT") {
		changes = session.ChangedFiles
	} else if strings.EqualFold(scope, "BRANCH") {
		changes = session.BranchFiles
	}
	change := CoderChangedFile{Path: relative, Status: "MODIFIED", Language: languageForPath(relative)}
	for _, candidate := range changes {
		if filepath.Clean(candidate.Path) == filepath.Clean(relative) {
			change = candidate
			break
		}
	}
	if change.Binary {
		return CoderDiff{Path: relative, OldPath: change.OldPath, Status: change.Status, Binary: true}, nil
	}
	var raw string
	if session.DiffMode == "GIT" && change.Status != "ADDED" {
		args := []string{"diff", "--no-ext-diff", "--unified=3"}
		if strings.EqualFold(scope, "BRANCH") && session.BaseBranch != "" {
			args = append(args, session.BaseBranch+"...HEAD")
		} else {
			args = append(args, "HEAD")
		}
		args = append(args, "--", filepath.ToSlash(relative))
		raw, err = runGit(session.RepositoryRoot, 10*time.Second, args...)
		if err != nil {
			return CoderDiff{}, err
		}
	} else if change.Status == "ADDED" {
		raw, err = addedFileDiff(absolute)
		if err != nil {
			return CoderDiff{}, err
		}
	} else {
		return CoderDiff{Path: relative, Status: change.Status, Additions: change.Additions, Deletions: change.Deletions}, nil
	}
	diff := parseUnifiedDiff(relative, change, raw)
	return diff, nil
}

func gitWorktreeChanges(root string) ([]CoderChangedFile, error) {
	output, err := runGit(root, 10*time.Second, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	fields := strings.Split(output, "\x00")
	changes := make([]CoderChangedFile, 0, len(fields))
	for index := 0; index < len(fields); index++ {
		entry := fields[index]
		if len(entry) < 4 {
			continue
		}
		xy := entry[:2]
		path := filepath.ToSlash(strings.TrimSpace(entry[3:]))
		change := CoderChangedFile{Path: path, Status: gitStatus(xy), Language: languageForPath(path)}
		if (strings.Contains(xy, "R") || strings.Contains(xy, "C")) && index+1 < len(fields) {
			index++
			change.OldPath = filepath.ToSlash(fields[index])
			change.Status = "RENAMED"
		}
		change.Additions, change.Deletions, change.Binary = gitFileStats(root, path, change.Status)
		change.Signature = gitChangeSignature(root, change)
		changes = append(changes, change)
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes, nil
}

func gitBranchChanges(root, base string) ([]CoderChangedFile, error) {
	if strings.TrimSpace(base) == "" {
		base = detectBaseBranch(root)
	}
	if base == "" {
		return []CoderChangedFile{}, nil
	}
	output, err := runGit(root, 10*time.Second, "diff", "--name-status", "-z", base+"...HEAD")
	if err != nil {
		return nil, err
	}
	fields := strings.Split(output, "\x00")
	changes := make([]CoderChangedFile, 0, len(fields)/2)
	for index := 0; index+1 < len(fields); {
		status := fields[index]
		index++
		if status == "" || index >= len(fields) {
			break
		}
		path := filepath.ToSlash(fields[index])
		index++
		change := CoderChangedFile{Path: path, Status: gitStatus(status), Language: languageForPath(path)}
		if strings.HasPrefix(status, "R") && index < len(fields) {
			change.OldPath = path
			change.Path = filepath.ToSlash(fields[index])
			change.Status = "RENAMED"
			index++
		}
		stats, _ := runGit(root, 5*time.Second, "diff", "--numstat", base+"...HEAD", "--", change.Path)
		change.Additions, change.Deletions, change.Binary = parseNumstat(stats)
		change.Signature = gitChangeSignature(root, change)
		changes = append(changes, change)
	}
	return changes, nil
}

func gitFileStats(root, path, status string) (int, int, bool) {
	if status == "ADDED" {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		data, err := os.ReadFile(absolute) // #nosec G304 -- Git supplied relative path is joined to the canonical repository.
		if err != nil || bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
			return 0, 0, true
		}
		return countLines(data), 0, false
	}
	output, err := runGit(root, 5*time.Second, "diff", "--numstat", "HEAD", "--", path)
	if err != nil {
		return 0, 0, false
	}
	added, deleted, binary := parseNumstat(output)
	if added == 0 && deleted == 0 && !binary {
		if patch, patchErr := runGit(root, 5*time.Second, "diff", "--no-ext-diff", "HEAD", "--", path); patchErr == nil {
			for _, line := range strings.Split(patch, "\n") {
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					added++
				} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					deleted++
				}
			}
		}
	}
	return added, deleted, binary
}

func parseNumstat(output string) (int, int, bool) {
	line := strings.TrimSpace(strings.Split(output, "\n")[0])
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0, 0, false
	}
	if parts[0] == "-" || parts[1] == "-" {
		return 0, 0, true
	}
	added, _ := strconv.Atoi(parts[0])
	deleted, _ := strconv.Atoi(parts[1])
	return added, deleted, false
}

func runGit(root string, timeout time.Duration, args ...string) (string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", errors.New("git unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	commandArgs := append([]string{"-C", root}, args...)
	command := exec.CommandContext(ctx, gitPath, commandArgs...) // #nosec G204 -- executable and argument boundaries are fixed; no shell is used.
	configureBackgroundProcess(command)
	var output bytes.Buffer
	command.Stdout = &limitedWriter{target: &output, remaining: maximumGitOutput}
	command.Stderr = &limitedWriter{target: &output, remaining: 32 << 10}
	err = command.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return output.String(), errors.New("git operation timed out")
	}
	if err != nil {
		return output.String(), fmt.Errorf("git operation failed")
	}
	return output.String(), nil
}

type limitedWriter struct {
	target    *bytes.Buffer
	remaining int
}

func (writer *limitedWriter) Write(data []byte) (int, error) {
	original := len(data)
	if writer.remaining <= 0 {
		return original, nil
	}
	if len(data) > writer.remaining {
		data = data[:writer.remaining]
	}
	_, _ = writer.target.Write(data)
	writer.remaining -= len(data)
	return original, nil
}

func safeRepositoryPath(root, input string) (string, string, error) {
	if input == "" || filepath.IsAbs(input) || strings.ContainsRune(input, 0) {
		return "", "", errors.New("invalid repository path")
	}
	clean := filepath.Clean(filepath.FromSlash(input))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", errors.New("repository traversal rejected")
	}
	canonicalRoot, err := security.CanonicalDir(root)
	if err != nil {
		return "", "", err
	}
	target, err := security.CanonicalTarget(filepath.Join(canonicalRoot, clean))
	if err != nil || !security.IsPathInside(canonicalRoot, target) {
		return "", "", errors.New("repository path escaped root")
	}
	return filepath.ToSlash(clean), target, nil
}

func addedFileDiff(path string) (string, error) {
	file, err := os.Open(path) // #nosec G304 -- canonical repository-jailed path.
	if err != nil {
		return "", err
	}
	defer file.Close()
	reader := bufio.NewScanner(file)
	reader.Buffer(make([]byte, 64<<10), maximumDiffBytes)
	var builder strings.Builder
	builder.WriteString("@@ -0,0 +1 @@\n")
	lines := 0
	for reader.Scan() && builder.Len() < maximumDiffBytes {
		builder.WriteString("+")
		builder.WriteString(reader.Text())
		builder.WriteByte('\n')
		lines++
	}
	if err := reader.Err(); err != nil {
		return "", err
	}
	return strings.Replace(builder.String(), "+1 @@", "+1,"+strconv.Itoa(lines)+" @@", 1), nil
}

func parseUnifiedDiff(path string, change CoderChangedFile, raw string) CoderDiff {
	diff := CoderDiff{Path: path, OldPath: change.OldPath, Status: change.Status, Binary: change.Binary, Additions: change.Additions, Deletions: change.Deletions, Hunks: []CoderDiffHunk{}}
	if len(raw) > maximumDiffBytes {
		raw = raw[:maximumDiffBytes]
		diff.Truncated = true
	}
	var current *CoderDiffHunk
	oldLine, newLine := 0, 0
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "@@") {
			hunk := CoderDiffHunk{Header: line, Lines: []CoderDiffLine{}}
			diff.Hunks = append(diff.Hunks, hunk)
			current = &diff.Hunks[len(diff.Hunks)-1]
			oldLine, newLine = parseHunkStarts(line)
			continue
		}
		if current == nil || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		diffLine := CoderDiffLine{Kind: "context", OldNumber: oldLine, NewNumber: newLine, Content: strings.TrimPrefix(line, " ")}
		switch {
		case strings.HasPrefix(line, "+"):
			diffLine.Kind, diffLine.OldNumber, diffLine.Content = "added", 0, strings.TrimPrefix(line, "+")
			newLine++
		case strings.HasPrefix(line, "-"):
			diffLine.Kind, diffLine.NewNumber, diffLine.Content = "deleted", 0, strings.TrimPrefix(line, "-")
			oldLine++
		default:
			oldLine++
			newLine++
		}
		current.Lines = append(current.Lines, diffLine)
	}
	if diff.Additions == 0 && diff.Deletions == 0 {
		for _, hunk := range diff.Hunks {
			for _, line := range hunk.Lines {
				if line.Kind == "added" {
					diff.Additions++
				} else if line.Kind == "deleted" {
					diff.Deletions++
				}
			}
		}
	}
	return diff
}

func parseHunkStarts(header string) (int, int) {
	parts := strings.Fields(header)
	parse := func(value string) int {
		value = strings.TrimLeft(value, "+-")
		value = strings.SplitN(value, ",", 2)[0]
		number, _ := strconv.Atoi(value)
		return number
	}
	if len(parts) < 3 {
		return 0, 0
	}
	return parse(parts[1]), parse(parts[2])
}

func signatures(files []CoderChangedFile) map[string]string {
	result := make(map[string]string, len(files))
	for _, file := range files {
		result[file.Path] = file.Signature
	}
	return result
}

func changesSinceBaseline(current []CoderChangedFile, baseline map[string]string) []CoderChangedFile {
	result := make([]CoderChangedFile, 0, len(current))
	for _, file := range current {
		if baseline[file.Path] != file.Signature {
			result = append(result, file)
		}
	}
	return result
}

func changeSignature(change CoderChangedFile) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%d\x00%t", change.Path, change.OldPath, change.Status, change.Additions, change.Deletions, change.Binary)))
	return hex.EncodeToString(digest[:])
}

func gitChangeSignature(root string, change CoderChangedFile) string {
	content := ""
	if change.Status == "ADDED" {
		absolute, err := security.CanonicalTarget(filepath.Join(root, filepath.FromSlash(change.Path)))
		if err == nil && security.IsPathInside(root, absolute) {
			if data, readErr := os.ReadFile(absolute); readErr == nil { // #nosec G304 -- canonical repository-jailed path.
				digest := sha256.Sum256(data)
				content = hex.EncodeToString(digest[:])
			}
		}
	} else if patch, err := runGit(root, 8*time.Second, "diff", "--binary", "HEAD", "--", change.Path); err == nil {
		digest := sha256.Sum256([]byte(patch))
		content = hex.EncodeToString(digest[:])
	}
	digest := sha256.Sum256([]byte(changeSignature(change) + "\x00" + content))
	return hex.EncodeToString(digest[:])
}

func gitStatus(status string) string {
	switch {
	case strings.Contains(status, "R"):
		return "RENAMED"
	case strings.Contains(status, "D"):
		return "DELETED"
	case strings.Contains(status, "A") || status == "??":
		return "ADDED"
	default:
		return "MODIFIED"
	}
}

func detectBaseBranch(root string) string {
	if output, err := runGit(root, 4*time.Second, "symbolic-ref", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimSpace(output)
	}
	for _, candidate := range []string{"origin/main", "origin/master", "main", "master"} {
		if _, err := runGit(root, 4*time.Second, "rev-parse", "--verify", candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func scanFingerprints(root string) (map[string]FileFingerprint, error) {
	canonicalRoot, canonicalErr := security.CanonicalDir(root)
	if canonicalErr != nil {
		return nil, canonicalErr
	}
	root = canonicalRoot
	result := map[string]FileFingerprint{}
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() && path != root && skipDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() || count >= maximumScanFiles {
			return nil
		}
		canonical, err := security.CanonicalTarget(path)
		if err != nil || !security.IsPathInside(root, canonical) {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() > 4<<20 {
			return nil
		}
		data, err := os.ReadFile(canonical) // #nosec G304 -- canonical project-root-jailed path.
		if err != nil {
			return nil
		}
		digest := sha256.Sum256(data)
		relative, _ := filepath.Rel(root, canonical)
		result[filepath.ToSlash(relative)] = FileFingerprint{Hash: hex.EncodeToString(digest[:]), Size: info.Size(), Binary: bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data), Lines: countLines(data)}
		count++
		return nil
	})
	return result, err
}

func fingerprintChanges(before, after map[string]FileFingerprint) []CoderChangedFile {
	result := []CoderChangedFile{}
	for path, current := range after {
		prior, existed := before[path]
		if existed && prior.Hash == current.Hash {
			continue
		}
		status := "ADDED"
		added, deleted := current.Lines, 0
		if existed {
			status = "MODIFIED"
			added = maxInt(0, current.Lines-prior.Lines)
			deleted = maxInt(0, prior.Lines-current.Lines)
		}
		change := CoderChangedFile{Path: path, Status: status, Additions: added, Deletions: deleted, Binary: current.Binary, Language: languageForPath(path)}
		change.Signature = changeSignature(change)
		result = append(result, change)
	}
	for path, prior := range before {
		if _, exists := after[path]; !exists {
			change := CoderChangedFile{Path: path, Status: "DELETED", Deletions: prior.Lines, Binary: prior.Binary, Language: languageForPath(path)}
			change.Signature = changeSignature(change)
			result = append(result, change)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	return bytes.Count(data, []byte{'\n'}) + 1
}

func skipDirectory(name string) bool {
	switch strings.ToLower(name) {
	case ".git", "node_modules", "vendor", "build", "dist", ".idea", ".vscode":
		return true
	default:
		return false
	}
}

func languageForPath(path string) string {
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	aliases := map[string]string{"js": "javascript", "jsx": "javascript", "ts": "typescript", "tsx": "typescript", "py": "python", "go": "go", "rs": "rust", "java": "java", "cs": "csharp", "cpp": "cpp", "c": "c", "html": "html", "css": "css", "json": "json", "md": "markdown", "yaml": "yaml", "yml": "yaml"}
	if language := aliases[extension]; language != "" {
		return language
	}
	return "text"
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
