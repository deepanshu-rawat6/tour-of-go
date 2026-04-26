package scanner

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// ElasticIPScanner finds unassociated Elastic IPs.
type ElasticIPScanner struct{}

func (ElasticIPScanner) Scan(ctx context.Context, cfg aws.Config, region, accountID string) ([]Resource, error) {
	cfg.Region = region
	client := ec2.NewFromConfig(cfg)
	return scanElasticIPs(ctx, client, region, accountID)
}

func scanElasticIPs(ctx context.Context, client ec2API, region, accountID string) ([]Resource, error) {
	// DescribeAddresses does not paginate — returns all at once.
	out, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, err
	}
	var resources []Resource
	for _, addr := range out.Addresses {
		if addr.AssociationId != nil {
			continue // in use
		}
		resources = append(resources, Resource{
			ID:        aws.ToString(addr.AllocationId),
			Type:      "elastic-ip",
			Region:    region,
			AccountID: accountID,
			Tags:      tagsFromEC2(addr.Tags),
			Metadata:  map[string]string{"public_ip": aws.ToString(addr.PublicIp)},
		})
	}
	return resources, nil
}
