// STATUS: DIAMANT VGT SUPREME
package coder

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryTrackerDetectsAgentChangesAndDiff(t *testing.T) {
	root := initializeTestRepository(t)
	tracker := RepositoryTracker{}
	repository, branch, mode := tracker.InspectRoot(root)
	if mode != "GIT" || repository == "" || branch == "" {
		t.Fatalf("expected git repository, got mode=%s root=%s branch=%s", mode, repository, branch)
	}
	baseline, _, err := tracker.CaptureBaseline(repository, mode)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "alpha.txt"), "one\nchanged\n")
	writeTestFile(t, filepath.Join(root, "new.go"), "package demo\n\nfunc Value() int { return 7 }\n")
	session := CoderSession{ProjectRoot: root, RepositoryRoot: repository, DiffMode: mode, BaselineChanges: baseline}
	changes, err := tracker.Changes(session, "AGENT")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %#v", changes)
	}
	diff, err := tracker.Diff(sessionWithChanges(session, changes), "alpha.txt", "AGENT")
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Hunks) == 0 || diff.Additions != 1 || diff.Deletions != 1 {
		t.Fatalf("unexpected diff: %#v", diff)
	}
}

func TestRepositoryTrackerNoticesSameNumstatContentChange(t *testing.T) {
	root := initializeTestRepository(t)
	tracker := RepositoryTracker{}
	repository, _, mode := tracker.InspectRoot(root)
	writeTestFile(t, filepath.Join(root, "alpha.txt"), "one\nfirst\n")
	first, err := gitWorktreeChanges(repository)
	if err != nil {
		t.Fatal(err)
	}
	baseline := signatures(first)
	writeTestFile(t, filepath.Join(root, "alpha.txt"), "one\nother\n")
	current, err := gitWorktreeChanges(repository)
	if err != nil {
		t.Fatal(err)
	}
	if got := changesSinceBaseline(current, baseline); len(got) != 1 {
		t.Fatalf("same-size edit was not detected: %#v", got)
	}
	if mode != "GIT" {
		t.Fatalf("unexpected mode %s", mode)
	}
}

func TestRepositoryTrackerRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, _, err := safeRepositoryPath(root, "../escape.txt"); err == nil {
		t.Fatal("path traversal must be rejected")
	}
	if _, _, err := safeRepositoryPath(root, filepath.Join(root, "absolute.txt")); err == nil {
		t.Fatal("absolute path must be rejected")
	}
}

func TestFilesystemFallbackTracksAddedModifiedDeletedAndBinary(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "changed.txt"), "before\n")
	writeTestFile(t, filepath.Join(root, "deleted.txt"), "gone\n")
	baseline, err := scanFingerprints(root)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "changed.txt"), "after\nsecond\n")
	if err := os.Remove(filepath.Join(root, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "binary.bin"), []byte{0, 1, 2, 3}, 0o600); err != nil {
		t.Fatal(err)
	}
	current, err := scanFingerprints(root)
	if err != nil {
		t.Fatal(err)
	}
	changes := fingerprintChanges(baseline, current)
	if len(changes) != 3 {
		t.Fatalf("expected 3 fallback changes, got %#v", changes)
	}
	if !findChange(changes, "binary.bin").Binary {
		t.Fatal("binary file not classified")
	}
}

func initializeTestRepository(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	runTestGit(t, root, "init", "-b", "main")
	runTestGit(t, root, "config", "user.email", "vgt@example.invalid")
	runTestGit(t, root, "config", "user.name", "VGT Test")
	writeTestFile(t, filepath.Join(root, "alpha.txt"), "one\ntwo\n")
	runTestGit(t, root, "add", "alpha.txt")
	runTestGit(t, root, "commit", "-m", "baseline")
	return root
}

func runTestGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func sessionWithChanges(session CoderSession, changes []CoderChangedFile) CoderSession {
	session.ChangedFiles = changes
	return session
}

func findChange(changes []CoderChangedFile, path string) CoderChangedFile {
	for _, change := range changes {
		if change.Path == path {
			return change
		}
	}
	return CoderChangedFile{}
}
