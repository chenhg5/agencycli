// Package workspace provides helpers for locating and validating an
// agencycli workspace on the local filesystem.
//
// An agencycli workspace is any directory that contains a .agencycli/agency.yaml
// file. Commands discover the workspace root by walking up the directory
// tree from the current working directory, the same way git finds .git.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// AIOSDir is the hidden metadata directory at the workspace root.
	AIOSDir = ".agencycli"
	// AgencyFile is the agency config file inside AIOSDir.
	AgencyFile = "agency.yaml"
)

// FindRoot walks up from start until it finds a directory that contains
// .agencycli/agency.yaml, then returns that directory's absolute path.
func FindRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("workspace: resolve path: %w", err)
	}

	dir := abs
	for {
		marker := filepath.Join(dir, AIOSDir, AgencyFile)
		if _, err := os.Stat(marker); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(
				"not inside an agencycli workspace "+
					"(no .agencycli/agency.yaml found in %q or any parent directory)",
				abs,
			)
		}
		dir = parent
	}
}

// FindRootFromCWD calls FindRoot starting from the current working directory.
func FindRootFromCWD() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("workspace: get cwd: %w", err)
	}
	return FindRoot(cwd)
}
