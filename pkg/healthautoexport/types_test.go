package healthautoexport_test

import (
	"fmt"
	"testing"

	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

func TestTime_String(t *testing.T) {
	tests := []struct {
		name string
		time *healthautoexport.Time
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
