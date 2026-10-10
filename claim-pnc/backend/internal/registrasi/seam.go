package registrasi

import (
	"context"
	"time"
)

// # Seam modul Registrasi
//
// Seluruh antarmuka di bawah ini dideklarasikan di paket yang MEMAKAINYA, bukan di paket
// yang mengisinya (`docs/Steering/08-TECHNICAL-STRATEGY.md` §4.2). Bentuknya ditentukan
// kebutuhan aturan bisnis, bukan kemampuan basis data.

// ClaimRepo adalah seam ke penyimpanan klaim.
type ClaimRepo interface {
	// Save menuliskan klaim sebagai SATU transaksi utuh: klaim, objek, coverage, dan
	// spreading tersimpan bersama atau tidak sama sekali (`TKT-F2-003`).
	Save(ctx context.Context, k Claim) error

	Get(ctx context.Context, id string) (Claim, error)
	GetByNumber(ctx context.Context, number string) (Claim, error)

	// FindDuplicates mencari klaim lain yang memenuhi salah satu kunci duplikasi.
	//
	// exceptID membuat penyimpanan ulang klaim yang sama tidak dianggap duplikat
	// terhadap dirinya sendiri. Klaim yang ditandai terhapus (`ADR-0012`) TIDAK ikut
	// terhitung, dan pencarian mencakup klaim yang dibuat KEDUA sistem selama masa
	// paralel — nomor `PNC-xxxx` dari Pega maupun `PNCN.YY.xxxx` dari sini.
	FindDuplicates(ctx context.Context, key []DuplicateKey, exceptID string) ([]DuplicateClaim, error)

	// SaveCommitteeNote menuliskan isian modal "Transfer Claim ke Komite" satu jaminan.
	// object dan coverage berbasis 1 (URUTAN_OBJEK, URUTAN). InitialName dan CommitteeDate
	// tidak ditulis.
	SaveCommitteeNote(ctx context.Context, claimID string, object, coverage int, n CommitteeNote) error
}

// TaskRepo adalah seam ke penyimpanan tugas.
type TaskRepo interface {
	Save(ctx context.Context, t Task) error
	Get(ctx context.Context, id string) (Task, error)

	// OpenTaskForClaim mengembalikan tugas yang masih menunggu untuk sebuah klaim.
	// Sebuah klaim hanya boleh punya satu tugas terbuka pada satu waktu.
	OpenTaskForClaim(ctx context.Context, claimID string) (Task, error)

	// Inbox mengembalikan tugas yang layak muncul di layar seorang pengguna:
	// tugas Worklist miliknya, ditambah tugas Workbasket yang belum bertuan pada
	// antrean yang ia berwenang (`D-79`), ditambah tugas Worklist terbuka pada tahap
	// yang boleh dikerjakan grupnya (M_LOGIN_GROUP_PNC — lihat access.go).
	Inbox(ctx context.Context, operator string, workbasket, stages []string) ([]Task, error)
}

// InboxMirror adalah seam ke daftar kerja yang dibaca layar My Inbox
// (POOLDATA.T_CLAIMLIST_ADMIN) — lihat inbox_entry.go.
//
// Ia dipanggil DI DALAM transaksi yang sama dengan penyimpanan klaim dan tugasnya, sehingga
// baris daftar kerja tidak pernah tertinggal dari keadaan klaim: gagal menulisnya
// menggagalkan seluruh perubahan.
type InboxMirror interface {
	Mirror(ctx context.Context, e InboxEntry) error
}

// AccountDirectory membaca Master Rekening (POOLDATA.LST_ACCOUNT) untuk isian No Rekening
// penerima klaim — pengganti activity `GetDataBankMaster` yang tidak ada di export.
type AccountDirectory interface {
	// FindAccount mengembalikan rekening bernomor itu, atau ErrAccountNotFound. Nomor yang
	// sama di lebih dari satu bank mengembalikan yang BANKID-nya terkecil.
	FindAccount(ctx context.Context, number string) (BankAccount, error)
}

// PolicyRepo adalah seam ke snapshot polis.
//
// Kepemilikan data polis ada pada GISFW (`ADR-0006`); modul ini hanya MEMBACA. Seam-nya
// sempit dengan sengaja: apa pun yang dibutuhkan registrasi harus muat di dalam Polis,
// dan bila tidak muat itu pertanda batas konteks sedang dilanggar.
type PolicyRepo interface {
	Get(ctx context.Context, policyNumber string) (Policy, error)
}

// NumberIssuer adalah seam ke generator nomor klaim (`TKT-F2-006`, `ADR-0009`).
//
// Nomor klaim TIDAK DAPAT ditarik kembali setelah terbit — ia muncul di surat ke
// tertanggung dan di PLA/DLA ke reasuransi. Karena itu penerbitannya berada di balik
// seam tersendiri, bukan di dalam repo klaim: pemanggilnya harus terlihat.
type NumberIssuer interface {
	// Issue mengembalikan satu nomor baru berformat PNCN.YY.xxxx (`D-71`).
	Issue(ctx context.Context, at time.Time) (string, error)
}

// Parameter adalah seam ke nilai bisnis yang di sistem lama tertanam di dalam rule
// (`D-15`, `ADR-0025`).
//
// Dua nilai yang lewat sini — ambang Notice of Large Losses dan daftar penerimanya —
// di `Activity/InputRegister_act-Act.xml` ditulis sebagai konstanta `1000000000` pada
// langkah 53 dan sebagai tujuh alamat surel dalam satu string pada langkah 53.1.
// Mengubahnya di sana menuntut deployment.
//
// Modul `F-4` yang akan mengisinya dari master belum ada; pengisi sementara ada di
// repo/memori, di luar lapisan aturan. Yang penting: TIDAK ADA satu pun alamat surel di
// dalam paket ini.
type Parameter interface {
	// LargeLossThreshold mengembalikan nilai estimasi yang memicu Notice of Large Losses.
	LargeLossThreshold(ctx context.Context) (Money, error)

	// LargeLossRecipients mengembalikan penerima pemberitahuan untuk satu lini bisnis.
	//
	// `D-67` melarang akun pribadi sebagai penerima; sistem lama memuat sedikitnya enam
	// akun Gmail pribadi di jalur produksi. Penegakannya ada di pengisi seam ini.
	LargeLossRecipients(ctx context.Context, line LineOfBusiness) ([]string, error)
}

// ExchangeRateSource adalah seam ke kurs valuta asing.
//
// `ADR-0015` menetapkan kurs yang dipakai adalah kurs TANGGAL KEJADIAN, dan kurs yang
// tidak ditemukan MENOLAK klaim alih-alih memakai nilai bawaan.
type ExchangeRateSource interface {
	Find(ctx context.Context, currency string, date time.Time) (ExchangeRate, error)
}

// Assigner adalah seam ke aturan routing (`ADR-0019`, `TKT-B06-002`).
//
// Tiga aturan yang dipakai alur ini tidak ada di export (`R-04`): PNCAdminRouter,
// PNCTeknikRouter, dan RouterRCLDoctor. Algoritma pembagian bebannya terbaca dari tempat
// lain — `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` mengurutkan dengan
// `ORDER BY counter_quota ASC`, yakni petugas dengan beban paling sedikit mendapat tugas
// berikutnya.
type Assigner interface {
	// Assign memilih penerima tugas untuk sebuah tahap.
	//
	// Untuk tahap Workbasket, ia mengembalikan nama antreannya dan tidak memilih orang.
	Assign(ctx context.Context, stage Stage, claim Claim, caller string) (Assignee, error)
}

// AttendanceSource adalah seam ke absensi PIC — Connect REST `ServiceGetDataAbsenPIC`
// (`GET .../HCC/Absen/attendance/{PIC}/{yyyyMMdd}?caseId=`).
//
// Galat dikembalikan apa adanya; pemanggil memperlakukannya seperti Pega (step 15.6 menelan
// galat penguraian), yaitu sebagai absensi kosong.
type AttendanceSource interface {
	Attendance(ctx context.Context, operator string, date time.Time, claimNumber string) (Attendance, error)
}

// NotificationKind menamai peristiwa yang layak diberitahukan ke luar modul.
type NotificationKind string

const (
	// NotificationLargeLoss adalah Notice of Large Losses — pemberitahuan wajib ke
	// Underwriting dan jajaran pimpinan saat estimasi melampaui ambang.
	NotificationLargeLoss NotificationKind = "NOTICE_OF_LARGE_LOSSES"

	// NotificationClaimRegistered dikirim saat nomor klaim terbit.
	NotificationClaimRegistered NotificationKind = "KLAIM_TERDAFTAR"
)

// Notification adalah peristiwa yang dikirim keluar modul.
//
// Modul ini TIDAK mengirim surel sendiri — pengiriman milik `S-3`. Yang dilakukan di
// sini hanya menerbitkan peristiwanya, dengan penerima yang sudah diambil dari master.
type Notification struct {
	Kind         NotificationKind
	ClaimNumber  string
	PolicyNumber string
	Recipients   []string

	// RupiahValue adalah nilai estimasi setelah konversi kurs, dalam sen.
	RupiahValue Money

	// Revision menandai pemberitahuan KEDUA dan seterusnya atas klaim yang sama.
	//
	// Sistem lama membedakan keduanya lewat subjek surel, bukan lewat penerima:
	// `Activity/SendEmailLargeLoss_act.xml` langkah 7 memakai subjek
	// "NOTICE OF LARGE LOSSES" ketika `ClaimData.FlagNOLL` masih kosong, dan langkah 8
	// memakai "NOTICE OF LARGE LOSSES (REVISE)" ketika ia sudah bernilai "1".
	//
	// Perbedaannya bukan kosmetik: penerima membaca subjek untuk tahu apakah angka yang
	// dikirim menggantikan angka sebelumnya.
	Revision bool

	At time.Time
}

// Notifier adalah seam ke pemberitahuan (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3.6).
//
// # Kontrak yang mengikat pengisinya
//
// Kirim dipanggil DI DALAM transaksi penyimpanan klaim, dan karena itu ia wajib
// MENCATAT peristiwa, bukan mengirimkannya. Pengisi yang membuka koneksi SMTP di sini
// akan membuat surel yang gagal terkirim membatalkan pendaftaran klaim yang sah — akibat
// yang jauh lebih buruk daripada pemberitahuan yang terlambat.
//
// Pengirimannya sendiri milik `S-3`, yang membaca catatan itu di luar transaksi. Yang
// dijamin batas transaksi adalah janji `TKT-B02-004`: klaim yang melampaui ambang
// menerbitkan TEPAT SATU peristiwa Notice of Large Losses — tidak nol karena surel
// gagal, tidak dua karena permintaan diulang.
type Notifier interface {
	Send(ctx context.Context, p Notification) error
}

// AuditTrail adalah satu baris jejak yang tidak pernah diubah maupun dihapus
// (`ADR-0026`).
type AuditTrail struct {
	ClaimID     string
	ClaimNumber string
	Event       string
	Actor       string
	At          time.Time
	Note        string
}

// AuditRecorder adalah seam ke jejak audit.
type AuditRecorder interface {
	Record(ctx context.Context, j AuditTrail) error
}

// IDGenerator adalah seam ke pembangkit pengenal internal.
//
// Ia terpisah dari NumberIssuer dengan sengaja: pengenal internal boleh acak dan tidak
// punya makna bisnis, sedangkan nomor klaim berurut, terlihat pengguna, dan tidak dapat
// ditarik kembali.
type IDGenerator interface {
	New() string
}

// ClaimReportLink menautkan klaim kembali ke berkas Receive Document asalnya.
//
// # Kenapa seam ini ada, dan apa yang gagal tanpanya
//
// Berkas laporan menentukan posisinya dari DUA kolom pada
// `POOLDATA.T_CLAIM_RECIVEDCLAIM`, dan ketiga keadaannya persis yang dipakai layar lama:
//
//	NOKLAIM kosong, TRANSFERASM kosong  → Not Transferred
//	NOKLAIM kosong, TRANSFERASM terisi  → Not Registered
//	NOKLAIM terisi, TRANSFERASM terisi  → Outstanding
//
// Tanpa seam ini, menekan Register Klaim BENAR-BENAR membuat klaim — tetapi berkasnya
// tetap duduk di Not Transferred, dan petugas menyimpulkan tombolnya tidak bekerja. Itu
// keluhan nyata (Work Owner, 2026-09-24), dan kelas kegagalan yang paling mahal: yang
// berhasil tetapi tampak gagal.
//
// # Kenapa DUA method, bukan satu
//
// Keduanya terjadi pada saat yang berbeda. Berkas diserahkan begitu klaim dibuka,
// sedangkan nomor klaim baru terbit di UJUNG tahap Input Register (`ADR-0009`) — sampai
// saat itu tidak ada nomor untuk dituliskan. Menyatukannya menjadi satu method berarti
// salah satunya dipanggil dengan nilai kosong, dan kolom yang terisi string kosong TIDAK
// sama dengan kolom yang masih NULL bagi kueri di atas.
//
// # Batas yang mengikat pengisinya
//
// Selama masa paralel, tepat satu sistem menulis sebuah baris (`P-1`, `ADR-0004`).
// Berkas milik Pega hanya boleh dibaca, dan pengisi WAJIB menolak menulis ke sana —
// bukan mengandalkan layar yang sudah mematikan tombolnya.
// ClaimReportSnapshot adalah isi berkas laporan yang DIBAWA ke klaim saat ia dibuka.
//
// Ia tipe milik modul ini, bukan tipe modul Inbox Laporan Klaim. Mengimpor tipe domain
// modul lain membuat keduanya tidak dapat dipindahkan sendiri-sendiri; yang lewat seam
// adalah bentuk yang DIBUTUHKAN registrasi, bukan bentuk yang kebetulan dimiliki
// sumbernya.
//
// Isinya mengikuti `Activity/CreateRegisterKlaimPNC_act.xml` langkah 14 — satu-satunya
// tempat di sistem lama yang menyalin isi berkas RCV ke klaim. Langkah itu memuat 22
// pasang; yang dibawa ke sini adalah pasangan yang PUNYA kolom pada tabel berkas.
//
// # Satu medan klaim yang sengaja TIDAK ada di sini
//
// `ClaimData.DateReceived` — Tanggal Terima Dokumen — di Pega datang dari
// `ReceiveDocument.DateOfSentDocument`, dan properti itu TIDAK punya kolom pada
// `POOLDATA.T_CLAIM_RECIVEDCLAIM`. Tidak ada yang dapat dibawa, sehingga medannya tetap
// kosong dan gerbang validasi Input Register yang akan menuntutnya.
//
// Ia sempat saya isi dari `TANGGALTERIMADOKUMEN`. Itu keliru dua kali: kolom itu menyimpan
// `ReceivedDate`, dan `ReceivedDate` memberi makan `ReportDate` — bukan `DateReceived`.
// Akibatnya Tanggal Lapor tertinggal kosong, padahal aturan "Tanggal Kejadian ≤ Tanggal
// Lapor ≤ Tanggal Terima Dokumen" bersandar padanya. (Batas tujuh hari yang semula ikut
// disebut di sini DICABUT 2026-10-06 — lihat registrasi/validation.go.)
type ClaimReportSnapshot struct {
	// DateOfLoss ← ReceiveDocument.TglKejadian
	DateOfLoss time.Time

	// ReportDate ← @toDateTime(ReceiveDocument.ReceivedDate)
	//
	// Kolomnya `TANGGALTERIMADOKUMEN`, dan namanya menyesatkan: yang tersimpan di sana
	// adalah Tanggal Lapor, bukan Tanggal Terima Dokumen.
	ReportDate time.Time

	// ReporterName ← ReceiveDocument.Sender
	ReporterName string

	// ReporterPhone ← ReceiveDocument.TelpPengirim
	ReporterPhone string

	// ReporterEmail ← ReceiveDocument.EmailPengirim
	ReporterEmail string

	// Location ← ReceiveDocument.LokasiKejadian
	Location string

	// Chronology ← ReceiveDocument.KronologisKejadian
	Chronology string

	// EstimateValue ← ReceiveDocument.Estimasi, dalam SEN (`ADR-0016`).
	EstimateValue Money

	// PolicyNumber ← ReceiveDocument.PolicyNo
	//
	// Ia dibawa untuk DIPERIKSA, bukan dipakai: pemanggil sudah menyebut nomor polis,
	// dan keduanya harus sama. Berbeda berarti berkas dan klaim menunjuk polis yang lain.
	PolicyNumber string

	// ClaimNumber ← NOKLAIM: nomor klaim yang sudah terbit dari berkas ini. Terisi berarti
	// berkasnya sudah diregistrasi, dan Register Klaim kedua ditolak (Work Owner,
	// 2026-09-29) — tanpa itu terbit PNCN kedua dan NOKLAIM berkasnya tertimpa.
	ClaimNumber string
}

type ClaimReportLink interface {
	// MarkHandedOver menandai berkas sudah diserahkan untuk diregistrasi.
	MarkHandedOver(ctx context.Context, reportID string, at time.Time) error

	// AttachClaimNumber menuliskan nomor klaim yang terbit dari berkas itu.
	AttachClaimNumber(ctx context.Context, reportID, claimNumber string) error

	// Snapshot membaca isi berkas yang dibawa ke klaim.
	//
	// Tanpa ini, klaim lahir kosong dan petugas mengetik ulang seluruh isi berkas yang
	// baru saja diisinya — di Pega tidak demikian, dan itu terlihat langsung di layar.
	Snapshot(ctx context.Context, reportID string) (ClaimReportSnapshot, error)
}

// ReportAlreadyRegisteredError: berkas Receive Document sudah punya nomor klaim, sehingga
// Register Klaim kedua ditolak (Work Owner, 2026-09-29).
type ReportAlreadyRegisteredError struct {
	ReportID    string
	ClaimNumber string
}

func (e *ReportAlreadyRegisteredError) Error() string {
	return "registrasi: laporan " + e.ReportID + " sudah diregistrasi sebagai klaim " + e.ClaimNumber
}

// AreaLevel adalah satu tingkat daftar pilihan wilayah kejadian.
type AreaLevel string

const (
	AreaCountry  AreaLevel = "negara"
	AreaProvince AreaLevel = "provinsi"
	AreaCity     AreaLevel = "kota"
	AreaDistrict AreaLevel = "kabupaten"
	AreaVillage  AreaLevel = "kelurahan"
)

// AreaOption adalah satu pilihan pada daftar wilayah.
//
// PostalCode hanya terisi pada tingkat kelurahan: memilih kelurahan mengisi Kode Pos,
// persis seperti Pega yang memetakan `ZipCode` baris RW ke `ClaimData.PostalCode`.
type AreaOption struct {
	ID         string
	Name       string
	PostalCode string
}

// AreaDirectory membaca master wilayah untuk daftar pilihan bertingkat layar Input
// Register.
//
// Tingkatnya dan sumbernya, terverifikasi 2026-09-26 dengan contoh DI YOGYAKARTA → KAB.
// SLEMAN → KEC. DEPOK → KEL. CATURTUNGGAL → 55281:
//
//	negara     POOLDATA.COUNTRY         (BrowseCountry_RD, ada di export)
//	provinsi   POOLDATA.PROVINCE        (BrowseProvince_RD HILANG dari export)
//	kota       POOLDATA.CITYINPUT       (BrowseCity_RD, ada di export)
//	kabupaten  POOLDATA.DISTRICTINPUT   (BrowseDistrictInputC_RD HILANG dari export)
//	kelurahan  POOLDATA.M_RW            (BrowseRWInput_RD HILANG dari export)
//
// parent adalah nilai tingkat di atasnya. Khusus provinsi, ia NAMA negara, bukan kodenya:
// PROVINCE.NATIONID memakai skema kode yang berbeda dari COUNTRY.ID (Indonesia 100009 di
// COUNTRY, sedangkan provinsinya ber-NATIONID 100028 — kode SWEDEN di COUNTRY), sedangkan
// NATIONNAME cocok. Report definition aslinya hilang, sehingga ini inferensi dari data.
type AreaDirectory interface {
	Options(ctx context.Context, level AreaLevel, parent string) ([]AreaOption, error)
}

// CauseOfLossOption adalah satu pilihan Penyebab Kerugian: ID-nya `D_COL_ID` (yang
// disimpan klaim di CAUSEOFLOSSID), Name-nya `DESCRIPTION` (yang dilihat petugas).
type CauseOfLossOption struct {
	ID   string
	Name string
}

// CauseOfLossDirectory membaca pilihan Penyebab Kerugian sebuah kode bisnis polis.
//
// Di Pega isian ini autocomplete ber-sumber `BrowseCouseOfLoss_Business` atas kelas
// `ASM-FW-GCNMFW-Int-V_D_CAUSE_OF_LOSS_BUSINESS`, ber-parameter `id` =
// `Policy.Quotation.BusinessCode` (`Section/ViewCoverageGridContent-Section.xml`). Report
// definition-nya HILANG dari export (`R-16`), sehingga penyaringnya disimpulkan dari
// view-nya: BISNISID = kode bisnis, STS_AKTIF = '1'.
//
// Penyebab kerugian TIDAK ada di dokumen polis — CoverageList polis tidak punya kunci
// apa pun untuknya (diperiksa 2026-10-01). Ia dipilih petugas; yang dapat diisi otomatis
// hanyalah kode bisnis yang pilihannya tepat satu.
type CauseOfLossDirectory interface {
	CauseOfLossOptions(ctx context.Context, businessCode string) ([]CauseOfLossOption, error)
}

// SingleCauseOfLoss mengembalikan satu-satunya pilihan, atau false bila pilihannya kosong
// atau lebih dari satu — saat itu petugas yang memilih.
func SingleCauseOfLoss(options []CauseOfLossOption) (CauseOfLossOption, bool) {
	if len(options) != 1 {
		return CauseOfLossOption{}, false
	}
	return options[0], true
}

// UnassignedTasks adalah seam agent `AutoPICAgent` (`TransferAllCaseNotAssigned`): tugas yang
// diparkir di antrean ServicePNC karena klaimnya belum punya PIC Teknik.
type UnassignedTasks interface {
	// UnassignedTechnicalTasks adalah `BrowseCaseNotAssigned`: tugas terbuka milik
	// ServicePNC yang klaimnya belum ber-PIC Teknik (kosong atau `-`) dan bernomor polis.
	UnassignedTechnicalTasks(ctx context.Context) ([]Task, error)

	// LockUnassigned mengunci satu tugas di dalam transaksi, bila tugas itu masih terbuka dan
	// masih milik ServicePNC; selain itu ErrTaskNotFound. Kunci inilah yang membuat dua
	// instans aplikasi tidak memproses klaim yang sama dua kali.
	LockUnassigned(ctx context.Context, taskID string) (Task, error)

	// Reassign memindahkan tugas terbuka dari ServicePNC ke operator itu.
	Reassign(ctx context.Context, taskID, to string) error
}
