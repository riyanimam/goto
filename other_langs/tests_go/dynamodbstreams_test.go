package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"strings"
	"testing"
)

const (
	TableName = "TableWithStreamEnabled"
)

func TestDescribeStream(t *testing.T) {
	cfg, err := NewMotoAWSConfig(context.Background(), "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)

	CreateTableInput := &dynamodb.CreateTableInput{
		TableName: aws.String(TableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("SK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("PK"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("SK"),
				KeyType:       types.KeyTypeRange,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
		StreamSpecification: &types.StreamSpecification{
			StreamEnabled:  aws.Bool(true),
			StreamViewType: "NEW_AND_OLD_IMAGES",
		},
	}

	ctx := context.Background()
	result, err := ddbClient.CreateTable(ctx, CreateTableInput)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	streamArn := *result.TableDescription.LatestStreamArn

	ddbsClient := dynamodbstreams.NewFromConfig(cfg)

	DescribeStreamInput := &dynamodbstreams.DescribeStreamInput{
		StreamArn: aws.String(streamArn),
	}

	streamInfo, err := ddbsClient.DescribeStream(ctx, DescribeStreamInput)
	if err != nil {
		t.Fatalf("describe stream: %v", err)
	}

	streamArn = *streamInfo.StreamDescription.StreamArn
	checkFor := TableName + "/stream/"
	if !strings.Contains(streamArn, checkFor) {
		t.Errorf("StreamArn does not contain our table name [%s].", TableName)
	}
}
