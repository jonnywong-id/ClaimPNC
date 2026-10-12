package inboxmanageradmin

import "strings"

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery, dan di situlah kewenangan tab diperiksa.
//
// # Kenapa hanya satu isian
//
// Karena layar lama memang hanya punya satu. Report Definition `ManagementAdminView` tidak
// menyaring menurut kata kunci sama sekali — satu-satunya parameternya adalah `OrgUnit`,
// dan nilainya ditetapkan section per kontainer, bukan diketik pengguna. Tidak ada kotak
// cari, tidak ada dropdown, dan tidak ada pemilih tanggal di ketiga grid.
//
// Menambahkan kotak cari di sini akan menjadi kemampuan baru, bukan pemindahan — dan
// kemampuan baru menempuh keputusan tersendiri, bukan diselipkan karena tampak berguna.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti tab pertama yang boleh dilihat
	// pemanggil.
	Tab string
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolom dan unit organisasinya.
	Tab Tab

	// Caller adalah identitas pemanggil.
	//
	// Tidak satu pun kueri menyaring menurut nilai ini — layar ini pandangan penyelia.
	// Yang memakainya adalah pemeriksaan kewenangan tab di NewQuery dan jejak log.
	Caller Caller
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// # Urutan pemeriksaannya disengaja
//
// Identitas lebih dulu, lalu kewenangan menyeluruh, baru tab yang diminta. Dengan urutan
// itu pengguna yang tidak berhak atas satu tab pun memperoleh jawaban yang menjelaskan
// keadaannya — bukan "tab tidak dikenal" untuk tab yang sebenarnya ada.
func NewQuery(input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	visible := VisibleTabs(cleanCaller)
	if len(visible) == 0 {
		return Query{}, ErrNoTabAllowed
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

	return Query{Tab: tab, Caller: cleanCaller}, nil
}
