package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kdubb1337/phog-cli/internal/api"
	"github.com/kdubb1337/phog-cli/internal/output"
)

var (
	personsListSearch     string
	personsListDistinctID string
	personsListLimit      int
	personsListCursor     string
	personsGetEvents      bool
	personsGetAfter       string
)

var personsCmd = &cobra.Command{
	Use:     "persons",
	Short:   "Identified users (and their event history)",
	Aliases: []string{"person", "users", "user"},
}

var personsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List persons for the project",
	Example: `  phog persons list --limit 25 --json
  phog persons list --search "@example.com"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := api.New()
		if err != nil {
			return err
		}
		page, err := c.PersonsList(cmd.Context(), api.PersonsListParams{
			Search:     personsListSearch,
			DistinctID: personsListDistinctID,
			Limit:      personsListLimit,
			Cursor:     personsListCursor,
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

var personsGetCmd = &cobra.Command{
	Use:   "get <distinct_id>",
	Short: "Get one person by distinct_id (optionally with recent events)",
	Args:  cobra.ExactArgs(1),
	Example: `  phog persons get user_42 --json
  phog persons get user_42 --events --after 30d`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := api.New()
		if err != nil {
			return err
		}
		person, err := c.PersonGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if person.UUID == "" {
			return output.Errorf(3, "person_not_found", "no person found with distinct_id=%q", args[0])
		}
		if !personsGetEvents {
			return output.Emit(person)
		}
		after, err := api.ParseDuration(personsGetAfter)
		if err != nil {
			return output.Errorf(2, "bad_after", err.Error())
		}
		evPage, err := c.EventsList(cmd.Context(), api.EventsListParams{
			DistinctID: args[0],
			After:      after,
			Limit:      100,
		})
		if err != nil {
			return err
		}
		return output.Emit(map[string]any{
			"person": person,
			"events": evPage.Results,
		})
	},
}

func init() {
	personsListCmd.Flags().StringVar(&personsListSearch, "search", "", "match against name, distinct_id, email")
	personsListCmd.Flags().StringVar(&personsListDistinctID, "distinct-id", "", "exact distinct_id filter")
	personsListCmd.Flags().IntVar(&personsListLimit, "limit", 25, "max persons to return")
	personsListCmd.Flags().StringVar(&personsListCursor, "cursor", "", "pagination cursor from previous response")

	personsGetCmd.Flags().BoolVar(&personsGetEvents, "events", false, "also fetch this person's recent events")
	personsGetCmd.Flags().StringVar(&personsGetAfter, "after", "30d", "with --events, event time lower bound (7d, 24h, RFC3339)")

	personsCmd.AddCommand(personsListCmd, personsGetCmd)
	rootCmd.AddCommand(personsCmd)
}
