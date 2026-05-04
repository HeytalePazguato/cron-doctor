# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in cron-doctor, please report it
responsibly.

**Preferred channel:** open a private security advisory through GitHub:
[Report a vulnerability](https://github.com/HeytalePazguato/cron-doctor/security/advisories/new).

**Do NOT** open a public GitHub issue for security vulnerabilities.

You can expect an initial response within 7 days. Confirmed issues will be
fixed in the next release; the advisory will be published with credit (unless
you prefer to remain anonymous).

## Scope

cron-doctor is a read-only static-analysis CLI. It:

- Reads crontab files from disk or stdin and prints reports to stdout.
- Calls `os.Stat` on absolute-path commands referenced in the crontab to
  check existence, executable bit, and world-writable mode.
- Runs `cron.Schedule.Next` (pure computation) to compute fire times.
- Does **not** execute any commands listed in the crontab.
- Does **not** open network sockets, make HTTP requests, or load remote code.
- Does **not** write to any file outside the working directory unless the
  user explicitly redirects stdout.

The most plausible threat model is a maliciously crafted crontab causing the
parser to crash, hang, or consume excessive memory. Reports along those lines
are in scope.

## Supported Versions

Only the latest released minor version is supported with security updates.

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| older   | :x:                |
