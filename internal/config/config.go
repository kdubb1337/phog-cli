// Package config persists phog settings to ~/.phog/config.json so users can
// run the CLI without exporting env vars on every shell.
//
// Precedence when resolving a setting (highest wins):
//
//  1. Explicit flag (--account, --profile)
//  2. Env var (PHOG_API_KEY, PHOG_PROJECT_ID, PHOG_HOST, PHOG_PROFILE)
//  3. Stored profile selected by --profile flag
//  4. Stored profile pointed to by `active` in the config file
//  5. Profile named "default" if it exists
//  6. Zero values
//
// The config file is written with mode 0600 (user-readable only). Token
// storage is plaintext on disk; Steipete's S3 principle calls for OS keychain
// by default with a file backend escape hatch. We ship the escape hatch first;
// keychain is a v2 add.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Profile holds one PostHog connection's credentials and target.
type Profile struct {
	APIKey    string `json:"api_key,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	Host      string `json:"host,omitempty"`
}

// File is the on-disk shape of ~/.phog/config.json.
type File struct {
	Active   string             `json:"active,omitempty"`
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

// Settings is the resolved view a command sees after Resolve().
type Settings struct {
	APIKey      string
	Host        string
	ProjectID   string
	Profile     string // resolved profile name (empty if none)
	Account     string
	ConfigPath  string
	FromProfile bool // true if any field came from a stored profile
}

var current Settings

// Get returns the resolved settings. Commands call this to build API clients.
func Get() Settings { return current }

// Resolve merges flag values with env vars and stored profiles, then caches
// the result for retrieval by API clients via Get(). Called from
// root.go PersistentPreRunE.
func Resolve(profileFlag, accountFlag string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	file, _ := Load() // missing file is fine; empty config

	// Pick the profile name. Precedence: flag > env > file.active > "default".
	name := profileFlag
	if name == "" {
		name = os.Getenv("PHOG_PROFILE")
	}
	if name == "" {
		name = file.Active
	}
	if name == "" {
		if _, ok := file.Profiles["default"]; ok {
			name = "default"
		}
	}

	p, fromProfile := file.Profiles[name]

	// Env vars override profile values per-field.
	apiKey := firstNonEmpty(os.Getenv("PHOG_API_KEY"), p.APIKey)
	projectID := firstNonEmpty(os.Getenv("PHOG_PROJECT_ID"), p.ProjectID)
	host := firstNonEmpty(os.Getenv("PHOG_HOST"), p.Host, "https://us.posthog.com")
	host = strings.TrimRight(host, "/")

	account := accountFlag
	if account == "" {
		account = os.Getenv("PHOG_ACCOUNT")
	}

	current = Settings{
		APIKey:      apiKey,
		Host:        host,
		ProjectID:   projectID,
		Profile:     name,
		Account:     account,
		ConfigPath:  path,
		FromProfile: fromProfile && (apiKey == p.APIKey || projectID == p.ProjectID || host == strings.TrimRight(p.Host, "/")),
	}
	return nil
}

// Path returns the absolute path to ~/.phog/config.json (creating the parent
// dir on first call so writes don't fail later).
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".phog")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads ~/.phog/config.json. Returns an empty File (not an error) when
// the file does not exist yet.
func Load() (File, error) {
	path, err := Path()
	if err != nil {
		return File{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{Profiles: map[string]Profile{}}, nil
		}
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	return f, nil
}

// Save writes the config to disk with mode 0600.
func Save(f File) error {
	path, err := Path()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	// Write to a sibling temp file and rename so we never leave a half-written config.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SetProfile writes a profile (creating or updating). If active is true the
// profile becomes the active one. Pass empty fields to leave them unchanged
// on an existing profile.
func SetProfile(name string, p Profile, active bool) (File, error) {
	if name == "" {
		return File{}, errors.New("profile name must not be empty")
	}
	f, err := Load()
	if err != nil {
		return File{}, err
	}
	existing := f.Profiles[name]
	merged := mergeProfile(existing, p)
	f.Profiles[name] = merged
	if active || f.Active == "" {
		f.Active = name
	}
	return f, Save(f)
}

// DeleteProfile removes a profile. If it was active, active falls back to
// the first remaining profile (alphabetical) or "" if none remain.
func DeleteProfile(name string) (File, error) {
	f, err := Load()
	if err != nil {
		return File{}, err
	}
	if _, ok := f.Profiles[name]; !ok {
		return f, fmt.Errorf("profile %q not found", name)
	}
	delete(f.Profiles, name)
	if f.Active == name {
		f.Active = ""
		for n := range f.Profiles {
			if f.Active == "" || n < f.Active {
				f.Active = n
			}
		}
	}
	return f, Save(f)
}

// UseProfile sets the active profile pointer. Errors if the profile doesn't exist.
func UseProfile(name string) (File, error) {
	f, err := Load()
	if err != nil {
		return File{}, err
	}
	if _, ok := f.Profiles[name]; !ok {
		return f, fmt.Errorf("profile %q not found; create it with `phog auth add` first", name)
	}
	f.Active = name
	return f, Save(f)
}

// Redact returns a profile with the API key masked for display.
func Redact(p Profile) Profile {
	if p.APIKey == "" {
		return p
	}
	suffix := p.APIKey
	if len(suffix) > 4 {
		suffix = suffix[len(suffix)-4:]
	}
	p.APIKey = "phx_…" + suffix
	return p
}

func mergeProfile(into, from Profile) Profile {
	if from.APIKey != "" {
		into.APIKey = from.APIKey
	}
	if from.ProjectID != "" {
		into.ProjectID = from.ProjectID
	}
	if from.Host != "" {
		into.Host = from.Host
	}
	return into
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
