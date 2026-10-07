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
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/platform/pagination"
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

	// StatusClaimAnalyst adalah Status Klaim yang IKUT ditulis kedua tombol Kirim.
	//
	// # Dari mana angkanya, dan kenapa ia bukan tebakan
	//
	// `Activity/PUCLPost-Act.xml` langkah 15, berprekondisi `param.Status==1`, berketerangan
	// **"jika KIRIM KE ANALYSt PUCLAPPROVE ke set 1"**, menetapkan DUA properti sekaligus:
	//
	//	.ClaimData.PUCLStatus.PUCLApprove  :=  1
	//	.ClaimData.StatusClaim             :=  "1151"
	//
	// Artinya terbaca langsung dari master `POOLDATA.V_STS_CLAIM`: **`1151` = "Analyst"**.
	// Jadi tombolnya tidak hanya mengeluarkan klaim dari antrean PUCL — ia MENYATAKAN klaim
	// itu kini berada di tangan Analyst.
	//
	// # Kenapa ini sempat terlewat
	//
	// Karena penandanya dicari dari arah yang salah: dari kueri inbox, yang hanya menyaring
	// `PUCL_APPROVE`. Kolom yang TIDAK dipakai menyaring apa pun karena itu tidak terlihat —
	// padahal ia yang menjawab "klaim ini sekarang di mana" pada laporan harian.
	//
	// # Berlaku untuk KEDUA tombol Kirim
	//
	// Langkah 15 berprekondisi `param.Status==1`, dan "Kirim ke PIC Teknik" mengirim `Status`
	// yang sama. Jadi klaim yang dikirim ke PIC Teknik pun berstatus "Analyst". Itu terbaca
	// janggal, tetapi itulah yang dikerjakan sistem lama — prekondisinya memang pada
	// `Status`, bukan pada tombolnya (`P-5`).
	StatusClaimAnalyst = "1151"

	// StatusCasePrinted dan StatusClaimWaitingDocument ditulis tombol "Download Dokumen".
	//
	// `Activity/PUCLPost-Act.xml` langkah 17, berprekondisi `param.statusCase==1`,
	// berketerangan **"jika sudah DOWNLOAD DOKUMEN set STATUSCase = 1"**:
	//
	//	.ClaimData.PUCLStatus.StatusCase  :=  param.statusCase   // "1"
	//	.ClaimData.StatusClaim            :=  "1157"
	//
	// `1157` berarti **"Document Waiting RCL/PUCL"** menurut master `POOLDATA.V_STS_CLAIM` —
	// klaimnya menunggu kelengkapan dokumen. Itu tepat menggambarkan tahap sesudah suratnya
	// dicetak.
	StatusCasePrinted          = "1"
	StatusClaimWaitingDocument = "1157"

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
//
// Page dimulai dari 1.
// Size adalah jumlah baris per halaman.
type Pagination = pagination.Request[pageSizes]

// pageSizes membawa ukuran halaman layar ini ke tipe generik pagination.
type pageSizes struct{}

func (pageSizes) Default() int { return DefaultPageSize }
func (pageSizes) Max() int     { return MaxPageSize }

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

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
//
// Total adalah jumlah SELURUH baris yang cocok di penyimpanan, bukan yang tampil.
// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan, bukan yang
// diminta. Layar menggambar penomoran halamannya dari sini.
type Page = pagination.Page[WorkItem, pageSizes]

// Slice memotong satu halaman dari seluruh baris yang sudah di tangan.
//
// Ia dipakai penyimpanan MEMORI saja. Penyimpanan SQL memotongnya di basis data dengan
// `OFFSET … FETCH NEXT`, dan perbedaan itu disengaja — lihat catatan paginasi di kepala
// repo/sqlstore/inboxrclpucl.sql.
//
// Keduanya tetap menghasilkan Page dengan arti yang sama, sehingga uji aturan modul yang
// berjalan di atas memori menyatakan hal yang benar tentang yang berjalan di Oracle.
func Slice(all []WorkItem, page Pagination) Page { return pagination.Slice(all, page) }

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

	// ActionParameters adalah parameter tersembunyi yang dikirim ke `PUCLPost`.
	ActionParameters ActionParameters

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

	// InsuredParty adalah penerima surat — baris "Kepada Yth." pada `SuratPUCL`.
	//
	// Di Pega ia dirakit dari dua properti polis: `Customer_C.pyCompany` bila tertanggungnya
	// badan hukum (digambar berawalan "PT."), atau `Customer_P.pyFirstName` bila perorangan.
	// Snapshot polis belum terbawa modul ini, sehingga yang dipakai `TC_PNC_PUCL.QQ_NAME` —
	// nama tertanggung pada klaim itu sendiri.
	//
	// Awalan "PT." karena itu TIDAK ditambahkan: tanpa properti pembedanya, menambahkannya
	// berarti menyebut perorangan sebagai badan hukum pada surat yang keluar ke cabang.
	InsuredParty string
}

// LetterDocument adalah isi surat RCL/PUCL yang hendak dicetak.
//
// # Kenapa ia tipe tersendiri, bukan LetterDraft apa adanya
//
// Karena keduanya menjawab pertanyaan berbeda. `LetterDraft` adalah APA YANG TERSIMPAN pada
// klaim; ini adalah APA YANG TERCETAK — termasuk tanggal dan nomor surat yang baru lahir saat
// tombolnya ditekan, dan tidak tersimpan di mana pun sebelumnya.
//
// Memakai satu tipe untuk keduanya membuat perender bergantung pada isian yang tidak
// dicetaknya, dan membuat layar kerja membawa isian yang hanya berarti saat mencetak.
type LetterDocument struct {
	// LetterDate adalah tanggal surat — "dd MMMM yyyy" dalam WIB.
	LetterDate string

	// LetterNumber adalah nomor surat, dirakit dari waktu sistem. Lihat NewLetterNumber.
	LetterNumber string

	// Recipient dan RecipientAddress adalah blok "Kepada Yth.".
	Recipient        string
	RecipientAddress string

	// Ketujuh berikut adalah tabel pertama surat, berurutan seperti di templat.
	SumInsured     string
	ContractNumber string
	PolicyNumber   string
	BusinessUnit   string
	InsuredName    string
	BillCurrency   string
	BillAmount     string
	LossDate       string

	// Subject dan ketiga keterangan adalah badan suratnya.
	Subject     string
	OpeningNote string
	BodyNote    string
	ClosingNote string

	// Nama kedua penanda tangan. Kosong untuk sekarang — lihat suratpdf.
	SignerLeftName  string
	SignerRightName string
}

// Nama berkas dan kategori surat yang diterbitkan, mengikuti Pega apa adanya.
//
// `PUCLPost` langkah 20–22 menetapkan `param.PDFName` dari JALUR klaim, dan ketiganya berbeda:
//
//	langkah 20  IsPA && RCL_PUCL==2   "PUCL"+".pdf"
//	langkah 21  IsPA && RCL_PUCL==1   "RCL"+".pdf"
//	langkah 22  IsPA && RCL_PUCL==3   "Notification"+".pdf"   — dan MSIG := "MSIG"
//
// Ketiganya memakai templat yang SAMA (`param.HTMLStream := "SuratPUCL"`); yang berbeda hanya
// nama berkasnya. Jalur Travel (langkah 23–24) memakai templat `SuratPUCL_TRAVEL` dengan nama
// berkas yang sama persis.
//
// Seluruhnya dilampirkan berkategori `Notification`.
const (
	LetterFileNamePUCL = "PUCL.pdf"
	LetterFileNameRCL  = "RCL.pdf"

	// LetterFileNameNotification — jalur ketiga, `RCL_PUCL = '3'`.
	//
	// Namanya kebetulan sama dengan LetterCategory, dan keduanya TIDAK disatukan: yang satu
	// nama berkas, yang satu kategori lampiran. Menyatukannya membuat perubahan pada salah
	// satu diam-diam mengubah yang lain.
	LetterFileNameNotification = "Notification.pdf"

	LetterCategory = "Notification"

	// DefaultAttachmentCategory adalah kategori lampiran bawaan Pega.
	//
	// Dipakai ketika petugas tidak memilih apa pun. Lihat Repo.AddDocument.
	DefaultAttachmentCategory = "File"
)

// namaBulan adalah nama bulan Indonesia untuk tanggal surat.
//
// Pega memakai `@CurrentDate("MMMM","WIB")`, yang mengikuti locale JVM-nya. Suratnya
// berbahasa Indonesia seluruhnya, jadi nama bulannya pun. Bila ternyata JVM produksi
// berlocale Inggris, inilah satu-satunya tempat yang perlu berubah.
var namaBulan = [...]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// Alur Register — nilai yang WAJIB sama persis dengan modul `registrasi`.
//
// Tugas yang ditulis modul ini dibaca inbox modul lain, dan keduanya mencocokkan kolom
// `TAHAP` sebagai teks. Satu huruf yang berbeda membuat tugasnya tersimpan dengan baik dan
// TIDAK PERNAH muncul di inbox mana pun — kegagalan yang tidak menghasilkan satu pun galat.
//
// Nilainya disalin dari `registrasi/flow.go`, dan sengaja TIDAK diimpor dari sana: tidak satu
// pun modul di aplikasi ini mengimpor modul lain, dan membuka pengecualian untuk tiga
// konstanta akan menautkan dua modul yang selama ini berdiri sendiri.
const (
	// StageRCLPUCL — `Assignment6`, tahap yang klaimnya sedang dikerjakan layar ini.
	StageRCLPUCL = "rcl-pucl"

	// StageSendToAnalyst — `Assignment5`, tujuan ticket `SendtoAnalysator`.
	StageSendToAnalyst = "kirim-analis"

	// QueueWorklist — tahap Send To Analis dipegang SATU orang, bukan antrean bersama.
	QueueWorklist = "WORKLIST"

	// TicketSendToAnalyst adalah nama ticket yang dilepas `PUCLPost` langkah 17.
	//
	// Ia dicatat sebagai alasan penutupan tugas lama, bukan dipakai sebagai penyaring —
	// supaya riwayat tugas menyebut APA yang memindahkannya, dan jejaknya dapat dilacak
	// kembali ke rule Pega yang sama.
	TicketSendToAnalyst = "SendtoAnalysator"
)

// Teks riwayat yang ditulis tiap tombol — parameter `statusNote` `InsertHistoryClaimPNC`.
//
// Ketiganya disalin APA ADANYA dari Pega, termasuk yang terbaca tidak rapi:
//
//   - spasi di ujung "Wait for Complete PUCL Document " ADA di Pega dan dipertahankan
//     (`P-5`). Ia terbawa ke kolom riwayat, dan membuangnya mengubah data yang tersimpan.
//   - "Send by PUCL to Analyst" berawalan huruf BESAR, "send by PUCL to PIC Teknis" huruf
//     kecil. Menyeragamkannya terlihat seperti kerapian dan sebenarnya mengubah data.
//
// Sumbernya berbeda, dan itu sebab perbedaan gayanya: yang pertama dan ketiga dari rangkaian
// TOMBOL di section, yang kedua dari `<statusNote>` `PUCLPost` langkah 35.
const (
	HistoryNotePrintLetter   = "Wait for Complete PUCL Document "
	HistoryNoteSendToAnalyst = "Send by PUCL to Analyst"
	HistoryNoteSendToPIC     = "send by PUCL to PIC Teknis"
)

// HistoryNoteFor memilih teks riwayat satu tindakan. Kosong berarti tindakan itu TIDAK
// menulis riwayat — "Tolak Klaim" dan "Save" memang tidak.
//
// Ia di paket domain, bukan di adapter, karena DUA jalur memakainya: jalur yang ditangani
// sendiri dan jalur yang menempuh layanan Pega. Dua salinan akan dapat berselisih, dan
// selisihnya baru terlihat sebagai dua baris riwayat berbeda untuk tombol yang sama.
func HistoryNoteFor(kind ClaimActionKind) string {
	switch kind {
	case ActionPrintLetter:
		return HistoryNotePrintLetter
	case ActionSendToAnalyst:
		return HistoryNoteSendToAnalyst
	case ActionSendToPICTeknik:
		return HistoryNoteSendToPIC
	default:
		return ""
	}
}

// LetterFileNameFor memilih nama berkas surat menurut jalur klaimnya.
//
// Jalur yang TIDAK dikenali — termasuk satu baris produksi yang kodenya kosong — jatuh ke
// `PUCL.pdf`. Itu bukan pilihan sembarang: layar ini adalah antrean PUCL, dan berkas bernama
// sesuatu lebih berguna daripada tindakan yang gagal karena jalurnya tidak terbaca.
func LetterFileNameFor(track string) string {
	// Dibandingkan tanpa memandang besar-kecil huruf, seperti bentuk sebelumnya: nilainya
	// berasal dari TrackOf yang sudah baku, tetapi pemanggil baru dapat saja memberi teks
	// yang diketik.
	switch {
	case strings.EqualFold(strings.TrimSpace(track), TrackRCL):
		return LetterFileNameRCL
	case strings.EqualFold(strings.TrimSpace(track), TrackNotification):
		return LetterFileNameNotification
	default:
		return LetterFileNamePUCL
	}
}

// NewLetterDate menggambar tanggal surat — "dd MMMM yyyy".
func NewLetterDate(now time.Time) string {
	t := now.In(wib)
	return fmt.Sprintf("%02d %s %04d", t.Day(), namaBulan[int(t.Month())-1], t.Year())
}

// NewLetterNumber merakit nomor surat.
//
// # Bentuknya ditiru PERSIS, termasuk yang terbaca aneh
//
// `PUCLPost` merakitnya dari sembilan penetapan properti bernama menyesatkan — `BankCIF`,
// `BLNumber`, `BookNo`, `CoverNo`, `AccountNo`, `CedingCo` — yang seluruhnya berisi potongan
// WAKTU SISTEM, bukan data bank maupun reasuransi:
//
//	{hh}{mm}{ss}/{kategori}.CL.AHID.ASM/{MM}/{yyyy}
//
// Jamnya **12 jam**, bukan 24: Pega memakai `@CurrentDate("hh","WIB")`, dan `hh` pada format
// Java adalah jam 01–12. Surat yang terbit pukul 14.05 karena itu bernomor berawalan `0205`.
// Itu ditiru apa adanya (`P-5`), bukan "diperbaiki" menjadi 24 jam.
//
// Akibatnya nomor surat TIDAK unik: dua surat berjarak 12 jam tepat pada bulan yang sama
// menghasilkan nomor yang sama persis. Dicatat, tidak diubah.
func NewLetterNumber(now time.Time, category string) string {
	t := now.In(wib)

	jam := t.Hour() % 12
	if jam == 0 {
		jam = 12
	}

	return fmt.Sprintf("%02d%02d%02d/%s.CL.AHID.ASM/%02d/%04d",
		jam, t.Minute(), t.Second(),
		strings.TrimSpace(category),
		int(t.Month()), t.Year())
}

// LetterRenderer membentuk PDF surat RCL/PUCL.
//
// Seam, bukan pemanggilan langsung: domain menyatakan APA yang dicetak, dan bagaimana ia
// menjadi PDF adalah urusan adapter. Uji aturan surat karena itu dapat berjalan tanpa
// membentuk satu berkas pun.
type LetterRenderer interface {
	Render(LetterDocument) ([]byte, error)
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

	// CompleteAt adalah **"Tanggal Kelengkapan Dokumen"** — isian WAJIB di section.
	//
	// Properti section-nya `TanggalTerimaDokumenPUCL`, dan kolomnya
	// `POOLDATA.TC_PNC_PUCL.TGL_TERIMA_DOKUMEN_PUCL` (`TIMESTAMP(6)`). Ditetapkan Work Owner
	// 2026-10-01; sebelumnya isian ini digambar bertanda "di clipboard Pega".
	//
	// Sudah berbentuk tampilan WIB, sama seperti seluruh tanggal modul ini.
	CompleteAt string

	// InsuredEmail adalah **"Email Tertanggung"**.
	//
	// Properti section-nya `EmailLod`, dan satu-satunya isian layar kerja yang diambil dari
	// LUAR tabel datar: `POOLDATA.T_CLAIM_PNC.EMAIL_LOD`, digabung lewat nomor case.
	//
	// # Kolomnya ADA tetapi KOSONG di seluruh baris
	//
	// Dihitung langsung 2026-10-01: `EMAIL_LOD` terisi pada **0 dari 2.206** baris
	// `T_CLAIM_PNC`. Jadi isian ini akan tergambar kosong, dan itu bukan kekeliruan
	// penyambungan — datanya memang belum pernah diisi.
	//
	// Dinyatakan di sini supaya kekosongannya tidak dilaporkan sebagai cacat kode, dan supaya
	// penelusurannya mengarah ke proses yang MENGISI kolom itu.
	InsuredEmail string

	// ReceivedDates adalah grid **"Tanggal Terima Dokumen"** — kolom Tanggal dan Keterangan.
	//
	// # Paling banyak SATU baris, dan itu batas DATA, bukan batas rancangan
	//
	// Grid-nya terikat page list `.ClaimData.PUCLStatus.DateReceivedDocument` berkelas
	// `ASM-FW-GCNMFW-Data-DateReceivedDocument`. Katalog Oracle dicari 2026-10-01: tidak ada
	// tabel bernama mirip itu, dan tidak ada satu pun tabel yang memuat `DATERECEIVED`
	// bersama `REMARKS`. Daftarnya memang hidup di dalam blob objek kerja.
	//
	// Yang terbaca hanyalah BARIS PERTAMANYA, lewat dua kolom hasil ekspos pada tabel objek
	// kerja — pola akhiran `_1` yang sama dengan `RCL_PUCL_1` dan `MSIG_1`:
	//
	//	RECEIVEDDATE_1  terisi 4.156 dari 7.723 · panjang maksimum 23
	//	KETERANGAN_1    terisi    85            · panjang maksimum 32
	//
	// Bentuknya tetap SENARAI meski isinya paling banyak satu, karena itulah bentuk gridnya
	// di layar lama. Begitu baris kedua dan seterusnya terbaca — lewat layanan Pega pada
	// `permintaan-artefak-pega.md` §12 — isinya bertambah tanpa mengubah kontrak.
	ReceivedDates []ReceivedDocumentDate
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

// ErrTechnicalPICUnknown dikembalikan saat klaim hendak dipindahkan ke tahap Send To Analis
// tetapi PIC Tekniknya tidak diketahui.
//
// # Kenapa MENOLAK, bukan membuat tugas tanpa pemilik
//
// Karena tahap Send To Analis adalah Worklist, dan tugas Worklist wajib bertuan sejak lahir
// (`D-26`). Tugas tanpa pemilik pada antrean yang bukan antrean bersama tidak akan muncul di
// inbox siapa pun — klaimnya hilang dari semua layar tanpa satu pun galat.
//
// Menolak membuat sebabnya terbaca, dan klaimnya tetap di antrean PUCL tempat ia sekarang.
var ErrTechnicalPICUnknown = errors.New(
	"inboxrclpucl: PIC Teknik klaim tidak diketahui")

// ErrDocumentNotFound dikembalikan saat dokumen tidak ada, ATAU ada tetapi bukan milik klaim
// yang diminta.
//
// Kedua keadaan itu sengaja TIDAK dibedakan. Membedakannya akan memberi tahu pemanggil bahwa
// sebuah id dokumen memang ada — keterangan yang tidak dibutuhkan siapa pun yang berhak, dan
// berguna justru bagi yang tidak.
var ErrDocumentNotFound = errors.New("inboxrclpucl: dokumen tidak ditemukan")

// ActionParameters adalah parameter TERSEMBUNYI yang dikirim layar ke `PUCLPost`.
//
// Ketiganya sel tanpa label di `SectionPenerimaanDokumenPUCL` —
// `.ClaimData.PUCLStatus.IDObject`, `.IDCoverage`, dan `.IDAdjustment` — dan ketiganya ikut
// pada setiap tombol yang memanggil `PUCLPost`: Download Dokumen, Tolak Klaim, dan kedua
// tombol Kirim.
//
// Ia TIDAK digambar di layar, sama seperti di Pega. Yang membawanya ke sini adalah jalur
// tulis yang sedang disiapkan — lihat `permintaan-artefak-pega.md` §12.
//
// # Nilai penampung sudah DICABUT
//
// Pada 2026-10-01 ketiganya sempat bernilai tetap `"1"` karena kolomnya belum ada. Work Owner
// menambahkan `ID_OBJECT`, `ID_COVERAGE`, dan `ID_ADJUSTMENT` ke `POOLDATA.TC_PNC_PUCL` pada
// hari yang sama, dan sejak itu ketiganya dibaca dari kolomnya. Konstanta penampung beserta
// pembentuk dan uji penjaganya dihapus — bukan disisakan bernilai `false`.
type ActionParameters struct {
	// IDObject — `.ClaimData.PUCLStatus.IDObject`, kolom `ID_OBJECT`.
	IDObject string

	// IDCoverage — `.ClaimData.PUCLStatus.IDCoverage`, kolom `ID_COVERAGE`.
	IDCoverage string

	// IDAdjustment — `.ClaimData.PUCLStatus.IDAdjustment`, kolom `ID_ADJUSTMENT`.
	IDAdjustment string
}

// ReceivedDocumentDate adalah satu baris grid "Tanggal Terima Dokumen".
//
// Kedua isiannya adalah kolom grid di `SectionPenerimaanDokumenPUCL`: `.DateReceived` dengan
// judul **"Tanggal"**, dan `.Remarks` dengan judul **"Keterangan"**.
type ReceivedDocumentDate struct {
	// Date sudah berbentuk tampilan WIB.
	Date string

	// Note adalah kolom "Keterangan". Ia sering kosong — kolomnya terisi pada 85 dari 7.723
	// baris — dan barisnya tetap digambar selama tanggalnya ada.
	Note string
}

// ReceivedDatesOf menyusun grid "Tanggal Terima Dokumen" dari baris pertamanya.
//
// Baris yang tanggalnya KOSONG tidak dibentuk sama sekali, meski keterangannya terisi: grid
// tanpa baris berarti daftarnya memang kosong, sedangkan satu baris bertanggal kosong terbaca
// sebagai entri yang ada tetapi belum diisi. Keduanya menuntut tindakan yang berbeda.
func ReceivedDatesOf(date, note string) []ReceivedDocumentDate {
	if strings.TrimSpace(date) == "" {
		return nil
	}
	return []ReceivedDocumentDate{{Date: DisplayPegaTime(date), Note: note}}
}

// pegaTimeLayouts adalah bentuk waktu yang DISIMPAN PEGA sebagai teks.
//
// `RECEIVEDDATE_1` bukan kolom TIMESTAMP melainkan VARCHAR2 berisi bentuk internal Pega —
// `20240911T143500.000 GMT`. Ia sampai ke layar APA ADANYA sampai 2026-10-01, dan terbaca
// sebagai kerusakan oleh siapa pun yang melihatnya.
var pegaTimeLayouts = []string{
	"20060102T150405.000 GMT",
	"20060102T150405 GMT",
	"20060102T150405.000Z",
	"20060102T150405Z",
}

// wib adalah zona tampilan. Seluruh waktu Pega disimpan GMT dan DIGAMBAR +7 — lihat `F-5`
// dan utang teknis 4.4 pada `03-CURRENT-ARCHITECTURE`.
//
// Pergeserannya ditulis TETAP, bukan lewat time.LoadLocation, karena basis data zona waktu
// tidak selalu tersedia pada peladen Windows — dan kegagalannya diam: time.LoadLocation
// mengembalikan UTC, sehingga seluruh jam tergambar tujuh jam lebih awal tanpa satu pun galat.
var wib = time.FixedZone("WIB", 7*60*60)

// DisplayPegaTime mengubah waktu bentuk Pega menjadi bentuk yang dibaca petugas.
//
// Keluarannya `dd/MM/yyyy HH:mm` dalam WIB — bentuk yang sama persis dengan layar lama
// (`D-13`), sehingga kedua layar dapat dibandingkan berdampingan tanpa menghitung sendiri.
//
// Nilai yang TIDAK dikenali dikembalikan APA ADANYA, bukan dikosongkan. Teks yang tidak
// terbaca mesin tetap dapat dibaca manusia dan tetap dapat dilaporkan; mengosongkannya
// menghapus satu-satunya petunjuk bahwa ada bentuk yang belum ditangani.
func DisplayPegaTime(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ""
	}
	for _, layout := range pegaTimeLayouts {
		at, err := time.Parse(layout, clean)
		if err != nil {
			continue
		}
		return at.In(wib).Format("02/01/2006 15:04")
	}
	return clean
}

// Document adalah satu baris daftar dokumen klaim — layar "Lihat Dokumen".
//
// # Kolomnya mengikuti dokumen yang BISA dibaca, bukan grid Pega
//
// `Section/ViewAttachmentPUCLDetail-Section.xml` menggambar grid dari Report Definition
// `GCNMGetAllAttachments` pada kelas `Link-Attachment`. Jalur itu bermuara di
// `PC_DATA_WORKATTACH.PZPVSTREAM`, blob serialisasi Pega — bukan berkas yang dapat
// diserahkan apa adanya.
//
// Yang dipakai `POOLDATA.DATA_ATTACHFILE`, yang menyimpan berkas MENTAH dan tertaut ke objek
// kerja lewat `IDPEGA`. Pilihan itu diambil dari pembacaan katalog dan data langsung pada
// 2026-10-01, bukan dari membaca section — lihat kueri `documents`.
type Document struct {
	// ID adalah `DATAID`. Ia dipakai sebagai bagian alamat pengambilan isinya.
	ID string

	// Name adalah nama berkas — `ATTACHNAME`.
	Name string

	// MimeType adalah `ATTACHMIMETYPE`. Tidak digambar; ia menentukan cara berkasnya
	// diserahkan.
	MimeType string

	// Category dan SubCategory adalah nama jenis dokumen, dicari ke master. Bila kodenya
	// tidak ada di master, KODENYA yang dibawa — bukan kosong.
	Category    string
	SubCategory string

	// UploadedAt sudah berbentuk tampilan WIB, sama seperti seluruh tanggal modul ini.
	UploadedAt string

	// UploadedBy adalah `INPUTOPERATOR`.
	UploadedBy string

	// PegaVisible menyatakan apakah barisnya TERGAMBAR di layar lampiran Pega.
	//
	// `GCNMGetAllAttachments` — report definition di balik "Lihat Dokumen" — berjalan di
	// kelas `Link-Attachment` dan menyaring atas `pyCategory`, yang isinya NAMA kategori
	// lampiran. Baris yang `CATEGORY`-nya kode angka berasal dari mekanisme lain dan tidak
	// pernah muncul di sana.
	//
	// Dibawa sebagai penanda, bukan disaring di dalam kueri, supaya lapisan atas dapat
	// MENGHITUNG yang tidak tergambar lalu menyatakannya. Daftar yang diam-diam lebih
	// pendek adalah kegagalan yang tidak menghasilkan satu pun galat.
	PegaVisible bool
}

// DocumentContent adalah isi satu dokumen beserta keterangan penyerahannya.
type DocumentContent struct {
	Name     string
	MimeType string

	// Content adalah isi berkasnya, apa adanya dari kolom BLOB.
	//
	// Tidak dibungkus base64: pembungkusan itu memanggil `pooldata.base64encode` yang `D-02`
	// larang, dan membesarkan muatan sepertiga tanpa satu pun manfaat — berkasnya diserahkan
	// ke peramban sebagai berkas, bukan sebagai teks di dalam JSON.
	Content []byte
}

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

	// Documents mengembalikan dokumen yang terlampir pada satu klaim.
	//
	// Ia melayani tombol "Lihat Dokumen" — satu-satunya tombol layar kerja yang MEMBACA,
	// sehingga ia satu-satunya yang dapat dibangun tanpa menunggu keputusan `P-1`.
	//
	// Klaim tanpa dokumen mengembalikan senarai kosong tanpa galat; itu keadaan yang wajar.
	Documents(ctx context.Context, caseNumber string) ([]Document, error)

	// DocumentContent mengembalikan isi SATU dokumen.
	//
	// Kepemilikannya diperiksa di dalam kueri — dokumennya wajib terbukti milik klaim yang
	// diminta. Id dokumen dapat ditebak, dan pemeriksaan yang berada di luar kueri dapat
	// terlewat oleh pemanggil baru.
	//
	// Id yang tidak ditemukan ATAU bukan milik klaim itu sama-sama menghasilkan
	// ErrDocumentNotFound — keduanya tidak dibedakan, supaya jawaban tidak memberi tahu
	// bahwa sebuah id ada tetapi milik klaim lain.
	DocumentContent(ctx context.Context, caseNumber, documentID string) (DocumentContent, error)

	// ReturnToAnalyst menandai klaim SUDAH SELESAI dikerjakan PUCL.
	//
	// Ia menulis `PUCLAPPROVE_1 = PUCLReturnedToAnalyst` pada `POOLDATA.TC_PNC_PUCL`, dan
	// itulah satu-satunya hal yang mengeluarkan klaim dari tab 2 dan 3 layar ini.
	//
	// # Kenapa ini BUKAN pelanggaran `P-1`
	//
	// `TC_PNC_PUCL` adalah tabel milik APLIKASI INI, bukan tabel engine Pega — lihat kepala
	// `inboxrclpucl.sql`. Tidak ada tabel `DATAPEGA` yang disentuh.
	//
	// # Kenapa klaimnya TIDAK hilang tanpa sampai ke siapa pun
	//
	// Kekhawatiran itu pernah ditulis di tempat ini, dan ia **gugur oleh data**. Klaim RCL/PUCL
	// sudah memegang baris `PC_ASSIGN_WORKLIST` Register_Flow-nya sendiri, dan pemegang baris
	// itu adalah PIC Teknik klaim tersebut — `PXASSIGNEDOPERATORID` sama persis dengan
	// `USERTEKNIS_1` pada objek kerjanya. Menandai klaim selesai di PUCL karena itu
	// MENGEMBALIKANNYA kepada PIC Teknik yang memang sudah memegangnya, bukan melenyapkannya.
	//
	// Work Owner menegaskan tujuan itu pada 2026-10-01: *"balik ke PIC Teknik"*. Arti
	// nilainya sendiri sudah ditetapkan 2026-09-30 — lihat PUCLReturnedToAnalyst.
	//
	// # Yang TIDAK dikerjakannya
	//
	// Baris `PC_ASSIGN_WORKBASKET` milik antrean `RCLPUCL` **dibiarkan**. Menghapusnya adalah
	// tindakan yang tidak dapat dipulihkan atas tabel Pega, dan tidak dibutuhkan agar layar
	// ini benar. Akibatnya disadari dan dicatat: selama masa paralel, klaimnya masih terlihat
	// di antrean RCL/PUCL milik Pega sampai Pega sendiri menyelesaikannya.
	//
	// Kunci yang tidak ditemukan menghasilkan ErrClaimNotFound.
	ReturnToAnalyst(ctx context.Context, reference, caller string) error

	// RecordHistory menulis SATU baris riwayat klaim — `InsertHistoryClaimPNC`.
	//
	// # Kenapa ia akhirnya dapat dikerjakan
	//
	// Catatan sebelumnya menyatakan riwayat tidak dapat ditulis karena "modul ini tidak punya
	// tabel riwayat klaim". **Itu keliru**, dan terbukti keliru saat rangkaian activity-nya
	// ditelusuri 2026-10-02: `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc` memuat satu
	// pernyataan, dan tabelnya ada:
	//
	//	INSERT INTO LIST_HISTORY_CLAIM_PNC (CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE)
	//	VALUES (CaseID, CURRENT_TIMESTAMP, StatusNote, UserUpdate);
	//
	// Ia tabel bisnis `POOLDATA`, bukan tabel engine Pega — sehingga `P-1` tidak dilanggar.
	//
	// # Nama parameter di rule Pega MENYESATKAN
	//
	// `RDB List/InsertHistoryClaimPNC-SQL.xml` memanggilnya dengan nama properti clipboard
	// `POLICY_NO`, `BUSINESS_CODE`, `BRANCH_CODE`, `BRANCH_NAME out` — dan tidak satu pun
	// berarti apa yang namanya katakan. Pemetaannya POSISIONAL:
	//
	//	posisi 1  POLICY_NO     -> CaseID
	//	posisi 2  BUSINESS_CODE -> StatusNote
	//	posisi 3  BRANCH_CODE   -> UserUpdate
	//	posisi 4  BRANCH_NAME   -> ErrMsg (keluaran)
	//
	// Membaca namanya dan bukan procedure-nya akan menulis nomor polis ke kolom CASEID.
	// Ini contoh lain dari alias menyesatkan yang `D-19` tetapkan untuk tidak dibawa.
	//
	// # Yang ditulis sebagai CaseID
	//
	// `pzInsKey` objek kerja, bukan nomor klaim — `PUCLPost` langkah 35 mengirim
	// `caseID = pyWorkPage.pzInsKey`. Bentuknya `ASM-FW-GCNMFW-WORK PNC-xxxx`, dan menulis
	// nomor telanjang akan membuat barisnya tidak sebaris dengan yang ditulis Pega.
	//
	// Kegagalannya TIDAK membatalkan tindakan pemanggil: riwayat yang gagal ditulis tidak
	// menghalangi klaim berpindah, dan menggagalkan seluruh tindakan karenanya akan membuat
	// petugas menekan tombolnya lagi — tanpa akibat, karena penandaannya sudah terjadi.
	RecordHistory(ctx context.Context, reference, statusNote, caller string) error

	// MoveToSendToAnalyst memindahkan klaim ke tahap **Send To Analis** pada alur Register.
	//
	// # Inilah yang benar-benar MEMINDAHKAN klaim
	//
	// Ketiga penulisan lain — `PUCL_APPROVE`, `STATUS_CLAIM`, riwayat — hanya mencatat
	// keadaan. Yang membuat klaim berhenti menjadi pekerjaan PUCL dan mulai menjadi
	// pekerjaan PIC Teknik adalah TUGASNYA, dan tugas hidup di `POOLDATA.CPNC_TUGAS`.
	//
	// # Ia meniru `SetTicket`, bukan konektor alur
	//
	// `PUCLPost` langkah 17 melepas ticket `SendtoAnalysator`, dan ticket itu menempel pada
	// shape `Send To Analis` (`Assignment5`). Itu **lompatan lateral** — klaim berpindah ke
	// tahap yang bukan tahap berikutnya menurut konektor.
	//
	// Konektor keluar shape RCL/PUCL sendiri menuju akhir flow, dan ia TIDAK PERNAH
	// dievaluasi pada jalur ini: ticket sudah melompat lebih dulu. Itu sebabnya tombolnya
	// meneruskan klaim alih-alih menutupnya.
	//
	// # Siapa pemiliknya
	//
	// PIC Teknik klaim itu sendiri, dibaca dari **`POOLDATA.T_CLAIM_PNC.PICTEKNIK`**.
	//
	// Tahap ini dirutekan `PNCTeknikRouter`, yang tidak ada di export (`R-04`) — tetapi
	// akibatnya terukur: dari 338 baris tahap teknis, pemegang tugasnya sama dengan PIC
	// Teknik klaimnya pada 320.
	//
	// Kolomnya `PICTEKNIK`, BUKAN `USERTEKNIS_1`. Yang kedua adalah salinan yang Pega ekspos
	// dari properti clipboard `ClaimData.UserTeknis`, dan export menunjukkan asalnya apa
	// adanya: `PICTEKNIK AS "UserTeknis"`. Membaca salinan berarti bergantung pada Pega
	// sempat menuliskannya — dan klaim yang dibuka aplikasi ini tidak melewati Pega.
	//
	// Pengisi seam membacanya sendiri; ia tidak diteruskan dari layar, karena nilai yang
	// disusun pemanggil dapat disusun siapa pun.
	//
	// # Dua tombol, SATU tujuan
	//
	// "Kirim Ke Analyst" dan "Kirim ke PIC Teknik" sama-sama menempuh `PUCLPost` dengan
	// `Status = 1`, dan prekondisi langkah 17 hanya menguji itu. Keduanya karena itu melepas
	// ticket yang sama dan berakhir di tahap yang sama.
	//
	// Klaim yang tidak punya tugas terbuka TETAP memperoleh tugas baru — bukan galat. Itu
	// keadaan yang wajar selama masa paralel: klaim yang dimulai di Pega belum pernah punya
	// tugas di tabel ini.
	MoveToSendToAnalyst(ctx context.Context, reference, caller string) error

	// SaveReceipt menyimpan kedua isian Penerimaan Dokumen yang DAPAT DIKETIK petugas.
	//
	// Ia melayani tombol "Save". Lihat ReceiptInput untuk apa yang disimpan dan kenapa hanya
	// dua, serta Repo.ReturnToAnalyst untuk alasan `P-1` tidak dilanggar.
	//
	// Kunci yang tidak ditemukan menghasilkan ErrClaimNotFound.
	SaveReceipt(ctx context.Context, reference string, in ReceiptInput, caller string) error

	// AddDocument melampirkan satu berkas ke klaim — tombol "Unggah Dokumen".
	//
	// Mengembalikan baris dokumen yang baru tersimpan, supaya pemanggil dapat menggambarnya
	// tanpa menarik ulang seluruh daftar.
	//
	// Kunci klaim yang tidak ditemukan menghasilkan ErrClaimNotFound.
	AddDocument(
		ctx context.Context,
		reference string,
		upload UploadedDocument,
		caller string,
	) (Document, error)

	// DocumentCategories mengembalikan pilihan kolom "Category" pada dialog unggah.
	//
	// Daftar KOSONG bukan galat: ia berarti masternya belum diisi, dan layar menggambarnya
	// sebagai dropdown tanpa pilihan — bukan gagal membuka dialognya.
	DocumentCategories(ctx context.Context) ([]DocumentCategory, error)

	// MarkLetterPrinted menandai surat RCL/PUCL sudah diterbitkan — tombol "Download Dokumen".
	//
	// Ia MEMINDAHKAN klaim dari tab "Cetak Surat" ke "Kelengkapan Dokumen", karena penyaring
	// tab pertama adalah `TGL_CETAK_DOKUMEN_PUCL IS NULL`.
	//
	// Ia TIDAK menerbitkan PDF suratnya. Lihat PlannedDifferences untuk alasannya.
	//
	// Kunci yang tidak ditemukan menghasilkan ErrClaimNotFound.
	MarkLetterPrinted(ctx context.Context, reference, caller string) error
}

// ReceiptInput adalah isian Penerimaan Dokumen yang diketik petugas.
//
// # Kenapa hanya DUA, padahal section punya TIGA isian yang dapat diketik
//
// `Section/SectionPenerimaanDokumenPUCL-Section.xml` memuat tiga sel ber-`pyReadOnly false`:
//
//	.ClaimData.EmailLOD                              pxTextInput  wajib=false
//	.ClaimData.PUCLStatus.TanggalTerimaDokumenPUCL   pxDateTime   wajib=TRUE
//	.ClaimData.PUCLStatus.KomentarPUCL               pxTextArea   wajib=TRUE
//
// Kedua yang WAJIB hidup di `POOLDATA.TC_PNC_PUCL`, tabel milik aplikasi ini. Yang ketiga —
// Email Tertanggung — hidup di `POOLDATA.T_CLAIM_PNC.EMAIL_LOD`, tabel LAIN yang modul ini
// tidak pernah tulis. Karena itu ia digambar hanya-baca di layar: isian yang dapat diketik
// tetapi diam-diam tidak tersimpan jauh lebih buruk daripada isian yang jelas tidak dapat
// diketik.
type ReceiptInput struct {
	// Note adalah "Catatan untuk Analyst" — `KomentarPUCL`. WAJIB, mengikuti `pyRequired`.
	//
	// Ia yang dibaca Analyst saat klaim kembali kepadanya, sehingga klaim yang dikirim tanpa
	// catatan membuat Analyst menerima pekerjaan tanpa tahu apa yang berubah.
	Note string

	// CompleteAt adalah "Tanggal Kelengkapan Dokumen" — `TanggalTerimaDokumenPUCL`. WAJIB.
	//
	// Dibawa sebagai TEKS, bukan time.Time, supaya lapisan transport tidak perlu menebak
	// bentuknya dan supaya pesan kesalahannya dapat menyebut nilai yang benar-benar diketik.
	// Penafsirannya dikerjakan Parse di bawah.
	CompleteAt string
}

// receiptLayouts adalah bentuk tanggal yang diterima, berurutan dari yang paling mungkin.
//
// `datetime-local` pada peramban mengirim `2006-01-02T15:04`; layar lama menggambarnya sebagai
// `dd/MM/yyyy HH:mm`. Keduanya diterima supaya nilai yang disalin petugas dari Pega tidak
// ditolak hanya karena bentuknya.
var receiptLayouts = []string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"02/01/2006 15:04:05",
	"02/01/2006 15:04",
	"2006-01-02",
	"02/01/2006",
}

// Validate memeriksa kedua isian wajib dan menafsirkan tanggalnya.
//
// Seluruh pelanggaran dikembalikan SEKALIGUS, bukan berhenti pada yang pertama — meniru Pega
// yang menampilkan semua pesan bersamaan (`P-5`, `11-CROSSCUTTING` §1.2).
func (in ReceiptInput) Validate() (time.Time, error) {
	violations := []Violation{}

	if strings.TrimSpace(in.Note) == "" {
		violations = append(violations, Violation{
			Field:   "catatan_untuk_analyst",
			Message: "Catatan untuk Analyst wajib diisi.",
		})
	}

	var at time.Time
	raw := strings.TrimSpace(in.CompleteAt)
	if raw == "" {
		violations = append(violations, Violation{
			Field:   "tanggal_kelengkapan_dokumen",
			Message: "Tanggal Kelengkapan Dokumen wajib diisi.",
		})
	} else {
		parsed, ok := parseReceiptTime(raw)
		if !ok {
			violations = append(violations, Violation{
				Field: "tanggal_kelengkapan_dokumen",
				Message: "Tanggal Kelengkapan Dokumen tidak terbaca. " +
					"Pakai bentuk dd/mm/yyyy jj:mm.",
			})
		}
		at = parsed
	}

	if len(violations) > 0 {
		return time.Time{}, NewValidationError(violations)
	}
	return at, nil
}

func parseReceiptTime(raw string) (time.Time, bool) {
	for _, layout := range receiptLayouts {
		if at, err := time.Parse(layout, raw); err == nil {
			return at, true
		}
	}
	return time.Time{}, false
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

// ---------------------------------------------------------------------------
// Tindakan tulis — seam ke Pega
// ---------------------------------------------------------------------------

// ErrActionNotAvailable dikembalikan saat tindakan diminta untuk klaim yang tombolnya memang
// TIDAK digambar.
//
// Ia penjagaan sisi peladen, bukan pengulangan pemeriksaan layar: tombol yang tidak tampak
// tetap dapat dipanggil langsung lewat alamatnya, dan `PUCLPost` tidak menolak klaim yang
// jalurnya keliru — ia mengerjakannya.
var ErrActionNotAvailable = errors.New("inboxrclpucl: tindakan tidak tersedia untuk klaim ini")

// ErrPegaServiceUnavailable dikembalikan saat layanan Pega belum tersedia atau tidak dapat
// dihubungi.
//
// Ia DIBEDAKAN dari galat lain dengan sengaja: pemanggil perlu menyatakan "belum tersambung"
// alih-alih "gagal", karena keduanya menuntut tindakan dari orang yang berbeda — yang pertama
// Tim Pega dan Infra, yang kedua tim pengembang.
var ErrPegaServiceUnavailable = errors.New("inboxrclpucl: layanan Pega belum tersedia")

// ClaimActionKind adalah tindakan yang diminta layar.
//
// # Kenapa satu seam untuk beberapa tombol, bukan satu metode per tombol
//
// Karena di Pega pun begitu: EMPAT tombol memanggil activity yang sama (`PUCLPost`), dan yang
// membedakannya hanya parameter. Satu metode per tombol akan menyalin rangkaian yang sama
// empat kali, dan menyembunyikan fakta bahwa keempatnya satu jalur.
type ClaimActionKind string

const (
	// ActionPrintLetter — tombol "Download Dokumen".
	//
	// `InsertMitraPA(tipe="cetak")` lalu `PUCLPost` dengan `Status` KOSONG. Ia menerbitkan
	// PDF suratnya DAN mengisi tanggal cetak, sehingga klaimnya berpindah tab.
	ActionPrintLetter ClaimActionKind = "cetak"

	// ActionRejectClaim — tombol "Tolak Klaim". `PUCLPost` dengan `Status = "0"`.
	ActionRejectClaim ClaimActionKind = "tolak"

	// ActionSendToAnalyst — tombol "Kirim Ke Analyst".
	//
	// `Status = "1"`, lalu penugasannya diselesaikan dengan menyerahkan flow action
	// **`SendtoRCLPUCL`** (`RULE-OBJ-FLOWACTION ASM-FW-GCNMFW-WORK-PNC SENDTORCLPUCL`,
	// dipakai `Register_Flow`) — itulah yang di layar Pega terlihat sebagai Finish Assignment.
	ActionSendToAnalyst ClaimActionKind = "kirim-analyst"

	// ActionSendToPICTeknik — tombol "Kirim ke PIC Teknik". Jalur Travel; di Pega ia
	// `InsertHistoryClaimPNC` lalu `PUCLPost`, bukan `InsertMitraPA`.
	ActionSendToPICTeknik ClaimActionKind = "kirim-pic-teknik"

	// ActionSave — tombol "Save". Activity LAIN: `SaveInputRegisterDetail2`.
	//
	// Ia tidak meneruskan klaim dan tidak menyelesaikan penugasan — satu-satunya tindakan
	// layar ini yang hanya menyimpan.
	ActionSave ClaimActionKind = "save"
)

// KnownClaimActions adalah seluruh tindakan yang dikenali.
//
// "Unggah Dokumen" TIDAK ada di sini, dan itu disengaja: ia menuntut unggahan berkas dan
// mesin lampiran Pega (`PZPVSTREAM`), bukan pemanggilan berparameter. Lihat
// `permintaan-artefak-pega.md` §12.7.
var KnownClaimActions = []ClaimActionKind{
	ActionPrintLetter,
	ActionRejectClaim,
	ActionSendToAnalyst,
	ActionSendToPICTeknik,
	ActionSave,
}

// ClaimActionOf mengubah teks menjadi tindakan yang dikenali.
func ClaimActionOf(raw string) (ClaimActionKind, bool) {
	candidate := ClaimActionKind(strings.TrimSpace(raw))
	for _, known := range KnownClaimActions {
		if candidate == known {
			return known, true
		}
	}
	return "", false
}

// ClaimActionCommand adalah permintaan menjalankan satu tindakan pada satu klaim.
//
// # Kenapa ketiga parameter dibawa, bukan dicari ulang di sisi Pega
//
// Karena `PUCLPost` menerimanya sebagai parameter, dan nilainya menentukan BARIS mana yang
// disetujui: langkah 11 menyetujui `AdjustmentList(<LAST>)` yang `CoverageID`-nya sama dengan
// `param.idCov`. Mencarinya ulang di sisi lain berarti dua tempat dapat memilih baris yang
// berbeda untuk satu klaim yang sama.
type ClaimActionCommand struct {
	// Kind adalah tindakan yang diminta.
	Kind ClaimActionKind

	// CaseNumber adalah nomor case — `PYID` di Pega.
	CaseNumber string

	// Ketiga parameter tersembunyi, dari kolom `ID_OBJECT`, `ID_COVERAGE`, `ID_ADJUSTMENT`.
	IDObject     string
	IDCoverage   string
	IDAdjustment string

	// Note adalah "Catatan untuk Analyst" — `KomentarPUCL`.
	Note string

	// Caller adalah petugas yang menekan tombolnya. Pega mencatat pelaku pada objek kerja,
	// dan tanpa ini jejaknya akan menunjuk akun integrasi, bukan orangnya.
	Caller string
}

// ClaimActions adalah seam ke tindakan yang MENULIS pada klaim.
//
// # Kenapa seam, dan kenapa pengisinya Pega
//
// Ketiga tombol yang memanggil `PUCLPost` menempuh activity Pega, dan kedua tombol Kirim
// ditambah penyerahan flow action **`SendtoRCLPUCL`** — yang di layar terlihat sebagai
// `Finish Assignment`. Penyerahan itulah yang membuat baris penugasan baru di
// `PC_ASSIGN_WORKLIST`, dan **hanya lewat baris itu klaim sampai ke Analyst**: inbox Analyst
// membaca `PC_ASM_FW_GCNMFW_WORK` INNER JOIN `PC_ASSIGN_WORKLIST`.
//
// KOREKSI 2026-10-01 — kekhawatiran di bawah ini GUGUR untuk kedua tombol Kirim.
//
// Di tempat ini sebelumnya tertulis bahwa menulis `PUCL_APPROVE = '1'` akan membuat klaim
// HILANG tanpa sampai ke siapa pun. Pemeriksaan basis data membuktikan sebaliknya: klaim
// RCL/PUCL **sudah** memegang baris `PC_ASSIGN_WORKLIST` Register_Flow-nya, dan pemegangnya
// adalah PIC Teknik klaim itu sendiri (`PXASSIGNEDOPERATORID` = `USERTEKNIS_1`). Menandainya
// selesai mengembalikan klaim kepada PIC Teknik — tujuan yang ditegaskan Work Owner.
//
// Karena itu kedua tombol Kirim TIDAK lagi melewati seam ini; keduanya memakai
// Repo.ReturnToAnalyst. Yang tersisa di seam ini adalah tindakan yang memang menuntut Pega:
// "Download Dokumen" (membuat PDF dan mengirim email), "Tolak Klaim", dan "Save".
//
// Saat Pega dimatikan, pengisi ini diganti logika kami sendiri; yang memakainya tidak berubah.
type ClaimActions interface {
	// Perform menjalankan satu tindakan pada satu klaim.
	//
	// Mengembalikan ErrPegaServiceUnavailable bila layanannya belum tersedia.
	Perform(ctx context.Context, cmd ClaimActionCommand) error
}

// CanSendToAnalyst menyatakan tindakan ini sah untuk klaim yang sedang dibuka.
//
// Syaratnya sama persis dengan syarat tampil tombolnya — `RCL_PUCL = 2 && IsPA` — dan sengaja
// diturunkan dari Buttons(), bukan ditulis ulang. Dua pemeriksaan yang terpisah dapat
// berselisih, dan selisihnya berarti tombol yang tampak tetapi ditolak, atau yang lebih buruk:
// tindakan yang berjalan padahal tombolnya tidak pernah ada.
func (d ClaimDetail) CanSendToAnalyst() bool {
	return d.Buttons().SendToAnalyst
}

// Allows menyatakan sebuah tindakan sah untuk klaim yang sedang dibuka.
//
// Syaratnya diturunkan dari Buttons() — tombol yang TIDAK digambar berarti tindakannya tidak
// sah. Dua pemeriksaan yang terpisah dapat berselisih, dan selisihnya berarti tindakan yang
// berjalan padahal tombolnya tidak pernah ada.
func (d ClaimDetail) Allows(kind ClaimActionKind) bool {
	b := d.Buttons()
	switch kind {
	case ActionPrintLetter:
		return b.DownloadDocument
	case ActionRejectClaim:
		return b.RejectClaim
	case ActionSendToAnalyst:
		return b.SendToAnalyst
	case ActionSendToPICTeknik:
		return b.SendToPICTeknik
	case ActionSave:
		return b.Save
	default:
		return false
	}
}

// UploadedDocument adalah satu berkas yang diunggah petugas lewat "Unggah Dokumen".
type UploadedDocument struct {
	// Name adalah nama berkasnya — `ATTACHNAME`.
	Name string

	// MimeType adalah jenis isinya — `ATTACHMIMETYPE`.
	//
	// Pega menyimpannya APA ADANYA dan sering berupa ekstensi saja (`pdf`), bukan jenis MIME
	// lengkap — terbaca dari lima baris terbaru `DATA_ATTACHFILE`. Yang dikirim layar
	// diteruskan tanpa diubah; menormalkannya akan membuat baris kami berbeda bentuk dari
	// baris Pega pada kolom yang sama (`P-5`).
	MimeType string

	// Note adalah keterangan — `ATTACHNOTE`. Boleh kosong.
	Note string

	// Content adalah isi berkasnya.
	Content []byte

	// Category adalah pilihan kolom "Category" — nama kategori lampiran.
	//
	// Kosong berarti petugas tidak memilih apa pun, dan kolom kategorinya dibiarkan kosong.
	Category string
}

// MaxDocumentSize membatasi besar satu berkas unggahan.
//
// # Kenapa ada batas, dan kenapa segini
//
// Isinya disimpan sebagai BLOB di dalam basis data, dan seluruhnya melewati memori aplikasi
// lebih dulu — baik saat diunggah maupun saat diunduh. Tanpa batas, satu berkas besar menahan
// memori peladen sebesar dirinya sendiri, dikali jumlah petugas yang mengunggah bersamaan.
//
// 10 MiB dipilih karena dokumen klaim yang nyata adalah surat dan kwitansi hasil pindai; lima
// baris terbaru `DATA_ATTACHFILE` seluruhnya `pdf`. Angkanya dapat dinaikkan bila ternyata
// kurang — yang tidak dapat diperbaiki belakangan adalah ketiadaan batas sama sekali.
const MaxDocumentSize = 10 << 20

// Validate memeriksa berkas yang diunggah.
//
// Seluruh pelanggaran dikembalikan SEKALIGUS, meniru Pega yang menampilkan semua pesan
// bersamaan (`P-5`).
func (u UploadedDocument) Validate() error {
	violations := []Violation{}

	if strings.TrimSpace(u.Name) == "" {
		violations = append(violations, Violation{
			Field:   "berkas",
			Message: "Nama berkas tidak terbaca.",
		})
	}
	if len(u.Content) == 0 {
		violations = append(violations, Violation{
			Field:   "berkas",
			Message: "Berkasnya kosong.",
		})
	}
	if len(u.Content) > MaxDocumentSize {
		violations = append(violations, Violation{
			Field:   "berkas",
			Message: "Berkas melebihi 10 MB.",
		})
	}

	if len(violations) > 0 {
		return NewValidationError(violations)
	}
	return nil
}

// DocumentCategory adalah satu pilihan kolom "Category" pada dialog unggah.
//
// Di Pega ia KATEGORI LAMPIRAN (`AcceptanceNote`, `ClaimFaceSheet`, `LOD`), bukan jenis
// dokumen. Yang mendefinisikannya rule `Rule-Obj-AttachmentCategory`, dan tipe rule itu tidak
// ada di export sama sekali (`R-16`) — daftarnya karena itu diturunkan dari kategori yang
// benar-benar dipakai lampiran klaim PNC.
type DocumentCategory struct {
	// Value adalah yang TERSIMPAN di `DATA_ATTACHFILE.CATEGORY`.
	Value string

	// Label adalah yang DIBACA petugas.
	//
	// Untuk sekarang selalu sama dengan Value. Di Pega keduanya berbeda — layar lama
	// menggambar "Acceptance Note" untuk nilai `AcceptanceNote` — tetapi teks itu hidup di
	// rule yang tidak diekspor, dan mengarangnya dengan memecah huruf besar akan mengubah
	// `ATTACHTEMPS` menjadi sesuatu yang tidak pernah ada di layar mana pun.
	//
	// Dipisahkan sejak awal supaya ketika rule-nya tiba, yang berubah hanya pengisiannya.
	Label string
}
