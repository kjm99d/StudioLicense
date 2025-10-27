package utils

import (
	"fmt"
	"sync"
	"time"
)

const (
	dbDateTimeLayout = "2006-01-02 15:04:05"
	dateOnlyLayout   = "2006-01-02"
)

var (
	seoulOnce sync.Once
	seoulLoc  *time.Location
)

// SeoulLocation returns the cached Asia/Seoul location.
func SeoulLocation() *time.Location {
	seoulOnce.Do(func() {
		loc, err := time.LoadLocation("Asia/Seoul")
		if err != nil {
			// Fallback to a fixed zone if the location database is unavailable.
			loc = time.FixedZone("Asia/Seoul", 9*60*60)
		}
		seoulLoc = loc
	})
	return seoulLoc
}

// NowSeoul returns the current time in the Asia/Seoul timezone.
func NowSeoul() time.Time {
	return time.Now().In(SeoulLocation())
}

// FormatDateTimeForDB formats a time for DATETIME columns.
func FormatDateTimeForDB(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(SeoulLocation()).Format(dbDateTimeLayout)
}

// FormatDateOnly formats a time as YYYY-MM-DD.
func FormatDateOnly(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(SeoulLocation()).Format(dateOnlyLayout)
}

// ParseUserDate parses incoming user-supplied date/time strings.
func ParseUserDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	loc := SeoulLocation()
	layouts := []string{
		time.RFC3339,
		dbDateTimeLayout,
		dateOnlyLayout,
	}

	for _, layout := range layouts {
		if layout == time.RFC3339 {
			if ts, err := time.Parse(layout, value); err == nil {
				return ts.In(loc), nil
			}
			continue
		}

		if ts, err := time.ParseInLocation(layout, value, loc); err == nil {
			return ts.In(loc), nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", value)
}

// ParseDBDate parses date strings retrieved from the database.
func ParseDBDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	loc := SeoulLocation()
	if ts, err := time.ParseInLocation(dbDateTimeLayout, value, loc); err == nil {
		return ts, nil
	}

	if ts, err := time.ParseInLocation(dateOnlyLayout, value, loc); err == nil {
		return ts, nil
	}

	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts.In(loc), nil
	}

	return time.Time{}, fmt.Errorf("unsupported db time format: %s", value)
}

// StartOfDay returns the midnight timestamp for the provided time in Seoul.
func StartOfDay(t time.Time) time.Time {
	loc := SeoulLocation()
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}
