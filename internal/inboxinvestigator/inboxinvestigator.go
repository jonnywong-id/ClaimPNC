// Package inboxinvestigator adalah inti modul Inbox Investigator.
//
// # Layar apa ini
//
// Menu `MENU_ID 48` "Inbox Investigator" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk
// harness `InboxInvestigator_Harness`. Judul yang dibaca pengguna di sistem lama adalah
// **"Inbox Investigator"**.
//
// Isinya **daftar pekerjaan yang menunggu Investigator** — klaim yang penugasannya berada
// di workbasket `InvestigatorPNC` dan belum selesai dikerjakan. Ia INBOX menurut keempat
// ciri `D-79`: barisnya pekerjaan, baris hilang setelah selesai, "hanya milik saya" adalah
// aturan kewenangan, dan barisnya punya tenggat.
//
// Modul INBOX pertama di aplikasi ini. Modul sebelumnya seluruhnya master data atau
// pencarian; pembedaan keduanya ditetapkan `D-79`, dan itulah yang menentukan bentuk modul
// ini — terutama kenapa penyaring "hanya workbasket InvestigatorPNC" bukan pilihan pengguna
// melainkan bagian dari kuerinya.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/InboxInvestigator_Harness-Harness.xml   pembungkus layar; judul; 2 activity
//	Section/InputInvestigator_Section-Section.xml   grid, kolom, pyPageSize=50, Operator
//	Report Definition/InboxRegisterCompliance_RD-RD.xml  20 kolom, 2 penyaring, MaxRecords
//	When/IsInvestigator-When.xml                    kewenangan membuka menu
//	RDB List/ExportDatainvestigator-SQL.xml         INVESTIGATOR_TF_DATE, rentang tanggal
//	Activity/ExportDataInvestigator-Act.xml         8 langkah ekspor, 13 judul kolom, syarat
//	RDB List/CountKlaimPUCL-SQL.xml                 bentuk gabungan work + workbasket
//	RDB List/ReminderPUCL-SQL.xml                   nama kolom fisik PC_ASM_FW_GCNMFW_WORK
//	Database/PEGA_CONVERT_JSONKLAIM_PNC.prc         jalur JSON klaim, bentuk waktu Pega
//	Database/m_menu_aplikasi_pnc.csv                MENU_ID 48 "Inbox Investigator"
//
// # Tiga hal dari layar lama yang TIDAK dibawa
//
//  1. **Grid kedua**. Section-nya memuat DUA grid (`…BBBB` dan `…BBBBB`) dengan kolom dan
//     parameter IDENTIK — sisa Save-As dari Inbox Compliance; yang kedua bahkan mengeja
//     "Nama Bisinis".
//
//  2. **Batas `pyMaxRecords = 500` yang senyap**. Lihat MaxRows.
//
//  3. **Langkah Report Definition pada activity ekspor**. `ExportDataInvestigator`
//     menjalankan `pxRetrieveReportData` berparameter `Operator = "InvestigatorPNC"` pada
//     langkah 2, lalu TIDAK PERNAH memakai hasilnya — berkas disusun dari halaman
//     `TempDataExport` milik RDB-List di langkah 5. Langkah yang tidak dipakai tidak
//     ditiru.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxinvestigator/          aturan modul + seam          ← paket ini
//	inboxinvestigator/usecase/  orkestrasi: buka daftar
//	inboxinvestigator/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxinvestigator/http/     lapisan transport modul ini  — handler, dto, rute
//
// # Yang BUKAN urusan paket ini
//
// Apa yang terjadi setelah sebuah baris dibuka. Di sistem lama baris menjalankan
// `SetAssignmentInboxPUCL_act` lalu `openAssignment` — mengambil penugasan dan membuka layar
// kerjanya. Layar kerja itu modul tersendiri yang belum ada, dan pencatatan hasil
// investigasi (`SetStatusInvestigator_Act`: `SurveyStatus=5`, `StatusClaim=1151`, kronologi
// TAT) ada di sana, bukan di sini. Modul ini mengetahui KEBERADAAN pekerjaannya — ia
// menyusun rujukannya, lihat Task.Reference — tetapi tidak mengubah satu baris pun.
package inboxinvestigator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Workbasket adalah antrean bersama yang isinya ditampilkan layar ini.
//
// Nilainya terbaca dari `Section/InputInvestigator_Section-Section.xml`, yang mengirimkannya
// ke Report Definition sebagai parameter:
//
//	<pyReportDefParams>
//	  <pyName>Operator</pyName>
//	  <pyValue>"InvestigatorPNC"</pyValue>
//
// Report Definition memakainya sebagai penyaring `newAssignPage.pxAssignedOperatorID =
// Param.Operator`.
//
// # Ia KONSTANTA, bukan penyaring yang dapat dipilih pengguna
//
// Di sistem lama nilainya tertanam di dalam section, sehingga layar ini hanya pernah
// menampilkan satu workbasket. Menjadikannya parameter permintaan berarti menyediakan cara
// membaca antrean peran lain — Compliance, RCL/PUCL, Komite — lewat endpoint Investigator.
// Itu bukan kesetaraan perilaku, melainkan kewenangan baru yang tidak pernah ada.
//
// Bahwa ia tertanam di kode dan bukan di master data adalah hal yang DISADARI dan
// bertentangan dengan `D-15`. Yang membuatnya tetap di sini: ia bukan nilai bisnis yang
// berubah menurut kebijakan, melainkan IDENTITAS layar ini — mengubahnya berarti layar ini
// menjadi layar lain. Bila kelak antrean investigator dipecah per lini bisnis, ia naik
// menjadi master data; sampai itu terjadi, memindahkannya hanya menambah satu tabel yang
// isinya satu baris dan tidak pernah berubah.
const Workbasket = "InvestigatorPNC"

// ResolvedWorkStatus adalah nilai `pyStatusWork` yang menandai pekerjaan sudah tuntas.
//
// Report Definition menyaring `.pyStatusWork != "Resolved-Completed"`, dan itulah yang
// membuat baris HILANG dari inbox setelah dikerjakan — ciri kedua Inbox pada `D-79`.
//
// Perhatikan bahwa penyaringnya "bukan selesai", bukan "sedang berjalan". Klaim yang
// berstatus `Resolved-Rejected` KARENA ITU TETAP TAMPIL. Itu perilaku sistem lama dan
// direplikasi apa adanya (`P-5`); apakah ia disengaja tidak dapat dibuktikan dari export.
const ResolvedWorkStatus = "Resolved-Completed"

// MaxRows membatasi jumlah baris yang dikembalikan satu permintaan.
//
// # Angkanya diwarisi, tetapi artinya berubah
//
// `pyMaxRecords = 500` pada `InboxRegisterCompliance_RD` MEMOTONG hasil tanpa memberi tahu
// siapa pun: baris ke-501 tidak pernah tampil, dan tidak ada apa pun di layar yang
// menyatakannya. Steering mencatat pola itu pada 54 dari 56 laporan
// (`15-NFR-PERFORMANCE-SCALABILITY.md`).
//
// Di sini angkanya dipertahankan — Work Owner memilih penyaringan dan paginasi dikerjakan
// PERAMBAN, dan mengirim seluruh antrean tanpa batas ke peramban akan mengubah layar
// menjadi tidak dapat dipakai jauh sebelum basis datanya keberatan. Yang BERUBAH adalah
// pemotongannya **dinyatakan**: Page.Truncated memberi tahu layar bahwa masih ada baris
// yang tidak terkirim, dan layar menyebutkannya kepada pengguna.
//
// Itu perbedaan yang menentukan. Batas yang diketahui adalah batas; batas yang senyap
// adalah data yang hilang.
const MaxRows = 500

// Task adalah satu baris inbox — satu klaim yang menunggu dikerjakan Investigator.
//
// # Kenapa namanya Task, bukan Claim
//
// Karena yang didaftar layar ini adalah PEKERJAAN, bukan klaim. Klaim yang sama dapat
// muncul di beberapa inbox pada waktu yang berbeda, dan yang membedakannya adalah penugasan
// — bukan klaimnya. `CONTEXT.md` menyebut satuan itu **Tugas**, dan `D-26` menetapkan ia
// selalu berada di Worklist atau di Workbasket. Yang ini di Workbasket.
//
// # Kesembilan kolom grid layar lama, pada urutannya
//
// Dibaca dari `Section/InputInvestigator_Section-Section.xml` beserta caption-nya. Kolom
// yang ADA di Report Definition tetapi TIDAK digambar grid — `SobName`, `GroupPanel`,
// `UserTeknis`, `PNCStatus`, `StatusClaim`, `isComplianceTransfer`, `TanggalBuatCompliance`
// — tidak dibawa: mengambil kolom yang tidak ada pembacanya hanya menambah lalu lintas,
// dan menampilkannya berarti mengarang kegunaan.
type Task struct {
	// Reference adalah `pzInsKey` — kunci teknis Pega, berbentuk
	// "ASM-FW-GCNMFW-WORK <nomor>" pada klaim warisan.
	//
	// Ia dibawa karena membuka pekerjaannya membutuhkannya, BUKAN untuk ditampilkan.
	// `03-CURRENT-ARCHITECTURE.md` §4.1 menyebut bocornya nama kelas Pega ke data bisnis
	// sebagai utang teknis, dan `D-22` menetapkan klaim terbitan sistem baru tidak pernah
	// menulis awalan itu lagi.
	//
	// Ia juga kunci yang menghubungkan baris ini ke T_CLAIM_OBJECTLIST dan
	// T_SURVEYORLIST; lihat berkas .sql.
	Reference string

	// CaseNumber adalah kolom "Nomor Case" — `pyID`.
	//
	// Inilah nomor yang disebut pengguna saat membicarakan sebuah pekerjaan. Dua format
	// hidup berdampingan selama masa paralel: `PNC-xxxx` dari Pega dan `PNCN.YY.xxxx` dari
	// sistem baru (`D-71`).
	CaseNumber string

	// PolicyNumber adalah kolom "No Polis" — `POLICYNO`.
	PolicyNumber string

	// InsuredName adalah kolom "Nama Tertanggung" — `QQNAME`.
	InsuredName string

	// ParticipantName adalah kolom "Nama Peserta" — `.ClaimData.ObjectList(1).ObjectName`,
	// yaitu nama objek pertanggungan PERTAMA pada klaim itu.
	//
	// Caption layar lamanya berbunyi "Nama Peserta", bukan "Nama Objek", dan itu dipakai
	// apa adanya (`D-13`). Penyebabnya terbaca: investigasi paling sering menyangkut lini
	// Personal Accident, tempat objek pertanggungan memang seorang peserta.
	//
	// # Satu klaim dapat punya banyak objek, dan hanya yang pertama yang tampil
	//
	// Itu perilaku sistem lama — Pega mengambil `ObjectList(1)`, elemen pertama page list.
	// Klaim berobjek banyak karena itu tampil seolah berobjek satu. Direplikasi apa adanya
	// (`P-5`); yang ditambahkan hanya kepastian URUTAN, lihat berkas .sql.
	ParticipantName string

	// BusinessName adalah kolom "Nama Bisnis" — `BUSINESSNAME`, lini bisnis klaim.
	//
	// Caption-nya muncul DUA KALI di section dengan ejaan berbeda — "Nama Bisnis" pada
	// judul kolom dan "Nama Bisinis" pada penyaring di atasnya. Salah ketik itu TIDAK
	// dibawa; yang dipakai ejaan yang benar.
	BusinessName string

	// BranchName adalah kolom "Nama Cabang" — `BRANCHNAME`, cabang yang menangani.
	BranchName string

	// AdminName adalah kolom "Nama Admin" — `pyOrigUserID`.
	//
	// Ia PEMBUAT kasus, bukan petugas yang sedang memegangnya. Pekerjaan di workbasket
	// memang belum bertuan (`D-26`) — itulah yang membuatnya antrean bersama — sehingga
	// tidak ada "sedang dikerjakan siapa" untuk ditampilkan.
	AdminName string

	// RegisteredAt adalah kolom "Tanggal Pendaftaran" — `pxCreateDateTime`.
	//
	// Nil bila kolomnya kosong. Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak
	// dapat dibedakan dari "belum diisi" saat ditampilkan, dan layar akan menuliskan
	// "01/01/0001" alih-alih tanda hubung.
	RegisteredAt *time.Time

	// SurveyDate adalah kolom KESEMBILAN grid, yang captionnya berbunyi
	// **"Lama Masuk Inbox"** — `.ClaimData.SurveyResults(1).SurveyDate`.
	//
	// # Captionnya menyebut durasi, isinya TANGGAL
	//
	// Ini bukan salah baca. Penelusuran sel per sel pada
	// `Section/InputInvestigator_Section-Section.xml` memasangkan kesembilan caption dengan
	// kesembilan sel datanya satu lawan satu, dan yang kesembilan berpasangan dengan
	// properti di atas:
	//
	//	Nomor Case          <-> .pyID                                   (tautan)
	//	No Polis            <-> .Policy.PolicyNo
	//	Nama Tertanggung    <-> .Policy.QQName
	//	Nama Peserta        <-> .ClaimData.ObjectList(1).ObjectName
	//	Nama Bisnis         <-> .Policy.Quotation.BusinessName
	//	Nama Cabang         <-> .Policy.Quotation.BranchName
	//	Nama Admin          <-> .pyOrigUserID
	//	Tanggal Pendaftaran <-> .pxCreateDateTime
	//	Lama Masuk Inbox    <-> .ClaimData.SurveyResults(1).SurveyDate   <-- INI
	//
	// Sebabnya terbaca: section ini Save-As dari inbox Compliance, tempat kolom bernama
	// sama memang berisi lama menunggu yang dirangkai `RDB List/GetSelisihJam_sql-SQL.xml`.
	// Di sini seseorang mengikat ulang selnya ke tanggal survei dan **captionnya tidak ikut
	// diganti** — bentuk yang sama dengan alias menyesatkan pada
	// `03-CURRENT-ARCHITECTURE.md` §4.2.
	//
	// Yang direplikasi adalah **isinya**: layar menampilkan tanggal survei di bawah caption
	// itu, persis sistem lama (`P-5`). Nama field di sini mengikuti ISI, bukan caption,
	// supaya pembaca kode berikutnya tidak menduga ada durasi yang harus dihitung.
	//
	// # Asalnya dokumen JSON klaim, dan versi pertama modul ini salah
	//
	// Nilainya dibaca dari `POOLDATA.JSON_KLAIM` jalur `$.SurveyResults[0].SurveyDate` —
	// jalur yang SAMA dengan properti Pega di atas. Versi pertama membacanya dari
	// `POOLDATA.T_SURVEYORLIST`, dan akibatnya kolom ini kosong di layar sementara Pega
	// menampilkan isinya. Pengukurannya ada di kepala inboxinvestigator.sql.
	//
	// Nil bila klaimnya belum punya tanggal survei.
	SurveyDate *time.Time

	// BusinessLine adalah kode lini bisnis klaim — `GROUPPANEL` pada `T_CLAIM_PNC`.
	//
	// Ia TIDAK digambar sebagai kolom grid; layar lama pun tidak menggambarnya. Ia dikirim
	// karena tab **Unggah Dokumen** pada formulir kerja memakainya untuk menyembunyikan
	// kategori yang tidak berlaku bagi lini itu — Travel menyembunyikan seluruh kategori
	// umum, dan Personal Accident menyembunyikan SALVAGE.
	//
	// Tanpa nilai ini, tab itu akan menggambar kategori yang di Pega tidak muncul.
	//
	// Kosong bila klaimnya tidak punya baris di `T_CLAIM_PNC`. Kosong berarti **tidak ada
	// yang disembunyikan**, bukan semuanya disembunyikan: kehilangan satu kode lini tidak
	// boleh membuat petugas kehilangan seluruh daftar dokumennya.
	BusinessLine string
}

// ExportRow adalah satu baris berkas **Export Data Investigation**.
//
// # Ia BUKAN baris inbox, dan itu bukan kelalaian penamaan
//
// Barisnya datang dari sumber yang sama sekali berbeda. Inbox membaca antrean workbasket;
// ekspor membaca `POOLDATA.T_CLAIM_PNC` menurut rentang `INVESTIGATOR_TF_DATE`, sehingga ia
// memuat klaim yang pernah ditransfer ke investigator — termasuk yang sudah selesai dan
// tidak lagi ada di antrean. Rinciannya di kepala kueri `investigator_export`.
//
// # Kesebelas nilai teks dikirim APA ADANYA
//
// `CheckBox*` bernilai `"true"`/`"false"`, `PasienTerdaftar` dan `KonfirmasiModelKwitansi`
// bernilai `"1"`/`"0"`, dan ketiganya ditulis ke berkas tanpa diterjemahkan — persis
// sistem lama, yang menyalin properti ke halaman ekspor tanpa satu pun transformasi
// (`P-5`). Satu-satunya yang diterjemahkan adalah HospitalKind; lihat field-nya.
type ExportRow struct {
	// InvestigatedAt adalah kolom "Tanggal Investigasi" — `T_CLAIM_PNC.INVESTIGATOR_TF_DATE`,
	// tanggal klaim dipindahkan ke investigator.
	//
	// Ia satu-satunya kolom berkas ini yang TIDAK berasal dari dokumen JSON: kueri sumber
	// `RDB List/ExportDatainvestigator-SQL.xml` mengambilnya langsung sebagai
	// `ASMBasTerritory`, dan karena ia kolom pertama halaman ekspor, ia menjadi kolom
	// pertama berkasnya.
	InvestigatedAt *time.Time

	// HospitalAddress — "Alamat RS Klinik", `SurveyList[0].AlamatRSKlinik`.
	HospitalAddress string

	// PaidByOtherInsurer — "Asuransi Lain", `SurveyList[0].CheckBoxAsuransiLain`.
	PaidByOtherInsurer string

	// PaidByPatient — "Pasien", `SurveyList[0].CheckBoxPasien`.
	PaidByPatient string

	// PaidByCompany — "Perusahaan", `SurveyList[0].CheckBoxPerusahaan`.
	PaidByCompany string

	// NoPayment — "Tidak ada pembayaran", `SurveyList[0].CheckBoxTidakadapembayaran`.
	//
	// Keempat field di atas adalah satu kelompok pilihan "siapa yang membayar" pada
	// formulir investigasi, bukan empat pertanyaan yang berdiri sendiri.
	NoPayment string

	// Investigated — "IsInvestigated", `SurveyList[0].IsInvestigated`.
	//
	// Nilainya `"1"` atau `"0"`, dan ia SEKALIGUS penyaring berkas ini; lihat
	// ExportFilter.Investigated.
	Investigated string

	// ReceiptConfirmation — "Konfirmasi Model Kwitansi",
	// `SurveyList[0].KonfirmasiModelKwitansi`.
	ReceiptConfirmation string

	// MedicalRecordNumber — "NoRekap Medis", `SurveyList[0].NoRekapMedis`.
	//
	// Ia nomor rekam medis, dan berkas yang memuatnya adalah berkas berisi DATA MEDIS.
	// `FR-R2` membatasi akses data medis pada peran Analyst Doctor dan RCL Dokter; tombol
	// ekspor ini berada di layar Investigator, dan pembatasannya ditegakkan middleware menu
	// yang sama dengan layarnya (`D-59`).
	MedicalRecordNumber string

	// PhoneCalled — "NoTelp DiHubungi", `SurveyList[0].NoTelpDiHubungi`.
	PhoneCalled string

	// PatientRegistered — "Pasien Terdaftar", `SurveyList[0].PasienTerdaftar`.
	PatientRegistered string

	// Remarks — "Remaks", `SurveyList[0].Remaks`.
	//
	// Ejaan "Remaks" dipertahankan karena itulah judul kolom yang dibaca pengguna di berkas
	// lama (`Activity/ExportDataInvestigator-Act.xml:2332`), dan pengguna mencocokkan berkas
	// baru dengan berkas lama kolom per kolom. Yang diperbaiki adalah nama field di kode
	// ini, bukan judul yang dilihat orang.
	Remarks string

	// HospitalKindCode adalah `SurveyList[0].SelectRS` APA ADANYA — `"1"`, `"0"`, atau
	// kosong.
	//
	// Kolom "Jenis Rumah Sakit" pada berkas TIDAK menuliskan kode ini melainkan hasil
	// terjemahannya; lihat HospitalKindLabel. Kodenya yang disimpan di sini, bukan
	// labelnya, supaya satu-satunya tempat terjemahan itu hidup adalah method di bawah.
	HospitalKindCode string
}

// HospitalKindLabel menerjemahkan kode jenis rumah sakit menjadi teks berkas.
//
// SATU-SATUNYA kolom ekspor yang diterjemahkan, dan terjemahannya ditulis apa adanya di
// activity lama (`Activity/ExportDataInvestigator-Act.xml:2089`):
//
//	@if(SelectRS == "1", "Rumah Sakit", "NON Rumah Sakit")
//
// Perhatikan bentuknya: nilai apa pun selain `"1"` — termasuk KOSONG — menghasilkan
// "NON Rumah Sakit". Itu direplikasi apa adanya (`P-5`), meski artinya klaim yang
// pertanyaannya belum dijawab tampil seolah sudah dijawab "bukan rumah sakit".
//
// Ia method domain, bukan baris di lapisan transport: ia aturan yang diwarisi dari sistem
// lama, dan aturan yang hidup di satu tempat adalah inti alasan migrasi ini dikerjakan
// (`04-FUTURE-ARCHITECTURE.md` §2).
//
// # "NON Rumah Sakit" di sini, "Non Rumah Sakit" di formulir — KEDUANYA benar
//
// Teks di sini disalin dari `@if` pada activity ekspor. Teks pada radio "Tempat Kejadian"
// di formulir disalin dari caption rule `Property/SelectRS-property.xml`, yang mengejanya
// **"Non Rumah Sakit"**.
//
// Keduanya memang berbeda di Pega, dan perbedaannya dipertahankan: berkas ekspor dibanding
// kolom per kolom dengan berkas lama (`P-5`), sedangkan layar dibandingkan dengan layar
// lama (`D-13`). Menyeragamkannya akan merusak salah satu perbandingan itu.
func (r ExportRow) HospitalKindLabel() string {
	if r.HospitalKindCode == "1" {
		return "Rumah Sakit"
	}
	return "NON Rumah Sakit"
}

// ExportFilter adalah ketiga kendali di kepala layar lama yang memang milik tombol ekspor.
//
// Ketiganya `pyVisible = ALWAYS` di section, dan ketiganya BUKAN penyaring grid — kueri
// yang memakainya adalah kueri ekspor, bukan kueri daftar. Lihat Filter.
type ExportFilter struct {
	// From adalah isian "Dari" — `TempInvestigate.DateOfLoss`.
	//
	// Namanya di sistem lama menyesatkan: property-nya bernama DateOfLoss, tetapi yang
	// disaringnya `INVESTIGATOR_TF_DATE`. Alias menyesatkan yang sama dengan
	// `03-CURRENT-ARCHITECTURE.md` §4.2; di sini namanya mengikuti ARTINYA.
	From time.Time

	// To adalah isian "Sampai" — `TempInvestigate.DateReceived`, hari terakhir yang IKUT.
	//
	// Kueri lama memakai `trunc(kolom) <= to_date(…)`, sehingga hari yang diketik ikut
	// terbawa seluruhnya. Perilaku itu dipertahankan; cara mencapainya berubah, lihat
	// kepala kueri `investigator_export`.
	To time.Time

	// Investigated adalah dropdown "Pilih Investigation" —
	// `TempInvestigateChose.IsInvestigated`, bernilai `"1"` atau `"0"`.
	//
	// # Ia WAJIB, dan itu perilaku sistem lama
	//
	// Langkah penyalinan di activity lama bersyarat
	// `SurveyList(1).IsInvestigated == TempInvestigateChose.IsInvestigated`. Tanpa pilihan,
	// syarat itu membandingkan dengan teks kosong dan berkasnya terbit tanpa satu baris pun
	// terisi. Di sini ketiadaan pilihan ditolak sebagai galat yang terbaca, bukan dijawab
	// dengan berkas kosong tanpa keterangan.
	//
	// Daftar pilihannya tidak ada di export — rule Property `IsInvestigated` termasuk ±242
	// rule yang hilang (`R-16`). Kedua nilainya terbaca dari data: `"1"` dan `"0"`.
	Investigated string
}

// InvestigatedYes dan InvestigatedNo adalah kedua nilai sah ExportFilter.Investigated.
const (
	InvestigatedYes = "1"
	InvestigatedNo  = "0"
)

// ErrExportFilterInvalid menandai permintaan ekspor yang penyaringnya tidak dapat dipakai.
//
// Ia galat KLIEN, bukan galat server: yang salah adalah apa yang diminta, bukan kemampuan
// menjawabnya. Lapisan transport memetakannya menjadi 400, dan pesannya ditulis supaya
// terbaca pengguna — bukan supaya terbaca pengembang.
var ErrExportFilterInvalid = errors.New("penyaring ekspor tidak sah")

// Validate memastikan ketiga kendali ekspor dapat dipakai.
//
// # Ketiganya diperiksa, bukan dibiarkan jatuh ke kueri
//
// Rentang terbalik menghasilkan berkas kosong tanpa satu pun keterangan, dan pilihan
// investigasi yang kosong menghasilkan hal yang sama — keduanya persis perilaku sistem lama
// yang membuat pengguna menyimpulkan datanya tidak ada. Memeriksanya di sini mengubah
// kebingungan menjadi kalimat.
func (f ExportFilter) Validate() error {
	if f.From.IsZero() {
		return fmt.Errorf("%w: tanggal \"Dari\" wajib diisi", ErrExportFilterInvalid)
	}
	if f.To.IsZero() {
		return fmt.Errorf("%w: tanggal \"Sampai\" wajib diisi", ErrExportFilterInvalid)
	}
	if f.To.Before(f.From) {
		return fmt.Errorf(
			"%w: tanggal \"Sampai\" lebih awal daripada \"Dari\"", ErrExportFilterInvalid)
	}
	switch f.Investigated {
	case InvestigatedYes, InvestigatedNo:
	default:
		return fmt.Errorf(
			"%w: pilihan \"Pilih Investigation\" wajib diisi", ErrExportFilterInvalid)
	}
	return nil
}

// MaxExportRows membatasi banyaknya baris pada satu berkas ekspor.
//
// Sistem lama tidak punya batas di sini — `ExportDatainvestigator-SQL` tidak memuat
// `pyMaxRecords` maupun ROWNUM, sehingga rentang tanggal yang lebar menarik seluruh klaim
// yang pernah ditransfer ke investigator sekaligus.
//
// Angka ini karena itu bukan aturan bisnis melainkan penjaga, dan nilainya disamakan dengan
// modul ekspor lain supaya aplikasi ini tidak punya dua batas berbeda tanpa alasan. Berkas
// yang menyentuhnya diberi tanda di baris terakhir — bukan dipotong dalam diam, yang persis
// cacat `pyMaxRecords = 500` yang modul ini hindari di tempat lain.
const MaxExportRows = 50_000

// Filter mempersempit daftar yang dibaca layar.
//
// # Yang TIDAK ada di sini, dan itu disengaja
//
// **Tidak ada penyaring workbasket.** Ia konstanta; lihat Workbasket.
//
// **Tidak ada penyaring rentang tanggal**, meski layar lama punya isian "Dari" dan "Sampai".
// Keduanya milik tombol **Export Data Investigation**, bukan milik grid: kueri yang
// memakainya adalah `RDB List/ExportDatainvestigator-SQL.xml`, dan ia menyaring
// `INVESTIGATOR_TF_DATE` — kolom yang tidak dibaca grid sama sekali. Menerapkannya pada
// daftar berarti menyaring yang tampil dengan penyaring yang dibuat untuk hal lain.
//
// **Tidak ada penyaring status.** Report Definition menyaring `pyStatusWork` dengan nilai
// tetap, bukan dengan pilihan pengguna; lihat ResolvedWorkStatus.
type Filter struct {
	// Keyword mempersempit daftar pada Nomor Case, No Polis, Nama Tertanggung, Nama
	// Peserta, Nama Bisnis, Nama Cabang, dan Nama Admin.
	//
	// DITAMBAHKAN terhadap sistem lama, yang menyaring di peramban lewat kotak isian per
	// kolom pada kepala grid. Kosong berarti tanpa penyaring.
	//
	// # Layar TIDAK memakainya hari ini
	//
	// Work Owner memilih penyaringan dikerjakan peramban (2026-09-23), sehingga layar
	// memakai pencarian bawaan `DataTable` atas baris yang sudah di tangan — sama seperti
	// seluruh layar master. Penyaring ini tetap disediakan supaya perpindahan ke
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
	//
	// Inilah satu-satunya perbedaan yang disengaja terhadap `pyMaxRecords = 500` sistem
	// lama, yang memotong tanpa memberi tahu siapa pun. Lihat MaxRows.
	Truncated bool
}

// Repo adalah seam ke penyimpanan inbox investigator SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat
// kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas,
// dan memang tidak boleh ada.
//
// # Seam INI hanya membaca — yang menulis seam tersendiri
//
// Repo tidak punya satu pun method yang menulis, dan itu disengaja. Mencatat hasil
// investigasi beserta perpindahan klaim ke Analyst hidup di `InvestigationRepo`, seam
// terpisah dengan penyimpanannya sendiri (`POOLDATA.TC_PNC_INVESTIGASI`).
//
// Memisahkannya menjaga satu hal: daftar antrean tetap dapat dibaca meski penyimpanan hasil
// investigasi belum tersedia di sebuah entitas — persis keadaan tiga portal yang tabelnya
// belum dibuat. Mengambil pekerjaan dari antrean tetap milik modul Penugasan, bukan di sini.
type Repo interface {
	// List mengembalikan pekerjaan yang menunggu di workbasket Investigator, terpotong
	// pada MaxRows.
	List(ctx context.Context, filter Filter) (Page, error)

	// Export mengembalikan baris berkas Export Data Investigation, terpotong pada
	// MaxExportRows.
	//
	// Ia TETAP sebuah pembacaan — menerbitkan berkas tidak mengubah satu baris pun.
	Export(ctx context.Context, filter ExportFilter) ([]ExportRow, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan antrean
// pekerjaan satu badan hukum — lengkap dengan nama tertanggung dan nama peserta — kepada
// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// TANPA seam Clock, dan ketiadaannya adalah hasil koreksi.
//
// Modul ini sempat memilikinya, karena kolom "Lama Masuk Inbox" dibaca sebagai durasi yang
// dihitung terhadap sekarang. Penelusuran sel per sel membuktikan kolom itu **menampilkan
// tanggal survei apa adanya** (lihat Task.SurveyDate), sehingga tidak ada satu pun nilai di
// modul ini yang bergantung pada jam dinding.
//
// Seam yang tidak ada yang bervariasi di baliknya bukan seam — ia hanya bahan rakitan yang
// harus diisi tanpa pernah dipakai.
