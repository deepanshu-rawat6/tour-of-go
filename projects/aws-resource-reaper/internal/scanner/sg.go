package scanner

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// SecurityGroupScanner finds security groups not attached to any ENI.
type SecurityGroupScanner struct{}

func (SecurityGroupScanner) Scan(ctx context.Context, cfg aws.Config, region, accountID string) ([]Resource, error) {
	cfg.Region = region
	client := ec2.NewFromConfig(cfg)
	return scanSecurityGroups(ctx, client, region, accountID)
}

func scanSecurityGroups(ctx context.Context, client ec2API, region, accountID string) ([]Resource, error) {
	// Collect all SG IDs referenced by ENIs.
	inUse := make(map[string]struct{})
	var eniToken *string
	for {
		out, err := client.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
			NextToken: eniToken,
		})
		if err != nil {
			return nil, err
		}
		for _, eni := range out.NetworkInterfaces {
			for _, g := range eni.Groups {
				inUse[aws.ToString(g.GroupId)] = struct{}{}
			}
		}
		if out.NextToken == nil {
			break
		}
		eniToken = out.NextToken
	}

	// List all SGs and filter out those in use or the default group.
	var resources []Resource
	var sgToken *string
	for {
		out, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			NextToken: sgToken,
		})
		if err != nil {
			return nil, err
		}
		for _, sg := range out.SecurityGroups {
			id := aws.ToString(sg.GroupId)
			if aws.ToString(sg.GroupName) == "default" {
				continue
			}
			if _, used := inUse[id]; used {
				continue
			}
			resources = append(resources, Resource{
				ID:        id,
				Type:      "security-group",
				Region:    region,
				AccountID: accountID,
				Tags:      tagsFromEC2(sg.Tags),
				Metadata: map[string]string{
					"name":   aws.ToString(sg.GroupName),
					"vpc_id": aws.ToString(sg.VpcId),
				},
			})
		}
		if out.NextToken == nil {
			break
		}
		sgToken = out.NextToken
	}
	return resources, nil
}
