package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NodeSpy/vop/internal/config"
)

// chdirWithDirProfile creates a temp repo containing a .vop file, chdirs
// into it for the duration of the test, and returns the repo path.
func chdirWithDirProfile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if content != "" {
		if err := os.WriteFile(filepath.Join(dir, config.DirProfileFilename), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	return dir
}

// clearProfileEnv unsets every profile-pinning variable for the test.
func clearProfileEnv(t *testing.T) {
	t.Helper()
	for _, key := range profileEnvVars {
		t.Setenv(key, "")
	}
}

func TestDefaultProfile_FromDirFile(t *testing.T) {
	clearProfileEnv(t)
	dir := chdirWithDirProfile(t, "tap\n")

	got := defaultProfile()
	if got.Name != "tap" {
		t.Errorf("name = %q, want %q", got.Name, "tap")
	}
	if got.Source != sourceDir {
		t.Errorf("source = %q, want %q", got.Source, sourceDir)
	}
	if !strings.HasPrefix(got.Origin, dir) {
		t.Errorf("origin = %q, want a path under %q", got.Origin, dir)
	}
}

func TestDefaultProfile_EnvBeatsDirFile(t *testing.T) {
	clearProfileEnv(t)
	chdirWithDirProfile(t, "tap\n")
	t.Setenv("AGENT_DECK_VOP_PROFILE", "ednition")

	got := defaultProfile()
	if got.Name != "ednition" {
		t.Errorf("name = %q, want the env-pinned profile", got.Name)
	}
	if got.Source != sourceEnv {
		t.Errorf("source = %q, want %q", got.Source, sourceEnv)
	}
}

func TestDefaultProfile_VopProfileBeatsAgentDeck(t *testing.T) {
	// Inside `vop shell tap` within an agent-deck session, the shell you
	// are actually standing in is the more specific answer.
	clearProfileEnv(t)
	chdirWithDirProfile(t, "")
	t.Setenv("VOP_PROFILE", "tap")
	t.Setenv("AGENT_DECK_VOP_PROFILE", "ednition")

	if got := defaultProfile(); got.Name != "tap" {
		t.Errorf("name = %q, want %q", got.Name, "tap")
	}
}

func TestDefaultProfile_NoneConfigured(t *testing.T) {
	clearProfileEnv(t)
	chdirWithDirProfile(t, "")

	if got := defaultProfile(); got.Name != "" {
		t.Errorf("name = %q, want empty", got.Name)
	}
}

func TestDefaultProfile_MalformedFileDoesNotBlock(t *testing.T) {
	// A broken marker file degrades to "no default", not to a hard error —
	// the user can still name a profile explicitly.
	clearProfileEnv(t)
	chdirWithDirProfile(t, "tap and some junk\n")

	if got := defaultProfile(); got.Name != "" {
		t.Errorf("name = %q, want empty for a malformed file", got.Name)
	}
}

func TestProfileOrDefault_ArgWins(t *testing.T) {
	clearProfileEnv(t)
	chdirWithDirProfile(t, "tap\n")
	t.Setenv("AGENT_DECK_VOP_PROFILE", "ednition")

	got := profileOrDefault([]string{"teachermade"})
	if got.Name != "teachermade" {
		t.Errorf("name = %q, want the explicit argument", got.Name)
	}
	if got.Source != sourceArg {
		t.Errorf("source = %q, want %q", got.Source, sourceArg)
	}
}

func TestRequireDefaultProfile_ErrorNamesTheOptions(t *testing.T) {
	clearProfileEnv(t)
	chdirWithDirProfile(t, "")

	_, err := requireDefaultProfile(nil, "exec")
	if err == nil {
		t.Fatal("expected an error when no profile can be resolved")
	}
	for _, want := range []string{"vop exec <profile>", config.DirProfileFilename} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q:\n%s", want, err)
		}
	}
}

func TestDisplayPath_ShortensHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home directory")
	}
	in := filepath.Join(home, "work", "repo", ".vop")
	want := filepath.Join("~", "work", "repo", ".vop")
	if got := displayPath(in); got != want {
		t.Errorf("displayPath(%q) = %q, want %q", in, got, want)
	}
	// A path that merely shares a prefix with $HOME must not be mangled.
	if got := displayPath(home + "-backup/x"); got != home+"-backup/x" {
		t.Errorf("displayPath mangled a sibling path: %q", got)
	}
}
