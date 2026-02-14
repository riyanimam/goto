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

	if creds.AccessKeyID != defaultCredential {
		t.Fatalf("unexpected access key: got %q want %q", creds.AccessKeyID, defaultCredential)
	}
}
