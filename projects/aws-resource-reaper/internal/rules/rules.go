package rules

import (
	"fmt"

	"github.com/tour-of-go/aws-resource-reaper/internal/scanner"
)

// Action describes what the reaper should do with a resource.
type Action string

const (
	ActionDelete    Action = "Delete"
	ActionTerminate Action = "Terminate"
	ActionRecommend Action = "Recommend"
	ActionSkip      Action = "Skip"
)

// Finding is the result of evaluating a rule against a resource.
type Finding struct {
	Resource               scanner.Resource
	Action                 Action
	Reason                 string
	EstimatedMonthlySavings float64
}

// Rule evaluates a single resource and returns a Finding (nil means skip).
type Rule interface {
	Evaluate(r scanner.Resource) *Finding
}

// Evaluator runs all rules against a slice of resources.
type Evaluator struct {
	Rules []Rule
}

// DefaultEvaluator returns an Evaluator with all built-in rules.
func DefaultEvaluator() *Evaluator {
	return &Evaluator{Rules: []Rule{
		UnattachedEBSRule{},
		UnattachedEIPRule{},
		GravitonMigrationRule{},
		OldRDSSnapshotRule{},
		ZeroTrafficALBRule{},
		UnusedSGRule{},
	}}
}

// Evaluate applies all rules to each resource and returns non-skip findings.
func (e *Evaluator) Evaluate(resources []scanner.Resource) []Finding {
	var findings []Finding
	for _, r := range resources {
		for _, rule := range e.Rules {
			if f := rule.Evaluate(r); f != nil && f.Action != ActionSkip {
				findings = append(findings, *f)
				break // first matching rule wins
			}
		}
	}
	return findings
}

// --- Rules ---

// UnattachedEBSRule flags unattached EBS volumes for deletion.
type UnattachedEBSRule struct{}

func (UnattachedEBSRule) Evaluate(r scanner.Resource) *Finding {
	if r.Type != "ebs-volume" {
		return nil
	}
	sizeGB := 0
	fmt.Sscanf(r.Metadata["size_gb"], "%d", &sizeGB)
	// gp2 ~$0.10/GB/month
	savings := float64(sizeGB) * 0.10
	return &Finding{
		Resource:               r,
		Action:                 ActionDelete,
		Reason:                 "EBS volume is unattached (state: available)",
		EstimatedMonthlySavings: savings,
	}
}

// UnattachedEIPRule flags unassociated Elastic IPs for release.
type UnattachedEIPRule struct{}

func (UnattachedEIPRule) Evaluate(r scanner.Resource) *Finding {
	if r.Type != "elastic-ip" {
		return nil
	}
	return &Finding{
		Resource:               r,
		Action:                 ActionDelete,
		Reason:                 "Elastic IP is not associated with any resource ($0.005/hr idle charge)",
		EstimatedMonthlySavings: 3.60, // ~$0.005/hr × 720hr
	}
}

// GravitonMigrationRule recommends migrating x86 instances to Graviton.
type GravitonMigrationRule struct{}

func (GravitonMigrationRule) Evaluate(r scanner.Resource) *Finding {
	if r.Type != "ec2-instance" {
		return nil
	}
	suggest := r.Metadata["graviton_suggest"]
	if suggest == "" {
		return nil
	}
	return &Finding{
		Resource: r,
		Action:   ActionRecommend,
		Reason:   fmt.Sprintf("Migrate from %s to %s (Graviton) for ~20%% cost reduction", r.Metadata["instance_type"], suggest),
	}
}

// OldRDSSnapshotRule flags manual RDS snapshots older than 30 days for deletion.
type OldRDSSnapshotRule struct{}

func (OldRDSSnapshotRule) Evaluate(r scanner.Resource) *Finding {
	if r.Type != "rds-snapshot" {
		return nil
	}
	sizeGB := 0
	fmt.Sscanf(r.Metadata["size_gb"], "%d", &sizeGB)
	savings := float64(sizeGB) * 0.095 // RDS snapshot ~$0.095/GB/month
	return &Finding{
		Resource:               r,
		Action:                 ActionDelete,
		Reason:                 fmt.Sprintf("Manual RDS snapshot older than 30 days (created: %s)", r.Metadata["created"]),
		EstimatedMonthlySavings: savings,
	}
}

// ZeroTrafficALBRule flags ALBs with no traffic in 7 days for deletion.
type ZeroTrafficALBRule struct{}

func (ZeroTrafficALBRule) Evaluate(r scanner.Resource) *Finding {
	if r.Type != "alb" {
		return nil
	}
	return &Finding{
		Resource:               r,
		Action:                 ActionDelete,
		Reason:                 "ALB has received zero requests in the last 7 days (~$16/month idle)",
		EstimatedMonthlySavings: 16.00,
	}
}

// UnusedSGRule flags security groups not attached to any ENI for deletion.
type UnusedSGRule struct{}

func (UnusedSGRule) Evaluate(r scanner.Resource) *Finding {
	if r.Type != "security-group" {
		return nil
	}
	return &Finding{
		Resource: r,
		Action:   ActionDelete,
		Reason:   fmt.Sprintf("Security group '%s' is not attached to any network interface", r.Metadata["name"]),
	}
}
