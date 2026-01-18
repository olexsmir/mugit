package humanize

import (
	"fmt"
	"time"
)

// Time returns a human-readable relative time string (e.g., "3 hours ago").
func Time(t time.Time) string {
	return formatDuration(time.Since(t)) + " ago"
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "less than a minute"
	case d < 2*time.Minute:
		return "1 minute"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes", d/time.Minute)
	case d < 2*time.Hour:
		return "1 hour"
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours", d/time.Hour)
	case d < 48*time.Hour:
		return "1 day"
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%d days", d/(24*time.Hour))
	case d < 60*24*time.Hour:
		return "1 month"
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%d months", d/(30*24*time.Hour))
	case d < 2*365*24*time.Hour:
		return "1 year"
	default:
		return fmt.Sprintf("%d years", d/(365*24*time.Hour))
	}
}
