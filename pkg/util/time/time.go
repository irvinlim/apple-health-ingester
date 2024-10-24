package time

import (
	"fmt"
	"time"
)

// MinTimeNonZero returns the smallest time.Time from the input timestamps, excluding zero times.
func MinTimeNonZero(ts ...time.Time) time.Time {
	var minTime time.Time
	for _, t := range ts {
		if !t.IsZero() && (minTime.IsZero() || t.Before(minTime)) {
			minTime = t
		}
	}
	return minTime
}

// MaxTime returns the largest time.Time from the input timestamps.
func MaxTime(ts ...time.Time) time.Time {
	var maxTime time.Time
	for _, t := range ts {
		if maxTime.IsZero() || t.After(maxTime) {
			maxTime = t
		}
	}
	return maxTime
}

// FormatTimeRange outputs a time range as a formatted string.
func FormatTimeRange(start, end time.Time, layout string) string {
	return fmt.Sprintf("%v - %v", start.Format(layout), end.Format(layout))
}
