package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/skill"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check prerequisites",
		RunE:  cmdCheck,
	}
}

func cmdCheck(_ *cobra.Command, _ []string) error {
	fmt.Printf("\n%s  Prerequisites%s\n\n", ui.Bold, ui.Reset)

	opMissing := false

	// op CLI is only required for profiles using CLI backend (no service account token).
	// Profiles using SDK backend work without the op binary.
	opPath, opErr := exec.LookPath("op")
	if opErr != nil {
		opMissing = true
		ui.Warn("op  not found (needed for CLI-backend profiles)")
		ui.Warn("  Install: https://developer.1password.com/docs/cli/get-started/")
	} else {
		ui.Success("op  %s", opPath)
	}

	fmt.Printf("\n%s  Configuration%s\n\n", ui.Bold, ui.Reset)
	cfgPath := configFilePath()
	fmt.Printf("  Dir:  %s\n", config.DefaultConfigDir())
	fmt.Printf("  File: %s\n\n", cfgPath)

	hasCLIProfiles := false
	c, err := config.Load(cfgPath)
	if err != nil {
		ui.Warn("Config not found — run 'vop add <name>' to get started.")
	} else {
		ui.Success("Config file exists")
		names := c.ProfileNames()
		ui.Info("%d profile(s) configured", len(names))
		for _, name := range names {
			if p, ok := c.Profiles[name]; ok && !p.UsesSDK() {
				hasCLIProfiles = true
				break
			}
		}
		if len(names) > 0 {
			_ = cmdLs(nil, nil)
		}
	}

	if !opMissing {
		client := getCLIClient()
		if client.IsInstalled() {
			fmt.Printf("%s  1Password Accounts%s\n\n", ui.Bold, ui.Reset)
			accounts, err := client.ListAccounts()
			if err != nil {
				ui.Warn("Cannot list 1Password accounts (not signed in)")
			} else {
				for _, a := range accounts {
					fmt.Printf("  %s  (%s)\n", a.URL, a.Email)
				}
			}
			fmt.Println()
		}
	}

	reportSkillState()

	if opMissing && hasCLIProfiles {
		ui.Error("op CLI is required for CLI-backend profiles — see above.")
	} else {
		ui.Success("All prerequisites met.")
	}
	fmt.Println()
	return nil
}

// reportSkillState tells the user whether AI agents on this machine know how to
// use vop. `vop check` is the one command every install path points at, so this
// is where the skill becomes discoverable regardless of how vop was installed.
func reportSkillState() {
	fmt.Printf("%s  Agent skill%s\n\n", ui.Bold, ui.Reset)

	target, err := skillTargetDir(false, "")
	if err != nil {
		ui.Warn("Cannot locate the agent skills directory: %s", err)
		fmt.Println()
		return
	}
	path := filepath.Join(target, skill.InstalledName)
	state, hint := skillState(path)

	switch state {
	case "current":
		ui.Success("Installed: %s", displayPath(path))
	default:
		ui.Warn("%s: %s", state, displayPath(path))
	}
	if hint != "" {
		ui.Info("%s", hint)
	}
	fmt.Println()
}
