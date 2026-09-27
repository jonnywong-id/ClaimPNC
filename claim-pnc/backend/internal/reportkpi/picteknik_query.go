package reportkpi

import (
	"strings"
	"time"
)

// Nama field penyaring tab KPI PIC Teknik pada kontrak API.
const FieldBusinessLine = "lini_bisnis"

// PICQueryInput adalah isian mentah penyaring tab KPI PIC Teknik.
type PICQueryInput struct {
	// Line adalah pilihan dropdown lini bisnis.
	Line string

	// From dan To adalah batas periode, berbentuk `YYYY-MM-DD`.
	From string
	To   string
}

// PICTeknikQuery adalah penyaring yang sudah dipastikan lengkap.
type PICTeknikQuery struct {
	Line  BusinessLine
	Range DateRange
	Caller
}

// NewPICTeknikQuery memvalidasi isian penyaring tab KPI PIC Teknik.
//
// Seluruh pelanggaran dikumpulkan, tidak berhenti pada yang pertama — meniru perilaku Pega
// yang menampilkan seluruh pesan sekaligus, dan itu bagian dari kesetaraan (`P-5`), bukan
// pilihan gaya.
//
// Periode WAJIB, sama seperti kedua tab lain: `PNCReportKPI_act` menolak dengan pesan
// "Periode tanggal masih kosong" sebelum satu kueri pun dijalankan.
func NewPICTeknikQuery(input PICQueryInput, caller Caller) (PICTeknikQuery, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return PICTeknikQuery{}, ErrCallerUnknown
	}

	violations := []Violation{}

	rawLine := strings.TrimSpace(input.Line)
	line, knownLine := FindBusinessLine(rawLine)
	if !knownLine {
		violations = append(violations, Violation{
			Field:   FieldBusinessLine,
			Message: messageForBusinessLine(rawLine),
		})
	}

	from := strings.TrimSpace(input.From)
	to := strings.TrimSpace(input.To)

	fromDate, fromValid := parseDate(from)
	if !fromValid {
		violations = append(violations, Violation{
			Field:   FieldDateFrom,
			Message: messageForDate(from, "Dari"),
		})
	}

	toDate, toValid := parseDate(to)
	if !toValid {
		violations = append(violations, Violation{
			Field:   FieldDateTo,
			Message: messageForDate(to, "Sampai"),
		})
	}

	if fromValid && toValid && fromDate.After(toDate) {
		violations = append(violations, Violation{
			Field:   FieldDateTo,
			Message: "Tanggal \"Sampai\" tidak boleh lebih awal daripada \"Dari\".",
		})
	}

	if len(violations) > 0 {
		return PICTeknikQuery{}, NewValidationError(violations)
	}

	return PICTeknikQuery{
		Line:   line.Code,
		Range:  DateRange{From: from, To: to},
		Caller: cleanCaller,
	}, nil
}

// Span mengembalikan penyaring yang dipakai lapisan penyimpanan.
func (q PICTeknikQuery) Span() PICQuery {
	from, _ := parseDate(q.Range.From)
	to, _ := parseDate(q.Range.To)
	return PICQuery{Line: q.Line, From: from, To: to}
}

// Dates mengembalikan kedua batas periode sebagai tanggal.
func (q PICTeknikQuery) Dates() (from, to time.Time) {
	from, _ = parseDate(q.Range.From)
	to, _ = parseDate(q.Range.To)
	return from, to
}

// messageForBusinessLine membedakan "belum dipilih" dari "tidak dikenal".
func messageForBusinessLine(raw string) string {
	if raw == "" {
		return "Lini bisnis wajib dipilih."
	}

	known := make([]string, 0, len(businessLines))
	for _, line := range businessLines {
		known = append(known, string(line.Code))
	}
	return "Lini bisnis \"" + raw + "\" tidak dikenal. Pilihan: " +
		strings.Join(known, ", ") + "."
}
