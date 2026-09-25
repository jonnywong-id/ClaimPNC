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

// Tiga nilai `PROTECTION_TYPE_ID` yang MENGUBAH PERILAKU.
//
// # Bedakan dari label
//
// Sejak master `POOLDATA.M_CLAIM_PROTECTION_TYPE` diterima (Work Owner, 2026-09-24), NAMA
// setiap tipe datang dari data — kesembilannya, termasuk yang dulu tidak diketahui.
// Ketiga konstanta di bawah TETAP di kode karena ia bukan label melainkan PERCABANGAN:
//
//	'2'  memisahkan antrean akseptasi menjadi PREMI dan NON PREMI
//	     (`InboxOpenProtection2_RD_collection` menyaring `= "2"`, dan
//	      `InboxOpenProtection2_RD` menyaring `!= "2"` untuk peran lain)
//	'7'  memunculkan panel "Detail Perubahan DOL"
//	     (`Section/InputProtectionSection-Section.xml:2728`)
//	'8'  memunculkan panel "Detail Perubahan Cause Of Loss" (`:3638`)
//
// Mengganti NAMA sebuah tipe di master tidak boleh mengubah satu pun dari ketiganya, dan
// tidak akan — yang dibandingkan kodenya, bukan namanya.
//
// # Kenapa ketiganya tidak ikut dipindah ke master
//
// Master hanya memuat `PROTECTION_TYPE_ID` dan `PROTECTION_TYPE_NAME`; tidak ada kolom yang
// menyatakan "tipe ini masuk antrean premi" atau "tipe ini menuntut detail perubahan".
// Menurunkan perilaku dari nama akan membuat satu suntingan ejaan mengubah antrean
// akseptasi — tanpa galat dan tanpa gejala.
const (
	// TypePremium adalah proteksi klaim PREMI, yang diakseptasi peran penagihan premi.
	// Master menamainya "Premi Belum Lunas".
	TypePremium = "2"

	// TypeChangeLossDate adalah permintaan perubahan Tanggal Kejadian (DOL).
	TypeChangeLossDate = "7"

	// TypeChangeCauseOfLoss adalah permintaan perubahan Penyebab Kerugian.
	TypeChangeCauseOfLoss = "8"
)

// ProtectionType adalah satu baris master `POOLDATA.M_CLAIM_PROTECTION_TYPE`.
//
// Dipakai mengisi pilihan tipe pada form dan menampilkan namanya di daftar. Sengaja hanya
// dua field: itulah seluruh isi tabelnya.
type ProtectionType struct {
	// ID adalah `PROTECTION_TYPE_ID`, `VARCHAR2(2)`. Nilai yang ada hari ini '1'…'9'.
	ID string

	// Name adalah `PROTECTION_TYPE_NAME`, yang dilihat pengguna.
	Name string
}

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
	// Ia SEKALIGUS kunci baris: kolom `OPEN_PROTECTION_ID` pada
	// `POOLDATA.T_CLAIM_OPENPROTECTION` bertipe `VARCHAR2(20)` dan berisi nomor ini apa
	// adanya. Tidak ada kunci teknis terpisah.
	//
	// Dua bentuk hidup berdampingan permanen:
	//
	//	OPC-XXX         warisan Pega, dibaca apa adanya
	//	OPCN.YY.xxxx    terbitan aplikasi ini (keputusan Work Owner 2026-09-23)
	//
	// Lihat FormatNumber untuk alasan prefixnya dibedakan.
	Number string

	// PolicyNumber — kolom **"No Polis"**, `.PolicyNo`. Wajib diisi. `VARCHAR2(20)`.
	PolicyNumber string

	// ClaimNumber — kolom **"No Klaim"**, `.CaseID`. Kolom `CLAIM_NO`.
	//
	// KOSONG selama proteksi belum ditautkan ke klaim, dan justru saat kosong itulah
	// pemohon masih boleh menyuntingnya. Lihat Editable.
	//
	// # Bentuknya
	//
	//	PNC-xxxx        warisan Pega — 118 dari 160 baris produksi tepat delapan huruf
	//	PNCN.YY.xxxx    sistem baru (`D-71`) — dua belas huruf
	//
	// # Kolomnya terlalu sempit, dan ini BELUM tertutup
	//
	// `CLAIM_NO` bertipe `VARCHAR2(10)`, sedangkan `PNCN.YY.xxxx` butuh dua belas. Yang
	// muat hanya sampai klaim ke-99.
	//
	// Di produksi pun ada dua baris ber-`CASEID` sebelas dan enam belas huruf; keduanya
	// TIDAK berpola `PNC-` sama sekali — kolomnya teks bebas di Pega, sehingga salah ketik
	// ikut tersimpan.
	//
	// `MaxClaimNumberLength` karena itu ditetapkan menurut lebar SASARAN, bukan lebar hari
	// ini. Sampai kolomnya dilebarkan, penyimpanan nomor klaim baru gagal `ORA-12899` —
	// dan itu memang yang seharusnya terlihat: memvalidasi pada sepuluh akan menolak
	// seluruh nomor klaim sistem baru dengan pesan yang seolah-olah menyalahkan pengguna.
	ClaimNumber string

	// ClaimReference adalah **ClaimID**, `.PNCCaseID`. Kolom `ID_CLAIM`, `VARCHAR2(100)`.
	//
	// # Pada baris BARU ia sama dengan ClaimNumber
	//
	// Work Owner menegaskan pada 2026-09-24 bahwa ClaimNo dan ClaimID kini berisi nilai
	// yang sama, yaitu `PNCN.YY.xxxx`. Adapter karena itu MENURUNKANNYA dari ClaimNumber;
	// form tidak menanyakannya kedua kalinya.
	//
	// # Pada baris WARISAN ia BERBEDA, dan karena itu field ini tetap ada
	//
	// Data Pega menyimpan kunci teknisnya di sini — `ASM-FW-GCNMFW-WORK PNC-xxxx`, yaitu
	// nama kelas work pool ditambah nomor klaim (102 dari 103 baris terisi berpola itu).
	// `D-22` membuang prefix tersebut dari data baru.
	//
	// Menghapus field ini dan menampilkan ClaimNumber sebagai gantinya akan membuat baris
	// warisan TAMPAK seolah ClaimID-nya sama dengan nomor klaimnya — padahal tidak.
	ClaimReference string

	// Type — kolom **"Tipe Proteksi"**, `.TypeProtection`. Kolom `PROTECTION_TYPE_ID`,
	// `VARCHAR2(2)`. Lihat konstanta di atas.
	Type string

	// TypeName adalah nama tipe dari master, yang DILIHAT pengguna.
	//
	// Ia hasil pembacaan `POOLDATA.M_CLAIM_PROTECTION_TYPE`, bukan kolom di tabel proteksi
	// — sehingga mengganti nama sebuah tipe langsung berlaku pada seluruh baris lama tanpa
	// satu pun pembaruan data.
	//
	// KOSONG bila kodenya tidak ada di master. Layar menampilkan kodenya apa adanya dalam
	// keadaan itu, bukan tanda hubung: kode yang tak dikenal adalah data yang perlu
	// ditanyakan, dan menyembunyikannya membuatnya tidak pernah ditanyakan.
	TypeName string

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
// # Pencacahnya sequence, bukan MAX+1
//
// Work Owner menetapkan `POOLDATA.CLAIM_PROTECTION_SEQ` pada 2026-09-24. Adapter membaca
// `NEXTVAL`-nya, lalu fungsi ini menyusun bentuk teksnya.
//
// Sequence itu GLOBAL — tidak direset tiap awal tahun — sehingga nomor urut menembus
// pergantian tahun (`OPCN.26.0009` diikuti `OPCN.27.0010`). Segmen tahun karena itu
// PENANDA, bukan penghitung per tahun. Konsekuensinya diterima: yang harus unik adalah
// nomornya, dan sequence menjaminnya tanpa bergantung pada isi tabel.
//
// # Dua hal yang BERBEDA dari sintaks yang Work Owner tuliskan
//
// Sintaks yang diberikan berbunyi
// `'OPCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(seq.NEXTVAL)`. Yang dipakai di
// sini berbeda pada dua hal, keduanya disengaja dan keduanya dilaporkan:
//
//  1. Nomor urut DIPADATKAN NOL sampai empat digit. `TO_CHAR(seq.NEXTVAL)` tanpa format
//     mask membuat lebar segmen terakhir berubah-ubah, sehingga ".10" mendahului ".9" saat
//     diurutkan sebagai teks — dan daftar proteksi memang diurutkan sebagai teks pada
//     pemutus serinya. `D-71` butir 2 mencatat cacat yang sama untuk nomor klaim. Ia
//     diketahui sebelum satu pun nomor terbit, sehingga memperbaikinya tidak memutus data
//     historis apa pun.
//
//  2. Tahunnya datang dari PEMANGGIL, bukan dari `SYSDATE`. `SYSDATE` adalah jam server
//     basis data; nomor yang terbit di sekitar pergantian tahun akan mengambil tahun dari
//     sana dan bergeser terhadap tanggal WIB (`R-12`, `D-71` catatan ketiga). Di sini
//     sumber waktunya satu dan dapat diuji.
//
// Keduanya menyalin perlakuan `inboxlaporanklaim.FormatReportNumber` atas `RCVN.YY.xxxx`.
// Perlu dicatat derajatnya: bentuk `RCVN` itu tiruan tim pengembang atas `D-71`, BUKAN
// keputusan Work Owner — sedangkan `OPCN` di sini memang keputusan Work Owner.
//
// # Batas kolomnya
//
// `OPEN_PROTECTION_ID` bertipe `VARCHAR2(20)`, sedangkan delapan huruf terpakai awalan dan
// pemisah. Tersisa dua belas digit untuk nomor urut, sementara `MAXVALUE` sequence memuat
// lima belas. Selisih itu baru menggigit pada nomor urut ke-1.000.000.000.000 dan dicatat
// di `docs/kolom-open-protection.md`, bukan dijaga di kode.
//
// Nomor urut yang melewati empat digit TIDAK dipotong; ia tumbuh menjadi lima digit.
// Memotongnya akan menerbitkan nomor ganda, dan nomor ganda jauh lebih mahal daripada
// segmen yang melebar.
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
	//
	// claim adalah klaim yang SUDAH DITEMUKAN pemanggil. Adapter menurunkan darinya nomor
	// polis, nama objek, nama cabang, dan kedua nilai "sebelum" pada panel detail perubahan —
	// nilai-nilai yang di Pega disalin `Activity/OpenProtection-Act.xml`, bukan diketik.
	Create(ctx context.Context, draft Draft, claim Claim, by string, at time.Time) (Protection, error)

	// Update menyunting proteksi yang belum tertaut klaim.
	//
	// Adapter WAJIB menolak penyuntingan proteksi yang sudah tertaut maupun sudah
	// diakseptasi, meski pemanggil sudah memeriksanya lebih dulu. Pemeriksaan di lapisan
	// atas menjaga pengguna dari kesalahan; pemeriksaan di penyimpanan menjaga data dari
	// dua permintaan yang tiba bersamaan.
	Update(ctx context.Context, number string, draft Draft, claim Claim, by string, at time.Time) (Protection, error)
}

// TypeRepo adalah seam ke master tipe proteksi.
//
// # Kenapa seam TERSENDIRI, bukan method tambahan pada Repo
//
// Ia membaca TABEL LAIN (`POOLDATA.M_CLAIM_PROTECTION_TYPE`), isinya berubah dengan irama
// yang sama sekali berbeda, dan ia MURNI BACA — tidak ada satu pun jalur di modul ini yang
// menulis master. Menggabungkannya ke Repo akan membuat setiap fake pengujian proteksi
// terpaksa ikut memalsukan master, padahal sebagian besar pengujian tidak memedulikannya.
//
// Modul `inboxacceptopenprotection` mendeklarasikan seam-nya SENDIRI atas tabel yang sama.
// Itu disengaja: keduanya paket domain yang tidak boleh saling mengimpor, dan yang dibagi
// hanyalah tabelnya — bukan tipenya.
type TypeRepo interface {
	// ListTypes membaca seluruh tipe proteksi, terurut menurut kodenya.
	//
	// Tidak ada penyaring aktif/nonaktif: masternya hanya punya dua kolom, dan menambahkan
	// penyaring yang tidak punya kolom berarti mengarang.
	ListTypes(ctx context.Context) ([]ProtectionType, error)
}

// RepoSelector memilih Repo dan TypeRepo milik satu portal entitas.
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
//
// Ketiganya dikembalikan BERSAMAAN karena ketiganya berasal dari koneksi yang sama.
// Memilihnya lewat pemanggilan terpisah membuka kemungkinan proteksi dibaca dari portal yang
// satu dan klaimnya dari portal yang lain — kelas cacat yang tidak menghasilkan galat apa
// pun, hanya data milik badan hukum yang keliru.
type RepoSelector func(portalAlias string) (Stores, error)

// Stores mengumpulkan seluruh seam penyimpanan milik SATU portal.
//
// Dibungkus struct, bukan dikembalikan sebagai beberapa nilai berjejer, karena jumlahnya
// sudah tiga dan masih mungkin bertambah. Setiap penambahan pada bentuk berjejer mengubah
// tanda tangan fungsi dan menyentuh setiap pemanggil beserta setiap uji — biaya yang tidak
// dibayar oleh manfaat apa pun.
type Stores struct {
	// Protections menyimpan permintaan proteksi. Wajib.
	Protections Repo

	// Types membaca master tipe proteksi. Wajib.
	Types TypeRepo

	// Claims mencari klaim yang ditaut. Wajib.
	Claims ClaimRepo
}

// ── Pencarian klaim ──────────────────────────────────────────────────────────────

// Claim adalah data klaim yang DITURUNKAN ke form, bukan diketik pengguna.
//
// # Kenapa modul ini membaca data klaim sama sekali
//
// `Activity/OpenProtection-Act.xml` — yang di Pega dipicu field **No Klaim** — memuat
// klaimnya lewat Report Definition `BrowseCaseList`, lalu MENYALIN KELUAR nilai-nilai di
// bawah ke halaman proteksi. Jadi di sistem lama pun nilai-nilai ini tidak pernah diketik:
// ia ditimpa setiap kali klaim dicari.
//
// Implementasi pertama modul ini keliru menjadikannya isian bebas. Akibatnya bukan sekadar
// merepotkan: seseorang dapat menyimpan permintaan "ubah DOL" yang menyebut DOL SEBELUM
// yang tidak pernah menjadi DOL klaim itu — dan petugas akseptasi menyetujuinya tanpa cara
// mengetahuinya.
//
// # Sumbernya tabel milik Pega, dan itu HANYA DIBACA
//
// `P-1` melarang dua sistem MENULIS satu tabel; membaca tidak dilarang. Tidak ada satu pun
// pernyataan tulis ke tabel klaim di modul ini.
type Claim struct {
	// Number adalah nomor klaim itu sendiri, sebagaimana ditemukan.
	Number string

	// PolicyNumber — `.PolicyNo`, disalin `OpenProtection` ke `pyWorkPage.PolicyNo`.
	PolicyNumber string

	// InsuredName — `.Policy.QQName`, "Nama Tertanggung" pada form.
	InsuredName string

	// LossDate adalah DOL klaim, yang menjadi **Current Date Of Loss** pada panel tipe '7'.
	//
	// Pointer supaya "klaim tidak punya DOL" dapat dibedakan dari "1 Januari tahun 1".
	LossDate *time.Time

	// CauseOfLoss adalah penyebab kerugian klaim, yang menjadi **Cause Of Loss Dipilih**
	// pada panel tipe '8'.
	CauseOfLoss string

	// ObjectName dan BranchName melengkapi kedua panel.
	ObjectName string
	BranchName string
}

// ClaimRepo adalah seam ke pencarian klaim.
//
// Dideklarasikan di sini, di paket yang memakainya. Ia MURNI BACA — tidak ada satu pun
// method yang menulis, dan itu bukan kebetulan melainkan batas yang dijaga `P-1`.
type ClaimRepo interface {
	// FindClaim mencari klaim menurut nomornya.
	//
	// Mengembalikan ErrClaimNotFound bila tidak ada — bukan Claim kosong. Claim kosong akan
	// membuat form terisi nilai kosong seolah klaimnya ditemukan tanpa data, dan permintaan
	// tersimpan menunjuk klaim yang tidak pernah ada.
	FindClaim(ctx context.Context, number string) (Claim, error)
}
