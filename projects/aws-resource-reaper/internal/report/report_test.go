package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tour-of-go/aws-resource-reaper/internal/rules"
	"github.com/tour-of-go/aws-resource-reaper/internal/scanner"
)

var testFindings = []rules.Finding{
	{
		Resource:               scanner.Resource{ID: "vol-abc", Type: "ebs-volume", Region: "us-east-1", AccountID: "123456789012"},
		Action:                 rules.ActionDelete,
		Reason:                 "EBS volume is unattached",
		EstimatedMonthlySavings: 10.00,
	},
	{
		Resource:               scanner.Resource{ID: "i-xyz", Type: "ec2-instance", Region: "eu-west-1", AccountID: "987654321098"},
		Action:                 rules.ActionRecommend,
		Reason:                 "Migrate to Graviton",
		EstimatedMonthlySavings: 0,
	},
}

func TestPrintText(t *testing.T) {
	var buf bytes.Buffer
	r := &Reporter{Format: "text"}
	if err := r.Print(testFindings, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"vol-abc", "i-xyz", "Delete", "Recommend", "Total findings: 2", "$10.00/mo"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output:\n%s", want, out)
		}
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	r := &Reporter{Format: "json"}
	if err := r.Print(testFindings, &buf); err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if result["total_findings"].(float64) != 2 {
		t.Errorf("expected total_findings=2, got %v", result["total_findings"])
	}
	if result["estimated_monthly_savings"].(float64) != 10.00 {
		t.Errorf("expected savings=10.00, got %v", result["estimated_monthly_savings"])
	}
}
