package library

import (
	"fmt"
	"math"
	"time"
)

// FormatHours renders playtime as "12.5h", or "?" when playtime is unknown.
func FormatHours(d time.Duration, known bool) string {
	if !known {
		return "?"
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatAgo renders how long ago t was, the way the Steam client does:
// "today", "yesterday", "5 days ago", "3 weeks ago", "6 months ago",
// "2 years ago". The zero time renders as "-".
func FormatAgo(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	// Count calendar days, so 23:59 yesterday is "yesterday", not "today".
	y1, m1, d1 := t.In(now.Location()).Date()
	y2, m2, d2 := now.Date()
	days := int(time.Date(y2, m2, d2, 0, 0, 0, 0, time.UTC).Sub(time.Date(y1, m1, d1, 0, 0, 0, 0, time.UTC)).Hours() / 24)

	switch {
	case days <= 0:
		return "today"
	case days == 1:
		return "yesterday"
	case days < 14:
		return fmt.Sprintf("%d days ago", days)
	case days < 60:
		return fmt.Sprintf("%d weeks ago", int(math.Round(float64(days)/7)))
	case days < 365 && math.Round(float64(days)/30.44) < 12:
		return fmt.Sprintf("%d months ago", int(math.Round(float64(days)/30.44)))
	}
	if years := int(math.Round(float64(days) / 365.25)); years > 1 {
		return fmt.Sprintf("%d years ago", years)
	}
	return "1 year ago"
}

// FormatDate renders a date as 2006-01-02, or "-" for the zero time.
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02")
}

// FormatSize renders a byte count the way Steam does, in powers of 1024:
// "812 MB", "12.4 GB". Zero renders as "-".
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes <= 0 {
		return "-"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v, i := float64(bytes), 0
	for v >= unit && i < len(units)-1 {
		v /= unit
		i++
	}
	if v >= 100 || i == 0 {
		return fmt.Sprintf("%.0f %s", v, units[i])
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}
