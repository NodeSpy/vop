package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/NodeSpy/vop/internal/skill"
)

// skillTexts returns the two documents that describe vop to an agent.
func skillTexts(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{
		"guide": skill.Guide(skill.Vars{}),
		"stub":  skill.Stub(),
	}
}

// codeSpans extracts the contents of every inline code span and fenced code
// block. Only these are checked for command names — prose says things like "vop
// fetches credentials", and a bare word-after-vop scan would flag the English.
func codeSpans(text string) []string {
	var spans []string

	fence := regexp.MustCompile("(?s)```[a-z]*\n(.*?)```")
	for _, m := range fence.FindAllStringSubmatch(text, -1) {
		spans = append(spans, strings.Split(m[1], "\n")...)
	}

	withoutFences := fence.ReplaceAllString(text, "")
	inline := regexp.MustCompile("`([^`\n]+)`")
	for _, m := range inline.FindAllStringSubmatch(withoutFences, -1) {
		spans = append(spans, m[1])
	}
	return spans
}

// TestSkillDocs_OnlyReferenceRealCommands is the guard that keeps the agent
// instructions honest: removing or renaming a subcommand fails here until the
// docs are updated, instead of silently telling agents to run something that
// no longer exists.
func TestSkillDocs_OnlyReferenceRealCommands(t *testing.T) {
	root := NewRootCmd()
	known := map[string]bool{}
	for _, c := range root.Commands() {
		known[c.Name()] = true
		for _, sub := range c.Commands() {
			known[c.Name()+" "+sub.Name()] = true
		}
	}

	// `vop <profile>` is the documented shell shorthand, so a placeholder in
	// angle brackets or an example profile name is not a command reference.
	placeholder := regexp.MustCompile(`^<.+>$`)
	exampleProfiles := map[string]bool{"tap": true, "prod": true, "ednition": true}

	ref := regexp.MustCompile(`\bvop ([a-z][a-z-]*|<[a-z]+>)( [a-z][a-z-]*)?`)

	for name, text := range skillTexts(t) {
		for _, span := range codeSpans(text) {
			for _, m := range ref.FindAllStringSubmatch(span, -1) {
				first := m[1]
				if placeholder.MatchString(first) || exampleProfiles[first] || strings.HasPrefix(first, "-") {
					continue
				}
				pair := first + strings.TrimRight(m[2], " ")
				if known[strings.TrimSpace(pair)] || known[first] {
					continue
				}
				t.Errorf("%s.md references `vop %s`, which is not a registered command", name, first)
			}
		}
	}
}

// TestSkillStub_ExitCodesMatchConstants covers the one place the exit codes are
// hardcoded. The guide interpolates them from these same constants, so it can't
// drift; the stub spells them out and can.
func TestSkillStub_ExitCodesMatchConstants(t *testing.T) {
	stub := skill.Stub()
	for _, want := range []string{
		fmt.Sprintf("**Exit %d**", ExitCooldown),
		fmt.Sprintf("**Exit %d**", ExitLocked),
	} {
		if !strings.Contains(stub, want) {
			t.Errorf("stub.md should document %q — exit code constants changed?", want)
		}
	}
}

// TestSkillGuide_DocumentsEveryProfileEnvVar keeps the pinning rules complete:
// a new profile-pinning variable that the guide doesn't mention is a variable
// agents will ignore.
func TestSkillGuide_DocumentsEveryProfileEnvVar(t *testing.T) {
	guide := skill.Guide(skill.Vars{})
	for _, key := range profileEnvVars {
		if !strings.Contains(guide, key) {
			t.Errorf("guide.md does not mention %s, which pins the active profile", key)
		}
	}
}

func TestSkillGuide_InterpolatesLiveValues(t *testing.T) {
	guide := skill.Guide(skill.Vars{Version: "v9.9.9", ProfileStatus: "`tap` (from ./.vop)"})
	if strings.Contains(guide, "{{") {
		t.Error("guide.md has an unsubstituted placeholder")
	}
	for _, want := range []string{"v9.9.9", "`tap` (from ./.vop)"} {
		if !strings.Contains(guide, want) {
			t.Errorf("guide should contain %q", want)
		}
	}
}

func TestSkillInstall_WritesStubAndReportsStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(dir, ".claude"))

	path := filepath.Join(dir, ".claude", "skills", "vop", "SKILL.md")
	if state, _ := skillState(path); state != "missing" {
		t.Errorf("state before install = %q, want missing", state)
	}

	root := NewRootCmd()
	root.SetArgs([]string{"skill", "install"})
	if err := root.Execute(); err != nil {
		t.Fatalf("install: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("stub not installed: %v", err)
	}
	if !strings.HasPrefix(string(content), "---\nname: vop\n") {
		t.Error("installed stub should start with skill frontmatter")
	}
	if state, _ := skillState(path); state != "current" {
		t.Errorf("state after install = %q, want current", state)
	}
}

func TestSkillInstall_RefusesToClobberForeignFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(dir, ".claude"))
	target := filepath.Join(dir, ".claude", "skills", "vop")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	handWritten := "---\nname: vop\n---\n\nmy own notes\n"
	path := filepath.Join(target, "SKILL.md")
	if err := os.WriteFile(path, []byte(handWritten), 0644); err != nil {
		t.Fatal(err)
	}

	root := NewRootCmd()
	root.SetArgs([]string{"skill", "install"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected install to refuse an unmanaged SKILL.md")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should point at --force, got: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != handWritten {
		t.Error("the existing file was modified despite the refusal")
	}

	root = NewRootCmd()
	root.SetArgs([]string{"skill", "install", "--force"})
	if err := root.Execute(); err != nil {
		t.Fatalf("--force install: %v", err)
	}
	if state, _ := skillState(path); state != "current" {
		t.Errorf("state after --force = %q, want current", state)
	}
}

func TestSkillStubVersionOf(t *testing.T) {
	if v, ok := skill.StubVersionOf(skill.Stub()); !ok || v != skill.StubVersion {
		t.Errorf("StubVersionOf(own stub) = (%d, %v), want (%d, true)", v, ok, skill.StubVersion)
	}
	if _, ok := skill.StubVersionOf("some other skill file"); ok {
		t.Error("StubVersionOf should not claim a foreign file")
	}
}
