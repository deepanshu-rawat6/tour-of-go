package scanner

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// ec2API is the subset of the EC2 client used by EBS and ElasticIP scanners.
type ec2API interface {
	DescribeVolumes(ctx context.Context, in *ec2.DescribeVolumesInput, opts ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
	DescribeAddresses(ctx context.Context, in *ec2.DescribeAddressesInput, opts ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	DescribeInstances(ctx context.Context, in *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeNetworkInterfaces(ctx context.Context, in *ec2.DescribeNetworkInterfacesInput, opts ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
	DescribeSecurityGroups(ctx context.Context, in *ec2.DescribeSecurityGroupsInput, opts ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
}

// EBSScanner finds unattached (available) EBS volumes.
type EBSScanner struct{}

func (EBSScanner) Scan(ctx context.Context, cfg aws.Config, region, accountID string) ([]Resource, error) {
	cfg.Region = region
	client := ec2.NewFromConfig(cfg)
	return scanEBS(ctx, client, region, accountID)
}

func scanEBS(ctx context.Context, client ec2API, region, accountID string) ([]Resource, error) {
	var resources []Resource
	var nextToken *string
	for {
		out, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
			Filters: []ec2types.Filter{
				{Name: aws.String("status"), Values: []string{"available"}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, v := range out.Volumes {
			resources = append(resources, Resource{
				ID:        aws.ToString(v.VolumeId),
				Type:      "ebs-volume",
				Region:    region,
				AccountID: accountID,
				Tags:      tagsFromEC2(v.Tags),
				Metadata: map[string]string{
					"size_gb":     formatInt(int(aws.ToInt32(v.Size))),
					"volume_type": string(v.VolumeType),
				},
			})
		}
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return resources, nil
}
