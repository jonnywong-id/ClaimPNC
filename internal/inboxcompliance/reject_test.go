package inboxcompliance_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
)

// Uji penomoran alasan DICABUT 2026-10-08.
//
// `RejectLetter.NumberedReasons` menomori ulang grid Alasan persis seperti
// `DownloadPDFRejectRefund` langkah 5 dan 6 — dan logika itu memang ada di Pega.
// Tetapi hasilnya TIDAK PERNAH SAMPAI KE SURAT: templat menggambar `1.` `2.` `3.`
// sebagai teks tetap tanpa isi.
//
// Work Owner menetapkan templat diikuti apa adanya, sehingga penomoran itu kehilangan
// satu-satunya pemakainya dan ikut dicabut bersama ujinya. Temuannya tetap tercatat di
// `docs/keputusan-implementasi.md` §221 — yang dibuang kodenya, bukan pengetahuannya.

// Tanggal surat memakai nama bulan Indonesia dan WIB, bukan zona server.
//
// Waktu ujinya sengaja dipilih pukul 20.00 UTC: di WIB ia sudah HARI BERIKUTNYA. Tanpa
// pergeseran zona, surat yang dibuat malam hari akan bertanggal mundur satu hari.
func TestTanggalSuratMemakaiWIB(t *testing.T) {
	t.Parallel()

	malam := time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	require.Equal(t, "08 Oktober 2026", inboxcompliance.NewRejectLetterDate(malam))

	pagi := time.Date(2026, 1, 5, 2, 0, 0, 0, time.UTC)
	require.Equal(t, "05 Januari 2026", inboxcompliance.NewRejectLetterDate(pagi))
}

// Nomor surat memakai MENIT dan DETIK, bukan pencacah — ditiru apa adanya (`P-5`).
//
// `DownloadPDFReject` langkah 1: `AcceptanceBank = FormatDateTime(now, "mmss")`.
func TestNomorSuratMemakaiMenitDanDetik(t *testing.T) {
	t.Parallel()

	// 20.15 UTC = 03.15 WIB keesokan harinya; menit 15, detik 32.
	saat := time.Date(2026, 10, 7, 20, 15, 32, 0, time.UTC)
	require.Equal(t, "1532/CL.AHID.ASM/10/2026", inboxcompliance.NewRejectLetterNumber(saat))

	// Menit dan detik satu digit tetap diberi nol di depan, seperti format "mmss" Pega.
	awal := time.Date(2026, 3, 1, 1, 4, 5, 0, time.UTC)
	require.Equal(t, "0405/CL.AHID.ASM/03/2026", inboxcompliance.NewRejectLetterNumber(awal))
}

// Nomor surat TIDAK unik, dan itu memang perilaku sistem lama yang ditiru.
//
// Diuji secara tegas supaya siapa pun yang kelak "memperbaikinya" menjadi sequence tahu
// bahwa ia sedang mengubah perilaku yang disengaja, bukan membetulkan cacat yang luput.
func TestNomorSuratSengajaTidakUnikAntarBulanYangSama(t *testing.T) {
	t.Parallel()

	pagi := time.Date(2026, 10, 2, 1, 15, 32, 0, time.UTC)
	sore := time.Date(2026, 10, 25, 9, 15, 32, 0, time.UTC)

	require.Equal(t,
		inboxcompliance.NewRejectLetterNumber(pagi),
		inboxcompliance.NewRejectLetterNumber(sore),
		"dua surat berbeda hari pada bulan yang sama bernomor sama — perilaku sistem lama")
}
