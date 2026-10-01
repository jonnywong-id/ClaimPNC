// Package inboxrclpucl adalah inti modul Inbox RCL/PUCL.
//
// # Nama modul ini
//
// Diambil dari nama yang dipakai Work Owner dan dari layar lamanya sendiri: butir menu
// `MENU_ID 61` pada `Database/m_menu_aplikasi_pnc.csv` berbunyi **"Inbox RCL/PUCL"**, dan
// `Harness/RCLPUCL_Harness-Harness.xml` memuat judul yang sama persis di dalam layarnya
// (`pyCaption Inbox RCL/PUCL`). `D-81` menetapkan nama modul mengikuti nama yang dipakai
// Work Owner.
//
// # Artefak Pega yang dibaca
//
// Layar ini LENGKAP di export, sehingga hampir seluruh bentuknya terbaca dari bukti:
//
//	Harness/RCLPUCL_Harness-Harness.xml                 judul layar, susunan tab
//	Section/InputPUCL-RCL_Section-Section.xml           kontainer tab — TIDAK punya grid
//	Section/InboxCetakSuratPUCLRCL_Section-Section.xml  grid tab 1, 2 isian tanggal
//	Section/InboxKelengkapanDocPUCLRCL_Section-*.xml    grid tab 2, tombol Reminder PUCL
//	Section/InboxAJSMSIG_Section-Section.xml            grid tab 3
//	Report Definition/InboxPUCL_RD-RD.xml               penyaring tab 1
//	Report Definition/InboxPUCLCetakSurat_RD-RD.xml     penyaring tab 2
//	Report Definition/InboxMISG_RD-RD.xml               penyaring tab 3
//	RDB List/ReminderPUCL-SQL.xml                       SQL hasil generate RD tab 2 — nama kolom
//	RDB List/GetDataPUCLRCLForDailyReport-SQL.xml       kueri ekspor tab 1
//	Activity/ExportCetakSurat_act-Act.xml               tombol ekspor tab 1
//	Activity/ExportKelengkapanDoc_act-Act.xml           tombol ekspor tab 2
//	Activity/ExportAJSMSIGDoc_act-Act.xml               tombol ekspor tab 3
//	When/IsRCLPUCL-When.xml                             siapa yang melihat menunya
//
// # Apa itu Inbox RCL/PUCL
//
// Antrean **klaim yang ditolak atau diproses ulang**. `CONTEXT.md` mendefinisikan RCL
// sebagai Rejected Klaim dan PUCL sebagai Proses Ulang Klaim.
//
// Ia benar-benar Inbox menurut `D-79`: barisnya adalah **tugas** yang menunggu tindakan,
// barisnya **hilang** begitu tugasnya selesai (penyaring `PYSTATUSWORK <> Resolved-Completed`),
// dan barisnya menempuh penugasan — ia diambil dari `DATAPEGA.PC_ASSIGN_WORKBASKET`, bukan
// dari tabel data acuan.
//
// # SATU ANTREAN, TIGA PARTISI — dan itulah bentuk sebenarnya layar ini
//
// Ketiga tab membaca antrean bersama yang SAMA (`RCLPUCL`) dan tabel yang sama. Yang
// membedakan hanyalah tiga penyaring, dan ketiganya menyangkut PERJALANAN SURAT PUCL:
//
//	tab 1 Cetak Surat           surat BELUM dicetak    TANGGALCETAKDOKUMENPUCL_1 IS NULL
//	                                                   dan STATUSCASE_1 = '0'
//	tab 2 Kelengkapan Dokumen   surat SUDAH dicetak    TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL
//	                            MASIH di tangan PUCL   PUCLAPPROVE_1 <> '1'
//	                            bukan jalur MSIG       MSIG_1 IS NULL
//	tab 3 Klaim MSIG            sama seperti tab 2     MSIG_1 = 'MSIG'
//	                            tetapi jalur MSIG
//
// Urutannya bukan kebetulan: ia mengikuti perjalanan satu klaim RCL/PUCL dari belum
// bersurat menjadi sudah bersurat. Satu klaim berpindah tab dengan sendirinya begitu
// suratnya dicetak.
//
// # DUA HAL YANG HARUS DISADARI SEBELUM MEMBACA SISA BERKAS INI
//
// PERTAMA — kolom `MSIG_1` nyaris tidak pernah terisi, tetapi ia BUKAN kosong.
//
// `claim-pnc/docs/kolom-t-claimlist-admin.md` dibaca dari katalog Oracle pada 2026-09-22 dan
// mendaftar seluruh kolom ber-`NUM_DISTINCT > 0` pada tabel yang sama. `RCL_PUCL_1`,
// `PUCLAPPROVE_1`, `STATUSCASE_1`, dan `TANGGALKIRIMPUCL_1` ADA di sana; `MSIG_1` TIDAK —
// sehingga modul ini sempat dibangun dengan dugaan bahwa kolomnya tidak pernah diisi.
//
// **Dugaan itu terbantah 2026-09-30**, dengan menghitung barisnya langsung:
// `SELECT MSIG_1, COUNT(*) … GROUP BY MSIG_1` pada portal ASM mengembalikan `MSIG` satu
// baris dan kosong 7.721 baris. Kolomnya terisi, hanya sangat jarang. Inventaris katalog
// itu rupanya tidak menangkap kolom yang isinya sesedikit ini.
//
// Work Owner menjelaskan pada hari yang sama apa yang DITANDAI kolom itu: **klaim yang
// datanya dari atau untuk perusahaan MSIG**.
//
// Akibatnya: **tab 3 nyaris selalu kosong**, dan tab 2 (`MSIG_1 IS NULL`) menampung
// selebihnya. Keputusan Work Owner 2026-09-23: bangun apa adanya, nyatakan temuannya ke
// pengguna. Lihat PlannedDifferences.
//
// KEDUA — judul kolom yang SAMA menunjuk kolom yang BERBEDA di layar ini dan di layar
// Inbox Manager Receive / PUCL. Lihat catatan pada WorkItem.Track dan WorkItem.ExpiryStatus.
// Ini bukan kekeliruan pembacaan; ia utang teknis §4.2 yang memang begitu di sistem lama,
// dan masing-masing layar membawa pemetaannya sendiri (`D-13`, `P-5`).
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxrclpucl

import (
	"context"
	"errors"
	"strings"
)

// WorkItem adalah satu baris pada grid — satu klaim RCL/PUCL yang menunggu tindakan.
//
// Kesembilan isian yang digambar sama persis di KETIGA tab; yang berbeda hanya judul satu
// kolom (lihat tab.go). Karena itu tidak ada satu pun isian di sini yang kosong pada
// sebagian tab — berbeda dari modul Inbox Manager Receive / PUCL, tempat sembilan dari
// enam belas isian hanya berlaku pada salah satu tab.
type WorkItem struct {
	// Reference adalah kunci yang dipakai membuka layar kerja klaim ini.
	//
	// Ia TIDAK digambar sebagai kolom; yang memakainya adalah tautan baris
	// (`pzGridOpenAction` di sistem lama) dan tombol rincian di sistem baru.
	//
	// # Isinya berubah 2026-10-01, dan ini terlihat di alamat layar
	//
	// Sebelumnya ia kunci teknis `PZINSKEY` berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx` — nama
	// kelas internal Pega tertanam di dalam kunci data bisnis, utang teknis §4.1 yang `D-22`
	// dan `D-71` hapus untuk klaim baru.
	//
	// Sejak layar ini membaca `POOLDATA.TC_PNC_PUCL`, isinya **nomor case** (`PNC-1865`):
	// tabel datar itu hanya menyimpan satu kunci, dan kunci itu `PYID`. Akibatnya alamat
	// layar kerja memuat bentuk pendek, dan tautan lama berbentuk panjang TIDAK akan
	// ditemukan.
	//
	// Ia karena itu bernilai SAMA dengan CaseID. Keduanya tetap dipisah sebagai dua isian:
	// yang satu kontrak tautan, yang satu kolom yang digambar, dan keduanya dapat kembali
	// berbeda bila kelak tabel datar menyimpan kunci teknisnya pula.
	Reference string

	// CaseID — kolom **"Nomor Case"** <- `PYID`.
	//
	// Judulnya memang "Nomor Case", bukan "Nomor Klaim". Itu teks layar lama, dan `D-13`
	// menetapkan teks yang dilihat pengguna mengikutinya apa adanya.
	CaseID string

	PolicyNumber string // "No Polis"         <- POLICYNO
	InsuredName  string // "Nama Tertanggung" <- QQNAME

	// InboxEntryAt — kolom **"Tanggal Masuk Inbox"** <- `TANGGALKIRIMPUCL_1`.
	//
	// Perhatikan ia BUKAN `PXCREATEDATETIME`. Keduanya mudah tertukar karena judulnya
	// terbaca seperti "kapan barisnya dibuat", tetapi yang digambar section adalah
	// `.ClaimData.PUCLStatus.TanggalKirimPUCL` — kapan klaimnya DIKIRIM ke jalur RCL/PUCL,
	// bukan kapan objek kerjanya lahir. Sebuah klaim dapat lahir berbulan-bulan sebelum ia
	// masuk antrean ini.
	//
	// Layar Inbox Manager Receive / PUCL menggambar `PXCREATEDATETIME` di bawah judul yang
	// sama persis. Keduanya dibawa apa adanya menurut layarnya masing-masing (`P-5`).
	InboxEntryAt string

	// AnalystNote — kolom **"Deskripsi Analyst"** <- `KOMENTARANALISATOR_1`.
	//
	// Alias Pega-nya menyesatkan dan tidak dibawa (`D-19`): kolom yang sama dialiaskan
	// "LOGSEARCH" di `ReminderPUCL-SQL.xml`, "NoteKomite" di `GetReminderPUCL-SQL.xml`, dan
	// "CloseClaimNote" di `GetDataPUCLRCLForDailyReport-SQL.xml`. Tidak satu pun menyatakan
	// isinya.
	AnalystNote string

	// Track — kolom **"Status RCL/PUCL"** pada tab 1 dan 2, **"Status"** pada tab 3
	// <- `RCL_PUCL_1`.
	//
	// # PERANGKAP PENAMAAN YANG HARUS DIBACA SEBELUM MENYAMAKANNYA DENGAN MODUL LAIN
	//
	// Judul "Status RCL/PUCL" di layar INI menunjuk `RCL_PUCL_1` — kode jalurnya.
	// Judul "Status RCL/PUCL" di layar Inbox Manager Receive / PUCL menunjuk
	// `STATUSKLAIM_1` — kolom yang BERBEDA pada tabel yang SAMA.
	//
	// Keduanya diverifikasi dari sel grid masing-masing section, bukan disimpulkan. Yang
	// dibawa adalah pemetaan milik layarnya sendiri (`D-13`), dan menyamakan keduanya akan
	// menampilkan kolom yang salah tanpa satu pun galat.
	//
	// Kolomnya menyimpan ANGKA: `1` RCL, `2` PUCL, `3` **Notification**. Ketiganya
	// digambar sebagai teks, dan `3` pun punya teksnya sendiri — lihat catatan pada TrackOf
	// untuk bukti bahwa Pega memang menggambarnya, dan mengapa modul ini sempat
	// mengosongkannya. Lihat pula TrackHidden untuk akibat yang lebih besar dari kode itu.
	Track string

	// LetterPrintedAt — kolom **"Tanggal Cetak Surat"** <- `TANGGALCETAKDOKUMENPUCL_1`.
	//
	// Ia sekaligus PENYARING yang memisahkan tab 1 dari tab 2 dan 3. Pada tab 1 ia SELALU
	// kosong — penyaringnya `IS NULL` — dan itu bukan data hilang melainkan justru arti
	// tab itu: surat belum dicetak. Kolomnya tetap digambar di sana karena section lama
	// menggambarnya (`D-13`).
	LetterPrintedAt string

	// ClaimAge — kolom **"Lama Klaim"** <- `LAMAKLAIM_1`.
	//
	// # Judulnya menyebut DURASI; isinya TANGGAL
	//
	// Work Owner menjelaskan 2026-09-30: isinya **tanggal kirim untuk proses PUCL** — bukan
	// lamanya klaim. Judul "Lama Klaim" karena itu menyesatkan sejak di Pega, sama seperti
	// alias-alias yang didaftar di kepala berkas `.sql`. Judulnya tetap dibawa apa adanya
	// (`D-13`); yang tidak dibawa adalah salah artinya.
	//
	// # Ia nyaris kembar dengan "Tanggal Masuk Inbox", dan itu sudah diukur
	//
	// `TANGGALKIRIMPUCL_1` — yang digambar sebagai "Tanggal Masuk Inbox" — namanya
	// menyatakan hal yang sama persis, dan jumlah nilai berbedanya pun nyaris sama di
	// produksi: **88 lawan 86** (`docs/kolom-t-claimlist-admin.md` §B.3).
	//
	// Pembacaan langsung pada 2026-09-30 menjelaskan mengapa: keduanya ditulis pada langkah
	// yang SAMA dan hanya terpaut milidetik —
	//
	//	kirim = 2025-06-13T14:41:01.532+07:00
	//	lama  = 2025-06-13T14:41:01.531+07:00
	//
	// Dari 7.722 baris, 61 memuat keduanya persis sama dan 25 berbeda. Setelah digambar
	// sampai satuan detik, kedua kolom karena itu kerap terbaca IDENTIK.
	//
	// Keduanya tetap digambar. Duplikasinya ada di Pega, dan keputusan Work Owner
	// 2026-09-30 adalah mengikuti Pega apa adanya (`P-5`).
	//
	// # Kenapa TEKS, dan kenapa tidak dihitung sendiri
	//
	// Teks, karena DDL-nya tidak pernah diterima (`R-08`) dan yang terverifikasi barulah
	// satu basis data dari enam. Pada portal ASM kolomnya `TIMESTAMP(6)`, dan penggambarnya
	// (`DisplayTimeText`) melewatkan bentuk yang tidak dikenalinya apa adanya — sehingga
	// portal yang menyimpannya sebagai teks tetap terlayani.
	//
	// Tidak dihitung sendiri, karena tidak satu pun kueri di export MENGHITUNGNYA; keempatnya
	// hanya MEMILIH kolomnya. Modul `inboxcloseclaim` dan `inboxanalystdoctor` memang
	// menghitung kolom serupa dari selisih tanggal, tetapi itu keputusan Work Owner untuk
	// layar yang Report Definition-nya TIDAK mengambil kolom durasi apa pun. Di sini
	// keadaannya kebalikannya: kolomnya diambil section secara eksplisit, sehingga
	// menggantinya dengan hitungan sendiri berarti menampilkan angka yang berbeda dari yang
	// dilihat pengguna hari ini.
	ClaimAge string

	// ExpiryStatus — kolom **"Status Kadaluarsa"** <- `STATUSKLAIM_1`.
	//
	// # PERANGKAP KEDUA, dan arahnya berlawanan dengan yang pertama
	//
	// Judul "Status Kadaluarsa" di layar INI menunjuk `STATUSKLAIM_1`.
	// Judul "Status Kadaluarsa" di layar Inbox Manager Receive / PUCL menunjuk
	// `STATUSCASE_1`.
	//
	// Jadi kedua layar memakai DUA judul yang sama untuk EMPAT kolom, bersilangan:
	//
	//	judul                layar ini        Inbox Manager Receive / PUCL
	//	-------------------- ---------------- ----------------------------
	//	Status RCL/PUCL      RCL_PUCL_1       STATUSKLAIM_1
	//	Status Kadaluarsa    STATUSKLAIM_1    STATUSCASE_1
	//
	// Ia diverifikasi dari sel grid ketiga section layar ini, dan `STATUSCASE_1` memang
	// TIDAK digambar satu sel pun di sini — meski Report Definition tab 1 MENYARING
	// dengannya. Itu perilaku sistem lama apa adanya; keduanya dibawa menurut layarnya
	// masing-masing (`P-5`).
	ExpiryStatus string

	// CreatedAt — kolom **"Tanggal Dibuat"** <- `PXCREATEDATETIME`.
	//
	// # Ia kolom yang MENGURUTKAN, dan sebelumnya tidak digambar sama sekali
	//
	// Urutan baris layar ini `ORDER BY PXCREATEDATETIME DESC` — mengikuti Report Definition
	// sistem lama apa adanya. Sistem lama tidak menggambar kolomnya, sehingga tabelnya
	// terbaca TIDAK URUT: yang tampil sebagai "Tanggal Masuk Inbox" adalah
	// `TANGGALKIRIMPUCL_1`, dan keduanya dapat terpaut berbulan-bulan karena sebuah klaim
	// lahir jauh sebelum ia masuk antrean ini.
	//
	// Keputusan Work Owner 2026-09-30: **kolom pengurutnya ditampilkan**.
	//
	// Yang menentukan pilihan itu di antara tiga kemungkinan: menambah kolom **tidak
	// mengubah satu baris pun**. Urutan, isi, dan pembagian halamannya sama persis dengan
	// sebelumnya, sehingga uji kesetaraan gerbang 1 tidak tersentuh. Mengganti kunci
	// urutnya — pilihan yang tampak lebih langsung — akan memindahkan baris antarhalaman,
	// dan itu selisih yang jauh lebih mahal untuk hal yang sama.
	CreatedAt string
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
//
// # Kenapa modul ini menuntutnya, padahal tidak ada kueri yang menyaring menurut pemanggil
//
// Karena antrean ini BERSAMA, bukan milik seseorang: penyaringnya `PXASSIGNEDOPERATORID =
// 'RCLPUCL'`, dan `RCLPUCL` adalah akun antrean, bukan nama orang. Setiap petugas yang
// membuka layar ini melihat daftar yang sama persis.
//
// Selama pemeriksaan peran belum ada (`TKT-F3-004`), satu-satunya hal yang menyatakan siapa
// yang membukanya adalah jejak di log — dan jejak itu tidak dapat ditulis tanpa identitas.
// `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang justru untuk keadaan ini.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama. Bukan NIK.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// WorkClassClaim adalah kelas objek kerja yang dibaca layar ini.
//
// `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` menampung beberapa kelas objek kerja sekaligus dan hanya
// dibedakan kolom `PXOBJCLASS`. Melupakan penyaring itu mencampur klaim dengan berkas
// penerimaan dokumen — dan keduanya punya `PYID`, `POLICYNO`, serta `QQNAME`, sehingga
// hasilnya TIDAK menghasilkan satu pun galat.
//
// Nilainya terbaca sebagai `pyClassName` pada ketiga Report Definition layar ini, dan
// dipakai sebagai syarat gabungan oleh `RDB List/ReminderPUCL-SQL.xml`.
const WorkClassClaim = "ASM-FW-GCNMFW-Work-PNC"

// RCLPUCLWorkbasket adalah akun antrean bersama yang memegang pekerjaan RCL/PUCL.
//
// Nilainya BUKAN tebakan dan bukan pinjaman dari modul lain: ketiga section layar ini
// mengirimkannya sendiri sebagai parameter Report Definition —
//
//	<pyRDParams><assign>"RCLPUCL"</assign></pyRDParams>
//
// dan ketiga Report Definition-nya menyaring `AssignBasket.pxAssignedOperatorID =
// Param.assign`. SQL hasil generate-nya menuliskannya sebagai literal
// (`RDB List/ReminderPUCL-SQL.xml`), sehingga nilainya dapat dibaca langsung.
//
// Ia akun fungsional, bukan nama orang, sehingga menuliskannya di sini tidak melanggar
// `D-67`.
const RCLPUCLWorkbasket = "RCLPUCL"

// WorkStatusCompleted adalah status kerja yang DIKELUARKAN dari ketiga tab.
//
// Penyaringnya `<>`, bukan `=`. Satu tanda yang salah membalik seluruh isi layar: yang
// tampil menjadi klaim yang sudah tuntas, dan tidak ada apa pun di layar yang menandakannya.
//
// Perhatikan ia HANYA menyebut `Resolved-Completed`. `Resolved-Rejected` TIDAK dikecualikan,
// sehingga klaim yang ditolak TETAP muncul — dan itu memang benar: klaim yang ditolak justru
// pekerjaan utama antrean RCL.
const WorkStatusCompleted = "Resolved-Completed"

// Jalur penanganan klaim.
//
// Kolom `RCL_PUCL_1` menyimpan angka; teks inilah yang digambar.
//
// # DARI MANA TEKSNYA, dan kenapa sumber yang pertama dipakai KELIRU
//
// Modul ini semula menerjemahkannya mengikuti `RDB List/GetReminderPUCL-SQL.xml:7-9`:
//
//	case when a.RCL_PUCL_1 = '1' then 'RCL'
//	     when a.RCL_PUCL_1 = '2' then 'PUCL'
//	End
//
// `CASE` itu tanpa `ELSE`, sehingga kode `3` menghasilkan sel KOSONG — dan itu direplikasi
// apa adanya selama beberapa hari.
//
// **Kueri itu bukan yang memasok grid.** Ia memasok pengingat PUCL. Sel grid-nya sendiri
// adalah kontrol daftar pilihan baca-saja —
//
//	Section/InboxAJSMSIG_Section-Section.xml
//	  <pyValue>.ClaimData.PUCLStatus.RCL_PUCL</pyValue>
//	  <pyFormat>pxRadioButtons</pyFormat>
//	  <pyEditOptions>Read-only</pyEditOptions>
//
// — dan kontrol semacam itu menggambar **label** pilihan yang terpilih, bukan kodenya. Label
// itu hidup di rule Property `RCL_PUCL`, yang **tidak ada di export**: seluruh folder
// `Property/` hanya memuat satu berkas, dan bukan yang ini (`R-16`).
//
// Jadi teksnya memang tidak dapat dibaca dari artefak mana pun. Yang membuktikannya layar
// Pega yang berjalan: tab "Klaim MSIG" menggambar **"Notification"** untuk `PNC-1503`, dan
// baris itu terverifikasi `RCL_PUCL_1 = '3'` di basis data (2026-09-30). Ia sejalan dengan
// keterangan Work Owner pada tanggal yang sama: `1` RCL · `2` PUCL · `3` Notification.
//
// Nilai di luar ketiganya tetap menghasilkan teks KOSONG. Satu baris di produksi memang
// berkode kosong, dan sel kosong adalah jawaban yang benar untuknya.
const (
	// TrackRCL — Rejected Klaim (`CONTEXT.md`).
	TrackRCL = "RCL"

	// TrackPUCL — Proses Ulang Klaim (`CONTEXT.md`).
	TrackPUCL = "PUCL"

	// TrackNotification — bukan pekerjaan RCL maupun PUCL, melainkan pemberitahuan.
	//
	// Ia jalur yang MENYEMBUNYIKAN layar kerja di Pega; lihat TrackHidden.
	TrackNotification = "Notification"
)

// Kode mentah jalur penanganan sebagaimana tersimpan di `RCL_PUCL_1`.
//
// Dikumpulkan sebagai konstanta supaya penerjemah SQL dan penerjemah penyimpanan memori
// tidak dapat berselisih tanpa ketahuan.
const (
	TrackCodeRCL          = "1"
	TrackCodePUCL         = "2"
	TrackCodeNotification = "3"
)

// TrackOf menerjemahkan kode jalur menjadi teks yang digambar grid.
//
// Ia ada di paket domain, bukan hanya di SQL, karena kedua pengisi seam wajib menghasilkan
// teks yang sama persis — dan uji aturan modul yang berjalan di atas memori hanya menyatakan
// sesuatu tentang Oracle bila keduanya memakai penerjemah yang sama.
//
// Nilai yang tidak dikenali menghasilkan teks KOSONG. Ia sengaja tidak diganti "—" maupun
// kode mentahnya: yang pertama milik layar, yang kedua akan menampilkan angka yang tidak
// berarti apa pun bagi pengguna. Satu baris di produksi memang berkode kosong.
func TrackOf(code string) string {
	switch strings.TrimSpace(code) {
	case TrackCodeRCL:
		return TrackRCL
	case TrackCodePUCL:
		return TrackPUCL
	case TrackCodeNotification:
		return TrackNotification
	default:
		return ""
	}
}

// Nilai penyaring yang memisahkan ketiga tab.
//
// Ketiganya dikumpulkan di sini, berdampingan, supaya perbedaannya tidak dapat terlewat saat
// membaca — `D-15` melarang nilai bisnis tertanam berulang kali di dalam kode, dan di sini
// alasannya lebih tajam: satu nilai yang salah memindahkan seluruh isi layar ke tab yang
// keliru tanpa satu pun galat.
const (
	// ExpiryStatusActive adalah nilai `STATUSCASE_1` yang menempatkan klaim di tab
	// "Cetak Surat" — penyaring `C` pada `InboxPUCL_RD`.
	//
	// Artinya BELUM DIKETAHUI. Kolomnya punya tiga nilai berbeda di produksi
	// (`docs/kolom-t-claimlist-admin.md` §B.3) dan tidak ada master yang menerjemahkannya
	// di export mana pun. Yang diketahui hanyalah `'0'` inilah yang dipakai penyaring, dan
	// itu dibawa apa adanya.
	//
	// Keputusan Work Owner 2026-09-30: **"ikuti apa adanya saja dari Pega"** — artinya
	// tidak akan dicari, dan penyaringnya tidak akan diubah. Peringatan `-periksa` tetap
	// dipasang: bila nilai `'0'` kelak berubah di Pega, tab 1 akan kosong tanpa satu pun
	// galat, dan peringatan itulah satu-satunya yang akan menyebut sebabnya.
	//
	// Perhatikan kolom ini MENYARING tab 1 tetapi TIDAK digambar satu sel pun di layar —
	// judul "Status Kadaluarsa" menunjuk `STATUSKLAIM_1`. Lihat WorkItem.ExpiryStatus.
	ExpiryStatusActive = "0"

	// PUCLReturnedToAnalyst adalah nilai `PUCLAPPROVE_1` yang MENGELUARKAN klaim dari tab 2
	// dan 3 — penyaring `D` pada kedua Report Definition-nya.
	//
	// # Arti kedua nilainya, dan kenapa namanya BUKAN "PUCLApproved"
	//
	// Work Owner, 2026-09-30:
	//
	//	'0'  Kirim Ke PUCL          — klaimnya berada di tangan PUCL, MENUNGGU dikerjakan
	//	'1'  PUCL kirim Ke Analyst  — PUCL sudah selesai dan mengembalikannya ke Analyst
	//
	// Nama lamanya `PUCLApproved` menyatakan "disetujui", dan itu bukan yang terjadi:
	// yang ditandai adalah **kepada siapa klaimnya sekarang berada**, bukan putusan setuju
	// atau tolak. Nama yang menyatakan putusan pada kolom yang menyatakan posisi adalah
	// persis jenis kekeliruan yang `D-19` larang dibawa dari sistem lama.
	//
	// Penyaring `<> '1'` karena itu berarti **"masih di tangan PUCL"**, dan tab 2 memang
	// antrean pekerjaan PUCL. Perilakunya benar; yang keliru hanya namanya, dan itu
	// diperbaiki di sini tanpa menyentuh satu pun kueri.
	//
	// # Nilai KOSONG tetap tidak lolos, dan sekarang itu dapat dinilai
	//
	// `NULL <> '1'` menghasilkan UNKNOWN, bukan TRUE, sehingga klaim yang penandanya belum
	// pernah diisi TIDAK muncul — di Oracle maupun PostgreSQL. Dengan arti di atas, itu
	// **masuk akal**: klaim yang belum pernah dikirim ke PUCL memang bukan pekerjaan PUCL.
	//
	// Yang tetap perlu dihitung adalah klaim yang suratnya SUDAH dicetak tetapi penandanya
	// kosong. Klaim seperti itu keluar dari tab 1 (suratnya sudah dicetak) dan tidak masuk
	// tab 2 maupun 3 — ia tidak terlihat di tab mana pun. Kueri hitungnya dicetak
	// `-periksa`.
	PUCLReturnedToAnalyst = "1"

	// PUCLWithPUCL adalah nilai `PUCLAPPROVE_1` yang MENAHAN klaim di tab 2 dan 3.
	//
	// Tidak dipakai satu pun penyaring — penyaringnya menyebut nilai yang dikecualikan,
	// bukan nilai yang diterima. Ia ditulis di sini supaya kedua nilai kolom ini terbaca
	// berdampingan: tanpa pasangannya, `PUCLReturnedToAnalyst` terbaca seolah satu-satunya
	// nilai yang mungkin.
	PUCLWithPUCL = "0"

	// MSIGMarker adalah nilai `MSIG_1` yang menempatkan klaim di tab "Klaim MSIG" —
	// penyaring `E` pada `InboxMISG_RD`.
	//
	// Ia menandai **klaim yang datanya dari atau untuk perusahaan MSIG** (Work Owner,
	// 2026-09-30). Kolomnya TERISI di produksi, tetapi sangat jarang — satu baris dari
	// 7.722 pada portal ASM, dihitung langsung 2026-09-30. Lihat catatan di kepala paket:
	// dugaan bahwa ia tidak pernah terisi terbantah oleh hitungan itu.
	MSIGMarker = "MSIG"
)

// GroupPanelPA adalah kode Group Panel Personal Accident.
//
// Ia dipakai HANYA oleh cabang kedua laporan harian, dan tidak oleh satu pun grid.
// `GetDataPUCLRCLForDailyReport-SQL.xml` menambahkan seluruh klaim ber-Group Panel `002`
// pada rentang tanggal yang sama — tanpa gabungan antrean bersama sama sekali — sehingga
// laporan memuat klaim PA yang tidak pernah masuk antrean RCL/PUCL.
//
// Bahwa `002` adalah Personal Accident terbaca di `CONTEXT.md` dan Business Understanding
// §1, dan kolomnya memang dibaca kueri lain pada kelas yang sama
// (`RDB List/GetDataRCVallKlaimPATravel-SQL.xml:10`).
const GroupPanelPA = "002"

// GroupPanelTravel adalah kode Group Panel Travel.
//
// Ia dipakai sebagai syarat tampil tombol "Kirim ke PIC Teknik" pada
// `Section/SectionPenerimaanDokumenPUCL-Section.xml` lewat When rule `IsTravel`, yang isinya
// satu perbandingan terhadap `"005"` (`When/IsTravel-When.xml`).
//
// Bahwa `005` adalah Travel terbaca di `CONTEXT.md` dan Business Understanding §1.
const GroupPanelTravel = "005"

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 50
//
// Karena itu ukuran halaman ketiga grid layar ini — `<pyPageSize>50</pyPageSize>` pada
// ketiga section, diperiksa satu per satu. Angka itu BERBEDA dari sebagian modul inbox lain
// yang memakai 25, dan perbedaannya tidak diseragamkan: yang dipakai adalah angka layarnya
// sendiri.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK, bukan
// dipenuhi diam-diam — memenuhinya membuat batas menjadi saran, bukan batas.
//
// PegaMaxRecords disimpan sebagai catatan, BUKAN untuk ditegakkan. `ADR-0011` mencatat batas
// 500 pada 54 dari 56 laporan sebagai pemotongan diam-diam, bukan paginasi. Angkanya ada di
// sini supaya selisihnya dapat dinyatakan ke pengguna, bukan supaya ditiru.
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
	PegaMaxRecords  = 500
)

// Normalize mengembalikan paginasi yang sudah dibetulkan ke rentang yang sah.
//
// Nilai di luar rentang DIBETULKAN, tidak ditolak: halaman dan ukuran datang dari parameter
// query yang mudah salah ketik, dan menolak seluruh permintaan karena `halaman=0` akan
// membuat layar gagal tanpa alasan yang terbaca pengguna.
func (p Pagination) Normalize() Pagination {
	clean := p
	if clean.Page < 1 {
		clean.Page = 1
	}
	if clean.Size < 1 {
		clean.Size = DefaultPageSize
	}
	if clean.Size > MaxPageSize {
		clean.Size = MaxPageSize
	}
	return clean
}

// Offset adalah jumlah baris yang dilewati sebelum halaman yang diminta.
func (p Pagination) Offset() int {
	clean := p.Normalize()
	return (clean.Page - 1) * clean.Size
}

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
type Page struct {
	Items []WorkItem

	// Total adalah jumlah SELURUH baris yang cocok di penyimpanan, bukan yang tampil.
	Total int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan, bukan yang
	// diminta. Layar menggambar penomoran halamannya dari sini.
	Pagination Pagination
}

// TotalPages adalah jumlah halaman, minimal 1 supaya layar tidak pernah menggambar
// "halaman 1 dari 0" saat hasilnya kosong.
func (p Page) TotalPages() int {
	size := p.Pagination.Normalize().Size
	if p.Total <= 0 {
		return 1
	}
	pages := p.Total / size
	if p.Total%size != 0 {
		pages++
	}
	return pages
}

// Slice memotong satu halaman dari seluruh baris yang sudah di tangan.
//
// Ia dipakai penyimpanan MEMORI saja. Penyimpanan SQL memotongnya di basis data dengan
// `OFFSET … FETCH NEXT`, dan perbedaan itu disengaja — lihat catatan paginasi di kepala
// repo/sqlstore/inboxrclpucl.sql.
//
// Keduanya tetap menghasilkan Page dengan arti yang sama, sehingga uji aturan modul yang
// berjalan di atas memori menyatakan hal yang benar tentang yang berjalan di Oracle.
func Slice(all []WorkItem, page Pagination) Page {
	clean := page.Normalize()

	result := Page{Total: len(all), Pagination: clean, Items: []WorkItem{}}

	offset := clean.Offset()
	if offset >= len(all) {
		return result
	}

	end := offset + clean.Size
	if end > len(all) {
		end = len(all)
	}

	result.Items = all[offset:end]
	return result
}

// DailyReportRow adalah satu baris LAPORAN HARIAN RCL/PUCL — keluaran tombol ekspor tab
// "Cetak Surat".
//
// # Kenapa ia tipe TERSENDIRI, bukan WorkItem
//
// Karena isinya memang berbeda, dan menyamakannya akan menyembunyikan perbedaan yang justru
// harus terlihat. `Activity/ExportCetakSurat_act-Act.xml` menjalankan
// `RDB List/GetDataPUCLRCLForDailyReport-SQL.xml`, BUKAN Report Definition grid-nya, dan
// kueri itu berbeda dalam empat hal sekaligus:
//
//   - Penyaringnya RENTANG TANGGAL atas `TANGGALKIRIMPUCL_1` — grid tidak menyaring tanggal
//     sama sekali.
//   - Ia TIDAK menyaring `TANGGALCETAKDOKUMENPUCL_1 IS NULL` maupun `STATUSCASE_1 = '0'`,
//     sehingga memuat klaim yang suratnya SUDAH dicetak — yang di grid ada di tab lain.
//   - Ia TIDAK menyaring `PYSTATUSWORK`, sehingga memuat klaim yang sudah selesai.
//   - Ia ber-`UNION` dengan cabang kedua yang mengambil seluruh klaim ber-Group Panel `002`
//     (Personal Accident) dalam rentang yang sama TANPA gabungan antrean bersama sama
//     sekali.
//
// Akibatnya isi berkas ekspor TIDAK SAMA dengan isi grid yang sedang dilihat. Itu perilaku
// sistem lama, dan Work Owner memutuskan 2026-09-23 untuk mereplikasinya apa adanya
// (`P-5`). Ia dinyatakan ke pengguna lewat PlannedDifferences, bukan disamarkan.
type DailyReportRow struct {
	// Reference adalah `PZINSKEY`. Ia diambil kueri lama sebagai kolom kedua tanpa alias,
	// dan dibawa supaya baris laporan dapat ditelusuri balik ke klaimnya.
	Reference string

	CaseID       string // "Nomor Case"       <- PYID
	PolicyNumber string // "No Polis"         <- POLICYNO
	InsuredName  string // "Nama Tertanggung" <- QQNAME
	SentAt       string // "Tanggal Kirim RCL/PUCL" <- TANGGALKIRIMPUCL_1
	AnalystNote  string // "Deskripsi Analyst"      <- KOMENTARANALISATOR_1

	// LetterPrintedAt — "Tanggal Cetak Surat" <- TANGGALCETAKDOKUMENPUCL_1.
	LetterPrintedAt string

	// Track — "Status RCL/PUCL" <- RCL_PUCL_1.
	//
	// Kueri lama mengambil kolomnya MENTAH — tanpa `CASE` penerjemah, berbeda dari
	// `GetReminderPUCL`. Di sini ia tetap diterjemahkan lewat TrackOf supaya berkas ekspor
	// dan layar menyebut hal yang sama dengan kata yang sama; kode mentah `1`/`2` di dalam
	// berkas Excel tidak berarti apa pun bagi pembacanya. Itu selisih terencana, dan
	// dinyatakan lewat PlannedDifferences.
	Track string

	// ClaimStatus — "Status Klaim" <- STATUSCLAIM_1.
	//
	// # Perhatikan ia `STATUSCLAIM_1`, BUKAN `STATUSKLAIM_1`
	//
	// Kedua nama kolom itu hanya berbeda satu huruf dan artinya berbeda jauh. Yang ini
	// Status Klaim ber-33 kode `1134`–`1166` (`R-06`); yang satunya status jalur RCL/PUCL
	// yang digambar grid sebagai "Status Kadaluarsa". Menukarnya tidak menghasilkan satu
	// pun galat.
	//
	// Ia hanya ada di LAPORAN, tidak di grid mana pun.
	ClaimStatus string
}

// DateRange adalah rentang tanggal laporan harian.
//
// Kedua batasnya WAJIB, mengikuti kueri lama yang memakai keduanya sebagai `>=` dan `<=`
// tanpa penjaga apa pun — kueri itu bahkan menyisipkan keduanya langsung ke teks SQL
// (`{TempRCLPUCLReport.AlasanKlaim}` dan `{TempRCLPUCLReport.NoteKasir}`), yang di sini
// diganti parameter binding tanpa perkecualian (`08-TECHNICAL-STRATEGY.md` §4.3).
//
// Bentuknya `YYYY-MM-DD`, bukan `dd/mm/yyyy` seperti di sistem lama. Alasannya: yang
// mengirimnya adalah kontrak API, bukan layar Pega, dan bentuk ISO tidak ambigu antara
// tanggal dan bulan. Penerjemahannya ke bentuk yang dimengerti basis data dikerjakan
// penyimpanan, bukan pemanggil.
type DateRange struct {
	From string
	To   string
}

// IsZero menyatakan rentangnya tidak diisi sama sekali.
func (d DateRange) IsZero() bool {
	return strings.TrimSpace(d.From) == "" && strings.TrimSpace(d.To) == ""
}

// ClaimDetail adalah isi LAYAR KERJA RCL/PUCL untuk satu klaim.
//
// # Layar apa ini, dan kenapa ia ada di modul antrean
//
// Ia yang terbuka di Pega saat petugas mengklik nomor klaim di antrean ini. Tautannya
// menjalankan `SetAssignmentInboxPUCL_act(inskey = .pzInsKey)` — Open Assignment — dan yang
// menunggu di sana adalah flow action `SendtoRCLPUCL`, yang menyisipkan section bernama
// sama.
//
// Section itu SEMPAT HILANG dari export dan diterima Work Owner pada 2026-09-24. Isinya
// kontainer dua bagian, keduanya terbaca dari buktinya sendiri:
//
//	SectionLampiranSuratPUCL      "Lampiran Surat"     -> Letter
//	SectionPenerimaanDokumenPUCL  "Penerimaan Dokumen" -> DocumentReceipt
//
// # Di Pega ia layar TULIS; di sini ia BACA saja
//
// Memo penulis rule-nya sendiri pada bagian kedua berbunyi "add button save". Bagian
// pertama menyusun lampiran surat RCL/PUCL — itulah tindakan "Cetak Surat" yang mengisi
// `TANGGALCETAKDOKUMENPUCL_1` dan memindahkan klaimnya antartab.
//
// Keduanya menulis objek kerja, dan selama masa paralel tabel itu milik Pega (`P-1`).
// Yang dibawa ke sini karena itu hanya PEMBACAANNYA.
//
// # Satu catatan visibilitas yang terbaca dari rule-nya
//
// `pyMemo` pada `SendtoRCLPUCL` berbunyi:
//
//	visibility when .ClaimData.PUCLStatus.RCL_PUCL != 3
//
// Jadi ada nilai jalur **`3`** — di luar `1` (RCL) dan `2` (PUCL) — yang menyembunyikan
// seluruh layar ini. Artinya belum diketahui, dan `TrackOf` memang mengembalikan teks
// kosong untuknya. Lihat TrackHidden.
type ClaimDetail struct {
	// Reference adalah `PZINSKEY`, kunci yang dipakai membukanya.
	Reference string

	// ClaimNumber adalah nomor case — `PYID`, yang digambar sebagai judul layar.
	ClaimNumber string

	// Letter adalah bagian "Lampiran Surat".
	Letter LetterDraft

	// DocumentReceipt adalah bagian "Penerimaan Dokumen".
	DocumentReceipt DocumentReceipt

	// MSIG adalah penanda jalur MSIG — kolom `MSIG` pada tabel datar.
	//
	// Ia dibawa ke layar kerja, bukan hanya dipakai menyaring tab 3, karena DUA tombol
	// Lampiran Surat bergantung padanya. Lihat ShowsDownloadDocument dan ShowsCloseClaim.
	MSIG string

	// GroupPanel adalah kode lini bisnis — kolom `GROUPPANEL` pada tabel datar.
	//
	// Ia dibawa karena dua tombol Penerimaan Dokumen memilih penerima klaim berdasarkan
	// lini bisnisnya: PA ke Analyst, Travel ke PIC Teknik. Lihat ShowsSendToAnalyst.
	GroupPanel string
}

// ScreenButtons menyatakan tombol mana yang DIGAMBAR untuk sebuah klaim.
//
// # Kenapa ia dihitung di domain, bukan di React
//
// Karena syaratnya aturan bisnis, bukan tata letak: "Kirim Ke Analyst" dan "Kirim ke PIC
// Teknik" memilih PENERIMA klaim berdasarkan lini bisnisnya, dan "Tolak Klaim" hanya sah di
// jalur RCL. Menaruhnya di komponen React akan menyebarkannya ke tempat yang tidak dapat
// diuji tanpa merender layar, dan menempatkannya di luar jangkauan uji kesetaraan.
//
// Seluruh syaratnya dibaca langsung dari `pyUserData/pyCondition` tiap sel `pxButton`,
// bukan disimpulkan dari nama tombolnya.
type ScreenButtons struct {
	// DownloadDocument — "Download Dokumen" pada Lampiran Surat.
	DownloadDocument bool

	// CloseClaim — "Tutup Klaim" pada Lampiran Surat.
	CloseClaim bool

	// UploadDocument — "Unggah Dokumen" pada Penerimaan Dokumen. Selalu digambar.
	UploadDocument bool

	// ViewDocument — "Lihat Dokumen" pada Penerimaan Dokumen. Selalu digambar.
	ViewDocument bool

	// Save — "Save" pada Penerimaan Dokumen. Selalu digambar.
	Save bool

	// RejectClaim — "Tolak Klaim" pada Penerimaan Dokumen.
	RejectClaim bool

	// SendToAnalyst — "Kirim Ke Analyst" pada Penerimaan Dokumen.
	SendToAnalyst bool

	// SendToPICTeknik — "Kirim ke PIC Teknik" pada Penerimaan Dokumen.
	SendToPICTeknik bool
}

// Buttons menghitung tombol yang digambar untuk klaim ini.
//
// Tombol bagian Penerimaan Dokumen dikembalikan `false` seluruhnya ketika tabnya sendiri
// tidak digambar — kalau tidak, layar akan menyatakan tombol yang tabnya tidak ada.
func (d ClaimDetail) Buttons() ScreenButtons {
	b := ScreenButtons{
		DownloadDocument: d.ShowsDownloadDocument(),
		CloseClaim:       d.ShowsCloseClaim(),
	}
	if !d.ShowsDocumentReceipt() {
		return b
	}
	b.UploadDocument = true
	b.ViewDocument = true
	b.Save = true
	b.RejectClaim = d.ShowsRejectClaim()
	b.SendToAnalyst = d.ShowsSendToAnalyst()
	b.SendToPICTeknik = d.ShowsSendToPICTeknik()
	return b
}

// ShowsDownloadDocument menyatakan tombol "Download Dokumen" digambar.
//
//	pyCondition  .ClaimData.PUCLStatus.MSIG != 'MSIG'
//
// Jadi ia tombol jalur NON-MSIG. Klaim MSIG tidak mengunduh surat dari layar ini.
func (d ClaimDetail) ShowsDownloadDocument() bool {
	return strings.TrimSpace(d.MSIG) != MSIGMarker
}

// ShowsCloseClaim menyatakan tombol "Tutup Klaim" digambar.
//
//	pyCondition  .ClaimData.PUCLStatus.RCL_PUCL = 3 && .ClaimData.PUCLStatus.MSIG = 'MSIG'
//
// # Syaratnya menjelaskan satu hal yang semula terbaca ganjil
//
// Kode `3` (Notification) MENYEMBUNYIKAN tab "Penerimaan Dokumen" (lihat
// ShowsDocumentReceipt), sehingga klaim MSIG ber-jalur Notification hanya punya satu tab —
// dan tanpa tombol ini tidak ada satu pun tindakan yang dapat diselesaikan padanya. "Tutup
// Klaim" adalah satu-satunya jalan keluarnya, dan itulah sebabnya ia muncul tepat di
// perpotongan kedua syarat itu.
//
// Ia dan "Download Dokumen" SALING MENIADAKAN: yang satu menuntut `MSIG != 'MSIG'`, yang
// lain `MSIG = 'MSIG'`. Satu klaim tidak pernah menampilkan keduanya.
//
// Perhatikan pula akibat yang mudah terlewat: klaim ber-`MSIG = 'MSIG'` yang jalurnya BUKAN
// `3` tidak menampilkan satu pun dari keduanya. Itu perilaku sistem lama apa adanya, bukan
// lubang pembacaan.
func (d ClaimDetail) ShowsCloseClaim() bool {
	return strings.TrimSpace(d.Letter.TrackCode) == TrackCodeNotification &&
		strings.TrimSpace(d.MSIG) == MSIGMarker
}

// ShowsRejectClaim menyatakan tombol "Tolak Klaim" digambar.
//
//	pyCondition  .ClaimData.PUCLStatus.RCL_PUCL = 1
//
// Jadi menolak klaim hanya sah di jalur **RCL**, bukan PUCL. Itu sejalan dengan artinya:
// RCL adalah jalur klaim yang ditolak (`CONTEXT.md`).
//
// # Ada tombol "Tolak Klaim" KEDUA, dan ia tidak pernah tampil
//
// Section-nya memuat dua sel `pxButton` ber-label sama. Yang kedua ber-`pyCondition` **`1==2`**
// — syarat yang tidak pernah benar. Ia tombol yang dimatikan dengan cara dikarang syaratnya,
// bukan dihapus; perilakunya di sistem lama adalah TIDAK PERNAH DIGAMBAR, dan itulah yang
// ditiru. Lihat juga catatan "Reminder PUCL" di bawah.
func (d ClaimDetail) ShowsRejectClaim() bool {
	return strings.TrimSpace(d.Letter.TrackCode) == TrackCodeRCL
}

// ShowsSendToAnalyst menyatakan tombol "Kirim Ke Analyst" digambar.
//
//	pyCondition  .ClaimData.PUCLStatus.RCL_PUCL = 2 && IsPA
//
// Jadi ia tombol jalur **PUCL** pada lini **Personal Accident** saja.
func (d ClaimDetail) ShowsSendToAnalyst() bool {
	return strings.TrimSpace(d.Letter.TrackCode) == TrackCodePUCL &&
		strings.TrimSpace(d.GroupPanel) == GroupPanelPA
}

// ShowsSendToPICTeknik menyatakan tombol "Kirim ke PIC Teknik" digambar.
//
//	pyCondition  .ClaimData.PUCLStatus.RCL_PUCL = 2 && IsTravel
//
// Ia kembaran ShowsSendToAnalyst untuk lini **Travel**, dan perbedaannya BUKAN kata: Analyst
// dan PIC Teknik dua peran berbeda, sehingga tombol yang salah meneruskan klaim ke orang yang
// salah.
//
// # Ini mengoreksi catatan `keputusan-implementasi.md` §82.4
//
// Di sana tertulis "Kirim ke PIC Teknik" adalah nama yang **tidak ada di layar mana pun**.
// Itu keliru: rule `pyButtonLabel Kirim ke PIC Teknik` ADA di section ini. Yang benar dari
// catatan itu hanyalah bahwa menggambarnya TANPA syarat — seperti versi sebelumnya — salah,
// karena pada klaim PA yang tampil memang "Kirim Ke Analyst".
func (d ClaimDetail) ShowsSendToPICTeknik() bool {
	return strings.TrimSpace(d.Letter.TrackCode) == TrackCodePUCL &&
		strings.TrimSpace(d.GroupPanel) == GroupPanelTravel
}

// ShowsDocumentReceipt menyatakan tab "Penerimaan Dokumen" digambar untuk klaim ini.
//
// # Syaratnya nyata, dan letaknya bukan di tempat yang semula dikira
//
// `Section/SendtoRCLPUCL-Section.xml` memasang syarat itu pada KONTAINER tab kedua —
// kontainer ber-`pyTitle Penerimaan Dokumen` yang menyisipkan
// `SectionPenerimaanDokumenPUCL`:
//
//	<pyContainerVisibleWhen>.ClaimData.PUCLStatus.RCL_PUCL != 3</pyContainerVisibleWhen>
//
// Jadi kode `3` (Notification) menyembunyikan **satu tab**, bukan seluruh layar kerja.
//
// # Ini mengoreksi pembacaan sebelumnya
//
// `pyMemo` pada flow action berbunyi *"visibility when .ClaimData.PUCLStatus.RCL_PUCL != 3"*
// tanpa menyebut apa yang disembunyikannya, dan modul ini membacanya sebagai "seluruh layar"
// (`keputusan-implementasi.md` §45.2b dan §78 keputusan 5). Work Owner melaporkan 2026-09-30
// bahwa klaim MSIG **hanya menampilkan Lampiran Surat** — dan penelusuran ke berkas
// section-nya membenarkan laporan itu, bukan pembacaan kami.
//
// # Kenapa ia SEKARANG ditegakkan, padahal 2026-09-24 sengaja tidak
//
// Karena yang ditunda saat itu adalah syarat yang dikira memblokir SELURUH layar — menolak
// membuka klaim. Yang sebenarnya diatur hanyalah tab mana yang digambar, dan itu murni
// tampilan: ia tidak menolak siapa pun, tidak menyentuh data, dan tidak bertabrakan dengan
// `P-1`. Menegakkannya membuat layar ini LEBIH setara, bukan kurang.
func (d ClaimDetail) ShowsDocumentReceipt() bool {
	return strings.TrimSpace(d.Letter.TrackCode) != TrackHidden
}

// LetterDraft adalah bagian "Lampiran Surat" — bahan surat RCL/PUCL.
//
// # Tiga isiannya DITURUNKAN, bukan disimpan
//
// `Activity/SetDataLampiranSuratRCLPUCL_Act-Act.xml` mengisinya dari anak-anak klaim, dan
// ketiganya hanya diisi bila masih kosong (precondition `.<isian>==""`):
//
//	.UP            <- pyWorkPage.ClaimData.ObjectList(1).ObjectName
//	.NamaPeserta   <- pyWorkPage.ClaimData.ObjectList(1).ObjectName
//	.JumlahTagihan <- pyWorkPage.ClaimData.ObjectList(1).ObjectCoverageList(1).AdjustmentList(1).ProposeValue
//
// Perhatikan indeksnya SELALU `(1)` — objek pertama, coverage pertama, adjustment pertama.
// Klaim dengan banyak objek hanya membawa yang pertama ke suratnya, dan itu perilaku
// sistem lama apa adanya.
type LetterDraft struct {
	// Track adalah jalur penanganan — "RCL" atau "PUCL". Dari `RCL_PUCL_1`.
	Track string

	// TrackCode adalah kode jalur MENTAH.
	//
	// Ia dibawa selain Track karena nilai `3` menyembunyikan seluruh layar ini di Pega,
	// dan teks kosong dari `TrackOf` tidak dapat dibedakan dari kode yang memang kosong.
	// Lapisan atas memakainya untuk menjelaskan layar yang seharusnya tidak terbuka.
	TrackCode string

	// AnalystNote adalah "Deskripsi Analyst" — `KOMENTARANALISATOR_1`.
	AnalystNote string

	// PolicyNumber adalah `.Policy.PolicyNo` — `POLICYNO`.
	PolicyNumber string

	// LossDate adalah `.ClaimData.DateOfLoss` — `DATEOFLOSS_1`.
	LossDate string

	// InsuredName adalah "Nama Peserta" — DITURUNKAN dari objek pertama.
	InsuredName string

	// SumInsured adalah "UP" (Uang Pertanggungan) — DITURUNKAN, dan isinya NAMA OBJEK.
	//
	// # Ia diisi dari SUMBER YANG SAMA dengan InsuredName, dan itu MEMANG BENAR
	//
	// Kedua penetapan di `SetDataLampiranSuratRCLPUCL_Act` menunjuk ekspresi yang sama
	// persis: `pyWorkPage.ClaimData.ObjectList(1).ObjectName`. Kolom "UP" di layar surat
	// karena itu berisi nama objek, bukan angka.
	//
	// # Kenapa catatan ini panjang
	//
	// Karena ia terbaca seperti cacat, dan pernah diperlakukan sebagai cacat. Pada
	// 2026-09-24 ia sempat "diperbaiki" menjadi `SumTSI` pada coverage pertama — lengkap
	// dengan subkueri, uji, dan pernyataan selisih terencana. Work Owner **meralatnya pada
	// hari yang sama**: UP memang ObjectName.
	//
	// Perbaikan itu dicabut seluruhnya, dan `P-5` kembali berlaku apa adanya: perilaku
	// direplikasi kecuali perbaikannya diputuskan eksplisit — dan untuk yang ini TIDAK.
	//
	// Siapa pun yang hendak "memperbaikinya" lagi: nama isian ini menyesatkan, tetapi
	// isinya tidak. Yang bernama Uang Pertanggungan di sini bukan nilai pertanggungan.
	SumInsured string

	// BillAmount adalah "Jumlah Tagihan" — DITURUNKAN dari `PROPOSE_VALUE` adjustment
	// pertama pada coverage pertama objek pertama.
	BillAmount string

	// Keempat isian berikut MENUTUP empat dari sembilan isian yang dulu bertanda
	// "di clipboard Pega".
	//
	// # Kenapa mereka berpindah dari clipboard ke kolom
	//
	// Bukan karena pencarian yang lebih teliti. `TC_PNC_PUCL` yang berjalan memang punya
	// kolomnya — `PERIHAL`, `KETERANGAN1`, `KETERANGAN2`, `KETERANGAN3` — dan terisi.
	// Dibaca langsung 2026-10-01, dan isinya cocok kata demi kata dengan layar Pega.
	//
	// Keempatnya TIDAK ada di `Database/CREATE_TABLE_3.SQL`: berkas itu mendefinisikan 26
	// kolom, tabelnya punya 30. Lihat catatan pada kueri `detail`.

	// Subject adalah "Perihal" — `PERIHAL`.
	//
	// Di Pega ia PILIHAN dari master `POOLDATA.M_PERIHAL_RCLPUCL` (12 baris), digambar
	// kontrol `pxAutoComplete`. Yang tersimpan pada klaim adalah teksnya, bukan kodenya.
	Subject string

	// OpeningNote adalah "Keterangan Pembuka" — `KETERANGAN1`.
	OpeningNote string

	// BodyNote adalah "Keterangan Isi" — `KETERANGAN2`.
	BodyNote string

	// ClosingNote adalah "Keterangan Penutup" — `KETERANGAN3`.
	ClosingNote string
}

// DocumentReceipt adalah bagian "Penerimaan Dokumen".
//
// Hanya satu isiannya punya kolom yang diketahui. Sisanya — tanggal terima dokumen PUCL,
// email LOD, dan daftar berulang "Tanggal terima Dokumen / Tanggal / Keterangan" —
// TIDAK punya kolom yang dapat ditemukan di seluruh export.
//
// Isian itu tetap DIGAMBAR di layar, bukan dihilangkan: isian yang belum terbawa harus
// terlihat, bukan tersamar sebagai layar yang sudah setara. Preseden yang sama dipakai
// "Jumlah Lembar Dokumen" pada modul Inbox Manager Receive / PUCL.
type DocumentReceipt struct {
	// PUCLNote adalah `.ClaimData.PUCLStatus.KomentarPUCL` — `KOMENTARPUCL_1`.
	//
	// Judulnya di layar **"Catatan untuk Analyst"**, bukan "Komentar PUCL". Itu
	// `pyLabelFieldValue` pada selnya, dan `D-13` menetapkan teks layar dibawa apa adanya.
	//
	// Ia berpasangan dengan "Catatan dari Analyst" di bagian Lampiran Surat: yang satu
	// catatan Analyst untuk PUCL, yang satu balasan PUCL untuk Analyst. Menyamakan
	// keduanya akan menukar arah percakapannya.
	//
	// Satu-satunya isian bagian ini yang punya kolom terverifikasi; ia muncul di
	// `RDB List/ReminderPUCL-SQL.xml` sebagai `KOMENTARPUCL_1`.
	PUCLNote string
}

// TrackHidden adalah kode jalur yang menyembunyikan tab "Penerimaan Dokumen" di Pega.
//
// Ia sama dengan TrackCodeNotification; dua nama untuk satu kode, karena keduanya menyatakan
// hal yang berbeda — yang satu ARTI kodenya, yang satu AKIBATNYA pada layar.
//
// # Apa persisnya yang disembunyikan
//
// Satu tab, bukan seluruh layar. Syaratnya terpasang pada kontainer tab kedua di
// `Section/SendtoRCLPUCL-Section.xml`:
//
//	<pyContainerVisibleWhen>.ClaimData.PUCLStatus.RCL_PUCL != 3</pyContainerVisibleWhen>
//
// Lihat ClaimDetail.ShowsDocumentReceipt untuk penegakannya, dan untuk catatan mengapa
// pembacaan sebelumnya — "menyembunyikan seluruh layar" — keliru.
//
// # Kenapa masuk akal ia begitu
//
// Klaim ber-kode `3` bukan pekerjaan RCL maupun PUCL melainkan **pemberitahuan**, sehingga
// tidak ada dokumen yang ditunggu dan tidak ada yang dikirim kembali ke Analyst. Yang
// tersisa hanya suratnya.
const TrackHidden = TrackCodeNotification

// ErrClaimNotFound dikembalikan saat kunci klaim tidak ditemukan di portal yang dipilih.
//
// Ia dibedakan dari galat teknis dengan sengaja: kunci yang benar pada portal yang SALAH
// menghasilkan keadaan ini, dan itu keterangan yang harus sampai ke pengguna (`R-20`).
var ErrClaimNotFound = errors.New("inboxrclpucl: klaim tidak ditemukan")

// Repo adalah seam ke antrean RCL/PUCL pada SATU portal.
//
// Dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang memenuhinya. Diisi
// `repo/sqlstore` terhadap Oracle dan `repo/memory` untuk pengujian.
//
// # Tidak ada satu pun operasi yang menulis, dan itu keputusan, bukan kelalaian
//
// Layar lama punya dua tindakan yang menulis: mencetak surat PUCL/RCL — yang mengisi
// `TANGGALCETAKDOKUMENPUCL_1` sehingga klaimnya BERPINDAH dari tab 1 ke tab 2 — dan mengirim
// Reminder PUCL. Keduanya menyentuh tabel objek kerja dan tabel penugasan, dan keduanya
// masih dimiliki Pega selama masa paralel (`P-1`).
//
// Keputusan Work Owner 2026-09-23: tombolnya tetap DIGAMBAR, aksinya ditolak dengan alasan
// yang terbaca. Operasi yang tidak tersedia di seam ini karena itu tidak dapat dipakai kode
// yang ditulis kemudian tanpa keputusan sadar.
type Repo interface {
	// List mengembalikan SATU HALAMAN baris yang cocok beserta jumlah seluruhnya.
	//
	// Paginasi diserahkan ke pengisi seam, bukan dikerjakan pemanggil, supaya pengisi SQL
	// dapat memotongnya di basis data. Pengisi memori memakai Slice untuk hasil yang sama.
	List(ctx context.Context, query Query, page Pagination) (Page, error)

	// DailyReport mengembalikan SATU HALAMAN baris laporan harian pada rentang tanggal.
	//
	// Ia terpisah dari List karena kuerinya memang berbeda — lihat DailyReportRow. Ia
	// dipaginasi pula, meski laporan biasanya diambil sekaligus: ekspor menulis hasilnya
	// potong demi potong ke jawaban, dan menariknya sekaligus akan memaksa seluruh baris
	// berkumpul di memori lebih dulu.
	DailyReport(ctx context.Context, rng DateRange, page Pagination) ([]DailyReportRow, int, error)

	// Detail mengembalikan isi layar kerja RCL/PUCL untuk satu klaim.
	//
	// Kuncinya `pzInsKey` — parameter yang sama dengan `inskey` pada tautan Pega.
	//
	// Kunci yang tidak ditemukan menghasilkan ErrClaimNotFound, BUKAN nilai kosong: klaim
	// yang tidak ada dan klaim yang seluruh isiannya kosong terlihat sama di layar, dan
	// hanya yang pertama yang merupakan kekeliruan.
	Detail(ctx context.Context, reference string) (ClaimDetail, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal, bukan satu penyimpanan
//
// `ADR-0030` menetapkan satu database per entitas, bukan satu database bersama dengan
// penanda entitas. Klaim milik Asuransi Sinar Mas dan klaim milik Simas Insurtech karena itu
// tidak pernah berada di tabel yang sama.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal atau koneksinya belum hidup menghasilkan galat — TIDAK PERNAH
// dialihkan ke koneksi utama. Jatuh ke koneksi default berarti menampilkan klaim satu badan
// hukum kepada petugas badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
type RepoSelector func(portalAlias string) (Repo, error)
