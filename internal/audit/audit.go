// Package audit runs lint-style checks on parsed crontab lines.
//
// Each check lives in its own function in checks.go and accepts a parsed
// line plus, when needed, surrounding context. Adding a new check means
// adding one function and one entry to perLineChecks (or wholeFileChecks
// for cross-line analyses).
package audit

import (
	"sort"
	"time"

	"github.com/HeytalePazguato/cron-doctor/internal/parser"
)

// Severity classifies the urgency of a finding.
type Severity string

const (
	SevError Severity = "error"
	SevWarn  Severity = "warn"
	SevInfo  Severity = "info"
)

// Finding is a single audit result attached to a line.
type Finding struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
}

// Config tunes audit behavior.
type Config struct {
	// OverlapWindow is the maximum time gap between two job fire times that
	// counts as a potential overlap. Defaults to 5 minutes.
	OverlapWindow time.Duration
	// MailTo is the value of the MAILTO env var declared in the crontab
	// (empty when unset). Used by the missing-redirection check.
	MailTo string
	// IsSystem indicates the file is a system-format crontab (with a user
	// column). Enables the run-as-root check.
	IsSystem bool
	// Now lets tests pin the clock; defaults to time.Now().
	Now func() time.Time
}

// Result is the per-line audit output. Length matches len(lines).
type Result [][]Finding

// Run executes every check against the parsed lines and returns findings
// keyed by their position in the input slice.
func Run(lines []*parser.Line, cfg Config) Result {
	if cfg.OverlapWindow == 0 {
		cfg.OverlapWindow = 5 * time.Minute
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	now := cfg.Now()
	out := make(Result, len(lines))

	for i, ln := range lines {
		out[i] = nil
		if ln.Type == parser.LineParseError {
			out[i] = append(out[i], Finding{SevError, "parse_error", ln.Error})
			continue
		}
		if ln.Type != parser.LineJob {
			continue
		}
		out[i] = append(out[i], checkMissingScript(ln)...)
		out[i] = append(out[i], checkWorldWritable(ln)...)
		out[i] = append(out[i], checkMissingTimeout(ln)...)
		out[i] = append(out[i], checkMissingRedirection(ln, cfg.MailTo)...)
		if cfg.IsSystem {
			out[i] = append(out[i], checkRunAsRoot(ln)...)
		}
		out[i] = append(out[i], checkNoFlock(ln, now)...)
	}

	mergeInto(out, checkOverlap(lines, cfg.OverlapWindow, now))
	mergeInto(out, checkDriftSmell(lines))

	for i := range out {
		sort.SliceStable(out[i], func(a, b int) bool {
			return sevRank(out[i][a].Severity) < sevRank(out[i][b].Severity)
		})
	}
	return out
}

func sevRank(s Severity) int {
	switch s {
	case SevError:
		return 0
	case SevWarn:
		return 1
	case SevInfo:
		return 2
	}
	return 3
}

func mergeInto(dst Result, extra Result) {
	for i := range extra {
		if i >= len(dst) {
			break
		}
		dst[i] = append(dst[i], extra[i]...)
	}
}

// Summary tallies findings across an entire result set.
type Summary struct {
	Lines    int `json:"lines"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

// Summarize counts findings by severity. parsedLines is the total
// number of non-blank, non-comment lines (jobs and parse errors).
func Summarize(lines []*parser.Line, r Result) Summary {
	s := Summary{}
	for _, ln := range lines {
		if ln.Type == parser.LineJob || ln.Type == parser.LineParseError {
			s.Lines++
		}
	}
	for _, fs := range r {
		for _, f := range fs {
			switch f.Severity {
			case SevError:
				s.Errors++
			case SevWarn:
				s.Warnings++
			case SevInfo:
				s.Info++
			}
		}
	}
	return s
}
