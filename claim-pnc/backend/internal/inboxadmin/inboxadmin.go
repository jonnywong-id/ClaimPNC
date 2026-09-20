// Package inboxadmin adalah inti modul Inbox Admin.
//
// # Layar apa ini
//
// Menu `MENU_ID 63` "Inbox Admin" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk harness
// `PNCInboxAdmin`. Ia berada di kelompok menu `2` (Proses Produksi), urutan 1153.
//
// Isinya **daftar pekerjaan milik petugas admin klaim**, dipecah menjadi beberapa tab. Per
// `D-79` ia benar-benar Inbox, bukan layar data acuan: barisnya pekerjaan yang diambil dari
// DATAPEGA.PC_ASSIGN_WORKLIST, hilang begitu klaimnya selesai
// (`PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')`), dan punya tenggat
// berupa kolom Aging. Karena itu modul ini milik `U-3`, bukan `U-6`.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/PNCInboxAdmin-Harness.xml                   pembungkus layar; judul "Inbox Admin"
//	Section/PNCInboxAdmin-Section.xml                   bilah tab, 8 kontainer grid, 9 tombol
//	Section/PNCAdminShow_sec-Section.xml                sel Case ID; dipanggil 6×
//	Section/ButtonPagingInbox-Section.xml               First/Previous/Next/Last + "Total Data :"
//	Activity/SetTempClaimRegistandNotRegist-Act.xml     38 langkah: pemilih kueri per tab + filter
//	RDB List/BrowseClaimALL-SQL.xml                     tab ALL
//	RDB List/BrowseClaimNotRegistAll-SQL.xml            tab Unregistered RCV dan RCV Online
//	RDB List/BrowseRequestSurvey-SQL.xml                tab Request Survey
//	RDB List/GetRequestDokumenKomunikasi-SQL.xml        tab Request Dokumen
//	RDB List/GetAllCaseAdmin-SQL.xml                    tab All Case Admin
//	RDB List/GetKlaimCabang-SQL.xml                     tab Branch Claim / LOD
//	RDB List/GetReminderPUCL-SQL.xml                    tab Status RCL/PUCL
//
// # Kenapa nama field di sini tidak mirip nama properti Pega
//
// Karena di layar ini nama properti Pega bukan hanya menyesatkan — ia **berarti hal yang
// berbeda dari tab ke tab**. Kedelapan tab berbagi satu halaman klipboard yang sama, dan
// setiap kueri mengisi ulang properti yang sama dengan kolom yang berbeda:
//
//	Properti    di tab ALL                 di tab Request Survey
//	----------- -------------------------- ---------------------------
//	.RCVID      A.SOBNAME (sumber bisnis)  b.surveyor (nama surveyor)
//	.Keterangan PXCREATEOPERATOR (pembuat) b.branch (cabang survei)
//	.Kurir      B.PXFLOWNAME (nama flow)   a.BranchName (cabang polis)
//	.UserAdmin  nama cabang klaim          SUBSTR(surveyid,20,30) (no survei)
//
// Membawa nama seperti itu ke sistem baru berarti mewariskan kekacauan yang justru menjadi
// alasan migrasi (`03-CURRENT-ARCHITECTURE.md` §4.2). Yang dipakai di sini adalah padanan
// Inggris dari `CONTEXT.md` sesuai `D-19` dan `D-80`; pemetaan lengkap tiga arah ada di
// repo/sqlstore/inboxadmin.sql.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxadmin/          aturan modul + seam          ← paket ini
//	inboxadmin/usecase/  orkestrasi: daftar tab, isi satu tab
//	inboxadmin/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxadmin/http/     lapisan transport modul ini  — handler, dto, rute
package inboxadmin

import (
	"context"
	"strings"
	"time"
)

// WorkItem adalah satu baris pekerjaan di Inbox Admin.
//
// # Kenapa satu bentuk untuk delapan tab, dan sebagian isiannya kosong
//
// Karena begitulah sistem lama menyusunnya: kedelapan grid membaca halaman klipboard yang
// sama (`TempClaimPNCAll`, `TempAllCase`, `TempKlaimCabang`, `TempReqDoc`, `TempPUCL`),
// dan tiap tab hanya menampilkan sebagian kolomnya. Memecahnya menjadi delapan bentuk
// berarti delapan pemindai, delapan DTO, dan delapan tabel di layar — padahal yang berbeda
// hanyalah kolom mana yang terlihat.
//
// Kolom yang tidak dibawa kueri tab yang sedang terbuka bernilai kosong, dan layar
// menyembunyikannya mengikuti Tab.Columns — bukan menampilkan kolom kosong yang membuat
// pengguna menduga datanya hilang.
type WorkItem struct {
	// CaseID adalah nomor yang dibaca pengguna di kolom pertama.
	//
	// Isinya berbeda menurut tab, dan itu bentuk sistem lama: enam tab memakai `A.PYID`
	// (berformat `PNC-xxxx` atau `RCV-xxxx`), sedangkan tab Request Dokumen dan All Case
	// Admin memakai `T_CLAIM_PNC.CLAIMNO`. Judul kolomnya pun ikut berbeda — "Case ID"
	// pada yang pertama, "No Klaim" pada yang kedua.
	CaseID string

	// Reference adalah kunci teknis Pega — `PZINSKEY`, berbentuk
	// "ASM-FW-GCNMFW-WORK <nomor>" pada baris warisan.
	//
	// Ia dibawa karena tombol "Lihat Detail Klaim" membutuhkannya, BUKAN untuk
	// ditampilkan. `03-CURRENT-ARCHITECTURE.md` §4.1 menyebut bocornya nama kelas Pega ke
	// data bisnis sebagai utang teknis, dan `D-22` menetapkan klaim terbitan sistem baru
	// tidak pernah menulis awalan itu lagi.
	Reference string

	// PolicyNumber — `POLICYNO` / `NOPOLIS`.
	PolicyNumber string

	// InsuredName adalah Nama Tertanggung — `QQNAME`.
	//
	// Pada tab Status RCL/PUCL ia datang lewat alias `NewTelpTertanggung`, yang namanya
	// menyebut nomor telepon padahal isinya `a.qqname`.
	InsuredName string

	// BusinessName adalah lini bisnis — `BUSINESSNAME`, di grid lama bernama
	// `.JenisDokumen`.
	BusinessName string

	// BusinessSource adalah Business source — `SOBNAME`, di grid lama bernama `.RCVID`.
	BusinessSource string

	// BranchName adalah Branch Name — `BRANCHNAME`, di grid lama `.NumberOfDocument`.
	BranchName string

	// ClaimBranch adalah Branch Claim / Cabang Klaim — nama cabang hasil pencarian
	// `POOLDATA.BRANCH` berdasarkan `KODECABANG_1`. Di grid lama bernama `.UserAdmin`.
	ClaimBranch string

	// Creator adalah petugas yang membuat barisnya — `PXCREATEOPERATOR`, di grid lama
	// bernama `.Keterangan`.
	Creator string

	// LossDate adalah Tanggal Kejadian — `DATEOFLOSS_1` / `DATEOFLOSS`.
	//
	// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari
	// "belum diisi" saat ditampilkan.
	LossDate *time.Time

	// ReportDate adalah Report Date — `REPORTDATE_1` / `REPORTDATE`, di grid lama
	// bernama `.ReceivedDate`. Ia dasar hitungan Aging Report.
	ReportDate *time.Time

	// InputDate adalah Input Date — `PXCREATEDATETIME`. Ia dasar hitungan Total Aging.
	InputDate *time.Time

	// LODDate adalah tanggal masuk antrean LOD — `T_CLAIM_JOB_PERSONALACCIDENT.INSERTDATE`,
	// di grid lama dibawa alias `DateForAging`. Ia dasar hitungan Aging LOD, dan hanya
	// terisi pada tab Branch Claim.
	LODDate *time.Time

	// Note adalah catatan klaim yang belum diregistrasi — `NOTREGISTNOTE_1`, di grid lama
	// dibawa alias `TelpPengirim` yang namanya menyebut nomor telepon pengirim.
	Note string

	// ClaimPosition adalah Claim Position — hasil `CASE PYSTATUSWORK` yang berbunyi
	// "On Progress", "Reject", atau "Close". Di grid lama bernama `.PosisiProgressID`.
	ClaimPosition string

	// ClaimStatus adalah Claim Status — `V_STS_CLAIM.LSC_NOTE` yang dicari berdasarkan
	// `STATUSCLAIM_1`. Di grid lama bernama `.StatusLock`.
	ClaimStatus string

	// LODStatus adalah LOD Status — "Belum Upload" atau "Sudah Upload", hasil `CASE`
	// bertingkat atas `STATUSLOD` yang artinya BERBEDA menurut `FLAGBISNIS`: pada OTO dan
	// BFI nilai `1` berarti belum upload, sedangkan pada lini lain nilai `1` berarti sudah.
	// Di grid lama bernama `.StatusWorkCase`.
	LODStatus string

	// RequestDate adalah Tanggal Request survei — `T_REQ_SURVEY.INPUTDATE`, di grid lama
	// dibawa alias `StatusKomunikasi`.
	RequestDate *time.Time

	// PolicyBranch adalah Cabang Polis — `A.BRANCHNAME`, di grid lama dibawa alias
	// `Kurir`.
	PolicyBranch string

	// SurveyBranch adalah Cabang Survey — `T_REQ_SURVEY.BRANCH`, di grid lama dibawa
	// alias `Keterangan`.
	SurveyBranch string

	// TechnicalPIC adalah PIC Klaim — `USERTEKNIS_1`, di grid lama dibawa alias
	// `Resource`.
	TechnicalPIC string

	// Surveyor adalah nama surveyor — `T_REQ_SURVEY.SURVEYOR`, di grid lama dibawa alias
	// `RCVID`.
	Surveyor string

	// SurveyNumber adalah No Survey — `SUBSTR(T_REQ_SURVEY.SURVEYID, 20, 30)`, di grid
	// lama dibawa alias `UserAdmin`.
	//
	// Pemangkasan 19 karakter pertama itu membuang awalan kelas Pega, persis seperti yang
	// `D-22` tetapkan tidak lagi ditulis oleh sistem baru.
	SurveyNumber string

	// InboxDate adalah Tanggal Masuk Inbox pada tab Status RCL/PUCL —
	// `TANGGALKIRIMPUCL_1`, di grid lama dibawa alias `pxCreateDateTime`.
	InboxDate *time.Time

	// AnalystNote adalah Deskripsi Analyst — `KOMENTARANALISATOR_1`, di grid lama dibawa
	// alias `NoteKomite`.
	AnalystNote string

	// RCLPUCLStatus adalah Status RCL/PUCL — "RCL" bila `RCL_PUCL_1` bernilai `1`,
	// "PUCL" bila `2`, dan kosong selain itu.
	RCLPUCLStatus string

	// LetterPrintDate adalah Tanggal Cetak Surat — `TANGGALCETAKDOKUMENPUCL_1`, di grid
	// lama dibawa alias `TanggalCetakDLA`.
	LetterPrintDate *time.Time

	// ClaimAge adalah Lama Klaim — `LAMAKLAIM_1`, di grid lama dibawa alias `LOGSEEN`
	// yang di modul lain berarti jatah lihat data proteksi.
	ClaimAge string

	// ExpiryStatus adalah Status Kadaluarsa — `STATUSKLAIM_1`, di grid lama dibawa alias
	// `StsAcceptance`.
	ExpiryStatus string

	// ReportAgingDays adalah Aging Report: jumlah hari sejak ReportDate sampai hari ini.
	//
	// Pointer supaya "tidak berlaku pada tab ini" dapat dibedakan dari "nol hari" — dan
	// keduanya memang berbeda: nol hari berarti dilaporkan hari ini.
	ReportAgingDays *int

	// TotalAgingDays adalah Total Aging: jumlah hari sejak InputDate sampai hari ini.
	TotalAgingDays *int

	// LODAgingDays adalah Aging LOD: jumlah hari sejak LODDate sampai hari ini.
	//
	// # Kenapa ia HARI KALENDER, sementara sistem lama memakai hari kerja
	//
	// Sistem lama menghitungnya lewat `GCNMTimeDifferenceWorkCalender_Act`, yang membaca
	// kalender libur `GENERAL.HRD_LBR` lewat DB Link (`RDB List/CheckHoliday_SQL-SQL.xml`).
	// DB Link itu belum punya API pengganti (`R-03`), dan Work Owner memutuskan 2026-09-20
	// modul ini MENUNGGU API tersebut alih-alih menembus DB Link.
	//
	// Sampai API-nya ada, keempat angka Aging dihitung dalam HARI KALENDER. Layar menyatakan
	// keterbatasan itu di bawah judul tabel — tanpa itu, angka yang lebih besar daripada
	// angka di Pega akan dilaporkan berulang kali sebagai kerusakan modul.
	LODAgingDays *int

	// RequestAgingDays adalah Aging pada tab Request Survey: jumlah hari sejak
	// RequestDate sampai hari ini.
	RequestAgingDays *int
}

// agingFrom menghitung selisih hari kalender antara sebuah tanggal dan sekarang.
//
// Mengembalikan nil bila tanggalnya kosong, sehingga kolom Aging pada baris yang tanggal
// dasarnya belum terisi tampil kosong — bukan tampil "0 hari" yang menyatakan hal yang
// tidak benar.
//
// Selisihnya dihitung terhadap TANGGAL, bukan terhadap timestamp: aturan bisnis berbasis
// hari kalender dihitung terhadap tanggal WIB (`08-TECHNICAL-STRATEGY.md` §4.4), sehingga
// baris yang masuk pukul 23.00 dan dibaca pukul 01.00 esok harinya berumur satu hari, bukan
// nol hari.
func agingFrom(at *time.Time, now time.Time) *int {
	if at == nil {
		return nil
	}

	base := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	days := int(today.Sub(base).Hours() / 24)
	if days < 0 {
		// Tanggal di masa depan menghasilkan nol, bukan angka negatif. Baris seperti itu
		// memang ada di data warisan, dan "minus tiga hari" tidak berarti apa pun bagi
		// petugas yang membaca kolom tenggat.
		days = 0
	}
	return &days
}

// WithAging mengembalikan salinan baris yang keempat kolom Aging-nya sudah terisi.
//
// Ia dihitung di sini, bukan di SQL, karena dua alasan: hasilnya dapat diuji secara
// deterministik lewat seam Clock, dan kueri tetap portabel tanpa fungsi tanggal khas
// Oracle (`08-TECHNICAL-STRATEGY.md` §4.3).
func (w WorkItem) WithAging(now time.Time) WorkItem {
	filled := w
	filled.ReportAgingDays = agingFrom(w.ReportDate, now)
	filled.TotalAgingDays = agingFrom(w.InputDate, now)
	filled.LODAgingDays = agingFrom(w.LODDate, now)
	filled.RequestAgingDays = agingFrom(w.RequestDate, now)
	return filled
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 25
//
// Karena itu yang tertulis di `Section/PNCInboxAdmin-Section.xml` sebagai ukuran halaman
// grid-nya. Ia tidak dikarang, dan tidak disamakan dengan modul lain yang memakai 20.
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
	DefaultPageSize = 25
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
// # Kenapa dipotong di aplikasi, bukan di basis data
//
// Keputusan Work Owner 2026-09-20: paginasi layar ini **direplikasi apa adanya**. Sistem
// lama menarik SELURUH baris yang cocok saat layar dibuka — klausa paginasinya
// (`{ASIS:TempContents.NoteKasir}`) baru terpasang setelah tombol halaman ditekan — lalu
// menghitung totalnya dengan `@SizeOfPropertyList(TempClaimPNCAll.pxResults)`, yakni
// menghitung baris yang sudah terlanjur ditarik.
//
// Keberatan atas konsekuensinya sudah disampaikan sebelum keputusan diambil dan keputusan
// ditegaskan; ia dicatat di sini supaya menjadi keputusan yang tercatat, bukan kelalaian.
//
// Yang diterima secara sadar: terhadap `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` yang berisi
// puluhan juta baris (`D-10`), tab tanpa penyaring cabang dapat menarik seluruh klaim yang
// masih berjalan ke memori aplikasi. Usecase karena itu MEMPERINGATKAN lewat log begitu
// satu permintaan melampaui LargeResultWarning.
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

// LargeResultWarning adalah jumlah baris yang, begitu terlampaui, dicatat sebagai
// peringatan di log.
//
// Ia TIDAK memotong hasil dan tidak mengubah perilaku apa pun — memotongnya akan menyalahi
// keputusan "replikasi apa adanya". Yang dilakukannya hanya membuat akibat keputusan itu
// terlihat operator sebelum ia terlihat sebagai aplikasi yang kehabisan memori.
const LargeResultWarning = 5000

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama.
	//
	// Tiga tab menyaring menurut nilai ini, dan itulah yang membuat mereka menjadi "inbox
	// saya" alih-alih daftar seluruh klaim: Request Survey menyaring
	// `PXASSIGNEDOPERATORID`, sedangkan Request Dokumen dan All Case Admin menyaring
	// `PXCREATEOPNAME`.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// Repo adalah seam ke daftar pekerjaan SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat
// kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas,
// dan memang tidak boleh ada.
//
// # Kenapa hanya ada satu operasi, dan tidak ada satu pun yang menulis
//
// Karena layar ini tidak mengubah apa pun. Ia membaca antrean; registrasi klaim (`B-2`),
// akseptasi (`B-10`), dan penutupan klaim terjadi di layar lain. Operasi yang tidak
// tersedia di seam ini tidak dapat dipakai kode yang ditulis kemudian tanpa keputusan
// sadar — dan pada masa paralel itu penting, karena seluruh tabel yang dibaca modul ini
// adalah milik Pega (`P-1`).
type Repo interface {
	// List mengembalikan SELURUH baris yang cocok, belum dipaginasi.
	//
	// Ia sengaja tidak menerima Pagination: pemotongan halaman terjadi di aplikasi
	// (lihat Slice), dan menaruhnya di sini akan menyembunyikan bahwa seluruh baris
	// memang ditarik.
	List(ctx context.Context, query Query) ([]WorkItem, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan antrean kerja
// satu badan hukum kepada petugas badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// Ia ada supaya hitungan Aging dapat diuji secara deterministik. Seluruh waktu yang
// dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan tidak pernah dengan
// menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}
