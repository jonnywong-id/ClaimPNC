// Package daftardetailtipedokumenhttp adalah lapisan transport modul Daftar Detail Tipe
// Dokumen: bentuk permintaan dan respons, pemetaan galat, handler, dan pendaftaran
// rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// portalhttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `daftardetailtipedokumenhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package daftardetailtipedokumenhttp

import (
	"claim-pnc/internal/daftardetailtipedokumen"
	"claim-pnc/internal/daftardetailtipedokumen/usecase"
)

// DetailTypeDTO adalah bentuk satu rincian dokumen yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari daftardetailtipedokumen.DetailType. Memakai tipe modul
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama fieldnya berbahasa Indonesia karena ia kontrak API, bukan nama internal (`D-80`),
// dan kata yang dipakai mengikuti label isian di layar Pega
// (`Section/BrowseListDetailTypeDocument-Section.xml`) supaya satu istilah berlaku dari
// layar sampai ke kontrak.
type DetailTypeDTO struct {
	// ID adalah kunci baris. Dibuat sistem; tidak pernah diisi pengguna.
	ID string `json:"id"`

	// DocumentTypeID adalah DOC_TYPE_ID — label layar "ID Tipe Dokumen".
	DocumentTypeID string `json:"id_tipe_dokumen"`

	// DocumentTypeName adalah TYPE_DOCUMENT — nama tipe dokumen menurut masternya.
	//
	// HANYA DIBACA. Kosong berarti ID-nya tidak ada di master — dan barisnya tetap
	// dikirim, bukan disembunyikan, supaya petugas dapat memperbaikinya.
	DocumentTypeName string `json:"nama_tipe_dokumen"`

	// Detail adalah DETAIL_DOCUMENT — label layar "Detail Dokumen".
	Detail string `json:"detail_dokumen"`

	// InsuredStatus adalah STS_INSURED — label layar "Status Tertanggung". Teks bebas.
	InsuredStatus string `json:"status_tertanggung"`

	// CauseOfLossID adalah DOC_COL_ID — label layar "Dokumen kolom ID".
	CauseOfLossID string `json:"id_penyebab_kerugian"`

	// CauseOfLossDescription adalah DOC_COL_INFO — label layar "Dokumen kolom ID".
	//
	// **TERSIMPAN, bukan hasil join.** Inilah isian yang benar-benar dilihat dan diketik
	// petugas; `id_penyebab_kerugian` di atas hanya kode yang menyertainya.
	CauseOfLossDescription string `json:"keterangan_penyebab_kerugian"`

	// ObjectDocumentID adalah OBJ_DOC — label layar "Objek Dokumen".
	ObjectDocumentID string `json:"id_objek_dokumen"`

	// ObjectDocumentDescription adalah OBJ_DOC_DESC — label layar "Objek Dokumen".
	//
	// **TERSIMPAN**, dengan alasan yang sama seperti keterangan di atas.
	ObjectDocumentDescription string `json:"keterangan_objek_dokumen"`

	// Risk adalah RISK — label layar "Resiko".
	//
	// Dikirim sebagai TEKS, bukan angka, karena itulah bentuk kolomnya — dan karena baris
	// warisan dapat memuat isi yang bukan angka. Mengirimnya sebagai angka akan memaksa
	// server menolak baris yang sebenarnya tersimpan dan harus tetap dapat diperbaiki.
	Risk string `json:"resiko"`

	// Businesses adalah aturan per lini bisnis.
	//
	// SELALU dikirim, dan pada daftar SELALU kosong — daftar memang tidak membacanya.
	// Klien tidak boleh menyimpulkan "tidak ada lini bisnis" dari hasil daftar; yang
	// berwenang hanya hasil pengambilan satu baris.
	Businesses []BusinessRuleDTO `json:"bisnis"`
}

// BusinessRuleDTO adalah aturan dokumen pada satu lini bisnis.
type BusinessRuleDTO struct {
	// BusinessID adalah DFT_BISNIS_ID — label layar "ID Bisnis".
	BusinessID string `json:"id_bisnis"`

	// BusinessName adalah NOTE pada POOLDATA.BUSINESS. HANYA DIBACA.
	BusinessName string `json:"nama_bisnis"`

	// Mandatory adalah STS_WAJIB — label layar "Status Wajib".
	//
	// Dikirim sebagai boolean, bukan teks seperti di basis data. Teksnya bentuk
	// penyimpanan, dan membocorkannya ke kontrak berarti setiap klien harus mengetahui
	// bahwa kolom itu memuat empat nilai berbeda di produksi — "Ya", "Tidak", "1", "0".
	Mandatory bool `json:"status_wajib"`

	// MinDocument adalah MIN_DOC — label layar "Minimum Dokumen".
	MinDocument int `json:"minimum_dokumen"`
}

// ChoiceDTO adalah satu pilihan pada isian berdaftar.
//
// Satu bentuk untuk keempat daftar, karena keempatnya memang berisi hal yang sama: satu
// kode dan satu keterangan. Empat tipe yang isinya identik hanya menambah empat tempat
// yang harus diubah bersamaan.
type ChoiceDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/detail-tipe-dokumen.
type ListResponse struct {
	Detail []DetailTypeDTO `json:"detail_tipe_dokumen"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	Total int `json:"total"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna, bukan entitas lain. Pada aplikasi yang
	// melayani empat badan hukum, "data siapa ini" tidak boleh hanya diandaikan (`R-20`).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban untuk satu baris: ambil, tambah, dan ubah.
type SingleResponse struct {
	Detail DetailTypeDTO `json:"detail_tipe_dokumen"`
	Portal string        `json:"portal"`
}

// ReferenceResponse adalah jawaban GET /api/master/detail-tipe-dokumen/pilihan.
//
// Keempat daftar dikirim dalam SATU respons meski berasal dari empat pemanggilan seam,
// karena form selalu membutuhkan keempatnya bersamaan.
type ReferenceResponse struct {
	DocumentTypes   []ChoiceDTO `json:"tipe_dokumen"`
	CausesOfLoss    []ChoiceDTO `json:"penyebab_kerugian"`
	ObjectDocuments []ChoiceDTO `json:"objek_dokumen"`
	Businesses      []ChoiceDTO `json:"bisnis"`

	// Unavailable menyebut master mana yang gagal dibaca.
	//
	// Ia ada supaya layar dapat membedakan "masternya kosong" dari "masternya tidak dapat
	// dibaca" — dua keadaan yang tampak sama persis di layar (daftar pilihan kosong)
	// tetapi menuntut kalimat yang berbeda kepada petugas.
	Unavailable []string `json:"tidak_tersedia"`

	Portal string `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// ID TIDAK pernah datang dari klien: pada penambahan ia diterbitkan penyimpanan, dan pada
// perubahan ia diambil dari jalur URL. Dua sumber untuk satu nilai berarti keduanya dapat
// berbeda, dan yang mana yang menang menjadi pertanyaan yang tidak perlu ada.
//
// # Dua keterangan ikut dikirim, satu tidak
//
// `keterangan_penyebab_kerugian` dan `keterangan_objek_dokumen` ADA di sini karena
// keduanya isian yang diketik petugas dan benar-benar tersimpan. `nama_tipe_dokumen`
// TIDAK ADA karena ia satu-satunya yang hasil join — mengirimkannya hanya akan menyimpan
// salinan yang dapat menyimpang dari masternya.
//
// Kode DAN keterangan keduanya dikirim, bukan keterangannya saja. Sebabnya perilaku layar
// lama: autocomplete-nya menyalin kode ke properti tersembunyi saat sebuah pilihan
// dipilih, tetapi isiannya tetap dapat diisi teks yang tidak ada di master
// (`pyAllowFreeFormInput=true`) — dan pada keadaan itu yang tersimpan hanyalah
// keterangannya, tanpa kode.
type SaveRequest struct {
	DocumentTypeID            string              `json:"id_tipe_dokumen"`
	Detail                    string              `json:"detail_dokumen"`
	InsuredStatus             string              `json:"status_tertanggung"`
	CauseOfLossID             string              `json:"id_penyebab_kerugian"`
	CauseOfLossDescription    string              `json:"keterangan_penyebab_kerugian"`
	ObjectDocumentID          string              `json:"id_objek_dokumen"`
	ObjectDocumentDescription string              `json:"keterangan_objek_dokumen"`
	Risk                      string              `json:"resiko"`
	Businesses                []BusinessSaveEntry `json:"bisnis"`
}

// BusinessSaveEntry adalah satu baris grid bisnis yang dikirim layar.
//
// Tanpa nama bisnis: namanya milik master dan dibaca lewat join, tidak disimpan di baris
// ini. Mengirimkannya hanya akan menyimpan salinan yang dapat menyimpang dari masternya.
type BusinessSaveEntry struct {
	BusinessID  string `json:"id_bisnis"`
	Mandatory   bool   `json:"status_wajib"`
	MinDocument int    `json:"minimum_dokumen"`
}

// ViolationDTO adalah satu pelanggaran isian.
type ViolationDTO struct {
	Field   string `json:"isian"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan, detail}`. Klien membedakan jenis
// galat lewat `kode`, tidak pernah dengan mencocokkan teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(row daftardetailtipedokumen.DetailType) DetailTypeDTO {
	businesses := make([]BusinessRuleDTO, 0, len(row.Businesses))
	for _, business := range row.Businesses {
		businesses = append(businesses, BusinessRuleDTO{
			BusinessID:   business.BusinessID,
			BusinessName: business.BusinessName,
			Mandatory:    business.Mandatory,
			MinDocument:  business.MinDocument,
		})
	}
	return DetailTypeDTO{
		ID:                        row.ID,
		DocumentTypeID:            row.DocumentTypeID,
		DocumentTypeName:          row.DocumentTypeName,
		Detail:                    row.Detail,
		InsuredStatus:             row.InsuredStatus,
		CauseOfLossID:             row.CauseOfLossID,
		CauseOfLossDescription:    row.CauseOfLossDescription,
		ObjectDocumentID:          row.ObjectDocumentID,
		ObjectDocumentDescription: row.ObjectDocumentDescription,
		Risk:                      row.Risk,
		Businesses:                businesses,
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya daftar kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri, dan
// satu layar yang lupa akan gagal saat masternya masih kosong. Berlaku pula untuk senarai
// bisnis di dalam setiap barisnya.
func toListDTO(list []daftardetailtipedokumen.DetailType) []DetailTypeDTO {
	result := make([]DetailTypeDTO, 0, len(list))
	for _, row := range list {
		result = append(result, toDTO(row))
	}
	return result
}

// toReferenceResponse mengubah keempat daftar pilihan.
func toReferenceResponse(references usecase.References, portal string) ReferenceResponse {
	documentTypes := make([]ChoiceDTO, 0, len(references.DocumentTypes))
	for _, row := range references.DocumentTypes {
		documentTypes = append(documentTypes, ChoiceDTO{ID: row.ID, Name: row.Name})
	}

	causes := make([]ChoiceDTO, 0, len(references.CausesOfLoss))
	for _, row := range references.CausesOfLoss {
		causes = append(causes, ChoiceDTO{ID: row.ID, Name: row.Description})
	}

	objects := make([]ChoiceDTO, 0, len(references.ObjectDocuments))
	for _, row := range references.ObjectDocuments {
		objects = append(objects, ChoiceDTO{ID: row.ID, Name: row.Description})
	}

	businesses := make([]ChoiceDTO, 0, len(references.Businesses))
	for _, row := range references.Businesses {
		businesses = append(businesses, ChoiceDTO{ID: row.ID, Name: row.Name})
	}

	unavailable := make([]string, 0, len(references.Unavailable))
	unavailable = append(unavailable, references.Unavailable...)

	return ReferenceResponse{
		DocumentTypes:   documentTypes,
		CausesOfLoss:    causes,
		ObjectDocuments: objects,
		Businesses:      businesses,
		Unavailable:     unavailable,
		Portal:          portal,
	}
}

// toInput mengubah isian form menjadi nilai domain.
func toInput(request SaveRequest) daftardetailtipedokumen.Input {
	businesses := make([]daftardetailtipedokumen.BusinessInput, 0, len(request.Businesses))
	for _, business := range request.Businesses {
		businesses = append(businesses, daftardetailtipedokumen.BusinessInput{
			BusinessID:  business.BusinessID,
			Mandatory:   business.Mandatory,
			MinDocument: business.MinDocument,
		})
	}
	return daftardetailtipedokumen.Input{
		DocumentTypeID:            request.DocumentTypeID,
		Detail:                    request.Detail,
		InsuredStatus:             request.InsuredStatus,
		CauseOfLossID:             request.CauseOfLossID,
		CauseOfLossDescription:    request.CauseOfLossDescription,
		ObjectDocumentID:          request.ObjectDocumentID,
		ObjectDocumentDescription: request.ObjectDocumentDescription,
		Risk:                      request.Risk,
		Businesses:                businesses,
	}
}
