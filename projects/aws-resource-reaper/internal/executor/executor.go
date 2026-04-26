package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/tour-of-go/aws-resource-reaper/internal/rules"
)

// Executor carries out destructive actions against AWS resources.
type Executor struct {
	AwsConfig aws.Config
	In        io.Reader // for confirmation prompt; defaults to os.Stdin
}

// Execute prints the plan, prompts for confirmation, then executes each finding.
// Recommend findings are printed as advisory only — no API call is made.
func (e *Executor) Execute(ctx context.Context, findings []rules.Finding) error {
	destructive := countDestructive(findings)
	if destructive == 0 {
		fmt.Println("No destructive actions to execute.")
		return nil
	}

	fmt.Printf("\n⚠️  About to execute %d destructive action(s). Recommendations are advisory only.\n", destructive)
	fmt.Printf("Proceed? [y/N]: ")

	reader := e.In
	if reader == nil {
		reader = os.Stdin
	}
	scanner := bufio.NewScanner(reader)
	scanner.Scan()
	if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	for _, f := range findings {
		if f.Action == rules.ActionRecommend {
			logger.Info("advisory", "resource_id", f.Resource.ID, "type", f.Resource.Type,
				"account", f.Resource.AccountID, "region", f.Resource.Region, "reason", f.Reason)
			continue
		}

		cfg := e.AwsConfig
		cfg.Region = f.Resource.Region

		var execErr error
		switch f.Resource.Type {
		case "ebs-volume":
			execErr = deleteEBSVolume(ctx, cfg, f.Resource.ID)
		case "elastic-ip":
			execErr = releaseElasticIP(ctx, cfg, f.Resource.ID)
		case "ec2-instance":
			execErr = terminateEC2Instance(ctx, cfg, f.Resource.ID)
		case "rds-snapshot":
			execErr = deleteRDSSnapshot(ctx, cfg, f.Resource.ID)
		case "alb":
			execErr = deleteALB(ctx, cfg, f.Resource.ID)
		case "security-group":
			execErr = deleteSecurityGroup(ctx, cfg, f.Resource.ID)
		default:
			logger.Warn("unknown resource type, skipping", "type", f.Resource.Type, "id", f.Resource.ID)
			continue
		}

		if execErr != nil {
			logger.Error("action failed",
				"resource_id", f.Resource.ID, "type", f.Resource.Type,
				"account", f.Resource.AccountID, "region", f.Resource.Region,
				"action", f.Action, "error", execErr.Error())
		} else {
			logger.Info("action executed",
				"resource_id", f.Resource.ID, "type", f.Resource.Type,
				"account", f.Resource.AccountID, "region", f.Resource.Region,
				"action", f.Action, "reason", f.Reason)
		}
	}
	return nil
}

func countDestructive(findings []rules.Finding) int {
	n := 0
	for _, f := range findings {
		if f.Action != rules.ActionRecommend && f.Action != rules.ActionSkip {
			n++
		}
	}
	return n
}

func deleteEBSVolume(ctx context.Context, cfg aws.Config, id string) error {
	_, err := ec2.NewFromConfig(cfg).DeleteVolume(ctx, &ec2.DeleteVolumeInput{VolumeId: aws.String(id)})
	return err
}

func releaseElasticIP(ctx context.Context, cfg aws.Config, allocationID string) error {
	_, err := ec2.NewFromConfig(cfg).ReleaseAddress(ctx, &ec2.ReleaseAddressInput{AllocationId: aws.String(allocationID)})
	return err
}

func terminateEC2Instance(ctx context.Context, cfg aws.Config, id string) error {
	_, err := ec2.NewFromConfig(cfg).TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{id}})
	return err
}

func deleteRDSSnapshot(ctx context.Context, cfg aws.Config, id string) error {
	_, err := rds.NewFromConfig(cfg).DeleteDBSnapshot(ctx, &rds.DeleteDBSnapshotInput{DBSnapshotIdentifier: aws.String(id)})
	return err
}

func deleteALB(ctx context.Context, cfg aws.Config, arn string) error {
	_, err := elasticloadbalancingv2.NewFromConfig(cfg).DeleteLoadBalancer(ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{LoadBalancerArn: aws.String(arn)})
	return err
}

func deleteSecurityGroup(ctx context.Context, cfg aws.Config, id string) error {
	_, err := ec2.NewFromConfig(cfg).DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: aws.String(id)})
	return err
}
