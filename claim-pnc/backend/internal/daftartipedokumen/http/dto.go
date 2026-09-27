// Package daftartipedokumenhttp adalah lapisan transport modul Daftar Tipe Dokumen:
// bentuk permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// portalhttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `daftartipedokumenhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package daftartipedokumenhttp

import "claim-pnc/internal/daftartipedokumen"

// DocumentTypeDTO adalah bentuk satu tipe dokumen yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari daftartipedokumen.DocumentType. Memakai tipe modul
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama fieldnya berbahasa Indonesia karena ia kontrak API, bukan nama internal (`D-80`).
type DocumentTypeDTO struct {
	// ID adalah kolom ID. Dibuat sistem; tidak pernah diisi pengguna.
	ID string `json:"id"`

	// Type adalah TYPE_DOCUMENT. Di grid Pega ia berlabel "Tipe Dokumen", di formnya
	// "Jenis Dokumen".
	Type string `json:"tipe_dokumen"`

	// ProcessStatus adalah STS_PROSES.
	//
	// Namanya `status_proses` mengikuti label layar lama, dan itu perlu dibaca dengan
	// hati-hati: ia **teks bebas**, bukan penanda aktif/non-aktif. Klien tidak boleh
	// memperlakukannya sebagai enum. Alasannya ada di kepala paket `daftartipedokumen`.
	ProcessStatus string `json:"status_proses"`
}

// ListResponse adalah jawaban GET /api/master/tipe-dokumen.
type ListResponse struct {
	DocumentType []DocumentTypeDTO `json:"tipe_dokumen"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	Total int `json:"total"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna, bukan entitas lain. Pada aplikasi yang melayani
	// empat badan hukum, "data siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban untuk satu tipe dokumen: ambil, tambah, dan ubah.
type SingleResponse struct {
	DocumentType DocumentTypeDTO `json:"tipe_dokumen"`
	Portal       string          `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// Hanya kedua isian yang benar-benar dapat diketik pengguna yang diterima. ID TIDAK
// pernah datang dari klien: pada penambahan ia diterbitkan penyimpanan, dan pada
// perubahan ia diambil dari jalur URL. Dua sumber untuk satu nilai berarti keduanya dapat
// berbeda, dan yang mana yang menang menjadi pertanyaan yang tidak perlu ada.
//
// USER_EDIT dan TGL_EDIT juga tidak ada di sini, dan itu disengaja: keduanya jejak simpan
// yang diisi server dari sesi dan jam sistem. Menerimanya dari klien berarti mengizinkan
// pemanggil mengaku sebagai orang lain pada kolom yang justru dipakai menelusuri siapa
// yang mengubah apa.
//
// Sentinel `"UnknownID"` yang dipakai sistem lama untuk membedakan tambah dari ubah tidak
// dibawa — di sini keduanya dua rute yang berbeda.
type SaveRequest struct {
	Type          string `json:"tipe_dokumen"`
	ProcessStatus string `json:"status_proses"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}`. Klien membedakan jenis galat lewat
// `kode`, tidak pernah dengan mencocokkan teks `pesan`.
//
// Tidak ada field `detail` di sini, dan itu konsekuensi langsung dari keputusan Work Owner
// 2026-09-21: layar ini tanpa validasi, sehingga tidak ada pelanggaran per isian yang
// dapat dilaporkan. Menyediakan fieldnya "untuk berjaga-jaga" akan menyiratkan ada aturan
// yang sebenarnya tidak ada.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(doc daftartipedokumen.DocumentType) DocumentTypeDTO {
	return DocumentTypeDTO{
		ID:            doc.ID,
		Type:          doc.Type,
		ProcessStatus: doc.ProcessStatus,
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya daftar kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri, dan
// satu layar yang lupa akan gagal saat masternya masih kosong.
func toListDTO(list []daftartipedokumen.DocumentType) []DocumentTypeDTO {
	result := make([]DocumentTypeDTO, 0, len(list))
	for _, doc := range list {
		result = append(result, toDTO(doc))
	}
	return result
}
