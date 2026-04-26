package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tour-of-go/aws-resource-reaper/internal/auth"
	"github.com/tour-of-go/aws-resource-reaper/internal/config"
	"github.com/tour-of-go/aws-resource-reaper/internal/executor"
	"github.com/tour-of-go/aws-resource-reaper/internal/report"
	"github.com/tour-of-go/aws-resource-reaper/internal/rules"
	"github.com/tour-of-go/aws-resource-reaper/internal/scanner"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan AWS resources and report FinOps findings (dry-run by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScan(cmd.Context(), false)
	},
}

var executeCmd = &cobra.Command{
	Use:   "execute",
	Short: "Scan and execute remediation actions (requires --dry-run=false)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScan(cmd.Context(), true)
	},
}

func runScan(ctx context.Context, execute bool) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if concurrency > 0 {
		cfg.Concurrency = concurrency
	}

	authProvider, err := auth.New(ctx)
	if err != nil {
		return fmt.Errorf("initializing auth: %w", err)
	}

	engine := &scanner.Engine{
		Auth:     authProvider,
		Scanners: scanner.DefaultScanners(),
	}

	fmt.Fprintf(os.Stderr, "Scanning %d account(s) × %d region(s) with concurrency=%d...\n",
		len(cfg.Accounts), len(cfg.Regions), cfg.Concurrency)

	resources, err := engine.Scan(ctx, cfg)
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	findings := rules.DefaultEvaluator().Evaluate(resources)

	reporter := &report.Reporter{Format: outputFmt}
	if err := reporter.Print(findings, os.Stdout); err != nil {
		return err
	}

	if execute && !dryRun {
		// For execute subcommand, we need a base AWS config for the executor.
		// The executor re-scopes per resource region using the finding's AccountID.
		// Here we pass the base config; the executor sets cfg.Region per call.
		baseCfg, err := authProvider.ForAccount(ctx, cfg.Accounts[0])
		if err != nil {
			return err
		}
		ex := &executor.Executor{AwsConfig: baseCfg}
		return ex.Execute(ctx, findings)
	}

	if dryRun {
		fmt.Fprintln(os.Stderr, "\nDry-run mode: no changes made. Use 'execute --dry-run=false' to apply.")
	}
	return nil
}
