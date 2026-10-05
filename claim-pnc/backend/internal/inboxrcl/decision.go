package inboxrcl

// # Keputusan dokter RCL — `Activity/SendToPUCL-Act.xml`
//
// Keempat tombol layar kerja memanggil activity yang SAMA dengan parameter `StatusRCL`
// berbeda:
//
//	mode RCL  (RCL_PUCL 1)  Setuju       -> "SETUJU"       (Section/RCLDokter-Section.xml)
//	                        Tidak Setuju -> layar Alasan Dokter -> Kirim -> "TidakSetuju"
//	mode MSIG (RCL_PUCL 3)  Submit       -> "MSIG"         (Section/RCLDokter-Section.xml)
//	                        Back         -> layar Alasan Dokter -> Kirim -> "BackMSIG"
//
// Layar Alasan Dokter adalah `Section/sendToAnalystTolakRCL_sect-Section.xml` (Flow Action
// lokal `sendToAnalystRCL`); isiannya `.ClaimData.AlasanDokterRejectRCL`, TIDAK wajib diisi
// (`pyRequired false`).
//
// # Langkah yang dibawa, dan yang sengaja tidak
//
// Seluruh langkah `SendToPUCL` yang mengubah kolom `TC_PNC_PUCL` dibawa apa adanya —
// lihat Plan. Tiga yang TIDAK dibawa, karena di Pega memang tidak pernah terjadi
// (keputusan Work Owner 2026-10-05, "ikuti perilaku nyata Pega"):
//
//	langkah  1  RCLSendKomiteReject_act — case Komite penolakan, PDF "Reject Claim.pdf", email
//	            ke komite. Tidak satu pun lampiran berkategori RejectClaim pernah ada di Pega,
//	            dan dari 44 riwayat "Send to Dokter" (2020–2026) tidak satu pun diikuti case
//	            Komite. Sebabnya terbaca di rule: PDF dan email bersyarat pada halaman
//	            `TempCommiteClaim`, yang hanya diisi activity komite. Yang DIBAWA dari activity
//	            itu hanya langkah 1-nya: `PUCLStatus.StatusKlaim := "AKTIF"` (StatusClaim 1142
//	            pada langkah yang sama tertimpa langkah 8/10/16 sesudahnya).
//	langkah 15  SuratRCLSementaraPA_Act — surat "RCL.pdf" dan tanggal cetak. Hanya 8 lampiran
//	            RCL sepanjang sejarah Pega, tidak satu pun dibuat saat tombol Dokter ditekan;
//	            klaim yang disetujui Dokter justru dicetak suratnya belakangan oleh petugas
//	            RCL/PUCL. Menyetel tanggal cetak di sini akan melompati tab "Cetak Surat"
//	            Inbox RCL/PUCL.
//	langkah 23  InsertJsonClaimNonMBU_act — JSON_KLAIM tidak ditulis aplikasi ini (`D-02`).
//
// `ClaimData.isComplianceTransfer` (langkah 10) tidak punya kolom di TC_PNC_PUCL dan tidak
// ditulis. Langkah 13 (`JumlahTagihan`, `NamaPeserta`) hanya mengisi properti clipboard yang
// tidak disimpan di tabel mana pun yang dibaca modul ini.
//
// # Dua perbandingan yang tidak pernah cocok — dibawa apa adanya
//
// Langkah 4 dan 14 membandingkan `param.StatusRCL == "Setuju"` (huruf kecil), padahal tombol
// mengirim `"SETUJU"`. Akibatnya kedua langkah itu selalu berlanjut ke syarat keduanya
// (`RCL_PUCL == 2`), yang tidak pernah benar pada antrean ini. Keduanya karena itu tidak
// berpengaruh, dan Plan tidak memuatnya.

import (
	"strings"
	"time"
)

// Decision adalah nilai `param.StatusRCL` yang dikirim tombol.
type Decision string

// Keempat keputusan dokter RCL, nilainya apa adanya dari kedua section.
const (
	DecisionApprove  Decision = "SETUJU"      // mode RCL, tombol Setuju
	DecisionSubmit   Decision = "MSIG"        // mode MSIG, tombol Submit
	DecisionDisagree Decision = "TidakSetuju" // mode RCL, Tidak Setuju -> Kirim
	DecisionBackMSIG Decision = "BackMSIG"    // mode MSIG, Back -> Kirim
)

// ParseDecision mengenali nilai keputusan; selebihnya false.
func ParseDecision(raw string) (Decision, bool) {
	switch d := Decision(strings.TrimSpace(raw)); d {
	case DecisionApprove, DecisionSubmit, DecisionDisagree, DecisionBackMSIG:
		return d, true
	}
	return "", false
}

// AllowedFor menyatakan apakah tombol itu memang ada pada mode layar. Section menggambar
// Setuju/Tidak Setuju hanya pada mode RCL, Back/Submit hanya pada mode MSIG.
func (d Decision) AllowedFor(m Mode) bool {
	switch d {
	case DecisionApprove, DecisionDisagree:
		return m == ModeRCL
	case DecisionSubmit, DecisionBackMSIG:
		return m == ModeMSIG
	}
	return false
}

// ReturnsToAnalyst menyatakan keputusan yang mengembalikan klaim ke analis (langkah 16 dan
// 21) — kedua keputusan yang melewati layar Alasan Dokter.
func (d Decision) ReturnsToAnalyst() bool {
	return d == DecisionDisagree || d == DecisionBackMSIG
}

// Nilai yang ditulis `SendToPUCL`.
const (
	StatusClaimRCLDoctor = "1155" // langkah 8  — posisi RCL Dokter
	StatusClaimRCL       = "1153" // langkah 10 — aktif RCL
	StatusClaimAnalyst   = "1151" // langkah 16 — kembali ke Analyst

	StatusKlaimActive = "AKTIF" // RCLSendKomiteReject_act langkah 1 · SendToPUCL langkah 10
	StatusCaseOpen    = "0"     // langkah 5, 7, 10
	PUCLApproveYes    = "1"
	PUCLApproveNo     = "0" // langkah 6, 7
)

// Catatan riwayat yang ditulis `InsertHistoryClaimPNC` (langkah 18 dan 19).
const (
	HistorySentToDoctor = "Send to Dokter"
	HistorySentToMSIG   = "Send to inbox MSIG/RCL"
)

// Tujuan Ticket rule (langkah 12 dan 21) di alur Register aplikasi ini.
const (
	TicketSendToPUCL    = "SendtoPUCL"
	TicketSendToAnalyst = "SendtoAnalysator"

	StageRCLPUCL       = "rcl-pucl"     // Assignment6 "RCL/PUCL", antrean bersama
	StageSendToAnalyst = "kirim-analis" // Assignment5 "Send To Analis", worklist PIC Teknik

	StageNameRCLPUCL       = "RCL/PUCL"
	StageNameSendToAnalyst = "Send To Analis"

	QueueWorkbasket = "WORKBASKET"
	QueueWorklist   = "WORKLIST"

	// WorkbasketRCLPUCL adalah akun antrean bersama tahap RCL/PUCL — nilai
	// `pxAssignedOperatorID` penugasan workbasket itu (`RDB List/CountKlaimPUCL-SQL.xml`).
	// Akun fungsional, bukan nama orang.
	WorkbasketRCLPUCL = "RCLPUCL"
)

// PUCLState adalah kolom TC_PNC_PUCL yang DIBACA `SendToPUCL` sebelum menulis.
type PUCLState struct {
	Mode         Mode
	StatusCase   string
	PUCLApprove  string
	StatusKlaim  string
	DischargedAt time.Time // TGL_SELESAI_RI — `TanggalSelesaiRawatInap`
	DateOfLoss   time.Time // DATE_OF_LOSS

	LetterPrintedAt time.Time // TGL_CETAK_DOKUMEN_PUCL
	DoctorReason    string    // ALASAN_DOKTER_REJECT_RCL
}

// Outcome adalah keadaan TC_PNC_PUCL SESUDAH keputusan, beserta perpindahan tahapnya.
//
// Teks kosong dan waktu nol berarti NULL — sama dengan Pega, yang menyimpan "" sebagai NULL.
type Outcome struct {
	StatusClaim     string
	StatusKlaim     string
	StatusCase      string
	PUCLApprove     string
	LetterPrintedAt time.Time
	SentToPUCLAt    time.Time // TGL_KIRIM_PUCL
	AnalystSentAt   time.Time // TGL_ANALYST_SEND_RCL
	ClaimAgeFrom    time.Time // LAMA_KLAIM
	DoctorReason    string

	// Ticket dan NextStage adalah tujuan lompatan. NextStageName adalah PXTASKLABEL-nya.
	Ticket        string
	NextStage     string
	NextStageName string
	NextQueue     string

	// History adalah catatan riwayat, dalam urutan ditulis.
	History []string
}

// Plan menjalankan langkah-langkah `SendToPUCL` atas keadaan sekarang.
//
// Urutannya mengikat: beberapa langkah membaca nilai yang ditulis langkah sebelumnya
// (langkah 7 membaca PUCLApprove sesudah langkah 6, langkah 10 menimpa LamaKlaim langkah 2/3).
func Plan(state PUCLState, d Decision, doctorReason string, now time.Time) (Outcome, error) {
	if !d.AllowedFor(state.Mode) {
		return Outcome{}, ErrDecisionNotAllowed
	}
	now = now.UTC()

	out := Outcome{
		StatusClaim:     "",
		StatusKlaim:     strings.TrimSpace(state.StatusKlaim),
		StatusCase:      strings.TrimSpace(state.StatusCase),
		PUCLApprove:     strings.TrimSpace(state.PUCLApprove),
		LetterPrintedAt: state.LetterPrintedAt,
		DoctorReason:    state.DoctorReason,
	}

	// RCLSendKomiteReject_act langkah 1 — dipanggil SendToPUCL langkah 1 bila RCL_PUCL == 1,
	// tombol apa pun.
	if state.Mode == ModeRCL {
		out.StatusKlaim = StatusKlaimActive
	}

	// Langkah 2 dan 3 — LamaKlaim dari tanggal selesai rawat inap, atau tanggal kejadian.
	if !state.DischargedAt.IsZero() {
		out.ClaimAgeFrom = state.DischargedAt
	} else {
		out.ClaimAgeFrom = state.DateOfLoss
	}

	// Langkah 5 — StatusCase "0" hanya bila belum berisi.
	if out.StatusCase == "" {
		out.StatusCase = StatusCaseOpen
	}

	// Langkah 6 dan 7 — kirim ulang setelah pernah disetujui PUCL.
	if out.PUCLApprove == PUCLApproveYes && out.StatusCase == "1" {
		out.PUCLApprove = PUCLApproveNo
	}
	if out.PUCLApprove == PUCLApproveYes && out.StatusCase != "1" {
		out.PUCLApprove = PUCLApproveNo
		out.StatusCase = StatusCaseOpen
		out.LetterPrintedAt = time.Time{}
	}

	// Langkah 8 — posisi RCL Dokter (RCL dan MSIG, seluruh tombol).
	out.StatusClaim = StatusClaimRCLDoctor
	out.AnalystSentAt = now
	out.SentToPUCLAt = now

	// Langkah 10 — Setuju pada mode RCL.
	if d == DecisionApprove {
		out.StatusKlaim = StatusKlaimActive
		out.SentToPUCLAt = now
		out.ClaimAgeFrom = now
		out.StatusCase = StatusCaseOpen
		out.LetterPrintedAt = time.Time{}
		out.PUCLApprove = ""
		out.StatusClaim = StatusClaimRCL
	}

	// Langkah 16 — Tidak Setuju / Back: kembali ke analis. Alasan Dokter diisi layar
	// sendToAnalystTolakRCL_sect sebelum activity dijalankan.
	if d.ReturnsToAnalyst() {
		out.AnalystSentAt = time.Time{}
		out.StatusCase = ""
		out.StatusClaim = StatusClaimAnalyst
		out.DoctorReason = strings.TrimSpace(doctorReason)
	}

	// Langkah 18 dan 19 — riwayat.
	out.History = []string{HistorySentToDoctor}
	if d == DecisionSubmit || d == DecisionDisagree {
		out.History = append(out.History, HistorySentToMSIG)
	}

	// Langkah 12, 20, 21 — tiket terakhir yang dipasang menentukan tujuan. Pada Tidak
	// Setuju/Back langkah 20 memasang RCLDokter lalu langkah 21 menimpanya dengan
	// SendtoAnalysator.
	if d.ReturnsToAnalyst() {
		out.Ticket = TicketSendToAnalyst
		out.NextStage = StageSendToAnalyst
		out.NextStageName = StageNameSendToAnalyst
		out.NextQueue = QueueWorklist
	} else {
		out.Ticket = TicketSendToPUCL
		out.NextStage = StageRCLPUCL
		out.NextStageName = StageNameRCLPUCL
		out.NextQueue = QueueWorkbasket
	}

	return out, nil
}

// legacyWorkKeyPrefix adalah awalan kelas Pega pada kunci klaim warisan (`D-22`).
const legacyWorkKeyPrefix = "ASM-FW-GCNMFW-WORK "

// HistoryKey adalah nilai `LIST_HISTORY_CLAIM_PNC.CASEID` sebuah klaim.
//
// Pega menulis `pzInsKey` (`ASM-FW-GCNMFW-WORK PNC-xxxx`). Klaim yang dibuka aplikasi ini
// (`PNCN.YY.xxxx`) tidak punya kunci Pega dan riwayatnya ditulis dengan nomornya apa adanya —
// sama dengan baris yang sudah ditulis modul Registrasi.
func HistoryKey(claimNumber string) string {
	number := strings.TrimSpace(claimNumber)
	if strings.HasPrefix(strings.ToUpper(number), "PNC-") {
		return legacyWorkKeyPrefix + number
	}
	return number
}
