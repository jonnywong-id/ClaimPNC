// Package inboxbandinghargasalvagehttp adalah lapisan transport modul Inbox Banding Harga
// Salvage.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe domain
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya — dan
// di modul ini bocornya akan nyata: AppealRow membawa isian dari DUA tabel sekaligus, dan yang
// terisi bergantung pada tab mana yang sedang dibuka.
package inboxbandinghargasalvagehttp

import (
	"strconv"
	"time"

	"claim-pnc/internal/inboxbandinghargasalvage"
	"claim-pnc/internal/inboxbandinghargasalvage/usecase"
	"claim-pnc/internal/platform/apierror"
)

// dateLayout adalah bentuk tanggal pada kontrak API: YYYY-MM-DD.
//
// Jam sengaja tidak dikirim. Kolom "Tanggal Request" pada grid Pega tidak menampilkan jam,
// dan mengirimkannya akan membuat layar harus memutuskan zona waktu mana yang dipakai
// menampilkannya — keputusan yang seharusnya hanya ada di satu tempat (`F-5`).
const dateLayout = "2006-01-02"

// AppealRowDTO adalah satu baris pada layar.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid, bukan nama
// properti Pega: `harga_request`, bukan `email`.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang digambar
// dari `kolom` pada tab yang sedang terbuka — dan karena kedua tab membaca tabel yang berbeda,
// isian di luar tabnya memang kosong.
type AppealRowDTO struct {
	// Digambar KEDUA tab.
	ClaimNo string `json:"no_klaim"`

	// Digambar tab "Request Banding Harga".
	RequestDate  *string `json:"tanggal_request"`
	DetailObject string  `json:"detail_object"`
	ItemName     string  `json:"nama_barang"`
	ItemPrice    string  `json:"harga_barang"`
	RequestPrice string  `json:"harga_request"`
	RequestNote  string  `json:"note_request"`
	CheckerNote  string  `json:"note_checker"`

	// Aging adalah umur banding dalam bentuk yang dibaca pengguna: `<n> days`.
	//
	// Teksnya mengikuti layar lama, yang menyusunnya sebagai `… || ' days'`. Yang TIDAK
	// diikuti adalah pengurutannya: di sini ia diurutkan sebagai angka di basis data
	// (selisih terencana nomor 1).
	//
	// Kosong bila tanggal request-nya kosong. Layar lama menggambar " days" tanpa angka di
	// keadaan itu — akibat `NULL || ' days'` pada Oracle — dan sel kosong lebih jujur
	// daripada satuan tanpa bilangan.
	Aging string `json:"aging"`

	// Digambar tab "History Cheker".
	//
	// SalvageType digambar dengan judul "Object Name". Judulnya mengikuti layar lama
	// (`D-13`); nama isiannya menyebut apa yang benar-benar ada di dalamnya (`D-19`).
	SalvageType     string `json:"object_name"`
	SalvageLocation string `json:"lokasi_salvage"`
	PIC             string `json:"pic"`

	// Keduanya TIDAK digambar sebagai kolom.
	//
	// Ia dikirim karena tombol Approve/Reject kelak membutuhkannya — pasangan (no klaim,
	// id salvage) itulah yang menjadi kunci keputusan banding — dan supaya pengguna yang
	// melihat antrean komite lain dapat memastikan barisnya memang milik siapa.
	SalvageID     string `json:"id_salvage"`
	CommitteeName string `json:"nama_komite"`
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	// Key menyebut isian mana pada baris yang digambar.
	Key string `json:"kunci"`

	// Title adalah judul kolom yang dibaca pengguna.
	Title string `json:"judul"`

	// Numeric menandai kolom angka, supaya layar meratakannya ke kanan dan memformatnya
	// dengan pemisah ribuan.
	Numeric bool `json:"angka"`
}

// TabDTO adalah satu antrean kerja beserta bentuk gridnya.
type TabDTO struct {
	Code        string `json:"kode"`
	Name        string `json:"nama"`
	Description string `json:"keterangan"`

	// PegaParam adalah nilai `tipe` sistem lama untuk tab ini.
	//
	// Ia ikut dikirim demi ketelusuran, bukan untuk digambar: saat uji kesetaraan gerbang 1
	// menemukan selisih, inilah yang menghubungkan tab di layar dengan langkah activity di
	// export.
	PegaParam string `json:"parameter_pega"`

	Columns []ColumnDTO `json:"kolom"`
}

// MetadataResponse adalah jawaban GET /api/inbox-banding-harga-salvage/tab.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"tab"`
	DefaultTab string   `json:"tab_bawaan"`

	SearchLabel       string `json:"label_cari"`
	SearchPlaceholder string `json:"petunjuk_cari"`

	// PlannedDifferences menyatakan hal yang SENGAJA berbeda dari layar lama.
	//
	// Ia dipisah dari Limitations karena keduanya menjawab pertanyaan berbeda: yang ini
	// "mengapa angkanya tidak sama dengan Pega", yang itu "mengapa tombolnya tidak ada".
	// DecisionColumns adalah kolom panel rincian pada grid History Cheker.
	DecisionColumns []ColumnDTO `json:"kolom_rincian"`

	PlannedDifferences []string `json:"selisih_terencana"`

	// Limitations menyatakan hal yang BELUM berjalan penuh beserta alasannya, dalam kalimat
	// yang dapat langsung ditampilkan ke pengguna.
	Limitations []string `json:"keterbatasan"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal yang
	// sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`
}

// PaginationDTO adalah keterangan halaman.
type PaginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// AppliedFilterDTO adalah penyaring yang BENAR-BENAR dipakai.
//
// Ia dikirim balik supaya kotak cari di layar selalu memperlihatkan kata kunci yang
// benar-benar menyaring — bukan yang sempat diketik lalu tidak jadi terkirim.
type AppliedFilterDTO struct {
	Keyword string `json:"cari"`
}

// QueueDTO menyatakan antrean SIAPA yang sedang dibaca.
//
// # Kenapa ini dikirim ke layar, bukan disimpan diam-diam di server
//
// Karena di layar ini ia dapat BERBEDA dari pemanggilnya, dan perbedaan itu tidak terlihat di
// mana pun kecuali diberitahukan. Seorang petugas dapat membuka layar ini lalu melihat antrean
// milik komite lain — aturan bernama orang yang ditiru dari Pega atas keputusan Work Owner
// (lihat inboxbandinghargasalvage/komite.go).
//
// Tanpa keterangan ini, ia akan menyimpulkan antreannya sendiri kosong.
type QueueDTO struct {
	// Owner adalah nama komite yang antreannya sedang dibaca.
	Owner string `json:"milik"`

	// Delegated menyatakan antrean itu BUKAN milik pemanggil sendiri.
	Delegated bool `json:"diwakilkan"`

	// DelegationNotice dan QueueNotice adalah kalimat siap tampil, kosong bila tidak
	// berlaku. Ia disusun server supaya layar tidak perlu memuat aturannya sendiri.
	DelegationNotice string `json:"catatan_perwakilan"`
	QueueNotice      string `json:"catatan_giliran"`
}

// ListResponse adalah jawaban GET /api/inbox-banding-harga-salvage.
type ListResponse struct {
	Tab        TabDTO           `json:"tab"`
	Items      []AppealRowDTO   `json:"baris"`
	Pagination PaginationDTO    `json:"paginasi"`
	Filter     AppliedFilterDTO `json:"penyaring"`
	Queue      QueueDTO         `json:"antrean"`
	Portal     string           `json:"portal"`
}

// SummaryRowDTO adalah satu baris tabel ringkas "Status Salvage / Jumlah".
type SummaryRowDTO struct {
	Label string `json:"status_salvage"`
	Tab   string `json:"tab"`
	Count int    `json:"jumlah"`
}

// SummaryResponse adalah jawaban GET /api/inbox-banding-harga-salvage/ringkas.
type SummaryResponse struct {
	Rows   []SummaryRowDTO `json:"baris"`
	Queue  QueueDTO        `json:"antrean"`
	Portal string          `json:"portal"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO = apierror.FieldError

// ErrorResponse adalah bentuk galat modul ini.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toAppealRowDTO mengubah satu baris.
func toAppealRowDTO(row inboxbandinghargasalvage.AppealRow) AppealRowDTO {
	return AppealRowDTO{
		ClaimNo: row.ClaimNo,

		RequestDate:  toDateString(row.RequestDate),
		DetailObject: row.DetailObject,
		ItemName:     row.ItemName,
		ItemPrice:    row.ItemPrice,
		RequestPrice: row.RequestPrice,
		RequestNote:  row.RequestNote,
		CheckerNote:  row.CheckerNote,
		Aging:        agingText(row),

		SalvageType:     row.SalvageType,
		SalvageLocation: row.SalvageLocation,
		PIC:             row.PIC,

		SalvageID:     row.SalvageID,
		CommitteeName: row.CommitteeName,
	}
}

// agingText menyusun teks kolom Aging: `<n> days`.
//
// Kosong bila barisnya tidak punya tanggal request — lihat AppealRowDTO.Aging.
func agingText(row inboxbandinghargasalvage.AppealRow) string {
	if row.RequestDate == nil {
		return ""
	}
	return strconv.Itoa(row.AgingDays) + " days"
}

// toAppealRowListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toAppealRowListDTO(rows []inboxbandinghargasalvage.AppealRow) []AppealRowDTO {
	result := make([]AppealRowDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, toAppealRowDTO(row))
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxbandinghargasalvage.Tab) TabDTO {
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
		PegaParam:   tab.PegaParam,
		Columns:     columns,
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

	limitations := make([]string, 0, len(meta.Limitations))
	limitations = append(limitations, meta.Limitations...)

	decisionColumns := make([]ColumnDTO, 0, len(meta.DecisionColumns))
	for _, column := range meta.DecisionColumns {
		decisionColumns = append(decisionColumns, ColumnDTO{
			Key:     column.Key,
			Title:   column.Title,
			Numeric: column.Numeric,
		})
	}

	return MetadataResponse{
		Tabs:               tabs,
		DefaultTab:         meta.DefaultTab,
		DecisionColumns:    decisionColumns,
		SearchLabel:        meta.SearchLabel,
		SearchPlaceholder:  meta.SearchPlaceholder,
		PlannedDifferences: differences,
		Limitations:        limitations,
		Portal:             portalAlias,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxbandinghargasalvage.Page) PaginationDTO {
	return PaginationDTO{
		Page:       page.Pagination.Page,
		Size:       page.Pagination.Size,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
	}
}

// toQueueDTO menyusun keterangan antrean siapa yang sedang dibaca.
func toQueueDTO(reviewer inboxbandinghargasalvage.Reviewer) QueueDTO {
	return QueueDTO{
		Owner:            reviewer.Name,
		Delegated:        reviewer.Delegated,
		DelegationNotice: reviewer.DelegationNotice(),
		QueueNotice:      reviewer.QueueNotice(),
	}
}

// toListResponse merakit jawaban isi satu tab.
func toListResponse(listed usecase.Listed, portalAlias string) ListResponse {
	return ListResponse{
		Tab:        toTabDTO(listed.Query.Tab),
		Items:      toAppealRowListDTO(listed.Page.Items),
		Pagination: toPaginationDTO(listed.Page),
		Filter:     AppliedFilterDTO{Keyword: listed.Query.Keyword},
		Queue:      toQueueDTO(listed.Query.Reviewer),
		Portal:     portalAlias,
	}
}

// toSummaryResponse merakit jawaban tabel ringkas.
func toSummaryResponse(
	summary inboxbandinghargasalvage.Summary,
	reviewer inboxbandinghargasalvage.Reviewer,
	portalAlias string,
) SummaryResponse {
	rows := make([]SummaryRowDTO, 0, len(summary.Rows))
	for _, row := range summary.Rows {
		rows = append(rows, SummaryRowDTO{
			Label: row.Label,
			Tab:   row.Tab,
			Count: row.Count,
		})
	}

	return SummaryResponse{
		Rows:   rows,
		Queue:  toQueueDTO(reviewer),
		Portal: portalAlias,
	}
}

// toDateString memformat tanggal, atau nil bila kosong.
//
// Pointer, bukan teks kosong: `null` menyatakan "tidak ada tanggalnya", sedangkan `""` akan
// terbaca layar sebagai tanggal yang gagal diformat.
func toDateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(dateLayout)
	return &formatted
}

// DecisionDTO adalah satu keputusan banding harga pada panel rincian History Cheker.
type DecisionDTO struct {
	ApprovedAt   *string `json:"tanggal_approve"`
	DetailObject string  `json:"detail_object"`
	ItemName     string  `json:"nama_barang"`
	ItemPrice    string  `json:"harga_barang"`
	RequestPrice string  `json:"harga_request_keputusan"`

	// Keputusan dikirim sebagai KODE dan LABEL sekaligus.
	//
	// Kode supaya layar dapat membandingkannya tanpa mencocokkan teks yang dapat berubah —
	// dan di layar ini itu berguna, karena "Setuju" dan "Tidak setuju" layak digambar
	// dengan warna yang berbeda. Label supaya layar tidak perlu memuat tabel terjemahannya
	// sendiri; tabel itu dibaca dari rule Pega dan tempatnya di backend.
	Decision      string `json:"jawaban_checker_kode"`
	DecisionLabel string `json:"jawaban_checker"`

	CheckerName string `json:"nama_checker"`
}

// DecisionsResponse adalah jawaban GET /api/inbox-banding-harga-salvage/riwayat/{noKlaim}.
type DecisionsResponse struct {
	ClaimNo string        `json:"no_klaim"`
	Items   []DecisionDTO `json:"baris"`
	Queue   QueueDTO      `json:"antrean"`
	Portal  string        `json:"portal"`
}

// toDecisionDTO mengubah satu keputusan.
func toDecisionDTO(decision inboxbandinghargasalvage.Decision) DecisionDTO {
	return DecisionDTO{
		ApprovedAt:   toDateString(decision.ApprovedAt),
		DetailObject: decision.DetailObject,
		ItemName:     decision.ItemName,
		ItemPrice:    decision.ItemPrice,
		RequestPrice: decision.RequestPrice,

		Decision:      decision.Status,
		DecisionLabel: inboxbandinghargasalvage.DecisionLabel(decision.Status),

		CheckerName: decision.CommitteeName,
	}
}

// toDecisionsResponse merakit jawaban panel rincian.
//
// Senarai kosong, bukan nil: klaim tanpa keputusan adalah keadaan yang sah, dan `null`
// memaksa layar memeriksanya lebih dulu.
func toDecisionsResponse(decided usecase.Decided, portalAlias string) DecisionsResponse {
	items := make([]DecisionDTO, 0, len(decided.Items))
	for _, decision := range decided.Items {
		items = append(items, toDecisionDTO(decision))
	}

	return DecisionsResponse{
		ClaimNo: decided.Query.ClaimNo,
		Items:   items,
		Queue:   toQueueDTO(decided.Query.Reviewer),
		Portal:  portalAlias,
	}
}

// DecisionRequest adalah badan POST /api/inbox-banding-harga-salvage/keputusan.
type DecisionRequest struct {
	DetailObject string `json:"detail_object"`
	SalvageID    string `json:"id_salvage"`
	RequestPrice string `json:"harga_request"`
	Note         string `json:"catatan"`

	// Approve membedakan kedua tombol. `true` menerima harga tandingan balai lelang.
	Approve bool `json:"setujui"`
}

// DecisionResultResponse adalah jawaban keputusan yang tersimpan.
//
// Ia menyatakan langkah mana yang BENAR-BENAR berjalan, bukan sekadar "berhasil". Di layar
// ini "disetujui" tidak selalu berarti "harganya berubah" — penerapan harga di Pega hanya
// dilakukan jenjang terakhir — dan pesan yang menyiratkan lebih akan membuat pengguna
// mengira sesuatu sudah terjadi padahal belum.
type DecisionResultResponse struct {
	Recorded       bool `json:"tersimpan"`
	PriceApplied   bool `json:"harga_diterapkan"`
	DocumentMarked bool `json:"dokumen_ditandai"`

	// Message adalah kalimat siap tampil yang menjelaskan ketiganya.
	Message string `json:"pesan"`

	Portal string `json:"portal"`
}

// toDecisionResultResponse merakit jawaban keputusan.
func toDecisionResultResponse(
	result inboxbandinghargasalvage.DecisionResult,
	approved bool,
	portalAlias string,
) DecisionResultResponse {
	return DecisionResultResponse{
		Recorded:       result.Recorded,
		PriceApplied:   result.PriceApplied,
		DocumentMarked: result.DocumentMarked,
		Message:        decisionMessage(result, approved),
		Portal:         portalAlias,
	}
}

// decisionMessage menyusun kalimat yang menjelaskan apa yang benar-benar terjadi.
//
// Ia disusun server, bukan layar, supaya penjelasannya tidak berselisih dengan langkah yang
// benar-benar dijalankan — keduanya lahir dari nilai yang sama.
func decisionMessage(result inboxbandinghargasalvage.DecisionResult, approved bool) string {
	putusan := "Penolakan"
	if approved {
		putusan = "Persetujuan"
	}

	pesan := putusan + " tersimpan dan banding ini berpindah ke History Cheker."

	if approved && !result.PriceApplied {
		pesan += " Harga barang BELUM diubah: di layar lama, penerapan harga hanya " +
			"dilakukan jenjang komite terakhir."
	}
	if approved && result.PriceApplied {
		pesan += " Harga barang diperbarui mengikuti harga request balai lelang."
	}

	pesan += " Balai lelang TIDAK diberi tahu dari aplikasi ini — pengirimannya belum " +
		"dibangun."

	return pesan
}

// ─────────────────────────────────────────────────────────────────────────────
// Dialog "Lihat File"
// ─────────────────────────────────────────────────────────────────────────────

// DocumentDTO adalah satu baris pada dialog dokumen banding.
type DocumentDTO struct {
	ID string `json:"id"`

	// Kategori SELALU bernilai sama — ia konstanta, bukan kolom basis data.
	//
	// Ia tetap dikirim, alih-alih dibiarkan layar menuliskannya sendiri, supaya nilainya
	// hidup di satu tempat. Lihat inboxbandinghargasalvage.DocumentCategory.
	Kategori string `json:"kategori"`

	Nama string `json:"nama"`

	// TanggalUnggah berasal dari `SALAVAGEDOCUMENT.TGLINS`, bukan dari tabel lampiran.
	// Null berarti kolomnya kosong.
	TanggalUnggah *time.Time `json:"tanggal_unggah"`
}

// DocumentsResponse adalah jawaban `GET /api/inbox-banding-harga-salvage/dokumen`.
type DocumentsResponse struct {
	// Baris KOSONG berarti banding itu tidak punya dokumen pendukung — keadaan yang sah,
	// bukan galat.
	Baris []DocumentDTO `json:"baris"`

	Portal string `json:"portal"`
}

// toDocumentsResponse merakit jawaban daftar dokumen.
func toDocumentsResponse(
	documents []inboxbandinghargasalvage.DocumentRow,
	portalAlias string,
) DocumentsResponse {
	baris := make([]DocumentDTO, 0, len(documents))
	for _, document := range documents {
		baris = append(baris, DocumentDTO{
			ID:            document.ID,
			Kategori:      document.Category(),
			Nama:          document.Name,
			TanggalUnggah: document.UploadedAt,
		})
	}

	return DocumentsResponse{Baris: baris, Portal: portalAlias}
}
