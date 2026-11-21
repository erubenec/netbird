// Package schedule describes the recurring time windows during which a
// reverse proxy service is reachable, and evaluates them against a clock.
package schedule

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	// Embed the tz database: minimal container images ship no zoneinfo, and
	// without it every IANA timezone fails to load.
	_ "time/tzdata"
)

const (
	dateLayout = "2006-01-02"
	// maxWindows bounds the number of windows so a mapping stays small.
	maxWindows = 64
	// endOfDay is the only accepted end time that is not a valid clock time.
	endOfDay = "24:00"
)

// weekdays maps the accepted day names to time.Weekday.
var weekdays = map[string]time.Weekday{
	"sun": time.Sunday,
	"mon": time.Monday,
	"tue": time.Tuesday,
	"wed": time.Wednesday,
	"thu": time.Thursday,
	"fri": time.Friday,
	"sat": time.Saturday,
}

// Window is a daily time range that repeats on the listed days.
// Start is inclusive and End is exclusive, both as "HH:MM" in the schedule
// timezone. End may be "24:00". A window never crosses midnight; split it into
// two windows on consecutive days instead.
type Window struct {
	Days  []string `json:"days"`
	Start string   `json:"start"`
	End   string   `json:"end"`
}

// Schedule restricts when a service is reachable. A nil *Schedule means the
// service is always reachable.
type Schedule struct {
	// Timezone is an IANA name such as "Europe/Madrid".
	Timezone string `json:"timezone"`
	// Windows lists the active time ranges. Empty means active all day, every
	// day, within the date range.
	Windows []Window `json:"windows,omitempty"`
	// StartDate is the first active day, "YYYY-MM-DD" in the schedule timezone.
	StartDate string `json:"start_date,omitempty"`
	// EndDate is the last active day (inclusive), "YYYY-MM-DD".
	EndDate string `json:"end_date,omitempty"`
}

// Copy returns a deep copy of the schedule, or nil for a nil receiver.
func (s *Schedule) Copy() *Schedule {
	if s == nil {
		return nil
	}
	out := *s
	out.Windows = make([]Window, len(s.Windows))
	for i, w := range s.Windows {
		out.Windows[i] = Window{Days: slices.Clone(w.Days), Start: w.Start, End: w.End}
	}
	if len(s.Windows) == 0 {
		out.Windows = nil
	}
	return &out
}

// Validate checks that the schedule can be evaluated.
func (s *Schedule) Validate() error {
	if s == nil {
		return nil
	}
	if _, err := s.location(); err != nil {
		return err
	}
	if len(s.Windows) > maxWindows {
		return fmt.Errorf("schedule has %d windows, maximum is %d", len(s.Windows), maxWindows)
	}
	for i, w := range s.Windows {
		if _, _, err := w.bounds(); err != nil {
			return fmt.Errorf("schedule window %d: %w", i+1, err)
		}
		if _, err := w.weekdaySet(); err != nil {
			return fmt.Errorf("schedule window %d: %w", i+1, err)
		}
	}
	start, err := parseDate(s.StartDate)
	if err != nil {
		return fmt.Errorf("schedule start_date: %w", err)
	}
	end, err := parseDate(s.EndDate)
	if err != nil {
		return fmt.Errorf("schedule end_date: %w", err)
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return errors.New("schedule end_date is before start_date")
	}
	return nil
}

// Active reports whether the service is reachable at t. It fails closed: a
// schedule that cannot be evaluated is never active.
func (s *Schedule) Active(t time.Time) bool {
	if s == nil {
		return true
	}
	loc, err := s.location()
	if err != nil {
		return false
	}
	local := t.In(loc)

	if !s.inDateRange(local) {
		return false
	}
	if len(s.Windows) == 0 {
		return true
	}

	minutes := local.Hour()*60 + local.Minute()
	for _, w := range s.Windows {
		if w.contains(local.Weekday(), minutes) {
			return true
		}
	}
	return false
}

func (s *Schedule) location() (*time.Location, error) {
	if s.Timezone == "" {
		return nil, errors.New("schedule timezone is required")
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return nil, fmt.Errorf("schedule timezone %q: %w", s.Timezone, err)
	}
	return loc, nil
}

func (s *Schedule) inDateRange(local time.Time) bool {
	today, err := parseDate(local.Format(dateLayout))
	if err != nil {
		return false
	}
	start, err := parseDate(s.StartDate)
	if err != nil {
		return false
	}
	end, err := parseDate(s.EndDate)
	if err != nil {
		return false
	}
	if !start.IsZero() && today.Before(start) {
		return false
	}
	return end.IsZero() || !today.After(end)
}

func (w Window) contains(day time.Weekday, minutes int) bool {
	days, err := w.weekdaySet()
	if err != nil || !slices.Contains(days, day) {
		return false
	}
	start, end, err := w.bounds()
	if err != nil {
		return false
	}
	return minutes >= start && minutes < end
}

func (w Window) weekdaySet() ([]time.Weekday, error) {
	if len(w.Days) == 0 {
		return nil, errors.New("at least one day is required")
	}
	out := make([]time.Weekday, 0, len(w.Days))
	for _, name := range w.Days {
		day, ok := weekdays[strings.ToLower(name)]
		if !ok {
			return nil, fmt.Errorf("invalid day %q, expected sun, mon, tue, wed, thu, fri or sat", name)
		}
		out = append(out, day)
	}
	return out, nil
}

// bounds returns the window start and end as minutes since midnight.
func (w Window) bounds() (int, int, error) {
	start, err := parseClock(w.Start)
	if err != nil {
		return 0, 0, fmt.Errorf("start: %w", err)
	}
	end, err := parseClock(w.End)
	if err != nil {
		return 0, 0, fmt.Errorf("end: %w", err)
	}
	if start >= end {
		return 0, 0, fmt.Errorf("start %s must be before end %s", w.Start, w.End)
	}
	return start, end, nil
}

func parseClock(v string) (int, error) {
	if v == endOfDay {
		return 24 * 60, nil
	}
	t, err := time.Parse("15:04", v)
	if err != nil {
		return 0, fmt.Errorf("invalid time %q, expected HH:MM", v)
	}
	return t.Hour()*60 + t.Minute(), nil
}

// parseDate parses "YYYY-MM-DD" and returns the zero time for an empty value.
func parseDate(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(dateLayout, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", v)
	}
	return t, nil
}
