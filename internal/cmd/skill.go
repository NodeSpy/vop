package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/NodeSpy/vop/internal/skill"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/NodeSpy/vop/internal/version"
	"github.com/spf13/cobra"
)

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Print the agent instructions for using vop",
		Long: `Print the instructions an AI agent should follow when using vop.

The text comes from this binary, so it always describes the vop you are actually
running, and it names the profile that resolves in the current directory.

'vop skill install' writes a short stub into an agent's skills directory. The
stub only tells the agent to run 'vop skill' for the real instructions, so it
does not go out of date when vop is upgraded.`,
		Args: cobra.NoArgs,
		RunE: cmdSkill,
	}

	cmd.AddCommand(newSkillInstallCmd())
	cmd.AddCommand(newSkillStatusCmd())
	cmd.AddCommand(newSkillStubCmd())

	return cmd
}

func cmdSkill(_ *cobra.Command, _ []string) error {
	fmt.Print(skill.Guide(skill.Vars{
		Version:       version.String(),
		ProfileStatus: profileStatusLine(),
		ExitCooldown:  strconv.Itoa(ExitCooldown),
		ExitLocked:    strconv.Itoa(ExitLocked),
	}))
	return nil
}

// profileStatusLine describes what a profile-less vop command would use here.
// Resolution is cheap and touches no credential source, so the guide can state
// it as fact rather than telling the agent how to work it out.
func profileStatusLine() string {
	r := defaultProfile()
	if r.Name == "" {
		return "none — name one explicitly, or ask the user which to use (`vop ls`)"
	}
	return fmt.Sprintf("`%s` (from %s)", r.Name, r.Describe())
}

func newSkillStubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stub",
		Short: "Print the stub that 'skill install' writes",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Print(skill.Stub())
			return nil
		},
	}
}

func newSkillInstallCmd() *cobra.Command {
	var project bool
	var dir string
	var force bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the vop skill stub for an AI agent",
		Long: `Write the vop skill stub to an agent's skills directory.

Defaults to the user-level Claude Code directory (~/.claude/skills/vop, or
$CLAUDE_CONFIG_DIR if set). Use --project for the current project's
.claude/skills, or --dir for anywhere else.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			target, err := skillTargetDir(project, dir)
			if err != nil {
				return err
			}
			path := filepath.Join(target, skill.InstalledName)

			// Refuse to overwrite something we didn't write. An unrecognized
			// file here is a hand-written skill, and clobbering it silently is
			// how you lose someone's work.
			if existing, err := os.ReadFile(path); err == nil {
				if _, ok := skill.StubVersionOf(string(existing)); !ok && !force {
					return fmt.Errorf("%s already exists and was not installed by vop.\n"+
						"  Inspect it, then re-run with --force to replace it", displayPath(path))
				}
			}

			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(skill.Stub()), 0644); err != nil {
				return err
			}

			ui.Success("Installed vop skill: %s", displayPath(path))
			ui.Info("The stub points agents at 'vop skill' for the full instructions,")
			ui.Info("so it stays correct across vop upgrades.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&project, "project", false, "install into ./.claude/skills instead of the user directory")
	cmd.Flags().StringVar(&dir, "dir", "", "install into this skills directory (a vop/ subdirectory is created)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite a SKILL.md vop didn't install")

	return cmd
}

func newSkillStatusCmd() *cobra.Command {
	var project bool
	var dir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report whether the vop skill stub is installed and current",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			target, err := skillTargetDir(project, dir)
			if err != nil {
				return err
			}
			path := filepath.Join(target, skill.InstalledName)

			state, _ := skillState(path)
			fmt.Printf("%s\t%s\n", state, displayPath(path))
			return nil
		},
	}

	cmd.Flags().BoolVar(&project, "project", false, "check ./.claude/skills instead of the user directory")
	cmd.Flags().StringVar(&dir, "dir", "", "check this skills directory")

	return cmd
}

// skillState classifies an installed stub. The returned hint is empty unless
// there is something to do about it.
func skillState(path string) (state, hint string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "missing", "run: vop skill install"
	}
	installed, ok := skill.StubVersionOf(string(content))
	switch {
	case !ok:
		return "unmanaged", "not installed by vop — inspect it before replacing"
	case installed < skill.StubVersion:
		return "outdated", "run: vop skill install"
	default:
		return "current", ""
	}
}

// skillTargetDir resolves where the stub lives. Claude Code reads
// $CLAUDE_CONFIG_DIR when set, so honour it rather than hardcoding ~/.claude.
func skillTargetDir(project bool, dir string) (string, error) {
	switch {
	case dir != "":
		return filepath.Join(dir, "vop"), nil
	case project:
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".claude", "skills", "vop"), nil
	}

	base := os.Getenv("CLAUDE_CONFIG_DIR")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot locate your home directory: %w", err)
		}
		base = filepath.Join(home, ".claude")
	}
	return filepath.Join(base, "skills", "vop"), nil
}
