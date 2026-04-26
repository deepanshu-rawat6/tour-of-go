package poller

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// SGComparator fetches live Security Group attributes.
type SGComparator struct{}

func (SGComparator) ResourceType() string { return "aws_security_group" }

func (SGComparator) FetchLive(ctx context.Context, cfg aws.Config, id string) (map[string]interface{}, error) {
	out, err := ec2.NewFromConfig(cfg).DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{id},
	})
	if err != nil {
		return nil, err
	}
	if len(out.SecurityGroups) == 0 {
		return nil, fmt.Errorf("security group %s not found", id)
	}
	sg := out.SecurityGroups[0]
	return map[string]interface{}{
		"name":        aws.ToString(sg.GroupName),
		"description": aws.ToString(sg.Description),
		"vpc_id":      aws.ToString(sg.VpcId),
		"tags":        flattenEC2Tags(sg.Tags),
	}, nil
}
