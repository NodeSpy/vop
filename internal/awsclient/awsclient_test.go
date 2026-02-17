package awsclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// --- mock STS client ---

type mockSTS struct {
	GetCallerIdentityFunc func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
	GetSessionTokenFunc   func(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error)
}

func (m *mockSTS) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	if m.GetCallerIdentityFunc != nil {
		return m.GetCallerIdentityFunc(ctx, params, optFns...)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSTS) GetSessionToken(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error) {
	if m.GetSessionTokenFunc != nil {
		return m.GetSessionTokenFunc(ctx, params, optFns...)
	}
	return nil, fmt.Errorf("not implemented")
}

// --- mock IAM client ---

type mockIAM struct {
	ListMFADevicesFunc  func(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error)
	CreateAccessKeyFunc func(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
	DeleteAccessKeyFunc func(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
}

func (m *mockIAM) ListMFADevices(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error) {
	if m.ListMFADevicesFunc != nil {
		return m.ListMFADevicesFunc(ctx, params, optFns...)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockIAM) CreateAccessKey(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
	if m.CreateAccessKeyFunc != nil {
		return m.CreateAccessKeyFunc(ctx, params, optFns...)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockIAM) DeleteAccessKey(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
	if m.DeleteAccessKeyFunc != nil {
		return m.DeleteAccessKeyFunc(ctx, params, optFns...)
	}
	return nil, fmt.Errorf("not implemented")
}

// --- interface compliance ---

var _ stsAPI = (*mockSTS)(nil)
var _ iamAPI = (*mockIAM)(nil)

// --- tests ---

func TestGetCallerIdentity_Success(t *testing.T) {
	mock := &mockSTS{
		GetCallerIdentityFunc: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return &sts.GetCallerIdentityOutput{
				Account: aws.String("123456789012"),
				Arn:     aws.String("arn:aws:iam::123456789012:user/testuser"),
				UserId:  aws.String("AIDAEXAMPLE"),
			}, nil
		},
	}

	id, err := getCallerIdentity(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Account != "123456789012" {
		t.Errorf("expected account '123456789012', got %q", id.Account)
	}
	if id.Arn != "arn:aws:iam::123456789012:user/testuser" {
		t.Errorf("expected arn 'arn:aws:iam::123456789012:user/testuser', got %q", id.Arn)
	}
	if id.UserID != "AIDAEXAMPLE" {
		t.Errorf("expected userID 'AIDAEXAMPLE', got %q", id.UserID)
	}
}

func TestGetCallerIdentity_Error(t *testing.T) {
	mock := &mockSTS{
		GetCallerIdentityFunc: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return nil, fmt.Errorf("invalid credentials")
		},
	}

	_, err := getCallerIdentity(context.Background(), mock)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "sts:GetCallerIdentity failed: invalid credentials" {
		t.Errorf("unexpected error message: %q", got)
	}
}

func TestGetCallerIdentity_NilFields(t *testing.T) {
	mock := &mockSTS{
		GetCallerIdentityFunc: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return &sts.GetCallerIdentityOutput{
				// All fields nil — aws.ToString should return ""
			}, nil
		},
	}

	id, err := getCallerIdentity(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Account != "" || id.Arn != "" || id.UserID != "" {
		t.Errorf("expected empty strings for nil fields, got %+v", id)
	}
}

func TestGetSessionToken_Success(t *testing.T) {
	expiry := time.Date(2026, 2, 18, 12, 0, 0, 0, time.UTC)
	mock := &mockSTS{
		GetSessionTokenFunc: func(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error) {
			if aws.ToString(params.SerialNumber) != "arn:aws:iam::123456789012:mfa/testuser" {
				t.Errorf("unexpected serial: %s", aws.ToString(params.SerialNumber))
			}
			if aws.ToString(params.TokenCode) != "123456" {
				t.Errorf("unexpected token: %s", aws.ToString(params.TokenCode))
			}
			return &sts.GetSessionTokenOutput{
				Credentials: &ststypes.Credentials{
					AccessKeyId:     aws.String("ASIATEMP"),
					SecretAccessKey: aws.String("tempsecret"),
					SessionToken:    aws.String("token123"),
					Expiration:      &expiry,
				},
			}, nil
		},
	}

	creds, err := getSessionToken(context.Background(), mock,
		"arn:aws:iam::123456789012:mfa/testuser", "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AccessKeyID != "ASIATEMP" {
		t.Errorf("expected 'ASIATEMP', got %q", creds.AccessKeyID)
	}
	if creds.SecretAccessKey != "tempsecret" {
		t.Errorf("expected 'tempsecret', got %q", creds.SecretAccessKey)
	}
	if creds.SessionToken != "token123" {
		t.Errorf("expected 'token123', got %q", creds.SessionToken)
	}
	if creds.Expiration != "2026-02-18T12:00:00Z" {
		t.Errorf("expected '2026-02-18T12:00:00Z', got %q", creds.Expiration)
	}
}

func TestGetSessionToken_Error(t *testing.T) {
	mock := &mockSTS{
		GetSessionTokenFunc: func(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error) {
			return nil, fmt.Errorf("access denied")
		},
	}

	_, err := getSessionToken(context.Background(), mock, "serial", "code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListMFADevices_Success(t *testing.T) {
	mock := &mockIAM{
		ListMFADevicesFunc: func(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error) {
			if aws.ToString(params.UserName) != "testuser" {
				t.Errorf("expected username 'testuser', got %q", aws.ToString(params.UserName))
			}
			return &iam.ListMFADevicesOutput{
				MFADevices: []iamtypes.MFADevice{
					{SerialNumber: aws.String("arn:aws:iam::123456789012:mfa/testuser")},
				},
			}, nil
		},
	}

	serial, err := listMFADevices(context.Background(), mock, "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if serial != "arn:aws:iam::123456789012:mfa/testuser" {
		t.Errorf("unexpected serial: %q", serial)
	}
}

func TestListMFADevices_EmptyUsername(t *testing.T) {
	mock := &mockIAM{
		ListMFADevicesFunc: func(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error) {
			if params.UserName != nil {
				t.Errorf("expected nil username, got %q", aws.ToString(params.UserName))
			}
			return &iam.ListMFADevicesOutput{
				MFADevices: []iamtypes.MFADevice{
					{SerialNumber: aws.String("arn:aws:iam::123456789012:mfa/caller")},
				},
			}, nil
		},
	}

	serial, err := listMFADevices(context.Background(), mock, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if serial != "arn:aws:iam::123456789012:mfa/caller" {
		t.Errorf("unexpected serial: %q", serial)
	}
}

func TestListMFADevices_NoDevices(t *testing.T) {
	mock := &mockIAM{
		ListMFADevicesFunc: func(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error) {
			return &iam.ListMFADevicesOutput{
				MFADevices: []iamtypes.MFADevice{},
			}, nil
		},
	}

	_, err := listMFADevices(context.Background(), mock, "testuser")
	if err == nil {
		t.Fatal("expected error for no MFA devices, got nil")
	}
	if got := err.Error(); got != "no MFA devices found for this IAM user" {
		t.Errorf("unexpected error: %q", got)
	}
}

func TestListMFADevices_APIError(t *testing.T) {
	mock := &mockIAM{
		ListMFADevicesFunc: func(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error) {
			return nil, fmt.Errorf("access denied")
		},
	}

	_, err := listMFADevices(context.Background(), mock, "testuser")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateAccessKey_Success(t *testing.T) {
	mock := &mockIAM{
		CreateAccessKeyFunc: func(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
			if aws.ToString(params.UserName) != "testuser" {
				t.Errorf("expected username 'testuser', got %q", aws.ToString(params.UserName))
			}
			return &iam.CreateAccessKeyOutput{
				AccessKey: &iamtypes.AccessKey{
					AccessKeyId:     aws.String("AKIANEWKEY"),
					SecretAccessKey: aws.String("newsecret"),
				},
			}, nil
		},
	}

	key, err := createAccessKey(context.Background(), mock, "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.AccessKeyID != "AKIANEWKEY" {
		t.Errorf("expected 'AKIANEWKEY', got %q", key.AccessKeyID)
	}
	if key.SecretAccessKey != "newsecret" {
		t.Errorf("expected 'newsecret', got %q", key.SecretAccessKey)
	}
}

func TestCreateAccessKey_EmptyUsername(t *testing.T) {
	mock := &mockIAM{
		CreateAccessKeyFunc: func(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
			if params.UserName != nil {
				t.Errorf("expected nil username, got %q", aws.ToString(params.UserName))
			}
			return &iam.CreateAccessKeyOutput{
				AccessKey: &iamtypes.AccessKey{
					AccessKeyId:     aws.String("AKIASELF"),
					SecretAccessKey: aws.String("selfsecret"),
				},
			}, nil
		},
	}

	key, err := createAccessKey(context.Background(), mock, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.AccessKeyID != "AKIASELF" {
		t.Errorf("expected 'AKIASELF', got %q", key.AccessKeyID)
	}
}

func TestCreateAccessKey_Error(t *testing.T) {
	mock := &mockIAM{
		CreateAccessKeyFunc: func(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
			return nil, fmt.Errorf("limit exceeded")
		},
	}

	_, err := createAccessKey(context.Background(), mock, "testuser")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteAccessKey_Success(t *testing.T) {
	mock := &mockIAM{
		DeleteAccessKeyFunc: func(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
			if aws.ToString(params.AccessKeyId) != "AKIAOLDKEY" {
				t.Errorf("expected key ID 'AKIAOLDKEY', got %q", aws.ToString(params.AccessKeyId))
			}
			if aws.ToString(params.UserName) != "testuser" {
				t.Errorf("expected username 'testuser', got %q", aws.ToString(params.UserName))
			}
			return &iam.DeleteAccessKeyOutput{}, nil
		},
	}

	err := deleteAccessKey(context.Background(), mock, "AKIAOLDKEY", "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAccessKey_EmptyUsername(t *testing.T) {
	mock := &mockIAM{
		DeleteAccessKeyFunc: func(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
			if params.UserName != nil {
				t.Errorf("expected nil username, got %q", aws.ToString(params.UserName))
			}
			return &iam.DeleteAccessKeyOutput{}, nil
		},
	}

	err := deleteAccessKey(context.Background(), mock, "AKIAOLDKEY", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAccessKey_Error(t *testing.T) {
	mock := &mockIAM{
		DeleteAccessKeyFunc: func(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
			return nil, fmt.Errorf("no such key")
		},
	}

	err := deleteAccessKey(context.Background(), mock, "AKIABAD", "testuser")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStaticConfig(t *testing.T) {
	cfg := staticConfig("AKIATEST", "secrettest")
	if cfg.Region != "us-east-1" {
		t.Errorf("expected region 'us-east-1', got %q", cfg.Region)
	}
	if cfg.Credentials == nil {
		t.Fatal("expected credentials provider, got nil")
	}

	// Verify credentials resolve correctly.
	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("unexpected error retrieving creds: %v", err)
	}
	if creds.AccessKeyID != "AKIATEST" {
		t.Errorf("expected 'AKIATEST', got %q", creds.AccessKeyID)
	}
	if creds.SecretAccessKey != "secrettest" {
		t.Errorf("expected 'secrettest', got %q", creds.SecretAccessKey)
	}
}

func TestListMFADevices_MultipleDevices(t *testing.T) {
	// Should return the first device's serial.
	mock := &mockIAM{
		ListMFADevicesFunc: func(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error) {
			return &iam.ListMFADevicesOutput{
				MFADevices: []iamtypes.MFADevice{
					{SerialNumber: aws.String("arn:aws:iam::123456789012:mfa/first")},
					{SerialNumber: aws.String("arn:aws:iam::123456789012:mfa/second")},
				},
			}, nil
		},
	}

	serial, err := listMFADevices(context.Background(), mock, "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if serial != "arn:aws:iam::123456789012:mfa/first" {
		t.Errorf("expected first device serial, got %q", serial)
	}
}
