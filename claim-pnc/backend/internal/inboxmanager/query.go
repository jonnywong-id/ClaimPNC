package inboxmanager

import (
	"strings"
	"time"
)

// PeriodMode membedakan kedua bentuk penyaring periode.
//
// Keduanya dibaca dari `Activity/PNCGetDashboardProduktivitasInbox_Act`, yang menyusun
// predikatnya dalam dua bentuk yang berbeda:
//
//	bulan   to_char(tglklaim,'mm-rrrr') = to_char(<satu tanggal>,'mm-rrrr')
//	rentang trunc(tglklaim) >= <dari> AND trunc(tglklaim) <= <sampai>
//
// Section-nya pun menawarkan keduanya: `<pyCaption Periode>`, `<pyCaption Bulan & Tahun>`,
// `<pyCaption Dari>`, dan `<pyCaption Sampai>`.
type PeriodMode string

const (
	// PeriodMonth menyaring satu bulan takwim.
	PeriodMonth PeriodMode = "bulan"

	// PeriodRange menyaring rentang tanggal.
	PeriodRange PeriodMode = "rentang"
)

// Period adalah penyaring periode yang sudah dibetulkan menjadi SELANG SETENGAH TERBUKA.
//
// # Kenapa selang setengah terbuka, bukan dua fungsi tanggal
//
// Karena kedua bentuk Pega menyentuh kolomnya pada setiap baris — `to_char(tglklaim,…)` dan
// `trunc(tglklaim)` — sehingga index atas kolom itu tidak dapat dipakai. Keduanya juga
// dilarang `08-TECHNICAL-STRATEGY.md` §4.3.
//
// Penggantinya `TGLKLAIM >= :dari AND TGLKLAIM < :sampai`, dengan kedua batas dihitung di Go.
// Hasilnya identik dan kolomnya tetap dapat di-index.
//
// # Kenapa batas atasnya EKSKLUSIF
//
// Karena `TGLKLAIM` bertipe `TIMESTAMP(6)`, bukan `DATE` tanpa jam. Batas atas yang inklusif
// (`<= 30 September`) akan membuang seluruh baris tanggal 30 September yang berjam selain
// tengah malam — cacat yang tidak menghasilkan satu pun pesan galat dan hanya terlihat bila
// seseorang menghitung ulang dengan tangan.
//
// Pega tidak mengalaminya karena ia membungkus kolomnya dengan `trunc()`. Kita membayar
// kesetaraan itu dengan menggeser batas atas satu hari, bukan dengan ikut membungkus kolom.
type Period struct {
	// From adalah batas bawah, INKLUSIF.
	From time.Time

	// Until adalah batas atas, EKSKLUSIF.
	Until time.Time

	// PriorFrom dan PriorUntil adalah selang yang sama satu tahun sebelumnya.
	//
	// Keduanya ikut dihitung di sini, bukan di SQL, karena di Pega pun begitu — predikat
	// pembandingnya disusun activity dengan `- INTERVAL '1' YEAR`. Menghitungnya di Go
	// membuat kuerinya tidak perlu mengenal aritmetika tanggal sama sekali.
	PriorFrom  time.Time
	PriorUntil time.Time
}

// Empty menyatakan periode yang tidak diisi sama sekali.
func (p Period) Empty() bool {
	return p.From.IsZero() && p.Until.IsZero()
}

// PeriodInput adalah isian periode mentah dari layar.
type PeriodInput struct {
	// Mode memilih bentuk penyaringnya.
	Mode PeriodMode

	// Month adalah bulan yang dipilih pada mode bulan, berbentuk "YYYY-MM".
	Month string

	// From dan Until adalah batas pada mode rentang, berbentuk "YYYY-MM-DD".
	From  string
	Until string
}

// Blank menyatakan tidak ada satu pun isian periode yang dikirim.
func (p PeriodInput) Blank() bool {
	return strings.TrimSpace(p.Month) == "" &&
		strings.TrimSpace(p.From) == "" &&
		strings.TrimSpace(p.Until) == ""
}

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja: Query
// hanya lahir lewat NewQuery, dan di situlah kewenangan tab diperiksa.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti tab pertama yang boleh dilihat
	// pemanggil.
	Tab string

	// Period adalah penyaring periode. Diabaikan pada tab yang tidak punya penyaring itu.
	Period PeriodInput

	// Page adalah paginasi antrean. Diabaikan pada tab dashboard dan tab ringkasan.
	Page Pagination

	// Reinsurer menyaring grid ketiga tab Outstanding — `LEADER`, `MEMBER`, atau `FAC-IN`.
	// Kosong berarti seluruhnya, sama seperti pilihan "All" di layar lama.
	Reinsurer string

	// CategoryOS menyaring grid ketiga menurut tahapan progres klaim. Kosong berarti
	// seluruhnya.
	CategoryOS string
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolom dan aturan keputusannya.
	Tab Tab

	// Period adalah periode yang sudah dibetulkan. Kosong pada tab tanpa penyaring periode.
	Period Period

	// Page adalah paginasi antrean yang sudah dibetulkan.
	Page Pagination

	// LineBusiness adalah lini bisnis yang menyaring dashboard — padanan
	// `OperatorID.pyPosition` sistem lama.
	//
	// Ia disalin dari Caller supaya repo tidak perlu tahu apa pun tentang identitas: yang
	// dibutuhkannya hanyalah nilai penyaringnya.
	LineBusiness string

	// Reinsurer dan CategoryOS menyaring grid ketiga tab Outstanding. Kosong berarti
	// seluruhnya.
	//
	// Keduanya TIDAK divalidasi terhadap daftar pilihan. Alasannya bukan kelonggaran:
	// "Kategori OS" dibaca dari master yang dapat berubah tanpa deploy, dan menolak nilai
	// yang tidak dikenal berarti layar yang terbuka sejak sebelum master berubah menjadi
	// gagal — bukan sekadar kosong. Nilai yang tidak cocok apa pun menghasilkan grid
	// kosong, dan itu jawaban yang benar.
	Reinsurer  string
	CategoryOS string

	// Caller adalah identitas pemanggil.
	Caller Caller
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// # Urutan pemeriksaannya disengaja
//
// Identitas lebih dulu, lalu tab yang diminta beserta kewenangannya, baru isinya. Dengan
// urutan itu pengguna yang salah alamat memperoleh jawaban yang menjelaskan keadaannya —
// bukan "periode tidak sah" untuk tab yang sebenarnya memang tidak boleh ia buka.
func NewQuery(input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	visible := VisibleTabs(cleanCaller)
	if len(visible) == 0 {
		return Query{}, ErrTabNotAllowed
	}

	code := strings.TrimSpace(input.Tab)
	if code == "" {
		code = DefaultTabFor(cleanCaller)
	}

	tab, known := FindTab(code)
	if !known {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Tab tidak dikenal.",
		}})
	}

	// Kewenangan ditegakkan DI SINI, bukan hanya dengan menyembunyikan tab di layar.
	// Tautan lama yang masih menyimpan kode tab di riwayat peramban akan sampai ke sini,
	// dan begitu pula permintaan yang disusun tangan.
	if !CanSee(cleanCaller, tab) {
		return Query{}, ErrTabNotAllowed
	}

	// Ukuran halaman tab dipakai hanya bila pemanggil TIDAK menyebut ukurannya sendiri.
	// Dengan begitu `?ukuran=` tetap berlaku, dan tab yang tidak menyebut ukurannya jatuh ke
	// DefaultPageSize seperti sebelumnya.
	page := input.Page
	if page.Size <= 0 && tab.PageSize > 0 {
		page.Size = tab.PageSize
	}

	query := Query{
		Tab:          tab,
		Page:         page.Normalize(),
		LineBusiness: cleanCaller.LineBusiness,
		Reinsurer:    strings.ToUpper(strings.TrimSpace(input.Reinsurer)),
		CategoryOS:   strings.TrimSpace(input.CategoryOS),
		Caller:       cleanCaller,
	}

	if !tab.HasPeriodFilter {
		// Periode yang dikirim untuk tab yang tidak punya penyaringnya DIBUANG, bukan
		// ditolak: layar tidak menggambar isiannya. Menolaknya akan menggagalkan permintaan
		// yang sebenarnya tidak salah.
		return query, nil
	}

	period, err := parsePeriod(input.Period, tab.RangeSameYearOnly)
	if err != nil {
		return Query{}, err
	}
	query.Period = period

	return query, nil
}

// parsePeriod membetulkan isian periode menjadi selang setengah terbuka, beserta selang yang
// sama satu tahun sebelumnya.
//
// Isian kosong BUKAN galat: kedua dashboard berperiode tetap bermakna tanpa penyaring — di
// Pega pun predikatnya diisi teks kosong bila pengguna belum memilih apa pun
// (`PNCGetDashboardProduktivitasInbox_Act`, Property-Set bernilai `""`).
func parsePeriod(input PeriodInput, sameYearOnly bool) (Period, error) {
	if input.Blank() {
		return Period{}, nil
	}

	mode := PeriodMode(strings.TrimSpace(string(input.Mode)))
	if mode == "" {
		mode = PeriodMonth
		if strings.TrimSpace(input.From) != "" || strings.TrimSpace(input.Until) != "" {
			mode = PeriodRange
		}
	}

	switch mode {
	case PeriodMonth:
		return parseMonth(input.Month)
	case PeriodRange:
		return parseRange(input.From, input.Until, sameYearOnly)
	default:
		return Period{}, NewValidationError([]Violation{{
			Field:   FieldPeriod,
			Message: "Bentuk periode hanya boleh \"bulan\" atau \"rentang\".",
		}})
	}
}

// parseMonth membentuk selang satu bulan takwim dari isian "YYYY-MM".
func parseMonth(value string) (Period, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return Period{}, NewValidationError([]Violation{{
			Field:   FieldPeriod,
			Message: "Bulan & Tahun wajib diisi.",
		}})
	}

	at, err := time.Parse("2006-01", clean)
	if err != nil {
		return Period{}, NewValidationError([]Violation{{
			Field:   FieldPeriod,
			Message: "Bulan & Tahun tidak dikenali. Contoh yang benar: 2026-09.",
		}})
	}

	from := time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, time.UTC)
	return periodWithPrior(from, from.AddDate(0, 1, 0)), nil
}

// parseRange membentuk selang dari sepasang tanggal "YYYY-MM-DD".
func parseRange(fromValue, untilValue string, sameYearOnly bool) (Period, error) {
	violations := []Violation{}

	from, okFrom := parseDate(fromValue)
	if !okFrom {
		violations = append(violations, Violation{
			Field:   FieldPeriod,
			Message: "Tanggal \"Dari\" tidak dikenali. Contoh yang benar: 2026-09-01.",
		})
	}

	until, okUntil := parseDate(untilValue)
	if !okUntil {
		violations = append(violations, Violation{
			Field:   FieldPeriod,
			Message: "Tanggal \"Sampai\" tidak dikenali. Contoh yang benar: 2026-09-30.",
		})
	}

	// Rentang yang melintasi tahun ditolak pada tab yang Pega pun menolaknya, dengan pesan
	// yang SAMA PERSIS — `Activity/DashboardKlaim_act` langkah 4 menetapkannya sebagai
	// `local.message`, dan langkah 5 melompat ke blok galat bila kedua tahun berbeda.
	if okFrom && okUntil && sameYearOnly && from.Year() != until.Year() {
		violations = append(violations, Violation{
			Field:   FieldPeriod,
			Message: MessageRangeSameYear,
		})
	}

	if okFrom && okUntil && until.Before(from) {
		violations = append(violations, Violation{
			Field:   FieldPeriod,
			Message: "Tanggal \"Sampai\" tidak boleh mendahului tanggal \"Dari\".",
		})
	}

	if len(violations) > 0 {
		return Period{}, NewValidationError(violations)
	}

	// Batas atas digeser satu hari supaya ia EKSKLUSIF sementara yang dipilih pengguna
	// tetap ikut terhitung. Lihat catatan pada Period.
	return periodWithPrior(from, until.AddDate(0, 0, 1)), nil
}

// parseDate membaca satu tanggal "YYYY-MM-DD".
func parseDate(value string) (time.Time, bool) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return time.Time{}, false
	}
	at, err := time.Parse("2006-01-02", clean)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC), true
}

// periodWithPrior melengkapi sebuah selang dengan selang yang sama satu tahun sebelumnya.
//
// Pengurangannya memakai AddDate, bukan pengurangan 365 hari: tahun kabisat membuat keduanya
// berbeda satu hari, dan yang ditiru adalah `- INTERVAL '1' YEAR` milik Oracle yang juga
// bekerja pada takwim, bukan pada jumlah hari.
func periodWithPrior(from, until time.Time) Period {
	return Period{
		From:       from,
		Until:      until,
		PriorFrom:  from.AddDate(-1, 0, 0),
		PriorUntil: until.AddDate(-1, 0, 0),
	}
}
