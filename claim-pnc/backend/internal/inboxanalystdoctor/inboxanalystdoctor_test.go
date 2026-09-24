package inboxanalystdoctor_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxanalystdoctor"
)

// wib adalah zona tampilan yang dipakai seluruh uji di berkas ini.
//
// Offset tetap, bukan `LoadLocation`: uji tidak boleh gagal di mesin yang basis data zona
// waktunya tidak terpasang.
var wib = time.FixedZone("WIB", 7*60*60)

// TestPenandaAntreanTidakTertukarDenganCompliance mengunci kedua nilai berdampingan.
//
// Keduanya diuji bersama, bukan sendiri-sendiri, karena yang berbahaya bukan nilainya salah
// melainkan nilainya TERTUKAR: `"1"` menempatkan klaim di antrean Compliance dan `"2"` di
// antrean ini, dan satu digit yang salah memindahkan seluruh isi layar tanpa satu pun galat.
func TestPenandaAntreanTidakTertukarDenganCompliance(t *testing.T) {
	require.Equal(t, "2", inboxanalystdoctor.TransferAnalystDoctor,
		"penanda antrean Analyst Doctor — Report Definition/InboxAnalystDoctor_RD-RD.xml")
	require.Equal(t, "1", inboxanalystdoctor.TransferCompliance,
		"penanda antrean Compliance — When/IsCompliance-When.xml")
	require.NotEqual(t, inboxanalystdoctor.TransferCompliance,
		inboxanalystdoctor.TransferAnalystDoctor)
}

// TestHanyaResolvedCompletedYangDikecualikan menjaga perbedaan terhadap Inbox Outstanding.
//
// `InboxAnalystDoctor_RD` menyaring `!= "Resolved-Completed"` SAJA. Menambahkan
// `Resolved-Rejected` — seperti yang dilakukan Inbox Outstanding — akan menghilangkan klaim
// yang ditolak dari antrean ini, dan itu perubahan perilaku tanpa dasar keputusan apa pun.
func TestHanyaResolvedCompletedYangDikecualikan(t *testing.T) {
	require.Equal(t, "Resolved-Completed", inboxanalystdoctor.StatusKerjaSelesai)
}

func TestFilterNormalizeMengisiNilaiBawaan(t *testing.T) {
	clean := inboxanalystdoctor.Filter{}.Normalize()

	require.Equal(t, inboxanalystdoctor.DefaultLimit, clean.Limit)
	require.Equal(t, 0, clean.Offset)
	require.Equal(t, "", clean.Search)
}

func TestFilterNormalizeMemangkasBatasBerlebihan(t *testing.T) {
	// `10-API-STRATEGY.md` §4: permintaan di atas batas DIPANGKAS, dan angka yang dipakai
	// dikembalikan ke layar. Memenuhinya apa adanya akan membuat satu permintaan menarik
	// ribuan baris klaim (`D-10`).
	clean := inboxanalystdoctor.Filter{Limit: 5000}.Normalize()

	require.Equal(t, inboxanalystdoctor.MaxLimit, clean.Limit)
}

func TestFilterNormalizeMembersihkanSpasiDanOffsetNegatif(t *testing.T) {
	clean := inboxanalystdoctor.Filter{Search: "  PNCN.26  ", Offset: -10}.Normalize()

	require.Equal(t, "PNCN.26", clean.Search)
	require.Equal(t, 0, clean.Offset)
}

func TestDurationDaysMenghitungTerhadapTanggalWIB(t *testing.T) {
	// Tugas masuk 18 September pukul 09.00 UTC = 16.00 WIB.
	// Dilihat 20 September pukul 01.00 UTC = 08.00 WIB.
	// Selisih jamnya 40 jam — dibagi 24 menghasilkan 1. Yang benar adalah 2, karena yang
	// dihitung adalah pergantian TANGGAL.
	task := inboxanalystdoctor.AnalystDoctorTask{
		RegisteredAt: time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC)

	require.Equal(t, 2, task.DurationDays(now, wib))
}

// TestDurationDaysTidakBergeserKarenaZonaWaktu mengunci sebab yang paling mudah terlewat.
//
// Tugas yang masuk pukul 02.00 WIB tanggal 19 tersimpan sebagai 19.00 UTC tanggal 18. Bila
// konversinya dilupakan dan hitungannya dilakukan atas tanggal UTC, umurnya bertambah satu
// hari — pada SETIAP tugas yang masuk antara pukul 00.00 dan 07.00 WIB.
func TestDurationDaysTidakBergeserKarenaZonaWaktu(t *testing.T) {
	task := inboxanalystdoctor.AnalystDoctorTask{
		RegisteredAt: time.Date(2026, time.September, 18, 19, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, time.September, 19, 3, 0, 0, 0, time.UTC) // 10.00 WIB tanggal 19

	require.Equal(t, 0, task.DurationDays(now, wib), "keduanya tanggal 19 WIB")
	require.Equal(t, 1, task.DurationDays(now, time.UTC), "pembanding: tanpa konversi ia bergeser")
}

func TestDurationDaysNolSaatTanggalPendaftaranKosong(t *testing.T) {
	task := inboxanalystdoctor.AnalystDoctorTask{}

	require.Equal(t, 0, task.DurationDays(time.Now(), wib))
}

// TestDurationDaysTidakPernahNegatif menjaga data cacat tidak berubah menjadi angka aneh.
//
// Tanggal pendaftaran di masa depan adalah data yang salah, bukan durasi negatif. Layar
// menampilkan nol; yang memperbaikinya adalah datanya.
func TestDurationDaysTidakPernahNegatif(t *testing.T) {
	task := inboxanalystdoctor.AnalystDoctorTask{
		RegisteredAt: time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC)

	require.Equal(t, 0, task.DurationDays(now, wib))
}
