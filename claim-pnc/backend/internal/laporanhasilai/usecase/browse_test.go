package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/laporanhasilai"
	"claim-pnc/internal/laporanhasilai/repo/memory"
	"claim-pnc/internal/laporanhasilai/usecase"
)

func day(year int, month time.Month, date int) time.Time {
	return time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
}

// wholeSeptember adalah rentang yang mencakup seluruh baris contoh tahun 2026.
func wholeSeptember() laporanhasilai.Filter {
	return laporanhasilai.Filter{
		From: day(2026, time.September, 1),
		To:   day(2026, time.September, 30),
	}
}

func serviceWithSample(t *testing.T) *usecase.Service {
	t.Helper()

	store := memory.NewRepo(memory.SampleRows()...)
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (laporanhasilai.Repo, error) {
			if alias != "asm" {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return service
}

// TestNewServiceRejectsMissingSelector membuktikan rakitan setengah jadi gagal saat start.
func TestNewServiceRejectsMissingSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// TestPortalIsCheckedBeforeTheDates mengunci urutan pemeriksaan.
//
// Permintaan tanpa portal yang sah harus ditolak sebagai soal PORTAL, bukan soal isian
// tanggal. Menukarnya akan membuat pengguna sibuk membetulkan tanggal yang sudah benar —
// dan pada aplikasi empat badan hukum, salah portal bukan kesalahan kecil (`R-20`).
func TestPortalIsCheckedBeforeTheDates(t *testing.T) {
	service := serviceWithSample(t)

	_, err := service.Search(context.Background(), "entitas-lain",
		laporanhasilai.Filter{}, laporanhasilai.Pagination{})

	require.Error(t, err)

	var validation *laporanhasilai.ValidationError
	require.False(t, errors.As(err, &validation),
		"galat portal seharusnya tidak menyamar sebagai galat isian")
}

// TestSearchRejectsEmptyDates membuktikan validasi berjalan di lapisan ini, sekali.
func TestSearchRejectsEmptyDates(t *testing.T) {
	service := serviceWithSample(t)

	_, err := service.Search(context.Background(), "asm",
		laporanhasilai.Filter{}, laporanhasilai.Pagination{})

	var validation *laporanhasilai.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violations, 2)
}

// TestExportRejectsEmptyDatesToo membuktikan ekspor menempuh validasi yang SAMA.
//
// Bila validasinya hanya dipasang di jalur layar, tombol Export akan menjadi pintu belakang
// yang menembak basis data tanpa penyaring apa pun.
func TestExportRejectsEmptyDatesToo(t *testing.T) {
	service := serviceWithSample(t)

	_, err := service.ListForExport(context.Background(), "asm",
		laporanhasilai.Filter{}, laporanhasilai.Pagination{})

	var validation *laporanhasilai.ValidationError
	require.True(t, errors.As(err, &validation))
}

// TestRowsOutsideTheGateAreExcluded membuktikan ketiga penyaring benar-benar menggigit.
//
// Tiga baris contoh sengaja dibuat untuk HILANG: satu ber-TANGGALKOMITE kosong, satu yang
// kasusnya belum `Resolved-Completed`, dan satu di luar rentang tanggal. Tanpa uji ini,
// penyaring yang lupa tidak akan pernah ketahuan — barisnya hanya "muncul", dan tidak ada
// yang tampak salah.
func TestRowsOutsideTheGateAreExcluded(t *testing.T) {
	service := serviceWithSample(t)

	result, err := service.Search(context.Background(), "asm", wholeSeptember(),
		laporanhasilai.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	require.Equal(t, 6, result.Page.Total,
		"tiga baris contoh seharusnya tersaring keluar")

	for _, row := range result.Page.Rows {
		require.NotEqual(t, "PNCN.26.0005", row.ClaimNumber, "baris tanpa tanggal komite lolos")
		require.NotEqual(t, "PNCN.26.0006", row.ClaimNumber, "baris kasus belum selesai lolos")
		require.NotEqual(t, "PNCN.25.0099", row.ClaimNumber, "baris di luar rentang lolos")
	}
}

// TestClaimNumberIsBlankOnLaterCommitteeSteps mengunci keputusan Work Owner 2026-09-26.
//
// Padanan `CASE WHEN B.KOMITEKE = '1' THEN B.NO_KLAIM ELSE '' END`.
func TestClaimNumberIsBlankOnLaterCommitteeSteps(t *testing.T) {
	service := serviceWithSample(t)

	result, err := service.Search(context.Background(), "asm", wholeSeptember(),
		laporanhasilai.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	// Baris ketiga pada urutan KOMITE_ID, KOMITEKE adalah jenjang kedua KMT-000101.
	require.GreaterOrEqual(t, len(result.Page.Rows), 3)
	third := result.Page.Rows[2]
	require.Equal(t, "KMT-000101|2|OBJ-01|CVG-01", third.ID)
	require.Equal(t, "", third.ClaimNumber, "nomor klaim seharusnya dikosongkan")
}

// TestOneRowPerAIAssessment membuktikan satu jenjang komite dapat menghasilkan beberapa
// baris — satu per objek pertanggungan.
//
// Ini yang membedakan layar ini dari inbox Komite, yang justru MENGAGREGASI tabel yang sama
// supaya barisnya tidak berganda. Keduanya benar untuk dirinya sendiri.
func TestOneRowPerAIAssessment(t *testing.T) {
	service := serviceWithSample(t)

	result, err := service.Search(context.Background(), "asm", wholeSeptember(),
		laporanhasilai.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	require.Equal(t, "KMT-000101|1|OBJ-01|CVG-01", result.Page.Rows[0].ID)
	require.Equal(t, "KMT-000101|1|OBJ-02|CVG-01", result.Page.Rows[1].ID)
	require.Equal(t, "DITERIMA", result.Page.Rows[0].AIStatus)
	require.Equal(t, "DITOLAK", result.Page.Rows[1].AIStatus)
}

// TestSummaryCountsTheWholeFilterNotThePage adalah uji terpenting di berkas ini.
//
// Activity lama mencacah dengan menelusuri seluruh hasil yang sudah ada di klipboard. Di
// sini barisnya dipaginasi, sehingga pencacahan yang ikut dipaginasi akan menghitung satu
// halaman dan menyebutnya total — dan angkanya akan BERUBAH setiap pengguna berpindah
// halaman, tanpa satu pun di layar yang menjelaskan kenapa.
func TestSummaryCountsTheWholeFilterNotThePage(t *testing.T) {
	service := serviceWithSample(t)

	firstPage, err := service.Search(context.Background(), "asm", wholeSeptember(),
		laporanhasilai.Pagination{Page: 1, Size: 2})
	require.NoError(t, err)
	require.Len(t, firstPage.Page.Rows, 2, "halaman seharusnya memuat dua baris")

	wholeSet, err := service.Search(context.Background(), "asm", wholeSeptember(),
		laporanhasilai.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	require.Equal(t, wholeSet.Summary, firstPage.Summary,
		"ringkasan seharusnya tidak bergantung pada halaman yang sedang terbuka")
}

// TestSummaryNumbersFollowTheLegacyRule mengunci arti setiap angka ringkasan.
//
// Enam baris lolos penyaring:
//
//	AI       DITERIMA 3 · DITOLAK 2 · kosong 1
//	Komite   kode 1 → 4 · kode 2 → 1 · kode 0 → 1
func TestSummaryNumbersFollowTheLegacyRule(t *testing.T) {
	service := serviceWithSample(t)

	result, err := service.Search(context.Background(), "asm", wholeSeptember(),
		laporanhasilai.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	ai := result.Summary.AI
	require.Equal(t, "AI", ai.Subject)
	require.Equal(t, 3, ai.Accepted)
	require.Equal(t, 2, ai.Rejected)
	require.Equal(t, 1, ai.Pending, "baris ber-AI Status kosong masuk Menunggu")
	require.Equal(t, 5, ai.Total(), "Total tidak menyertakan yang menunggu")

	committee := result.Summary.Committee
	require.Equal(t, "Komite", committee.Subject)
	require.Equal(t, 4, committee.Accepted)
	require.Equal(t, 1, committee.Rejected)
	require.Equal(t, 1, committee.Pending)
	require.Equal(t, 5, committee.Total())

	require.Equal(t, result.Page.Total, ai.Rows(),
		"pencacah ringkasan dan pencacah paginasi harus membaca himpunan yang sama")
	require.Equal(t, result.Page.Total, committee.Rows())
}

// TestFiveColumnsStayEmpty mengunci keputusan Work Owner 2026-09-26.
//
// Kelimanya kosong karena kuerinya memang tidak memilih kolomnya — persis seperti Pega.
// Uji ini bukan untuk melarang selamanya; ia untuk memastikan perubahannya DISENGAJA.
func TestFiveColumnsStayEmpty(t *testing.T) {
	service := serviceWithSample(t)

	result, err := service.Search(context.Background(), "asm", wholeSeptember(),
		laporanhasilai.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	for _, row := range result.Page.Rows {
		require.Empty(t, row.ObjectName, "Object Name seharusnya kosong")
		require.Empty(t, row.AcceptNote, "Note AI Terima seharusnya kosong")
		require.Empty(t, row.RejectNote, "Note AI Tolak seharusnya kosong")
		require.Empty(t, row.CoverageFinal, "Coverage Final seharusnya kosong")
		require.Empty(t, row.ChronologyCategory, "Kategori Kronologi seharusnya kosong")
	}
}

// TestPageBeyondRangeIsEmptyButKeepsTheTotal membuktikan halaman di luar jangkauan tidak
// menjadi galat.
//
// Nomor halaman datang dari tautan paginasi, dan tautan yang basi bukan kesalahan yang
// dapat diperbaiki pengguna dengan mengetik. Totalnya tetap dikirim supaya paginator dapat
// mengembalikan pengguna ke halaman yang ada.
func TestPageBeyondRangeIsEmptyButKeepsTheTotal(t *testing.T) {
	service := serviceWithSample(t)

	result, err := service.Search(context.Background(), "asm", wholeSeptember(),
		laporanhasilai.Pagination{Page: 99, Size: 50})
	require.NoError(t, err)

	require.Empty(t, result.Page.Rows)
	require.Equal(t, 6, result.Page.Total)
}

// TestNarrowRangeExcludesTheEdges membuktikan rentangnya benar-benar menyaring.
//
// Rentang 3–5 September memuat empat baris: dua objek pada 3 September dan satu jenjang
// kedua pada 5 September. Baris 8 September dan sesudahnya harus hilang.
func TestNarrowRangeExcludesTheEdges(t *testing.T) {
	service := serviceWithSample(t)

	result, err := service.Search(context.Background(), "asm", laporanhasilai.Filter{
		From: day(2026, time.September, 3),
		To:   day(2026, time.September, 5),
	}, laporanhasilai.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	require.Equal(t, 3, result.Page.Total)
}

// TestUpperBoundIncludesTheWholeLastDay membuktikan batas atas tidak memotong hari terakhir.
//
// Inilah yang dijaga `Filter.ToExclusive`. Bila batas atasnya dikirim apa adanya, baris
// pada tanggal terakhir yang dipilih pengguna akan hilang — dan hilangnya senyap.
func TestUpperBoundIncludesTheWholeLastDay(t *testing.T) {
	service := serviceWithSample(t)

	result, err := service.Search(context.Background(), "asm", laporanhasilai.Filter{
		From: day(2026, time.September, 12),
		To:   day(2026, time.September, 12),
	}, laporanhasilai.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	require.Equal(t, 1, result.Page.Total, "baris pada tanggal batas atas seharusnya ikut")
}

// TestEnsurePortalReadySeparatesTwoFailures membuktikan kesiapan portal dapat diperiksa
// tanpa membaca apa pun.
func TestEnsurePortalReadySeparatesTwoFailures(t *testing.T) {
	service := serviceWithSample(t)

	require.NoError(t, service.EnsurePortalReady("asm"))
	require.Error(t, service.EnsurePortalReady("entitas-lain"))
}
