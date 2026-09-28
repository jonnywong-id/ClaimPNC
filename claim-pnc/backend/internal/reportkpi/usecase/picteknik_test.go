package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
	reportkpimemory "claim-pnc/internal/reportkpi/repo/memory"
	"claim-pnc/internal/reportkpi/usecase"
)

// Uji perakitan tab KPI PIC Teknik.
//
// Berbeda dari uji domain, yang ini menjalankan SELURUH jalurnya: penyaring, pembacaan
// keempat kelompok bahan, perhitungan hari kerja, pencarian pita, sampai rekapitulasi. Itu
// yang membuktikan keempat komponen benar-benar tersambung — uji domain hanya membuktikan
// rumusnya benar bila bahannya sudah di tangan.

const samplePortal = "ASM"

func picService(t *testing.T) *usecase.Service {
	t.Helper()

	store := reportkpimemory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (reportkpi.Repo, error) {
			require.Equal(t, samplePortal, alias)
			return store, nil
		},
	})
	require.NoError(t, err)
	return service
}

func picCaller() reportkpi.Caller {
	return reportkpi.Caller{Login: "contohpenyelia"}
}

func picInput() reportkpi.PICQueryInput {
	return reportkpi.PICQueryInput{
		Line: string(reportkpi.LineNonMBU),
		From: "2026-03-01",
		To:   "2026-03-31",
	}
}

// rowOf mengambil satu baris komponen dari sebuah kartu skor.
func rowOf(t *testing.T, card reportkpi.PICScorecard, component string) reportkpi.PICRow {
	t.Helper()
	for _, row := range card.Rows {
		if row.Component == component {
			return row
		}
	}
	t.Fatalf("komponen %s tidak ada pada kartu %s", component, card.PIC)
	return reportkpi.PICRow{}
}

// cardOf mengambil kartu skor satu PIC.
func cardOf(t *testing.T, result reportkpi.PICTeknikResult, pic string) reportkpi.PICScorecard {
	t.Helper()
	for _, card := range result.Scorecards {
		if card.PIC == pic {
			return card
		}
	}
	t.Fatalf("kartu skor %s tidak ada", pic)
	return reportkpi.PICScorecard{}
}

// Perakitan penuh menghasilkan satu kartu per PIC, masing-masing empat komponen.
func TestPICTeknikMerakitKartuPerPetugas(t *testing.T) {
	scored, err := picService(t).PICTeknik(
		context.Background(), samplePortal, picInput(), picCaller())
	require.NoError(t, err)

	require.Len(t, scored.Result.Scorecards, 2)
	require.Equal(t, reportkpi.LineNonMBU, scored.Result.Line)

	for _, card := range scored.Result.Scorecards {
		require.Len(t, card.Rows, 4, "kartu %s tidak lengkap", card.PIC)
	}

	require.True(t, cardOf(t, scored.Result, "CONTOHPICSATU").Leader)
	require.False(t, cardOf(t, scored.Result, "CONTOHPICDUA").Leader)
}

// Angka komponen Progress dapat diperiksa dengan tangan — dan hasilnya TERBALIK.
//
// CONTOHPICSATU 9 dari 10 tepat waktu → 90% → tangga MENURUN → nilai 1.
// CONTOHPICDUA  2 dari 10 tepat waktu → 20% → pita 0–20      → nilai 5.
//
// Yang rajin bernilai 1, yang jarang bernilai 5. Itu perilaku Pega yang direplikasi, bukan
// kekeliruan contohnya.
func TestPICTeknikProgresTerbalikSepertiPega(t *testing.T) {
	scored, err := picService(t).PICTeknik(
		context.Background(), samplePortal, picInput(), picCaller())
	require.NoError(t, err)

	rajin := rowOf(t, cardOf(t, scored.Result, "CONTOHPICSATU"), reportkpi.PICComponentProgress)
	require.Equal(t, float64(10), rajin.Total)
	require.Equal(t, float64(9), rajin.Achieved)
	require.InDelta(t, 90, rajin.Percent.Value, 0.0001)
	require.Equal(t, float64(1), rajin.Value.Value)

	jarang := rowOf(t, cardOf(t, scored.Result, "CONTOHPICDUA"), reportkpi.PICComponentProgress)
	require.InDelta(t, 20, jarang.Percent.Value, 0.0001)
	require.Equal(t, float64(5), jarang.Value.Value)
}

// Nilai berbobot hanya ada pada kartu, bukan pada baris — dan berskala 0–15.
func TestPICTeknikNilaiBerbobotBerskalaLimaBelas(t *testing.T) {
	scored, err := picService(t).PICTeknik(
		context.Background(), samplePortal, picInput(), picCaller())
	require.NoError(t, err)

	card := cardOf(t, scored.Result, "CONTOHPICSATU")
	require.True(t, card.Weighted.Present)
	require.InDelta(t, 13.5, card.Weighted.Value, 0.0001,
		"round(0,9)*100/20 = 4,5 lalu ×(15/5) = 13,5")
}

// Analisa memakai ambang 10 hari kerja dari kolom DAY, dan hari libur ikut dipotong.
func TestPICTeknikAnalisaMemakaiAmbangSepuluhHari(t *testing.T) {
	scored, err := picService(t).PICTeknik(
		context.Background(), samplePortal, picInput(), picCaller())
	require.NoError(t, err)

	rajin := rowOf(t, cardOf(t, scored.Result, "CONTOHPICSATU"), reportkpi.PICComponentAnalysis)
	require.Equal(t, float64(1), rajin.Total)
	require.Equal(t, float64(1), rajin.Achieved, "3 hari kerja, di bawah ambang 10")
	require.InDelta(t, 100, rajin.Percent.Value, 0.0001)
	require.Equal(t, float64(5), rajin.Value.Value, "tangga Analisa MENAIK")

	lambat := rowOf(t, cardOf(t, scored.Result, "CONTOHPICDUA"), reportkpi.PICComponentAnalysis)
	require.Equal(t, float64(0), lambat.Achieved, "melewati ambang 10 hari kerja")
	require.Equal(t, float64(1), lambat.Value.Value)
}

// Akseptasi memilih pasangan tanggal menurut tim klaim, dan itu terbukti pada hasilnya.
func TestPICTeknikAkseptasiMemilihPasanganMenurutTim(t *testing.T) {
	scored, err := picService(t).PICTeknik(
		context.Background(), samplePortal, picInput(), picCaller())
	require.NoError(t, err)

	leader := rowOf(t, cardOf(t, scored.Result, "CONTOHPICSATU"), reportkpi.PICComponentAcceptance)
	require.Equal(t, float64(1), leader.Achieved,
		"LEADER memakai ReceiveLOD → Accepted, 1 hari kerja")

	member := rowOf(t, cardOf(t, scored.Result, "CONTOHPICDUA"), reportkpi.PICComponentAcceptance)
	require.Equal(t, float64(0), member.Achieved,
		"MEMBER memakai Committee → Accepted, 8 hari kerja — melewati ambang 1")
}

// Rekapitulasi Leader hadir, punya keempat baris, dan menjumlahkan totalnya.
func TestPICTeknikRekapitulasiLeader(t *testing.T) {
	scored, err := picService(t).PICTeknik(
		context.Background(), samplePortal, picInput(), picCaller())
	require.NoError(t, err)

	leader := scored.Result.Leader
	require.True(t, leader.Leader)
	require.Len(t, leader.Rows, 4)

	progres := rowOf(t, leader, reportkpi.PICComponentProgress)
	require.Equal(t, float64(20), progres.Total, "10 + 10")
	require.Equal(t, float64(11), progres.Achieved, "9 + 2")
	require.InDelta(t, 55, progres.Percent.Value, 0.0001, "(90 + 20) / 2")
	require.Equal(t, float64(3), progres.Value.Value, "(1 + 5) / 2 lalu dibulatkan")
}

// Periode di luar data menghasilkan kartu KOSONG, bukan galat — dan itu perlu diperhatikan.
//
// Pada tangga MENURUN, tidak ada data sama sekali menghasilkan persen 0 dan karena itu nilai
// TERTINGGI. Perilaku itu berasal dari `@if(total==0, 0, …)` di sistem lama.
func TestPICTeknikPeriodeKosongMenghasilkanNilaiTertinggiPadaTanggaMenurun(t *testing.T) {
	input := picInput()
	input.From = "2026-07-01"
	input.To = "2026-07-31"

	scored, err := picService(t).PICTeknik(
		context.Background(), samplePortal, input, picCaller())
	require.NoError(t, err)

	progres := rowOf(t, cardOf(t, scored.Result, "CONTOHPICSATU"), reportkpi.PICComponentProgress)
	require.Equal(t, float64(0), progres.Total)
	require.InDelta(t, 0, progres.Percent.Value, 0.0001)
	require.Equal(t, float64(5), progres.Value.Value,
		"tanpa data sama sekali, tangga menurun memberi nilai tertinggi")
}

// Penyaring yang tidak lengkap ditolak sebelum satu pun kueri dijalankan.
func TestPICTeknikMenolakPenyaringTidakLengkap(t *testing.T) {
	_, err := picService(t).PICTeknik(
		context.Background(), samplePortal, reportkpi.PICQueryInput{}, picCaller())

	var invalid *reportkpi.ValidationError
	require.ErrorAs(t, err, &invalid)
	require.Len(t, invalid.Violations, 3)
}
