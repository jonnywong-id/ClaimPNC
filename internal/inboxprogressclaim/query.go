package inboxprogressclaim

import (
	"strings"
	"time"
)

// BusinessLine adalah lini bisnis yang menyaring rekap Progress Klaim per PIC.
//
// # Dari mana keempatnya, dan kenapa TIDAK ada pilihan "semua"
//
// Dari `Activity/GetProgressPerPIC-Act.xml` langkah 2–5, yang bercabang menjadi empat dan
// tidak punya cabang kelima. Predikat aslinya, apa adanya:
//
//	NONMBU   GROUP_PANEL IN ('003','004','006')
//	         AND groupbisnisid NOT IN ('09','11','16','25')
//	TRAVEL   GROUP_PANEL IN ('005')
//	BONDING  groupbisnisid IN ('09','11','16','25')
//	PA       GROUP_PANEL IN ('002')
//
// Tidak ada cabang tanpa saringan, dan itu bukan kelalaian: nilai yang sama juga dipakai
// mencocokkan `MST_USER_TEKNIK.TYPE_BUSINESS`, sehingga tanpa lini bisnis kueri tidak
// menemukan satu pun petugas. Karena itu lini bisnis WAJIB diisi pada region ini.
//
// # Keempat predikat ini dipilih Work Owner, dan itu keputusan atas dua versi yang berbeda
//
// Sistem lama memuat DUA definisi lini bisnis yang tidak sama:
//
//	                 jalur grid Outstanding        jalur Export & per PIC
//	NONMBU           groupbisnisid NOT IN          GROUP_PANEL IN ('003','004','006')
//	                 ('06','09','11','16')         AND groupbisnisid NOT IN ('09','11','16','25')
//	BONDING          groupbisnisid IN ('11','16')  groupbisnisid IN ('09','11','16','25')
//	PA               groupbisnisid IN ('06')       GROUP_PANEL IN ('002')
//	TRAVEL           tidak ada                     GROUP_PANEL IN ('005')
//
// Keputusan Work Owner 2026-09-21: yang dipakai versi kanan. Ia satu-satunya yang
// benar-benar dieksekusi di Pega — versi kiri menulis ke `tempgetpic.CaseID`, properti yang
// tidak dibaca kueri mana pun, sehingga tidak pernah berpengaruh sejak awal.
type BusinessLine string

// Keempat lini bisnis.
//
// Nilainya huruf besar persis seperti yang dibandingkan activity lama, dan itu bukan gaya
// penulisan melainkan kontrak: ia dikirim layar sebagai parameter query, dan nilai yang
// sama dicocokkan ke `MST_USER_TEKNIK.TYPE_BUSINESS`.
const (
	BusinessNonMBU  BusinessLine = "NONMBU"
	BusinessTravel  BusinessLine = "TRAVEL"
	BusinessBonding BusinessLine = "BONDING"
	BusinessPA      BusinessLine = "PA"
)

// businessLines adalah keempatnya dalam urutan cabang di activity lama.
var businessLines = []BusinessLine{
	BusinessNonMBU, BusinessTravel, BusinessBonding, BusinessPA,
}

// BusinessLines mengembalikan isi dropdown lini bisnis.
func BusinessLines() []BusinessLine {
	result := make([]BusinessLine, len(businessLines))
	copy(result, businessLines)
	return result
}

// Label adalah teks yang dibaca pengguna pada dropdown.
func (b BusinessLine) Label() string {
	switch b {
	case BusinessNonMBU:
		return "Non-MBU"
	case BusinessTravel:
		return "Travel"
	case BusinessBonding:
		return "Bonding"
	case BusinessPA:
		return "Personal Accident"
	default:
		return string(b)
	}
}

// ParseBusinessLine membaca lini bisnis dari sebuah teks.
//
// Nilai kosong dan nilai yang tidak dikenal sama-sama menghasilkan false. Berbeda dengan
// Inbox Admin, di sini kosong BUKAN berarti "semua" — region ini memang tidak punya pilihan
// semua.
func ParseBusinessLine(raw string) (BusinessLine, bool) {
	value := BusinessLine(strings.ToUpper(strings.TrimSpace(raw)))
	for _, known := range businessLines {
		if known == value {
			return value, true
		}
	}
	return "", false
}

// dateLayout adalah bentuk tanggal pada kontrak API: YYYY-MM-DD.
//
// Sistem lama memakai `dd/mm/yyyy` karena ia merangkainya langsung ke dalam teks SQL. Di
// sini tanggal adalah nilai yang di-bind, dan bentuk ISO dipakai supaya pengurutan teks dan
// pengurutan tanggal tidak lagi berbeda (`09-DATABASE-STRATEGY.md` §3.2).
const dateLayout = "2006-01-02"

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// View adalah kode region yang diminta. Kosong berarti DefaultView.
	View string

	// Keyword adalah isi kotak cari. Hanya berlaku pada region klaim.
	Keyword string

	// Business adalah lini bisnis. Hanya berlaku — dan WAJIB — pada region per PIC.
	Business string

	// From dan To adalah rentang tanggal registrasi, berbentuk `YYYY-MM-DD`. Hanya
	// berlaku pada region per PIC, dan boleh kosong keduanya.
	From string
	To   string
}

// Query adalah permintaan isi satu region yang sudah tervalidasi.
type Query struct {
	// View adalah region yang diminta, lengkap dengan kolom dan kemampuannya.
	View View

	// Keyword adalah kata kunci yang BENAR-BENAR dipakai, sudah dipangkas.
	//
	// Ia selalu kosong pada region yang tidak mendukung pencarian. Tanpa penetapan itu,
	// dua permintaan yang hasilnya pasti sama akan tampak berbeda di log dan di kunci
	// cache.
	Keyword string

	// Business adalah lini bisnis yang dipakai. Kosong pada region klaim.
	Business BusinessLine

	// From dan To adalah rentang tanggal registrasi yang dipakai, atau nil.
	From *time.Time
	To   *time.Time

	// Caller adalah identitas pemanggil.
	Caller Caller
}

// ClaimQuery adalah permintaan isi region klaim, sebagaimana diterima Repo.
//
// Ia lebih sempit daripada Query dengan sengaja: repo region klaim tidak boleh dapat
// membaca lini bisnis maupun rentang tanggal, karena keduanya memang tidak menyaring di
// sana.
type ClaimQuery struct {
	// View adalah kode region — ViewOutstanding atau ViewNextFollowUp. Keduanya memakai
	// kueri yang sama dengan satu saringan tambahan pada yang kedua.
	View string

	// Keyword menyaring nomor klaim, nomor polis, dan nama PIC sekaligus — persis seperti
	// di sistem lama.
	Keyword string

	// Today adalah tanggal hari ini menurut seam Clock, dipakai region Next Follow Up
	// untuk memutuskan mana yang sudah jatuh tempo.
	//
	// Ia DIBAWA sebagai nilai, bukan dibaca kueri dari `SYSDATE`. Dua alasan: hasilnya
	// dapat diuji secara deterministik, dan `SYSDATE` adalah jam server basis data yang
	// tidak sama dengan tanggal WIB yang dipakai aturan bisnis (`F-5`).
	Today time.Time
}

// PICQuery adalah permintaan isi rekap per PIC, sebagaimana diterima Repo.
type PICQuery struct {
	// Business adalah lini bisnis. Selalu terisi — region ini menolak permintaan tanpa
	// lini bisnis.
	Business BusinessLine

	// From dan To adalah rentang tanggal registrasi, atau nil bila tidak disaring.
	//
	// Sistem lama menerapkan keduanya sekaligus atau tidak sama sekali: prakondisinya
	// `TempRefresh.Remark=="" && TempRefresh.City==""`, sehingga satu isian yang terisi
	// sudah cukup menyalakan saringannya — dan isian yang kosong menjadi tanggal kosong
	// di dalam `to_date`, yang pada Oracle menghasilkan galat. Di sini keduanya dinilai
	// terpisah dan yang kosong berarti tanpa batas.
	From *time.Time
	To   *time.Time

	// Caller adalah identitas pemanggil; kuerinya menyaring `pic` menurut login ini.
	Caller Caller

	// Today adalah tanggal hari ini menurut seam Clock, dipakai pencacah "jatuh tempo
	// hari ini". Alasannya sama dengan ClaimQuery.Today.
	Today time.Time
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
func NewQuery(input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	code := strings.TrimSpace(input.View)
	if code == "" {
		code = DefaultView
	}

	view, known := FindView(code)
	if !known {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldView,
			Message: "Bagian layar tidak dikenal.",
		}})
	}

	query := Query{View: view, Caller: cleanCaller}
	violations := []Violation{}

	if view.SupportsSearch {
		query.Keyword = strings.TrimSpace(input.Keyword)
	}

	if view.SupportsBusinessFilter {
		business, valid := ParseBusinessLine(input.Business)
		switch {
		case strings.TrimSpace(input.Business) == "":
			violations = append(violations, Violation{
				Field: FieldBusiness,
				Message: "Pilih lini bisnis lebih dulu. Rekap per PIC di sistem lama " +
					"selalu terikat satu lini bisnis, dan tanpa itu tidak ada petugas " +
					"yang cocok.",
			})
		case !valid:
			violations = append(violations, Violation{
				Field:   FieldBusiness,
				Message: "Pilihan lini bisnis tidak dikenal.",
			})
		default:
			query.Business = business
		}
	}

	if view.SupportsDateRange {
		from, err := parseDate(input.From, FieldFrom)
		if err != nil {
			violations = append(violations, *err)
		} else {
			query.From = from
		}

		to, err := parseDate(input.To, FieldTo)
		if err != nil {
			violations = append(violations, *err)
		} else {
			query.To = to
		}

		// Rentang terbalik ditolak, bukan ditukar diam-diam. Menukarnya berarti menjawab
		// pertanyaan yang tidak diajukan, dan pengguna tidak pernah tahu bahwa yang ia
		// ketik bukan yang ia lihat.
		if query.From != nil && query.To != nil && query.To.Before(*query.From) {
			violations = append(violations, Violation{
				Field:   FieldTo,
				Message: "Tanggal akhir mendahului tanggal awal.",
			})
		}
	}

	if len(violations) > 0 {
		return Query{}, NewValidationError(violations)
	}

	return query, nil
}

// ClaimQuery menurunkan permintaan region klaim dari permintaan yang sudah tervalidasi.
//
// `now` datang dari seam Clock dan dipangkas ke tanggalnya saja: aturan "jatuh tempo hari
// ini" berbasis HARI kalender, bukan jam (`08-TECHNICAL-STRATEGY.md` §4.4).
func (q Query) ClaimQuery(now time.Time) ClaimQuery {
	return ClaimQuery{View: q.View.Code, Keyword: q.Keyword, Today: startOfDay(now)}
}

// PICQuery menurunkan permintaan rekap per PIC dari permintaan yang sudah tervalidasi.
func (q Query) PICQuery(now time.Time) PICQuery {
	return PICQuery{
		Business: q.Business,
		From:     q.From,
		To:       q.To,
		Caller:   q.Caller,
		Today:    startOfDay(now),
	}
}

// startOfDay memangkas sebuah waktu menjadi tengah malam pada tanggal yang sama.
//
// Tanpa pemangkasan ini, "jatuh tempo hari ini" akan berarti "jatuh tempo sebelum jam
// sekian hari ini", sehingga baris yang sama muncul dan menghilang tergantung kapan layar
// dibuka.
func startOfDay(at time.Time) time.Time {
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
}

// parseDate membaca satu isian tanggal.
//
// Isian kosong menghasilkan nil tanpa galat: rentang yang hanya berbatas satu sisi memang
// sah, dan pada sistem lama isian yang kosong berarti saringannya tidak dipasang.
func parseDate(raw, field string) (*time.Time, *Violation) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		return nil, &Violation{
			Field:   field,
			Message: "Tanggal tidak terbaca. Bentuk yang diterima YYYY-MM-DD.",
		}
	}
	return &parsed, nil
}
