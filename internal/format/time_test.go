package format

import (
	"testing"
	"time"
)

func TestRelativeTimeFormatsDesignBrackets(t *testing.T) {
	now := time.Date(2026, 5, 26, 13, 45, 0, 0, time.UTC)

	tests := []struct {
		name string
		when time.Time
		want string
	}{
		{name: "absent", when: time.Time{}, want: "never"},
		{name: "future", when: now.Add(5 * time.Second), want: "just now"},
		{name: "under sixty seconds", when: now.Add(-59 * time.Second), want: "just now"},
		{name: "one minute", when: now.Add(-1 * time.Minute), want: "1 minute ago"},
		{name: "plural minutes", when: now.Add(-5 * time.Minute), want: "5 minutes ago"},
		{name: "fifty nine minutes", when: now.Add(-59 * time.Minute), want: "59 minutes ago"},
		{name: "one hour", when: now.Add(-1 * time.Hour), want: "1 hour ago"},
		{name: "plural hours", when: now.Add(-5 * time.Hour), want: "5 hours ago"},
		{name: "twenty three hours", when: now.Add(-23 * time.Hour), want: "23 hours ago"},
		{name: "yesterday at one day", when: now.Add(-24 * time.Hour), want: "yesterday"},
		{name: "yesterday before two days", when: now.Add(-47 * time.Hour), want: "yesterday"},
		{name: "two days", when: now.Add(-48 * time.Hour), want: "2 days ago"},
		{name: "six days", when: now.Add(-6 * 24 * time.Hour), want: "6 days ago"},
		{name: "one week", when: now.Add(-7 * 24 * time.Hour), want: "1 week ago"},
		{name: "plural weeks", when: now.Add(-21 * 24 * time.Hour), want: "3 weeks ago"},
		{name: "four weeks", when: now.Add(-28 * 24 * time.Hour), want: "on Tue Apr 28"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RelativeTime(tt.when, now); got != tt.want {
				t.Fatalf("RelativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}
