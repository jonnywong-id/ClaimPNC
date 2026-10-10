// Package inboxcompliancehttp adalah lapisan transport modul Inbox Compliance.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: WorkItem membawa Reference, kunci
// teknis Pega yang tidak pernah ditampilkan.
package inboxcompliancehttp

import (
	"strings"
	"time"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/usecase"
	"claim-pnc/internal/platform/clock"
)

// dateLayout adalah bentuk tanggal pada kontrak API: YYYY-MM-DD.
//
// Jam sengaja tidak dikirim. Tak satu pun kolom di kedua grid menampilkan jam, dan
// mengirimkannya akan membuat layar harus memutuskan zona waktu mana yang dipakai
// menampilkannya — keputusan yang seharusnya hanya ada di satu tempat (`F-5`).
const (
	dateLayout = "2006-01-02"

	// dateTimeLayout dipakai SATU kolom saja: Tanggal Kirim Audit Compliance pada tab
	// Post Audit, yang di layar Pega menampilkan jamnya. Lihat WorkItemDTO.PostAuditSent.
	dateTimeLayout = "2006-01-02 15:04"
)

// WorkItemDTO adalah satu baris pekerjaan.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang
// digambar dari `kolom` pada tab yang sedang terbuka — bukan dari ada-tidaknya isian,
// karena isian yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya
// menghilang begitu saja.
type WorkItemDTO struct {
	// Reference adalah kunci teknis yang dibutuhkan tombol buka detail klaim.
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom.
	Reference string `json:"referensi"`

	CaseID       string `json:"nomor_case"`
	ClaimNumber  string `json:"no_klaim"`
	PolicyNumber string `json:"no_polis"`
	InsuredName  string `json:"nama_tertanggung"`
	BusinessName string `json:"nama_bisnis"`
	BranchName   string `json:"nama_cabang"`
	AdminName    string `json:"nama_admin"`

	ComplianceSent *string `json:"tanggal_kirim_compliance"`

	// PostAuditSent membawa TANGGAL DAN JAM — `2026-04-22 13:46` — berbeda dari kolom
	// tanggal lain di modul ini yang hanya membawa tanggalnya.
	//
	// Bukan ketidakseragaman yang terlewat: layar Pega menampilkan kolom ini sebagai
	// `22/04/25 13:46`, dan `D-13` menetapkan tampilan ditiru. Kolom tanggal lain memang
	// tidak menampilkan jam di sana.
	PostAuditSent *string `json:"tanggal_kirim_post_audit"`

	ComplianceRemarks string `json:"catatan_compliance"`

	// Aging adalah teks yang digambar di kolom Aging, mengikuti bentuk sistem lama —
	// "5 hours ago", "2 days 3 hours ago". Kosong bila tidak dapat dihitung.
	Aging string `json:"aging"`

	// AgingHours adalah angka mentahnya, sudah dipotong akhir pekan.
	//
	// Ia dikirim BERDAMPINGAN dengan teksnya, bukan menggantikannya, karena teks "2 days
	// 3 hours ago" tidak dapat dibandingkan sebagai angka — mengurutkannya sebagai teks
	// menaruh "10 hours ago" sebelum "2 days ago". Layar hari ini belum mengurutkan kolom
	// Aging (lihat catatan di InboxCompliancePage.tsx), dan angka ini yang membuatnya
	// dapat dilakukan kelak tanpa mengubah kontrak.
	//
	// null berarti tidak dapat dihitung — dan itu BERBEDA dari 0, yang berarti baru saja
	// masuk antrean (`P-5` butir 13).
	AgingHours *float64 `json:"aging_jam"`

	// Outstanding adalah kolom "OutStanding" pada tab Post Audit — "1 year 5 months ago".
	//
	// Ia BUKAN kolom Aging dengan nama lain: dasarnya waktu kalender apa adanya, sedangkan
	// Aging memotong akhir pekan. Lihat inboxcompliance.FormatElapsed.
	Outstanding string `json:"outstanding"`
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	// Key menyebut isian mana pada baris yang digambar.
	Key string `json:"kunci"`

	// Title adalah judul kolom yang dibaca pengguna.
	Title string `json:"judul"`
}

// TabDTO adalah satu antrean kerja beserta bentuk gridnya.
type TabDTO struct {
	Code        string `json:"kode"`
	Name        string `json:"nama"`
	Description string `json:"keterangan"`

	Columns []ColumnDTO `json:"kolom"`

	// Available menyatakan tab ini sudah dapat dilayani. Tab yang belum dapat dilayani
	// tetap dikirim — lihat inboxcompliance.Tab.Available.
	Available bool `json:"tersedia"`

	// Blocker menyebut apa yang kurang dan siapa pemiliknya. Kosong bila tersedia.
	Blocker string `json:"penghalang,omitempty"`
}

// MetadataResponse adalah jawaban GET /api/inbox-compliance/tab.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"tab"`
	DefaultTab string   `json:"tab_bawaan"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal
	// yang sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`

	// Limitations menyatakan hal yang BELUM berjalan penuh di modul ini beserta
	// alasannya, dalam kalimat yang dapat langsung ditampilkan ke pengguna.
	//
	// Ia dikirim sebagai data, bukan ditulis tetap di layar, supaya ia hilang dengan
	// sendirinya begitu penghalangnya hilang — tanpa menyunting frontend.
	Limitations []string `json:"keterbatasan"`
}

// PaginationDTO adalah keterangan halaman.
type PaginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// ListResponse adalah jawaban GET /api/inbox-compliance.
type ListResponse struct {
	Tab        TabDTO        `json:"tab"`
	Items      []WorkItemDTO `json:"baris"`
	Pagination PaginationDTO `json:"paginasi"`
	Portal     string        `json:"portal"`
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

// limitations adalah keterbatasan modul ini yang perlu diketahui pengguna.
//
// Keduanya bukan cacat, melainkan akibat keadaan yang sudah tercatat. Menuliskannya di layar
// membuat pengguna tidak melaporkannya berulang kali sebagai kerusakan.
var limitations = []string{
	"Kolom Aging memotong hari Sabtu dan Minggu, tetapi TIDAK memotong hari libur " +
		"nasional dan tidak mengenal jam kerja. Itu perilaku yang sama dengan sistem " +
		"lama, dan angkanya karena itu tidak dapat dibandingkan dengan angka TAT pada " +
		"laporan KPI yang memakai dasar berbeda.",

	// Catatan kedua DIHAPUS pada 2026-10-05 atas keputusan Work Owner.
	//
	// Isinya: tab Post Audit menampilkan seluruh baris `T_CLAIM_COMPLIANCE_H` karena tabel
	// itu tidak punya kolom status, sehingga penyaring `pyStatusWork = "New"` milik Report
	// Definition lama tidak dapat direplikasi.
	//
	// Work Owner menyatakan tabelnya diisi mengikuti aplikasi Pega, sehingga penyaringnya
	// sudah terjadi di SUMBER dan tidak perlu diulang di sini — dan permintaan menambah
	// kolom status ditolak dengan alasan yang sama. Dengan begitu peringatan "daftar ini
	// bisa lebih panjang daripada di Pega" menjadi dugaan, bukan keterangan, dan
	// peringatan yang isinya dugaan hanya melatih pengguna mengabaikan kotak ini.
	//
	// Keadaan yang menyebabkannya TIDAK berubah, dan tetap tercatat di
	// `docs/keputusan-implementasi.md` §31. Yang berubah hanya tempat mencatatnya.
}

// toWorkItemDTO mengubah satu baris.
func toWorkItemDTO(item inboxcompliance.WorkItem) WorkItemDTO {
	return WorkItemDTO{
		Reference:         item.Reference,
		CaseID:            item.CaseID,
		ClaimNumber:       item.ClaimNumber,
		PolicyNumber:      item.PolicyNumber,
		InsuredName:       item.InsuredName,
		BusinessName:      item.BusinessName,
		BranchName:        item.BranchName,
		AdminName:         item.AdminName,
		ComplianceSent:    toDateString(item.ComplianceSentDate),
		PostAuditSent:     toDateTimeString(item.PostAuditSentDate),
		ComplianceRemarks: item.ComplianceRemarks,
		Aging:             item.AgingLabel(),
		AgingHours:        item.AgingHours,
		Outstanding:       item.Outstanding,
	}
}

// toWorkItemListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toWorkItemListDTO(items []inboxcompliance.WorkItem) []WorkItemDTO {
	result := make([]WorkItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toWorkItemDTO(item))
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxcompliance.Tab) TabDTO {
	columns := make([]ColumnDTO, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		columns = append(columns, ColumnDTO{Key: column.Key, Title: column.Title})
	}

	return TabDTO{
		Code:        tab.Code,
		Name:        tab.Name,
		Description: tab.Description,
		Columns:     columns,
		Available:   tab.Available,
		Blocker:     tab.Blocker,
	}
}

// toMetadataResponse merakit jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		tabs = append(tabs, toTabDTO(tab))
	}

	return MetadataResponse{
		Tabs:        tabs,
		DefaultTab:  meta.DefaultTab,
		Portal:      portalAlias,
		Limitations: limitations,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxcompliance.Page) PaginationDTO {
	return PaginationDTO{
		Page:       page.Pagination.Page,
		Size:       page.Pagination.Size,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
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

// toDateString memformat tanggal menurut TANGGAL WIB, atau nil bila kosong.
//
// Pointer, bukan teks kosong: `null` menyatakan "tidak ada tanggalnya", sedangkan `""` akan
// terbaca layar sebagai tanggal yang gagal diformat.
//
// # Kenapa WIB, bukan UTC
//
// Karena yang dikirim adalah TANGGAL tanpa jam, dan tanggal hanya punya arti setelah zona
// waktunya ditetapkan. Klaim yang masuk pukul 06.00 WIB tanggal 2 adalah pukul 23.00 UTC
// tanggal 1 — memformatnya sebagai UTC akan menampilkan tanggal kemarin kepada petugas yang
// baru saja memasukkannya pagi itu.
//
// Pergeseran zonanya tetap terjadi di satu tempat saja, yakni clock.ZoneWIB (`F-5`).
func toDateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.In(clock.ZoneWIB).Format(dateLayout)
	return &formatted
}

// toDateTimeString memformat waktu berikut jamnya menurut WIB, atau nil bila kosong.
//
// Zonanya WIB dengan alasan yang sama seperti toDateString, dan di sini akibatnya lebih
// besar: yang bergeser bukan hanya tanggalnya melainkan jam yang benar-benar terbaca di
// kolom. Pukul 13.46 WIB adalah pukul 06.46 UTC, dan petugas yang mengirimnya siang hari
// akan melihat jam pagi.
func toDateTimeString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.In(clock.ZoneWIB).Format(dateTimeLayout)
	return &formatted
}

// SendPostAuditRequest adalah badan permintaan POST /api/inbox-compliance/post-audit.
type SendPostAuditRequest struct {
	// Reference adalah kunci klaim yang dikirim, yakni isian `referensi` pada baris tab
	// Compliance. Layar mengirimkannya kembali apa adanya; ia tidak pernah diketik.
	Reference string `json:"referensi"`

	// Remarks adalah Catatan. Boleh kosong — kolomnya nullable dan tidak ada bukti di
	// export bahwa ia wajib.
	Remarks string `json:"catatan"`
}

// SendPostAuditResponse adalah jawaban pengiriman yang berhasil.
type SendPostAuditResponse struct {
	// CaseID adalah nomor yang terbit, misalnya `CPL.26.1`.
	CaseID string `json:"nomor_case"`

	ClaimNumber  string `json:"no_klaim"`
	InsuredName  string `json:"nama_tertanggung"`
	PolicyNumber string `json:"no_polis"`
	Remarks      string `json:"catatan"`

	// SentAt membawa tanggal berikut jam, sama seperti kolomnya di tab Post Audit.
	SentAt *string `json:"tanggal_kirim_post_audit"`

	Portal string `json:"portal"`
}

// toSendPostAuditResponse merakit jawaban pengiriman.
func toSendPostAuditResponse(sent usecase.Sent, portalAlias string) SendPostAuditResponse {
	at := sent.Entry.SentAt

	return SendPostAuditResponse{
		CaseID:       sent.Entry.CaseID,
		ClaimNumber:  sent.Entry.ClaimNumber,
		InsuredName:  sent.Entry.InsuredName,
		PolicyNumber: sent.Entry.PolicyNumber,
		Remarks:      sent.Entry.Remarks,
		SentAt:       toDateTimeString(&at),
		Portal:       portalAlias,
	}
}

// ── Form Compliance Checker ──────────────────────────────────────────────────

// ChoiceDTO adalah satu Pilihan Compliance beserta labelnya.
//
// Keempatnya datang dari server, bukan disalin ke layar, dengan alasan yang sama seperti
// daftar tab: nilainya adalah hasil pembacaan `Property/PilihanCompliance_property.xml`,
// dan tempat pembacaan itu tercatat adalah backend.
type ChoiceDTO struct {
	Value string `json:"nilai"`
	Label string `json:"label"`
}

// DecisionDTO adalah keputusan Compliance yang sudah tersimpan.
//
// Ia muncul pada respons pembukaan form HANYA bila klaimnya sudah pernah diputuskan —
// pointer pada CheckerResponse, bukan struct kosong, supaya layar dapat membedakan "belum
// pernah diputuskan" dari "diputuskan dengan pilihan 0", yang berarti Fraud/Tolak.
type DecisionDTO struct {
	Choice string `json:"pilihan"`

	// ChoiceLabel disertakan supaya layar tidak perlu mencocokkan nilainya sendiri ke
	// daftar pilihan. Kosong bila nilainya tidak dikenal — keadaan yang hanya mungkin
	// terjadi bila barisnya ditulis tangan ke basis data.
	ChoiceLabel string `json:"pilihan_label"`

	Note string `json:"note"`

	// Comments adalah grid komentar — tempat petugas Compliance menulis.
	//
	// Jangan tertukar dengan Remarks di bawahnya: yang ini DIISI petugas, yang itu
	// ditampilkan saja.
	Comments []CommentDTO `json:"komentar"`

	// Remarks adalah catatan **Investigator** — `.ClaimData.ComplianceRemark`, yang pada
	// form Pega bertanda `pyEditOptions=Read-only`. Ia datang dari klaim, bukan dari
	// layar, dan karena itu tidak ada padanannya di SubmitDecisionRequest.
	Remarks string `json:"catatan_investigator"`

	DecidedBy string  `json:"diputuskan_oleh"`
	DecidedAt *string `json:"diputuskan_pada"`

	// Kedua tanggal berikut terisi HANYA pada pilihannya masing-masing: ValidatedAt pada
	// Bayar/Valid, SentToPostAuditAt pada Bayar/PostAudit.
	ValidatedAt       *string `json:"tanggal_valid"`
	SentToPostAuditAt *string `json:"tanggal_kirim_post_audit"`
}

// CheckerResponse adalah form Compliance Checker yang terbuka.
type CheckerResponse struct {
	// Claim adalah baris antrean yang dibuka — bentuk yang SAMA dengan baris pada daftar,
	// bukan bentuk kedua. Menyusun bentuk kedua akan membuat keduanya dapat berbeda.
	Claim WorkItemDTO `json:"klaim"`

	// Choices adalah keempat Pilihan Compliance.
	Choices []ChoiceDTO `json:"pilihan"`

	// Decision adalah keputusan yang sudah tersimpan, bila ada.
	Decision *DecisionDTO `json:"keputusan"`

	// Actions adalah tombol mana yang boleh digambar untuk klaim ini.
	//
	// Ia datang dari SERVER, bukan disimpulkan layar dari lini bisnis, karena syaratnya
	// dibaca dari rule Pega dan tempat pembacaan itu tercatat adalah backend — alasan yang
	// sama seperti daftar Pilihan Compliance. Menyimpulkannya di layar berarti aturan yang
	// sama hidup di dua tempat, dan yang di layar tidak mengikat apa pun.
	Actions ActionsDTO `json:"tombol"`

	// Limitation adalah kalimat yang WAJIB sampai ke petugas, bukan hanya ke DBA.
	//
	// Form ini mencatat keputusan; ia belum menjalankan alurnya. Klaimnya di Pega tidak
	// berpindah status dan tidak keluar dari antrean, karena tabel klaim masih dimiliki
	// Pega selama masa paralel (`P-1`). Menyembunyikannya akan membuat petugas mengira
	// pekerjaannya selesai padahal klaimnya masih menunggu di Pega.
	Limitation string `json:"keterbatasan"`

	// Medan `dokumen` dan `dokumen_gagal_dibaca` DICABUT 2026-10-08 bersama gridnya —
	// lihat catatan pada usecase.CheckerOpened.

	// RejectPrefill mengisi tiga isian form Surat Penolakan saat dialognya dibuka.
	//
	// Dikirim bersama form, bukan lewat permintaan tersendiri saat dialognya dibuka —
	// lihat usecase.CheckerOpened.RejectPrefill untuk alasannya.
	RejectPrefill RejectPrefillDTO `json:"pra_isi_surat_penolakan"`

	// ShowDocumentChecklist menyatakan tab "Dokumen" DIGAMBAR, yakni `IsTravel`.
	//
	// Terpisah dari panjang DocumentChecklist, alasan yang sama dengan
	// ShowSurveyResults: lini Travel yang masternya belum diisi tetap menggambar tabnya
	// dengan tabel kosong, dan itu tidak dapat dibedakan dari lini lain bila layar hanya
	// melihat jumlah barisnya.
	ShowDocumentChecklist bool `json:"tampilkan_daftar_dokumen"`

	// DocumentChecklist mengisi tab "Dokumen" — hanya terisi pada lini Travel.
	DocumentChecklist []DocumentChecklistDTO `json:"daftar_dokumen"`

	// DocumentChecklistFailed menyatakan daftar periksa GAGAL dibaca.
	//
	// Tanpa penanda ini, tabel kosong karena gangguan basis data tak terbedakan dari
	// lini yang masternya memang belum diisi — dan petugas dapat menyimpulkan tidak ada
	// dokumen yang wajib.
	DocumentChecklistFailed bool `json:"daftar_dokumen_gagal_dibaca"`

	// ShowSurveyResults menyatakan blok "Hasil Investigasi" DIGAMBAR, yakni `IsPA`.
	//
	// Terpisah dari panjang SurveyResults dengan sengaja. Klaim PA yang belum pernah
	// disurvei tetap menggambar bloknya — grid kosong, persis seperti di Pega — dan itu
	// tidak dapat dibedakan dari lini non-PA bila layar hanya melihat jumlah barisnya.
	//
	// Layar TIDAK menyimpulkan `IsPA` sendiri, alasan yang sama seperti tombol: syaratnya
	// dibaca dari rule Pega, dan tempat pembacaan itu tercatat adalah backend.
	ShowSurveyResults bool `json:"tampilkan_hasil_investigasi"`

	// SurveyResults mengisi blok "Hasil Investigasi" — hanya terisi pada lini PA.
	//
	// Senarai kosong, bukan null, supaya layar tidak perlu membedakan dua bentuk
	// "tidak ada baris".
	SurveyResults []SurveyResultDTO `json:"hasil_investigasi"`

	// SurveyResultsFailed menyatakan hasil investigasi GAGAL dibaca.
	//
	// Tanpa penanda ini, blok kosong karena gangguan basis data tak terbedakan dari blok
	// kosong karena klaimnya memang belum disurvei — dan petugas akan memutuskan klaim
	// PA dengan mengira tidak ada hasil investigasi, padahal ada.
	SurveyResultsFailed bool `json:"hasil_investigasi_gagal_dibaca"`

	Portal string `json:"portal"`
}

// SurveyResultDTO adalah satu baris blok "Hasil Investigasi".
//
// Keempat isiannya persis keempat kolom grid `isPA_PNC` pada
// `Section/ViewHasilSurvey-Section.xml` — tidak lebih. Lihat inboxcompliance.SurveyResult.
type SurveyResultDTO struct {
	// SurveyedAt bertipe pointer, sama dengan seluruh tanggal lain di modul ini:
	// `null` berarti kolomnya kosong, bukan 1 Januari tahun nol.
	SurveyedAt *string `json:"tanggal_investigasi"`

	ObjectName     string `json:"nama_peserta"`
	ObjectLocation string `json:"lokasi_objek"`
	Status         string `json:"status"`
}

// RejectPrefillDTO adalah isian form Surat Penolakan yang terbawa dari klaim.
//
// Hanya TIGA dari empat isian `AutoFillFormReject_Pre`. Yang keempat — Tanggal Keluar
// Rawat Inap — tidak punya kolom di `T_CLAIM_PNC`, sehingga ia selalu diketik petugas.
type RejectPrefillDTO struct {
	PatientName   string `json:"nama_pasien"`
	IncidentPlace string `json:"tempat_kejadian"`

	// IncidentDate bertipe pointer, sama dengan seluruh tanggal lain di modul ini:
	// `null` berarti kolomnya kosong, bukan 1 Januari tahun nol.
	IncidentDate *string `json:"tanggal_kejadian"`
}

func toRejectPrefillDTO(prefill inboxcompliance.RejectPrefill) RejectPrefillDTO {
	return RejectPrefillDTO{
		PatientName:   prefill.PatientName,
		IncidentPlace: prefill.IncidentPlace,

		// toDateString, bukan toDateTimeString: Tanggal Kejadian adalah tanggal murni,
		// dan jam pada surat penolakan tidak punya arti apa pun.
		IncidentDate: toDateString(prefill.IncidentDate),
	}
}

// DocumentDTO adalah satu baris grid dokumen.
//
// Ketiga kolom datanya persis sel 55–57 `Section/CompliancePNC-Section.xml`; kolom
// keempat di Pega adalah ikon tanpa judul, dan di sini ia menjadi `id_penyimpanan`.
type DocumentDTO struct {
	ID       string `json:"id"`
	Name     string `json:"nama_file"`
	MimeType string `json:"tipe_file"`
	Category string `json:"kategori"`

	// StorageID dikirim karena tombol "lihat" membutuhkannya untuk meminta tautan baru
	// lewat `NewLinkDokumenPNC`. Ia bukan kolom yang digambar.
	StorageID string `json:"id_penyimpanan"`

	UploadedAt *string `json:"tanggal_unggah"`
}

func toDocumentDTOs(documents []inboxcompliance.Document) []DocumentDTO {
	rows := make([]DocumentDTO, 0, len(documents))
	for _, document := range documents {
		rows = append(rows, DocumentDTO{
			ID:         document.ID,
			Name:       document.Name,
			MimeType:   document.MimeType,
			Category:   document.Category,
			StorageID:  document.StorageID,
			UploadedAt: toDateTimeString(document.UploadedAt),
		})
	}
	return rows
}

func toSurveyResultDTOs(results []inboxcompliance.SurveyResult) []SurveyResultDTO {
	rows := make([]SurveyResultDTO, 0, len(results))
	for _, result := range results {
		rows = append(rows, SurveyResultDTO{
			SurveyedAt:     toDateTimeString(result.SurveyedAt),
			ObjectName:     result.ObjectName,
			ObjectLocation: result.ObjectLocation,
			Status:         result.Status,
		})
	}
	return rows
}

// SubmitDecisionRequest adalah badan permintaan penyimpanan keputusan.
type SubmitDecisionRequest struct {
	// Choice adalah nilai Pilihan Compliance — `"0"`…`"3"`.
	//
	// Teks, bukan angka, karena `pyStandardValue` memang teks. Mengirimnya sebagai angka
	// akan membuat `0` tidak dapat dibedakan dari field yang tidak dikirim sama sekali.
	Choice string `json:"pilihan"`

	// Note adalah "Note Lainya" — layar hanya menampilkannya saat Choice bernilai `"3"`.
	Note string `json:"note"`

	// Comments adalah baris grid komentar.
	//
	// Tidak ada `catatan` di sini, dan ketiadaannya disengaja: catatan Investigator
	// read-only di Pega, sehingga menerimanya dari layar berarti membiarkan klien
	// menimpanya.
	Comments []CommentRequestDTO `json:"komentar"`

	// Action adalah TOMBOL yang ditekan — `"simpan"` atau `"kirim"`.
	//
	// Keduanya dibedakan karena di Pega tombolnya memang mengerjakan hal yang berbeda:
	// "Simpan Data" tidak memanggil `SetComplianceResult`, sehingga ia tidak memindahkan
	// klaim. Lihat inboxcompliance.ActionSave.
	//
	// Kosong diperlakukan sebagai `"simpan"` — bentuk yang paling tidak berakibat.
	Action string `json:"aksi"`
}

// CommentDTO adalah satu baris grid komentar yang sudah tersimpan.
type CommentDTO struct {
	// Index adalah nomor baris yang dilihat petugas — grid Pega bernomor.
	Index int `json:"urutan"`

	// Date adalah Tanggal Komentar. Pointer supaya baris tanpa tanggal — keadaan yang
	// hanya mungkin terjadi bila barisnya ditulis tangan ke basis data — tidak tampil
	// sebagai tahun 1.
	Date *string `json:"tanggal"`

	Text string `json:"komentar"`
}

// CommentRequestDTO adalah satu baris grid komentar sebagaimana dikirim layar.
type CommentRequestDTO struct {
	// Date boleh kosong: Pega mengisi sel Tanggal Komentar dengan waktu sekarang sebagai
	// nilai BAWAAN yang masih dapat diubah petugas. Kosong berarti pakai bawaan itu.
	Date string `json:"tanggal"`

	Text string `json:"komentar"`
}

// SubmitDecisionResponse adalah hasil penyimpanan keputusan.
type SubmitDecisionResponse struct {
	Decision DecisionDTO `json:"keputusan"`

	// PostAudit terisi HANYA pada pilihan Bayar/PostAudit, yakni ketika baris Post Audit
	// ikut terbit. Layar memakainya untuk menampilkan nomor yang terbit pada pesan
	// berhasil.
	PostAudit *SendPostAuditResponse `json:"post_audit"`

	Limitation string `json:"keterbatasan"`
	Portal     string `json:"portal"`
}

// checkerLimitation adalah kalimat keterbatasan yang dikirim bersama kedua respons di atas.
//
// Ia satu konstanta, bukan dua kalimat yang ditulis terpisah, supaya keduanya tidak dapat
// berbeda saat salah satunya diperbaiki.
const checkerLimitation = "Keputusan tersimpan di aplikasi baru. Status klaim di sistem " +
	"lama belum ikut berubah dan klaimnya masih menunggu di antrean Compliance Pega."

func toChoiceDTOs(choices []inboxcompliance.Choice) []ChoiceDTO {
	out := make([]ChoiceDTO, 0, len(choices))
	for _, c := range choices {
		out = append(out, ChoiceDTO{Value: c.Value, Label: c.Label})
	}
	return out
}

func toDecisionDTO(decision inboxcompliance.Decision) DecisionDTO {
	label := ""
	if choice, known := inboxcompliance.FindChoice(decision.Choice); known {
		label = choice.Label
	}

	decidedAt := decision.DecidedAt
	return DecisionDTO{
		Choice:            decision.Choice,
		ChoiceLabel:       label,
		Note:              decision.Note,
		Comments:          toCommentDTOs(decision.Comments),
		Remarks:           decision.Remarks,
		DecidedBy:         decision.DecidedBy,
		DecidedAt:         toDateTimeString(&decidedAt),
		ValidatedAt:       toDateTimeString(decision.ValidatedAt),
		SentToPostAuditAt: toDateTimeString(decision.SentToPostAuditAt),
	}
}

// ActionsDTO adalah kelima tombol pada form Compliance Checker.
//
// Syaratnya dibaca dari DUA bilah tombol Pega yang saling melengkapi — `CompliancePNC`
// S14 (`!IsTravel`) dan `ComplianceChecker` S4 (`IsTravel`). Lihat FormActions untuk
// rinciannya, termasuk koreksi atas pembacaan pertama yang hanya melihat salah satunya.
type ActionsDTO struct {
	UploadDocument       bool `json:"unggah_dokumen"`
	DownloadRejectLetter bool `json:"unduh_dokumen_reject"`
	Save                 bool `json:"simpan"`

	// SendToAnalyst tampil pada lini PA.
	SendToAnalyst bool `json:"kirim_ke_analyst"`

	// SendToTechnician tampil pada lini Travel.
	//
	// Tombolnya tampil tetapi BELUM dapat dijalankan — Data Transform `SendToPIC` tidak
	// ada di export (`R-16`). Lihat FormActions.
	SendToTechnician bool `json:"kirim_ke_pic_teknik"`
}

func toActionsDTO(actions inboxcompliance.FormActions) ActionsDTO {
	return ActionsDTO{
		UploadDocument:       actions.UploadDocument,
		DownloadRejectLetter: actions.DownloadRejectLetter,
		Save:                 actions.Save,
		SendToAnalyst:        actions.SendToAnalyst,
		SendToTechnician:     actions.SendToTechnician,
	}
}

func toCheckerResponse(opened usecase.CheckerOpened, portalAlias string) CheckerResponse {
	response := CheckerResponse{
		Claim:      toWorkItemDTO(opened.Case.Claim),
		Choices:    toChoiceDTOs(opened.Choices),
		Actions:    toActionsDTO(opened.Case.Actions()),
		Limitation: checkerLimitation,

		// `IsPA` dibaca dari lini bisnis klaimnya, sumber yang sama dengan tombol —
		// bukan disimpulkan dari ada-tidaknya baris survei.
		ShowSurveyResults: opened.Case.Claim.GroupPanel ==
			inboxcompliance.GroupPanelPersonalAccident,

		SurveyResults:       toSurveyResultDTOs(opened.SurveyResults),
		SurveyResultsFailed: opened.SurveyResultsError != nil,

		RejectPrefill: toRejectPrefillDTO(opened.RejectPrefill),

		// `IsTravel` dibaca dari lini bisnis klaimnya, sumber yang sama dengan tombol —
		// bukan disimpulkan dari ada-tidaknya baris daftar periksa.
		ShowDocumentChecklist: opened.Case.Claim.GroupPanel ==
			inboxcompliance.GroupPanelTravel,

		DocumentChecklist:       toDocumentChecklistDTOs(opened.DocumentChecklist),
		DocumentChecklistFailed: opened.DocumentChecklistError != nil,

		Portal: portalAlias,
	}

	if opened.Case.Decision != nil {
		decision := toDecisionDTO(*opened.Case.Decision)
		response.Decision = &decision
	}

	return response
}

func toSubmitDecisionResponse(
	decided usecase.Decided, portalAlias string,
) SubmitDecisionResponse {
	response := SubmitDecisionResponse{
		Decision:   toDecisionDTO(decided.Decision),
		Limitation: checkerLimitation,
		Portal:     portalAlias,
	}

	if decided.PostAudit != nil {
		postAudit := toSendPostAuditResponse(
			usecase.Sent{Entry: *decided.PostAudit, Claim: decided.Claim},
			portalAlias,
		)
		response.PostAudit = &postAudit
	}

	return response
}

// toCommentDTOs memetakan grid komentar ke bentuk yang dibaca layar.
//
// Senarai kosong dipulangkan sebagai `[]`, bukan `null`: layar menggambar grid dengan
// memetakan senarai itu, dan `null` memaksa setiap pemanggil memeriksanya lebih dulu.
func toCommentDTOs(comments []inboxcompliance.Comment) []CommentDTO {
	out := make([]CommentDTO, 0, len(comments))
	for _, comment := range comments {
		date := comment.Date
		out = append(out, CommentDTO{
			Index: comment.Index,
			Date:  toDateTimeString(&date),
			Text:  comment.Text,
		})
	}
	return out
}

// toCommentInputs memetakan baris grid dari layar ke bentuk domain.
//
// # Tanggal yang tidak dapat diurai DIABAIKAN, bukan diadukan
//
// Bukan kelonggaran: sel Tanggal Komentar di Pega punya nilai bawaan
// `@(Pega-RULES:DateTime).CurrentDateTime()`, sehingga "tidak ada tanggal yang sah" dan
// "tanggal tidak dikirim" berakhir pada perlakuan yang sama — pakai waktu keputusan.
// Menolak permintaannya justru akan membuang komentar yang sudah diketik petugas hanya
// karena selnya salah format.
func toCommentInputs(rows []CommentRequestDTO) []inboxcompliance.CommentInput {
	out := make([]inboxcompliance.CommentInput, 0, len(rows))
	for _, row := range rows {
		input := inboxcompliance.CommentInput{Text: row.Text}

		if trimmed := strings.TrimSpace(row.Date); trimmed != "" {
			// Diurai dalam WIB, zona yang sama dengan yang dipakai menampilkannya
			// (`F-5`). Mengurainya sebagai UTC akan menggeser setiap tanggal 7 jam —
			// persis kelas cacat yang `R-12` catat.
			if parsed, err := time.ParseInLocation(
				dateTimeLayout, trimmed, clock.ZoneWIB,
			); err == nil {
				input.Date = &parsed
			}
		}

		out = append(out, input)
	}
	return out
}

// OpenDocumentResponse adalah tautan siap buka untuk satu dokumen.
//
// Tautan penyimpanan yang BELUM dibungkus penampil sengaja TIDAK dikirim. Layar tidak
// membutuhkannya, dan mengirimkannya berarti menaruh tautan bertanda tangan di satu
// tempat lagi — padahal siapa pun yang memegangnya dapat membuka berkasnya.
type OpenDocumentResponse struct {
	Name      string `json:"nama_file"`
	ViewerURL string `json:"tautan"`
	Portal    string `json:"portal"`
}

// toDocumentDTO memetakan satu dokumen — dipakai jawaban unggah.
func toDocumentDTO(document inboxcompliance.Document) DocumentDTO {
	return DocumentDTO{
		ID:         document.ID,
		Name:       document.Name,
		MimeType:   document.MimeType,
		Category:   document.Category,
		StorageID:  document.StorageID,
		UploadedAt: toDateTimeString(document.UploadedAt),
	}
}

// GenerateRejectLetterRequest adalah isian form Download Dokumen Reject.
//
// Delapan isian beserta grid "Alasan" — persis `Section/FormRejectClaim_section`. Nama
// field JSON berbahasa Indonesia mengikuti `D-80`: ia kontrak, bukan nama internal.
//
// Seluruh tanggal bertipe TEKS, bukan tanggal. Isian "Tanggal Keluar Rawat Inap" tidak
// punya sumber basis data sehingga diketik bebas, dan satu-satunya tujuan nilai-nilai ini
// adalah digambar ke surat. Menguraikannya lalu menggambarnya kembali hanya menambah satu
// tempat yang dapat menolak masukan yang Pega terima.
type GenerateRejectLetterRequest struct {
	Recipient     string `json:"up"`
	Position      string `json:"jabatan"`
	PatientName   string `json:"nama_pasien"`
	IncidentPlace string `json:"tempat_kejadian"`
	IncidentDate  string `json:"tanggal_kejadian"`
	DischargeDate string `json:"tanggal_keluar_rawat_inap"`
	PaidAmount    string `json:"nilai_klaim_dibayarkan"`
	PaymentDate   string `json:"tanggal_pembayaran"`

	// Reasons adalah grid "Alasan" apa adanya, termasuk baris kosongnya.
	//
	// Baris kosong TIDAK ditolak di sini — ia dibuang saat penomoran. Menolaknya akan
	// membuat petugas yang menyisakan satu baris kosong di grid mendapat pesan kesalahan
	// alih-alih surat, dan baris kosong di grid adalah kejadian normal.
	Reasons []string `json:"alasan"`
}

// GenerateRejectLetterResponse adalah jawaban penerbitan surat.
type GenerateRejectLetterResponse struct {
	// Document adalah baris dokumen yang baru tercatat, berbentuk sama dengan baris grid
	// Dokumen — sehingga layar dapat menyisipkannya tanpa memuat ulang seluruh grid.
	Document DocumentDTO `json:"dokumen"`

	// Replaced menyatakan surat sebelumnya DIGANTI, bukan ditambahkan.
	//
	// Dikirim supaya layar dapat mengatakan "surat diperbarui" alih-alih "surat dibuat".
	// Tanpa ini, perilaku ganti-bukan-tambah tidak terlihat sama sekali oleh petugas, dan
	// ia dapat mengira suratnya yang lama masih ada.
	Replaced bool `json:"mengganti_surat_sebelumnya"`

	Portal string `json:"portal"`
}

// DocumentChecklistDTO adalah satu baris tab "Dokumen".
//
// Keempat kolomnya persis grid `UploadDocument` pada layar Pega — tidak lebih. Lihat
// inboxcompliance.DocumentChecklistItem.
type DocumentChecklistDTO struct {
	// CategoryID tidak digambar; ia dibawa karena tombol Unggah menempelkan berkas pada
	// kategori ini.
	CategoryID string `json:"id_kategori"`

	CategoryName string `json:"kategori"`

	// MandatoryLabel TEKS, bukan boolean — tiga nilai yang mungkin: "Ya", "Tidak", dan
	// KOSONG. Kosong memang terjadi di Pega; lihat catatan pada tipe domainnya.
	MandatoryLabel string `json:"wajib_unggah"`

	// MinUpload pointer: kolomnya boleh NULL, dan `0` berarti lain daripada "tidak
	// ditentukan".
	MinUpload *int `json:"minimal_unggah"`

	UploadedCount int `json:"total_sudah_diunggah"`
}

func toDocumentChecklistDTOs(
	items []inboxcompliance.DocumentChecklistItem,
) []DocumentChecklistDTO {
	rows := make([]DocumentChecklistDTO, 0, len(items))
	for _, item := range items {
		rows = append(rows, DocumentChecklistDTO{
			CategoryID:     item.CategoryID,
			CategoryName:   item.CategoryName,
			MandatoryLabel: item.MandatoryLabel,
			MinUpload:      item.MinUpload,
			UploadedCount:  item.UploadedCount,
		})
	}
	return rows
}

// ListDocumentsResponse adalah jawaban daftar lampiran klaim.
type ListDocumentsResponse struct {
	// Documents memakai bentuk baris yang sama dengan DocumentDTO — satu bentuk dokumen
	// di seluruh modul, bukan satu per layar.
	Documents []DocumentDTO `json:"dokumen"`

	Portal string `json:"portal"`
}

// ChangeDocumentCategoryRequest adalah badan permintaan pemindahan kategori.
type ChangeDocumentCategoryRequest struct {
	// Category adalah `DOC_TYPE_DT_ID` kategori TUJUAN.
	//
	// Hanya ini yang datang dari badan; klaim dan dokumennya dari jalur — lihat
	// handler-nya.
	Category string `json:"kategori"`
}
