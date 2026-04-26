package poller

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type ec2API interface {
	DescribeInstances(ctx context.Context, in *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

// EC2Comparator fetches live EC2 instance attributes.
type EC2Comparator struct{}

func (EC2Comparator) ResourceType() string { return "aws_instance" }

func (EC2Comparator) FetchLive(ctx context.Context, cfg aws.Config, id string) (map[string]interface{}, error) {
	return fetchEC2(ctx, ec2.NewFromConfig(cfg), id)
}

func fetchEC2(ctx context.Context, client ec2API, id string) (map[string]interface{}, error) {
	out, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	})
	if err != nil {
		return nil, err
	}
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			attrs := map[string]interface{}{
				"instance_type": string(inst.InstanceType),
				"ami":           aws.ToString(inst.ImageId),
				"subnet_id":     aws.ToString(inst.SubnetId),
				"vpc_id":        aws.ToString(inst.VpcId),
				"key_name":      aws.ToString(inst.KeyName),
				"tags":          flattenEC2Tags(inst.Tags),
			}
			var sgIDs []string
			for _, sg := range inst.SecurityGroups {
				sgIDs = append(sgIDs, aws.ToString(sg.GroupId))
			}
			attrs["vpc_security_group_ids"] = sgIDs
			return attrs, nil
		}
	}
	return nil, fmt.Errorf("instance %s not found", id)
}
