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
- **52 services** — broad coverage of the most commonly used AWS services

## Supported Services

| Service | Operations |
|---------|-----------|
| **S3** | CreateBucket, DeleteBucket, ListBuckets, HeadBucket, PutObject, GetObject, HeadObject, DeleteObject, ListObjectsV2, CopyObject |
| **SQS** | CreateQueue, DeleteQueue, ListQueues, GetQueueUrl, GetQueueAttributes, SetQueueAttributes, SendMessage, ReceiveMessage, DeleteMessage, PurgeQueue |
| **STS** | GetCallerIdentity, AssumeRole, GetSessionToken |
| **DynamoDB** | CreateTable, DeleteTable, DescribeTable, ListTables, PutItem, GetItem, DeleteItem, Query, Scan |
| **SNS** | CreateTopic, DeleteTopic, ListTopics, Subscribe, Unsubscribe, ListSubscriptions, Publish |
| **Secrets Manager** | CreateSecret, GetSecretValue, PutSecretValue, DeleteSecret, ListSecrets, DescribeSecret, UpdateSecret |
| **Lambda** | CreateFunction, GetFunction, DeleteFunction, ListFunctions, Invoke, UpdateFunctionCode, UpdateFunctionConfiguration |
| **CloudWatch Logs** | CreateLogGroup, DeleteLogGroup, DescribeLogGroups, CreateLogStream, DeleteLogStream, DescribeLogStreams, PutLogEvents, GetLogEvents, FilterLogEvents |
| **IAM** | CreateUser, GetUser, DeleteUser, ListUsers, CreateRole, GetRole, DeleteRole, ListRoles, CreatePolicy, GetPolicy, DeletePolicy, ListPolicies, AttachRolePolicy, DetachRolePolicy |
| **EC2** | RunInstances, DescribeInstances, TerminateInstances, CreateVpc, DescribeVpcs, DeleteVpc, CreateSecurityGroup, DescribeSecurityGroups, DeleteSecurityGroup, CreateSubnet, DescribeSubnets, DeleteSubnet |
| **Kinesis** | CreateStream, DeleteStream, DescribeStream, ListStreams, PutRecord, GetRecords, GetShardIterator |
| **EventBridge** | CreateEventBus, DeleteEventBus, ListEventBuses, PutRule, DeleteRule, ListRules, PutTargets, RemoveTargets, ListTargetsByRule, PutEvents |
| **SSM Parameter Store** | PutParameter, GetParameter, GetParameters, DeleteParameter, DescribeParameters, GetParametersByPath |
| **KMS** | CreateKey, DescribeKey, ListKeys, Encrypt, Decrypt, GenerateDataKey, CreateAlias, ListAliases, DeleteAlias, ScheduleKeyDeletion |
| **CloudFormation** | CreateStack, DeleteStack, DescribeStacks, ListStacks, UpdateStack |
| **ECR** | CreateRepository, DeleteRepository, DescribeRepositories, ListImages, PutImage, BatchGetImage, GetAuthorizationToken |
| **Route 53** | CreateHostedZone, GetHostedZone, DeleteHostedZone, ListHostedZones, ChangeResourceRecordSets, ListResourceRecordSets |
| **ECS** | CreateCluster, DeleteCluster, DescribeClusters, ListClusters, RegisterTaskDefinition, DeregisterTaskDefinition, ListTaskDefinitions, RunTask, StopTask, ListTasks, DescribeTasks, CreateService, DeleteService, UpdateService, ListServices, DescribeServices |
| **ELBv2** | CreateLoadBalancer, DeleteLoadBalancer, DescribeLoadBalancers, CreateTargetGroup, DeleteTargetGroup, DescribeTargetGroups, RegisterTargets, DeregisterTargets, DescribeTargetHealth, CreateListener, DeleteListener, DescribeListeners |
| **RDS** | CreateDBInstance, DeleteDBInstance, DescribeDBInstances, ModifyDBInstance, CreateDBCluster, DeleteDBCluster, DescribeDBClusters |
| **CloudWatch** | PutMetricData, GetMetricData, ListMetrics, PutMetricAlarm, DescribeAlarms, DeleteAlarms |
| **Step Functions** | CreateStateMachine, DeleteStateMachine, DescribeStateMachine, ListStateMachines, StartExecution, DescribeExecution, ListExecutions, StopExecution |
| **ACM** | RequestCertificate, DescribeCertificate, ListCertificates, DeleteCertificate |
| **SES v2** | CreateEmailIdentity, GetEmailIdentity, ListEmailIdentities, SendEmail, DeleteEmailIdentity |
| **Cognito Identity Provider** | CreateUserPool, DescribeUserPool, DeleteUserPool, ListUserPools, CreateUserPoolClient, AdminCreateUser, AdminGetUser, AdminDeleteUser, ListUsers |
| **API Gateway V2** | CreateApi, GetApi, DeleteApi, GetApis, CreateStage, GetStages, DeleteStage, CreateRoute, GetRoutes, DeleteRoute |
| **CloudFront** | CreateDistribution, GetDistribution, DeleteDistribution, ListDistributions, UpdateDistribution |
| **EKS** | CreateCluster, DescribeCluster, DeleteCluster, ListClusters, CreateNodegroup, DescribeNodegroup, DeleteNodegroup, ListNodegroups |
| **ElastiCache** | CreateCacheCluster, DeleteCacheCluster, DescribeCacheClusters, ModifyCacheCluster, CreateReplicationGroup, DeleteReplicationGroup, DescribeReplicationGroups |
| **Firehose** | CreateDeliveryStream, DeleteDeliveryStream, DescribeDeliveryStream, ListDeliveryStreams, PutRecord |
| **Athena** | StartQueryExecution, GetQueryExecution, GetQueryResults, ListQueryExecutions, CreateWorkGroup, GetWorkGroup, DeleteWorkGroup, ListWorkGroups |
| **Glue** | CreateDatabase, GetDatabase, DeleteDatabase, GetDatabases, CreateTable, GetTable, DeleteTable, GetTables, CreateCrawler, GetCrawler, DeleteCrawler, StartCrawler, ListCrawlers |
| **Auto Scaling** | CreateAutoScalingGroup, DescribeAutoScalingGroups, DeleteAutoScalingGroup, UpdateAutoScalingGroup, CreateLaunchConfiguration, DescribeLaunchConfigurations, DeleteLaunchConfiguration, SetDesiredCapacity |
| **API Gateway** | CreateRestApi, GetRestApi, DeleteRestApi, GetRestApis, CreateResource, GetResources, PutMethod, PutIntegration |
| **Cognito Identity** | CreateIdentityPool, DescribeIdentityPool, DeleteIdentityPool, ListIdentityPools, UpdateIdentityPool |
| **Organizations** | CreateOrganization, DescribeOrganization, ListAccounts, CreateAccount, DescribeAccount, CreateOrganizationalUnit, ListOrganizationalUnitsForParent |
| **DynamoDB Streams** | ListStreams, DescribeStream, GetShardIterator, GetRecords |
| **EFS** | CreateFileSystem, DescribeFileSystems, DeleteFileSystem, CreateMountTarget, DescribeMountTargets, DeleteMountTarget |
| **Batch** | CreateComputeEnvironment, DescribeComputeEnvironments, DeleteComputeEnvironment, CreateJobQueue, DescribeJobQueues, DeleteJobQueue, SubmitJob, DescribeJobs |
| **CodeBuild** | CreateProject, BatchGetProjects, ListProjects, DeleteProject, StartBuild, BatchGetBuilds |
| **CodePipeline** | CreatePipeline, GetPipeline, DeletePipeline, ListPipelines, UpdatePipeline |
| **CloudTrail** | CreateTrail, GetTrail, DeleteTrail, DescribeTrails, StartLogging, StopLogging, GetTrailStatus, LookupEvents |
| **Config** | PutConfigRule, DescribeConfigRules, DeleteConfigRule, PutConfigurationRecorder, DescribeConfigurationRecorders, PutDeliveryChannel |
| **WAF v2** | CreateWebACL, GetWebACL, DeleteWebACL, ListWebACLs, UpdateWebACL, CreateIPSet, GetIPSet, DeleteIPSet, ListIPSets |
| **Redshift** | CreateCluster, DescribeClusters, DeleteCluster, ModifyCluster |
| **EMR** | RunJobFlow, DescribeCluster, ListClusters, TerminateJobFlows, AddJobFlowSteps, ListSteps |
| **Backup** | CreateBackupVault, DeleteBackupVault, ListBackupVaults, DescribeBackupVault, CreateBackupPlan, GetBackupPlan, DeleteBackupPlan |
| **EventBridge Scheduler** | CreateSchedule, GetSchedule, DeleteSchedule, ListSchedules, UpdateSchedule |
| **X-Ray** | PutTraceSegments, GetTraceSummaries, BatchGetTraces, CreateGroup, GetGroup, DeleteGroup, GetGroups |
| **OpenSearch** | CreateDomain, DescribeDomain, DeleteDomain, ListDomainNames, UpdateDomainConfig |
| **Service Discovery** | CreatePrivateDnsNamespace, CreateService, GetService, DeleteService, ListServices, RegisterInstance, DeregisterInstance, ListInstances |
| **Transfer Family** | CreateServer, DescribeServer, DeleteServer, ListServers, CreateUser, DescribeUser, DeleteUser |

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

## How It Works

1. **`awsmock.Start(t)`** starts an `httptest.Server` that accepts all AWS SDK requests.
2. **`mock.AWSConfig(ctx)`** returns an `aws.Config` with:
   - `BaseEndpoint` pointing to the mock server
   - Static test credentials (`AKIAIOSFODNN7EXAMPLE`)
   - Region set to `us-east-1`
3. You pass this config to any AWS SDK v2 client (`s3.NewFromConfig(cfg)`, etc.).
4. The mock server routes each request to the correct service handler based on the
   `Authorization` header credential scope or the `X-Amz-Target` header.
5. Each service stores state in memory (maps, slices) with mutex-based thread safety.
6. When your test finishes, `t.Cleanup` automatically shuts down the server.

## Usage Examples

### DynamoDB

```go
func TestDynamoDB(t *testing.T) {
    mock := awsmock.Start(t)
    cfg, _ := mock.AWSConfig(context.Background())
    client := dynamodb.NewFromConfig(cfg)
    ctx := context.Background()

    // Create table.
    client.CreateTable(ctx, &dynamodb.CreateTableInput{
        TableName: aws.String("users"),
        KeySchema: []types.KeySchemaElement{
            {AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
        },
        AttributeDefinitions: []types.AttributeDefinition{
            {AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
        },
        BillingMode: types.BillingModePayPerRequest,
    })

    // Put and get items.
    client.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String("users"),
        Item: map[string]types.AttributeValue{
            "id":   &types.AttributeValueMemberS{Value: "1"},
            "name": &types.AttributeValueMemberS{Value: "Alice"},
        },
    })
}
```

### Lambda

```go
func TestLambda(t *testing.T) {
    mock := awsmock.Start(t)
    cfg, _ := mock.AWSConfig(context.Background())
    client := lambda.NewFromConfig(cfg)
    ctx := context.Background()

    // Create function.
    client.CreateFunction(ctx, &lambda.CreateFunctionInput{
        FunctionName: aws.String("my-handler"),
        Runtime:      types.RuntimePython312,
        Role:         aws.String("arn:aws:iam::123456789012:role/lambda-role"),
        Handler:      aws.String("index.handler"),
        Code:         &types.FunctionCode{ZipFile: []byte("fake")},
    })

    // Invoke returns the payload you send (echo behavior).
    resp, _ := client.Invoke(ctx, &lambda.InvokeInput{
        FunctionName: aws.String("my-handler"),
        Payload:      []byte(`{"key":"value"}`),
    })
    // resp.Payload == []byte(`{"key":"value"}`)
}
```

### IAM

```go
func TestIAM(t *testing.T) {
    mock := awsmock.Start(t)
    cfg, _ := mock.AWSConfig(context.Background())
    client := iam.NewFromConfig(cfg)
    ctx := context.Background()

    // Create role.
    client.CreateRole(ctx, &iam.CreateRoleInput{
        RoleName:                 aws.String("my-role"),
        AssumeRolePolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[]}`),
    })

    // Create and attach policy.
    policyResp, _ := client.CreatePolicy(ctx, &iam.CreatePolicyInput{
        PolicyName:     aws.String("my-policy"),
        PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[]}`),
    })
    client.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
        RoleName:  aws.String("my-role"),
        PolicyArn: policyResp.Policy.Arn,
    })
}
```

### EC2

```go
func TestEC2(t *testing.T) {
    mock := awsmock.Start(t)
    cfg, _ := mock.AWSConfig(context.Background())
    client := ec2.NewFromConfig(cfg)
    ctx := context.Background()

    // Create VPC + run instances.
    vpcResp, _ := client.CreateVpc(ctx, &ec2.CreateVpcInput{
        CidrBlock: aws.String("10.0.0.0/16"),
    })
    runResp, _ := client.RunInstances(ctx, &ec2.RunInstancesInput{
        ImageId:      aws.String("ami-12345678"),
        InstanceType: "t2.micro",
        MinCount:     aws.Int32(1),
        MaxCount:     aws.Int32(1),
    })

    // Terminate when done.
    client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
        InstanceIds: []string{*runResp.Instances[0].InstanceId},
    })
    client.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: vpcResp.Vpc.VpcId})
}
```

### SSM Parameter Store

```go
func TestSSM(t *testing.T) {
    mock := awsmock.Start(t)
    cfg, _ := mock.AWSConfig(context.Background())
    client := ssm.NewFromConfig(cfg)
    ctx := context.Background()

    // Store and retrieve configuration.
    client.PutParameter(ctx, &ssm.PutParameterInput{
        Name:  aws.String("/app/db/host"),
        Value: aws.String("localhost"),
        Type:  types.ParameterTypeString,
    })

    resp, _ := client.GetParameter(ctx, &ssm.GetParameterInput{
        Name: aws.String("/app/db/host"),
    })
    // *resp.Parameter.Value == "localhost"
}
```

### KMS

```go
func TestKMS(t *testing.T) {
    mock := awsmock.Start(t)
    cfg, _ := mock.AWSConfig(context.Background())
    client := kms.NewFromConfig(cfg)
    ctx := context.Background()

    // Create a key and encrypt/decrypt.
    keyResp, _ := client.CreateKey(ctx, &kms.CreateKeyInput{
        Description: aws.String("test key"),
    })

    encResp, _ := client.Encrypt(ctx, &kms.EncryptInput{
        KeyId:     keyResp.KeyMetadata.KeyId,
        Plaintext: []byte("secret"),
    })

    decResp, _ := client.Decrypt(ctx, &kms.DecryptInput{
        CiphertextBlob: encResp.CiphertextBlob,
    })
    // string(decResp.Plaintext) == "secret"
}
```

### SQS

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

### STS

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

## How to Use This Package in Your Project

### Step 1: Add the dependency

```bash
go get github.com/riyanimam/goto
```

### Step 2: Import and start in your test

```go
import awsmock "github.com/riyanimam/goto"

func TestMyFunction(t *testing.T) {
    mock := awsmock.Start(t)
    cfg, _ := mock.AWSConfig(context.Background())
    // Pass cfg to your code that creates AWS clients
}
```

### Step 3: Refactor your production code to accept aws.Config

The key design pattern is **dependency injection of the AWS config**. Instead of
creating AWS clients with hardcoded configs, pass `aws.Config` as a parameter:

```go
// ❌ Hard to test
func ProcessOrders() {
    cfg, _ := config.LoadDefaultConfig(context.Background())
    client := sqs.NewFromConfig(cfg)
    // ...
}

// ✅ Testable — inject the config
func ProcessOrders(cfg aws.Config) {
    client := sqs.NewFromConfig(cfg)
    // ...
}

// In tests:
func TestProcessOrders(t *testing.T) {
    mock := awsmock.Start(t)
    cfg, _ := mock.AWSConfig(context.Background())
    ProcessOrders(cfg) // Uses mock server
}

// In production:
func main() {
    cfg, _ := config.LoadDefaultConfig(context.Background())
    ProcessOrders(cfg) // Uses real AWS
}
```

### Step 4: Reset state between subtests if needed

```go
func TestMultipleScenarios(t *testing.T) {
    mock := awsmock.Start(t)

    t.Run("scenario1", func(t *testing.T) {
        // ... create resources ...
    })

    mock.Reset() // Clear all service state

    t.Run("scenario2", func(t *testing.T) {
        // Fresh state, no leftover resources
    })
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
│   or X-Amz-Target header        │
└──────────────┬───────────────────┘
               │
┌──────────────▼───────────────────┐
│       Service Handlers           │
│   S3, SQS, STS, DynamoDB, SNS,  │
│   Secrets Manager, Lambda, IAM,  │
│   CloudWatch Logs, EC2, Kinesis, │
│   EventBridge, SSM, KMS, ECR,    │
│   CloudFormation, Route 53, ECS, │
│   ELBv2, RDS, CloudWatch, ACM,   │
│   Step Functions, SES, Cognito,  │
│   API Gateway, CloudFront, EKS,  │
│   ElastiCache, Firehose, Athena, │
│   Glue, Auto Scaling, Batch,     │
│   CodeBuild, CodePipeline, EMR,  │
│   CloudTrail, Config, WAF v2,    │
│   Redshift, Backup, Scheduler,   │
│   X-Ray, OpenSearch, EFS,        │
│   Organizations, DynamoDB Streams,│
│   Service Discovery, Transfer    │
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
