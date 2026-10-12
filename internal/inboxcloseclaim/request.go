package inboxcloseclaim

import (
	"context"
	"strings"
	"time"
)

// RequestKind adalah jenis permintaan yang dapat diajukan dari layar ini.
//
// Keduanya tombol nyata di `Section/InboxManagerReopen1_Sec-Section.xml` — "ReOpen" dan
// "Copy Klaim" — yang di sistem lama memanggil `SaveReOpenAct` lalu membuka modal
// `GCNMReopenConfirmation` dan `GCNMCopyClaimConfirmation`.
//
// # KETIGA RULE ITU TIDAK ADA DI EXPORT
//
// Penelusuran seluruh export, dan hasilnya nol pada setiap jalur:
//
//	GCNMReopenConfirmation · GCNMCopyClaimConfirmation · FilterDashboardClaimclose  nol berkas
//	rule mana pun yang menulis status '1164' (Reopen Claim)                         nol
//	rule mana pun yang menyentuh PYREOPENCOUNT / PYREOPENTIMESTAMP                  nol
//	activity *CopyClaim* / *CopyKlaim*                                              nol
//	Ticket rule atau Flow untuk reopen                                              nol
//
// `SaveReOpenAct` yang MEMANG ada ternyata satu langkah `Property-Set` dan tidak menulis
// apa pun. Yang tersisa hanyalah jejak bahwa reopen pernah terjadi: `PYREOPENTIMESTAMP`
// terisi pada 47 klaim dan `PYREOPENCOUNT` punya 7 nilai berbeda — keduanya kolom bawaan
// Pega. `UpdateStatus` OOTB bahkan menyimpan catatannya sendiri: *"A resolved work object
// should be reopened first"*.
//
// Jadi reopen di sistem lama adalah **mekanisme platform Pega**, bukan rule bisnis yang
// dapat dibaca. Aturan yang dipakai di sini karena itu **ditetapkan Work Owner 2026-09-23**
// dan dicatat sebagai keputusannya — bukan sebagai hasil pembacaan export. Konsekuensinya
// mengikuti `D-56`: gerbang 1 untuk kedua aksi ini tidak dapat berupa uji kesetaraan,
// melainkan uji fungsional terhadap aturan itu.
type RequestKind string

const (
	// RequestReopen membuka kembali klaim yang sudah tutup.
	RequestReopen RequestKind = "reopen"

	// RequestCopy membuat klaim baru dari klaim yang sudah tutup.
	RequestCopy RequestKind = "salin"
)

// ParseRequestKind membaca jenis permintaan dari jalur URL.
func ParseRequestKind(raw string) (RequestKind, bool) {
	value := RequestKind(strings.ToLower(strings.TrimSpace(raw)))
	switch value {
	case RequestReopen, RequestCopy:
		return value, true
	default:
		return "", false
	}
}

// Efek yang DIKEHENDAKI pada klaim saat permintaan reopen dijalankan.
//
// Ditetapkan Work Owner 2026-09-23. Keempatnya adalah kolom yang MEMANG ADA di
// `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, dan tiga di antaranya memang sudah terisi pada klaim
// yang pernah dibuka kembali:
//
//	PYSTATUSWORK        kembali ke 'New'
//	STATUSCLAIM_1       menjadi '1164' — "Reopen Claim" pada master 33 kode
//	PYREOPENCOUNT       bertambah satu
//	PYREOPENTIMESTAMP   diisi waktu permintaan dijalankan
//
// # Kenapa nilainya ikut DISIMPAN pada setiap baris permintaan
//
// Supaya permintaan yang dijalankan bulan depan dijalankan menurut aturan yang berlaku
// SAAT IA DIAJUKAN. Bila kelak Work Owner mengubah aturannya, baris lama tetap menyimpan
// niat aslinya — dan siapa pun yang menjalankannya tidak perlu menebak aturan mana yang
// berlaku. Menyimpan aturan hanya di dalam kode membuat jejaknya hilang begitu kodenya
// berubah.
const (
	EfekStatusKerjaReopen = "New"
	EfekStatusKlaimReopen = "1164"
)

// LingkupSalinBaku adalah apa yang disalin oleh Copy Klaim.
//
// Ditetapkan Work Owner 2026-09-23: **snapshot polis, objek pertanggungan, dan coverage;
// nilai estimasi/usulan/akseptasi/pembayaran TIDAK disalin.** Klaim baru terbit dengan
// nomornya sendiri berformat `PNCN.YY.xxxx` (`D-71`) dan mulai dari tahap registrasi.
//
// Ia disimpan pada barisnya dengan alasan yang sama seperti kedua konstanta di atas: yang
// menjalankan permintaan ini bukan aplikasi ini, dan ia tidak boleh menebak lingkupnya.
const LingkupSalinBaku = "polis_objek_coverage"

// RequestStatus adalah keadaan sebuah permintaan.
//
// # Aplikasi ini TIDAK PERNAH mengubahnya
//
// Ia hanya menulis baris ber-`menunggu` dan membacanya kembali. Yang memindahkannya ke
// `dijalankan` adalah pihak yang benar-benar mengeksekusi — Pega atau DBA — dengan hak
// aksesnya sendiri. Akun aplikasi tidak diberi hak `UPDATE` maupun `DELETE` atas tabel
// ini, dan itu ditegakkan di GRANT, bukan di kode: aturan yang hanya ada di dalam kode
// dapat dilanggar oleh kode berikutnya.
type RequestStatus string

const (
	RequestPending  RequestStatus = "menunggu"
	RequestExecuted RequestStatus = "dijalankan"
	RequestCanceled RequestStatus = "dibatalkan"
)

// ClaimRequest adalah satu permintaan tindakan atas sebuah klaim yang sudah tutup.
//
// # Kenapa PERMINTAAN, bukan tindakan langsung
//
// `P-1` dan `ADR-0004` menetapkan satu tabel hanya ditulis satu sistem, dan
// `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` hari ini ditulis Pega. Work Owner memutuskan 2026-09-23
// aplikasi ini menulis ke **tabelnya sendiri** berisi permintaan beserta pelaku dan
// waktunya; eksekusinya tetap di Pega.
//
// Dengan begitu `P-1` utuh, jejak audit `S-5` terpenuhi, dan tidak ada dua sistem yang
// saling menimpa — kegagalan yang bila terjadi TIDAK menghasilkan galat apa pun, hanya
// data yang berubah sendiri.
//
// # Akibat yang HARUS diketahui pengguna
//
// Klaim TIDAK langsung berubah. Barisnya tetap ada di layar ini sampai Pega menjalankan
// permintaannya. Karena itu layar wajib menandai baris yang permintaannya sudah terkirim —
// tanpa itu, pengguna yang tidak melihat perubahan apa pun akan menekan tombolnya lagi.
type ClaimRequest struct {
	// ID adalah pengenal acak 128 bit dalam heksadesimal, dibangkitkan aplikasi.
	//
	// Acak, bukan berurut: pengenal permintaan tidak boleh membocorkan berapa banyak
	// permintaan yang sudah tercatat.
	ID string

	Kind RequestKind

	// ClaimID adalah `PZINSKEY` klaim yang dimaksud — yang di sistem lama dikirim tombolnya
	// sebagai parameter `casePNC`.
	//
	// TIDAK ada foreign key ke tabel warisan, dan ketiadaannya disengaja: constraint dari
	// tabel milik kita ke tabel milik Pega akan membuat pengarsipan di sisi Pega GAGAL
	// karena baris kita. Kita tidak boleh menghalangi sistem yang sedang melayani produksi.
	ClaimID string

	// ClaimNumber disimpan SEBAGAI SALINAN, bukan dirujuk.
	//
	// Supaya jejak ini tetap terbaca utuh bila klaim warisannya kelak diarsipkan.
	ClaimNumber string

	// Reason adalah alasan yang diketik pengguna. BOLEH kosong.
	//
	// Work Owner memilih efek reopen tanpa mewajibkan alasan (2026-09-23). Kolomnya tetap
	// ada supaya pengguna yang ingin menjelaskan dapat melakukannya, dan supaya kewajiban
	// itu dapat dinyalakan kelak tanpa perubahan skema.
	Reason string

	Status RequestStatus

	// EffectWorkStatus, EffectClaimStatus, dan CopyScope adalah niat yang tercatat.
	// Lihat konstanta di atas.
	EffectWorkStatus  string
	EffectClaimStatus string
	CopyScope         string

	// ActorLogin adalah login pemanggil, dinormalkan huruf besar.
	//
	// Inilah kunci yang Work Owner tetapkan untuk mencocokkan identitas sesi dengan
	// `OPERATOR_ID` sistem lama (`keputusan-implementasi.md` §16.5).
	ActorLogin string

	// ActorName disimpan BERSAMA permintaannya, tidak dirujuk ke tabel pengguna: jejak yang
	// namanya diambil lewat join akan BERUBAH ketika orangnya berganti nama — dan jejak
	// yang dapat berubah bukan jejak.
	ActorName string

	// RequestedAt adalah waktu permintaan, dalam UTC (`DB-8`).
	RequestedAt time.Time
}

// OperatorKey menormalkan login menjadi bentuk yang dipakai sebagai kunci.
//
// Huruf besar dan tanpa spasi tepi. Alasannya nyata, bukan kerapian: export memuat tiga
// nama peran yang muncul dalam dua kapitalisasi sekaligus (`D-58`), dan perbandingan yang
// hanya satu sisinya diseragamkan tidak pernah cocok — gagalnya diam.
func OperatorKey(login string) string {
	return strings.ToUpper(strings.TrimSpace(login))
}

// Normalize merapikan permintaan sebelum disimpan.
func (r ClaimRequest) Normalize() ClaimRequest {
	r.ClaimID = strings.TrimSpace(r.ClaimID)
	r.ClaimNumber = strings.TrimSpace(r.ClaimNumber)
	r.Reason = strings.TrimSpace(r.Reason)
	r.ActorLogin = OperatorKey(r.ActorLogin)
	r.ActorName = strings.TrimSpace(r.ActorName)

	if r.Status == "" {
		r.Status = RequestPending
	}
	if r.RequestedAt.IsZero() {
		// Dibiarkan kosong supaya pemanggil yang lupa mengisinya tertangkap Validate,
		// bukan diam-diam diberi waktu sekarang di tempat yang salah.
		return r
	}
	r.RequestedAt = r.RequestedAt.UTC()
	return r
}

// Panjang maksimum isian, mengikuti lebar kolom pada migrasi `0006`.
//
// Diperiksa di Go supaya pengguna menerima pesan yang dapat dibaca, bukan galat Oracle
// ORA-12899 yang menyebut nama kolom internal.
const (
	MaxReasonLength = 1500
	MaxClaimIDLen   = 64
)

// Validate memeriksa permintaan sebelum disimpan.
//
// Seluruh pelanggaran dikembalikan SEKALIGUS, tidak berhenti pada yang pertama
// (`11-CROSSCUTTING` §1.2 butir 1) — meniru perilaku Pega yang menampilkan semua pesan
// bersamaan.
func (r ClaimRequest) Validate() error {
	var violations []Violation

	if r.Kind != RequestReopen && r.Kind != RequestCopy {
		violations = append(violations, Violation{
			Field:   FieldKind,
			Message: "Jenis permintaan tidak dikenal.",
		})
	}
	if r.ClaimID == "" {
		violations = append(violations, Violation{
			Field:   FieldClaimID,
			Message: "Klaim yang dimaksud tidak disebutkan.",
		})
	} else if len(r.ClaimID) > MaxClaimIDLen {
		violations = append(violations, Violation{
			Field:   FieldClaimID,
			Message: "Kunci klaim melebihi panjang yang dapat disimpan.",
		})
	}
	if r.ActorLogin == "" {
		violations = append(violations, Violation{
			Field:   FieldActor,
			Message: "Identitas pemohon tidak dikenali.",
		})
	}
	if len([]rune(r.Reason)) > MaxReasonLength {
		violations = append(violations, Violation{
			Field:   FieldReason,
			Message: "Alasan melebihi 1.500 karakter.",
		})
	}
	if r.RequestedAt.IsZero() {
		violations = append(violations, Violation{
			Field:   FieldRequestedAt,
			Message: "Waktu permintaan tidak terisi.",
		})
	}

	if len(violations) == 0 {
		return nil
	}
	return NewValidationError(violations)
}

// RequestRepo adalah seam ke penyimpanan permintaan.
//
// Ia TERPISAH dari Repo meski keduanya melayani satu layar, dan pembelahannya mengikuti
// kepemilikan tabel — bukan ukuran berkas: yang satu tidak boleh menulis apa pun, yang
// lain menulis ke tabel milik aplikasi ini sendiri. Menyatukannya akan membuat aturan itu
// bergantung pada kehati-hatian orang yang menyuntingnya berikutnya.
type RequestRepo interface {
	// Record menyimpan satu permintaan.
	//
	// Mengembalikan ErrRequestPending bila permintaan sejenis atas klaim yang sama masih
	// menunggu dijalankan.
	Record(ctx context.Context, request ClaimRequest) error

	// PendingFor mengembalikan permintaan yang masih menunggu untuk setiap klaim yang
	// disebut, dikunci ClaimID.
	//
	// Banyak klaim sekaligus, bukan satu per satu: meminta keadaan per baris tabel adalah
	// kueri di dalam perulangan, hambatan peringkat ketiga pada `15-NFR` §3.2.
	PendingFor(ctx context.Context, claimIDs []string) (map[string][]ClaimRequest, error)
}

// RequestRepoSelector memilih RequestRepo milik satu portal entitas.
//
// Alasannya sama dengan RepoSelector: tabel permintaan dibuat migrasi `0006` di basis data
// SETIAP entitas, karena klaim yang dirujuknya pun ada di sana.
type RequestRepoSelector func(portalAlias string) (RequestRepo, error)

// IDGenerator adalah seam ke pembangkit pengenal permintaan.
//
// Pengenalnya acak, bukan berurut: ia tidak punya makna bisnis dan tidak boleh membocorkan
// berapa banyak permintaan yang sudah tercatat.
//
// Nama methodnya `New`, sama dengan seam sejenis pada modul `komite` dan `registrasi`.
// Keseragaman itu bukan kerapian: ketiganya diisi pembangkit yang sama bentuknya, dan nama
// yang berbeda akan membuat salah satunya tampak seperti seam yang lain.
type IDGenerator interface {
	New() string
}

// Clock adalah seam ke waktu (`F-5`).
//
// Waktu TIDAK pernah dibaca langsung dari jam sistem di dalam modul: `08-TECHNICAL-STRATEGY`
// §4.4 menetapkan konversi dan pembacaan waktu terpusat, dan seam ini yang membuat seluruh
// aturan berbasis waktu dapat diuji secara deterministik.
type Clock interface {
	Now() time.Time
}
