package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/credserver"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newCredProcessCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cred-process <profile>",
		Short: "Output credentials for AWS credential_process",
		Long: `Output AWS credentials in the format expected by credential_process.

This command is intended to be used in ~/.aws/config:

  [profile myprofile]
  credential_process = vop cred-process myprofile

The AWS SDK will call this command automatically when credentials are
needed and will cache them until the Expiration time. When they expire,
the SDK calls vop again to get fresh credentials -- no manual refresh
needed.

If a vop credential server is running, credentials are fetched from it
(fast, no 1Password interaction needed). Otherwise, credentials are
fetched directly from 1Password.`,
		Args:              cobra.ExactArgs(1),
		RunE:              cmdCredProcess,
		ValidArgsFunction: completeProfiles,
	}
}

// credProcessOutput matches the AWS credential_process JSON contract.
// See: https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
type credProcessOutput struct {
	Version         int    `json:"Version"`
	AccessKeyId     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken,omitempty"`
	Expiration      string `json:"Expiration,omitempty"`
}

func cmdCredProcess(_ *cobra.Command, args []string) error {
	profileName := args[0]

	// Suppress all informational output -- credential_process must only
	// write the JSON object to stdout.
	ui.Quiet = true
	defer func() { ui.Quiet = false }()

	// Try the credential server first (fast path, no 1Password needed).
	if srvClient := credserver.NewClient(); srvClient != nil {
		if awsCreds, err := srvClient.FetchCreds(profileName); err == nil {
			return emitCredProcessJSON(awsCreds)
		}
	}

	// Fall back to direct 1Password fetch.
	c, err := loadConfig()
	if err != nil {
		return err
	}

	profile, err := requireProfile(c, profileName)
	if err != nil {
		return err
	}

	opClient, err := getClientForProfile(profile)
	if err != nil {
		return err
	}

	awsCreds, err := creds.Fetch(profile, profileName, opClient, c, opClientFor())
	if err != nil {
		return err
	}

	// Push to server as side effect so future calls hit the fast path.
	pushToServer(profileName, awsCreds)

	return emitCredProcessJSON(awsCreds)
}

func emitCredProcessJSON(awsCreds *creds.AWSCredentials) error {
	output := credProcessOutput{
		Version:         1,
		AccessKeyId:     awsCreds.AccessKeyID,
		SecretAccessKey: awsCreds.SecretAccessKey,
		SessionToken:    awsCreds.SessionToken,
		Expiration:      awsCreds.Expiration,
	}

	data, err := json.Marshal(output)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}
