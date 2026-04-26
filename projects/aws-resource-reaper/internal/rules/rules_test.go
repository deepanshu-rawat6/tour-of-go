package rules

import (
	"testing"

	"github.com/tour-of-go/aws-resource-reaper/internal/scanner"
)

func resource(typ string, meta map[string]string) scanner.Resource {
	return scanner.Resource{ID: "test-id", Type: typ, Region: "us-east-1", AccountID: "123", Metadata: meta}
}

var ruleTests = []struct {
	name     string
	rule     Rule
	resource scanner.Resource
	wantAct  Action
	wantNil  bool
}{
	{
		name:     "EBS unattached → Delete",
		rule:     UnattachedEBSRule{},
		resource: resource("ebs-volume", map[string]string{"size_gb": "100", "volume_type": "gp2"}),
		wantAct:  ActionDelete,
	},
	{
		name:     "EBS rule ignores non-EBS",
		rule:     UnattachedEBSRule{},
		resource: resource("ec2-instance", nil),
		wantNil:  true,
	},
	{
		name:     "EIP unassociated → Delete",
		rule:     UnattachedEIPRule{},
		resource: resource("elastic-ip", map[string]string{"public_ip": "1.2.3.4"}),
		wantAct:  ActionDelete,
	},
	{
		name:     "EC2 x86 → Recommend",
		rule:     GravitonMigrationRule{},
		resource: resource("ec2-instance", map[string]string{"instance_type": "m5.large", "graviton_suggest": "m6g.large"}),
		wantAct:  ActionRecommend,
	},
	{
		name:     "EC2 Graviton → nil (no suggestion)",
		rule:     GravitonMigrationRule{},
		resource: resource("ec2-instance", map[string]string{"instance_type": "m6g.large", "graviton_suggest": ""}),
		wantNil:  true,
	},
	{
		name:     "RDS old snapshot → Delete",
		rule:     OldRDSSnapshotRule{},
		resource: resource("rds-snapshot", map[string]string{"size_gb": "200", "created": "2024-01-01T00:00:00Z"}),
		wantAct:  ActionDelete,
	},
	{
		name:     "ALB zero traffic → Delete",
		rule:     ZeroTrafficALBRule{},
		resource: resource("alb", map[string]string{"dns_name": "my-alb.elb.amazonaws.com"}),
		wantAct:  ActionDelete,
	},
	{
		name:     "SG unused → Delete",
		rule:     UnusedSGRule{},
		resource: resource("security-group", map[string]string{"name": "my-sg", "vpc_id": "vpc-1"}),
		wantAct:  ActionDelete,
	},
	{
		name:     "SG rule ignores non-SG",
		rule:     UnusedSGRule{},
		resource: resource("ebs-volume", nil),
		wantNil:  true,
	},
}

func TestRules(t *testing.T) {
	for _, tc := range ruleTests {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.rule.Evaluate(tc.resource)
			if tc.wantNil {
				if f != nil {
					t.Errorf("expected nil finding, got action=%s", f.Action)
				}
				return
			}
			if f == nil {
				t.Fatal("expected finding, got nil")
			}
			if f.Action != tc.wantAct {
				t.Errorf("expected action %s, got %s", tc.wantAct, f.Action)
			}
			if f.Reason == "" {
				t.Error("expected non-empty reason")
			}
		})
	}
}

func TestEvaluator_DropsSkip(t *testing.T) {
	ev := DefaultEvaluator()
	resources := []scanner.Resource{
		resource("ebs-volume", map[string]string{"size_gb": "50", "volume_type": "gp2"}),
		resource("elastic-ip", map[string]string{"public_ip": "1.2.3.4"}),
		resource("unknown-type", nil), // no rule matches → no finding
	}
	findings := ev.Evaluate(resources)
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}
}
