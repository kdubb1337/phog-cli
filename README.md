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

```
export PHOG_API_KEY=phx_yourtoken
export PHOG_PROJECT_ID=12345
export PHOG_HOST=https://us.posthog.com   # or https://eu.posthog.com, or self-hosted

phog doctor
phog events list --limit 10 --json
phog events list --event '$pageview' --after 24h --json
phog query "SELECT event, count() FROM events WHERE timestamp > now() - INTERVAL 1 DAY GROUP BY event ORDER BY count() DESC LIMIT 20"
```

Or persist the key as a profile:

```
phog auth add phx_yourtoken --profile default
phog profile list
```

## Output rules

- stdout = data, stderr = humans
- Auto-JSON when piped; `--human` forces tables in a pipe
- `--compact` for high-gravity fields only; `--select` for explicit projection
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
