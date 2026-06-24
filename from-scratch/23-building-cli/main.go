// FS-23: Building a Modern CLI with Cobra
// Demonstrates: subcommands, flags, args validation, Viper config,
// output formats, shell completion, persistent flags, error handling.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Build-time variables (set via ldflags: -X main.version=v1.0.0)
var (
	version   = "dev"
	buildTime = "unknown"
	commit    = "none"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── Root command ──────────────────────────────────────────────────────────

var rootCmd = &cobra.Command{
	Use:   "devctl",
	Short: "Platform engineering CLI",
	Long: `devctl — a modern platform engineering CLI.
Manage deployments, inspect pods, and configure clusters.`,

	// PersistentPreRun runs before every subcommand — good for config loading
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return loadConfig()
	},
}

func init() {
	// Persistent flags — available to ALL subcommands
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default ~/.devctl/config.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	// Bind persistent flags to viper so they can also be set via env/config
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Register all subcommands
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(podsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

func loadConfig() error {
	cfgFile, _ := rootCmd.PersistentFlags().GetString("config")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(home + "/.devctl")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv()                    // DEVCTL_VERBOSE=true overrides flag
	viper.SetEnvPrefix("DEVCTL")
	viper.ReadInConfig()                    // silently ignore if no config file
	return nil
}

// ── deploy command ────────────────────────────────────────────────────────

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy an application",
	Example: `  devctl deploy --env prod --image myapp:v1.2
  devctl deploy --env staging --image myapp:v1.2 --dry-run`,
	RunE: runDeploy,
}

func init() {
	// Local flags — only on deploy
	deployCmd.Flags().StringP("env", "e", "", "target environment (required)")
	deployCmd.Flags().StringP("image", "i", "", "Docker image to deploy (required)")
	deployCmd.Flags().Bool("dry-run", false, "print what would happen, don't actually deploy")
	deployCmd.Flags().IntP("replicas", "r", 1, "number of replicas")
	deployCmd.Flags().Duration("timeout", 5*time.Minute, "deployment timeout")

	// Mark required flags — Cobra returns error if not provided
	deployCmd.MarkFlagRequired("env")
	deployCmd.MarkFlagRequired("image")

	// Tab completion for --env flag
	deployCmd.RegisterFlagCompletionFunc("env", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dev", "staging", "prod"}, cobra.ShellCompDirectiveDefault
	})
}

func runDeploy(cmd *cobra.Command, args []string) error {
	env, _ := cmd.Flags().GetString("env")
	image, _ := cmd.Flags().GetString("image")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	replicas, _ := cmd.Flags().GetInt("replicas")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	verbose := viper.GetBool("verbose")

	if verbose {
		fmt.Printf("Config: %s\n", viper.ConfigFileUsed())
	}

	if dryRun {
		fmt.Printf("[DRY RUN] Would deploy:\n")
		fmt.Printf("  image:    %s\n", image)
		fmt.Printf("  env:      %s\n", env)
		fmt.Printf("  replicas: %d\n", replicas)
		fmt.Printf("  timeout:  %s\n", timeout)
		return nil
	}

	fmt.Printf("Deploying %s to %s (%d replicas)...\n", image, env, replicas)
	// In a real CLI: call K8s API, update Helm chart, etc.
	fmt.Println("✓ Deployment complete")
	return nil
}

// ── pods command (group — no Run) ─────────────────────────────────────────

var podsCmd = &cobra.Command{
	Use:   "pods",
	Short: "Manage pods",
}

func init() {
	podsCmd.AddCommand(podsListCmd)
	podsCmd.AddCommand(podsLogsCmd)
}

// pods list
var podsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pods",
	RunE: func(cmd *cobra.Command, args []string) error {
		ns, _ := cmd.Flags().GetString("namespace")
		output, _ := cmd.Flags().GetString("output")

		// Mock data — in real CLI: kubectl/client-go call
		pods := []map[string]string{
			{"name": "api-abc123", "status": "Running", "node": "node-1"},
			{"name": "api-def456", "status": "Running", "node": "node-2"},
			{"name": "worker-xyz", "status": "Pending", "node": ""},
		}

		switch output {
		case "json":
			return json.NewEncoder(cmd.OutOrStdout()).Encode(pods)
		case "wide":
			fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-10s %-10s\n", "NAME", "STATUS", "NODE")
			for _, p := range pods {
				fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-10s %-10s\n", p["name"], p["status"], p["node"])
			}
		default: // table
			fmt.Fprintf(cmd.OutOrStdout(), "Pods in namespace: %s\n", ns)
			for _, p := range pods {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s\n", p["name"], p["status"])
			}
		}
		return nil
	},
}

func init() {
	podsListCmd.Flags().StringP("namespace", "n", "default", "Kubernetes namespace")
	podsListCmd.Flags().StringP("output", "o", "table", "output format: table|json|wide")
	// Completion for --output
	podsListCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "wide"}, cobra.ShellCompDirectiveDefault
	})
}

// pods logs <pod-name>
var podsLogsCmd = &cobra.Command{
	Use:   "logs <pod-name>",
	Short: "Stream pod logs",
	Args:  cobra.ExactArgs(1), // exactly one positional arg required
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0] // validated by cobra.ExactArgs
		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetInt("tail")

		fmt.Fprintf(cmd.OutOrStdout(), "Logs for pod: %s (tail=%d, follow=%v)\n", podName, tail, follow)
		// Mock log lines
		for i := tail; i > 0; i-- {
			fmt.Fprintf(cmd.OutOrStdout(), "[%s] INFO request handled in 12ms\n",
				time.Now().Format(time.RFC3339))
		}
		if follow {
			fmt.Fprintln(cmd.OutOrStdout(), "(following — Ctrl+C to stop)")
			// In real CLI: stream from K8s API
		}
		return nil
	},
}

func init() {
	podsLogsCmd.Flags().BoolP("follow", "f", false, "stream logs continuously")
	podsLogsCmd.Flags().Int("tail", 20, "number of recent lines to show")
}

// ── config command ────────────────────────────────────────────────────────

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key>=<value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parts := strings.SplitN(args[0], "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("format must be key=value, got: %s", args[0])
		}
		key, val := parts[0], parts[1]

		home, _ := os.UserHomeDir()
		cfgDir := home + "/.devctl"
		os.MkdirAll(cfgDir, 0755)

		viper.Set(key, val)
		viper.SetConfigFile(cfgDir + "/config.yaml")
		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Printf("Set %s = %s\n", key, val)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		val := viper.GetString(key)
		if val == "" {
			return fmt.Errorf("key %q not found", key)
		}
		fmt.Println(val)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
}

// ── version command ───────────────────────────────────────────────────────

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		short, _ := cmd.Flags().GetBool("short")
		if short {
			fmt.Println(version)
			return
		}
		fmt.Printf("devctl %s\ncommit: %s\nbuilt:  %s\n", version, commit, buildTime)
	},
}

func init() {
	versionCmd.Flags().Bool("short", false, "print version number only")
}
