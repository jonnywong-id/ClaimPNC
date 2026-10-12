// Package masterdokumentravelhttp adalah lapisan transport modul Master Dokumen Travel:
// bentuk permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// portalhttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `masterdokumentravelhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package masterdokumentravelhttp

import "claim-pnc/internal/masterdokumentravel"

// TravelDocumentDTO adalah bentuk satu jenis dokumen travel yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari masterdokumentravel.TravelDocument. Memakai tipe modul
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama fieldnya berbahasa Indonesia karena ia kontrak API, bukan nama internal (`D-80`).
type TravelDocumentDTO struct {
	// ID adalah DOCID. Dibuat sistem; tidak pernah diisi pengguna.
	ID string `json:"id"`

	// Name adalah NAMADOKUMEN. Di layar Pega ia berlabel "Judul Dokumen".
	Name string `json:"judul"`
}

// ListResponse adalah jawaban GET /api/master/dokumen-travel.
type ListResponse struct {
	TravelDocument []TravelDocumentDTO `json:"dokumen_travel"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	Total int `json:"total"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna, bukan entitas lain. Pada aplikasi yang
	// melayani empat badan hukum, "data siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban untuk satu dokumen: ambil, tambah, dan ubah.
type SingleResponse struct {
	TravelDocument TravelDocumentDTO `json:"dokumen_travel"`
	Portal         string            `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// Hanya judul yang diterima. ID TIDAK pernah datang dari klien: pada penambahan ia
// diterbitkan penyimpanan, dan pada perubahan ia diambil dari jalur URL. Dua sumber
// untuk satu nilai berarti keduanya dapat berbeda, dan yang mana yang menang menjadi
// pertanyaan yang tidak perlu ada.
//
// Sentinel `"UnknownID"` yang dipakai sistem lama untuk membedakan tambah dari ubah
// tidak dibawa — di sini keduanya dua rute yang berbeda.
type SaveRequest struct {
	Name string `json:"judul"`
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
func toDTO(doc masterdokumentravel.TravelDocument) TravelDocumentDTO {
	return TravelDocumentDTO{ID: doc.ID, Name: doc.Name}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya daftar kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal saat masternya masih kosong.
func toListDTO(list []masterdokumentravel.TravelDocument) []TravelDocumentDTO {
	result := make([]TravelDocumentDTO, 0, len(list))
	for _, doc := range list {
		result = append(result, toDTO(doc))
	}
	return result
}
