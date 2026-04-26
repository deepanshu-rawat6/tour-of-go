package poller

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Comparator fetches the live state of a specific AWS resource type.
type Comparator interface {
	// ResourceType returns the Terraform resource type this comparator handles.
	ResourceType() string
	// FetchLive returns the live attributes of the resource with the given ID.
	FetchLive(ctx context.Context, cfg aws.Config, id string) (map[string]interface{}, error)
}
