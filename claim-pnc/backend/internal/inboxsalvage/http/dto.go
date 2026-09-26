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

	// DetailKey menyatakan dengan APA panel rincian dibuka pada daftar ini.
	//
	// SETIAP daftar punya tombol Detail — itu keadaan di Pega. Yang berbeda adalah
	// kuncinya: `"pengajuan"` pada ketujuh daftar berbaris pengajuan, `"klaim"` pada
	// keenam daftar berbaris klaim, yang barisnya tidak membawa ID pengajuan sama sekali.
	//
	// Layar memakainya untuk memilih rute mana yang ditembak dan isian mana yang dikirim
	// sebagai kunci. Menyimpulkannya di layar berarti pengetahuan yang sama hidup di dua
	// tempat, dan yang satu akan tertinggal.
	DetailKey string `json:"kunci_rincian"`
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
		DetailKey:   string(inboxsalvage.DetailKeyOf(tab.Family)),
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

// DetailBarangDTO adalah satu baris grid barang pada panel Detail Salvage.
type DetailBarangDTO struct {
	Name string `json:"nama_barang"`

	// Count dan Unit datang TERPISAH, meski layar lama merangkainya di dalam SQL
	// (`count(namabarang) || ' ' || satuan`). Dipisah supaya jumlahnya dapat diratakan
	// sebagai angka dan satuannya tetap terbaca sendiri.
	Count int    `json:"jumlah"`
	Unit  string `json:"satuan"`

	TotalValue    string `json:"total_nilai"`
	SoldStatus    string `json:"status_terjual"`
	WinnerName    string `json:"nama_pemenang"`
	AcceptanceNo  string `json:"no_akseptasi"`
	AcceptedValue string `json:"nilai_akseptasi"`
	Remark        string `json:"remark"`
}

// HistoryRowDTO adalah satu baris grid "Detail History Salvage".
//
// Judul kolomnya di layar lama: Tanggal Input · Nomor Klaim · PIC · Nilai Minimum ·
// Posisi Salvage.
type HistoryRowDTO struct {
	SalvageID string `json:"id_salvage"`
	InputDate string `json:"tanggal_input"`
	ClaimNo   string `json:"no_klaim"`
	PIC       string `json:"pic"`

	// MinimumValue berjudul "Nilai Minimum" di grid, sementara kolom yang sama berjudul
	// "Estimasi" pada panel rincian. Judulnya berbeda; kolomnya sama.
	MinimumValue string `json:"nilai_minimum"`

	// Position sudah berupa kalimat — "Sudah Aksep Checker", "Salvage Waive", dan
	// seterusnya. Pemetaannya BERBEDA dari "Posisi Salvage" pada panel rincian, meski
	// keduanya berasal dari kolom yang sama.
	Position string `json:"posisi_salvage"`
}

// DetailResponse adalah isi panel "Detail Salvage".
//
// Nama field mengikuti judul isian pada `Section/DataDetail_Salvage-Section.xml` (`D-13`).
type DetailResponse struct {
	SalvageID string `json:"id_salvage"`
	ClaimNo   string `json:"no_klaim"`

	// HasSubmission menyatakan klaim ini benar-benar punya pengajuan salvage.
	//
	// Selalu benar bila panel dibuka dari baris pengajuan. Dapat SALAH bila dibuka dari
	// baris klaim — dan pada daftar Salvage Outstanding ia justru yang lazim, sebab
	// daftar itu berisi klaim yang salvage-nya belum ditandai sama sekali.
	//
	// Layar memakainya untuk menyatakan keadaannya sebagai kalimat, alih-alih menggambar
	// panel yang seluruh isiannya kosong dan terbaca seperti gagal dimuat.
	HasSubmission bool `json:"ada_pengajuan"`

	// PIC dan LossDate berasal dari KLAIM, bukan dari pengajuan.
	PIC      string `json:"pic"`
	LossDate string `json:"tanggal_kejadian"`

	// BusinessName adalah lini bisnis klaimnya — satu-satunya isian panel ini yang tidak
	// berasal dari tabel salvage.
	BusinessName string `json:"nama_bisnis"`

	InputDate     string `json:"tanggal_input_salvage"`
	SalvageType   string `json:"jenis_salvage"`
	Quantity      string `json:"quantity_salvage"`
	EstimateValue string `json:"estimasi"`
	Location      string `json:"lokasi_salvage"`

	TransferGADate string `json:"tanggal_transfer_ga"`

	// TransferStatus adalah KODE, Position adalah labelnya. Keduanya dikirim: kodenya
	// untuk penelusuran ke Pega, labelnya untuk dibaca.
	TransferStatus string `json:"kode_posisi_salvage"`
	Position       string `json:"posisi_salvage"`

	AcceptanceDate string `json:"tanggal_akseptasi"`
	AcceptanceNo   string `json:"no_akseptasi"`
	Remark         string `json:"remark"`
	Currency       string `json:"mata_uang"`

	ObjectName   string `json:"nama_object"`
	CoverageName string `json:"nama_coverage"`

	// AcceptedValue adalah "Nilai Salvage". Kolom yang sama menentukan "Status Lelang"
	// pada grid; di panel ini ia digambar sebagai nilai.
	AcceptedValue string `json:"nilai_salvage"`

	Email      string `json:"email"`
	OfferValue string `json:"nilai_penawaran"`
	WinnerName string `json:"nama_pemenang"`

	AuctionDate string `json:"tanggal_lelang"`

	SurveyorName  string `json:"nama_pic_survey"`
	SurveyorPhone string `json:"no_telp_pic_survey"`
	SurveyorEmail string `json:"email_pic_survey"`

	InJabodetabek bool `json:"lokasi_salvage_di_jabodetabek"`

	// LegacyBeforeJuly2023 menandai pengajuan yang dibuat sebelum 17 Juli 2023.
	//
	// ARTINYA tidak diketahui — tidak ada satu pun rule di export yang memakainya selain
	// menggambarnya. Ia dikirim apa adanya, dan layar menyebutnya sebagai penanda tanpa
	// menafsirkannya.
	LegacyBeforeJuly2023 bool `json:"pengajuan_sebelum_juli_2023"`

	Items []DetailBarangDTO `json:"barang"`

	// History adalah SELURUH pengajuan salvage milik klaim ini, terbaru lebih dulu.
	//
	// Digambar pada form "Menambahkan Data Salvage" sebagai grid "Detail History
	// Salvage". Kosong berarti klaim ini belum pernah diajukan salvage sama sekali.
	History []HistoryRowDTO `json:"riwayat"`

	Portal string `json:"portal"`
}

func toDetailResponse(detail inboxsalvage.Detail, portalAlias string) DetailResponse {
	items := make([]DetailBarangDTO, 0, len(detail.Items))
	for _, item := range detail.Items {
		items = append(items, DetailBarangDTO{
			Name:          item.Name,
			Count:         item.Count,
			Unit:          item.Unit,
			TotalValue:    item.TotalValue,
			SoldStatus:    item.SoldStatus,
			WinnerName:    item.WinnerName,
			AcceptanceNo:  item.AcceptanceNo,
			AcceptedValue: item.AcceptedValue,
			Remark:        item.Remark,
		})
	}

	history := make([]HistoryRowDTO, 0, len(detail.History))
	for _, row := range detail.History {
		history = append(history, HistoryRowDTO{
			SalvageID:    row.SalvageID,
			InputDate:    row.InputDate,
			ClaimNo:      row.ClaimNo,
			PIC:          row.PIC,
			MinimumValue: row.MinimumValue,
			Position:     row.Position,
		})
	}

	return DetailResponse{
		SalvageID:            detail.SalvageID,
		ClaimNo:              detail.ClaimNo,
		HasSubmission:        detail.HasSubmission,
		PIC:                  detail.PIC,
		LossDate:             detail.LossDate,
		BusinessName:         detail.BusinessName,
		InputDate:            detail.InputDate,
		SalvageType:          detail.SalvageType,
		Quantity:             detail.Quantity,
		EstimateValue:        detail.EstimateValue,
		Location:             detail.Location,
		TransferGADate:       detail.TransferGADate,
		TransferStatus:       detail.TransferStatus,
		Position:             detail.Position,
		AcceptanceDate:       detail.AcceptanceDate,
		AcceptanceNo:         detail.AcceptanceNo,
		Remark:               detail.Remark,
		Currency:             detail.Currency,
		ObjectName:           detail.ObjectName,
		CoverageName:         detail.CoverageName,
		AcceptedValue:        detail.AcceptedValue,
		Email:                detail.Email,
		OfferValue:           detail.OfferValue,
		WinnerName:           detail.WinnerName,
		AuctionDate:          detail.AuctionDate,
		SurveyorName:         detail.SurveyorName,
		SurveyorPhone:        detail.SurveyorPhone,
		SurveyorEmail:        detail.SurveyorEmail,
		InJabodetabek:        detail.InJabodetabek,
		LegacyBeforeJuly2023: detail.LegacyBeforeJuly2023,
		Items:                items,
		History:              history,
		Portal:               portalAlias,
	}
}
