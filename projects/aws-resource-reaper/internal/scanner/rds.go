package scanner

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// rdsAPI is the subset of the RDS client used by RDSSnapshotScanner.
type rdsAPI interface {
	DescribeDBSnapshots(ctx context.Context, in *rds.DescribeDBSnapshotsInput, opts ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error)
}

const rdsSnapshotAgeDays = 30

// RDSSnapshotScanner finds manual RDS snapshots older than 30 days.
type RDSSnapshotScanner struct{}

func (RDSSnapshotScanner) Scan(ctx context.Context, cfg aws.Config, region, accountID string) ([]Resource, error) {
	cfg.Region = region
	client := rds.NewFromConfig(cfg)
	return scanRDSSnapshots(ctx, client, region, accountID)
}

func scanRDSSnapshots(ctx context.Context, client rdsAPI, region, accountID string) ([]Resource, error) {
	cutoff := time.Now().AddDate(0, 0, -rdsSnapshotAgeDays)
	var resources []Resource
	var marker *string
	for {
		out, err := client.DescribeDBSnapshots(ctx, &rds.DescribeDBSnapshotsInput{
			SnapshotType: aws.String("manual"),
			Marker:       marker,
		})
		if err != nil {
			return nil, err
		}
		for _, snap := range out.DBSnapshots {
			if snap.SnapshotCreateTime == nil || snap.SnapshotCreateTime.After(cutoff) {
				continue
			}
			resources = append(resources, Resource{
				ID:        aws.ToString(snap.DBSnapshotIdentifier),
				Type:      "rds-snapshot",
				Region:    region,
				AccountID: accountID,
				Metadata: map[string]string{
					"db_instance":   aws.ToString(snap.DBInstanceIdentifier),
					"created":       snap.SnapshotCreateTime.Format(time.RFC3339),
					"size_gb":       formatInt(int(aws.ToInt32(snap.AllocatedStorage))),
					"engine":        aws.ToString(snap.Engine),
				},
			})
		}
		if out.Marker == nil {
			break
		}
		marker = out.Marker
	}
	return resources, nil
}
