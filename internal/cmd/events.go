package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kdubb1337/phog-cli/internal/api"
	"github.com/kdubb1337/phog-cli/internal/output"
)

var (
	eventsListEvent     string
	eventsListPerson    string
	eventsListAfter     string
	eventsListBefore    string
	eventsListLimit     int
	eventsListCursor    string
	eventsListPageviews bool
)

var eventsCmd = &cobra.Command{
	Use:     "events",
	Short:   "Query events (and $pageview web activity)",
	Aliases: []string{"event"},
}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events with optional filters (event name, person, time range)",
	Example: `  # Last 25 events (any type)
  phog events list --limit 25 --json

  # Page views in the last 24h
  phog events list --pageviews --after 24h

  # Page views in the last 7d, only URL + distinct_id
  phog events list --event '$pageview' --after 7d --select=timestamp,distinct_id,properties.$current_url

  # All events for one user
  phog events list --person 1a2b3c... --after 30d`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := api.New()
		if err != nil {
			return err
		}
		after, err := api.ParseDuration(eventsListAfter)
		if err != nil {
			return output.Errorf(2, "bad_after", err.Error())
		}
		before, err := api.ParseDuration(eventsListBefore)
		if err != nil {
			return output.Errorf(2, "bad_before", err.Error())
		}
		event := eventsListEvent
		if eventsListPageviews && event == "" {
			event = "$pageview"
		}
		page, err := c.EventsList(cmd.Context(), api.EventsListParams{
			Event:      event,
			DistinctID: eventsListPerson,
			After:      after,
			Before:     before,
			Limit:      eventsListLimit,
			Cursor:     eventsListCursor,
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

var eventsGetCmd = &cobra.Command{
	Use:     "get <event_id>",
	Short:   "Get a single event by ID",
	Args:    cobra.ExactArgs(1),
	Example: `  phog events get 01890b6c-... --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := api.New()
		if err != nil {
			return err
		}
		ev, err := c.EventGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return output.Emit(ev)
	},
}

func init() {
	eventsListCmd.Flags().StringVar(&eventsListEvent, "event", "", "filter to a single event name (e.g. '$pageview', '$autocapture', 'signup')")
	eventsListCmd.Flags().BoolVar(&eventsListPageviews, "pageviews", false, "shortcut for --event='$pageview' (web activity)")
	eventsListCmd.Flags().StringVar(&eventsListPerson, "person", "", "filter to a single distinct_id")
	eventsListCmd.Flags().StringVar(&eventsListAfter, "after", "", "lower bound: 7d, 24h, 30m, or RFC3339 timestamp")
	eventsListCmd.Flags().StringVar(&eventsListBefore, "before", "", "upper bound: 7d, 24h, 30m, or RFC3339 timestamp")
	eventsListCmd.Flags().IntVar(&eventsListLimit, "limit", 25, "max events to return (server-capped at 100)")
	eventsListCmd.Flags().StringVar(&eventsListCursor, "cursor", "", "pagination cursor from previous response")

	eventsCmd.AddCommand(eventsListCmd, eventsGetCmd)
	rootCmd.AddCommand(eventsCmd)
}
