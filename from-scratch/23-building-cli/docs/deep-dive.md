# 23-building-cli: Deep Dive

## Cobra Command Hierarchy

Every Cobra CLI is a tree of `*cobra.Command` objects. Each command has:
- `Use` — the name + args pattern shown in help
- `Short` — one-line description (shown in parent's help)
- `Long` — full description (shown in this command's help)
- `RunE` — the function to execute (returns error)
- `Args` — validates positional arguments before RunE runs

```
rootCmd (devctl)
├── deployCmd        (devctl deploy)
├── podsCmd          (devctl pods)
│   ├── podsListCmd  (devctl pods list)
│   └── podsLogsCmd  (devctl pods logs <pod-name>)
├── configCmd        (devctl config)
│   ├── configSetCmd (devctl config set key=value)
│   └── configGetCmd (devctl config get key)
└── versionCmd       (devctl version)
```

## Flags: Persistent vs Local

```go
// Persistent: available to this command AND all children
rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
//                         ^type  ^long     ^short ^default ^description

// Local: only on this specific command
deployCmd.Flags().StringP("env", "e", "", "environment")
```

**Flag types:**
```go
cmd.Flags().String("name", "default", "usage")   // string
cmd.Flags().Int("count", 0, "usage")              // int
cmd.Flags().Bool("dry-run", false, "usage")       // bool (--dry-run sets true)
cmd.Flags().Duration("timeout", 5*time.Minute, "usage") // time.Duration
cmd.Flags().StringSlice("tags", nil, "usage")     // --tags a,b,c or --tags a --tags b
```

**Reading flag values:**
```go
// Inside RunE:
name, _ := cmd.Flags().GetString("name")
dryRun, _ := cmd.Flags().GetBool("dry-run")
// Or bind to a variable at init time (cleaner):
var env string
deployCmd.Flags().StringVarP(&env, "env", "e", "", "environment")
// Now env is set before RunE is called
```

## Args Validation

Cobra validates positional arguments before calling RunE:

```go
Args: cobra.ExactArgs(1)       // exactly 1 arg required
Args: cobra.MinimumNArgs(1)    // at least 1
Args: cobra.MaximumNArgs(3)    // at most 3
Args: cobra.RangeArgs(1, 3)    // between 1 and 3
Args: cobra.NoArgs             // no args allowed
Args: cobra.ArbitraryArgs      // anything goes (default)

// Custom validation:
Args: func(cmd *cobra.Command, args []string) error {
    if !strings.HasPrefix(args[0], "v") {
        return fmt.Errorf("version must start with 'v', got: %s", args[0])
    }
    return nil
},
```

## Viper — Config + Flags + Env Integration

Viper lets you read from flags, config files, and environment variables with automatic precedence:

```
Priority (highest to lowest):
  1. Explicit Set (viper.Set("key", val))
  2. Flag value (--verbose)
  3. Environment variable (DEVCTL_VERBOSE=true)
  4. Config file (~/.devctl/config.yaml)
  5. Default value
```

```go
// Bind a flag to viper — now readable via viper.GetBool("verbose")
viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

// Env vars: DEVCTL_CLUSTER maps to viper key "cluster"
viper.SetEnvPrefix("DEVCTL")
viper.AutomaticEnv()

// Read in any order — viper resolves precedence automatically
cluster := viper.GetString("cluster")
```

## Shell Completion

Cobra generates completion scripts for bash/zsh/fish/PowerShell automatically:

```bash
# Generate and install bash completion
devctl completion bash > /etc/bash_completion.d/devctl

# zsh
devctl completion zsh > "${fpath[1]}/_devctl"

# fish
devctl completion fish > ~/.config/fish/completions/devctl.fish
```

**Custom completion for flag values:**

```go
deployCmd.RegisterFlagCompletionFunc("env",
    func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        // Return completions that start with toComplete
        envs := []string{"dev", "staging", "prod"}
        return envs, cobra.ShellCompDirectiveDefault
    })

// Now: devctl deploy --env <TAB>  →  dev  staging  prod
```

**Custom completion for positional args:**

```go
podsLogsCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    // Fetch actual pod names from K8s API
    pods := []string{"api-abc123", "api-def456", "worker-xyz"}
    return pods, cobra.ShellCompDirectiveDefault
}
// Now: devctl pods logs <TAB>  →  api-abc123  api-def456  worker-xyz
```

## Testing a Cobra CLI

```go
func TestDeployDryRun(t *testing.T) {
    // Capture output via cmd.SetOut
    var buf bytes.Buffer
    rootCmd.SetOut(&buf)

    // Set args programmatically
    rootCmd.SetArgs([]string{"deploy", "--env", "staging", "--image", "myapp:v1", "--dry-run"})
    err := rootCmd.Execute()

    assert.NoError(t, err)
    assert.Contains(t, buf.String(), "[DRY RUN]")
    assert.Contains(t, buf.String(), "myapp:v1")
}
```

## Build-Time Version Injection

```makefile
# Inject version, commit, buildtime at compile time — zero runtime cost
go build \
  -ldflags="-X main.version=v1.2.3 \
             -X main.commit=$(git rev-parse --short HEAD) \
             -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o devctl .
```

```bash
$ devctl version
devctl v1.2.3
commit: abc1234
built:  2024-01-15T10:30:00Z
```

## Distribution with GoReleaser

```yaml
# .goreleaser.yaml
builds:
  - main: .
    binary: devctl
    ldflags:
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.buildTime={{.Date}}
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

archives:
  - format: tar.gz
    name_template: "devctl_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

brews:
  - name: devctl
    homepage: https://github.com/my-org/devctl
    tap:
      owner: my-org
      name: homebrew-tap
```

```bash
# Release:
goreleaser release --clean
# Publishes: GitHub release + brew tap + checksums
```

## Real-World Cobra CLIs to Study

| CLI | What to learn from it |
|-----|----------------------|
| `kubectl` | Complex subcommand trees, dynamic completion, plugin system |
| `gh` (GitHub CLI) | OAuth flow, API integration, output formatting |
| `helm` | Config management, template rendering, error messages |
| `docker` | Daemon communication, streaming output, TUI elements |
