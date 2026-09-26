// Package mastergroupingsparepart adalah inti modul Master Grouping Sparepart.
//
// # Apa yang dimodelkan di sini
//
// Penautan sebuah **suku cadang alat berat** ke sebuah **panel bodi** pada sebuah
// **kendaraan**. Satu baris menjawab satu pertanyaan: suku cadang bernomor ini, pada panel
// ini dan sisi ini, dipakai kendaraan bernomor rangka ini.
//
// Baris-baris yang menunjuk kendaraan yang sama dikumpulkan di bawah satu **Nomor Grup**.
// Itulah arti kata "grouping" pada nama layarnya — bukan penggolongan suku cadang, melainkan
// pengelompokan menurut kendaraan.
//
// Ia master keempat dari keluarga alat berat, setelah Master Bengkel, Master Panel, dan
// Master Sparepart. Keempatnya berbagi alur persetujuan tiga tab yang sama.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/GroupingSparePart_HE-Harness.xml                  layar, MENU_ID 32
//	Section/MasterGroupingSparepartHE-Section.xml             judul + tombol Tambah
//	Section/PNCMasterGroupingSparepartHE-Section.xml          3 tab + pencarian
//	Section/MasterGroupingSparepartHEApproval-Section.xml     grid + form, 11 isian berlabel
//	Section/MasterGroupingSparepartHEApprove-Section.xml      tab disetujui
//	Section/MasterGroupingSparepartHEReject-Section.xml       tab ditolak
//	Section/ApprovalPNCMasterGroupingSparepartHE-Section.xml  persetujuan di Inbox Manager
//	RDB List/GetDataMasterGrouping-SQL.xml                    daftar + pencarian duplikat
//	RDB List/UpdateGroupingSparepartHE-SQL.xml                simpan lewat procedure
//	RDB List/GetNewNoGroup-SQL.xml                            nomor grup baru
//	RDB List/GetNoGroup-SQL.xml                               nomor grup yang sudah ada
//	RDB List/CountMasterGrupSparepartManager-SQL.xml          pencacah antrean persetujuan
//	RDB List/GetDataSisiPanel-SQL.xml                         pilihan Sisi
//	RDB List/BrowseTypeHE_Sql-SQL.xml                         pilihan Tipe Kendaraan
//	Database/PEGA_M_GROUPING_SPAREPART_HE.prc                 pembentukan ID
//	Activity/UpdateGroupingSparepartHE_act-Act.xml            urutan langkah simpan
//	Activity/SetValueGroupingMasterHE-Act.xml                 pemuatan baris ke form
//	Activity/SetDataSparepart-Act.xml                         pengisian otomatis dari sparepart
//	Activity/GetSisiPanel-Act.xml                             penyandian sisi
//	Database/m_menu_aplikasi_pnc.csv                          MENU_ID 32
//
// # Layar ini salinan layar Master Sparepart, dan itu meninggalkan jejak di mana-mana
//
// `Harness/GroupingSparePart_HE-Harness.xml` menyebut asalnya sendiri:
//
//	pzOriginalInstanceKey = RULE-HTML-HARNESS DATA-PORTAL SPAREPART_HE #...
//
// Akibatnya kolom grouping dipetakan ke properti klipboard milik Master Sparepart, dan
// pemetaannya tidak masuk akal sama sekali. `RDB List/GetDataMasterGrouping-SQL.xml`:
//
//	A.NAMA_PANEL           AS "PANJANG"     <- label layar "Nama Panel"
//	A.SISI_PANEL           AS "LEBAR"       <- label layar "Sisi"
//	B.NO_RANGKA            AS "TINGGI"      <- label layar "No Rangka"
//	B.TIPE                 AS "QTY_PESAN"   <- label layar "Tipe Kendaraan"
//	A.GROUPING_DGN_RANGKA  AS "BERAT"       <- label layar "Grouping Dengan No Rangka"
//	A.CATATAN              AS "MAX_STOCK"   <- label layar "Catatan"
//	A.ID_PANEL             AS "MIN_STOCK"   <- isian tersembunyi
//	A.NO_GROUP_RANGKA      AS "pyID"        <- nomor grup
//
// Nama panel benar-benar tersimpan di properti bernama `PANJANG`, dan catatan di properti
// bernama `MAX_STOCK`. Itu persis bentuk utang yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat,
// dan TIDAK SATU PUN dibawa: penamaan di sini mengikuti isinya (`D-19`, `D-80`).
//
// Jejak yang sama ada di procedure-nya. `Database/PEGA_M_GROUPING_SPAREPART_HE.prc:37`
// menuliskan pesan galat `'UPDATE PEGA_PANEL_HE_CLAIM Error : '` — nama procedure LAIN,
// tertinggal dari salin-tempel.
//
// # Empat hal yang membedakannya dari Master Sparepart
//
//  1. **DUA tabel, bukan satu.** `SPAREPART_HE_VIN_KEY` sebagai induk dan
//     `SPAREPART_HE_VIN_GROUP` sebagai pendampingnya; lihat banner pada berkas .sql.
//  2. **Kunci alami EMPAT kolom**, bukan tiga kolom yang masing-masing unik sendiri; lihat
//     NaturalKey.
//  3. **ID bukan dari sequence**, melainkan `MAX(ID)+1`; lihat IDSource.
//  4. **Lima isian diturunkan dari Master Sparepart**, bukan diketik; lihat Input.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package mastergroupingsparepart

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ApprovalStatus adalah posisi sebuah baris dalam alur persetujuan.
//
// Nilainya tetap "0", "1", "2" seperti kolom `APPROVAL` pada
// `POOLDATA.SPAREPART_HE_VIN_KEY`: tabelnya masih dibaca sistem lama selama masa paralel
// (ADR-0004), sehingga mengubah sandi nilainya akan membuat kedua sistem membaca baris yang
// sama secara berbeda.
//
// Ketiga sandinya terbaca dari `Section/PNCMasterGroupingSparepartHE-Section.xml`, yang
// memuat tiga section dengan nilai penyaring yang berbeda, dan dikuatkan
// `RDB List/CountMasterGrupSparepartManager-SQL.xml` yang mencacah `A.APPROVAL=0`.
//
// Sandinya kebetulan sama dengan Master Bengkel, Master Panel, Master Sparepart, Master Auto
// Claim, dan Master Rekening. Keenamnya sengaja TIDAK dipakai bersama: tabelnya berbeda, dan
// tipe bersama membuat perubahan di satu master menyeret master lain.
type ApprovalStatus string

const (
	// StatusPending — diajukan, belum diputuskan.
	//
	// Berbeda dari Master Sparepart yang menetapkannya tanpa syarat,
	// `Activity/UpdateGroupingSparepartHE_act` menetapkan
	// `TempSparepart.APPROVAL := Param.Approval` — nilainya datang dari pemanggil. Layar
	// penyuntingannya selalu mengirim "0"; layar persetujuan yang mengirim "1" atau "2".
	// Di modul ini keduanya dipisah menjadi dua jalur; lihat usecase.Service.Decide.
	StatusPending ApprovalStatus = "0"

	// StatusApproved — disetujui. Hanya baris berstatus ini yang dipakai sistem hilir.
	StatusApproved ApprovalStatus = "1"

	// StatusRejected — ditolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
//
// Layar lama tidak memberi nama pada ketiga tabnya — nama sectionnya `…Approval`,
// `…Approve`, dan `…Reject`. Teks di sini mengikuti kata yang sama, dengan "Waiting
// Approval" untuk yang pertama supaya terbaca sebagai antrean dan bukan sebagai tindakan
// (`D-13`). Sama persis dengan Master Bengkel, Master Panel, dan Master Sparepart.
func (s ApprovalStatus) Label() string {
	switch s {
	case StatusPending:
		return "Waiting Approval"
	case StatusApproved:
		return "Approve"
	case StatusRejected:
		return "Reject"
	default:
		return ""
	}
}

// Known menyatakan status ini termasuk salah satu dari tiga yang sah.
func (s ApprovalStatus) Known() bool {
	return s == StatusPending || s == StatusApproved || s == StatusRejected
}

// Grouping adalah satu baris grouping sparepart.
//
// Ia gabungan satu baris `POOLDATA.SPAREPART_HE_VIN_KEY` dengan pendampingnya di
// `POOLDATA.SPAREPART_HE_VIN_GROUP`, persis seperti yang dibaca
// `RDB List/GetDataMasterGrouping-SQL.xml`. Label yang dipakai layar dibaca dari
// `pyLabelFieldValue` pada `Section/MasterGroupingSparepartHEApproval-Section.xml`.
//
// # Lima kolom yang TIDAK diketik pengguna
//
// PartName, CategoryID, TypeID, PartCode, dan ProductionDate seluruhnya DITURUNKAN dari
// Master Sparepart begitu Nomor Sparepart diisi. `Activity/SetDataSparepart-Act.xml`
// menyalin kelimanya dari hasil pencarian atas `POOLDATA.SPAREPART_HE`, dan menolak dengan
// "Data Sparepart tidak ditemukan" bila nomornya tidak ada.
//
// Kelimanya tetap DISIMPAN di tabel ini, bukan dibaca ulang lewat join setiap kali. Itu
// meniru sistem lama apa adanya — dan itu pula yang membuat baris lama tetap terbaca
// sekalipun sparepart asalnya kemudian diubah namanya.
//
// # Seluruh isian bertipe TEKS
//
// Tidak ada satu pun nilai uang maupun perhitungan di modul ini, sehingga tidak ada satu pun
// titik yang membulatkan angka (`I-12`, `D-51`). Nomor rangka dan nomor sparepart pun teks:
// keduanya pengenal, bukan bilangan — nomor rangka kendaraan memuat huruf.
type Grouping struct {
	// ID adalah kolom ID pada tabel induk — kunci baris ini.
	//
	// Ia TIDAK diketik pengguna. `Database/PEGA_M_GROUPING_SPAREPART_HE.prc:11` menerbitkannya
	// sebagai `NVL(MAX(ID),0) + 1`; lihat IDSource.
	ID string

	// PartNumber adalah kolom NO_PART — "Nomor Sparepart".
	//
	// Satu-satunya isian identitas sparepart yang benar-benar DIKETIK. Ia pula anggota
	// pertama kunci alami; lihat NaturalKey.
	PartNumber string

	// PartName adalah kolom NAMA_PART — "Nama Sparepart". Diturunkan; lihat catatan pada tipe.
	PartName string

	// CategoryID adalah kolom KATEGORI_SPART — "Kategory Sparepart".
	//
	// Perhatikan ejaan labelnya: "Kategory" dengan y, apa adanya di layar lama. Yang
	// tersimpan adalah nilai yang disalin dari `SPAREPART_HE.KATEGORI_SPART`, yaitu
	// PART_CATEGORY_ID — bukan namanya. Diturunkan; lihat catatan pada tipe.
	CategoryID string

	// TypeID adalah kolom TIPE_SPART — "Type Sparepart".
	//
	// Berisi PART_SECTION_ID, disalin dari `SPAREPART_HE.TIPE_SPART`. Diturunkan.
	TypeID string

	// PartCode adalah kolom KODE_PART. Diturunkan, dan TIDAK digambar di layar.
	//
	// Ia tidak punya label pada `Section/MasterGroupingSparepartHEApproval`, tetapi
	// `Activity/SetDataSparepart` mengisinya dan `Activity/UpdateGroupingSparepartHE_act`
	// langkah 1 memakainya sebagai penanda bahwa pengisian otomatis belum berjalan.
	PartCode string

	// ProductionDate adalah kolom PROD_DATE. Diturunkan, dan TIDAK digambar di layar.
	//
	// TEKS, bukan tanggal, mengikuti Master Sparepart yang merendernya `pxTextInput` tanpa
	// `pyDateTimeFormat` sama sekali.
	ProductionDate string

	// PanelID adalah kolom ID_PANEL — isian TERSEMBUNYI di layar lama.
	//
	// Ia diisi otomatis saat pengguna memilih Nama Panel: autocomplete-nya membaca
	// `Report Definition/BrowseMasterPanel_HE_RD-RD.xml` dan menyalin `.ID_PANEL` ke
	// `TempSparepart.MIN_STOCK` lewat `pySetValueOnSelect`.
	//
	// Ia yang kemudian dipakai mencari pilihan Sisi; lihat SideKey.
	PanelID string

	// PanelName adalah kolom NAMA_PANEL — "Nama Panel".
	//
	// Isinya nama panel dari `POOLDATA.PANEL_HE.NAME`, bukan nama lokasi. Anggota kedua kunci
	// alami.
	PanelName string

	// PanelSide adalah kolom SISI_PANEL — "Sisi".
	//
	// Sandinya sama dengan `POOLDATA.LOKASI_PANEL_HE.SISI_PANEL`: "-", "1" KIRI, "2" KANAN.
	// Anggota keempat kunci alami. Lihat Side pada lookup.go.
	PanelSide string

	// ChassisNumber adalah kolom NO_RANGKA — "No Rangka".
	//
	// PERHATIKAN: kolom bernama sama ADA DI KEDUA TABEL, dan sistem lama membaca yang satu
	// tetapi memeriksa yang lain — `GetDataMasterGrouping` memilih `B.NO_RANGKA` sementara
	// pemeriksaan duplikat menyaring `A.NO_RANGKA`. Lihat banner pada berkas .sql.
	//
	// Anggota ketiga kunci alami.
	ChassisNumber string

	// VehicleType adalah kolom TIPE pada tabel pendamping — "Tipe Kendaraan".
	//
	// Nilainya dipilih dari `branddetail`; lihat VehicleType pada lookup.go.
	VehicleType string

	// GroupWithChassis adalah kolom GROUPING_DGN_RANGKA — "Grouping Dengan No Rangka".
	//
	// Bila diisi, baris ini BERGABUNG ke grup kendaraan yang nomor rangkanya disebut di sini,
	// alih-alih membuka grup baru. Nilainya sebuah NOMOR RANGKA, bukan nomor grup — dan itu
	// yang membuat labelnya terbaca benar.
	//
	// Kosong berarti baris ini membuka grup baru; lihat GroupNumber.
	GroupWithChassis string

	// GroupNumber adalah kolom NO_GROUP_RANGKA — nomor grup kendaraan.
	//
	// TIDAK diketik dan TIDAK digambar di layar: ia diterbitkan penyimpanan. Di Pega ia
	// tersimpan pada properti `pyID`, properti bawaan Pega yang dipinjam untuk memikul nomor
	// grup — satu lagi jejak salin-tempel.
	//
	// Lihat ComposeGroupNumber untuk bentuknya.
	GroupNumber string

	// Note adalah kolom CATATAN — "Catatan".
	Note string

	// Status adalah kolom APPROVAL pada tabel induk.
	Status ApprovalStatus
}

// NaturalKey adalah keempat kolom yang bersama-sama harus unik.
//
// Asalnya `Activity/UpdateGroupingSparepartHE_act-Act.xml` langkah 3, yang merangkai
// penyaring pencarian duplikat sebagai TEKS SQL:
//
//	TempData.ID := "AND A.NO_PART= '"      + TempSparepart.NO_SPART + "'"
//	             + "AND A.NAMA_PANEL ='"   + TempSparepart.PANJANG  + "'"
//	             + "AND A.NO_RANGKA ='"    + TempSparepart.TINGGI   + "'"
//	             + " AND A.SISI_PANEL= '"  + TempSparepart.LEBAR    + "'"
//
// Bila ditemukan, langkah 15 menampilkan pesan "Data sudah ada".
//
// # Tiga hal yang terbaca dari potongan itu, dan ketiganya penting
//
//  1. **Kuncinya empat kolom bersama-sama**, bukan empat kolom yang masing-masing unik.
//     Satu nomor sparepart boleh muncul berkali-kali — memang itu gunanya layar ini.
//  2. **Nomor rangka yang diperiksa adalah `A.NO_RANGKA`**, kolom pada tabel INDUK,
//     sedangkan yang ditampilkan daftar adalah `B.NO_RANGKA` pada tabel pendamping.
//  3. **Nilainya dirangkai langsung ke teks SQL.** Satu tanda kutip tunggal pada nomor
//     sparepart sudah cukup untuk mengubah arti kuerinya. Tidak dibawa: seluruh nilai di
//     modul ini menempuh parameter binding tanpa perkecualian
//     (`08-TECHNICAL-STRATEGY.md` §4.3).
//
// Ia tipe tersendiri, bukan empat argumen berjajar, supaya urutan keempatnya tidak dapat
// tertukar di antara pemanggil dan adapter — keempatnya bertipe string.
type NaturalKey struct {
	PartNumber    string
	PanelName     string
	ChassisNumber string
	PanelSide     string
}

// KeyOf menyusun kunci alami sebuah baris.
func KeyOf(g Grouping) NaturalKey {
	return NaturalKey{
		PartNumber:    g.PartNumber,
		PanelName:     g.PanelName,
		ChassisNumber: g.ChassisNumber,
		PanelSide:     g.PanelSide,
	}
}

// SideKey adalah pasangan yang menentukan pilihan Sisi sebuah panel.
//
// Asalnya `RDB List/GetDataSisiPanel-SQL.xml`:
//
//	select sisi_panel AS "NAME" from pooldata.lokasi_panel_he
//	 where id_panel = {TempSparepart.MIN_STOCK} and nama = {TempSparepart.PANJANG}
//
// KEDUANYA dipakai bersama, bukan salah satu: `id_panel` mempersempit ke satu panel, dan
// `nama` mempersempit lagi di dalamnya. Lihat catatan pada LookupRepo.ListSides untuk
// ketidakpastian yang belum tertutup soal kolom `NAMA`.
type SideKey struct {
	// PanelID adalah nilai ID_PANEL, diisi autocomplete Nama Panel.
	PanelID string

	// PanelName adalah nilai yang diketik pada isian Nama Panel.
	PanelName string
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Ketujuh isiannya mengikuti caption `Section/MasterGroupingSparepartHEApproval-Section.xml`
// apa adanya. Yang ADA di tabel tetapi TIDAK di sini:
//
//	ID               kunci baris; diterbitkan saat penambahan, lihat IDSource
//	NO_GROUP_RANGKA  nomor grup; diterbitkan penyimpanan, lihat ComposeGroupNumber
//	APPROVAL         selalu StatusPending pada penyimpanan lewat layar ini
//	NAMA_PART        \
//	KATEGORI_SPART    |
//	TIPE_SPART        > kelimanya DITURUNKAN dari Master Sparepart menurut NO_PART;
//	KODE_PART         |  lihat catatan pada Grouping
//	PROD_DATE        /
type Input struct {
	// PartNumber adalah "Nomor Sparepart". Wajib — ia yang menentukan kelima isian turunan.
	PartNumber string

	// PanelID adalah ID_PANEL, dikirim isian tersembunyi di balik autocomplete Nama Panel.
	PanelID string

	// PanelName adalah "Nama Panel".
	PanelName string

	// PanelSide adalah "Sisi".
	PanelSide string

	// ChassisNumber adalah "No Rangka".
	ChassisNumber string

	// VehicleType adalah "Tipe Kendaraan".
	VehicleType string

	// GroupWithChassis adalah "Grouping Dengan No Rangka".
	GroupWithChassis string

	// Note adalah "Catatan".
	Note string
}

// Panjang maksimum isian teks.
//
// SELURUHNYA ASUMSI YANG DISADARI, bukan angka yang diterima dari Work Owner maupun dibaca
// dari DDL: kedua tabel modul ini tidak ada DDL-nya di export (`R-08`), dan sistem lama tidak
// memeriksa panjang satu pun isian — tidak ada satu pun `pyMaxLength` pada layarnya.
//
// Batasnya tetap dipasang karena tanpa itu penolakan datang dari basis data sebagai
// ORA-12899 — galat teknis yang tidak menuntun pengguna ke mana pun.
//
// Angka yang sama diulang di `GroupingForm.tsx`. Bila berubah, KEDUA tempat harus ikut
// berubah — utang yang disadari dari menduplikasi sebuah angka, dijaga terlihat oleh uji di
// mastergroupingsparepart_test.go.
const (
	// MaxPartNumberLength mengikuti Master Sparepart, yang nilainya disalin ke sini.
	MaxPartNumberLength = 50

	// MaxPanelNameLength mengikuti Master Panel, yang nilainya disalin ke sini.
	MaxPanelNameLength = 100

	// MaxChassisLength — nomor rangka kendaraan menurut standar VIN 17 karakter; dilebihkan
	// karena alat berat tidak selalu mengikutinya.
	MaxChassisLength = 50

	// MaxSideLength — sandinya satu karakter ("-", "1", "2"); dilebihkan karena baris lama
	// dapat memuat apa pun.
	MaxSideLength = 30

	// MaxVehicleTypeLength mengikuti `branddetail.TYPENAME`.
	MaxVehicleTypeLength = 100

	// MaxNoteLength — catatan bebas.
	MaxNoteLength = 500
)

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("mastergroupingsparepart: grouping tidak ditemukan")

	// ErrDuplicate: keempat kunci alami sudah dipakai baris lain.
	//
	// Padanan pesan `"Data sudah ada"` pada
	// `Activity/UpdateGroupingSparepartHE_act-Act.xml` langkah 15.
	ErrDuplicate = errors.New("mastergroupingsparepart: grouping dengan kunci itu sudah ada")

	// ErrPartNotFound: nomor sparepart yang diketik tidak ada di Master Sparepart.
	//
	// Padanan pesan `"Data Sparepart tidak ditemukan"` pada
	// `Activity/SetDataSparepart-Act.xml`.
	ErrPartNotFound = errors.New("mastergroupingsparepart: sparepart tidak ditemukan")

	// ErrGroupChassisNotFound: nomor rangka yang disebut pada "Grouping Dengan No Rangka"
	// tidak dipakai baris mana pun, sehingga tidak ada grup yang dapat diikuti.
	//
	// Padanan pesan `"Nomor rangka dalam grouping tidak ditemukan"` pada
	// `Activity/UpdateGroupingSparepartHE_act-Act.xml` langkah 2 dan 16.
	ErrGroupChassisNotFound = errors.New("mastergroupingsparepart: nomor rangka grouping tidak ditemukan")

	// ErrUnknownStatus: status persetujuan di luar "0", "1", "2".
	ErrUnknownStatus = errors.New("mastergroupingsparepart: status persetujuan tidak dikenal")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis data —
	// layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (`P-5`).
// `Activity/UpdateGroupingSparepartHE_act` menyusun dua pesan galat berdampingan —
// `local.err` untuk duplikat dan `local.err2` untuk nomor rangka — lalu menampilkannya lewat
// `Page-Set-Messages`. Yang berbeda di sini hanyalah seluruhnya dikirim dalam satu jawaban.
type ValidationError struct {
	Violation []Violation
}

// OneViolation membungkus satu pelanggaran menjadi ValidationError.
//
// Dipakai lapisan aplikasi untuk pemeriksaan yang menuntut pembacaan basis data — keberadaan
// sparepart, keberadaan nomor rangka grup, keunikan kunci alami — supaya galatnya sampai ke
// layar dalam bentuk yang SAMA dengan pelanggaran isian lain, dan menempel pada isiannya.
func OneViolation(field, message string) error {
	return &ValidationError{Violation: []Violation{{Field: field, Message: message}}}
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "mastergroupingsparepart: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas — bukan
// nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// TIDAK ada yang di-UPPERCASE. Kunci alaminya memang dibandingkan tanpa memandang besar-kecil
// huruf, tetapi yang di-uppercase adalah PEMBANDINGNYA — bukan nilai yang disimpan. Memaksa
// huruf besar pada nama panel akan mengubah tampilan setiap baris yang disunting, dan itu
// selisih yang tidak diminta siapa pun.
func (i Input) Clean() Input {
	trim := strings.TrimSpace

	return Input{
		PartNumber:       trim(i.PartNumber),
		PanelID:          trim(i.PanelID),
		PanelName:        trim(i.PanelName),
		PanelSide:        trim(i.PanelSide),
		ChassisNumber:    trim(i.ChassisNumber),
		VehicleType:      trim(i.VehicleType),
		GroupWithChassis: trim(i.GroupWithChassis),
		Note:             trim(i.Note),
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Yang diwajibkan, dan dari mana asalnya
//
// EMPAT isian wajib, dan keempatnya persis anggota kunci alami: Nomor Sparepart, Nama Panel,
// No Rangka, dan Sisi. Bukan karena layar lama menandainya `pyRequired` — ia tidak menandai
// satu pun — melainkan karena kunci alaminya menuntutnya: kunci yang salah satu anggotanya
// kosong tidak dapat membedakan dua baris.
//
// Sistem lama menerima keempatnya kosong, lalu memeriksa duplikat atas kunci yang seluruhnya
// kosong — sehingga baris kedua yang kosong SELALU ditolak dengan "Data sudah ada", pesan
// yang tidak menuntun ke mana pun. Menolaknya di sini dengan menyebut isian yang kurang
// adalah SELISIH YANG DIRENCANAKAN, dan ia menolak isian yang di sistem lama pun tidak
// pernah benar-benar tersimpan dua kali.
//
// # Yang TIDAK diwajibkan, meski tampak wajar
//
// **Tipe Kendaraan boleh kosong.** Ia bukan anggota kunci alami, dan tidak satu pun rule
// memeriksanya. Mewajibkannya akan menolak baris yang hari ini tersimpan tanpa keluhan.
//
// **ID Panel boleh kosong.** Ia isian tersembunyi yang diisi autocomplete, dan baris lama
// dapat memuatnya kosong. Menolaknya akan membuat baris lama tidak dapat disimpan ulang.
func (i Input) Check() error {
	var violation []Violation

	for _, r := range []struct {
		field, label, value string
		max                 int
	}{
		{"nomor_sparepart", "Nomor sparepart", i.PartNumber, MaxPartNumberLength},
		{"nama_panel", "Nama panel", i.PanelName, MaxPanelNameLength},
		{"no_rangka", "No rangka", i.ChassisNumber, MaxChassisLength},
		{"sisi", "Sisi", i.PanelSide, MaxSideLength},
	} {
		violation = append(violation, checkRequired(r.field, r.label, r.value, r.max)...)
	}

	for _, r := range []struct {
		field, label, value string
		max                 int
	}{
		{"id_panel", "ID panel", i.PanelID, MaxPanelNameLength},
		{"tipe_kendaraan", "Tipe kendaraan", i.VehicleType, MaxVehicleTypeLength},
		{"grouping_dengan_no_rangka", "Grouping dengan no rangka", i.GroupWithChassis, MaxChassisLength},
		{"catatan", "Catatan", i.Note, MaxNoteLength},
	} {
		violation = append(violation, checkLength(r.field, r.label, r.value, r.max)...)
	}

	violation = append(violation, i.checkGroupTarget()...)

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// checkGroupTarget menolak baris yang menggabungkan dirinya dengan dirinya sendiri.
//
// SELISIH YANG DIRENCANAKAN terhadap sistem lama, yang tidak memeriksanya. Di sana, mengisi
// "Grouping Dengan No Rangka" dengan nomor rangka baris itu sendiri membuat
// `RDB List/GetNoGroup-SQL.xml` menemukan barisnya sendiri pada penyuntingan — atau tidak
// menemukan apa pun pada penambahan, lalu ditolak dengan pesan yang membingungkan karena
// nomor rangkanya jelas-jelas sedang diketik di layar yang sama.
//
// Pemeriksaannya mengabaikan besar-kecil huruf dengan alasan yang sama seperti pencarian
// grupnya sendiri; lihat Repo.FindGroupByChassis.
func (i Input) checkGroupTarget() []Violation {
	if i.GroupWithChassis == "" || i.ChassisNumber == "" {
		return nil
	}
	if !strings.EqualFold(i.GroupWithChassis, i.ChassisNumber) {
		return nil
	}
	return []Violation{{
		Field: "grouping_dengan_no_rangka",
		Message: "Nomor rangka ini sama dengan No Rangka baris ini sendiri. " +
			"Kosongkan bila ingin membuka grup baru.",
	}}
}

// checkRequired memeriksa satu isian wajib beserta panjangnya.
func checkRequired(field, label, value string, max int) []Violation {
	if value == "" {
		return []Violation{{Field: field, Message: label + " wajib diisi."}}
	}
	return checkLength(field, label, value, max)
}

// checkLength memeriksa panjang satu isian yang boleh kosong.
func checkLength(field, label, value string, max int) []Violation {
	if len(value) > max {
		return []Violation{{
			Field:   field,
			Message: fmt.Sprintf("%s paling panjang %d karakter.", label, max),
		}}
	}
	return nil
}

// Filter menyaring daftar yang dibaca layar.
//
// Ia cerminan ketiga tab `Section/PNCMasterGroupingSparepartHE-Section.xml`, yang ketiganya
// membaca `GetDataMasterGrouping` yang sama dan hanya berbeda pada nilai APPROVAL-nya —
// `Activity/GetDataMasterGrouping-Act.xml` merangkainya sebagai
// `"AND A.APPROVAL=" + "'" + PARAM.Approve + "'"`.
type Filter struct {
	// Status wajib salah satu dari tiga yang dikenal.
	Status ApprovalStatus

	// Keyword mempersempit daftar pada nomor sparepart, nama sparepart, nama panel, DAN
	// nomor rangka.
	//
	// Keempatnya sekaligus karena keempatnya cara orang mencari baris di layar ini: petugas
	// mencari "panel apa saja yang terpasang di rangka ini" jauh lebih sering daripada
	// mencari satu baris tertentu. Kosong berarti tanpa penyaring.
	Keyword string
}

// GroupNumberPrefix adalah awalan nomor grup yang diterbitkan sistem lama.
//
// `Activity/UpdateGroupingSparepartHE_act-Act.xml` langkah 7:
//
//	TempSparepart.pyID := "000" + TempNewNoGroup.pxResults(1).ID
//
// Ia PERANGKAIAN TEKS, bukan pengisian nol sampai lebar tertentu. Nomor 9 menjadi "0009" dan
// nomor 10 menjadi "00010" — lebarnya bertambah, bukan tetap. Ditiru apa adanya; lihat
// ComposeGroupNumber.
const GroupNumberPrefix = "000"

// ComposeGroupNumber merangkai nomor grup dari nomor urutnya.
//
// Ia berada di paket domain, bukan di adapter, karena BENTUK NOMOR GRUP adalah aturan
// domain: ia yang menentukan bagaimana sebuah grup kendaraan dikenali, dan ia harus sama
// persis pada adapter SQL maupun adapter memori. Satu tempat, satu bentuk — dan satu uji yang
// menjaganya.
//
// # Kenapa awalannya dipertahankan meski ganjil
//
// Karena baris yang sudah ada di basis data memakainya. Menerbitkan "1" sementara data lama
// memuat "0001" akan membuat dua bentuk hidup berdampingan untuk hal yang sama, dan pencarian
// grup menjadi bergantung pada bentuk mana yang kebetulan tersimpan.
//
// Lebar yang berubah-ubah adalah akibat yang diterima, bukan yang diperbaiki: memperbaikinya
// berarti menambah butir ke daftar perbaikan eksplisit `P-5`, dan daftar itu milik `D-49` —
// keputusan Work Owner, bukan tafsiran modul. Akibat yang harus disadari: mengurutkan nomor
// grup sebagai TEKS tidak sama dengan mengurutkannya sebagai angka mulai dari nomor 10.
func ComposeGroupNumber(sequence int64) string {
	return GroupNumberPrefix + strconv.FormatInt(sequence, 10)
}

// ParseGroupNumber membaca nomor urut dari sebuah nomor grup.
//
// Ia kebalikan ComposeGroupNumber, dan ia memaafkan bentuk apa pun yang berakhir dengan
// angka: baris lama dapat memuat "0001" maupun "1", karena sistem lama menulis keduanya —
// `GetNewNoGroup` menghasilkan bentuk berawalan sedangkan `GetNoGroup` mengembalikannya
// lewat `TO_NUMBER` yang membuang awalannya. Lihat usecase.Service untuk perlakuan atas
// ketidakseragaman itu.
//
// Nilai yang bukan angka sama sekali menghasilkan galat, bukan nol diam-diam: nol akan
// membuat penerbitan nomor berikutnya mengulang nomor yang sudah dipakai.
func ParseGroupNumber(text string) (int64, error) {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return 0, nil
	}

	// Setiap karakternya harus DIGIT. ParseInt sendiri tidak cukup: ia menerima tanda "+" dan
	// "-" di depan, sehingga "+1" akan terbaca sebagai grup 1 dan bertabrakan dengan "0001"
	// yang sudah ada. Tanda itu tidak pernah sah pada nomor grup.
	for _, one := range clean {
		if one < '0' || one > '9' {
			return 0, fmt.Errorf("mastergroupingsparepart: nomor grup %q bukan angka", text)
		}
	}

	// Awalan nol dibuang supaya "0001" dan "1" menghasilkan nomor yang sama. Keduanya memang
	// ada di basis data: sistem lama menulis bentuk berawalan saat grup DIBUKA dan bentuk
	// polos saat grup DIIKUTI.
	trimmed := strings.TrimLeft(clean, "0")
	if trimmed == "" {
		// Seluruhnya nol — "000", "0000". Sah, dan artinya nol.
		return 0, nil
	}

	number, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		// Hanya tersisa satu kemungkinan: angkanya terlalu panjang untuk int64. Ia tetap galat
		// — nomor yang tidak dapat dibandingkan tidak dapat dipakai menerbitkan nomor
		// berikutnya.
		return 0, fmt.Errorf("mastergroupingsparepart: nomor grup %q bukan angka: %w", text, err)
	}
	return number, nil
}

// IDSource menerbitkan ID baru.
//
// Ia seam tersendiri, bukan method pada Repo, karena isinya bukan urusan grouping melainkan
// urusan **penomoran**.
//
// # Bentuknya BERBEDA dari ketiga master alat berat lain
//
// `Database/PEGA_M_GROUPING_SPAREPART_HE.prc:11`:
//
//	SELECT NVL(MAX(ID),0) + 1 INTO id_count from POOLDATA.m_sparepart_he_vin_key;
//
// Tidak ada kode situs, tidak ada sequence, tidak ada pengisian nol — hanya pencacah
// `MAX(ID)+1`. Master Bengkel, Master Panel, dan Master Sparepart ketiganya memakai
// `kode_situs || LPAD(sequence, n, '0')`; modul ini tidak.
//
// Menyeragamkannya akan menerbitkan ID yang tidak sebentuk dengan ID yang sudah ada di tabel,
// dan ID yang lebih panjang berpotensi tidak muat pada kolom yang lebarnya belum diketahui
// (`R-08`). Ditiru apa adanya.
//
// # Keterbatasan yang disadari
//
// `MAX(ID)+1` adalah pola yang tidak aman terhadap penambahan bersamaan: dua permintaan yang
// membacanya pada saat yang sama menerima angka yang sama. Sistem lama tidak menutupnya sama
// sekali. Di sini ia DIPERSEMPIT dengan menjalankan pembacaan dan penyisipan di dalam satu
// transaksi yang mengunci barisnya lebih dulu; yang benar-benar menutupnya adalah sequence
// atau constraint unik, dan keduanya menunggu DDL (`R-08`) serta prosedur perubahan skema
// (`D-63`).
type IDSource interface {
	// NextID mengembalikan ID berikutnya.
	NextID(ctx context.Context) (string, error)

	// NextGroupNumber mengembalikan nomor grup berikutnya, sudah berbentuk akhir.
	//
	// Padanan `RDB List/GetNewNoGroup-SQL.xml` ditambah perangkaian awalan pada
	// `UpdateGroupingSparepartHE_act` langkah 7 — keduanya disatukan di sini supaya bentuk
	// nomor grup tidak tersebar ke dua tempat.
	NextGroupNumber(ctx context.Context) (string, error)
}

// Repo adalah seam ke penyimpanan grouping sparepart SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat kueri
// (ADR-0030 Opsi 1).
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring.
	List(ctx context.Context, filter Filter) ([]Grouping, error)

	// Get mengembalikan satu baris; ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (Grouping, error)

	// FindByKey mencari baris menurut keempat kunci alaminya; ErrNotFound bila tidak ada.
	//
	// Padanan pencarian duplikat pada
	// `Activity/UpdateGroupingSparepartHE_act-Act.xml` langkah 3.
	FindByKey(ctx context.Context, key NaturalKey) (Grouping, error)

	// FindGroupByChassis mengembalikan nomor grup yang dipakai sebuah nomor rangka.
	//
	// Padanan `RDB List/GetNoGroup-SQL.xml`:
	//
	//	select TO_NUMBER(NO_GROUP_RANGKA) AS "ID" from POOLDATA.sparepart_he_vin_key
	//	 where NO_RANGKA = {TempSparepart.BERAT}
	//
	// Yang dikembalikan di sini adalah nomor grup APA ADANYA, bukan hasil `TO_NUMBER`.
	// Alasannya ada pada usecase.Service.resolveGroupNumber.
	//
	// ErrGroupChassisNotFound bila nomor rangkanya tidak dipakai baris mana pun.
	FindGroupByChassis(ctx context.Context, chassis string) (string, error)

	// Insert menyisipkan baris baru pada KEDUA tabel.
	//
	// Pemeriksaan kunci alami berada DI DALAM operasi repo, bukan dipecah menjadi "cek" lalu
	// "sisip" di lapisan aplikasi. Sistem lama memecahnya — pencarian duplikat di langkah 3,
	// penyimpanan di langkah 13 — dan jarak di antara keduanya adalah lubang balapan yang
	// tidak dijaga apa pun.
	//
	// KETERBATASAN YANG DISADARI. Tanpa constraint unik pada keempat kolom itu, lubang itu
	// hanya dipersempit, tidak ditutup. Penutupnya adalah constraint di basis data, dan itu
	// menunggu DDL (`R-08`) beserta prosedur perubahan skema (`D-63`).
	Insert(ctx context.Context, g Grouping) error

	// Update menyimpan perubahan pada baris yang sudah ada pada KEDUA tabel; ErrNotFound bila
	// barisnya hilang di antara pemuatan layar dan penyimpanan.
	Update(ctx context.Context, g Grouping) error

	// SetStatus menetapkan APPROVAL sejumlah baris sekaligus.
	//
	// Yang dikembalikan adalah jumlah baris yang benar-benar berubah, supaya pemanggil dapat
	// membedakan "tidak ada yang dipilih" dari "yang dipilih sudah tidak ada".
	SetStatus(ctx context.Context, id []string, status ApprovalStatus) (int, error)
}

// RepoSelector memilih Store milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan hukum
// ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Store, error)

// Store menyatukan ketiga seam yang dipakai layanan modul ini.
//
// Ketiganya tetap DIDEKLARASIKAN terpisah — Repo untuk kedua tabel master, LookupRepo untuk
// empat sumber acuan yang hanya dibaca, IDSource untuk penomoran — karena ketiganya menjawab
// pertanyaan yang berbeda dan dapat berubah sendiri-sendiri. Yang disatukan hanyalah CARA
// MEMILIHNYA: ketiganya selalu berasal dari koneksi entitas yang sama, sehingga tiga pemilih
// terpisah hanya akan membuka kemungkinan ketiganya menunjuk entitas berbeda.
type Store interface {
	Repo
	LookupRepo
	IDSource
}
