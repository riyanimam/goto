package awsmock_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	applicationautoscalingtypes "github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/appsync"
	appsynctypes "github.com/aws/aws-sdk-go-v2/service/appsync/types"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	athenatypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	batchtypes "github.com/aws/aws-sdk-go-v2/service/batch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwltypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	codebuildtypes "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	codepipelinetypes "github.com/aws/aws-sdk-go-v2/service/codepipeline/types"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cidptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	configtypes "github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/aws/aws-sdk-go-v2/service/dax"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	emrtypes "github.com/aws/aws-sdk-go-v2/service/emr/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	firehosetypes "github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	fsxtypes "github.com/aws/aws-sdk-go-v2/service/fsx/types"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	gluetypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	mqtypes "github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/aws/aws-sdk-go-v2/service/neptune"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	r53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	sdtypes "github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	ssoadmintypes "github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
	transfertypes "github.com/aws/aws-sdk-go-v2/service/transfer/types"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	wafv2types "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	"github.com/aws/aws-sdk-go-v2/service/xray"

	awsmock "github.com/riyanimam/goto"
)

// TestSTSGetCallerIdentity verifies that the mock STS service returns
// a valid GetCallerIdentity response.
func TestSTSGetCallerIdentity(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := sts.NewFromConfig(cfg)
	resp, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		t.Fatalf("GetCallerIdentity: %v", err)
	}

	if resp.Account == nil || *resp.Account != "123456789012" {
		t.Errorf("expected account 123456789012, got %v", resp.Account)
	}
	if resp.Arn == nil || !strings.Contains(*resp.Arn, "123456789012") {
		t.Errorf("expected ARN containing account ID, got %v", resp.Arn)
	}
	if resp.UserId == nil || *resp.UserId == "" {
		t.Errorf("expected non-empty UserId, got %v", resp.UserId)
	}
}

// TestSTSAssumeRole verifies that the mock STS service returns
// valid temporary credentials for AssumeRole.
func TestSTSAssumeRole(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := sts.NewFromConfig(cfg)
	resp, err := client.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String("arn:aws:iam::123456789012:role/test-role"),
		RoleSessionName: aws.String("test-session"),
		DurationSeconds: aws.Int32(3600),
	})
	if err != nil {
		t.Fatalf("AssumeRole: %v", err)
	}

	if resp.Credentials == nil {
		t.Fatal("expected credentials, got nil")
	}
	if resp.Credentials.AccessKeyId == nil || *resp.Credentials.AccessKeyId == "" {
		t.Error("expected non-empty AccessKeyId")
	}
	if resp.AssumedRoleUser == nil || resp.AssumedRoleUser.Arn == nil {
		t.Error("expected non-nil AssumedRoleUser")
	}
}

// TestS3BucketOperations tests create, list, head, and delete bucket operations.
func TestS3BucketOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Create bucket.
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("test-bucket"),
	})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// List buckets - should include our bucket.
	listResp, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		t.Fatalf("ListBuckets: %v", err)
	}

	found := false
	for _, b := range listResp.Buckets {
		if b.Name != nil && *b.Name == "test-bucket" {
			found = true
			break
		}
	}
	if !found {
		t.Error("test-bucket not found in ListBuckets response")
	}

	// Head bucket.
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String("test-bucket"),
	})
	if err != nil {
		t.Fatalf("HeadBucket: %v", err)
	}

	// Delete bucket.
	_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String("test-bucket"),
	})
	if err != nil {
		t.Fatalf("DeleteBucket: %v", err)
	}

	// Verify it's gone.
	listResp, err = client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		t.Fatalf("ListBuckets after delete: %v", err)
	}
	if len(listResp.Buckets) != 0 {
		t.Errorf("expected 0 buckets after delete, got %d", len(listResp.Buckets))
	}
}

// TestS3ObjectOperations tests put, get, head, and delete object operations.
func TestS3ObjectOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Create bucket.
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("obj-bucket"),
	})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Put object.
	content := "hello, world!"
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String("obj-bucket"),
		Key:         aws.String("test-key"),
		Body:        strings.NewReader(content),
		ContentType: aws.String("text/plain"),
	})
	if err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	// Get object.
	getResp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String("obj-bucket"),
		Key:    aws.String("test-key"),
	})
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	defer getResp.Body.Close()

	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != content {
		t.Errorf("expected body %q, got %q", content, string(body))
	}
	if getResp.ContentType != nil && *getResp.ContentType != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %v", *getResp.ContentType)
	}

	// Head object.
	headResp, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String("obj-bucket"),
		Key:    aws.String("test-key"),
	})
	if err != nil {
		t.Fatalf("HeadObject: %v", err)
	}
	if headResp.ContentLength == nil || *headResp.ContentLength != int64(len(content)) {
		t.Errorf("expected content length %d, got %v", len(content), headResp.ContentLength)
	}

	// Delete object.
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String("obj-bucket"),
		Key:    aws.String("test-key"),
	})
	if err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}

	// Verify it's gone - GetObject should fail.
	_, err = client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String("obj-bucket"),
		Key:    aws.String("test-key"),
	})
	if err == nil {
		t.Error("expected error after deleting object, got nil")
	}
}

// TestS3ListObjects tests listing objects with prefix filtering.
func TestS3ListObjects(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Create bucket.
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("list-bucket"),
	})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Put multiple objects.
	keys := []string{"docs/readme.md", "docs/guide.md", "images/logo.png", "root.txt"}
	for _, key := range keys {
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String("list-bucket"),
			Key:    aws.String(key),
			Body:   strings.NewReader("content of " + key),
		})
		if err != nil {
			t.Fatalf("PutObject(%s): %v", key, err)
		}
	}

	// List all objects.
	listResp, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String("list-bucket"),
	})
	if err != nil {
		t.Fatalf("ListObjectsV2: %v", err)
	}
	if len(listResp.Contents) != 4 {
		t.Errorf("expected 4 objects, got %d", len(listResp.Contents))
	}

	// List with prefix.
	listResp, err = client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String("list-bucket"),
		Prefix: aws.String("docs/"),
	})
	if err != nil {
		t.Fatalf("ListObjectsV2 with prefix: %v", err)
	}
	if len(listResp.Contents) != 2 {
		t.Errorf("expected 2 docs/* objects, got %d", len(listResp.Contents))
	}
}

// TestS3CopyObject tests copying an object between keys.
func TestS3CopyObject(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Create bucket and object.
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("copy-bucket"),
	})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	content := "original content"
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String("copy-bucket"),
		Key:    aws.String("source-key"),
		Body:   strings.NewReader(content),
	})
	if err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	// Copy object.
	_, err = client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String("copy-bucket"),
		Key:        aws.String("dest-key"),
		CopySource: aws.String("copy-bucket/source-key"),
	})
	if err != nil {
		t.Fatalf("CopyObject: %v", err)
	}

	// Verify the copy.
	getResp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String("copy-bucket"),
		Key:    aws.String("dest-key"),
	})
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	defer getResp.Body.Close()

	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != content {
		t.Errorf("expected copied body %q, got %q", content, string(body))
	}
}

// TestSQSQueueOperations tests create, list, get URL, and delete queue operations.
func TestSQSQueueOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := sqs.NewFromConfig(cfg)

	// Create queue.
	createResp, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("test-queue"),
	})
	if err != nil {
		t.Fatalf("CreateQueue: %v", err)
	}
	if createResp.QueueUrl == nil || *createResp.QueueUrl == "" {
		t.Fatal("expected non-empty QueueUrl")
	}
	queueURL := *createResp.QueueUrl

	// List queues.
	listResp, err := client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		t.Fatalf("ListQueues: %v", err)
	}
	if len(listResp.QueueUrls) != 1 {
		t.Errorf("expected 1 queue, got %d", len(listResp.QueueUrls))
	}

	// Get queue URL.
	urlResp, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String("test-queue"),
	})
	if err != nil {
		t.Fatalf("GetQueueUrl: %v", err)
	}
	if *urlResp.QueueUrl != queueURL {
		t.Errorf("expected URL %q, got %q", queueURL, *urlResp.QueueUrl)
	}

	// Delete queue.
	_, err = client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		t.Fatalf("DeleteQueue: %v", err)
	}
}

// TestSQSMessageOperations tests send, receive, and delete message operations.
func TestSQSMessageOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := sqs.NewFromConfig(cfg)

	// Create queue.
	createResp, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("msg-queue"),
	})
	if err != nil {
		t.Fatalf("CreateQueue: %v", err)
	}
	queueURL := *createResp.QueueUrl

	// Send message.
	sendResp, err := client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String("hello, queue!"),
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if sendResp.MessageId == nil || *sendResp.MessageId == "" {
		t.Error("expected non-empty MessageId")
	}

	// Receive message.
	recvResp, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
	})
	if err != nil {
		t.Fatalf("ReceiveMessage: %v", err)
	}
	if len(recvResp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(recvResp.Messages))
	}
	if *recvResp.Messages[0].Body != "hello, queue!" {
		t.Errorf("expected body %q, got %q", "hello, queue!", *recvResp.Messages[0].Body)
	}

	// Delete message.
	_, err = client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: recvResp.Messages[0].ReceiptHandle,
	})
	if err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}

	// Verify message is gone.
	recvResp, err = client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
	})
	if err != nil {
		t.Fatalf("ReceiveMessage after delete: %v", err)
	}
	if len(recvResp.Messages) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(recvResp.Messages))
	}
}

// TestMockServerReset verifies that Reset clears all state.
func TestMockServerReset(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Create a bucket.
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("reset-bucket"),
	})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Reset the server.
	mock.Reset()

	// Bucket should be gone.
	listResp, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		t.Fatalf("ListBuckets after reset: %v", err)
	}
	if len(listResp.Buckets) != 0 {
		t.Errorf("expected 0 buckets after reset, got %d", len(listResp.Buckets))
	}
}

// TestDynamoDBTableOperations tests create, describe, list, and delete table operations.
func TestDynamoDBTableOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	// Create table.
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("test-table"),
		KeySchema: []dbtypes.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: dbtypes.KeyTypeHash},
		},
		AttributeDefinitions: []dbtypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: dbtypes.ScalarAttributeTypeS},
		},
		BillingMode: dbtypes.BillingModePayPerRequest,
	})
	if err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	// Describe table.
	descResp, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String("test-table"),
	})
	if err != nil {
		t.Fatalf("DescribeTable: %v", err)
	}
	if descResp.Table == nil || descResp.Table.TableName == nil {
		t.Fatal("expected non-nil table description")
	}
	if *descResp.Table.TableName != "test-table" {
		t.Errorf("expected table name test-table, got %s", *descResp.Table.TableName)
	}
	if descResp.Table.TableStatus != dbtypes.TableStatusActive {
		t.Errorf("expected ACTIVE status, got %s", descResp.Table.TableStatus)
	}

	// List tables.
	listResp, err := client.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		t.Fatalf("ListTables: %v", err)
	}
	if len(listResp.TableNames) != 1 || listResp.TableNames[0] != "test-table" {
		t.Errorf("expected [test-table], got %v", listResp.TableNames)
	}

	// Delete table.
	_, err = client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String("test-table"),
	})
	if err != nil {
		t.Fatalf("DeleteTable: %v", err)
	}

	// Verify it's gone.
	listResp, err = client.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		t.Fatalf("ListTables after delete: %v", err)
	}
	if len(listResp.TableNames) != 0 {
		t.Errorf("expected 0 tables after delete, got %d", len(listResp.TableNames))
	}
}

// TestDynamoDBItemOperations tests put, get, and delete item operations.
func TestDynamoDBItemOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	// Create table.
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("items-table"),
		KeySchema: []dbtypes.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: dbtypes.KeyTypeHash},
		},
		AttributeDefinitions: []dbtypes.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: dbtypes.ScalarAttributeTypeS},
		},
		BillingMode: dbtypes.BillingModePayPerRequest,
	})
	if err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	// Put item.
	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("items-table"),
		Item: map[string]dbtypes.AttributeValue{
			"id":   &dbtypes.AttributeValueMemberS{Value: "item-1"},
			"name": &dbtypes.AttributeValueMemberS{Value: "Test Item"},
		},
	})
	if err != nil {
		t.Fatalf("PutItem: %v", err)
	}

	// Get item.
	getResp, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("items-table"),
		Key: map[string]dbtypes.AttributeValue{
			"id": &dbtypes.AttributeValueMemberS{Value: "item-1"},
		},
	})
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if getResp.Item == nil {
		t.Fatal("expected non-nil item")
	}
	if v, ok := getResp.Item["name"].(*dbtypes.AttributeValueMemberS); !ok || v.Value != "Test Item" {
		t.Errorf("expected name 'Test Item', got %v", getResp.Item["name"])
	}

	// Scan items.
	scanResp, err := client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String("items-table"),
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if scanResp.Count != 1 {
		t.Errorf("expected 1 item in scan, got %d", scanResp.Count)
	}

	// Delete item.
	_, err = client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String("items-table"),
		Key: map[string]dbtypes.AttributeValue{
			"id": &dbtypes.AttributeValueMemberS{Value: "item-1"},
		},
	})
	if err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}

	// Verify item is gone.
	getResp, err = client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("items-table"),
		Key: map[string]dbtypes.AttributeValue{
			"id": &dbtypes.AttributeValueMemberS{Value: "item-1"},
		},
	})
	if err != nil {
		t.Fatalf("GetItem after delete: %v", err)
	}
	if getResp.Item != nil {
		t.Error("expected nil item after delete")
	}
}

// TestSNSTopicOperations tests create, list, and delete topic operations.
func TestSNSTopicOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := sns.NewFromConfig(cfg)

	// Create topic.
	createResp, err := client.CreateTopic(ctx, &sns.CreateTopicInput{
		Name: aws.String("test-topic"),
	})
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	if createResp.TopicArn == nil || *createResp.TopicArn == "" {
		t.Fatal("expected non-empty TopicArn")
	}
	if !strings.Contains(*createResp.TopicArn, "test-topic") {
		t.Errorf("expected TopicArn to contain 'test-topic', got %s", *createResp.TopicArn)
	}

	// List topics.
	listResp, err := client.ListTopics(ctx, &sns.ListTopicsInput{})
	if err != nil {
		t.Fatalf("ListTopics: %v", err)
	}
	if len(listResp.Topics) != 1 {
		t.Errorf("expected 1 topic, got %d", len(listResp.Topics))
	}

	// Delete topic.
	_, err = client.DeleteTopic(ctx, &sns.DeleteTopicInput{
		TopicArn: createResp.TopicArn,
	})
	if err != nil {
		t.Fatalf("DeleteTopic: %v", err)
	}

	// Verify it's gone.
	listResp, err = client.ListTopics(ctx, &sns.ListTopicsInput{})
	if err != nil {
		t.Fatalf("ListTopics after delete: %v", err)
	}
	if len(listResp.Topics) != 0 {
		t.Errorf("expected 0 topics after delete, got %d", len(listResp.Topics))
	}
}

// TestSNSSubscription tests subscribe and list subscriptions.
func TestSNSSubscription(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := sns.NewFromConfig(cfg)

	// Create topic.
	createResp, err := client.CreateTopic(ctx, &sns.CreateTopicInput{
		Name: aws.String("sub-topic"),
	})
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	topicArn := *createResp.TopicArn

	// Subscribe.
	subResp, err := client.Subscribe(ctx, &sns.SubscribeInput{
		TopicArn: aws.String(topicArn),
		Protocol: aws.String("email"),
		Endpoint: aws.String("test@example.com"),
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if subResp.SubscriptionArn == nil || *subResp.SubscriptionArn == "" {
		t.Fatal("expected non-empty SubscriptionArn")
	}

	// List subscriptions.
	listResp, err := client.ListSubscriptions(ctx, &sns.ListSubscriptionsInput{})
	if err != nil {
		t.Fatalf("ListSubscriptions: %v", err)
	}
	if len(listResp.Subscriptions) != 1 {
		t.Errorf("expected 1 subscription, got %d", len(listResp.Subscriptions))
	}

	// Unsubscribe.
	_, err = client.Unsubscribe(ctx, &sns.UnsubscribeInput{
		SubscriptionArn: subResp.SubscriptionArn,
	})
	if err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}

	// Verify it's gone.
	listResp, err = client.ListSubscriptions(ctx, &sns.ListSubscriptionsInput{})
	if err != nil {
		t.Fatalf("ListSubscriptions after unsubscribe: %v", err)
	}
	if len(listResp.Subscriptions) != 0 {
		t.Errorf("expected 0 subscriptions after unsubscribe, got %d", len(listResp.Subscriptions))
	}
}

// TestSNSPublish tests publishing a message to a topic.
func TestSNSPublish(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := sns.NewFromConfig(cfg)

	// Create topic.
	createResp, err := client.CreateTopic(ctx, &sns.CreateTopicInput{
		Name: aws.String("publish-topic"),
	})
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}

	// Publish message.
	pubResp, err := client.Publish(ctx, &sns.PublishInput{
		TopicArn: createResp.TopicArn,
		Message:  aws.String("hello, world!"),
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if pubResp.MessageId == nil || *pubResp.MessageId == "" {
		t.Error("expected non-empty MessageId")
	}
}

// TestSecretsManagerOperations tests create, get, update, list, and delete secret operations.
func TestSecretsManagerOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := secretsmanager.NewFromConfig(cfg)

	// Create secret.
	createResp, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String("test-secret"),
		SecretString: aws.String("super-secret-value"),
		Description:  aws.String("A test secret"),
	})
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if createResp.ARN == nil || *createResp.ARN == "" {
		t.Fatal("expected non-empty ARN")
	}
	if createResp.Name == nil || *createResp.Name != "test-secret" {
		t.Errorf("expected name 'test-secret', got %v", createResp.Name)
	}

	// Get secret value.
	getResp, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String("test-secret"),
	})
	if err != nil {
		t.Fatalf("GetSecretValue: %v", err)
	}
	if getResp.SecretString == nil || *getResp.SecretString != "super-secret-value" {
		t.Errorf("expected secret value 'super-secret-value', got %v", getResp.SecretString)
	}

	// Update secret (PutSecretValue).
	_, err = client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String("test-secret"),
		SecretString: aws.String("updated-secret-value"),
	})
	if err != nil {
		t.Fatalf("PutSecretValue: %v", err)
	}

	// Get updated secret value.
	getResp, err = client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String("test-secret"),
	})
	if err != nil {
		t.Fatalf("GetSecretValue after update: %v", err)
	}
	if getResp.SecretString == nil || *getResp.SecretString != "updated-secret-value" {
		t.Errorf("expected updated secret value, got %v", getResp.SecretString)
	}

	// List secrets.
	listResp, err := client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
	if err != nil {
		t.Fatalf("ListSecrets: %v", err)
	}
	if len(listResp.SecretList) != 1 {
		t.Errorf("expected 1 secret, got %d", len(listResp.SecretList))
	}

	// Describe secret.
	descResp, err := client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String("test-secret"),
	})
	if err != nil {
		t.Fatalf("DescribeSecret: %v", err)
	}
	if descResp.Name == nil || *descResp.Name != "test-secret" {
		t.Errorf("expected name 'test-secret', got %v", descResp.Name)
	}

	// Delete secret.
	_, err = client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId: aws.String("test-secret"),
	})
	if err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}

	// Verify it's gone from list.
	listResp, err = client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
	if err != nil {
		t.Fatalf("ListSecrets after delete: %v", err)
	}
	if len(listResp.SecretList) != 0 {
		t.Errorf("expected 0 secrets after delete, got %d", len(listResp.SecretList))
	}
}

// TestLambdaFunctionOperations tests create, get, list, invoke, and delete function operations.
func TestLambdaFunctionOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := lambda.NewFromConfig(cfg)

	// Create function.
	createResp, err := client.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String("my-function"),
		Runtime:      lambdatypes.RuntimePython312,
		Role:         aws.String("arn:aws:iam::123456789012:role/lambda-role"),
		Handler:      aws.String("index.handler"),
		Code: &lambdatypes.FunctionCode{
			ZipFile: []byte("fake-code"),
		},
	})
	if err != nil {
		t.Fatalf("CreateFunction: %v", err)
	}
	if createResp.FunctionName == nil || *createResp.FunctionName != "my-function" {
		t.Errorf("expected function name 'my-function', got %v", createResp.FunctionName)
	}
	if createResp.FunctionArn == nil || !strings.Contains(*createResp.FunctionArn, "my-function") {
		t.Errorf("expected ARN containing 'my-function', got %v", createResp.FunctionArn)
	}

	// Get function.
	getResp, err := client.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String("my-function"),
	})
	if err != nil {
		t.Fatalf("GetFunction: %v", err)
	}
	if getResp.Configuration == nil || *getResp.Configuration.FunctionName != "my-function" {
		t.Error("expected function configuration with name 'my-function'")
	}

	// List functions.
	listResp, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		t.Fatalf("ListFunctions: %v", err)
	}
	if len(listResp.Functions) != 1 {
		t.Errorf("expected 1 function, got %d", len(listResp.Functions))
	}

	// Invoke function.
	invokeResp, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: aws.String("my-function"),
		Payload:      []byte(`{"key":"value"}`),
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if invokeResp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", invokeResp.StatusCode)
	}

	// Delete function.
	_, err = client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: aws.String("my-function"),
	})
	if err != nil {
		t.Fatalf("DeleteFunction: %v", err)
	}

	// Verify it's gone.
	listResp, err = client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		t.Fatalf("ListFunctions after delete: %v", err)
	}
	if len(listResp.Functions) != 0 {
		t.Errorf("expected 0 functions after delete, got %d", len(listResp.Functions))
	}
}

// TestCloudWatchLogsOperations tests log group, stream, and event operations.
func TestCloudWatchLogsOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := cloudwatchlogs.NewFromConfig(cfg)

	// Create log group.
	_, err = client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String("/test/logs"),
	})
	if err != nil {
		t.Fatalf("CreateLogGroup: %v", err)
	}

	// Describe log groups.
	descResp, err := client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{})
	if err != nil {
		t.Fatalf("DescribeLogGroups: %v", err)
	}
	if len(descResp.LogGroups) != 1 {
		t.Errorf("expected 1 log group, got %d", len(descResp.LogGroups))
	}
	if descResp.LogGroups[0].LogGroupName == nil || *descResp.LogGroups[0].LogGroupName != "/test/logs" {
		t.Errorf("expected log group name '/test/logs'")
	}

	// Create log stream.
	_, err = client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String("/test/logs"),
		LogStreamName: aws.String("stream-1"),
	})
	if err != nil {
		t.Fatalf("CreateLogStream: %v", err)
	}

	// Describe log streams.
	streamResp, err := client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String("/test/logs"),
	})
	if err != nil {
		t.Fatalf("DescribeLogStreams: %v", err)
	}
	if len(streamResp.LogStreams) != 1 {
		t.Errorf("expected 1 stream, got %d", len(streamResp.LogStreams))
	}

	// Put log events.
	_, err = client.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String("/test/logs"),
		LogStreamName: aws.String("stream-1"),
		LogEvents: []cwltypes.InputLogEvent{
			{Timestamp: aws.Int64(1000), Message: aws.String("hello log")},
		},
	})
	if err != nil {
		t.Fatalf("PutLogEvents: %v", err)
	}

	// Get log events.
	getResp, err := client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("/test/logs"),
		LogStreamName: aws.String("stream-1"),
	})
	if err != nil {
		t.Fatalf("GetLogEvents: %v", err)
	}
	if len(getResp.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(getResp.Events))
	}
	if getResp.Events[0].Message == nil || *getResp.Events[0].Message != "hello log" {
		t.Errorf("expected message 'hello log', got %v", getResp.Events[0].Message)
	}

	// Delete log group.
	_, err = client.DeleteLogGroup(ctx, &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: aws.String("/test/logs"),
	})
	if err != nil {
		t.Fatalf("DeleteLogGroup: %v", err)
	}

	// Verify it's gone.
	descResp, err = client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{})
	if err != nil {
		t.Fatalf("DescribeLogGroups after delete: %v", err)
	}
	if len(descResp.LogGroups) != 0 {
		t.Errorf("expected 0 log groups after delete, got %d", len(descResp.LogGroups))
	}
}

// TestIAMUserOperations tests create, get, list, and delete user operations.
func TestIAMUserOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := iam.NewFromConfig(cfg)

	// Create user.
	createResp, err := client.CreateUser(ctx, &iam.CreateUserInput{
		UserName: aws.String("test-user"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if createResp.User == nil || *createResp.User.UserName != "test-user" {
		t.Error("expected user with name 'test-user'")
	}

	// Get user.
	getResp, err := client.GetUser(ctx, &iam.GetUserInput{
		UserName: aws.String("test-user"),
	})
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if *getResp.User.UserName != "test-user" {
		t.Errorf("expected user name 'test-user', got %s", *getResp.User.UserName)
	}

	// List users.
	listUsersResp, err := client.ListUsers(ctx, &iam.ListUsersInput{})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(listUsersResp.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(listUsersResp.Users))
	}

	// Delete user.
	_, err = client.DeleteUser(ctx, &iam.DeleteUserInput{
		UserName: aws.String("test-user"),
	})
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	// Verify it's gone.
	listUsersResp, err = client.ListUsers(ctx, &iam.ListUsersInput{})
	if err != nil {
		t.Fatalf("ListUsers after delete: %v", err)
	}
	if len(listUsersResp.Users) != 0 {
		t.Errorf("expected 0 users after delete, got %d", len(listUsersResp.Users))
	}
}

// TestIAMRoleOperations tests create, get, list, and delete role operations.
func TestIAMRoleOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := iam.NewFromConfig(cfg)

	// Create role.
	createResp, err := client.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String("test-role"),
		AssumeRolePolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[]}`),
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if createResp.Role == nil || *createResp.Role.RoleName != "test-role" {
		t.Error("expected role with name 'test-role'")
	}

	// List roles.
	listResp, err := client.ListRoles(ctx, &iam.ListRolesInput{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(listResp.Roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(listResp.Roles))
	}

	// Delete role.
	_, err = client.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String("test-role"),
	})
	if err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
}

// TestEC2InstanceOperations tests run, describe, and terminate instance operations.
func TestEC2InstanceOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := ec2.NewFromConfig(cfg)

	// Run instances.
	runResp, err := client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:      aws.String("ami-12345678"),
		InstanceType: "t2.micro",
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("RunInstances: %v", err)
	}
	if len(runResp.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(runResp.Instances))
	}
	instanceID := *runResp.Instances[0].InstanceId
	if !strings.HasPrefix(instanceID, "i-") {
		t.Errorf("expected instance ID starting with 'i-', got %s", instanceID)
	}

	// Describe instances.
	descResp, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		t.Fatalf("DescribeInstances: %v", err)
	}
	if len(descResp.Reservations) == 0 || len(descResp.Reservations[0].Instances) == 0 {
		t.Fatal("expected at least one instance in reservations")
	}

	// Terminate instances.
	termResp, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		t.Fatalf("TerminateInstances: %v", err)
	}
	if len(termResp.TerminatingInstances) != 1 {
		t.Errorf("expected 1 terminating instance, got %d", len(termResp.TerminatingInstances))
	}
}

// TestEC2VpcOperations tests create, describe, and delete VPC operations.
func TestEC2VpcOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := ec2.NewFromConfig(cfg)

	// Create VPC.
	vpcResp, err := client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/16"),
	})
	if err != nil {
		t.Fatalf("CreateVpc: %v", err)
	}
	if vpcResp.Vpc == nil || vpcResp.Vpc.VpcId == nil {
		t.Fatal("expected non-nil VPC")
	}
	vpcID := *vpcResp.Vpc.VpcId

	// Describe VPCs.
	descResp, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		t.Fatalf("DescribeVpcs: %v", err)
	}
	if len(descResp.Vpcs) != 1 {
		t.Errorf("expected 1 VPC, got %d", len(descResp.Vpcs))
	}

	// Delete VPC.
	_, err = client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil {
		t.Fatalf("DeleteVpc: %v", err)
	}
}

// TestKinesisStreamOperations tests create, describe, list, put record, and delete stream operations.
func TestKinesisStreamOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := kinesis.NewFromConfig(cfg)

	// Create stream.
	_, err = client.CreateStream(ctx, &kinesis.CreateStreamInput{
		StreamName: aws.String("test-stream"),
		ShardCount: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}

	// Describe stream.
	descResp, err := client.DescribeStream(ctx, &kinesis.DescribeStreamInput{
		StreamName: aws.String("test-stream"),
	})
	if err != nil {
		t.Fatalf("DescribeStream: %v", err)
	}
	if descResp.StreamDescription == nil || *descResp.StreamDescription.StreamName != "test-stream" {
		t.Error("expected stream name 'test-stream'")
	}

	// List streams.
	listResp, err := client.ListStreams(ctx, &kinesis.ListStreamsInput{})
	if err != nil {
		t.Fatalf("ListStreams: %v", err)
	}
	if len(listResp.StreamNames) != 1 {
		t.Errorf("expected 1 stream, got %d", len(listResp.StreamNames))
	}

	// Put record.
	putResp, err := client.PutRecord(ctx, &kinesis.PutRecordInput{
		StreamName:   aws.String("test-stream"),
		Data:         []byte("hello kinesis"),
		PartitionKey: aws.String("key-1"),
	})
	if err != nil {
		t.Fatalf("PutRecord: %v", err)
	}
	if putResp.SequenceNumber == nil || *putResp.SequenceNumber == "" {
		t.Error("expected non-empty sequence number")
	}

	// Delete stream.
	_, err = client.DeleteStream(ctx, &kinesis.DeleteStreamInput{
		StreamName: aws.String("test-stream"),
	})
	if err != nil {
		t.Fatalf("DeleteStream: %v", err)
	}

	// Verify it's gone.
	listResp, err = client.ListStreams(ctx, &kinesis.ListStreamsInput{})
	if err != nil {
		t.Fatalf("ListStreams after delete: %v", err)
	}
	if len(listResp.StreamNames) != 0 {
		t.Errorf("expected 0 streams after delete, got %d", len(listResp.StreamNames))
	}
}

// TestEventBridgeOperations tests event bus, rule, target, and put events operations.
func TestEventBridgeOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := eventbridge.NewFromConfig(cfg)

	// List event buses - should have the default bus.
	busResp, err := client.ListEventBuses(ctx, &eventbridge.ListEventBusesInput{})
	if err != nil {
		t.Fatalf("ListEventBuses: %v", err)
	}
	if len(busResp.EventBuses) < 1 {
		t.Error("expected at least 1 event bus (default)")
	}

	// Create a custom event bus.
	createBusResp, err := client.CreateEventBus(ctx, &eventbridge.CreateEventBusInput{
		Name: aws.String("custom-bus"),
	})
	if err != nil {
		t.Fatalf("CreateEventBus: %v", err)
	}
	if createBusResp.EventBusArn == nil || *createBusResp.EventBusArn == "" {
		t.Error("expected non-empty EventBusArn")
	}

	// Put rule.
	ruleResp, err := client.PutRule(ctx, &eventbridge.PutRuleInput{
		Name:         aws.String("test-rule"),
		EventPattern: aws.String(`{"source":["test"]}`),
	})
	if err != nil {
		t.Fatalf("PutRule: %v", err)
	}
	if ruleResp.RuleArn == nil || *ruleResp.RuleArn == "" {
		t.Error("expected non-empty RuleArn")
	}

	// List rules.
	rulesResp, err := client.ListRules(ctx, &eventbridge.ListRulesInput{})
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rulesResp.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rulesResp.Rules))
	}

	// Put events.
	eventsResp, err := client.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []ebtypes.PutEventsRequestEntry{
			{
				Source:     aws.String("test"),
				DetailType: aws.String("TestEvent"),
				Detail:     aws.String(`{"key":"value"}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("PutEvents: %v", err)
	}
	if eventsResp.FailedEntryCount != 0 {
		t.Errorf("expected 0 failed entries, got %d", eventsResp.FailedEntryCount)
	}

	// Delete rule and bus.
	_, err = client.DeleteRule(ctx, &eventbridge.DeleteRuleInput{
		Name: aws.String("test-rule"),
	})
	if err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}

	_, err = client.DeleteEventBus(ctx, &eventbridge.DeleteEventBusInput{
		Name: aws.String("custom-bus"),
	})
	if err != nil {
		t.Fatalf("DeleteEventBus: %v", err)
	}
}

// TestSSMParameterOperations tests put, get, describe, get by path, and delete parameter operations.
func TestSSMParameterOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := ssm.NewFromConfig(cfg)

	// Put parameter.
	putResp, err := client.PutParameter(ctx, &ssm.PutParameterInput{
		Name:  aws.String("/app/database/host"),
		Value: aws.String("db.example.com"),
		Type:  ssmtypes.ParameterTypeString,
	})
	if err != nil {
		t.Fatalf("PutParameter: %v", err)
	}
	if putResp.Version != 1 {
		t.Errorf("expected version 1, got %d", putResp.Version)
	}

	// Get parameter.
	getResp, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String("/app/database/host"),
	})
	if err != nil {
		t.Fatalf("GetParameter: %v", err)
	}
	if getResp.Parameter == nil || *getResp.Parameter.Value != "db.example.com" {
		t.Errorf("expected value 'db.example.com', got %v", getResp.Parameter)
	}

	// Put another parameter for path testing.
	_, err = client.PutParameter(ctx, &ssm.PutParameterInput{
		Name:  aws.String("/app/database/port"),
		Value: aws.String("5432"),
		Type:  ssmtypes.ParameterTypeString,
	})
	if err != nil {
		t.Fatalf("PutParameter port: %v", err)
	}

	// Get parameters by path.
	pathResp, err := client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
		Path:      aws.String("/app/database"),
		Recursive: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("GetParametersByPath: %v", err)
	}
	if len(pathResp.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(pathResp.Parameters))
	}

	// Describe parameters.
	descResp, err := client.DescribeParameters(ctx, &ssm.DescribeParametersInput{})
	if err != nil {
		t.Fatalf("DescribeParameters: %v", err)
	}
	if len(descResp.Parameters) != 2 {
		t.Errorf("expected 2 parameter descriptions, got %d", len(descResp.Parameters))
	}

	// Delete parameter.
	_, err = client.DeleteParameter(ctx, &ssm.DeleteParameterInput{
		Name: aws.String("/app/database/host"),
	})
	if err != nil {
		t.Fatalf("DeleteParameter: %v", err)
	}
}

// TestKMSKeyOperations tests create, describe, list, encrypt, decrypt, and alias operations.
func TestKMSKeyOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := kms.NewFromConfig(cfg)

	// Create key.
	createResp, err := client.CreateKey(ctx, &kms.CreateKeyInput{
		Description: aws.String("Test encryption key"),
	})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if createResp.KeyMetadata == nil || createResp.KeyMetadata.KeyId == nil {
		t.Fatal("expected non-nil KeyMetadata")
	}
	keyID := *createResp.KeyMetadata.KeyId

	// Describe key.
	descResp, err := client.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		t.Fatalf("DescribeKey: %v", err)
	}
	if descResp.KeyMetadata == nil || *descResp.KeyMetadata.Description != "Test encryption key" {
		t.Error("expected description 'Test encryption key'")
	}

	// List keys.
	listResp, err := client.ListKeys(ctx, &kms.ListKeysInput{})
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(listResp.Keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(listResp.Keys))
	}

	// Encrypt.
	encResp, err := client.Encrypt(ctx, &kms.EncryptInput{
		KeyId:     aws.String(keyID),
		Plaintext: []byte("secret data"),
	})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if len(encResp.CiphertextBlob) == 0 {
		t.Error("expected non-empty ciphertext")
	}

	// Decrypt.
	decResp, err := client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob: encResp.CiphertextBlob,
	})
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decResp.Plaintext) != "secret data" {
		t.Errorf("expected plaintext 'secret data', got %q", string(decResp.Plaintext))
	}

	// Create alias.
	_, err = client.CreateAlias(ctx, &kms.CreateAliasInput{
		AliasName:   aws.String("alias/test-key"),
		TargetKeyId: aws.String(keyID),
	})
	if err != nil {
		t.Fatalf("CreateAlias: %v", err)
	}

	// List aliases.
	aliasResp, err := client.ListAliases(ctx, &kms.ListAliasesInput{})
	if err != nil {
		t.Fatalf("ListAliases: %v", err)
	}
	if len(aliasResp.Aliases) != 1 {
		t.Errorf("expected 1 alias, got %d", len(aliasResp.Aliases))
	}

	// Delete alias.
	_, err = client.DeleteAlias(ctx, &kms.DeleteAliasInput{
		AliasName: aws.String("alias/test-key"),
	})
	if err != nil {
		t.Fatalf("DeleteAlias: %v", err)
	}
}

// TestCloudFormationStackOperations tests create, describe, list, update, and delete stack operations.
func TestCloudFormationStackOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := cloudformation.NewFromConfig(cfg)

	// Create stack.
	createResp, err := client.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:    aws.String("test-stack"),
		TemplateBody: aws.String(`{"AWSTemplateFormatVersion":"2010-09-09","Resources":{}}`),
	})
	if err != nil {
		t.Fatalf("CreateStack: %v", err)
	}
	if createResp.StackId == nil || *createResp.StackId == "" {
		t.Error("expected non-empty StackId")
	}

	// Describe stacks.
	descResp, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String("test-stack"),
	})
	if err != nil {
		t.Fatalf("DescribeStacks: %v", err)
	}
	if len(descResp.Stacks) != 1 {
		t.Errorf("expected 1 stack, got %d", len(descResp.Stacks))
	}
	if *descResp.Stacks[0].StackName != "test-stack" {
		t.Errorf("expected stack name 'test-stack', got %s", *descResp.Stacks[0].StackName)
	}

	// List stacks.
	listResp, err := client.ListStacks(ctx, &cloudformation.ListStacksInput{})
	if err != nil {
		t.Fatalf("ListStacks: %v", err)
	}
	if len(listResp.StackSummaries) != 1 {
		t.Errorf("expected 1 stack summary, got %d", len(listResp.StackSummaries))
	}

	// Update stack.
	_, err = client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:    aws.String("test-stack"),
		TemplateBody: aws.String(`{"AWSTemplateFormatVersion":"2010-09-09","Resources":{"Bucket":{}}}`),
	})
	if err != nil {
		t.Fatalf("UpdateStack: %v", err)
	}

	// Delete stack.
	_, err = client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String("test-stack"),
	})
	if err != nil {
		t.Fatalf("DeleteStack: %v", err)
	}

	// Verify it's gone.
	descResp, err = client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String("test-stack"),
	})
	if err != nil {
		t.Fatalf("DescribeStacks after delete: %v", err)
	}
	if len(descResp.Stacks) != 0 {
		t.Errorf("expected 0 stacks after delete, got %d", len(descResp.Stacks))
	}
}

// TestECRRepositoryOperations tests create, describe, list images, put image, and delete repository operations.
func TestECRRepositoryOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := ecr.NewFromConfig(cfg)

	// Create repository.
	createResp, err := client.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName: aws.String("my-app"),
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	if createResp.Repository == nil || *createResp.Repository.RepositoryName != "my-app" {
		t.Error("expected repository name 'my-app'")
	}

	// Describe repositories.
	descResp, err := client.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
	if err != nil {
		t.Fatalf("DescribeRepositories: %v", err)
	}
	if len(descResp.Repositories) != 1 {
		t.Errorf("expected 1 repository, got %d", len(descResp.Repositories))
	}

	// Put image.
	putResp, err := client.PutImage(ctx, &ecr.PutImageInput{
		RepositoryName: aws.String("my-app"),
		ImageTag:       aws.String("latest"),
		ImageManifest:  aws.String(`{"schemaVersion":2}`),
	})
	if err != nil {
		t.Fatalf("PutImage: %v", err)
	}
	if putResp.Image == nil || putResp.Image.ImageId == nil {
		t.Error("expected non-nil image result")
	}

	// List images.
	listResp, err := client.ListImages(ctx, &ecr.ListImagesInput{
		RepositoryName: aws.String("my-app"),
	})
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	if len(listResp.ImageIds) != 1 {
		t.Errorf("expected 1 image, got %d", len(listResp.ImageIds))
	}

	// Get authorization token.
	authResp, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		t.Fatalf("GetAuthorizationToken: %v", err)
	}
	if len(authResp.AuthorizationData) != 1 {
		t.Errorf("expected 1 auth data, got %d", len(authResp.AuthorizationData))
	}

	// Delete repository.
	_, err = client.DeleteRepository(ctx, &ecr.DeleteRepositoryInput{
		RepositoryName: aws.String("my-app"),
	})
	if err != nil {
		t.Fatalf("DeleteRepository: %v", err)
	}

	// Verify it's gone.
	descResp, err = client.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
	if err != nil {
		t.Fatalf("DescribeRepositories after delete: %v", err)
	}
	if len(descResp.Repositories) != 0 {
		t.Errorf("expected 0 repositories after delete, got %d", len(descResp.Repositories))
	}
}

// ─── Route 53 ───────────────────────────────────────────────────────────────

func TestRoute53HostedZoneOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := route53.NewFromConfig(cfg)

	// Create hosted zone.
	createResp, err := client.CreateHostedZone(ctx, &route53.CreateHostedZoneInput{
		Name:            aws.String("example.com."),
		CallerReference: aws.String("unique-ref-1"),
	})
	if err != nil {
		t.Fatalf("CreateHostedZone: %v", err)
	}
	if createResp.HostedZone == nil {
		t.Fatal("expected HostedZone in response")
	}
	zoneID := createResp.HostedZone.Id
	// Extract just the zone ID (remove /hostedzone/ prefix).
	zoneIDStr := *zoneID
	if idx := strings.LastIndex(zoneIDStr, "/"); idx >= 0 {
		zoneIDStr = zoneIDStr[idx+1:]
	}

	// List hosted zones.
	listResp, err := client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		t.Fatalf("ListHostedZones: %v", err)
	}
	if len(listResp.HostedZones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(listResp.HostedZones))
	}

	// Change resource record sets (add an A record).
	_, err = client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneIDStr),
		ChangeBatch: &r53types.ChangeBatch{
			Changes: []r53types.Change{
				{
					Action: r53types.ChangeActionCreate,
					ResourceRecordSet: &r53types.ResourceRecordSet{
						Name: aws.String("app.example.com."),
						Type: r53types.RRTypeA,
						TTL:  aws.Int64(300),
						ResourceRecords: []r53types.ResourceRecord{
							{Value: aws.String("1.2.3.4")},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ChangeResourceRecordSets: %v", err)
	}

	// List resource record sets.
	rrsResp, err := client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneIDStr),
	})
	if err != nil {
		t.Fatalf("ListResourceRecordSets: %v", err)
	}
	// Should have NS + SOA + our new A record.
	if len(rrsResp.ResourceRecordSets) < 3 {
		t.Errorf("expected at least 3 record sets, got %d", len(rrsResp.ResourceRecordSets))
	}

	// Delete hosted zone.
	_, err = client.DeleteHostedZone(ctx, &route53.DeleteHostedZoneInput{
		Id: aws.String(zoneIDStr),
	})
	if err != nil {
		t.Fatalf("DeleteHostedZone: %v", err)
	}

	// Verify it's gone.
	listResp, err = client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		t.Fatalf("ListHostedZones after delete: %v", err)
	}
	if len(listResp.HostedZones) != 0 {
		t.Errorf("expected 0 zones after delete, got %d", len(listResp.HostedZones))
	}
}

// ─── ECS ────────────────────────────────────────────────────────────────────

func TestECSClusterAndServiceOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := ecs.NewFromConfig(cfg)

	// Create cluster.
	clusterResp, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String("test-cluster"),
	})
	if err != nil {
		t.Fatalf("CreateCluster: %v", err)
	}
	if *clusterResp.Cluster.ClusterName != "test-cluster" {
		t.Errorf("expected cluster name 'test-cluster', got %q", *clusterResp.Cluster.ClusterName)
	}

	// List clusters.
	listResp, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(listResp.ClusterArns) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(listResp.ClusterArns))
	}

	// Register task definition.
	tdResp, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family: aws.String("my-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:   aws.String("web"),
				Image:  aws.String("nginx:latest"),
				Cpu:    256,
				Memory: aws.Int32(512),
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterTaskDefinition: %v", err)
	}
	if *tdResp.TaskDefinition.Family != "my-task" {
		t.Errorf("expected family 'my-task', got %q", *tdResp.TaskDefinition.Family)
	}
	tdArn := tdResp.TaskDefinition.TaskDefinitionArn

	// Create service.
	svcResp, err := client.CreateService(ctx, &ecs.CreateServiceInput{
		ServiceName:    aws.String("web-service"),
		Cluster:        aws.String("test-cluster"),
		TaskDefinition: tdArn,
		DesiredCount:   aws.Int32(2),
	})
	if err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	if *svcResp.Service.ServiceName != "web-service" {
		t.Errorf("expected service name 'web-service', got %q", *svcResp.Service.ServiceName)
	}
	if svcResp.Service.DesiredCount != 2 {
		t.Errorf("expected desired count 2, got %d", svcResp.Service.DesiredCount)
	}

	// List services.
	svcListResp, err := client.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: aws.String("test-cluster"),
	})
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if len(svcListResp.ServiceArns) != 1 {
		t.Errorf("expected 1 service, got %d", len(svcListResp.ServiceArns))
	}

	// Delete service.
	_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
		Service: aws.String("web-service"),
		Cluster: aws.String("test-cluster"),
	})
	if err != nil {
		t.Fatalf("DeleteService: %v", err)
	}

	// Delete cluster.
	_, err = client.DeleteCluster(ctx, &ecs.DeleteClusterInput{
		Cluster: aws.String("test-cluster"),
	})
	if err != nil {
		t.Fatalf("DeleteCluster: %v", err)
	}
}

// ─── ELBv2 ──────────────────────────────────────────────────────────────────

func TestELBv2LoadBalancerOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := elasticloadbalancingv2.NewFromConfig(cfg)

	// Create load balancer.
	lbResp, err := client.CreateLoadBalancer(ctx, &elasticloadbalancingv2.CreateLoadBalancerInput{
		Name: aws.String("test-lb"),
	})
	if err != nil {
		t.Fatalf("CreateLoadBalancer: %v", err)
	}
	if len(lbResp.LoadBalancers) != 1 {
		t.Fatalf("expected 1 load balancer, got %d", len(lbResp.LoadBalancers))
	}
	lbArn := lbResp.LoadBalancers[0].LoadBalancerArn

	// Create target group.
	tgResp, err := client.CreateTargetGroup(ctx, &elasticloadbalancingv2.CreateTargetGroupInput{
		Name:     aws.String("test-tg"),
		Protocol: elbv2types.ProtocolEnumHttp,
		Port:     aws.Int32(80),
	})
	if err != nil {
		t.Fatalf("CreateTargetGroup: %v", err)
	}
	if len(tgResp.TargetGroups) != 1 {
		t.Fatalf("expected 1 target group, got %d", len(tgResp.TargetGroups))
	}
	tgArn := tgResp.TargetGroups[0].TargetGroupArn

	// Create listener.
	lnResp, err := client.CreateListener(ctx, &elasticloadbalancingv2.CreateListenerInput{
		LoadBalancerArn: lbArn,
		Protocol:        elbv2types.ProtocolEnumHttp,
		Port:            aws.Int32(80),
		DefaultActions: []elbv2types.Action{
			{Type: elbv2types.ActionTypeEnumForward, TargetGroupArn: tgArn},
		},
	})
	if err != nil {
		t.Fatalf("CreateListener: %v", err)
	}
	if len(lnResp.Listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(lnResp.Listeners))
	}

	// Describe load balancers.
	descLBResp, err := client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err != nil {
		t.Fatalf("DescribeLoadBalancers: %v", err)
	}
	if len(descLBResp.LoadBalancers) != 1 {
		t.Errorf("expected 1 LB, got %d", len(descLBResp.LoadBalancers))
	}

	// Describe target groups.
	descTGResp, err := client.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{})
	if err != nil {
		t.Fatalf("DescribeTargetGroups: %v", err)
	}
	if len(descTGResp.TargetGroups) != 1 {
		t.Errorf("expected 1 TG, got %d", len(descTGResp.TargetGroups))
	}

	// Clean up.
	_, _ = client.DeleteTargetGroup(ctx, &elasticloadbalancingv2.DeleteTargetGroupInput{
		TargetGroupArn: tgArn,
	})
	_, _ = client.DeleteLoadBalancer(ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
		LoadBalancerArn: lbArn,
	})

	// Verify LBs are gone.
	descLBResp, err = client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err != nil {
		t.Fatalf("DescribeLoadBalancers after delete: %v", err)
	}
	if len(descLBResp.LoadBalancers) != 0 {
		t.Errorf("expected 0 LBs after delete, got %d", len(descLBResp.LoadBalancers))
	}
}

// ─── RDS ────────────────────────────────────────────────────────────────────

func TestRDSInstanceOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := rds.NewFromConfig(cfg)

	// Create DB instance.
	createResp, err := client.CreateDBInstance(ctx, &rds.CreateDBInstanceInput{
		DBInstanceIdentifier: aws.String("test-db"),
		DBInstanceClass:      aws.String("db.t3.micro"),
		Engine:               aws.String("mysql"),
		MasterUsername:       aws.String("admin"),
		MasterUserPassword:   aws.String("password123"),
	})
	if err != nil {
		t.Fatalf("CreateDBInstance: %v", err)
	}
	if *createResp.DBInstance.DBInstanceIdentifier != "test-db" {
		t.Errorf("expected identifier 'test-db', got %q", *createResp.DBInstance.DBInstanceIdentifier)
	}
	if *createResp.DBInstance.Engine != "mysql" {
		t.Errorf("expected engine 'mysql', got %q", *createResp.DBInstance.Engine)
	}

	// Describe DB instances.
	descResp, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		t.Fatalf("DescribeDBInstances: %v", err)
	}
	if len(descResp.DBInstances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(descResp.DBInstances))
	}

	// Modify DB instance.
	modResp, err := client.ModifyDBInstance(ctx, &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String("test-db"),
		DBInstanceClass:      aws.String("db.t3.medium"),
	})
	if err != nil {
		t.Fatalf("ModifyDBInstance: %v", err)
	}
	if *modResp.DBInstance.DBInstanceClass != "db.t3.medium" {
		t.Errorf("expected class 'db.t3.medium', got %q", *modResp.DBInstance.DBInstanceClass)
	}

	// Delete DB instance.
	_, err = client.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: aws.String("test-db"),
		SkipFinalSnapshot:    aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("DeleteDBInstance: %v", err)
	}

	// Verify empty.
	descResp, err = client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		t.Fatalf("DescribeDBInstances after delete: %v", err)
	}
	if len(descResp.DBInstances) != 0 {
		t.Errorf("expected 0 instances after delete, got %d", len(descResp.DBInstances))
	}
}

func TestRDSClusterOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := rds.NewFromConfig(cfg)

	// Create DB cluster.
	createResp, err := client.CreateDBCluster(ctx, &rds.CreateDBClusterInput{
		DBClusterIdentifier: aws.String("test-cluster"),
		Engine:              aws.String("aurora-mysql"),
		MasterUsername:      aws.String("admin"),
		MasterUserPassword:  aws.String("password123"),
	})
	if err != nil {
		t.Fatalf("CreateDBCluster: %v", err)
	}
	if *createResp.DBCluster.DBClusterIdentifier != "test-cluster" {
		t.Errorf("expected identifier 'test-cluster', got %q", *createResp.DBCluster.DBClusterIdentifier)
	}

	// Describe DB clusters.
	descResp, err := client.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{})
	if err != nil {
		t.Fatalf("DescribeDBClusters: %v", err)
	}
	if len(descResp.DBClusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(descResp.DBClusters))
	}

	// Delete DB cluster.
	_, err = client.DeleteDBCluster(ctx, &rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String("test-cluster"),
		SkipFinalSnapshot:   aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("DeleteDBCluster: %v", err)
	}
}

// ─── CloudWatch (metrics) ───────────────────────────────────────────────────

func TestCloudWatchMetricOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := cloudwatch.NewFromConfig(cfg)

	// Put metric data.
	_, err = client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("MyApp"),
		MetricData: []cwtypes.MetricDatum{
			{
				MetricName: aws.String("RequestCount"),
				Value:      aws.Float64(42.0),
				Unit:       cwtypes.StandardUnitCount,
			},
		},
	})
	if err != nil {
		t.Fatalf("PutMetricData: %v", err)
	}

	// List metrics.
	listResp, err := client.ListMetrics(ctx, &cloudwatch.ListMetricsInput{
		Namespace: aws.String("MyApp"),
	})
	if err != nil {
		t.Fatalf("ListMetrics: %v", err)
	}
	if len(listResp.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(listResp.Metrics))
	}
	if *listResp.Metrics[0].MetricName != "RequestCount" {
		t.Errorf("expected metric name 'RequestCount', got %q", *listResp.Metrics[0].MetricName)
	}

	// Put metric alarm.
	_, err = client.PutMetricAlarm(ctx, &cloudwatch.PutMetricAlarmInput{
		AlarmName:          aws.String("HighRequestCount"),
		Namespace:          aws.String("MyApp"),
		MetricName:         aws.String("RequestCount"),
		ComparisonOperator: cwtypes.ComparisonOperatorGreaterThanThreshold,
		Threshold:          aws.Float64(100),
		Period:             aws.Int32(300),
		EvaluationPeriods:  aws.Int32(1),
		Statistic:          cwtypes.StatisticAverage,
	})
	if err != nil {
		t.Fatalf("PutMetricAlarm: %v", err)
	}

	// Describe alarms.
	alarmResp, err := client.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{})
	if err != nil {
		t.Fatalf("DescribeAlarms: %v", err)
	}
	if len(alarmResp.MetricAlarms) != 1 {
		t.Fatalf("expected 1 alarm, got %d", len(alarmResp.MetricAlarms))
	}
	if *alarmResp.MetricAlarms[0].AlarmName != "HighRequestCount" {
		t.Errorf("expected alarm name 'HighRequestCount', got %q", *alarmResp.MetricAlarms[0].AlarmName)
	}

	// Delete alarms.
	_, err = client.DeleteAlarms(ctx, &cloudwatch.DeleteAlarmsInput{
		AlarmNames: []string{"HighRequestCount"},
	})
	if err != nil {
		t.Fatalf("DeleteAlarms: %v", err)
	}

	// Verify empty.
	alarmResp, err = client.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{})
	if err != nil {
		t.Fatalf("DescribeAlarms after delete: %v", err)
	}
	if len(alarmResp.MetricAlarms) != 0 {
		t.Errorf("expected 0 alarms after delete, got %d", len(alarmResp.MetricAlarms))
	}
}

// ─── Step Functions ─────────────────────────────────────────────────────────

func TestStepFunctionsStateMachineOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := sfn.NewFromConfig(cfg)

	// Create state machine.
	definition := `{"StartAt": "Hello", "States": {"Hello": {"Type": "Pass", "End": true}}}`
	createResp, err := client.CreateStateMachine(ctx, &sfn.CreateStateMachineInput{
		Name:       aws.String("test-sm"),
		Definition: aws.String(definition),
		RoleArn:    aws.String("arn:aws:iam::123456789012:role/step-role"),
	})
	if err != nil {
		t.Fatalf("CreateStateMachine: %v", err)
	}
	smArn := createResp.StateMachineArn
	if smArn == nil || !strings.Contains(*smArn, "test-sm") {
		t.Errorf("expected state machine ARN containing 'test-sm', got %v", smArn)
	}

	// Describe state machine.
	descResp, err := client.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
		StateMachineArn: smArn,
	})
	if err != nil {
		t.Fatalf("DescribeStateMachine: %v", err)
	}
	if *descResp.Name != "test-sm" {
		t.Errorf("expected name 'test-sm', got %q", *descResp.Name)
	}
	if *descResp.Definition != definition {
		t.Errorf("definition mismatch")
	}

	// List state machines.
	listResp, err := client.ListStateMachines(ctx, &sfn.ListStateMachinesInput{})
	if err != nil {
		t.Fatalf("ListStateMachines: %v", err)
	}
	if len(listResp.StateMachines) != 1 {
		t.Fatalf("expected 1 state machine, got %d", len(listResp.StateMachines))
	}

	// Start execution.
	execResp, err := client.StartExecution(ctx, &sfn.StartExecutionInput{
		StateMachineArn: smArn,
		Name:            aws.String("exec-1"),
		Input:           aws.String(`{"key":"value"}`),
	})
	if err != nil {
		t.Fatalf("StartExecution: %v", err)
	}
	execArn := execResp.ExecutionArn

	// Describe execution.
	descExecResp, err := client.DescribeExecution(ctx, &sfn.DescribeExecutionInput{
		ExecutionArn: execArn,
	})
	if err != nil {
		t.Fatalf("DescribeExecution: %v", err)
	}
	if *descExecResp.Name != "exec-1" {
		t.Errorf("expected execution name 'exec-1', got %q", *descExecResp.Name)
	}

	// Stop execution.
	_, err = client.StopExecution(ctx, &sfn.StopExecutionInput{
		ExecutionArn: execArn,
	})
	if err != nil {
		t.Fatalf("StopExecution: %v", err)
	}

	// Delete state machine.
	_, err = client.DeleteStateMachine(ctx, &sfn.DeleteStateMachineInput{
		StateMachineArn: smArn,
	})
	if err != nil {
		t.Fatalf("DeleteStateMachine: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListStateMachines(ctx, &sfn.ListStateMachinesInput{})
	if err != nil {
		t.Fatalf("ListStateMachines after delete: %v", err)
	}
	if len(listResp.StateMachines) != 0 {
		t.Errorf("expected 0 state machines, got %d", len(listResp.StateMachines))
	}
}

// ─── ACM ────────────────────────────────────────────────────────────────────

func TestACMCertificateOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := acm.NewFromConfig(cfg)

	// Request certificate.
	reqResp, err := client.RequestCertificate(ctx, &acm.RequestCertificateInput{
		DomainName: aws.String("example.com"),
	})
	if err != nil {
		t.Fatalf("RequestCertificate: %v", err)
	}
	certArn := reqResp.CertificateArn
	if certArn == nil || *certArn == "" {
		t.Fatal("expected non-empty certificate ARN")
	}

	// Describe certificate.
	descResp, err := client.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
		CertificateArn: certArn,
	})
	if err != nil {
		t.Fatalf("DescribeCertificate: %v", err)
	}
	if *descResp.Certificate.DomainName != "example.com" {
		t.Errorf("expected domain 'example.com', got %q", *descResp.Certificate.DomainName)
	}

	// List certificates.
	listResp, err := client.ListCertificates(ctx, &acm.ListCertificatesInput{})
	if err != nil {
		t.Fatalf("ListCertificates: %v", err)
	}
	if len(listResp.CertificateSummaryList) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(listResp.CertificateSummaryList))
	}

	// Delete certificate.
	_, err = client.DeleteCertificate(ctx, &acm.DeleteCertificateInput{
		CertificateArn: certArn,
	})
	if err != nil {
		t.Fatalf("DeleteCertificate: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListCertificates(ctx, &acm.ListCertificatesInput{})
	if err != nil {
		t.Fatalf("ListCertificates after delete: %v", err)
	}
	if len(listResp.CertificateSummaryList) != 0 {
		t.Errorf("expected 0 certs after delete, got %d", len(listResp.CertificateSummaryList))
	}
}

// ─── SES ────────────────────────────────────────────────────────────────────

func TestSESEmailOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := sesv2.NewFromConfig(cfg)

	// Create email identity.
	_, err = client.CreateEmailIdentity(ctx, &sesv2.CreateEmailIdentityInput{
		EmailIdentity: aws.String("sender@example.com"),
	})
	if err != nil {
		t.Fatalf("CreateEmailIdentity: %v", err)
	}

	// Get email identity.
	getResp, err := client.GetEmailIdentity(ctx, &sesv2.GetEmailIdentityInput{
		EmailIdentity: aws.String("sender@example.com"),
	})
	if err != nil {
		t.Fatalf("GetEmailIdentity: %v", err)
	}
	if !getResp.VerifiedForSendingStatus {
		t.Error("expected VerifiedForSendingStatus to be true")
	}

	// List email identities.
	listResp, err := client.ListEmailIdentities(ctx, &sesv2.ListEmailIdentitiesInput{})
	if err != nil {
		t.Fatalf("ListEmailIdentities: %v", err)
	}
	if len(listResp.EmailIdentities) != 1 {
		t.Fatalf("expected 1 identity, got %d", len(listResp.EmailIdentities))
	}

	// Send email.
	sendResp, err := client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination: &sesv2types.Destination{
			ToAddresses: []string{"recipient@example.com"},
		},
		Content: &sesv2types.EmailContent{
			Simple: &sesv2types.Message{
				Subject: &sesv2types.Content{Data: aws.String("Test Subject")},
				Body: &sesv2types.Body{
					Text: &sesv2types.Content{Data: aws.String("Test body")},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendEmail: %v", err)
	}
	if sendResp.MessageId == nil || *sendResp.MessageId == "" {
		t.Error("expected non-empty MessageId")
	}

	// Delete identity.
	_, err = client.DeleteEmailIdentity(ctx, &sesv2.DeleteEmailIdentityInput{
		EmailIdentity: aws.String("sender@example.com"),
	})
	if err != nil {
		t.Fatalf("DeleteEmailIdentity: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListEmailIdentities(ctx, &sesv2.ListEmailIdentitiesInput{})
	if err != nil {
		t.Fatalf("ListEmailIdentities after delete: %v", err)
	}
	if len(listResp.EmailIdentities) != 0 {
		t.Errorf("expected 0 identities after delete, got %d", len(listResp.EmailIdentities))
	}
}

// TestCognitoUserPoolOperations verifies that the mock Cognito Identity Provider
// service supports user pool and user management.
func TestCognitoUserPoolOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := cognitoidentityprovider.NewFromConfig(cfg)

	// Create user pool.
	createResp, err := client.CreateUserPool(ctx, &cognitoidentityprovider.CreateUserPoolInput{
		PoolName: aws.String("test-pool"),
	})
	if err != nil {
		t.Fatalf("CreateUserPool: %v", err)
	}
	if createResp.UserPool == nil || createResp.UserPool.Id == nil {
		t.Fatal("expected user pool with ID")
	}
	poolID := *createResp.UserPool.Id
	if *createResp.UserPool.Name != "test-pool" {
		t.Errorf("expected pool name test-pool, got %s", *createResp.UserPool.Name)
	}

	// Describe user pool.
	descResp, err := client.DescribeUserPool(ctx, &cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: aws.String(poolID),
	})
	if err != nil {
		t.Fatalf("DescribeUserPool: %v", err)
	}
	if *descResp.UserPool.Name != "test-pool" {
		t.Errorf("expected pool name test-pool, got %s", *descResp.UserPool.Name)
	}

	// Create user pool client.
	clientResp, err := client.CreateUserPoolClient(ctx, &cognitoidentityprovider.CreateUserPoolClientInput{
		UserPoolId: aws.String(poolID),
		ClientName: aws.String("test-client"),
	})
	if err != nil {
		t.Fatalf("CreateUserPoolClient: %v", err)
	}
	if clientResp.UserPoolClient == nil || clientResp.UserPoolClient.ClientId == nil {
		t.Fatal("expected client with ID")
	}

	// Admin create user.
	userResp, err := client.AdminCreateUser(ctx, &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId: aws.String(poolID),
		Username:   aws.String("testuser"),
		UserAttributes: []cidptypes.AttributeType{
			{Name: aws.String("email"), Value: aws.String("test@example.com")},
		},
	})
	if err != nil {
		t.Fatalf("AdminCreateUser: %v", err)
	}
	if *userResp.User.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", *userResp.User.Username)
	}

	// Admin get user.
	getResp, err := client.AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(poolID),
		Username:   aws.String("testuser"),
	})
	if err != nil {
		t.Fatalf("AdminGetUser: %v", err)
	}
	if *getResp.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", *getResp.Username)
	}

	// List users.
	listResp, err := client.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		UserPoolId: aws.String(poolID),
	})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(listResp.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(listResp.Users))
	}

	// Admin delete user.
	_, err = client.AdminDeleteUser(ctx, &cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(poolID),
		Username:   aws.String("testuser"),
	})
	if err != nil {
		t.Fatalf("AdminDeleteUser: %v", err)
	}

	// List user pools.
	poolsResp, err := client.ListUserPools(ctx, &cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("ListUserPools: %v", err)
	}
	if len(poolsResp.UserPools) != 1 {
		t.Errorf("expected 1 pool, got %d", len(poolsResp.UserPools))
	}

	// Delete user pool.
	_, err = client.DeleteUserPool(ctx, &cognitoidentityprovider.DeleteUserPoolInput{
		UserPoolId: aws.String(poolID),
	})
	if err != nil {
		t.Fatalf("DeleteUserPool: %v", err)
	}

	// Verify empty.
	poolsResp, err = client.ListUserPools(ctx, &cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("ListUserPools after delete: %v", err)
	}
	if len(poolsResp.UserPools) != 0 {
		t.Errorf("expected 0 pools after delete, got %d", len(poolsResp.UserPools))
	}
}

// TestAPIGatewayV2Operations verifies that the mock API Gateway V2
// service supports API, stage, and route management.
func TestAPIGatewayV2Operations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := apigatewayv2.NewFromConfig(cfg)

	// Create API.
	createResp, err := client.CreateApi(ctx, &apigatewayv2.CreateApiInput{
		Name:         aws.String("test-api"),
		ProtocolType: "HTTP",
	})
	if err != nil {
		t.Fatalf("CreateApi: %v", err)
	}
	if createResp.ApiId == nil || *createResp.ApiId == "" {
		t.Fatal("expected API with ID")
	}
	apiID := *createResp.ApiId

	// Get API.
	getResp, err := client.GetApi(ctx, &apigatewayv2.GetApiInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		t.Fatalf("GetApi: %v", err)
	}
	if *getResp.Name != "test-api" {
		t.Errorf("expected API name test-api, got %s", *getResp.Name)
	}

	// Create stage.
	stageResp, err := client.CreateStage(ctx, &apigatewayv2.CreateStageInput{
		ApiId:     aws.String(apiID),
		StageName: aws.String("prod"),
	})
	if err != nil {
		t.Fatalf("CreateStage: %v", err)
	}
	if *stageResp.StageName != "prod" {
		t.Errorf("expected stage name prod, got %s", *stageResp.StageName)
	}

	// Get stages.
	stagesResp, err := client.GetStages(ctx, &apigatewayv2.GetStagesInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		t.Fatalf("GetStages: %v", err)
	}
	if len(stagesResp.Items) != 1 {
		t.Errorf("expected 1 stage, got %d", len(stagesResp.Items))
	}

	// Create route.
	routeResp, err := client.CreateRoute(ctx, &apigatewayv2.CreateRouteInput{
		ApiId:    aws.String(apiID),
		RouteKey: aws.String("GET /items"),
	})
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	if routeResp.RouteId == nil || *routeResp.RouteId == "" {
		t.Fatal("expected route with ID")
	}

	// Get routes.
	routesResp, err := client.GetRoutes(ctx, &apigatewayv2.GetRoutesInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		t.Fatalf("GetRoutes: %v", err)
	}
	if len(routesResp.Items) != 1 {
		t.Errorf("expected 1 route, got %d", len(routesResp.Items))
	}

	// List APIs.
	apisResp, err := client.GetApis(ctx, &apigatewayv2.GetApisInput{})
	if err != nil {
		t.Fatalf("GetApis: %v", err)
	}
	if len(apisResp.Items) != 1 {
		t.Errorf("expected 1 API, got %d", len(apisResp.Items))
	}

	// Delete API (cascades stages and routes).
	_, err = client.DeleteApi(ctx, &apigatewayv2.DeleteApiInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		t.Fatalf("DeleteApi: %v", err)
	}

	// Verify empty.
	apisResp, err = client.GetApis(ctx, &apigatewayv2.GetApisInput{})
	if err != nil {
		t.Fatalf("GetApis after delete: %v", err)
	}
	if len(apisResp.Items) != 0 {
		t.Errorf("expected 0 APIs after delete, got %d", len(apisResp.Items))
	}
}

// TestCloudFrontDistributionOperations verifies that the mock CloudFront
// service supports distribution CRUD operations.
func TestCloudFrontDistributionOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := cloudfront.NewFromConfig(cfg)

	// Create distribution.
	createResp, err := client.CreateDistribution(ctx, &cloudfront.CreateDistributionInput{
		DistributionConfig: &cftypes.DistributionConfig{
			CallerReference: aws.String("test-ref-1"),
			Comment:         aws.String("test distribution"),
			Enabled:         aws.Bool(true),
			Origins: &cftypes.Origins{
				Quantity: aws.Int32(1),
				Items: []cftypes.Origin{
					{
						DomainName: aws.String("mybucket.s3.amazonaws.com"),
						Id:         aws.String("S3Origin"),
					},
				},
			},
			DefaultCacheBehavior: &cftypes.DefaultCacheBehavior{
				TargetOriginId:       aws.String("S3Origin"),
				ViewerProtocolPolicy: cftypes.ViewerProtocolPolicyAllowAll,
				ForwardedValues: &cftypes.ForwardedValues{
					QueryString: aws.Bool(false),
					Cookies: &cftypes.CookiePreference{
						Forward: cftypes.ItemSelectionNone,
					},
				},
				MinTTL: aws.Int64(0),
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateDistribution: %v", err)
	}
	if createResp.Distribution == nil || createResp.Distribution.Id == nil {
		t.Fatal("expected distribution with ID")
	}
	distID := *createResp.Distribution.Id

	// Get distribution.
	getResp, err := client.GetDistribution(ctx, &cloudfront.GetDistributionInput{
		Id: aws.String(distID),
	})
	if err != nil {
		t.Fatalf("GetDistribution: %v", err)
	}
	if *getResp.Distribution.Id != distID {
		t.Errorf("expected dist ID %s, got %s", distID, *getResp.Distribution.Id)
	}

	// List distributions.
	listResp, err := client.ListDistributions(ctx, &cloudfront.ListDistributionsInput{})
	if err != nil {
		t.Fatalf("ListDistributions: %v", err)
	}
	if listResp.DistributionList == nil || len(listResp.DistributionList.Items) != 1 {
		t.Errorf("expected 1 distribution in list")
	}

	// Delete distribution.
	_, err = client.DeleteDistribution(ctx, &cloudfront.DeleteDistributionInput{
		Id:      aws.String(distID),
		IfMatch: getResp.ETag,
	})
	if err != nil {
		t.Fatalf("DeleteDistribution: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListDistributions(ctx, &cloudfront.ListDistributionsInput{})
	if err != nil {
		t.Fatalf("ListDistributions after delete: %v", err)
	}
	if listResp.DistributionList != nil && len(listResp.DistributionList.Items) != 0 {
		t.Errorf("expected 0 distributions after delete, got %d", len(listResp.DistributionList.Items))
	}
}

// TestEKSClusterOperations verifies that the mock EKS service supports
// cluster and nodegroup management.
func TestEKSClusterOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := eks.NewFromConfig(cfg)

	// Create cluster.
	createResp, err := client.CreateCluster(ctx, &eks.CreateClusterInput{
		Name:    aws.String("test-cluster"),
		Version: aws.String("1.29"),
		RoleArn: aws.String("arn:aws:iam::123456789012:role/eks-role"),
		ResourcesVpcConfig: &ekstypes.VpcConfigRequest{
			SubnetIds: []string{"subnet-123"},
		},
	})
	if err != nil {
		t.Fatalf("CreateCluster: %v", err)
	}
	if createResp.Cluster == nil || *createResp.Cluster.Name != "test-cluster" {
		t.Fatal("expected cluster with name test-cluster")
	}

	// Describe cluster.
	descResp, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String("test-cluster"),
	})
	if err != nil {
		t.Fatalf("DescribeCluster: %v", err)
	}
	if *descResp.Cluster.Version != "1.29" {
		t.Errorf("expected version 1.29, got %s", *descResp.Cluster.Version)
	}

	// Create nodegroup.
	ngResp, err := client.CreateNodegroup(ctx, &eks.CreateNodegroupInput{
		ClusterName:   aws.String("test-cluster"),
		NodegroupName: aws.String("test-ng"),
		NodeRole:      aws.String("arn:aws:iam::123456789012:role/node-role"),
		Subnets:       []string{"subnet-123"},
	})
	if err != nil {
		t.Fatalf("CreateNodegroup: %v", err)
	}
	if *ngResp.Nodegroup.NodegroupName != "test-ng" {
		t.Errorf("expected nodegroup name test-ng, got %s", *ngResp.Nodegroup.NodegroupName)
	}

	// List nodegroups.
	ngListResp, err := client.ListNodegroups(ctx, &eks.ListNodegroupsInput{
		ClusterName: aws.String("test-cluster"),
	})
	if err != nil {
		t.Fatalf("ListNodegroups: %v", err)
	}
	if len(ngListResp.Nodegroups) != 1 {
		t.Errorf("expected 1 nodegroup, got %d", len(ngListResp.Nodegroups))
	}

	// Delete nodegroup.
	_, err = client.DeleteNodegroup(ctx, &eks.DeleteNodegroupInput{
		ClusterName:   aws.String("test-cluster"),
		NodegroupName: aws.String("test-ng"),
	})
	if err != nil {
		t.Fatalf("DeleteNodegroup: %v", err)
	}

	// List clusters.
	clustersResp, err := client.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(clustersResp.Clusters) != 1 {
		t.Errorf("expected 1 cluster, got %d", len(clustersResp.Clusters))
	}

	// Delete cluster.
	_, err = client.DeleteCluster(ctx, &eks.DeleteClusterInput{
		Name: aws.String("test-cluster"),
	})
	if err != nil {
		t.Fatalf("DeleteCluster: %v", err)
	}

	// Verify empty.
	clustersResp, err = client.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		t.Fatalf("ListClusters after delete: %v", err)
	}
	if len(clustersResp.Clusters) != 0 {
		t.Errorf("expected 0 clusters after delete, got %d", len(clustersResp.Clusters))
	}
}

// TestElastiCacheClusterOperations verifies that the mock ElastiCache
// service supports cache cluster CRUD operations.
func TestElastiCacheClusterOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := elasticache.NewFromConfig(cfg)

	// Create cache cluster.
	createResp, err := client.CreateCacheCluster(ctx, &elasticache.CreateCacheClusterInput{
		CacheClusterId: aws.String("test-cache"),
		Engine:         aws.String("redis"),
		CacheNodeType:  aws.String("cache.t3.micro"),
		NumCacheNodes:  aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("CreateCacheCluster: %v", err)
	}
	if createResp.CacheCluster == nil || *createResp.CacheCluster.CacheClusterId != "test-cache" {
		t.Fatal("expected cache cluster with ID test-cache")
	}

	// Describe cache clusters.
	descResp, err := client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
		CacheClusterId: aws.String("test-cache"),
	})
	if err != nil {
		t.Fatalf("DescribeCacheClusters: %v", err)
	}
	if len(descResp.CacheClusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(descResp.CacheClusters))
	}
	if *descResp.CacheClusters[0].Engine != "redis" {
		t.Errorf("expected engine redis, got %s", *descResp.CacheClusters[0].Engine)
	}

	// Delete cache cluster.
	_, err = client.DeleteCacheCluster(ctx, &elasticache.DeleteCacheClusterInput{
		CacheClusterId: aws.String("test-cache"),
	})
	if err != nil {
		t.Fatalf("DeleteCacheCluster: %v", err)
	}

	// Verify empty.
	descResp, err = client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{})
	if err != nil {
		t.Fatalf("DescribeCacheClusters after delete: %v", err)
	}
	if len(descResp.CacheClusters) != 0 {
		t.Errorf("expected 0 clusters after delete, got %d", len(descResp.CacheClusters))
	}
}

// TestFirehoseDeliveryStreamOperations verifies that the mock Firehose
// service supports delivery stream management and record delivery.
func TestFirehoseDeliveryStreamOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := firehose.NewFromConfig(cfg)

	// Create delivery stream.
	createResp, err := client.CreateDeliveryStream(ctx, &firehose.CreateDeliveryStreamInput{
		DeliveryStreamName: aws.String("test-stream"),
	})
	if err != nil {
		t.Fatalf("CreateDeliveryStream: %v", err)
	}
	if createResp.DeliveryStreamARN == nil || *createResp.DeliveryStreamARN == "" {
		t.Fatal("expected delivery stream ARN")
	}

	// Describe delivery stream.
	descResp, err := client.DescribeDeliveryStream(ctx, &firehose.DescribeDeliveryStreamInput{
		DeliveryStreamName: aws.String("test-stream"),
	})
	if err != nil {
		t.Fatalf("DescribeDeliveryStream: %v", err)
	}
	if *descResp.DeliveryStreamDescription.DeliveryStreamName != "test-stream" {
		t.Errorf("expected stream name test-stream, got %s",
			*descResp.DeliveryStreamDescription.DeliveryStreamName)
	}

	// Put record.
	putResp, err := client.PutRecord(ctx, &firehose.PutRecordInput{
		DeliveryStreamName: aws.String("test-stream"),
		Record: &firehosetypes.Record{
			Data: []byte("hello world"),
		},
	})
	if err != nil {
		t.Fatalf("PutRecord: %v", err)
	}
	if putResp.RecordId == nil || *putResp.RecordId == "" {
		t.Error("expected non-empty RecordId")
	}

	// List delivery streams.
	listResp, err := client.ListDeliveryStreams(ctx, &firehose.ListDeliveryStreamsInput{})
	if err != nil {
		t.Fatalf("ListDeliveryStreams: %v", err)
	}
	if len(listResp.DeliveryStreamNames) != 1 {
		t.Errorf("expected 1 stream, got %d", len(listResp.DeliveryStreamNames))
	}

	// Delete delivery stream.
	_, err = client.DeleteDeliveryStream(ctx, &firehose.DeleteDeliveryStreamInput{
		DeliveryStreamName: aws.String("test-stream"),
	})
	if err != nil {
		t.Fatalf("DeleteDeliveryStream: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListDeliveryStreams(ctx, &firehose.ListDeliveryStreamsInput{})
	if err != nil {
		t.Fatalf("ListDeliveryStreams after delete: %v", err)
	}
	if len(listResp.DeliveryStreamNames) != 0 {
		t.Errorf("expected 0 streams after delete, got %d", len(listResp.DeliveryStreamNames))
	}
}

// TestAthenaQueryOperations verifies that the mock Athena
// service supports query execution and workgroup management.
func TestAthenaQueryOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := athena.NewFromConfig(cfg)

	// Create workgroup.
	_, err = client.CreateWorkGroup(ctx, &athena.CreateWorkGroupInput{
		Name:        aws.String("test-wg"),
		Description: aws.String("test workgroup"),
	})
	if err != nil {
		t.Fatalf("CreateWorkGroup: %v", err)
	}

	// List workgroups.
	wgResp, err := client.ListWorkGroups(ctx, &athena.ListWorkGroupsInput{})
	if err != nil {
		t.Fatalf("ListWorkGroups: %v", err)
	}
	if len(wgResp.WorkGroups) < 2 { // primary + test-wg
		t.Errorf("expected at least 2 workgroups, got %d", len(wgResp.WorkGroups))
	}

	// Start query execution.
	startResp, err := client.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
		QueryString: aws.String("SELECT 1"),
		ResultConfiguration: &athenatypes.ResultConfiguration{
			OutputLocation: aws.String("s3://test-bucket/results/"),
		},
	})
	if err != nil {
		t.Fatalf("StartQueryExecution: %v", err)
	}
	if startResp.QueryExecutionId == nil || *startResp.QueryExecutionId == "" {
		t.Fatal("expected query execution ID")
	}
	execID := *startResp.QueryExecutionId

	// Get query execution.
	getResp, err := client.GetQueryExecution(ctx, &athena.GetQueryExecutionInput{
		QueryExecutionId: aws.String(execID),
	})
	if err != nil {
		t.Fatalf("GetQueryExecution: %v", err)
	}
	if *getResp.QueryExecution.Query != "SELECT 1" {
		t.Errorf("expected query 'SELECT 1', got %s", *getResp.QueryExecution.Query)
	}

	// Get query results.
	resultsResp, err := client.GetQueryResults(ctx, &athena.GetQueryResultsInput{
		QueryExecutionId: aws.String(execID),
	})
	if err != nil {
		t.Fatalf("GetQueryResults: %v", err)
	}
	if resultsResp.ResultSet == nil {
		t.Error("expected result set")
	}

	// List query executions.
	listResp, err := client.ListQueryExecutions(ctx, &athena.ListQueryExecutionsInput{})
	if err != nil {
		t.Fatalf("ListQueryExecutions: %v", err)
	}
	if len(listResp.QueryExecutionIds) != 1 {
		t.Errorf("expected 1 query execution, got %d", len(listResp.QueryExecutionIds))
	}

	// Delete workgroup.
	_, err = client.DeleteWorkGroup(ctx, &athena.DeleteWorkGroupInput{
		WorkGroup: aws.String("test-wg"),
	})
	if err != nil {
		t.Fatalf("DeleteWorkGroup: %v", err)
	}
}

// TestGlueDatabaseAndTableOperations verifies that the mock Glue
// service supports database, table, and crawler management.
func TestGlueDatabaseAndTableOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := glue.NewFromConfig(cfg)

	// Create database.
	_, err = client.CreateDatabase(ctx, &glue.CreateDatabaseInput{
		DatabaseInput: &gluetypes.DatabaseInput{
			Name:        aws.String("test-db"),
			Description: aws.String("test database"),
		},
	})
	if err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}

	// Get database.
	dbResp, err := client.GetDatabase(ctx, &glue.GetDatabaseInput{
		Name: aws.String("test-db"),
	})
	if err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	if *dbResp.Database.Name != "test-db" {
		t.Errorf("expected database name test-db, got %s", *dbResp.Database.Name)
	}

	// Create table.
	_, err = client.CreateTable(ctx, &glue.CreateTableInput{
		DatabaseName: aws.String("test-db"),
		TableInput: &gluetypes.TableInput{
			Name:      aws.String("test-table"),
			TableType: aws.String("EXTERNAL_TABLE"),
			StorageDescriptor: &gluetypes.StorageDescriptor{
				Location: aws.String("s3://bucket/prefix/"),
				Columns: []gluetypes.Column{
					{Name: aws.String("id"), Type: aws.String("int")},
					{Name: aws.String("name"), Type: aws.String("string")},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	// Get table.
	tableResp, err := client.GetTable(ctx, &glue.GetTableInput{
		DatabaseName: aws.String("test-db"),
		Name:         aws.String("test-table"),
	})
	if err != nil {
		t.Fatalf("GetTable: %v", err)
	}
	if *tableResp.Table.Name != "test-table" {
		t.Errorf("expected table name test-table, got %s", *tableResp.Table.Name)
	}

	// Get tables.
	tablesResp, err := client.GetTables(ctx, &glue.GetTablesInput{
		DatabaseName: aws.String("test-db"),
	})
	if err != nil {
		t.Fatalf("GetTables: %v", err)
	}
	if len(tablesResp.TableList) != 1 {
		t.Errorf("expected 1 table, got %d", len(tablesResp.TableList))
	}

	// Create crawler.
	_, err = client.CreateCrawler(ctx, &glue.CreateCrawlerInput{
		Name:         aws.String("test-crawler"),
		Role:         aws.String("arn:aws:iam::123456789012:role/glue-role"),
		DatabaseName: aws.String("test-db"),
		Targets: &gluetypes.CrawlerTargets{
			S3Targets: []gluetypes.S3Target{
				{Path: aws.String("s3://bucket/prefix/")},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateCrawler: %v", err)
	}

	// Get crawler.
	crawlerResp, err := client.GetCrawler(ctx, &glue.GetCrawlerInput{
		Name: aws.String("test-crawler"),
	})
	if err != nil {
		t.Fatalf("GetCrawler: %v", err)
	}
	if *crawlerResp.Crawler.Name != "test-crawler" {
		t.Errorf("expected crawler name test-crawler, got %s", *crawlerResp.Crawler.Name)
	}

	// Delete table.
	_, err = client.DeleteTable(ctx, &glue.DeleteTableInput{
		DatabaseName: aws.String("test-db"),
		Name:         aws.String("test-table"),
	})
	if err != nil {
		t.Fatalf("DeleteTable: %v", err)
	}

	// Delete crawler.
	_, err = client.DeleteCrawler(ctx, &glue.DeleteCrawlerInput{
		Name: aws.String("test-crawler"),
	})
	if err != nil {
		t.Fatalf("DeleteCrawler: %v", err)
	}

	// Delete database.
	_, err = client.DeleteDatabase(ctx, &glue.DeleteDatabaseInput{
		Name: aws.String("test-db"),
	})
	if err != nil {
		t.Fatalf("DeleteDatabase: %v", err)
	}

	// Verify empty.
	dbsResp, err := client.GetDatabases(ctx, &glue.GetDatabasesInput{})
	if err != nil {
		t.Fatalf("GetDatabases after delete: %v", err)
	}
	if len(dbsResp.DatabaseList) != 0 {
		t.Errorf("expected 0 databases after delete, got %d", len(dbsResp.DatabaseList))
	}
}

// ─── Auto Scaling ───────────────────────────────────────────────────────────

func TestAutoScalingGroupOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := autoscaling.NewFromConfig(cfg)

	// Create launch configuration.
	_, err = client.CreateLaunchConfiguration(ctx, &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName: aws.String("test-lc"),
		ImageId:                 aws.String("ami-12345678"),
		InstanceType:            aws.String("t2.micro"),
	})
	if err != nil {
		t.Fatalf("CreateLaunchConfiguration: %v", err)
	}

	// Create auto scaling group.
	_, err = client.CreateAutoScalingGroup(ctx, &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:    aws.String("test-asg"),
		LaunchConfigurationName: aws.String("test-lc"),
		MinSize:                 aws.Int32(1),
		MaxSize:                 aws.Int32(3),
		DesiredCapacity:         aws.Int32(2),
	})
	if err != nil {
		t.Fatalf("CreateAutoScalingGroup: %v", err)
	}

	// Describe auto scaling groups.
	descResp, err := client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		t.Fatalf("DescribeAutoScalingGroups: %v", err)
	}
	if len(descResp.AutoScalingGroups) != 1 {
		t.Fatalf("expected 1 ASG, got %d", len(descResp.AutoScalingGroups))
	}
	if *descResp.AutoScalingGroups[0].AutoScalingGroupName != "test-asg" {
		t.Errorf("expected ASG name test-asg, got %s", *descResp.AutoScalingGroups[0].AutoScalingGroupName)
	}

	// Update auto scaling group.
	_, err = client.UpdateAutoScalingGroup(ctx, &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String("test-asg"),
		MaxSize:              aws.Int32(5),
	})
	if err != nil {
		t.Fatalf("UpdateAutoScalingGroup: %v", err)
	}

	// Verify update.
	descResp, err = client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{"test-asg"},
	})
	if err != nil {
		t.Fatalf("DescribeAutoScalingGroups after update: %v", err)
	}
	if len(descResp.AutoScalingGroups) != 1 {
		t.Fatalf("expected 1 ASG after update, got %d", len(descResp.AutoScalingGroups))
	}

	// Delete auto scaling group.
	_, err = client.DeleteAutoScalingGroup(ctx, &autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: aws.String("test-asg"),
	})
	if err != nil {
		t.Fatalf("DeleteAutoScalingGroup: %v", err)
	}

	// Delete launch configuration.
	_, err = client.DeleteLaunchConfiguration(ctx, &autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: aws.String("test-lc"),
	})
	if err != nil {
		t.Fatalf("DeleteLaunchConfiguration: %v", err)
	}

	// Verify empty.
	descResp, err = client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		t.Fatalf("DescribeAutoScalingGroups after delete: %v", err)
	}
	if len(descResp.AutoScalingGroups) != 0 {
		t.Errorf("expected 0 ASGs after delete, got %d", len(descResp.AutoScalingGroups))
	}
}

// ─── API Gateway V1 ─────────────────────────────────────────────────────────

func TestAPIGatewayV1Operations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := apigateway.NewFromConfig(cfg)

	// Create REST API.
	createResp, err := client.CreateRestApi(ctx, &apigateway.CreateRestApiInput{
		Name:        aws.String("test-rest-api"),
		Description: aws.String("A test REST API"),
	})
	if err != nil {
		t.Fatalf("CreateRestApi: %v", err)
	}
	if createResp.Id == nil || *createResp.Id == "" {
		t.Fatal("expected REST API with ID")
	}
	apiID := *createResp.Id

	// Get REST API.
	getResp, err := client.GetRestApi(ctx, &apigateway.GetRestApiInput{
		RestApiId: aws.String(apiID),
	})
	if err != nil {
		t.Fatalf("GetRestApi: %v", err)
	}
	if *getResp.Name != "test-rest-api" {
		t.Errorf("expected name test-rest-api, got %s", *getResp.Name)
	}

	// List REST APIs.
	listResp, err := client.GetRestApis(ctx, &apigateway.GetRestApisInput{})
	if err != nil {
		t.Fatalf("GetRestApis: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Errorf("expected 1 REST API, got %d", len(listResp.Items))
	}

	// Delete REST API.
	_, err = client.DeleteRestApi(ctx, &apigateway.DeleteRestApiInput{
		RestApiId: aws.String(apiID),
	})
	if err != nil {
		t.Fatalf("DeleteRestApi: %v", err)
	}

	// Verify empty.
	listResp, err = client.GetRestApis(ctx, &apigateway.GetRestApisInput{})
	if err != nil {
		t.Fatalf("GetRestApis after delete: %v", err)
	}
	if len(listResp.Items) != 0 {
		t.Errorf("expected 0 REST APIs after delete, got %d", len(listResp.Items))
	}
}

// ─── Cognito Identity ───────────────────────────────────────────────────────

func TestCognitoIdentityPoolOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := cognitoidentity.NewFromConfig(cfg)

	// Create identity pool.
	createResp, err := client.CreateIdentityPool(ctx, &cognitoidentity.CreateIdentityPoolInput{
		IdentityPoolName:               aws.String("test-identity-pool"),
		AllowUnauthenticatedIdentities: true,
	})
	if err != nil {
		t.Fatalf("CreateIdentityPool: %v", err)
	}
	if createResp.IdentityPoolId == nil || *createResp.IdentityPoolId == "" {
		t.Fatal("expected identity pool with ID")
	}
	poolID := *createResp.IdentityPoolId

	// Describe identity pool.
	descResp, err := client.DescribeIdentityPool(ctx, &cognitoidentity.DescribeIdentityPoolInput{
		IdentityPoolId: aws.String(poolID),
	})
	if err != nil {
		t.Fatalf("DescribeIdentityPool: %v", err)
	}
	if *descResp.IdentityPoolName != "test-identity-pool" {
		t.Errorf("expected pool name test-identity-pool, got %s", *descResp.IdentityPoolName)
	}

	// List identity pools.
	listResp, err := client.ListIdentityPools(ctx, &cognitoidentity.ListIdentityPoolsInput{
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("ListIdentityPools: %v", err)
	}
	if len(listResp.IdentityPools) != 1 {
		t.Errorf("expected 1 identity pool, got %d", len(listResp.IdentityPools))
	}

	// Update identity pool.
	_, err = client.UpdateIdentityPool(ctx, &cognitoidentity.UpdateIdentityPoolInput{
		IdentityPoolId:                 aws.String(poolID),
		IdentityPoolName:               aws.String("updated-pool"),
		AllowUnauthenticatedIdentities: false,
	})
	if err != nil {
		t.Fatalf("UpdateIdentityPool: %v", err)
	}

	// Delete identity pool.
	_, err = client.DeleteIdentityPool(ctx, &cognitoidentity.DeleteIdentityPoolInput{
		IdentityPoolId: aws.String(poolID),
	})
	if err != nil {
		t.Fatalf("DeleteIdentityPool: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListIdentityPools(ctx, &cognitoidentity.ListIdentityPoolsInput{
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("ListIdentityPools after delete: %v", err)
	}
	if len(listResp.IdentityPools) != 0 {
		t.Errorf("expected 0 identity pools after delete, got %d", len(listResp.IdentityPools))
	}
}

// ─── Organizations ──────────────────────────────────────────────────────────

func TestOrganizationsOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := organizations.NewFromConfig(cfg)

	// Create organization.
	createResp, err := client.CreateOrganization(ctx, &organizations.CreateOrganizationInput{})
	if err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}
	if createResp.Organization == nil {
		t.Fatal("expected organization in response")
	}
	if createResp.Organization.Id == nil || *createResp.Organization.Id == "" {
		t.Error("expected non-empty organization ID")
	}

	// Describe organization.
	descResp, err := client.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err != nil {
		t.Fatalf("DescribeOrganization: %v", err)
	}
	if descResp.Organization == nil {
		t.Fatal("expected organization in describe response")
	}

	// List accounts.
	listResp, err := client.ListAccounts(ctx, &organizations.ListAccountsInput{})
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if listResp.Accounts == nil {
		t.Error("expected non-nil accounts list")
	}
}

// ─── DynamoDB Streams ───────────────────────────────────────────────────────

func TestDynamoDBStreamsOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := dynamodbstreams.NewFromConfig(cfg)

	// List streams (expect empty).
	listResp, err := client.ListStreams(ctx, &dynamodbstreams.ListStreamsInput{})
	if err != nil {
		t.Fatalf("ListStreams: %v", err)
	}
	if listResp.Streams == nil {
		t.Error("expected non-nil streams list")
	}
}

// ─── EFS ────────────────────────────────────────────────────────────────────

func TestEFSFileSystemOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := efs.NewFromConfig(cfg)

	// Create file system.
	createResp, err := client.CreateFileSystem(ctx, &efs.CreateFileSystemInput{
		CreationToken: aws.String("test-fs-token"),
	})
	if err != nil {
		t.Fatalf("CreateFileSystem: %v", err)
	}
	if createResp.FileSystemId == nil || *createResp.FileSystemId == "" {
		t.Fatal("expected file system with ID")
	}
	fsID := *createResp.FileSystemId

	// Describe file systems.
	descResp, err := client.DescribeFileSystems(ctx, &efs.DescribeFileSystemsInput{})
	if err != nil {
		t.Fatalf("DescribeFileSystems: %v", err)
	}
	if len(descResp.FileSystems) != 1 {
		t.Fatalf("expected 1 file system, got %d", len(descResp.FileSystems))
	}
	if *descResp.FileSystems[0].FileSystemId != fsID {
		t.Errorf("expected file system ID %s, got %s", fsID, *descResp.FileSystems[0].FileSystemId)
	}

	// Delete file system.
	_, err = client.DeleteFileSystem(ctx, &efs.DeleteFileSystemInput{
		FileSystemId: aws.String(fsID),
	})
	if err != nil {
		t.Fatalf("DeleteFileSystem: %v", err)
	}

	// Verify empty.
	descResp, err = client.DescribeFileSystems(ctx, &efs.DescribeFileSystemsInput{})
	if err != nil {
		t.Fatalf("DescribeFileSystems after delete: %v", err)
	}
	if len(descResp.FileSystems) != 0 {
		t.Errorf("expected 0 file systems after delete, got %d", len(descResp.FileSystems))
	}
}

// ─── Batch ──────────────────────────────────────────────────────────────────

func TestBatchComputeEnvironmentOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := batch.NewFromConfig(cfg)

	// Create compute environment.
	createResp, err := client.CreateComputeEnvironment(ctx, &batch.CreateComputeEnvironmentInput{
		ComputeEnvironmentName: aws.String("test-compute-env"),
		Type:                   batchtypes.CETypeManaged,
		State:                  batchtypes.CEStateEnabled,
	})
	if err != nil {
		t.Fatalf("CreateComputeEnvironment: %v", err)
	}
	if createResp.ComputeEnvironmentArn == nil || *createResp.ComputeEnvironmentArn == "" {
		t.Error("expected non-empty compute environment ARN")
	}

	// Describe compute environments.
	descResp, err := client.DescribeComputeEnvironments(ctx, &batch.DescribeComputeEnvironmentsInput{})
	if err != nil {
		t.Fatalf("DescribeComputeEnvironments: %v", err)
	}
	if len(descResp.ComputeEnvironments) != 1 {
		t.Fatalf("expected 1 compute environment, got %d", len(descResp.ComputeEnvironments))
	}
	if *descResp.ComputeEnvironments[0].ComputeEnvironmentName != "test-compute-env" {
		t.Errorf("expected name test-compute-env, got %s", *descResp.ComputeEnvironments[0].ComputeEnvironmentName)
	}

	// Delete compute environment.
	_, err = client.DeleteComputeEnvironment(ctx, &batch.DeleteComputeEnvironmentInput{
		ComputeEnvironment: aws.String("test-compute-env"),
	})
	if err != nil {
		t.Fatalf("DeleteComputeEnvironment: %v", err)
	}

	// Verify empty.
	descResp, err = client.DescribeComputeEnvironments(ctx, &batch.DescribeComputeEnvironmentsInput{})
	if err != nil {
		t.Fatalf("DescribeComputeEnvironments after delete: %v", err)
	}
	if len(descResp.ComputeEnvironments) != 0 {
		t.Errorf("expected 0 compute environments after delete, got %d", len(descResp.ComputeEnvironments))
	}
}

// ─── CodeBuild ──────────────────────────────────────────────────────────────

func TestCodeBuildProjectOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := codebuild.NewFromConfig(cfg)

	// Create project.
	createResp, err := client.CreateProject(ctx, &codebuild.CreateProjectInput{
		Name: aws.String("test-project"),
		Source: &codebuildtypes.ProjectSource{
			Type:     codebuildtypes.SourceTypeCodecommit,
			Location: aws.String("https://git-codecommit.us-east-1.amazonaws.com/v1/repos/my-repo"),
		},
		Artifacts: &codebuildtypes.ProjectArtifacts{
			Type: codebuildtypes.ArtifactsTypeNoArtifacts,
		},
		Environment: &codebuildtypes.ProjectEnvironment{
			Type:        codebuildtypes.EnvironmentTypeLinuxContainer,
			Image:       aws.String("aws/codebuild/standard:5.0"),
			ComputeType: codebuildtypes.ComputeTypeBuildGeneral1Small,
		},
		ServiceRole: aws.String("arn:aws:iam::123456789012:role/codebuild-role"),
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if createResp.Project == nil || createResp.Project.Name == nil {
		t.Fatal("expected project with name")
	}
	if *createResp.Project.Name != "test-project" {
		t.Errorf("expected project name test-project, got %s", *createResp.Project.Name)
	}

	// List projects.
	listResp, err := client.ListProjects(ctx, &codebuild.ListProjectsInput{})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(listResp.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(listResp.Projects))
	}

	// Batch get projects.
	batchResp, err := client.BatchGetProjects(ctx, &codebuild.BatchGetProjectsInput{
		Names: []string{"test-project"},
	})
	if err != nil {
		t.Fatalf("BatchGetProjects: %v", err)
	}
	if len(batchResp.Projects) != 1 {
		t.Fatalf("expected 1 project in batch get, got %d", len(batchResp.Projects))
	}

	// Start build.
	buildResp, err := client.StartBuild(ctx, &codebuild.StartBuildInput{
		ProjectName: aws.String("test-project"),
	})
	if err != nil {
		t.Fatalf("StartBuild: %v", err)
	}
	if buildResp.Build == nil || buildResp.Build.Id == nil {
		t.Fatal("expected build with ID")
	}

	// Delete project.
	_, err = client.DeleteProject(ctx, &codebuild.DeleteProjectInput{
		Name: aws.String("test-project"),
	})
	if err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListProjects(ctx, &codebuild.ListProjectsInput{})
	if err != nil {
		t.Fatalf("ListProjects after delete: %v", err)
	}
	if len(listResp.Projects) != 0 {
		t.Errorf("expected 0 projects after delete, got %d", len(listResp.Projects))
	}
}

// ─── CodePipeline ───────────────────────────────────────────────────────────

func TestCodePipelineOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := codepipeline.NewFromConfig(cfg)

	// Create pipeline.
	createResp, err := client.CreatePipeline(ctx, &codepipeline.CreatePipelineInput{
		Pipeline: &codepipelinetypes.PipelineDeclaration{
			Name:    aws.String("test-pipeline"),
			RoleArn: aws.String("arn:aws:iam::123456789012:role/pipeline-role"),
			Stages: []codepipelinetypes.StageDeclaration{
				{
					Name: aws.String("Source"),
					Actions: []codepipelinetypes.ActionDeclaration{
						{
							Name: aws.String("SourceAction"),
							ActionTypeId: &codepipelinetypes.ActionTypeId{
								Category: codepipelinetypes.ActionCategorySource,
								Owner:    codepipelinetypes.ActionOwnerAws,
								Provider: aws.String("S3"),
								Version:  aws.String("1"),
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreatePipeline: %v", err)
	}
	if createResp.Pipeline == nil || createResp.Pipeline.Name == nil {
		t.Fatal("expected pipeline with name")
	}
	if *createResp.Pipeline.Name != "test-pipeline" {
		t.Errorf("expected pipeline name test-pipeline, got %s", *createResp.Pipeline.Name)
	}

	// Get pipeline.
	getResp, err := client.GetPipeline(ctx, &codepipeline.GetPipelineInput{
		Name: aws.String("test-pipeline"),
	})
	if err != nil {
		t.Fatalf("GetPipeline: %v", err)
	}
	if *getResp.Pipeline.Name != "test-pipeline" {
		t.Errorf("expected pipeline name test-pipeline, got %s", *getResp.Pipeline.Name)
	}

	// List pipelines.
	listResp, err := client.ListPipelines(ctx, &codepipeline.ListPipelinesInput{})
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}
	if len(listResp.Pipelines) != 1 {
		t.Errorf("expected 1 pipeline, got %d", len(listResp.Pipelines))
	}

	// Delete pipeline.
	_, err = client.DeletePipeline(ctx, &codepipeline.DeletePipelineInput{
		Name: aws.String("test-pipeline"),
	})
	if err != nil {
		t.Fatalf("DeletePipeline: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListPipelines(ctx, &codepipeline.ListPipelinesInput{})
	if err != nil {
		t.Fatalf("ListPipelines after delete: %v", err)
	}
	if len(listResp.Pipelines) != 0 {
		t.Errorf("expected 0 pipelines after delete, got %d", len(listResp.Pipelines))
	}
}

// ─── CloudTrail ─────────────────────────────────────────────────────────────

func TestCloudTrailOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := cloudtrail.NewFromConfig(cfg)

	// Create trail.
	createResp, err := client.CreateTrail(ctx, &cloudtrail.CreateTrailInput{
		Name:         aws.String("test-trail"),
		S3BucketName: aws.String("my-trail-bucket"),
	})
	if err != nil {
		t.Fatalf("CreateTrail: %v", err)
	}
	if createResp.Name == nil || *createResp.Name != "test-trail" {
		t.Errorf("expected trail name test-trail, got %v", createResp.Name)
	}

	// Describe trails.
	descResp, err := client.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{})
	if err != nil {
		t.Fatalf("DescribeTrails: %v", err)
	}
	if len(descResp.TrailList) != 1 {
		t.Fatalf("expected 1 trail, got %d", len(descResp.TrailList))
	}

	// Get trail.
	getResp, err := client.GetTrail(ctx, &cloudtrail.GetTrailInput{
		Name: aws.String("test-trail"),
	})
	if err != nil {
		t.Fatalf("GetTrail: %v", err)
	}
	if *getResp.Trail.Name != "test-trail" {
		t.Errorf("expected trail name test-trail, got %s", *getResp.Trail.Name)
	}

	// Start logging.
	_, err = client.StartLogging(ctx, &cloudtrail.StartLoggingInput{
		Name: aws.String("test-trail"),
	})
	if err != nil {
		t.Fatalf("StartLogging: %v", err)
	}

	// Get trail status.
	statusResp, err := client.GetTrailStatus(ctx, &cloudtrail.GetTrailStatusInput{
		Name: aws.String("test-trail"),
	})
	if err != nil {
		t.Fatalf("GetTrailStatus: %v", err)
	}
	if statusResp.IsLogging == nil || !*statusResp.IsLogging {
		t.Error("expected IsLogging to be true after StartLogging")
	}

	// Stop logging.
	_, err = client.StopLogging(ctx, &cloudtrail.StopLoggingInput{
		Name: aws.String("test-trail"),
	})
	if err != nil {
		t.Fatalf("StopLogging: %v", err)
	}

	// Delete trail.
	_, err = client.DeleteTrail(ctx, &cloudtrail.DeleteTrailInput{
		Name: aws.String("test-trail"),
	})
	if err != nil {
		t.Fatalf("DeleteTrail: %v", err)
	}

	// Verify empty.
	descResp, err = client.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{})
	if err != nil {
		t.Fatalf("DescribeTrails after delete: %v", err)
	}
	if len(descResp.TrailList) != 0 {
		t.Errorf("expected 0 trails after delete, got %d", len(descResp.TrailList))
	}
}

// ─── Config Service ─────────────────────────────────────────────────────────

func TestConfigServiceOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := configservice.NewFromConfig(cfg)

	// Put config rule.
	_, err = client.PutConfigRule(ctx, &configservice.PutConfigRuleInput{
		ConfigRule: &configtypes.ConfigRule{
			ConfigRuleName: aws.String("test-rule"),
			Source: &configtypes.Source{
				Owner:            configtypes.OwnerAws,
				SourceIdentifier: aws.String("S3_BUCKET_VERSIONING_ENABLED"),
			},
			Description: aws.String("Test config rule"),
		},
	})
	if err != nil {
		t.Fatalf("PutConfigRule: %v", err)
	}

	// Describe config rules.
	descResp, err := client.DescribeConfigRules(ctx, &configservice.DescribeConfigRulesInput{})
	if err != nil {
		t.Fatalf("DescribeConfigRules: %v", err)
	}
	if len(descResp.ConfigRules) != 1 {
		t.Fatalf("expected 1 config rule, got %d", len(descResp.ConfigRules))
	}
	if *descResp.ConfigRules[0].ConfigRuleName != "test-rule" {
		t.Errorf("expected rule name test-rule, got %s", *descResp.ConfigRules[0].ConfigRuleName)
	}

	// Delete config rule.
	_, err = client.DeleteConfigRule(ctx, &configservice.DeleteConfigRuleInput{
		ConfigRuleName: aws.String("test-rule"),
	})
	if err != nil {
		t.Fatalf("DeleteConfigRule: %v", err)
	}

	// Verify empty.
	descResp, err = client.DescribeConfigRules(ctx, &configservice.DescribeConfigRulesInput{})
	if err != nil {
		t.Fatalf("DescribeConfigRules after delete: %v", err)
	}
	if len(descResp.ConfigRules) != 0 {
		t.Errorf("expected 0 config rules after delete, got %d", len(descResp.ConfigRules))
	}
}

// ─── WAFv2 ──────────────────────────────────────────────────────────────────

func TestWAFv2WebACLOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := wafv2.NewFromConfig(cfg)

	// Create web ACL.
	createResp, err := client.CreateWebACL(ctx, &wafv2.CreateWebACLInput{
		Name:  aws.String("test-web-acl"),
		Scope: wafv2types.ScopeRegional,
		DefaultAction: &wafv2types.DefaultAction{
			Allow: &wafv2types.AllowAction{},
		},
		VisibilityConfig: &wafv2types.VisibilityConfig{
			CloudWatchMetricsEnabled: true,
			MetricName:               aws.String("test-metric"),
			SampledRequestsEnabled:   true,
		},
	})
	if err != nil {
		t.Fatalf("CreateWebACL: %v", err)
	}
	if createResp.Summary == nil || createResp.Summary.Id == nil {
		t.Fatal("expected web ACL summary with ID")
	}
	aclID := *createResp.Summary.Id
	lockToken := *createResp.Summary.LockToken

	// Get web ACL.
	getResp, err := client.GetWebACL(ctx, &wafv2.GetWebACLInput{
		Name:  aws.String("test-web-acl"),
		Scope: wafv2types.ScopeRegional,
		Id:    aws.String(aclID),
	})
	if err != nil {
		t.Fatalf("GetWebACL: %v", err)
	}
	if *getResp.WebACL.Name != "test-web-acl" {
		t.Errorf("expected web ACL name test-web-acl, got %s", *getResp.WebACL.Name)
	}

	// List web ACLs.
	listResp, err := client.ListWebACLs(ctx, &wafv2.ListWebACLsInput{
		Scope: wafv2types.ScopeRegional,
	})
	if err != nil {
		t.Fatalf("ListWebACLs: %v", err)
	}
	if len(listResp.WebACLs) != 1 {
		t.Errorf("expected 1 web ACL, got %d", len(listResp.WebACLs))
	}

	// Delete web ACL.
	_, err = client.DeleteWebACL(ctx, &wafv2.DeleteWebACLInput{
		Name:      aws.String("test-web-acl"),
		Scope:     wafv2types.ScopeRegional,
		Id:        aws.String(aclID),
		LockToken: aws.String(lockToken),
	})
	if err != nil {
		t.Fatalf("DeleteWebACL: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListWebACLs(ctx, &wafv2.ListWebACLsInput{
		Scope: wafv2types.ScopeRegional,
	})
	if err != nil {
		t.Fatalf("ListWebACLs after delete: %v", err)
	}
	if len(listResp.WebACLs) != 0 {
		t.Errorf("expected 0 web ACLs after delete, got %d", len(listResp.WebACLs))
	}
}

// ─── Redshift ───────────────────────────────────────────────────────────────

func TestRedshiftClusterOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := redshift.NewFromConfig(cfg)

	// Create cluster.
	createResp, err := client.CreateCluster(ctx, &redshift.CreateClusterInput{
		ClusterIdentifier:  aws.String("test-cluster"),
		NodeType:           aws.String("dc2.large"),
		MasterUsername:     aws.String("admin"),
		MasterUserPassword: aws.String("Password1!"),
		NumberOfNodes:      aws.Int32(2),
		DBName:             aws.String("testdb"),
	})
	if err != nil {
		t.Fatalf("CreateCluster: %v", err)
	}
	if createResp.Cluster == nil || createResp.Cluster.ClusterIdentifier == nil {
		t.Fatal("expected cluster with identifier")
	}
	if *createResp.Cluster.ClusterIdentifier != "test-cluster" {
		t.Errorf("expected cluster ID test-cluster, got %s", *createResp.Cluster.ClusterIdentifier)
	}

	// Describe clusters.
	descResp, err := client.DescribeClusters(ctx, &redshift.DescribeClustersInput{})
	if err != nil {
		t.Fatalf("DescribeClusters: %v", err)
	}
	if len(descResp.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(descResp.Clusters))
	}

	// Modify cluster.
	_, err = client.ModifyCluster(ctx, &redshift.ModifyClusterInput{
		ClusterIdentifier: aws.String("test-cluster"),
		NumberOfNodes:     aws.Int32(4),
	})
	if err != nil {
		t.Fatalf("ModifyCluster: %v", err)
	}

	// Delete cluster.
	_, err = client.DeleteCluster(ctx, &redshift.DeleteClusterInput{
		ClusterIdentifier:        aws.String("test-cluster"),
		SkipFinalClusterSnapshot: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("DeleteCluster: %v", err)
	}

	// Verify empty.
	descResp, err = client.DescribeClusters(ctx, &redshift.DescribeClustersInput{})
	if err != nil {
		t.Fatalf("DescribeClusters after delete: %v", err)
	}
	if len(descResp.Clusters) != 0 {
		t.Errorf("expected 0 clusters after delete, got %d", len(descResp.Clusters))
	}
}

// ─── EMR ────────────────────────────────────────────────────────────────────

func TestEMRClusterOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := emr.NewFromConfig(cfg)

	// Run job flow.
	runResp, err := client.RunJobFlow(ctx, &emr.RunJobFlowInput{
		Name:         aws.String("test-cluster"),
		ReleaseLabel: aws.String("emr-6.9.0"),
		Instances: &emrtypes.JobFlowInstancesConfig{
			MasterInstanceType: aws.String("m5.xlarge"),
			SlaveInstanceType:  aws.String("m5.xlarge"),
			InstanceCount:      aws.Int32(3),
		},
		Applications: []emrtypes.Application{
			{Name: aws.String("Spark")},
		},
	})
	if err != nil {
		t.Fatalf("RunJobFlow: %v", err)
	}
	if runResp.JobFlowId == nil || *runResp.JobFlowId == "" {
		t.Fatal("expected job flow ID")
	}
	clusterID := *runResp.JobFlowId

	// List clusters.
	listResp, err := client.ListClusters(ctx, &emr.ListClustersInput{})
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(listResp.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(listResp.Clusters))
	}

	// Describe cluster.
	descResp, err := client.DescribeCluster(ctx, &emr.DescribeClusterInput{
		ClusterId: aws.String(clusterID),
	})
	if err != nil {
		t.Fatalf("DescribeCluster: %v", err)
	}
	if descResp.Cluster == nil || descResp.Cluster.Name == nil {
		t.Fatal("expected cluster with name")
	}
	if *descResp.Cluster.Name != "test-cluster" {
		t.Errorf("expected cluster name test-cluster, got %s", *descResp.Cluster.Name)
	}

	// Terminate job flows.
	_, err = client.TerminateJobFlows(ctx, &emr.TerminateJobFlowsInput{
		JobFlowIds: []string{clusterID},
	})
	if err != nil {
		t.Fatalf("TerminateJobFlows: %v", err)
	}
}

// ─── Backup ─────────────────────────────────────────────────────────────────

func TestBackupVaultOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := backup.NewFromConfig(cfg)

	// Create backup vault.
	_, err = client.CreateBackupVault(ctx, &backup.CreateBackupVaultInput{
		BackupVaultName: aws.String("test-vault"),
	})
	if err != nil {
		t.Fatalf("CreateBackupVault: %v", err)
	}

	// List backup vaults.
	listResp, err := client.ListBackupVaults(ctx, &backup.ListBackupVaultsInput{})
	if err != nil {
		t.Fatalf("ListBackupVaults: %v", err)
	}
	if len(listResp.BackupVaultList) != 1 {
		t.Fatalf("expected 1 backup vault, got %d", len(listResp.BackupVaultList))
	}

	// Describe backup vault.
	descResp, err := client.DescribeBackupVault(ctx, &backup.DescribeBackupVaultInput{
		BackupVaultName: aws.String("test-vault"),
	})
	if err != nil {
		t.Fatalf("DescribeBackupVault: %v", err)
	}
	if *descResp.BackupVaultName != "test-vault" {
		t.Errorf("expected vault name test-vault, got %s", *descResp.BackupVaultName)
	}

	// Delete backup vault.
	_, err = client.DeleteBackupVault(ctx, &backup.DeleteBackupVaultInput{
		BackupVaultName: aws.String("test-vault"),
	})
	if err != nil {
		t.Fatalf("DeleteBackupVault: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListBackupVaults(ctx, &backup.ListBackupVaultsInput{})
	if err != nil {
		t.Fatalf("ListBackupVaults after delete: %v", err)
	}
	if len(listResp.BackupVaultList) != 0 {
		t.Errorf("expected 0 backup vaults after delete, got %d", len(listResp.BackupVaultList))
	}
}

// ─── EventBridge Scheduler ──────────────────────────────────────────────────

func TestSchedulerOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := scheduler.NewFromConfig(cfg)

	// Create schedule.
	createResp, err := client.CreateSchedule(ctx, &scheduler.CreateScheduleInput{
		Name:               aws.String("test-schedule"),
		ScheduleExpression: aws.String("rate(1 hour)"),
		Target: &schedulertypes.Target{
			Arn:     aws.String("arn:aws:lambda:us-east-1:123456789012:function:my-func"),
			RoleArn: aws.String("arn:aws:iam::123456789012:role/scheduler-role"),
		},
		FlexibleTimeWindow: &schedulertypes.FlexibleTimeWindow{
			Mode: schedulertypes.FlexibleTimeWindowModeOff,
		},
	})
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	if createResp.ScheduleArn == nil || *createResp.ScheduleArn == "" {
		t.Error("expected non-empty schedule ARN")
	}

	// Get schedule.
	getResp, err := client.GetSchedule(ctx, &scheduler.GetScheduleInput{
		Name: aws.String("test-schedule"),
	})
	if err != nil {
		t.Fatalf("GetSchedule: %v", err)
	}
	if *getResp.Name != "test-schedule" {
		t.Errorf("expected schedule name test-schedule, got %s", *getResp.Name)
	}

	// List schedules.
	listResp, err := client.ListSchedules(ctx, &scheduler.ListSchedulesInput{})
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(listResp.Schedules) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(listResp.Schedules))
	}

	// Delete schedule.
	_, err = client.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{
		Name: aws.String("test-schedule"),
	})
	if err != nil {
		t.Fatalf("DeleteSchedule: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListSchedules(ctx, &scheduler.ListSchedulesInput{})
	if err != nil {
		t.Fatalf("ListSchedules after delete: %v", err)
	}
	if len(listResp.Schedules) != 0 {
		t.Errorf("expected 0 schedules after delete, got %d", len(listResp.Schedules))
	}
}

// ─── X-Ray ──────────────────────────────────────────────────────────────────

func TestXRayGroupOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := xray.NewFromConfig(cfg)

	// Create group.
	createResp, err := client.CreateGroup(ctx, &xray.CreateGroupInput{
		GroupName:        aws.String("test-group"),
		FilterExpression: aws.String("service(\"my-service\")"),
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if createResp.Group == nil || createResp.Group.GroupName == nil {
		t.Fatal("expected group with name")
	}
	if *createResp.Group.GroupName != "test-group" {
		t.Errorf("expected group name test-group, got %s", *createResp.Group.GroupName)
	}

	// Get group.
	getResp, err := client.GetGroup(ctx, &xray.GetGroupInput{
		GroupName: aws.String("test-group"),
	})
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if *getResp.Group.GroupName != "test-group" {
		t.Errorf("expected group name test-group, got %s", *getResp.Group.GroupName)
	}

	// Get groups.
	groupsResp, err := client.GetGroups(ctx, &xray.GetGroupsInput{})
	if err != nil {
		t.Fatalf("GetGroups: %v", err)
	}
	if len(groupsResp.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groupsResp.Groups))
	}

	// Delete group.
	_, err = client.DeleteGroup(ctx, &xray.DeleteGroupInput{
		GroupName: aws.String("test-group"),
	})
	if err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}

	// Verify empty.
	groupsResp, err = client.GetGroups(ctx, &xray.GetGroupsInput{})
	if err != nil {
		t.Fatalf("GetGroups after delete: %v", err)
	}
	if len(groupsResp.Groups) != 0 {
		t.Errorf("expected 0 groups after delete, got %d", len(groupsResp.Groups))
	}
}

// ─── OpenSearch ─────────────────────────────────────────────────────────────

func TestOpenSearchDomainOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := opensearch.NewFromConfig(cfg)

	// Create domain.
	createResp, err := client.CreateDomain(ctx, &opensearch.CreateDomainInput{
		DomainName:    aws.String("test-domain"),
		EngineVersion: aws.String("OpenSearch_2.5"),
	})
	if err != nil {
		t.Fatalf("CreateDomain: %v", err)
	}
	if createResp.DomainStatus == nil || createResp.DomainStatus.DomainName == nil {
		t.Fatal("expected domain status with name")
	}
	if *createResp.DomainStatus.DomainName != "test-domain" {
		t.Errorf("expected domain name test-domain, got %s", *createResp.DomainStatus.DomainName)
	}

	// Describe domain.
	descResp, err := client.DescribeDomain(ctx, &opensearch.DescribeDomainInput{
		DomainName: aws.String("test-domain"),
	})
	if err != nil {
		t.Fatalf("DescribeDomain: %v", err)
	}
	if *descResp.DomainStatus.DomainName != "test-domain" {
		t.Errorf("expected domain name test-domain, got %s", *descResp.DomainStatus.DomainName)
	}

	// List domain names.
	listResp, err := client.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
	if err != nil {
		t.Fatalf("ListDomainNames: %v", err)
	}
	if len(listResp.DomainNames) != 1 {
		t.Errorf("expected 1 domain, got %d", len(listResp.DomainNames))
	}

	// Delete domain.
	_, err = client.DeleteDomain(ctx, &opensearch.DeleteDomainInput{
		DomainName: aws.String("test-domain"),
	})
	if err != nil {
		t.Fatalf("DeleteDomain: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
	if err != nil {
		t.Fatalf("ListDomainNames after delete: %v", err)
	}
	if len(listResp.DomainNames) != 0 {
		t.Errorf("expected 0 domains after delete, got %d", len(listResp.DomainNames))
	}
}

// ─── Service Discovery ─────────────────────────────────────────────────────

func TestServiceDiscoveryOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := servicediscovery.NewFromConfig(cfg)

	// Create namespace.
	nsResp, err := client.CreatePrivateDnsNamespace(ctx, &servicediscovery.CreatePrivateDnsNamespaceInput{
		Name: aws.String("test.local"),
		Vpc:  aws.String("vpc-12345"),
	})
	if err != nil {
		t.Fatalf("CreatePrivateDnsNamespace: %v", err)
	}
	if nsResp.OperationId == nil || *nsResp.OperationId == "" {
		t.Fatal("expected operation ID")
	}

	// Create service.
	svcResp, err := client.CreateService(ctx, &servicediscovery.CreateServiceInput{
		Name:        aws.String("test-service"),
		NamespaceId: aws.String("ns-12345"),
		DnsConfig: &sdtypes.DnsConfig{
			DnsRecords: []sdtypes.DnsRecord{
				{
					Type: sdtypes.RecordTypeA,
					TTL:  aws.Int64(60),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	if svcResp.Service == nil || svcResp.Service.Id == nil {
		t.Fatal("expected service with ID")
	}
	serviceID := *svcResp.Service.Id

	// List services.
	listResp, err := client.ListServices(ctx, &servicediscovery.ListServicesInput{})
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if len(listResp.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(listResp.Services))
	}

	// Get service.
	getResp, err := client.GetService(ctx, &servicediscovery.GetServiceInput{
		Id: aws.String(serviceID),
	})
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	if *getResp.Service.Name != "test-service" {
		t.Errorf("expected service name test-service, got %s", *getResp.Service.Name)
	}

	// Delete service.
	_, err = client.DeleteService(ctx, &servicediscovery.DeleteServiceInput{
		Id: aws.String(serviceID),
	})
	if err != nil {
		t.Fatalf("DeleteService: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListServices(ctx, &servicediscovery.ListServicesInput{})
	if err != nil {
		t.Fatalf("ListServices after delete: %v", err)
	}
	if len(listResp.Services) != 0 {
		t.Errorf("expected 0 services after delete, got %d", len(listResp.Services))
	}
}

// ─── Transfer Family ────────────────────────────────────────────────────────

func TestTransferServerOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := transfer.NewFromConfig(cfg)

	// Create server.
	createResp, err := client.CreateServer(ctx, &transfer.CreateServerInput{
		EndpointType:         transfertypes.EndpointTypePublic,
		IdentityProviderType: transfertypes.IdentityProviderTypeServiceManaged,
		Protocols:            []transfertypes.Protocol{transfertypes.ProtocolSftp},
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	if createResp.ServerId == nil || *createResp.ServerId == "" {
		t.Fatal("expected server ID")
	}
	serverID := *createResp.ServerId

	// List servers.
	listResp, err := client.ListServers(ctx, &transfer.ListServersInput{})
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if len(listResp.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(listResp.Servers))
	}

	// Describe server.
	descResp, err := client.DescribeServer(ctx, &transfer.DescribeServerInput{
		ServerId: aws.String(serverID),
	})
	if err != nil {
		t.Fatalf("DescribeServer: %v", err)
	}
	if descResp.Server == nil || descResp.Server.ServerId == nil {
		t.Fatal("expected server in describe response")
	}
	if *descResp.Server.ServerId != serverID {
		t.Errorf("expected server ID %s, got %s", serverID, *descResp.Server.ServerId)
	}

	// Delete server.
	_, err = client.DeleteServer(ctx, &transfer.DeleteServerInput{
		ServerId: aws.String(serverID),
	})
	if err != nil {
		t.Fatalf("DeleteServer: %v", err)
	}

	// Verify empty.
	listResp, err = client.ListServers(ctx, &transfer.ListServersInput{})
	if err != nil {
		t.Fatalf("ListServers after delete: %v", err)
	}
	if len(listResp.Servers) != 0 {
		t.Errorf("expected 0 servers after delete, got %d", len(listResp.Servers))
	}
}

// TestApplicationAutoScalingOperations verifies the Application Auto Scaling mock.
func TestApplicationAutoScalingOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := applicationautoscaling.NewFromConfig(cfg)

	// Register scalable target.
	_, err = client.RegisterScalableTarget(ctx, &applicationautoscaling.RegisterScalableTargetInput{
		ServiceNamespace:  applicationautoscalingtypes.ServiceNamespaceEcs,
		ResourceId:        aws.String("service/default/my-service"),
		ScalableDimension: applicationautoscalingtypes.ScalableDimensionECSServiceDesiredCount,
		MinCapacity:       aws.Int32(1),
		MaxCapacity:       aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("RegisterScalableTarget: %v", err)
	}

	// Describe scalable targets.
	descResp, err := client.DescribeScalableTargets(ctx, &applicationautoscaling.DescribeScalableTargetsInput{
		ServiceNamespace: applicationautoscalingtypes.ServiceNamespaceEcs,
	})
	if err != nil {
		t.Fatalf("DescribeScalableTargets: %v", err)
	}
	if len(descResp.ScalableTargets) != 1 {
		t.Fatalf("expected 1 scalable target, got %d", len(descResp.ScalableTargets))
	}

	// Deregister scalable target.
	_, err = client.DeregisterScalableTarget(ctx, &applicationautoscaling.DeregisterScalableTargetInput{
		ServiceNamespace:  applicationautoscalingtypes.ServiceNamespaceEcs,
		ResourceId:        aws.String("service/default/my-service"),
		ScalableDimension: applicationautoscalingtypes.ScalableDimensionECSServiceDesiredCount,
	})
	if err != nil {
		t.Fatalf("DeregisterScalableTarget: %v", err)
	}

	// Verify deregistered.
	descResp, err = client.DescribeScalableTargets(ctx, &applicationautoscaling.DescribeScalableTargetsInput{
		ServiceNamespace: applicationautoscalingtypes.ServiceNamespaceEcs,
	})
	if err != nil {
		t.Fatalf("DescribeScalableTargets after deregister: %v", err)
	}
	if len(descResp.ScalableTargets) != 0 {
		t.Errorf("expected 0 scalable targets after deregister, got %d", len(descResp.ScalableTargets))
	}
}

// TestResourceGroupsTaggingAPIOperations verifies the Resource Groups Tagging API mock.
func TestResourceGroupsTaggingAPIOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := resourcegroupstaggingapi.NewFromConfig(cfg)

	// Tag resources.
	_, err = client.TagResources(ctx, &resourcegroupstaggingapi.TagResourcesInput{
		ResourceARNList: []string{
			"arn:aws:s3:::my-bucket",
			"arn:aws:ec2:us-east-1:123456789012:instance/i-12345",
		},
		Tags: map[string]string{
			"Environment": "production",
			"Team":        "platform",
		},
	})
	if err != nil {
		t.Fatalf("TagResources: %v", err)
	}

	// Get resources.
	getResp, err := client.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{})
	if err != nil {
		t.Fatalf("GetResources: %v", err)
	}
	if len(getResp.ResourceTagMappingList) != 2 {
		t.Fatalf("expected 2 tagged resources, got %d", len(getResp.ResourceTagMappingList))
	}

	// Get tag keys.
	keysResp, err := client.GetTagKeys(ctx, &resourcegroupstaggingapi.GetTagKeysInput{})
	if err != nil {
		t.Fatalf("GetTagKeys: %v", err)
	}
	if len(keysResp.TagKeys) != 2 {
		t.Errorf("expected 2 tag keys, got %d", len(keysResp.TagKeys))
	}

	// Get tag values.
	valsResp, err := client.GetTagValues(ctx, &resourcegroupstaggingapi.GetTagValuesInput{
		Key: aws.String("Environment"),
	})
	if err != nil {
		t.Fatalf("GetTagValues: %v", err)
	}
	if len(valsResp.TagValues) != 1 || valsResp.TagValues[0] != "production" {
		t.Errorf("expected tag value 'production', got %v", valsResp.TagValues)
	}

	// Untag resources.
	_, err = client.UntagResources(ctx, &resourcegroupstaggingapi.UntagResourcesInput{
		ResourceARNList: []string{"arn:aws:s3:::my-bucket"},
		TagKeys:         []string{"Environment"},
	})
	if err != nil {
		t.Fatalf("UntagResources: %v", err)
	}
}

// TestSSOAdminOperations verifies the SSO Admin mock.
func TestSSOAdminOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := ssoadmin.NewFromConfig(cfg)
	instanceArn := "arn:aws:sso:::instance/ssoins-1234567890abcdef"

	// Create permission set.
	createResp, err := client.CreatePermissionSet(ctx, &ssoadmin.CreatePermissionSetInput{
		InstanceArn:     aws.String(instanceArn),
		Name:            aws.String("AdminAccess"),
		Description:     aws.String("Full admin access"),
		SessionDuration: aws.String("PT8H"),
	})
	if err != nil {
		t.Fatalf("CreatePermissionSet: %v", err)
	}
	if createResp.PermissionSet == nil || createResp.PermissionSet.PermissionSetArn == nil {
		t.Fatal("expected permission set with ARN")
	}
	permSetArn := *createResp.PermissionSet.PermissionSetArn

	// List permission sets.
	listResp, err := client.ListPermissionSets(ctx, &ssoadmin.ListPermissionSetsInput{
		InstanceArn: aws.String(instanceArn),
	})
	if err != nil {
		t.Fatalf("ListPermissionSets: %v", err)
	}
	if len(listResp.PermissionSets) != 1 {
		t.Fatalf("expected 1 permission set, got %d", len(listResp.PermissionSets))
	}

	// Describe permission set.
	descResp, err := client.DescribePermissionSet(ctx, &ssoadmin.DescribePermissionSetInput{
		InstanceArn:      aws.String(instanceArn),
		PermissionSetArn: aws.String(permSetArn),
	})
	if err != nil {
		t.Fatalf("DescribePermissionSet: %v", err)
	}
	if descResp.PermissionSet == nil || *descResp.PermissionSet.Name != "AdminAccess" {
		t.Errorf("expected name AdminAccess, got %v", descResp.PermissionSet)
	}

	// Create account assignment.
	_, err = client.CreateAccountAssignment(ctx, &ssoadmin.CreateAccountAssignmentInput{
		InstanceArn:      aws.String(instanceArn),
		PermissionSetArn: aws.String(permSetArn),
		PrincipalId:      aws.String("user-123"),
		PrincipalType:    ssoadmintypes.PrincipalTypeUser,
		TargetId:         aws.String("123456789012"),
		TargetType:       ssoadmintypes.TargetTypeAwsAccount,
	})
	if err != nil {
		t.Fatalf("CreateAccountAssignment: %v", err)
	}

	// Delete permission set.
	_, err = client.DeletePermissionSet(ctx, &ssoadmin.DeletePermissionSetInput{
		InstanceArn:      aws.String(instanceArn),
		PermissionSetArn: aws.String(permSetArn),
	})
	if err != nil {
		t.Fatalf("DeletePermissionSet: %v", err)
	}
}

// TestAppSyncOperations verifies the AppSync mock.
func TestAppSyncOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := appsync.NewFromConfig(cfg)

	// Create GraphQL API.
	createResp, err := client.CreateGraphqlApi(ctx, &appsync.CreateGraphqlApiInput{
		Name:               aws.String("my-api"),
		AuthenticationType: appsynctypes.AuthenticationTypeApiKey,
	})
	if err != nil {
		t.Fatalf("CreateGraphqlApi: %v", err)
	}
	if createResp.GraphqlApi == nil || createResp.GraphqlApi.ApiId == nil {
		t.Fatal("expected graphql api with ID")
	}
	apiId := *createResp.GraphqlApi.ApiId

	// Get GraphQL API.
	getResp, err := client.GetGraphqlApi(ctx, &appsync.GetGraphqlApiInput{
		ApiId: aws.String(apiId),
	})
	if err != nil {
		t.Fatalf("GetGraphqlApi: %v", err)
	}
	if *getResp.GraphqlApi.Name != "my-api" {
		t.Errorf("expected name my-api, got %s", *getResp.GraphqlApi.Name)
	}

	// List GraphQL APIs.
	listResp, err := client.ListGraphqlApis(ctx, &appsync.ListGraphqlApisInput{})
	if err != nil {
		t.Fatalf("ListGraphqlApis: %v", err)
	}
	if len(listResp.GraphqlApis) != 1 {
		t.Fatalf("expected 1 API, got %d", len(listResp.GraphqlApis))
	}

	// Delete GraphQL API.
	_, err = client.DeleteGraphqlApi(ctx, &appsync.DeleteGraphqlApiInput{
		ApiId: aws.String(apiId),
	})
	if err != nil {
		t.Fatalf("DeleteGraphqlApi: %v", err)
	}

	// Verify deleted.
	listResp, err = client.ListGraphqlApis(ctx, &appsync.ListGraphqlApisInput{})
	if err != nil {
		t.Fatalf("ListGraphqlApis after delete: %v", err)
	}
	if len(listResp.GraphqlApis) != 0 {
		t.Errorf("expected 0 APIs after delete, got %d", len(listResp.GraphqlApis))
	}
}

// TestMSKClusterOperations verifies the MSK/Kafka mock.
func TestMSKClusterOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := kafka.NewFromConfig(cfg)

	// Create cluster.
	createResp, err := client.CreateCluster(ctx, &kafka.CreateClusterInput{
		ClusterName:         aws.String("my-kafka-cluster"),
		KafkaVersion:        aws.String("3.5.1"),
		NumberOfBrokerNodes: aws.Int32(3),
		BrokerNodeGroupInfo: &kafkatypes.BrokerNodeGroupInfo{
			InstanceType:  aws.String("kafka.m5.large"),
			ClientSubnets: []string{"subnet-1", "subnet-2", "subnet-3"},
		},
	})
	if err != nil {
		t.Fatalf("CreateCluster: %v", err)
	}
	if createResp.ClusterArn == nil {
		t.Fatal("expected cluster ARN")
	}
	clusterArn := *createResp.ClusterArn

	// List clusters.
	listResp, err := client.ListClusters(ctx, &kafka.ListClustersInput{})
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(listResp.ClusterInfoList) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(listResp.ClusterInfoList))
	}

	// Delete cluster.
	_, err = client.DeleteCluster(ctx, &kafka.DeleteClusterInput{
		ClusterArn: aws.String(clusterArn),
	})
	if err != nil {
		t.Fatalf("DeleteCluster: %v", err)
	}

	// Verify deleted.
	listResp, err = client.ListClusters(ctx, &kafka.ListClustersInput{})
	if err != nil {
		t.Fatalf("ListClusters after delete: %v", err)
	}
	if len(listResp.ClusterInfoList) != 0 {
		t.Errorf("expected 0 clusters after delete, got %d", len(listResp.ClusterInfoList))
	}
}

// TestNeptuneClusterOperations verifies the Neptune mock.
func TestNeptuneClusterOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := neptune.NewFromConfig(cfg)

	// Create DB cluster.
	_, err = client.CreateDBCluster(ctx, &neptune.CreateDBClusterInput{
		DBClusterIdentifier: aws.String("my-neptune-cluster"),
		Engine:              aws.String("neptune"),
	})
	if err != nil {
		t.Fatalf("CreateDBCluster: %v", err)
	}

	// Describe DB clusters.
	descResp, err := client.DescribeDBClusters(ctx, &neptune.DescribeDBClustersInput{})
	if err != nil {
		t.Fatalf("DescribeDBClusters: %v", err)
	}
	if len(descResp.DBClusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(descResp.DBClusters))
	}
	if *descResp.DBClusters[0].DBClusterIdentifier != "my-neptune-cluster" {
		t.Errorf("expected cluster ID my-neptune-cluster, got %s", *descResp.DBClusters[0].DBClusterIdentifier)
	}

	// Delete DB cluster.
	_, err = client.DeleteDBCluster(ctx, &neptune.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String("my-neptune-cluster"),
	})
	if err != nil {
		t.Fatalf("DeleteDBCluster: %v", err)
	}

	// Verify deleted.
	descResp, err = client.DescribeDBClusters(ctx, &neptune.DescribeDBClustersInput{})
	if err != nil {
		t.Fatalf("DescribeDBClusters after delete: %v", err)
	}
	if len(descResp.DBClusters) != 0 {
		t.Errorf("expected 0 clusters after delete, got %d", len(descResp.DBClusters))
	}
}

// TestGuardDutyDetectorOperations verifies the GuardDuty mock.
func TestGuardDutyDetectorOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := guardduty.NewFromConfig(cfg)

	// Create detector.
	createResp, err := client.CreateDetector(ctx, &guardduty.CreateDetectorInput{
		Enable: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("CreateDetector: %v", err)
	}
	if createResp.DetectorId == nil || *createResp.DetectorId == "" {
		t.Fatal("expected detector ID")
	}
	detectorId := *createResp.DetectorId

	// Get detector.
	getResp, err := client.GetDetector(ctx, &guardduty.GetDetectorInput{
		DetectorId: aws.String(detectorId),
	})
	if err != nil {
		t.Fatalf("GetDetector: %v", err)
	}
	if getResp.Status != "ENABLED" {
		t.Errorf("expected status ENABLED, got %s", getResp.Status)
	}

	// List detectors.
	listResp, err := client.ListDetectors(ctx, &guardduty.ListDetectorsInput{})
	if err != nil {
		t.Fatalf("ListDetectors: %v", err)
	}
	if len(listResp.DetectorIds) != 1 {
		t.Fatalf("expected 1 detector, got %d", len(listResp.DetectorIds))
	}

	// Delete detector.
	_, err = client.DeleteDetector(ctx, &guardduty.DeleteDetectorInput{
		DetectorId: aws.String(detectorId),
	})
	if err != nil {
		t.Fatalf("DeleteDetector: %v", err)
	}

	// Verify deleted.
	listResp, err = client.ListDetectors(ctx, &guardduty.ListDetectorsInput{})
	if err != nil {
		t.Fatalf("ListDetectors after delete: %v", err)
	}
	if len(listResp.DetectorIds) != 0 {
		t.Errorf("expected 0 detectors after delete, got %d", len(listResp.DetectorIds))
	}
}

// TestMQBrokerOperations verifies the Amazon MQ mock.
func TestMQBrokerOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := mq.NewFromConfig(cfg)

	// Create broker.
	createResp, err := client.CreateBroker(ctx, &mq.CreateBrokerInput{
		BrokerName:         aws.String("my-broker"),
		EngineType:         mqtypes.EngineTypeActivemq,
		EngineVersion:      aws.String("5.17.6"),
		HostInstanceType:   aws.String("mq.m5.large"),
		DeploymentMode:     mqtypes.DeploymentModeSingleInstance,
		PubliclyAccessible: aws.Bool(false),
	})
	if err != nil {
		t.Fatalf("CreateBroker: %v", err)
	}
	if createResp.BrokerId == nil || *createResp.BrokerId == "" {
		t.Fatal("expected broker ID")
	}
	brokerId := *createResp.BrokerId

	// Describe broker.
	descResp, err := client.DescribeBroker(ctx, &mq.DescribeBrokerInput{
		BrokerId: aws.String(brokerId),
	})
	if err != nil {
		t.Fatalf("DescribeBroker: %v", err)
	}
	if *descResp.BrokerName != "my-broker" {
		t.Errorf("expected name my-broker, got %s", *descResp.BrokerName)
	}

	// List brokers.
	listResp, err := client.ListBrokers(ctx, &mq.ListBrokersInput{})
	if err != nil {
		t.Fatalf("ListBrokers: %v", err)
	}
	if len(listResp.BrokerSummaries) != 1 {
		t.Fatalf("expected 1 broker, got %d", len(listResp.BrokerSummaries))
	}

	// Delete broker.
	_, err = client.DeleteBroker(ctx, &mq.DeleteBrokerInput{
		BrokerId: aws.String(brokerId),
	})
	if err != nil {
		t.Fatalf("DeleteBroker: %v", err)
	}

	// Verify deleted.
	listResp, err = client.ListBrokers(ctx, &mq.ListBrokersInput{})
	if err != nil {
		t.Fatalf("ListBrokers after delete: %v", err)
	}
	if len(listResp.BrokerSummaries) != 0 {
		t.Errorf("expected 0 brokers after delete, got %d", len(listResp.BrokerSummaries))
	}
}

// TestDAXClusterOperations verifies the DAX mock.
func TestDAXClusterOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := dax.NewFromConfig(cfg)

	// Create cluster.
	createResp, err := client.CreateCluster(ctx, &dax.CreateClusterInput{
		ClusterName:       aws.String("my-dax-cluster"),
		NodeType:          aws.String("dax.r5.large"),
		ReplicationFactor: 3,
		IamRoleArn:        aws.String("arn:aws:iam::123456789012:role/dax-role"),
	})
	if err != nil {
		t.Fatalf("CreateCluster: %v", err)
	}
	if createResp.Cluster == nil || createResp.Cluster.ClusterName == nil {
		t.Fatal("expected cluster with name")
	}

	// Describe clusters.
	descResp, err := client.DescribeClusters(ctx, &dax.DescribeClustersInput{})
	if err != nil {
		t.Fatalf("DescribeClusters: %v", err)
	}
	if len(descResp.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(descResp.Clusters))
	}
	if *descResp.Clusters[0].ClusterName != "my-dax-cluster" {
		t.Errorf("expected cluster name my-dax-cluster, got %s", *descResp.Clusters[0].ClusterName)
	}

	// Delete cluster.
	_, err = client.DeleteCluster(ctx, &dax.DeleteClusterInput{
		ClusterName: aws.String("my-dax-cluster"),
	})
	if err != nil {
		t.Fatalf("DeleteCluster: %v", err)
	}

	// Verify deleted.
	descResp, err = client.DescribeClusters(ctx, &dax.DescribeClustersInput{})
	if err != nil {
		t.Fatalf("DescribeClusters after delete: %v", err)
	}
	if len(descResp.Clusters) != 0 {
		t.Errorf("expected 0 clusters after delete, got %d", len(descResp.Clusters))
	}
}

// TestFSxFileSystemOperations verifies the FSx mock.
func TestFSxFileSystemOperations(t *testing.T) {
	mock := awsmock.Start(t)
	ctx := context.Background()

	cfg, err := mock.AWSConfig(ctx)
	if err != nil {
		t.Fatalf("AWSConfig: %v", err)
	}

	client := fsx.NewFromConfig(cfg)

	// Create file system.
	createResp, err := client.CreateFileSystem(ctx, &fsx.CreateFileSystemInput{
		FileSystemType:  fsxtypes.FileSystemTypeLustre,
		StorageCapacity: aws.Int32(1200),
		SubnetIds:       []string{"subnet-12345"},
		Tags: []fsxtypes.Tag{
			{Key: aws.String("Name"), Value: aws.String("my-fsx")},
		},
	})
	if err != nil {
		t.Fatalf("CreateFileSystem: %v", err)
	}
	if createResp.FileSystem == nil || createResp.FileSystem.FileSystemId == nil {
		t.Fatal("expected file system with ID")
	}
	fsId := *createResp.FileSystem.FileSystemId

	// Describe file systems.
	descResp, err := client.DescribeFileSystems(ctx, &fsx.DescribeFileSystemsInput{})
	if err != nil {
		t.Fatalf("DescribeFileSystems: %v", err)
	}
	if len(descResp.FileSystems) != 1 {
		t.Fatalf("expected 1 file system, got %d", len(descResp.FileSystems))
	}

	// Delete file system.
	_, err = client.DeleteFileSystem(ctx, &fsx.DeleteFileSystemInput{
		FileSystemId: aws.String(fsId),
	})
	if err != nil {
		t.Fatalf("DeleteFileSystem: %v", err)
	}

	// Verify deleted.
	descResp, err = client.DescribeFileSystems(ctx, &fsx.DescribeFileSystemsInput{})
	if err != nil {
		t.Fatalf("DescribeFileSystems after delete: %v", err)
	}
	if len(descResp.FileSystems) != 0 {
		t.Errorf("expected 0 file systems after delete, got %d", len(descResp.FileSystems))
	}
}
