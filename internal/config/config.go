package config

import (
	"os"
	"strings"
)

// Settings holds resolved configuration for the PostHog client.
// Precedence: explicit flag > env var > stored profile > zero value.
// (Profile-file loading is stubbed; wire ~/.phog/config.yaml here when needed.)
type Settings struct {
	APIKey    string
	Host      string
	ProjectID string
	Profile   string
	Account   string
}

var current Settings

// Resolve merges flag values with env vars and stores the result for retrieval
// by API clients via Get(). Called from root.go PersistentPreRunE.
func Resolve(profile, account string) error {
	if account == "" {
		account = os.Getenv("PHOG_ACCOUNT")
	}
	current = Settings{
		APIKey:    os.Getenv("PHOG_API_KEY"),
		Host:      strings.TrimRight(envDefault("PHOG_HOST", "https://us.posthog.com"), "/"),
		ProjectID: os.Getenv("PHOG_PROJECT_ID"),
		Profile:   profile,
		Account:   account,
	}
	return nil
}

// Get returns the resolved settings. Commands call this to build API clients.
func Get() Settings { return current }

// SetAPIKey overrides the in-memory API key (used by `phog auth add` flows).
func SetAPIKey(k string) { current.APIKey = k }

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
