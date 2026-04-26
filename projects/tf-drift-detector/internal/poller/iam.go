package poller

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// IAMRoleComparator fetches live IAM role attributes.
type IAMRoleComparator struct{}

func (IAMRoleComparator) ResourceType() string { return "aws_iam_role" }

func (IAMRoleComparator) FetchLive(ctx context.Context, cfg aws.Config, id string) (map[string]interface{}, error) {
	out, err := iam.NewFromConfig(cfg).GetRole(ctx, &iam.GetRoleInput{RoleName: aws.String(id)})
	if err != nil {
		return nil, err
	}
	if out.Role == nil {
		return nil, fmt.Errorf("role %s not found", id)
	}
	r := out.Role
	return map[string]interface{}{
		"name":                  aws.ToString(r.RoleName),
		"path":                  aws.ToString(r.Path),
		"assume_role_policy":    aws.ToString(r.AssumeRolePolicyDocument),
		"max_session_duration":  r.MaxSessionDuration,
		"description":           aws.ToString(r.Description),
	}, nil
}
