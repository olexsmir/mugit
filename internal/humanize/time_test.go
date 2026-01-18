package humanize

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "less than a minute"},
		{30 * time.Second, "less than a minute"},
		{1 * time.Minute, "1 minute"},
		{90 * time.Second, "1 minute"},
		{2 * time.Minute, "2 minutes"},
		{43 * time.Minute, "43 minutes"},
		{1 * time.Hour, "1 hour"},
		{90 * time.Minute, "1 hour"},
		{2 * time.Hour, "2 hours"},
		{23 * time.Hour, "23 hours"},
		{24 * time.Hour, "1 day"},
		{36 * time.Hour, "1 day"},
		{48 * time.Hour, "2 days"},
		{29 * 24 * time.Hour, "29 days"},
		{30 * 24 * time.Hour, "1 month"},
		{45 * 24 * time.Hour, "1 month"},
		{60 * 24 * time.Hour, "2 months"},
		{364 * 24 * time.Hour, "12 months"},
		{365 * 24 * time.Hour, "1 year"},
		{500 * 24 * time.Hour, "1 year"},
		{730 * 24 * time.Hour, "2 years"},
		{1000 * 24 * time.Hour, "2 years"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
