package audit

import "testing"

func TestExplain(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"0 4 * * 1-5", "At 04:00, weekdays"},
		{"*/15 9-17 * * 1-5", "Every 15 minutes between 09:00 and 17:00, weekdays"},
		{"0 * * * *", "Every hour, on the hour"},
		{"@daily", "Every day at 00:00."},
		{"@reboot", "At system boot."},
		{"0 0 1 * *", "At 00:00, on the 1st"},
		{"30 2 * * 0,6", "At 02:30, weekends"},
	}
	for _, tc := range cases {
		got := Explain(tc.in)
		if got != tc.want {
			t.Errorf("Explain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
