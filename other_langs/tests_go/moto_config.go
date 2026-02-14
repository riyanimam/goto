package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const (
	DefaultMotoEndpoint    = "http://localhost:5000"
	DefaultMotoRegion      = "us-east-1"
	defaultCredentialValue = "EXAMPLE"
)

func NewMotoAWSConfig(ctx context.Context, endpoint string) (aws.Config, error) {
	if endpoint == "" {
		endpoint = DefaultMotoEndpoint
	}

	return config.LoadDefaultConfig(
		ctx,
		config.WithBaseEndpoint(endpoint),
		config.WithRegion(DefaultMotoRegion),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(defaultCredentialValue, defaultCredentialValue, defaultCredentialValue),
		),
	)
}
