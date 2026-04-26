package poller

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// RDSComparator fetches live RDS instance attributes.
type RDSComparator struct{}

func (RDSComparator) ResourceType() string { return "aws_db_instance" }

func (RDSComparator) FetchLive(ctx context.Context, cfg aws.Config, id string) (map[string]interface{}, error) {
	out, err := rds.NewFromConfig(cfg).DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(id),
	})
	if err != nil {
		return nil, err
	}
	if len(out.DBInstances) == 0 {
		return nil, fmt.Errorf("db instance %s not found", id)
	}
	db := out.DBInstances[0]
	return map[string]interface{}{
		"engine":             aws.ToString(db.Engine),
		"engine_version":     aws.ToString(db.EngineVersion),
		"instance_class":     aws.ToString(db.DBInstanceClass),
		"allocated_storage":  db.AllocatedStorage,
		"multi_az":           db.MultiAZ,
		"storage_encrypted":  db.StorageEncrypted,
		"publicly_accessible": db.PubliclyAccessible,
	}, nil
}
