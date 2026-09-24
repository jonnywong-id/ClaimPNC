package inboxkomunikasicabang

import "strings"

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery, dan NewQuery MENUNTUT batas cabang yang sudah
// diselesaikan.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti DefaultTab.
	Tab string
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
//
// # Kenapa tidak ada satu pun penyaring bebas di sini
//
// Karena grid layar lama tidak punya satu pun yang sampai ke kuerinya. Ia memang punya
// kotak "Filter" dengan dua isian — `TempInputFilterKomunikasi.City` (pengirim) dan
// `.ClaimID` (urutan) — dan `PNCGetInboxKomunikasiCabang_Act` langkah 2 menyalin keduanya
// ke variabel lokal…
//
//	Local.filterPengirim := TempInputFilterKomunikasi.City
//	Local.filterUrutan   := TempInputFilterKomunikasi.ClaimID
//
// …lalu TIDAK memakainya lagi. Tidak satu pun dari kedua variabel itu muncul di langkah
// berikutnya, dan tidak satu pun dari kedua kueri grid menyaring menurut pengirim maupun
// mengubah urutannya.
//
// Jadi kotak Filter di layar lama TIDAK BERFUNGSI. Ia tidak dibawa (`P-5` — yang
// direplikasi adalah perilaku, dan perilakunya adalah "tidak menyaring apa pun"), dan
// menghidupkannya adalah kemampuan baru yang belum diputuskan siapa pun.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolom dan penyaringnya.
	Tab Tab

	// Branch adalah batas data yang berlaku bagi pemanggil.
	//
	// Ia WAJIB sudah diselesaikan sebelum Query terbentuk. Menaruhnya di sini, bukan
	// menyerahkannya ke penyimpanan, membuat batas itu tidak dapat terlewat: sebuah Repo
	// tidak dapat dipanggil tanpa membawanya.
	Branch BranchFilter

	// Caller adalah identitas pemanggil.
	Caller Caller
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// Batas cabang diterima SUDAH JADI, bukan diturunkan di sini: penurunannya menyentuh DB
// Link dan karena itu dapat gagal, sementara berkas ini adalah lapisan domain yang tidak
// boleh mengenal I/O apa pun. Yang menyelesaikannya adalah usecase; lihat ResolveBranch.
func NewQuery(input QueryInput, branch BranchFilter, caller Caller) (Query, error) {
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

	return Query{Tab: tab, Branch: branch, Caller: cleanCaller}, nil
}

// ResolveBranch menyusun batas data dari hasil penerjemahan kode cabang.
//
// # Kenapa ia fungsi domain, padahal penerjemahannya terjadi di luar
//
// Karena yang menentukan ARTI dari "tidak terbaca" adalah aturan bisnis, bukan adapter.
// Adapter hanya melaporkan apa yang ditemukannya; keputusan bahwa cabang yang tidak terbaca
// diperlakukan sebagai kantor pusat (`P-5`, keputusan Work Owner 2026-09-24) adalah aturan,
// dan aturan hidup di sini.
//
// Menaruhnya di adapter berarti kedua pengisi seam — SQL dan memori — masing-masing
// memutuskan sendiri, dan dua keputusan di dua tempat dapat menyimpang tanpa ketahuan.
//
// Tiga masukan, dua keluaran:
//
//	code="1002", resolved=true   -> {Code:"1002", HeadOffice:false, Resolved:true}
//	code="100081", resolved=true -> {Code:"1",    HeadOffice:true,  Resolved:true}
//	code="", resolved=false      -> {Code:"1",    HeadOffice:true,  Resolved:false}
//
// Baris kedua dan ketiga menghasilkan penyaring yang SAMA — itulah yang sistem lama
// lakukan. Yang membedakannya hanyalah Resolved, dan itu dipakai untuk menjelaskan keadaan
// ketiga kepada pengguna alih-alih membiarkannya tampak seperti keadaan kedua.
func ResolveBranch(code string, resolved bool) BranchFilter {
	clean := strings.TrimSpace(code)

	if !resolved || clean == "" {
		// Cabangnya tidak dapat diturunkan. Sistem lama menjatuhkannya ke kantor pusat
		// lewat precondition `KodeCabang == ""`, dan itu direplikasi.
		return BranchFilter{Code: HeadOfficeCode, HeadOffice: true, Resolved: false}
	}

	if clean == HeadOfficeBranch {
		// Petugas kantor pusat. Perhatikan kode yang DIPAKAI MENYARING bukan `100081`
		// melainkan `1`: kolom `COMMUNICATE_FROM` menyimpan penanda kanal, bukan kode
		// cabang, dan kantor pusat ditandai `1` di sana.
		//
		// Menukar keduanya menghasilkan daftar kosong tanpa satu pun galat — dan itu
		// persis kelas cacat yang `docs/keputusan-implementasi.md` §19.3 catat pada modul
		// Inbox Laporan Klaim.
		return BranchFilter{Code: HeadOfficeCode, HeadOffice: true, Resolved: true}
	}

	return BranchFilter{Code: clean, HeadOffice: false, Resolved: true}
}

// DetailInput adalah isian mentah permintaan layar Detail Komunikasi.
type DetailInput struct {
	// ID adalah nomor percakapan — `Param.KOMID` pada `DetailKomunikasi_dt`.
	ID string
}

// NewDetailRequest memeriksa permintaan layar detail.
//
// Ia mengembalikan nomor percakapan yang sudah dipangkas, bukan tipe tersendiri: yang
// dibutuhkan penyimpanan hanyalah nomornya beserta batas cabang, dan membungkus dua nilai
// dalam tipe baru tidak menambah satu pun jaminan.
func NewDetailRequest(input DetailInput, caller Caller) (string, error) {
	if caller.Clean().Login == "" {
		return "", ErrCallerUnknown
	}

	id := strings.TrimSpace(input.ID)
	if id == "" {
		return "", NewValidationError([]Violation{{
			Field:   FieldConversation,
			Message: "Nomor percakapan tidak disebutkan.",
		}})
	}

	return id, nil
}
