// Package inboxrclpuclhttp adalah lapisan transport modul Inbox RCL/PUCL.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: WorkItem membawa Reference, isian yang
// tidak pernah digambar sebagai kolom.
package inboxrclpuclhttp

import (
	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/usecase"
)

// WorkItemDTO adalah satu baris pada grid.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang digambar
// dari `kolom` pada tab yang sedang terbuka — bukan dari ada-tidaknya isian, karena isian
// yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya menghilang
// begitu saja.
//
// Di layar ini sifat itu bukan kehalusan: `tanggal_cetak_surat` SELALU kosong pada tab
// "Cetak Surat" — penyaringnya `IS NULL` — dan kolomnya tetap harus digambar karena layar
// lama menggambarnya.
type WorkItemDTO struct {
	// Reference adalah kunci teknis Pega yang dibutuhkan tombol rincian.
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom.
	Reference string `json:"referensi"`

	CaseID       string `json:"no_case"`
	PolicyNumber string `json:"no_polis"`
	InsuredName  string `json:"nama_tertanggung"`

	// InboxEntryAt adalah "Tanggal Masuk Inbox" — tanggal klaimnya DIKIRIM ke jalur
	// RCL/PUCL, bukan tanggal objek kerjanya dibuat. Keduanya kolom yang berbeda, dan yang
	// dipakai mengurutkan justru yang TIDAK dikirim ke sini.
	InboxEntryAt string `json:"tanggal_masuk_inbox"`

	AnalystNote string `json:"deskripsi_analyst"`

	// Track adalah jalur penanganan — "RCL" atau "PUCL".
	//
	// Kosong bila kode jalurnya tidak dikenali, meniru `CASE` tanpa `ELSE` di sistem lama.
	// Judul kolomnya berbeda antartab: "Status RCL/PUCL" pada dua tab pertama, "Status"
	// pada tab Klaim MSIG — dan perbedaan itu datang dari `kolom`, bukan dari sini.
	Track string `json:"status_rcl_pucl"`

	// LetterPrintedAt SELALU kosong pada tab "Cetak Surat", dan itu bukan data hilang —
	// justru itulah arti tab tersebut.
	LetterPrintedAt string `json:"tanggal_cetak_surat"`

	// ClaimAge dikirim sebagai TEKS, bukan angka. Satuannya tidak diketahui: tidak satu pun
	// kueri di export menghitungnya, dan tidak ada DDL yang menyatakan tipenya (`R-08`).
	ClaimAge string `json:"lama_klaim"`

	ExpiryStatus string `json:"status_kadaluarsa"`
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	// Key menyebut isian mana pada baris yang digambar.
	Key string `json:"kunci"`

	// Title adalah judul kolom yang dibaca pengguna.
	Title string `json:"judul"`
}

// TabDTO adalah satu tab beserta bentuk gridnya.
type TabDTO struct {
	Code        string `json:"kode"`
	Name        string `json:"nama"`
	Description string `json:"keterangan"`

	Columns []ColumnDTO `json:"kolom"`

	// HasDateRangeReport menyatakan tombol ekspor tab ini menghasilkan LAPORAN HARIAN
	// berbasis rentang tanggal, bukan salinan tabel.
	//
	// Layar memakainya untuk menampilkan kedua isian tanggal — dan untuk menjelaskan bahwa
	// isian itu TIDAK menyaring tabel di bawahnya.
	HasDateRangeReport bool `json:"punya_laporan_rentang_tanggal"`

	// Notice adalah keterangan yang berlaku pada tab ini saja, digambar di atas grid.
	// Kosong berarti tidak ada.
	Notice string `json:"catatan,omitempty"`

	// Ketiga isian berikut menyatakan tab yang digambar tetapi belum dapat diisi.
	//
	// Tidak ada tab terhalang di layar ini. Isian tetap dikirim sebagai DATA supaya
	// penambahan tab terhalang kelak tidak menuntut perubahan kontrak API maupun suntingan
	// frontend.
	Blocked       bool   `json:"terhalang"`
	BlockedReason string `json:"alasan_terhalang,omitempty"`
	BlockedOwner  string `json:"pemilik_penghalang,omitempty"`
}

// MetadataResponse adalah jawaban GET /api/inbox-rcl-pucl/tab.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"tab"`
	DefaultTab string   `json:"tab_bawaan"`

	// ReportColumns adalah kolom berkas laporan harian.
	//
	// Dikirim supaya layar dapat menyebutkan isi berkasnya SEBELUM diunduh — isinya
	// berbeda dari tabel yang sedang dilihat, dan perbedaan itu tidak boleh baru ketahuan
	// setelah berkasnya dibuka.
	ReportColumns []ColumnDTO `json:"kolom_laporan"`

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string `json:"selisih_terencana"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal
	// yang sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`
}

// PaginationDTO adalah keterangan halaman.
type PaginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// ListResponse adalah jawaban GET /api/inbox-rcl-pucl.
type ListResponse struct {
	Tab        TabDTO        `json:"tab"`
	Items      []WorkItemDTO `json:"baris"`
	Pagination PaginationDTO `json:"paginasi"`
	Portal     string        `json:"portal"`
}

// LetterDraftDTO adalah bagian "Lampiran Surat" pada layar kerja.
//
// Tiga isiannya DITURUNKAN dari anak klaim, bukan dibaca dari kolomnya sendiri — lihat
// `inboxrclpucl.LetterDraft`. Nama fieldnya tetap berbahasa Indonesia karena ia kontrak yang
// dibaca layar (`D-80`), dan mengikuti judul isian di section aslinya (`D-13`).
type LetterDraftDTO struct {
	Track string `json:"rcl_pucl"`

	// TrackCode adalah kode jalur MENTAH.
	//
	// Dikirim selain `rcl_pucl` karena nilai `3` menyembunyikan seluruh layar ini di Pega,
	// dan teks kosong dari penerjemah tidak dapat dibedakan dari kode yang memang kosong.
	TrackCode string `json:"kode_rcl_pucl"`

	AnalystNote  string `json:"deskripsi_analyst"`
	PolicyNumber string `json:"no_polis"`
	LossDate     string `json:"tanggal_kejadian"`

	// InsuredName dan SumInsured sama-sama DITURUNKAN dari anak klaim, tetapi dari kolom
	// yang BERBEDA — dan itu selisih terencana terhadap Pega, tempat keduanya diisi dari
	// ekspresi yang sama persis sehingga "UP" ikut berisi nama objek.
	//
	// `up` karena itu berisi ANGKA di sini. Lihat `inboxrclpucl.LetterDraft.SumInsured`
	// untuk keputusannya dan bukti sumbernya.
	InsuredName string `json:"nama_peserta"`
	SumInsured  string `json:"up"`

	BillAmount string `json:"jumlah_tagihan"`
}

// DocumentReceiptDTO adalah bagian "Penerimaan Dokumen".
//
// Hanya satu isiannya punya kolom yang diketahui. Sisanya tidak dikirim sama sekali — dan
// itu disengaja: mengirim isian kosong yang tidak punya sumber akan membuat layar mengira
// datanya memang belum diisi, padahal kolomnya yang belum ditemukan. Yang menjelaskan
// ketiadaannya adalah `UnmappedFields`.
type DocumentReceiptDTO struct {
	PUCLNote string `json:"komentar_pucl"`
}

// ClaimDetailResponse adalah jawaban GET /api/inbox-rcl-pucl/klaim/{referensi}.
type ClaimDetailResponse struct {
	Reference   string `json:"referensi"`
	ClaimNumber string `json:"no_case"`

	Letter          LetterDraftDTO     `json:"lampiran_surat"`
	DocumentReceipt DocumentReceiptDTO `json:"penerimaan_dokumen"`

	// UnmappedFields menyebut isian layar lama yang BELUM punya kolom terverifikasi.
	//
	// Ia dikirim sebagai DATA, bukan ditulis tetap di layar, supaya daftarnya menyusut di
	// satu tempat begitu kolomnya ditemukan — dan supaya pengguna yang membandingkan kedua
	// layar berdampingan tahu mana yang belum terbawa alih-alih mengira datanya hilang.
	UnmappedFields []string `json:"isian_belum_terpetakan"`

	// WriteBlocked menyatakan layar ini di Pega adalah layar TULIS.
	//
	// Layar memakainya untuk menjelaskan mengapa tidak ada satu pun tombol simpan di sini,
	// alih-alih membiarkan pengguna mencarinya.
	WriteBlocked bool `json:"tindakan_masih_di_pega"`

	Portal string `json:"portal"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toWorkItemDTO mengubah satu baris.
func toWorkItemDTO(item inboxrclpucl.WorkItem) WorkItemDTO {
	return WorkItemDTO{
		Reference:       item.Reference,
		CaseID:          item.CaseID,
		PolicyNumber:    item.PolicyNumber,
		InsuredName:     item.InsuredName,
		InboxEntryAt:    item.InboxEntryAt,
		AnalystNote:     item.AnalystNote,
		Track:           item.Track,
		LetterPrintedAt: item.LetterPrintedAt,
		ClaimAge:        item.ClaimAge,
		ExpiryStatus:    item.ExpiryStatus,
	}
}

// toWorkItemListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toWorkItemListDTO(items []inboxrclpucl.WorkItem) []WorkItemDTO {
	result := make([]WorkItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toWorkItemDTO(item))
	}
	return result
}

// toColumnListDTO mengubah senarai kolom.
func toColumnListDTO(columns []inboxrclpucl.Column) []ColumnDTO {
	result := make([]ColumnDTO, 0, len(columns))
	for _, column := range columns {
		result = append(result, ColumnDTO{Key: column.Key, Title: column.Title})
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxrclpucl.Tab) TabDTO {
	return TabDTO{
		Code:               tab.Code,
		Name:               tab.Name,
		Description:        tab.Description,
		Columns:            toColumnListDTO(tab.Columns),
		HasDateRangeReport: tab.HasDateRangeReport,
		Notice:             tab.Notice,
		Blocked:            tab.Blocked,
		BlockedReason:      tab.BlockedReason,
		BlockedOwner:       tab.BlockedOwner,
	}
}

// toMetadataResponse merakit jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		tabs = append(tabs, toTabDTO(tab))
	}

	differences := make([]string, 0, len(meta.PlannedDifferences))
	differences = append(differences, meta.PlannedDifferences...)

	return MetadataResponse{
		Tabs:               tabs,
		DefaultTab:         meta.DefaultTab,
		ReportColumns:      toColumnListDTO(meta.ReportColumns),
		PlannedDifferences: differences,
		Portal:             portalAlias,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxrclpucl.Page) PaginationDTO {
	return PaginationDTO{
		Page:       page.Pagination.Page,
		Size:       page.Pagination.Size,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
	}
}

// unmappedLetterFields adalah isian layar kerja yang tidak dapat diisi dari kolom tabel.
//
// # Sebabnya BUKAN kolom yang hilang
//
// Work Owner menjelaskan 2026-09-24: kesembilannya diambil dari **clipboard** Pega —
// `.ClaimData.PUCLStatus.NIK`, `.BusinessUnitSeksi`, `.Perihal`, `.Keterangan1`…`3`,
// `.ClaimData.EmailLOD`, `.TanggalTerimaDokumenPUCL`, dan daftar `.DateReceivedDocument`.
//
// Properti clipboard yang tidak dioptimasi TIDAK punya kolom sendiri; nilainya hidup di
// dalam objek kerja Pega. Itu menjelaskan mengapa pencarian ke seluruh export tidak
// menemukan satu pun kolomnya, dan mengapa mencarinya lagi tidak akan menemukannya.
//
// Akibatnya berbeda dari "kolom belum ditemukan": ini bukan pertanyaan yang dapat dijawab
// DBA dengan menunjuk kolom, melainkan keadaan yang baru berubah bila propertinya diekspos
// sebagai kolom, atau bila modul ini kelak memiliki tabelnya sendiri.
//
// Namanya diambil dari `pyLabelFieldValue` pada sel masing-masing, bukan dari nama properti
// Pega — yang membacanya petugas klaim. Satu di antaranya patut disadari: label **"No
// Kontrak"** menempel pada properti `.ClaimData.PUCLStatus.NIK`. Label dan properti di situ
// memang tidak sejalan, dan yang dibawa adalah LABEL-nya (`D-13`).
//
// Kesembilannya juga dicari di SELURUH export — `RDB List/`, `Database/*.prc`, `*.fnc`, dan kedua
// berkas CSV master — tanpa satu pun kemunculan sebagai kolom. Inventaris katalog Oracle
// (`docs/kolom-t-claimlist-admin.md`, 2026-09-22) pun tidak mendaftarkannya, sementara
// keenam kolom PUCL lain lengkap di sana.
//
// Ia ditulis sebagai kalimat yang dibaca pengguna, bukan nama properti Pega: yang membacanya
// petugas klaim, bukan orang yang menelusuri rule.
var unmappedLetterFields = []string{
	"No Kontrak",
	"Business Unit / Seksi",
	"Perihal",
	"Keterangan Pembuka",
	"Keterangan Isi",
	"Keterangan Penutup",
	"Email Tertanggung",
	"Tanggal Kelengkapan Dokumen",
	"Tanggal terima Dokumen (Tanggal · Keterangan)",
}

// toClaimDetailResponse merakit jawaban layar kerja satu klaim.
func toClaimDetailResponse(
	detail inboxrclpucl.ClaimDetail,
	portalAlias string,
) ClaimDetailResponse {
	unmapped := make([]string, 0, len(unmappedLetterFields))
	unmapped = append(unmapped, unmappedLetterFields...)

	return ClaimDetailResponse{
		Reference:   detail.Reference,
		ClaimNumber: detail.ClaimNumber,

		Letter: LetterDraftDTO{
			Track:        detail.Letter.Track,
			TrackCode:    detail.Letter.TrackCode,
			AnalystNote:  detail.Letter.AnalystNote,
			PolicyNumber: detail.Letter.PolicyNumber,
			LossDate:     detail.Letter.LossDate,
			InsuredName:  detail.Letter.InsuredName,
			SumInsured:   detail.Letter.SumInsured,
			BillAmount:   detail.Letter.BillAmount,
		},

		DocumentReceipt: DocumentReceiptDTO{
			PUCLNote: detail.DocumentReceipt.PUCLNote,
		},

		UnmappedFields: unmapped,

		// Selalu true selama masa paralel. Ia dikirim sebagai isian, bukan ditulis tetap
		// di layar, supaya ia dapat berubah di satu tempat begitu kepemilikan tabelnya
		// berpindah (`P-1`).
		WriteBlocked: true,

		Portal: portalAlias,
	}
}

// toListResponse merakit jawaban isi satu tab.
func toListResponse(listed usecase.Listed, portalAlias string) ListResponse {
	return ListResponse{
		Tab:        toTabDTO(listed.Query.Tab),
		Items:      toWorkItemListDTO(listed.Page.Items),
		Pagination: toPaginationDTO(listed.Page),
		Portal:     portalAlias,
	}
}
