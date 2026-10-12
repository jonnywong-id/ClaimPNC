// Package daftardetaildokumentravelhttp adalah lapisan transport modul Daftar Detail
// Dokumen Travel: bentuk permintaan dan respons, pemetaan galat, handler, dan
// pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// portalhttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `daftardetaildokumentravelhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package daftardetaildokumentravelhttp

import "claim-pnc/internal/daftardetaildokumentravel"

// DetailDTO adalah bentuk satu aturan kelengkapan dokumen yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari daftardetaildokumentravel.Detail. Memakai tipe modul
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama fieldnya berbahasa Indonesia karena ia kontrak API, bukan nama internal (`D-80`),
// dan kata yang dipakai mengikuti label isian di layar Pega
// (`Section/BrowseDocumentTravel-Section.xml`) supaya satu istilah berlaku dari layar
// sampai ke kontrak.
type DetailDTO struct {
	// ID adalah kunci baris. Dibuat sistem; tidak pernah diisi pengguna.
	ID string `json:"id"`

	// DocumentID adalah DOCID — label layar "ID Dokumen".
	DocumentID string `json:"id_dokumen"`

	// DocumentName adalah DOCUMENTNAME — label layar "Nama Dokumen".
	DocumentName string `json:"nama_dokumen"`

	// Mandatory adalah STSWAJIB — label layar "Status Wajib".
	//
	// Dikirim sebagai boolean, bukan angka 1/0 seperti di basis data. Angkanya bentuk
	// penyimpanan, dan membocorkannya ke kontrak berarti setiap klien harus mengetahui
	// arti 1 dan 0 — padahal `BrowseDocTravel-Act` pun sudah mengubahnya menjadi
	// "Ya"/"Tidak" sebelum menampilkannya.
	Mandatory bool `json:"status_wajib"`

	// MinUpload adalah MINUNGGAH — label layar "Minimal Unggah".
	MinUpload int `json:"minimal_unggah"`
}

// DocumentDTO adalah satu pilihan pada isian ID Dokumen.
type DocumentDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/daftar-detail-dokumen-travel.
type ListResponse struct {
	Detail []DetailDTO `json:"detail_dokumen_travel"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	Total int `json:"total"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna, bukan entitas lain. Pada aplikasi yang
	// melayani empat badan hukum, "data siapa ini" tidak boleh hanya diandaikan
	// (`R-20`).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban untuk satu baris: ambil, tambah, dan ubah.
type SingleResponse struct {
	Detail DetailDTO `json:"detail_dokumen_travel"`
	Portal string    `json:"portal"`
}

// DocumentListResponse adalah jawaban GET /api/master/dokumen-travel-pilihan.
type DocumentListResponse struct {
	Document []DocumentDTO `json:"dokumen"`
	Portal   string        `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// Keempat field inilah seluruh isi form di layar Pega — tidak ada yang lain. Pembatasan
// per Plan dan Jaminan yang sempat ada di sini DICABUT 2026-10-03: grid itu memang ada di
// `Section/BrowseDocumentTravel-Section.xml`, tetapi tidak ada di aplikasi Pega yang
// berjalan, dan Work Owner yang memeriksa layarnya langsung.
//
// ID TIDAK pernah datang dari klien: pada penambahan ia diterbitkan penyimpanan, dan
// pada perubahan ia diambil dari jalur URL. Dua sumber untuk satu nilai berarti keduanya
// dapat berbeda, dan yang mana yang menang menjadi pertanyaan yang tidak perlu ada.
type SaveRequest struct {
	DocumentID   string `json:"id_dokumen"`
	DocumentName string `json:"nama_dokumen"`
	Mandatory    bool   `json:"status_wajib"`
	MinUpload    int    `json:"minimal_unggah"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}`. Klien membedakan jenis galat lewat
// `kode`, tidak pernah dengan mencocokkan teks `pesan`.
//
// Tidak ada field `detail` di sini, dan itu konsekuensi langsung dari keputusan Work
// Owner 2026-09-21: layar ini tanpa validasi, sehingga tidak ada pelanggaran per isian
// yang dapat dilaporkan. Menyediakan fieldnya "untuk berjaga-jaga" akan menyiratkan ada
// aturan yang sebenarnya tidak ada.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(row daftardetaildokumentravel.Detail) DetailDTO {
	return DetailDTO{
		ID:           row.ID,
		DocumentID:   row.DocumentID,
		DocumentName: row.DocumentName,
		Mandatory:    row.Mandatory,
		MinUpload:    row.MinUpload,
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya daftar kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal saat masternya masih kosong.
func toListDTO(list []daftardetaildokumentravel.Detail) []DetailDTO {
	result := make([]DetailDTO, 0, len(list))
	for _, row := range list {
		result = append(result, toDTO(row))
	}
	return result
}

// toDocumentListDTO mengubah pilihan ID Dokumen.
func toDocumentListDTO(list []daftardetaildokumentravel.Document) []DocumentDTO {
	result := make([]DocumentDTO, 0, len(list))
	for _, row := range list {
		result = append(result, DocumentDTO{ID: row.ID, Name: row.Name})
	}
	return result
}

// toInput mengubah isian form menjadi nilai domain.
func toInput(request SaveRequest) daftardetaildokumentravel.Input {
	return daftardetaildokumentravel.Input{
		DocumentID:   request.DocumentID,
		DocumentName: request.DocumentName,
		Mandatory:    request.Mandatory,
		MinUpload:    request.MinUpload,
	}
}
