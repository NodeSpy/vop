package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

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

	// --- Section 1: Human context (status + paths) ---
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

	// --- Section 2: Copy block ---
	instructions := config.DefaultAgentInstructions
	if profile != nil {
		instructions = profile.AgentInstructions()
	}

	var block strings.Builder
	block.WriteString("## AWS Credentials (vop)\n")
	block.WriteString("\n")
	block.WriteString("This project uses vop-managed AWS credentials.\n")
	block.WriteString("The following environment variables are set in the current shell:\n")
	fmt.Fprintf(&block, "  AWS_SHARED_CREDENTIALS_FILE=%s\n", credFile)
	fmt.Fprintf(&block, "  VOP_CRED_FILE=%s\n", jsonFile)
	block.WriteString("\n")
	block.WriteString("Use AWS_SHARED_CREDENTIALS_FILE for AWS CLI/SDK operations.\n")
	block.WriteString("Run `vop refresh` if credentials have expired.\n")
	block.WriteString("\n")
	block.WriteString(wrapText(instructions, 64, ""))

	printCopyBlock(block.String())
	fmt.Println()

	// --- Section 3: Human context (policy info + credential_process) ---
	policyLabel := "default (read-only)"
	if profile != nil && profile.AgentPolicy != "" {
		policyLabel = "custom"
	}
	fmt.Printf("  %sAgent policy:%s %s\n", ui.Bold, ui.Reset, policyLabel)
	fmt.Printf("  %sChange with:%s  vop edit %s\n", ui.Dim, ui.Reset, profileName)
	fmt.Println()

	fmt.Printf("  %sAWS config (credential_process):%s\n", ui.Bold, ui.Reset)
	fmt.Printf("    [profile %s]\n", profileName)
	fmt.Printf("    credential_process = vop cred-process %s\n", profileName)
	fmt.Println()

	return nil
}

// printCopyBlock prints content inside a clearly demarcated box for copy-pasting.
// The border lines are ANSI-colored; the content between them is plain text.
func printCopyBlock(content string) {
	label := " Copy into your AI agent config (e.g. CLAUDE.md) "
	topBorder := "┌─" + label + strings.Repeat("─", 12) + "┐"
	borderWidth := utf8.RuneCountInString(topBorder)
	botBorder := "└" + strings.Repeat("─", borderWidth-2) + "┘"

	fmt.Printf("  %s%s%s%s\n", ui.Cyan, ui.Bold, topBorder, ui.Reset)
	fmt.Println()
	for _, line := range strings.Split(content, "\n") {
		fmt.Printf("  %s\n", line)
	}
	fmt.Println()
	fmt.Printf("  %s%s%s%s\n", ui.Cyan, ui.Bold, botBorder, ui.Reset)
}

// wrapText wraps text at word boundaries to fit within width characters per line,
// prefixing each line with indent. Preserves existing newlines.
func wrapText(text string, width int, indent string) string {
	var result strings.Builder
	for i, paragraph := range strings.Split(text, "\n") {
		if i > 0 {
			result.WriteByte('\n')
		}
		if paragraph == "" {
			result.WriteString(indent)
			continue
		}
		words := strings.Fields(paragraph)
		lineLen := 0
		for j, word := range words {
			wLen := utf8.RuneCountInString(word)
			if j == 0 {
				result.WriteString(indent)
				result.WriteString(word)
				lineLen = utf8.RuneCountInString(indent) + wLen
			} else if lineLen+1+wLen > width {
				result.WriteByte('\n')
				result.WriteString(indent)
				result.WriteString(word)
				lineLen = utf8.RuneCountInString(indent) + wLen
			} else {
				result.WriteByte(' ')
				result.WriteString(word)
				lineLen += 1 + wLen
			}
		}
	}
	return result.String()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
