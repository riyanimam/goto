package awsmock_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sts"

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
