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
// the first one found. Nil (with no error) means there isn't one.
//
// The search stops after examining the repository root (a directory
// containing .git), $HOME, or the filesystem root — whichever comes first.
// That keeps a single .vop at the top of a repo covering every subdirectory
// without letting the search wander into unrelated parts of the filesystem.
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

		// Boundaries, checked after the directory itself so a .vop at the
		// repo root (or in $HOME) is still honoured.
		if isRepoRoot(dir) {
			return nil, nil
		}
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

func isRepoRoot(dir string) bool {
	// .git is a directory in a normal clone and a file in a worktree or
	// submodule; both mark a root.
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
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
