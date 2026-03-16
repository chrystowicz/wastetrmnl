package timeutil

import (
	"fmt"
	"math"
	"strings"
	"time"
)

var WeekdayNames = map[time.Weekday]string{
	time.Monday:    "poniedziałek",
	time.Tuesday:   "wtorek",
	time.Wednesday: "środa",
	time.Thursday:  "czwartek",
	time.Friday:    "piątek",
	time.Saturday:  "sobota",
	time.Sunday:    "niedziela",
}

// ParseDate parses a date string in either "2006-01-02" or "02.01.2006" format,
// returning a time.Time anchored to midnight in loc.
func ParseDate(dateStr string, loc *time.Location) (time.Time, error) {
	normalized := strings.ReplaceAll(dateStr, ".", "-")
	for _, layout := range []string{"2006-01-02", "02-01-2006"} {
		if d, err := time.Parse(layout, normalized); err == nil {
			return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date format: %s", dateStr)
}

// RelativeDays returns a human-readable Polish string describing how far dateStr
// is from today ("dzisiaj", "jutro", "za N dni"), falling back to the raw string.
func RelativeDays(dateStr string, loc *time.Location) string {
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	date, err := ParseDate(dateStr, loc)
	if err != nil {
		return dateStr
	}

	days := int(math.Round(date.Sub(today).Hours() / 24))
	switch {
	case days == 0:
		return "dzisiaj"
	case days == 1:
		return "jutro"
	case days > 1:
		return fmt.Sprintf("za %d dni", days)
	default:
		return dateStr
	}
}

// TodayString returns a Polish sentence describing the current day and date.
func TodayString(loc *time.Location) string {
	now := time.Now().In(loc)
	return fmt.Sprintf("Dzisiaj jest %s, %s", WeekdayNames[now.Weekday()], now.Format("02.01.2006"))
}

// DurationUntilNextRun returns the duration until 00:05 the following day in loc.
func DurationUntilNextRun(loc *time.Location) time.Duration {
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, loc)
	return next.Sub(now)
}
