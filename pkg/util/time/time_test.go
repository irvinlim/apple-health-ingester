package time

import (
	"testing"
	"time"
)

var (
	ts1        = mktime("2024-10-24T17:58:09Z")
	ts2        = mktime("2024-10-24T17:58:10Z")
	tsNegative = mktime("1969-10-24T17:58:10Z")
)

func TestMinTimeNonZero(t *testing.T) {
	tests := []struct {
		name string
		ts   []time.Time
		want time.Time
	}{
		{name: "no inputs"},
		{name: "zero timestamp", ts: []time.Time{{}}},
		{name: "non-zero timestamp", ts: []time.Time{ts1}, want: ts1},
		{name: "multiple timestamps", ts: []time.Time{ts1, ts2}, want: ts1},
		{name: "only negative timestamp", ts: []time.Time{tsNegative}, want: tsNegative},
		{name: "negative timestamp in list", ts: []time.Time{ts1, tsNegative}, want: tsNegative},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MinTimeNonZero(tt.ts...); !got.Equal(tt.want) {
				t.Errorf("MinTimeNonZero not equal, want %v got %v", tt.want, got)
			}
		})
	}
}

func TestMaxTime(t *testing.T) {
	tests := []struct {
		name string
		ts   []time.Time
		want time.Time
	}{
		{name: "no inputs"},
		{name: "zero timestamp", ts: []time.Time{{}}},
		{name: "non-zero timestamp", ts: []time.Time{ts1}, want: ts1},
		{name: "multiple timestamps", ts: []time.Time{ts1, ts2}, want: ts2},
		{name: "only negative timestamp", ts: []time.Time{tsNegative}, want: tsNegative},
		{name: "negative timestamp in list", ts: []time.Time{ts1, tsNegative}, want: ts1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxTime(tt.ts...); !got.Equal(tt.want) {
				t.Errorf("MinTimeNonZero not equal, want %v got %v", tt.want, got)
			}
		})
	}
}

func TestFormatTimeRange(t *testing.T) {
	tests := []struct {
		name   string
		start  time.Time
		end    time.Time
		layout string
		want   string
	}{
		{name: "basic test", start: ts1, end: ts2, layout: time.RFC3339, want: "2024-10-24T17:58:09Z - 2024-10-24T17:58:10Z"},
		{name: "custom time format", start: ts1, end: ts2, layout: time.DateTime, want: "2024-10-24 17:58:09 - 2024-10-24 17:58:10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatTimeRange(tt.start, tt.end, tt.layout); got != tt.want {
				t.Errorf("FormatTimeRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mktime(ts string) time.Time {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		panic(err)
	}
	return t
}
