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
//
// A linked git worktree lives outside the repository's own directory tree —
// typically under a tool's scratch area — so walking up from it never reaches
// a .vop the user placed above their main checkout. When the physical walk
// finds nothing, the search is retried from the repository's main working
// tree, so a worktree inherits the same default its main checkout would.
func FindDirProfile(startDir string) (*DirProfile, error) {
	dp, err := walkForDirProfile(startDir)
	if err != nil || dp != nil {
		return dp, err
	}
	if main := mainWorktreeDir(startDir); main != "" {
		return walkForDirProfile(main)
	}
	return nil, nil
}

// walkForDirProfile performs the upward .vop search from startDir, honouring
// the $HOME and filesystem-root boundaries.
func walkForDirProfile(startDir string) (*DirProfile, error) {
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

// mainWorktreeDir returns the path of the repository's main working tree when
// startDir sits inside a linked git worktree, or "" otherwise — including when
// startDir is a plain repository, is the main worktree itself, or isn't in a
// repository at all. It reads git's on-disk layout directly rather than
// shelling out to git.
func mainWorktreeDir(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}
	for range dirProfileWalkLimit {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				// A .git directory is a plain repo or the main worktree; its
				// own ancestors were already covered by the physical walk.
				return ""
			}
			return resolveMainWorktree(gitPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}

// resolveMainWorktree maps a linked worktree's .git file to the path of the
// repository's main working tree, or "" if gitFile isn't a worktree pointer.
//
// A linked worktree's .git is a file holding "gitdir: <path>" where <path> is
// the per-worktree admin dir (…/.git/worktrees/<name>). That dir's commondir
// file locates the shared .git directory, whose parent is the main worktree.
func resolveMainWorktree(gitFile string) string {
	data, err := os.ReadFile(gitFile)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	gitdir, ok := strings.CutPrefix(line, "gitdir:")
	if !ok {
		return ""
	}
	gitdir = strings.TrimSpace(gitdir)
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(filepath.Dir(gitFile), gitdir)
	}

	commonRaw, err := os.ReadFile(filepath.Join(filepath.Clean(gitdir), "commondir"))
	if err != nil {
		// No commondir means this isn't a linked worktree (e.g. a submodule's
		// gitdir points straight at the parent repo's admin dir).
		return ""
	}
	common := strings.TrimSpace(string(commonRaw))
	if !filepath.IsAbs(common) {
		common = filepath.Join(gitdir, common)
	}
	// common is the shared ".git" directory; the working tree is its parent.
	return filepath.Dir(filepath.Clean(common))
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
