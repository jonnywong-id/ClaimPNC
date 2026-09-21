// Package pelaporanklaimhttp adalah lapisan transport modul Pelaporan Klaim: bentuk
// permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp,
// portalhttp, dan masterstatushttp: foldernya `http` supaya letaknya seragam antarmodul,
// nama paketnya `pelaporanklaimhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package pelaporanklaimhttp

import (
	"time"

	"claim-pnc/internal/pelaporanklaim"
)

// DateFormat adalah bentuk tanggal kalender pada API modul ini.
//
// Tanggal murni — tanggal kejadian dan tanggal terima dokumen — dikirim sebagai `YYYY-MM-DD`
// tanpa jam dan tanpa zona. Keduanya menjawab pertanyaan "hari apa", bukan "detik ke
// berapa", dan membawa jam pada keduanya hanya mengundang pergeseran satu hari saat
// zonanya ditafsirkan berbeda di ujung yang lain — persis kelas cacat yang `R-12` catat.
//
// Waktu peristiwa — kapan dicatat, kapan ditransfer — tetap dikirim lengkap dengan zona
// (RFC 3339 UTC), karena untuk keduanya detik memang berarti.
const DateFormat = "2006-01-02"

// ReportDTO adalah bentuk satu laporan klaim yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari pelaporanklaim.ClaimReport. Memakai tipe modul langsung
// sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (docs/Steering/08-TECHNICAL-STRATEGY.md §2 aturan 4).
//
// Nama field JSON berbahasa Indonesia mengikuti seluruh API aplikasi ini — ia KONTRAK,
// bukan nama internal, dan mengubahnya adalah perubahan yang merusak klien (`D-80`).
type ReportDTO struct {
	Number string `json:"nomor"`

	ReporterName string `json:"nama_pelapor"`
	SenderEmail  string `json:"email_pengirim"`
	SenderPhone  string `json:"telepon_pengirim"`
	CourierName  string `json:"nama_kurir"`
	EmailSubject string `json:"subjek_email"`

	PolicyNumber    string `json:"nomor_polis"`
	InsuredName     string `json:"nama_tertanggung"`
	InsuredEmail    string `json:"email_tertanggung"`
	BusinessCode    string `json:"kode_bisnis"`
	GroupPanel      string `json:"group_panel"`
	ReferenceNumber string `json:"nomor_referensi"`

	LossDate       string `json:"tanggal_kejadian"`
	LossLocation   string `json:"lokasi_kejadian"`
	Chronology     string `json:"kronologi"`
	DamageDetails  string `json:"rincian_kerusakan"`
	DriverLicense  string `json:"sim_pengendara"`
	EstimatedValue string `json:"nilai_estimasi"`
	ClaimType      string `json:"tipe_klaim"`

	DocumentCount        int    `json:"jumlah_dokumen"`
	DocumentReceivedDate string `json:"tanggal_terima_dokumen"`

	ClaimNumber          string `json:"nomor_klaim"`
	Transferred          bool   `json:"ditransfer"`
	TransferredAt        string `json:"tanggal_transfer"`
	RegisteredAt         string `json:"tanggal_registrasi"`
	NotTransferredReason string `json:"alasan_belum_transfer"`
	NotRegisteredNote    string `json:"catatan_belum_registrasi"`

	// Stage dan StageLabel DIHITUNG server, bukan disimpan dan bukan dihitung layar.
	//
	// Menghitungnya di layar berarti aturan tahap hidup di dua tempat, dan dua tempat
	// itu akan berselisih pada perubahan berikutnya. Label ikut dikirim supaya layar
	// tidak perlu memuat daftar terjemahannya sendiri.
	Stage      string `json:"tahap"`
	StageLabel string `json:"tahap_label"`

	// CanTransfer dan CanEdit dihitung server dari aturan yang sama yang ditegakkan
	// usecase. Tanpa keduanya, layar harus menirukan aturannya sendiri untuk memutuskan
	// tombol mana yang hidup — dan tiruan itulah yang akan menyimpang.
	CanTransfer bool `json:"dapat_ditransfer"`
	CanEdit     bool `json:"dapat_diubah"`

	BranchCode string `json:"kode_cabang"`
	CreatedBy  string `json:"diinput_oleh"`
	CreatedAt  string `json:"diinput_pada"`
	UpdatedAt  string `json:"diubah_pada"`
}

// SummaryDTO adalah jumlah laporan per tahap, untuk lencana di atas tiap tab.
type SummaryDTO struct {
	Stage string `json:"tahap"`
	Label string `json:"label"`
	Count int    `json:"jumlah"`
}

// ListResponse adalah jawaban GET /api/pelaporan-klaim.
type ListResponse struct {
	Reports []ReportDTO `json:"laporan"`

	// Total adalah banyaknya baris yang cocok SEBELUM dipotong paginasi — bukan panjang
	// senarai di atas. Tanpa pembedaan ini, layar tidak dapat mengetahui masih ada
	// halaman berikutnya.
	Total  int `json:"jumlah"`
	Limit  int `json:"batas"`
	Offset int `json:"lewati"`

	// Summary memuat KELIMA tahap selalu, termasuk yang jumlahnya nol. Tab yang menghilang
	// saat kosong membuat pengguna mengira tabnya tidak ada.
	Summary []SummaryDTO `json:"ringkasan"`
}

// SingleResponse adalah jawaban untuk satu laporan: ambil, catat, ubah, transfer, dan
// penautan klaim.
type SingleResponse struct {
	Report ReportDTO `json:"laporan"`
}

// SaveRequest adalah isian form catat dan ubah.
//
// Yang TIDAK diterima dari klien, dan alasannya masing-masing:
//
//   - nomor — dibuat penyimpanan saat mencatat, dan berada di jalur URL saat mengubah.
//   - ditransfer, tanggal transfer, nomor klaim, tanggal registrasi — keduanya punya
//     jalurnya sendiri, supaya perpindahan tahap tidak dapat terjadi sebagai efek samping
//     penyuntingan biasa.
//   - diinput oleh, diinput pada — datang dari sesi dan dari jam, bukan dari klien.
type SaveRequest struct {
	ReporterName string `json:"nama_pelapor"`
	SenderEmail  string `json:"email_pengirim"`
	SenderPhone  string `json:"telepon_pengirim"`
	CourierName  string `json:"nama_kurir"`
	EmailSubject string `json:"subjek_email"`

	PolicyNumber    string `json:"nomor_polis"`
	InsuredName     string `json:"nama_tertanggung"`
	InsuredEmail    string `json:"email_tertanggung"`
	BusinessCode    string `json:"kode_bisnis"`
	GroupPanel      string `json:"group_panel"`
	ReferenceNumber string `json:"nomor_referensi"`

	LossDate       string `json:"tanggal_kejadian"`
	LossLocation   string `json:"lokasi_kejadian"`
	Chronology     string `json:"kronologi"`
	DamageDetails  string `json:"rincian_kerusakan"`
	DriverLicense  string `json:"sim_pengendara"`
	EstimatedValue string `json:"nilai_estimasi"`
	ClaimType      string `json:"tipe_klaim"`

	DocumentCount        int    `json:"jumlah_dokumen"`
	DocumentReceivedDate string `json:"tanggal_terima_dokumen"`

	NotTransferredReason string `json:"alasan_belum_transfer"`
	NotRegisteredNote    string `json:"catatan_belum_registrasi"`
}

// LinkClaimRequest adalah isian aksi penautan laporan ke klaim.
type LinkClaimRequest struct {
	ClaimNumber string `json:"nomor_klaim"`
}

// ViolationDTO adalah satu aturan yang dilanggar beserta isian yang melanggarnya.
//
// Field dikirim supaya layar dapat menandai kolom yang salah, bukan sekadar menampilkan
// satu pesan di atas form. Pada form berisi 20 kolom, pembedaan itu menentukan.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Code dimaksudkan untuk dibaca program, Message untuk dibaca manusia. Klien membedakan
// jenis galat lewat Code — bukan dengan mencocokkan teks Message.
//
// Bentuknya sama persis dengan respons galat modul lain. Menyatukannya menjadi satu tipe
// bersama adalah lingkup `TKT-F1-004`, kontrak galat yang mengikat seluruh aplikasi, dan
// tiket itu masih terhalang keputusan Work Owner.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Details hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toDTO mengubah entitas domain menjadi bentuk yang dikirim ke peramban.
func toDTO(r pelaporanklaim.ClaimReport) ReportDTO {
	stage := r.Stage()
	return ReportDTO{
		Number:               r.Number,
		ReporterName:         r.ReporterName,
		SenderEmail:          r.SenderEmail,
		SenderPhone:          r.SenderPhone,
		CourierName:          r.CourierName,
		EmailSubject:         r.EmailSubject,
		PolicyNumber:         r.PolicyNumber,
		InsuredName:          r.InsuredName,
		InsuredEmail:         r.InsuredEmail,
		BusinessCode:         r.BusinessCode,
		GroupPanel:           r.GroupPanel,
		ReferenceNumber:      r.ReferenceNumber,
		LossDate:             dateToText(r.LossDate),
		LossLocation:         r.LossLocation,
		Chronology:           r.Chronology,
		DamageDetails:        r.DamageDetails,
		DriverLicense:        r.DriverLicense,
		EstimatedValue:       r.EstimatedValue,
		ClaimType:            r.ClaimType,
		DocumentCount:        r.DocumentCount,
		DocumentReceivedDate: dateToText(r.DocumentReceivedDate),
		ClaimNumber:          r.ClaimNumber,
		Transferred:          r.Transferred,
		TransferredAt:        timeToText(r.TransferredAt),
		RegisteredAt:         timeToText(r.RegisteredAt),
		NotTransferredReason: r.NotTransferredReason,
		NotRegisteredNote:    r.NotRegisteredNote,
		Stage:                string(stage),
		StageLabel:           stage.Label(),
		CanTransfer:          r.CanTransfer(),
		CanEdit:              !r.IsRegistered(),
		BranchCode:           r.BranchCode,
		CreatedBy:            r.CreatedBy,
		CreatedAt:            timeToText(&r.CreatedAt),
		UpdatedAt:            timeToText(&r.UpdatedAt),
	}
}

// toDomain mengubah isian form menjadi entitas domain.
//
// Tanggal yang tidak dapat dibaca menjadi nil, bukan galat: bentuk tanggal diperiksa
// terpisah di handler supaya pesannya dapat menunjuk kolom yang salah.
func toDomain(req SaveRequest) pelaporanklaim.ClaimReport {
	return pelaporanklaim.ClaimReport{
		ReporterName:         req.ReporterName,
		SenderEmail:          req.SenderEmail,
		SenderPhone:          req.SenderPhone,
		CourierName:          req.CourierName,
		EmailSubject:         req.EmailSubject,
		PolicyNumber:         req.PolicyNumber,
		InsuredName:          req.InsuredName,
		InsuredEmail:         req.InsuredEmail,
		BusinessCode:         req.BusinessCode,
		GroupPanel:           req.GroupPanel,
		ReferenceNumber:      req.ReferenceNumber,
		LossDate:             textToDate(req.LossDate),
		LossLocation:         req.LossLocation,
		Chronology:           req.Chronology,
		DamageDetails:        req.DamageDetails,
		DriverLicense:        req.DriverLicense,
		EstimatedValue:       req.EstimatedValue,
		ClaimType:            req.ClaimType,
		DocumentCount:        req.DocumentCount,
		DocumentReceivedDate: textToDate(req.DocumentReceivedDate),
		NotTransferredReason: req.NotTransferredReason,
		NotRegisteredNote:    req.NotRegisteredNote,
	}
}

// dateToText mengubah tanggal kalender menjadi `YYYY-MM-DD`, atau teks kosong bila belum
// diisi.
func dateToText(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(DateFormat)
}

// timeToText mengubah waktu peristiwa menjadi RFC 3339 UTC, atau teks kosong bila belum
// terjadi.
func timeToText(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// textToDate membaca `YYYY-MM-DD` menjadi tanggal, atau nil bila kosong maupun tidak
// terbaca.
//
// Tanggalnya disimpan sebagai tengah malam UTC. Ia tanggal KALENDER — tidak punya jam
// sama sekali — dan memberinya zona apa pun selain satu zona tetap akan membuat nilainya
// bergeser saat dipindahkan antar lingkungan.
func textToDate(text string) *time.Time {
	text = trimSpace(text)
	if text == "" {
		return nil
	}
	t, err := time.ParseInLocation(DateFormat, text, time.UTC)
	if err != nil {
		return nil
	}
	return &t
}
