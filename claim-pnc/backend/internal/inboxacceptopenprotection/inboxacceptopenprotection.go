// Package inboxacceptopenprotection adalah inti modul Inbox Accept Open Protection.
//
// # Nama modul ini
//
// Diambil dari judul yang tertulis DI DALAM layarnya sendiri —
// `Section/InputProtection_Section-Section.xml:338` memuat `pyCaption` berbunyi
// **"Inbox Accept Open Protection"**.
//
// Butir menu yang membukanya bernama "Inbox Open Protection"
// (`Navigation/pyCaseWorkerNavigation-Navigation.xml:21846`), tetapi nama itu TIDAK dipakai:
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
// Modul ini menulis TIGA kolom saja: `APPROVAL_STATUS`, `RESOLVED_DATETIME`, dan
// `RESOLVED_BY`. Kolom pembuatan — nomor, polis, klaim, tipe, keterangan, detail
// perubahan — dimiliki modul `inputreqprotection` dan tidak pernah disentuh di sini.
//
// Pembagian itu yang menjaga `P-1` tetap berlaku meski dua modul menyentuh satu tabel:
// keduanya menulis kolom berbeda pada tahap hidup yang berbeda.
//
// # ⚠️ Yang BELUM dibawa: pembukaan proteksi di sistem polis
//
// Di Pega, menyetujui akseptasi TIDAK berhenti pada penandaan status. Flow action
// `AksepProtection` memanggil `Activity/InsertOpenProtectionCase-Act.xml`, yang membuka
// proteksi di SISTEM POLIS SUMBER lewat DB Link — tiga entitas sekaligus:
//
//	RDB List/CheckProtectTable-SQL.xml            SELECT general.mst_buka_proteksi@asmd
//	RDB List/InsertOpenPortectionCaseASM-SQL.xml  INSERT general.mst_buka_proteksi@asmd
//	RDB List/InsertOpenPortectionCase-SQL.xml     INSERT general.mst_buka_proteksi@simasnet
//	RDB List/InsertOpenPortectionCaseSMI-SQL.xml  INSERT general.mst_buka_proteksi@smi
//
// Polanya idempoten — `SELECT COUNT(1) … IF vCount = 0 THEN INSERT` — sesuai `pyMemo`
// rule-nya sendiri: "Cek belum ada baru insert".
//
// Modul ini HANYA menandai `APPROVAL_STATUS`. Akibatnya nyata dan harus dinyatakan:
// **persetujuan di sistem baru belum menimbulkan akibat apa pun di sistem polis.** Proteksi
// tercatat disetujui di tabel ini, tetapi polisnya belum benar-benar terbuka.
//
// Ini BUKAN kelalaian yang dapat ditambal di sini. `D-25` menetapkan seluruh DB Link
// diganti pemanggilan API, dan "API Buka Proteksi" adalah satu dari enam API pengganti yang
// **belum dibangun tim pemilik sistem** (`R-03`). `20-DETAIL-KOMITE-DBLINK.md` §2.6 sudah
// menandainya sebagai satu-satunya objek remote yang MENULIS, dan menuntut penggantinya
// berupa API idempoten — bukan salinan berkala.
//
// Sampai API itu ada, akseptasi di sini wajib dibaca sebagai KEPUTUSAN, bukan sebagai
// pembukaan proteksi. Jalur yang membuka proteksi masih Pega.
//
// ## Empat syarat yang menjaga pembukaan itu — dicatat untuk saat API-nya tiba
//
// `InsertOpenProtectionCase` menjaga langkah penyimpanannya dengan prakondisi
//
//	.TypeProtection != "" && .Keterangan != "" && .CaseID != "" && .PolicyNo != ""
//
// EMPAT syarat, bukan dua. Modul ini memeriksa dua — `CLAIM_NO` dan `POLICY_NO` — dan itu
// BENAR: keduanya diambil dari penyaring `InboxOpenProtection2_RD`, yang menentukan apa yang
// muncul di antrean. Kedua syarat lain menjaga hal yang berbeda.
//
// Perbedaan akibatnya yang penting: di Pega, prakondisi yang tidak terpenuhi membuat
// langkahnya DILEWATI — akseptasinya tetap tercatat, hanya proteksinya tidak dibuka di
// sistem polis. Ia bukan penolakan.
//
// Karena itu `TypeProtection` dan `Keterangan` SENGAJA TIDAK ditambahkan ke syarat akseptasi
// di sini. Menambahkannya akan menolak akseptasi yang di Pega tetap berhasil — lebih ketat
// daripada sistem lama tanpa dasar, dan itu melanggar `P-5`. Keduanya menjadi syarat pada
// adapter pembuka proteksi, saat API-nya dibangun.
//
// ## Yang ikut terbaca dari activity itu, dan belum punya tempat di sistem baru
//
//	Param.APPNAME == "ASM" | "SIMASNET" | "SMI"    tiga entitas, sejalan dengan 3 INSERT
//	@LengthOfPageList(ProtectDouble.pxResults) > 1  pemeriksaan proteksi GANDA
//	Local.typeprotection == "9"                     tipe 9 punya penanganan tersendiri
//	pxRequestor.pxReqServer == <host dev>           hostname hardcode (`D-15`)
//
// Keempatnya milik jalur pembukaan proteksi, bukan jalur akseptasi, sehingga tidak ada yang
// perlu dikerjakan di modul ini hari ini. Pemeriksaan proteksi ganda layak diperhatikan
// khusus saat API dibangun: ia aturan bisnis, bukan detail teknis.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxacceptopenprotection

import (
	"context"
	"strings"
	"time"

	"claim-pnc/internal/platform/pagination"
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

// Normalize mengembalikan antrean yang dijamin dikenali.
//
// Yang tidak dikenali jatuh ke NON PREMI — arah yang sama dengan Filter.Normalize, dan
// alasannya di sana. Keduanya WAJIB sepakat: pemeriksaan kewenangan memakai nilai ini,
// sedangkan yang benar-benar menyaring memakai nilai dari Filter. Bila keduanya berbeda,
// seseorang dapat lolos pemeriksaan atas satu antrean lalu dilayani antrean yang lain.
func (q Queue) Normalize() Queue {
	if q.Valid() {
		return q
	}
	return QueueNonPremium
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

// Tiga kode tipe proteksi yang menjadi PERCABANGAN, bukan sekadar label.
//
// Ketiganya tetap konstanta di kode — bukan dibaca dari master — karena master hanya memuat
// `PROTECTION_TYPE_ID` dan `PROTECTION_TYPE_NAME`; tidak ada kolom yang menyatakan "tipe ini
// masuk antrean premi" atau "tipe ini menuntut detail perubahan". Menurunkan perilaku dari
// NAMA akan membuat satu suntingan ejaan mengubah antrean akseptasi, tanpa galat dan tanpa
// gejala.
//
// Nilainya sama dengan konstanta senama di `inputreqprotection`, dan memang harus sama:
// modul itu yang MENULIS `OLD_DATA`/`NEW_DATA` menurut tipe, modul ini yang MEMBACANYA.
const (
	// TypePremium memisahkan kedua antrean. Terbukti dari penyaring kedua Report
	// Definition, bukan dari daftar nilai — daftar itu tinggal di rule Property yang tidak
	// ikut diekspor (`R-16`). Master menamainya "Premi Belum Lunas".
	TypePremium = "2"

	// TypeChangeLossDate adalah permintaan perubahan Tanggal Kejadian (DOL).
	TypeChangeLossDate = "7"

	// TypeChangeCauseOfLoss adalah permintaan perubahan Penyebab Kerugian.
	TypeChangeCauseOfLoss = "8"
)

// ChangeDetail adalah isi panel "Detail Perubahan" pada form akseptasi.
//
// # Kenapa ia ada, dan kenapa sempat hilang
//
// `Section/AcceptProtectionSection-Section.xml` memuat container bersyarat
//
//	pyContainerVisibleWhen: .TypeProtection==8 || .TypeProtection==7
//	  .TypeProtection==7 -> pyTitle "Detail Perubahan DOL"
//	  .TypeProtection==8 -> pyTitle "Detail Perubahan Cause Of Loss"
//
// Bagi kedua tipe itu, **melihat nilai sebelum dan sesudah ADALAH inti keputusannya** — itu
// satu-satunya alasan permintaannya diajukan. Menyembunyikannya membuat petugas menyetujui
// perubahan tanpa tahu apa yang diubah.
//
// Versi pertama modul ini tidak memuatnya, dengan alasan tertulis bahwa "layar akseptasi
// menampilkannya sebagai keterangan". Alasan itu KELIRU: Pega menampilkannya sebagai panel
// terurai berjudul sendiri, bukan melebur ke Keterangan.
//
// # Bentuknya mengikuti sepasang kolom yang melayani dua tipe
//
// `OLD_DATA` dan `NEW_DATA` berisi hal berbeda tergantung tipe (`kolom-open-protection.md`
// §3). Struct ini karena itu memuat dua pasang field yang TIDAK PERNAH terisi bersamaan —
// disatukan karena keduanya sama-sama menjawab "apa yang diminta berubah", dan memisahkannya
// menjadi dua tipe akan memaksa setiap pemanggil bercabang.
type ChangeDetail struct {
	// LossDateBefore dan LossDateAfter dipakai TypeChangeLossDate.
	//
	// Label layarnya "Current Date Of Loss" dan "Next Date Of Loss".
	//
	// Keduanya pointer supaya "tidak diisi" dapat dibedakan dari "tanggal nol" — tanggal nol
	// pada permintaan perubahan DOL tidak berarti apa-apa, dan menampilkannya sebagai
	// 1 Januari tahun 1 akan tampak seperti data rusak.
	LossDateBefore *time.Time
	LossDateAfter  *time.Time

	// CauseOfLossBefore dan CauseOfLossAfter dipakai TypeChangeCauseOfLoss.
	//
	// Label layarnya "Cause Of Loss Dipilih" dan "Next Cause Of Loss" — "Dipilih" berarti
	// yang berlaku pada klaim hari ini, bukan yang baru diminta.
	//
	// Padanannya di `inputreqprotection` bernama `CauseOfLossID` dan `CauseOfLossMasterID`.
	// Dinamai ulang di sini supaya arah perubahannya terbaca dari namanya sendiri; yang
	// dipetakan tetap kolom yang sama.
	CauseOfLossBefore string
	CauseOfLossAfter  string

	// ObjectName dan BranchName melengkapi kedua panel, dan berada di dalam container
	// bersyarat yang sama.
	ObjectName string
	BranchName string
}

// Empty menyatakan tidak ada satu pun detail perubahan yang terisi.
//
// Dipakai layar untuk membedakan "tipe ini memang tidak punya detail" dari "punya, tetapi
// kolomnya kosong" — yang kedua terjadi pada baris warisan Pega, yang `OLD_DATA` dan
// `NEW_DATA`-nya tidak punya kolom asal saat disalin (`kolom-open-protection.md` §9).
func (d ChangeDetail) Empty() bool {
	return d.LossDateBefore == nil && d.LossDateAfter == nil &&
		strings.TrimSpace(d.CauseOfLossBefore) == "" &&
		strings.TrimSpace(d.CauseOfLossAfter) == "" &&
		strings.TrimSpace(d.ObjectName) == "" &&
		strings.TrimSpace(d.BranchName) == ""
}

// ShowsChangeDetail menyatakan tipe proteksi ini memunculkan panel "Detail Perubahan".
//
// Meniru `pyContainerVisibleWhen: .TypeProtection==8 || .TypeProtection==7` apa adanya.
// Tipe lain tidak memunculkannya, dan isian yang kebetulan tersimpan di kedua kolom pada
// tipe lain TIDAK ditampilkan — sejalan dengan `encodeChangeDetail` di `inputreqprotection`
// yang memang tidak pernah menuliskannya.
func ShowsChangeDetail(protectionType string) bool {
	switch strings.TrimSpace(protectionType) {
	case TypeChangeLossDate, TypeChangeCauseOfLoss:
		return true
	default:
		return false
	}
}

// ── Antrean ketiga yang TIDAK dibawa ─────────────────────────────────────────────

// Layar lama memuat grid KETIGA, `InboxOpenProtection2_RD_collection_askredit`, yang
// syarat tampilnya adalah
//
//	OperatorID.pyUserIdentifier == <satu alamat Gmail pribadi>
//
// (`Section/InputProtection_Section-Section.xml:10342`). Penyaringnya pun lebih longgar:
// hanya `PolicyNo IS NOT NULL AND AcceptStatus IS NULL`, tanpa syarat nomor klaim dan tanpa
// syarat tipe — sehingga orang itu melihat proteksi yang tidak terlihat siapa pun.
//
// Dua grid lainnya juga memeriksa satu Operator ID tertentu pada syarat tampilnya (`:1592`
// dan `:6036`).
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

// Nilai `APPROVAL_STATUS`.
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

// Status menerjemahkan keputusan menjadi nilai kolom `APPROVAL_STATUS`.
func (d Decision) Status() string {
	if d == DecisionApprove {
		return AcceptApproved
	}
	return AcceptRejected
}

// Protection adalah satu baris pada layar akseptasi.
//
// Field di sini adalah yang DITAMPILKAN layar ini beserta yang dibutuhkan form
// akseptasinya — termasuk detail perubahan DOL dan Cause of Loss dalam bentuk terurai,
// karena form Pega menampilkannya sebagai panel tersendiri. Lihat ChangeDetail.
//
// Yang MENGUBAHNYA tetap modul `inputreqprotection`; modul ini hanya membacanya.
type Protection struct {
	Number       string // kolom "No Proteksi"
	PolicyNumber string // kolom "No Polis"
	ClaimNumber  string // kolom "No Klaim"

	// ClaimReference adalah `ID_CLAIM` — kunci klaim pada `POOLDATA.T_CLAIM_PNC.CLAIMID`.
	//
	// Bagi klaim warisan Pega ia berbentuk `ASM-FW-GCNMFW-WORK PNC-1865`; bagi klaim sistem
	// baru ia sama dengan nomor klaimnya (`inputreqprotection.ClaimReferenceOf`).
	//
	// TIDAK ditampilkan layar. Ia ada karena penerapan perubahan ke klaim memakainya sebagai
	// kunci — memakai `ClaimNumber` tidak akan menemukan baris mana pun untuk klaim warisan.
	ClaimReference string
	Type           string // kolom "Tipe Proteksi" — PROTECTION_TYPE_ID

	// TypeName adalah nama tipe dari `POOLDATA.M_CLAIM_PROTECTION_TYPE`, yang DILIHAT
	// petugas akseptasi.
	//
	// Dibaca lewat LEFT JOIN, bukan disimpan di tabel proteksi — sehingga mengganti nama
	// sebuah tipe langsung berlaku pada seluruh baris lama.
	//
	// KOSONG bila kodenya tidak terdaftar di master. Layar menampilkan kodenya apa adanya
	// dalam keadaan itu: petugas yang menyetujui pembukaan proteksi berhak tahu bahwa tipe
	// yang dihadapinya tidak dikenal sistem.
	TypeName string

	InputDate time.Time // kolom "Tanggal Proteksi Dibuat"
	Note      string    // kolom "Keterangan"
	CreatedBy string    // kolom "User Create"

	// Change adalah isi panel "Detail Perubahan", terisi hanya bagi tipe '7' dan '8'.
	//
	// Dibaca dari `OLD_DATA`, `NEW_DATA`, `OBJECT_NAME`, dan `BRANCH_NAME` — keempatnya
	// ditulis modul `inputreqprotection` dan TIDAK PERNAH disentuh di sini (`P-1`).
	Change ChangeDetail

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

// Batas paginasi.
//
// DefaultLimit mengikuti `<pyPageSize>20</pyPageSize>` yang terpasang pada KETIGA grid
// `Section/InputProtection_Section-Section.xml` — dibaca dari section, bukan dari harness:
// harness hanya meng-embed section, dan yang menentukan ukuran halaman adalah grid.
//
// Angkanya SENGAJA disebut di sini. Modul inbox lain memakai 50 karena section masing-masing
// memang 50 (`inboxrclpucl`, `inboxanalystdoctor`); layar ini 20. Menyeragamkan keduanya
// akan mengubah jumlah baris per halaman tanpa dasar — dan `P-5` menuntut perilaku yang
// sama dengan Pega kecuali yang diputuskan berbeda secara tertulis.
//
// MaxLimit BUKAN angka Pega. Ia pagar aplikasi ini sendiri terhadap permintaan yang
// meminta halaman raksasa, sejalan dengan `10-API-STRATEGY.md` §4 yang menolak — bukan
// memenuhi — permintaan di atas batas.
const (
	DefaultLimit = 20
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
	f.Queue = f.Queue.Normalize()

	f.Limit, f.Offset = pagination.LimitOffset(f.Limit, f.Offset, DefaultLimit, MaxLimit)
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
