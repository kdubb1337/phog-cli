# phog

Agent-native CLI for [PostHog](https://posthog.com) — query events, web activity (`$pageview`), saved insights, persons, and run arbitrary HogQL against your project.

The binary is intentionally named `phog` (not `posthog`) to avoid colliding with PostHog's official [`posthog-cli`](https://github.com/PostHog/posthog-cli) (Rust) on PATH.

## Install

```
brew install kdubb1337/tap/phog
# or
go install github.com/kdubb1337/phog-cli/cmd/phog@latest
```

## Getting an API key

PostHog uses **Personal API keys** (scoped, per-user tokens) for the REST API and HogQL query endpoint. The project's `phc_*` write key used by the JS/SDK ingest path will NOT work here.

1. Sign in to PostHog (US cloud: <https://us.posthog.com>, EU cloud: <https://eu.posthog.com>, or your self-hosted host).
2. Open **Settings → Personal API keys**: <https://us.posthog.com/settings/user-api-keys> (EU: <https://eu.posthog.com/settings/user-api-keys>).
3. Click **Create personal API key**, give it a label (e.g. `phog-cli`), and grant the **read scopes** you need. For this CLI's default verbs:
   - `query:read` — required for `phog query` (HogQL) and event listing
   - `insight:read` — required for `phog insights`
   - `person:read` — required for `phog persons`
   - `project:read` — required for `phog doctor` project verification
4. Copy the token (starts with `phx_`). **PostHog shows it only once** — save it to your password manager.

Find your **project ID** in the URL after switching projects: `https://us.posthog.com/project/<PROJECT_ID>/...`. Or via `phog doctor` once `PHOG_API_KEY` is set.

## Quick start

Persist the token + project ID to a profile (recommended — no env vars needed for future calls):

```
phog auth add phx_yourtoken --project 12345
# or with an explicit host (EU cloud / self-hosted):
phog auth add phx_yourtoken --project 12345 --host https://eu.posthog.com
# interactive (paste the token at the prompt):
phog auth add --project 12345
# from a secret manager:
op read "op://Personal/PostHog/token" | phog auth add --project 12345

phog doctor                       # verifies creds + project reachability
phog events list --limit 10 --json
phog events list --pageviews --after 24h --json
phog query "SELECT event, count() FROM events WHERE timestamp > now() - INTERVAL 1 DAY GROUP BY event ORDER BY count() DESC LIMIT 20"
```

The config lives at `~/.phog/config.json` (mode 0600). Manage multiple profiles:

```
phog auth add phx_prodtoken --project 99999 --host https://eu.posthog.com --profile prod
phog profile use prod             # switch active
phog --profile prod doctor        # one-off override without switching active
phog profile list
phog auth remove prod --force
```

If you prefer env vars, they still work and override the active profile per-call:

```
export PHOG_API_KEY=phx_yourtoken
export PHOG_PROJECT_ID=12345
export PHOG_HOST=https://us.posthog.com
```

Precedence (highest wins): `--profile` flag → `PHOG_*` env vars → active profile → profile named `default` → zero values.

## Output rules

- stdout = data, stderr = humans
- Auto-JSON when piped; `--human` forces tables in a pipe
- `--compact` for high-gravity fields only; `--select` for explicit projection
- Null-valued keys stripped by default (use `--keep-nulls` to preserve them)
- Exit codes: `0` ok, `2` usage, `3` not-found, `4` auth, `5` api, `6` conflict, `7` rate-limit, `8` network, `9` validation, `124` timeout

See `phog agent-context` for the full schema.

## For agents

A bundled `SKILL.md` ships with the binary. Install it into your agent of choice:

```
phog skill install claude            # ~/.claude/skills/phog
phog skill install claude codex      # multiple
phog skill install --all             # every known agent
phog skill install --dir ~/custom    # custom path
phog skill install claude --mode=copy --force
```

Known targets: `claude` (`~/.claude/skills`), `codex` (`~/.codex/skills`), `gemini` (`~/.gemini/skills`), `openhands` (`~/.openhands/microagents`), `agents` (`~/.agents/skills`, the cross-agent universal path).

Default mode is `symlink` so edits to the source SKILL.md propagate instantly. Pass `--mode=copy` for a snapshot install. Check status with `phog skill list`; remove with `phog skill uninstall <agent>`.

To find the source path of the bundled skill:

```
phog skill path
```

Or read it directly at `skills/phog/SKILL.md` in this repo.

## Development

```
make tools     # install pinned dev tools
make           # build
make ci        # fmt + lint + test + build
```

See `AGENTS.md` for the full contributor guide.
