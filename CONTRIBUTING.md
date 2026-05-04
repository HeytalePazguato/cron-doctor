# Contributing to cron-doctor

Thanks for your interest! cron-doctor is a small, focused CLI and
contributions of any size are welcome — bug reports, fixes, new audit
checks, doc improvements.

## Quick start

```sh
git clone https://github.com/HeytalePazguato/cron-doctor.git
cd cron-doctor
git checkout develop
go build ./cmd/cron-doctor
go test ./...
```

Requires Go 1.22 or newer.

## Branch flow

```
develop  →  release/<version>  →  main
```

- All feature work, bug fixes, and refactors target `develop`.
- `main` is reserved for stable releases — never PR directly to it.
- Release stabilization happens on `release/<version>` branches cut from
  `develop`.

See the [README's "Branching & releases"](README.md#branching--releases)
section for the full versioning, tagging, and CI flow.

## Adding a new audit check

Audit checks are designed to be one-file additions:

1. Add a function in `internal/audit/checks.go` that takes a `*parser.Line`
   (and any other state it needs) and returns `[]Finding`.
2. Wire it into `Run` in `internal/audit/audit.go`.
3. Add a test in `internal/audit/checks_test.go`.
4. Add a fixture under `testdata/` if existing ones don't cover it.
5. Add a row to the **Checks** table in `README.md`.
6. Add an entry under `[Unreleased]` in `CHANGELOG.md`.

## Pull requests

- One logical change per PR; keep diffs reviewable.
- Run `go vet ./... && go test -race ./...` locally before opening the PR.
- Update `CHANGELOG.md` under `[Unreleased]`.
- Don't bump `VERSION` in feature PRs — that happens on the release branch.
- The `ci` workflow must pass on Linux, macOS, and Windows across Go 1.22
  and 1.23 before a PR can merge.

## Commit messages

Conventional-commits style is appreciated but not enforced:

```
feat: add no-flock heuristic for high-frequency jobs
fix: handle CRLF line endings in system crontabs
docs: clarify --calendar collision call-out format
```

The release-branch workflow looks for `[beta]` / `[rc]` keywords in commit
messages to choose pre-release stages — keep those out of normal commits.

## Reporting bugs

Use the [bug report template](https://github.com/HeytalePazguato/cron-doctor/issues/new?template=bug_report.yml).
A minimal reproducing crontab snippet is gold.

## Asking questions / proposing ideas

Use [Discussions](https://github.com/HeytalePazguato/cron-doctor/discussions)
for open-ended questions and ideas. Reserve issues for actionable bugs and
concrete feature requests.

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
