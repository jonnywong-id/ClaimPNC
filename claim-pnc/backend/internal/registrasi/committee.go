package registrasi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Komite adjustment klaim PNCN — tombol Transfer Komite pada grid Adjustment & Akseptasi.
//
// # Sumber Pega
//
//   - `Activity/ValidationTypePaymentAdj-Act.xml` — pemeriksaan sebelum transfer, lalu
//     memanggil SetListComiteeClaimPerObjAdj bila baris belum pernah ditransfer
//     (`.AdjustmentList(idAdj).CaseIDKomite == ""`).
//   - `Activity/SetListComiteeClaimPerObjAdj-Act.xml` — nilai pembanding, daftar komite
//     dari EMAILKOMITE, `KomiteLoop := pxResultCount`, `KomiteCount := 1`,
//     `IsKomiteTransfer := 1`, `AcceptanceStatus := 0`, `StatusClaim := "1149"`.
//   - `Activity/SetChildKomitePerAdjustment_act-Act.xml` — case `Work-Komite`,
//     `TransferType := "2"`.
//   - `Activity/InsertUpdateKomiteList-Act.xml` → `INSERTDATAKOMITELIST.prc` — satu baris
//     T_CLAIM_KOMITE_LIST per anggota (KOMITE_ID, NAMAKOMITE = OPERATOR_ID, KOMITEKE urut).
//   - `Activity/KomitePost_Adjustment-Act.xml` — putusan: setuju menaikkan KomiteCount;
//     setuju di jenjang terakhir `AcceptanceStatus := 1`; tolak `AcceptanceStatus := 2`,
//     sisa anggota ditandai 2 (step 31) dan case ditutup (step 60).
//
// # Kasus komite hidup di T_CLAIM_KOMITE_LIST, satu baris per anggota
//
// Case `Work-Komite` Pega tidak dibuat: kasus komite klaim PNCN adalah baris-baris
// T_CLAIM_KOMITE_LIST ber-KOMITE_ID sama (keputusan Work Owner 2026-09-28; NO_KLAIM sudah
// diperpanjang). Anggota yang sedang ditunggu adalah baris berkeputusan 0 dengan KOMITEKE
// terkecil — tidak ada tugas CPNC_TUGAS kedua per klaim.

// Nilai kolom T_CLAIM_KOMITE_LIST dan T_CLAIM_ADJUSTMENT yang dipakai komite.
const (
	// CommitteeTransferType adalah TYPEKOMITE — `childPageKomite.TransferType := "2"`.
	CommitteeTransferType = "2"

	// Keputusan anggota (STATUSAPPROVE) dan status akseptasi adjustment (STATUSAKSEPTASI)
	// memakai kode yang sama.
	DecisionPending = "0"
	DecisionApprove = "1"
	DecisionReject  = "2"

	// STATUSCASE: pyStatusWork case komite.
	CommitteeCaseOpen   = "New"
	CommitteeCaseClosed = "Resolved-Completed"

	// CommitteeCasePrefix membedakan kasus komite aplikasi ini dari `KMT-` Pega. Nomornya
	// `KMTN.YY.n` seperti PNCN dan RCVN (Work Owner, 2026-09-29).
	CommitteeCasePrefix = "KMTN"

	// CommitteeCaseIDMaxLength adalah lebar KOMITE_ID di T_CLAIM_KOMITE_LIST dan
	// TC_PNC_KOMITE: VARCHAR2(10). `KMTN.26.99` tepat sepuluh; nomor ke-100 dalam satu
	// tahun tidak muat sampai kolomnya dilebarkan.
	CommitteeCaseIDMaxLength = 10

	// StatusClaimCommittee adalah StatusClaim `1149` Claim Committee.
	StatusClaimCommittee ClaimStatus = "1149"

	// Nilai pembanding untuk Tolak Klaim (`SetListComiteeClaimPerObjAdj` step 36, entitas
	// ASM): sengaja di atas ambang tertinggi supaya seluruh jenjang ikut menyetujui.
	rejectionCommitteeValue Money = 500_000_001 * 100
)

// ErrCommitteeNotFound: kasus komite tidak ada.
var ErrCommitteeNotFound = errors.New("registrasi: kasus komite tidak ditemukan")

// CommitteeApprover adalah satu jenjang penyetuju hasil penjenjangan EMAILKOMITE.
type CommitteeApprover struct {
	OperatorID string
	Name       string
}

// CommitteeMember adalah satu baris T_CLAIM_KOMITE_LIST.
type CommitteeMember struct {
	CaseID      string // KOMITE_ID
	ClaimNumber string // NO_KLAIM
	Operator    string // NAMAKOMITE — OPERATOR_ID anggota
	Level       int    // KOMITEKE
	Decision    string // STATUSAPPROVE
	CaseStatus  string // STATUSCASE
	Note        string // NOTEKOMITE

	TransferType string  // TYPEKOMITE
	PaymentType  string  // PAYMENTTYPE
	ShareASM     Percent // SHAREASM
	Value        Money   // NILAIKLAIM

	CreatedAt time.Time // DATEOFCOMMITE_CREATE
	DecidedAt time.Time // TANGGALKOMITE
}

// CommitteeCase adalah satu kasus komite: kepalanya (POOLDATA.TC_PNC_KOMITE — pengganti
// case `Work-Komite` Pega) dan anggotanya (T_CLAIM_KOMITE_LIST), urut KOMITEKE.
type CommitteeCase struct {
	ID          string // KOMITE_ID
	ClaimID     string // CLAIMID
	ClaimNumber string // NO_KLAIM

	// Alamat baris adjustment — kunci T_CLAIM_ADJUSTMENT: OBJECTID objek, urutan coverage
	// di dalam objek, urutan adjustment.
	ObjectID        string
	CoverageSeq     int
	AdjustmentSeq   int
	PaymentType     string // PAYMENTTYPE — `childPageKomite.Type`
	TransferType    string // TRANSFERTYPE — "2"
	Line            string // TYPE_BUSINESS — lini komite (`tempAdj.pyMemo`)
	Band            string // TYPE_KOMITE — pita Non-MBU (`tempAdj.AcceptedNo`); kosong di lini lain
	Currency        string
	Rate            ExchangeRate
	AdjustmentValue Money // nilai bagian ASM baris itu (AdjustmentValue)
	Value           Money // nilai pembanding dalam IDR (`ConvertAdjustmentValue`)
	Applicant       string

	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
	DecidedAt time.Time

	Members []CommitteeMember
}

// Status adalah pyStatusWork kasus: New selama ada jenjang yang ditunggu.
func (c CommitteeCase) Status() string {
	if c.Outcome() != "" {
		return CommitteeCaseClosed
	}
	return CommitteeCaseOpen
}

// Level adalah KomiteCount: jenjang yang sedang ditunggu, atau jumlah jenjang bila selesai.
func (c CommitteeCase) Level() int {
	if m, ok := c.Current(); ok {
		return m.Level
	}
	return len(c.Members)
}

// Current mengembalikan anggota yang sedang ditunggu keputusannya.
func (c CommitteeCase) Current() (CommitteeMember, bool) {
	for _, m := range c.Members {
		if m.CaseStatus == CommitteeCaseClosed {
			return CommitteeMember{}, false
		}
		if m.Decision == DecisionPending {
			return m, true
		}
		if m.Decision != DecisionApprove {
			return CommitteeMember{}, false
		}
	}
	return CommitteeMember{}, false
}

// Outcome adalah status akseptasi hasil komite: kosong selama masih berjalan.
func (c CommitteeCase) Outcome() string {
	if len(c.Members) == 0 {
		return ""
	}
	for _, m := range c.Members {
		if m.Decision == DecisionReject {
			return DecisionReject
		}
		if m.Decision != DecisionApprove {
			return ""
		}
	}
	return DecisionApprove
}

// ErrNotCommitteeTurn: pemanggil bukan anggota yang sedang ditunggu.
var ErrNotCommitteeTurn = errors.New("registrasi: bukan giliran pemanggil memutuskan komite ini")

// Decide mencatat keputusan anggota yang sedang ditunggu (`KomitePost_Adjustment`).
//
// Setuju pada jenjang terakhir dan tolak pada jenjang mana pun menutup kasus: seluruh baris
// menjadi Resolved-Completed, dan anggota sesudah penolak ditandai tolak (step 31).
func (c *CommitteeCase) Decide(operator, decision, note string, now time.Time) error {
	if decision != DecisionApprove && decision != DecisionReject {
		return fmt.Errorf("%w: keputusan komite %q", ErrInvalidAction, decision)
	}
	current, ok := c.Current()
	if !ok || !strings.EqualFold(strings.TrimSpace(current.Operator), strings.TrimSpace(operator)) {
		return ErrNotCommitteeTurn
	}
	closed := false
	for i := range c.Members {
		m := &c.Members[i]
		if m.Level == current.Level && strings.EqualFold(m.Operator, current.Operator) {
			m.Decision, m.Note, m.DecidedAt = decision, strings.TrimSpace(note), now
		}
	}
	if decision == DecisionReject {
		for i := range c.Members {
			if c.Members[i].Decision == DecisionPending {
				c.Members[i].Decision = DecisionReject
			}
		}
		closed = true
	} else if _, more := c.Current(); !more {
		closed = true
	}
	if closed {
		for i := range c.Members {
			c.Members[i].CaseStatus = CommitteeCaseClosed
		}
	}
	return nil
}

// FormatCommitteeCaseID menyusun KOMITE_ID `KMTN.YY.n` — nomor urut tanpa nol di depan,
// deret per tahun. Nomor yang melampaui lebar kolom KOMITE_ID ditolak, bukan dipotong:
// memotong akan menerbitkan nomor ganda.
func FormatCommitteeCaseID(year int, sequence int64) (string, error) {
	id := fmt.Sprintf("%s.%02d.%d", CommitteeCasePrefix, year%100, sequence)
	if len(id) > CommitteeCaseIDMaxLength {
		return "", fmt.Errorf("registrasi: nomor komite %s melampaui KOMITE_ID VARCHAR2(%d)", id, CommitteeCaseIDMaxLength)
	}
	return id, nil
}

// NewCommitteeCase membentuk kasus komite dari daftar penyetuju, jenjang 1 lebih dulu.
func NewCommitteeCase(id, claimNumber string, approvers []CommitteeApprover, line SettlementLine, value Money, now time.Time) CommitteeCase {
	c := CommitteeCase{
		ID: id, ClaimNumber: claimNumber, PaymentType: line.PaymentType, TransferType: CommitteeTransferType,
		Currency: line.Currency, Rate: line.Rate, AdjustmentValue: line.Value, Value: value,
		CreatedAt: now, UpdatedAt: now,
	}
	for i, a := range approvers {
		c.Members = append(c.Members, CommitteeMember{
			CaseID: id, ClaimNumber: claimNumber, Operator: strings.TrimSpace(a.OperatorID), Level: i + 1,
			Decision: DecisionPending, CaseStatus: CommitteeCaseOpen,
			TransferType: CommitteeTransferType, PaymentType: line.PaymentType, ShareASM: line.ShareASM,
			Value: value, CreatedAt: now,
		})
	}
	return c
}

// Lini komite Non-MBU.
const (
	CommitteeLineNonMBU   = "NONMBU"
	CommitteeLineNonMBUAB = "NONMBUAB"
)

// nonMBUGroupLimit: nilai Non-MBU sampai Rp 50.000.000 diputus komite grup A/B
// (`SetEmailKomite` step 4, "kalo < 50jt sesuaikan grup A, B, C").
const nonMBUGroupLimit Money = 50_000_000 * 100

// CommitteeLine adalah TYPE_BUSINESS EMAILKOMITE untuk polis dan nilai pembanding ini
// (`SetEmailKomite` step 4 dan 15–19).
//
//   - Bonding/BondingKBG/AsuransiKredit/CustomBond atau kode 10145/10168 → BONDING;
//     Travel → TRAVEL; PA → PA.
//   - Non-MBU sampai Rp 50.000.000 → NONMBUAB (grup A/B). Data Pega 2025–2026 menegaskannya:
//     seluruh 34 adjustment Non-MBU ≤ Rp 50 jt yang diputus sejak Juni 2025 diputus satu anggota
//     baris NONMBUAB, bukan tangga NONMBU (yang pada pita 1 baru mulai Rp 5.000.000).
//     Grup C (NONMBUC, `UserTeknisGroup == "C"`) tidak dibedakan: registrasi tidak menyimpan
//     grup PIC Teknik, dan master hidup memberi penyetuju yang sama bagi keduanya.
//   - Non-MBU di atasnya → NONMBU; pitanya dipilih modul komite (`D-70`).
//
// Syarat berbasis nama orang pada step 6–11 tidak dibawa (`D-52`).
func CommitteeLine(p Policy, value Money) string {
	switch strings.TrimSpace(p.BusinessType) {
	case "Bonding", "BondingKBG", "AsuransiKredit", "CustomBond":
		return "BONDING"
	case "Travel":
		return "TRAVEL"
	case "PA":
		return "PA"
	}
	switch strings.TrimSpace(p.BusinessCode) {
	case "10145", "10168":
		return "BONDING"
	}
	if value <= nonMBUGroupLimit {
		return CommitteeLineNonMBUAB
	}
	return CommitteeLineNonMBU
}

// CommitteeValue adalah nilai pembanding ambang (`tempAdj.ConvertAdjustmentValue`), dalam
// rupiah, dibulatkan ke atas ke rupiah penuh (`SetEmailKomite` step 3: abs lalu ceiling).
//
//   - Fee adjuster: AdjusterFeeValue — gross fee (step 30).
//   - Lainnya: AdjustmentValue (bagian ASM), atau GrossValue bila Sinar Mas leader
//     koasuransi (step 28, 31).
//   - Tolak Klaim, atau Total Klaim di bawah risiko sendiri tanpa nilai bayar: nilai tetap
//     di atas ambang tertinggi (step 36, entitas ASM).
func CommitteeValue(line SettlementLine, p Policy) Money {
	if line.PaymentType == PaymentReject || (line.Propose < line.RiskValue && line.Value == 0) {
		return rejectionCommitteeValue
	}
	basis := line.Value
	switch {
	case line.PaymentType == PaymentAdjusterFee:
		basis = line.Gross
	case strings.EqualFold(strings.TrimSpace(p.Coinsurance.Role), "LEADER"):
		basis = line.Gross
	}
	v := basis.Convert(line.Rate)
	if v < 0 {
		v = -v
	}
	if rem := v % 100; rem != 0 {
		v += 100 - rem
	}
	return v
}

// CommitteeRoute adalah hasil penjenjangan: pita yang dipilih (TYPE_KOMITE, hanya
// Non-MBU) dan penyetuju berurutan jenjang.
type CommitteeRoute struct {
	Band      string
	Approvers []CommitteeApprover
}

// CommitteeTiering menghitung penyetuju — modul komite (`B-7`) di balik seam.
type CommitteeTiering interface {
	// Route mengembalikan pita dan penyetuju. Applicant (pengaju) dikecualikan dari calon
	// penyetuju (Work Owner, 2026-09-18).
	Route(ctx context.Context, line string, value Money, applicant string) (CommitteeRoute, error)
}

// CommitteeStore menyimpan kasus komite: kepalanya di POOLDATA.TC_PNC_KOMITE, anggotanya di
// POOLDATA.T_CLAIM_KOMITE_LIST.
type CommitteeStore interface {
	// NextCaseID menerbitkan KOMITE_ID berikutnya untuk tahun (WIB) dari `at`:
	// KMTN.26.1, KMTN.26.2, …
	NextCaseID(ctx context.Context, at time.Time) (string, error)
	Save(ctx context.Context, c CommitteeCase) error
	Get(ctx context.Context, caseID string) (CommitteeCase, error)
	// Pending mengembalikan baris anggota yang sedang ditunggu dan milik operator itu.
	Pending(ctx context.Context, operator string) ([]CommitteeMember, error)
}
