package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
	"claim-pnc/internal/reportkpi/repo/memory"
)

// maret adalah periode yang memuat seluruh baris contoh KECUALI dua yang sengaja di luar.
func maret(t *testing.T, tipe reportkpi.ReportType) reportkpi.Query {
	t.Helper()

	query, err := reportkpi.NewQuery(reportkpi.QueryInput{
		ReportType: string(tipe),
		From:       "2026-03-01",
		To:         "2026-03-31",
	}, reportkpi.Caller{Login: "PENYELIACONTOH"})
	require.NoError(t, err)

	return query
}

func allPage() reportkpi.Pagination {
	return reportkpi.Pagination{Page: 1, Size: reportkpi.MaxPageSize}
}

// Rata-ratanya benar, dan pembulatannya dua desimal seperti `ROUND(AVG(...),2)`.
//
// Adjuster contoh pertama punya dua kasus OUTSTANDING bernilai 3 dan 4 pada komponen
// pertama, sehingga rata-ratanya 3,5.
func TestRingkasanMerataRatakanSeluruhKasusAdjuster(t *testing.T) {
	store := memory.NewSampleStore()

	rows, err := store.Summary(context.Background(), maret(t, reportkpi.TypeOutstanding))
	require.NoError(t, err)

	var found bool
	for _, row := range rows {
		if row.Adjuster != "PT ADJUSTER NUSA CONTOH" {
			continue
		}
		found = true
		require.Equal(t, reportkpi.TypeOutstanding, row.ReportType)
		require.True(t, row.Scores[reportkpi.ComponentSurvey].Present)
		require.InDelta(t, 3.5, row.Scores[reportkpi.ComponentSurvey].Value, 0.001)
	}
	require.True(t, found, "adjuster contoh pertama tidak ditemukan")
}

// Tipe ALL menghasilkan DUA baris untuk adjuster yang punya kedua tipe.
//
// Itu bukan baris ganda: kueri lamanya memang menggabungkan dua kelompok dengan
// `UNION ALL`. Uji ini menjaga perilaku itu tetap ada di pengisi memori pula, supaya layar
// diuji terhadap bentuk yang sama dengan produksi.
func TestTipeAllMenghasilkanSatuBarisPerTipe(t *testing.T) {
	store := memory.NewSampleStore()

	rows, err := store.Summary(context.Background(), maret(t, reportkpi.TypeAll))
	require.NoError(t, err)

	tipe := map[reportkpi.ReportType]int{}
	for _, row := range rows {
		if row.Adjuster == "PT ADJUSTER NUSA CONTOH" {
			tipe[row.ReportType]++
		}
	}
	require.Equal(t, 1, tipe[reportkpi.TypeOutstanding])
	require.Equal(t, 1, tipe[reportkpi.TypeFinal])
}

// Komponen yang tidak pernah dinilai menghasilkan nilai KOSONG, bukan nol.
//
// Adjuster contoh kedua tidak punya nilai INTERIM REPORT sama sekali. Rata-rata yang
// mengembalikan 0 di sini akan membuat adjuster itu terlihat mendapat nilai terburuk,
// padahal komponennya belum dinilai.
func TestKomponenTanpaNilaiTidakMenjadiNol(t *testing.T) {
	store := memory.NewSampleStore()

	rows, err := store.Summary(context.Background(), maret(t, reportkpi.TypeOutstanding))
	require.NoError(t, err)

	for _, row := range rows {
		if row.Adjuster != "CV SURVEI CONTOH SEJAHTERA" {
			continue
		}
		require.False(t, row.Scores[reportkpi.ComponentInterimReport].Present,
			"komponen yang belum dinilai tidak boleh punya nilai")
		require.True(t, row.Scores[reportkpi.ComponentSurvey].Present,
			"komponen lain pada baris yang sama tetap terhitung")
	}
}

// Nilai yang bukan angka diabaikan dari rata-rata, TETAPI barisnya tetap ikut.
//
// Adjuster contoh ketiga punya satu kasus dengan PRELIMINARY ADVICE berisi "N/A".
// Kedelapan komponen lainnya harus tetap terhitung — membuang seluruh barisnya akan
// menghapus penilaian yang sah karena satu sel yang rusak.
func TestNilaiBukanAngkaTidakMembuangBarisnya(t *testing.T) {
	store := memory.NewSampleStore()

	rows, err := store.Summary(context.Background(), maret(t, reportkpi.TypeFinal))
	require.NoError(t, err)

	var found bool
	for _, row := range rows {
		if row.Adjuster != "PT PENILAI CONTOH PRATAMA" {
			continue
		}
		found = true
		require.False(t, row.Scores[reportkpi.ComponentPreliminaryAdvice].Present)
		require.True(t, row.Scores[reportkpi.ComponentSurvey].Present)
		require.InDelta(t, 5, row.Scores[reportkpi.ComponentSurvey].Value, 0.001)
	}
	require.True(t, found, "baris ber-nilai rusak seharusnya tetap muncul")
}

// Batas atas periode IKUT terhitung.
//
// Ini cacat setengah-terbuka yang paling sering lolos uji: penyaring yang menulis
// `< sampai` alih-alih `< sampai + 1 hari` membuang seluruh baris pada hari terakhir tanpa
// satu pun galat. Baris contoh sengaja diletakkan pada 1 dan 31 Maret.
func TestKeduaTepiPeriodeIkutTerhitung(t *testing.T) {
	store := memory.NewSampleStore()

	result, err := store.Detail(context.Background(), maret(t, reportkpi.TypeFinal), allPage())
	require.NoError(t, err)

	tanggal := map[string]bool{}
	for _, row := range result.Rows {
		tanggal[row.ScoredOn] = true
	}
	require.True(t, tanggal["2026-03-01"], "batas bawah periode tidak ikut terhitung")
	require.True(t, tanggal["2026-03-31"], "batas atas periode tidak ikut terhitung")
}

// Penyaring periode BENAR-BENAR menyaring.
//
// Dua baris contoh sengaja berada sehari sebelum dan sehari sesudah Maret. Tanpa keduanya,
// penyaring yang tidak bekerja sama sekali tidak dapat dibedakan dari penyaring yang
// bekerja.
func TestBarisDiLuarPeriodeTidakIkut(t *testing.T) {
	store := memory.NewSampleStore()

	result, err := store.Detail(context.Background(), maret(t, reportkpi.TypeFinal), allPage())
	require.NoError(t, err)

	for _, row := range result.Rows {
		require.NotEqual(t, "PT LUAR CONTOH PERIODE", row.Adjuster,
			"baris di luar periode ikut terbawa")
	}
}

func TestPenyaringAdjusterMempersempitHasil(t *testing.T) {
	store := memory.NewSampleStore()

	query := maret(t, reportkpi.TypeFinal)
	query.Adjuster = "PT TEPI CONTOH MANDIRI"

	result, err := store.Detail(context.Background(), query, allPage())
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	for _, row := range result.Rows {
		require.Equal(t, "PT TEPI CONTOH MANDIRI", row.Adjuster)
	}
}

// Paginasi memotong TANPA mengubah jumlah seluruhnya.
//
// Total yang ikut terpotong membuat layar menampilkan jumlah halaman yang salah, dan
// pengguna berhenti di halaman pertama mengira itu seluruhnya.
func TestPaginasiMemotongTanpaMengubahTotal(t *testing.T) {
	store := memory.NewSampleStore()
	query := maret(t, reportkpi.TypeAll)

	seluruhnya, err := store.Detail(context.Background(), query, allPage())
	require.NoError(t, err)
	require.Greater(t, seluruhnya.Total, 2)

	halamanPertama, err := store.Detail(context.Background(), query,
		reportkpi.Pagination{Page: 1, Size: 2})
	require.NoError(t, err)
	require.Len(t, halamanPertama.Rows, 2)
	require.Equal(t, seluruhnya.Total, halamanPertama.Total)

	halamanKedua, err := store.Detail(context.Background(), query,
		reportkpi.Pagination{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, seluruhnya.Total, halamanKedua.Total)
	require.NotEqual(t, halamanPertama.Rows[0].CaseID, halamanKedua.Rows[0].CaseID)
}

// Halaman di luar jangkauan menghasilkan daftar kosong, BUKAN panik.
//
// Nomor halaman datang dari parameter query dan mudah diketik melewati batas.
func TestHalamanDiLuarJangkauanKosongTanpaPanik(t *testing.T) {
	store := memory.NewSampleStore()

	result, err := store.Detail(context.Background(), maret(t, reportkpi.TypeAll),
		reportkpi.Pagination{Page: 999, Size: 10})
	require.NoError(t, err)
	require.Empty(t, result.Rows)
	require.Greater(t, result.Total, 0, "total tetap menyebut jumlah yang sebenarnya")
}

// Daftar adjuster TIDAK disaring oleh adjuster yang sedang dipilih.
//
// Kalau disaring, dropdown akan menyisakan satu pilihan saja dan pengguna tidak dapat
// berpindah adjuster lagi.
func TestDaftarAdjusterTidakMenyempitKarenaPilihanYangSedangAktif(t *testing.T) {
	store := memory.NewSampleStore()

	query := maret(t, reportkpi.TypeAll)
	query.Adjuster = "PT TEPI CONTOH MANDIRI"

	names, err := store.Adjusters(context.Background(), query)
	require.NoError(t, err)
	require.Greater(t, len(names), 1,
		"daftar pilihan tidak boleh menyempit karena pilihan yang sedang aktif")
}

// Daftar adjuster IKUT menyempit mengikuti periode.
//
// Itulah yang menjaga janji pada selisih terencana: setiap pilihan yang muncul pasti
// menghasilkan baris.
func TestDaftarAdjusterMengikutiPeriode(t *testing.T) {
	store := memory.NewSampleStore()

	names, err := store.Adjusters(context.Background(), maret(t, reportkpi.TypeAll))
	require.NoError(t, err)
	require.NotContains(t, names, "PT LUAR CONTOH PERIODE")
}

// Penyimpanan kosong menghasilkan daftar KOSONG, bukan nil.
//
// `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua memaksa setiap layar
// memeriksanya lebih dulu.
func TestPenyimpananKosongMenghasilkanDaftarKosong(t *testing.T) {
	store := memory.NewStore()
	query := maret(t, reportkpi.TypeAll)

	ringkasan, err := store.Summary(context.Background(), query)
	require.NoError(t, err)
	require.NotNil(t, ringkasan)
	require.Empty(t, ringkasan)

	rincian, err := store.Detail(context.Background(), query, allPage())
	require.NoError(t, err)
	require.NotNil(t, rincian.Rows)
	require.Empty(t, rincian.Rows)
	require.Zero(t, rincian.Total)

	names, err := store.Adjusters(context.Background(), query)
	require.NoError(t, err)
	require.NotNil(t, names)
	require.Empty(t, names)
}

// Rincian memuat kesembilan kode komponen sebagai kunci, meski nilainya kosong.
//
// Layar menggambar kolomnya dari metadata; kunci yang hilang membuat sel-nya kosong tanpa
// dapat dibedakan dari komponen yang memang belum dinilai.
func TestSetiapBarisMemuatKesembilanKodeKomponen(t *testing.T) {
	store := memory.NewSampleStore()

	result, err := store.Detail(context.Background(), maret(t, reportkpi.TypeAll), allPage())
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	for _, row := range result.Rows {
		for _, code := range reportkpi.ComponentCodes() {
			_, exists := row.Scores[code]
			require.Truef(t, exists, "baris %s tidak memuat komponen %s", row.CaseID, code)
		}
	}
}
