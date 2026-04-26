package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/tour-of-go/aws-resource-reaper/internal/rules"
	"github.com/tour-of-go/aws-resource-reaper/internal/scanner"
)

func finding(typ string, action rules.Action) rules.Finding {
	return rules.Finding{
		Resource: scanner.Resource{ID: "test-id", Type: typ, Region: "us-east-1", AccountID: "123"},
		Action:   action,
		Reason:   "test reason",
	}
}

func TestExecute_AbortOnNo(t *testing.T) {
	ex := &Executor{In: strings.NewReader("n\n")}
	findings := []rules.Finding{finding("ebs-volume", rules.ActionDelete)}
	// Should not panic or call any AWS API — just abort.
	if err := ex.Execute(context.Background(), findings); err != nil {
		t.Fatal(err)
	}
}

func TestExecute_NoDestructive(t *testing.T) {
	ex := &Executor{In: strings.NewReader("")}
	findings := []rules.Finding{finding("ec2-instance", rules.ActionRecommend)}
	// No destructive actions → no prompt, no error.
	if err := ex.Execute(context.Background(), findings); err != nil {
		t.Fatal(err)
	}
}

func TestCountDestructive(t *testing.T) {
	findings := []rules.Finding{
		finding("ebs-volume", rules.ActionDelete),
		finding("ec2-instance", rules.ActionRecommend),
		finding("alb", rules.ActionDelete),
		finding("elastic-ip", rules.ActionSkip),
	}
	if n := countDestructive(findings); n != 2 {
		t.Errorf("expected 2 destructive, got %d", n)
	}
}
