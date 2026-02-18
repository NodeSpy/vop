package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NodeSpy/vop/internal/creds"
	"github.com/spf13/cobra"
)

func newDumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "dump [profile] [format]",
		Short:             "Dump active session credentials (json|env|creds)",
		Args:              cobra.RangeArgs(0, 2),
		RunE:              cmdDump,
		ValidArgsFunction: completeProfiles,
	}
}

func cmdDump(_ *cobra.Command, args []string) error {
	profile := ""
	format := "json"

	if len(args) >= 1 {
		profile = args[0]
	}
	if len(args) >= 2 {
		format = args[1]
	}

	// If no profile given, try to detect from environment
	if profile == "" {
		credFile := os.Getenv("VOP_CRED_FILE")
		if credFile != "" {
			if _, err := os.Stat(credFile); err == nil {
				profile = os.Getenv("VOP_PROFILE")
			}
		}
	}

	if profile == "" {
		return fmt.Errorf("usage: vop dump [profile] [json|env|creds]")
	}

	dir := creds.RuntimeDir()
	base := filepath.Join(dir, profile)
	jsonFile := base + ".json"
	credFile := base + ".credentials"

	noSession := fmt.Errorf("no active session for '%s'. Start one with: vop shell %s", profile, profile)

	switch format {
	case "json":
		data, err := os.ReadFile(jsonFile)
		if err != nil {
			return noSession
		}
		fmt.Print(string(data))

	case "env":
		c, _, err := creds.ReadJSONFile(jsonFile)
		if err != nil {
			return noSession
		}
		fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", c.AccessKeyID)
		fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", c.SecretAccessKey)
		if c.SessionToken != "" {
			fmt.Printf("export AWS_SESSION_TOKEN=%s\n", c.SessionToken)
		}

	case "aws-credentials", "credentials", "creds":
		data, err := os.ReadFile(credFile)
		if err != nil {
			return noSession
		}
		fmt.Print(string(data))

	default:
		return fmt.Errorf("unknown format: %s. Use: json, env, or creds", format)
	}

	return nil
}
