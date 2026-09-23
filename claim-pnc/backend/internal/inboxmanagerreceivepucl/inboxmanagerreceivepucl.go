// Package inboxmanagerreceivepucl adalah inti modul Inbox Manager Receive / PUCL.
//
// # Layar apa ini
//
// Menu `MENU_ID 56` "Inbox Manager Receive / PUCL" pada POOLDATA.M_MENU_APLIKASI_PNC, yang
// menunjuk harness `ReceiveDoucument_Harness`. Ia berada di kelompok menu `2` (Proses
// Produksi), urutan 1146 — tepat sesudah `MENU_ID 55` "Inbox Claim Treaty Non Prop".
//
// Isinya **pandangan penyelia atas dua antrean yang berbeda sama sekali**, disatukan dalam
// satu layar karena keduanya sama-sama menunggu tindakan manajer:
//
//	Receive    berkas penerimaan dokumen klaim yang masih punya penugasan terbuka
//	RCL/PUCL   klaim yang ditolak (RCL) atau diproses ulang (PUCL) dan menunggu keputusan
//
// Per `D-79` keduanya benar-benar Inbox: barisnya diambil dari tabel penugasan Pega, dan
// hilang begitu penugasannya selesai. Karena itu modul ini milik `U-3`, bukan `U-6`.
//
// # Dua antrean, dua KELAS OBJEK KERJA yang berbeda
//
// Ini yang paling mudah salah baca, dan akibatnya tidak menghasilkan satu pun galat:
//
//	            tab Receive                          tab RCL/PUCL
//	kelas       ASM-FW-GCNMFW-Work-ReceiveDocument   ASM-FW-GCNMFW-Work-PNC
//	tabel       PC_ASM_FW_GCNMFW_WORK (keduanya)     PC_ASM_FW_GCNMFW_WORK (keduanya)
//	pembeda     PXOBJCLASS                           PXOBJCLASS
//	penugasan   PC_ASSIGN_WORKLIST                   PC_ASSIGN_WORKBASKET
//	kolom       POLICYNO, QQNAME, DATEOFLOSS_1       KOMENTARANALISATOR_1, RCL_PUCL_1, …
//
// Kedua kelas hidup di SATU tabel yang sama dan hanya dibedakan `PXOBJCLASS`. Melupakan
// penyaring itu akan mencampur berkas penerimaan dokumen dengan klaim — dua hal yang nomor,
// kolom, dan artinya berbeda, tetapi bentuk barisnya cukup mirip untuk lolos tanpa terlihat.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/ReceiveDoucument_Harness-Harness.xml    pembungkus layar (Data-Portal)
//	Section/InboxManagerReceive_Section-Section.xml layout TABBED, 2 judul tab, 3 grid
//	Report Definition/ManagementRecieveView-RD.xml  kedua grid Receive (PA dan NON-MBU)
//	Report Definition/InboxRCLPUCL_RD-RD.xml        grid RCL/PUCL
//	RDB List/CountKlaimPUCL-SQL.xml                 penyaring antrean RCL/PUCL
//	RDB List/GetReminderPUCL-SQL.xml                pemetaan kolom RCL/PUCL
//	RDB List/ReminderPUCL-SQL.xml                   pemetaan kolom RCL/PUCL (lengkap)
//	Database/PROCINSERTDATARECIVEDKLAIM.prc         kolom berkas penerimaan dokumen
//
// # Tiga hal di layar lama yang TIDAK dapat dibawa apa adanya
//
// Ketiganya diputuskan Work Owner 2026-09-22 dan dinyatakan ke pengguna lewat
// PlannedDifferences — bukan hanya dicatat di sini.
//
// **`.ReceiveDocument.TypeOfClaim` tidak punya kolom basis data.** Ia ditandai
// `<pzPropertyType>unexposed</pzPropertyType>` di `ManagementRecieveView-RD.xml`, dengan dua
// peringatan Pega "Not optimized for reporting" dan "Not optimized for filtering" yang
// menyebut properti itu satu-satunya. Artinya ia hidup di dalam blob Pega dan TIDAK dapat
// disaring SQL. Padahal justru properti inilah pembeda kedua grid Receive — yang satu
// dijalankan dengan `Position1="PA"` dan yang lain `"NONMBU"`. Penggantinya Group Panel;
// lihat ClaimTypePA.
//
// **Filter unit organisasi tidak pernah diisi.** `ManagementRecieveView` menyaring
// `newAssignPage.pxAssignedOrgUnit = Param.OrgUnit`, tetapi section mengirim `<OrgUnit/>`
// kosong dan tidak ada satu pun activity di seluruh export yang mengisinya. Penyaringnya
// karena itu tidak dibawa — bukan dihilangkan, melainkan memang tidak pernah berlaku.
//
// **Report Definition RCL/PUCL tidak menyaring antreannya.** `InboxRCLPUCL_RD` tidak punya
// join sama sekali dan hanya menyaring `pyStatusWork <> "Resolved-Completed"`; parameternya
// bernama `assign` dan TIDAK DIPAKAI satu filter pun. Ditiru apa adanya, tab itu akan
// menampilkan seluruh klaim yang belum selesai. Lihat RCLPUCLWorkbasket.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxmanagerreceivepucl/          aturan modul + seam          ← paket ini
//	inboxmanagerreceivepucl/usecase/  orkestrasi: rakit tab, isi satu tab
//	inboxmanagerreceivepucl/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxmanagerreceivepucl/http/     lapisan transport modul ini  — handler, dto, rute
package inboxmanagerreceivepucl

import (
	"context"
	"strings"
)

// WorkItem adalah satu baris pekerjaan pada layar ini.
//
// # Kenapa SATU bentuk untuk dua antrean yang berbeda kelas
//
// Karena keduanya digambar grid yang sama bentuknya, dan yang membedakan hanyalah kolom
// mana yang terlihat — itu ditetapkan Tab.Columns, bukan oleh bentuk barisnya. Bentuk yang
// dipecah dua akan memaksa lapisan transport, penyimpanan, dan layar masing-masing
// bercabang, padahal percabangannya hanya satu: tab mana yang terbuka.
//
// Isian yang tidak berlaku bagi sebuah tab bernilai kosong, dan itu TIDAK pernah terlihat
// pengguna: kolomnya memang tidak digambar di tab itu.
type WorkItem struct {
	// Reference adalah kunci teknis Pega — `PZINSKEY`.
	//
	// Ia dikirim ke layar tetapi TIDAK pernah digambar sebagai kolom; yang memakainya
	// adalah tombol rincian. Ia pula kunci gabungan ke POOLDATA.T_CLAIM_RECIVEDCLAIM,
	// yang menyimpan `CLAIMID` berisi nilai pzInsKey apa adanya
	// (`Database/PROCINSERTDATARECIVEDKLAIM.prc` — `TCLAIMID := {TampunganPages.pzInsKey}`).
	Reference string

	// CaseID adalah nomor yang dibaca pengguna — `PYID`.
	//
	// Judulnya BERBEDA antar tab pada layar lama: "CaseID" pada tab Receive dan
	// "Nomor Case" pada tab RCL/PUCL. Keduanya isi yang sama; perbedaan judul
	// dipertahankan lewat Tab.Columns (`D-13`).
	CaseID string

	// PolicyNumber — `POLICYNO`. Dipakai KEDUA tab.
	PolicyNumber string

	// ClaimNumber adalah nomor klaim PNC yang terbit dari berkas penerimaan ini —
	// `PNCCASEID`, berjudul "PNC CaseID".
	//
	// HANYA tab Receive yang memilikinya. Ia kosong selama berkasnya belum diregistrasi
	// menjadi klaim, dan itu keadaan yang sah — bukan data hilang.
	ClaimNumber string

	// InsuredName — `QQNAME`, berjudul "Nama Tertanggung". Dipakai KEDUA tab.
	InsuredName string

	// LossDate adalah Tanggal Kejadian — `DATEOFLOSS_1`.
	//
	// HANYA tab Receive. Ia TEKS, bukan waktu: kolomnya belum pernah dilihat bentuknya
	// karena DDL tabel Pega tidak tersedia (`R-08`), dan mengubahnya menjadi tanggal di
	// sini berarti menebak formatnya untuk seluruh baris historis. Pemformatannya
	// dikerjakan layar, dan hanya bila bentuknya memang dikenali.
	LossDate string

	// ClaimType adalah Jenis Klaim yang digambar tab Receive.
	//
	// # Ia DITURUNKAN dari Group Panel, bukan dibaca apa adanya
	//
	// Sumber aslinya `.ReceiveDocument.TypeOfClaim`, yang tidak punya kolom basis data —
	// lihat catatan di kepala paket. Penggantinya `GROUPPANEL_1`, yang memang menyimpan
	// segmentasi lini bisnis dan memang punya kolom. Pemetaannya ada di ClaimTypeOf.
	ClaimType string

	// SenderName adalah Nama Pengirim — `POOLDATA.T_CLAIM_RECIVEDCLAIM.NAMAPELAPOR`.
	//
	// # Kenapa dari tabel cermin, bukan dari objek kerja
	//
	// Karena tidak ada kolom `SENDER_1` di mana pun: penelusuran seluruh export tidak
	// menemukan satu pun kueri yang membacanya. Yang ada adalah tabel cermin yang diisi
	// `PROCINSERTDATARECIVEDKLAIM`, dan di sanalah namanya tersimpan.
	//
	// Bahwa isian ini memang "nama pengirim" terbaca dari label layar input:
	// `Section/ViewInputReceiveDocument_sec-Section.xml` memberi `.ReceiveDocument.Sender`
	// judul "Nama Pengirim / Pelapor Dokumen". Ia BUKAN `.ReceiveDocument.Kurir`, yang
	// berjudul "Nama Kurir ASM" dan tersimpan di kolom NAMAKURIRASM.
	//
	// Risiko yang disadari: tabel itu TIDAK PERNAH DIBACA sistem lama — satu-satunya
	// penyentuhnya adalah procedure yang menulisinya. Gabungannya karena itu LEFT JOIN,
	// sehingga baris tanpa pasangan tetap muncul dengan isian kosong alih-alih hilang.
	SenderName string

	// DocumentReceivedDate adalah Tanggal Terima Dokumen —
	// `POOLDATA.T_CLAIM_RECIVEDCLAIM.TANGGALTERIMADOKUMEN`.
	//
	// Ia TEKS, dan itu bukan pilihan: parameter procedure yang mengisinya bertipe
	// `varchar2` (`PROCINSERTDATARECIVEDKLAIM.prc:4`), sementara parameter tanggal lain
	// pada procedure yang sama bertipe `DATE`. Kolomnya karena itu menyimpan teks, dan
	// bentuk teksnya tidak dapat diperiksa tanpa DDL (`R-08`).
	DocumentReceivedDate string

	// DocumentSheetCount adalah Jumlah Lembar Dokumen — `.ReceiveDocument.NumberOfDocument`,
	// berjudul "Total Jumlah Dokumen" di layar input.
	//
	// # Ia SELALU KOSONG, dan itu bukan kelalaian
	//
	// Tidak ada kolom `NUMBEROFDOCUMENT_1` di mana pun pada export, dan
	// `T_CLAIM_RECIVEDCLAIM` tidak punya kolom jumlah lembar sama sekali. Menebak nama
	// kolomnya menghasilkan kueri yang gagal saat pertama dijalankan di produksi — jauh
	// lebih mahal daripada satu kolom yang kosong dan dijelaskan.
	//
	// Kolomnya tetap DIGAMBAR, bukan dihilangkan: menghilangkannya membuat layar tampak
	// setara dengan Pega padahal ada isian yang belum terbawa, dan itu persis yang tidak
	// boleh terjadi pada uji kesetaraan gerbang 1.
	DocumentSheetCount string

	// InboxEntryAt adalah Tanggal Masuk Inbox — `PXCREATEDATETIME`.
	//
	// HANYA tab RCL/PUCL yang menggambarnya. Ia dipakai KEDUA tab untuk mengurutkan.
	InboxEntryAt string

	// AnalystNote adalah Deskripsi Analyst — `KOMENTARANALISATOR_1`.
	//
	// HANYA tab RCL/PUCL. Alias Pega-nya menyesatkan di dua tempat berbeda:
	// `GetReminderPUCL-SQL.xml:6` menamainya "NoteKomite" dan `ReminderPUCL-SQL.xml`
	// menamainya "LOGSEARCH". Keduanya tidak dibawa (`D-19`).
	AnalystNote string

	// Track menyatakan klaim ini berada di jalur RCL atau PUCL — `RCL_PUCL_1`.
	//
	// Kolomnya menyimpan ANGKA, bukan teks: `1` berarti RCL, `2` berarti PUCL. Penerjemahan
	// itu ada di kueri sistem lama apa adanya (`GetReminderPUCL-SQL.xml:7-9`), bukan
	// dikarang di sini — lihat TrackRCL dan TrackPUCL.
	Track string

	// TrackStatus adalah Status RCL/PUCL — `STATUSKLAIM_1`.
	//
	// Ia BUKAN `STATUSCLAIM_1`, dan keduanya ada pada tabel yang sama. Yang ini status
	// jalur RCL/PUCL; yang itu Status Klaim ber-33 kode `1134`–`1166` (`R-06`). Kedua nama
	// kolomnya hanya berbeda satu huruf, dan menukarnya tidak menghasilkan satu pun galat.
	TrackStatus string

	// LetterPrintedAt adalah Tanggal Cetak Surat — `TANGGALCETAKDOKUMENPUCL_1`.
	//
	// HANYA tab RCL/PUCL. Kosong berarti suratnya belum dicetak, dan itu keadaan yang sah.
	LetterPrintedAt string

	// ClaimAge adalah Lama Klaim — `LAMAKLAIM_1`.
	//
	// # Kenapa TEKS, bukan angka
	//
	// Karena satuannya tidak diketahui. Tidak satu pun kueri di export menghitungnya —
	// keduanya (`ReminderPUCL`, `GetReminderPUCL`) hanya MEMILIH kolomnya, dan tidak ada
	// DDL yang menyatakan tipenya (`R-08`). Mengubahnya menjadi angka berhari di sini
	// berarti menetapkan satuan yang belum pernah dipastikan, lalu menampilkannya sebagai
	// fakta. Ia dibawa apa adanya dan digambar apa adanya.
	ClaimAge string

	// ExpiryStatus adalah Status Kadaluarsa — `STATUSCASE_1`.
	//
	// HANYA tab RCL/PUCL. Alias Pega-nya `ClaimData.PUCLStatus.Stat25L`
	// (`ReminderPUCL-SQL.xml`) — nama yang terpotong oleh batas panjang alias Oracle dan
	// tidak menyatakan apa pun; tidak dibawa (`D-19`).
	ExpiryStatus string
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
//
// # Kenapa modul ini tetap menuntutnya, padahal tidak ada kueri yang menyaring menurut
// pemanggil
//
// Karena layar ini adalah pandangan PENYELIA atas pekerjaan orang lain: seluruh barisnya
// memuat nama tertanggung dan nomor polis milik pekerjaan yang bukan milik pemanggil.
// Selama pemeriksaan peran belum ada (`TKT-F3-004`), satu-satunya hal yang menyatakan siapa
// yang membukanya adalah jejak di log — dan jejak itu tidak dapat ditulis tanpa identitas.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// Kelas objek kerja yang membedakan kedua antrean.
//
// Keduanya hidup di SATU tabel — DATAPEGA.PC_ASM_FW_GCNMFW_WORK — dan hanya dibedakan kolom
// `PXOBJCLASS`. Nilainya dikumpulkan sebagai konstanta supaya SQL dan penyimpanan memori
// tidak dapat berselisih tanpa ketahuan.
const (
	// WorkClassReceiveDocument adalah berkas penerimaan dokumen klaim.
	//
	// Nilainya terbaca di `ManagementRecieveView-RD.xml` sebagai `pyClassName`, dan dipakai
	// sebagai penyaring `pxobjclass` oleh 15 kueri lain di export — antara lain
	// `RDB List/ViewTableBrowseRCVInProcess-SQL.xml`.
	WorkClassReceiveDocument = "ASM-FW-GCNMFW-Work-ReceiveDocument"

	// WorkClassClaim adalah objek kerja klaim PNC.
	//
	// Nilainya terbaca di `InboxRCLPUCL_RD-RD.xml` sebagai `pyClassName`, dan dipakai
	// sebagai syarat gabungan oleh `RDB List/CountKlaimPUCL-SQL.xml` maupun
	// `RDB List/ReminderPUCL-SQL.xml`.
	WorkClassClaim = "ASM-FW-GCNMFW-Work-PNC"
)

// RCLPUCLWorkbasket adalah akun antrean bersama yang memegang pekerjaan RCL/PUCL.
//
// # Kenapa penyaring ini ADA di sini padahal Report Definition-nya tidak punya
//
// `InboxRCLPUCL_RD` yang memasok grid RCL/PUCL tidak punya join sama sekali, dan satu-satunya
// filternya adalah `pyStatusWork != "Resolved-Completed"`. Parameternya bernama `assign` —
// dideklarasikan, tetapi tidak dirujuk satu filter pun.
//
// Ditiru apa adanya, tab itu akan menampilkan SELURUH klaim PNC yang belum selesai, bukan
// antrean RCL/PUCL. Pada basis data berisi puluhan juta baris (`D-10`), hasilnya bukan sekadar
// keliru melainkan tidak dapat dipakai sama sekali.
//
// Dua kueri lain pada domain yang SAMA menyaringnya dengan cara yang sama persis, dan
// keduanya ada di export:
//
//	RDB List/CountKlaimPUCL-SQL.xml   b."PXASSIGNEDOPERATORID" = 'RCLPUCL'
//	RDB List/ReminderPUCL-SQL.xml     "AssignBasket"."PXASSIGNEDOPERATORID" = 'RCLPUCL'
//
// Nilai itulah yang dipakai di sini. Arahnya MENYEMPITKAN — lebih sedikit baris, bukan lebih
// banyak — sehingga kekeliruan yang mungkin tersisa tidak dapat membocorkan baris yang
// seharusnya tersembunyi. Ia dinyatakan ke pengguna lewat PlannedDifferences, bukan
// disembunyikan sebagai detail kueri.
//
// Ia BUKAN nama orang melainkan akun fungsional, sehingga menuliskannya di sini tidak
// melanggar `D-67`.
const RCLPUCLWorkbasket = "RCLPUCL"

// Status kerja yang DIKELUARKAN dari tab RCL/PUCL.
//
// Satu-satunya filter `InboxRCLPUCL_RD`, dan dibawa apa adanya. Perhatikan ia hanya
// mengeluarkan yang SELESAI — klaim yang `Resolved-Rejected` tetap muncul, dan itu memang
// benar: klaim yang ditolak justru pekerjaan utama antrean RCL.
const WorkStatusCompleted = "Resolved-Completed"

// Jalur penanganan klaim pada tab RCL/PUCL.
//
// Kolom `RCL_PUCL_1` menyimpan angka; teks inilah yang digambar. Penerjemahannya ada di
// `RDB List/GetReminderPUCL-SQL.xml:7-9` apa adanya:
//
//	case when a.RCL_PUCL_1 = '1' then 'RCL'
//	     when a.RCL_PUCL_1 = '2' then 'PUCL'
//	End
//
// Nilai di luar keduanya menghasilkan teks kosong, sama seperti `CASE` tanpa `ELSE` di sana.
const (
	// TrackRCL — Rejected Klaim (`CONTEXT.md`).
	TrackRCL = "RCL"

	// TrackPUCL — Proses Ulang Klaim (`CONTEXT.md`).
	TrackPUCL = "PUCL"
)

// Kode Group Panel yang menggantikan `.ReceiveDocument.TypeOfClaim`.
//
// # Kenapa Group Panel, dan apa yang berubah karenanya
//
// `.ReceiveDocument.TypeOfClaim` tidak punya kolom basis data — lihat catatan di kepala
// paket. Group Panel adalah segmentasi lini bisnis yang MEMANG punya kolom (`GROUPPANEL_1`),
// memang dibaca kueri lain pada kelas yang sama
// (`RDB List/GetDataRCVallKlaimPATravel-SQL.xml:10`), dan memang membedakan Personal
// Accident dari lini lain: `002` adalah PA (`CONTEXT.md`, Business Understanding §1).
//
// Keputusan Work Owner 2026-09-22. Ia SELISIH TERENCANA, bukan pemindahan: kedua penanda
// tidak dijamin selalu sepadan, dan berkas yang Group Panel-nya kosong tidak muncul di tab
// mana pun — sama seperti berkas yang `TypeOfClaim`-nya kosong tidak muncul di grid mana pun
// pada layar lama.
const (
	// GroupPanelPA adalah Personal Accident.
	GroupPanelPA = "002"
)

// ClaimTypeOf menerjemahkan Group Panel menjadi Jenis Klaim yang digambar grid.
//
// Ia ada di paket domain, bukan di SQL maupun di layar, karena kedua pengisi seam wajib
// menghasilkan teks yang sama persis — dan uji aturan modul yang berjalan di atas memori
// hanya menyatakan sesuatu tentang Oracle bila keduanya memakai penerjemah yang sama.
func ClaimTypeOf(groupPanel string) string {
	if strings.TrimSpace(groupPanel) == GroupPanelPA {
		return ClaimTypePA
	}
	return ClaimTypeNonMBU
}

// Teks Jenis Klaim yang digambar grid.
//
// Ejaannya mengikuti nilai parameter Report Definition apa adanya
// (`Section/InboxManagerReceive_Section-Section.xml` — `<Position1>"PA"</Position1>` dan
// `<Position1>"NONMBU"</Position1>`), bukan ejaan yang lebih rapi. Pengguna yang
// membandingkan kedua layar berdampingan membaca teks yang sama.
const (
	ClaimTypePA     = "PA"
	ClaimTypeNonMBU = "NONMBU"
)

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 50
//
// Karena itu ukuran halaman ketiga grid pada
// `Section/InboxManagerReceive_Section-Section.xml` — `<pyPageSize>50</pyPageSize>`, sama
// pada grid Receive PA, Receive NON-MBU, maupun RCL/PUCL. Ia BERBEDA dari modul inbox lain
// yang memakai 25, dan perbedaan itu tidak diseragamkan: yang dipakai adalah angka layarnya
// sendiri.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK,
// bukan dipenuhi diam-diam — memenuhinya membuat batas menjadi saran, bukan batas.
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
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
// `OFFSET … FETCH NEXT`, dan perbedaan itu disengaja — lihat catatan paginasi di
// repo/sqlstore/inboxmanagerreceivepucl.sql.
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

// Repo adalah seam ke kedua antrean layar ini pada SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat
// kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas,
// dan memang tidak boleh ada.
//
// # Tidak ada satu pun operasi yang menulis
//
// Keputusan Work Owner 2026-09-22, dan batasnya sama dengan seluruh modul inbox sebelumnya.
// Kedua tabel penugasan dan tabel objek kerja masih dimiliki Pega selama masa paralel
// (`P-1`), sehingga aksi tulis apa pun di sini akan membuat dua sistem menulis tabel yang
// sama.
//
// Operasi yang tidak tersedia di seam ini tidak dapat dipakai kode yang ditulis kemudian
// tanpa keputusan sadar.
type Repo interface {
	// List mengembalikan SATU HALAMAN baris yang cocok beserta jumlah seluruhnya.
	//
	// Paginasi diserahkan ke pengisi seam, bukan dikerjakan pemanggil, supaya pengisi SQL
	// dapat memotongnya di basis data. Pengisi memori memakai Slice untuk hasil yang sama.
	List(ctx context.Context, query Query, page Pagination) (Page, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan nama tertanggung
// dan nomor polis satu badan hukum kepada petugas badan hukum lain tanpa satu pun pesan
// galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
