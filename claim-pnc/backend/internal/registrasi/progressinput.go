package registrasi

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Tombol "Input Progress Claim" pada tab Progress Claim & Komunikasi.
//
// Sumber Pega:
//
//	Section/ShowDetailProgress_sect          tombol per baris posisi, tampil bila STATUSPOSISI != 'Done'
//	Flow Action/inputprogress12              modal `InputProgressClaim1`
//	Activity/ValidationInputProgressClaim_act  submit
//	  ValidationProgressClaim_act1           STATUS PERLU dari master + "sudah diinput hari ini"
//	  ValidasiTglNextFU_act                  Tgl Next Follow Up
//	  ValidasiStatusProgress                 Status wajib
//	  PNCInsertProgressClaim                 PROGRESS_CLAIM_PNC kategori "PROGRESS" (cabang ELSE)
//	Activity/SaveMasterEmailBrokerKlaim      INSERTDATABROKEREMAILMASTER (ditulis ulang di Go, D-02)
//
// Yang BELUM dibangun karena rule-nya belum ada di export atau termasuk tahap email kedua:
// persetujuan atasan (`InsertToLeaderAksepProgress` → `InsertStatusProgressLeaderAP`),
// email pemberitahuan ke broker, kiriman ke Collection (Status Progress 1 = 016),
// `UpdateKlaimRekanan` (klaim PRODIGI), `PNCInsertMitraLog_Act`, dan blok Claim Control /
// Cooperation Clause.

// ErrProgressInputUnavailable berarti penyimpanan modal Input Progress Claim tidak dipasang.
var ErrProgressInputUnavailable = errors.New("progress input storage is not configured on this server")

// ProgressCategoryPlain adalah `categoryinsert` yang dikirim ValidationInputProgressClaim_act
// langkah 11 — bukan INSERT/UPDATE/UPDATE_1, sehingga prosedur menempuh cabang ELSE: satu baris
// progres di bawah posisi yang dipilih, posisinya tidak ditutup.
const ProgressCategoryPlain = "PROGRESS"

// ProgressStatusPrompt adalah teks pilihan kosong dropdown Status; ValidasiStatusProgress
// memperlakukannya sama dengan kosong.
const ProgressStatusPrompt = "MOHON ISI STATUS TERLEBIH DAHULU"

// ProgressFollowUpDefault adalah NEXT_FOLLOWUP bila Tgl Next Follow Up kosong
// (`PROGRESS_CLAIM_PNC.prc:7-11`, sysdate+7).
const ProgressFollowUpDefault = 7 * 24 * time.Hour

// Pesan Pega apa adanya.
const (
	MsgProgressNeedFromMaster  = "Silakan Pilih Status Progress 2 Yang Ada di Master."
	MsgProgressAlreadyToday    = "Status Progress ini Telah di Inputkan pada hari ini"
	MsgProgressFollowUpPast    = "Tanggal Next Follow Up tidak boleh kurang dari hari ini"
	MsgProgressStatusEmpty     = "Harap isi Status Terlebih Dahulu"
	MsgProgressEmailEmpty      = "Email progress klaim harus diisi"
	MsgProgressNeedNotFound    = "STATUS PROGRESS 2 TIDAK DITEMUKAN"
	MsgProgressDocTypeEmpty    = "Tipe dokumen masih kosong"
	MsgProgressStatus1Empty    = "Status Progress 1 harus diisi"
	MsgProgressRemarksEmpty    = "Remarks harus diisi"
	MsgProgressPositionClosed  = "Posisi progres ini sudah Done"
	MsgProgressPositionMissing = "Posisi progres tidak ditemukan"
	MsgProgressRemarksTooLong  = "Remarks maksimal 2000 karakter"
)

// ProgressRemarksMax adalah batas Remarks (`InputProgressClaim1`, max 2000) — sama dengan
// panjang KETERANGAN.
const ProgressRemarksMax = 2000

// ViolationProgress menandai pelanggaran modal Input Progress Claim.
const ViolationProgress ViolationCode = "progres"

// ProgressDocumentStatus adalah Status Progress 1 "WAITING DOCUMENT CLAIM": Tipe Dokumen tampil.
const ProgressDocumentStatus = "003"

// progressEmailStatuses: Email Progress Klaim wajib (003 Waiting Document Claim, 008 Waiting
// LOD, 011 Waiting Reinstatement).
var progressEmailStatuses = map[string]bool{"003": true, "008": true, "011": true}

// ProgressEmailStatuses adalah Status Progress 1 yang mewajibkan Email Progress Klaim, urut.
func ProgressEmailStatuses() []string { return []string{"003", "008", "011"} }

// ProgressNeedsEmail menyatakan Email Progress Klaim wajib untuk Status Progress 1 ini.
func ProgressNeedsEmail(progress1 string) bool {
	return progressEmailStatuses[strings.TrimSpace(progress1)]
}

// ProgressPosition adalah satu baris GCNM_PROGRESS_POSISI_PNC.
type ProgressPosition struct {
	ID     string
	CaseID string
	Name   string
	Status string
	Date   time.Time
}

// Open menyatakan tombol Input Progress Claim tampil (`.CountryID != 'Done'`).
func (p ProgressPosition) Open() bool { return p.Status != AcceptanceProgressDone }

// ProgressOption adalah satu pilihan master progres.
type ProgressOption struct {
	ID   string
	Name string
}

// ProgressNeed adalah satu baris grid STATUS PERLU: ID_MST dan teksnya.
type ProgressNeed struct {
	ID   string
	Name string
}

// ProgressInput adalah isian modal yang siap ditulis.
type ProgressInput struct {
	ClaimNumber  string
	PositionID   string
	Progress1    string
	Needs        []ProgressNeed
	Status       string
	Remarks      string
	NextFollowUp time.Time // nol: sysdate+7
	DocType      string
	User         string
	At           time.Time
}

// LastNeed adalah STATUS_PROGRESS2 yang ditulis: Pega mengisi `temppyWorkPage.CountryID` dari
// baris terakhir grid (ValidationProgressClaim_act1 langkah 4).
func (p ProgressInput) LastNeed() ProgressNeed {
	if len(p.Needs) == 0 {
		return ProgressNeed{}
	}
	return p.Needs[len(p.Needs)-1]
}

// BrokerEmail adalah isian SaveMasterEmailBrokerKlaim.
type BrokerEmail struct {
	Email        string
	PolicyNumber string
	GroupPanel   string
	BusinessCode string
	BranchCode   string
	ClaimID      string
}

// ProgressInputStore membaca master dan menulis progres.
type ProgressInputStore interface {
	Positions(ctx context.Context, claimNumber string) ([]ProgressPosition, error)
	Status1Options(ctx context.Context) ([]ProgressOption, error)
	StatusOptions(ctx context.Context) ([]string, error)
	DocTypes(ctx context.Context) ([]string, error)
	// NeedOptions: STATUS PERLU satu Status Progress 1; docType menyaring kolom TIPE.
	NeedOptions(ctx context.Context, progress1, docType string) ([]ProgressOption, error)
	// NeedName: STS_PROGRESS2 satu ID_MST (ViewStatusProgress111); false bila tidak ada.
	NeedName(ctx context.Context, id string) (string, bool, error)
	// InputToday: teks STATUS PROGRESS 2 (GET_POSISI_PROGRESS2) progres non-AUTO pada hari itu.
	InputToday(ctx context.Context, claimNumber string, day time.Time) ([]string, error)
	Insert(ctx context.Context, p ProgressInput) error
	BrokerEmail(ctx context.Context, b BrokerEmail) (string, error)
	SaveBrokerEmail(ctx context.Context, b BrokerEmail) error
}

// CheckProgressInput memeriksa isian yang tidak memerlukan basis data, dengan urutan Pega.
func CheckProgressInput(p ProgressInput, email string, today time.Time) []Violation {
	var v []Violation
	add := func(field, msg string) {
		v = append(v, Violation{Code: ViolationProgress, Field: field, Message: msg})
	}
	if strings.TrimSpace(p.Progress1) == "" {
		add("status_progress_1", MsgProgressStatus1Empty)
	}
	if len(p.Needs) == 0 {
		add("status_perlu", MsgProgressNeedFromMaster)
	}
	for _, n := range p.Needs {
		if strings.TrimSpace(n.ID) == "" {
			add("status_perlu", MsgProgressNeedFromMaster)
			break
		}
	}
	if !p.NextFollowUp.IsZero() && !dayAfter(p.NextFollowUp, today) {
		add("tgl_next_follow_up", MsgProgressFollowUpPast)
	}
	if s := strings.TrimSpace(p.Status); s == "" || s == ProgressStatusPrompt {
		add("status", MsgProgressStatusEmpty)
	}
	if ProgressNeedsEmail(p.Progress1) && strings.TrimSpace(email) == "" {
		add("email_progress", MsgProgressEmailEmpty)
	}
	if r := strings.TrimSpace(p.Remarks); r == "" {
		add("remarks", MsgProgressRemarksEmpty)
	} else if len([]rune(r)) > ProgressRemarksMax {
		add("remarks", MsgProgressRemarksTooLong)
	}
	return v
}

// dayAfter: tanggal kalender a sesudah tanggal kalender b (keduanya sudah WIB).
func dayAfter(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC).After(time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC))
}

// AlreadyInputToday meniru ValidationProgressClaim_act1 langkah 11: teks STATUS PERLU baris
// terakhir TERKANDUNG di STATUS PROGRESS 2 progres hari ini.
func AlreadyInputToday(need string, today []string) bool {
	need = strings.TrimSpace(need)
	if need == "" {
		return false
	}
	for _, t := range today {
		if strings.Contains(t, need) {
			return true
		}
	}
	return false
}
