// Package awsmock provides an in-memory mock of AWS services for testing.
//
// awsmock starts a local HTTP server that emulates AWS service APIs,
// allowing you to test code that uses the AWS SDK for Go v2 without
// making real API calls.
//
// # Quick Start
//
//	func TestMyFunction(t *testing.T) {
//	    mock := awsmock.Start(t)
//
//	    cfg := mock.AWSConfig(context.Background())
//	    client := s3.NewFromConfig(cfg)
//
//	    _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
//	        Bucket: aws.String("my-bucket"),
//	    })
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	}
//
// # Supported Services
//
// awsmock currently supports the following AWS services:
//   - S3 (Simple Storage Service)
//   - SQS (Simple Queue Service)
//   - STS (Security Token Service)
//   - DynamoDB
//   - SNS (Simple Notification Service)
//   - Secrets Manager
//   - Lambda
//   - CloudWatch Logs
//   - IAM (Identity and Access Management)
//   - EC2 (Elastic Compute Cloud)
//   - Kinesis Data Streams
//   - EventBridge
//   - SSM Parameter Store
//   - KMS (Key Management Service)
//   - CloudFormation
//   - ECR (Elastic Container Registry)
//   - Route 53 (DNS)
//   - ECS (Elastic Container Service)
//   - ELBv2 (Elastic Load Balancing v2)
//   - RDS (Relational Database Service)
//   - CloudWatch (Metrics and Alarms)
//   - Step Functions
//   - ACM (Certificate Manager)
//   - SES v2 (Simple Email Service)
//   - Cognito Identity Provider
//   - API Gateway V2 (HTTP/WebSocket APIs)
//   - CloudFront (CDN)
//   - EKS (Elastic Kubernetes Service)
//   - ElastiCache (Redis/Memcached)
//   - Firehose (Kinesis Data Firehose)
//   - Athena (SQL Query Service)
//   - Glue (ETL/Data Catalog)
//   - Auto Scaling
//   - API Gateway (REST APIs)
//   - Cognito Identity (Federated Identities)
//   - Organizations
//   - DynamoDB Streams
//   - EFS (Elastic File System)
//   - Batch
//   - CodeBuild
//   - CodePipeline
//   - CloudTrail
//   - Config
//   - WAF v2 (Web Application Firewall)
//   - Redshift
//   - EMR (Elastic MapReduce)
//   - Backup
//   - EventBridge Scheduler
//   - X-Ray
//   - OpenSearch
//   - Service Discovery (Cloud Map)
//   - Transfer Family (SFTP/FTPS/FTP)
//
// Additional services can be added by implementing the [Service] interface.
package awsmock
