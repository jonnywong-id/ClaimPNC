package registrasi

import (
	"context"
	"strings"
	"time"
)

// # Persetujuan / Akseptasi (LOD)
//
// Tombol "Persetujuan / Akseptasi" pada grid Adjustment (`Section/ShowAdjustment_sect.xml`)
// membuka form `Section/AcceptationLOD_Sect.xml`; tombol Simpan-nya menjalankan
// `Activity/SetAdjustmentAcceptation-act.xml`. Aturan di berkas ini disalin dari activity itu
// beserta yang dipanggilnya (`SetValidationReceiver`, `CheckAttachmentLOD`,
// `ValidationDLA_Act`), dengan keputusan Work Owner 2026-09-29:
//
//   - nomor akseptasi terbit SAMA SEPERTI PEGA — juga bila Persetujuan Tertanggung = 0;
//     hanya pengisian isian akseptasi dan Status Klaim 1161 yang menuntut persetujuan = 1;
//   - PA (Group Panel 002) menunggu: akseptasinya berlanjut ke Transfer Kasir
//     (`TransferToKasir_act`) yang bergantung pada paket `gl.pkg_pelunasan_kasir` yang tidak
//     ada di export;
//   - Outstanding Acceptance (`OsAkseptasiKlaim`, `HitServiceOSAkseptasiClaimNonMBU`) dan SLIK
//     OJK (`InsertDataSlinkManualyStepF06_1`) tidak dibangun — dicatat sebagai integrasi
//     tertunda.

// Status Klaim sesudah akseptasi LOD — `SetAdjustmentAcceptation` langkah 67.3.1 dan 72.
const StatusClaimAccepted ClaimStatus = "1161"

// AcceptanceCode adalah KODE nomor akseptasi pada `PLA_DLA.prc` TIPE 'ALOD' (langkah 62).
const AcceptanceCode = "A"

// Persetujuan Tertanggung (`.AcceptationStatusLOD`, STATUSAKSEPTASILOD).
const (
	LODAgreed    = "1"
	LODDisagreed = "0"
)

// AcceptanceForm adalah isian form AcceptationLOD_Sect satu baris adjustment.
type AcceptanceForm struct {
	LODStatus          string    // .AcceptationStatusLOD — "Persetujuan Tertanggung"
	Type               string    // .TipeAkseptasi — "Tipe Akseptasi Klaim"
	PrintDate          time.Time // .PrintDateLOD — "Tanggal Cetak"
	ReceiveDate        time.Time // .ReceiveDateLOD — "Tanggal Terima LOD"
	PayableDate        time.Time // .TanggalBolehBayar — "Tanggal Boleh Bayar"
	AnalystReceiveDate time.Time // .ReceiveDateAnalist — "Tanggal Terima LOD" (PA)
	LODValue           Money     // .AcceptanceValueLOD — "Nilai LOD"
	HasLODValue        bool
	ReceiverID         string // .Receiver — IDReceiver penerima klaim
	CommitteeName      string // .KomiteAccepted — "Nama Komite Akseptasi"
	Remark             string // .RemarkAccepted — text area "Penerima Klaim"
	MinutesNote        string // .UploadNoteLOD — "Berita Acara"
}

// Acceptance adalah isian akseptasi yang tersimpan pada baris adjustment.
type Acceptance struct {
	Form         AcceptanceForm
	ReceiverName string    // RECEIVERNAME
	AcceptedAt   time.Time // .AcceptedDate / .AcceptedDateTime — TGLAKSEPTASI

	// LODType adalah `.PDFType` terakhir yang dicetak lewat Print LOD — PDFTYPE. Pega
	// menyimpannya hanya di clipboard; di sini disimpan agar form akseptasi dapat
	// menampilkannya (baca saja) seperti Pega.
	LODType string
}

// AcceptanceDLAState adalah satu baris T_DLALIST adjustment lain yang sudah disetujui LOD —
// bahan `ValidationDLA_Act`.
type AcceptanceDLAState struct {
	Number  string // NODLA
	Printed bool   // ISDLA = '1'
	Sent    bool   // ISKIRIM = '1'
}

// Kode dan pesan pelanggaran akseptasi. Pesan dari Pega disalin apa adanya; pesan
// tambahan berbahasa Inggris (`D-80`).
const (
	ViolationAcceptanceNotAllowed ViolationCode = "akseptasi_tidak_berlaku"
	ViolationAcceptanceRequired   ViolationCode = "akseptasi_wajib"
	ViolationAcceptanceLODValue   ViolationCode = "akseptasi_nilai_lod"
	ViolationAcceptanceReceiver   ViolationCode = "akseptasi_penerima"
	ViolationAcceptanceNumbered   ViolationCode = "akseptasi_sudah_bernomor"
	ViolationAcceptanceAttachment ViolationCode = "akseptasi_lampiran"
	ViolationAcceptanceDLA        ViolationCode = "akseptasi_dla"
	ViolationAcceptancePremium    ViolationCode = "akseptasi_premi"
	ViolationAcceptancePA         ViolationCode = "akseptasi_pa"
)

const (
	msgAcceptanceBlank      = "Value cannot be blank"
	msgAcceptanceLOD        = "Nilai akseptasi lebih kecil dari nilai LOD"
	msgAcceptanceHasNumber  = "Sudah Ada Nomor Akseptasi"
	msgAcceptanceAttachment = "Lampiran Kosong, Harus Upload File"
	msgAcceptancePremium    = "Premi belum lunas, tidak bisa akseptasi adjustment."
	msgAcceptancePA         = "Acceptance for Personal Accident is not available yet: it continues to the cashier transfer (TransferToKasir_act), which depends on package gl.pkg_pelunasan_kasir that is not in the Pega export."
	msgAcceptanceNotAllowed = "Persetujuan / Akseptasi is not available for this adjustment."
)

func acceptanceViolation(code ViolationCode, field, message string) error {
	return &ValidationError{Violation: []Violation{{Code: code, Field: field, Message: message}}}
}

// CanAccept adalah aturan tombol Persetujuan / Akseptasi `ShowAdjustment_sect`: tampil bila
// adjustment disetujui komite, mati bila Persetujuan Tertanggung 0, nomor akseptasi sudah
// terisi, atau komite menolak (`.AcceptanceStatus == 2`). PA ditolak di sini atas keputusan Work Owner.
func CanAccept(line SettlementLine, p Policy) error {
	if strings.TrimSpace(line.AcceptanceStatus) != DecisionApprove ||
		strings.TrimSpace(line.AcceptanceLODStatus) == LODDisagreed {
		return acceptanceViolation(ViolationAcceptanceNotAllowed, "akseptasi", msgAcceptanceNotAllowed)
	}
	// `SetAdjustmentAcceptation` langkah 24: "Sudah Ada Nomor Akseptasi" pada .UploadNoteLOD.
	if strings.TrimSpace(line.AcceptedNo) != "" {
		return acceptanceViolation(ViolationAcceptanceNumbered, "berita_acara", msgAcceptanceHasNumber)
	}
	if p.Line == LinePersonalAccident {
		return acceptanceViolation(ViolationAcceptancePA, "akseptasi", msgAcceptancePA)
	}
	return nil
}

// AcceptanceAttachmentRequired adalah syarat pemanggilan `CheckAttachmentLOD`
// (`SetAdjustmentAcceptation` langkah 28): wajib ada berkas unggahan KECUALI Group Panel
// 002/005, Bonding/BondingKBG/CustomBond, kode bisnis 10145/10168, "ASURANSI KREDIT", atau
// Tipe Pembayaran 3/4/7.
func AcceptanceAttachmentRequired(line SettlementLine, p Policy) bool {
	switch p.Line {
	case LinePersonalAccident, LineTravel:
		return false
	}
	switch strings.TrimSpace(p.BusinessType) {
	case "Bonding", "BondingKBG", "CustomBond":
		return false
	}
	switch strings.TrimSpace(p.BusinessCode) {
	case "10145", "10168":
		return false
	}
	if strings.EqualFold(strings.TrimSpace(p.BusinessName), "ASURANSI KREDIT") {
		return false
	}
	switch strings.TrimSpace(line.PaymentType) {
	case PaymentSalvage, PaymentAdjusterFee, paymentCollectionFee:
		return false
	}
	return true
}

// paymentCollectionFee adalah Tipe Pembayaran 7 ("Collection Fee" pada AcceptanceNotePDF);
// layar ini tidak membentuknya, tetapi aturan Pega menyebutnya.
const paymentCollectionFee = "7"

// AcceptanceComparedValue adalah nilai pembanding Nilai LOD (`SetAdjustmentAcceptation`
// langkah 8): AdjustmentValue (dua desimal) untuk tipe 1/2/5, SalvageValue untuk tipe 3,
// AdjusterFeeValue untuk tipe 4/7. Di modul ini AdjustmentValue adalah Value (bagian ASM)
// dan AdjusterFeeValue adalah Gross baris fee adjuster.
func AcceptanceComparedValue(line SettlementLine) (Money, bool) {
	switch strings.TrimSpace(line.PaymentType) {
	case PaymentFinal, PaymentInterim, PaymentAdjustment:
		return line.Value, true
	case PaymentSalvage:
		return line.SalvageB, true
	case PaymentAdjusterFee, paymentCollectionFee:
		return line.Gross, true
	}
	return 0, false
}

// AcceptanceCheck adalah bahan pemeriksaan Simpan yang dibaca dari luar baris.
type AcceptanceCheck struct {
	Receivers []Receiver
	Files     int // jumlah berkas unggahan "Unggah Dokumen Persetujuan LOD"
	// OtherDLA adalah T_DLALIST adjustment lain klaim ini yang Persetujuan Tertanggung-nya 1.
	OtherDLA []AcceptanceDLAState
	// Location adalah ClaimData.Location — Non-MBU tanpa lokasi ditolak
	// (`AcceptationLOD_PreAct` langkah 17).
	Location string
}

// ValidateAcceptance memeriksa isian form dan aturan Simpan. Pega berhenti pada pesan
// pertama; di sini seluruhnya dikumpulkan (Coding Standards §4.2), urutannya mengikuti Pega.
func ValidateAcceptance(line SettlementLine, p Policy, f AcceptanceForm, c AcceptanceCheck) (Receiver, error) {
	var v collector
	travel := p.Line == LineTravel

	// Isian wajib AcceptationLOD_Sect.
	if s := strings.TrimSpace(f.LODStatus); s != LODAgreed && s != LODDisagreed {
		v.add(ViolationAcceptanceRequired, "persetujuan_tertanggung", msgAcceptanceBlank)
	}
	if !travel {
		if f.ReceiveDate.IsZero() {
			v.add(ViolationAcceptanceRequired, "tanggal_terima_lod", msgAcceptanceBlank)
		}
		if f.PayableDate.IsZero() {
			v.add(ViolationAcceptanceRequired, "tanggal_boleh_bayar", msgAcceptanceBlank)
		}
	}
	if p.Line.IsNonMBU() && !f.HasLODValue {
		v.add(ViolationAcceptanceRequired, "nilai_lod", msgAcceptanceBlank)
	}

	if p.Line.IsNonMBU() && strings.TrimSpace(c.Location) == "" {
		v.add(ViolationAcceptanceLocation, "akseptasi", MsgAcceptanceLocation)
	}

	// Langkah 8 — dilewati untuk Travel.
	if !travel && f.HasLODValue {
		if value, ok := AcceptanceComparedValue(line); ok && value < f.LODValue {
			v.add(ViolationAcceptanceLODValue, "nilai_lod", msgAcceptanceLOD)
		}
	}

	// Langkah 20–21: SetValidationReceiver.
	receiver, found := Receiver{}, false
	for _, r := range c.Receivers {
		if strings.TrimSpace(r.ID) == strings.TrimSpace(f.ReceiverID) && strings.TrimSpace(f.ReceiverID) != "" {
			receiver, found = r, true
			break
		}
	}
	if !found || strings.TrimSpace(receiver.AccountNo) == "" || strings.TrimSpace(receiver.BankName) == "" {
		v.add(ViolationAcceptanceReceiver, "penerima", ReceiverIncompleteMessage(receiver.Name))
	}

	// Langkah 28–30: CheckAttachmentLOD dan ValidationDLA_Act.
	if AcceptanceAttachmentRequired(line, p) && c.Files == 0 {
		v.add(ViolationAcceptanceAttachment, "berita_acara", msgAcceptanceAttachment)
	}
	for _, d := range c.OtherDLA {
		switch {
		case !d.Printed:
			v.add(ViolationAcceptanceDLA, "akseptasi", "No DLA "+d.Number+" belum diprint")
		case !d.Sent:
			v.add(ViolationAcceptanceDLA, "akseptasi", "No DLA "+d.Number+" belum dikirim")
		}
	}
	return receiver, v.err()
}

// ReceiverIncompleteMessage adalah pesan `SetValidationReceiver` bila penerima tidak dipilih
// atau rekening/bank-nya kosong.
func ReceiverIncompleteMessage(name string) string {
	return "Penerima Klaim Atas Nama" + " " + strings.TrimSpace(name) + " " +
		"Data Tidak Lengkap, Silahkan mengisi dahulu dipenerima klaim..."
}

// ErrAcceptancePremiumUnpaid adalah blok premi belum lunas (langkah 48–49).
func ErrAcceptancePremiumUnpaid() error {
	return acceptanceViolation(ViolationAcceptancePremium, "berita_acara", msgAcceptancePremium)
}

// Accept menulis isian akseptasi ke baris (langkah 64 dan 67.3.1). Nomor selalu ditulis;
// isian lainnya hanya bila tertanggung setuju — persis Pega. Persetujuan Tertanggung dan
// isian tanggal/nilai form ikut tersimpan pada keduanya, seperti Obj-Save form Pega.
func (line *SettlementLine) Accept(a Acceptance, number string) {
	line.AcceptedNo = number
	line.AcceptanceLODStatus = strings.TrimSpace(a.Form.LODStatus)
	line.Acceptance = a
}

// AcceptanceSource adalah seam ke data akseptasi di luar baris adjustment.
type AcceptanceSource interface {
	// NextNumber menerbitkan nomor akseptasi (ACCEPTLOD_SEQ + baris POOLDATA.ACCEPTLOD).
	// Di dalam UnitOfWork.
	NextNumber(ctx context.Context, year int) (string, error)
	// Save menulis isian akseptasi baris (kolom yang ada + kolom migrasi 0013). Di dalam
	// UnitOfWork, sesudah ClaimRepo.Save.
	Save(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, line SettlementLine) error
	// RecordLODPrint mencatat Print LOD: PRINTLOD_DATE diisi hanya bila masih kosong
	// (`AutoPrintPDFDraftLOD` langkah 1: `.PrintDateLOD == ""` → CurrentDateTime), PDFTYPE
	// diisi jenis yang dicetak.
	RecordLODPrint(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, printedAt time.Time, lodType string) error
	// SetLODType menyimpan PDFTYPE baris — dropdown Tipe LOD kolom Adjustment.
	SetLODType(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, lodType string) error
	// Fields membaca tujuh isian migrasi 0013 ke baris.
	Fields(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, line *SettlementLine) error
	// OtherDLA membaca T_DLALIST adjustment lain klaim itu yang Persetujuan Tertanggung-nya 1.
	OtherDLA(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int) ([]AcceptanceDLAState, error)
	// AddHistory menulis LIST_HISTORY_CLAIM_PNC (`PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc`).
	AddHistory(ctx context.Context, caseID, note, user string, at time.Time) error
	// OpenPosition membaca ID posisi progres terbuka ("On Progress") terakhir bernama position
	// (GCNM_PROGRESS_POSISI_PNC.POSISI) — padanan `.PosisiProgressID` adjustment (AKSEPTASI)
	// dan `.Komite.PosisiProgressID` (KOMITE). Kosong bila tidak ada.
	OpenPosition(ctx context.Context, claimNumber, position string) (string, error)
	// StartProgress menjalankan PNCInsertProgressClaim kategori INSERT (PROGRESS_CLAIM_PNC.prc):
	// membuka posisi baru lalu menulis baris progresnya; mengembalikan ID posisi (idposisi).
	StartProgress(ctx context.Context, p ProgressStart) (string, error)
	// AddProgress menulis GCNM_PROGRESS_CLAIM kategori UPDATE (`PROGRESS_CLAIM_PNC.prc`).
	AddProgress(ctx context.Context, p ProgressUpdate) error

	// PolicyCaseID membaca Policy.CaseID dokumen polis (parameter caseId layanan premi).
	PolicyCaseID(ctx context.Context, policyNumber, prodKe string) (string, error)
	// OpenProtectionApproved: ada Open Protection premi (TypePro 2) yang disetujui.
	OpenProtectionApproved(ctx context.Context, policyNumber, claimID, claimNumber string) (bool, error)
	// TravelClientName membaca CLIENTNAME agen leader polis (`BrowseClientNameTravel_SQL`).
	TravelClientName(ctx context.Context, policyNumber string) (string, error)
}

// ProgressUpdate adalah satu pemanggilan `PNCInsertProgressClaim` kategori UPDATE.
type ProgressUpdate struct {
	ClaimNumber string // Param.Claimno — PNCCASEID
	PositionID  string // Param.IDCase — adjustment .PosisiProgressID
	Note        string
	Progress1   string
	Progress2   string
	Position    string // statusposisi
	User        string
	At          time.Time // PROGRESSDATEDONE posisi
}

// ProgressStart adalah satu pemanggilan `PNCInsertProgressClaim` kategori INSERT.
type ProgressStart struct {
	ClaimNumber string // Param.Claimno — CLAIMNO posisi dan PNCCASEID progres
	CaseID      string // Param.IDCase — CASEID posisi
	Position    string // Param.Posisi — POSISI, mis. "AKSEPTASI"
	Note        string
	Progress1   string
	Progress2   string
	User        string
	At          time.Time // PROGRESSDATE; NEXT_FOLLOWUP = At + 7 hari (tglfollow kosong)
}

// Posisi progres dan progres putusan komite — `KomitePost_Adjustment` langkah 19–23
// (setuju di jenjang terakhir) dan 61–62 (tolak).
const (
	PositionAcceptance  = "AKSEPTASI"
	PositionCommittee   = "KOMITE"
	ProgressOnProgress  = "On Progress"
	CommitteeAcceptNote = "Auto Create AKSEPTASI" // 014 / 60
	CommitteeFinishNote = "Auto Finish KOMITE"    // 006 / 24, Done
	CommitteeRejectNote = "Auto Reject KOMITE"    // 006 / 24, Done
	CommitteeProgress1  = "006"
	CommitteeProgress2  = "24"
)

// Progres akseptasi — `SetAdjustmentAcceptation` langkah 69–70 dan 95–96.
const (
	AcceptanceProgress1     = "014"
	AcceptanceProgress2     = "60"
	AcceptanceProgressDone  = "Done"
	AcceptanceProgressNote1 = "Auto FInish Akseptasi"
	AcceptanceProgressNote2 = "Auto Akseptasi By LOD"
)

// AcceptanceHistoryNote adalah catatan riwayat langkah 68.
func AcceptanceHistoryNote(number string) string { return "Claim Accepted with No " + number }
