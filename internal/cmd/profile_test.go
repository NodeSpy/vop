package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/NodeSpy/vop/internal/config"
)

// runProfileCmd runs `vop profile` with the given flags and returns stdout.
// Only stdout is captured: callers of this command substitute it, so anything
// that leaks onto stdout beyond the bare name is a bug.
func runProfileCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	cmd := newProfileCmd()
	cmd.SetArgs(args)
	runErr := cmd.Execute()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), runErr
}

func TestProfileCmd_PrintsOnlyTheName(t *testing.T) {
	clearProfileEnv(t)
	chdirWithDirProfile(t, "tap\n")
	setupTestConfig(t, map[string]*config.Profile{"tap": {OPAccount: "acct"}})

	out, err := runProfileCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "tap\n" {
		t.Errorf("stdout = %q, want %q", out, "tap\n")
	}
}

func TestProfileCmd_Export(t *testing.T) {
	clearProfileEnv(t)
	chdirWithDirProfile(t, "tap\n")
	setupTestConfig(t, map[string]*config.Profile{"tap": {OPAccount: "acct"}})

	out, err := runProfileCmd(t, "--export")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "export VOP_PROFILE=tap\n" {
		t.Errorf("stdout = %q, want an export statement", out)
	}
}

func TestProfileCmd_EnvWins(t *testing.T) {
	clearProfileEnv(t)
	chdirWithDirProfile(t, "tap\n")
	t.Setenv("AGENT_DECK_VOP_PROFILE", "ednition")
	setupTestConfig(t, map[string]*config.Profile{
		"tap":      {OPAccount: "acct"},
		"ednition": {OPAccount: "acct"},
	})

	out, err := runProfileCmd(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ednition\n" {
		t.Errorf("stdout = %q, want %q", out, "ednition\n")
	}
}

func TestProfileCmd_NoneResolved(t *testing.T) {
	clearProfileEnv(t)
	chdirWithDirProfile(t, "")
	setupTestConfig(t, nil)

	out, err := runProfileCmd(t)
	if err == nil {
		t.Fatal("expected an error when no profile resolves")
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty", out)
	}
	for _, want := range []string{"VOP_PROFILE", ".vop"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}
