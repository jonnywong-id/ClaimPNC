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

// Comment adalah satu baris pada grid komentar — `.ClaimData.ComplianceList`, kelas
// `ASM-FW-GCNMFW-Data-Compliance`.
//
// # Inilah tempat petugas Compliance menulis, bukan ComplianceRemark
//
// `Section/CompliancePNC-Section.xml` memasang dua kotak teks panjang yang mudah tertukar,
// dan hanya SATU di antaranya dapat diisi:
//
//	.ClaimData.ComplianceRemark   "Keterangan dari Investigator"   pyEditOptions=Read-only
//	.ComplianceList(n).Compliance "Komentar"                       dapat disunting, bergrid
//
// Yang pertama MENAMPILKAN catatan Investigator; yang kedua yang diisi petugas. Kekeliruan
// itu sempat terbawa ke sini — form versi sebelumnya menjadikan ComplianceRemark kotak
// isian — dan dikoreksi setelah Section-nya terbaca.
type Comment struct {
	// Index adalah urutan baris pada grid, mulai dari 1.
	//
	// Ia disimpan, bukan diturunkan dari urutan baca, karena grid Pega bernomor
	// (`pyGridNumbering=true`) dan nomor itu yang dilihat petugas.
	Index int

	// Date adalah Tanggal Komentar — `.ComplianceDate`.
	//
	// Pada baris baru, Pega mengisinya dengan `@(Pega-RULES:DateTime).CurrentDateTime()`
	// sebagai nilai bawaan yang MASIH dapat diubah petugas. Jadi ia bukan stempel waktu
	// sistem: dua baris dapat bertanggal sama, dan baris kemarin dapat ditulis hari ini.
	Date time.Time

	// Text adalah isi komentarnya — `.Compliance`, kotak teks panjang.
	Text string
}

// CommentInput adalah satu baris grid komentar sebagaimana dikirim layar.
type CommentInput struct {
	// Date kosong berarti layar tidak mengirim tanggal, dan nilai bawaan Pega dipakai —
	// waktu saat keputusan disimpan.
	Date *time.Time

	// Text adalah isi komentarnya.
	Text string
}

// Lini bisnis yang menentukan tombol mana yang muncul pada form Compliance Checker.
//
// Keduanya dibaca dari ekspresi yang benar-benar dieksekusi Pega, bukan dari labelnya:
//
//	When/IsPA-When.xml       pyLogic "A"       A: Policy.Quotation.GroupPanel = "002"
//	When/IsTravel-When.xml   pyLogic "A OR B"  A: Policy.Quotation.GroupPanel = "005"
//	                                           B: ClaimData.PolicyData.Quotation
//	                                              .GroupPanel = "005"
//
// `IsTravel` memeriksa DUA jalur halaman lalu meng-OR-kannya karena snapshot polis di
// klipboard Pega dapat berada di salah satu dari keduanya. Di sini nilainya satu kolom,
// sehingga kedua cabang itu runtuh menjadi satu perbandingan — bukan penyederhanaan,
// melainkan akibat tidak adanya dua klipboard.
//
// Satu jebakan yang sengaja TIDAK diikuti: `pyConditionString` pada `IsTravel` berbunyi
// `Kode Bisnis = "77"`. Itu label tampilan yang BASI — ia hanya satu sementara kondisinya
// dua, dan angka 77 tidak muncul di mana pun lagi dalam berkas itu. Yang dieksekusi adalah
// `pyConditionValue1`, dan keduanya tegas `"005"`.
const (
	GroupPanelPersonalAccident = "002"
	GroupPanelTravel           = "005"
)

// FormActions adalah tombol mana yang boleh tampil pada form Compliance Checker.
//
// # Pega punya DUA bilah tombol, bukan satu
//
// Kelima tombol yang sama muncul di dua tempat, dan syarat WADAH-nya saling melengkapi:
//
//	Section/CompliancePNC-Section.xml      layout S14, sel 76–80, wadah `!IsTravel`
//	Section/ComplianceChecker-Section.xml  layout S4,  sel 15–19, wadah `IsTravel`
//
// Jadi setiap klaim selalu mendapat tepat satu bilah. Syarat per tombolnya identik di
// keduanya:
//
//	Unggah Dokumen           pyVisible=ALWAYS
//	Download Dokumen Reject  pyVisible=ALWAYS
//	Simpan Data              pyVisible=ALWAYS
//	Kirim ke Analyst         pyVisible=OTHER  pyCondition=IsPA
//	Kirim ke PIC Teknik      pyVisible=OTHER  pyCondition=IsTravel
//
// # Koreksi atas versi pertama berkas ini
//
// Versi pertama membaca S14 SAJA, lalu menyimpulkan `!IsTravel` sebagai syarat ketiga
// tombol umum — sehingga klaim Travel digambar TANPA tombol Simpan sama sekali, dan
// kesimpulan itu bahkan sempat ditulis sebagai catatan lingkup penguji. Ia **salah**:
// yang `!IsTravel` adalah wadahnya, bukan tombolnya, dan wadah pasangannya ada di section
// induk yang saat itu saya kira hilang dari export (`R-16`). `CompliancePNC-Section.xml`
// ADA, 374 KB.
//
// Pelajarannya dicatat karena berulang: **syarat wadah bukan syarat isi.** Membaca satu
// layout tanpa induknya menghasilkan aturan yang terbalik justru pada lini yang paling
// jarang diuji.
type FormActions struct {
	// UploadDocument — "Unggah Dokumen". Syarat Pega: selalu.
	UploadDocument bool

	// DownloadRejectLetter — "Download Dokumen Reject". Syarat Pega: selalu.
	DownloadRejectLetter bool

	// Save — "Simpan Data". Syarat Pega: selalu.
	Save bool

	// SendToAnalyst — "Kirim ke Analyst". Syarat Pega: `IsPA`.
	SendToAnalyst bool

	// SendToTechnician — "Kirim ke PIC Teknik". Syarat Pega: `IsTravel`.
	//
	// Tombolnya digambar, tetapi apa yang dikerjakannya BELUM dibangun: Data Transform
	// `SendToPIC` yang dipanggilnya tidak ada di export (`R-16`), sehingga akibatnya pada
	// klaim tidak dapat ditiru — hanya ditebak. Layar karena itu menggambar tombolnya dan
	// menolak menjalankannya, bukan menyembunyikannya.
	//
	// Menyembunyikannya akan membuat petugas Travel mengira modulnya belum selesai;
	// menjalankannya dengan tebakan akan memindahkan klaim ke tempat yang belum tentu
	// benar. Yang pertama menyesatkan, yang kedua merusak data.
	SendToTechnician bool
}

// ActionsFor menentukan tombol yang tampil untuk satu klaim.
//
// GroupPanel kosong — yang mungkin terjadi karena `find_compliance_claim` meng-`LEFT JOIN`
// tabel klaim — membuat IsTravel dan IsPA sama-sama salah. Itu persis perilaku Pega:
// `compareTwoValues("", "=", "005")` bernilai salah. Akibatnya klaim tanpa lini bisnis
// mendapat ketiga tombol umum, dan tidak mendapat kedua tombol pengiriman.
func ActionsFor(claim WorkItem) FormActions {
	return FormActions{
		UploadDocument:       true,
		DownloadRejectLetter: true,
		Save:                 true,
		SendToAnalyst:        claim.GroupPanel == GroupPanelPersonalAccident,
		SendToTechnician:     claim.GroupPanel == GroupPanelTravel,
	}
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

// Actions adalah tombol yang tampil untuk klaim ini.
func (c CheckerCase) Actions() FormActions { return ActionsFor(c.Claim) }

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
	//
	// Layar hanya menampilkannya ketika Choice bernilai ChoiceOther — syarat
	// `.ClaimData.PilihanCompliance==3` pada selnya. Nilainya tetap DISIMPAN apa pun
	// pilihannya, meniru Pega: di sana kondisinya sisi klien, sehingga isian yang
	// tersembunyi tetap ikut tersimpan saat Obj-Save.
	Note string

	// Comments adalah grid komentar — `.ClaimData.ComplianceList`.
	//
	// Inilah tempat petugas menulis. Lihat catatan pada tipe Comment.
	Comments []Comment

	// Remarks adalah `.ClaimData.ComplianceRemark` — catatan **Investigator**, yang pada
	// form ini READ-ONLY dan hanya ditampilkan.
	//
	// Ia tetap ikut disimpan pada keputusan karena `SetComplianceResult` langkah 5
	// menyalinnya ke `childPageCompliance.ComplianceRemarks`, yang menjadi kolom "Catatan"
	// pada tab Post Audit. Sumbernya KLAIM, bukan isian layar — karena itu ia tidak ada
	// di DecisionInput.
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

// StatusNote adalah keterangan yang masuk baris riwayat — parameter `statusNote` pada
// `Call InsertHistoryClaimPNC`, langkah 16–18 `SetComplianceResult`.
//
// Ketiga kalimatnya dibaca HARFIAH dari parameter langkah itu:
//
//	pilihan "0"  "Send by Compliance to Analyst and CPL Status is FRAUD"
//	pilihan "1"  "Send by Compliance to Analyst and CPL Status is Valid"
//	pilihan "2"  "Send by Compliance to Analyst and CPL Status is Post Audit"
//
// # Lain-Lain tidak punya kalimat, dan itu BUKAN kelalaian di sini
//
// Pega tidak punya langkah riwayat ber-syarat `pilihan == "3"` sama sekali, sehingga
// memilih Lain-Lain **tidak menulis baris riwayat apa pun**. Itu perilaku Pega, dan
// Work Owner memutuskan ditiru apa adanya (2026-10-07).
//
// Nilai kosong di sini berarti "tidak ada baris riwayat", dan pemanggil WAJIB
// memperlakukannya begitu — bukan menulis baris berketerangan kosong.
//
// # Kekeliruan yang dikoreksi
//
// Versi sebelumnya mengarang kalimatnya sendiri — "Compliance FRAUD", "Compliance Valid",
// dan seterusnya — beserta satu kalimat untuk Lain-Lain yang di Pega tidak ada. Ia ditulis
// sebelum parameter langkahnya terbaca, dan ketiganya salah. Baris riwayat adalah jejak
// audit; keterangan yang dikarang membuat jejak itu bercerita hal yang tidak terjadi.
func (d Decision) StatusNote() string {
	switch d.Choice {
	case ChoiceFraud:
		return "Send by Compliance to Analyst and CPL Status is FRAUD"
	case ChoiceValid:
		return "Send by Compliance to Analyst and CPL Status is Valid"
	case ChoicePostAudit:
		return "Send by Compliance to Analyst and CPL Status is Post Audit"
	}
	return ""
}

// DecisionInput adalah isian mentah dari layar, belum divalidasi.
type DecisionInput struct {
	// Reference adalah `PZINSKEY` klaim yang diputuskan.
	Reference string

	// Choice adalah nilai Pilihan Compliance yang dipilih.
	Choice string

	// Note adalah Note Lainya — `.ClaimData.NotePilihanCompliance`.
	Note string

	// Comments adalah baris-baris grid komentar yang dikirim layar.
	//
	// Baris yang teksnya kosong DIBUANG, tidak ditolak: grid Pega menambah baris kosong
	// setiap kali petugas menekan Enter, sehingga baris kosong di ujung adalah kejadian
	// normal — bukan kesalahan yang perlu diadukan.
	Comments []CommentInput

	// Action menyatakan TOMBOL MANA yang ditekan. Lihat konstanta di bawah.
	//
	// Kosong diperlakukan sebagai ActionSave — bentuk yang paling tidak berakibat, dan
	// yang menjaga klien lama tetap bekerja.
	Action string
}

// Aksi pada form Compliance Checker — padanan tombol mana yang ditekan.
//
// # Kenapa keduanya DIBEDAKAN, dan kenapa itu penting
//
// Ketiga tombol Pega menjalankan hal yang berbeda, terbaca dari `pyActionSets` selnya:
//
//	sel 78  Simpan Data           save  +  InsertHistoryClaimPNC_compilance
//	                              TANPA SetComplianceResult
//	sel 79  Kirim ke Analyst      save  +  SetComplianceResult
//	sel 80  Kirim ke PIC Teknik           SetComplianceResult
//	                              +  InsertHistoryClaimPNC ("Send by Compliance to PIC Teknis")
//
// Yang memasang Ticket — dan karenanya memindahkan klaim keluar dari antrean Compliance —
// hanyalah `SetComplianceResult`. Jadi **"Simpan Data" TIDAK memindahkan klaim.**
//
// Versi pertama modul ini menjalankan seluruh akibat pada satu jalur simpan, sehingga
// menekan "Simpan Data" memindahkan klaim keluar dari antrean. Itu perilaku tombol
// "Kirim", bukan "Simpan" — dan akibatnya tidak dapat dibatalkan petugas, karena formnya
// tidak dapat dibuka lagi setelah klaimnya berpindah.
const (
	// ActionSave — "Simpan Data". Keputusan tersimpan, klaim TETAP di antrean.
	ActionSave = "simpan"

	// ActionSend — "Kirim ke Analyst" atau "Kirim ke PIC Teknik".
	//
	// SATU nilai untuk kedua tombol, karena `SetComplianceResult` yang sama dipanggil
	// keduanya dan tujuannya ditentukan **lini bisnis**, bukan tombolnya. Lihat
	// NewAssignmentMove.
	ActionSend = "kirim"
)

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
	NoteMaxLength = 4000

	// CommentMaxLength DIBACA, bukan ditebak: `pyMaxLength` pada properti `Compliance`
	// kelas `ASM-FW-GCNMFW-Data-Compliance` bernilai **512**, dikuatkan `pzEntryCode`
	// `sTN512`.
	//
	// Angka itu BARU — catatan pengembangnya berbunyi "extend max lenght", diubah
	// 8 April 2026 langsung di sistem produksi, dari versi 01-01-03 ke 01-03-15. Jadi
	// ia memang pernah diperpanjang, dan dapat diperpanjang lagi.
	CommentMaxLength = 512
)

// CommentMaxRows adalah batas jumlah baris grid komentar yang diterima satu permintaan.
//
// Grid Pega tidak membatasinya — `pyPageSize=20` hanya mengatur paginasi tampilan, bukan
// jumlah baris. Batas ini DITAMBAHKAN, bukan dibaca: tanpa batas apa pun, satu permintaan
// dapat mengirim sejuta baris dan menahan koneksi basis data selama penyisipannya.
const CommentMaxRows = 200

// Nama isian tambahan yang dapat ditunjuk pelanggaran pada form ini.
const (
	FieldChoice   = "pilihan"
	FieldNote     = "note"
	FieldComments = "komentar"
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

	comments, commentViolations := buildComments(input.Comments, decidedAt)
	violations = append(violations, commentViolations...)

	if len(violations) > 0 {
		return Decision{}, NewValidationError(violations)
	}

	decision := Decision{
		Reference: claim.Reference,
		Choice:    choice,
		Note:      note,
		Comments:  comments,
		DecidedBy: decidedBy,
		DecidedAt: decidedAt,

		// Sumbernya KLAIM, bukan layar: `.ClaimData.ComplianceRemark` adalah catatan
		// Investigator yang pada form ini read-only. Lihat catatan pada Decision.Remarks.
		Remarks: claim.ComplianceRemarks,
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

// buildComments menyaring dan menomori baris grid komentar.
//
// # Tiga aturan, dan hanya yang ketiga yang ditambahkan
//
//  1. Baris berteks kosong DIBUANG tanpa diadukan. Grid Pega menambah baris setiap kali
//     petugas menekan Enter, sehingga baris kosong di ujung adalah kejadian normal.
//  2. Tanggal yang tidak dikirim diisi waktu keputusan — padanan nilai bawaan
//     `@(Pega-RULES:DateTime).CurrentDateTime()` pada sel Tanggal Komentar.
//  3. Jumlah baris dibatasi CommentMaxRows. Ini BUKAN aturan Pega; alasannya ada pada
//     konstanta itu.
//
// Penomorannya mengikuti urutan setelah penyaringan, bukan sebelum: nomor yang dilihat
// petugas pada grid Pega juga berurutan tanpa lubang.
func buildComments(inputs []CommentInput, decidedAt time.Time) ([]Comment, []Violation) {
	if len(inputs) > CommentMaxRows {
		return nil, []Violation{{
			Field: FieldComments,
			Message: "Komentar terlalu banyak. Maksimum " +
				strconv.Itoa(CommentMaxRows) + " baris sekali simpan.",
		}}
	}

	var (
		comments   []Comment
		violations []Violation
	)

	for _, input := range inputs {
		text := strings.TrimSpace(input.Text)
		if text == "" {
			continue
		}

		if len(text) > CommentMaxLength {
			violations = append(violations, Violation{
				Field: FieldComments,
				Message: "Komentar baris ke-" + strconv.Itoa(len(comments)+1) +
					" terlalu panjang. Maksimum " +
					strconv.Itoa(CommentMaxLength) + " karakter.",
			})
			continue
		}

		date := decidedAt
		if input.Date != nil {
			date = *input.Date
		}

		comments = append(comments, Comment{
			Index: len(comments) + 1,
			Date:  date,
			Text:  text,
		})
	}

	return comments, violations
}
