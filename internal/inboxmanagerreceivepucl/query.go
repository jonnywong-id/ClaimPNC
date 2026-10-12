package inboxmanagerreceivepucl

import "strings"

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti DefaultTab.
	Tab string
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
//
// # Kenapa tidak ada satu pun penyaring di sini
//
// Karena layar lama tidak punya satu pun. Ketiga gridnya dipasok Report Definition tanpa
// kotak cari, tanpa dropdown, dan tanpa checkbox — yang ada hanyalah parameter `OrgUnit`
// yang tidak pernah diisi dan parameter `Position1` yang nilainya ditetapkan section, bukan
// dipilih pengguna.
//
// Menambahkan kotak cari di sini berarti menambah kemampuan yang tidak pernah ada — dan pada
// layar yang sedang diuji kesetaraannya, kemampuan tambahan adalah selisih yang harus
// dipertanggungjawabkan. Yang membedakan ketiga tab sudah dibawa Tab itu sendiri.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolom dan antrean sumbernya.
	Tab Tab

	// Caller adalah identitas pemanggil.
	//
	// Tidak satu pun kueri menyaring menurut nilai ini — layar ini pandangan penyelia atas
	// pekerjaan orang lain. Ia dibawa untuk jejak log; lihat Caller.
	Caller Caller
}

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
			Message: "Tab tidak dikenal.",
		}})
	}

	// Tab terhalang DITOLAK di sini, bukan dibiarkan sampai ke penyimpanan.
	//
	// Tidak ada tab terhalang di layar ini hari ini, dan pemeriksaannya tetap ada: begitu
	// sebuah tab ditandai terhalang kelak, permintaannya akan sampai ke repo yang tidak
	// punya kueri untuknya dan gagal sebagai galat internal 500 — jawaban yang tidak
	// menyebut sebabnya kepada siapa pun. Ditolak di sini, alasannya sampai ke layar apa
	// adanya.
	if tab.Blocked {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: tab.BlockedReason,
		}})
	}

	return Query{Tab: tab, Caller: cleanCaller}, nil
}
