package awsmock

import (
	"github.com/riyanimam/goto/services/acm"
	"github.com/riyanimam/goto/services/cloudformation"
	"github.com/riyanimam/goto/services/cloudwatch"
	"github.com/riyanimam/goto/services/cloudwatchlogs"
	"github.com/riyanimam/goto/services/dynamodb"
	"github.com/riyanimam/goto/services/ec2"
	"github.com/riyanimam/goto/services/ecr"
	"github.com/riyanimam/goto/services/ecs"
	"github.com/riyanimam/goto/services/elbv2"
	"github.com/riyanimam/goto/services/eventbridge"
	"github.com/riyanimam/goto/services/iam"
	"github.com/riyanimam/goto/services/kinesis"
	"github.com/riyanimam/goto/services/kms"
	"github.com/riyanimam/goto/services/lambda"
	"github.com/riyanimam/goto/services/rds"
	"github.com/riyanimam/goto/services/route53"
	"github.com/riyanimam/goto/services/s3"
	"github.com/riyanimam/goto/services/secretsmanager"
	"github.com/riyanimam/goto/services/ses"
	"github.com/riyanimam/goto/services/sns"
	"github.com/riyanimam/goto/services/sqs"
	"github.com/riyanimam/goto/services/ssm"
	"github.com/riyanimam/goto/services/stepfunctions"
	"github.com/riyanimam/goto/services/sts"
)

// builtinServices returns the default set of service mocks.
func builtinServices() []Service {
	return []Service{
		sts.New(),
		s3.New(),
		sqs.New(),
		dynamodb.New(),
		sns.New(),
		secretsmanager.New(),
		lambda.New(),
		cloudwatchlogs.New(),
		iam.New(),
		ec2.New(),
		kinesis.New(),
		eventbridge.New(),
		ssm.New(),
		kms.New(),
		cloudformation.New(),
		ecr.New(),
		route53.New(),
		ecs.New(),
		elbv2.New(),
		rds.New(),
		cloudwatch.New(),
		stepfunctions.New(),
		acm.New(),
		ses.New(),
	}
}
