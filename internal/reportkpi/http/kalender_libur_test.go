package reportkpihttp_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
	"claim-pnc/internal/reportkpi/repo/memory"
	reportkpisql "claim-pnc/internal/reportkpi/repo/sqlstore"
)

// tanpaKalender adalah repo contoh yang kalender hari liburnya TIDAK dapat dibaca.
//
// Ia meniru keadaan nyata pada 2026-10-08: koneksi kedua portal
// (`ANEKA_<PORTAL_ALIAS>_*`) belum terpasang, sehingga `GENERAL.HRD_LBR` tidak terjangkau
// sementara seluruh objek lain terbaca normal.
type tanpaKalender struct{ reportkpi.Repo }

func (tanpaKalender) Holidays(context.Context, time.Time, time.Time) ([]time.Time, error) {
	return nil, reportkpisql.ErrHolidayCalendarUnavailable
}

// Ketiadaan koneksi kedua dijawab dengan pesan yang DAPAT DITINDAKLANJUTI, bukan dengan
// "Terjadi kesalahan pada sistem".
//
// # Kenapa uji ini ada
//
// Ia menutup jarak antara "galatnya sudah dibedakan di repo" dan "pengguna benar-benar
// melihat bedanya". Keduanya tidak otomatis sama: galat yang tidak dikenali lapisan
// transport akan diteruskan ke penulis galat cadangan dan kembali menjadi kalimat umum —
// persis keadaan yang hendak diperbaiki.
//
// Yang dituntut di sini bukan sekadar status dan kodenya, melainkan ISI pesannya: ia harus
// menyebut nama variabel yang perlu diisi. Pesan yang hanya berkata "koneksi kedua tidak
// tersedia" memaksa pembacanya mencari tahu koneksi yang mana, dan itu meninggalkan
// pembacanya di tempat yang hampir sama buntunya.
func TestKalenderLiburTidakTerbacaDijawabDenganArahan(t *testing.T) {
	h := newHarness(t, tanpaKalender{memory.NewSampleStore()})

	rec := h.do(http.MethodGet, "/report-kpi/pic-teknik?lini_bisnis=NONMBU&"+period)

	// 503, bukan 500: tidak ada yang rusak — sambungannya yang belum dipasang.
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	body := decode(t, rec)
	require.Equal(t, "koneksi_kedua_belum_terpasang", body["kode"])

	pesan, _ := body["pesan"].(string)
	require.Contains(t, pesan, "ANEKA_<PORTAL>_HOST",
		"pesannya harus menyebut variabel yang perlu diisi, bukan hanya bahwa sesuatu kurang")
	require.Contains(t, pesan, "hari libur")
	require.NotContains(t, pesan, "Terjadi kesalahan pada sistem")
}
