package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
	"github.com/tour-of-go/tf-drift-detector/internal/alert"
	"github.com/tour-of-go/tf-drift-detector/internal/config"
	"github.com/tour-of-go/tf-drift-detector/internal/poller"
	"github.com/tour-of-go/tf-drift-detector/internal/state"
	"github.com/tour-of-go/tf-drift-detector/internal/tracker"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run a one-shot drift check and exit",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}
			return runCycle(cmd.Context(), cfg)
		},
	}
}

func daemonCmd() *cobra.Command {
	var intervalFlag string
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run drift checks continuously at a set interval",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}
			if intervalFlag != "" {
				cfg.Interval = intervalFlag
			}
			interval, err := cfg.ParseInterval()
			if err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
			defer stop()

			trk := tracker.New(cfg.DriftStateFile)
			defer func() {
				if err := trk.Persist(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to persist drift state: %v\n", err)
				}
				fmt.Fprintln(os.Stderr, "Drift state persisted. Shutting down.")
			}()

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			fmt.Fprintf(os.Stderr, "Daemon started. Interval: %s. Press Ctrl+C to stop.\n", interval)

			// Run immediately on start, then on each tick
			if err := runCycleWithTracker(ctx, cfg, trk); err != nil {
				fmt.Fprintf(os.Stderr, "cycle error: %v\n", err)
			}

			for {
				select {
				case <-ticker.C:
					if err := runCycleWithTracker(ctx, cfg, trk); err != nil {
						fmt.Fprintf(os.Stderr, "cycle error: %v\n", err)
					}
				case <-ctx.Done():
					return nil
				}
			}
		},
	}
	cmd.Flags().StringVar(&intervalFlag, "interval", "", "Override check interval (e.g. 5m, 30s)")
	return cmd
}

// runCycle is the one-shot path: load state → poll → diff → alert (no tracker).
func runCycle(ctx context.Context, cfg *config.Config) error {
	awsCfg, err := loadAWSConfig(ctx, cfg.AWSRegion)
	if err != nil {
		return err
	}
	resources, err := state.Load(ctx, cfg.Backend)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Loaded %d managed resources from state.\n", len(resources))

	engine := &poller.Engine{
		Comparators:  poller.DefaultComparators(),
		IgnoreFields: cfg.IgnoreFields,
		Concurrency:  cfg.Concurrency,
	}
	results, err := engine.Poll(ctx, awsCfg, resources)
	if err != nil {
		return fmt.Errorf("polling: %w", err)
	}

	var drifted []poller.DriftResult
	for _, r := range results {
		if r.Drifted {
			drifted = append(drifted, r)
		}
	}

	alerter := buildAlerter(cfg)
	return alerter.Send(ctx, drifted, nil)
}

// runCycleWithTracker is the daemon path: same as runCycle but uses the tracker.
func runCycleWithTracker(ctx context.Context, cfg *config.Config, trk *tracker.Tracker) error {
	awsCfg, err := loadAWSConfig(ctx, cfg.AWSRegion)
	if err != nil {
		return err
	}
	resources, err := state.Load(ctx, cfg.Backend)
	if err != nil {
		return err
	}

	engine := &poller.Engine{
		Comparators:  poller.DefaultComparators(),
		IgnoreFields: cfg.IgnoreFields,
		Concurrency:  cfg.Concurrency,
	}
	results, err := engine.Poll(ctx, awsCfg, resources)
	if err != nil {
		return err
	}

	newDrift, resolved := trk.Update(results)
	if len(newDrift) == 0 && len(resolved) == 0 {
		fmt.Fprintf(os.Stderr, "[%s] No new drift. Known drifted resources: %d\n",
			time.Now().Format(time.RFC3339), trk.KnownCount())
		return nil
	}

	alerter := buildAlerter(cfg)
	return alerter.Send(ctx, newDrift, resolved)
}

func buildAlerter(cfg *config.Config) *alert.MultiAlerter {
	var alerters []alert.Alerter
	if cfg.Alerts.Slack.WebhookURL != "" {
		alerters = append(alerters, alert.SlackAlerter{WebhookURL: cfg.Alerts.Slack.WebhookURL})
	}
	if cfg.Alerts.Discord.WebhookURL != "" {
		alerters = append(alerters, alert.DiscordAlerter{WebhookURL: cfg.Alerts.Discord.WebhookURL})
	}
	if cfg.Alerts.Stdout || len(alerters) == 0 {
		alerters = append(alerters, alert.StdoutAlerter{})
	}
	return alert.NewMulti(alerters...)
}

func loadAWSConfig(ctx context.Context, region string) (aws.Config, error) {
	opts := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}
	return awsconfig.LoadDefaultConfig(ctx, opts...)
}
