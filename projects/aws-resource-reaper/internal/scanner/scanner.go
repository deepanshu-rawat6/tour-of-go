package scanner

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Resource represents a discovered AWS resource.
type Resource struct {
	ID        string
	Type      string
	Region    string
	AccountID string
	Tags      map[string]string
	Metadata  map[string]string
}

// Scanner discovers resources of a specific type in one region.
type Scanner interface {
	Scan(ctx context.Context, cfg aws.Config, region, accountID string) ([]Resource, error)
}
