package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	outputFmt   string
	concurrency int
	dryRun      bool
)

var rootCmd = &cobra.Command{
	Use:   "reaper",
	Short: "AWS Resource Reaper — FinOps tool for scanning and cleaning idle AWS resources",
	Long: `AWS Resource Reaper scans resources across multiple AWS accounts and regions,
evaluates them against FinOps rules, and reports (or removes) idle/wasteful resources.

Deployed on EC2/ECS in a central management account; assumes IAM roles into target accounts.

Dry-run mode is ON by default. Pass --dry-run=false to enable live execution.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "path to YAML config file")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "text", "output format: text|json")
	rootCmd.PersistentFlags().IntVar(&concurrency, "concurrency", 0, "max concurrent account×region scans (0 = use config value)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", true, "dry-run mode (default true); set false to execute deletions")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(executeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
