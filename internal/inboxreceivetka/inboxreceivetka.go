// Package inboxreceivetka adalah inti modul Inbox Receive TKA.
//
// # Layar apa ini
//
// Menu `MENU_ID 49` "Inbox Receive TKA" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk
// harness `InboxTKA_Harness` dan berinduk `MENU_ID 2` (kelompok INBOX). Judul yang dibaca
// pengguna terbaca dua kali dan sama: MENU_DESC pada master menu, dan caption berformat
// `Heading 1` pada `Section/InboxTKA_Section-Section.xml`.
//
// Isinya **klaim TKA yang tanggal kelengkapan dokumennya belum diisi**. Pengguna mengisi
// tanggal itu di dalam grid lalu menekan Submit; begitu terisi, barisnya HILANG dari daftar.
//
// Ia INBOX menurut keempat ciri `D-79`: barisnya pekerjaan, baris hilang setelah dikerjakan,
// "hanya yang jadi tanggung jawab saya" adalah aturan kewenangan, dan barisnya punya tenggat
// — kolom Aging justru mengukur berapa lama ia sudah menunggu.
//
// # Ia BUKAN kembaran Inbox Investigator, dan tiga hal membedakannya
//
//  1. **Tanpa workbasket.** Report Definition-nya mendeklarasikan `newAssignPage`
//     (`Assign-WorkBasket`) di PagesAndClasses, tetapi TIDAK PERNAH merujuknya — nol
//     `newAssignPage.*`, nol `pxAssignedOperatorID`. Ia sisa Save-As. Yang menentukan
//     keanggotaan daftar ini adalah penanda TKA, bukan antrean.
//
//  2. **Modul ini MENULIS.** Lihat Repo.Complete.
//
//  3. **Sumbernya tabel kerja Pega yang SAMA dengan Report Definition-nya** —
//     `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega dan diverifikasi ke basis data produksi, bukan
// dikarang:
//
//	Harness/InboxTKA_Harness-Harness.xml          pembungkus layar
//	Section/InboxTKA_Section-Section.xml          judul, 7 kolom, pyPageSize=50, tombol Submit
//	Report Definition/InboxTKA_RD-RD.xml          3 penyaring, urutan, pyMaxRecords=500
//	Activity/SubmitTanggalLengkapTKA-Act.xml      11 langkah alur simpan; pesan galat
//	Activity/SendEmailNotification_act.xml        pengirim surel bawaan Pega
//	HTML/NotificationKelengkapanTKA-HTML.xml      badan surel beserta keenam labelnya
//	Database/m_menu_aplikasi_pnc.csv              MENU_ID 49 "Inbox Receive TKA"
//	katalog DATAPEGA.PC_ASM_FW_GCNMFW_WORK        nama dan tipe kolom, 2026-09-24
//
// # Ketiga penyaring layar lama, dan kolom yang mewakilinya
//
// Report Definition menyatakan tiga penyaring dengan logika `A AND B AND C`:
//
//	A  .ClaimData.TKA               =  "1"                  -> TKA_1 = '1'
//	B  .ClaimData.TanggalDokLengkap IS NULL                 -> TANGGALDOKLENGKAP IS NULL
//	C  .pyStatusWork                != "Resolved-Completed" -> lihat ResolvedWorkStatus
//
// # Penyaring A sempat dinyatakan buntu, dan itu KELIRU
//
// Catatan terdahulu menyimpulkan `.ClaimData.TKA` tidak dapat dipakai karena nol kolom
// fisik di export, sehingga sumbernya dialihkan ke sebuah tabel turunan. Kesimpulan itu
// salah, dan kesalahannya ada dua:
//
//  1. Export memuat RULE, bukan skema basis data. Kolom yang hanya dipakai Report
//     Definition tidak akan pernah muncul di sana, karena SQL-nya dibangun Pega saat
//     dijalankan.
//  2. Report Definition menjalankan SQL terhadap kolom. Sebuah properti hanya dapat menjadi
//     PENYARING bila ia di-expose — sehingga kenyataan bahwa RD ini berjalan sudah
//     membuktikan kolomnya ada.
//
// Diverifikasi ke katalog: kolomnya **`TKA_1`**, `VARCHAR2(1)`. Terhadap dua klaim yang
// benar-benar tampil di layar Pega, keduanya bernilai `'1'`, dan cacah dengan ketiga
// penyaring di atas menghasilkan **tepat 2** — sama dengan jumlah baris pada layar itu.
//
// Akibatnya modul ini membaca tabel, kolom, dan penyaring yang SAMA PERSIS dengan Pega.
// Tidak ada penanda yang ditebak, dan tidak ada tabel turunan yang perlu diisi lebih dulu.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxreceivetka/              aturan modul + seam          ← paket ini
//	inboxreceivetka/usecase/      orkestrasi: buka daftar, simpan tanggal
//	inboxreceivetka/repo/         pengisi seam penyimpanan     — sqlstore, memory
//	inboxreceivetka/notification/ pengisi seam Notifier        — smtp, fake
//	inboxreceivetka/http/         lapisan transport modul ini  — handler, dto, rute
package inboxreceivetka

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ResolvedWorkStatus adalah nilai `pyStatusWork` yang menandai pekerjaan sudah tuntas.
//
// Report Definition menyaring `.pyStatusWork != "Resolved-Completed"` (penyaring C).
//
// # Ia dipakai lewat gabungan, dan gabungan itu TIDAK BOLEH menyembunyikan baris
//
// `T_CLAIM_TKA_H` tidak punya kolom kunci apa pun — tanpa `CLAIMID`, tanpa `PZINSKEY`,
// tanpa primary key. Status pekerjaan karena itu hanya dapat dicapai lewat dua lompatan:
// `NO_KLAIM` → `T_CLAIM_PNC.CLAIMNO` → `CLAIMID` = `PZINSKEY`.
//
// Gabungannya **LEFT**, bukan INNER, dan itu keputusan yang disengaja: baris TKA yang
// klaimnya tidak ditemukan tetap TAMPIL. Gabungan INNER akan membuangnya diam-diam, dan
// pekerjaan yang hilang tanpa jejak jauh lebih mahal daripada pekerjaan yang tampil
// berlebih — yang pertama tidak ada yang menyadarinya, yang kedua langsung terlihat.
//
// Perhatikan pula bahwa penyaringnya "bukan selesai", bukan "sedang berjalan". Klaim
// berstatus `Resolved-Rejected` KARENA ITU TETAP TAMPIL. Itu perilaku sistem lama dan
// direplikasi apa adanya (`P-5`).
const ResolvedWorkStatus = "Resolved-Completed"

// MaxRows membatasi jumlah baris yang dikembalikan satu permintaan.
//
// Angkanya diwarisi dari `pyMaxRecords = 500` pada `InboxTKA_RD`, tetapi **artinya
// berubah**. Di sistem lama batas itu memotong tanpa memberi tahu siapa pun: baris ke-501
// tidak pernah tampil, dan tidak ada apa pun di layar yang menyatakannya. Di sini
// pemotongannya DINYATAKAN lewat Page.Truncated, dan layar menyebutkannya kepada pengguna.
//
// Batas yang diketahui adalah batas; batas yang senyap adalah data yang hilang.
const MaxRows = 500

// Task adalah satu baris inbox — satu klaim TKA yang menunggu tanggal kelengkapan dokumen.
//
// # Ketujuh kolom grid layar lama, pada urutannya
//
// Dibaca dari `Section/InboxTKA_Section-Section.xml` dengan memasangkan caption dan sel
// satu lawan satu menurut urutan kemunculannya di dalam berkas:
//
//	Nomor Klaim             <-> .ClaimData.ClaimNo
//	No Polis                <-> .Policy.PolicyNo
//	Nama Tertanggung        <-> .Policy.QQName
//	Nama Peserta            <-> .Policy.TheInsured
//	Date Of Loss            <-> .ClaimData.DateOfLoss
//	Aging                   <-> .ClaimData.RegisterDate
//	Tanggal Dokumen Lengkap <-> .ClaimData.TanggalDokLengkap   (isian, bukan tampilan)
//
// # Pemasangan ketiga dan keempat sempat diragukan, dan kini terbukti
//
// `InboxTKA_RD` melabeli kedua properti itu TERBALIK — `.Policy.QQName` diberi label
// "Nama Peserta" dan `.Policy.TheInsured` diberi label "Nama Tertanggung". Yang benar
// adalah Section, dan buktinya datang dari sumber KETIGA yang berdiri sendiri:
// `HTML/NotificationKelengkapanTKA-HTML.xml` menuliskan label "Nama Tertanggung" pada
// `TempHTML.District`, dan `Activity/SubmitTanggalLengkapTKA-Act.xml` mengisi
// `TempHTML.District` dari `Param.QQName`. Label "Nama Peserta" berpasangan dengan
// `TempHTML.DistrictID`, yang diisi dari `Param.Insured`.
//
// Label pada Report Definition karena itu tidak dibawa. Nama properti `District` dan
// `DistrictID` untuk menampung nama orang adalah bentuk yang sama dengan alias menyesatkan
// pada `03-CURRENT-ARCHITECTURE.md` §4.2, dan juga tidak dibawa.
type Task struct {
	// Reference adalah `pzInsKey` — kunci teknis Pega, dan **kunci baris** modul ini.
	//
	// Ia selalu terisi: sumbernya `PC_ASM_FW_GCNMFW_WORK`, tabel yang menggerakkan
	// kuerinya, bukan hasil gabungan.
	//
	// Ia dibawa untuk penelusuran, BUKAN untuk ditampilkan. `D-22` menetapkan klaim
	// terbitan sistem baru tidak pernah menulis awalan `ASM-FW-GCNMFW-WORK` lagi.
	Reference string

	// ClaimKey adalah `CLAIMID` pada `POOLDATA.T_CLAIM_PNC`, dan ia **dapat kosong**.
	//
	// Nilainya sama dengan Reference ketika klaimnya ditemukan. Kosong berarti pekerjaan
	// ini ADA di tabel kerja Pega tetapi tidak punya pasangan di tabel klaim bisnis.
	//
	// # Kosong bukan galat, melainkan keterangan yang harus sampai ke layar
	//
	// Baris seperti itu TETAP TAMPIL — gabungannya sengaja LEFT, sama seperti Pega yang
	// juga menampilkannya. Yang tidak dapat dilakukan hanyalah mengisinya: tidak ada baris
	// klaim yang dapat diperbarui, sehingga Complete menolaknya dengan ErrClaimMissing.
	//
	// Layar memakai medan ini untuk mematikan isian pada baris itu lebih dulu, alih-alih
	// membiarkan pengguna menekan tombol yang sudah pasti gagal.
	ClaimKey string

	// ClaimNumber adalah kolom "Nomor Klaim" — `PYID`.
	//
	// Report Definition menampilkan `.ClaimData.ClaimNo`, dan properti itu TIDAK di-expose
	// sebagai kolom. Penggantinya bukan pendekatan: layar Pega menampilkan `PNC-1546` dan
	// `PNC-1729` pada kolom itu, yaitu nilai `PYID` keduanya.
	ClaimNumber string

	// PolicyNumber adalah kolom "No Polis" — `POLICYNO`.
	PolicyNumber string

	// InsuredName adalah kolom "Nama Tertanggung" — `QQNAME`.
	InsuredName string

	// ParticipantName adalah kolom "Nama Peserta" — `POOLDATA.T_GENERAL.THEINSURED`.
	//
	// `.Policy.TheInsured` tidak di-expose sebagai kolom pada tabel kerja Pega; kolom
	// `INSUREDNAME` yang ada di sana terbukti KOSONG pada kedua baris uji. Nilainya karena
	// itu diambil dari tabel polis, dijembatani `T_CLAIM_PNC` lewat `NOPOLIS` + `PRODKE`.
	//
	// Terverifikasi terhadap `PNC-1546`: `T_GENERAL.THEINSURED` berisi nama yang sama
	// persis dengan yang ditampilkan Pega pada kolom itu.
	//
	// Dapat kosong bila polisnya tidak ditemukan; layar menuliskannya sebagai tanda hubung.
	ParticipantName string

	// DateOfLoss adalah kolom "Date Of Loss" — `DATEOFLOSS_1`.
	//
	// Nil bila kolomnya kosong. Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak
	// dapat dibedakan dari "belum diisi" saat ditampilkan, dan layar akan menuliskan
	// "01/01/0001" alih-alih tanda hubung.
	DateOfLoss *time.Time

	// RegisteredOn mengisi kolom yang di layar lama bercaption **"Aging"** —
	// `.ClaimData.RegisterDate`, kolom `REGISTERDATE_1`.
	//
	// # Captionnya menyebut lama menunggu, isinya TANGGAL
	//
	// Bukan salah baca, dan kali ini terbukti dari data: `PNC-1729` ber-`REGISTERDATE_1`
	// `20240319`, dan layar Pega menuliskan **"2 years 6 months ago"** — tepat selisihnya
	// terhadap hari pembacaan. `PNC-1546` ber-`20230510` menghasilkan "3 years ago".
	//
	// Jadi Pega menyimpan TANGGAL dan merendernya sebagai waktu relatif saat ditampilkan.
	// Itu yang ditiru: server mengirim tanggalnya, layar yang menyusun kalimatnya.
	//
	// # Kenapa perhitungannya di layar, bukan di sini
	//
	// "Berapa lama menunggu" bergantung pada KAPAN ia dibaca. Menghitungnya di server
	// berarti nilainya membeku pada saat permintaan, dan modul ini akan membutuhkan seam
	// Clock hanya demi satu label tampilan. Tanggalnya sendiri tidak bergantung pada jam
	// dinding, dan itulah yang dikirim.
	//
	// Nama medan mengikuti ISI, bukan caption — supaya pembaca kode berikutnya tidak
	// menduga ada durasi yang tersimpan di sini. Yang mengikuti Pega adalah judul kolom di
	// layar (`D-13`).
	//
	// Kolomnya `VARCHAR2(32)` berisi teks berformat `yyyymmdd`; penguraiannya di adapter.
	// Nil bila kosong atau bentuknya tidak dikenali.
	RegisteredOn *time.Time
}

// Filter mempersempit daftar yang dibaca layar.
//
// # Yang TIDAK ada di sini, dan itu disengaja
//
// **Tidak ada penyaring penanda TKA.** Ia diwakili keanggotaan tabel; lihat banner paket.
//
// **Tidak ada penyaring tanggal dokumen lengkap.** Nilainya tetap: daftar ini menurut
// definisinya hanya memuat yang BELUM terisi (penyaring B). Menjadikannya pilihan pengguna
// berarti menyediakan cara membaca klaim yang sudah selesai lewat layar inbox.
//
// **Tidak ada penyaring status.** Report Definition menyaring `pyStatusWork` dengan nilai
// tetap, bukan dengan pilihan pengguna; lihat ResolvedWorkStatus.
type Filter struct {
	// Keyword mempersempit daftar pada Nomor Klaim, No Polis, Nama Tertanggung, dan Nama
	// Peserta.
	//
	// DITAMBAHKAN terhadap sistem lama, yang menyaring di peramban lewat kotak isian per
	// kolom pada kepala grid (`pyGridFiltering = true`). Kosong berarti tanpa penyaring.
	//
	// # Layar TIDAK memakainya hari ini
	//
	// Work Owner memilih penyaringan dikerjakan peramban, sehingga layar memakai pencarian
	// bawaan `DataTable` atas baris yang sudah di tangan — sama seperti Inbox Investigator
	// dan seluruh layar master. Penyaring ini tetap disediakan supaya perpindahan ke
	// penyaringan sisi server kelak (`TKT-U2-001`) tidak menuntut perubahan kontrak.
	Keyword string
}

// Clean memangkas spasi di kedua ujung penyaring.
func (f Filter) Clean() Filter {
	return Filter{Keyword: strings.TrimSpace(f.Keyword)}
}

// Page adalah satu halaman inbox beserta keterangan pemotongannya.
//
// Keduanya dikembalikan bersama, bukan lewat dua panggilan terpisah, supaya keterangan
// "terpotong" tidak dapat berasal dari saat yang berbeda dengan barisnya.
type Page struct {
	// Tasks adalah baris yang terkirim, sebanyak-banyaknya MaxRows.
	Tasks []Task

	// Truncated menyatakan masih ada baris yang cocok tetapi TIDAK terkirim.
	Truncated bool
}

// Completion adalah satu pengisian tanggal kelengkapan dokumen.
//
// Ia satuan kerja tombol Submit pada layar lama: satu baris, satu tanggal.
type Completion struct {
	// ClaimNumber menunjuk baris yang diisi — `NO_KLAIM`.
	ClaimNumber string

	// CompletedAt adalah tanggal kelengkapan dokumen yang diketik pengguna.
	//
	// TANGGAL, bukan stempel waktu: kolom penyimpannya `DATE` pada kedua tabel, dan sel
	// isiannya di Pega pun bertipe tanggal. Bagian jam tidak pernah dipakai.
	CompletedAt time.Time
}

// Clean memangkas spasi dan membuang bagian jam dari tanggalnya.
//
// Pemotongan jam dilakukan DI SINI, bukan di dalam SQL: `TRUNC` adalah fungsi Oracle yang
// `D-20` larang dari kueri portabel, dan memotongnya di Go membuat nilainya sama persis di
// kedua basis data.
func (c Completion) Clean() Completion {
	moment := c.CompletedAt
	return Completion{
		ClaimNumber: strings.TrimSpace(c.ClaimNumber),
		CompletedAt: time.Date(
			moment.Year(), moment.Month(), moment.Day(), 0, 0, 0, 0, time.UTC),
	}
}

// Validate menolak pengisian yang tidak lengkap.
//
// # Hanya SATU aturan, dan itu memang seluruh aturan sistem lama
//
// `Activity/SubmitTanggalLengkapTKA-Act.xml` memeriksa tepat dua hal lewat precondition —
// `Param.ClaimNo==""` dan `Param.Tanggal==""` — lalu `Page-Set-Messages` menuliskan pesan
// **"Silahkan isi tanggal terlebih dahulu"**.
//
// Tidak ada pemeriksaan lain di sana: tidak ada batas atas, tidak ada larangan tanggal di
// masa depan, dan tidak ada perbandingan terhadap Date Of Loss. Ketiadaannya direplikasi
// apa adanya (`P-5`) — menambahkan aturan yang tidak pernah ada akan menolak masukan yang
// selama ini sah, dan itu perubahan perilaku yang tidak diminta siapa pun.
func (c Completion) Validate() error {
	if strings.TrimSpace(c.ClaimNumber) == "" {
		return ErrClaimNumberRequired
	}
	if c.CompletedAt.IsZero() {
		return ErrDateRequired
	}
	return nil
}

// Galat domain modul ini.
//
// Keempatnya bertipe tersendiri, bukan teks, supaya transport dapat memetakannya ke kode
// HTTP tanpa mencocokkan kalimat (`08-TECHNICAL-STRATEGY.md` §4.2).
var (
	// ErrClaimNumberRequired: permintaan tidak menyebut baris mana yang diisi.
	ErrClaimNumberRequired = errors.New("inboxreceivetka: nomor klaim wajib disebut")

	// ErrDateRequired: tanggal kelengkapan dokumen kosong.
	//
	// Padanan `Page-Set-Messages` pada langkah ke-10 activity lama.
	ErrDateRequired = errors.New("inboxreceivetka: tanggal kelengkapan dokumen wajib diisi")

	// ErrTaskNotFound: nomor klaimnya tidak ada di inbox TKA, atau tanggalnya sudah terisi.
	//
	// Keduanya digabung dengan sengaja: dari sudut pandang pengguna, baris yang sudah
	// dikerjakan orang lain dan baris yang tidak pernah ada sama-sama berarti "tidak ada
	// lagi yang perlu Anda kerjakan di sini". Membedakannya hanya akan memberi tahu
	// pemanggil bahwa sebuah nomor klaim ADA — keterangan yang tidak ia butuhkan.
	ErrTaskNotFound = errors.New("inboxreceivetka: pekerjaan tidak ditemukan di inbox TKA")

	// ErrClaimMissing: barisnya ada di inbox TKA, tetapi klaimnya tidak ada di
	// `T_CLAIM_PNC`.
	//
	// Ia BUKAN ErrTaskNotFound. Pekerjaannya terlihat di layar dan pengguna baru saja
	// menekan Submit; yang gagal adalah menemukan klaim yang harus diperbaruinya. Pesannya
	// karena itu harus menyebut bahwa datanya tidak sinkron, bukan bahwa pekerjaannya
	// tidak ada — pengguna sedang menatap barisnya.
	ErrClaimMissing = errors.New(
		"inboxreceivetka: klaim tidak ditemukan pada data klaim utama")

	// ErrClaimAmbiguous: satu nomor klaim menunjuk lebih dari satu baris.
	//
	// Karena tidak ada satu pun constraint keunikan (`R-08`), ini keadaan yang mungkin
	// terjadi. Ia menghentikan penulisan, bukan memilih salah satu barisnya — memperbarui
	// baris yang salah pada data nilai klaim adalah kerusakan senyap.
	ErrClaimAmbiguous = errors.New(
		"inboxreceivetka: nomor klaim menunjuk lebih dari satu baris")
)

// Repo adalah seam ke penyimpanan inbox TKA SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat
// kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas,
// dan memang tidak boleh ada.
type Repo interface {
	// List mengembalikan klaim TKA yang tanggal kelengkapan dokumennya belum diisi,
	// terpotong pada MaxRows.
	List(ctx context.Context, filter Filter) (Page, error)

	// Complete mengisi tanggal kelengkapan dokumen dan mengembalikan baris yang terisi.
	//
	// # Kenapa ia mengembalikan Task, bukan sekadar galat
	//
	// Surel pemberitahuannya memuat enam nilai milik baris itu — nomor klaim, nomor polis,
	// nama tertanggung, nama peserta, Date Of Loss, dan tanggal yang baru diisi. Membacanya
	// lewat panggilan terpisah SETELAH pengisian tidak mungkin: barisnya sudah tidak
	// memenuhi penyaring inbox, sehingga ia hilang. Membacanya SEBELUM pengisian membuka
	// jarak waktu yang dapat diisi orang lain.
	//
	// Karena itu pembacaan dan penulisan terjadi di dalam SATU transaksi, dan nilainya
	// dikembalikan dari sana.
	//
	// # SATU kolom yang ditulis, dan tabel engine Pega TIDAK disentuh
	//
	//	POOLDATA.T_CLAIM_PNC.TGLDOKLENGKAP
	//
	// Sistem lama menulis case Pega, dan nilainya mengalir ke kolom itu lewat konversi JSON
	// (`Database/PEGA_CONVERT_JSONKLAIM_PNC.prc`, yang memetakan kunci JSON
	// `TanggalDokLengkap` ke `TGLDOKLENGKAP`). Kita menulisnya langsung.
	//
	// Kolom `PC_ASM_FW_GCNMFW_WORK.TANGGALDOKLENGKAP` yang menjadi penyaring Pega sengaja
	// **tidak** ikut ditulis. Pega menyimpan nilai sebenarnya di BLOB kasus dan menyalinnya
	// ke kolom itu; menulis kolomnya dari luar berarti nilainya tertimpa tanpa satu pun
	// tanda begitu Pega menyimpan kasusnya lagi — dan pekerjaan yang tampil di layar ini
	// seluruhnya masih berjalan.
	//
	// Yang membuat barisnya tetap hilang dari layar adalah penyaring daftarnya, yang
	// memeriksa KEDUA kolom tanggal. Harganya satu dan ia terlihat: selama masa paralel,
	// layar Pega masih menampilkan klaim itu sebagai belum lengkap. Ketidakcocokan yang
	// terlihat jauh lebih murah daripada data yang hilang tanpa jejak.
	//
	// # Galat yang WAJIB dihasilkan
	//
	// Pengisi seam ini wajib membatalkan transaksi dan menghasilkan galat bila jumlah baris
	// yang tersentuh bukan tepat satu. Penjaga `TGLDOKLENGKAP IS NULL` pada pernyataannya
	// itulah yang mengubah perlombaan dua permintaan menjadi penolakan yang bersuara —
	// dan karena itu surel ganda tidak dapat terjadi.
	Complete(ctx context.Context, one Completion) (Task, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan pekerjaan satu
// badan hukum — lengkap dengan nama tertanggung dan nama peserta — kepada petugas badan
// hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Notice adalah peristiwa domain "tanggal kelengkapan dokumen sudah diisi".
//
// # Ia peristiwa, bukan perintah kirim surel
//
// Isinya menyatakan APA YANG TERJADI, bukan siapa yang harus diberi tahu
// (`04-FUTURE-ARCHITECTURE.md` §3.6). Itulah yang membuat penerimanya menjadi urusan
// konfigurasi dan bukan urusan pemanggil — dan itu pula yang menutup pola hardcode yang
// hidup di layar lama; lihat Notifier.
//
// Keenam medannya sama persis dengan keenam baris tabel pada
// `HTML/NotificationKelengkapanTKA-HTML.xml`, pada urutan yang sama.
type Notice struct {
	ClaimNumber     string
	PolicyNumber    string
	InsuredName     string
	ParticipantName string
	DateOfLoss      *time.Time
	CompletedAt     time.Time
}

// Notifier adalah seam ke pemberitahuan.
//
// Pengisinya ada di notification/: SMTP untuk produksi, dan perekam untuk pengujian —
// dua pengisi nyata, sehingga seam ini bukan seam hipotetis.
//
// # Apa yang digantikan, dan apa yang TIDAK dibawa
//
// Langkah ke-7 dan ke-8 `Activity/SubmitTanggalLengkapTKA-Act.xml`: menyusun badan surel
// dari `NotificationKelengkapanTKA`, lalu memanggil `SendEmailNotification`.
//
// Activity `SendEmailNotification` ternyata **ADA di export** — pada berkas
// `Activity/SendEmailNotification_act.xml`, berakhiran `_act` alih-alih `-Act` seperti
// berkas lain, dan itulah sebabnya audit gap terdahulu melewatkannya. Isinya bukan logika
// bisnis melainkan **activity bawaan Pega** (`Pega-IntegrationEngine 08-03-01`, kelas
// `@baseclass`) dengan satu langkah Java pengirim SMTP. Tidak ada yang perlu
// direkayasa-balik darinya; ia memetakan langsung ke adapter SMTP.
//
// **Tiga hal dari pemanggilan lamanya sengaja TIDAK dibawa:**
//
//  1. **Penerima yang ditentukan siapa yang login.** Activity lama bercabang pada tiga
//     Operator ID yang tertanam di dalam rule, dan masing-masing memilih daftar penerima
//     berbeda — salah satunya sebuah akun Gmail pribadi di jalur produksi. Pola "nama orang
//     menjadi syarat" ini sudah dicabut sekali pada `D-52` untuk penjenjangan komite, dan
//     dicabut lagi di sini. Penerima datang dari konfigurasi (`D-15`, `D-67`).
//
//  2. **Kata sandi SMTP dan alamat server yang tertanam di dalam rule.** Keduanya terbaca
//     di export sebagai teks biasa, bersama `UseSSL=false` (`R-17`). Nilainya tidak
//     direproduksi di berkas mana pun yang di-commit (`D-69`); yang dipakai adalah
//     `SMTP_*` pada platform/config, sama seperti modul Master Rekening.
//
//  3. **Kegagalan surel yang membatalkan pekerjaan.** Lihat catatan pada usecase.
type Notifier interface {
	// NotifyDocumentCompleted mengabarkan bahwa satu klaim TKA sudah dilengkapi
	// dokumennya.
	NotifyDocumentCompleted(ctx context.Context, notice Notice) error
}
