# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

<!--
  Add entries here while working on `develop` or `release/*`. On stable
  release, rename this section to `[X.Y.Z] - YYYY-MM-DD` and start a new
  empty `[Unreleased]` block.

  Use these subsections, omitting any that don't apply:
    ### Added       — new features
    ### Changed     — changes in existing functionality
    ### Deprecated  — soon-to-be removed features
    ### Removed     — removed features
    ### Fixed       — bug fixes
    ### Security    — vulnerability fixes
-->

## [0.0.1] - 2026-05-04

Initial release.

### Added

- CLI with three input modes: file path, `-` for stdin, and inline expression
  (single quoted arg or 5/6 separate schedule-field args).
- Plain-English explanation of every schedule (`0 4 * * 1-5` → "At 04:00,
  weekdays") plus the next three fire times.
- Crontab parser supporting:
  - 5-field user crontabs.
  - 6-field system crontabs (`/etc/crontab`, `/etc/cron.d/*`) with a user column.
  - `@reboot`, `@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly` descriptors.
  - Environment-variable assignments (`MAILTO=`, `PATH=`, `SHELL=`, …).
  - Graceful per-line parse-error recovery.
- Audit checks (each one its own function in `internal/audit/checks.go`):
  - `parse_error`, `missing_script`, `world_writable`, `not_executable`,
    `missing_timeout`, `missing_redirection`, `root_in_user_path`,
    `overlap_risk`, `no_flock`, `drift_smell`.
- Output modes: default text report, `--no-color`, `--json`, `--calendar`
  (7-day ASCII heatmap with collision call-outs).
- ANSI color when stdout is a TTY; auto-disabled otherwise; `--no-color`
  to force off.
- `--version` flag, with version/commit/date stamped at build time via
  `-ldflags`.
- Distribution: GitHub Releases (Linux/macOS/Windows, amd64/arm64),
  Homebrew tap, Scoop bucket, multi-arch GHCR Docker image
  (`ghcr.io/heytalepazguato/cron-doctor`), `go install`, POSIX `install.sh`.
- CI matrix (Go 1.22 + 1.23 on Linux, macOS, Windows).
- Branch-flow workflows: gate (`ci.yml`), `prerelease.yml` (dev artifacts +
  tagged release/* prereleases), `release.yml` (`main` → tag + GitHub
  Release + Homebrew + Scoop, with idempotence guard).

[Unreleased]: https://github.com/HeytalePazguato/cron-doctor/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/HeytalePazguato/cron-doctor/releases/tag/v0.0.1
