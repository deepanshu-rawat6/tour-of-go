package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "detect",
		Short: "tf-drift-detector — detect Terraform drift across AWS resources",
		Long: `Compares Terraform state (S3 or local) against live AWS infrastructure.
Alerts on drift via Slack, Discord, or stdout.

Dry-run by default — use 'run' for one-shot or 'daemon' for continuous monitoring.`,
	}
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "path to config file")
	root.AddCommand(runCmd(), daemonCmd())
	return root
}
