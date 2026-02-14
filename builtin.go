package awsmock

import (
	"github.com/riyanimam/goto/services/s3"
	"github.com/riyanimam/goto/services/sqs"
	"github.com/riyanimam/goto/services/sts"
)

// builtinServices returns the default set of service mocks.
func builtinServices() []Service {
	return []Service{
		sts.New(),
		s3.New(),
		sqs.New(),
	}
}
