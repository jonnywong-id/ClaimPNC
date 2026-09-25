package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
	"claim-pnc/internal/reportkpi/repo/memory"
)

// maretAdmin adalah periode yang memuat seluruh baris contoh KECUALI yang sengaja di luar.
func maretAdmin(t *testing.T, group reportkpi.AdminGroup) reportkpi.AdminQuery {
	t.Helper()

	query, err := reportkpi.NewAdminQuery(reportkpi.AdminQueryInput{
		Group: string(group),
		From:  "2026-03-01",
		To:    "2026-03-31",
	}, reportkpi.Caller{Login: "PENYELIACONTOH"})
	require.NoError(t, err)

	return query
}

// metricOf mengambil satu metrik kartu skor menurut kodenya.
func metricOf(t *testing.T, card reportkpi.AdminScorecard, code string) reportkpi.Metric {
	t.Helper()

	for _, metric := range card.Metrics {
		if metric.Code == code {
			return metric
		}
	}
	t.Fatalf("metrik %q tidak ada di kartu skor", code)
	return reportkpi.Metric{}
}

// Kartu skor NON-MBU dihitung BERLAPIS, dan uji ini memeriksa SETIAP lapisnya.
//
// Memeriksa hasil akhir saja tidak cukup: dua kekeliruan yang saling menutup akan lolos,
// dan pada kartu penilaian kinerja itu berarti angka yang salah terlihat masuk akal.
//
// Baris contohnya menghasilkan, dan seluruhnya dapat dihitung dengan tangan:
//
//	leader   1 dari 4 melewati SLA -> 25%  -> nilai 0 -> subtotal  0
//	member   0 dari 3 melewati SLA ->  0%  -> nilai 5 -> subtotal 40
//	total 40 · pencapaian 40/((3/5)*90) = 0,74 -> TIDAK TERCAPAI TARGET
func TestKartuSkorNonMBUDihitungBerlapis(t *testing.T) {
	store := memory.NewSampleStore()
	query := maretAdmin(t, reportkpi.AdminGroupNonMBU)

	totals, err := store.AdminTotals(context.Background(), query)
	require.NoError(t, err)

	card := reportkpi.BuildScorecard(query.Group, query.Range, totals)

	require.InDelta(t, 1, metricOf(t, card, reportkpi.MetricLeaderOverSLA).Value.Value, 0.001)
	require.InDelta(t, 4, metricOf(t, card, reportkpi.MetricLeaderTotal).Value.Value, 0.001)
	require.InDelta(t, 25, metricOf(t, card, reportkpi.MetricLeaderPercent).Value.Value, 0.001)
	require.InDelta(t, 0, metricOf(t, card, reportkpi.MetricLeaderScore).Value.Value, 0.001)

	require.InDelta(t, 0, metricOf(t, card, reportkpi.MetricMemberOverSLA).Value.Value, 0.001)
	require.InDelta(t, 3, metricOf(t, card, reportkpi.MetricMemberTotal).Value.Value, 0.001)
	require.InDelta(t, 0, metricOf(t, card, reportkpi.MetricMemberPercent).Value.Value, 0.001)
	require.InDelta(t, 5, metricOf(t, card, reportkpi.MetricMemberScore).Value.Value, 0.001)

	require.InDelta(t, 0, metricOf(t, card, reportkpi.MetricLeaderSubtotal).Value.Value, 0.001)
	require.InDelta(t, 40, metricOf(t, card, reportkpi.MetricMemberSubtotal).Value.Value, 0.001)
	require.InDelta(t, 40, metricOf(t, card, reportkpi.MetricQuantitativeSum).Value.Value, 0.001)
	require.InDelta(t, 0.74, metricOf(t, card, reportkpi.MetricAchievementRatio).Value.Value, 0.001)

	require.Equal(t, reportkpi.AchievementMissed, card.Achievement)
}

// Ambang "melewati SLA" NON-MBU adalah `> 1`, bukan `>= 1`.
//
// Baris contoh pertama ber-TAT tepat 1,0 dan karena itu TIDAK terhitung melewati. Batas
// itulah yang paling mudah keliru dibaca, dan satu tanda yang tertukar menggeser
// persentase, nilai, subtotal, dan kesimpulannya sekaligus.
func TestAmbangSLANonMBUTidakMenghitungTepatSatuHari(t *testing.T) {
	store := memory.NewSampleStore()
	query := maretAdmin(t, reportkpi.AdminGroupNonMBU)

	totals, err := store.AdminTotals(context.Background(), query)
	require.NoError(t, err)

	// Empat klaim leader: TAT 1,0 · 0,5 · 0,25 · 2,5. Hanya yang 2,5 yang melewati.
	require.InDelta(t, 1, totals.LeaderOverSLA.Value, 0.001,
		"TAT tepat 1,0 hari kerja seharusnya TIDAK terhitung melewati SLA")
}

// Ambang PA berbeda: `> 0`, bukan `> 1`.
//
// Perbedaan itu ditiru dari kueri apa adanya. Menyamakannya dengan NON-MBU akan mengubah
// angka pada kartu skor PA.
func TestAmbangSLAPALebihKetatDaripadaNonMBU(t *testing.T) {
	store := memory.NewSampleStore()
	query := maretAdmin(t, reportkpi.AdminGroupPA)

	totals, err := store.AdminTotals(context.Background(), query)
	require.NoError(t, err)

	// Empat klaim PA pada Maret: TAT registrasi 0,5 · 0 · 3 · 0,25.
	// Tiga di antaranya > 0.
	require.InDelta(t, 4, totals.RegisterTotal.Value, 0.001)
	require.InDelta(t, 3, totals.RegisterOverSLA.Value, 0.001,
		"pada PA, TAT 0,5 hari kerja SUDAH terhitung melewati SLA")
}

// Pembagi "TOTAL KLAIM BAYAR" pada PA TIDAK mengikuti periode yang dipilih.
//
// Ia terkunci pada rentang 2023 di dalam kueri lama. Uji ini membuktikan keanehan itu
// direplikasi — dan sekaligus menjaganya tidak "diperbaiki" diam-diam.
func TestTotalKlaimBayarPATerkunciPadaRentang2023(t *testing.T) {
	store := memory.NewSampleStore()

	maret, err := store.AdminTotals(context.Background(), maretAdmin(t, reportkpi.AdminGroupPA))
	require.NoError(t, err)

	lain, err := reportkpi.NewAdminQuery(reportkpi.AdminQueryInput{
		Group: string(reportkpi.AdminGroupPA),
		From:  "2026-01-01",
		To:    "2026-01-31",
	}, reportkpi.Caller{Login: "PENYELIACONTOH"})
	require.NoError(t, err)

	januari, err := store.AdminTotals(context.Background(), lain)
	require.NoError(t, err)

	require.InDelta(t, 2, maret.PaymentTotal.Value, 0.001,
		"kedua baris contoh 2023 seharusnya terhitung")
	require.InDelta(t, maret.PaymentTotal.Value, januari.PaymentTotal.Value, 0.001,
		"TOTAL KLAIM BAYAR tidak boleh berubah ketika periode diganti")
}

// Klaim PA tanpa akseptasi ikut TOTAL KLAIM REGIST tetapi TIDAK ikut pencacah pembayaran.
func TestKlaimPATanpaAkseptasiTidakIkutPencacahPembayaran(t *testing.T) {
	store := memory.NewSampleStore()

	totals, err := store.AdminTotals(context.Background(), maretAdmin(t, reportkpi.AdminGroupPA))
	require.NoError(t, err)

	// Empat klaim Maret, satu tanpa akseptasi. Yang melewati SLA pembayaran: dua
	// (TAT 0,5 dan 4); yang TAT-nya 0 dan yang tanpa akseptasi tidak ikut.
	require.InDelta(t, 4, totals.RegisterTotal.Value, 0.001)
	require.InDelta(t, 2, totals.PaymentOverSLA.Value, 0.001)
}

// Periode tanpa satu pun klaim menghasilkan metrik KOSONG, bukan nol dan bukan panik.
//
// Di basis data keadaan itu menghasilkan `ORA-01476`. Di sini ia ditahan, dan hasilnya
// dinyatakan tidak ada — sehingga layar dapat menggambarkan keadaannya.
func TestPeriodeTanpaKlaimMenghasilkanMetrikKosong(t *testing.T) {
	store := memory.NewSampleStore()

	kosong, err := reportkpi.NewAdminQuery(reportkpi.AdminQueryInput{
		Group: string(reportkpi.AdminGroupNonMBU),
		From:  "2020-01-01",
		To:    "2020-01-31",
	}, reportkpi.Caller{Login: "PENYELIACONTOH"})
	require.NoError(t, err)

	totals, err := store.AdminTotals(context.Background(), kosong)
	require.NoError(t, err)

	card := reportkpi.BuildScorecard(kosong.Group, kosong.Range, totals)

	require.False(t, metricOf(t, card, reportkpi.MetricLeaderPercent).Value.Present,
		"persentase tanpa klaim tidak dapat dihitung")
	require.False(t, metricOf(t, card, reportkpi.MetricAchievementRatio).Value.Present)
	require.Empty(t, card.Achievement,
		"tanpa rasio, kesimpulan tercapai/tidak tidak boleh dikarang")

	// Cacahnya tetap ADA dan bernilai nol — nol klaim adalah fakta, bukan ketiadaan.
	require.True(t, metricOf(t, card, reportkpi.MetricLeaderTotal).Value.Present)
	require.InDelta(t, 0, metricOf(t, card, reportkpi.MetricLeaderTotal).Value.Value, 0.001)
}

// Kartu skor PA TIDAK punya baris kesimpulan.
//
// Kuerinya memang tidak menghitungnya. Mengarangnya akan menampilkan penilaian yang tidak
// pernah dibuat Pega.
func TestKartuSkorPATidakPunyaKesimpulan(t *testing.T) {
	store := memory.NewSampleStore()
	query := maretAdmin(t, reportkpi.AdminGroupPA)

	totals, err := store.AdminTotals(context.Background(), query)
	require.NoError(t, err)

	card := reportkpi.BuildScorecard(query.Group, query.Range, totals)
	require.Empty(t, card.Achievement)
	require.Len(t, card.Metrics, 6)
}

// Identitas koordinator mengikuti ACTIVITY, bukan teks kueri.
//
// Keduanya menyebut nama yang berbeda pada kelompok NON-MBU, dan activity berjalan
// belakangan — sehingga itulah yang selama ini dilihat pengguna (`P-5`).
func TestNamaKoordinatorMengikutiActivityBukanTeksKueri(t *testing.T) {
	card := reportkpi.BuildScorecard(
		reportkpi.AdminGroupNonMBU,
		reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"},
		reportkpi.AdminTotals{},
	)

	require.Equal(t, "MORASOTARDODOTARIGAN", card.Identity.Coordinator)
	require.NotEqual(t, reportkpi.CoordinatorInQuery, card.Identity.Coordinator)
	require.Equal(t, "96030583", card.Identity.NIK)
}

// TANGGAL EFEKTIF disusun Go dari rentang periode, berbentuk `dd/mm/yyyy - dd/mm/yyyy`.
func TestTanggalEfektifDisusunDariPeriode(t *testing.T) {
	card := reportkpi.BuildScorecard(
		reportkpi.AdminGroupPA,
		reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"},
		reportkpi.AdminTotals{},
	)
	require.Equal(t, "01/03/2026 - 31/03/2026", card.EffectiveOn)
}

// Grid rincian dipaginasi, dan totalnya tidak ikut terpotong.
func TestRincianAdminDipaginasi(t *testing.T) {
	store := memory.NewSampleStore()
	query := maretAdmin(t, reportkpi.AdminGroupNonMBU)

	seluruhnya, err := store.AdminDetail(context.Background(), query,
		reportkpi.Pagination{Page: 1, Size: reportkpi.MaxPageSize})
	require.NoError(t, err)
	require.Equal(t, 7, seluruhnya.Total, "empat leader dan tiga member pada Maret")

	halaman, err := store.AdminDetail(context.Background(), query,
		reportkpi.Pagination{Page: 1, Size: 3})
	require.NoError(t, err)
	require.Len(t, halaman.Rows, 3)
	require.Equal(t, 7, halaman.Total)
}

// Baris di luar periode tidak ikut, pada kedua kelompok.
func TestBarisDiLuarPeriodeTidakIkutPadaTabAdmin(t *testing.T) {
	store := memory.NewSampleStore()

	result, err := store.AdminDetail(context.Background(),
		maretAdmin(t, reportkpi.AdminGroupNonMBU),
		reportkpi.Pagination{Page: 1, Size: reportkpi.MaxPageSize})
	require.NoError(t, err)

	for _, row := range result.Rows {
		require.NotEqual(t, "CONTOH-ADM-0008", row.ClaimNumber)
		require.NotEqual(t, "CONTOH-ADM-0009", row.ClaimNumber)
	}
}

// Kelompok yang diminta benar-benar menyaring: baris PA tidak muncul di NON-MBU.
func TestKelompokMenyaringBarisRincian(t *testing.T) {
	store := memory.NewSampleStore()

	result, err := store.AdminDetail(context.Background(),
		maretAdmin(t, reportkpi.AdminGroupPA),
		reportkpi.Pagination{Page: 1, Size: reportkpi.MaxPageSize})
	require.NoError(t, err)
	require.NotEmpty(t, result.Rows)

	for _, row := range result.Rows {
		require.Empty(t, row.BusinessName,
			"baris PA tidak punya nama bisnis — kolom itu milik NON-MBU")
	}
}
