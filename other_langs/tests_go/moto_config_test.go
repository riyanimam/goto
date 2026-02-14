package main

import (
	"context"
	"testing"
)

func TestNewMotoAWSConfigDefaults(t *testing.T) {
	cfg, err := NewMotoAWSConfig(context.Background(), "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Region != DefaultMotoRegion {
		t.Fatalf("unexpected region: got %q want %q", cfg.Region, DefaultMotoRegion)
	}

	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("retrieve credentials: %v", err)
	}

	if creds.AccessKeyID != defaultCredentialValue {
		t.Fatalf("unexpected access key: got %q want %q", creds.AccessKeyID, defaultCredentialValue)
	}

	if creds.SecretAccessKey != defaultCredentialValue {
		t.Fatalf("unexpected secret key: got %q want %q", creds.SecretAccessKey, defaultCredentialValue)
	}
}

func TestNewMotoAWSConfigWithCustomEndpoint(t *testing.T) {
	cfg, err := NewMotoAWSConfig(context.Background(), "http://localhost:5999")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Region != DefaultMotoRegion {
		t.Fatalf("unexpected region: got %q want %q", cfg.Region, DefaultMotoRegion)
	}
}
