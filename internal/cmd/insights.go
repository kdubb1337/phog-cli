package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kdubb1337/phog-cli/internal/api"
	"github.com/kdubb1337/phog-cli/internal/output"
)

var (
	insightsListSearch string
	insightsListLimit  int
	insightsListCursor string
)

var insightsCmd = &cobra.Command{
	Use:     "insights",
	Short:   "Saved insights (precomputed queries / dashboard tiles)",
	Aliases: []string{"insight"},
}

var insightsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved insights for the project",
	Example: `  phog insights list --limit 25 --json
  phog insights list --search "signup funnel" --select=id,short_id,name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := api.New()
		if err != nil {
			return err
		}
		page, err := c.InsightsList(cmd.Context(), api.InsightsListParams{
			Search: insightsListSearch,
			Limit:  insightsListLimit,
			Cursor: insightsListCursor,
		})
		if err != nil {
			return err
		}
		cursor := api.FollowCursor(page.Next)
		hint := ""
		if cursor != "" {
			hint = "more results available; pass --cursor=" + cursor
		}
		return output.EmitPage(page.Results, cursor, hint)
	},
}

var insightsGetCmd = &cobra.Command{
	Use:   "get <id-or-short-id>",
	Short: "Get one insight (with its computed result) by numeric ID or short_id",
	Args:  cobra.ExactArgs(1),
	Example: `  phog insights get 12345 --json
  phog insights get aBc123 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := api.New()
		if err != nil {
			return err
		}
		ins, err := c.InsightGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return output.Emit(ins)
	},
}

func init() {
	insightsListCmd.Flags().StringVar(&insightsListSearch, "search", "", "filter by name/description substring")
	insightsListCmd.Flags().IntVar(&insightsListLimit, "limit", 25, "max insights to return")
	insightsListCmd.Flags().StringVar(&insightsListCursor, "cursor", "", "pagination cursor from previous response")

	insightsCmd.AddCommand(insightsListCmd, insightsGetCmd)
	rootCmd.AddCommand(insightsCmd)
}
