// Package inboxkomunikasicabanghttp adalah lapisan transport modul Inbox Komunikasi Cabang.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: Conversation membawa RepliedAt dan
// Status, dua isian yang tidak digambar kolom mana pun tetapi ikut ke berkas ekspor.
package inboxkomunikasicabanghttp

import (
	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/inboxkomunikasicabang/usecase"
)

// ConversationDTO adalah satu baris pada grid.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang digambar
// dari `kolom` pada tab yang sedang terbuka — bukan dari ada-tidaknya isian, karena isian
// yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya menghilang
// begitu saja.
//
// Di layar ini sifat itu bukan kehalusan: `jawaban_terakhir` dan `penjawab` SELALU kosong
// pada tab "Belum Dijawab" — penyaringnya `IS NULL` — dan justru karena itu kedua kolomnya
// memang tidak didaftarkan pada tab tersebut.
type ConversationDTO struct {
	// ID adalah nomor percakapan, dibutuhkan tombol "Detail Komunikasi".
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom. Nama fieldnya `komunikasi`, bukan
	// `no_klaim`: tabel ini tidak memuat nomor klaim sama sekali, dan alias Pega yang
	// menyebutnya "ClaimNo" menyesatkan (`D-19`).
	ID string `json:"komunikasi"`

	// CreatedAt adalah kolom "Tanggal" — tanggal pesannya dikirim.
	CreatedAt string `json:"tanggal"`

	// Sender adalah kolom "Pengirim(Dari)", SUDAH dirakit.
	//
	// # Kenapa dirakit di server, bukan di layar
	//
	// Karena bentuknya ditetapkan `Section/PengirimKomunikasi-Section.xml`, dan tempat
	// pembacaan section itu tercatat adalah backend. Merakitnya di layar berarti pola
	// `asal (operator)` hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
	// diperbaiki.
	//
	// Kedua bahannya tetap dikirim terpisah di bawah, supaya layar dapat menyorot salah
	// satunya bila kelak dibutuhkan tanpa harus mengurai teks yang sudah jadi.
	Sender string `json:"pengirim"`

	// SenderOrigin adalah ASAL pesan — "PUSAT" atau kode cabangnya.
	SenderOrigin string `json:"asal"`

	// SenderOperator adalah Operator ID pengirimnya.
	SenderOperator string `json:"operator_pengirim"`

	// Message adalah kolom "Pesan".
	Message string `json:"pesan"`

	// Reply adalah kolom "Jawaban Terakhir".
	//
	// SELALU kosong pada tab "Belum Dijawab", dan itu bukan data hilang — justru itulah arti
	// tab tersebut.
	Reply string `json:"jawaban_terakhir"`

	// Replier adalah kolom "Penjawab(Dari)", SUDAH dirakit.
	//
	// Bentuknya `nama (tujuan)` mengikuti `Section/PenjawabKomunikasi-Section.xml`.
	// Perhatikan susunannya TERBALIK dari kolom pengirim — yang satu menaruh asal di depan,
	// yang satu menaruh nama di depan. Itu bentuk kedua section-nya apa adanya (`D-13`).
	Replier string `json:"penjawab"`

	// Recipient adalah TUJUAN pesan — "PUSAT" atau "CABANG".
	//
	// Perhatikan ia tidak pernah menyebut cabang MANA, berbeda dari `asal` yang menyebut
	// kodenya. Asimetri itu ada di layar lama dan direplikasi (`P-5`).
	Recipient string `json:"tujuan"`

	// Status adalah `KOMUNIKASISTATUS` MENTAH.
	//
	// Artinya tidak diketahui — tidak ada master yang menerjemahkannya di export mana pun —
	// sehingga ia tidak digambar sebagai kolom di tabel. Ia ikut ke berkas ekspor, tempat
	// nilai mentah masih berguna bagi yang menelusuri.
	Status string `json:"status_register"`

	// RepliedAt adalah tanggal balasan.
	//
	// Ia DASAR PENGURUTAN tab "Sudah Dijawab" tetapi tidak digambar sebagai kolom. Dikirim
	// supaya urutan tabel dapat dijelaskan bila dipertanyakan, dan supaya berkas ekspor
	// dapat memuatnya.
	RepliedAt string `json:"tanggal_jawaban"`
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

	// Notice adalah keterangan yang berlaku pada tab ini saja, digambar di atas grid.
	// Kosong berarti tidak ada.
	Notice string `json:"catatan,omitempty"`
}

// SummaryDTO adalah kedua pencacah di atas grid.
//
// Ia menggantikan diagram lingkaran layar lama (`.pyTemplateChart` pada section), yang
// memasok kedua angka yang sama dengan label "Answered" dan "Not Answered".
type SummaryDTO struct {
	NotAnswered int `json:"belum_dijawab"`
	Answered    int `json:"sudah_dijawab"`

	// Total adalah jumlah keduanya.
	//
	// Ia dikirim, bukan dihitung layar, karena keduanya BUKAN saling melengkapi: percakapan
	// yang salah satu kolom balasannya terisi sendirian tidak terhitung di mana pun.
	// Menghitungnya di layar akan membuat orang menduga angka itu jumlah seluruh percakapan
	// — dan ia bukan.
	Total int `json:"total"`
}

// BranchDTO menyatakan batas data yang sedang berlaku.
//
// # Kenapa ia dikirim ke layar sama sekali
//
// Karena petugas yang tidak tahu daftarnya sedang disaring akan menyimpulkan tidak ada
// percakapan, padahal yang benar adalah tidak ada percakapan DI CABANGNYA. Itu pelajaran
// yang sudah tercatat pada modul Inbox Laporan Klaim (`keputusan-implementasi.md` §20.4).
type BranchDTO struct {
	// Code adalah kode yang dipakai menyaring.
	Code string `json:"kode"`

	// HeadOffice menyatakan penyaringnya jatuh ke jalur kantor pusat.
	HeadOffice bool `json:"kantor_pusat"`

	// Resolved menyatakan kode cabang pemanggil benar-benar terbaca.
	//
	// `kantor_pusat` true dengan `terbaca` false berarti pemanggilnya TIDAK terdaftar di
	// HRD dan sedang dilayani sebagai kantor pusat karena itu (`P-5`). Layar menyatakan
	// keadaan itu apa adanya; menyembunyikannya berarti pelebaran batas data yang tidak
	// diketahui siapa pun yang terkena.
	Resolved bool `json:"terbaca"`

	// Notice adalah kalimat siap baca yang menjelaskan batasnya kepada pengguna.
	//
	// Ia datang dari server, bukan diketik di layar, supaya penjelasannya berubah di satu
	// tempat saat keputusannya kelak ditinjau ulang.
	Notice string `json:"keterangan"`
}

// MetadataResponse adalah jawaban GET /api/inbox-komunikasi-cabang/tab.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"tab"`
	DefaultTab string   `json:"tab_bawaan"`

	// ExportColumns adalah kolom berkas ekspor.
	ExportColumns []ColumnDTO `json:"kolom_ekspor"`

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string `json:"selisih_terencana"`

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

// ListResponse adalah jawaban GET /api/inbox-komunikasi-cabang.
type ListResponse struct {
	Tab        TabDTO            `json:"tab"`
	Items      []ConversationDTO `json:"baris"`
	Pagination PaginationDTO     `json:"paginasi"`
	Summary    SummaryDTO        `json:"ringkasan"`
	Branch     BranchDTO         `json:"batas_cabang"`
	Portal     string            `json:"portal"`
}

// AttachmentDTO adalah satu lampiran pada layar detail.
type AttachmentDTO struct {
	// DocumentID adalah rujukan ke penyimpanan dokumen.
	//
	// Ia dikirim tetapi belum dapat dipakai mengunduh apa pun: pengambilan berkasnya
	// menempuh seam DocumentStore (`D-16`) yang belum dibangun di modul ini.
	DocumentID string `json:"id_dokumen"`

	TypeName   string `json:"jenis_dokumen"`
	DetailName string `json:"rincian_dokumen"`
	Note       string `json:"catatan"`
	UploadedAt string `json:"tanggal_unggah"`

	// Uploaded menyatakan lampirannya sudah benar-benar diunggah.
	//
	// Ia dikirim sebagai penanda, bukan disimpulkan layar dari `tanggal_unggah` yang kosong:
	// kesimpulan yang sama di dua tempat dapat menyimpang, dan yang di server-lah yang
	// mengikuti aturan sistem lama.
	Uploaded bool `json:"sudah_diunggah"`
}

// ThreadMessageDTO adalah satu baris pada utas layar detail.
type ThreadMessageDTO struct {
	CreatedAt string `json:"tanggal"`

	// Sender adalah kolom "Pengirim", sudah dirakit — sama bentuknya dengan di grid.
	Sender         string `json:"pengirim"`
	SenderOrigin   string `json:"asal"`
	SenderOperator string `json:"operator_pengirim"`

	Message string `json:"pesan"`

	// Reply digambar sebagai baris tersendiri di bawah pesannya, bukan sebagai kolom.
	//
	// Section lama tidak menggambarnya sama sekali; ia ditambahkan karena satu baris tabel
	// menyimpan pesan DAN balasannya, sehingga utas yang membuangnya terbaca separuh. Lihat
	// PlannedDifferences.
	Reply       string `json:"jawaban"`
	ReplierName string `json:"penjawab"`
	RepliedAt   string `json:"tanggal_jawaban"`
}

// ConversationDetailResponse adalah jawaban
// GET /api/inbox-komunikasi-cabang/komunikasi/{komunikasi}.
type ConversationDetailResponse struct {
	ID string `json:"komunikasi"`

	// Origin adalah asal percakapan, dipakai judul layar detail.
	Origin string `json:"asal"`

	Messages    []ThreadMessageDTO `json:"pesan"`
	Attachments []AttachmentDTO    `json:"lampiran"`

	// WriteStillInPega menyatakan kotak balasan digambar tetapi belum dapat dipakai.
	//
	// Ia dikirim sebagai DATA, bukan ditulis tetap di layar, supaya penghidupan tombolnya
	// kelak tidak menuntut suntingan frontend.
	WriteStillInPega bool `json:"tindakan_masih_di_pega"`

	Branch BranchDTO `json:"batas_cabang"`
	Portal string    `json:"portal"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO struct {
	Field   string `json:"isian"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk baku jawaban galat.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toMetadataResponse menyusun jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		tabs = append(tabs, toTabDTO(tab))
	}

	return MetadataResponse{
		Tabs:               tabs,
		DefaultTab:         meta.DefaultTab,
		ExportColumns:      toColumnDTOs(meta.ExportColumns),
		PlannedDifferences: meta.PlannedDifferences,
		Portal:             portalAlias,
	}
}

// toTabDTO menyusun satu tab.
func toTabDTO(tab inboxkomunikasicabang.Tab) TabDTO {
	return TabDTO{
		Code:        tab.Code,
		Name:        tab.Name,
		Description: tab.Description,
		Columns:     toColumnDTOs(tab.Columns),
		Notice:      tab.Notice,
	}
}

// toColumnDTOs menyusun daftar kolom.
func toColumnDTOs(columns []inboxkomunikasicabang.Column) []ColumnDTO {
	result := make([]ColumnDTO, 0, len(columns))
	for _, column := range columns {
		result = append(result, ColumnDTO{Key: column.Key, Title: column.Title})
	}
	return result
}

// toListResponse menyusun jawaban daftar.
func toListResponse(listed usecase.Listed, portalAlias string) ListResponse {
	items := make([]ConversationDTO, 0, len(listed.Page.Items))
	for _, item := range listed.Page.Items {
		items = append(items, toConversationDTO(item))
	}

	return ListResponse{
		Tab:   toTabDTO(listed.Query.Tab),
		Items: items,
		Pagination: PaginationDTO{
			Page:       listed.Page.Pagination.Page,
			Size:       listed.Page.Pagination.Size,
			Total:      listed.Page.Total,
			TotalPages: listed.Page.TotalPages(),
		},
		Summary: SummaryDTO{
			NotAnswered: listed.Summary.NotAnswered,
			Answered:    listed.Summary.Answered,
			Total:       listed.Summary.Total(),
		},
		Branch: toBranchDTO(listed.Query.Branch),
		Portal: portalAlias,
	}
}

// toConversationDTO menyusun satu baris grid.
func toConversationDTO(item inboxkomunikasicabang.Conversation) ConversationDTO {
	return ConversationDTO{
		ID:             item.ID,
		CreatedAt:      item.CreatedAt,
		Sender:         joinWithParenthesis(item.SenderOrigin, item.SenderOperator),
		SenderOrigin:   item.SenderOrigin,
		SenderOperator: item.SenderOperator,
		Message:        item.Message,
		Reply:          item.Reply,
		Replier:        joinWithParenthesis(item.ReplierName, item.RecipientOrigin),
		Recipient:      item.RecipientOrigin,
		Status:         item.Status,
		RepliedAt:      item.RepliedAt,
	}
}

// toConversationDetailResponse menyusun jawaban layar detail.
func toConversationDetailResponse(
	detail inboxkomunikasicabang.ConversationDetail,
	branch inboxkomunikasicabang.BranchFilter,
	portalAlias string,
) ConversationDetailResponse {
	messages := make([]ThreadMessageDTO, 0, len(detail.Messages))
	for _, message := range detail.Messages {
		messages = append(messages, ThreadMessageDTO{
			CreatedAt:      message.CreatedAt,
			Sender:         joinWithParenthesis(message.SenderOrigin, message.SenderOperator),
			SenderOrigin:   message.SenderOrigin,
			SenderOperator: message.SenderOperator,
			Message:        message.Message,
			Reply:          message.Reply,
			ReplierName:    message.ReplierName,
			RepliedAt:      message.RepliedAt,
		})
	}

	attachments := make([]AttachmentDTO, 0, len(detail.Attachments))
	for _, attachment := range detail.Attachments {
		attachments = append(attachments, AttachmentDTO{
			DocumentID: attachment.DocumentID,
			TypeName:   attachment.TypeName,
			DetailName: attachment.DetailName,
			Note:       attachment.Note,
			UploadedAt: attachment.UploadedAt,
			Uploaded:   attachment.Uploaded(),
		})
	}

	return ConversationDetailResponse{
		ID:               detail.ID,
		Origin:           detail.Origin,
		Messages:         messages,
		Attachments:      attachments,
		WriteStillInPega: true,
		Branch:           toBranchDTO(branch),
		Portal:           portalAlias,
	}
}

// toBranchDTO menyusun keterangan batas data.
func toBranchDTO(filter inboxkomunikasicabang.BranchFilter) BranchDTO {
	return BranchDTO{
		Code:       filter.Code,
		HeadOffice: filter.HeadOffice,
		Resolved:   filter.Resolved,
		Notice:     branchNotice(filter),
	}
}

// branchNotice menyusun kalimat yang menjelaskan batas data kepada pengguna.
//
// Ketiga keadaan dijelaskan BERBEDA, dan itu inti gunanya: keadaan ketiga — cabang yang
// tidak terbaca — tampak persis seperti keadaan kedua di layar, dan hanya kalimat inilah
// yang membedakannya.
func branchNotice(filter inboxkomunikasicabang.BranchFilter) string {
	switch {
	case filter.HeadOffice && !filter.Resolved:
		return "Kode cabang Anda tidak dapat dibaca dari data kepegawaian, sehingga " +
			"daftar ini menampilkan percakapan KANTOR PUSAT. Bila Anda petugas cabang, " +
			"laporkan — daftar yang tampil kemungkinan bukan milik cabang Anda."

	case filter.HeadOffice:
		return "Menampilkan percakapan KANTOR PUSAT — yang dikirim ke pusat maupun yang " +
			"dikirim dari pusat."

	default:
		return "Menampilkan percakapan cabang " + filter.Code + " saja — yang dikirim ke " +
			"cabang ini maupun yang dikirim darinya. Percakapan cabang lain tidak tampil."
	}
}

// joinWithParenthesis merakit `kiri (kanan)`, melewatkan bagian yang kosong.
//
// Kedua section perakit — `PengirimKomunikasi` dan `PenjawabKomunikasi` — menggambar tanda
// kurung sebagai teks tetap (`pyCaption (` dan `pyCaption )`), sehingga di layar lama sel
// yang salah satu bahannya kosong tetap menampilkan kurung kosong seperti `PUSAT ()`.
//
// Itu TIDAK direplikasi, dan ini satu-satunya tempat modul ini memperbaiki tampilan tanpa
// menunggu keputusan: kurung kosong bukan informasi, tidak mengubah baris mana yang tampil,
// dan tidak dapat disalahartikan sebagai data. Selisihnya berhenti di tanda baca.
func joinWithParenthesis(left, right string) string {
	switch {
	case left == "" && right == "":
		return ""
	case right == "":
		return left
	case left == "":
		return right
	default:
		return left + " (" + right + ")"
	}
}
