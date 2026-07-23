package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/NodeSpy/vop/internal/awsclient"
	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newRotateCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "rotate [profile]",
		Short:             "Rotate AWS access keys",
		Args:              cobra.MaximumNArgs(1),
		RunE:              cmdRotate,
		ValidArgsFunction: completeProfiles,
	}
}

func cmdRotate(_ *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	profileName := ""
	if len(args) > 0 {
		profileName = args[0]
	}

	if profileName == "" {
		names := c.ProfileNames()
		if len(names) == 0 {
			return fmt.Errorf("no profiles configured. Run 'vop add' to create one")
		}
		if len(names) == 1 {
			profileName = names[0]
		} else {
			fmt.Println()
			selected, selErr := ui.Select("Profile to rotate", names)
			if selErr != nil {
				return selErr
			}
			profileName = selected
			fmt.Println()
		}
	}

	profile, err := requireProfile(c, profileName)
	if err != nil {
		return err
	}

	if profile.IsAssumedRole() {
		return fmt.Errorf("cannot rotate keys for assumed-role profile %q: key rotation must be performed on the source profile (%q)", profileName, profile.SourceProfile)
	}

	if profile.UsesCredentialsCommand() {
		return fmt.Errorf("cannot rotate keys for profile %q: base credentials come from `credentials_command` and vop can't write back to an arbitrary command. Rotate manually and update the source (e.g. `pass edit aws/prod`)", profileName)
	}

	client, err := getClientForProfile(profile)
	if err != nil {
		return err
	}

	// Authenticate through the normal credential flow (handles MFA/STS).
	sessionCreds, err := creds.Fetch(profile, profileName, client, c, opClientFor())
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Read the permanent (IAM) credentials — we need both to know what to
	// delete and to roll back if verification fails.
	oldAK, oldSK, _, err := creds.FetchBaseKeys(profile, client)
	if err != nil {
		return err
	}
	ui.Info("Current access key: %s", oldAK)

	storageLabel := "1Password"
	if profile.UsesAWSCredentialsFile() {
		storageLabel = fmt.Sprintf("~/.aws/credentials [%s]", profile.AWSCredentialsProfile)
	}

	// Confirm before proceeding.
	fmt.Println()
	ui.Warn("This will rotate the AWS access key for profile '%s'.", profileName)
	ui.Warn("A new key will be created, stored in %s, and the old key deleted.", storageLabel)
	fmt.Println()
	if !ui.PromptYN("Proceed with rotation?", false) {
		ui.Info("Rotation cancelled.")
		return nil
	}
	fmt.Println()

	// Use session credentials (MFA-authenticated) for IAM operations.
	sAK := sessionCreds.AccessKeyID
	sSK := sessionCreds.SecretAccessKey
	sST := sessionCreds.SessionToken

	// 1. Create new key using session credentials.
	ui.Info("Creating new access key...")
	var newKey *awsclient.AccessKey
	if sST != "" {
		newKey, err = awsclient.CreateAccessKeyWithSession(ctx, sAK, sSK, sST, profile.IAMUsername)
	} else {
		newKey, err = awsclient.CreateAccessKey(ctx, sAK, sSK, profile.IAMUsername)
	}
	if err != nil {
		return err
	}

	newAK := newKey.AccessKeyID
	newSK := newKey.SecretAccessKey
	if newAK == "" || newSK == "" {
		return fmt.Errorf("AWS returned empty credentials")
	}
	ui.Info("New access key: %s", newAK)

	// 2. Persist the new permanent key to whichever storage this profile uses.
	err = creds.WriteBaseKeys(profile, client, newAK, newSK)
	if err != nil {
		// Write failed — delete the orphaned AWS key so the user isn't
		// left with two keys and no record of the new secret.
		ui.Warn("Deleting orphaned AWS key %s (%s update failed)...", newAK, storageLabel)
		var delErr error
		if sST != "" {
			delErr = awsclient.DeleteAccessKeyWithSession(ctx, sAK, sSK, sST, newAK, profile.IAMUsername)
		} else {
			delErr = awsclient.DeleteAccessKey(ctx, sAK, sSK, newAK, profile.IAMUsername)
		}
		if delErr != nil {
			ui.Error("Failed to delete orphaned key %s — clean up manually in AWS console.", newAK)
		}
		return fmt.Errorf("failed to update %s: %w", storageLabel, err)
	}
	ui.Success("%s updated.", storageLabel)

	// 3. Verify new credentials. If MFA is configured, we need to get a
	// fresh STS session with the new key. We must wait for the TOTP to
	// roll to a new code — STS rejects reuse of the same code.
	var identity *awsclient.CallerIdentity
	var testErr error

	if profile.MFATOTPItem != "" || profile.MFATOTPCommand != "" {
		// Get the current TOTP so we know what to wait past.
		firstTOTP, _ := creds.FetchTOTP(profile, client)

		ui.Info("Waiting for new TOTP window and key propagation...")
		// Poll until the TOTP changes (new 30s window) or timeout after 45s.
		deadline := time.Now().Add(45 * time.Second)
		var freshTOTP string
		for time.Now().Before(deadline) {
			time.Sleep(3 * time.Second)
			freshTOTP, _ = creds.FetchTOTP(profile, client)
			if freshTOTP != "" && freshTOTP != firstTOTP {
				break
			}
		}
		if freshTOTP == "" || freshTOTP == firstTOTP {
			testErr = fmt.Errorf("timed out waiting for TOTP to change")
		} else {
			// For MFA serial, prefer explicit config, then the AWS
			// credentials/config file, then any 1P field, else fall back
			// to iam:ListMFADevices via session creds (the new key may not
			// yet be MFA-authorized without a session).
			serialNumber := profile.MFASerial
			if serialNumber == "" && profile.UsesAWSCredentialsFile() {
				credsPath := creds.ResolveAWSCredentialsPath(profile.AWSCredentialsFile)
				serialNumber, _ = creds.LookupAWSMFASerial(creds.DefaultAWSConfigFile(), credsPath, profile.AWSCredentialsProfile)
			}
			if serialNumber == "" && profile.OPItem != "" && !profile.UsesAWSCredentialsFile() {
				serialNumber, _ = client.ReadField(profile.OPAccount, profile.OPItem, profile.FieldName("mfa serial"))
			}
			if serialNumber == "" {
				if sST != "" {
					serialNumber, _ = awsclient.ListMFADevices(ctx, sAK, sSK, profile.IAMUsername)
				} else {
					serialNumber, _ = awsclient.ListMFADevices(ctx, newAK, newSK, profile.IAMUsername)
				}
			}
			if serialNumber == "" {
				testErr = fmt.Errorf("could not determine MFA serial for verification")
			} else {
				ui.Info("Verifying new key with fresh TOTP...")
				newSession, stsErr := awsclient.GetSessionToken(ctx, newAK, newSK, serialNumber, freshTOTP)
				if stsErr != nil {
					testErr = fmt.Errorf("STS verification failed: %w", stsErr)
				} else {
					identity, testErr = awsclient.GetCallerIdentityWithSession(
						ctx, newSession.AccessKeyID, newSession.SecretAccessKey, newSession.SessionToken,
					)
				}
			}
		}
	} else {
		ui.Info("Waiting for propagation...")
		time.Sleep(5 * time.Second)
		identity, testErr = awsclient.GetCallerIdentity(ctx, newAK, newSK)
	}

	if testErr != nil {
		// ROLLBACK: restore old key in storage, delete the new AWS key.
		ui.Error("New credentials failed verification: %s", testErr)
		ui.Warn("Rolling back %s...", storageLabel)

		_ = creds.WriteBaseKeys(profile, client, oldAK, oldSK)

		ui.Warn("Deleting failed key: %s", newAK)
		if sST != "" {
			_ = awsclient.DeleteAccessKeyWithSession(ctx, sAK, sSK, sST, newAK, profile.IAMUsername)
		} else {
			_ = awsclient.DeleteAccessKey(ctx, sAK, sSK, newAK, profile.IAMUsername)
		}

		return fmt.Errorf("rotation aborted. Old credentials restored")
	}

	ui.Success("New credentials verified:")
	fmt.Printf("    Account: %s\n", identity.Account)
	fmt.Printf("    ARN:     %s\n", identity.Arn)

	// 4. Delete old key using session credentials.
	ui.Info("Deleting old access key: %s", oldAK)
	if sST != "" {
		err = awsclient.DeleteAccessKeyWithSession(ctx, sAK, sSK, sST, oldAK, profile.IAMUsername)
	} else {
		err = awsclient.DeleteAccessKey(ctx, sAK, sSK, oldAK, profile.IAMUsername)
	}
	if err != nil {
		ui.Warn("Failed to delete old key %s — clean up manually.", oldAK)
	}

	fmt.Println()
	fmt.Printf("%s%sRotation complete for '%s'.%s\n", ui.Green, ui.Bold, profileName, ui.Reset)
	fmt.Printf("  Old: %s%s%s (deleted)\n", ui.Red, oldAK, ui.Reset)
	fmt.Printf("  New: %s%s%s (active)\n\n", ui.Green, newAK, ui.Reset)
	return nil
}
