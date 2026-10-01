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
	// Ketiga kodenya punya teks — "RCL", "PUCL", "Notification". Kosong hanya bila
	// kodenya sendiri kosong atau di luar ketiganya.
	// Judul kolomnya berbeda antartab: "Status RCL/PUCL" pada dua tab pertama, "Status"
	// pada tab Klaim MSIG — dan perbedaan itu datang dari `kolom`, bukan dari sini.
	Track string `json:"status_rcl_pucl"`

	// LetterPrintedAt SELALU kosong pada tab "Cetak Surat", dan itu bukan data hilang —
	// justru itulah arti tab tersebut.
	LetterPrintedAt string `json:"tanggal_cetak_surat"`

	// ClaimAge dikirim sebagai TEKS TANGGAL, bukan angka dan bukan durasi.
	//
	// Judulnya menyebut durasi; isinya **tanggal kirim untuk proses PUCL** (Work Owner,
	// 2026-09-30), dan kolomnya terverifikasi bertipe `TIMESTAMP(6)` di Oracle. Judulnya
	// tetap dibawa apa adanya (`D-13`); yang dibentuk hanya isinya, oleh
	// `inboxrclpucl.DisplayTimeText` di penyimpanan.
	ClaimAge string `json:"lama_klaim"`

	ExpiryStatus string `json:"status_kadaluarsa"`

	// CreatedAt adalah kolom yang MENGURUTKAN tabel ini — `PXCREATEDATETIME`.
	//
	// Ia tidak ada di layar lama. Ditambahkan 2026-09-30 atas keputusan Work Owner supaya
	// tabelnya tidak lagi terbaca acak; urutan barisnya sendiri tidak berubah sedikit pun.
	CreatedAt string `json:"tanggal_dibuat"`
}

// DifferenceDTO adalah satu selisih terhadap Pega yang sudah diputuskan.
//
// Dua bagian, karena pembacanya dua: `ringkas` untuk petugas klaim yang sedang memakai
// layar, `rincian` untuk penguji kesetaraan yang sedang mencari pemetaan `P-5`-nya. Layar
// menggambar yang pertama dan menyembunyikan yang kedua di balik satu ketukan.
//
// Keduanya SELALU dikirim. Mengirim `rincian` hanya saat diminta akan menuntut satu
// permintaan tambahan per butir, padahal seluruh isinya teks tetap yang sudah ada di memori.
type DifferenceDTO struct {
	Summary string `json:"ringkas"`
	Detail  string `json:"rincian"`
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
	PlannedDifferences []DifferenceDTO `json:"selisih_terencana"`

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

	// InsuredName dan SumInsured DITURUNKAN dari kolom yang SAMA — nama objek pertama —
	// dan itu BUKAN selisih, melainkan perilaku Pega apa adanya.
	//
	// `up` karena itu berisi NAMA OBJEK, bukan angka. Ia terbaca seperti salin-tempel yang
	// keliru dan sempat "diperbaiki" menjadi nilai pertanggungan pada 2026-09-24; Work
	// Owner meralatnya hari itu juga, dan perbaikannya dicabut seluruhnya. Lihat
	// `inboxrclpucl.LetterDraft.SumInsured`.
	InsuredName string `json:"nama_peserta"`
	SumInsured  string `json:"up"`

	BillAmount string `json:"jumlah_tagihan"`

	// Keempat isian surat, dibaca dari kolom `TC_PNC_PUCL` sejak 2026-10-01. Sebelumnya
	// keempatnya digambar bertanda "di clipboard Pega".
	Subject     string `json:"perihal"`
	OpeningNote string `json:"keterangan_pembuka"`
	BodyNote    string `json:"keterangan_isi"`
	ClosingNote string `json:"keterangan_penutup"`
}

// DocumentReceiptDTO adalah bagian "Penerimaan Dokumen".
//
// TIGA dari empat isiannya kini punya kolom. Yang tersisa tanpa sumber hanyalah daftar
// "Tanggal terima Dokumen", dan ketiadaannya dijelaskan `UnmappedFields` — bukan dikirim
// sebagai isian kosong, yang akan membuat layar mengira datanya memang belum diisi.
type DocumentReceiptDTO struct {
	PUCLNote string `json:"komentar_pucl"`

	// CompleteAt — "Tanggal Kelengkapan Dokumen", dari `TGL_TERIMA_DOKUMEN_PUCL`.
	CompleteAt string `json:"tanggal_kelengkapan_dokumen"`

	// ReceivedDates — grid "Tanggal Terima Dokumen".
	//
	// Paling banyak SATU baris: daftarnya page list tanpa tabel, dan hanya baris pertamanya
	// yang diekspos sebagai kolom. Lihat `inboxrclpucl.DocumentReceipt.ReceivedDates`.
	ReceivedDates []ReceivedDateDTO `json:"tanggal_terima_dokumen"`

	// ReceivedDatesPartial menyatakan daftar di atas MUNGKIN tidak lengkap.
	//
	// Dikirim sebagai data, bukan ditulis tetap di layar, supaya ia hilang di satu tempat
	// begitu baris kedua dan seterusnya terbaca lewat layanan Pega.
	ReceivedDatesPartial bool `json:"tanggal_terima_dokumen_sebagian"`

	// InsuredEmail — "Email Tertanggung", dari `T_CLAIM_PNC.EMAIL_LOD`.
	//
	// Kolomnya ADA tetapi kosong pada seluruh 2.206 baris (dihitung 2026-10-01), sehingga
	// isian ini akan tergambar kosong. Itu keadaan DATA, bukan isian yang belum terpetakan —
	// karena itu ia TIDAK masuk `UnmappedFields`.
	InsuredEmail string `json:"email_tertanggung"`
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

	// ShowsDocumentReceipt menyatakan tab "Penerimaan Dokumen" digambar untuk klaim ini.
	//
	// Ia DATA dari server, bukan pemeriksaan kode jalur di layar. Kodenya (`3`) adalah nilai
	// milik sistem lama, dan menaruh perbandingannya di frontend berarti satu nilai bisnis
	// hidup di dua tempat yang dapat berselisih tanpa ketahuan (`D-15`).
	ShowsDocumentReceipt bool `json:"tab_penerimaan_dokumen_tampil"`

	// Buttons menyatakan TOMBOL mana yang digambar untuk klaim ini.
	//
	// Alasannya sama dengan ShowsDocumentReceipt: syaratnya memakai nilai milik sistem lama
	// (`RCL_PUCL`, `MSIG`, Group Panel `002`/`005`), dan memeriksanya di React berarti
	// aturan bisnis hidup di dua tempat.
	Buttons ScreenButtonsDTO `json:"tombol"`

	// WriteBlocked menyatakan layar ini di Pega adalah layar TULIS.
	//
	// Layar memakainya untuk menjelaskan mengapa tidak ada satu pun tombol simpan di sini,
	// alih-alih membiarkan pengguna mencarinya.
	WriteBlocked bool `json:"tindakan_masih_di_pega"`

	Portal string `json:"portal"`
}

// ScreenButtonsDTO menyatakan tombol mana yang digambar.
//
// Nama isiannya memakai NAMA TOMBOL seperti yang terbaca pengguna, bukan nama kondisinya.
// Orang yang membandingkan tanggapan ini dengan layar Pega berdampingan mencari nama yang
// tertulis di tombolnya.
type ScreenButtonsDTO struct {
	DownloadDocument bool `json:"download_dokumen"`
	CloseClaim       bool `json:"tutup_klaim"`
	UploadDocument   bool `json:"unggah_dokumen"`
	ViewDocument     bool `json:"lihat_dokumen"`
	Save             bool `json:"save"`
	RejectClaim      bool `json:"tolak_klaim"`
	SendToAnalyst    bool `json:"kirim_ke_analyst"`
	SendToPICTeknik  bool `json:"kirim_ke_pic_teknik"`
}

func screenButtonsOf(b inboxrclpucl.ScreenButtons) ScreenButtonsDTO {
	return ScreenButtonsDTO{
		DownloadDocument: b.DownloadDocument,
		CloseClaim:       b.CloseClaim,
		UploadDocument:   b.UploadDocument,
		ViewDocument:     b.ViewDocument,
		Save:             b.Save,
		RejectClaim:      b.RejectClaim,
		SendToAnalyst:    b.SendToAnalyst,
		SendToPICTeknik:  b.SendToPICTeknik,
	}
}

// ReceivedDateDTO adalah satu baris grid "Tanggal Terima Dokumen".
//
// Nama isiannya mengikuti JUDUL KOLOM yang dibaca pengguna, bukan nama properti Pega.
type ReceivedDateDTO struct {
	Tanggal    string `json:"tanggal"`
	Keterangan string `json:"keterangan"`
}

func receivedDatesOf(rows []inboxrclpucl.ReceivedDocumentDate) []ReceivedDateDTO {
	// Senarai KOSONG, bukan nil, supaya JSON-nya `[]` dan bukan `null`. Layar membedakan
	// "daftarnya kosong" dari "isiannya tidak ada".
	out := make([]ReceivedDateDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, ReceivedDateDTO{Tanggal: row.Date, Keterangan: row.Note})
	}
	return out
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
		CreatedAt:       item.CreatedAt,
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

	differences := make([]DifferenceDTO, 0, len(meta.PlannedDifferences))
	for _, difference := range meta.PlannedDifferences {
		differences = append(differences, DifferenceDTO{
			Summary: difference.Summary,
			Detail:  difference.Detail,
		})
	}

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
// # Daftarnya TURUN dari sembilan menjadi lima pada 2026-10-01
//
// Empat di antaranya — Perihal, Keterangan Pembuka, Keterangan Isi, Keterangan Penutup —
// ternyata PUNYA kolom di `POOLDATA.TC_PNC_PUCL` yang berjalan (`PERIHAL`, `KETERANGAN1`,
// `KETERANGAN2`, `KETERANGAN3`), dan terisi. Keempatnya kini dibaca apa adanya.
//
// Yang mengubah keadaan bukan pencarian yang lebih teliti melainkan **kolomnya memang baru
// ada**: `Database/CREATE_TABLE_3.SQL` mendefinisikan 26 kolom, tabel yang berjalan punya 30.
// Ini persis jalan keluar yang §79.3 sebut sebagai satu-satunya — properti clipboard diekspos
// menjadi kolom — dan ia sudah ditempuh untuk keempat isian itu.
//
// # Kelima yang tersisa, dan sebabnya BUKAN kolom yang hilang
//
// `.ClaimData.PUCLStatus.NIK`, `.BusinessUnitSeksi`, `.ClaimData.EmailLOD`,
// `.TanggalTerimaDokumenPUCL`, dan daftar `.DateReceivedDocument`.
//
// Properti clipboard yang tidak dioptimasi TIDAK punya kolom sendiri; nilainya hidup di
// dalam objek kerja Pega. Itu menjelaskan mengapa pencarian ke seluruh export tidak
// menemukan satu pun kolomnya, dan mengapa mencarinya lagi tidak akan menemukannya.
//
// Satu hipotesis sempat tampak menjanjikan dan GUGUR: `EMAIL_1` pada tabel objek kerja Pega
// terisi 61 dari 64 baris antrean, berisi daftar alamat berkoma — tetapi pada klaim yang
// layarnya diperiksa ia memuat SATU alamat sementara Pega menggambar DUA. Memakainya akan
// menampilkan penerima surat yang salah tanpa satu pun galat.
//
// Namanya diambil dari `pyLabelFieldValue` pada sel masing-masing, bukan dari nama properti
// Pega — yang membacanya petugas klaim. Satu di antaranya patut disadari: label **"No
// Kontrak"** menempel pada properti `.ClaimData.PUCLStatus.NIK`. Label dan properti di situ
// memang tidak sejalan, dan yang dibawa adalah LABEL-nya (`D-13`).
// Daftarnya menyusut dua kali: dari sembilan menjadi lima saat keempat kolom surat ditemukan
// (§84.2), lalu menjadi TIGA pada 2026-10-01 saat "Email Tertanggung" dan "Tanggal
// Kelengkapan Dokumen" ikut terpetakan atas penetapan Work Owner.
var unmappedLetterFields = []string{
	"No Kontrak",
	"Business Unit / Seksi",
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
			Subject:      detail.Letter.Subject,
			OpeningNote:  detail.Letter.OpeningNote,
			BodyNote:     detail.Letter.BodyNote,
			ClosingNote:  detail.Letter.ClosingNote,
		},

		DocumentReceipt: DocumentReceiptDTO{
			PUCLNote:             detail.DocumentReceipt.PUCLNote,
			ReceivedDates:        receivedDatesOf(detail.DocumentReceipt.ReceivedDates),
			ReceivedDatesPartial: true,
			CompleteAt:           detail.DocumentReceipt.CompleteAt,
			InsuredEmail:         detail.DocumentReceipt.InsuredEmail,
		},

		UnmappedFields: unmapped,

		// Syarat tab kedua ditegakkan di sini, bukan di layar. Lihat
		// `inboxrclpucl.ClaimDetail.ShowsDocumentReceipt`.
		ShowsDocumentReceipt: detail.ShowsDocumentReceipt(),

		// Susunan tombolnya pun ditegakkan di domain. Lihat
		// `inboxrclpucl.ClaimDetail.Buttons`.
		Buttons: screenButtonsOf(detail.Buttons()),

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

// DocumentDTO adalah satu baris daftar dokumen klaim.
//
// Nama isiannya Indonesia (`D-80`) dan mengikuti judul yang dilihat pengguna, bukan nama
// kolomnya.
type DocumentDTO struct {
	ID           string `json:"id"`
	Nama         string `json:"nama"`
	Kategori     string `json:"kategori"`
	SubKategori  string `json:"sub_kategori"`
	DiunggahPada string `json:"diunggah_pada"`
	DiunggahOleh string `json:"diunggah_oleh"`
}

// DocumentListResponse adalah jawaban daftar dokumen satu klaim.
type DocumentListResponse struct {
	Dokumen []DocumentDTO `json:"dokumen"`

	// Catatan menyatakan bahwa daftar ini mungkin TIDAK lengkap.
	//
	// Ia dikirim sebagai DATA, bukan ditulis tetap di layar, supaya ia dapat hilang di satu
	// tempat begitu jalur lampiran bawaan Pega ikut terbaca. Alasannya di kueri `documents`:
	// dokumen yang hanya ada di tabel lampiran Pega tidak muncul lewat jalur ini, dan daftar
	// kosong tanpa keterangan terbaca sebagai "klaim ini tidak berdokumen".
	Catatan string `json:"catatan"`

	Portal string `json:"portal"`
}

// documentListNote adalah keterangan yang menyertai setiap daftar dokumen.
const documentListNote = "Dokumen yang diunggah lewat jalur lama Pega belum tentu muncul di " +
	"daftar ini. Bila dokumen yang Anda cari tidak ada, periksa klaimnya di Pega."

func toDocumentListResponse(
	documents []inboxrclpucl.Document,
	portal string,
) DocumentListResponse {
	rows := make([]DocumentDTO, 0, len(documents))
	for _, document := range documents {
		rows = append(rows, DocumentDTO{
			ID:           document.ID,
			Nama:         document.Name,
			Kategori:     document.Category,
			SubKategori:  document.SubCategory,
			DiunggahPada: document.UploadedAt,
			DiunggahOleh: document.UploadedBy,
		})
	}
	return DocumentListResponse{Dokumen: rows, Catatan: documentListNote, Portal: portal}
}
