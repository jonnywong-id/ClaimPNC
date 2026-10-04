package registrasi

import (
	"context"
	"strings"
	"time"
)

// # Tombol "Tutup Klaim"
//
// `Section/ClaimSurvey_sect.xml` → local action `PreventRejectClaim` ("Prevent Close Claim",
// `Section/PreventRejectClaim-sect.xml`) → tombol Ya menjalankan `Activity/CloseClaim-act.xml`.
//
// Yang dibawa (cakupan "inti", Work Owner 2026-10-04):
//   - pemeriksaan langkah 3 (salvage TBA), 8 (catatan), 11 (PIC), 12 (`ValidationAdjustmentKomite`),
//     13 (`ValidationDLA_Act`: DLA belum diprint / belum dikirim);
//   - tutup sementara (`SetStatusCloseSementara`): penanda ISPENDINGCLOSE dan log, tugas tetap terbuka;
//   - tutup permanen (langkah 15–19, 27): Status Klaim 1143, alur selesai, tanggal tutup, tugas
//     ditutup (`ASMForceCaseClose`), TOTAL_JOB PIC dikurangi (`UpdateTotalJob_sql`), dashboard
//     STSKLAIM = 3 (`UpdateStsKlaimClose_sql`), log `CloseRejectClaim_Log` (`Insert_to_log_SQL`).
//
// Yang TIDAK dibawa: langkah 4 (`Param.ERRMSG` — tidak ada yang mengisinya), langkah 5–7 (feedback
// AI — data CONF_SCORE/SendFeedbackAI tidak ada di domain), `IsTransferPIC` pada langkah 11 (tidak
// berkolom), pengecualian tiga klaim Pega yang tertulis tetap di `ValidationAdjustmentKomite`,
// penulisan OS akseptasi di langkah yang sama, serta integrasi ServiceCloseClaimNonMBU,
// InsertJsonClaimNonMBU_act, RunConvertJSONKLAIM, dan email pelapor (ditunda).

// StatusClosed adalah Status Klaim tutup — "Close Claim for this object" (CloseClaim langkah 15).
const StatusClosed ClaimStatus = "1143"

// ActionCloseClaim adalah local action tombol Tutup Klaim, dicatat sebagai tindakan penutup tugas.
const ActionCloseClaim = "PreventRejectClaim"

// SalvageStatusTBA adalah STSSALVAGE yang menahan penutupan (CloseClaim langkah 3).
const SalvageStatusTBA = "5"

// Isi kolom ACTION CloseRejectClaim_Log (`SetStatusCloseSementara` langkah 2).
const (
	ClosureActionClose     = "Close"
	ClosureActionTemporary = "Temp Close"
)

// Panjang kolom T_CLAIM_PNC.
const (
	ClosureNoteMax  = 4000 // CLOSECLAIMNOTE
	ClosureShortMax = 100  // USULAN, EFFORT_CLOSE, KENDALA_CLOSE
)

// Pesan Pega, disalin apa adanya.
const (
	MsgClosureSalvageTBA = "Tidak Bisa Close Status Salvage TBA"
	MsgClosureNote       = "Catatan harus di isi"
	MsgClosureNotPIC     = "Error Close Harus Di PIC"
	MsgClosureCommittee  = "Tidak Bisa Tutup Klaim, karena ada adjustment yang masih di komite."
)

// ViolationClosure menolak penutupan klaim.
const ViolationClosure ViolationCode = "tutup_klaim_ditolak"

// Closure adalah isian dialog Prevent Close Claim yang punya kolom di T_CLAIM_PNC.
//
// Survey Kepuasan, Manual/Paperless, No Reff Broker, Banding, dan Alasan Keterlambatan TIDAK ada
// di sini: kelimanya tanpa kolom, dan keempat dropdown-nya tanpa daftar pilihan di export.
type Closure struct {
	Note      string // CLOSECLAIMNOTE — Catatan (wajib)
	Proposal  string // USULAN — Usulan
	Effort    string // EFFORT_CLOSE — Effort Sebelum Close
	Obstacle  string // KENDALA_CLOSE — Kendala Sebelum Close
	Temporary bool   // ISPENDINGCLOSE — kotak tutup sementara (`.IsPendingClose`)
}

// ClosureLog adalah satu baris POOLDATA.CloseRejectClaim_Log (`Insert_to_log_SQL`).
type ClosureLog struct {
	CaseKey     string // CASEID — pzInsKey
	ClaimNumber string // NO_KLAIM
	User        string // USER_INPUT
	Action      string // ACTION
	Note        string // NOTE
	At          time.Time
}

// ClosureStore menulis akibat penutupan klaim pada tabel warisan.
type ClosureStore interface {
	// SaveClosure menulis isian dialog dan ISPENDINGCLOSE; closedAt bukan nol menulis CLOSECLAIMDATE.
	SaveClosure(ctx context.Context, claimID string, c Closure, closedAt time.Time) error
	// LogClosure menulis CloseRejectClaim_Log.
	LogClosure(ctx context.Context, l ClosureLog) error
	// MarkDashboardClosed — `UpdateStsKlaimClose_sql`: PEGA_DASHBOARDPNC.STSKLAIM = '3'.
	MarkDashboardClosed(ctx context.Context, claimNumber string) error
	// ReleaseTechnicalPIC — `UpdateTotalJob_sql`: MST_USER_TEKNIS.TOTAL_JOB dikurangi satu.
	ReleaseTechnicalPIC(ctx context.Context, operatorID string) error
	// LegacyOperator — `GetOperatorID`: OLD_OPERATOR_ID pengguna di T_ACCESS_GROUP_PNC; kosong bila tidak ada.
	LegacyOperator(ctx context.Context, operatorID string) (string, error)
	// PendingClose membaca ISPENDINGCLOSE klaim — dibaca layar klaim dan tombol Tutup Klaim saja,
	// bukan setiap kali klaim dimuat.
	PendingClose(ctx context.Context, claimID string) (bool, error)
}

// ValidateClosure menjalankan pemeriksaan CloseClaim langkah 3, 8, 11, dan 12 dalam urutan Pega.
// Pega berhenti pada pemeriksaan pertama yang gagal, sehingga yang dikembalikan paling banyak satu.
func ValidateClosure(k Claim, c Closure, caller, legacyCaller string) *Violation {
	reject := func(msg string) *Violation {
		return &Violation{Code: ViolationClosure, Field: "catatan_tutup", Message: msg}
	}

	// Langkah 3 — entitas ASM, bukan Travel/PA, salvage TBA, bukan tutup sementara.
	travelPA := k.Policy.Line == LineTravel || k.Policy.Line == LinePersonalAccident
	if strings.EqualFold(strings.TrimSpace(k.Portal), "ASM") && !travelPA &&
		strings.TrimSpace(k.SalvageStatus) == SalvageStatusTBA && !c.Temporary {
		return reject(MsgClosureSalvageTBA)
	}

	// Langkah 8.
	if strings.TrimSpace(c.Note) == "" {
		return reject(MsgClosureNote)
	}

	// Langkah 11 — hanya PIC Teknis klaim itu (identitas sekarang atau identitas lamanya).
	pic := strings.TrimSpace(k.TechnicalPIC)
	isPIC := pic != "" && (strings.EqualFold(pic, strings.TrimSpace(caller)) ||
		(strings.TrimSpace(legacyCaller) != "" && strings.EqualFold(pic, strings.TrimSpace(legacyCaller))))
	if !isPIC {
		return reject(MsgClosureNotPIC)
	}

	// Langkah 12 — `ValidationAdjustmentKomite`: adjustment dengan AcceptanceStatus 0 masih di komite.
	for _, o := range k.InsuredItem {
		for _, cov := range o.Coverage {
			for _, line := range cov.Settlement {
				if strings.TrimSpace(line.AcceptanceStatus) == DecisionPending {
					return reject(MsgClosureCommittee)
				}
			}
		}
	}
	return nil
}
