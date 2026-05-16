package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/kdubb1337/phog-cli/internal/api"
	"github.com/kdubb1337/phog-cli/internal/output"
)

var (
	queryFile string
	queryRaw  bool
)

var queryCmd = &cobra.Command{
	Use:   "query [hogql]",
	Short: "Run a HogQL (SQL-like) query against the project",
	Long: `Run an arbitrary HogQL query against the project. HogQL is PostHog's
SQL dialect; it can read 'events', 'persons', 'sessions', 'cohort_people',
and other virtual tables.

Pass the query as a positional argument, via --file, or piped on stdin.`,
	Example: `  # Top 20 events in the last day
  phog query "SELECT event, count() FROM events WHERE timestamp > now() - INTERVAL 1 DAY GROUP BY event ORDER BY count() DESC LIMIT 20"

  # Same, from a file
  phog query --file ./top-events.sql --json

  # From stdin
  cat funnel.sql | phog query

  # Just the result rows (no columns metadata)
  phog query "SELECT 1" --raw`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q, err := resolveQueryText(args)
		if err != nil {
			return err
		}
		if q == "" {
			return output.Errorf(2, "missing_query",
				"provide a HogQL query as a positional arg, via --file, or on stdin")
		}
		c, err := api.New()
		if err != nil {
			return err
		}
		res, err := c.Query(cmd.Context(), q)
		if err != nil {
			return err
		}
		if queryRaw {
			return output.Emit(res.Results)
		}
		return output.Emit(res)
	},
}

func resolveQueryText(args []string) (string, error) {
	switch {
	case queryFile != "":
		b, err := os.ReadFile(queryFile)
		if err != nil {
			return "", output.Errorf(2, "read_file", "read --file: %v", err)
		}
		return string(b), nil
	case len(args) == 1:
		return args[0], nil
	default:
		fi, err := os.Stdin.Stat()
		if err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", output.Errorf(2, "read_stdin", "read stdin: %v", err)
			}
			return string(b), nil
		}
		return "", nil
	}
}

func init() {
	queryCmd.Flags().StringVar(&queryFile, "file", "", "path to a file containing the HogQL query")
	queryCmd.Flags().BoolVar(&queryRaw, "raw", false, "emit only the result rows (drop columns/types/timings)")

	rootCmd.AddCommand(queryCmd)
}
