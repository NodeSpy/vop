package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NodeSpy/vop/internal/ui"
	"github.com/NodeSpy/vop/internal/version"
	"github.com/spf13/cobra"
)

const installURL = "https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh"

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update vop to the latest version",
		RunE:  cmdUpdate,
	}
}

// isHomebrewInstall returns true if the running binary appears to be managed
// by Homebrew. It checks whether the executable path (resolved through
// symlinks) lives under a known Homebrew prefix.
func isHomebrewInstall() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}

	// Resolve symlinks — Homebrew symlinks from bin/ into Cellar/.
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}

	// Homebrew prefixes: /opt/homebrew (Apple Silicon), /usr/local/Cellar
	// (Intel Mac), or Linux homebrew at /home/linuxbrew/.linuxbrew.
	homebrewPrefixes := []string{
		"/opt/homebrew/",
		"/usr/local/Cellar/",
		"/home/linuxbrew/.linuxbrew/",
	}

	for _, prefix := range homebrewPrefixes {
		if strings.HasPrefix(resolved, prefix) {
			return true
		}
	}

	return false
}

func cmdUpdate(_ *cobra.Command, _ []string) error {
	fmt.Printf("Current version: %s\n\n", version.Full())

	if isHomebrewInstall() {
		return updateViaHomebrew()
	}
	return updateViaInstallScript()
}

func updateViaHomebrew() error {
	ui.Info("Detected Homebrew installation.")
	fmt.Println()

	if !ui.PromptYN("Update via Homebrew?", true) {
		ui.Info("Update cancelled.")
		return nil
	}

	fmt.Println()
	ui.Info("Running: brew upgrade vop")

	cmd := exec.Command("brew", "upgrade", "vop")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}

	fmt.Println()
	ui.Success("Update complete.")
	return nil
}

func updateViaInstallScript() error {
	if !ui.PromptYN("Download and install the latest version?", true) {
		ui.Info("Update cancelled.")
		return nil
	}

	fmt.Println()
	ui.Info("Downloading latest version...")

	cmd := exec.Command("bash", "-c", fmt.Sprintf("curl -fsSL %s | bash", installURL))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println()
	ui.Success("Update complete.")
	return nil
}
