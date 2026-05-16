package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kdubb1337/phog-cli/internal/config"
	"github.com/kdubb1337/phog-cli/internal/output"
)

// ExitCoder is the interface satisfied by errors that carry a process exit code.
// Both API errors and usage errors implement this; main.go reads it to set os.Exit.
type ExitCoder interface {
	error
	ExitCode() int
}

var (
	// Persistent flags — available on every subcommand.
	flagJSON      bool
	flagHuman     bool
	flagCSV       bool
	flagCompact   bool
	flagSelect    string
	flagKeepNulls bool
	flagAccount   string
	flagProfile   string
	flagQuiet     bool
	flagVerbose   bool
	flagDebug     bool
	flagNoInput   bool
	flagYes       bool
	flagForce     bool
	flagDryRun    bool

	// Set by goreleaser via -ldflags.
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "phog",
	Short: "Agent-native CLI for PostHog (events, web activity, insights, persons, HogQL)",
	Long: `phog is a hand-crafted, agent-native CLI for PostHog.

It wraps the PostHog REST API and HogQL query endpoint so agents can query
events, $pageview-style web activity, saved insights, persons, and run
arbitrary HogQL — all with deterministic output, typed exit codes, and
--dry-run on every mutation.

The binary is intentionally named "phog" (not "posthog") to avoid colliding
with PostHog's official posthog-cli (Rust) on PATH.

Output rules:
  - stdout is data; stderr is human progress and errors.
  - When stdout is piped, JSON is emitted by default.
  - Exit codes:
      0=ok 1=generic 2=usage 3=not-found 4=auth 5=api
      6=conflict 7=rate-limit 8=network 9=validation 124=timeout

Discover the structured schema with:
  phog agent-context

See the bundled skill (commands + workflows for agents) with:
  phog skill path

Install the bundled skill into your agent(s) of choice:
  phog skill install claude codex
  phog skill install --all`,
	Version:           fmt.Sprintf("%s (%s, %s)", version, commit, date),
	SilenceUsage:      true,
	SilenceErrors:     true,
	PersistentPreRunE: bindFlagsAndProfile,
}

// Execute runs the root command.
func Execute() error { return rootCmd.Execute() }

func init() {
	pf := rootCmd.PersistentFlags()
	pf.BoolVar(&flagJSON, "json", false, "emit JSON on stdout (default when stdout is piped)")
	pf.BoolVar(&flagHuman, "human", false, "force human-readable table output even when piped")
	pf.BoolVar(&flagCSV, "csv", false, "emit CSV with header row")
	pf.BoolVar(&flagCompact, "compact", false, "drop to high-gravity fields only (id, name, status, primary timestamp)")
	pf.StringVar(&flagSelect, "select", "", "comma-separated field projection (e.g. --select=id,name)")
	pf.BoolVar(&flagKeepNulls, "keep-nulls", false, "keep null-valued keys in output (default: stripped to save tokens)")
	pf.StringVar(&flagAccount, "account", os.Getenv("PHOG_ACCOUNT"), "account to use (env: PHOG_ACCOUNT)")
	pf.StringVar(&flagProfile, "profile", "", "named profile of saved configuration")
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "suppress progress output on stderr")
	pf.BoolVar(&flagVerbose, "verbose", false, "verbose progress on stderr")
	pf.BoolVar(&flagDebug, "debug", false, "debug-level progress on stderr")
	pf.BoolVar(&flagNoInput, "no-input", false, "disable all interactive prompts; fail closed")
	pf.BoolVar(&flagYes, "yes", false, "answer yes to confirmation prompts")
	pf.BoolVar(&flagForce, "force", false, "force destructive operations (no confirmation)")
	pf.BoolVar(&flagDryRun, "dry-run", false, "show what would happen without doing it")

	rootCmd.MarkFlagsMutuallyExclusive("json", "human", "csv")
	rootCmd.MarkFlagsMutuallyExclusive("quiet", "verbose", "debug")
}

func bindFlagsAndProfile(cmd *cobra.Command, args []string) error {
	// 1. Resolve output mode: explicit flag > TTY detection > default JSON when piped.
	output.Configure(output.Options{
		JSON:      flagJSON,
		Human:     flagHuman,
		CSV:       flagCSV,
		Compact:   flagCompact,
		Select:    flagSelect,
		KeepNulls: flagKeepNulls,
		Quiet:     flagQuiet,
		Verbose:   flagVerbose,
		Debug:     flagDebug,
	})
	// 2. Resolve identity: --profile -> --account -> $PHOG_ACCOUNT -> stored default.
	return config.Resolve(flagProfile, flagAccount)
}
