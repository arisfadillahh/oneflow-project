package httpapi

import (
	"testing"
	"time"
)

func TestBookingScheduleValidation(t *testing.T) {
	schedule := bookingServiceSchedule{DurationMinutes: 60, BufferMinutes: 15, Timezone: "Asia/Jakarta", Status: "active", Availability: defaultBookingAvailability()}
	now := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, start string
		allowed     bool
	}{
		{"open", "2026-09-07T09:00:00+07:00", true},
		{"past", "2026-09-04T10:00:00+07:00", false},
		{"before opening", "2026-09-07T08:30:00+07:00", false},
		{"end after closing", "2026-09-07T16:30:00+07:00", false},
		{"weekend", "2026-09-12T10:00:00+07:00", false},
		{"exact closing", "2026-09-07T16:00:00+07:00", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, _ := time.Parse(time.RFC3339, tc.start)
			err := validateBookingSchedule(schedule, start, start.Add(time.Hour), now)
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v, error=%v", tc.allowed, err)
			}
		})
	}
	schedule.Timezone = "Invalid/Zone"
	if validateBookingSchedule(schedule, now.Add(3*time.Hour), now.Add(4*time.Hour), now) == nil {
		t.Fatal("invalid timezone accepted")
	}
}
