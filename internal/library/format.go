package library

import (
	"fmt"
	"time"
)

// FormatHours renders playtime as "12.5h", or "?" when playtime is unknown.
func FormatHours(d time.Duration, known bool) string {
	if !known {
		return "?"
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatDate renders a date as 2006-01-02, or "-" for the zero time.
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02")
}
