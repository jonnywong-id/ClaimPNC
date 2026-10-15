package inboxbandinghargasalvage

import "strings"

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti DefaultTab.
	Tab string

	// Keyword adalah isi kotak "Cari No Klaim".
	Keyword string
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolomnya.
	Tab Tab

	// Reviewer menyatakan antrean SIAPA yang dibaca, dan giliran siapa yang ditunggu.
	//
	// Ia diturunkan dari pemanggil, bukan diterima dari layar. Menerimanya dari layar berarti
	// siapa pun dapat membaca antrean komite mana pun dengan mengubah satu parameter —
	// tepat kelas kebocoran yang `R-20` peringatkan.
	Reviewer Reviewer

	// Keyword adalah nomor klaim yang dicari, sudah dipangkas dan dihurufbesarkan.
	//
	// Kosong berarti tidak sedang mencari.
	Keyword string
}

// Searching menyatakan permintaan ini sedang menyaring dengan kata kunci.
func (q Query) Searching() bool {
	return q.Keyword != ""
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// # Kenapa pencarian COCOK PERSIS, bukan mengandung
//
// Karena begitulah kuerinya di Pega. Kedua kotak cari menyusun penyaringnya sebagai
// `"and noklaim='" + Param.noklaim + "' "` — tanda sama dengan, bukan `LIKE '%…%'`
// (`Activity/SetReqSalvage_Act-Act.xml` langkah 5 dan 11). Petunjuk di bawah kotaknya pun
// menyebut satu nomor utuh: `Contoh : PNC-1234`.
//
// Perilakunya ditiru apa adanya (`P-5`): mengetik separuh nomor klaim TIDAK menghasilkan
// baris, dan layar menyatakannya lewat teks petunjuk supaya pengguna tidak menyimpulkan
// antreannya kosong.
//
// # Satu hal yang TIDAK ditiru, dan alasannya
//
// Perbandingannya di sini diseragamkan ke huruf besar dan spasinya dipangkas. Di Pega ia
// dibandingkan apa adanya, sehingga pengguna yang mengetik huruf kecil tidak menemukan apa
// pun — dan gagalnya diam. Itu bukan aturan bisnis; ia akibat membandingkan teks tanpa
// menormalkannya. Nomor klaim sendiri selalu tersimpan huruf kapital, sehingga satu-satunya
// selisih yang mungkin adalah pengguna yang mengetik huruf kecil.
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
			Message: "Tab tidak dikenal.",
		}})
	}

	return Query{
		Tab:      tab,
		Reviewer: ReviewerFor(cleanCaller),
		Keyword:  strings.ToUpper(strings.TrimSpace(input.Keyword)),
	}, nil
}
