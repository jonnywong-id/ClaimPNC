package usecase

import (
	"context"
	"sort"
	"time"

	"claim-pnc/internal/reportkpi"
)

// Perakitan tab KPI PIC Teknik.
//
// # Kenapa perakitannya ada di sini dan bukan di dalam SQL
//
// Di sistem lama, nilai tab ini TIDAK dihitung oleh kueri. Kueri hanya mengembalikan baris
// mentah; yang menghitung adalah langkah-langkah activity — empat sub-activity, masing-masing
// melakukan perulangan per klaim, memanggil kalender hari kerja, lalu mencacah yang tepat
// waktu.
//
// Menaikkannya ke Go bukan pilihan gaya melainkan `D-02`: basis data menjadi penyimpanan
// murni. Yang wajib dijaga adalah hasilnya (`P-5`), dan itulah yang diuji.
//
// # Satu kueri untuk seluruh PIC, bukan satu per orang
//
// Sistem lama menjalankan keempat kelompok kueri SEKALI PER PIC — pada lini dengan dua puluh
// petugas, itu delapan puluh perjalanan ke basis data untuk satu layar. Di sini masing-masing
// dijalankan sekali lalu dikelompokkan di memori.
//
// Hasilnya sama karena penyaringnya memang hanya berbeda pada nama PIC-nya. Yang berubah
// hanya berapa kali kueri dijalankan, dan itu bukan perilaku yang dilihat pengguna.

// PICTeknikScored adalah jawaban lengkap tab KPI PIC Teknik.
type PICTeknikScored struct {
	Result reportkpi.PICTeknikResult
	Filter reportkpi.PICTeknikQuery
	Portal string
}

// PICTeknik merakit kartu skor seluruh PIC pada satu lini bisnis dan periode.
func (s *Service) PICTeknik(
	ctx context.Context,
	portal string,
	input reportkpi.PICQueryInput,
	caller reportkpi.Caller,
) (PICTeknikScored, error) {
	query, err := reportkpi.NewPICTeknikQuery(input, caller)
	if err != nil {
		return PICTeknikScored{}, err
	}

	repo, err := s.repoSelector(portal)
	if err != nil {
		return PICTeknikScored{}, err
	}

	span := query.Span()

	pics, err := repo.PICs(ctx, query.Line)
	if err != nil {
		return PICTeknikScored{}, err
	}

	// Hari libur diambil SEKALI untuk seluruh periode, bukan per klaim.
	//
	// Sistem lama memanggil `CheckHoliday_SQL` di dalam perulangan — satu perjalanan ke DB
	// link untuk setiap klaim yang dinilai. Hasilnya sama; yang berbeda hanya berapa kali
	// ia ditanyakan.
	from, to := query.Dates()
	holidays, err := repo.Holidays(ctx, from, to)
	if err != nil {
		return PICTeknikScored{}, err
	}

	bands, err := s.loadBands(ctx, repo)
	if err != nil {
		return PICTeknikScored{}, err
	}

	days, err := s.loadThresholds(ctx, repo)
	if err != nil {
		return PICTeknikScored{}, err
	}

	progress, err := repo.ProgressCounts(ctx, span)
	if err != nil {
		return PICTeknikScored{}, err
	}

	analysis, err := repo.AnalysisSpans(ctx, span)
	if err != nil {
		return PICTeknikScored{}, err
	}

	acceptance, err := repo.AcceptanceSpans(ctx, span)
	if err != nil {
		return PICTeknikScored{}, err
	}

	closure, err := repo.ClosureSpans(ctx, span)
	if err != nil {
		return PICTeknikScored{}, err
	}

	cards := buildPICCards(pics, holidays, bands, days, progress, analysis, acceptance, closure)

	return PICTeknikScored{
		Result: reportkpi.PICTeknikResult{
			Line:       query.Line,
			Scorecards: cards,
			Leader:     reportkpi.AggregateLeader(cards),
		},
		Filter: query,
		Portal: portal,
	}, nil
}

// bandSet memegang pita seluruh komponen, termasuk kedua varian SLA.
type bandSet struct {
	byJob     map[string][]reportkpi.Band
	slaLeader []reportkpi.Band
	slaMember []reportkpi.Band
}

// forComponent memilih pita yang berlaku untuk satu komponen dan satu tim.
func (b bandSet) forComponent(component reportkpi.PICComponent, team string) []reportkpi.Band {
	if component.Code != reportkpi.PICComponentSLA {
		return b.byJob[component.Job]
	}
	if team == reportkpi.TeamLeader {
		return b.slaLeader
	}
	return b.slaMember
}

// loadBands membaca seluruh pita sekali di muka.
//
// Pita SLA dibaca DUA KALI dengan penyaring `NOTE` yang berbeda, karena Leader dan Member
// punya barisnya sendiri di tabel — dan yang membedakannya hanya kolom itu.
func (s *Service) loadBands(ctx context.Context, repo reportkpi.Repo) (bandSet, error) {
	set := bandSet{byJob: map[string][]reportkpi.Band{}}

	for _, component := range reportkpi.PICComponents() {
		if component.Code == reportkpi.PICComponentSLA {
			continue
		}
		bands, err := repo.Bands(ctx, component.Job, "")
		if err != nil {
			return bandSet{}, err
		}
		set.byJob[component.Job] = bands
	}

	leader, err := repo.Bands(ctx, reportkpi.JobSLA, reportkpi.TeamLeader)
	if err != nil {
		return bandSet{}, err
	}
	member, err := repo.Bands(ctx, reportkpi.JobSLA, reportkpi.TeamMember)
	if err != nil {
		return bandSet{}, err
	}

	// Pita SLA di tabel tidak dipisah per tim — kolom NOTE hanya terisi pada dua baris
	// teratas, dan sisanya kosong. Kueri lama menyaring `note = ?`, sehingga yang kembali
	// hanya satu baris. Bila penyaringnya tidak menghasilkan apa-apa, dipakai pita SLA
	// tanpa penyaring supaya komponennya tidak diam-diam kehilangan nilai.
	if len(leader) == 0 || len(member) == 0 {
		all, err := repo.Bands(ctx, reportkpi.JobSLA, "")
		if err != nil {
			return bandSet{}, err
		}
		if len(leader) == 0 {
			leader = all
		}
		if len(member) == 0 {
			member = all
		}
	}

	set.slaLeader = leader
	set.slaMember = member
	return set, nil
}

// thresholdSet memegang ambang hari tiap komponen.
type thresholdSet struct {
	analysis   reportkpi.Score
	acceptance reportkpi.Score
	slaLeader  reportkpi.Score
	slaMember  reportkpi.Score
}

// loadThresholds membaca kolom `DAY` untuk ketiga komponen yang mengukur lama.
//
// Progress tidak punya ambang hari, dan itu benar: ia mengukur ketepatan tindak lanjut
// terhadap tanggal janji yang sudah tersimpan per baris progres, bukan lama pengerjaan.
func (s *Service) loadThresholds(ctx context.Context, repo reportkpi.Repo) (thresholdSet, error) {
	analysis, err := repo.ThresholdDays(ctx, reportkpi.JobAnalysis, "")
	if err != nil {
		return thresholdSet{}, err
	}
	acceptance, err := repo.ThresholdDays(ctx, reportkpi.JobAcceptance, "")
	if err != nil {
		return thresholdSet{}, err
	}
	slaLeader, err := repo.ThresholdDays(ctx, reportkpi.JobSLA, reportkpi.TeamLeader)
	if err != nil {
		return thresholdSet{}, err
	}
	slaMember, err := repo.ThresholdDays(ctx, reportkpi.JobSLA, reportkpi.TeamMember)
	if err != nil {
		return thresholdSet{}, err
	}
	return thresholdSet{
		analysis:   analysis,
		acceptance: acceptance,
		slaLeader:  slaLeader,
		slaMember:  slaMember,
	}, nil
}

// buildPICCards merakit kartu skor tiap PIC dari seluruh bahan yang sudah dibaca.
func buildPICCards(
	pics []reportkpi.PICProfile,
	holidays []time.Time,
	bands bandSet,
	days thresholdSet,
	progress []reportkpi.ProgressCount,
	analysis []reportkpi.DateSpan,
	acceptance []reportkpi.AcceptanceSpan,
	closure []reportkpi.ClosureSpan,
) []reportkpi.PICScorecard {
	progressByPIC := map[string]reportkpi.ProgressCount{}
	for _, row := range progress {
		progressByPIC[row.PIC] = row
	}

	analysisByPIC := map[string][]reportkpi.DateSpan{}
	for _, row := range analysis {
		analysisByPIC[row.PIC] = append(analysisByPIC[row.PIC], row)
	}

	acceptanceByPIC := map[string][]reportkpi.AcceptanceSpan{}
	for _, row := range acceptance {
		acceptanceByPIC[row.PIC] = append(acceptanceByPIC[row.PIC], row)
	}

	closureByPIC := map[string][]reportkpi.ClosureSpan{}
	for _, row := range closure {
		closureByPIC[row.PIC] = append(closureByPIC[row.PIC], row)
	}

	sorted := make([]reportkpi.PICProfile, len(pics))
	copy(sorted, pics)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].OperatorID < sorted[j].OperatorID })

	cards := make([]reportkpi.PICScorecard, 0, len(sorted))
	for _, profile := range sorted {
		card := reportkpi.PICScorecard{PIC: profile.OperatorID, Leader: profile.Leader}

		for _, component := range reportkpi.PICComponents() {
			switch component.Code {
			case reportkpi.PICComponentProgress:
				counts := progressByPIC[profile.OperatorID]
				card.Rows = append(card.Rows, reportkpi.ScoreRow(
					component, profile.OperatorID,
					counts.Total, counts.OnTime,
					bands.forComponent(component, ""),
				))

			case reportkpi.PICComponentAnalysis:
				rows := analysisByPIC[profile.OperatorID]
				total, achieved := countWithin(rows, holidays, days.analysis)
				card.Rows = append(card.Rows, reportkpi.ScoreRow(
					component, profile.OperatorID, total, achieved,
					bands.forComponent(component, ""),
				))

			case reportkpi.PICComponentAcceptance:
				rows := acceptanceByPIC[profile.OperatorID]
				total, achieved := countAcceptance(rows, holidays, days.acceptance)
				card.Rows = append(card.Rows, reportkpi.ScoreRow(
					component, profile.OperatorID, total, achieved,
					bands.forComponent(component, ""),
				))

			case reportkpi.PICComponentSLA:
				// Dua petugas dikecualikan dari komponen ini, dengan mencocokkan NAMA
				// ORANG di dalam kode. Direplikasi dari Pega — lihat ExcludedFromSLA.
				if reportkpi.ExcludedFromSLA(profile.OperatorID) {
					continue
				}
				rows := closureByPIC[profile.OperatorID]
				team := teamOfRows(rows)
				total, achieved := countClosure(rows, holidays, days, team)
				card.Rows = append(card.Rows, reportkpi.ScoreRow(
					component, profile.OperatorID, total, achieved,
					bands.forComponent(component, team),
				))
			}
		}

		// Komponen Progress ditambah nilai berbobotnya.
		//
		// Ia TIDAK menggantikan nilai pita: keduanya hidup berdampingan di sistem lama,
		// disimpan di kolom yang berbeda, dan berskala berbeda pula (0–15 versus 1–5).
		for i := range card.Rows {
			component, found := reportkpi.FindPICComponent(card.Rows[i].Component)
			if !found || !component.Weighted {
				continue
			}
			card.Weighted = reportkpi.WeightedScoreOf(
				card.Rows[i].Total, card.Rows[i].Achieved, component.Weight,
			)
		}

		cards = append(cards, card)
	}

	return cards
}

// countWithin mencacah berapa rentang yang selisih hari kerjanya TIDAK melebihi ambang.
//
// Ambang yang kosong berarti barisnya tidak dapat dinilai sama sekali — pembilangnya nol,
// pembaginya tetap utuh. Itu lebih jujur daripada memakai ambang bawaan yang dikarang:
// nilainya akan tampak sah dan tidak ada yang tahu ia bukan dari tabel.
func countWithin(
	rows []reportkpi.DateSpan,
	holidays []time.Time,
	threshold reportkpi.Score,
) (total, achieved float64) {
	total = float64(len(rows))
	if !threshold.Present {
		return total, 0
	}
	for _, row := range rows {
		if float64(reportkpi.WorkingDaysBetween(row.Start, row.End, holidays)) <= threshold.Value {
			achieved++
		}
	}
	return total, achieved
}

// countAcceptance mencacah Akseptasi Klaim, yang pasangan tanggalnya bergantung tim klaim.
func countAcceptance(
	rows []reportkpi.AcceptanceSpan,
	holidays []time.Time,
	threshold reportkpi.Score,
) (total, achieved float64) {
	total = float64(len(rows))
	if !threshold.Present {
		return total, 0
	}
	for _, row := range rows {
		start, end, ok := row.Span()
		if !ok {
			continue
		}
		if float64(reportkpi.WorkingDaysBetween(start, end, holidays)) <= threshold.Value {
			achieved++
		}
	}
	return total, achieved
}

// countClosure mencacah SLA Klaim, yang ambangnya berbeda antara Leader dan Member.
func countClosure(
	rows []reportkpi.ClosureSpan,
	holidays []time.Time,
	days thresholdSet,
	team string,
) (total, achieved float64) {
	total = float64(len(rows))

	threshold := days.slaMember
	if team == reportkpi.TeamLeader {
		threshold = days.slaLeader
	}
	if !threshold.Present {
		return total, 0
	}

	for _, row := range rows {
		if float64(reportkpi.WorkingDaysBetween(row.RegisterDate, row.CloseDate, holidays)) <= threshold.Value {
			achieved++
		}
	}
	return total, achieved
}

// teamOfRows menyimpulkan tim seorang PIC dari klaim yang ditanganinya.
//
// Di sistem lama tim diambil per KLAIM (`LEADER_MEMBER` klaimnya), bukan per orang — dan
// ambang SLA-nya dipilih dari sana. Di sini tim diambil dari klaim PERTAMA petugas itu,
// karena ambangnya dipakai untuk seluruh barisnya sekaligus.
//
// Keduanya menghasilkan angka yang sama selama seorang PIC menangani klaim dengan tim yang
// sama, dan itu yang terjadi menurut isi kolomnya. Bila kelak tidak, selisihnya muncul pada
// komponen SLA saja.
func teamOfRows(rows []reportkpi.ClosureSpan) string {
	if len(rows) == 0 {
		return reportkpi.TeamMember
	}
	return reportkpi.TeamOf(rows[0].Team)
}
