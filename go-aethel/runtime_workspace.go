package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	runtimeVendorDirectory = "VGT"
	runtimeAppDirectory    = "AETHEL"
	runtimeWorkspaceName   = "vgt_workspace"
)

// configureRuntimeWorkingDirectory separates persistent operator state from
// replaceable binaries while retaining compatibility with relative paths.
func configureRuntimeWorkingDirectory() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}

	dataRoot := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if dataRoot == "" {
		dataRoot, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve per-user data directory: %w", err)
		}
	}

	runtimeDir, err := prepareRuntimeDirectory(executablePath, dataRoot)
	if err != nil {
		return "", err
	}
	if err := os.Chdir(runtimeDir); err != nil {
		return "", fmt.Errorf("bind runtime directory: %w", err)
	}
	return runtimeDir, nil
}

func prepareRuntimeDirectory(executablePath, dataRoot string) (string, error) {
	if strings.TrimSpace(executablePath) == "" || strings.TrimSpace(dataRoot) == "" {
		return "", errors.New("runtime paths must not be empty")
	}

	absoluteDataRoot, err := filepath.Abs(dataRoot)
	if err != nil {
		return "", fmt.Errorf("resolve data root: %w", err)
	}
	runtimeDir := filepath.Join(absoluteDataRoot, runtimeVendorDirectory, runtimeAppDirectory)
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return "", fmt.Errorf("create runtime directory: %w", err)
	}

	legacyWorkspace := filepath.Join(filepath.Dir(filepath.Clean(executablePath)), runtimeWorkspaceName)
	persistentWorkspace := filepath.Join(runtimeDir, runtimeWorkspaceName)
	if !sameRuntimePath(legacyWorkspace, persistentWorkspace) {
		if err := migrateLegacyWorkspace(legacyWorkspace, persistentWorkspace, runtimeDir); err != nil {
			return "", err
		}
	}
	return runtimeDir, nil
}

// migrateLegacyWorkspace is copy-only. The source remains available for
// recovery until the operator removes it explicitly.
func migrateLegacyWorkspace(source, destination, runtimeDir string) error {
	sourceInfo, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect legacy workspace: %w", err)
	}
	if !sourceInfo.IsDir() {
		return errors.New("legacy workspace is not a directory")
	}

	destinationInfo, err := os.Stat(destination)
	if err == nil {
		if !destinationInfo.IsDir() {
			return errors.New("persistent workspace path is not a directory")
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect persistent workspace: %w", err)
	}

	stagingRoot, err := os.MkdirTemp(runtimeDir, ".workspace-migration-")
	if err != nil {
		return fmt.Errorf("create migration staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stagingRoot) }()

	stagedWorkspace := filepath.Join(stagingRoot, runtimeWorkspaceName)
	if err := copyWorkspaceDirectory(source, stagedWorkspace); err != nil {
		return fmt.Errorf("copy legacy workspace: %w", err)
	}
	if err := os.Rename(stagedWorkspace, destination); err != nil {
		return fmt.Errorf("activate persistent workspace: %w", err)
	}
	return nil
}

func copyWorkspaceDirectory(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link rejected: %s", path)
		}

		relativePath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destination, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("non-regular workspace entry rejected: %s", path)
		}
		return copyWorkspaceFile(path, targetPath)
	})
}

func copyWorkspaceFile(source, destination string) (copyErr error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { copyErr = errors.Join(copyErr, input.Close()) }()

	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { copyErr = errors.Join(copyErr, output.Close()) }()

	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return output.Sync()
}

func sameRuntimePath(left, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
