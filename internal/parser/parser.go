// Package parser reads crontab text into structured Lines.
//
// It accepts user crontabs (5 schedule fields), system crontabs
// (5 fields + user column, used in /etc/crontab and /etc/cron.d/*),
// the standard @reboot/@hourly/@daily/@weekly/@monthly/@yearly
// descriptors, environment-variable assignments, blank lines and
// comments. Malformed lines are reported with their line number and
// parsing continues.
package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/robfig/cron/v3"
)

// LineType discriminates between the kinds of crontab lines.
type LineType int

const (
	LineBlank LineType = iota
	LineComment
	LineEnv
	LineJob
	LineParseError
)

// Line is a single parsed crontab entry.
type Line struct {
	Number int
	Raw    string
	Type   LineType

	// Job fields.
	Schedule     string        // raw schedule expression (e.g. "0 4 * * 1-5" or "@daily")
	User         string        // populated for system crontabs only
	Command      string        // command to execute
	CronSchedule cron.Schedule // parsed schedule (nil for @reboot)
	IsReboot     bool

	// Env-var fields.
	EnvKey   string
	EnvValue string

	// Populated when Type == LineParseError.
	Error string
}

// ParseResult is the output of Parse.
type ParseResult struct {
	Lines    []*Line
	EnvVars  map[string]string
	IsSystem bool
}

var envRegexp = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$`)

// Parse reads a crontab from r. If isSystem is true, lines are expected to
// carry an extra user column between the schedule fields and the command.
func Parse(r io.Reader, isSystem bool) (*ParseResult, error) {
	result := &ParseResult{
		EnvVars:  make(map[string]string),
		IsSystem: isSystem,
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	num := 0
	for scanner.Scan() {
		num++
		raw := scanner.Text()
		line := parseLine(num, raw, isSystem)
		if line.Type == LineEnv {
			result.EnvVars[line.EnvKey] = line.EnvValue
		}
		result.Lines = append(result.Lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner: %w", err)
	}
	return result, nil
}

func parseLine(num int, raw string, isSystem bool) *Line {
	line := &Line{Number: num, Raw: raw}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		line.Type = LineBlank
		return line
	}
	if strings.HasPrefix(trimmed, "#") {
		line.Type = LineComment
		return line
	}
	if !strings.HasPrefix(trimmed, "@") && !startsWithCronField(trimmed) {
		if m := envRegexp.FindStringSubmatch(raw); m != nil {
			line.Type = LineEnv
			line.EnvKey = m[1]
			line.EnvValue = unquote(m[2])
			return line
		}
	}
	return parseJobLine(line, trimmed, isSystem)
}

// startsWithCronField returns true if the line clearly begins with a cron field
// (a digit, '*', or list/range/step punctuation). This keeps env-var detection
// from swallowing job lines whose command happens to contain '='.
func startsWithCronField(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	switch {
	case c >= '0' && c <= '9':
		return true
	case c == '*' || c == '?':
		return true
	}
	return false
}

func unquote(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' || first == '\'') && first == last {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func parseJobLine(line *Line, content string, isSystem bool) *Line {
	line.Type = LineJob
	if strings.HasPrefix(content, "@") {
		return parseDescriptorLine(line, content, isSystem)
	}
	fields := splitN(content, 5)
	if len(fields) < 5 {
		line.Type = LineParseError
		line.Error = "expected at least 5 schedule fields"
		return line
	}
	line.Schedule = strings.Join(fields[:5], " ")
	rest := ""
	if len(fields) > 5 {
		rest = fields[5]
	}
	if isSystem {
		userTok, command := splitFirstToken(rest)
		if userTok == "" {
			line.Type = LineParseError
			line.Error = "system crontab line missing user field"
			return line
		}
		line.User = userTok
		line.Command = command
	} else {
		line.Command = rest
	}
	sched, err := cron.ParseStandard(line.Schedule)
	if err != nil {
		line.Type = LineParseError
		line.Error = fmt.Sprintf("invalid schedule: %v", err)
		return line
	}
	line.CronSchedule = sched
	return line
}

func parseDescriptorLine(line *Line, content string, isSystem bool) *Line {
	desc, rest := splitFirstToken(content)
	line.Schedule = desc
	if isSystem {
		userTok, command := splitFirstToken(rest)
		if userTok == "" && desc != "@reboot" {
			line.Type = LineParseError
			line.Error = "system crontab line missing user field"
			return line
		}
		line.User = userTok
		line.Command = command
	} else {
		line.Command = rest
	}
	if desc == "@reboot" {
		line.IsReboot = true
		return line
	}
	sched, err := cron.ParseStandard(desc)
	if err != nil {
		line.Type = LineParseError
		line.Error = fmt.Sprintf("invalid descriptor: %v", err)
		return line
	}
	line.CronSchedule = sched
	return line
}

// splitN tokenizes s by whitespace into at most n leading tokens plus a final
// remainder string that preserves any internal whitespace (so commands keep
// their original spacing).
func splitN(s string, n int) []string {
	out := make([]string, 0, n+1)
	s = trimLeftSpace(s)
	for i := 0; i < n; i++ {
		if s == "" {
			return out
		}
		idx := indexSpace(s)
		if idx < 0 {
			out = append(out, s)
			return out
		}
		out = append(out, s[:idx])
		s = trimLeftSpace(s[idx:])
	}
	if s != "" {
		out = append(out, s)
	}
	return out
}

func splitFirstToken(s string) (first, rest string) {
	s = trimLeftSpace(s)
	idx := indexSpace(s)
	if idx < 0 {
		return s, ""
	}
	return s[:idx], trimLeftSpace(s[idx:])
}

func trimLeftSpace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[i:]
}

func indexSpace(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return i
		}
	}
	return -1
}
