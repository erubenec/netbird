package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func at(t *testing.T, zone, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation(zone)
	require.NoError(t, err)
	ts, err := time.ParseInLocation("2006-01-02 15:04", value, loc)
	require.NoError(t, err)
	return ts
}

func workdays() []string { return []string{"mon", "tue", "wed", "thu", "fri"} }

func TestSchedule_Nil_AlwaysActive(t *testing.T) {
	var s *Schedule
	assert.True(t, s.Active(time.Now()))
	assert.NoError(t, s.Validate())
	assert.Nil(t, s.Copy())
}

func TestSchedule_Active_Windows(t *testing.T) {
	s := &Schedule{
		Timezone: "Europe/Madrid",
		Windows:  []Window{{Days: workdays(), Start: "08:00", End: "17:00"}},
	}
	require.NoError(t, s.Validate())

	tests := []struct {
		name string
		when string
		want bool
	}{
		{"monday inside", "2026-11-23 09:30", true},
		{"start is inclusive", "2026-11-23 08:00", true},
		{"end is exclusive", "2026-11-23 17:00", false},
		{"last minute", "2026-11-23 16:59", true},
		{"before opening", "2026-11-23 07:59", false},
		{"saturday inside hours", "2026-11-21 10:00", false},
		{"sunday inside hours", "2026-11-22 10:00", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, s.Active(at(t, "Europe/Madrid", tc.when)))
		})
	}
}

// The window is defined in the schedule timezone, so the same instant must
// give the same answer regardless of the zone the clock is expressed in.
func TestSchedule_Active_UsesScheduleTimezone(t *testing.T) {
	s := &Schedule{
		Timezone: "Europe/Madrid",
		Windows:  []Window{{Days: workdays(), Start: "08:00", End: "17:00"}},
	}
	// 09:00 in Madrid (UTC+1 in November) is 08:00 UTC and 03:00 in New York.
	instant := at(t, "Europe/Madrid", "2026-11-23 09:00")
	assert.True(t, s.Active(instant.In(time.UTC)))
	assert.True(t, s.Active(instant.In(time.FixedZone("x", -5*3600))))
	// 17:30 in Madrid is still 16:30 UTC, which would look like office hours.
	late := at(t, "Europe/Madrid", "2026-11-23 17:30")
	assert.False(t, s.Active(late.In(time.UTC)))
}

func TestSchedule_Active_DaylightSaving(t *testing.T) {
	s := &Schedule{
		Timezone: "Europe/Madrid",
		Windows:  []Window{{Days: []string{"mon"}, Start: "08:00", End: "10:00"}},
	}
	// Winter (UTC+1): 08:30 local is 07:30 UTC.
	assert.True(t, s.Active(time.Date(2026, 11, 23, 7, 30, 0, 0, time.UTC)))
	// Summer (UTC+2): 08:30 local is 06:30 UTC, and 07:30 UTC is already 09:30.
	assert.True(t, s.Active(time.Date(2026, 7, 6, 6, 30, 0, 0, time.UTC)))
	assert.True(t, s.Active(time.Date(2026, 7, 6, 7, 30, 0, 0, time.UTC)))
	assert.False(t, s.Active(time.Date(2026, 7, 6, 8, 30, 0, 0, time.UTC)))
}

func TestSchedule_Active_MultipleWindows(t *testing.T) {
	s := &Schedule{
		Timezone: "UTC",
		Windows: []Window{
			{Days: []string{"mon"}, Start: "06:00", End: "08:00"},
			{Days: []string{"mon"}, Start: "20:00", End: "24:00"},
		},
	}
	require.NoError(t, s.Validate())
	assert.True(t, s.Active(at(t, "UTC", "2026-11-23 07:00")))
	assert.False(t, s.Active(at(t, "UTC", "2026-11-23 12:00")))
	assert.True(t, s.Active(at(t, "UTC", "2026-11-23 23:59")))
	assert.False(t, s.Active(at(t, "UTC", "2026-11-24 00:00")), "24:00 ends at midnight")
}

func TestSchedule_Active_DateRange(t *testing.T) {
	s := &Schedule{Timezone: "UTC", StartDate: "2026-12-01", EndDate: "2026-12-31"}
	require.NoError(t, s.Validate())
	assert.False(t, s.Active(at(t, "UTC", "2026-11-30 23:59")))
	assert.True(t, s.Active(at(t, "UTC", "2026-12-01 00:00")), "no windows means all day")
	assert.True(t, s.Active(at(t, "UTC", "2026-12-31 23:59")), "end date is inclusive")
	assert.False(t, s.Active(at(t, "UTC", "2027-01-01 00:00")))
}

func TestSchedule_Active_DateRangeInScheduleTimezone(t *testing.T) {
	s := &Schedule{Timezone: "Asia/Tokyo", StartDate: "2026-12-01"}
	// 2026-11-30 16:00 UTC is already 2026-12-01 01:00 in Tokyo.
	assert.True(t, s.Active(time.Date(2026, 11, 30, 16, 0, 0, 0, time.UTC)))
	assert.False(t, s.Active(time.Date(2026, 11, 30, 14, 0, 0, 0, time.UTC)))
}

func TestSchedule_Active_FailsClosed(t *testing.T) {
	bad := []*Schedule{
		{Timezone: "Not/AZone"},
		{Timezone: ""},
		{Timezone: "UTC", StartDate: "garbage"},
		{Timezone: "UTC", Windows: []Window{{Days: []string{"funday"}, Start: "08:00", End: "09:00"}}},
		{Timezone: "UTC", Windows: []Window{{Days: []string{"mon"}, Start: "9am", End: "10:00"}}},
	}
	for _, s := range bad {
		assert.False(t, s.Active(at(t, "UTC", "2026-11-23 08:30")), "%+v must not be active", s)
	}
}

func TestSchedule_Validate(t *testing.T) {
	valid := Window{Days: []string{"mon"}, Start: "08:00", End: "17:00"}
	tests := []struct {
		name    string
		s       Schedule
		wantErr string
	}{
		{"ok", Schedule{Timezone: "UTC", Windows: []Window{valid}}, ""},
		{"day names are case insensitive", Schedule{Timezone: "UTC", Windows: []Window{{Days: []string{"MON"}, Start: "08:00", End: "09:00"}}}, ""},
		{"missing timezone", Schedule{Windows: []Window{valid}}, "timezone is required"},
		{"unknown timezone", Schedule{Timezone: "Mars/Base"}, "timezone"},
		{"no days", Schedule{Timezone: "UTC", Windows: []Window{{Start: "08:00", End: "09:00"}}}, "at least one day"},
		{"bad day", Schedule{Timezone: "UTC", Windows: []Window{{Days: []string{"xyz"}, Start: "08:00", End: "09:00"}}}, "invalid day"},
		{"start after end", Schedule{Timezone: "UTC", Windows: []Window{{Days: []string{"mon"}, Start: "18:00", End: "09:00"}}}, "must be before"},
		{"equal bounds", Schedule{Timezone: "UTC", Windows: []Window{{Days: []string{"mon"}, Start: "09:00", End: "09:00"}}}, "must be before"},
		{"bad clock", Schedule{Timezone: "UTC", Windows: []Window{{Days: []string{"mon"}, Start: "25:00", End: "26:00"}}}, "invalid time"},
		{"bad date", Schedule{Timezone: "UTC", EndDate: "01/12/2026"}, "invalid date"},
		{"reversed dates", Schedule{Timezone: "UTC", StartDate: "2026-12-31", EndDate: "2026-12-01"}, "before start_date"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.s.Validate()
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestSchedule_Validate_TooManyWindows(t *testing.T) {
	s := Schedule{Timezone: "UTC"}
	for i := 0; i <= maxWindows; i++ {
		s.Windows = append(s.Windows, Window{Days: []string{"mon"}, Start: "08:00", End: "09:00"})
	}
	assert.Error(t, s.Validate())
}

func TestSchedule_Copy_IsDeep(t *testing.T) {
	orig := &Schedule{Timezone: "UTC", Windows: []Window{{Days: []string{"mon"}, Start: "08:00", End: "09:00"}}}
	cp := orig.Copy()
	cp.Windows[0].Days[0] = "tue"
	cp.Windows[0].Start = "10:00"
	assert.Equal(t, "mon", orig.Windows[0].Days[0])
	assert.Equal(t, "08:00", orig.Windows[0].Start)
}
