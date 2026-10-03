package clock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
)

// TestFixedClockHoldsAndAdvances membuktikan jam tetap disimpan UTC dan hanya bergeser lewat Advance.
func TestFixedClockHoldsAndAdvances(t *testing.T) {
	start := time.Date(2026, 9, 18, 10, 0, 0, 0, clock.ZoneWIB)
	fixed := clock.FixedAt(start)
	require.Equal(t, time.UTC, fixed.Now().Location())
	require.True(t, fixed.Now().Equal(start))

	fixed.Advance(90 * time.Minute)
	require.True(t, fixed.Now().Equal(start.Add(90*time.Minute)))
}

// TestSystemClockIsUTC membuktikan jam sistem selalu mengembalikan UTC.
func TestSystemClockIsUTC(t *testing.T) {
	var c clock.Clock = clock.System{}
	before := time.Now()
	now := c.Now()
	require.Equal(t, time.UTC, now.Location())
	require.False(t, now.Before(before.Add(-time.Second)))
}

// TestDaysBetweenCountsWIBCalendarDays membuktikan jarak hari dihitung pada kalender WIB.
func TestDaysBetweenCountsWIBCalendarDays(t *testing.T) {
	// 16:59 UTC = 23:59 WIB tanggal 1; 17:00 UTC = 00:00 WIB tanggal 2.
	a := time.Date(2026, 1, 1, 16, 59, 0, 0, time.UTC)
	b := time.Date(2026, 1, 1, 17, 0, 0, 0, time.UTC)
	require.Equal(t, 1, clock.DaysBetween(a, b))
	require.Equal(t, -1, clock.DaysBetween(b, a))
	require.Equal(t, 7, clock.DaysBetween(a, a.AddDate(0, 0, 7)))
}

// TestAddDaysShiftsWIBDate membuktikan AddDays mengembalikan tengah malam WIB tanggal tujuan.
func TestAddDaysShiftsWIBDate(t *testing.T) {
	at := time.Date(2026, 1, 31, 18, 0, 0, 0, time.UTC) // 1 Feb 01:00 WIB
	got := clock.AddDays(at, 30)
	require.Equal(t, time.Date(2026, 3, 3, 0, 0, 0, 0, clock.ZoneWIB), got)
}
