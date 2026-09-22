package inboxclaimtreatyprop

import "strings"

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti DefaultTab.
	Tab string

	// SeeAll adalah keadaan checkbox "See All Claim".
	//
	// Ia diabaikan pada tab yang tidak mendukungnya — lihat Query.SeeAll.
	SeeAll bool
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
//
// # Kenapa tidak ada kata kunci maupun penyaring lini bisnis di sini
//
// Karena ketiga kueri sistem lama tidak punya satu pun. `GetClaimTreaty_SQL`,
// `GetClaimTreatyAllAdmin_SQL`, dan `GetClaimTreatyTeknik_SQL` menyaring HANYA dengan
// `PXREFOBJECTKEY LIKE '%CLMP%'` ditambah pemilik antreannya. Menambahkan kotak cari di
// sini berarti menambah kemampuan yang tidak pernah ada — dan pada layar yang sedang diuji
// kesetaraannya, kemampuan tambahan adalah selisih yang harus dipertanggungjawabkan.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolom dan kemampuannya.
	Tab Tab

	// SeeAll adalah keadaan checkbox yang BENAR-BENAR dipakai.
	//
	// Ia selalu false pada tab yang tidak mendukungnya — bukan nilai yang dikirim layar.
	// Tanpa penetapan itu, dua permintaan yang hasilnya pasti sama akan tampak berbeda di
	// log dan di kunci cache.
	SeeAll bool

	// Caller adalah identitas pemanggil. Tab pertama menyaring menurut nilai ini.
	Caller Caller
}

// ScopedToCaller menyatakan permintaan ini hanya mengambil pekerjaan milik pemanggil.
//
// Ia BUKAN sekadar Tab.ScopedToCaller: mencentang "See All Claim" pada tab yang memang
// menyaring pemilik akan MELEPAS penyaring itu. Dua penentu yang harus dibaca bersama
// sering hanya dibaca salah satunya, jadi jawabannya disediakan di satu tempat.
func (q Query) ScopedToCaller() bool {
	return q.Tab.ScopedToCaller && !q.SeeAll
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
	// Kalau dibiarkan, ia akan sampai ke repo yang tidak punya kueri untuknya dan gagal
	// sebagai galat internal 500 — jawaban yang tidak menyebut sebabnya kepada siapa pun.
	// Ditolak di sini, alasannya sampai ke layar apa adanya.
	if tab.Blocked {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: tab.BlockedReason,
		}})
	}

	query := Query{Tab: tab, Caller: cleanCaller}
	if tab.SupportsSeeAll {
		query.SeeAll = input.SeeAll
	}

	return query, nil
}
