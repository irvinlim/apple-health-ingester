package healthautoexport_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
	"github.com/irvinlim/apple-health-ingester/pkg/util/testutils"
)

func TestTime_String(t *testing.T) {
	tests := []struct {
		name string
		time healthautoexport.Time
		want string
	}{
		{
			name: "RFC3339 timestamp",
			time: mktime("2021-11-26 02:35:10 +0800"),
			want: "2021-11-26T02:35:10+08:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fmt.Sprintf("%v", tt.time)
			if s != tt.want {
				t.Errorf("String() got = %v, want %v", s, tt.want)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      healthautoexport.Time
		wantError assert.ErrorAssertionFunc
	}{
		{
			name:  "24 hour time",
			input: "2021-11-26 02:35:10 +0800",
			want:  healthautoexport.NewTime(mustParseTimeRFC3339("2021-11-26T02:35:10+08:00")),
		},
		{
			name:  "12 hour time",
			input: "2024-09-21 7:57:00 am +1000",
			want:  healthautoexport.NewTime(mustParseTimeRFC3339("2024-09-21T07:57:00+10:00")),
		},
		{
			// See https://github.com/irvinlim/apple-health-ingester/issues/25
			name:  "12 hour time with ICU 72.1 (E280AF)",
			input: "2024-09-21 7:57:00\xe2\x80\xafam +1000",
			want:  healthautoexport.NewTime(mustParseTimeRFC3339("2024-09-21T07:57:00+10:00")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := healthautoexport.ParseTime(tt.input)
			if testutils.WantError(t, tt.wantError, err) {
				return
			}
			if !cmp.Equal(tt.want, parsed) {
				t.Errorf("want %v, got %v", tt.want, parsed)
			}
		})
	}
}

func TestTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      healthautoexport.Time
		wantError assert.ErrorAssertionFunc
	}{
		{
			name:      "cannot unmarshal empty body",
			input:     "",
			wantError: testutils.AssertErrorContains("unexpected end of JSON input"),
		},
		{
			name:      "not a string",
			input:     "123",
			wantError: testutils.AssertErrorContains("cannot unmarshal number"),
		},
		{
			name:      "cannot unmarshal empty string",
			input:     `""`,
			wantError: testutils.AssertErrorContains("failed to parse time"),
		},
		{
			name:      "cannot unmarshal invalid time format",
			input:     `"invalid"`,
			wantError: testutils.AssertErrorContains("failed to parse time"),
		},
		{
			name:  "unmarshal null to zero time",
			input: "null",
			want:  healthautoexport.Time{},
		},
		{
			name:  "unmarshal time",
			input: `"2021-11-26 02:35:10 +0800"`,
			want:  healthautoexport.NewTime(mustParseTimeRFC3339("2021-11-26T02:35:10+08:00")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output healthautoexport.Time
			if !testutils.WantError(t, tt.wantError, json.Unmarshal([]byte(tt.input), &output)) {
				assert.Equal(t, tt.want, output)
			}
		})
	}
}

func mustParseTimeRFC3339(s string) time.Time {
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return parsed
}
