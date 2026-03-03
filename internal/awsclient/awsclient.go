// Package awsclient wraps AWS SDK v2 operations used by vop,
// replacing the previous dependency on the aws CLI binary.
package awsclient

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// stsAPI defines the STS operations we use (for testing).
type stsAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
	GetSessionToken(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error)
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

// iamAPI defines the IAM operations we use (for testing).
type iamAPI interface {
	ListMFADevices(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error)
	CreateAccessKey(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
	DeleteAccessKey(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
}

// CallerIdentity holds the result of sts:GetCallerIdentity.
type CallerIdentity struct {
	Account string
	Arn     string
	UserID  string
}

// SessionCredentials holds temporary STS credentials.
type SessionCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      string
}

// AccessKey holds a newly created IAM access key.
type AccessKey struct {
	AccessKeyID     string
	SecretAccessKey string
}

// staticConfig returns an aws.Config using static credentials.
func staticConfig(accessKey, secretKey string) aws.Config {
	return aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}
}

// sessionConfig returns an aws.Config using temporary session credentials.
func sessionConfig(accessKey, secretKey, sessionToken string) aws.Config {
	return aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken),
	}
}

// GetCallerIdentity calls sts:GetCallerIdentity to verify credentials.
func GetCallerIdentity(ctx context.Context, accessKey, secretKey string) (*CallerIdentity, error) {
	cfg := staticConfig(accessKey, secretKey)
	client := sts.NewFromConfig(cfg)
	return getCallerIdentity(ctx, client)
}

// GetCallerIdentityWithSession is like GetCallerIdentity but uses temporary session credentials.
func GetCallerIdentityWithSession(ctx context.Context, accessKey, secretKey, sessionToken string) (*CallerIdentity, error) {
	cfg := sessionConfig(accessKey, secretKey, sessionToken)
	client := sts.NewFromConfig(cfg)
	return getCallerIdentity(ctx, client)
}

func getCallerIdentity(ctx context.Context, client stsAPI) (*CallerIdentity, error) {
	out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("sts:GetCallerIdentity failed: %w", err)
	}

	return &CallerIdentity{
		Account: aws.ToString(out.Account),
		Arn:     aws.ToString(out.Arn),
		UserID:  aws.ToString(out.UserId),
	}, nil
}

// GetSessionToken calls sts:GetSessionToken with MFA to get temporary credentials.
func GetSessionToken(ctx context.Context, accessKey, secretKey, serialNumber, tokenCode string) (*SessionCredentials, error) {
	cfg := staticConfig(accessKey, secretKey)
	client := sts.NewFromConfig(cfg)
	return getSessionToken(ctx, client, serialNumber, tokenCode)
}

func getSessionToken(ctx context.Context, client stsAPI, serialNumber, tokenCode string) (*SessionCredentials, error) {
	out, err := client.GetSessionToken(ctx, &sts.GetSessionTokenInput{
		SerialNumber: aws.String(serialNumber),
		TokenCode:    aws.String(tokenCode),
	})
	if err != nil {
		return nil, fmt.Errorf("sts:GetSessionToken failed: %w", err)
	}

	return &SessionCredentials{
		AccessKeyID:     aws.ToString(out.Credentials.AccessKeyId),
		SecretAccessKey: aws.ToString(out.Credentials.SecretAccessKey),
		SessionToken:    aws.ToString(out.Credentials.SessionToken),
		Expiration:      out.Credentials.Expiration.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// ListMFADevices returns the MFA serial number for the given IAM user.
// If username is empty, it uses the caller's identity.
// When multiple devices exist, the first one is returned.
func ListMFADevices(ctx context.Context, accessKey, secretKey, username string) (string, error) {
	cfg := staticConfig(accessKey, secretKey)
	client := iam.NewFromConfig(cfg)
	return listMFADevices(ctx, client, username)
}

// ListAllMFADevices returns all MFA serial numbers for the given IAM user.
// If username is empty, it uses the caller's identity.
func ListAllMFADevices(ctx context.Context, accessKey, secretKey, username string) ([]string, error) {
	cfg := staticConfig(accessKey, secretKey)
	client := iam.NewFromConfig(cfg)
	return listAllMFADevices(ctx, client, username)
}

func listMFADevices(ctx context.Context, client iamAPI, username string) (string, error) {
	serials, err := listAllMFADevices(ctx, client, username)
	if err != nil {
		return "", err
	}
	return serials[0], nil
}

func listAllMFADevices(ctx context.Context, client iamAPI, username string) ([]string, error) {
	input := &iam.ListMFADevicesInput{}
	if username != "" {
		input.UserName = aws.String(username)
	}

	out, err := client.ListMFADevices(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("iam:ListMFADevices failed: %w", err)
	}

	if len(out.MFADevices) == 0 {
		return nil, fmt.Errorf("no MFA devices found for this IAM user")
	}

	serials := make([]string, len(out.MFADevices))
	for i, d := range out.MFADevices {
		serials[i] = aws.ToString(d.SerialNumber)
	}
	return serials, nil
}

// CreateAccessKey calls iam:CreateAccessKey to create a new access key.
// If username is empty, it creates a key for the caller.
func CreateAccessKey(ctx context.Context, accessKey, secretKey, username string) (*AccessKey, error) {
	cfg := staticConfig(accessKey, secretKey)
	client := iam.NewFromConfig(cfg)
	return createAccessKey(ctx, client, username)
}

// CreateAccessKeyWithSession is like CreateAccessKey but uses temporary session credentials.
func CreateAccessKeyWithSession(ctx context.Context, accessKey, secretKey, sessionToken, username string) (*AccessKey, error) {
	cfg := sessionConfig(accessKey, secretKey, sessionToken)
	client := iam.NewFromConfig(cfg)
	return createAccessKey(ctx, client, username)
}

func createAccessKey(ctx context.Context, client iamAPI, username string) (*AccessKey, error) {
	input := &iam.CreateAccessKeyInput{}
	if username != "" {
		input.UserName = aws.String(username)
	}

	out, err := client.CreateAccessKey(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("iam:CreateAccessKey failed: %w", err)
	}

	return &AccessKey{
		AccessKeyID:     aws.ToString(out.AccessKey.AccessKeyId),
		SecretAccessKey: aws.ToString(out.AccessKey.SecretAccessKey),
	}, nil
}

// DeleteAccessKey calls iam:DeleteAccessKey to delete an access key.
// If username is empty, it deletes a key for the caller.
func DeleteAccessKey(ctx context.Context, accessKey, secretKey, targetKeyID, username string) error {
	cfg := staticConfig(accessKey, secretKey)
	client := iam.NewFromConfig(cfg)
	return deleteAccessKey(ctx, client, targetKeyID, username)
}

// DeleteAccessKeyWithSession is like DeleteAccessKey but uses temporary session credentials.
func DeleteAccessKeyWithSession(ctx context.Context, accessKey, secretKey, sessionToken, targetKeyID, username string) error {
	cfg := sessionConfig(accessKey, secretKey, sessionToken)
	client := iam.NewFromConfig(cfg)
	return deleteAccessKey(ctx, client, targetKeyID, username)
}

func deleteAccessKey(ctx context.Context, client iamAPI, targetKeyID, username string) error {
	input := &iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(targetKeyID),
	}
	if username != "" {
		input.UserName = aws.String(username)
	}

	_, err := client.DeleteAccessKey(ctx, input)
	if err != nil {
		return fmt.Errorf("iam:DeleteAccessKey failed: %w", err)
	}
	return nil
}

// AssumeRole calls sts:AssumeRole using static (long-lived) source credentials.
func AssumeRole(ctx context.Context, sourceKey, sourceSecret, roleARN, sessionName, externalID string) (*SessionCredentials, error) {
	cfg := staticConfig(sourceKey, sourceSecret)
	client := sts.NewFromConfig(cfg)
	return assumeRole(ctx, client, roleARN, sessionName, externalID)
}

// AssumeRoleWithSession is like AssumeRole but uses temporary session credentials
// (e.g. after MFA via sts:GetSessionToken).
func AssumeRoleWithSession(ctx context.Context, sourceKey, sourceSecret, sourceToken, roleARN, sessionName, externalID string) (*SessionCredentials, error) {
	cfg := sessionConfig(sourceKey, sourceSecret, sourceToken)
	client := sts.NewFromConfig(cfg)
	return assumeRole(ctx, client, roleARN, sessionName, externalID)
}

func assumeRole(ctx context.Context, client stsAPI, roleARN, sessionName, externalID string) (*SessionCredentials, error) {
	if sessionName == "" {
		sessionName = "vop"
	}
	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(sessionName),
	}
	if externalID != "" {
		input.ExternalId = aws.String(externalID)
	}
	out, err := client.AssumeRole(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("sts:AssumeRole failed: %w", err)
	}
	return &SessionCredentials{
		AccessKeyID:     aws.ToString(out.Credentials.AccessKeyId),
		SecretAccessKey: aws.ToString(out.Credentials.SecretAccessKey),
		SessionToken:    aws.ToString(out.Credentials.SessionToken),
		Expiration:      out.Credentials.Expiration.Format("2006-01-02T15:04:05Z"),
	}, nil
}
