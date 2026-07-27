package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeDirProfile creates dir (and parents) and puts a .vop file in it.
func writeDirProfile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, DirProfileFilename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// markRepoRoot makes dir look like a git repository root.
func markRepoRoot(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestFindDirProfile_InCurrentDir(t *testing.T) {
	dir := t.TempDir()
	writeDirProfile(t, dir, "tap\n")

	got, err := FindDirProfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Name != "tap" {
		t.Fatalf("got %+v, want name 'tap'", got)
	}
	if got.Path != filepath.Join(dir, DirProfileFilename) {
		t.Errorf("path = %q", got.Path)
	}
}

func TestFindDirProfile_WalksUpToRepoRoot(t *testing.T) {
	root := t.TempDir()
	markRepoRoot(t, root)
	writeDirProfile(t, root, "tap\n")

	deep := filepath.Join(root, "modules", "vpc", "sub")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := FindDirProfile(deep)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Name != "tap" {
		t.Fatalf("got %+v, want name 'tap' from the repo root", got)
	}
}

func TestFindDirProfile_NearestWins(t *testing.T) {
	root := t.TempDir()
	markRepoRoot(t, root)
	writeDirProfile(t, root, "tap")
	sub := filepath.Join(root, "staging")
	writeDirProfile(t, sub, "ednition")

	got, err := FindDirProfile(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Name != "ednition" {
		t.Fatalf("got %+v, want the nearest file to win", got)
	}
}

func TestFindDirProfile_StopsAtRepoRoot(t *testing.T) {
	outer := t.TempDir()
	// A .vop above the repo root must not leak into the repo.
	writeDirProfile(t, outer, "tap")
	repo := filepath.Join(outer, "repo")
	markRepoRoot(t, repo)
	deep := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := FindDirProfile(deep)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("search escaped the repo root: got %+v", got)
	}
}

func TestFindDirProfile_GitFileMarksRoot(t *testing.T) {
	// Worktrees and submodules have .git as a file, not a directory.
	outer := t.TempDir()
	writeDirProfile(t, outer, "tap")
	repo := filepath.Join(outer, "worktree")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: /elsewhere\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := FindDirProfile(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("search escaped a worktree root: got %+v", got)
	}
}

func TestFindDirProfile_NoneFound(t *testing.T) {
	dir := t.TempDir()
	markRepoRoot(t, dir)
	got, err := FindDirProfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("got %+v, want nil", got)
	}
}

func TestFindDirProfile_SkipsCommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	writeDirProfile(t, dir, "\n# which account this repo deploys to\n\n  teachermade  \nignored-second-line\n")

	got, err := FindDirProfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Name != "teachermade" {
		t.Fatalf("got %+v, want 'teachermade'", got)
	}
}

func TestFindDirProfile_EmptyFileStopsSearch(t *testing.T) {
	root := t.TempDir()
	markRepoRoot(t, root)
	writeDirProfile(t, root, "tap")
	// An empty .vop is a deliberate "no default here" marker; it must not
	// fall through to the parent's.
	sub := filepath.Join(root, "scratch")
	writeDirProfile(t, sub, "# intentionally no default\n")

	got, err := FindDirProfile(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("got %+v, want nil", got)
	}
}

func TestFindDirProfile_RejectsMalformedName(t *testing.T) {
	dir := t.TempDir()
	writeDirProfile(t, dir, "tap extra junk\n")

	_, err := FindDirProfile(dir)
	if err == nil {
		t.Fatal("expected an error for a multi-token profile name")
	}
}

func TestFindDirProfile_IgnoresDirectoryNamedDotVop(t *testing.T) {
	root := t.TempDir()
	markRepoRoot(t, root)
	if err := os.MkdirAll(filepath.Join(root, DirProfileFilename), 0755); err != nil {
		t.Fatal(err)
	}
	got, err := FindDirProfile(root)
	if err != nil {
		t.Fatalf("a directory named .vop should be ignored, got error: %v", err)
	}
	if got != nil {
		t.Fatalf("got %+v, want nil", got)
	}
}

func TestFindDirProfile_TruncatesOversizedFile(t *testing.T) {
	dir := t.TempDir()
	// A huge file must not be slurped whole; the read is capped and the
	// first line still parses.
	big := "tap\n" + string(make([]byte, maxDirProfileSize*4))
	writeDirProfile(t, dir, big)

	got, err := FindDirProfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Name != "tap" {
		t.Fatalf("got %+v, want 'tap'", got)
	}
}
