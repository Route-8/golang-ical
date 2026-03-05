package ics

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDuration(t *testing.T) {
	date, _ := time.Parse(time.RFC822, time.RFC822)
	duration := time.Duration(float64(time.Hour) * 2)

	testCases := []struct {
		name   string
		start  time.Time
		end    time.Time
		output string
	}{
		{
			name:  "test set duration - start",
			start: date,
			output: `BEGIN:VEVENT
UID:test-duration
DTSTART:20060102T150400Z
DTEND:20060102T170400Z
END:VEVENT
`,
		},
		{
			name: "test set duration - end",
			end:  date,
			output: `BEGIN:VEVENT
UID:test-duration
DTEND:20060102T150400Z
DTSTART:20060102T130400Z
END:VEVENT
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEvent("test-duration")
			if !tc.start.IsZero() {
				e.SetStartAt(tc.start)
			}
			if !tc.end.IsZero() {
				e.SetEndAt(tc.end)
			}
			err := e.SetDuration(duration)

			// we're not testing for encoding here so lets make the actual output line breaks == expected line breaks
			text := strings.Replace(e.Serialize(), "\r\n", "\n", -1)

			assert.Equal(t, tc.output, text)
			assert.Equal(t, nil, err)
		})
	}
}

func TestDefaultTimezone(t *testing.T) {
	eastern, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	pacific, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)

	t.Run("no default uses time.Local", func(t *testing.T) {
		e := NewEvent("test")
		e.SetProperty(ComponentPropertyDtStart, "20260309")
		got, err := e.GetAllDayStartAt()
		require.NoError(t, err)
		expected, _ := time.ParseInLocation("20060102", "20260309", time.Local)
		assert.Equal(t, expected, got)
	})

	t.Run("event-level default timezone", func(t *testing.T) {
		e := NewEvent("test")
		e.SetDefaultTimezone(eastern)
		e.SetProperty(ComponentPropertyDtStart, "20260309")
		got, err := e.GetAllDayStartAt()
		require.NoError(t, err)
		expected, _ := time.ParseInLocation("20060102", "20260309", eastern)
		assert.Equal(t, expected, got)
		assert.Equal(t, "America/New_York", got.Location().String())
	})

	t.Run("calendar-level default propagates to events", func(t *testing.T) {
		cal := NewCalendar()
		cal.SetDefaultTimezone(eastern)
		e := cal.AddEvent("test")
		e.SetProperty(ComponentPropertyDtStart, "20260309")

		events := cal.Events()
		require.Len(t, events, 1)
		got, err := events[0].GetAllDayStartAt()
		require.NoError(t, err)
		assert.Equal(t, "America/New_York", got.Location().String())
	})

	t.Run("event-level override takes precedence", func(t *testing.T) {
		cal := NewCalendar()
		cal.SetDefaultTimezone(eastern)
		e := cal.AddEvent("test")
		e.SetDefaultTimezone(pacific)
		e.SetProperty(ComponentPropertyDtStart, "20260309")

		events := cal.Events()
		require.Len(t, events, 1)
		got, err := events[0].GetAllDayStartAt()
		require.NoError(t, err)
		assert.Equal(t, "America/Los_Angeles", got.Location().String())
	})

	t.Run("explicit TZID wins over default", func(t *testing.T) {
		e := NewEvent("test")
		e.SetDefaultTimezone(pacific)
		e.AddProperty(ComponentPropertyDtStart, "20260309T100000", &KeyValues{Key: "TZID", Value: []string{"America/New_York"}})
		got, err := e.GetStartAt()
		require.NoError(t, err)
		assert.Equal(t, "America/New_York", got.Location().String())
	})

	t.Run("UTC suffix wins over default", func(t *testing.T) {
		e := NewEvent("test")
		e.SetDefaultTimezone(pacific)
		e.SetProperty(ComponentPropertyDtStart, "20260309T100000Z")
		got, err := e.GetStartAt()
		require.NoError(t, err)
		assert.Equal(t, "UTC", got.Location().String())
	})

	t.Run("local datetime uses default timezone", func(t *testing.T) {
		e := NewEvent("test")
		e.SetDefaultTimezone(eastern)
		e.SetProperty(ComponentPropertyDtStart, "20260309T100000")
		got, err := e.GetStartAt()
		require.NoError(t, err)
		assert.Equal(t, "America/New_York", got.Location().String())
	})

	t.Run("nil default reverts to time.Local", func(t *testing.T) {
		e := NewEvent("test")
		e.SetDefaultTimezone(eastern)
		e.SetDefaultTimezone(nil)
		e.SetProperty(ComponentPropertyDtStart, "20260309")
		got, err := e.GetAllDayStartAt()
		require.NoError(t, err)
		expected, _ := time.ParseInLocation("20060102", "20260309", time.Local)
		assert.Equal(t, expected, got)
	})

	t.Run("AddVEvent propagates default timezone", func(t *testing.T) {
		cal := NewCalendar()
		cal.SetDefaultTimezone(eastern)
		e := NewEvent("test")
		e.SetProperty(ComponentPropertyDtStart, "20260309")
		cal.AddVEvent(e)

		events := cal.Events()
		require.Len(t, events, 1)
		got, err := events[0].GetAllDayStartAt()
		require.NoError(t, err)
		assert.Equal(t, "America/New_York", got.Location().String())
	})

	t.Run("SetDefaultTimezone after ParseCalendar propagates to parsed events", func(t *testing.T) {
		calStr := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260309T100000
DTEND:20260309T110000
SUMMARY:Test Event
END:VEVENT
END:VCALENDAR`
		cal, err := ParseCalendar(strings.NewReader(calStr))
		require.NoError(t, err)

		// Before setting timezone, times should use time.Local
		events := cal.Events()
		require.Len(t, events, 1)
		got, err := events[0].GetStartAt()
		require.NoError(t, err)
		assert.Equal(t, time.Local.String(), got.Location().String())

		// After setting timezone, times should use the new timezone
		cal.SetDefaultTimezone(eastern)

		got, err = events[0].GetStartAt()
		require.NoError(t, err)
		assert.Equal(t, "America/New_York", got.Location().String())
	})
}
