---
layout: default
title: cron-doctor
description: Audit your crontab. Plain-English explanations, ten lint checks, calendar view, JSON output.
---

# cron-doctor

A CLI that audits crontabs and prints a human-readable report. Single static
Go binary, no daemon, no network.

[View on GitHub](https://github.com/HeytalePazguato/cron-doctor){: .btn }
[Download latest release](https://github.com/HeytalePazguato/cron-doctor/releases/latest){: .btn }

## Install

**Homebrew (macOS, Linux):**

```sh
brew install HeytalePazguato/tap/cron-doctor
```

**Scoop (Windows):**

```pwsh
scoop bucket add cron-doctor https://github.com/HeytalePazguato/scoop-bucket
scoop install cron-doctor
```

**Go:**

```sh
go install github.com/HeytalePazguato/cron-doctor/cmd/cron-doctor@latest
```

**Pre-built binary (Linux, macOS):**

```sh
curl -sSL https://raw.githubusercontent.com/HeytalePazguato/cron-doctor/main/install.sh | sh
```

## What it does

Run it on a crontab and get a per-line report:

```
Line 5: 0 */6 * * * /usr/local/bin/backup.sh >> /var/log/backup.log 2>&1
  Every 6 hours at :00
  Next: 2026-05-04 00:00, 2026-05-04 06:00, 2026-05-04 12:00.
  ✗ ERROR: Script /usr/local/bin/backup.sh does not exist.

Line 8: @weekly /usr/local/bin/rotate-logs.sh > /dev/null 2>&1
  Every Sunday at 00:00.
  Next: 2026-05-10 00:00, 2026-05-17 00:00, 2026-05-24 00:00.

2 lines parsed — 1 errors, 0 warnings, 0 info.
```

## Audit checks

| Code | Severity | Triggers when |
|---|---|---|
| `parse_error` | error | Schedule is invalid or required fields are missing. |
| `missing_script` | error | The command's first absolute-path token doesn't exist. |
| `world_writable` | error | The script file is mode `o+w`. |
| `not_executable` | warn | The script file exists but lacks the execute bit. |
| `missing_timeout` | warn | Command runs `curl`/`wget`/`nc`/`ssh` without a timeout flag. |
| `missing_redirection` | warn | No `>`/`>>`/`2>`/`&>` and `MAILTO` is unset. |
| `root_in_user_path` | warn | System crontab line runs as `root` from `/home/` or `/tmp/`. |
| `overlap_risk` | warn | Two jobs share a script and fire within 5 minutes of each other. |
| `no_flock` | warn | Job fires more often than every 10 minutes without `flock` or `pidof`. |
| `drift_smell` | info | Multiple lines share the exact `0 * * * *` schedule. |

## Output modes

- **Text** (default) — color when stdout is a TTY, plain otherwise.
- **`--no-color`** — guaranteed plain text for piping or CI.
- **`--json`** — machine-readable; per-line raw, schedule, explanation, next runs, findings.
- **`--calendar`** — 7-day ASCII heatmap of fire times, with collision call-outs.

See the [README](https://github.com/HeytalePazguato/cron-doctor#readme) for full
sample outputs and the [CHANGELOG](https://github.com/HeytalePazguato/cron-doctor/blob/main/CHANGELOG.md)
for release history.

## Links

- [Issues](https://github.com/HeytalePazguato/cron-doctor/issues)
- [Discussions](https://github.com/HeytalePazguato/cron-doctor/discussions)
- [Releases](https://github.com/HeytalePazguato/cron-doctor/releases)
- [Security policy](https://github.com/HeytalePazguato/cron-doctor/blob/main/SECURITY.md)
