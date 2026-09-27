package inboxcompliance

import "strings"

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
//
// # Kenapa isinya hanya satu
//
// Karena layar ini memang hanya punya satu pilihan: tab mana yang terbuka. Tidak ada kotak
// cari, tidak ada dropdown lini bisnis, tidak ada rentang tanggal —
// `Section/InputCompliance_Section-Section.xml` tidak memuat satu pun field masukan.
//
// Ini berbeda dari modul Inbox Admin, yang punya keduanya. Perbedaannya nyata di sistem
// lama, bukan penyederhanaan di sini.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti DefaultTab.
	Tab string
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolom dan kemampuannya.
	Tab Tab

	// Workbasket adalah antrean yang dibaca.
	//
	// Ia bagian dari Query, bukan konstanta yang dibaca langsung oleh repo, karena dua
	// hal: ia terbaca di log sehingga jelas antrean mana yang sedang dibaca, dan uji
	// dapat menggantinya tanpa mengubah kode repo.
	Workbasket string
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// # Kenapa tidak ada pemeriksaan identitas pemanggil di sini
//
// Berbeda dari modul Inbox Admin — yang tiga tabnya menyaring menurut login pemanggil —
// antrean layar ini adalah WORKBASKET, yakni antrean BERSAMA. Isinya sama bagi setiap
// petugas Compliance yang berwenang membukanya, sehingga identitas pemanggil tidak ikut
// menentukan baris mana yang tampil.
//
// Yang menentukan siapa boleh membukanya adalah kewenangan menu, dan itu ditegakkan di
// lapisan transport, bukan di sini. Penegakannya sendiri masih `TKT-F3-005` — lihat
// catatan di http/routes.go.
func NewQuery(input QueryInput) (Query, error) {
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

	if !tab.Available {
		return Query{}, NewTabNotReadyError(tab)
	}

	return Query{Tab: tab, Workbasket: WorkbasketCompliance}, nil
}
