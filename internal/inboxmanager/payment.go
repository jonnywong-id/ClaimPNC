package inboxmanager

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Panel detail Payment Klaim Akseptasi — tombol DETAILS dan SIMPAN.
//
// Asal: `Section/Akseptasi_PaymentLeader1` (disertakan `Sec_PaymentAkseptasiKlaimCase1`, sub-tab
// "Approval Payment Akseptasi"). Tombol DETAILS menjalankan `Activity/GetDataAkseptasiKlaimKasir`,
// SIMPAN menjalankan `Activity/SaveApprovalAkseptasiPaymentLeader_Act`.
//
// # Sumbernya tabel POOLDATA, bukan case Pega (keputusan Work Owner 2026-10-14)
//
// Di Pega, DETAILS membuka case klaim (`Obj-Open-By-Handle pyWorkPage`) dan menyalin adjustment
// yang `AcceptedNo`-nya cocok. Isi case itu BLOB yang tidak terbaca aplikasi ini; padanannya
// ditulis Pega sendiri ke tabel lewat `PEGA_CONVERT_JSONKLAIM_PNC.prc` (baris 1065–1086):
//
//	Nama Tertanggung        pyWorkPage.Policy.QQName           T_CLAIMLIST_ADMIN.QQNAME
//	No Akseptasi            .AcceptedNo                        T_CLAIM_ADJUSTMENT.NOAKSEPTASI
//	Penerima Klaim          .AcceptanceDocument.PayableTo      T_CLAIM_ADJUSTMENT.RECEIVERNAME (*)
//	Tipe Pembayaran         .PaymentType                       T_CLAIM_ADJUSTMENT.PAYMENTTYPE
//	Gross Value             .GrossValue                        T_CLAIM_ADJUSTMENT.GROSSVALUE
//	Nilai Klaim (ASM Share) .AdjustmentValue                   T_CLAIM_ADJUSTMENT.ASM_SHARE_VALUE
//	Mata Uang               .Currency (BrowseCurrency_RD)      T_CLAIM_ADJUSTMENT.CURRENCY -> POOLDATA.CURRENCY
//	Share ASM (%)           .ShareASM                          T_CLAIM_ADJUSTMENT.ASM_SHARE
//	Remark Akseptasi        .RemarkAcceptedLeader              T_CLAIM_AKSEPTASI_CHECKER.PICNOTES
//
// (*) BELUM DIVERIFIKASI terhadap layar Pega: `PayableTo` tidak dipetakan prosedur itu;
// `RECEIVERNAME` adalah nama penerima adjustment yang sama. Pemiliknya Work Owner.
//
// "Remark Akseptasi" diambil dari tabel checker karena di sanalah catatan PIC tersimpan sejak
// diajukan: pada 2026-10-14 kolomnya terisi pada 8 dari 8 baris yang menunggu.

// Pilihan "Status Approval" — Prompt List `ASM-FW-GCNMFW-Data-Adjustment.AcceptanceStatus`
// (RS GCNMFW:01-03-06).
//
// Rule properti itu TIDAK ada di export; daftarnya dibaca dari layar rule Pega yang dikirim
// Work Owner 2026-10-14. Kodenya sama dengan isi `T_CLAIM_AKSEPTASI_CHECKER.STSAPP` (0, 1, 2).
const (
	PaymentInCommittee = "0" // Sedang Proses Komite
	PaymentAccepted    = "1" // Setuju
	PaymentNotAccepted = "2" // Tidak Setuju
)

// PaymentStatusOptions adalah ketiga pilihan dropdown, berurutan seperti di Pega.
var PaymentStatusOptions = []FilterOption{
	{Value: PaymentInCommittee, Label: "Sedang Proses Komite"},
	{Value: PaymentAccepted, Label: "Setuju"},
	{Value: PaymentNotAccepted, Label: "Tidak Setuju"},
}

// PaymentStatusLabel menamai kode status. Kode yang tidak dikenal ditulis apa adanya.
func PaymentStatusLabel(code string) string {
	clean := strings.TrimSpace(code)
	for _, option := range PaymentStatusOptions {
		if option.Value == clean {
			return option.Label
		}
	}
	return clean
}

// ErrPaymentNotFound berarti nomor akseptasi itu tidak sedang menunggu persetujuan.
var ErrPaymentNotFound = errors.New("inboxmanager: nomor akseptasi tidak menunggu persetujuan")

// PaymentDetail adalah isi panel "Detail Penerima Akseptasi" beserta riwayat dan lampirannya.
type PaymentDetail struct {
	AcceptedNo string
	ClaimID    string
	ClaimNo    string
	PIC        string

	InsuredName     string
	Receiver        string
	PaymentTypeCode string
	PaymentType     string
	GrossValue      string // teks presisi penuh, tidak melewati float (`I-12`)
	ClaimValue      string // Nilai Klaim (ASM Share)
	Currency        string
	ShareASM        string
	RemarkPIC       string

	// AdjustmentFound menyatakan baris adjustment-nya ditemukan. Bila tidak, isian dari
	// T_CLAIM_ADJUSTMENT kosong — dan layar menyatakannya, bukan menampilkan nol.
	AdjustmentFound bool

	// PICEmail adalah alamat PIC dari MST_USER_TEKNIK — penerima surel SIMPAN.
	PICEmail string

	History   []PaymentHistoryRow
	Documents []PaymentDocument
}

// PaymentHistoryRow adalah satu baris grid "History Payment Akseptasi Manager".
//
// Kueri aslinya (`HistoryManagerPaymetApproval_SQL`) TIDAK ada di export. Penggantinya baris
// T_CLAIM_AKSEPTASI_CHECKER untuk klaim yang sama yang SUDAH diputuskan — keputusan Work Owner
// 2026-10-14; BELUM DIVERIFIKASI terhadap layar Pega. Kolomnya mengikuti section: No Klaim,
// No Akseptasi, Tanggal, Note Manager, Note PIC, Status.
type PaymentHistoryRow struct {
	ClaimNo     string
	AcceptedNo  string
	Date        *time.Time
	NoteManager string
	NotePIC     string
	StatusCode  string
	Status      string
}

// PaymentDocument adalah satu lampiran tombol "Dokumen" — `POOLDATA.DATA_ATTACHFILE` dengan
// `DATAID` = nomor akseptasi (`GetAttachmentFromDB2_Sql`, `GetOldAttachmentFromDB2_Sql`).
//
// Lampiran yang di Pega diambil dari case (DLA dan Draft Persetujuan) TIDAK ikut — keputusan
// Work Owner 2026-10-14.
type PaymentDocument struct {
	// Key mengenali lampiran di dalam daftar nomor akseptasi ini. Ia OPAK bagi layar.
	Key       string
	Name      string
	MimeType  string
	Note      string
	InputDate *time.Time

	// ImageID terisi pada lampiran yang tersimpan di layanan penyimpanan; kosong pada
	// lampiran lama yang berkasnya tersimpan langsung di kolom ATTACHFILE.
	ImageID string

	// HasFile menyatakan kolom ATTACHFILE berisi — lampiran lama yang dapat diunduh langsung.
	HasFile bool
}

// Linked menyatakan lampiran ini dibuka lewat tautan layanan penyimpanan.
func (d PaymentDocument) Linked() bool { return strings.TrimSpace(d.ImageID) != "" }

// Openable menyatakan lampiran ini punya isi yang dapat dibuka. Baris tanpa IMAGEID dan tanpa
// berkas tetap ditampilkan — Pega pun mendaftarkannya — tetapi tidak dapat dibuka.
func (d PaymentDocument) Openable() bool { return d.Linked() || d.HasFile }

// PaymentDocumentKey adalah kunci lampiran ke-n (mulai 1) di daftar satu nomor akseptasi.
//
// Kuncinya nomor urut, bukan IMAGEID: lampiran lama tidak punya IMAGEID, dan nomor urut di
// bawah satu nomor akseptasi tidak dapat menunjuk lampiran nomor lain.
func PaymentDocumentKey(n int) string { return strconv.Itoa(n) }

// PaymentDocumentIndex membaca kunci lampiran menjadi nomor urutnya (mulai 1). Kunci yang
// bukan bilangan bulat positif mengembalikan false.
func PaymentDocumentIndex(key string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(key))
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// ErrPaymentDocumentNotFound berarti kunci itu tidak menunjuk lampiran yang dapat dibuka.
var ErrPaymentDocumentNotFound = errors.New("inboxmanager: lampiran akseptasi tidak ditemukan")

// ErrDocumentLinkUnavailable berarti alamat lampiran di layanan penyimpanan tidak dapat dibaca
// atau diperpanjang — layanannya belum terpasang, atau gagal dihubungi.
var ErrDocumentLinkUnavailable = errors.New("inboxmanager: tautan lampiran tidak tersedia")

// PaymentDocumentContent adalah isi satu lampiran lama.
type PaymentDocumentContent struct {
	Name     string
	MimeType string
	Content  []byte
}

// PaymentSave adalah permintaan SIMPAN yang sudah tervalidasi.
type PaymentSave struct {
	AcceptedNo string
	Status     string
	Remark     string
	Caller     Caller
}

// AdjustmentFlag adalah nilai `.FlagAtasan` yang ditulis SIMPAN:
// `@if(Param.stsapprove==1,"","1")` (`SaveApprovalAkseptasiPaymentLeader_Act` :1639-1641).
func (s PaymentSave) AdjustmentFlag() string {
	if s.Status == PaymentAccepted {
		return ""
	}
	return "1"
}

// NewPaymentSave memeriksa permintaan SIMPAN.
//
// # Aturannya
//
//   - Setuju (1) DITAHAN: di Pega ia menjalankan transfer ke kasir, dan rantai itu belum
//     lengkap — alasan yang sama dengan jalur setuju pada tab ini (lihat tab.go).
//   - Tidak Setuju (2) menuntut Remark Approval — aturan tab ini yang sudah berlaku
//     (`ReasonRequiredOnReject`), lebih ketat daripada Pega.
//   - Sedang Proses Komite (0) boleh tanpa catatan; barisnya TETAP menunggu, persis Pega
//     yang menulis `STSAPP = 0` dan karena itu tidak mengeluarkannya dari antrean.
func NewPaymentSave(acceptedNo, status, remark string, caller Caller) (PaymentSave, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return PaymentSave{}, ErrCallerUnknown
	}

	tab, _ := FindTab(TabPaymentAkseptasi)
	if !CanSee(clean, tab) {
		return PaymentSave{}, ErrTabNotAllowed
	}

	cleanStatus := strings.TrimSpace(status)
	if cleanStatus == PaymentAccepted && tab.Decision.ApproveBlockedReason != "" {
		return PaymentSave{}, ErrApproveBlocked
	}

	violations := []Violation{}
	if strings.TrimSpace(acceptedNo) == "" {
		violations = append(violations, Violation{
			Field: FieldKeys, Message: "Nomor akseptasi wajib diisi.",
		})
	}

	known := false
	for _, option := range PaymentStatusOptions {
		if option.Value == cleanStatus {
			known = true
		}
	}
	if !known {
		violations = append(violations, Violation{
			Field: FieldVerdict, Message: "Status Approval tidak dikenal.",
		})
	}

	cleanRemark := strings.TrimSpace(remark)
	if cleanStatus == PaymentNotAccepted && tab.Decision.ReasonRequiredOnReject && cleanRemark == "" {
		violations = append(violations, Violation{
			Field: FieldReason, Message: "Alasan penolakan wajib diisi.",
		})
	}

	if len(violations) > 0 {
		return PaymentSave{}, NewValidationError(violations)
	}

	return PaymentSave{
		AcceptedNo: strings.TrimSpace(acceptedNo),
		Status:     cleanStatus,
		Remark:     cleanRemark,
		Caller:     clean,
	}, nil
}

// PaymentSaveResult adalah hasil SIMPAN.
type PaymentSaveResult struct {
	// Changed adalah jumlah baris checker yang berubah. Nol berarti nomor akseptasi itu sudah
	// diputuskan orang lain sementara layar masih yang lama.
	Changed int

	// AdjustmentMarked menyatakan penanda adjustment (FLAGATASAN, STSPENERIMAKLAIM) tertulis.
	// False bila kolomnya belum dibuat DBA (migrasi 0017) atau adjustment-nya tidak ditemukan.
	AdjustmentMarked bool

	// Notified menyatakan surel ke PIC terkirim.
	Notified bool
}

// PaymentNotice adalah bahan surel SIMPAN.
//
// Subjeknya disalin dari activity (:2806): "Pengecekan akseptasi CLAIM <no klaim> | <nama
// tertanggung> | No Aksep <no akseptasi>". Penerimanya hanya PIC — alamat CC di activity tidak
// pernah terkirim karena parameter yang dipakai (`Local.bcemail`) tidak pernah diisi.
type PaymentNotice struct {
	To          string
	ClaimNo     string
	InsuredName string
	AcceptedNo  string
	Status      string
	Remark      string
	DecidedBy   string
}

// Subject adalah baris subjek surel, apa adanya dari Pega.
func (n PaymentNotice) Subject() string {
	return "Pengecekan akseptasi CLAIM " + n.ClaimNo + " | " + n.InsuredName + " | No Aksep " + n.AcceptedNo
}

// PaymentNotifier mengirim surel SIMPAN — padanan `UploadingFileUntukSendByEmail`.
type PaymentNotifier interface {
	NotifyPaymentChecked(ctx context.Context, notice PaymentNotice) error
}

// DocumentLinker membuka lampiran yang tersimpan di layanan penyimpanan: membaca alamatnya, dan
// memperpanjangnya bila kedaluwarsa. Dipenuhi modul Dokumen Penunjang lewat cmd.
type DocumentLinker interface {
	Link(ctx context.Context, portal, imageID, by string) (string, error)
}

// PaymentRules adalah aturan milik modul lain yang dipakai panel ini, dirangkai di cmd.
type PaymentRules struct {
	// PaymentTypeName menamai kode Tipe Pembayaran — `registrasi.PaymentTypeName`.
	PaymentTypeName func(code string) string
}
