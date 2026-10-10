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

// Action adalah tindakan yang diminta salah satu tombol yang belum dibangun.
//
// # Yang tersisa hanya DUA, dan keduanya ada di layar RINCIAN
//
// Ketiga tombol yang sempat dicatat sebagai "belum dibangun" pada catatan sesi sebelumnya
// ternyata bukan tiga hal yang sejenis, dan hanya satu yang nyata:
//
//	"Detail Claim"  nyata  -> DIBANGUN, lihat detail.go
//	"Detail"        hanya tampil bagi satu Operator ID yang ditulis tetap di dalam rule
//	                (`pyContainerVisibleWhen = OperatorID.pyUserIdentifier=='KBRU_PNC'`)
//	"DLA"           berada di dalam wadah bersyarat `1==2` — TIDAK PERNAH tergambar
//
// Kedua yang terakhir TIDAK dibawa, dan karena itu tidak punya tombol maupun tindakan di
// sini: tombol yang di Pega pun tidak pernah muncul tidak perlu dijawab alasannya.
//
// Yang benar-benar tersisa adalah kedua tombol unduh massal pada layar rincian.
type Action string

const (
	// ActionDownloadAllPLA adalah tombol **"Download ALL PLA"** pada layar rincian.
	ActionDownloadAllPLA Action = "unduh-semua-pla"

	// ActionDownloadAllDLA adalah tombol **"Download ALL DLA"**.
	ActionDownloadAllDLA Action = "unduh-semua-dla"
)

// actionReasons memetakan tiap tindakan ke alasan yang dibaca pengguna.
//
// Kalimatnya menyebut APA yang belum ada dan apa yang DAPAT dilakukan hari ini. Pembaca
// layar ini adalah pihak luar yang tidak dapat kita latih, sehingga "belum tersedia" tanpa
// jalan keluar akan berakhir sebagai telepon ke Service Center.
var actionReasons = map[Action]string{
	ActionDownloadAllPLA: "Tombol \"Download ALL PLA\" belum tersedia di sistem baru. Ia " +
		"menggabungkan seluruh dokumen PLA klaim ini menjadi satu unduhan, dan rule " +
		"yang melakukannya (`SetDocumentPLADLA`, `GetLinkViewDoc_Act`) tidak ada di " +
		"export Pega. Untuk sementara, unduhlah dokumennya satu per satu lewat tombol " +
		"\"Dokumen\" pada tiap baris PLA.",

	ActionDownloadAllDLA: "Tombol \"Download ALL DLA\" belum tersedia di sistem baru, " +
		"dengan sebab yang sama seperti \"Download ALL PLA\". Unduhlah dokumennya satu " +
		"per satu lewat tombol \"Dokumen\" pada tiap baris DLA.",
}

// NotAvailableError menyatakan sebuah tombol ditekan yang tindakannya belum dibangun.
//
// Ia membawa TINDAKANNYA supaya lapisan transport dapat menjawab alasan yang tepat. Tanpa
// itu, kedua tombol menjawab kalimat yang sama — dan pengguna yang menekan "Download ALL
// DLA" akan membaca penjelasan tentang PLA.
type NotAvailableError struct {
	Action Action
}

// NewNotAvailable membentuk penolakan untuk satu tindakan.
//
// Tindakan yang TIDAK dikenal tetap menghasilkan penolakan, bukan galat lain: yang dituju
// pengguna memang tombol yang belum dibangun, dan nama tindakan yang salah ketik di alamat
// bukan sesuatu yang perlu dibedakan di layar.
func NewNotAvailable(raw string) *NotAvailableError {
	action := Action(strings.TrimSpace(raw))
	if _, known := actionReasons[action]; !known {
		return &NotAvailableError{}
	}
	return &NotAvailableError{Action: action}
}

// Reason adalah kalimat yang dibaca pengguna.
func (e *NotAvailableError) Reason() string {
	if reason, known := actionReasons[e.Action]; known {
		return reason
	}
	return "Tindakan ini belum tersedia di sistem baru. Kerjakan lewat Pega."
}

// Error menyusun pesan ringkas untuk log. Yang dibaca pengguna adalah Reason.
func (e *NotAvailableError) Error() string {
	if e.Action == "" {
		return "inboxpladla: tindakan tidak dikenal belum tersedia"
	}
	return "inboxpladla: tindakan " + string(e.Action) + " belum tersedia"
}

// Unwrap membuat errors.Is(err, ErrWriteNotAvailable) tetap benar.
func (e *NotAvailableError) Unwrap() error { return ErrWriteNotAvailable }

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

// tabNames menyusun daftar nama tab untuk pesan galat.
func tabNames() string {
	names := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		names = append(names, tab.Name)
	}
	return strings.Join(names, ", ")
}

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
		// Pesannya menyebut PILIHANNYA, bukan sekadar menyatakan kodenya salah.
		//
		// Ia disusun dari daftar tab itu sendiri, bukan ditulis ulang di sini: sejak
		// tampilannya menjadi tujuh, kalimat yang ditulis tangan akan tertinggal pada
		// penambahan berikutnya dan menyebut pilihan yang tidak lagi lengkap.
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Daftar tidak dikenal. Pilih salah satu dari: " + tabNames() + ".",
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
