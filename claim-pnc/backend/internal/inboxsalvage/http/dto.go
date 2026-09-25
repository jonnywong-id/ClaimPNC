// Package inboxsalvagehttp adalah lapisan transport modul Inbox Salvage.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: Row membawa TransferStatus dan
// RequestNote, dua isian yang tidak pernah digambar sebagai kolom tetapi menentukan isi
// kolom lain.
package inboxsalvagehttp

import (
	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/usecase"
)

// RowDTO adalah satu baris pada grid mana pun.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang digambar
// dari `kolom` pada daftar yang sedang terbuka — bukan dari ada-tidaknya isian, karena isian
// yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya menghilang
// begitu saja.
//
// Di layar ini sifat itu bukan kehalusan: `catatan` SELALU kosong pada kelima daftar yang
// menggambarnya, dan kolomnya tetap harus digambar karena layar lama menggambarnya.
type RowDTO struct {
	// Reference adalah kunci baris yang dibutuhkan tombol rincian.
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom. Isinya berbeda menurut keluarga
	// daftarnya — nomor klaim pada enam daftar berbasis klaim, ID salvage pada ketujuh
	// daftar berbasis pengajuan.
	Reference string `json:"referensi"`

	ClaimNo   string `json:"no_klaim"`
	SalvageID string `json:"id_salvage"`

	// InputDate adalah "Tanggal Input" — tanggal PENGAJUAN salvage dibuat.
	//
	// Bukan tanggal kejadian. Alias di kueri lama menyebutnya `"DateOfLoss"`, dan
	// menyalinnya akan menampilkan tanggal yang salah tanpa satu pun galat.
	InputDate string `json:"tanggal_input"`

	// LossDate adalah "Tgl Kejadian". Hanya daftar Salvage Outstanding menggambarnya.
	LossDate string `json:"tanggal_kejadian"`

	PIC string `json:"pic"`

	// BusinessName berjudul "COB" pada daftar Salvage Outstanding dan "Lokasi" pada kelima
	// daftar berbasis klaim lainnya. Kolomnya sama; judulnya yang berbeda, dan perbedaan
	// itu datang dari `kolom`, bukan dari sini.
	BusinessName string `json:"nama_bisnis"`

	ObjectName      string `json:"nama_objek"`
	SalvageType     string `json:"jenis_salvage"`
	SalvageLocation string `json:"lokasi_salvage"`

	// AuctionStatus adalah "Status Lelang" — "Terjual" atau "Belum Terjual".
	//
	// Ia hasil hitungan atas nilai akseptasi, bukan kolom tersendiri.
	AuctionStatus string `json:"status_lelang"`

	// Nilai uang dikirim sebagai TEKS, bukan angka.
	//
	// `D-51` menetapkan nilai uang disimpan presisi penuh dan hanya dibulatkan saat
	// ditampilkan. Mengirimnya sebagai angka JSON berarti melewatkannya lewat bilangan
	// pecahan biner JavaScript, yang membulatkannya sebelum pembulatan yang disengaja
	// sempat terjadi.
	EstimateValue string `json:"nilai_pengajuan_pic"`
	RequestValue  string `json:"nilai_request_balai_lelang"`

	Email  string `json:"email"`
	Remark string `json:"keterangan_pic"`

	// SubmissionType adalah "Tipe Pengajuan" — "Pengajuan Baru" atau "Request Balai
	// Lelang". Ia diturunkan dari terisi-tidaknya catatan request, bukan kolom tersendiri.
	SubmissionType string `json:"tipe_pengajuan"`

	// Aging adalah "Aging", berbentuk `"N day"`. Satuannya HARI KALENDER, bukan hari
	// kerja seperti di Pega — lihat selisih terencana.
	Aging string `json:"aging"`

	// Note adalah "Catatan". SELALU kosong, dan itu keadaan di Pega pula.
	Note string `json:"catatan"`
}

func toRowDTO(row inboxsalvage.Row) RowDTO {
	return RowDTO{
		Reference:       row.Reference,
		ClaimNo:         row.ClaimNo,
		SalvageID:       row.SalvageID,
		InputDate:       row.InputDate,
		LossDate:        row.LossDate,
		PIC:             row.PIC,
		BusinessName:    row.BusinessName,
		ObjectName:      row.ObjectName,
		SalvageType:     row.SalvageType,
		SalvageLocation: row.SalvageLocation,
		AuctionStatus:   row.AuctionStatus,
		EstimateValue:   row.EstimateValue,
		RequestValue:    row.RequestValue,
		Email:           row.Email,
		Remark:          row.Remark,
		SubmissionType:  row.SubmissionType,
		Aging:           row.Aging,
		Note:            row.Note,
	}
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	Key     string `json:"kunci"`
	Title   string `json:"judul"`
	Numeric bool   `json:"angka"`
}

// TabDTO adalah satu daftar beserta kolom dan keterangannya.
type TabDTO struct {
	Code        string      `json:"kode"`
	Name        string      `json:"nama"`
	Description string      `json:"keterangan"`
	Columns     []ColumnDTO `json:"kolom"`

	// SearchLabel adalah judul kotak pencarian, kosong bila daftar ini tidak punya.
	SearchLabel string `json:"label_pencarian"`

	// SearchExact menyatakan pencarian daftar ini COCOK PERSIS.
	//
	// Layar memakainya untuk menjelaskan ke pengguna mengapa separuh nomor klaim tidak
	// menghasilkan apa-apa — keterangan yang tidak ada di Pega, dan yang ketiadaannya
	// membuat perilaku ini terbaca sebagai kerusakan.
	SearchExact bool `json:"pencarian_cocok_persis"`

	// Notice adalah keterangan yang berlaku pada daftar ini saja.
	Notice string `json:"catatan_daftar,omitempty"`
}

func toTabDTO(tab inboxsalvage.Tab) TabDTO {
	columns := make([]ColumnDTO, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		columns = append(columns, ColumnDTO{
			Key:     column.Key,
			Title:   column.Title,
			Numeric: column.Numeric,
		})
	}

	return TabDTO{
		Code:        tab.Code,
		Name:        tab.Name,
		Description: tab.Description,
		Columns:     columns,
		SearchLabel: tab.SearchLabel,
		SearchExact: tab.SearchExact,
		Notice:      tab.Notice,
	}
}

// StatusOptionDTO adalah satu pilihan daftar "Status Salvage" pada form Tambah.
type StatusOptionDTO struct {
	Code  string `json:"kode"`
	Label string `json:"label"`
}

// MetadataResponse adalah keterangan layar.
type MetadataResponse struct {
	Tabs          []TabDTO          `json:"daftar"`
	DefaultTab    string            `json:"daftar_bawaan"`
	StatusOptions []StatusOptionDTO `json:"pilihan_status_salvage"`
	UploadColumns []string          `json:"kolom_berkas_unggahan"`

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	//
	// Ia DIKIRIM ke layar, bukan hanya tercatat di kode. Selisih yang hanya tercatat di
	// komentar akan dilaporkan berulang kali sebagai kerusakan oleh orang yang
	// membandingkan layar baru dengan Pega berdampingan.
	PlannedDifferences []string `json:"selisih_terencana"`

	// Portal adalah alias entitas yang sedang dijawab.
	//
	// Ia dikirim supaya layar dapat memastikan jawabannya berasal dari portal yang sedang
	// dipilih — bukan dari portal utama sebagai cadangan (`R-20`).
	Portal string `json:"portal"`
}

func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		tabs = append(tabs, toTabDTO(tab))
	}

	options := make([]StatusOptionDTO, 0, len(meta.StatusOptions))
	for _, option := range meta.StatusOptions {
		options = append(options, StatusOptionDTO{Code: option.Code, Label: option.Label})
	}

	return MetadataResponse{
		Tabs:               tabs,
		DefaultTab:         meta.DefaultTab,
		StatusOptions:      options,
		UploadColumns:      meta.UploadColumns,
		PlannedDifferences: meta.PlannedDifferences,
		Portal:             portalAlias,
	}
}

// PageDTO adalah keterangan paginasi yang BENAR-BENAR dipakai.
type PageDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// ListResponse adalah isi satu daftar.
type ListResponse struct {
	Tab    TabDTO   `json:"daftar"`
	Rows   []RowDTO `json:"baris"`
	Paging PageDTO  `json:"paginasi"`

	// Search adalah kata kunci yang BENAR-BENAR dipakai.
	//
	// Ia dikembalikan karena tidak selalu sama dengan yang dikirim: kata kunci pada daftar
	// yang tidak punya kotak pencarian dibuang, bukan ditolak. Layar yang menggambar
	// kotaknya dari sini tidak akan menampilkan kata kunci yang sebenarnya diabaikan.
	Search string `json:"cari"`

	Portal string `json:"portal"`
}

func toListResponse(listed usecase.Listed, portalAlias string) ListResponse {
	rows := make([]RowDTO, 0, len(listed.Page.Items))
	for _, row := range listed.Page.Items {
		rows = append(rows, toRowDTO(row))
	}

	return ListResponse{
		Tab:  toTabDTO(listed.Query.Tab),
		Rows: rows,
		Paging: PageDTO{
			Page:       listed.Page.Pagination.Page,
			Size:       listed.Page.Pagination.Size,
			Total:      listed.Page.Total,
			TotalPages: listed.Page.TotalPages(),
		},
		Search: listed.Query.Search,
		Portal: portalAlias,
	}
}

// StatusCountDTO adalah satu baris tabel ringkas "Status Salvage / Jumlah".
type StatusCountDTO struct {
	Label string `json:"status_salvage"`
	Total int    `json:"jumlah"`

	// Tab adalah kode daftar yang dibuka bila barisnya diklik. Kosong berarti barisnya
	// tidak menuju daftar mana pun — dan satu baris memang begitu.
	Tab string `json:"daftar,omitempty"`
}

// CountsResponse adalah tabel ringkas di atas grid.
type CountsResponse struct {
	Rows   []StatusCountDTO `json:"baris"`
	Portal string           `json:"portal"`
}

func toCountsResponse(counts []inboxsalvage.StatusCount, portalAlias string) CountsResponse {
	rows := make([]StatusCountDTO, 0, len(counts))
	for _, count := range counts {
		rows = append(rows, StatusCountDTO{
			Label: count.Label,
			Total: count.Total,
			Tab:   count.Tab,
		})
	}
	return CountsResponse{Rows: rows, Portal: portalAlias}
}

// DetailItemRequest adalah satu baris grid "Detail Item Salvage" pada permintaan simpan.
type DetailItemRequest struct {
	Name     string `json:"nama_item"`
	Quantity string `json:"jumlah_item"`
	Unit     string `json:"satuan"`
	Remarks  string `json:"remark"`
}

// CreateRequest adalah badan permintaan simpan pengajuan salvage.
//
// Nama field mengikuti judul isian pada `Section/TambahData_Salvage-Section.xml` (`D-13`),
// bukan nama kolom basis data — kolomnya beralias menyesatkan dan `D-19` melarang
// membawanya ke kontrak.
type CreateRequest struct {
	// Mode kosong berarti membuat baru.
	Mode      string `json:"mode"`
	SalvageID string `json:"id_salvage"`

	ClaimNo      string `json:"nomor_klaim"`
	ObjectID     string `json:"id_object"`
	ObjectName   string `json:"nama_object"`
	CoverageID   string `json:"id_coverage"`
	CoverageName string `json:"nama_coverage"`

	InputDate     string `json:"tanggal_input"`
	SalvageType   string `json:"jenis_salvage"`
	Status        string `json:"status_salvage"`
	Location      string `json:"lokasi_salvage"`
	InJabodetabek bool   `json:"lokasi_salvage_di_jabodetabek"`
	Currency      string `json:"mata_uang"`
	MinimumValue  string `json:"minimum_salvage"`
	Quantity      string `json:"quantity_salvage"`
	OfferValue    string `json:"nilai_penawaran"`
	InsuredShare  string `json:"share_tertanggung"`
	Remark        string `json:"remark"`
	Email         string `json:"email"`

	SurveyorName  string `json:"nama_pic_survey"`
	SurveyorPhone string `json:"no_telp_pic_survey"`
	SurveyorEmail string `json:"email_pic_survey"`

	Items []DetailItemRequest `json:"detail_item_salvage"`
}

func (c CreateRequest) toInput() inboxsalvage.FormInput {
	mode := inboxsalvage.FormModeInsert
	if c.Mode == string(inboxsalvage.FormModeUpdate) {
		mode = inboxsalvage.FormModeUpdate
	}

	items := make([]inboxsalvage.DetailItem, 0, len(c.Items))
	for _, item := range c.Items {
		items = append(items, inboxsalvage.DetailItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Unit:     item.Unit,
			Remarks:  item.Remarks,
		})
	}

	return inboxsalvage.FormInput{
		Mode:          mode,
		SalvageID:     c.SalvageID,
		ClaimNo:       c.ClaimNo,
		ObjectID:      c.ObjectID,
		ObjectName:    c.ObjectName,
		CoverageID:    c.CoverageID,
		CoverageName:  c.CoverageName,
		InputDate:     c.InputDate,
		SalvageType:   c.SalvageType,
		Status:        c.Status,
		Location:      c.Location,
		InJabodetabek: c.InJabodetabek,
		Currency:      c.Currency,
		MinimumValue:  c.MinimumValue,
		Quantity:      c.Quantity,
		OfferValue:    c.OfferValue,
		InsuredShare:  c.InsuredShare,
		Remark:        c.Remark,
		Email:         c.Email,
		SurveyorName:  c.SurveyorName,
		SurveyorPhone: c.SurveyorPhone,
		SurveyorEmail: c.SurveyorEmail,
		Items:         items,
	}
}

// CreateResponse adalah jawaban penyimpanan pengajuan.
type CreateResponse struct {
	SalvageID string `json:"id_salvage"`
	ItemCount int    `json:"jumlah_detail_item"`

	// Message adalah kalimat yang ditampilkan ke pengguna.
	//
	// Ia menyebut apa yang TIDAK terjadi pula — pengajuan tidak dikirim ke balai lelang
	// dan tidak ada email yang terkirim. Tanpa itu, pengguna akan menunggu jawaban balai
	// lelang yang tidak pernah datang.
	Message string `json:"pesan"`

	Portal string `json:"portal"`
}

// UploadResponse adalah hasil pembacaan berkas "Upload Detail Salvage".
//
// Ia TIDAK menyimpan apa pun — lihat inboxsalvage.ParseUpload. Yang dikembalikan adalah
// baris yang siap dimasukkan ke grid form, dan pengguna masih harus menekan Submit.
type UploadResponse struct {
	Items []DetailItemRequest `json:"detail_item_salvage"`

	// Message menegaskan berkasnya BELUM tersimpan.
	Message string `json:"pesan"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
//
// # Kenapa namanya `field`, bukan `isian`
//
// Karena itulah nama yang BENAR-BENAR dibaca layar. `APIError.violations()` pada
// `frontend/src/api/client.ts` menyusun peta `kolom → pesan` dengan membaca `field` atau
// `kolom`, dan tidak mengenal nama lain sama sekali:
//
//	const column = item.field ?? item.kolom
//
// Nama yang di luar keduanya SAMPAI ke layar tetapi tidak pernah terbaca, sehingga
// pelanggarannya hilang tanpa satu pun galat — pengguna menerima pesan umum di atas form
// alih-alih tanda di isian yang salah. Pada form berisi tujuh belas isian, perbedaan itu
// menentukan apakah ia tahu isian mana yang harus diperbaiki.
//
// Ketiga modul yang sudah ada belum seragam soal ini — `masterstatus` dan `inboxrclpucl`
// memakai `field`, `inboxkomunikasicabang` memakai `isian` yang tidak terbaca layar. Yang
// dipakai di sini adalah yang terbukti bekerja. Penyeragamannya adalah `TKT-F1-004`, yang
// masih terhalang.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}
