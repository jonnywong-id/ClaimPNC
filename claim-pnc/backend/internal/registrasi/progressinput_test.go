package registrasi_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

func progressMessages(v []registrasi.Violation) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		out = append(out, x.Message)
	}
	return out
}

// Pesan Pega apa adanya, urut seperti ValidationInputProgressClaim_act.
func TestCheckProgressInput(t *testing.T) {
	today := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	ok := registrasi.ProgressInput{Progress1: "004", Needs: []registrasi.ProgressNeed{{ID: "59", Name: "HASIL SURVEY"}},
		Status: "SUDAH BERHASIL DIHUBUNGKAN", Remarks: "catatan"}
	require.Empty(t, registrasi.CheckProgressInput(ok, "", today))

	bad := ok
	bad.Needs = []registrasi.ProgressNeed{{ID: "59"}, {ID: ""}}
	bad.Status = registrasi.ProgressStatusPrompt
	bad.NextFollowUp = time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	bad.Remarks = ""
	require.Equal(t, []string{registrasi.MsgProgressNeedFromMaster, registrasi.MsgProgressFollowUpPast,
		registrasi.MsgProgressStatusEmpty, registrasi.MsgProgressRemarksEmpty}, progressMessages(registrasi.CheckProgressInput(bad, "", today)))

	// Besok sah; Email Progress Klaim wajib untuk 003/008/011.
	email := ok
	email.Progress1 = "008"
	email.NextFollowUp = time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	require.Equal(t, []string{registrasi.MsgProgressEmailEmpty}, progressMessages(registrasi.CheckProgressInput(email, " ", today)))
	require.Empty(t, registrasi.CheckProgressInput(email, "a@b.c", today))

	none := ok
	none.Needs = nil
	none.Progress1 = ""
	require.Equal(t, []string{registrasi.MsgProgressStatus1Empty, registrasi.MsgProgressNeedFromMaster},
		progressMessages(registrasi.CheckProgressInput(none, "", today)))

	long := ok
	long.Remarks = strings.Repeat("a", registrasi.ProgressRemarksMax+1)
	require.Equal(t, []string{registrasi.MsgProgressRemarksTooLong}, progressMessages(registrasi.CheckProgressInput(long, "", today)))
}

// STATUS PERLU baris terakhir dibandingkan secara "terkandung" (@contains).
func TestAlreadyInputToday(t *testing.T) {
	require.True(t, registrasi.AlreadyInputToday("BELUM DI FU", []string{"X, BELUM DI FU"}))
	require.False(t, registrasi.AlreadyInputToday("BELUM DI FU", []string{"NASABAH TIDAK BISA DIHUBUNGI"}))
	require.False(t, registrasi.AlreadyInputToday("", []string{"A"}))
	p := registrasi.ProgressInput{Needs: []registrasi.ProgressNeed{{ID: "1"}, {ID: "2"}}}
	require.Equal(t, "2", p.LastNeed().ID)
	require.Equal(t, "", registrasi.ProgressInput{}.LastNeed().ID)
	require.True(t, registrasi.ProgressPosition{Status: "On Progress"}.Open())
	require.False(t, registrasi.ProgressPosition{Status: "Done"}.Open())
	require.True(t, registrasi.ProgressNeedsEmail(" 011 "))
	require.False(t, registrasi.ProgressNeedsEmail("004"))
}
