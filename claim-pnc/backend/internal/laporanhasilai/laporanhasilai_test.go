package laporanhasilai_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/laporanhasilai"
)

func day(year int, month time.Month, date int) time.Time {
	return time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
}

// TestCommitteeLabelFollowsTheLegacyCase membuktikan ketiga kode diterjemahkan persis
// seperti `CASE` pada `RDB List/CountAIDiterima_SQL-SQL.xml`.
func TestCommitteeLabelFollowsTheLegacyCase(t *testing.T) {
	require.Equal(t, "DITERIMA", laporanhasilai.CommitteeLabel("1"))
	require.Equal(t, "DITOLAK", laporanhasilai.CommitteeLabel("2"))
	require.Equal(t, "MENUNGGU", laporanhasilai.CommitteeLabel("0"))
}

// TestUnknownCommitteeCodeStaysBlank membuktikan cabang `ELSE ''` dipertahankan.
//
// Kode yang tidak dikenal menghasilkan sel kosong, bukan "DITOLAK". Modul `komite` memang
// MENOLAK meniru pola `ELSE 'DITOLAK'` dari rule lain — tetapi kueri layar ini sejak awal
// sudah menutup dengan `ELSE ''`, dan itulah yang ditiru di sini.
func TestUnknownCommitteeCodeStaysBlank(t *testing.T) {
	require.Equal(t, "", laporanhasilai.CommitteeLabel(""))
	require.Equal(t, "", laporanhasilai.CommitteeLabel("9"))
	require.Equal(t, "", laporanhasilai.CommitteeLabel("DITERIMA"))
}

// TestCommitteeLabelIgnoresPadding membuktikan spasi pada kolom CHAR tidak menggagalkannya.
func TestCommitteeLabelIgnoresPadding(t *testing.T) {
	require.Equal(t, "DITERIMA", laporanhasilai.CommitteeLabel("  1 "))
}

// TestTotalExcludesPending mengunci arti kolom "Total" pada grid ringkasan.
//
// Ia `Local.terima + Local.tolak`, BUKAN jumlah baris. Baris yang menunggu tidak ikut, dan
// itu ditiru apa adanya dari `Activity/SearchDataLaporanAI-Act.xml`.
func TestTotalExcludesPending(t *testing.T) {
	tally := laporanhasilai.Tally{
		Subject:  laporanhasilai.SubjectAI,
		Accepted: 30,
		Rejected: 12,
		Pending:  5,
	}

	require.Equal(t, 42, tally.Total(), "Total seharusnya Diterima + Ditolak")
	require.Equal(t, 47, tally.Rows(), "jumlah baris seharusnya menyertakan yang menunggu")
	require.NotEqual(t, tally.Rows(), tally.Total(),
		"selisih inilah yang membuat kolom Menunggu dibutuhkan")
}

// TestSummaryOrderPutsCommitteeFirst membuktikan urutan baris ringkasan tidak berubah.
//
// Activity lama menambahkan "Komite" lebih dulu lewat `<APPEND>`, baru "AI". Mengubahnya
// berarti layar baru berbeda dari layar yang sudah dihafal penggunanya (`D-13`).
func TestSummaryOrderPutsCommitteeFirst(t *testing.T) {
	summary := laporanhasilai.Summary{
		Committee: laporanhasilai.Tally{Subject: laporanhasilai.SubjectCommittee},
		AI:        laporanhasilai.Tally{Subject: laporanhasilai.SubjectAI},
	}

	tallies := summary.Tallies()
	require.Len(t, tallies, 2)
	require.Equal(t, "Komite", tallies[0].Subject)
	require.Equal(t, "AI", tallies[1].Subject)
}

// TestBothDatesAreRequired membuktikan kedua isian wajib, dan keduanya dilaporkan SEKALIGUS.
//
// Layar lama tidak dapat berjalan tanpanya — isian kosong menghasilkan
// `to_date('//','dd/mm/yyyy')` yang ditolak Oracle. Yang berubah hanyalah bentuk
// penolakannya. Kedua pesan datang bersamaan, meniru Pega yang menampilkan seluruh pesan
// sekaligus (`P-5`).
func TestBothDatesAreRequired(t *testing.T) {
	err := laporanhasilai.Filter{}.Validate()
	require.Error(t, err)

	var validation *laporanhasilai.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violations, 2, "kedua pelanggaran harus dilaporkan sekaligus")

	field := []string{validation.Violations[0].Field, validation.Violations[1].Field}
	require.ElementsMatch(t, []string{"dari", "sampai"}, field)
}

// TestOneMissingDateReportsOnlyThatOne membuktikan pesannya menunjuk isian yang benar.
func TestOneMissingDateReportsOnlyThatOne(t *testing.T) {
	err := laporanhasilai.Filter{From: day(2026, time.September, 1)}.Validate()

	var validation *laporanhasilai.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violations, 1)
	require.Equal(t, "sampai", validation.Violations[0].Field)
}

// TestReversedRangeIsRejected membuktikan rentang terbalik ditolak.
func TestReversedRangeIsRejected(t *testing.T) {
	err := laporanhasilai.Filter{
		From: day(2026, time.September, 10),
		To:   day(2026, time.September, 1),
	}.Validate()

	var validation *laporanhasilai.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violations, 1)
	require.Equal(t, "sampai", validation.Violations[0].Field)
}

// TestReversedRangeIsNotReportedWhenADateIsMissing membuktikan pesan ketiga tidak muncul.
//
// Meminta pengguna membetulkan urutan tanggal yang salah satunya belum ada adalah pesan
// yang tidak dapat ditindaklanjuti siapa pun.
func TestReversedRangeIsNotReportedWhenADateIsMissing(t *testing.T) {
	err := laporanhasilai.Filter{To: day(2026, time.September, 1)}.Validate()

	var validation *laporanhasilai.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violations, 1)
	require.Equal(t, "dari", validation.Violations[0].Field)
}

// TestSameDayRangeIsAccepted membuktikan satu hari penuh adalah rentang yang sah.
//
// Pengguna yang ingin melihat satu hari saja mengisi tanggal yang sama pada kedua isian,
// dan itu harus berhasil — bukan ditolak sebagai rentang kosong.
func TestSameDayRangeIsAccepted(t *testing.T) {
	filter := laporanhasilai.Filter{
		From: day(2026, time.September, 3),
		To:   day(2026, time.September, 3),
	}
	require.NoError(t, filter.Validate())
	require.Equal(t, day(2026, time.September, 4), filter.ToExclusive(),
		"batas atas satu hari penuh seharusnya tengah malam hari berikutnya")
}

// TestCleanDropsTheClockWithoutShiftingTheDay membuktikan jam dibuang tanpa menggeser
// tanggalnya.
//
// Ini yang menjaga `R-12` tidak lahir kembali: kolom pembandingnya `DATE` Oracle — jam
// dinding tanpa zona — dan mengonversinya akan menggeser tanggal sehari pada sebagian
// nilai. Tidak ada penambahan tujuh jam di mana pun.
func TestCleanDropsTheClockWithoutShiftingTheDay(t *testing.T) {
	// Pukul 23:30 di zona +07:00 — nilai yang akan mundur sehari bila dikonversi ke UTC.
	wib := time.FixedZone("WIB", 7*60*60)
	filter := laporanhasilai.Filter{
		From: time.Date(2026, time.September, 3, 23, 30, 0, 0, wib),
		To:   time.Date(2026, time.September, 4, 23, 30, 0, 0, wib),
	}.Clean()

	require.Equal(t, day(2026, time.September, 3), filter.From)
	require.Equal(t, day(2026, time.September, 4), filter.To)
}

// TestCleanKeepsZeroAsZero membuktikan tanggal yang belum diisi tidak berubah menjadi
// tanggal nol tahun 1.
//
// Tanpa ini, isian kosong akan lolos Validate sebagai tanggal yang "ada" dan membawa
// seluruh tabel ke dalam rentangnya.
func TestCleanKeepsZeroAsZero(t *testing.T) {
	clean := laporanhasilai.Filter{}.Clean()
	require.True(t, clean.From.IsZero())
	require.True(t, clean.To.IsZero())
}

// TestPaginationNormalizesInsteadOfRejecting membuktikan nilai di luar rentang dibetulkan.
func TestPaginationNormalizesInsteadOfRejecting(t *testing.T) {
	clean := laporanhasilai.Pagination{Page: 0, Size: 0}.Normalize()
	require.Equal(t, 1, clean.Page)
	require.Equal(t, laporanhasilai.DefaultPageSize, clean.Size)

	require.Equal(t, laporanhasilai.MaxPageSize,
		laporanhasilai.Pagination{Page: 1, Size: 5000}.Normalize().Size,
		"ukuran halaman di atas batas seharusnya dibetulkan, bukan dipenuhi")
}

// TestOffsetCountsFromZero membuktikan halaman pertama tidak melewati satu baris pun.
func TestOffsetCountsFromZero(t *testing.T) {
	require.Equal(t, 0, laporanhasilai.Pagination{Page: 1, Size: 50}.Offset())
	require.Equal(t, 50, laporanhasilai.Pagination{Page: 2, Size: 50}.Offset())
	require.Equal(t, 0, laporanhasilai.Pagination{Page: -3, Size: 50}.Offset())
}
