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

	// Coverages adalah pembatasan plan dan jaminan.
	//
	// SELALU dikirim, dan pada daftar SELALU kosong — daftar memang tidak membacanya.
	// Klien tidak boleh menyimpulkan "tidak ada pembatasan" dari hasil daftar; yang
	// berwenang hanya hasil pengambilan satu baris.
	Coverages []CoverageDTO `json:"jaminan"`
}

// CoverageDTO adalah satu pembatasan plan dan jaminan.
type CoverageDTO struct {
	ID           string `json:"id"`
	PlanID       string `json:"id_plan"`
	PlanName     string `json:"nama_plan"`
	CoverageID   string `json:"id_jaminan"`
	CoverageName string `json:"nama_jaminan"`
}

// DocumentDTO adalah satu pilihan pada isian ID Dokumen.
type DocumentDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// PlanDTO adalah satu pilihan pada isian Nama Plan.
type PlanDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// CoverageOptionDTO adalah satu pilihan pada isian Nama Jaminan.
//
// PlanID ikut dikirim supaya layar dapat menyaring jaminan menurut plan yang sudah
// dipilih tanpa menembak server lagi untuk setiap baris grid.
type CoverageOptionDTO struct {
	ID     string `json:"id"`
	Name   string `json:"nama"`
	PlanID string `json:"id_plan"`
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

// PlanListResponse adalah jawaban GET /api/master/plan-travel.
//
// Plan dan jaminan dikirim dalam SATU respons meski berasal dari dua pemanggilan seam,
// karena layar selalu membutuhkan keduanya bersamaan: grid coverage tidak dapat
// menampilkan satu baris pun tanpa keduanya, dan memisahkannya menjadi dua permintaan
// hanya menambah satu keadaan setengah-siap yang harus dijaga layar.
type PlanListResponse struct {
	Plan     []PlanDTO           `json:"plan"`
	Coverage []CoverageOptionDTO `json:"jaminan"`
	Portal   string              `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// ID TIDAK pernah datang dari klien: pada penambahan ia diterbitkan penyimpanan, dan
// pada perubahan ia diambil dari jalur URL. Dua sumber untuk satu nilai berarti keduanya
// dapat berbeda, dan yang mana yang menang menjadi pertanyaan yang tidak perlu ada.
type SaveRequest struct {
	DocumentID   string              `json:"id_dokumen"`
	DocumentName string              `json:"nama_dokumen"`
	Mandatory    bool                `json:"status_wajib"`
	MinUpload    int                 `json:"minimal_unggah"`
	Coverages    []CoverageSaveEntry `json:"jaminan"`
}

// CoverageSaveEntry adalah satu baris grid plan dan jaminan yang dikirim layar.
//
// Tanpa ID: seluruh daftar DIGANTI setiap kali disimpan, sehingga kunci baris lamanya
// tidak berguna bagi server.
//
// Nama DAN kode keduanya dikirim, bukan kode saja. Sebabnya perilaku layar lama:
// autocomplete-nya menyalin kode ke properti tersembunyi saat sebuah pilihan dipilih,
// tetapi isiannya tetap dapat diisi teks yang tidak ada di master — dan pada keadaan itu
// yang tersimpan hanyalah namanya, tanpa kode. Mengirim kode saja akan membuang isian
// yang sah menjadi baris kosong.
type CoverageSaveEntry struct {
	PlanID       string `json:"id_plan"`
	PlanName     string `json:"nama_plan"`
	CoverageID   string `json:"id_jaminan"`
	CoverageName string `json:"nama_jaminan"`
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
	coverages := make([]CoverageDTO, 0, len(row.Coverages))
	for _, coverage := range row.Coverages {
		coverages = append(coverages, CoverageDTO{
			ID:           coverage.ID,
			PlanID:       coverage.PlanID,
			PlanName:     coverage.PlanName,
			CoverageID:   coverage.CoverageID,
			CoverageName: coverage.CoverageName,
		})
	}
	return DetailDTO{
		ID:           row.ID,
		DocumentID:   row.DocumentID,
		DocumentName: row.DocumentName,
		Mandatory:    row.Mandatory,
		MinUpload:    row.MinUpload,
		Coverages:    coverages,
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya daftar kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal saat masternya masih kosong. Berlaku pula untuk
// senarai coverage di dalam setiap barisnya.
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

// toPlanListDTO mengubah pilihan Nama Plan.
func toPlanListDTO(list []daftardetaildokumentravel.Plan) []PlanDTO {
	result := make([]PlanDTO, 0, len(list))
	for _, row := range list {
		result = append(result, PlanDTO{ID: row.ID, Name: row.Name})
	}
	return result
}

// toCoverageOptionListDTO mengubah pilihan Nama Jaminan.
func toCoverageOptionListDTO(list []daftardetaildokumentravel.CoverageOption) []CoverageOptionDTO {
	result := make([]CoverageOptionDTO, 0, len(list))
	for _, row := range list {
		result = append(result, CoverageOptionDTO{ID: row.ID, Name: row.Name, PlanID: row.PlanID})
	}
	return result
}

// toInput mengubah isian form menjadi nilai domain.
func toInput(request SaveRequest) daftardetaildokumentravel.Input {
	coverages := make([]daftardetaildokumentravel.CoverageInput, 0, len(request.Coverages))
	for _, coverage := range request.Coverages {
		coverages = append(coverages, daftardetaildokumentravel.CoverageInput{
			PlanID:       coverage.PlanID,
			PlanName:     coverage.PlanName,
			CoverageID:   coverage.CoverageID,
			CoverageName: coverage.CoverageName,
		})
	}
	return daftardetaildokumentravel.Input{
		DocumentID:   request.DocumentID,
		DocumentName: request.DocumentName,
		Mandatory:    request.Mandatory,
		MinUpload:    request.MinUpload,
		Coverages:    coverages,
	}
}
