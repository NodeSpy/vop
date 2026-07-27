package cmd

import (
	"context"
	"fmt"

	"github.com/NodeSpy/vop/internal/awsclient"
	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "test [profile]",
		Short:             "Test credential source + AWS connectivity",
		Args:              cobra.MaximumNArgs(1),
		RunE:              cmdTest,
		ValidArgsFunction: completeProfiles,
	}
}

func cmdTest(_ *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	resolved, err := requireDefaultProfile(args, "test")
	if err != nil {
		return err
	}
	name, profile, err := resolveProfile(c, resolved.Name)
	if err != nil {
		return err
	}

	client, err := getClientForProfile(profile)
	if err != nil {
		return err
	}

	fmt.Println()
	ui.Info("Testing profile: %s", name)
	ui.Info("Backend: %s", describeBackend(profile))

	// 1Password sign-in — only when this profile actually touches 1P.
	if profile.UsesOnePassword() {
		if err := client.EnsureSignedIn(profile.OPAccount); err != nil {
			return fmt.Errorf("sign-in failed for %s: %w", profile.OPAccount, err)
		}
		ui.Success("Signed into 1Password: %s", profile.OPAccount)
	}

	// Base credentials, from whichever source the profile is configured with.
	ak, sk, sessionToken, err := creds.FetchBaseKeys(profile, client)
	if err != nil {
		ui.Error("Cannot read credentials for profile '%s'", name)
		ui.Warn("Check the credential source: %s", describeBackend(profile))
		return err
	}
	ui.Success("Can read credentials from: %s", describeSource(profile))

	// MFA — via mfa_totp_command or a 1P TOTP item, whichever is configured.
	if profile.MFATOTPItem != "" || profile.MFATOTPCommand != "" {
		if _, err := creds.FetchTOTP(profile, client); err != nil {
			ui.Error("Cannot get TOTP from %s", describeTOTPSource(profile))
			ui.Warn("Check the TOTP source configuration.")
		} else {
			ui.Success("Can read TOTP from: %s", describeTOTPSource(profile))
		}
	}

	// AWS credentials. sts:GetCallerIdentity requires no permissions, so the
	// base keys are enough to prove they are valid and unexpired even when
	// the account requires MFA for everything else.
	ui.Info("Testing AWS credentials...")
	var identity *awsclient.CallerIdentity
	if sessionToken != "" {
		identity, err = awsclient.GetCallerIdentityWithSession(context.Background(), ak, sk, sessionToken)
	} else {
		identity, err = awsclient.GetCallerIdentity(context.Background(), ak, sk)
	}
	if err != nil {
		ui.Error("AWS credentials failed")
		ui.Warn("Credentials from %s may be expired or invalid.", describeSource(profile))
		return err
	}

	ui.Success("AWS credentials valid:")
	fmt.Printf("    Account: %s\n", identity.Account)
	fmt.Printf("    ARN:     %s\n", identity.Arn)
	fmt.Println()

	return nil
}

// describeBackend names the machinery a profile uses to fetch its base keys.
func describeBackend(p *config.Profile) string {
	switch {
	case p.UsesCredentialsCommand():
		return "credentials_command"
	case p.UsesAWSCredentialsFile():
		return "AWS shared credentials file"
	case p.UsesSDK():
		return "1Password SDK (service account)"
	default:
		return "op CLI"
	}
}

// describeSource names the concrete place the base keys come from.
func describeSource(p *config.Profile) string {
	switch {
	case p.UsesCredentialsCommand():
		return p.CredentialsCommand
	case p.UsesAWSCredentialsFile():
		return fmt.Sprintf("%s [%s]",
			creds.ResolveAWSCredentialsPath(p.AWSCredentialsFile), p.AWSCredentialsProfile)
	default:
		return p.OPItem
	}
}

// describeTOTPSource names where the MFA code comes from, mirroring the
// precedence in creds.FetchTOTP.
func describeTOTPSource(p *config.Profile) string {
	if p.MFATOTPCommand != "" {
		return p.MFATOTPCommand
	}
	return p.MFATOTPItem
}
