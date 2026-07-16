//go:build !windows

package security

import "os"

func replacePrivateFile(source, destination string) error {
	return os.Rename(source, destination)
}
