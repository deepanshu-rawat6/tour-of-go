package poller

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// LambdaComparator fetches live Lambda function attributes.
type LambdaComparator struct{}

func (LambdaComparator) ResourceType() string { return "aws_lambda_function" }

func (LambdaComparator) FetchLive(ctx context.Context, cfg aws.Config, id string) (map[string]interface{}, error) {
	out, err := lambda.NewFromConfig(cfg).GetFunction(ctx, &lambda.GetFunctionInput{FunctionName: aws.String(id)})
	if err != nil {
		return nil, err
	}
	if out.Configuration == nil {
		return nil, fmt.Errorf("lambda %s not found", id)
	}
	c := out.Configuration
	attrs := map[string]interface{}{
		"function_name": aws.ToString(c.FunctionName),
		"runtime":       string(c.Runtime),
		"handler":       aws.ToString(c.Handler),
		"memory_size":   c.MemorySize,
		"timeout":       c.Timeout,
		"role":          aws.ToString(c.Role),
	}
	if c.Environment != nil {
		attrs["environment"] = c.Environment.Variables
	}
	return attrs, nil
}
