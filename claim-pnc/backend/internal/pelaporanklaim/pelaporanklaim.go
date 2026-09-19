// Package pelaporanklaim adalah inti modul Pelaporan Klaim (`B-14`).
//
// # Nama modul ini, dan kenapa bukan "Receive Document"
//
// Sistem lama menamai case-nya `ASM-FW-GCNMFW-Work-ReceiveDocument` dan flow-nya
// `Flow/InputReceiveDocument.xml`. Tetapi yang dilihat PENGGUNA bukan nama itu:
//
//   - Menu portalnya berbunyi "Inbox Laporan Klaim"
//     (`Navigation/pyCaseWorkerNavigation-Navigation.xml:19864`).
//   - Saat case dibuat, progresnya dicatat dengan posisi "LAPORAN KLAIM" dan catatan
//     "Auto Create Laporan" (`Activity/CreateNewCaseRCV-Act.xml` step 7).
//   - Developernya sendiri menyebutnya "inbox pelaporan klaim" di komentar
//     (`Section/ViewStatusReceiveDocument-Section.xml`).
//
// `D-81` menetapkan nama modul diambil dari nama yang dipakai Work Owner, dan Work Owner
// menyebutnya **Pelaporan Klaim**. Ketiga bukti di atas menunjukkan nama itu bukan
// karangan — ia nama yang dipakai sistem lama sendiri di permukaan yang dilihat orang.
//
// # Bahasa penamaan
//
// Nama paket dan nama folder berbahasa Indonesia karena ia NAMA MODUL (`D-81`). Seluruh
// nama di dalamnya — tipe, fungsi, method, field, parameter, variabel — berbahasa Inggris
// (`D-80`). Komentar tetap bahasa Indonesia, begitu pula nama field JSON, nama kolom basis
// data, dan teks yang dilihat pengguna; ketiganya termasuk lima pengecualian `D-80`.
//
// # Apa yang dicatat modul ini
//
// Laporan kerugian yang masuk SEBELUM klaim diregistrasi: siapa yang melapor, bagaimana
// menghubunginya, polis dan tertanggung yang dirujuk, apa yang terjadi, di mana, kapan,
// dan berapa perkiraan kerugiannya.
//
// Ia BUKAN pencatatan ekspedisi dan nomor resi. Keduanya memang ada di sistem lama,
// tetapi pada tempat yang berbeda: `ClaimData.EkspedisiKlaim` dan
// `ClaimData.NoResiEskpedisi` pada KLAIM, yang ditulis
// `Activity/SendDataDariCabangKeKantorPusat_ACT-Act.xml` (berkelas
// `ASM-FW-GCNMFW-Work-PNC`) ke `POOLDATA.T_CLAIM_DATACABANG`. Itu aksi "Transfer ke
// Kantor Pusat" pada klaim, bukan layar ini.
//
// # Daur hidupnya ditentukan dua penanda, bukan satu kolom status
//
// Sistem lama tidak menyimpan status laporan sebagai kolom. Ia menurunkannya dari
// kombinasi dua penanda, terbaca dari `RDB List/BrowseClaimRCV_Aksep-SQL.xml`:
//
//	PNCCASEID null  + STATUSLOCK null      -> belum ditransfer ke ASM
//	PNCCASEID null  + STATUSLOCK terisi    -> sudah ditransfer, belum diregistrasi
//	PNCCASEID terisi+ STATUSLOCK terisi    -> sudah diregistrasi menjadi klaim
//
// Pola itu dipertahankan apa adanya: Stage DIHITUNG, tidak disimpan. Menyimpannya sebagai
// kolom tersendiri akan membuat dua sumber kebenaran yang dapat berselisih.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package pelaporanklaim

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

// Batas panjang teks.
//
// Angkanya diambil dari lebar kolom yang dibuat migrasi `0003`, bukan dari sistem lama —
// layar Pega tidak membatasi apa pun. Batas di aplikasi ada supaya isian yang kepanjangan
// ditolak dengan pesan yang dapat diperbaiki pengguna, bukan sebagai galat `500` dari
// basis data.
const (
	MaxNameLength         = 100
	MaxEmailLength        = 100
	MaxPhoneLength        = 30
	MaxPolicyNumberLength = 50
	MaxCodeLength         = 20
	MaxLongTextLength     = 4000
)

// Stage adalah posisi sebuah laporan dalam perjalanannya menjadi klaim.
//
// Ia DIHITUNG dari isi laporan, bukan disimpan — lihat komentar paket.
type Stage string

const (
	// StageNotTransferred: laporan sudah dicatat cabang, belum dikirim ke ASM pusat.
	StageNotTransferred Stage = "BELUM_TRANSFER"

	// StageNotRegistered: sudah sampai ASM, belum menjadi klaim.
	StageNotRegistered Stage = "BELUM_REGISTRASI"

	// StageRegistered: sudah menjadi klaim; nomor klaimnya terisi.
	StageRegistered Stage = "SUDAH_REGISTRASI"

	// StageAccepted dan StageRejected adalah kelanjutan yang ditentukan MODUL LAIN.
	//
	// Sistem lama menghitung keduanya dengan menengok tabel klaim:
	// `ViewTableBrowseRCVAcc-SQL.xml` memeriksa `t_claim_adjustment.noakseptasi`, dan
	// `ViewTableBrowseRCVReject-SQL.xml` memeriksa `t_claim_pnc.statuswork`.
	//
	// Kedua tabel itu milik `B-5` dan `B-10` yang belum dibangun. Karena itu modul ini
	// tidak menengok ke sana; ia menyediakan satu field (Outcome) yang diisi modul klaim
	// saat hasilnya diketahui. Hari ini field itu selalu kosong, dan kedua tahap ini
	// karena itu belum pernah terjadi — bukan disembunyikan, melainkan belum ada yang
	// mengisinya.
	StageAccepted Stage = "SUDAH_AKSEPTASI"
	StageRejected Stage = "DITOLAK"
)

// ClaimOutcome menyatakan bagaimana klaim yang lahir dari laporan ini berakhir.
//
// Nilainya berbahasa Indonesia karena ia sandi yang tersimpan di kolom basis data, bukan
// nama di dalam kode — termasuk pengecualian `D-80`.
type ClaimOutcome string

const (
	// OutcomeNone berlaku selama klaimnya belum selesai — dan selama modul klaim belum
	// ada, ia berlaku untuk seluruh laporan.
	OutcomeNone     ClaimOutcome = ""
	OutcomeAccepted ClaimOutcome = "DIAKSEPTASI"
	OutcomeRejected ClaimOutcome = "DITOLAK"
)

// ClaimReport adalah satu laporan kerugian yang masuk.
//
// Nama field mengikuti padanan Inggris dari `CONTEXT.md`, bukan nama properti Pega — dan
// itu bukan kerapian. Pemetaan properti klipboard ke kolom di sistem lama MENYESATKAN
// SECARA AKTIF; satu berkas memuat 26 pemetaan dan sebagian besarnya salah arti. Tiga
// yang terburuk, dari `RDB List/Rcv_ProcInsertRecivedDocument-SQL.xml:102-124`:
//
//	TampunganPages.TelpTertanggung          -> kolom USERINPUT      (petugas penginput)
//	TampunganPages.TanggalSelesaiRawatInap  -> kolom REGISTDATE     (tanggal registrasi)
//	TampunganPages.NamaSurveyor             -> kolom NAMAKURIRASM   (nama kurir)
//
// Pemetaan tiga arah properti->kolom->domain ditulis lengkap di kepala
// `repo/sqlstore/report.sql` — satu-satunya tempat ketiganya dapat dibandingkan
// berdampingan. Ini melaksanakan `03-CURRENT-ARCHITECTURE` §4.2.
type ClaimReport struct {
	// Number adalah nomor register laporan, setara `pyID` pada case RCV.
	//
	// Dibuat sistem saat laporan dicatat dan tidak pernah berubah. Bentuknya dibahas di
	// repo/sqlstore — prefix case RCV di sistem lama TIDAK ADA di export, sehingga
	// bentuknya adalah keputusan baru, bukan peniruan.
	Number string

	// --- Pelapor ---

	// ReporterName adalah `ReceiveDocument.Sender` -> kolom NAMAPELAPOR.
	ReporterName string
	// SenderEmail adalah `ReceiveDocument.EmailPengirim` -> kolom EMAILPENGIRIM.
	SenderEmail string
	// SenderPhone adalah `ReceiveDocument.TelpPengirim` -> kolom TLPPENGIRIM.
	SenderPhone string
	// CourierName adalah `ReceiveDocument.Kurir` -> kolom NAMAKURIRASM.
	//
	// Nilai "Auto Service" punya arti khusus di sistem lama: laporan bertanda itu
	// DIKECUALIKAN dari hitungan PA/Travel
	// (`RDB List/GetDataRCVallKlaimPATravel-SQL.xml`). Daftar pilihan lengkapnya tidak
	// ada di export, sehingga field ini teks bebas — menebak daftarnya akan mengulang
	// kesalahan yang sudah tercatat pada Master Rekening (`keputusan-implementasi.md`
	// §10.9).
	CourierName string
	// EmailSubject adalah `ReceiveDocument.SubjectEmail` -> kolom SUBJECTEMAIL.
	EmailSubject string

	// --- Polis dan tertanggung, sebagaimana disebut pelapor ---
	//
	// Nilai di sini adalah APA YANG DILAPORKAN, bukan hasil pembacaan polis. Snapshot
	// polis yang sah baru terbentuk saat registrasi (`D-04`, modul `B-1`). Membiarkannya
	// sebagai isian bebas adalah perilaku sistem lama dan memang benar: pelapor sering
	// menyebut nomor polis yang keliru, dan laporannya tetap harus dapat dicatat.

	PolicyNumber    string // `PolicyNo` -> NOPOLIS
	InsuredName     string // `QQName` -> NAMATERTANGGUNG
	InsuredEmail    string // `ReceiveDocument.EmailLOD` -> EMAILTERTANGGUNG
	BusinessCode    string // `Policy.Quotation.BusinessCode` -> BUSINESSCODE
	GroupPanel      string // `Policy.Quotation.GroupPanel` -> GROUPPANEL
	ReferenceNumber string // `Policy.BookNo` -> NOREFERENSI

	// --- Kerugian yang dilaporkan ---

	// LossDate adalah `ReceiveDocument.TglKejadian` -> kolom DOL.
	//
	// Boleh kosong: laporan pertama sering masuk sebelum tanggal kejadiannya pasti.
	// Validasi tanggal yang sesungguhnya — DOL <= lapor <= terima dokumen <= hari ini —
	// adalah invarian `I-2` yang ditegakkan saat REGISTRASI (`B-2`), bukan di sini.
	LossDate *time.Time

	LossLocation  string // `LokasiKejadian` -> LOKASIKEJADIAN
	Chronology    string // `KronologisKejadian` -> KRONOLOGIKEJADIAN
	DamageDetails string // `RincianKerusakan` -> RINCIANKERUSAKAN
	DriverLicense string // `ReceiveDocument.SIM` -> SIMPENGENDARA

	// EstimatedValue adalah `ReceiveDocument.Estimasi`, perkiraan kerugian menurut
	// pelapor.
	//
	// Di sistem lama ia TIDAK tersimpan di `T_CLAIM_RECIVEDCLAIM`: activity
	// `rcv_InsertRecivedDocumentClaim` menyalinnya ke `TampunganPages.EstimationValue`,
	// tetapi procedure-nya tidak punya parameter untuk itu
	// (`Database/PROCINSERTDATARECIVEDKLAIM.prc:2-28`). Nilainya hanya hidup di blob
	// case Pega. Karena tabel modul ini baru dan milik aplikasi, ia disimpan sebagaimana
	// mestinya — datanya memang sudah diisi pengguna, hanya tidak pernah mendarat di
	// kolom mana pun.
	//
	// # Kenapa teks, padahal ia nilai uang
	//
	// Justru KARENA ia nilai uang. `09-DATABASE-STRATEGY.md` §5 melarang uang disimpan
	// sebagai `float`, dan `I-12` menuntut presisi penuh. Teks desimal adalah satu-satunya
	// bentuk di pustaka standar Go yang membawa angka desimal tanpa kehilangan satu digit
	// pun; `float64` akan membulatkan diam-diam, dan proyek ini belum punya pustaka
	// desimal sebagai dependensi.
	//
	// Kolomnya di basis data juga VARCHAR2, dan itu penyimpangan sadar dari §5 yang
	// alasannya ditulis lengkap di migrasi `0003`. Ringkasnya: menuliskan teks desimal ke
	// kolom NUMBER menyerahkan konversinya kepada NLS Oracle, yang menolak titik sebagai
	// pemisah desimal pada sesi berlokal koma — kegagalan yang bergantung lingkungan dan
	// tidak dapat dibuktikan tanpa basis data.
	//
	// Bentuknya diperiksa LooksLikeMoney sebelum menyentuh penyimpanan, dan dipagari lagi
	// dengan CHECK di basis data.
	//
	// Nilai ini TIDAK dipakai perhitungan apa pun di modul ini. Ambang komite dan Notice
	// of Large Losses dihitung dari nilai pada KLAIM (`B-5`), bukan dari perkiraan
	// pelapor.
	EstimatedValue string

	// ClaimType adalah `ReceiveDocument.TypeOfClaim`, berlabel "Tipe Klaim" pada
	// `Report Definition/BrowseCaseReceivedDocList_RD-RD.xml`. Daftar pilihannya tidak
	// ada di export; teks bebas, dengan alasan yang sama seperti CourierName.
	ClaimType string

	// DocumentCount adalah `ReceiveDocument.NumberOfDocument`, berlabel "Jumlah Dokumen".
	//
	// Di sistem lama ia dihitung `Activity/SumDocumentReceive-Act.xml` dengan menjumlah
	// `NumberOfDocument` seluruh baris daftar dokumen. Selama modul lampiran (`S-1`)
	// belum ada, ia diisi pengguna — dan itu yang membuatnya tetap berguna hari ini.
	DocumentCount int

	// DocumentReceivedDate adalah `ReceiveDocument.ReceivedDate`, berlabel "Tanggal
	// Terima Dokumen".
	//
	// PERBEDAAN YANG DISENGAJA dari sistem lama. Kolom tujuannya di sana,
	// `TANGGALTERIMADOKUMEN`, bertipe VARCHAR2 — tanggal disimpan sebagai teks
	// (`Database/PROCINSERTDATARECIVEDKLAIM.prc:4`). Akibatnya pengurutan menjadi
	// pengurutan teks dan penyaringan rentang tidak dapat memakai indeks; persis cacat
	// yang `09-DATABASE-STRATEGY` §3.2 perintahkan dihapus. Di tabel baru ia DATE.
	DocumentReceivedDate *time.Time

	// --- Daur hidup ---

	// ClaimNumber adalah `ReceiveDocument.PNCCaseID` -> kolom NOKLAIM.
	//
	// Kosong selama laporan belum diregistrasi. Prefix `"ASM-FW-GCNMFW-WORK "` yang
	// disambungkan procedure lama (`PROCINSERTDATARECIVEDKLAIM.prc:57`) TIDAK dibawa —
	// `D-22` dan `D-71` menetapkan kunci teknis Pega tidak lagi bocor ke data bisnis.
	ClaimNumber string

	// Transferred adalah `ReceiveDocument.StatusLock`.
	//
	// Namanya di sistem lama menyesatkan: ia bukan kunci baris melainkan penanda bahwa
	// laporan sudah dikirim ke ASM pusat. Yang benar-benar mengunci layar adalah
	// `StatusLockFile`, yang diturunkan darinya oleh
	// `Activity/Pre_ActReceiveDocument-Act.xml`.
	Transferred bool

	TransferredAt *time.Time // `ReceiveDocument.DateOfSendASM` -> TRANSFERASM
	RegisteredAt  *time.Time // `ReceiveDocument.DateOfRegistrcv` -> REGISTDATE

	// NotTransferredReason adalah `ReceiveDocument.Keterangan` -> ALASANBLMTRANSFER.
	//
	// Perhatikan namanya di layar hanya "Keterangan" — nama kolomnyalah yang
	// menjelaskan artinya.
	NotTransferredReason string

	// NotRegisteredNote adalah `ReceiveDocument.NotRegistNote` -> KETERANGANBLMREGIST.
	NotRegisteredNote string

	// Outcome diisi modul klaim, bukan modul ini. Lihat StageAccepted.
	Outcome ClaimOutcome

	// --- Jejak ---

	BranchCode string    // `ReceiveDocument.KodeCabang` -> KODECABANG
	CreatedBy  string    // `TampunganRCV.pxCreateOperator` -> kolom USERINPUT
	CreatedAt  time.Time // `pxCreateDateTime` -> TANGGALINPUTDOKUMEN
	UpdatedAt  time.Time
}

// Stage menghitung posisi laporan dari isinya.
//
// Urutan pemeriksaan mengikuti urutan kejadian sebenarnya, dan hasil klaim diperiksa
// LEBIH DULU: laporan yang klaimnya sudah selesai tetap punya nomor klaim, sehingga
// memeriksa nomor klaim lebih dulu akan menahan seluruhnya di "sudah registrasi".
func (r ClaimReport) Stage() Stage {
	switch r.Outcome {
	case OutcomeAccepted:
		return StageAccepted
	case OutcomeRejected:
		return StageRejected
	}
	if strings.TrimSpace(r.ClaimNumber) != "" {
		return StageRegistered
	}
	if r.Transferred {
		return StageNotRegistered
	}
	return StageNotTransferred
}

// Label menyebut tahap dalam bahasa yang dibaca pengguna.
//
// Teksnya diambil dari judul tab pada `Section/ViewStatusReceiveDocument-Section.xml`
// dan diterjemahkan ke bahasa Indonesia — layar lama mencampur dua bahasa ("Data hasn't
// been transferred" bersebelahan dengan "Claim yang sudah Registrasi"), dan mencampurnya
// kembali tidak menolong siapa pun.
//
// Teks di bawah termasuk pengecualian `D-80`: ia dilihat pengguna, bukan nama di dalam
// kode.
func (s Stage) Label() string {
	switch s {
	case StageNotTransferred:
		return "Belum ditransfer"
	case StageNotRegistered:
		return "Belum diregistrasi"
	case StageRegistered:
		return "Sudah diregistrasi"
	case StageAccepted:
		return "Sudah diakseptasi"
	case StageRejected:
		return "Ditolak"
	default:
		return string(s)
	}
}

// Known menyatakan tahap ini termasuk yang dikenal modul.
func (s Stage) Known() bool {
	switch s {
	case StageNotTransferred, StageNotRegistered, StageRegistered,
		StageAccepted, StageRejected:
		return true
	default:
		return false
	}
}

// IsRegistered menyatakan laporan ini sudah menjadi klaim.
func (r ClaimReport) IsRegistered() bool {
	return strings.TrimSpace(r.ClaimNumber) != ""
}

// CanTransfer menyatakan laporan ini masih boleh ditransfer ke ASM pusat.
func (r ClaimReport) CanTransfer() bool {
	return !r.Transferred && !r.IsRegistered()
}

// Clean mengembalikan salinan dengan spasi tepi seluruh teks dibuang.
//
// Perapian ini nyata gunanya: nomor polis yang diketik dengan spasi di ujung tidak akan
// pernah cocok saat laporan ini kelak dicari untuk ditautkan ke klaim.
func (r ClaimReport) Clean() ClaimReport {
	r.Number = strings.TrimSpace(r.Number)
	r.ReporterName = strings.TrimSpace(r.ReporterName)
	r.SenderEmail = strings.TrimSpace(r.SenderEmail)
	r.SenderPhone = strings.TrimSpace(r.SenderPhone)
	r.CourierName = strings.TrimSpace(r.CourierName)
	r.EmailSubject = strings.TrimSpace(r.EmailSubject)
	r.PolicyNumber = strings.TrimSpace(r.PolicyNumber)
	r.InsuredName = strings.TrimSpace(r.InsuredName)
	r.InsuredEmail = strings.TrimSpace(r.InsuredEmail)
	r.BusinessCode = strings.TrimSpace(r.BusinessCode)
	r.GroupPanel = strings.TrimSpace(r.GroupPanel)
	r.ReferenceNumber = strings.TrimSpace(r.ReferenceNumber)
	r.LossLocation = strings.TrimSpace(r.LossLocation)
	r.Chronology = strings.TrimSpace(r.Chronology)
	r.DamageDetails = strings.TrimSpace(r.DamageDetails)
	r.DriverLicense = strings.TrimSpace(r.DriverLicense)
	r.EstimatedValue = strings.TrimSpace(r.EstimatedValue)
	r.ClaimType = strings.TrimSpace(r.ClaimType)
	r.ClaimNumber = strings.TrimSpace(r.ClaimNumber)
	r.NotTransferredReason = strings.TrimSpace(r.NotTransferredReason)
	r.NotRegisteredNote = strings.TrimSpace(r.NotRegisteredNote)
	r.BranchCode = strings.TrimSpace(r.BranchCode)
	r.CreatedBy = strings.TrimSpace(r.CreatedBy)
	return r
}

// Validate mengumpulkan SELURUH pelanggaran aturan isian, bukan berhenti pada yang
// pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama
// menampilkan seluruh pesan validasi sekaligus (`12-CROSSCUTTING.md` §1.2 butir 1), dan
// mengembalikannya satu per satu akan membuat pengguna menekan Simpan berkali-kali untuk
// menemukan kesalahan berikutnya. Form ini punya 20 kolom; sekali-satu akan menyiksa.
//
// # Kenapa hanya nama pelapor yang wajib
//
// Layar Pega TIDAK MEWAJIBKAN APA PUN. Itu bukan kelalaian melainkan sifat pekerjaannya:
// laporan kerugian datang lewat telepon dan surel dengan kelengkapan yang berbeda-beda,
// dan petugas harus dapat mencatatnya SEKARANG lalu melengkapinya kemudian. Menolak
// laporan yang belum lengkap berarti laporan itu tidak tercatat sama sekali.
//
// Satu-satunya yang diwajibkan adalah nama pelapor — tanpa itu laporannya tidak dapat
// ditindaklanjuti siapa pun, dan tidak ada yang dapat dihubungi untuk melengkapinya.
// Kelengkapan yang sesungguhnya ditegakkan saat REGISTRASI (`B-2`), tempat invarian
// `I-2` sampai `I-10` berlaku.
func (r ClaimReport) Validate() []Violation {
	clean := r.Clean()
	var violations []Violation

	if clean.ReporterName == "" {
		violations = append(violations, Violation{
			Field:   FieldReporterName,
			Message: "Nama pelapor wajib diisi.",
		})
	}

	limits := []struct {
		field string
		value string
		max   int
		label string
	}{
		{FieldReporterName, clean.ReporterName, MaxNameLength, "Nama pelapor"},
		{FieldSenderEmail, clean.SenderEmail, MaxEmailLength, "Email pengirim"},
		{FieldSenderPhone, clean.SenderPhone, MaxPhoneLength, "Telepon pengirim"},
		{FieldCourierName, clean.CourierName, MaxNameLength, "Nama kurir"},
		{FieldEmailSubject, clean.EmailSubject, MaxNameLength, "Subjek email"},
		{FieldPolicyNumber, clean.PolicyNumber, MaxPolicyNumberLength, "Nomor polis"},
		{FieldInsuredName, clean.InsuredName, MaxNameLength, "Nama tertanggung"},
		{FieldInsuredEmail, clean.InsuredEmail, MaxEmailLength, "Email tertanggung"},
		{FieldReferenceNumber, clean.ReferenceNumber, MaxPolicyNumberLength, "Nomor referensi"},
		{FieldClaimType, clean.ClaimType, MaxCodeLength, "Tipe klaim"},
		{FieldDriverLicense, clean.DriverLicense, MaxPolicyNumberLength, "SIM pengendara"},
		{FieldLossLocation, clean.LossLocation, MaxLongTextLength, "Lokasi kejadian"},
		{FieldChronology, clean.Chronology, MaxLongTextLength, "Kronologi kejadian"},
		{FieldDamageDetails, clean.DamageDetails, MaxLongTextLength, "Rincian kerusakan"},
		{FieldNotTransferredReason, clean.NotTransferredReason, MaxLongTextLength, "Alasan belum transfer"},
		{FieldNotRegisteredNote, clean.NotRegisteredNote, MaxLongTextLength, "Catatan belum registrasi"},
	}
	for _, l := range limits {
		// Dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
		// membuat batas terasa berubah-ubah bagi pengguna.
		if utf8.RuneCountInString(l.value) > l.max {
			violations = append(violations, Violation{
				Field:   l.field,
				Message: l.label + " paling panjang " + itoa(l.max) + " karakter.",
			})
		}
	}

	if clean.SenderEmail != "" && !LooksLikeEmail(clean.SenderEmail) {
		violations = append(violations, Violation{
			Field:   FieldSenderEmail,
			Message: "Email pengirim tidak berbentuk alamat surel.",
		})
	}
	if clean.InsuredEmail != "" && !LooksLikeEmail(clean.InsuredEmail) {
		violations = append(violations, Violation{
			Field:   FieldInsuredEmail,
			Message: "Email tertanggung tidak berbentuk alamat surel.",
		})
	}

	if r.DocumentCount < 0 {
		violations = append(violations, Violation{
			Field:   FieldDocumentCount,
			Message: "Jumlah dokumen tidak boleh negatif.",
		})
	}

	if clean.EstimatedValue != "" && !LooksLikeMoney(clean.EstimatedValue) {
		violations = append(violations, Violation{
			Field:   FieldEstimatedValue,
			Message: "Estimasi kerugian harus berupa angka, paling banyak dua angka di belakang koma.",
		})
	}

	return violations
}

// MaxMoneyDigits membatasi bagian bulat nilai uang.
//
// Angkanya mengikuti lebar kolom `NUMBER(18,2)` yang `09-DATABASE-STRATEGY.md` §5
// tetapkan untuk nilai uang: 18 angka berarti 16 di depan koma dan 2 di belakangnya.
const MaxMoneyDigits = 16

// LooksLikeMoney memeriksa bentuk sebuah nilai uang dalam teks desimal.
//
// Ia sengaja hanya menerima bentuk yang dapat dikirim apa adanya ke kolom `NUMBER(18,2)`:
// angka, paling banyak satu titik desimal, paling banyak dua angka di belakangnya, dan
// tanpa pemisah ribuan. Pemisah ribuan sengaja DITOLAK dan bukan dibuang diam-diam —
// "1.500" berarti seribu lima ratus bagi sebagian orang dan satu koma lima bagi yang
// lain, dan menebaknya berarti salah pada separuh kasus.
//
// Tanda minus juga ditolak: estimasi kerugian negatif tidak punya arti.
func LooksLikeMoney(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	whole, fraction, hasDot := strings.Cut(value, ".")
	if hasDot && (len(fraction) == 0 || len(fraction) > 2) {
		return false
	}
	if len(whole) == 0 || len(whole) > MaxMoneyDigits {
		return false
	}
	for _, part := range []string{whole, fraction} {
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// LooksLikeEmail memeriksa bentuk alamat surel seadanya.
//
// Sengaja longgar. Satu-satunya cara membuktikan sebuah alamat sah adalah mengirim surel
// ke sana; pemeriksaan yang ketat hanya akan menolak alamat sah yang bentuknya tidak
// biasa. Yang ditangkap di sini adalah salah ketik yang jelas — tanpa "@", tanpa titik
// sesudahnya, atau berisi spasi.
func LooksLikeEmail(address string) bool {
	address = strings.TrimSpace(address)
	if address == "" || strings.ContainsAny(address, " \t\r\n") {
		return false
	}
	at := strings.LastIndex(address, "@")
	if at <= 0 || at == len(address)-1 {
		return false
	}
	domain := address[at+1:]
	dot := strings.LastIndex(domain, ".")
	return dot > 0 && dot < len(domain)-1
}

// Filter menyaring daftar laporan.
//
// Ia sengaja BUKAN bahasa penyaring generik. `09-API-STRATEGY` §4 melarangnya, dan
// alasannya bukan kerapian: pola `{ASIS:...}` warisan — yang merangkai potongan SQL dari
// nilai properti — muncul TIGA KALI pada setiap kueri inbox lama
// (`RDB List/ViewTableBrowseRCVInProcess-SQL.xml` dan dua saudaranya). Penyaring yang
// setiap kemungkinannya disebut namanya di sini tidak dapat dipakai menyisipkan SQL.
type Filter struct {
	// Stage menyaring menurut posisi laporan. Kosong berarti seluruh tahap.
	Stage Stage

	// Search mencocokkan nomor laporan, nomor klaim, nomor polis, nama tertanggung, dan
	// nama pelapor.
	Search string

	// BranchCode membatasi laporan pada satu cabang.
	//
	// Batas data per cabang yang sesungguhnya (`11-SECURITY.md` §3.2) adalah
	// `TKT-F3-005` dan belum ada — hari ini field ini dipakai penyaringan biasa, BUKAN
	// sebagai kendali akses. Dicatat supaya tidak terbaca seolah kendalinya sudah ada.
	BranchCode string

	Limit  int
	Offset int
}

// DefaultLimit dan MaxLimit membatasi banyaknya baris per permintaan.
//
// Angka 50 mengikuti `pyPageSize` pada
// `Report Definition/BrowseCaseReceivedDocList_RD-RD.xml`; batas atasnya 200, bukan 500
// seperti `pyMaxRecords` lama — `14-NFR` mencatat batas 500 itu MEMOTONG hasil diam-diam
// alih-alih memaginasinya, dan itu yang tidak dibawa.
const (
	DefaultLimit = 50
	MaxLimit     = 200
)

// Normalize mengembalikan filter dengan nilai yang aman dipakai kueri.
func (f Filter) Normalize() Filter {
	f.Search = strings.TrimSpace(f.Search)
	f.BranchCode = strings.TrimSpace(f.BranchCode)
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return f
}

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
type Page struct {
	Reports []ClaimReport

	// Total adalah banyaknya baris yang cocok SEBELUM dipotong paginasi.
	Total int
}

// StageSummary adalah jumlah laporan per tahap, untuk lencana di atas tiap tab.
//
// Sistem lama menghitungnya dengan satu kueri berisi enam SUM(CASE WHEN ...)
// (`RDB List/BrowseClaimRCV_Aksep-SQL.xml`). Bentuk itu dipertahankan — satu perjalanan
// ke basis data untuk seluruh angka, bukan satu kueri per tab.
type StageSummary map[Stage]int

// Repo adalah seam ke penyimpanan laporan klaim.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memory (pengujian dan pengembangan
// tanpa basis data). Antarmukanya berbicara dalam istilah domain, bukan istilah SQL.
//
// Tidak ada Delete. Itu bukan kelalaian: `ADR-0012` menetapkan penghapusan lunak
// menyeluruh, dan laporan yang sudah tertaut klaim tidak boleh hilang karena klaimnya
// merujuknya. Laporan yang keliru dicatat ditandai lewat catatannya, bukan dibuang.
type Repo interface {
	// List mengembalikan satu halaman laporan beserta jumlah seluruh yang cocok.
	List(ctx context.Context, f Filter) (Page, error)

	// Summary menghitung jumlah laporan per tahap, menghormati Search dan BranchCode
	// pada filter tetapi MENGABAIKAN Stage — angkanya dipakai seluruh tab sekaligus.
	Summary(ctx context.Context, f Filter) (StageSummary, error)

	// Get mengembalikan satu laporan. Mengembalikan ErrNotFound bila nomornya tidak ada.
	Get(ctx context.Context, number string) (ClaimReport, error)

	// Insert menyimpan laporan baru dan mengembalikannya LENGKAP DENGAN NOMOR yang
	// dibuat penyimpanan. Nomor tidak pernah datang dari pemanggil.
	Insert(ctx context.Context, r ClaimReport) (ClaimReport, error)

	// Update mengubah laporan yang sudah ada. Nomor, pencatat, dan waktu pencatatan
	// tidak ikut berubah.
	Update(ctx context.Context, r ClaimReport) (ClaimReport, error)
}

// itoa mengubah bilangan kecil menjadi teks tanpa menarik strconv ke lapisan domain
// hanya untuk satu pesan.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
