package inboxcloseclaim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcloseclaim"
)

func jakarta(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}

// TestDisplayStatusFollowsBrowseClaimAll menjaga pemetaan yang diambil dari SQL Pega, bukan
// dari tebakan.
func TestDisplayStatusFollowsBrowseClaimAll(t *testing.T) {
	selesai := inboxcloseclaim.ClosedClaim{ProcessStatus: inboxcloseclaim.StatusKerjaSelesai}
	ditolak := inboxcloseclaim.ClosedClaim{ProcessStatus: inboxcloseclaim.StatusKerjaDitolak}

	require.Equal(t, inboxcloseclaim.DisplayClose, selesai.DisplayStatus())
	require.Equal(t, inboxcloseclaim.DisplayReject, ditolak.DisplayStatus())
}

// TestUnknownStatusIsShownAsIs menjaga cabang default menampilkan nilai apa adanya.
//
// Memaksanya menjadi "Close" akan menyembunyikan keadaan yang tidak diduga: status asing
// yang tampil mentah segera ditanyakan pengguna, status asing yang menyamar tidak pernah.
func TestUnknownStatusIsShownAsIs(t *testing.T) {
	claim := inboxcloseclaim.ClosedClaim{ProcessStatus: "  Pending-Something  "}
	require.Equal(t, inboxcloseclaim.DisplayStatus("Pending-Something"), claim.DisplayStatus())
}

// TestDurationCountsUntilClosingDate adalah uji kolom "Lama Waktu Klaim".
//
// Yang diuji bukan sekadar aritmetikanya melainkan TITIK AKHIRNYA: klaim yang tutup tiga
// tahun lalu bukan klaim berumur seribu hari.
func TestDurationCountsUntilClosingDate(t *testing.T) {
	loc := jakarta(t)
	daftar := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)
	tutup := time.Date(2026, 1, 11, 3, 0, 0, 0, time.UTC)

	claim := inboxcloseclaim.ClosedClaim{RegisteredAt: daftar, ClosedAt: &tutup}

	// Hari ini jauh setelah klaim tutup. Bila titik akhirnya keliru memakai "sekarang",
	// hasilnya ratusan hari, bukan sepuluh.
	now := time.Date(2029, 6, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, 10, claim.DurationDays(now, loc))
}

// TestDurationFallsBackToResolvedThenNow menjaga ketiga tingkat cadangannya.
func TestDurationFallsBackToResolvedThenNow(t *testing.T) {
	loc := jakarta(t)
	daftar := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)
	selesai := time.Date(2026, 1, 6, 3, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 21, 3, 0, 0, 0, time.UTC)

	tanpaTutup := inboxcloseclaim.ClosedClaim{RegisteredAt: daftar, ResolvedAt: &selesai}
	require.Equal(t, 5, tanpaTutup.DurationDays(now, loc), "cadangan pertama: PYRESOLVEDTIMESTAMP")

	tanpaKeduanya := inboxcloseclaim.ClosedClaim{RegisteredAt: daftar}
	require.Equal(t, 20, tanpaKeduanya.DurationDays(now, loc), "cadangan kedua: sekarang")
}

// TestDurationCountsCalendarDaysInWIB menjaga keputusan menghitung terhadap TANGGAL, bukan
// selisih jam dibagi 24.
//
// Klaim yang didaftarkan pukul 23.00 WIB dan ditutup pukul 01.00 keesokan harinya sudah
// berumur satu hari bagi pengguna, meski selisihnya dua jam. Membagi selisih jam akan
// mengembalikan nol.
func TestDurationCountsCalendarDaysInWIB(t *testing.T) {
	loc := jakarta(t)

	// 23.00 WIB = 16.00 UTC pada hari yang sama.
	daftar := time.Date(2026, 3, 10, 16, 0, 0, 0, time.UTC)
	// 01.00 WIB keesokan harinya = 18.00 UTC pada hari yang sama.
	tutup := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)

	claim := inboxcloseclaim.ClosedClaim{RegisteredAt: daftar, ClosedAt: &tutup}
	require.Equal(t, 1, claim.DurationDays(time.Now(), loc))
}

// TestDurationNeverNegative menjaga data cacat tidak menjadi durasi negatif.
func TestDurationNeverNegative(t *testing.T) {
	loc := jakarta(t)
	daftar := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	tutup := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	claim := inboxcloseclaim.ClosedClaim{RegisteredAt: daftar, ClosedAt: &tutup}
	require.Equal(t, 0, claim.DurationDays(time.Now(), loc))
}

// TestBusinessLineParsing menjaga kelima nilai beserta perlakuan isian kosong.
func TestBusinessLineParsing(t *testing.T) {
	for _, line := range inboxcloseclaim.BusinessLines() {
		parsed, known := inboxcloseclaim.ParseBusinessLine(string(line))
		require.True(t, known)
		require.Equal(t, line, parsed)
	}

	// Kosong berarti ALL, bukan galat: layar yang baru dibuka belum memilih apa pun.
	parsed, known := inboxcloseclaim.ParseBusinessLine("   ")
	require.True(t, known)
	require.Equal(t, inboxcloseclaim.BusinessAll, parsed)

	// Huruf kecil diterima; nilainya dinormalkan.
	parsed, known = inboxcloseclaim.ParseBusinessLine("bonding")
	require.True(t, known)
	require.Equal(t, inboxcloseclaim.BusinessBonding, parsed)

	_, known = inboxcloseclaim.ParseBusinessLine("MBU")
	require.False(t, known, "lini yang tidak ada di activity lama harus ditolak")
}

// TestBusinessLinesMatchTheLegacyActivity menjaga daftarnya tetap lima, tidak lebih.
//
// Activity lama bercabang atas `TempView2.Remark` dengan lima cabang dan tidak ada cabang
// keenam. Menambah pilihan di sini berarti menambah perilaku yang tidak ada di Pega.
func TestBusinessLinesMatchTheLegacyActivity(t *testing.T) {
	require.Equal(t, []inboxcloseclaim.BusinessLine{
		inboxcloseclaim.BusinessAll,
		inboxcloseclaim.BusinessNonMBU,
		inboxcloseclaim.BusinessBonding,
		inboxcloseclaim.BusinessPA,
		inboxcloseclaim.BusinessTravel,
	}, inboxcloseclaim.BusinessLines())
}

// TestBusinessLinesIsACopy menjaga pemanggil tidak dapat mengubah daftar aslinya.
func TestBusinessLinesIsACopy(t *testing.T) {
	first := inboxcloseclaim.BusinessLines()
	first[0] = "DIUBAH"

	require.Equal(t, inboxcloseclaim.BusinessAll, inboxcloseclaim.BusinessLines()[0])
}

// TestTransferAndPaymentParsing menjaga ketiga nilai masing-masing penyaring.
func TestTransferAndPaymentParsing(t *testing.T) {
	transfer, known := inboxcloseclaim.ParseTransferStatus("sudah transfer")
	require.True(t, known)
	require.Equal(t, inboxcloseclaim.TransferDone, transfer)

	_, known = inboxcloseclaim.ParseTransferStatus("SUDAH")
	require.False(t, known, "nilai separuh tidak boleh diterima diam-diam")

	payment, known := inboxcloseclaim.ParsePaymentStatus("belum lunas")
	require.True(t, known)
	require.Equal(t, inboxcloseclaim.PaymentUnpaid, payment)

	payment, known = inboxcloseclaim.ParsePaymentStatus("")
	require.True(t, known)
	require.Equal(t, inboxcloseclaim.PaymentAny, payment)
}

// TestFilterNormalizeAppliesLimits menjaga batas paginasi dan nilai baku lini bisnis.
func TestFilterNormalizeAppliesLimits(t *testing.T) {
	// PageSize 25 diambil dari `.PageSize := 25` pada activity lama.
	require.Equal(t, 25, inboxcloseclaim.DefaultLimit)

	normalized := inboxcloseclaim.Filter{Limit: 0, Offset: -5}.Normalize()
	require.Equal(t, inboxcloseclaim.DefaultLimit, normalized.Limit)
	require.Equal(t, 0, normalized.Offset)
	require.Equal(t, inboxcloseclaim.BusinessAll, normalized.Business)

	tooBig := inboxcloseclaim.Filter{Limit: 5000}.Normalize()
	require.Equal(t, inboxcloseclaim.MaxLimit, tooBig.Limit)

	trimmed := inboxcloseclaim.Filter{
		Search:       "  PNC-1  ",
		PolicyNumber: "  POL-9  ",
		ClaimNumber:  "  PNCN  ",
		TechnicalPIC: "  BUDI  ",
	}.Normalize()
	require.Equal(t, "PNC-1", trimmed.Search)
	require.Equal(t, "POL-9", trimmed.PolicyNumber)
	require.Equal(t, "PNCN", trimmed.ClaimNumber)
	require.Equal(t, "BUDI", trimmed.TechnicalPIC)
}

// TestIsGCNMUserSelaluSalah mengunci keadaan yang diputuskan Work Owner 2026-09-23.
//
// Penjaga pengajuan ReOpen dan Copy Klaim adalah When rule `IsGCNMUser`, dan isinya
// `compareTwoValues(1, "=", 2)` — sebuah kondisi yang tidak pernah benar.
//
// Uji ini SENGAJA menegaskan sesuatu yang tampak seperti cacat. Alasannya: keadaan ini adalah
// keputusan, bukan kelalaian, dan tanpa uji yang menyebutkannya seseorang akan
// "memperbaikinya" menjadi `true` karena mengira menemukan bug.
//
// Bila Work Owner kelak membuka gerbangnya, uji INI yang gagal lebih dulu — dan
// kegagalannya adalah pengingat bahwa keputusannya berubah, bukan bahwa kodenya rusak.
// Hapus uji ini bersama perubahan itu, jangan sebelumnya.
func TestIsGCNMUserSelaluSalah(t *testing.T) {
	require.False(t, inboxcloseclaim.EvaluateIsGCNMUser(),
		"`When/IsGCNMUser-When.xml` berisi 1 = 2 — ia tidak pernah benar")

	require.False(t, inboxcloseclaim.CanRequestAction(),
		"pengajuan ReOpen dan Copy Klaim tertutup untuk setiap pengguna")

	require.NotEmpty(t, inboxcloseclaim.AlasanTidakBolehMengajukan,
		"penolakan wajib menyertakan alasan yang dapat dibaca pengguna")
}
