// Package format contains presentation helpers shared by non-UI layers.
package format

import (
	"fmt"
	"time"
)

const day = 24 * time.Hour

// RelativeTime formats a timestamp using the brackets defined in DESIGN.md.
func RelativeTime(when time.Time, now time.Time) string {
	if when.IsZero() {
		return "never"
	}

	delta := now.Sub(when)
	if delta < time.Minute {
		return "just now"
	}
	if delta < time.Hour {
		minutes := int(delta / time.Minute)
		return plural(minutes, "minute")
	}
	if delta < day {
		hours := int(delta / time.Hour)
		return plural(hours, "hour")
	}
	if delta < 2*day {
		return "yesterday"
	}
	if delta < 7*day {
		days := int(delta / day)
		return plural(days, "day")
	}
	if delta < 28*day {
		weeks := int(delta / (7 * day))
		return plural(weeks, "week")
	}

	return "on " + when.Format("Mon Jan 2")
}

func plural(count int, unit string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s ago", unit)
	}
	return fmt.Sprintf("%d %ss ago", count, unit)
}
