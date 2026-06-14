package timex

import "time"

// FormatTime formats a time.Time as RFC3339 in the given location.
func FormatTime(value time.Time, location *time.Location) string {
	if location == nil {
		return value.Format(time.RFC3339)
	}
	return value.In(location).Format(time.RFC3339)
}
