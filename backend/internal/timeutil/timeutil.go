package timeutil

import (
	"fmt"
	"time"
)

const DefaultDisplayTimezone = "Asia/Jakarta"

var jakartaLocation *time.Location

func init() {
	loc, err := time.LoadLocation(DefaultDisplayTimezone)
	if err != nil {
		// Fallback to fixed zone UTC+7 if tzdata isn't present
		jakartaLocation = time.FixedZone("WIB", 7*3600)
	} else {
		jakartaLocation = loc
	}
}

// Location returns the display timezone location
func JakartaLocation() *time.Location {
	return jakartaLocation
}

// UTCNow returns current time in UTC
func UTCNow() time.Time {
	return time.Now().UTC()
}

// ParseISO parses RFC3339/ISO8601 string and converts strictly to UTC
func ParseISO(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		t, err = time.Parse("2006-01-02", s)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp %q: %w", s, err)
	}
	return t.UTC(), nil
}

// FormatISO returns RFC3339 UTC string
func FormatISO(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// FormatDateOnly returns YYYY-MM-DD
func FormatDateOnly(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// ToDisplayTime converts a UTC timestamp to Asia/Jakarta (WIB)
func ToDisplayTime(t time.Time) time.Time {
	return t.In(jakartaLocation)
}

// FormatDisplay returns user-facing string in Asia/Jakarta timezone
func FormatDisplay(t time.Time) string {
	inWib := ToDisplayTime(t)
	return inWib.Format("02 Jan 2006 15:04 WIB")
}
