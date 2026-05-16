package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kdubb1337/phog-cli/internal/config"
	"github.com/kdubb1337/phog-cli/internal/output"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage named configuration profiles (save / use / show / list / delete)",
	Long: `Profiles bundle a PostHog Personal API key, project ID, and host together
under a name. The active profile is consulted automatically when env vars
are absent, so you can run phog without exporting anything.

Use ` + "`phog auth add`" + ` to create or update the credentials in a profile.
Use ` + "`phog profile save`" + ` to update the non-token fields without touching
the key (e.g. point an existing profile at a different project ID). Use
` + "`phog profile get`" + ` to inspect one (tokens redacted).`,
}

var (
	profileSaveProject string
	profileSaveHost    string
	profileSaveUse     bool
)

var profileSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Create or update a profile's project_id / host (without touching the token)",
	Args:  cobra.ExactArgs(1),
	Example: `  phog profile save default --project 12345
  phog profile save eu --host https://eu.posthog.com --project 99 --use`,
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := config.SetProfile(args[0], config.Profile{
			ProjectID: profileSaveProject,
			Host:      profileSaveHost,
		}, profileSaveUse)
		if err != nil {
			return output.Errorf(1, "save_profile", "%v", err)
		}
		return output.Emit(map[string]any{
			"profile": args[0],
			"active":  f.Active,
			"stored":  config.Redact(f.Profiles[args[0]]),
		})
	},
}

var profileUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the active profile (consulted when no env vars / flags override)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := config.UseProfile(args[0])
		if err != nil {
			return output.Errorf(3, "use_profile", "%v", err)
		}
		return output.Emit(map[string]any{"active": f.Active})
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get one profile (default: the active profile)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := config.Load()
		if err != nil {
			return output.Errorf(1, "load_config", "%v", err)
		}
		name := f.Active
		if len(args) == 1 {
			name = args[0]
		}
		if name == "" {
			return output.Errorf(2, "no_active", "no active profile and no name given")
		}
		p, ok := f.Profiles[name]
		if !ok {
			return output.Errorf(3, "profile_not_found", "profile %q not found", name)
		}
		return output.Emit(map[string]any{
			"name":   name,
			"active": name == f.Active,
			"stored": config.Redact(p),
		})
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Reuse the same shape as auth list.
		return authListCmd.RunE(cmd, args)
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile (requires --force)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !flagForce && !flagYes {
			return output.ErrorfHint(2, "needs_force",
				"pass --force to confirm profile deletion",
				"refusing to delete profile %q without --force", args[0])
		}
		f, err := config.DeleteProfile(args[0])
		if err != nil {
			return output.Errorf(3, "delete_profile", "%v", err)
		}
		return output.Emit(map[string]any{
			"removed":   args[0],
			"active":    f.Active,
			"remaining": len(f.Profiles),
		})
	},
}

func init() {
	profileSaveCmd.Flags().StringVar(&profileSaveProject, "project", "", "PostHog project ID")
	profileSaveCmd.Flags().StringVar(&profileSaveHost, "host", "", "PostHog host (e.g. https://us.posthog.com)")
	profileSaveCmd.Flags().BoolVar(&profileSaveUse, "use", false, "also set this profile as active")

	profileCmd.AddCommand(profileSaveCmd, profileUseCmd, profileShowCmd, profileListCmd, profileDeleteCmd)
	rootCmd.AddCommand(profileCmd)
}
