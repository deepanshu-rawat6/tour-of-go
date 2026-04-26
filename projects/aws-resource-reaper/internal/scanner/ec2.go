package scanner

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// x86Families are instance families that have a Graviton equivalent.
var x86Families = map[string]string{
	"m5": "m6g", "m5a": "m6g", "m5n": "m6g",
	"c5": "c6g", "c5a": "c6g", "c5n": "c6g",
	"r5": "r6g", "r5a": "r6g", "r5n": "r6g",
	"t3": "t4g", "t3a": "t4g",
}

// EC2Scanner flags x86 instances that have a Graviton equivalent.
type EC2Scanner struct{}

func (EC2Scanner) Scan(ctx context.Context, cfg aws.Config, region, accountID string) ([]Resource, error) {
	cfg.Region = region
	client := ec2.NewFromConfig(cfg)
	return scanEC2(ctx, client, region, accountID)
}

func scanEC2(ctx context.Context, client ec2API, region, accountID string) ([]Resource, error) {
	var resources []Resource
	var nextToken *string
	for {
		out, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			Filters: []ec2types.Filter{
				{Name: aws.String("instance-state-name"), Values: []string{"running"}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range out.Reservations {
			for _, inst := range r.Instances {
				family := instanceFamily(string(inst.InstanceType))
				graviton, ok := x86Families[family]
				if !ok {
					continue
				}
				resources = append(resources, Resource{
					ID:        aws.ToString(inst.InstanceId),
					Type:      "ec2-instance",
					Region:    region,
					AccountID: accountID,
					Tags:      tagsFromEC2(inst.Tags),
					Metadata: map[string]string{
						"instance_type":    string(inst.InstanceType),
						"graviton_suggest": graviton + sizeOf(string(inst.InstanceType)),
					},
				})
			}
		}
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return resources, nil
}

// instanceFamily returns the family prefix (e.g. "m5" from "m5.large").
func instanceFamily(instanceType string) string {
	parts := strings.SplitN(instanceType, ".", 2)
	return parts[0]
}

// sizeOf returns the size suffix (e.g. ".large" from "m5.large").
func sizeOf(instanceType string) string {
	idx := strings.Index(instanceType, ".")
	if idx < 0 {
		return ""
	}
	return instanceType[idx:]
}
