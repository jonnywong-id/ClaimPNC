package registrasi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// # Tab Survey tahap Choose Surveyor
//
// `Section/TabSurvey_sect.xml` — tiga sub-tab: Permintaan Survey (`RequestSurvey`), Tambah
// Survey (`InputSurvey`), Hasil Survey (`ViewHasilSurvey`). Grid Tambah Survey berada di atas
// `.ClaimData.ObjectList`; isian survey per objek di Pega tersimpan di objek kerja, di sini di
// kolom survey POOLDATA.T_CLAIM_OBJECTLIST (keputusan Work Owner 2026-10-11, lihat
// docs/ddl/t_claim_objectlist_survey.sql).
//
// Prosesnya ditiru PERSIS Pega (keputusan Work Owner 2026-10-11), termasuk survey (SRV) yang
// dibuat `Post_ChildSurvey_act` untuk SELURUH objek klaim — prekondisi `.pySelected` langkah
// 2 dinonaktifkan di rule — baik oleh Transfer Survei maupun Transfer Komite.

// Tipe Surveyor — `Property/SurveyorType-property.xml:160-180`.
const (
	SurveyorInternal     = "1"
	SurveyorLossAdjuster = "2"
	SurveyorExpert       = "3"
	SurveyorSurveyAgent  = "4"
)

// SurveyorTypeName adalah label pilihan Tipe Surveyor.
var SurveyorTypeName = map[string]string{
	SurveyorInternal:     "Internal Surveyor",
	SurveyorLossAdjuster: "Loss Adjuster",
	SurveyorExpert:       "Expert",
	SurveyorSurveyAgent:  "Survey Agent",
}

// SurveyorMasterType adalah M_SURVEY_ID V_D_SURVEYORS per tipe — `ChooseSurveyorType_act`
// (BrowseSurveyorTypeInternalSurveyor 1001, LossAdjuster 1002, Expert 1003, SurveyAgent 1004).
var SurveyorMasterType = map[string]string{
	SurveyorInternal:     "1001",
	SurveyorLossAdjuster: "1002",
	SurveyorExpert:       "1003",
	SurveyorSurveyAgent:  "1004",
}

// Status survey objek — `Property/ObjectStatus-property.xml:305-339`.
const (
	SurveyAwaitingApproval = "1" // Menunggu Persetujuan
	SurveyRejected         = "2" // Ditolak Komite
	SurveyInProgress       = "3" // Sedang Proses
	SurveyCancelled        = "4" // Batal Survey
	SurveyDone             = "5" // Selesai
)

// SurveyStatusName adalah label status survey objek.
var SurveyStatusName = map[string]string{
	SurveyAwaitingApproval: "Menunggu Persetujuan",
	SurveyRejected:         "Ditolak Komite",
	SurveyInProgress:       "Sedang Proses",
	SurveyCancelled:        "Batal Survey",
	SurveyDone:             "Selesai",
	"6":                    "YA",
	"7":                    "TIDAK",
}

// Nilai T_SURVEYORLIST yang ditulis alur ini.
const (
	SurveyRecordOnProgress              = "On Progress"       // STS_SURVEY — `Post_ChildSurvey_act` 2.15
	SurveyRecordCancelled               = "Batal Survey"      // STS_SURVEY saat Batal Survei (Work Owner 2026-10-11)
	SurveyWorkRejected                  = "Resolved-Rejected" // PYSTATUSWORK — `CancelSurvey` langkah 7 (ASMForceCaseClose)
	SurveyProgressPosition              = "SURVEY"            // Posisi progres — `PNCInsertProgressClaim`
	SurveyProgressNote                  = "Auto Create Survey"
	SurveyProgress1                     = "004"
	SurveyProgress2                     = "59"
	StatusClaimSurvey       ClaimStatus = "1145"            // `InternalSurveyor_act` 13, `ValidationAnalysis` 21
	SurveyCommitteeTransfer             = "1"               // TransferType Komite survey — `Post_ChildKomite_act` 3
	PABranchCodeAfterCancel             = "100081"          // `CancelSurvey` langkah 3 (lini PA)
	SurveyDirectorLimit     Money       = 100_000_001 * 100 // `ValidationAnalysis` 16: limit_bottom<=100000001
)

// Pesan Pega yang ditiru apa adanya.
const (
	MsgSurveyMustSelect    = "Harus Pilih Object"                                                          // setError_act / CheckTypeSurveyor_act
	MsgSurveyLocationEmpty = "Lokasi Objek Kosong "                                                        // setError_act 3.6
	MsgSurveyLoginEmpty    = "Login Surveyor Kosong"                                                       // setError_act 3.7
	MsgSurveyAllInProgress = "Semua Object Sedang Diproses"                                                // setError_act 6
	MsgSurveySurveyorBusy  = "Silakan pilih Surveyor lain, karena Surveyor ini sedang dalam proses survey" // SetErrorSRV_act
	MsgSurveyUseCommittee  = "Gunakan button Transfer Komite untuk survey ini"                             // InternalSurveyor_act 4.1
	MsgSurveyAdjusterEmpty = "Nama Adjuster Kosong"                                                        // Post_ChildSurvey_act 2.2
	MsgSurveyPremiumUnpaid = "Premi belum lunas, tidak bisa transfer komite."                              // ValidationPremi_act
	MsgSurveyNoCommittee   = "Error Case Komite tidak kebuat. Silakan transfer ulang"                      // tidak ada anggota komite survey
)

// ErrSurveyUnavailable berarti penyimpanan survey tidak dipasang (mode tanpa Oracle).
var ErrSurveyUnavailable = errors.New("survey storage is not configured on this server")

// ErrSurveyCommitteeAwaiting: putusan Komite survey menunggu layanan link email Pega
// (`KomiteAcceptSurvey`, `ShowKomiteSurveyConfirmation`) — keputusan Work Owner 2026-10-11.
var ErrSurveyCommitteeAwaiting = errors.New("survey committee decisions wait for the Pega email-link services (KomiteAcceptSurvey); they cannot be decided here yet")

// SurveyObject adalah satu baris grid Tambah Survey — satu objek klaim beserta isian surveynya.
type SurveyObject struct {
	Seq      int    // URUTAN
	ObjectID string // OBJECTID
	Name     string // OBJECTNAME
	Location string // LOKASI — `.ObjectLocation`

	Selected       bool   // .pySelected
	SurveyLocation string // .ObjectSurveyLocation — OBJECTSURVEYLOCATION
	SurveyorType   string // .SurveyorType — SURVEYORTYPE
	SurveyorName   string // .ObjectSurveyor
	SurveyorLogin  string // .ObjectSurveyorLogin
	SurveyorAddr   string // .SurveyorAddrress
	SurveyorEmail  string // .ObjectSurveyorOthers (email surveyor internal)
	BranchCode     string // .BranchCode — BRANCHCODE
	BranchName     string // .BranchName
	MarineName     string // .ObjectSurveyorMarine
	MarineLogin    string // .ObjectSurveyorMarineLogin
	Status         string // .ObjectStatus
	SurveyID       string // .ObjectSurveyID
	SurveyIDMarine string // .ObjectSurveyIDMarine
}

// SurveyorOption adalah satu baris V_D_SURVEYORS untuk pilihan surveyor.
type SurveyorOption struct {
	ID         string // D_SURVEY_ID
	Name       string // NAME
	Login      string // LOGIN_APLIKASI
	Address    string // ADDRESS
	Branch     string // BRANCH
	BranchName string // BRANCHNAME
	Email      string // EMAIL
	Contact    string // OTHER_CONTACT
}

// SurveyRecord adalah satu baris POOLDATA.T_SURVEYORLIST — `INSERT_SURVEYORLIST`.
type SurveyRecord struct {
	CaseID         string // CASEID — nomor survey (SRVN.YY.n)
	ClaimID        string // PNCCASEID
	SurveyType     string // SURVEYTYPE — `ClaimData.SurveyData.SurveyorType` (tingkat klaim)
	SurveyorName   string // SURVEYOR_NAME
	MarineName     string // SURVEYOR_NAME_MARINE
	SurveyDate     time.Time
	SurveyLocation string    // LOCATION_SURVEY / RESCHEDULE_LOCATION
	ObjectName     string    // OBJECT_NAME / NAMA_PASIEN
	ObjectLocation string    // LOCATION_OBJECT
	ObjectID       string    // IDOBJECT
	Index          string    // INDEX_SURVEY
	TreatmentDate  time.Time // TGLPERAWATAN — DateOfLoss
	Status         string    // STS_SURVEY
	At             time.Time // TGLINPUT
}

// SurveyRow adalah satu survey tersimpan klaim — pengganti `.ClaimData.SurveyResults`.
type SurveyRow struct {
	CaseID       string
	ObjectID     string
	SurveyorName string
	Status       string // STS_SURVEY
	WorkStatus   string // PYSTATUSWORK
}

// InProgress menyatakan survey ini masih "Sedang Proses" (SurveyStatus 3).
func (r SurveyRow) InProgress() bool {
	return !strings.EqualFold(strings.TrimSpace(r.WorkStatus), SurveyWorkRejected) &&
		!strings.EqualFold(strings.TrimSpace(r.Status), SurveyRecordCancelled)
}

// SurveyCommitteeMember adalah satu baris EMAILKOMITE komite survey.
type SurveyCommitteeMember struct {
	OperatorID string // OPERATOR_ID (BUSINESS_CODE)
	Email      string // EMAIL (BRANCH_CODE)
}

// SurveyCommitteeNote adalah isian modal Transfer Komite (`Section/ClaimComitee-sect.xml`).
type SurveyCommitteeNote struct {
	Date          time.Time // Date & Time — TempCommiteClaim.DateOfComitee
	Initial       string    // Inisial — ClaimData.UserTeknis
	AnalysisType  string    // Tipe Analisis — "1" Survey
	Circumstances string    // Circumtanses cause of Loss
	Nominated     string    // Nominated Adjuster
	Remarks       string
	Company       string // Company Name
	ContactPerson string
	OfficePhone   string
	Email         string
}

// SurveyRequest adalah Permintaan Survey (`RequestSurvey`) — baris POOLDATA.T_REQ_SURVEY.
type SurveyRequest struct {
	RequestorName string    // REQ_NAME — Survey Atas Permintaan
	Location      string    // LOCATION — Lokasi Survey
	Phone         string    // NOTELP — No. Telp
	Date          time.Time // DATE_SURVEY — Tanggal Request Survey
	Branch        string    // BRANCH — Cabang
	Surveyor      string    // SURVEYOR
	Email         string    // EMAIL_SURVEYOR
	ObjectName    string    // OBJECTNAME
}

// SurveyStore adalah seam penyimpanan tab Survey.
type SurveyStore interface {
	Objects(ctx context.Context, claimID string) ([]SurveyObject, error)
	SaveObjects(ctx context.Context, claimID string, objects []SurveyObject, by string, at time.Time) error
	Surveyors(ctx context.Context, surveyorType string) ([]SurveyorOption, error)
	NominatedOptions(ctx context.Context) ([]SurveyorOption, error)
	NextSurveyID(ctx context.Context, at time.Time) (string, error)
	InsertRecord(ctx context.Context, r SurveyRecord) error
	Rows(ctx context.Context, keys RecordKeys) ([]SurveyRow, error)
	Cancel(ctx context.Context, caseID string, at time.Time) error
	DirectorSurveyor(ctx context.Context, names []string) (bool, error)
	Limit(ctx context.Context, businessCode, syariah string) (Money, bool, error)
	CommitteeMembers(ctx context.Context, line string, belowDirector bool) ([]SurveyCommitteeMember, error)
	SaveNominated(ctx context.Context, claimID string, list []SurveyorOption) error
	SaveCommitteeNote(ctx context.Context, committeeID string, n SurveyCommitteeNote) error
	Request(ctx context.Context, keys RecordKeys) (SurveyRequest, bool, error)
}

// IsMarineHull adalah When `IsMarineHull`: BusinessType MarineHull/WreckRemoval, kode 10152,
// atau nama MARINE HULL INSURANCE.
func IsMarineHull(p Policy) bool {
	t := strings.TrimSpace(p.BusinessType)
	return t == "MarineHull" || t == "WreckRemoval" || strings.TrimSpace(p.BusinessCode) == "10152" ||
		strings.TrimSpace(p.BusinessName) == "MARINE HULL INSURANCE"
}

// CoInsuranceMember menyatakan Sinar Mas anggota (bukan leader) koasuransi —
// `CheckTypeSurveyor_act` langkah 4 (CoinsName memuat ASURANSI SINAR MAS, Leader false).
func CoInsuranceMember(p Policy) bool {
	return strings.EqualFold(strings.TrimSpace(p.Coinsurance.Role), "MEMBER")
}

// SurveyCommitteeLine adalah TYPE_BUSINESS komite survey — `ValidationAnalysis` 9–11
// (`IsNotTravelPA` NONMBU, `IsPA` PA, `IsTravel` TRAVEL).
func SurveyCommitteeLine(p Policy) string {
	switch {
	case p.Line == LineTravel || p.BusinessType == "Travel":
		return "TRAVEL"
	case p.Line == LinePersonalAccident:
		return "PA"
	}
	return CommitteeLineNonMBU
}

// CheckSurveyObjects adalah `setError_act` + `SetErrorSRV_act`: pemeriksaan sebelum Transfer
// Survei dan Transfer Komite. Lokasi survey yang kosong diisi lokasi objek lebih dulu (3.1).
// Pesan "Harus Pilih Object" dan "Semua Object Sedang Diproses" menghentikan pemeriksaan
// lain, seperti lompatan F1/F2 di rule.
func CheckSurveyObjects(objects []SurveyObject, rows []SurveyRow) []Violation {
	var v []Violation
	add := func(msg string) { v = append(v, Violation{Code: ViolationSurvey, Field: "survey", Message: msg}) }
	selected, inProgress := 0, 0
	for i := range objects {
		o := &objects[i]
		if o.Status == SurveyInProgress {
			inProgress++
		}
		if !o.Selected {
			continue
		}
		selected++
		if strings.TrimSpace(o.SurveyLocation) == "" {
			o.SurveyLocation = o.Location
		}
		if strings.TrimSpace(o.SurveyLocation) == "" {
			add(MsgSurveyLocationEmpty)
		}
		if strings.TrimSpace(o.SurveyorName) == "" && o.SurveyorType != SurveyorInternal && o.SurveyorType != SurveyorLossAdjuster {
			add(MsgSurveyLoginEmpty)
		}
	}
	if selected == 0 {
		return []Violation{{Code: ViolationSurvey, Field: "survey", Message: MsgSurveyMustSelect}}
	}
	if inProgress == len(objects) {
		return []Violation{{Code: ViolationSurvey, Field: "survey", Message: MsgSurveyAllInProgress}}
	}
	// SetErrorSRV_act: seluruh objek (dipilih atau tidak); kedua penanda tidak direset antar objek.
	sameSurveyor, sameStatus := false, false
	for _, o := range objects {
		for _, r := range rows {
			if strings.TrimSpace(r.ObjectID) != strings.TrimSpace(o.ObjectID) {
				continue
			}
			if strings.TrimSpace(o.SurveyorName) != "" && strings.TrimSpace(o.SurveyorName) == strings.TrimSpace(r.SurveyorName) {
				sameSurveyor = true
			}
			if o.Status == SurveyInProgress && r.InProgress() {
				sameStatus = true
			}
		}
	}
	if sameSurveyor && sameStatus {
		add(MsgSurveySurveyorBusy)
	}
	return v
}

// ViolationSurvey menandai pelanggaran tab Survey.
const ViolationSurvey ViolationCode = "survey"

// CancelSurveyObject adalah `CancelSurvey` langkah 2, 3, dan 8 pada objek klaim.
func CancelSurveyObject(objects []SurveyObject, index int, pa bool) {
	o := &objects[index]
	o.Status = SurveyCancelled
	o.Selected = false
	o.BranchCode, o.BranchName, o.SurveyorName, o.SurveyorType = "", "", "", ""
	if pa {
		for i := range objects {
			objects[i].SurveyorType = SurveyorInternal
			objects[i].BranchCode = PABranchCodeAfterCancel
		}
	}
	// Langkah 8 — tanpa syarat, pada SELURUH objek.
	for i := range objects {
		objects[i].BranchName, objects[i].SurveyorName, objects[i].SurveyorType = "", "", ""
	}
}

// SurveyCasePrefix membedakan survey terbitan aplikasi ini dari `SRV-` Pega, seperti PNCN, RCVN,
// dan KMTN.
const SurveyCasePrefix = "SRVN"

// FormatSurveyID menyusun nomor survey `SRVN.YY.n`.
func FormatSurveyID(year int, sequence int64) string {
	return fmt.Sprintf("%s.%02d.%d", SurveyCasePrefix, year%100, sequence)
}
