package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/reportkpi"
)

// Penyimpan di memori untuk tab KPI PIC Teknik.
//
// Ia BUKAN tiruan yang menjawab apa pun: ia menyimpan baris dan menyaringnya dengan aturan
// yang sama dengan kueri SQL-nya. Itu yang membuat uji aturan bisnis berjalan tanpa Oracle
// dan tetap membuktikan sesuatu.

// PICTeknikData adalah seluruh bahan tab KPI PIC Teknik untuk satu portal.
type PICTeknikData struct {
	PICs       []reportkpi.PICProfile
	Bands      []reportkpi.Band
	Thresholds []ThresholdRow
	Holidays   []time.Time
	Progress   []ProgressRow
	Analysis   []SpanRow
	Acceptance []AcceptanceRow
	Closure    []ClosureRow
}

// ThresholdRow adalah satu baris kolom `DAY` pada tabel pita.
type ThresholdRow struct {
	Job  string
	Note string
	Days float64
}

// ProgressRow adalah satu pembaruan progres.
//
// Disimpan sebagai PERISTIWA, bukan sebagai cacah jadi — supaya penyaring periodenya benar
// benar diuji, bukan dilewati.
type ProgressRow struct {
	PIC    string
	Line   reportkpi.BusinessLine
	At     time.Time
	OnTime bool
}

// SpanRow adalah satu klaim pada penilaian Analisa.
type SpanRow struct {
	PIC        string
	Line       reportkpi.BusinessLine
	Registered time.Time
	Start      time.Time
	End        time.Time
}

// AcceptanceRow adalah satu klaim pada penilaian Akseptasi.
type AcceptanceRow struct {
	PIC        string
	Line       reportkpi.BusinessLine
	Team       string
	ReceiveLOD time.Time
	Committee  time.Time
	Accepted   time.Time
}

// ClosureRow adalah satu klaim pada penilaian SLA.
type ClosureRow struct {
	PIC        string
	Line       reportkpi.BusinessLine
	Team       string
	Registered time.Time
	Closed     time.Time
}

// WithPICTeknik memasang bahan tab KPI PIC Teknik pada penyimpan.
func (s *Store) WithPICTeknik(data PICTeknikData) *Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.picTeknik = data
	return s
}

// PICs mengembalikan petugas satu lini, terurut.
func (s *Store) PICs(_ context.Context, line reportkpi.BusinessLine) ([]reportkpi.PICProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]reportkpi.PICProfile, 0, len(s.picTeknik.PICs))
	out = append(out, s.picTeknik.PICs...)
	sort.Slice(out, func(i, j int) bool { return out[i].OperatorID < out[j].OperatorID })
	_ = line
	return out, nil
}

// Bands mengembalikan pita satu komponen, menyaring seperti kueri SQL-nya.
func (s *Store) Bands(_ context.Context, job, note string) ([]reportkpi.Band, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []reportkpi.Band
	for _, band := range s.picTeknik.Bands {
		if band.Job != job {
			continue
		}
		if note != "" && band.Note != note {
			continue
		}
		out = append(out, band)
	}
	return out, nil
}

// ThresholdDays mengembalikan ambang hari satu komponen.
func (s *Store) ThresholdDays(_ context.Context, job, note string) (reportkpi.Score, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, row := range s.picTeknik.Thresholds {
		if row.Job != job {
			continue
		}
		if note != "" && row.Note != note {
			continue
		}
		return reportkpi.NewScore(row.Days), nil
	}
	return reportkpi.EmptyScore(), nil
}

// Holidays mengembalikan hari libur pada satu rentang, di luar akhir pekan.
//
// Akhir pekan dibuang di sini persis seperti kueri SQL-nya membuangnya — bukan di pemanggil.
// Kalau tidak, penyimpan ini akan berperilaku berbeda dari Oracle pada kasus yang justru
// paling mudah keliru.
func (s *Store) Holidays(_ context.Context, from, to time.Time) ([]time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []time.Time
	for _, day := range s.picTeknik.Holidays {
		if day.Before(from) || day.After(to) {
			continue
		}
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		out = append(out, day)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out, nil
}

// ProgressCounts mencacah pembaruan progres per PIC pada satu periode.
func (s *Store) ProgressCounts(
	_ context.Context, q reportkpi.PICQuery,
) ([]reportkpi.ProgressCount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := map[string]*reportkpi.ProgressCount{}
	for _, row := range s.picTeknik.Progress {
		if row.Line != q.Line || !withinDays(row.At, q.From, q.To) {
			continue
		}
		entry, seen := counts[row.PIC]
		if !seen {
			entry = &reportkpi.ProgressCount{PIC: row.PIC}
			counts[row.PIC] = entry
		}
		entry.Total++
		if row.OnTime {
			entry.OnTime++
		}
	}

	out := make([]reportkpi.ProgressCount, 0, len(counts))
	for _, entry := range counts {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PIC < out[j].PIC })
	return out, nil
}

// AnalysisSpans mengembalikan pasangan tanggal Analisa pada satu periode.
func (s *Store) AnalysisSpans(
	_ context.Context, q reportkpi.PICQuery,
) ([]reportkpi.DateSpan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []reportkpi.DateSpan
	for _, row := range s.picTeknik.Analysis {
		if row.Line != q.Line || !withinDays(row.Registered, q.From, q.To) {
			continue
		}
		if row.End.IsZero() {
			// Meniru `ANALYST_TFKOMITEDATE IS NOT NULL` pada kuerinya.
			continue
		}
		out = append(out, reportkpi.DateSpan{PIC: row.PIC, Start: row.Start, End: row.End})
	}
	return out, nil
}

// AcceptanceSpans mengembalikan pasangan tanggal Akseptasi pada satu periode.
func (s *Store) AcceptanceSpans(
	_ context.Context, q reportkpi.PICQuery,
) ([]reportkpi.AcceptanceSpan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []reportkpi.AcceptanceSpan
	for _, row := range s.picTeknik.Acceptance {
		if row.Line != q.Line || !withinDays(row.Accepted, q.From, q.To) {
			continue
		}
		out = append(out, reportkpi.AcceptanceSpan{
			PIC:            row.PIC,
			Team:           row.Team,
			ReceiveLOD:     row.ReceiveLOD,
			CommitteeDate:  row.Committee,
			AcceptanceDate: row.Accepted,
		})
	}
	return out, nil
}

// ClosureSpans mengembalikan pasangan tanggal SLA pada satu periode.
func (s *Store) ClosureSpans(
	_ context.Context, q reportkpi.PICQuery,
) ([]reportkpi.ClosureSpan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []reportkpi.ClosureSpan
	for _, row := range s.picTeknik.Closure {
		if row.Line != q.Line || !withinDays(row.Registered, q.From, q.To) {
			continue
		}
		if strings.TrimSpace(row.PIC) == "" {
			continue
		}
		out = append(out, reportkpi.ClosureSpan{
			PIC:          row.PIC,
			Team:         row.Team,
			RegisterDate: row.Registered,
			CloseDate:    row.Closed,
		})
	}
	return out, nil
}

// withinDays menyaring dengan rentang SETENGAH TERBUKA, sama seperti kueri SQL-nya.
//
// Batas atas inklusif pada TANGGALNYA — sebuah peristiwa pukul 23.00 pada hari terakhir
// tetap masuk, persis seperti `< :2 + INTERVAL '1' DAY`.
func withinDays(at, from, to time.Time) bool {
	if at.IsZero() {
		return false
	}
	day := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
	return !day.Before(from) && !day.After(to)
}
