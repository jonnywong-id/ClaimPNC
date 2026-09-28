package reportkpi

import (
	"strings"
	"time"
)

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// ReportType adalah pilihan dropdown "Pilih Tipe Report".
	ReportType string

	// Adjuster adalah pilihan dropdown "Pilih Adjuster". Kosong berarti SELURUH adjuster.
	Adjuster string

	// From dan To adalah batas periode, berbentuk `YYYY-MM-DD`.
	From string
	To   string
}

// DateRange adalah rentang periode yang sudah dipastikan lengkap dan berurutan.
//
// Isinya tetap TEKS `YYYY-MM-DD`, bukan time.Time, dan itu disengaja: satu-satunya yang
// memakainya adalah kueri, dan kueri membandingkannya sebagai tanggal kalender —
// membawanya sebagai time.Time menambahkan jam, menit, dan zona waktu yang tidak satu pun
// berarti di sini, lalu mengundang pertanyaan zona waktu yang tidak perlu dijawab.
type DateRange struct {
	From string
	To   string
}

// Query adalah permintaan yang sudah tervalidasi.
type Query struct {
	// ReportType selalu salah satu dari ketiga tipe yang dikenal.
	ReportType ReportType

	// Adjuster adalah nama adjuster yang dipilih. Kosong berarti seluruh adjuster.
	//
	// Ia dibandingkan PERSIS — `and adjuster = '<pilihan>'` di sistem lama — bukan dengan
	// `LIKE`. Nilainya datang dari dropdown, bukan dari ketikan bebas, sehingga pencocokan
	// sebagian hanya akan membuat satu pilihan menarik baris milik adjuster lain yang
	// namanya berawalan sama.
	Adjuster string

	// Range adalah periode penilaian yang disaring terhadap kolom `TANGGAL`.
	Range DateRange

	// Caller adalah identitas pemanggil, dipakai jejak log.
	Caller Caller
}

// dateLayout adalah bentuk tanggal yang diterima kontrak API.
//
// Bentuk ISO, bukan `dd/mm/yyyy` seperti di layar lama. Alasannya: yang mengirimnya adalah
// kontrak API, bukan layar Pega, dan `dd/mm/yyyy` tidak dapat dibedakan dari `mm/dd/yyyy`
// oleh pembacanya — satu kekeliruan yang menghasilkan rentang yang sah tetapi salah, tanpa
// satu pun galat. Penerjemahannya ke bentuk yang dimengerti basis data dikerjakan
// penyimpanan.
const dateLayout = "2006-01-02"

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// # Kenapa TIPE REPORT wajib
//
// Karena tanpa itu tidak ada yang dapat dibaca. `PNCReportKPIAdjuster_act` menjaga
// langkahnya dengan precondition `TempAdjComp.ASMFull==""||TempAdjComp.AcceptedNo==""`,
// dan dropdown-nya berbunyi "--Pilih--" — keduanya menyatakan hal yang sama: ia pilihan
// yang harus diambil, bukan yang punya nilai bawaan.
//
// # Kenapa PERIODE wajib, padahal layar lama tidak selalu memintanya
//
// Karena di layar lama, mengosongkannya MERUSAK. Kedua tanggal disisipkan langsung ke
// dalam teks SQL —
//
//	"and trunc(TANGGAL)>=to_date('" + awal + "','dd/mm/yyyy')"
//
// — sehingga tanggal kosong menghasilkan `to_date(<kosong>, 'dd/mm/yyyy')` dan galat basis
// data mentah yang sampai ke pengguna. Preconditionnya hanya menjaga tipe FINAL dan ALL;
// OUTSTANDING tidak dijaga sama sekali.
//
// Yang dilakukan di sini adalah memeriksanya LEBIH DULU dan menjawab dengan kalimat yang
// menyebut isian mana yang kurang. Itu selisih pada CARA GALAT DISAMPAIKAN, bukan pada
// hasil: rentang yang sah menghasilkan baris yang sama persis. Pola yang sama sudah
// dipakai modul Inbox RCL/PUCL pada laporan hariannya.
//
// # Kenapa SELURUH pelanggaran dikumpulkan
//
// Mengikuti `P-5` dan `11-CROSSCUTTING.md` §1.1. Pengguna yang menekan "Cari" dengan form
// kosong diberi tahu ketiganya sekaligus, bukan satu lalu satu lagi.
func NewQuery(input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	violations := []Violation{}

	rawType := strings.TrimSpace(input.ReportType)
	reportType, knownType := FindReportType(rawType)
	if !knownType {
		violations = append(violations, Violation{
			Field:   FieldReportType,
			Message: messageForReportType(rawType),
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

	// Urutan diperiksa hanya bila KEDUANYA terbaca. Memeriksanya lebih awal menghasilkan
	// pelanggaran ketiga yang membingungkan — pengguna diberi tahu urutannya salah padahal
	// yang salah adalah bentuk tanggalnya.
	if fromValid && toValid && fromDate.After(toDate) {
		violations = append(violations, Violation{
			Field:   FieldDateTo,
			Message: "Tanggal \"Sampai\" tidak boleh lebih awal daripada \"Dari\".",
		})
	}

	if len(violations) > 0 {
		return Query{}, NewValidationError(violations)
	}

	return Query{
		ReportType: reportType,
		Adjuster:   strings.TrimSpace(input.Adjuster),
		Range:      DateRange{From: from, To: to},
		Caller:     cleanCaller,
	}, nil
}

// AdminQueryInput adalah isian mentah penyaring tab KPI Admin.
//
// Ia jauh lebih sedikit daripada tab KPI Adjuster, dan itu mengikuti layar lama: tab ini
// hanya punya "Pilih Data KPI" dan sepasang tanggal. Tidak ada pilihan orang — kelompoknya
// sudah menentukan siapa yang dihitung, lewat enam Operator ID yang ditulis di dalam kueri.
type AdminQueryInput struct {
	// Group adalah pilihan dropdown "Pilih Data KPI".
	Group string

	// From dan To adalah batas periode, berbentuk `YYYY-MM-DD`.
	From string
	To   string
}

// AdminQuery adalah permintaan tab KPI Admin yang sudah tervalidasi.
type AdminQuery struct {
	Group  AdminGroup
	Range  DateRange
	Caller Caller
}

// NewAdminQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// # Kenapa periode WAJIB di sini pula
//
// Karena layar lama memang menolak tanpanya, dan penolakannya ditulis terang-terangan:
// `Activity/PNCReportKPIAdmin_Act-Act.xml` menyetel `local.errmsg := "Periode tanggal masih
// kosong"`. Ini satu-satunya validasi yang benar-benar ADA di layar lama — di tab KPI
// Adjuster ia harus kami adakan sendiri.
//
// Pesannya di sini dibuat lebih spesifik: ia menyebut isian MANA yang kurang, sedangkan
// pesan lama menyebut keduanya sekaligus tanpa membedakan.
func NewAdminQuery(input AdminQueryInput, caller Caller) (AdminQuery, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return AdminQuery{}, ErrCallerUnknown
	}

	violations := []Violation{}

	rawGroup := strings.TrimSpace(input.Group)
	group, knownGroup := FindAdminGroup(rawGroup)
	if !knownGroup {
		violations = append(violations, Violation{
			Field:   FieldAdminGroup,
			Message: messageForAdminGroup(rawGroup),
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
		return AdminQuery{}, NewValidationError(violations)
	}

	return AdminQuery{Group: group, Range: DateRange{From: from, To: to}, Caller: cleanCaller}, nil
}

// messageForAdminGroup membedakan "belum dipilih" dari "tidak dikenal".
func messageForAdminGroup(raw string) string {
	if raw == "" {
		return "Data KPI wajib dipilih."
	}

	known := make([]string, 0, len(adminGroups))
	for _, g := range adminGroups {
		known = append(known, string(g.Code))
	}
	return "Data KPI \"" + raw + "\" tidak dikenal. Pilihan: " +
		strings.Join(known, ", ") + "."
}

// parseDate membaca satu batas tanggal.
//
// Ia memakai `time.Parse` yang MENOLAK tanggal yang tidak ada — `2026-02-30` gagal di
// sini, sementara pemeriksaan berbasis pola akan meloloskannya lalu menyerahkannya ke
// basis data.
func parseDate(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(dateLayout, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// messageForDate membedakan "belum diisi" dari "salah bentuk".
//
// Keduanya dipisah karena tindakan pengguna berbeda: yang pertama menuntut ia mengisi,
// yang kedua menuntut ia memperbaiki. Pesan yang sama untuk keduanya membuat pengguna yang
// sudah mengisi mengira isiannya tidak terkirim.
func messageForDate(raw, label string) string {
	if raw == "" {
		return "Periode \"" + label + "\" wajib diisi."
	}
	return "Periode \"" + label + "\" tidak terbaca. Gunakan bentuk tahun-bulan-tanggal, " +
		"misalnya 2026-09-24."
}

// messageForReportType membedakan "belum dipilih" dari "tidak dikenal".
func messageForReportType(raw string) string {
	if raw == "" {
		return "Tipe report wajib dipilih."
	}

	known := make([]string, 0, len(reportTypes))
	for _, t := range reportTypes {
		known = append(known, string(t.Code))
	}
	return "Tipe report \"" + raw + "\" tidak dikenal. Pilihan: " +
		strings.Join(known, ", ") + "."
}
