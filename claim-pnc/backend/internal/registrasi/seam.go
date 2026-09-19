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
	// antrean yang ia berwenang (`D-79`).
	Inbox(ctx context.Context, operator string, workbasket []string) ([]Task, error)
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
