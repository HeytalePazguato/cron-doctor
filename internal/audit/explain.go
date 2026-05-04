package audit

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Explain turns a raw cron schedule string into an English description.
// Handles standard 5-field expressions and the @-descriptors. Falls back
// to the raw string if a field uses an unusual combination.
func Explain(schedule string) string {
	if strings.HasPrefix(schedule, "@") {
		return explainDescriptor(schedule)
	}
	parts := strings.Fields(schedule)
	if len(parts) != 5 {
		return schedule
	}
	minute := parseField(parts[0], 0, 59)
	hour := parseField(parts[1], 0, 23)
	dom := parseField(parts[2], 1, 31)
	month := parseField(parts[3], 1, 12)
	dow := parseDOW(parts[4])

	timePart := buildTimePhrase(minute, hour)
	datePart := buildDatePhrase(dom, month, dow)
	if datePart == "" {
		return capitalize(timePart)
	}
	return capitalize(timePart + ", " + datePart)
}

func explainDescriptor(d string) string {
	switch d {
	case "@reboot":
		return "At system boot."
	case "@hourly":
		return "Every hour, on the hour."
	case "@daily", "@midnight":
		return "Every day at 00:00."
	case "@weekly":
		return "Every Sunday at 00:00."
	case "@monthly":
		return "On the 1st of each month at 00:00."
	case "@yearly", "@annually":
		return "On January 1st at 00:00."
	}
	return d
}

type fieldKind int

const (
	fAny     fieldKind = iota // *
	fSingle                   // 5
	fRange                    // 5-10
	fStepAny                  // */N
	fStep                     // 5-10/2
	fList                     // 1,5,9
	fComplex                  // anything else
)

type fieldInfo struct {
	raw     string
	kind    fieldKind
	value   int   // fSingle
	start   int   // fRange, fStep
	end     int   // fRange, fStep
	step    int   // fStepAny, fStep
	list    []int // fList
	min     int
	max     int
	matches map[int]bool
}

func parseField(s string, min, max int) fieldInfo {
	fi := fieldInfo{raw: s, min: min, max: max, matches: map[int]bool{}}
	for v := min; v <= max; v++ {
		if matches(s, v, min, max) {
			fi.matches[v] = true
		}
	}
	switch {
	case s == "*":
		fi.kind = fAny
	case strings.HasPrefix(s, "*/"):
		n, err := strconv.Atoi(s[2:])
		if err == nil {
			fi.kind = fStepAny
			fi.step = n
		} else {
			fi.kind = fComplex
		}
	case strings.Contains(s, ","):
		var ints []int
		ok := true
		for _, p := range strings.Split(s, ",") {
			v, err := strconv.Atoi(p)
			if err != nil {
				ok = false
				break
			}
			ints = append(ints, v)
		}
		if ok {
			sort.Ints(ints)
			fi.kind = fList
			fi.list = ints
		} else {
			fi.kind = fComplex
		}
	case strings.Contains(s, "/") && strings.Contains(s, "-"):
		// e.g. 5-30/2
		slash := strings.Index(s, "/")
		rng := s[:slash]
		stp, err := strconv.Atoi(s[slash+1:])
		dash := strings.Index(rng, "-")
		if err == nil && dash > 0 {
			a, e1 := strconv.Atoi(rng[:dash])
			b, e2 := strconv.Atoi(rng[dash+1:])
			if e1 == nil && e2 == nil {
				fi.kind = fStep
				fi.start = a
				fi.end = b
				fi.step = stp
			} else {
				fi.kind = fComplex
			}
		} else {
			fi.kind = fComplex
		}
	case strings.Contains(s, "-"):
		dash := strings.Index(s, "-")
		a, e1 := strconv.Atoi(s[:dash])
		b, e2 := strconv.Atoi(s[dash+1:])
		if e1 == nil && e2 == nil {
			fi.kind = fRange
			fi.start = a
			fi.end = b
		} else {
			fi.kind = fComplex
		}
	default:
		v, err := strconv.Atoi(s)
		if err == nil {
			fi.kind = fSingle
			fi.value = v
		} else {
			fi.kind = fComplex
		}
	}
	return fi
}

// matches probes whether a single value would match a field expression.
// Used so the explainer can recognize patterns even when they're written
// in non-canonical forms (e.g. lowercase day names).
func matches(field string, v, min, max int) bool {
	field = strings.ToUpper(field)
	field = expandNames(field)
	for _, part := range strings.Split(field, ",") {
		if matchesAtom(part, v, min, max) {
			return true
		}
	}
	return false
}

func matchesAtom(s string, v, min, max int) bool {
	step := 1
	if i := strings.Index(s, "/"); i >= 0 {
		n, err := strconv.Atoi(s[i+1:])
		if err != nil || n <= 0 {
			return false
		}
		step = n
		s = s[:i]
	}
	var lo, hi int
	switch {
	case s == "*":
		lo, hi = min, max
	case strings.Contains(s, "-"):
		dash := strings.Index(s, "-")
		a, e1 := strconv.Atoi(s[:dash])
		b, e2 := strconv.Atoi(s[dash+1:])
		if e1 != nil || e2 != nil {
			return false
		}
		lo, hi = a, b
	default:
		n, err := strconv.Atoi(s)
		if err != nil {
			return false
		}
		lo, hi = n, n
	}
	if v < lo || v > hi {
		return false
	}
	return (v-lo)%step == 0
}

var dayNames = map[string]int{"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6}
var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

func expandNames(s string) string {
	for name, n := range dayNames {
		s = strings.ReplaceAll(s, name, strconv.Itoa(n))
	}
	for name, n := range monthNames {
		s = strings.ReplaceAll(s, name, strconv.Itoa(n))
	}
	return s
}

// parseDOW normalizes a day-of-week field. Sunday is both 0 and 7.
func parseDOW(s string) fieldInfo {
	upper := strings.ToUpper(expandNames(s))
	fi := parseField(upper, 0, 7)
	// Collapse 7 -> 0 in matches.
	if fi.matches[7] {
		fi.matches[0] = true
		delete(fi.matches, 7)
	}
	fi.max = 6
	return fi
}

func buildTimePhrase(minute, hour fieldInfo) string {
	mAny := minute.kind == fAny
	hAny := hour.kind == fAny

	switch {
	case minute.kind == fSingle && hour.kind == fSingle:
		return fmt.Sprintf("at %02d:%02d", hour.value, minute.value)
	case minute.kind == fStepAny && hAny:
		if minute.step == 1 {
			return "every minute"
		}
		return fmt.Sprintf("every %d minutes", minute.step)
	case mAny && hAny:
		return "every minute"
	case minute.kind == fSingle && hAny:
		if minute.value == 0 {
			return "every hour, on the hour"
		}
		return fmt.Sprintf("every hour at :%02d", minute.value)
	case minute.kind == fSingle && hour.kind == fStepAny:
		return fmt.Sprintf("every %d hours at :%02d", hour.step, minute.value)
	case minute.kind == fSingle && hour.kind == fRange:
		return fmt.Sprintf("from %02d:%02d to %02d:%02d, hourly", hour.start, minute.value, hour.end, minute.value)
	case minute.kind == fStepAny && hour.kind == fRange:
		return fmt.Sprintf("every %d minutes between %02d:00 and %02d:00", minute.step, hour.start, hour.end)
	case minute.kind == fStepAny && hour.kind == fSingle:
		return fmt.Sprintf("every %d minutes during hour %02d", minute.step, hour.value)
	case minute.kind == fSingle && hour.kind == fList:
		hours := joinTimes(hour.list, minute.value)
		return "at " + hours
	}
	// Fallback.
	return fmt.Sprintf("minute=%s hour=%s", minute.raw, hour.raw)
}

func joinTimes(hours []int, minute int) string {
	out := make([]string, len(hours))
	for i, h := range hours {
		out[i] = fmt.Sprintf("%02d:%02d", h, minute)
	}
	return joinAnd(out)
}

func buildDatePhrase(dom, month, dow fieldInfo) string {
	domAll := isAll(dom)
	monthAll := isAll(month)
	dowAll := isAll(dow)

	parts := []string{}
	if !dowAll {
		parts = append(parts, dowPhrase(dow))
	}
	if !domAll {
		parts = append(parts, domPhrase(dom))
	}
	if !monthAll {
		parts = append(parts, monthPhrase(month))
	}
	return strings.Join(parts, ", ")
}

func isAll(fi fieldInfo) bool {
	if fi.kind == fAny {
		return true
	}
	for v := fi.min; v <= fi.max; v++ {
		if !fi.matches[v] {
			return false
		}
	}
	return true
}

func dowPhrase(fi fieldInfo) string {
	weekdays := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}
	weekends := map[int]bool{0: true, 6: true}
	if equalSet(fi.matches, weekdays) {
		return "weekdays"
	}
	if equalSet(fi.matches, weekends) {
		return "weekends"
	}
	names := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	got := make([]string, 0)
	for d := 0; d <= 6; d++ {
		if fi.matches[d] {
			got = append(got, names[d])
		}
	}
	if len(got) == 1 {
		return pluralDay(got[0])
	}
	return "on " + joinAnd(got)
}

func pluralDay(s string) string {
	switch s {
	case "Sun":
		return "Sundays"
	case "Mon":
		return "Mondays"
	case "Tue":
		return "Tuesdays"
	case "Wed":
		return "Wednesdays"
	case "Thu":
		return "Thursdays"
	case "Fri":
		return "Fridays"
	case "Sat":
		return "Saturdays"
	}
	return s + "s"
}

func domPhrase(fi fieldInfo) string {
	switch fi.kind {
	case fSingle:
		return "on the " + ordinal(fi.value)
	case fStepAny:
		return fmt.Sprintf("every %d days", fi.step)
	case fRange:
		return fmt.Sprintf("from the %s to the %s", ordinal(fi.start), ordinal(fi.end))
	case fList:
		out := make([]string, len(fi.list))
		for i, v := range fi.list {
			out[i] = ordinal(v)
		}
		return "on the " + joinAnd(out)
	}
	return "on day " + fi.raw
}

func monthPhrase(fi fieldInfo) string {
	names := []string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	switch fi.kind {
	case fSingle:
		return "in " + names[fi.value]
	case fRange:
		return fmt.Sprintf("from %s to %s", names[fi.start], names[fi.end])
	case fList:
		out := make([]string, len(fi.list))
		for i, v := range fi.list {
			out[i] = names[v]
		}
		return "in " + joinAnd(out)
	case fStepAny:
		return fmt.Sprintf("every %d months", fi.step)
	}
	return "in month " + fi.raw
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(n) + suffix
}

func joinAnd(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func equalSet(a, b map[int]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}
