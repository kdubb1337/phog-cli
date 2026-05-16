package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kdubb1337/phog-cli/internal/config"
	"github.com/kdubb1337/phog-cli/internal/output"
)

var (
	authAddToken     string
	authAddProject   string
	authAddHost      string
	authAddProfile   string
	authAddSetActive bool
	authRemoveName   string
)

var authAddCmd = &cobra.Command{
	Use:   "add [token]",
	Short: "Save a PostHog Personal API key + project ID to a profile",
	Long: `Save a Personal API key (phx_*) plus project ID (and optional host) to
~/.phog/config.json under a named profile. After this, you can run any
phog command without setting PHOG_API_KEY / PHOG_PROJECT_ID env vars.

The token may be passed as a positional arg, via --token, piped on stdin,
or prompted for interactively if stdin is a TTY.

The first profile created becomes the active profile automatically.`,
	Example: `  # Interactive
  phog auth add --project 12345
  # then paste the phx_ token when prompted

  # Non-interactive (one-liner)
  phog auth add phx_yourtoken --project 12345

  # With explicit host (EU cloud, or self-hosted)
  phog auth add phx_yourtoken --project 12345 --host https://eu.posthog.com

  # Save to a named profile and switch to it
  phog auth add phx_yourtoken --project 67890 --profile prod --use

  # Pipe the token (e.g. from a secret manager)
  op read "op://Personal/PostHog/token" | phog auth add --project 12345`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := resolveTokenInput(args)
		if err != nil {
			return err
		}
		if token == "" {
			return output.ErrorfHint(2, "missing_token",
				"pass the token as a positional arg, --token, or pipe it on stdin",
				"no token provided")
		}
		if !strings.HasPrefix(token, "phx_") {
			output.Progress("warning: token does not start with 'phx_' — PostHog project keys (phc_*) cannot query the API.")
		}
		if authAddProject == "" {
			return output.ErrorfHint(2, "missing_project",
				"pass --project <numeric id from PostHog URL /project/<ID>/...>",
				"--project is required")
		}
		profile := authAddProfile
		if profile == "" {
			profile = "default"
		}
		f, err := config.SetProfile(profile, config.Profile{
			APIKey:    token,
			ProjectID: authAddProject,
			Host:      authAddHost,
		}, authAddSetActive)
		if err != nil {
			return output.Errorf(1, "save_profile", "save profile: %v", err)
		}
		path, _ := config.Path()
		output.Progress("saved profile %q to %s (active: %s)", profile, path, f.Active)
		return output.Emit(map[string]any{
			"profile":     profile,
			"active":      f.Active,
			"config_path": path,
			"stored":      config.Redact(f.Profiles[profile]),
		})
	},
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved profiles (tokens redacted)",
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := config.Load()
		if err != nil {
			return output.Errorf(1, "load_config", "%v", err)
		}
		path, _ := config.Path()
		type row struct {
			Name    string `json:"name"`
			Active  bool   `json:"active"`
			APIKey  string `json:"api_key,omitempty"`
			Project string `json:"project_id,omitempty"`
			Host    string `json:"host,omitempty"`
		}
		rows := make([]row, 0, len(f.Profiles))
		for name, p := range f.Profiles {
			r := config.Redact(p)
			rows = append(rows, row{
				Name:    name,
				Active:  name == f.Active,
				APIKey:  r.APIKey,
				Project: r.ProjectID,
				Host:    r.Host,
			})
		}
		return output.Emit(map[string]any{
			"config_path": path,
			"active":      f.Active,
			"profiles":    rows,
		})
	},
}

var authRemoveCmd = &cobra.Command{
	Use:   "remove [profile]",
	Short: "Delete a saved profile (default: the current active profile)",
	Example: `  phog auth remove prod
  phog auth remove --force            # remove the active profile without confirmation`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := authRemoveName
		if name == "" && len(args) == 1 {
			name = args[0]
		}
		if name == "" {
			f, _ := config.Load()
			name = f.Active
		}
		if name == "" {
			return output.Errorf(2, "missing_name", "no profile specified and no active profile to remove")
		}
		if !flagForce && !flagYes {
			return output.ErrorfHint(2, "needs_force",
				"pass --force to confirm profile deletion",
				"refusing to delete profile %q without --force", name)
		}
		f, err := config.DeleteProfile(name)
		if err != nil {
			return output.Errorf(3, "delete_profile", "%v", err)
		}
		return output.Emit(map[string]any{
			"removed":   name,
			"active":    f.Active,
			"remaining": len(f.Profiles),
		})
	},
}

// resolveTokenInput pulls the token from (in order): positional arg,
// --token flag, piped stdin, interactive prompt (only when stdin is a TTY
// and --no-input is not set).
func resolveTokenInput(args []string) (string, error) {
	if len(args) == 1 && args[0] != "" {
		return strings.TrimSpace(args[0]), nil
	}
	if authAddToken != "" {
		return strings.TrimSpace(authAddToken), nil
	}
	fi, _ := os.Stdin.Stat()
	if fi != nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", output.Errorf(2, "read_stdin", "read stdin: %v", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	if flagNoInput {
		return "", nil
	}
	fmt.Fprint(os.Stderr, "PostHog Personal API key (phx_...): ")
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", output.Errorf(2, "read_prompt", "read prompt: %v", err)
	}
	return strings.TrimSpace(line), nil
}

func init() {
	authAddCmd.Flags().StringVar(&authAddToken, "token", "", "Personal API key (phx_*) — alternative to positional arg or stdin")
	authAddCmd.Flags().StringVar(&authAddProject, "project", "", "PostHog project ID (required; numeric, from URL /project/<ID>/...)")
	authAddCmd.Flags().StringVar(&authAddHost, "host", "", "PostHog host (default: https://us.posthog.com)")
	authAddCmd.Flags().StringVar(&authAddProfile, "profile", "default", "profile name to save under")
	authAddCmd.Flags().BoolVar(&authAddSetActive, "use", false, "also set this profile as active (default: only if no active profile)")

	authRemoveCmd.Flags().StringVar(&authRemoveName, "profile", "", "profile name to delete (default: active)")

	authCmd.AddCommand(authAddCmd, authListCmd, authRemoveCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage credentials (save / list / remove profiles)",
}

func init() {
	rootCmd.AddCommand(authCmd)
}
