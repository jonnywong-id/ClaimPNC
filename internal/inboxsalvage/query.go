package inboxsalvage

import "strings"

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti DefaultTab.
	Tab string

	// Search adalah isi kotak "CARI NO KLAIM". Kosong berarti tidak menyaring.
	Search string
}

// Query adalah permintaan isi satu daftar yang sudah tervalidasi.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolom dan penyaringnya.
	Tab Tab

	// Search adalah kata kunci pencarian yang sudah dipangkas. Kosong berarti tidak
	// menyaring.
	//
	// Cara ia dipakai ditentukan Tab.SearchExact dan Tab.SearchByPIC, bukan oleh isian ini
	// sendiri — dengan begitu penyimpanan SQL dan penyimpanan memori tidak dapat
	// berselisih soal arti pencarian pada tab yang sama.
	Search string

	// Caller adalah identitas pemanggil.
	//
	// Pada tab Request Balai Lelang ia MENYARING (`PIC = <pemanggil>`); pada tab lain ia
	// dibawa untuk jejak log. Lihat Caller.
	Caller Caller
}

// maxSearchLength membatasi panjang kata kunci pencarian.
//
// Nomor klaim terpanjang di sistem lama berbentuk `PNC-xxxx`, dan nomor sistem baru
// `PNCN.YY.xxxx` (`D-71`) — keduanya jauh di bawah batas ini. Batasnya ada bukan untuk
// menolak nomor klaim yang sah, melainkan supaya kata kunci sepanjang satu megabita tidak
// pernah sampai ke basis data.
const maxSearchLength = 100

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
func NewQuery(input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	code := strings.TrimSpace(input.Tab)
	if code == "" {
		code = DefaultTab
	}

	tab, known := FindTab(code)
	if !known {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Daftar tidak dikenal.",
		}})
	}

	return NewQueryForTab(tab, input, cleanCaller)
}

// NewQueryForTab menyusun permintaan atas daftar yang SUDAH ditemukan pemanggil.
//
// Ia terpisah dari NewQuery karena keduanya menjawab hal yang berbeda: NewQuery menolak
// kode daftar yang tidak ditawarkan layar — itu pemeriksaan masukan pengguna — sedangkan
// ini menyusun permintaannya.
//
// Pemisahan itu dibutuhkan keempat daftar yang tidak ditawarkan: penyaringnya tetap diuji
// meski layar tidak menawarkannya, sebab tiga di antaranya harus kembali begitu kewenangan
// berbasis peran ada. Penyaring yang tidak diuji selama itu akan berhenti benar tanpa ada
// yang tahu.
//
// Lapisan transport TIDAK memanggilnya langsung — ia memanggil NewQuery, yang menjaga
// pintunya.
func NewQueryForTab(tab Tab, input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	search := strings.TrimSpace(input.Search)

	// Pencarian pada tab yang TIDAK punya kotak pencarian dibuang, bukan ditolak.
	//
	// Menolaknya akan membuat layar gagal ketika pengguna berpindah tab sementara kotak
	// pencariannya masih terisi — keadaan yang terjadi setiap hari dan bukan kesalahan
	// siapa pun.
	if tab.SearchLabel == "" {
		search = ""
	}

	if len(search) > maxSearchLength {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldSearch,
			Message: "Kata kunci pencarian terlalu panjang.",
		}})
	}

	return Query{Tab: tab, Search: search, Caller: cleanCaller}, nil
}

// Matches menyatakan apakah sebuah baris lolos pencarian query ini.
//
// # Kenapa ia di DOMAIN, bukan di penyimpanan memori
//
// Karena aturannya berbeda per tab — cocok persis pada tiga tab, mengandung pada tujuh tab,
// dan dua kolom sekaligus pada tiga tab — dan aturan itu adalah aturan bisnis, bukan detail
// penyimpanan. Menaruhnya di penyimpanan memori berarti penyimpanan SQL dapat memahaminya
// berbeda tanpa ada satu pun uji yang gagal.
//
// Penyimpanan SQL tidak memanggil fungsi ini — ia menyusun klausa WHERE yang setara — dan
// kesetaraan keduanya dijaga uji di query_test.go pada kedua sisi.
func (q Query) Matches(row Row) bool {
	if q.Search == "" {
		return true
	}

	if q.Tab.SearchExact {
		if strings.EqualFold(row.ClaimNo, q.Search) {
			return true
		}
		return q.Tab.SearchByPIC && strings.EqualFold(row.PIC, q.Search)
	}

	needle := strings.ToLower(q.Search)
	if strings.Contains(strings.ToLower(row.ClaimNo), needle) {
		return true
	}
	return q.Tab.SearchByPIC && strings.Contains(strings.ToLower(row.PIC), needle)
}
