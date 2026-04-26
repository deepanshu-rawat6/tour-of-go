package scanner

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	appconfig "github.com/tour-of-go/aws-resource-reaper/internal/config"
)

type mockAuth struct{}

func (mockAuth) ForAccount(_ context.Context, _ appconfig.AccountConfig) (aws.Config, error) {
	return aws.Config{}, nil
}

type fixedScanner struct{ resources []Resource }

func (f fixedScanner) Scan(_ context.Context, _ aws.Config, region, accountID string) ([]Resource, error) {
	r := make([]Resource, len(f.resources))
	copy(r, f.resources)
	for i := range r {
		r[i].Region = region
		r[i].AccountID = accountID
	}
	return r, nil
}

type errorScanner struct{}

func (errorScanner) Scan(_ context.Context, _ aws.Config, _, _ string) ([]Resource, error) {
	return nil, errors.New("scan failed")
}

func TestEngine_CollectsAllResults(t *testing.T) {
	cfg := &appconfig.Config{
		Accounts: []appconfig.AccountConfig{
			{ID: "111", RoleARN: "arn:aws:iam::111:role/R"},
			{ID: "222", RoleARN: "arn:aws:iam::222:role/R"},
			{ID: "333", RoleARN: "arn:aws:iam::333:role/R"},
		},
		Regions:     []string{"us-east-1", "eu-west-1", "ap-southeast-1"},
		Concurrency: 5,
	}
	engine := &Engine{
		Auth:     mockAuth{},
		Scanners: []Scanner{fixedScanner{resources: []Resource{{ID: "r1", Type: "ebs-volume"}}}},
	}
	results, err := engine.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	// 3 accounts × 3 regions × 1 resource each = 9
	if len(results) != 9 {
		t.Errorf("expected 9 results, got %d", len(results))
	}
}

func TestEngine_ErrorCancelsWork(t *testing.T) {
	cfg := &appconfig.Config{
		Accounts:    []appconfig.AccountConfig{{ID: "111", RoleARN: "arn:aws:iam::111:role/R"}},
		Regions:     []string{"us-east-1"},
		Concurrency: 5,
	}
	engine := &Engine{
		Auth:     mockAuth{},
		Scanners: []Scanner{errorScanner{}},
	}
	_, err := engine.Scan(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error from errorScanner")
	}
}
