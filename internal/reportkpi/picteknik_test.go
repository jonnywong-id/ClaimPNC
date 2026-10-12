package reportkpi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// Uji tab KPI PIC Teknik.
//
// Tiga uji pertama mengunci perilaku yang TAMPAK SEPERTI CACAT dan memang cacat — tetapi
// cacat milik Pega, yang direplikasi atas `P-5`. Uji-uji itu akan GAGAL bila seseorang
// "memperbaikinya" tanpa keputusan Work Owner, dan itulah gunanya.

// bandsProgress adalah pita UPDATE PROGRESS KLAIM apa adanya dari M_KPI_PNC.
//
// Perhatikan arahnya: MENURUN. Persentase kecil bernilai tinggi.
func bandsProgress() []reportkpi.Band {
	return []reportkpi.Band{
		{Job: reportkpi.JobProgress, Value: 5, Bottom: 0, Top: 20},
		{Job: reportkpi.JobProgress, Value: 4, Bottom: 20, Top: 25},
		{Job: reportkpi.JobProgress, Value: 3, Bottom: 25, Top: 30},
		{Job: reportkpi.JobProgress, Value: 2, Bottom: 30, Top: 35},
		{Job: reportkpi.JobProgress, Value: 1, Bottom: 35, Top: 100},
	}
}

// bandsAnalysis adalah pita ANALISA KLAIM — arahnya MENAIK.
func bandsAnalysis() []reportkpi.Band {
	return []reportkpi.Band{
		{Job: reportkpi.JobAnalysis, Value: 5, Bottom: 80, Top: 100},
		{Job: reportkpi.JobAnalysis, Value: 4, Bottom: 75, Top: 80},
		{Job: reportkpi.JobAnalysis, Value: 3, Bottom: 70, Top: 75},
		{Job: reportkpi.JobAnalysis, Value: 2, Bottom: 65, Top: 70},
		{Job: reportkpi.JobAnalysis, Value: 1, Bottom: 0, Top: 65},
	}
}

func progressComponent(t *testing.T) reportkpi.PICComponent {
	t.Helper()
	component, found := reportkpi.FindPICComponent(reportkpi.PICComponentProgress)
	require.True(t, found)
	return component
}

func analysisComponent(t *testing.T) reportkpi.PICComponent {
	t.Helper()
	component, found := reportkpi.FindPICComponent(reportkpi.PICComponentAnalysis)
	require.True(t, found)
	return component
}

// Tangga UPDATE PROGRESS dan SLA memang MENURUN — dan itu yang membuat nilainya terbalik.
//
// Uji ini mengunci temuannya: PIC yang 95% tepat waktu bernilai 1, sedangkan yang 10% tepat
// waktu bernilai 5. Perilaku itu direplikasi dari Pega, BUKAN dipilih di sini.
func TestTanggaProgressMenurunSehinggaNilaiTerbalik(t *testing.T) {
	component := progressComponent(t)
	bands := bandsProgress()

	rajin := reportkpi.ScoreRow(component, "CONTOHPIC", 100, 95, bands)
	malas := reportkpi.ScoreRow(component, "CONTOHPIC", 100, 10, bands)

	require.Equal(t, float64(95), rajin.Percent.Value)
	require.Equal(t, float64(1), rajin.Value.Value,
		"95% tepat waktu jatuh ke pita 35–100 dan bernilai 1 — terbalik, dan itu perilaku Pega")

	require.Equal(t, float64(10), malas.Percent.Value)
	require.Equal(t, float64(5), malas.Value.Value,
		"10% tepat waktu jatuh ke pita 0–20 dan bernilai 5")
}

// Tambalan 100% pada Progress dibawa apa adanya.
//
// Ia menghasilkan gejala yang dapat diperiksa langsung di Pega: dua PIC sama-sama tampil
// "99%" dengan nilai 5 dan 1.
func TestTambalanSeratusPersenPadaProgressDireplikasi(t *testing.T) {
	component := progressComponent(t)
	bands := bandsProgress()

	sempurna := reportkpi.ScoreRow(component, "CONTOHPIC", 10, 10, bands)
	require.Equal(t, float64(99), sempurna.Percent.Value,
		"persentasenya DITULIS 99, bukan 100")
	require.Equal(t, float64(5), sempurna.Value.Value, "nilainya DIPAKSA 5")

	hampir := reportkpi.ScoreRow(component, "CONTOHPIC", 100, 99, bands)
	require.Equal(t, float64(99), hampir.Percent.Value)
	require.Equal(t, float64(1), hampir.Value.Value)

	require.Equal(t, sempurna.Percent.Value, hampir.Percent.Value,
		"keduanya tampil 99% …")
	require.NotEqual(t, sempurna.Value.Value, hampir.Value.Value,
		"… tetapi nilainya berbeda. Inilah gejala yang dapat diperiksa langsung di Pega")
}

// Tangga ANALISA dan AKSEPTASI MENAIK, dan keduanya tidak terdampak.
//
// Dipasangkan dengan uji di atas supaya terlihat bahwa yang terbalik memang hanya dua dari
// empat — bukan seluruh tabnya.
func TestTanggaAnalisaMenaikDanTidakTerbalik(t *testing.T) {
	component := analysisComponent(t)
	bands := bandsAnalysis()

	require.Equal(t, float64(5),
		reportkpi.ScoreRow(component, "CONTOHPIC", 100, 95, bands).Value.Value)
	require.Equal(t, float64(1),
		reportkpi.ScoreRow(component, "CONTOHPIC", 100, 10, bands).Value.Value)
}

// Pita bertetangga BERTINDIH, dan yang pertama cocok yang dipakai.
//
// Di Pega hasilnya di titik batas bergantung urutan baris yang kebetulan dikembalikan
// Oracle; di sini urutannya tetap. Uji ini mengunci pilihan itu.
func TestTitikBatasPitaMemakaiYangPertamaCocok(t *testing.T) {
	bands := bandsProgress()

	band, found := reportkpi.BandFor(bands, 20)
	require.True(t, found)
	require.Equal(t, float64(5), band.Value,
		"nilai 20 memenuhi pita 0–20 DAN 20–25; yang pertama yang dipakai")

	require.True(t, bands[0].Matches(20))
	require.True(t, bands[1].Matches(20), "keduanya memang cocok — pitanya bertindih")
}

// Persentase dibulatkan SEBELUM dikalikan 100, seperti `@divide(x,y,3)*100`.
func TestPersentaseDibulatkanSebelumDikalikanSeratus(t *testing.T) {
	require.InDelta(t, 66.7, reportkpi.PercentOf(3, 2), 0.0001,
		"2 dari 3 menjadi 0,667 × 100 = 66,7 — bukan 66,667")
	require.Equal(t, float64(0), reportkpi.PercentOf(0, 0),
		"total nol menghasilkan 0, bukan nilai kosong — itu perilaku lama")
}

// Nilai berbobot mengikuti ketiga pembulatan `GetBobotNilaiKPIPNC`, berskala 0–15.
func TestNilaiBerbobotMengikutiTigaPembulatan(t *testing.T) {
	// 95/100 → 0,95 → ×100 = 95 → /20 = 4,75 → ×(15/5 = 3) = 14,25
	require.InDelta(t, 14.25, reportkpi.WeightedScoreOf(100, 95, 15).Value, 0.0001)

	// Sempurna menghasilkan bobot penuh.
	require.InDelta(t, 15, reportkpi.WeightedScoreOf(10, 10, 15).Value, 0.0001)

	require.False(t, reportkpi.WeightedScoreOf(0, 0, 15).Present,
		"total nol tidak dapat dibobot — kosong, bukan nol")
}

// Pita yang tidak ditemukan menghasilkan nilai KOSONG, bukan nol.
//
// Nol bukan salah satu nilai yang sah pada skala 1–5; menggambarnya justru menyamarkan pita
// yang berlubang.
func TestPitaTidakDitemukanMenghasilkanNilaiKosong(t *testing.T) {
	component := analysisComponent(t)
	sempit := []reportkpi.Band{{Job: reportkpi.JobAnalysis, Value: 5, Bottom: 90, Top: 100}}

	row := reportkpi.ScoreRow(component, "CONTOHPIC", 100, 50, sempit)
	require.True(t, row.Percent.Present)
	require.False(t, row.Value.Present)
}

// Akseptasi memilih pasangan tanggal menurut TIM klaim, bukan menurut tanggal yang terisi.
func TestAkseptasiMemilihPasanganTanggalMenurutTim(t *testing.T) {
	lod := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	komite := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	aksep := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)

	leader := reportkpi.AcceptanceSpan{
		Team: reportkpi.TeamLeader, ReceiveLOD: lod,
		CommitteeDate: komite, AcceptanceDate: aksep,
	}
	start, end, ok := leader.Span()
	require.True(t, ok)
	require.Equal(t, lod, start)
	require.Equal(t, aksep, end)

	member := leader
	member.Team = reportkpi.TeamMember
	start, _, ok = member.Span()
	require.True(t, ok)
	require.Equal(t, komite, start)

	facIn := leader
	facIn.Team = "FAC IN"
	_, _, ok = facIn.Span()
	require.True(t, ok, "FAC IN mengikuti jalur MEMBER")

	lain := leader
	lain.Team = "ENTAH"
	_, _, ok = lain.Span()
	require.False(t, ok,
		"tim yang bukan ketiganya TIDAK dinilai sama sekali — itu perilaku lama")
}

// Dua petugas dikecualikan dari penilaian SLA dengan mencocokkan NAMA ORANG (`D-15`).
func TestDuaPetugasDikecualikanDariSLA(t *testing.T) {
	require.True(t, reportkpi.ExcludedFromSLA("BAMBANGSETIADJIGUNAWAN"))
	require.True(t, reportkpi.ExcludedFromSLA("BAMBANGSETIADJIGUNAWAN_1"),
		"pencocokannya SEBAGIAN nama, bukan kesamaan penuh — itu @contains di Pega")
	require.True(t, reportkpi.ExcludedFromSLA("DANIELLISWANDI"))
	require.False(t, reportkpi.ExcludedFromSLA("CONTOHPICLAIN"))
	require.Len(t, reportkpi.SLAExcludedPICs(), 2)
}

// Hari kerja: akhir pekan dan hari libur dikeluarkan, hari mulai tidak dihitung.
func TestPerhitunganHariKerja(t *testing.T) {
	// Senin 2 Maret 2026 sampai Jumat 6 Maret 2026 → 4 hari kerja.
	senin := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	jumat := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)
	require.Equal(t, 4, reportkpi.WorkingDaysBetween(senin, jumat, nil))

	// Satu hari libur di tengahnya memotong satu.
	libur := []time.Time{time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)}
	require.Equal(t, 3, reportkpi.WorkingDaysBetween(senin, jumat, libur))

	// Menyeberangi akhir pekan: Jumat ke Senin berikutnya → 1 hari kerja.
	seninDepan := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	require.Equal(t, 1, reportkpi.WorkingDaysBetween(jumat, seninDepan, nil))

	// Hari yang sama → nol.
	require.Equal(t, 0, reportkpi.WorkingDaysBetween(senin, senin, nil))

	// Tanggal kosong tidak menghasilkan angka karangan.
	require.Equal(t, 0, reportkpi.WorkingDaysBetween(time.Time{}, jumat, nil))
}

// Tim disimpulkan dari `LEADER_MEMBER`: apa pun selain persis "LEADER" menjadi MEMBER.
func TestTimDisimpulkanDariKolomLeaderMember(t *testing.T) {
	require.Equal(t, reportkpi.TeamLeader, reportkpi.TeamOf("LEADER"))
	require.Equal(t, reportkpi.TeamMember, reportkpi.TeamOf("MEMBER"))
	require.Equal(t, reportkpi.TeamMember, reportkpi.TeamOf(""),
		"kosong pun menjadi MEMBER — itu yang tertulis di activity lama")
}

// Baris Leader merata-ratakan, membatasi, dan MEMBULATKAN nilainya — tetapi tidak persennya.
func TestRekapitulasiLeaderMembulatkanNilaiTetapiTidakPersen(t *testing.T) {
	component := progressComponent(t)

	cards := []reportkpi.PICScorecard{
		{PIC: "A", Rows: []reportkpi.PICRow{{
			Component: component.Code, Label: component.Label,
			Total: 10, Achieved: 9,
			Percent: reportkpi.NewScore(90), Value: reportkpi.NewScore(1),
		}}},
		{PIC: "B", Rows: []reportkpi.PICRow{{
			Component: component.Code, Label: component.Label,
			Total: 10, Achieved: 1,
			Percent: reportkpi.NewScore(10), Value: reportkpi.NewScore(5),
		}}},
	}

	leader := reportkpi.AggregateLeader(cards)
	require.True(t, leader.Leader)
	require.Equal(t, "Leader", leader.PIC)

	var row reportkpi.PICRow
	for _, candidate := range leader.Rows {
		if candidate.Component == component.Code {
			row = candidate
		}
	}

	require.Equal(t, float64(20), row.Total, "total dijumlahkan")
	require.Equal(t, float64(10), row.Achieved)
	require.InDelta(t, 50, row.Percent.Value, 0.0001, "persen dirata-ratakan, tanpa dibulatkan")
	require.Equal(t, float64(3), row.Value.Value, "nilai dirata-ratakan LALU dibulatkan")
}

// Tanpa satu pun PIC, rekapitulasinya kosong — bukan nol.
func TestRekapitulasiTanpaPICMenghasilkanNilaiKosong(t *testing.T) {
	leader := reportkpi.AggregateLeader(nil)
	require.Len(t, leader.Rows, 4, "keempat baris tetap ada supaya bentuk kartunya tetap")
	for _, row := range leader.Rows {
		require.False(t, row.Percent.Present)
		require.False(t, row.Value.Present)
	}
}

// Keempat komponen ada, berurutan, dan hanya Progress yang berbobot.
func TestKeempatKomponenPICTeknik(t *testing.T) {
	components := reportkpi.PICComponents()
	require.Len(t, components, 4)

	jobs := []string{}
	weighted := 0
	descending := 0
	for i, component := range components {
		require.Equal(t, i+1, component.Order, "urutannya mengikuti penanda TKA")
		jobs = append(jobs, component.Job)
		if component.Weighted {
			weighted++
			require.Equal(t, reportkpi.PICComponentProgress, component.Code)
			require.Equal(t, float64(15), component.Weight)
		}
		if component.DescendingBand {
			descending++
		}
	}

	require.Equal(t, []string{
		reportkpi.JobProgress, reportkpi.JobAnalysis,
		reportkpi.JobAcceptance, reportkpi.JobSLA,
	}, jobs)
	require.Equal(t, 1, weighted, "hanya Progress yang melewati pembobotan")
	require.Equal(t, 2, descending, "dua tangga menurun: Progress dan SLA")
}

// Keempat lini bisnis dikenali, dan yang lain ditolak.
func TestLiniBisnisDikenali(t *testing.T) {
	require.Len(t, reportkpi.BusinessLines(), 4)

	for _, code := range []string{"NONMBU", "PA", "TRAVEL", "BONDING", "nonmbu"} {
		_, found := reportkpi.FindBusinessLine(code)
		require.True(t, found, code)
	}

	_, found := reportkpi.FindBusinessLine("MBU")
	require.False(t, found)
}

// Penyaring tab ini menuntut lini bisnis DAN kedua tanggal, dan mengumpulkan pelanggarannya.
func TestPenyaringPICTeknikMengumpulkanSeluruhPelanggaran(t *testing.T) {
	caller := reportkpi.Caller{Login: "contohpenyelia"}

	_, err := reportkpi.NewPICTeknikQuery(reportkpi.PICQueryInput{}, caller)
	require.Error(t, err)

	var invalid *reportkpi.ValidationError
	require.ErrorAs(t, err, &invalid)
	require.Len(t, invalid.Violations, 3,
		"lini bisnis, Dari, dan Sampai — ketiganya dilaporkan sekaligus")

	query, err := reportkpi.NewPICTeknikQuery(reportkpi.PICQueryInput{
		Line: "nonmbu", From: "2026-03-01", To: "2026-03-31",
	}, caller)
	require.NoError(t, err)
	require.Equal(t, reportkpi.LineNonMBU, query.Line)
}

// Tab KPI PIC Teknik TIDAK lagi terhalang, dan punya grid enam kolom.
func TestTabPICTeknikSudahDibangun(t *testing.T) {
	tab := reportkpi.PICTeknikTab()
	require.False(t, tab.Blocked)
	require.Equal(t, "KPI PIC Teknik", tab.Title)

	grid := reportkpi.PICTeknikGrid()
	require.Equal(t, reportkpi.GridPICScorecard, grid.Code)
	require.Len(t, grid.Columns, 6)
}

// Selisih terencana tab ini menyebut ketiga cacat yang direplikasi.
//
// Ia ditampilkan di layar, dan tanpanya pembaca akan melaporkan angka yang benar sebagai
// kerusakan (`D-54`).
func TestSelisihTerencanaPICMenyebutKetigaCacat(t *testing.T) {
	require.NotEmpty(t, reportkpi.PICTeknikPlannedDifferences)

	joined := ""
	for _, item := range reportkpi.PICTeknikPlannedDifferences {
		joined += item + " "
	}

	require.Contains(t, joined, "MENURUN")
	require.Contains(t, joined, "TAMBALAN 100%")
	require.Contains(t, joined, "DIRINYA SENDIRI")
	require.Contains(t, joined, "TIDAK DIBANGUN",
		"agregasi per kelompok tim dinyatakan tidak dibangun, bukan diam-diam hilang")
}
