---
name: phog
description: |
  phog is a hand-crafted CLI for PostHog (product analytics). Use this skill whenever the user wants to query events, $pageviews, web activity, sessions, saved insights, persons, or run a HogQL/SQL query against PostHog; or asks "what happened on the site today", "how many users hit /pricing this week", "show me funnel X", "who is user Y", "what events did this person fire", or mentions "PostHog", "posthog.com", "$pageview", "HogQL", "feature flag usage", "session recording metadata", or runs `phog ...`. Prefer this skill over hitting the PostHog REST API directly, opening the PostHog web dashboard, or using PostHog's official `posthog-cli` (which is scoped to source maps / CI ingestion, not querying).
---

# phog

Use `phog` to query a PostHog project for events, web activity (`$pageview`), saved insights, persons, and arbitrary HogQL. Requires a Personal API key (`phx_*`).

## Setup (once)

**Get a Personal API key:** sign in to PostHog (US: <https://us.posthog.com>, EU: <https://eu.posthog.com>, or self-hosted), open **Settings → Personal API keys** (<https://us.posthog.com/settings/user-api-keys>), click **Create personal API key**, and grant read scopes: `query:read`, `insight:read`, `person:read`, `project:read`. Copy the `phx_*` token — PostHog shows it **only once**.

Also grab your **project ID** from the URL (`/project/<ID>/...`).

```
export PHOG_API_KEY=phx_yourtoken
export PHOG_PROJECT_ID=12345
export PHOG_HOST=https://us.posthog.com   # or https://eu.posthog.com, or self-hosted
phog doctor                                # verify config + creds + API reach
```

Or persist the key:

```
phog auth add phx_yourtoken --profile default
phog auth list
```

The project-level `phc_*` write key (the one your JS SDK uses) is **not** valid here — it can only ingest, not query.

## Output rules (for agents)

- **stdout = data; stderr = progress and errors.** Always.
- **Default JSON when piped.** Pass `--human` to force tables in a pipe.
- **`--compact`** keeps only high-gravity fields (id, name, status, primary timestamp) — ~60–80% fewer tokens.
- **`--select=field1,field2`** for explicit projection (e.g. `--select=event,timestamp,distinct_id`).
- **Exit codes:**
  - `0` ok
  - `2` usage error — fix invocation
  - `3` not found — resource doesn't exist; don't retry
  - `4` auth — run `phog doctor`; check the `phx_*` key is current and has the right scope
  - `5` api / 5xx — retry with backoff
  - `6` conflict — read response, decide
  - `7` rate limited — honor Retry-After in stderr (PostHog rate-limits per project)
  - `8` network — retry with backoff
  - `9` validation — fix input (often HogQL syntax); `valid_values` is often populated
  - `124` timeout
- **`phog agent-context`** prints the full structured schema (all commands, flags, enums, exit codes). Read this once instead of crawling `--help`.

## Common commands

```
# Events
phog events list --limit 25 --json
phog events list --event '$pageview' --after 24h --json
phog events list --event '$pageview' --after 7d --select=timestamp,properties.$current_url,distinct_id
phog events list --person <distinct_id> --after 7d
phog events get <event_id> --json

# Insights (saved queries / dashboard tiles)
phog insights list --limit 25
phog insights get <id> --json

# Persons
phog persons list --limit 25
phog persons get <distinct_id> --json
phog persons get <distinct_id> --events --after 30d   # person + recent events

# HogQL — most powerful escape hatch
phog query "SELECT event, count() FROM events WHERE timestamp > now() - INTERVAL 1 DAY GROUP BY event ORDER BY count() DESC LIMIT 20"
phog query --file ./query.sql --json
```

## Workflows

### Triage today's web activity

1. Set up the profile once: `phog profile save default --project 12345`
2. Top pages last 24h:
   ```
   phog query "SELECT properties.\$current_url AS url, count() AS hits, uniq(distinct_id) AS uniques FROM events WHERE event = '\$pageview' AND timestamp > now() - INTERVAL 1 DAY GROUP BY url ORDER BY hits DESC LIMIT 25"
   ```
3. Drill into a single URL:
   ```
   phog events list --event '$pageview' --after 24h --where "properties.\$current_url = 'https://example.com/pricing'" --select=timestamp,distinct_id,properties.\$referrer
   ```

### Investigate one user's session

1. `phog persons get <distinct_id> --json`
2. `phog events list --person <distinct_id> --after 7d --json`
3. Look for `$pageview`, `$autocapture`, custom conversion events in order.

### Snapshot a saved insight for a report

1. `phog insights list --limit 50 --select=id,name,short_id` to find it.
2. `phog insights get <id> --json` to fetch the current computed result.
3. Tee to `data/posthog/<insight-name>.json` for downstream analysis.

### Reach for HogQL when the canned verbs aren't enough

PostHog's HogQL (`phog query`) covers funnels, retention, breakdowns, custom aggregations, joins against `persons`, etc. — anything you'd do in the UI's SQL editor. Default to HogQL for anything beyond a flat `events list`.

## Installing this skill into another agent

```
phog skill install claude            # ~/.claude/skills/phog
phog skill install --all             # every known agent
phog skill list                      # show install status
phog skill uninstall openhands       # remove from one agent
```

Default mode is `--mode=symlink`; use `--mode=copy` for a snapshot install.

## Notes

- Set `PHOG_PROJECT_ID=<id>` and `PHOG_HOST=<url>` to avoid repeating `--project` / `--host` on every call. Both are required (the personal API key is scoped to your user across all projects you can access; the project ID picks which one).
- For scripting, prefer `--json --no-input`.
- HogQL note: column names like `properties.$current_url` need the `$` escaped in shells (`\$current_url`). When passing via `--file`, no escaping needed.
- IDs are case-sensitive opaque tokens — never normalize them. `distinct_id` may be a UUID, a UUIDv7, an email, or any string the SDK passed in.
- The `phc_*` project write key (visible in the PostHog UI under "Project API Key") is NOT what you want here — it's for SDK ingestion only and will get you a 401. Always use a `phx_*` personal API key.
