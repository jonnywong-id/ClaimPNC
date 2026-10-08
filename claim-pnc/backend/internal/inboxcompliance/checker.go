package inboxcompliance

import (
	"strconv"
	"strings"
	"time"
)

// Form Compliance Checker — layar yang terbuka ketika petugas menekan Nomor Case pada tab
// Compliance.
//
// # Jalur Pega yang ditiru, dibaca dari export
//
//	Section/InputComplianceDtl_Section-Section.xml  sel Nomor Case menjalankan
//	                                                SetAssignmentInboxPUCL_act(inskey=.pzInsKey)
//	Flow/Register_Flow.xml                          Assignment9 "Compliance",
//	                                                pyWorkBasket = CompliancePNC,
//	                                                satu transisi: ComplianceChecker → End1
//	Flow Action/ComplianceChecker-FA.xml            pySectionReference  = ComplianceChecker
//	                                                pyPreProcessingActivity = SetTypePDFAdjustment
//	                                                pyPreDataTransform  = SendToPIC
//	Section/ComplianceChecker-Section.xml           tiga tab + lima tombol
//	Activity/SetComplianceResult                    yang dijalankan tombol "Simpan Data"
//
// Jadi menekan Nomor Case TIDAK membuka detail klaim. Ia membuka assignment yang menunggu
// di workbasket `CompliancePNC`, dan satu-satunya flow action di atasnya adalah form ini.
//
// # Yang TIDAK dapat dibaca
//
// Section `CompliancePNC` — isi tab Compliance — **tidak ada di export** (`R-16`). Yang
// hilang hanya TATA LETAKNYA; isinya terbaca dari tempat lain dan tidak satu pun dikarang:
//
//	Property/PilihanCompliance_property.xml           empat nilai beserta labelnya
//	Section/CatatanToAnalyst_Section-Section.xml      caption "Pilihan Compliance",
//	                                                  "Note PilihanCompliance",
//	                                                  "Catatan Dari Compliance"
//	Activity/SetComplianceResult                      akibat tiap pilihan
//
// Urutan field, mana yang wajib, dan lebar kotaknya karena itu BELUM dapat dipastikan.

// Pilihan Compliance — `.ClaimData.PilihanCompliance`.
//
// Keempat nilai beserta labelnya dibaca apa adanya dari `pyStandardValue` dan
// `pyLocalizedValue` pada `Property/PilihanCompliance_property.xml`. Tidak ada yang
// ditambahkan, tidak ada yang dibuang.
const (
	ChoiceFraud     = "0" // Fraud / Tolak
	ChoiceValid     = "1" // Bayar / Valid
	ChoicePostAudit = "2" // Bayar / PostAudit
	ChoiceOther     = "3" // Lain-Lain
)

// Choice adalah satu pilihan beserta labelnya sebagaimana dibaca petugas.
type Choice struct {
	// Value adalah nilai yang tersimpan — `pyStandardValue`.
	Value string

	// Label adalah teks yang dilihat pengguna — `pyLocalizedValue`, dibawa apa adanya
	// karena `D-80` menetapkan teks layar mengikuti XML Pega.
	Label string
}

// choices adalah keempat pilihan dalam urutan nilainya di properti.
//
// Urutan itu bukan selera: `pyStandardValue` mencantumkannya `0,1,2,3`, dan layar Pega
// menggambar daftar pilihan dalam urutan yang sama.
var choices = []Choice{
	{Value: ChoiceFraud, Label: "Fraud / Tolak"},
	{Value: ChoiceValid, Label: "Bayar / Valid"},
	{Value: ChoicePostAudit, Label: "Bayar / PostAudit"},
	{Value: ChoiceOther, Label: "Lain-Lain"},
}

// Choices mengembalikan keempat pilihan Compliance.
//
// Salinan, bukan senarai aslinya: pemanggil yang mengurutkannya ulang tidak boleh mengubah
// urutan yang dilihat pemanggil berikutnya.
func Choices() []Choice {
	out := make([]Choice, len(choices))
	copy(out, choices)
	return out
}

// FindChoice mencari satu pilihan menurut nilainya.
func FindChoice(value string) (Choice, bool) {
	for _, c := range choices {
		if c.Value == value {
			return c, true
		}
	}
	return Choice{}, false
}

// CheckerCase adalah satu klaim sebagaimana dibuka form Compliance Checker.
//
// Ia MEMBUNGKUS WorkItem alih-alih mengulang kolomnya: baris yang dibuka form ini adalah
// baris yang sama dengan yang tampil di tab Compliance, dan menyalin kolomnya ke tipe kedua
// akan membuat keduanya dapat berbeda.
type CheckerCase struct {
	// Claim adalah baris antrean yang dibuka, lengkap dengan Aging.
	Claim WorkItem

	// Decision adalah keputusan yang SUDAH pernah disimpan atas klaim ini, bila ada.
	//
	// Pointer supaya "belum pernah diputuskan" dapat dibedakan dari "diputuskan dengan
	// pilihan 0" — dan keduanya memang berbeda, karena `0` berarti Fraud/Tolak.
	Decision *Decision
}

// Decision adalah keputusan Compliance atas satu klaim.
//
// # Kolomnya mengikuti apa yang `SetComplianceResult` tulis
//
//	langkah 1     .ClaimData.PilihanCompliance         → Choice
//	langkah 5     .ClaimData.ComplianceRemark          → Remarks
//	langkah 2     .CPLValidDate := now  bila pilihan 1 → ValidatedAt
//	langkah 5     .TanggalKirimPostAudit := now
//	              .ClaimData.PostAudtiTfAnalyst := now bila pilihan 2 → SentToPostAuditAt
//
// `NotePilihanCompliance` ikut dibawa karena `CatatanToAnalyst_Section` mengikatnya
// berdampingan dengan PilihanCompliance, dengan caption "Note PilihanCompliance".
type Decision struct {
	// Reference adalah `PZINSKEY` klaimnya — kunci yang sama dengan kolom tersembunyi
	// pada baris tab Compliance.
	Reference string

	// Choice adalah salah satu dari keempat nilai di atas.
	Choice string

	// Note adalah `.ClaimData.NotePilihanCompliance`.
	Note string

	// Remarks adalah `.ClaimData.ComplianceRemark`, yang `SetComplianceResult` langkah 5
	// salin ke `childPageCompliance.ComplianceRemarks`.
	Remarks string

	// DecidedBy adalah login petugas yang memutuskan.
	//
	// Pega tidak menyimpannya pada klaim — ia hanya menulis `OperatorID.pyUserIdentifier`
	// ke baris riwayat lewat `InsertHistoryClaimPNC`. Di sini ia disimpan pada barisnya
	// sendiri, karena `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang dan
	// keputusan ini menyangkut uang: `ChoiceFraud` menolak klaim.
	DecidedBy string

	// DecidedAt adalah waktu keputusan, dari seam Clock.
	DecidedAt time.Time

	// ValidatedAt terisi HANYA pada ChoiceValid — padanan `.CPLValidDate`.
	ValidatedAt *time.Time

	// SentToPostAuditAt terisi HANYA pada ChoicePostAudit — padanan
	// `.TanggalKirimPostAudit` dan `.ClaimData.PostAudtiTfAnalyst`, yang
	// `SetComplianceResult` langkah 5 isi keduanya dengan waktu yang sama.
	SentToPostAuditAt *time.Time
}

// StatusNote adalah keterangan yang masuk baris riwayat, meniru `InsertHistoryClaimPNC`
// pada langkah 6–8 `SetComplianceResult`.
//
// Ketiga kalimatnya dibaca dari parameter `statusNote` ketiga langkah itu. Langkah 3.3
// menangani `pilihan == 3`, dan keterangan langkah itu di Pega berbunyi "Set Status Post
// Audit" — KELIRU, karena Post Audit adalah `2`. Nilai `3` berlabel "Lain-Lain" menurut
// propertinya, dan itulah yang dipakai di sini.
func (d Decision) StatusNote() string {
	switch d.Choice {
	case ChoiceFraud:
		return "Compliance FRAUD"
	case ChoiceValid:
		return "Compliance Valid"
	case ChoicePostAudit:
		return "Compliance Post Audit"
	case ChoiceOther:
		return "Compliance Lain-Lain"
	}
	return ""
}

// DecisionInput adalah isian mentah dari layar, belum divalidasi.
type DecisionInput struct {
	// Reference adalah `PZINSKEY` klaim yang diputuskan.
	Reference string

	// Choice adalah nilai Pilihan Compliance yang dipilih.
	Choice string

	// Note adalah Note PilihanCompliance.
	Note string

	// Remarks adalah Catatan Dari Compliance.
	Remarks string
}

// Panjang maksimum kedua kotak teks.
//
// # Kenapa angkanya DITEBAK, dan kenapa itu tetap lebih baik daripada tanpa batas
//
// Properti `NotePilihanCompliance` dan `ComplianceRemark` keduanya **tidak ada di export**,
// sehingga lebar kolomnya tidak terbaca. Yang dipakai adalah 4000 — sama dengan
// `REMARKS VARCHAR2(4000 BYTE)` pada `T_CLAIM_COMPLIANCE_H`, satu-satunya kolom catatan di
// jalur ini yang lebarnya DAPAT dibaca.
//
// Bila kelak DDL-nya tiba dan angkanya lebih kecil, yang terjadi adalah ORA-12899 — galat
// yang tidak dapat ditampilkan kepada pengguna karena ia menyebut nama tabel dan kolom
// (`11-CROSSCUTTING.md` §1.2). Jadi batas yang ditebak terlalu besar tetap menyisakan
// kemungkinan galat itu; yang ia tutup hanyalah masukan yang jelas-jelas kelewatan.
//
// Satuannya BYTE, bukan karakter — sama alasannya dengan RemarksMaxLength.
const (
	NoteMaxLength            = 4000
	ComplianceRemarkMaxLength = 4000
)

// Nama isian tambahan yang dapat ditunjuk pelanggaran pada form ini.
const (
	FieldChoice           = "pilihan"
	FieldNote             = "note"
	FieldComplianceRemark = "catatan_compliance"
)

// NewDecision membentuk keputusan yang sah dari isian layar, atau menyatakan apa yang salah.
//
// Klaimnya diserahkan pemanggil dalam bentuk WorkItem yang SUDAH dipastikan berada di
// antrean Compliance — pemastian itu tugas usecase, sama seperti pada NewPostAuditEntry.
//
// Seluruh pelanggaran dikumpulkan, bukan yang pertama saja (`P-5`): form ini punya tiga
// isian yang dapat dilanggar sekaligus.
func NewDecision(
	input DecisionInput, claim WorkItem, decidedBy string, decidedAt time.Time,
) (Decision, error) {
	var violations []Violation

	choice := strings.TrimSpace(input.Choice)
	note := strings.TrimSpace(input.Note)
	remarks := strings.TrimSpace(input.Remarks)

	// Pilihan WAJIB, dan ini satu-satunya kewajiban yang ditegakkan di sini.
	//
	// Bukan karena ada bukti di export bahwa Pega mewajibkannya — section-nya hilang,
	// sehingga `pyRequired` tidak terbaca. Melainkan karena `SetComplianceResult` tidak
	// punya cabang untuk pilihan kosong: langkah 2, 3, dan 5 seluruhnya berprasyarat pada
	// nilai tertentu, sehingga menyimpan tanpa pilihan menghasilkan baris riwayat tanpa
	// keterangan dan klaim yang statusnya berubah menjadi 1151 tanpa satu pun akibat.
	//
	// Itu keadaan yang lebih buruk daripada menolak, dan penolakannya dicatat di sini
	// sebagai aturan yang DITAMBAHKAN — bukan dibaca.
	if choice == "" {
		violations = append(violations, Violation{
			Field:   FieldChoice,
			Message: "Pilihan Compliance belum dipilih.",
		})
	} else if _, known := FindChoice(choice); !known {
		violations = append(violations, Violation{
			Field:   FieldChoice,
			Message: "Pilihan Compliance tidak dikenal.",
		})
	}

	if len(note) > NoteMaxLength {
		violations = append(violations, Violation{
			Field: FieldNote,
			Message: "Note terlalu panjang. Maksimum " +
				strconv.Itoa(NoteMaxLength) + " karakter.",
		})
	}

	if len(remarks) > ComplianceRemarkMaxLength {
		violations = append(violations, Violation{
			Field: FieldComplianceRemark,
			Message: "Catatan terlalu panjang. Maksimum " +
				strconv.Itoa(ComplianceRemarkMaxLength) + " karakter.",
		})
	}

	if len(violations) > 0 {
		return Decision{}, NewValidationError(violations)
	}

	decision := Decision{
		Reference: claim.Reference,
		Choice:    choice,
		Note:      note,
		Remarks:   remarks,
		DecidedBy: decidedBy,
		DecidedAt: decidedAt,
	}

	// Kedua stempel waktu di bawah diisi DI SINI, bukan di basis data, karena keduanya
	// aturan bisnis: `SetComplianceResult` mengisinya lewat prasyarat langkah, bukan lewat
	// default kolom. Memindahkannya ke SQL akan menyembunyikan aturannya di tempat yang
	// `D-02` larang menyimpan logika.
	switch choice {
	case ChoiceValid:
		at := decidedAt
		decision.ValidatedAt = &at
	case ChoicePostAudit:
		at := decidedAt
		decision.SentToPostAuditAt = &at
	}

	return decision, nil
}
