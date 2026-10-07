// Package inboxpladlapredlahttp adalah lapisan transport modul Inbox PLA, DLA, Pre DLA.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: Row membawa ClaimKey, kunci objek
// kerja Pega yang `D-22` dan `D-71` justru hapus dari data bisnis. Ia memang dikirim, tetapi
// sebagai kunci teknis yang dinamai apa adanya, bukan sebagai "nama cabang" seperti
// aliasnya di Pega.
package inboxpladlapredlahttp

import (
	"time"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/inboxpladlapredla/usecase"
	"claim-pnc/internal/platform/apierror"
)

// RowDTO adalah satu baris antrean.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid, BUKAN
// alias Pega — tujuh dari delapan alias itu tidak menyatakan isinya.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang
// digambar dari `kolom` pada daftar yang sedang terbuka — bukan dari ada-tidaknya isian,
// karena isian yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya
// menghilang begitu saja.
//
// Di layar ini sifat itu bukan kehalusan: `tanggal_advice` memang KOSONG pada klaim yang
// seluruh dokumennya ber-`ISKIRIM = '0'`, dan kolomnya tetap harus digambar.
type RowDTO struct {
	// ClaimKey adalah `T_CLAIM_PNC.CLAIMID`, kunci objek kerja Pega.
	//
	// Ia dikirim tetapi TIDAK digambar sebagai kolom: tombol "Detail" memakainya, dan
	// grid rincian dicari dengannya — bukan dengan nomor klaim.
	ClaimKey string `json:"kunci_klaim"`

	ClaimNo  string `json:"no_klaim"`
	PolicyNo string `json:"no_polis"`
	Insured  string `json:"nama_tertanggung"`

	// Tanggal dikirim sebagai teks `YYYY-MM-DD`, bukan sebagai cap waktu.
	//
	// Tidak satu pun tanggal di layar ini dihitung — semuanya hanya digambar — dan
	// layar yang memformatnya ke bentuk Indonesia. Tanggal yang tidak ada dikirim
	// sebagai teks KOSONG, bukan `0001-01-01`.
	RegisterDate string `json:"tanggal_register"`
	LossDate     string `json:"tanggal_kejadian"`

	PICTeknik string `json:"pic_teknik"`

	// AdviceDate adalah tanggal dokumen terbaru yang belum terkirim.
	//
	// Ia KOSONG pada klaim yang seluruh dokumennya ber-`ISKIRIM = '0'` — kejanggalan
	// Pega yang sengaja dibawa (`P-5`). Lihat inboxpladlapredla.Row.AdviceDate.
	AdviceDate string `json:"tanggal_advice"`
}

func toRowDTO(row inboxpladlapredla.Row) RowDTO {
	return RowDTO{
		ClaimKey:     row.ClaimKey,
		ClaimNo:      row.ClaimNo,
		PolicyNo:     row.PolicyNo,
		Insured:      row.Insured,
		RegisterDate: row.RegisterDate,
		LossDate:     row.LossDate,
		PICTeknik:    row.PICTeknik,
		AdviceDate:   row.AdviceDate,
	}
}

// DocumentDTO adalah satu baris grid "Detail PLA List" / "Detail DLA List".
type DocumentDTO struct {
	AdviceNo   string `json:"no_advice"`
	Reinsurer  string `json:"reasuradur"`
	AdviceType string `json:"tipe"`

	// Revision hanya terisi pada PLA. `GetDLAList` tidak mengambilnya sama sekali.
	Revision string `json:"revisi"`

	AdviceDate string `json:"tanggal_dokumen"`

	// Sent adalah `ISKIRIM` apa adanya: kosong, `"0"`, atau `"1"`.
	//
	// Ia TIDAK diubah menjadi boolean. Kosong dan `"0"` sama-sama berarti belum terkirim
	// bagi penyaring daftar, tetapi tidak sama bagi kolom tanggal advice — dan
	// membedakannya adalah satu-satunya cara menjelaskan baris yang tanggalnya kosong.
	Sent string `json:"terkirim"`

	SentDate     string `json:"tanggal_kirim"`
	ReceivedDate string `json:"tanggal_terima"`
	Notes        string `json:"catatan"`
	Email        string `json:"email"`

	// AcceptanceNo hanya terisi pada DLA.
	AcceptanceNo string `json:"no_akseptasi"`
}

func toDocumentDTO(item inboxpladlapredla.Document) DocumentDTO {
	return DocumentDTO{
		AdviceNo:     item.AdviceNo,
		Reinsurer:    item.Reinsurer,
		AdviceType:   item.AdviceType,
		Revision:     item.Revision,
		AdviceDate:   item.AdviceDate,
		Sent:         item.Sent,
		SentDate:     item.SentDate,
		ReceivedDate: item.ReceivedDate,
		Notes:        item.Notes,
		Email:        item.Email,
		AcceptanceNo: item.AcceptanceNo,
	}
}

// PreDLADocumentDTO adalah satu baris panel "Print Pre DLA".
//
// Ia BUKAN DocumentDTO yang dipersempit. Sumbernya tabel lain (`T_PREDLALIST`), digabung
// ke kedua tabel lampiran, dan hanya enam kolom yang benar-benar dibaca. Memakai ulang
// DocumentDTO berarti mengirim lima medan yang selalu kosong — dan medan yang selalu
// kosong akan ditafsirkan layar sebagai data yang hilang, bukan sebagai data yang memang
// tidak ada di sumbernya.
type PreDLADocumentDTO struct {
	AdviceNo   string `json:"no_advice"`
	Reinsurer  string `json:"reasuradur"`
	AdviceType string `json:"tipe"`

	// SentDate adalah `TGLKIRIM`, beralias `"TglDLA"` di Pega.
	//
	// Aliasnya menyesatkan: ia tanggal KIRIM, bukan tanggal DLA. Judul kolomnya di Pega
	// sendiri sudah benar — "Tgl Kirim" — dan nama di sini mengikuti judulnya, bukan
	// aliasnya (`D-19`).
	SentDate string `json:"tanggal_kirim"`

	// Sent adalah `NVL(ISKIRIM, '0')`: `"0"` atau `"1"`, tidak pernah kosong.
	//
	// Berbeda dari DocumentDTO.Sent yang membawa kosong apa adanya. Di sini kuerinya
	// sendiri yang menggantinya, dan penggantian itu dibawa karena kolomnya DIGAMBAR —
	// sel kosong tidak terbaca sebagai "belum terkirim".
	Sent string `json:"terkirim"`

	// AttachmentKey adalah `PC_DATA_WORKATTACH.PZINSKEY`, kunci berkas lampirannya.
	//
	// Ia TIDAK digambar sebagai kolom. Ia dikirim karena tombol unduh di dalam panel
	// Pega (`PNCDownloadFile`) memakainya — dan tombol itu belum dibangun di sini,
	// karena penyimpanan dokumen (`D-16`) belum tersambung.
	//
	// Kuncinya SELALU terisi: baris yang lampirannya tidak ada tidak pernah sampai ke
	// panel ini sama sekali.
	AttachmentKey string `json:"kunci_lampiran"`
}

func toPreDLADocumentDTO(item inboxpladlapredla.PreDLADocument) PreDLADocumentDTO {
	return PreDLADocumentDTO{
		AdviceNo:      item.AdviceNo,
		Reinsurer:     item.Reinsurer,
		AdviceType:    item.AdviceType,
		SentDate:      item.SentDate,
		Sent:          item.Sent,
		AttachmentKey: item.AttachmentKey,
	}
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	Key   string `json:"kunci"`
	Title string `json:"judul"`

	// Date menandai kolom yang berisi tanggal, supaya layar memformatnya sebagai tanggal
	// WIB alih-alih menggambar teksnya apa adanya.
	Date bool `json:"tanggal"`
}

func toColumnDTOs(columns []inboxpladlapredla.Column) []ColumnDTO {
	out := make([]ColumnDTO, 0, len(columns))
	for _, column := range columns {
		out = append(out, ColumnDTO{
			Key:   column.Key,
			Title: column.Title,
			Date:  column.Date,
		})
	}
	return out
}

// TabDTO adalah satu daftar beserta kolom dan keterangannya.
type TabDTO struct {
	Code        string      `json:"kode"`
	Name        string      `json:"nama"`
	Description string      `json:"keterangan"`
	Columns     []ColumnDTO `json:"kolom"`

	// DocumentColumns adalah kolom grid rincian. Kosong berarti daftar ini tidak punya.
	DocumentColumns []ColumnDTO `json:"kolom_rincian"`

	// HasDocuments menyatakan daftar ini menggambar grid rincian.
	//
	// Ia dikirim TERPISAH dari panjang `kolom_rincian`, meski keduanya sejalan hari ini.
	// Layar yang menyimpulkannya dari panjang senarai akan menyembunyikan gridnya begitu
	// kolomnya kebetulan belum terisi — kegagalan yang terlihat seperti data kosong.
	HasDocuments bool `json:"punya_rincian"`

	// PrintColumns adalah kolom panel "Print Pre DLA". Kosong berarti daftar ini tidak
	// punya panelnya.
	PrintColumns []ColumnDTO `json:"kolom_cetak"`

	// HasPrintAction menyatakan setiap BARIS daftar ini punya tombol "Print Pre DLA".
	//
	// Ia dikirim, bukan disimpulkan layar dari kode daftarnya: inventaris tombol adalah
	// hasil pembacaan export, sama halnya dengan daftar kolom.
	HasPrintAction bool `json:"punya_cetak"`

	// RowActionLabel adalah judul tombol pada kolom aksi tiap baris.
	//
	// Ia BERBEDA per daftar — "Rincian" pada PLA dan DLA, "Print Pre DLA" pada Pre DLA —
	// karena yang dibuka pun berbeda: dua yang pertama membuka grid rincian di bawah
	// antrean, yang ketiga membuka panel tersendiri.
	RowActionLabel string `json:"label_aksi_baris"`

	// SearchLabel adalah judul kotak pencarian.
	SearchLabel string `json:"label_pencarian"`

	// DateLabel adalah judul rentang tanggal yang disaring daftar ini.
	//
	// Ia BERBEDA per daftar meski kotaknya sama-sama berjudul "Dari"/"Sampai" di Pega:
	// yang disaring adalah tanggal PLA, tanggal DLA, atau tanggal Pre-DLA. Menyebutnya
	// membuat pengguna tahu tanggal APA yang sedang ia batasi.
	DateLabel string `json:"label_tanggal"`
}

func toTabDTO(tab inboxpladlapredla.Tab) TabDTO {
	return TabDTO{
		Code:            tab.Code,
		Name:            tab.Name,
		Description:     tab.Description,
		Columns:         toColumnDTOs(tab.Columns),
		DocumentColumns: toColumnDTOs(tab.DocumentColumns),
		HasDocuments:    tab.HasDocuments(),
		PrintColumns:    toColumnDTOs(tab.PrintColumns),
		HasPrintAction:  tab.HasPrintAction,
		RowActionLabel:  tab.RowActionLabel,
		SearchLabel:     tab.SearchLabel,
		DateLabel:       tab.DateLabel,
	}
}

// MetadataResponse adalah keterangan layar.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"daftar"`
	DefaultTab string   `json:"daftar_bawaan"`

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

	return MetadataResponse{
		Tabs:               tabs,
		DefaultTab:         meta.DefaultTab,
		PlannedDifferences: meta.PlannedDifferences,
		Portal:             portalAlias,
	}
}

// PageDTO adalah keterangan paginasi.
type PageDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// FilterDTO adalah penyaring yang BENAR-BENAR dipakai.
//
// Ia dikembalikan karena tidak selalu sama dengan yang dikirim: daftar yang diminta kosong
// menjadi daftar bawaan, dan batas atas rentang digeser menjadi eksklusif di dalam. Layar
// menggambar isi kotaknya dari sini supaya yang terbaca pengguna adalah penyaring yang
// sungguh berlaku.
//
// Batas atas dikembalikan dalam bentuk YANG DIKETIK PENGGUNA — inklusif — bukan bentuk
// eksklusif yang dipakai kueri. Mengembalikan yang eksklusif akan membuat kotak "Sampai"
// bergeser satu hari setiap kali layar memuat ulang dirinya dari jawaban.
type FilterDTO struct {
	Search string `json:"cari"`
	From   string `json:"dari"`
	To     string `json:"sampai"`
}

// ListResponse adalah isi satu antrean.
type ListResponse struct {
	Tab    TabDTO    `json:"daftar"`
	Rows   []RowDTO  `json:"baris"`
	Paging PageDTO   `json:"paginasi"`
	Filter FilterDTO `json:"penyaring"`
	Portal string    `json:"portal"`
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
		Filter: FilterDTO{
			Search: listed.Query.Search,
			From:   dateText(listed.Query.From),
			To:     inclusiveDateText(listed.Query.To),
		},
		Portal: portalAlias,
	}
}

// DocumentsResponse adalah isi grid rincian satu klaim.
type DocumentsResponse struct {
	Tab      TabDTO        `json:"daftar"`
	ClaimKey string        `json:"kunci_klaim"`
	Rows     []DocumentDTO `json:"baris"`
	Portal   string        `json:"portal"`
}

func toDocumentsResponse(
	documented usecase.Documented,
	portalAlias string,
) DocumentsResponse {
	rows := make([]DocumentDTO, 0, len(documented.Items))
	for _, item := range documented.Items {
		rows = append(rows, toDocumentDTO(item))
	}

	return DocumentsResponse{
		Tab:      toTabDTO(documented.Tab),
		ClaimKey: documented.ClaimKey,
		Rows:     rows,
		Portal:   portalAlias,
	}
}

// PrintResponse adalah isi panel "Print Pre DLA" satu klaim.
type PrintResponse struct {
	Tab      TabDTO              `json:"daftar"`
	ClaimKey string              `json:"kunci_klaim"`
	Rows     []PreDLADocumentDTO `json:"baris"`
	Portal   string              `json:"portal"`
}

func toPrintResponse(printable usecase.Printable, portalAlias string) PrintResponse {
	rows := make([]PreDLADocumentDTO, 0, len(printable.Items))
	for _, item := range printable.Items {
		rows = append(rows, toPreDLADocumentDTO(item))
	}

	return PrintResponse{
		Tab:      toTabDTO(printable.Tab),
		ClaimKey: printable.ClaimKey,
		Rows:     rows,
		Portal:   portalAlias,
	}
}

// SendResponse adalah hasil pengiriman surat PLA/DLA.
//
// Jumlah penerima dan lampiran DIKIRIM, bukan hanya pesan berhasil. Petugas yang menekan
// tombol ini mengirim surat ke pihak luar, dan ia berhak tahu berapa alamat yang menerima
// dan berapa berkas yang ikut — terutama ketika lampirannya NOL, keadaan yang sah tetapi
// jarang diinginkan.
type SendResponse struct {
	Message     string `json:"pesan"`
	Recipients  int    `json:"penerima"`
	Attachments int    `json:"lampiran"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO = apierror.FieldError

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// dateText menuliskan satu batas rentang sebagai `YYYY-MM-DD`.
//
// Batas yang tidak dipasang menghasilkan teks KOSONG — itulah yang membuat kotak tanggal
// di layar tetap kosong, bukan terisi tanggal yang tidak pernah diketik siapa pun.
func dateText(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(inboxpladlapredla.DateLayout)
}

// inclusiveDateText mengembalikan batas atas ke bentuk YANG DIKETIK PENGGUNA.
//
// Query.To sudah digeser satu hari dan bersifat eksklusif — bentuk yang dibutuhkan kueri,
// bukan bentuk yang diketik pengguna. Mengirimkannya apa adanya akan membuat kotak
// "Sampai" bergeser satu hari setiap kali layar memuat ulang isinya dari jawaban, dan
// setelah tiga kali muat ulang penyaringnya sudah tiga hari lebih longgar daripada yang
// diminta — tanpa satu pun tanda.
func inclusiveDateText(value *time.Time) string {
	if value == nil {
		return ""
	}
	inclusive := value.AddDate(0, 0, -1)
	return inclusive.Format(inboxpladlapredla.DateLayout)
}
