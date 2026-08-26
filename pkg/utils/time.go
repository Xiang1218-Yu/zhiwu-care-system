package utils

import "time"

const dateLayout = "2006-01-02"

func ParseDate(value string) (time.Time, error) {
	return time.Parse(dateLayout, value)
}

func FormatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(dateLayout)
}

func Today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func DaysFromToday(value time.Time) int {
	return int(value.Sub(Today()).Hours() / 24)
}
