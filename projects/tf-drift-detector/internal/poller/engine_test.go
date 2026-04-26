package poller

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/tour-of-go/tf-drift-detector/internal/state"
)

type fixedComparator struct {
	rtype string
	attrs map[string]interface{}
	err   error
}

func (f fixedComparator) ResourceType() string { return f.rtype }
func (f fixedComparator) FetchLive(_ context.Context, _ aws.Config, _ string) (map[string]interface{}, error) {
	return f.attrs, f.err
}

func TestEngine_NoDrift(t *testing.T) {
	engine := &Engine{
		Comparators: map[string]Comparator{
			"aws_instance": fixedComparator{rtype: "aws_instance", attrs: map[string]interface{}{"instance_type": "t3.micro"}},
		},
		Concurrency: 5,
	}
	resources := []state.ManagedResource{
		{Type: "aws_instance", ID: "i-1", Attributes: map[string]interface{}{"instance_type": "t3.micro"}},
		{Type: "aws_instance", ID: "i-2", Attributes: map[string]interface{}{"instance_type": "t3.micro"}},
	}
	results, err := engine.Poll(context.Background(), aws.Config{}, resources)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Drifted {
			t.Errorf("expected no drift for %s", r.Resource.ID)
		}
	}
}

func TestEngine_DetectsDrift(t *testing.T) {
	engine := &Engine{
		Comparators: map[string]Comparator{
			"aws_instance": fixedComparator{rtype: "aws_instance", attrs: map[string]interface{}{"instance_type": "t3.large"}},
		},
		Concurrency: 5,
	}
	resources := []state.ManagedResource{
		{Type: "aws_instance", ID: "i-1", Attributes: map[string]interface{}{"instance_type": "t3.micro"}},
	}
	results, err := engine.Poll(context.Background(), aws.Config{}, resources)
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Drifted {
		t.Error("expected drift")
	}
}

func TestEngine_ResourceNotFound(t *testing.T) {
	engine := &Engine{
		Comparators: map[string]Comparator{
			"aws_instance": fixedComparator{rtype: "aws_instance", err: errors.New("not found")},
		},
		Concurrency: 5,
	}
	resources := []state.ManagedResource{
		{Type: "aws_instance", ID: "i-deleted", Attributes: map[string]interface{}{}},
	}
	results, err := engine.Poll(context.Background(), aws.Config{}, resources)
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Drifted {
		t.Error("expected drift for deleted resource")
	}
}
