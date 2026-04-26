package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/tour-of-go/tf-drift-detector/internal/config"
)

// ManagedResource is a resource extracted from Terraform state.
type ManagedResource struct {
	Type       string                 // e.g. "aws_instance"
	Name       string                 // TF resource name
	ID         string                 // primary resource ID
	Attributes map[string]interface{} // declared attributes from state
}

// SupportedTypes is the set of resource types this tool can compare.
var SupportedTypes = map[string]bool{
	"aws_instance":        true,
	"aws_s3_bucket":       true,
	"aws_security_group":  true,
	"aws_db_instance":     true,
	"aws_iam_role":        true,
	"aws_lambda_function": true,
}

// tfstate is the minimal structure we need from a terraform.tfstate file.
type tfstate struct {
	Resources []tfResource `json:"resources"`
}

type tfResource struct {
	Type      string       `json:"type"`
	Name      string       `json:"name"`
	Instances []tfInstance `json:"instances"`
}

type tfInstance struct {
	Attributes map[string]interface{} `json:"attributes"`
}

// Load fetches and parses Terraform state from the configured backend.
func Load(ctx context.Context, cfg appconfig.BackendConfig) ([]ManagedResource, error) {
	var data []byte
	var err error

	switch cfg.Type {
	case "s3":
		data, err = loadFromS3(ctx, cfg)
	case "local":
		data, err = os.ReadFile(cfg.FilePath)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", cfg.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}
	return parse(data)
}

func loadFromS3(ctx context.Context, cfg appconfig.BackendConfig) ([]byte, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))
	if err != nil {
		return nil, err
	}
	out, err := s3.NewFromConfig(awsCfg).GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(cfg.Key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func parse(data []byte) ([]ManagedResource, error) {
	var state tfstate
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing tfstate: %w", err)
	}
	var resources []ManagedResource
	for _, r := range state.Resources {
		if !SupportedTypes[r.Type] {
			continue
		}
		for _, inst := range r.Instances {
			id := extractID(inst.Attributes)
			if id == "" {
				continue
			}
			resources = append(resources, ManagedResource{
				Type:       r.Type,
				Name:       r.Name,
				ID:         id,
				Attributes: inst.Attributes,
			})
		}
	}
	return resources, nil
}

// extractID pulls the primary ID from the attributes map.
func extractID(attrs map[string]interface{}) string {
	for _, key := range []string{"id", "arn", "name"} {
		if v, ok := attrs[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
