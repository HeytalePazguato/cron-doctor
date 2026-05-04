# cron-doctor

[![CI](https://img.shields.io/github/actions/workflow/status/HeytalePazguato/cron-doctor/ci.yml?branch=main&label=ci)](https://github.com/HeytalePazguato/cron-doctor/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/HeytalePazguato/cron-doctor)](go.mod)
[![Go Reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/HeytalePazguato/cron-doctor)
[![Latest release](https://img.shields.io/github/v/release/HeytalePazguato/cron-doctor?sort=semver)](https://github.com/HeytalePazguato/cron-doctor/releases)
[![License](https://img.shields.io/github/license/HeytalePazguato/cron-doctor)](LICENSE)

A CLI that audits crontabs and prints a human-readable report. Single static
Go binary, no daemon, no network.

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

**Docker (GHCR, multi-arch):**

```sh
docker run --rm -v /etc/crontab:/etc/crontab:ro \
  ghcr.io/heytalepazguato/cron-doctor /etc/crontab
```

**Manual:** download the archive for your OS/arch from
[Releases](https://github.com/HeytalePazguato/cron-doctor/releases) — Linux,
macOS, and Windows on `amd64` and `arm64`.

**Build from source:**

```sh
git clone https://github.com/HeytalePazguato/cron-doctor.git
cd cron-doctor
go build -o cron-doctor ./cmd/cron-doctor
```

> **Why not npm / pip / cargo?** All three can technically ship Go binaries
> via wrapper packages (esbuild does this on npm), but cron-doctor's audience
> is sysadmins managing crontabs, not Node/Python/Rust developers. Adding a
> language runtime as a transitive dependency for a 5 MB static binary is
> overhead with no benefit. The right channels for a sysadmin CLI are the
> OS-native package managers above.

## Usage

Audit a crontab file:

```sh
cron-doctor /etc/crontab
```

Read from stdin:

```sh
crontab -l | cron-doctor -
```

Audit a single expression (no file required):

```sh
cron-doctor "0 4 * * 1-5"
cron-doctor "0 4 * * 1-5 /usr/local/bin/run.sh"
cron-doctor 0 4 '*' '*' 1-5            # 5 fields as separate args
```

Other modes:

```sh
cron-doctor --calendar /etc/crontab    # 7-day ASCII heatmap
cron-doctor --json /etc/crontab        # machine-readable output
cron-doctor --no-color /etc/crontab    # disable ANSI colors
```

### Sample output

#### Default text report

Run on `testdata/clean.crontab`. ANSI colors are added when stdout is a TTY;
`--no-color` produces the same body without escape codes.

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

#### `--no-color`

Run on `testdata/missing_scripts.crontab`. Identical to the default report,
just guaranteed plain text — useful when piping into `less`, redirecting to a
file, or running in CI.

```
Line 2: 0 5 * * * /opt/this-does-not-exist/run.sh > /dev/null 2>&1
  At 05:00
  Next: 2026-05-04 05:00, 2026-05-05 05:00, 2026-05-06 05:00.
  ✗ ERROR: Script /opt/this-does-not-exist/run.sh does not exist.

Line 3: * * * * * curl https://example.com/hook > /dev/null 2>&1
  Every minute
  Next: 2026-05-03 23:23, 2026-05-03 23:24, 2026-05-03 23:25.
  ⚠ WARN: Command runs curl without a timeout flag; a hung connection will hang the job.
  ⚠ WARN: Job fires more often than every 10 minutes without flock or pidof; concurrent runs may pile up.

2 lines parsed — 1 errors, 2 warnings, 0 info.
```

#### `--json`

Same fixture as above. Each line gets its raw text, parsed schedule, English
explanation, the next three fire times in RFC3339, and an array of findings.
A roll-up summary follows at the end.

```json
{
  "lines": [
    {
      "line_number": 2,
      "raw": "0 5 * * * /opt/this-does-not-exist/run.sh > /dev/null 2>&1",
      "schedule": "0 5 * * *",
      "command": "/opt/this-does-not-exist/run.sh > /dev/null 2>&1",
      "explanation": "At 05:00",
      "next_runs": [
        "2026-05-04T10:00:00Z",
        "2026-05-05T10:00:00Z",
        "2026-05-06T10:00:00Z"
      ],
      "findings": [
        {
          "severity": "error",
          "code": "missing_script",
          "message": "Script /opt/this-does-not-exist/run.sh does not exist."
        }
      ]
    },
    {
      "line_number": 3,
      "raw": "* * * * * curl https://example.com/hook > /dev/null 2>&1",
      "schedule": "* * * * *",
      "command": "curl https://example.com/hook > /dev/null 2>&1",
      "explanation": "Every minute",
      "next_runs": [
        "2026-05-04T04:22:00Z",
        "2026-05-04T04:23:00Z",
        "2026-05-04T04:24:00Z"
      ],
      "findings": [
        {
          "severity": "warn",
          "code": "missing_timeout",
          "message": "Command runs curl without a timeout flag; a hung connection will hang the job."
        },
        {
          "severity": "warn",
          "code": "no_flock",
          "message": "Job fires more often than every 10 minutes without flock or pidof; concurrent runs may pile up."
        }
      ]
    }
  ],
  "summary": {
    "lines": 2,
    "errors": 1,
    "warnings": 2,
    "info": 0
  }
}
```

#### `--calendar`

Run on `testdata/clean.crontab`. Hours along the top, days down the side; each
cell shows how many jobs fire in that hour (`.` = none). Hour-cells with two
or more fires are listed below the grid for collision investigation.

```
7-day calendar starting 2026-05-03

        0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19 20 21 22 23
Sun 03  .  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .
Mon 04  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .
Tue 05  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .
Wed 06  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .
Thu 07  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .
Fri 08  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .
Sat 09  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .  1  .  .  .  .  .
```

## Checks

| Code                  | Severity | Triggers when                                                                                    |
|-----------------------|----------|--------------------------------------------------------------------------------------------------|
| `parse_error`         | error    | Schedule is invalid or the line is missing required fields.                                      |
| `missing_script`      | error    | The command's first absolute-path token doesn't exist.                                           |
| `world_writable`      | error    | The script file is mode `o+w`.                                                                   |
| `not_executable`      | warn     | The script file exists but lacks the execute bit.                                                |
| `missing_timeout`     | warn     | Command runs `curl`/`wget`/`nc`/`ssh` without a timeout flag.                                    |
| `missing_redirection` | warn     | No `>`/`>>`/`2>`/`&>` and `MAILTO` is unset (cron will email the owner).                         |
| `root_in_user_path`   | warn     | System crontab line runs as `root` from `/home/` or `/tmp/`.                                     |
| `overlap_risk`        | warn     | Two jobs share a script and their next fire times fall within the overlap window (default 5 min).|
| `no_flock`            | warn     | Job fires more often than every 10 minutes without `flock` or `pidof`.                           |
| `drift_smell`         | info     | Multiple lines share the exact `0 * * * *` schedule (cargo-culted on-the-hour fires).            |

### Example finding

```
Line 3: */5 * * * * curl https://example.com/hook
  Every 5 minutes
  Next: 2026-05-03 12:05, 2026-05-03 12:10, 2026-05-03 12:15.
  ⚠ WARN: Command runs curl without a timeout flag; a hung connection will hang the job.
  ⚠ WARN: Job fires more often than every 10 minutes without flock or pidof; concurrent runs may pile up.
  ⚠ WARN: No output redirection and MAILTO is unset; cron will email the job owner.
```

## Project layout

```
cmd/cron-doctor/      CLI entry point and mode detection.
internal/parser/      Crontab tokenizer.
internal/audit/       English explainer + each lint check.
internal/report/      Text, JSON, and calendar renderers.
testdata/             Fixtures used by unit tests and demos.
```

Each audit check is its own function in `internal/audit/checks.go`. Adding a
new check means adding one function and wiring it into `Run` in
`internal/audit/audit.go`.

## Out of scope (v0.1)

- Editing crontabs (read-only tool).
- LLM features or any network calls.
- Web UI, daemon, watch mode.
- Schedule generation or assistance.
- Localization (English only).

## Branching & releases

```
develop  →  release/<version>  →  main
```

Never PR directly to `main`. `main` is reserved for stable releases only.

- **`develop`** — active development; daily integration target.
- **`release/<version>`** (e.g. `release/0.0.2`) — pre-release stabilization
  branch cut from `develop`.
- **`main`** — stable releases only; PRs come from `release/*`.

### Versioning source of truth

- Stable version lives in [`VERSION`](VERSION) and is bumped manually on the
  release branch before merging to `main`.
- Pre-release version is derived from the branch name
  (`release/0.0.2` → base `0.0.2`).
- Dev builds are stamped automatically as `0.0.<run_number>-dev`.
- The version is embedded into the binary at build time via
  `-ldflags="-X main.version=..."` and printed by `cron-doctor --version`.

### Changelog

User-visible changes are tracked in [`CHANGELOG.md`](CHANGELOG.md), following
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Add entries under
`[Unreleased]` while working on `develop` or `release/*`; on stable release,
rename that section to `[X.Y.Z] - YYYY-MM-DD` and start a fresh `[Unreleased]`.

### CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml))

Runs on push to `main`, `develop`, `release/**` and on PRs to `main`/`develop`.

| Trigger          | Job                                                  | Output                                                                                                                |
| ---------------- | ---------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| Any branch above | `lint-build-test` (Go 1.22 + 1.23, Linux/macOS/Win)  | Gate for everything else.                                                                                             |
| Push `develop`   | `dev-build` ([`prerelease.yml`](.github/workflows/prerelease.yml)) | Snapshot binaries uploaded as artifact `cron-doctor-0.0.<run>-dev` (90d retention, latest 3 kept).         |
| Push `release/*` | `prerelease`  ([`prerelease.yml`](.github/workflows/prerelease.yml)) | Auto-tagged GitHub pre-release + binaries (latest 5 kept). Stage from commit msg: default `alpha`, `[beta]`, `[rc]`. |
| Push `main`      | `release` ([`release.yml`](.github/workflows/release.yml)) | Git tag `v<VERSION>` + GitHub Release + Homebrew tap update + Scoop bucket update.                                   |

### Pre-release stage selection

Stage is chosen by commit-message keyword on the release branch, with an
auto-incrementing counter per stage:

- default → alpha (e.g. `v0.0.2-alpha.1`, `v0.0.2-alpha.2`)
- `[beta]` in commit msg → beta (`v0.0.2-beta.1`, …)
- `[rc]` in commit msg → rc (`v0.0.2-rc.1`, …)

### Pull requests

- PRs target `develop` (feature work) or `main` (release merge from `release/*`).
- CI gates merges by running vet + build + test on Go 1.22 and 1.23 across
  Linux, macOS, and Windows.
- A concurrency group cancels superseded runs on the same ref.

### Idempotence guard

The `release` job checks for an existing `v<VERSION>` tag and skips
tagging/publishing if it already exists — safe to re-run after a flake or
when `main` receives a non-version-bump commit.

### Required secrets

Repository → Settings → Secrets and variables → Actions:

| Secret                       | Used by                  | Why                                                                       |
| ---------------------------- | ------------------------ | ------------------------------------------------------------------------- |
| `GITHUB_TOKEN`               | all release jobs         | Auto-provided by Actions; tags & GitHub Release.                          |
| `HOMEBREW_TAP_GITHUB_TOKEN`  | release / prerelease     | PAT with `repo` scope on `HeytalePazguato/homebrew-tap`. Comment out the `brews:` block in `.goreleaser.yml` if you don't have a tap yet. |
| `SCOOP_GITHUB_TOKEN`         | release / prerelease     | PAT with `repo` scope on `HeytalePazguato/scoop-bucket`. Comment out the `scoops:` block in `.goreleaser.yml` if you don't have a bucket yet. |

## License

MIT — see [LICENSE](LICENSE).
