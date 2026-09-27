package mastergroupingsparepart

import "context"

// # Empat sumber acuan yang dibaca modul ini
//
// Seluruhnya milik master lain dan aplikasi ini HANYA MEMBACA (ADR-0004, penulis tunggal per
// tabel). Tidak satu pun disentuh oleh penyimpanan grouping:
//
//	POOLDATA.PANEL_HE         Nama Panel      — ID_PANEL, NAME        (milik Master Panel)
//	POOLDATA.LOKASI_PANEL_HE  Sisi            — SISI_PANEL            (milik Master Panel)
//	branddetail               Tipe Kendaraan  — ID, TYPENAME          (milik master kendaraan)
//	POOLDATA.SPAREPART_HE     isian turunan   — NO_SPART dan 5 kolom  (milik Master Sparepart)
//
// Keempatnya menyuapi isian yang di Pega berupa `pxAutoComplete`, `pxDropdown`, atau
// pengisian otomatis saat kehilangan fokus.

// Panel adalah satu pilihan pada isian "Nama Panel".
//
// Asalnya `Report Definition/BrowseMasterPanel_HE_RD-RD.xml` atas kelas
// `ASM-FW-GCNMFW-Int-PANEL_HE`, yang dipakai autocomplete pada
// `Section/MasterGroupingSparepartHEApproval-Section.xml`:
//
//	pySourceName     = BrowseMasterPanel_HE_RD
//	pyListSource     = reportdefinition
//	pyDisplayProperty= .NAME          -> isian Nama Panel
//	pyDisplayProperty= .ID_PANEL      -> TempSparepart.MIN_STOCK, tersembunyi
//	pyReportDefParams: APPROVAL = "1"
//
// # Kenapa penyaringnya APPROVAL = '1'
//
// Karena itulah yang diminta autocomplete-nya. Akibat yang harus disadari: **panel yang belum
// disetujui tidak muncul di daftar**, sehingga grouping tidak dapat menunjuknya. Itu perilaku
// sistem lama, dan ia konsisten secara bisnis — grouping menunjuk panel yang sudah berlaku.
type Panel struct {
	// ID adalah kolom ID_PANEL, disimpan sebagai SPAREPART_HE_VIN_KEY.ID_PANEL.
	ID string

	// Name adalah kolom NAME — yang dilihat pengguna, dan yang disimpan sebagai
	// SPAREPART_HE_VIN_KEY.NAMA_PANEL.
	//
	// Keduanya disimpan, bukan hanya ID-nya. Itu meniru sistem lama, dan ia yang membuat
	// baris lama tetap terbaca sekalipun panel asalnya kemudian diubah namanya — sekaligus
	// alasan NAMA_PANEL menjadi anggota kunci alami dan bukan ID_PANEL.
	Name string
}

// Side adalah nilai kolom SISI_PANEL.
//
// Ketiga sandinya dibaca dari `Activity/GetSisiPanel-Act.xml`, yang menyandikannya saat
// menyusun daftar pilihan:
//
//	.NAME := @If(.NAME=="-","-", @If(.NAME=="1","KIRI","KANAN"))
//
// Sandi yang sama sudah lebih dulu dipakai modul Master Panel, yang membacanya dari dua rule
// yang sepakat. Ia TIDAK diimpor dari sana: tipe bersama membuat perubahan di satu master
// menyeret master lain, dan kedua modul menyimpannya di kolom tabel yang berbeda.
type Side string

const (
	// SideNone — "-", tanpa sisi. Dipakai panel yang tidak punya kiri dan kanan.
	SideNone Side = "-"

	// SideLeft — "1", KIRI.
	SideLeft Side = "1"

	// SideRight — "2", KANAN.
	SideRight Side = "2"
)

// SideLabel mengembalikan sebutan sebuah sandi sisi dalam bahasa yang dibaca pengguna.
//
// Sandi di luar ketiganya dikembalikan APA ADANYA, bukan dipaksa menjadi "KANAN". Sistem lama
// memaksanya — `@If(.NAME=="1","KIRI","KANAN")` menjawab KANAN untuk apa pun yang bukan "-"
// dan bukan "1" — sehingga nilai rusak tampil sebagai nilai yang sah dan tidak ada yang tahu.
//
// SELISIH YANG DIRENCANAKAN, dan perlakuan yang sama sudah diambil modul Master Panel.
func SideLabel(code string) string {
	switch Side(code) {
	case SideNone:
		return "-"
	case SideLeft:
		return "KIRI"
	case SideRight:
		return "KANAN"
	default:
		return code
	}
}

// VehicleType adalah satu pilihan pada isian "Tipe Kendaraan".
//
// Asalnya `RDB List/BrowseTypeHE_Sql-SQL.xml`, lewat data page `D_TypeHEList` yang dijalankan
// `Activity/BrowseActV_HE_Type-Act.xml`:
//
//	select id as "BANK_ID", TYPENAME as "NAMA_BANK"
//	  from branddetail where type = 'ANEKA' and ACTIVESTATUS=1
//
// PERHATIKAN ALIAS KOLOMNYA. `id` dialiaskan `"BANK_ID"` dan `TYPENAME` dialiaskan
// `"NAMA_BANK"` — nama yang sama sekali tidak ada hubungannya dengan bank. Itu persis bentuk
// utang yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat, dan itu pula sebabnya `NAMA_BANK`
// muncul sebagai properti pada section Master Grouping Sparepart. Alias itu TIDAK dibawa.
//
// # Yang DISIMPAN adalah namanya, bukan ID-nya
//
// Autocomplete-nya menulis ke `TempSparepart.QTY_PESAN`, yang dipetakan ke `B.TIPE`, dan yang
// ditampilkan kembali oleh daftar adalah `B.TIPE` itu juga. Menyimpan ID-nya akan membuat
// daftar menampilkan angka.
type VehicleType struct {
	// ID adalah kolom `id` pada branddetail. Dibawa untuk penelusuran, TIDAK disimpan.
	ID string

	// Name adalah kolom TYPENAME — yang dilihat pengguna DAN yang disimpan.
	Name string
}

// PartRef adalah kelima isian yang diturunkan dari Master Sparepart.
//
// Asalnya `Activity/SetDataSparepart-Act.xml`, yang berjalan saat isian Nomor Sparepart
// kehilangan fokus:
//
//	TempSparepart.NAMA_SPART     := TempDataSparepart.pxResults(1).NAMA_SPART
//	TempSparepart.KATEGORI_SPART := TempDataSparepart.pxResults(1).KATEGORI_SPART
//	TempSparepart.TIPE_SPART     := TempDataSparepart.pxResults(1).TIPE_SPART
//	TempSparepart.PROD_DATE      := TempDataSparepart.pxResults(1).PROD_DATE
//	TempSparepart.KODE_SPART     := TempDataSparepart.pxResults(1).KODE_SPART
//
// Bila nomornya tidak ketemu, activity itu menjawab "Data Sparepart tidak ditemukan".
//
// # Kenapa kelimanya ikut DISIMPAN, bukan dibaca ulang lewat join
//
// Karena sistem lama menyimpannya, dan baris lama sudah memuatnya. Membacanya ulang lewat
// join akan mengubah isi baris lama begitu sparepart asalnya diubah namanya — perubahan yang
// tidak diminta siapa pun, dan yang membuat daftar tidak lagi sama dengan yang dilihat Pega
// atas data yang sama (`P-5`).
type PartRef struct {
	// Number adalah kolom NO_SPART — nomor yang diketik pengguna.
	Number string

	// Name adalah kolom NAMA_SPART, disimpan sebagai SPAREPART_HE_VIN_KEY.NAMA_PART.
	Name string

	// CategoryID adalah kolom KATEGORI_SPART; berisi PART_CATEGORY_ID.
	CategoryID string

	// TypeID adalah kolom TIPE_SPART; berisi PART_SECTION_ID.
	TypeID string

	// Code adalah kolom KODE_SPART, disimpan sebagai SPAREPART_HE_VIN_KEY.KODE_PART.
	Code string

	// ProductionDate adalah kolom PROD_DATE. TEKS, bukan tanggal.
	ProductionDate string
}

// MaxLookupRows membatasi banyaknya baris yang dikembalikan sebuah lookup.
//
// Sistem lama membatasi daftar panelnya di 500 (`pyMaxRecords` pada
// `BrowseMasterPanel_HE_RD`) dan tidak membatasi daftar tipe kendaraan sama sekali. Batas
// yang sama dipakai keduanya di sini, dan ia berada di backend — memotong di peramban berarti
// barisnya sudah terlanjur dibaca, dikirim, dan diurai.
//
// Lima ratus dipilih karena kedua daftar ini master penggolongan, bukan master transaksi.
// Bila kelak tersentuh, daftar akan terpotong diam-diam — dan itulah yang membuat pencacahnya
// ikut dilaporkan; lihat usecase.Service.Options.
const MaxLookupRows = 500

// LookupRepo adalah seam ke keempat sumber acuan, SATU portal.
//
// Ia terpisah dari Repo karena menjawab pertanyaan yang berbeda — "apa pilihan yang tersedia"
// alih-alih "apa isi master ini" — dan karena keempat sumbernya dimiliki master lain.
// Keempatnya tetap dipilih bersama lewat RepoSelector, karena keempatnya selalu berasal dari
// koneksi entitas yang sama.
type LookupRepo interface {
	// ListPanels mengembalikan panel yang sudah disetujui.
	//
	// Tanpa pencarian: autocomplete Pega pun memuat daftarnya lewat report definition yang
	// sama dan menyaringnya di peramban.
	ListPanels(ctx context.Context) ([]Panel, error)

	// ListSides mengembalikan sandi sisi yang tersedia pada sebuah panel.
	//
	// Padanan `RDB List/GetDataSisiPanel-SQL.xml`, yang menyaring `ID_PANEL` DAN `NAMA`
	// sekaligus.
	//
	// # Ketidakpastian yang belum tertutup, dan dinyatakan alih-alih disembunyikan
	//
	// Kolom `NAMA` pada `POOLDATA.LOKASI_PANEL_HE` TIDAK pernah muncul sebagai kolom yang
	// dibaca di seluruh export — hanya sebagai penyaring pada satu kueri ini. Dua pembacaan
	// yang sama-sama masuk akal:
	//
	//	(a) NAMA berisi NAMA LOKASI, sama dengan kolom LOKASI_PANEL di sebelahnya.
	//	    Ini yang diasumsikan modul Master Panel, dan penyimpanannya menulis keduanya
	//	    dengan nilai yang sama.
	//
	//	(b) NAMA berisi NAMA PANEL, salinan dari POOLDATA.PANEL_HE.NAME.
	//	    Ini yang dituntut modul INI: nilai yang dikirimkannya berasal dari autocomplete
	//	    atas `BrowseMasterPanel_HE_RD`, report definition atas PANEL_HE yang menampilkan
	//	    `.NAME` — nama panel, bukan nama lokasi.
	//
	// Bila (b) yang benar, jalur tulis Master Panel akan mengisi NAMA dengan nama lokasi dan
	// **mematikan daftar Sisi di layar ini tanpa satu pun pesan galat** — kelas kegagalan
	// yang paling mahal ditemukan.
	//
	// Modul Master Panel TIDAK disunting dari sini (isolasi modul yang sudah selesai).
	// Yang dikerjakan: kuerinya ditiru apa adanya, dan pemeriksaannya dipasang di
	// `claimpnc -periksa` yang menghitung berapa baris NAMA-nya cocok dengan NAME panel
	// induknya versus dengan LOKASI_PANEL-nya sendiri. Jawabannya empiris, dari data
	// produksi, bukan dari tebakan salah satu pihak.
	ListSides(ctx context.Context, key SideKey) ([]Side, error)

	// ListVehicleTypes mengembalikan tipe kendaraan yang aktif.
	ListVehicleTypes(ctx context.Context) ([]VehicleType, error)

	// FindPart mencari satu sparepart menurut nomornya, untuk mengisi kelima isian turunan.
	//
	// ErrPartNotFound bila nomornya tidak ada di Master Sparepart.
	//
	// Ia TIDAK menyaring APPROVAL, dan itu meniru sistem lama apa adanya:
	// `Activity/SetDataSparepart` mencarinya tanpa penyaring status sama sekali. Menambahkan
	// penyaring di sini akan menolak nomor sparepart yang hari ini diterima.
	FindPart(ctx context.Context, number string) (PartRef, error)
}
