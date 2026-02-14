# goto - AWS Mock Services for Go

[![CI](https://github.com/riyanimam/goto/actions/workflows/ci.yml/badge.svg)](https://github.com/riyanimam/goto/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/riyanimam/goto.svg)](https://pkg.go.dev/github.com/riyanimam/goto)
[![Go Report Card](https://goreportcard.com/badge/github.com/riyanimam/goto)](https://goreportcard.com/report/github.com/riyanimam/goto)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

`goto` is a pure-Go library that mocks AWS services for testing. It starts an
in-memory HTTP server that emulates real AWS API endpoints, so you can test code
that uses the AWS SDK for Go v2 without making any real API calls.

> **Inspired by [moto](https://github.com/getmoto/moto)** — the popular Python
> AWS mock library — `goto` brings the same idea to Go with idiomatic APIs and
> zero external service dependencies.

## Features

- **Zero configuration** — `awsmock.Start(t)` is all you need
- **Automatic cleanup** — the mock server stops when the test ends
- **Thread-safe** — safe for parallel tests
- **Pure Go** — no Python, no Docker, no external processes
- **AWS SDK v2** — works with `github.com/aws/aws-sdk-go-v2`

## Supported Services

| Service | Operations |
|---------|-----------|
| **S3** | CreateBucket, DeleteBucket, ListBuckets, HeadBucket, PutObject, GetObject, HeadObject, DeleteObject, ListObjectsV2, CopyObject |
| **SQS** | CreateQueue, DeleteQueue, ListQueues, GetQueueUrl, GetQueueAttributes, SetQueueAttributes, SendMessage, ReceiveMessage, DeleteMessage, PurgeQueue |
| **STS** | GetCallerIdentity, AssumeRole, GetSessionToken |
| **DynamoDB** | CreateTable, DeleteTable, DescribeTable, ListTables, PutItem, GetItem, DeleteItem, Query, Scan |
| **SNS** | CreateTopic, DeleteTopic, ListTopics, Subscribe, Unsubscribe, ListSubscriptions, Publish |
| **Secrets Manager** | CreateSecret, GetSecretValue, PutSecretValue, DeleteSecret, ListSecrets, DescribeSecret, UpdateSecret |

## Installation

```bash
go get github.com/riyanimam/goto
```

Requires **Go 1.22** or later.

## Quick Start

```go
package myapp_test

import (
    "context"
    "io"
    "strings"
    "testing"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"

    awsmock "github.com/riyanimam/goto"
)

func TestUploadToS3(t *testing.T) {
    // Start a mock AWS server — automatically stops when the test ends.
    mock := awsmock.Start(t)

    // Get an AWS config that routes all requests to the mock server.
    cfg, err := mock.AWSConfig(context.Background())
    if err != nil {
        t.Fatal(err)
    }

    // Create an S3 client using path-style addressing.
    client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.UsePathStyle = true
    })

    ctx := context.Background()

    // Create a bucket.
    _, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
        Bucket: aws.String("my-bucket"),
    })
    if err != nil {
        t.Fatal(err)
    }

    // Upload an object.
    _, err = client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("hello.txt"),
        Body:   strings.NewReader("Hello, World!"),
    })
    if err != nil {
        t.Fatal(err)
    }

    // Read it back.
    resp, err := client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("hello.txt"),
    })
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    if string(body) != "Hello, World!" {
        t.Errorf("got %q, want %q", body, "Hello, World!")
    }
}
```

## Usage with SQS

```go
func TestSendAndReceive(t *testing.T) {
    mock := awsmock.Start(t)

    cfg, err := mock.AWSConfig(context.Background())
    if err != nil {
        t.Fatal(err)
    }

    client := sqs.NewFromConfig(cfg)
    ctx := context.Background()

    // Create a queue.
    createResp, _ := client.CreateQueue(ctx, &sqs.CreateQueueInput{
        QueueName: aws.String("my-queue"),
    })

    // Send a message.
    client.SendMessage(ctx, &sqs.SendMessageInput{
        QueueUrl:    createResp.QueueUrl,
        MessageBody: aws.String(`{"event": "order.created"}`),
    })

    // Receive the message.
    recvResp, _ := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
        QueueUrl: createResp.QueueUrl,
    })

    if len(recvResp.Messages) != 1 {
        t.Fatalf("expected 1 message, got %d", len(recvResp.Messages))
    }
}
```

## Usage with STS

```go
func TestCallerIdentity(t *testing.T) {
    mock := awsmock.Start(t)

    cfg, err := mock.AWSConfig(context.Background())
    if err != nil {
        t.Fatal(err)
    }

    client := sts.NewFromConfig(cfg)
    resp, err := client.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
    if err != nil {
        t.Fatal(err)
    }

    // Mock account ID is always "123456789012".
    if *resp.Account != "123456789012" {
        t.Errorf("unexpected account: %s", *resp.Account)
    }
}
```

## Adding Custom Services

You can implement the `Service` interface to add support for additional AWS
services:

```go
type Service interface {
    Name() string           // AWS service name (e.g., "dynamodb")
    Handler() http.Handler  // HTTP handler for the service
    Reset()                 // Clears all in-memory state
}
```

Register your service when starting the mock:

```go
mock := awsmock.Start(t, awsmock.WithService(myCustomService))
```

## Architecture

```
┌──────────────────────────────────┐
│         Your Test Code           │
│   client := s3.NewFromConfig()   │
└──────────────┬───────────────────┘
               │ HTTP request
┌──────────────▼───────────────────┐
│         MockServer               │
│   net/http/httptest.Server       │
│   Routes by Authorization header │
└──────────────┬───────────────────┘
               │
┌──────────────▼───────────────────┐
│       Service Handlers           │
│   S3, SQS, STS, DynamoDB,       │
│   SNS, Secrets Manager           │
│   (in-memory, thread-safe)       │
└──────────────────────────────────┘
```

## Development

```bash
# Run tests
make test

# Run tests with race detection
make test-race

# Run linter
make lint

# Run all checks
make ci
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.
