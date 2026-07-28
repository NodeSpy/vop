package config

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DirProfileFilename is the per-directory marker naming the vop profile to
// use by default in that directory and its children.
const DirProfileFilename = ".vop"

// maxDirProfileSize caps how much of a .vop file we read. The file holds a
// profile name; anything larger is a mistake (or a hostile repo) and there's
// no reason to pull it into memory.
const maxDirProfileSize = 4 << 10

// dirProfileWalkLimit bounds the upward search so a pathological symlink
// arrangement can't spin.
const dirProfileWalkLimit = 64

// DirProfile is a profile name discovered from a .vop file on disk.
type DirProfile struct {
	Name string // profile name declared in the file
	Path string // absolute path to the .vop file it came from
}

// FindDirProfile walks up from startDir looking for a .vop file, returning
// the nearest one found. Nil (with no error) means there isn't one.
//
// The search stops after examining $HOME or the filesystem root — whichever
// comes first. Repository roots are deliberately not boundaries: a .vop above
// a checkout is the user's own directory layout, not repository content, so a
// single file at ~/Projects/acme can supply the default for every repo beneath
// it. Files inside a checkout still take precedence, since the nearest one
// wins, and an empty .vop at a repo root opts that repo out of inheriting.
func FindDirProfile(startDir string) (*DirProfile, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}
	home, _ := os.UserHomeDir()

	for range dirProfileWalkLimit {
		path := filepath.Join(dir, DirProfileFilename)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			name, err := readDirProfile(path)
			if err != nil {
				return nil, err
			}
			if name != "" {
				return &DirProfile{Name: name, Path: path}, nil
			}
			// An empty or comments-only file is a deliberate "no default
			// here" marker — stop rather than inheriting a parent's.
			return nil, nil
		}

		// Boundaries, checked after the directory itself so a .vop in $HOME
		// is still honoured.
		if home != "" && dir == home {
			return nil, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}
	return nil, nil
}

// readDirProfile returns the profile name declared in a .vop file: the first
// non-empty line that isn't a comment. Blank lines and lines starting with #
// are ignored so the file can explain itself.
func readDirProfile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxDirProfileSize))
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.ContainsAny(line, " \t") {
			return "", fmt.Errorf("%s: expected a single profile name, got %q", path, line)
		}
		return line, nil
	}
	return "", nil
}
