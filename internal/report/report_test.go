package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/HeytalePazguato/cron-doctor/internal/audit"
	"github.com/HeytalePazguato/cron-doctor/internal/parser"
)

func parse(t *testing.T, src string, sys bool) []*parser.Line {
	t.Helper()
	pr, err := parser.Parse(strings.NewReader(src), sys)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return pr.Lines
}

func TestTextOutput(t *testing.T) {
	lines := parse(t, "0 4 * * 1-5 /usr/local/bin/nope.sh > /dev/null 2>&1\n", false)
	r := audit.Run(lines, audit.Config{
		MailTo: "ops@example.com",
		Now:    func() time.Time { return time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC) },
	})
	var buf bytes.Buffer
	Text(&buf, lines, r, false)
	out := buf.String()
	if !strings.Contains(out, "Line 1:") {
		t.Errorf("missing line prefix: %s", out)
	}
	if !strings.Contains(out, "weekdays") {
		t.Errorf("missing English explanation: %s", out)
	}
	if !strings.Contains(out, "Next:") {
		t.Errorf("missing next-runs: %s", out)
	}
}

func TestJSONOutput(t *testing.T) {
	lines := parse(t, "0 4 * * 1-5 /usr/local/bin/nope.sh > /dev/null 2>&1\n", false)
	r := audit.Run(lines, audit.Config{
		MailTo: "ops@example.com",
		Now:    func() time.Time { return time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC) },
	})
	var buf bytes.Buffer
	if err := JSON(&buf, lines, r); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var got struct {
		Lines []struct {
			LineNumber  int      `json:"line_number"`
			NextRuns    []string `json:"next_runs"`
			Explanation string   `json:"explanation"`
		} `json:"lines"`
		Summary struct{ Errors, Warnings, Info int } `json:"summary"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Lines) != 1 {
		t.Fatalf("want 1 line, got %d", len(got.Lines))
	}
	if got.Lines[0].LineNumber != 1 {
		t.Errorf("line number: %d", got.Lines[0].LineNumber)
	}
	if len(got.Lines[0].NextRuns) != 3 {
		t.Errorf("want 3 next runs, got %d", len(got.Lines[0].NextRuns))
	}
	if got.Lines[0].Explanation == "" {
		t.Errorf("explanation empty")
	}
}

func TestExpressionOutput(t *testing.T) {
	lines := parse(t, "0 4 * * 1-5\n", false)
	r := audit.Run(lines, audit.Config{Now: func() time.Time { return time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC) }})
	var buf bytes.Buffer
	Expression(&buf, lines, r, false)
	out := buf.String()
	if strings.Contains(out, "Line 1:") {
		t.Errorf("expression mode should not include Line N: prefix; got %s", out)
	}
	if !strings.Contains(out, "Schedule:") {
		t.Errorf("missing Schedule: prefix; got %s", out)
	}
	if !strings.Contains(out, "No command provided") {
		t.Errorf("missing schedule-only note; got %s", out)
	}
}
