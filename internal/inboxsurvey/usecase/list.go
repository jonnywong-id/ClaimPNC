// Package usecase mengorkestrasi modul My Work (MENU_ID 50).
//
// Empat operasi, dan keempatnya hanya MEMBACA:
//
//	Metadata  judul kolom, judul tab, selisih terencana, dan keterbatasan
//	List      satu halaman satu tab
//	Counts    jumlah baris ketujuh tab, untuk bilah tab
//	KPI       ringkasan KPI adjuster
//
// Tidak ada operasi yang menulis. Menerima penugasan, menjadwal ulang survei, dan mengunggah
// laporan seluruhnya menempuh `Surveyor_Flow` — sebuah flow yang TIDAK ADA di export — dan
// penugasan masih dimiliki Pega selama masa paralel (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/inboxsurvey"
)

// Service melayani modul My Work.
type Service struct {
	repoSelector      inboxsurvey.RepoSelector
	directorySelector inboxsurvey.DirectorySelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector      inboxsurvey.RepoSelector
	DirectorySelector inboxsurvey.DirectorySelector
}

// NewService membentuk layanan modul My Work.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxsurvey/usecase: RepoSelector wajib diisi")
	}
	if o.DirectorySelector == nil {
		return nil, errors.New("inboxsurvey/usecase: DirectorySelector wajib diisi")
	}
	return &Service{
		repoSelector:      o.RepoSelector,
		directorySelector: o.DirectorySelector,
	}, nil
}

// Column adalah satu judul kolom pada layar.
//
// # Kenapa judul kolom datang dari server
//
// Karena ketiga belasnya adalah HASIL PEMBACAAN
// `Section/InboxSurvey_section-Section.xml`, dan tempat pembacaan itu tercatat adalah
// backend. Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu
// akan tertinggal saat yang lain diperbaiki.
type Column struct {
	// Key adalah nama field pada baris JSON yang diisi kolom ini.
	Key string

	// Title adalah judul yang DILIHAT pengguna, apa adanya dari section (`D-13`).
	//
	// Ketiga belasnya berbahasa Inggris di layar lama, dan dibiarkan begitu. `D-80`
	// menetapkan teks yang dilihat pengguna mengikuti layar Pega apa adanya; menerjemahkan
	// "Appointment No" menjadi "Nomor Janji" akan membuat pengguna yang hafal layarnya harus
	// belajar ulang — hal yang justru dihindari `D-13`.
	Title string

	// Note adalah keterangan yang ditempelkan pada judul, kosong bila tidak ada.
	Note string

	// Available menyatakan kolom ini benar-benar terisi dari data.
	//
	// Kolom yang TIDAK tersedia tetap digambar — `D-13` menetapkan bentuk layar mengikuti
	// Pega, dan menghapus tiga dari tiga belas kolom akan membuat pengguna yang hafal
	// layarnya mengira isinya hilang. Yang berubah: judulnya menyatakan sebabnya lewat Note,
	// alih-alih menampilkan sel kosong yang terbaca sebagai "data belum diisi".
	Available bool

	// Substitute menyatakan kolom ini TERISI, tetapi dari kolom yang BERBEDA dari Pega.
	//
	// # Kenapa keadaan ketiga, bukan cukup dua
	//
	// `Available` membedakan "terisi" dari "kosong". Ia TIDAK dapat menyatakan keadaan yang
	// paling berbahaya di antara keduanya: **terisi, wajar dilihat, dan bukan angka yang
	// sama dengan Pega**.
	//
	// Keadaan itu nyata pada tiga kolom. Pega membacanya dari kolom objek kerja yang
	// `POOLDATA.T_SURVEYORLIST` **tidak punya sama sekali** — terbukti dari daftar kolom
	// `INSERT_SURVEYORLIST.prc:31-32` dan `:38-52`:
	//
	//	judul layar         Pega                     di sini
	//	------------------- ------------------------ ------------------------
	//	Status ASM          a.ASMSTATUS_1            s.STS_SURVEY
	//	PIC Loss Adjuster   a.ADJUSTERPIC_1          s.SURVEYOR_NAME
	//	Location            a.RescheduleLocation_1   s.LOCATION_SURVEY
	//
	// Menandainya `Available: false` akan menyembunyikan data yang benar-benar ada dan
	// berguna. Membiarkannya polos akan menyamarkan selisih yang menyentuh `P-5`. Penanda
	// tersendiri inilah yang membuat pengguna melihat keduanya sekaligus.
	Substitute bool
}

// Tinggal SATU kolom yang belum tersedia — "Reference No" — sehingga keterangannya ditulis di
// tempatnya sendiri, bukan sebagai konstanta bersama.
//
// Konstanta bersama sempat ada dan dihapus 2026-10-03: begitu "Appointment No" hidup, satu-
// satunya pemakai lainnya tinggal satu, dan konstanta yang dipakai sekali hanya menjauhkan
// teksnya dari kolom yang dijelaskannya.

// columns adalah ketiga belas kolom layar, dalam urutan tampilnya.
//
// Urutannya mengikuti pembacaan grid pada `Section/InboxSurvey_section-Section.xml`, yang
// memuat ketiga belas judul itu berturut-turut pada ketiga varian gridnya.
var columns = []Column{
	{
		Key:   "appointment_no",
		Title: "Appointment No",
		Note: "Nomor berkas survei, dipotong dari POOLDATA.T_SURVEYORLIST.CASEID — sama " +
			"seperti Pega, yang memotongnya dengan @substring(.CaseID,19,30).",
		Available: true,
	},
	{
		Key:   "reference_no",
		Title: "Reference No",
		Note: "BELUM TERSEDIA. Kolomnya POOLDATA.T_SURVEYORLIST.REFNO sudah ada tetapi " +
			"seluruh barisnya masih kosong — menunggu jalur pengisian dari Tim Pega. " +
			"Asalnya terbukti dari keempat kueri tab: a.REFNO_1 AS \"UserName\".",
		Available: false,
	},
	{Key: "claim_no", Title: "Claim No", Available: true},
	{Key: "policy_no", Title: "Policy No", Available: true},
	{Key: "insured_name", Title: "Insured Name", Available: true},
	{Key: "cob", Title: "COB", Available: true},
	{
		Key:   "cause_of_loss",
		Title: "Cause Of Loss",
		Note: "Diambil dari kolom `LOSSTYPE`. Kueri Pega yang mengisinya hilang dari " +
			"export, sehingga pemetaannya belum dapat dibuktikan.",
		Available: true,
	},
	{
		Key:   "location",
		Title: "Location",
		Note: "PENGGANTI, dan terukur berbeda pada 660 dari 2.427 berkas. Pega menggambar " +
			"`RescheduleLocation_1` yang berasal dari `ObjectSurveyLocation`. Kolom ini " +
			"menggambar `LOCATION_SURVEY`, yang diisi `Param.ObjName` — yaitu lokasi objek " +
			"atau nama objek. Dua konsep, bukan dua salinan.",
		Available:  true,
		Substitute: true,
	},
	{Key: "pic_asm", Title: "PIC ASM", Available: true},
	{
		Key:   "pic_loss_adjuster",
		Title: "PIC Loss Adjuster",
		Note: "PENGGANTI, dan terukur berbeda orang. Pega menggambar `ADJUSTERPIC_1` — PIC " +
			"pada adjuster eksternal, terisi 1.796 dari 2.126 berkas type 2 dan hampir " +
			"tidak pernah pada surveyor internal. Kolom ini menggambar `SURVEYOR_NAME`, " +
			"padanan `SURVEYORNAME_1`. Dari 2.463 berkas, hanya 3 yang namanya sama.",
		Available:  true,
		Substitute: true,
	},
	{Key: "date_of_loss", Title: "Date of Loss", Available: true},
	{
		Key:   "aging",
		Title: "Aging",
		Note: "DIHITUNG dari tanggal janji survei dicatat (`TGLINPUT`) terhadap tanggal WIB " +
			"hari ini, bukan dibaca dari kolom. Kosong berarti tanggal masuknya tidak ada, " +
			"dan itu berbeda dari nol hari.",
		Available: true,
	},
	{
		Key:   "status_asm",
		Title: "Status ASM",
		Note: "Peran koasuransi ASM pada klaim ini — LEADER atau MEMBER — dari " +
			"`T_CLAIM_PNC.LEADER_MEMBER`. Padanannya di Pega `ASMSTATUS_1`, diukur di " +
			"produksi tanpa satu pun pertentangan pada 17.633 baris. Selisihnya 10 baris " +
			"(0,06%): Pega menggambar LEADER saat kolomnya kosong, sedangkan di sini " +
			"peran sebenarnya yang digambar.",
		Available: true,
	},
}

// Columns menyerahkan salinan daftar kolom, disesuaikan dengan kesiapan portal.
//
// # Kenapa daftarnya disesuaikan, bukan tetap
//
// Tiga kolom berubah sifat begitu kolom sumbernya siap:
//
//	reference_no        belum tersedia  ->  tersedia
//	location            pengganti       ->  sumber sebenarnya
//	pic_loss_adjuster   pengganti       ->  sumber sebenarnya
//
// Menuliskannya tetap akan membuat layar menyatakan hal yang tidak lagi benar — persis cacat
// yang ditemukan Work Owner 2026-10-07, ketika `pega_dev83` sudah terisi penuh dan layar tetap
// menyatakan seluruh barisnya kosong.
func Columns(ready inboxsurvey.Readiness) []Column {
	result := make([]Column, len(columns))
	copy(result, columns)

	if !ready.Complete() {
		return result
	}

	for i := range result {
		switch result[i].Key {
		case "reference_no":
			result[i].Available = true
			result[i].Note = "Dari POOLDATA.T_SURVEYORLIST.REFNO."
		case "location":
			result[i].Substitute = false
			result[i].Note = "Dari POOLDATA.T_SURVEYORLIST.RESCHEDULE_LOCATION — padanan " +
				"RescheduleLocation_1 yang digambar Pega."
		case "pic_loss_adjuster":
			result[i].Substitute = false
			result[i].Note = "Dari POOLDATA.T_SURVEYORLIST.ADJUSTER_PIC — padanan " +
				"ADJUSTERPIC_1 yang digambar Pega."
		}
	}
	return result
}

// TabInfo adalah satu tab beserta judul yang dilihat pengguna.
type TabInfo struct {
	Key   inboxsurvey.Tab
	Title string
	Note  string

	// Available menyatakan tab ini dapat dihitung dari data yang ada hari ini.
	//
	// Diambil dari domain, bukan ditulis tangan di sini — satu daftar yang disalin akan
	// tertinggal saat kolomnya tiba, dan tab yang tampak tersedia padahal bukan menghasilkan
	// daftar kosong yang terbaca sebagai "tidak ada pekerjaan".
	Available bool

	// UnavailableReason menyebut kenapa tab ini belum dapat dihitung, kosong bila ia bisa.
	UnavailableReason string
}

// tabs adalah ketujuh tab, dalam urutan tampilnya di section.
//
// Judulnya apa adanya dari `Section/InboxSurvey_section-Section.xml`, yang menuliskannya
// sebagai teks tebal: Outstanding · Invoice · Close · ALL · Not answered communication ·
// Not replied from ASM · Replied from ASM.
//
// Ketujuhnya TETAP digambar, termasuk yang belum dapat dihitung — lihat Column.Available.
var tabs = []TabInfo{
	{
		Key:   inboxsurvey.TabOutstanding,
		Title: "Outstanding",
		Note:  "Penugasan yang belum dikonfirmasi adjuster.",
	},
	{
		Key:   inboxsurvey.TabInvoice,
		Title: "Invoice",
		Note:  "Sudah dikonfirmasi dan berstatus Invoice Fee.",
	},
	{
		Key:   inboxsurvey.TabClose,
		Title: "Close",
		Note: "Berstatus Close Case. Di layar lama tab ini memakai status ALUR KERJA objek " +
			"survei, yang tidak tersedia di tabel penggantinya — lihat selisih terencana.",
	},
	{
		Key:   inboxsurvey.TabAll,
		Title: "ALL",
		Note: "Sudah dikonfirmasi adjuster. Namanya \"ALL\" di layar lama, tetapi ia bukan " +
			"seluruh baris.",
	},
	{
		Key:   inboxsurvey.TabNotAnswered,
		Title: "Not answered communication",
		Note:  "Ada pesan terbuka dari pihak lain yang belum Anda jawab.",
	},
	{
		Key:   inboxsurvey.TabNotReplied,
		Title: "Not replied from ASM",
		Note:  "Anda sudah mengirim pesan, ASM belum membalas.",
	},
	{
		Key:   inboxsurvey.TabReplied,
		Title: "Replied from ASM",
		Note:  "Pesan Anda sudah dibalas ASM.",
	},
}

// TabsInfo menyerahkan salinan daftar tab, beserta ketersediaan tiap tab.
//
// Ketersediaannya DIHITUNG di sini dari domain, bukan disimpan di dalam `tabs`. Menyimpannya
// berarti dua daftar yang harus disamakan dengan tangan, dan yang satu akan tertinggal.
func TabsInfo(ready inboxsurvey.Readiness) []TabInfo {
	result := make([]TabInfo, 0, len(tabs))
	for _, tab := range tabs {
		tab.Available = ready.TabAvailable(tab.Key)
		tab.UnavailableReason = ready.UnavailableReason(tab.Key)
		result = append(result, tab)
	}
	return result
}

// KPIColumn adalah satu judul kolom angka pada tab KPI.
type KPIColumn struct {
	Key   string
	Title string
}

// kpiColumns adalah kesembilan judul angka KPI, apa adanya dari section.
//
// Section menuliskannya dalam KAPITAL, dan bentuk itu dibawa: ia teks yang dilihat pengguna
// (`D-13`).
var kpiColumns = []KPIColumn{
	{Key: "penjadwalan_survey", Title: "PENJADWALAN SURVEY"},
	{Key: "immediate_advice", Title: "IMMEDIATE ADVICE"},
	{Key: "preliminary_advice", Title: "PRELIMINARY ADVICE"},
	{Key: "interim_report", Title: "INTERIM REPORT"},
	{Key: "update_progress", Title: "UPDATE PROGRESS"},
	{Key: "tanggapan_komunikasi", Title: "TANGGAPAN KOMUNIKASI"},
	{Key: "propose_adjustment", Title: "PROPOSE ADJUSTMENT"},
	{Key: "final_report", Title: "FINAL REPORT"},
	{Key: "nilai", Title: "NILAI"},
}

// KPIColumns menyerahkan salinan judul angka KPI.
func KPIColumns() []KPIColumn {
	result := make([]KPIColumn, len(kpiColumns))
	copy(result, kpiColumns)
	return result
}

// KPILeadingColumns adalah kolom KUNCI di depan kesembilan angka, menurut bentuk hasilnya.
//
// # Kenapa berubah-ubah, dan kenapa itu bukan pilihan tampilan
//
// Karena setiap bentuk menjawab pertanyaan yang berbeda, dan barisnya pun berbeda artinya:
//
//	per-adjuster         satu baris = seorang adjuster
//	per-adjuster-status  satu baris = seorang adjuster PADA SATU KATEGORI — dua baris per orang
//	per-tahun            satu baris = satu tahun
//	per-kuartal-tahun    satu baris = satu kuartal pada satu tahun — empat baris per tahun
//	detail               satu baris = satu BERKAS, dan angkanya mentah
//
// Judul "ADJUSTER" di atas kolom berisi "2026" akan membuat tahun terbaca sebagai nama orang
// yang kebetulan berupa angka. Dan tabel tanpa kolom kategori pada bentuk kedua akan
// menampilkan dua baris bernama sama dengan angka berbeda, tanpa satu pun keterangan kenapa.
func KPILeadingColumns(shape inboxsurvey.KPIShape) []KPIColumn {
	switch shape {
	case inboxsurvey.ShapePerAdjusterStatus:
		return []KPIColumn{
			{Key: "kelompok", Title: "ADJUSTER"},
			{Key: "status", Title: "STATUS SURVEY"},
		}

	case inboxsurvey.ShapePerYear:
		return []KPIColumn{{Key: "kelompok", Title: "TAHUN"}}

	case inboxsurvey.ShapePerQuarterYear:
		return []KPIColumn{
			{Key: "kelompok", Title: "TAHUN"},
			{Key: "kuartal", Title: "KUARTAL"},
		}

	case inboxsurvey.ShapeDetail:
		return []KPIColumn{
			{Key: "case_id", Title: "CASE ID"},
			{Key: "kelompok", Title: "ADJUSTER"},
			{Key: "bulan", Title: "BULAN"},
			{Key: "kuartal", Title: "KUARTAL"},
		}

	default:
		return []KPIColumn{{Key: "kelompok", Title: "ADJUSTER"}}
	}
}

// PlannedDifferences adalah perbedaan yang DISENGAJA terhadap layar Pega.
//
// Ia dikirim ke layar dan ditampilkan, bukan disimpan sebagai catatan teknis. `D-54`
// menetapkan selisih di luar 13 butir `P-5` menuntut persetujuan Work Owner tertulis, dan
// menyatakannya di layar itulah yang membuat keputusan itu terlihat oleh orang yang memakai
// layarnya — bukan hanya oleh orang yang membaca kodenya.
func PlannedDifferences() []string {
	return []string{
		"Sumber datanya BERGESER DUA KALI. Layar lama membaca objek kerja `Work-SurveyClaim` " +
			"di tabel engine Pega, yang dicabut dari pemakaian (Work Owner 2026-09-28). " +
			"Penggantinya yang pertama, `T_CLAIMLIST_ADMIN`, hanya memuat klaim yang tugasnya " +
			"berada di antrean Admin — sehingga survei yang SEDANG BERJALAN justru terbuang. " +
			"Sejak 2026-09-29 baris digerakkan `T_SURVEYORLIST`, header klaimnya diambil dari " +
			"`POOLDATA.T_CLAIM_PNC`, dan isian milik objek kerja survei dari tabel cermin " +
			"`POOLDATA.T_CLAIM_SURVEY_DATAPEGA`.",

		"Satu baris per BERKAS SURVEI, yaitu langkah TERAKHIR jejak perkembangannya. " +
			"`POOLDATA.T_SURVEYORLIST` menyimpan satu baris per perubahan status — diukur di " +
			"produksi 2026-09-29: 17.641 baris untuk 2.448 berkas survei, rata-rata 7,21 " +
			"langkah, terberat 176 langkah pada satu berkas. Menampilkan seluruhnya akan " +
			"membuat satu Claim No berulang tujuh kali rata-rata.",

		"Kolom Status ASM diambil dari `STS_SURVEY` pada langkah terakhir, bukan dari kolom " +
			"status adjuster milik objek kerja. Keduanya terbukti membawa domain yang sama — " +
			"sebaran nilainya memuat `Final Report`, `Invoice Fee`, dan `Close Case` — tetapi " +
			"kesamaannya belum pernah diuji baris per baris.",

		"Kolom Aging DIHITUNG, bukan dibaca. Layar lama membacanya dari kolom `AGING` pada " +
			"tabel datar; di sini ia umur janji survei sejak dicatat, dihitung terhadap " +
			"tanggal WIB. Keduanya berbeda jauh pada klaim lama yang surveinya baru " +
			"ditugaskan kemarin — dan yang dihitung di sini menjawab pertanyaan yang " +
			"sebenarnya diajukan adjuster.",

		"Halaman dipotong basis data, bukan setelah seluruh baris ditarik. Layar lama " +
			"menarik semuanya lalu menomori halamannya di memori dengan ukuran 15 baris; " +
			"di sini ukurannya 25 dan pemotongannya terjadi sebelum baris meninggalkan " +
			"basis data.",

		"Kotak cari Claim No dikerjakan SERVER. Layar lama menyaringnya di klipboard, " +
			"sehingga hasilnya hanya menyentuh halaman yang sedang terbuka.",
	}
}

// Limitations adalah keterbatasan yang berlaku hari ini dan akan hilang dengan sendirinya.
//
// Bedanya dengan PlannedDifferences: yang di atas adalah pilihan yang sudah diputuskan, yang
// di bawah adalah penghalang yang masih menunggu pihak lain. Keduanya dipisah supaya
// keterbatasan yang selesai dapat dihapus tanpa menyentuh keputusan yang masih berlaku.
func Limitations(ready inboxsurvey.Readiness) []string {
	if ready.Complete() {
		// Kelima kolom siap di portal ini. Keterbatasan yang menyangkut kolom kosong TIDAK
		// lagi berlaku dan dihapus — menyisakannya akan membuat layar menyatakan penghalang
		// yang sudah tidak ada, dan itu persis cacat yang diperbaiki 2026-10-07.
		return []string{
			"Membuka baris untuk mengerjakan surveinya belum tersedia. Di layar lama tautannya " +
				"membuka penugasan `Surveyor_Flow`, dan flow itu tidak ada di export — " +
				"`Flow/` hanya memuat empat, dan itu bukan salah satunya. Selama masa paralel " +
				"penugasan tetap dikerjakan di Pega.",

			"ENAM padanan kolom masih BELUM diuji ke basis data, dan seluruhnya dipakai hari " +
				"ini: Claim No (`CASEID_1`), Policy No (`POLICYNO`), Insured Name (`QQNAME`), " +
				"COB, PIC ASM (`USERTEKNIS_1`), dan Aging (`pxcreatedatetime` versus " +
				"`TGLINPUT`). Padanannya masuk akal, tetapi masuk akal bukan terbukti.",

			"Tab komunikasi di Pega hanya menampilkan berkas yang MASIH punya penugasan " +
				"terbuka — `BrowseCommunicationLostAdjuster` menyambung ke " +
				"`pc_assign_worklist` lewat `a.pzInsKey = b.pxrefobjectkey`. Modul ini tidak " +
				"membawa penyaring itu, sehingga tabnya dapat memuat lebih banyak baris.",

			"Pemeriksaan kewenangan menu belum ada (`TKT-F3-005`). Yang menjaga layar ini " +
				"sekarang adalah sesi, portal aktif, dan penyaring identitas surveyor.",
		}
	}

	return []string{
		"EMPAT dari tujuh tab dan SATU dari tiga belas kolom belum dapat diisi, dan sebabnya " +
			"BUKAN kolom yang tidak ada. Per 2026-09-30 `ADJUSTERACCEPT`, `REFNO`, dan " +
			"`PYSTATUSWORK` SUDAH ditambahkan ke `POOLDATA.T_SURVEYORLIST` — tetapi seluruh " +
			"17.641 barisnya masih kosong. Yang ditunggu sekarang adalah jalur PENGISIANNYA " +
			"dari Tim Pega, bukan `ALTER` berikutnya dari DBA.",

		"Kolom yang ADA tetapi KOSONG lebih berbahaya daripada kolom yang tidak ada, dan " +
			"itulah alasan tab-tab itu ditahan alih-alih dihidupkan. Tab Outstanding " +
			"menyaring `ADJUSTERACCEPT IS NULL`: dengan kolom yang seluruhnya kosong ia " +
			"menampilkan SELURUH antrean sebagai belum dikonfirmasi adjuster — terisi wajar, " +
			"angkanya masuk akal, isinya salah. Tab ALL dan Invoice sebaliknya, kosong sama " +
			"sekali, yang terbaca sebagai tidak ada pekerjaan.",

		"Berkas survei yang SUDAH ditutup atau dibatalkan di Pega masih ikut ditampilkan. " +
			"Penyaringnya sudah terpasang dan akan menyala SENDIRI begitu `PYSTATUSWORK` " +
			"terisi — tidak ada perubahan kode yang dibutuhkan. Diukur di produksi " +
			"2026-09-29: 1.070 dari 2.448 berkas survei sudah berstatus tutup.",

		"Kotak cari kehilangan satu kolom. Layar lama mencari pada Claim No DAN Reference " +
			"No; yang kedua ikut tertunda bersama kolomnya.",

		"Keempat kueri tab layar lama DITERIMA 2026-10-03, dan pemetaan kolom disusun " +
			"ulang terhadapnya. Sebelum itu ia disusun dari `BrowseLossAdjuster` dan " +
			"`BrowseInternalSurveyor` — dua kueri yang ternyata BUKAN penggerak grid ini — " +
			"sehingga lima hal keliru sekaligus: urutan daftar, asal Cause Of Loss, " +
			"Location, PIC Loss Adjuster, dan Status ASM.",

		"Kolom \"Status ASM\" menampilkan hal yang BERBEDA dari Pega, dan ini selisih yang " +
			"belum diputuskan. Pega menggambar `ASMSTATUS_1` (bernilai seperti `MEMBER`); " +
			"modul ini menggambar `STS_SURVEY` yang berisi progres adjuster. " +
			"`POOLDATA.T_SURVEYORLIST` tidak punya padanan `ASMSTATUS` sama sekali.",

		"ENAM padanan kolom masih BELUM diuji ke basis data, dan seluruhnya dipakai hari " +
			"ini: Claim No (`CASEID_1`), Policy No (`POLICYNO`), Insured Name (`QQNAME`), " +
			"Location (`RescheduleLocation_1` versus `LOCATION_SURVEY`), PIC Loss Adjuster " +
			"(`ADJUSTERPIC_1` versus `SURVEYOR_NAME`), dan Aging (`pxcreatedatetime` versus " +
			"`TGLINPUT`). Padanannya masuk akal, tetapi masuk akal bukan terbukti.",

		"Tab komunikasi di Pega hanya menampilkan berkas yang MASIH punya penugasan " +
			"terbuka — `BrowseCommunicationLostAdjuster` menyambung ke " +
			"`pc_assign_worklist` lewat `a.pzInsKey = b.pxrefobjectkey`. Modul ini tidak " +
			"membawa penyaring itu, sehingga tabnya dapat memuat lebih banyak baris.",

		"Membuka baris untuk mengerjakan surveinya belum tersedia. Di layar lama tautannya " +
			"membuka penugasan `Surveyor_Flow`, dan flow itu tidak ada di export — " +
			"`Flow/` hanya memuat empat, dan itu bukan salah satunya. Selama masa paralel " +
			"penugasan tetap dikerjakan di Pega.",

		"Pemeriksaan kewenangan menu belum ada (`TKT-F3-005`). Yang menjaga layar ini " +
			"sekarang adalah sesi, portal aktif, dan penyaring identitas surveyor.",
	}
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	Columns            []Column
	Tabs               []TabInfo
	KPIColumns         []KPIColumn
	PlannedDifferences []string
	Limitations        []string

	// DefaultTab adalah tab yang terbuka saat layar dibuka pertama kali.
	//
	// Ia tab bawaan yang BENAR-BENAR dapat dihitung, bukan DefaultTab domain apa adanya.
	// Bawaan domain adalah Outstanding, dan Outstanding termasuk yang belum dapat dihitung —
	// membuka layar di sana membuat kesan pertama setiap pengguna berupa layar tanpa isi, dan
	// kesan itu bertahan meski tab lain berisi.
	DefaultTab inboxsurvey.Tab

	// PageSize adalah ukuran halaman bawaan.
	PageSize int
}

// Metadata menyerahkan keterangan layar untuk SATU portal.
//
// # Kenapa ia butuh portal, padahal judul kolom sama di mana-mana
//
// Judulnya memang sama — yang berbeda adalah **ketersediaannya**. `D-75` menetapkan satu basis
// data per entitas, sehingga portal yang kolomnya sudah di-`ALTER` dan terisi dapat menyalakan
// tujuh tab, sementara portal sebelahnya baru tiga.
//
// Sampai 2026-10-07 fungsi ini tidak bergantung portal maupun basis data, dan ketersediaannya
// konstanta. Akibatnya layar menyatakan "seluruh barisnya masih kosong" pada portal yang justru
// sudah terisi penuh — ditemukan Work Owner, bukan oleh uji.
//
// Kegagalan membaca kesiapan TIDAK menjatuhkan keterangan layar: Readiness kosong berarti
// perilaku paling berhati-hati — tab ditahan beserta sebabnya, kolom memakai pengganti.
func (s *Service) Metadata(ctx context.Context, portal string) Metadata {
	var ready inboxsurvey.Readiness
	if repo, err := s.repoSelector(portal); err == nil && repo != nil {
		ready = repo.Readiness(ctx)
	}

	return Metadata{
		Columns:            Columns(ready),
		Tabs:               TabsInfo(ready),
		KPIColumns:         KPIColumns(),
		PlannedDifferences: PlannedDifferences(),
		Limitations:        Limitations(ready),
		DefaultTab:         ready.DefaultAvailableTab(),
		PageSize:           inboxsurvey.DefaultLimit,
	}
}

// Listed adalah satu halaman antrean beserta penyaring yang benar-benar dipakai.
type Listed struct {
	// Identity adalah identitas surveyor pemanggil.
	//
	// Dikirim balik ke layar, dan itu bukan kelebihan: seorang leader melihat pekerjaan
	// anggotanya, sehingga tanpa menyebut cakupannya pengguna tidak dapat menjelaskan kenapa
	// ia melihat baris atas nama orang lain.
	Identity inboxsurvey.SurveyorIdentity

	// Filter adalah penyaring setelah dinormalkan.
	//
	// Layar menggambar keadaan kotak cari, tab aktif, dan bilah halaman dari sini — bukan
	// dari isian yang ia kirim. Permintaan `batas=5000` dipangkas menjadi 100, dan tab yang
	// tidak dikenal dijatuhkan ke Outstanding.
	Filter inboxsurvey.Filter

	Page inboxsurvey.Page
}

// List mengambil satu halaman satu tab.
//
// # Urutannya: identitas, portal, jembatan, baru antrean
//
// Identitas diperiksa lebih dulu karena tanpanya tidak ada antrean yang dapat dibentuk sama
// sekali, dan memilih repo untuk permintaan yang pasti ditolak hanyalah satu perjalanan yang
// terbuang. Yang lebih penting: urutan ini membuat kegagalan sesi terbaca sebagai kegagalan
// sesi, bukan sebagai antrean kosong.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxsurvey.Caller,
	filter inboxsurvey.Filter,
) (Listed, error) {
	identity, repo, err := s.prepare(ctx, portalAlias, caller)
	if err != nil {
		return Listed{}, err
	}

	clean := filter.Normalize()

	// Kesiapan dibaca SEKALI di sini, bukan di dalam repo.
	//
	// Repo yang mengintip sendiri akan menjalankan dua kueri katalog tambahan pada SETIAP
	// permintaan, dan membuat setiap ujinya menuntut dua ekspektasi yang tidak ada
	// hubungannya dengan yang diuji.
	ready := repo.Readiness(ctx)

	page, err := repo.List(ctx, identity, clean, ready)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil antrean survei: %w", err)
	}

	return Listed{Identity: identity, Filter: clean, Page: page}, nil
}

// Counted adalah jumlah baris ketujuh tab beserta identitas yang dipakai menghitungnya.
type Counted struct {
	Identity inboxsurvey.SurveyorIdentity
	Counts   []inboxsurvey.TabCount
}

// Counts menghitung isi ketujuh tab.
//
// Ia rute TERSENDIRI, bukan bagian dari jawaban List. Alasannya: bilah tab tidak berubah saat
// pengguna berpindah halaman, sehingga menempelkannya pada setiap permintaan daftar akan
// menjalankan tujuh penjumlahan setiap kali tombol halaman ditekan.
func (s *Service) Counts(
	ctx context.Context,
	portalAlias string,
	caller inboxsurvey.Caller,
) (Counted, error) {
	identity, repo, err := s.prepare(ctx, portalAlias, caller)
	if err != nil {
		return Counted{}, err
	}

	counts, err := repo.Counts(ctx, identity, repo.Readiness(ctx))
	if err != nil {
		return Counted{}, fmt.Errorf("menghitung isi tab antrean survei: %w", err)
	}

	return Counted{Identity: identity, Counts: counts}, nil
}

// Scored adalah ringkasan KPI beserta penyaring yang benar-benar dipakai.
type Scored struct {
	Identity inboxsurvey.SurveyorIdentity
	Filter   inboxsurvey.KPIFilter
	Rows     []inboxsurvey.KPIRow
}

// KPI mengambil ringkasan KPI adjuster.
//
// # Ia disaring cakupan yang SAMA dengan antrean
//
// Itu keputusan, bukan kebetulan. `DETAIL_KPI_ADJUSTER` memuat penilaian SELURUH adjuster,
// dan di Pega layar KPI ini berada di dalam harness yang sama dengan antrean — yang berarti
// pemakainya sama. Menampilkan penilaian adjuster lain akan mengubah layar kerja menjadi
// papan peringkat yang tidak pernah diminta siapa pun.
// KPIYears mengisi dropdown "Tahun Kuartal".
//
// Ia TIDAK memeriksa isian panel: dropdown harus terisi SEBELUM pengguna memilih apa pun.
// Yang tetap diperiksa adalah identitas — daftar tahun pun disaring cakupan.
func (s *Service) KPIYears(
	ctx context.Context,
	portalAlias string,
	caller inboxsurvey.Caller,
) ([]string, error) {
	identity, repo, err := s.prepare(ctx, portalAlias, caller)
	if err != nil {
		return nil, err
	}

	years, err := repo.KPIYears(ctx, identity)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar tahun KPI: %w", err)
	}
	return years, nil
}

func (s *Service) KPI(
	ctx context.Context,
	portalAlias string,
	caller inboxsurvey.Caller,
	filter inboxsurvey.KPIFilter,
) (Scored, error) {
	identity, repo, err := s.prepare(ctx, portalAlias, caller)
	if err != nil {
		return Scored{}, err
	}

	clean := filter.Normalize()

	// Diperiksa SETELAH identitas, bukan sebelumnya.
	//
	// Pemanggil yang bukan surveyor harus mendapat jawaban itu lebih dulu — memberi tahu
	// "Status Survey belum dipilih" kepada orang yang memang tidak berhak membuka layar ini
	// akan membuatnya memilih, menekan Cari, lalu baru ditolak.
	if err := clean.Check(); err != nil {
		return Scored{}, err
	}

	rows, err := repo.KPI(ctx, identity, clean)
	if err != nil {
		return Scored{}, fmt.Errorf("mengambil ringkasan KPI adjuster: %w", err)
	}

	return Scored{Identity: identity, Filter: clean, Rows: rows}, nil
}

// prepare menjalankan tiga langkah yang sama bagi ketiga operasi pembaca.
//
// Ia ada supaya urutan pemeriksaannya TIDAK dapat berbeda antar operasi. Urutan yang berbeda
// menghasilkan galat yang berbeda untuk keadaan yang sama — dan layar yang menerima "sesi
// tidak terbaca" pada satu rute lalu "portal tidak dinyatakan" pada rute lain, untuk satu
// permintaan yang sama, tidak dapat menjelaskan apa pun kepada penggunanya.
func (s *Service) prepare(
	ctx context.Context,
	portalAlias string,
	caller inboxsurvey.Caller,
) (inboxsurvey.SurveyorIdentity, inboxsurvey.Repo, error) {
	if caller.Login == "" {
		return inboxsurvey.SurveyorIdentity{}, nil, inboxsurvey.ErrCallerUnknown
	}

	directory, err := s.directorySelector(portalAlias)
	if err != nil {
		return inboxsurvey.SurveyorIdentity{}, nil, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxsurvey.SurveyorIdentity{}, nil, err
	}

	identity, err := directory.ResolveSurveyor(ctx, caller.Login)
	if err != nil {
		// ErrNotSurveyor diteruskan APA ADANYA, tidak dibungkus menjadi galat lain dan tidak
		// diubah menjadi antrean kosong. Lapisan transport yang menerjemahkannya menjadi
		// jawaban yang dapat dibaca pengguna.
		if errors.Is(err, inboxsurvey.ErrNotSurveyor) {
			return inboxsurvey.SurveyorIdentity{}, nil, err
		}
		return inboxsurvey.SurveyorIdentity{}, nil,
			fmt.Errorf("menerjemahkan identitas surveyor: %w", err)
	}

	if len(identity.Scope) == 0 {
		// Identitas terbaca tetapi cakupannya kosong. Kueri antrean akan mengembalikan nol
		// baris, dan nol baris terbaca sebagai "tidak ada pekerjaan" — bukan sebagai data
		// master yang belum lengkap.
		return inboxsurvey.SurveyorIdentity{}, nil, inboxsurvey.ErrNotSurveyor
	}

	return identity, repo, nil
}
