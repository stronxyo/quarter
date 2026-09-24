package library

import (
	"testing"
	"time"
)

func TestFormatAgo(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.Local)
	daysAgo := func(d int) time.Time { return now.AddDate(0, 0, -d) }

	tests := []struct {
		t    time.Time
		want string
	}{
		{time.Time{}, "-"},
		{now.Add(-3 * time.Hour), "today"},
		{now.Add(time.Hour), "today"}, // clock skew
		{time.Date(2026, 9, 23, 23, 59, 0, 0, time.Local), "yesterday"},
		{daysAgo(11), "11 days ago"},
		{daysAgo(13), "13 days ago"},
		{daysAgo(14), "2 weeks ago"},
		{daysAgo(43), "6 weeks ago"},
		{daysAgo(65), "2 months ago"},
		{daysAgo(164), "5 months ago"},
		{daysAgo(173), "6 months ago"},
		{daysAgo(330), "11 months ago"},
		{daysAgo(364), "1 year ago"},
		{daysAgo(400), "1 year ago"},
		{daysAgo(800), "2 years ago"},
	}
	for _, tt := range tests {
		if got := FormatAgo(tt.t, now); got != tt.want {
			t.Errorf("FormatAgo(%s): got %q, want %q", tt.t.Format("2006-01-02 15:04"), got, tt.want)
		}
	}
}
