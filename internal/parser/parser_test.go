package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCleanFixture(t *testing.T) {
	r := openFixture(t, "clean.crontab")
	defer r.Close()
	pr, err := Parse(r, false)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := pr.EnvVars["MAILTO"]; got != "ops@example.com" {
		t.Errorf("MAILTO env: got %q", got)
	}
	jobs := jobLines(pr.Lines)
	if len(jobs) != 2 {
		t.Fatalf("want 2 jobs, got %d", len(jobs))
	}
	if jobs[0].Schedule != "0 */6 * * *" {
		t.Errorf("first schedule: %q", jobs[0].Schedule)
	}
	if jobs[1].Schedule != "@weekly" {
		t.Errorf("second schedule: %q", jobs[1].Schedule)
	}
}

func TestParseMalformedFixture(t *testing.T) {
	r := openFixture(t, "malformed.crontab")
	defer r.Close()
	pr, err := Parse(r, false)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	errs := 0
	jobs := 0
	for _, ln := range pr.Lines {
		switch ln.Type {
		case LineParseError:
			errs++
		case LineJob:
			jobs++
		}
	}
	if errs == 0 {
		t.Error("expected parse errors, got none")
	}
	if jobs == 0 {
		t.Error("expected at least one valid job to survive parse errors")
	}
}

func TestParseSystemFixture(t *testing.T) {
	r := openFixture(t, "etc.crontab")
	defer r.Close()
	pr, err := Parse(r, true)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	jobs := jobLines(pr.Lines)
	if len(jobs) != 2 {
		t.Fatalf("want 2 jobs, got %d", len(jobs))
	}
	for _, j := range jobs {
		if j.User != "root" {
			t.Errorf("expected user=root, got %q on line %d", j.User, j.Number)
		}
		if !strings.Contains(j.Command, "/") {
			t.Errorf("command lost path: %q", j.Command)
		}
	}
}

func TestParseInlineExpression(t *testing.T) {
	pr, err := Parse(strings.NewReader("0 4 * * 1-5\n"), false)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	jobs := jobLines(pr.Lines)
	if len(jobs) != 1 {
		t.Fatalf("want 1 job, got %d", len(jobs))
	}
	if jobs[0].Command != "" {
		t.Errorf("expected empty command, got %q", jobs[0].Command)
	}
	if jobs[0].CronSchedule == nil {
		t.Errorf("schedule did not parse")
	}
}

func TestParseEnvVar(t *testing.T) {
	pr, _ := Parse(strings.NewReader("PATH=/usr/local/bin:/usr/bin\n"), false)
	if pr.EnvVars["PATH"] != "/usr/local/bin:/usr/bin" {
		t.Errorf("env not captured: %v", pr.EnvVars)
	}
}

func TestParseReboot(t *testing.T) {
	pr, _ := Parse(strings.NewReader("@reboot /usr/local/bin/start.sh\n"), false)
	jobs := jobLines(pr.Lines)
	if len(jobs) != 1 || !jobs[0].IsReboot {
		t.Fatalf("expected one @reboot job, got %+v", jobs)
	}
}

func openFixture(t *testing.T, name string) *os.File {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture %s: %v", name, err)
	}
	return f
}

func jobLines(ls []*Line) []*Line {
	var out []*Line
	for _, l := range ls {
		if l.Type == LineJob {
			out = append(out, l)
		}
	}
	return out
}
