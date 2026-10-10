package schedule_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/schedule"
)

func TestParseTimes(t *testing.T) {
	got, err := schedule.ParseTimes(" 13:00 , 08:15 ")
	require.NoError(t, err)
	require.Equal(t, []schedule.TimeOfDay{{8, 15}, {13, 0}}, got)

	for _, bad := range []string{"", "8", "24:00", "08:60", "aa:bb"} {
		_, err := schedule.ParseTimes(bad)
		require.Error(t, err, bad)
	}
}

func TestNext(t *testing.T) {
	times := []schedule.TimeOfDay{{8, 15}, {13, 0}}
	wib := clock.ZoneWIB
	at := func(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, wib) }

	require.Equal(t, at(9, 8, 15), schedule.Next(at(9, 7, 0), times, wib))
	require.Equal(t, at(9, 13, 0), schedule.Next(at(9, 8, 15), times, wib), "tepat pada jamnya: putaran berikutnya")
	require.Equal(t, at(10, 8, 15), schedule.Next(at(9, 23, 0), times, wib))
	// Jam server UTC: 01:00 UTC = 08:00 WIB.
	require.Equal(t, at(9, 8, 15), schedule.Next(time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC), times, wib))
}

func TestDailyStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		schedule.Daily(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), "uji",
			[]schedule.TimeOfDay{{8, 15}}, clock.ZoneWIB, time.Now, func(context.Context) error { return nil })
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Daily tidak berhenti setelah konteks dibatalkan")
	}
}
