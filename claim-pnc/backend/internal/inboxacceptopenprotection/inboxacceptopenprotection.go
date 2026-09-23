// Package inboxacceptopenprotection adalah inti modul Inbox Accept Open Protection.
//
// # Nama modul ini
//
// Diambil dari judul yang tertulis DI DALAM layarnya sendiri —
// `Section/InputProtection_Section-Section.xml:338` memuat `pyCaption` berbunyi
// **"Inbox Accept Open Protection"**.
//
// Butir menu yang membukanya bernama "Inbox Open Protection"
// (`Navigation/pyCaseWorkerNavigation-Navigation.xml:53444`), tetapi nama itu TIDAK dipakai:
// ia bertabrakan dengan judul layar modul `inputreqprotection`, yang juga berbunyi
// "Inbox Open Protection". Kedua nama di Pega memang bersilang:
//
//	butir menu "Input Req Protection"   -> harness InputReqProtection_Harness
//	                                       judulnya "Inbox Open Protection"
//	butir menu "Inbox Open Protection"  -> harness InputProtection_Harness
//	                                       judulnya "Inbox Accept Open Protection"
//
// Kata "Accept" yang membedakan keduanya karena itu dipertahankan.
//
// # Yang dimigrasikan
//
//	Harness/InputProtection_Harness-Harness.xml                     layar rujukan
//	Section/InputProtection_Section-Section.xml                     kolom dan pemisahan antrean
//	Section/AcceptProtectionSection-Section.xml                     form akseptasi
//	Report Definition/InboxOpenProtection2_RD-RD.xml                antrean NON PREMI
//	Report Definition/InboxOpenProtection2_RD_collection-RD.xml     antrean PREMI
//	Flow/CreateProtection_Flow.xml                                  cabang Accept / Reject
//	When/IsAcceptProtection-When.xml                                arti status akseptasi
//
// # Kepemilikan tulis
//
// Modul ini menulis TIGA kolom saja: `STATUS_AKSEPTASI`, `TANGGAL_AKSEPTASI`, dan
// `DIAKSEP_OLEH`. Kolom pembuatan — nomor, polis, klaim, tipe, keterangan, detail
// perubahan — dimiliki modul `inputreqprotection` dan tidak pernah disentuh di sini.
//
// Pembagian itu yang menjaga `P-1` tetap berlaku meski dua modul menyentuh satu tabel:
// keduanya menulis kolom berbeda pada tahap hidup yang berbeda.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxacceptopenprotection

import (
	"context"
	"strings"
	"time"
)

// ── Antrean ──────────────────────────────────────────────────────────────────────

// Queue adalah antrean akseptasi yang sedang dibuka.
//
// Layar lama memuat TIGA grid berisi kolom yang sama persis, dibedakan hanya oleh syarat
// tampilnya. Dua di antaranya nyata sebagai pembagian kerja; yang ketiga tidak dibawa.
type Queue string

const (
	// QueuePremium adalah antrean proteksi klaim PREMI.
	//
	// `InboxOpenProtection2_RD_collection` menyaring `.TypeProtection = "2"`, dan gridnya
	// hanya tampil bagi access group `GCNMFW:PncCollection` — bagian penagihan premi.
	QueuePremium Queue = "premi"

	// QueueNonPremium adalah antrean selain PREMI.
	//
	// `InboxOpenProtection2_RD` menyaring `.TypeProtection != "2"`.
	QueueNonPremium Queue = "non-premi"
)

// Valid menyatakan antrean ini dikenali.
func (q Queue) Valid() bool {
	return q == QueuePremium || q == QueueNonPremium
}

// TypeFilter menerjemahkan antrean menjadi penyaring tipe proteksi.
//
// Mengembalikan nilai pembanding dan apakah pembandingannya "sama dengan". Bentuk itu
// dipilih supaya penyimpanan menuliskan `= :1` atau `<> :1` atas nilai yang sama — bukan
// dua kueri terpisah yang lama-lama berbeda isinya.
func (q Queue) TypeFilter() (value string, equal bool) {
	if q == QueuePremium {
		return TypePremium, true
	}
	return TypePremium, false
}

// TypePremium adalah kode tipe proteksi yang memisahkan kedua antrean.
//
// Nilainya terbukti dari penyaring kedua Report Definition, bukan dari daftar nilai —
// daftar itu tinggal di rule Property yang tidak ikut diekspor (`R-16`).
const TypePremium = "2"

// ── Antrean ketiga yang TIDAK dibawa ─────────────────────────────────────────────

// Layar lama memuat grid KETIGA, `InboxOpenProtection2_RD_collection_askredit`, yang
// syarat tampilnya adalah
//
//	OperatorID.pyUserIdentifier == <satu alamat Gmail pribadi>
//
// (`Section/InputProtection_Section-Section.xml:25104`). Penyaringnya pun lebih longgar:
// hanya `PolicyNo IS NOT NULL AND AcceptStatus IS NULL`, tanpa syarat nomor klaim dan tanpa
// syarat tipe — sehingga orang itu melihat proteksi yang tidak terlihat siapa pun.
//
// Dua grid lainnya juga memeriksa satu Operator ID tertentu pada syarat tampilnya (`:3989`
// dan `:14717`).
//
// # Ketiganya TIDAK dibawa
//
// `D-15` menetapkan tidak ada nilai bisnis yang boleh di-hardcode, dan `D-67` menetapkan
// tidak ada akun pribadi yang dibawa ke sistem baru. Alamat Gmail itu termasuk dalam 66
// alamat dan 24 Operator ID yang `F-4` hapus.
//
// Ini SELISIH TERENCANA yang harus dinyatakan di muka saat uji kesetaraan dijalankan
// (`P-5`): orang yang bersangkutan akan melihat antrean yang berbeda dari hari ini. Bila
// keleluasaan itu memang dibutuhkan bisnis, ia harus kembali sebagai PERAN di master data —
// bukan sebagai nama orang di dalam kode.

// ── Aggregate ────────────────────────────────────────────────────────────────────

// Nilai `STATUS_AKSEPTASI`.
//
//	""   belum diakseptasi — disimpan NULL, bukan teks kosong
//	"1"  disetujui  (`When/IsAcceptProtection-When.xml`: `.AcceptStatus = "1"`)
//	"2"  ditolak    (cabang Else pada `Flow/CreateProtection_Flow.xml`)
//
// Ketiga Report Definition menyaring dengan `IS NULL`. Baris yang menyimpan teks kosong
// tidak cocok dengan penyaring itu dan HILANG dari seluruh inbox tanpa satu pun galat.
const (
	AcceptPending  = ""
	AcceptApproved = "1"
	AcceptRejected = "2"
)

// Decision adalah keputusan akseptasi yang diambil petugas.
type Decision string

const (
	DecisionApprove Decision = "setuju"
	DecisionReject  Decision = "tolak"
)

// Valid menyatakan keputusan ini dikenali.
func (d Decision) Valid() bool {
	return d == DecisionApprove || d == DecisionReject
}

// Status menerjemahkan keputusan menjadi nilai kolom `STATUS_AKSEPTASI`.
func (d Decision) Status() string {
	if d == DecisionApprove {
		return AcceptApproved
	}
	return AcceptRejected
}

// Protection adalah satu baris pada layar akseptasi.
//
// Field di sini adalah yang DITAMPILKAN layar ini beserta yang dibutuhkan form
// akseptasinya. Ia sengaja tidak memuat detail perubahan DOL dan Cause of Loss dalam bentuk
// terurai: layar akseptasi menampilkannya sebagai keterangan, dan yang mengubahnya adalah
// modul `inputreqprotection`.
type Protection struct {
	Number       string // kolom "No Proteksi"
	PolicyNumber string // kolom "No Polis"
	ClaimNumber  string // kolom "No Klaim"
	Type         string // kolom "Tipe Proteksi"

	InputDate time.Time // kolom "Tanggal Proteksi Dibuat"
	Note      string    // kolom "Keterangan"
	CreatedBy string    // kolom "User Create"

	// AcceptStatus, AcceptedAt, dan AcceptedBy adalah tiga kolom yang DITULIS modul ini.
	AcceptStatus string
	AcceptedAt   *time.Time
	AcceptedBy   string

	// InsuredName, PolicyStart, dan PolicyEnd ditampilkan pada FORM akseptasi, bukan pada
	// daftar — `Section/AcceptProtectionSection-Section.xml` memuat "Nama Tertanggung",
	// "Start Date Time", dan "End Date Time".
	//
	// Ketiganya datang dari snapshot polis, bukan dari proteksi itu sendiri. Bila snapshot
	// belum tersedia, ketiganya kosong dan form tetap dapat diakseptasi — layar lama pun
	// tidak menjadikannya syarat.
	InsuredName string
	PolicyStart *time.Time
	PolicyEnd   *time.Time
}

// Pending menyatakan proteksi ini belum diakseptasi.
func (p Protection) Pending() bool {
	return strings.TrimSpace(p.AcceptStatus) == AcceptPending
}

// Approved menyatakan proteksi ini disetujui.
func (p Protection) Approved() bool {
	return strings.TrimSpace(p.AcceptStatus) == AcceptApproved
}

// QueueOf menyatakan antrean tempat sebuah proteksi seharusnya berada.
func QueueOf(protectionType string) Queue {
	if strings.TrimSpace(protectionType) == TypePremium {
		return QueuePremium
	}
	return QueueNonPremium
}

// ── Pembacaan daftar ─────────────────────────────────────────────────────────────

// Filter adalah penyaring dan paginasi layar akseptasi.
//
// # Tiga syarat yang TIDAK dapat dimatikan pemanggil
//
// `InboxOpenProtection2_RD` menyaring
//
//	CaseID IS NOT NULL  AND  PolicyNo IS NOT NULL  AND  AcceptStatus IS NULL
//
// Ketiganya bukan pilihan pengguna melainkan DEFINISI layar ini: yang ditampilkan hanyalah
// proteksi yang sudah lengkap dan belum diakseptasi. Karena itu ketiganya tidak muncul
// sebagai field di sini — penyimpanan menerapkannya selalu.
//
// Perhatikan bedanya dengan modul `inputreqprotection`, yang penyaringnya HANYA
// `AcceptStatus IS NULL`. Perbedaan itulah yang membuat proteksi rancangan — yang belum
// tertaut klaim — tidak pernah sampai ke meja petugas akseptasi.
type Filter struct {
	// Queue wajib diisi dan menentukan antrean mana yang dibaca.
	Queue Queue

	// Search mencari pada No Proteksi, No Polis, dan No Klaim sekaligus.
	//
	// Layar lama tidak punya kotak pencarian. Ia ditambahkan karena daftarnya tumbuh tanpa
	// batas seiring waktu — PENAMBAHAN yang disadari, dicatat di
	// `docs/keputusan-implementasi.md`.
	Search string

	Limit  int
	Offset int
}

// Batas paginasi. DefaultLimit mengikuti `pyPageSize` layar lama.
const (
	DefaultLimit = 50
	MaxLimit     = 100
)

// Normalize mengembalikan filter dengan nilai yang dijamin masuk akal.
//
// Antrean yang tidak dikenali jatuh ke NON PREMI, bukan ke PREMI. Arahnya dipilih dengan
// sengaja: NON PREMI adalah antrean yang penyaringnya `!= "2"`, sehingga kekeliruan
// menampilkan proteksi yang bukan haknya lebih kecil daripada sebaliknya — dan antrean
// PREMI menyangkut penagihan premi, yang pemiliknya satu peran tertentu.
func (f Filter) Normalize() Filter {
	f.Search = strings.TrimSpace(f.Search)

	if !f.Queue.Valid() {
		f.Queue = QueueNonPremium
	}
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
	Protections []Protection
	Total       int
}

// ── Seam ─────────────────────────────────────────────────────────────────────────

// Repo adalah seam ke penyimpanan proteksi.
//
// Dideklarasikan DI SINI, di paket yang memakainya. Diisi `repo/sqlstore` terhadap Oracle
// dan `repo/memory` untuk pengujian.
type Repo interface {
	// List membaca satu halaman antrean akseptasi.
	List(ctx context.Context, f Filter) (Page, error)

	// Get membaca satu proteksi menurut nomornya.
	//
	// TIDAK menyaring status akseptasi: form akseptasi harus tetap dapat dibuka untuk
	// proteksi yang baru saja diputuskan orang lain — supaya pesannya dapat menjelaskan apa
	// yang terjadi, bukan sekadar "tidak ditemukan".
	Get(ctx context.Context, number string) (Protection, error)

	// Decide menuliskan keputusan akseptasi.
	//
	// Adapter WAJIB menolak proteksi yang sudah diakseptasi, meski pemanggil sudah
	// memeriksanya. Pemeriksaan di lapisan atas menjaga pengguna dari kesalahan;
	// pemeriksaan di penyimpanan menjaga data dari dua petugas yang menekan tombol
	// bersamaan — dan pada layar antrean bersama, itu bukan kejadian langka.
	Decide(ctx context.Context, number string, d Decision, by string, at time.Time) (Protection, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Portal yang tidak dikenal menghasilkan galat — TIDAK PERNAH dialihkan ke koneksi utama.
// Jatuh ke koneksi default berarti seseorang mengakseptasi proteksi milik badan hukum lain
// (`R-20`, `TKT-F6-002`).
type RepoSelector func(portalAlias string) (Repo, error)
