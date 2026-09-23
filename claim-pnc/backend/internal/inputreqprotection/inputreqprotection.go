// Package inputreqprotection adalah inti modul Input Req Protection.
//
// # Nama modul ini
//
// Diambil dari butir menu Pega yang membuka layarnya:
// `Navigation/pyCaseWorkerNavigation-Navigation.xml` memuat `pyCaptionPrompt` berbunyi
// **"Input Req Protection"** yang menunjuk `InputReqProtection_Harness`. `D-81` menetapkan
// nama modul mengikuti nama yang dipakai Work Owner.
//
// Namanya SENGAJA tidak diambil dari judul di dalam harness itu sendiri, yang berbunyi
// "Inbox Open Protection". Judul itu bertabrakan dengan butir menu LAIN yang juga bernama
// "Inbox Open Protection" tetapi menunjuk harness berbeda — lihat modul
// `inboxacceptopenprotection`. Kedua nama di Pega memang bersilang.
//
// # Yang dimigrasikan
//
//	Harness/InputReqProtection_Harness-Harness.xml        layar rujukan
//	Section/InboxReqProtection_Section-Section.xml        kolom beserta judulnya
//	Report Definition/InboxReqOpenProtection_RD-RD.xml    penyaring dan pengurutan
//	Section/InputProtectionSection-Section.xml            form input
//	Activity/ValidationInputProtection-Act.xml            validasi form
//	Activity/InsertOpenProtectionCase-Act.xml             syarat penyimpanan
//	Flow/CreateProtection_Flow.xml                        alur yang menaunginya
//
// # Apa itu Open Protection
//
// Permintaan pembukaan proteksi atas sebuah polis, di luar alur klaim normal. Ia case type
// TERSENDIRI di Pega — `ASM-FW-GCNMFW-Work-OpenProtection`, dibuat
// `Activity/CreateCaseOpenProtection_act-Act.xml` dengan `pyID` sendiri — bukan bagian dari
// klaim.
//
// Alurnya tiga langkah (`Flow/CreateProtection_Flow.xml`):
//
//	Input Protection  ->  Akseptasi (workbasket ProtectionPNC)  ->  Diterima / Ditolak
//
// Modul ini memiliki langkah PERTAMA. Langkah kedua milik `inboxacceptopenprotection`.
//
// # JANGAN tertukar dengan "proteksi data"
//
// Modul `master-masking` dan `riwayat-klaim` memakai kata "proteksi" untuk hal yang SAMA
// SEKALI BERBEDA: jatah pencarian data nasabah, tersimpan di
// `POOLDATA.MST_PROTEKSI_DATA_PNC` berkolom LOGIN, PASSWORD, STS_KTP. Itu bukan Open
// Protection, dan kedua tabel tidak boleh dicampur.
//
// # Sumber datanya
//
// `POOLDATA.T_CLAIM_OPENPROTECTION` — tabel datar BARU, ditetapkan Work Owner 2026-09-23.
// Ia BELUM ADA; yang membuatnya Work Owner, bukan berkas migrasi di repositori ini. Daftar
// kolom beserta asalnya di Pega ada di `docs/kolom-open-protection.md`.
//
// Tabel baru dibutuhkan karena Open Protection tidak muat di tabel klaim: satu klaim
// menampung BANYAK proteksi (`Activity/CekStatusOPCKlaimPNCPengkinianData-Act.xml:1296`
// menulis `OpenProtectionList(<APPEND>)`), dan proteksi LAHIR TANPA KLAIM — justru baris
// tanpa klaim itulah yang masih dapat disunting pemohonnya.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inputreqprotection

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ── Tipe proteksi ────────────────────────────────────────────────────────────────

// Nilai `TIPE_PROTEKSI` yang TERBUKTI dari export.
//
// Ketiganya diturunkan dari perilaku yang dapat dilihat, bukan dari daftar nilai — daftar
// itu tinggal di rule Property, dan folder `Property/` tidak ikut diekspor (`R-16`).
//
//	'2'  dipisahkan menjadi antrean tersendiri milik peran PncCollection
//	     (`InboxOpenProtection2_RD_collection` menyaring `= "2"`, dan
//	      `InboxOpenProtection2_RD` menyaring `!= "2"` untuk peran lain)
//	'7'  memunculkan panel "Detail Perubahan DOL"
//	     (`Section/InputProtectionSection-Section.xml:2728`)
//	'8'  memunculkan panel "Detail Perubahan Cause Of Loss" (`:3638`)
//
// Nilai '1', '3', '4', '5', dan '6' juga dipakai di berbagai activity, tetapi LABELNYA
// TIDAK DIKETAHUI. Modul ini karena itu tidak memetakan kode ke label sama sekali: layar
// menampilkan kodenya apa adanya sampai Work Owner menyerahkan daftarnya.
//
// Menebak labelnya akan lebih buruk daripada menampilkan kode. Kode mentah di layar segera
// ditanyakan pengguna; label yang salah diterima begitu saja.
const (
	// TypePremium adalah proteksi klaim PREMI, yang diakseptasi peran penagihan premi.
	TypePremium = "2"

	// TypeChangeLossDate adalah permintaan perubahan Tanggal Kejadian (DOL).
	TypeChangeLossDate = "7"

	// TypeChangeCauseOfLoss adalah permintaan perubahan Penyebab Kerugian.
	TypeChangeCauseOfLoss = "8"
)

// IsPremium menyatakan sebuah tipe proteksi masuk antrean PREMI.
//
// Pembandingnya SATU nilai, dan sisanya NON PREMI — persis bentuk penyaring kedua Report
// Definition. Menuliskannya sebagai daftar nilai NON PREMI akan salah begitu Work Owner
// menambah tipe baru: tipe yang belum dikenal akan lenyap dari kedua antrean sekaligus.
func IsPremium(protectionType string) bool {
	return strings.TrimSpace(protectionType) == TypePremium
}

// NeedsChangeDetail menyatakan tipe proteksi ini menuntut panel detail perubahan terisi.
func NeedsChangeDetail(protectionType string) bool {
	t := strings.TrimSpace(protectionType)
	return t == TypeChangeLossDate || t == TypeChangeCauseOfLoss
}

// ── Status akseptasi ─────────────────────────────────────────────────────────────

// Nilai `STATUS_AKSEPTASI`.
//
//	""   belum diakseptasi — disimpan NULL, bukan teks kosong
//	"1"  disetujui  (`When/IsAcceptProtection-When.xml`: `.AcceptStatus = "1"`)
//	"2"  ditolak    (cabang Else pada `Flow/CreateProtection_Flow.xml`)
//
// # NULL, bukan teks kosong — dan ini bukan kerapian
//
// Ketiga Report Definition menyaring dengan operator `IS NULL`. Baris yang menyimpan teks
// kosong TIDAK COCOK dengan penyaring itu, sehingga ia hilang dari seluruh inbox — tanpa
// galat, tanpa pesan, tanpa gejala. Adapter penyimpanan WAJIB menuliskannya sebagai NULL.
const (
	AcceptPending  = ""
	AcceptApproved = "1"
	AcceptRejected = "2"
)

// ── Aggregate ────────────────────────────────────────────────────────────────────

// Protection adalah satu permintaan pembukaan proteksi.
//
// Field di sini adalah yang dibutuhkan LAYAR INI dan form inputnya. Ia bukan salinan utuh
// work object Pega; kolom yang hanya dipakai layar akseptasi hidup di modul
// `inboxacceptopenprotection`.
type Protection struct {
	// Number adalah kolom layar **"No Proteksi"** — `.pyID` di Pega.
	//
	// Ia SEKALIGUS kunci baris: kolom `ID` pada `POOLDATA.T_CLAIM_OPENPROTECTION` bertipe
	// `VARCHAR2(100)` dan berisi nomor ini apa adanya. Tidak ada kunci teknis terpisah.
	//
	// Sempat ada field `ID int64` di sini, dengan anggapan nomornya diterbitkan sequence
	// dan disimpan terpisah dari kunci baris. Pembacaan katalog Oracle pada 2026-09-23
	// membantahnya — dan menyimpan field yang selalu bernilai nol lebih menyesatkan
	// daripada tidak menyimpannya sama sekali.
	//
	// Dua bentuk hidup berdampingan permanen:
	//
	//	OPC-XXX         warisan Pega, dibaca apa adanya
	//	OPCN.YY.xxxx    terbitan aplikasi ini (keputusan Work Owner 2026-09-23)
	//
	// Lihat FormatNumber untuk alasan prefixnya dibedakan.
	Number string

	// PolicyNumber — kolom **"No Polis"**, `.PolicyNo`. Wajib diisi.
	PolicyNumber string

	// ClaimNumber — kolom **"No Klaim"**, `.CaseID`.
	//
	// KOSONG selama proteksi belum ditautkan ke klaim, dan justru saat kosong itulah
	// pemohon masih boleh menyuntingnya. Lihat Editable.
	ClaimNumber string

	// ClaimReference adalah klaim yang BENAR-BENAR DITEMUKAN, `.PNCCaseID`.
	//
	// Ia berbeda dari ClaimNumber: yang satu diketik pengguna, yang satu hasil pencarian.
	// `Activity/ValidationInputProtection-Act.xml` menolak penyimpanan ketika field ini
	// kosong, dengan pesan "Silakan Tulis dan Cari Ulang No Klaim" — mengetik nomor klaim
	// saja tidak cukup.
	ClaimReference string

	// Type — kolom **"Tipe Proteksi"**, `.TypeProtection`. Lihat konstanta di atas.
	Type string

	// InputDate — kolom **"Tanggal Proteksi Dibuat"**, `.InputDate`.
	InputDate time.Time

	// Note — kolom **"Keterangan"**, `.Keterangan`. Wajib diisi.
	Note string

	// AcceptStatus menentukan apakah baris ini masih tampil di inbox. Lihat konstanta.
	AcceptStatus string

	// CreatedBy — kolom **"User Create"**, `.pxCreateOpName`.
	CreatedBy string

	// CreatedAt adalah waktu baris dibuat, `.pxCreateDateTime`. Dipakai mengurutkan.
	CreatedAt time.Time

	// ChangeDetail terisi hanya untuk Type '7' dan '8'.
	ChangeDetail ChangeDetail
}

// ChangeDetail adalah isi panel "Detail Perubahan" pada form input.
//
// Kedua panel di Pega saling meniadakan — `.TypeProtection==7` memunculkan yang satu,
// `==8` memunculkan yang lain — sehingga field di sini tidak pernah terisi bersamaan.
// Keduanya tetap satu struct karena keduanya sama-sama menjelaskan APA yang diminta
// berubah, dan memisahkannya menjadi dua tipe akan memaksa setiap pemanggil bercabang.
type ChangeDetail struct {
	// LossDateBefore dan LossDateAfter dipakai Type '7'.
	//
	// Keduanya pointer supaya "tidak diisi" dapat dibedakan dari "tanggal nol". Tanggal nol
	// pada permintaan perubahan DOL adalah nilai yang tidak berarti apa-apa, dan
	// menampilkannya sebagai 1 Januari tahun 1 akan tampak seperti data rusak.
	LossDateBefore *time.Time
	LossDateAfter  *time.Time

	// CauseOfLossID dan CauseOfLossMasterID dipakai Type '8'.
	CauseOfLossID       string
	CauseOfLossMasterID string

	// ObjectName dan BranchName melengkapi keduanya.
	ObjectName string
	BranchName string
}

// Editable menyatakan proteksi ini masih boleh disunting pemohonnya.
//
// # Aturannya satu baris, dan ia ada di layar lama
//
// `Section/InboxReqProtection_Section-Section.xml:8657` menonaktifkan tautan baris ketika
// .CaseID tidak kosong. Jadi begitu sebuah proteksi tertaut ke klaim, pemohon tidak lagi dapat
// membukanya — ia sudah menjadi urusan akseptasi.
//
// # Yang BELUM terjawab, dan sengaja tidak ditebak di sini
//
// `Activity/InsertOpenProtectionCase-Act.xml:921` memasang prakondisi yang menuntut
// `.CaseID` TERISI sebelum sebuah langkah penyimpanan berjalan. Dibaca bersama aturan di
// atas, keduanya tampak bertentangan: kalau nomor klaim wajib sejak awal, setiap baris
// akan terkunci seketika dan tautannya tidak pernah aktif.
//
// Bacaan yang paling sesuai bukti: proteksi lahir sebagai rancangan tanpa klaim, dan
// prakondisi itu menjaga langkah yang memang hanya berlaku bagi proteksi yang sudah
// tertaut. Itu juga yang menjelaskan mengapa layar akseptasi menyaring
// `NO_KLAIM IS NOT NULL` sementara layar ini tidak menyaringnya sama sekali.
//
// Perilaku yang direplikasi di sini adalah yang TERLIHAT di layar — penguncian oleh
// ClaimNumber. Pertanyaannya dicatat di `docs/catatan-pengembangan.md` untuk Work Owner,
// bukan diselesaikan dengan tebakan.
func (p Protection) Editable() bool {
	return strings.TrimSpace(p.ClaimNumber) == ""
}

// Accepted menyatakan proteksi ini sudah diakseptasi, disetujui maupun ditolak.
func (p Protection) Accepted() bool {
	return strings.TrimSpace(p.AcceptStatus) != AcceptPending
}

// ── Penomoran ────────────────────────────────────────────────────────────────────

// NumberPrefix adalah penanda proteksi yang diterbitkan aplikasi ini.
//
// Nomor warisan Pega berbentuk `OPC-XXX`; prefix di sini sengaja DIBEDAKAN.
//
// # Kenapa dibedakan, dan atas dasar apa
//
// Keputusan Work Owner 2026-09-23, mengikuti alasan yang sama yang melahirkan `D-22` bagi
// nomor klaim (`docs/Steering/00-DECISION-LOG.md:528`): selama masa paralel Pega JUGA masih
// menerbitkan proteksi, dan pencacah `OPC-` miliknya ada di tabel engine yang tidak ikut
// diekspor — sehingga tidak dapat diikuti.
//
// Dua sistem yang menerbitkan `OPC-` dengan pencacah masing-masing akan menerbitkan nomor
// yang sama. Prefix berbeda menghapus kemungkinan itu sepenuhnya, sekaligus membuat asal
// sebuah baris terbaca langsung dari nomornya tanpa tabel pemetaan.
const NumberPrefix = "OPCN"

// FormatNumber menyusun nomor proteksi berbentuk `OPCN.YY.xxxx`.
//
// # Dua hal yang mengikuti nomor laporan, bukan mengikuti D-71 apa adanya
//
//  1. Nomor urut DIPADATKAN NOL sampai empat digit. `D-71` butir 2 mencatat bahwa
//     `TO_CHAR(seq.NEXTVAL)` tanpa format mask membuat lebar segmen terakhir berubah-ubah,
//     sehingga ".10" mendahului ".9" saat diurutkan sebagai teks. Cacat itu diketahui
//     sebelum satu pun nomor terbit, dan memperbaikinya sekarang tidak memutus data
//     historis apa pun.
//
//  2. Tahunnya datang dari PEMANGGIL, bukan dari `SYSDATE` basis data. `D-71` catatan
//     ketiga mencatat bahwa nomor yang terbit di sekitar pergantian tahun akan mengambil
//     tahun dari jam server basis data — bertaut dengan `R-12`. Di sini sumber waktunya
//     satu dan dapat diuji.
//
// Keduanya menyalin perlakuan `inboxlaporanklaim.FormatReportNumber` atas `RCVN.YY.xxxx`.
// Perlu dicatat derajatnya: bentuk `RCVN` itu tiruan tim pengembang atas `D-71`, BUKAN
// keputusan Work Owner — sedangkan `OPCN` di sini memang keputusan Work Owner.
//
// Nomor urut yang melewati empat digit TIDAK dipotong; ia tumbuh menjadi lima digit.
// Memotongnya akan menerbitkan nomor ganda, dan nomor ganda jauh lebih mahal daripada
// kolom yang melebar.
func FormatNumber(year int, sequence int64) string {
	return fmt.Sprintf("%s.%02d.%04d", NumberPrefix, year%100, sequence)
}

// IssuedHere menyatakan sebuah nomor diterbitkan aplikasi ini, bukan Pega.
//
// Pemeriksaannya dilakukan atas NOMOR, bukan atas kolom penanda, sehingga asal sebuah
// baris tetap terbaca meski barisnya disalin ke tempat lain — ke berkas ekspor, ke lampiran
// uji kesetaraan, atau ke surat.
func IssuedHere(number string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(number)), NumberPrefix+".")
}

// ── Pembacaan daftar ─────────────────────────────────────────────────────────────

// Filter adalah penyaring dan paginasi layar daftar.
//
// # Yang TIDAK ada di sini, dan itu disengaja
//
// Tidak ada penyaring status akseptasi. Layar ini menurut
// `Report Definition/InboxReqOpenProtection_RD-RD.xml` menampilkan HANYA proteksi yang
// belum diakseptasi — `.AcceptStatus IS NULL` adalah satu-satunya penyaringnya, dan ia
// bukan pilihan pengguna melainkan definisi layarnya. Menjadikannya parameter akan
// membuka jalan bagi pemanggil untuk melihat proteksi yang sudah selesai di layar yang
// bukan tempatnya.
type Filter struct {
	// Search mencari pada No Proteksi, No Polis, dan No Klaim sekaligus.
	//
	// Layar lama TIDAK punya kotak pencarian — sectionnya hanya memuat grid dan satu
	// tombol. Pencarian ditambahkan di sini karena penyaringnya hanya "belum diakseptasi",
	// sehingga daftarnya tumbuh tanpa batas seiring waktu dan tanpa pencarian akan menjadi
	// tidak terpakai. Ini PENAMBAHAN yang disadari, bukan replikasi; dicatat di
	// `docs/keputusan-implementasi.md`.
	Search string

	Limit  int
	Offset int
}

// Batas paginasi.
//
// DefaultLimit mengikuti `pyPageSize` layar lama, yang bernilai 50
// (`Report Definition/InboxReqOpenProtection_RD-RD.xml`). MaxLimit mengikuti
// `10-API-STRATEGY.md` §4, yang menetapkan 100 sebagai batas dan menolak permintaan yang
// lebih besar.
const (
	DefaultLimit = 50
	MaxLimit     = 100
)

// Normalize mengembalikan filter dengan nilai yang dijamin masuk akal.
func (f Filter) Normalize() Filter {
	f.Search = strings.TrimSpace(f.Search)

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
// Dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang memenuhinya.
// Diisi `repo/sqlstore` terhadap Oracle dan `repo/memory` untuk pengujian.
//
// # Kepemilikan tulis
//
// Modul ini menulis kolom PEMBUATAN: nomor, polis, klaim, tipe, keterangan, dan detail
// perubahan. Kolom AKSEPTASI — `STATUS_AKSEPTASI`, `TANGGAL_AKSEPTASI`, `DIAKSEP_OLEH` —
// TIDAK disentuh di sini; keduanya milik modul `inboxacceptopenprotection`.
//
// Pembagian itu yang menjaga `P-1` tetap berlaku meski dua modul menyentuh satu tabel:
// keduanya menulis kolom yang berbeda pada tahap hidup yang berbeda.
type Repo interface {
	// List membaca satu halaman proteksi yang belum diakseptasi.
	List(ctx context.Context, f Filter) (Page, error)

	// Get membaca satu proteksi menurut nomornya.
	//
	// Mengembalikan ErrNotFound bila tidak ada — bukan nilai kosong. Nilai kosong akan
	// membuat pemanggil menampilkan form berisi field kosong seolah proteksinya ada.
	Get(ctx context.Context, number string) (Protection, error)

	// HasDuplicate menyatakan sudah ada proteksi dengan polis dan tipe yang sama pada hari
	// yang sama.
	//
	// Ia di seam, bukan dihitung di memori setelah membaca daftar: menyaring di memori
	// menuntut seluruh proteksi hari itu ditarik lebih dulu, dan jawabannya tetap bisa
	// salah karena baris dapat lahir di antara pembacaan dan penyimpanan.
	//
	// Lihat DuplicateKey untuk asal aturannya.
	HasDuplicate(ctx context.Context, key DuplicateKey, exceptNumber string) (bool, error)

	// Create menyimpan proteksi baru dan mengembalikannya beserta nomor yang terbit.
	//
	// Penerbitan nomor terjadi DI DALAM adapter, di dalam transaksi yang sama dengan
	// penyisipan barisnya. Memisahkannya berarti nomor dapat terbit lalu barisnya gagal
	// disimpan, meninggalkan lubang pada deret — dan pada deret yang dibaca manusia,
	// lubang akan terus ditanyakan.
	Create(ctx context.Context, draft Draft, by string, at time.Time) (Protection, error)

	// Update menyunting proteksi yang belum tertaut klaim.
	//
	// Adapter WAJIB menolak penyuntingan proteksi yang sudah tertaut maupun sudah
	// diakseptasi, meski pemanggil sudah memeriksanya lebih dulu. Pemeriksaan di lapisan
	// atas menjaga pengguna dari kesalahan; pemeriksaan di penyimpanan menjaga data dari
	// dua permintaan yang tiba bersamaan.
	Update(ctx context.Context, number string, draft Draft, by string, at time.Time) (Protection, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal
//
// `ADR-0030` menetapkan satu basis data per entitas, bukan satu basis data bersama dengan
// penanda entitas. Proteksi milik Asuransi Sinar Mas dan proteksi milik Simas Insurtech
// karena itu tidak pernah berada di tabel yang sama.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal menghasilkan galat — TIDAK PERNAH dialihkan ke koneksi utama.
// Jatuh ke koneksi default berarti menampilkan proteksi satu badan hukum di layar badan
// hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
type RepoSelector func(portalAlias string) (Repo, error)
