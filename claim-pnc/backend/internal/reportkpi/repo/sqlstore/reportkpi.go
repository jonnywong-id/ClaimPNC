package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/reportkpi"
)

// Repo membaca KPI adjuster dari basis data satu portal.
//
// Ia MEMBACA saja. Tidak ada satu pun method yang menulis, dan itu bukan pekerjaan yang
// tertinggal — lihat doc paket `reportkpi`.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk penyimpanan di atas satu koneksi portal.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// Repo memenuhi seam yang dideklarasikan domain.
var _ reportkpi.Repo = (*Repo)(nil)

// Summary mengambil grid Summary KPI Adjuster.
func (r *Repo) Summary(
	ctx context.Context,
	q reportkpi.Query,
) ([]reportkpi.AdjusterSummary, error) {
	rows, err := r.db.QueryContext(ctx, query("summary"),
		reportTypeBind(q.ReportType),
		nullableText(q.Adjuster),
		q.Range.From,
		q.Range.To,
	)
	if err != nil {
		return nil, fmt.Errorf("reportkpi/sqlstore: membaca ringkasan: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []reportkpi.AdjusterSummary{}
	for rows.Next() {
		var (
			adjuster   sql.NullString
			reportType sql.NullString
			scores     = make([]sql.NullFloat64, len(scoreColumns))
		)

		target := []any{&adjuster, &reportType}
		for i := range scores {
			target = append(target, &scores[i])
		}
		if err := rows.Scan(target...); err != nil {
			return nil, fmt.Errorf("reportkpi/sqlstore: memindai ringkasan: %w", err)
		}

		result = append(result, reportkpi.AdjusterSummary{
			Adjuster:   adjuster.String,
			ReportType: reportkpi.ReportType(reportType.String),
			Scores:     toScores(scores),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reportkpi/sqlstore: membaca ringkasan: %w", err)
	}
	return result, nil
}

// Detail mengambil satu halaman grid Detail KPI Adjuster.
func (r *Repo) Detail(
	ctx context.Context,
	q reportkpi.Query,
	page reportkpi.Pagination,
) (reportkpi.DetailPage, error) {
	clean := page.Normalize()

	rows, err := r.db.QueryContext(ctx, query("detail"),
		reportTypeBind(q.ReportType),
		nullableText(q.Adjuster),
		q.Range.From,
		q.Range.To,
		clean.Offset(),
		clean.Size,
	)
	if err != nil {
		return reportkpi.DetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := reportkpi.DetailPage{Rows: []reportkpi.AdjusterDetail{}}
	for rows.Next() {
		var (
			adjuster   sql.NullString
			caseID     sql.NullString
			reportType sql.NullString
			scoredOn   sql.NullTime
			scores     = make([]sql.NullFloat64, len(scoreColumns))
			total      sql.NullInt64
		)

		target := []any{&adjuster, &caseID, &reportType, &scoredOn}
		for i := range scores {
			target = append(target, &scores[i])
		}
		target = append(target, &total)

		if err := rows.Scan(target...); err != nil {
			return reportkpi.DetailPage{}, fmt.Errorf(
				"reportkpi/sqlstore: memindai rincian: %w", err)
		}

		result.Rows = append(result.Rows, reportkpi.AdjusterDetail{
			Adjuster:   adjuster.String,
			CaseID:     caseID.String,
			ReportType: reportkpi.ReportType(reportType.String),
			ScoredOn:   isoDate(scoredOn),
			Scores:     toScores(scores),
		})
		result.Total = int(total.Int64)
	}
	if err := rows.Err(); err != nil {
		return reportkpi.DetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian: %w", err)
	}
	return result, nil
}

// Adjusters mengambil isi dropdown Adjuster.
func (r *Repo) Adjusters(ctx context.Context, q reportkpi.Query) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query("adjusters"),
		reportTypeBind(q.ReportType),
		q.Range.From,
		q.Range.To,
	)
	if err != nil {
		return nil, fmt.Errorf("reportkpi/sqlstore: membaca daftar adjuster: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []string{}
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("reportkpi/sqlstore: memindai adjuster: %w", err)
		}
		if name.Valid && name.String != "" {
			result = append(result, name.String)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reportkpi/sqlstore: membaca daftar adjuster: %w", err)
	}
	return result, nil
}

// SourceState adalah jawaban pemeriksaan kesiapan, dipakai `-periksa`.
type SourceState struct {
	// Rows adalah jumlah baris pada tabel sumber.
	Rows int

	// Types adalah nilai kolom TIPE yang benar-benar ada di basis data.
	Types []string
}

// CheckSource menjawab "tabel sumbernya ada, terbaca, dan berisi apa".
//
// Ia dipakai mode `-periksa`, yang TIDAK MENULIS apa pun. Kedua kueri di dalamnya
// membaca; keduanya digabung supaya satu panggilan menjawab seluruh yang perlu diketahui
// sebelum layar dibuka pertama kali.
func (r *Repo) CheckSource(ctx context.Context) (SourceState, error) {
	var state SourceState

	if err := r.db.QueryRowContext(ctx, query("check_source")).Scan(&state.Rows); err != nil {
		return SourceState{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca %s: %w", reportkpi.SourceTable, err)
	}

	rows, err := r.db.QueryContext(ctx, query("check_distinct_types"))
	if err != nil {
		return SourceState{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca nilai TIPE: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var value sql.NullString
		if err := rows.Scan(&value); err != nil {
			return SourceState{}, fmt.Errorf(
				"reportkpi/sqlstore: memindai nilai TIPE: %w", err)
		}
		if value.Valid {
			state.Types = append(state.Types, value.String)
		}
	}
	if err := rows.Err(); err != nil {
		return SourceState{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca nilai TIPE: %w", err)
	}
	return state, nil
}

// reportTypeBind menerjemahkan tipe laporan menjadi nilai bind.
//
// `ALL` menjadi NULL, dan itu yang membuat SATU kueri melayani ketiga tipe: penyaringnya
// berbentuk `(:1 IS NULL OR k.TIPE = :1)`, sehingga NULL berarti "seluruh tipe" tanpa
// perlu kueri kedua ber-`UNION ALL` seperti di Pega.
func reportTypeBind(t reportkpi.ReportType) any {
	if t == reportkpi.TypeAll {
		return nil
	}
	return string(t)
}

// nullableText menerjemahkan teks kosong menjadi NULL.
//
// Kosong berarti "tanpa penyaring", bukan "cocokkan dengan teks kosong". Keduanya
// menghasilkan hasil yang sangat berbeda, dan yang kedua selalu kosong.
func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// toScores menyusun peta nilai komponen dari hasil pindaian.
//
// Urutannya mengikuti scoreColumns, yang mengikuti urutan SELECT, yang mengikuti urutan
// reportkpi.Components(). Ketiganya diuji kesesuaiannya di query_test.go.
func toScores(values []sql.NullFloat64) map[string]reportkpi.Score {
	codes := reportkpi.ComponentCodes()

	result := make(map[string]reportkpi.Score, len(codes))
	for i, code := range codes {
		if i >= len(values) {
			break
		}
		if values[i].Valid {
			result[code] = reportkpi.NewScore(values[i].Float64)
			continue
		}
		result[code] = reportkpi.EmptyScore()
	}
	return result
}

// isoDate memformat tanggal menjadi `YYYY-MM-DD`, atau kosong bila tidak ada.
//
// Pemformatan dikerjakan DI SINI, bukan di SQL (`08-TECHNICAL-STRATEGY.md` §4.3). Yang
// kosong tetap kosong, tidak diganti tanggal nol — `0001-01-01` di layar jauh lebih
// membingungkan daripada sel yang memang kosong.
func isoDate(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}
