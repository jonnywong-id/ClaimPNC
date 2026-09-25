package sqlstore

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/reportkpi"
)

// Pembacaan tab KPI PIC Teknik.

// lineFilterMarker adalah penanda di berkas .sql yang diganti potongan penyaring lini.
const lineFilterMarker = "/*LINE_FILTER*/"

// lineFilters adalah penyaring Group Panel per lini bisnis, DITULIS APA ADANYA dari
// `PNCReportKPI_act`.
//
// # Kenapa ini potongan teks dan bukan parameter
//
// Keempatnya bukan empat nilai pada kolom yang sama. Non-MBU menyaring tiga `GROUP_PANEL`
// SEKALIGUS mengecualikan empat `GROUPBISNISID`; Bonding justru menyaring keempat kode itu
// dan tidak menyentuh `GROUP_PANEL` sama sekali. Bentuk predikatnya berbeda, bukan hanya
// isinya.
//
// # Kenapa ini bukan celah penyisipan SQL
//
// Yang dipilih adalah potongan TETAP dari peta ini, dan kuncinya adalah `BusinessLine` yang
// sudah divalidasi menjadi salah satu dari empat oleh `NewPICTeknikQuery`. Lini yang tidak
// dikenal tidak pernah sampai ke sini; bila toh sampai, `lineFilter` mengembalikan predikat
// yang MENOLAK SEMUA baris, bukan predikat kosong yang meloloskan semuanya.
//
// Keempat kode `('09','11','16','25')` adalah nilai bisnis yang di-hardcode (`D-15`).
// Keduanya dipertahankan apa adanya sampai `P-5` terpenuhi.
var lineFilters = map[reportkpi.BusinessLine]string{
	reportkpi.LineNonMBU: "AND a.GROUP_PANEL IN ('003', '004', '006') " +
		"AND a.GROUPBISNISID NOT IN ('09', '11', '16', '25')",
	reportkpi.LinePA:      "AND a.GROUP_PANEL IN ('002')",
	reportkpi.LineTravel:  "AND a.GROUP_PANEL IN ('005')",
	reportkpi.LineBonding: "AND a.GROUPBISNISID IN ('09', '11', '16', '25')",
}

// lineFilter memilih potongan penyaring satu lini.
//
// Lini yang tidak dikenal menghasilkan `AND 1 = 0` — menolak semuanya. Itu disengaja:
// penyaring yang hilang akan menampilkan data SELURUH lini kepada orang yang meminta satu
// lini, dan kekeliruan semacam itu tidak terlihat sebagai galat.
func lineFilter(line reportkpi.BusinessLine) string {
	if filter, known := lineFilters[line]; known {
		return filter
	}
	return "AND 1 = 0"
}

// picQuery mengambil kueri bernama dan memasang penyaring lininya.
func picQuery(name string, line reportkpi.BusinessLine) string {
	return strings.Replace(query(name), lineFilterMarker, lineFilter(line), 1)
}

// PICs mengembalikan petugas satu lini bisnis.
func (r *Repo) PICs(ctx context.Context, line reportkpi.BusinessLine) ([]reportkpi.PICProfile, error) {
	rows, err := r.db.QueryContext(ctx, query("pic_list"), string(line))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []reportkpi.PICProfile
	for rows.Next() {
		var operator sql.NullString
		var leader sql.NullString
		if err := rows.Scan(&operator, &leader); err != nil {
			return nil, err
		}
		id := strings.TrimSpace(operator.String)
		if id == "" {
			continue
		}
		result = append(result, reportkpi.PICProfile{
			OperatorID: id,
			// Kolomnya bertipe teks berisi "TRUE"/"FALSE", bukan boolean. Perbandingannya
			// mengabaikan huruf besar-kecil karena isinya tidak dijamin seragam.
			Leader: strings.EqualFold(strings.TrimSpace(leader.String), "TRUE"),
		})
	}
	return result, rows.Err()
}

// Bands mengembalikan pita nilai satu komponen, terurut menurut `ID`.
func (r *Repo) Bands(ctx context.Context, job, note string) ([]reportkpi.Band, error) {
	rows, err := r.db.QueryContext(ctx, query("bands"), job, nullableText(note))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []reportkpi.Band
	for rows.Next() {
		var jobName, noteText sql.NullString
		var value, bottom, top sql.NullFloat64
		if err := rows.Scan(&jobName, &value, &bottom, &top, &noteText); err != nil {
			return nil, err
		}
		result = append(result, reportkpi.Band{
			Job:    strings.TrimSpace(jobName.String),
			Value:  value.Float64,
			Bottom: bottom.Float64,
			Top:    top.Float64,
			Note:   strings.TrimSpace(noteText.String),
		})
	}
	return result, rows.Err()
}

// ThresholdDays mengembalikan ambang hari satu komponen.
//
// Tidak ada baris berarti komponennya memang tidak punya ambang hari, dan itu dikembalikan
// sebagai nilai KOSONG — bukan nol. Nol berarti "harus selesai di hari yang sama", dan itu
// pernyataan yang sangat berbeda dari "tidak diukur lamanya".
func (r *Repo) ThresholdDays(ctx context.Context, job, note string) (reportkpi.Score, error) {
	var days sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query("threshold_days"), job, nullableText(note)).Scan(&days)
	switch {
	case err == sql.ErrNoRows:
		return reportkpi.EmptyScore(), nil
	case err != nil:
		return reportkpi.EmptyScore(), err
	case !days.Valid:
		return reportkpi.EmptyScore(), nil
	default:
		return reportkpi.NewScore(days.Float64), nil
	}
}

// Holidays mengembalikan tanggal libur pada satu rentang, di luar akhir pekan.
func (r *Repo) Holidays(ctx context.Context, from, to time.Time) ([]time.Time, error) {
	rows, err := r.db.QueryContext(ctx, query("holidays"), from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []time.Time
	for rows.Next() {
		var day sql.NullTime
		if err := rows.Scan(&day); err != nil {
			return nil, err
		}
		if day.Valid {
			result = append(result, day.Time)
		}
	}
	return result, rows.Err()
}

// ProgressCounts mengembalikan cacah pembaruan progres per PIC.
func (r *Repo) ProgressCounts(
	ctx context.Context, q reportkpi.PICQuery,
) ([]reportkpi.ProgressCount, error) {
	rows, err := r.db.QueryContext(ctx, picQuery("progress_counts", q.Line), q.From, q.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []reportkpi.ProgressCount
	for rows.Next() {
		var pic sql.NullString
		var total, onTime sql.NullFloat64
		if err := rows.Scan(&pic, &total, &onTime); err != nil {
			return nil, err
		}
		result = append(result, reportkpi.ProgressCount{
			PIC:    strings.TrimSpace(pic.String),
			Total:  total.Float64,
			OnTime: onTime.Float64,
		})
	}
	return result, rows.Err()
}

// AnalysisSpans mengembalikan pasangan tanggal penilaian Analisa Klaim.
func (r *Repo) AnalysisSpans(
	ctx context.Context, q reportkpi.PICQuery,
) ([]reportkpi.DateSpan, error) {
	rows, err := r.db.QueryContext(ctx, picQuery("analysis_spans", q.Line), q.From, q.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []reportkpi.DateSpan
	for rows.Next() {
		var pic sql.NullString
		var start, end sql.NullTime
		if err := rows.Scan(&pic, &start, &end); err != nil {
			return nil, err
		}
		result = append(result, reportkpi.DateSpan{
			PIC:   strings.TrimSpace(pic.String),
			Start: start.Time,
			End:   end.Time,
		})
	}
	return result, rows.Err()
}

// AcceptanceSpans mengembalikan pasangan tanggal penilaian Akseptasi Klaim.
func (r *Repo) AcceptanceSpans(
	ctx context.Context, q reportkpi.PICQuery,
) ([]reportkpi.AcceptanceSpan, error) {
	rows, err := r.db.QueryContext(ctx, picQuery("acceptance_spans", q.Line), q.From, q.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []reportkpi.AcceptanceSpan
	for rows.Next() {
		var pic, team sql.NullString
		var lod, committee, accepted sql.NullTime
		if err := rows.Scan(&pic, &team, &lod, &committee, &accepted); err != nil {
			return nil, err
		}
		result = append(result, reportkpi.AcceptanceSpan{
			PIC:            strings.TrimSpace(pic.String),
			Team:           strings.TrimSpace(team.String),
			ReceiveLOD:     lod.Time,
			CommitteeDate:  committee.Time,
			AcceptanceDate: accepted.Time,
		})
	}
	return result, rows.Err()
}

// ClosureSpans mengembalikan pasangan tanggal penilaian SLA Klaim.
func (r *Repo) ClosureSpans(
	ctx context.Context, q reportkpi.PICQuery,
) ([]reportkpi.ClosureSpan, error) {
	rows, err := r.db.QueryContext(ctx, picQuery("closure_spans", q.Line), q.From, q.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []reportkpi.ClosureSpan
	for rows.Next() {
		var pic, team sql.NullString
		var registered, closed sql.NullTime
		if err := rows.Scan(&pic, &team, &registered, &closed); err != nil {
			return nil, err
		}
		result = append(result, reportkpi.ClosureSpan{
			PIC:          strings.TrimSpace(pic.String),
			Team:         strings.TrimSpace(team.String),
			RegisterDate: registered.Time,
			CloseDate:    closed.Time,
		})
	}
	return result, rows.Err()
}

// BandState adalah ringkasan isi tangga nilai satu komponen.
//
// Dipakai mode periksa. Ia menjawab pertanyaan yang TIDAK dapat dijawab dari kode: apakah
// tangga itu ada, berapa pitanya, dan ke arah mana ia tersusun.
type BandState struct {
	Job string

	// Bands adalah jumlah pita yang terbaca.
	Bands int

	// Descending menyatakan tangga ini tersusun MENURUN — pita terbawah bernilai tertinggi.
	//
	// Diperiksa dari DATA, bukan diasumsikan dari nama komponennya. Kalau kelak isi
	// tabelnya diperbaiki di produksi, pemeriksaan ini yang pertama menunjukkannya.
	Descending bool

	// Gaps menyebut lubang atau tumpang tindih antar pita yang berdampingan.
	Gaps []string
}

// CheckBands membaca tangga nilai satu komponen dan meringkas bentuknya.
func (r *Repo) CheckBands(ctx context.Context, job string) (BandState, error) {
	bands, err := r.Bands(ctx, job, "")
	if err != nil {
		return BandState{}, err
	}

	state := BandState{Job: job, Bands: len(bands)}
	if len(bands) < 2 {
		return state, nil
	}

	state.Descending = bands[0].Value > bands[len(bands)-1].Value

	for i := 1; i < len(bands); i++ {
		previous, current := bands[i-1], bands[i]
		switch {
		case current.Bottom > previous.Top:
			state.Gaps = append(state.Gaps, "lubang antara "+
				formatBound(previous.Top)+" dan "+formatBound(current.Bottom))
		case current.Bottom < previous.Top:
			state.Gaps = append(state.Gaps, "tumpang tindih antara "+
				formatBound(current.Bottom)+" dan "+formatBound(previous.Top))
		}
	}

	return state, nil
}

// formatBound menggambar satu batas pita.
func formatBound(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
