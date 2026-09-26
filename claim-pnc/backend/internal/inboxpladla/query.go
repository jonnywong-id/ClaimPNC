package inboxpladla

import (
	"errors"
	"strings"
)

// Galat domain modul Inbox PLA DLA.
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Di layar ini ia BUKAN sekadar soal jejak: login pemanggil adalah penyaring utama
	// ketiga daftarnya. Tanpa login, tidak ada daftar yang dapat disusun sama sekali.
	ErrCallerUnknown = errors.New("inboxpladla: identitas pemanggil tidak terbaca")

	// ErrWriteNotAvailable berarti aksi yang belum dibangun diminta.
	ErrWriteNotAvailable = errors.New("inboxpladla: tindakan ini belum tersedia")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
const (
	FieldTab    = "daftar"
	FieldSearch = "cari"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja (`P-5`).
type ValidationError struct {
	Violations []Violation
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
func NewValidationError(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: violations}
}

// Error menyusun pesan ringkas untuk log.
func (e *ValidationError) Error() string {
	if len(e.Violations) == 0 {
		return "inboxpladla: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "inboxpladla: " + strings.Join(parts, "; ")
}

// QueryInput adalah isian mentah dari layar, belum divalidasi.
type QueryInput struct {
	// Tab adalah kode daftar yang diminta. Kosong berarti DefaultTab.
	Tab string

	// Search adalah isi kotak "Claim No". Kosong berarti tidak menyaring.
	Search string
}

// Query adalah permintaan isi satu daftar yang sudah tervalidasi.
type Query struct {
	// Tab adalah daftar yang diminta, lengkap dengan kolom dan penyaringnya.
	Tab Tab

	// Search adalah kata kunci pencarian yang sudah dipangkas.
	//
	// Yang dicocokkan adalah `T_CLAIM_PNC.CLAIMID` — kunci objek kerja, bukan nomor
	// klaim. Itu perilaku Pega: `SetDataPLADLA` menyusun
	// `"and (b.claimid like '%" + TempPLA.pyID + "%')"`.
	Search string

	// ReinsurerCodes adalah kode reasuradur milik pemanggil, terurut MENURUN.
	//
	// Ia bagian dari Query, bukan parameter terpisah, karena ia PENYARING — sama
	// kedudukannya dengan kata kunci. Menaruhnya di luar akan memungkinkan sebuah
	// pemanggilan menyusun daftar tanpa penyaring reasuradur sama sekali, dan daftar itu
	// akan menampilkan klaim SELURUH mitra kepada satu mitra.
	ReinsurerCodes []string

	// Caller adalah identitas pemanggil.
	Caller Caller
}

// EffectiveReinsurerCodes adalah kode yang BENAR-BENAR dipakai menyaring daftar ini.
//
// Tab Close memakai seluruhnya; kedua tab lain memakai yang TERTINGGI saja. Perbedaan itu
// ada di kueri Pega — `reinscode in (…)` versus
// `reinscode = (… ORDER BY reinsurerid DESC FETCH NEXT 1 ROW ONLY)` — dan ia dipusatkan
// di sini supaya terbaca di satu tempat alih-alih tersembunyi di tiga kueri.
//
// Kode diserahkan penyimpanan dalam urutan MENURUN, sehingga yang pertama adalah yang
// tertinggi.
func (q Query) EffectiveReinsurerCodes() []string {
	if len(q.ReinsurerCodes) == 0 {
		return nil
	}
	if q.Tab.AllReinsurerCodes {
		return q.ReinsurerCodes
	}
	return q.ReinsurerCodes[:1]
}

// maxSearchLength membatasi panjang kata kunci pencarian.
const maxSearchLength = 100

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
func NewQuery(input QueryInput, caller Caller, reinsurerCodes []string) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	if len(reinsurerCodes) == 0 {
		return Query{}, ErrCallerNotAReinsurer
	}

	code := strings.TrimSpace(input.Tab)
	if code == "" {
		code = DefaultTab
	}

	tab, known := FindTab(code)
	if !known {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Daftar tidak dikenal. Pilih PLA, DLA, atau Close.",
		}})
	}

	search := strings.TrimSpace(input.Search)
	if len(search) > maxSearchLength {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldSearch,
			Message: "Kata kunci pencarian terlalu panjang.",
		}})
	}

	return Query{
		Tab:            tab,
		Search:         search,
		ReinsurerCodes: reinsurerCodes,
		Caller:         cleanCaller,
	}, nil
}

// Matches menyatakan apakah sebuah baris lolos pencarian query ini.
//
// # Kenapa ia di DOMAIN, bukan di penyimpanan memori
//
// Karena arti pencarian adalah aturan bisnis: ia mencocokkan KUNCI KERJA, bukan nomor
// klaim. Menaruhnya di penyimpanan memori berarti penyimpanan SQL dapat memahaminya
// berbeda tanpa ada satu pun uji yang gagal.
func (q Query) Matches(row Row) bool {
	if q.Search == "" {
		return true
	}
	return strings.Contains(
		strings.ToUpper(row.ClaimKey), strings.ToUpper(q.Search))
}
