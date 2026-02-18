package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent",
		Short: "Show credential paths for AI agents and external tools",
		Long: `Show the credential file paths and environment variables for the
active vop session. Use this to tell an AI agent or external tool
where to find AWS credentials.

If not inside a vop shell, shows what the paths would be for a
given profile.`,
		Args:              cobra.MaximumNArgs(1),
		RunE:              cmdAgent,
		ValidArgsFunction: completeProfiles,
	}
}

func cmdAgent(_ *cobra.Command, args []string) error {
	profileName := ""
	if len(args) > 0 {
		profileName = args[0]
	}
	if profileName == "" {
		profileName = os.Getenv("VOP_PROFILE")
	}
	if profileName == "" {
		return fmt.Errorf("not in a vop shell (VOP_PROFILE not set).\n  Specify a profile: vop agent <profile>")
	}

	// Load profile config for agent policy.
	c, err := loadConfig()
	if err != nil {
		return err
	}
	profile := c.Profiles[profileName]

	dir := creds.RuntimeDir()
	base := filepath.Join(dir, profileName)
	credFile := base + ".credentials"
	jsonFile := base + ".json"

	// Check if credential files actually exist
	credExists := fileExists(credFile)
	jsonExists := fileExists(jsonFile)

	fmt.Println()

	if credExists || jsonExists {
		ui.Success("Active vop session: %s", profileName)
	} else {
		ui.Warn("No active credential files for '%s'.", profileName)
		ui.Info("Start a session with: vop shell %s", profileName)
		fmt.Println()
		ui.Info("Showing expected paths:")
	}

	fmt.Println()
	fmt.Printf("  %sProfile:%s          %s\n", ui.Dim, ui.Reset, profileName)
	fmt.Printf("  %sCredentials file:%s %s\n", ui.Dim, ui.Reset, credFile)
	fmt.Printf("  %sJSON file:%s        %s\n", ui.Dim, ui.Reset, jsonFile)
	fmt.Println()
	fmt.Printf("  %sEnvironment variables set in vop shell:%s\n", ui.Bold, ui.Reset)
	fmt.Printf("    AWS_SHARED_CREDENTIALS_FILE=%s\n", credFile)
	fmt.Printf("    VOP_CRED_FILE=%s\n", jsonFile)
	fmt.Printf("    VOP_PROFILE=%s\n", profileName)
	fmt.Println()
	fmt.Printf("  %sFor AI agents / external tools:%s\n", ui.Bold, ui.Reset)
	fmt.Printf("    Use AWS_SHARED_CREDENTIALS_FILE for AWS CLI/SDK tools.\n")
	fmt.Printf("    Use VOP_CRED_FILE for the JSON credential file.\n")
	fmt.Printf("    Run 'vop refresh' if credentials have expired.\n")
	fmt.Println()

	// Show agent policy instructions.
	policyLabel := "readonly"
	if profile != nil && profile.AgentPolicy != "" {
		policyLabel = profile.AgentPolicy
	}
	instructions := ""
	if profile != nil {
		instructions = profile.AgentInstructions()
	} else {
		// No profile config available; use default.
		defaultProfile := &config.Profile{}
		instructions = defaultProfile.AgentInstructions()
	}
	fmt.Printf("  %sAgent policy:%s %s\n", ui.Bold, ui.Reset, policyLabel)
	fmt.Printf("    %s\n", instructions)
	fmt.Println()
	fmt.Printf("  %sChange with:%s vop edit %s\n", ui.Dim, ui.Reset, profileName)
	fmt.Println()

	fmt.Printf("  %sAWS config (credential_process):%s\n", ui.Bold, ui.Reset)
	fmt.Printf("    [profile %s]\n", profileName)
	fmt.Printf("    credential_process = vop cred-process %s\n", profileName)
	fmt.Println()

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
