// Package mastersparepart adalah inti modul Master Sparepart.
//
// # Apa yang dimodelkan di sini
//
// Daftar **suku cadang alat berat** beserta harga jual, dimensi, batas stok, dan
// penggolongannya. Setiap sparepart menunjuk satu **Kategori** dan satu **Tipe** yang
// dibaca dari dua tabel acuan, dan dapat menunjuk sparepart lain sebagai **substitusi**.
//
// Ia master ketiga dari keluarga alat berat, setelah Master Bengkel dan Master Panel.
// Ketiganya berbagi satu activity persetujuan yang sama di Pega; lihat
// usecase.Service.Decide.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/SparePart_HE-Harness.xml                     layar, MENU_ID 31
//	Section/MasterSparepartHE-Section.xml                tombol Tambah dan Refresh
//	Section/BrowseMasterSparepartHE-Section.xml          3 tab: Approve · Reject · Approval
//	Section/BrowseMasterSparepartHEApproval-Section.xml  grid + form, 20 isian berlabel
//	Section/BrowseMasterSparepartHEApprove-Section.xml   tab disetujui, pyPageSize=30
//	Section/BrowseMasterSparepartHEReject-Section.xml    tab ditolak
//	Section/ApprovalMasterSparepartHE-Section.xml        persetujuan borongan di Inbox Manager
//	Report Definition/BrowseSparepartHE_RD-RD.xml        23 kolom POOLDATA.SPAREPART_HE
//	RDB List/ValidationMasterSparepartNo-SQL.xml         tolak NO_SPART ganda
//	RDB List/ValidationMasterSparepartName-SQL.xml       tolak NAMA_SPART ganda
//	RDB List/ValidationMasterSparepartCode-SQL.xml       tolak KODE_SPART ganda
//	RDB List/UpdateSparepartHE-SQL.xml                   simpan lewat PEGA_M_SPAREPART_HE
//	RDB List/CountMasterSparepartManager-SQL.xml         pencacah antrean persetujuan
//	RDB List/GetIDDokumenSparepart-SQL.xml               id dokumen lampiran
//	RDB List/GetDataSparepart-SQL.xml                    pencarian satu baris menurut NO_SPART
//	Database/PEGA_M_SPAREPART_HE.prc                     pembentukan ID dan penyimpanan JSON
//	Activity/UpdateSparepartHE_act-Act.xml               urutan langkah simpan
//	Activity/ValidateMasterSparepart-Act.xml             tiga pesan galat nilai ganda
//	Activity/SetValueSparepartHE-Act.xml                 pemuatan baris ke form
//	Activity/BrowseTipeKategoriPart-Act.xml              isi kedua dropdown acuan
//	Activity/SetApprovalAllMaster-Act.xml                keputusan borongan, TIPE "M_SPAREPART_HE"
//	Database/m_menu_aplikasi_pnc.csv                     MENU_ID 31 "Master Sparepart"
//
// # Empat hal yang membedakannya dari Master Panel
//
//  1. **Pelaku tersimpan.** `POOLDATA.SPAREPART_HE` punya kolom `USER_UPDATE`, dan
//     `Activity/UpdateSparepartHE_act` mengisinya dengan `OperatorID.pyUserIdentifier`.
//     Master Panel tidak punya satu pun kolom pencatat pelaku.
//  2. **Tidak ada kolom alasan tolak.** Kedua puluh tiga kolomnya terbaca lengkap dari
//     `BrowseSparepartHE_RD`, dan tidak satu pun menampung catatan penolakan. Isian
//     "Catatan" pada layar persetujuan Pega karena itu tidak punya tujuan; lihat
//     usecase.Service.Decide.
//  3. **Dua tabel acuan.** Kategori dan Tipe dibaca dari tabel lain; lihat lookup.go.
//  4. **Delapan isian angka.** Harga, tiga batas stok, tiga dimensi, dan berat. Seluruhnya
//     disimpan sebagai TEKS apa adanya; lihat catatan pada Sparepart.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package mastersparepart

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ApprovalStatus adalah posisi sebuah baris dalam alur persetujuan.
//
// Nilainya tetap "0", "1", "2" seperti kolom `APPROVAL` pada `POOLDATA.SPAREPART_HE`:
// tabelnya masih dibaca sistem lama selama masa paralel (ADR-0004), sehingga mengubah
// sandi nilainya akan membuat kedua sistem membaca baris yang sama secara berbeda.
//
// Ketiga sandinya terbaca dari `Section/BrowseMasterSparepartHE-Section.xml`, yang memuat
// tiga section dengan nilai penyaring yang berbeda, dan dikuatkan
// `RDB List/CountMasterSparepartManager-SQL.xml` yang mencacah `APPROVAL = 0`.
//
// Sandinya kebetulan sama dengan Master Bengkel, Master Panel, Master Auto Claim, dan
// Master Rekening. Kelimanya sengaja TIDAK dipakai bersama: tabelnya berbeda, dan tipe
// bersama membuat perubahan di satu master menyeret master lain.
type ApprovalStatus string

const (
	// StatusPending — diajukan, belum diputuskan. Nilai lahir setiap baris baru DAN setiap
	// baris yang disunting: `Activity/UpdateSparepartHE_act` menetapkan
	// `TempSparepart.APPROVAL := "0"` tanpa syarat.
	StatusPending ApprovalStatus = "0"

	// StatusApproved — disetujui. Hanya baris berstatus ini yang dipakai sistem hilir.
	StatusApproved ApprovalStatus = "1"

	// StatusRejected — ditolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
//
// Layar lama tidak memberi nama pada ketiga tabnya — nama sectionnya `…Approve`,
// `…Reject`, dan `…Approval`. Teks di sini mengikuti kata yang sama, dengan "Waiting
// Approval" untuk yang terakhir supaya terbaca sebagai antrean dan bukan sebagai tindakan
// (D-13: kata layar lama diikuti, bukan salah bacanya). Sama persis dengan Master Panel
// dan Master Bengkel.
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

// Sparepart adalah satu baris master sparepart — satu baris `POOLDATA.SPAREPART_HE`.
//
// Kedua puluh tiga kolomnya dibaca dari `Report Definition/BrowseSparepartHE_RD-RD.xml`,
// ditambah DOKUMENID yang dibaca terpisah oleh `RDB List/GetIDDokumenSparepart-SQL.xml`
// atas tabel yang sama. Label yang dipakai layar dibaca dari `pyLabelFieldValue` pada
// `Section/BrowseMasterSparepartHEApproval-Section.xml`.
//
// # Kenapa kedelapan isian angka bertipe TEKS
//
// Harga jual, tiga batas stok, tiga dimensi, dan berat seluruhnya disimpan sebagai string
// dan ditulis kembali apa adanya. Angkanya HANYA dipakai untuk pemeriksaan, tidak pernah
// untuk perhitungan — sehingga tidak ada satu pun titik di modul ini yang membulatkan
// nilai uang (`I-12`, `D-51`).
//
// Itu bukan kemalasan melainkan pilihan yang sama dengan Master Bengkel dan Master Auto
// Claim, yang menyimpan persentasenya sebagai teks dengan alasan yang identik: DDL tabelnya
// belum ada (`R-08`), dan membaca `HARGA_JUAL` sebagai float64 lalu menuliskannya kembali
// akan mengubah "1250000.00" menjadi "1.25e+06" pada baris yang tidak pernah disunting
// siapa pun.
//
// # Kenapa lima kolom penanda bertipe teks bebas
//
// `JENIS_SPART`, `SATUAN`, `STS_AKTIF`, `STS_PART`, dan `SUBSTITUSI_SPART` dirender
// `pxRadioButtons`, `pxDropdown`, atau `pxTextInput` dengan `pyListSource=associated` —
// artinya pilihannya datang dari rule Field Value pada propertinya, dan tidak ada satu pun
// direktori Property maupun Field Value di export (`R-16`).
//
// Menebaknya berarti memutuskan domain kolom yang menentukan satuan sebuah harga. Baris
// lama karena itu dibaca apa adanya, dan layar menawarkan nilai yang SUDAH DIPAKAI baris
// lain sebagai pilihan — jawaban terbaik yang tersedia atas pertanyaan yang export tidak
// jawab. Perlakuan yang sama dipakai Master Panel untuk kesembilan penandanya.
type Sparepart struct {
	// ID adalah kolom ID — kunci baris ini.
	//
	// Ia TIDAK diketik pengguna. `Database/PEGA_M_SPAREPART_HE.prc:21` menerbitkannya
	// sebagai kode situs ditambah nomor urut SEPULUH digit; lihat IDSource.
	ID string

	// Name adalah kolom NAMA_SPART — "Nama Sparepart".
	//
	// Salah satu dari TIGA kunci alami; lihat ErrNameTaken.
	Name string

	// Number adalah kolom NO_SPART — "Nomor Sparepart".
	//
	// Kunci alami kedua. Ia pula yang dipakai `RDB List/GetDataSparepart-SQL.xml` untuk
	// menarik satu baris dari modul lain.
	Number string

	// Code adalah kolom KODE_SPART — "Kode Sparepart". Kunci alami ketiga.
	Code string

	// SellingPrice adalah kolom HARGA_JUAL — "Harga Jual (Rp)".
	//
	// Satu-satunya isian yang di Pega dirender `pxCurrency`; kesembilan angka lain memakai
	// `pxTextInput`. Ia juga yang menentukan apakah PriceUpdatedAt ikut berubah saat
	// disimpan — lihat usecase.Service.
	SellingPrice string

	// CategoryID adalah kolom KATEGORI_SPART, dan ia berisi **PART_CATEGORY_ID**, bukan
	// namanya.
	//
	// Terbaca dari `Activity/SetKategoriSparepart_act-Act.xml`, yang menerima nilainya
	// sebagai `Param.ID` lalu mencarinya lewat
	// `RDB List/BrowseSparepartCategoryClaimHE_sql-SQL.xml`:
	//
	//	where PART_CATEGORY_ID = {TempInputKategoriSparepart.CaseID}
	//
	// Salah membacanya sebagai nama berarti setiap baris lama menunjuk kategori yang tidak
	// akan pernah ditemukan.
	CategoryID string

	// TypeID adalah kolom TIPE_SPART, dan ia berisi **PART_SECTION_ID**.
	//
	// Terbaca dari `Activity/SetTipeSparepart_act-Act.xml` dengan pola yang sama persis.
	// Tipe bercabang dari Kategori; lihat lookup.go.
	TypeID string

	// Kedelapan isian angka. Seluruhnya TEKS; lihat catatan pada tipe ini.
	Weight        string // BERAT      · "Berat Sparepart (gram)"
	Length        string // PANJANG    · "Panjang Sparepart (cm)"
	Width         string // LEBAR      · "Lebar Sparepart (cm)"
	Height        string // TINGGI     · "Tinggi Sparepart (cm)"
	MinStock      string // MIN_STOCK  · "Stock Minimal"
	MaxStock      string // MAX_STOCK  · "Stock Maximal"
	OrderQuantity string // QTY_PESAN  · "Kuantitas Pesanan"

	// ProductionDate adalah kolom PROD_DATE — "Tanggal Produksi".
	//
	// TEKS, bukan tanggal, dan itu meniru layar lama apa adanya: Pega merendernya
	// `pxTextInput` tanpa `pyDateTimeFormat` sama sekali — bukan kontrol tanggal.
	//
	// ASUMSI YANG DISADARI: tipe kolomnya belum diketahui (`R-08`). Bila ia ternyata DATE
	// dan bukan VARCHAR2, penyimpanan akan gagal dengan galat konversi pada percobaan
	// pertama di staging — gagal keras dan terlihat, bukan diam-diam salah. Pemeriksaannya
	// sudah terpasang di `claimpnc -periksa`; lihat sparepart_check_date_columns pada
	// berkas .sql.
	ProductionDate string

	// Substitute adalah kolom SUBSTITUSI_SPART — "Part Substitusi".
	//
	// Isinya NAMA sparepart lain, bukan ID-nya: `Activity/ValidasiSparepart-Act.xml`
	// memeriksanya terhadap `NAMA_SPART` dan menjawab "Nama Sparepart tidak ada" bila tak
	// ketemu.
	//
	// Pemeriksaan itu TIDAK ditiru di sini — lihat Input.Check.
	Substitute string

	// Keempat penanda berikut nilai sahnya TIDAK DIKETAHUI; lihat catatan pada tipe ini.
	Kind         string // JENIS_SPART · "Jenis Sparepart"  · pxRadioButtons
	Unit         string // SATUAN      · "Satuan"           · pxDropdown
	ActiveStatus string // STS_AKTIF   · "Status Aktif"
	PartStatus   string // STS_PART    · "Status Sparepart" · pxDropdown

	// UpdatedBy adalah kolom USER_UPDATE — login petugas yang terakhir menyimpannya.
	//
	// Diisi penyimpanan, tidak pernah dari isian layar:
	// `Activity/UpdateSparepartHE_act` menetapkan
	// `TempSparepart.USER_UPDATE := OperatorID.pyUserIdentifier`.
	//
	// Login yang DIKETIK pengguna, bukan NIK — kolomnya sudah berisi login pada baris lama.
	UpdatedBy string

	// PriceUpdatedAt adalah kolom TGL_UPDATE_HARGA — "Tanggal Update".
	//
	// nil berarti belum pernah terisi. Diisi penyimpanan, tidak pernah dari isian layar.
	PriceUpdatedAt *time.Time

	// DocumentID adalah kolom DOKUMENID — lampiran yang ditautkan ke baris ini.
	//
	// Diisi `Activity/UpdateSparepartHE_act` dari hasil `PNCSaveAttachmentToDB`. Unggah
	// lampirannya TIDAK dibawa modul ini — lihat catatan pada usecase.Service.
	DocumentID string

	// Status adalah kolom APPROVAL.
	Status ApprovalStatus
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Kedua puluh isiannya mengikuti caption `Section/BrowseMasterSparepartHEApproval-Section.xml`
// apa adanya. Yang ADA di tabel tetapi TIDAK di sini, karena kelimanya diturunkan sistem:
//
//	ID                kunci baris; diterbitkan saat penambahan, lihat IDSource
//	APPROVAL          selalu StatusPending pada penyimpanan lewat layar ini
//	USER_UPDATE       diambil dari identitas pemanggil, bukan dari badan permintaan
//	TGL_UPDATE_HARGA  distempel penyimpanan; lihat usecase.Service
//	DOKUMENID         hasil unggah lampiran, jalur yang tidak dibawa modul ini
type Input struct {
	Name   string
	Number string
	Code   string

	SellingPrice string

	CategoryID string
	TypeID     string

	Weight        string
	Length        string
	Width         string
	Height        string
	MinStock      string
	MaxStock      string
	OrderQuantity string

	ProductionDate string
	Substitute     string

	Kind         string
	Unit         string
	ActiveStatus string
	PartStatus   string
}

// Panjang maksimum isian teks.
//
// SELURUHNYA ASUMSI YANG DISADARI, bukan angka yang diterima dari Work Owner maupun dibaca
// dari DDL: `POOLDATA.SPAREPART_HE` tidak ada DDL-nya di export (`R-08`), dan sistem lama
// tidak memeriksa panjang satu pun isian — tidak ada satu pun `pyMaxLength` pada layarnya.
//
// Batasnya tetap dipasang karena tanpa itu penolakan datang dari basis data sebagai
// ORA-12899 — galat teknis yang tidak menuntun pengguna ke mana pun.
//
// Angka yang sama diulang di `SparepartForm.tsx`. Bila berubah, KEDUA tempat harus ikut
// berubah — utang yang disadari dari menduplikasi sebuah angka, dijaga terlihat oleh uji
// di mastersparepart_test.go.
const (
	MaxNameLength   = 100
	MaxNumberLength = 50
	MaxCodeLength   = 50
	MaxNumericText  = 20
	MaxDateLength   = 30
	MaxMarkLength   = 30
)

// MaxPrice adalah batas atas harga jual yang masuk akal.
//
// Sistem lama tidak punya batas apa pun. Angka ini BUKAN aturan bisnis melainkan penjaring
// salah ketik: satu nol berlebih pada harga suku cadang mengubah Rp 1,25 juta menjadi Rp
// 12,5 juta, dan tidak ada apa pun di layar lama yang menahannya.
//
// Seratus miliar dipilih karena ia jauh di atas harga suku cadang alat berat mana pun,
// sehingga ia tidak akan pernah menolak isian yang benar. Bila kelak ada yang tertolak,
// itu pertanda angkanya perlu ditinjau — bukan pertanda pengguna salah.
const MaxPrice = 100_000_000_000

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("mastersparepart: sparepart tidak ditemukan")

	// ErrNameTaken: NAMA_SPART yang akan disisipkan sudah dipakai baris lain.
	//
	// Padanan `Activity/ValidateMasterSparepart` langkah "set errmsg jika ada nama yang
	// sama", dan pesan "Nama sparepart sudah ada" pada
	// `Activity/UpdateSparepartHE_act-Act.xml`.
	ErrNameTaken = errors.New("mastersparepart: nama sparepart sudah dipakai")

	// ErrNumberTaken: NO_SPART sudah dipakai baris lain.
	//
	// Padanan langkah "set errmsg jika ada nomor yang sama".
	ErrNumberTaken = errors.New("mastersparepart: nomor sparepart sudah dipakai")

	// ErrCodeTaken: KODE_SPART sudah dipakai baris lain.
	//
	// Padanan langkah "set errmsg jika ada kode yang sama".
	ErrCodeTaken = errors.New("mastersparepart: kode sparepart sudah dipakai")

	// ErrUnknownStatus: status persetujuan di luar "0", "1", "2".
	ErrUnknownStatus = errors.New("mastersparepart: status persetujuan tidak dikenal")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (`P-5`). `Activity/ValidateMasterSparepart`
// menyusun TIGA pesan galat berdampingan — `Local.errmsg`, `errmsg1`, dan `errmsg2` untuk
// nama, nomor, dan kode — lalu menampilkannya bersamaan lewat `Property-Set-Messages`.
// Yang berbeda di sini hanyalah seluruhnya dikirim dalam satu jawaban.
type ValidationError struct {
	Violation []Violation
}

// OneViolation membungkus satu pelanggaran menjadi ValidationError.
//
// Dipakai lapisan aplikasi untuk pemeriksaan yang menuntut pembacaan basis data —
// keunikan nama, nomor, dan kode — supaya galatnya sampai ke layar dalam bentuk yang SAMA
// dengan pelanggaran isian lain, dan menempel pada isiannya.
func OneViolation(field, message string) error {
	return &ValidationError{Violation: []Violation{{Field: field, Message: message}}}
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "mastersparepart: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// Ketiga kunci alami TIDAK di-UPPERCASE. Ketiga rule validasinya memang membandingkan
// `upper(...)`, tetapi yang di-uppercase di sana adalah PEMBANDINGNYA — bukan nilai yang
// disimpan. Memaksa huruf besar pada nama sparepart akan mengubah tampilan setiap baris
// yang disunting, dan itu selisih yang tidak diminta siapa pun.
func (i Input) Clean() Input {
	trim := strings.TrimSpace

	return Input{
		Name:   trim(i.Name),
		Number: trim(i.Number),
		Code:   trim(i.Code),

		SellingPrice: trim(i.SellingPrice),

		CategoryID: trim(i.CategoryID),
		TypeID:     trim(i.TypeID),

		Weight:        trim(i.Weight),
		Length:        trim(i.Length),
		Width:         trim(i.Width),
		Height:        trim(i.Height),
		MinStock:      trim(i.MinStock),
		MaxStock:      trim(i.MaxStock),
		OrderQuantity: trim(i.OrderQuantity),

		ProductionDate: trim(i.ProductionDate),
		Substitute:     trim(i.Substitute),

		Kind:         trim(i.Kind),
		Unit:         trim(i.Unit),
		ActiveStatus: trim(i.ActiveStatus),
		PartStatus:   trim(i.PartStatus),
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Yang diwajibkan, dan dari mana asalnya
//
// EMPAT isian wajib: Nomor, Nama, Kode, dan Harga Jual. Keempatnya bertanda
// `pyRequired=true` pada `Section/BrowseMasterSparepartHEApproval-Section.xml`, dan hanya
// keempat itu — keenam belas isian lain boleh kosong.
//
// Ketiga yang pertama persis yang punya rule keunikannya sendiri
// (`ValidationMasterSparepartNo`, `…Name`, `…Code`), sehingga kewajiban dan keunikan di
// sini berjalan berpasangan.
//
// # Yang DITAMBAHKAN terhadap sistem lama
//
//  1. **Kedelapan isian angka wajib berupa angka.** Sistem lama menerima apa pun, dan
//     `HARGA_JUAL` bertuliskan "seribu" tersimpan apa adanya lalu muncul di laporan.
//  2. **Harga jual tidak boleh negatif, dan dibatasi MaxPrice.** Penjaring salah ketik;
//     lihat MaxPrice.
//  3. **Stok minimal tidak boleh melampaui stok maksimal.** Sistem lama tidak
//     memeriksanya, dan pasangan yang terbalik membuat setiap pemeriksaan stok di modul
//     hilir selalu benar atau selalu salah.
//
// Ketiganya SELISIH YANG DIRENCANAKAN, dan ketiganya menolak isian yang di sistem lama
// diterima. Baris lama yang sudah memuat nilai seperti itu tetap DIBACA apa adanya —
// penolakan hanya terjadi saat barisnya disimpan ulang.
//
// # Yang TIDAK ditambahkan, meski tampak wajar
//
// **Part Substitusi tidak diperiksa keberadaannya.** `Activity/ValidasiSparepart-Act.xml`
// memeriksanya terhadap `NAMA_SPART` dan menjawab "Nama Sparepart tidak ada", tetapi
// activity itu dipanggil dari layar LAIN — bukan dari jalur simpan Master Sparepart.
// Memindahkannya ke sini akan menolak baris yang hari ini tersimpan tanpa keluhan.
func (i Input) Check() error {
	var violation []Violation

	for _, r := range []struct {
		field, label, value string
		max                 int
	}{
		{"nomor_sparepart", "Nomor sparepart", i.Number, MaxNumberLength},
		{"nama_sparepart", "Nama sparepart", i.Name, MaxNameLength},
		{"kode_sparepart", "Kode sparepart", i.Code, MaxCodeLength},
	} {
		violation = append(violation, checkRequired(r.field, r.label, r.value, r.max)...)
	}

	violation = append(violation, i.checkPrice()...)
	violation = append(violation, i.checkNumbers()...)
	violation = append(violation, i.checkStockRange()...)

	for _, r := range []struct {
		field, label, value string
		max                 int
	}{
		{"kategori_sparepart", "Kategori sparepart", i.CategoryID, MaxCodeLength},
		{"tipe_sparepart", "Tipe sparepart", i.TypeID, MaxCodeLength},
		{"tanggal_produksi", "Tanggal produksi", i.ProductionDate, MaxDateLength},
		{"part_substitusi", "Part substitusi", i.Substitute, MaxNameLength},
		{"jenis_sparepart", "Jenis sparepart", i.Kind, MaxMarkLength},
		{"satuan", "Satuan", i.Unit, MaxMarkLength},
		{"status_aktif", "Status aktif", i.ActiveStatus, MaxMarkLength},
		{"status_sparepart", "Status sparepart", i.PartStatus, MaxMarkLength},
	} {
		violation = append(violation, checkLength(r.field, r.label, r.value, r.max)...)
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// checkPrice memeriksa harga jual — satu-satunya angka yang WAJIB diisi.
func (i Input) checkPrice() []Violation {
	if i.SellingPrice == "" {
		return []Violation{{Field: "harga_jual", Message: "Harga jual wajib diisi."}}
	}
	if len(i.SellingPrice) > MaxNumericText {
		return []Violation{{
			Field: "harga_jual",
			Message: fmt.Sprintf("Harga jual paling panjang %d karakter.",
				MaxNumericText),
		}}
	}

	number, err := ParseNumber(i.SellingPrice)
	if err != nil {
		return []Violation{{
			Field:   "harga_jual",
			Message: "Harga jual harus berupa angka, misalnya 1250000 atau 1250000,50.",
		}}
	}
	switch {
	case number < 0:
		return []Violation{{Field: "harga_jual", Message: "Harga jual tidak boleh negatif."}}
	case number > MaxPrice:
		return []Violation{{
			Field:   "harga_jual",
			Message: "Harga jual terlalu besar. Periksa lagi jumlah nolnya.",
		}}
	}
	return nil
}

// checkNumbers memeriksa ketujuh angka yang boleh kosong.
//
// Nilainya TIDAK dipakai untuk apa pun selain pemeriksaan — yang tersimpan tetap teks apa
// adanya, sehingga pembulatan tidak pernah terjadi di jalur ini (`D-51`).
func (i Input) checkNumbers() []Violation {
	var violation []Violation

	for _, r := range []struct{ field, label, value string }{
		{"berat", "Berat sparepart", i.Weight},
		{"panjang", "Panjang sparepart", i.Length},
		{"lebar", "Lebar sparepart", i.Width},
		{"tinggi", "Tinggi sparepart", i.Height},
		{"stock_minimal", "Stock minimal", i.MinStock},
		{"stock_maximal", "Stock maximal", i.MaxStock},
		{"kuantitas_pesanan", "Kuantitas pesanan", i.OrderQuantity},
	} {
		if r.value == "" {
			continue
		}
		if len(r.value) > MaxNumericText {
			violation = append(violation, Violation{
				Field: r.field,
				Message: fmt.Sprintf("%s paling panjang %d karakter.",
					r.label, MaxNumericText),
			})
			continue
		}

		number, err := ParseNumber(r.value)
		if err != nil {
			violation = append(violation, Violation{
				Field:   r.field,
				Message: r.label + " harus berupa angka.",
			})
			continue
		}
		if number < 0 {
			violation = append(violation, Violation{
				Field:   r.field,
				Message: r.label + " tidak boleh negatif.",
			})
		}
	}

	return violation
}

// checkStockRange menolak pasangan batas stok yang terbalik.
//
// Hanya diperiksa bila KEDUANYA terisi dan keduanya angka; kalau salah satunya kosong,
// tidak ada yang dapat dibandingkan, dan kalau salah satunya bukan angka, pesannya sudah
// dilaporkan checkNumbers dan mengulanginya di sini hanya akan membingungkan.
func (i Input) checkStockRange() []Violation {
	if i.MinStock == "" || i.MaxStock == "" {
		return nil
	}

	low, lowErr := ParseNumber(i.MinStock)
	high, highErr := ParseNumber(i.MaxStock)
	if lowErr != nil || highErr != nil || low <= high {
		return nil
	}

	return []Violation{{
		Field:   "stock_minimal",
		Message: "Stock minimal tidak boleh melebihi stock maximal.",
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

// ParseNumber membaca sebuah isian angka.
//
// Koma DAN titik keduanya diterima sebagai pemisah desimal: petugas Indonesia mengetik
// "1250000,5" sementara nilai yang tersimpan di basis data memakai titik. Menolak salah
// satunya berarti menolak isian yang benar hanya karena papan ketiknya. Perlakuan yang
// sama dipakai Master Bengkel dan Master Auto Claim.
//
// Titik sebagai pemisah RIBUAN tidak dikenali, dan itu disengaja: "1.250" dapat berarti
// seribu dua ratus lima puluh ATAU satu koma dua lima nol, dan menebaknya berarti salah
// membaca harga separuh waktu. Yang ditolak tertolak dengan pesan, bukan diterima dengan
// nilai yang salah.
//
// ParseFloat dipakai, bukan Sscanf: Sscanf berhenti pada karakter pertama yang tidak cocok
// dan TETAP melapor sukses, sehingga "1250abc" akan lolos sebagai 1250.
func ParseNumber(text string) (float64, error) {
	clean := strings.TrimSpace(text)
	if strings.Count(clean, ",") > 1 {
		return 0, fmt.Errorf("mastersparepart: %q bukan angka", text)
	}

	number, err := strconv.ParseFloat(strings.ReplaceAll(clean, ",", "."), 64)
	if err != nil {
		return 0, fmt.Errorf("mastersparepart: %q bukan angka: %w", text, err)
	}
	// NaN dan Inf lolos ParseFloat lewat teks "NaN" dan "Inf". Keduanya bukan angka yang
	// dapat dipakai, dan perbandingan rentang di pemanggil tidak menangkapnya: setiap
	// perbandingan dengan NaN bernilai false.
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("mastersparepart: %q bukan angka", text)
	}
	return number, nil
}

// Filter menyaring daftar yang dibaca layar.
//
// Ia cerminan ketiga tab `Section/BrowseMasterSparepartHE-Section.xml`, yang ketiganya
// membaca `BrowseSparepartHE_RD` yang sama dan hanya berbeda pada nilai APPROVAL-nya.
//
// Keyword DITAMBAHKAN terhadap sistem lama. Alasannya bukan kelengkapan: `pyMaxRecords`
// pada report definition-nya bernilai **500** (`BrowseSparepartHE_RD-RD.xml`), sehingga
// daftar Pega memang terpotong di 500 baris tanpa satu pun cara mempersempitnya dari
// layar. Pencarian di sini yang menggantikan pemotongan itu.
type Filter struct {
	// Status wajib salah satu dari tiga yang dikenal.
	Status ApprovalStatus

	// Keyword mempersempit daftar pada nama, nomor, DAN kode sparepart.
	//
	// Ketiganya sekaligus, bukan nama saja seperti Master Panel: ketiganya kunci alami di
	// modul ini, dan petugas gudang mencari sparepart lewat nomornya jauh lebih sering
	// daripada lewat namanya. Kosong berarti tanpa penyaring.
	Keyword string
}

// IDSource menerbitkan ID baru.
//
// Ia seam tersendiri, bukan method pada Repo, karena isinya bukan urusan master sparepart
// melainkan urusan **penomoran**: kode situs dan pola yang sama dipakai keluarga procedure
// `PEGA_M_*` lain pada basis data yang sama.
//
// # Bentuknya, dibaca dari Database/PEGA_M_SPAREPART_HE.prc:12,21
//
//	SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
//	id := id_site || lpad(to_Char(SPAREPART_HE_SEQ.nextval),10,'0');
//
// Ditiru persis, termasuk pembandingnya yang berupa TEKS '1' dan bukan angka.
//
// PERHATIKAN LEBARNYA: **sepuluh** digit, bukan enam seperti Master Panel. Ketiga
// procedure keluarga alat berat ditulis dengan pola yang sama dan lebarnya tetap berbeda;
// menyeragamkannya akan menerbitkan ID yang tidak sebentuk dengan ID yang sudah ada.
type IDSource interface {
	// NextID mengembalikan ID berikutnya.
	NextID(ctx context.Context) (string, error)
}

// ComposeID merangkai ID dari kode situs dan nomor urut.
//
// Ia berada di paket domain, bukan di adapter, karena BENTUK KUNCI adalah aturan domain:
// ia yang menentukan bagaimana sebuah sparepart dikenali, dan ia harus sama persis pada
// adapter SQL maupun adapter memori. Satu tempat, satu bentuk — dan satu uji yang
// menjaganya.
//
// Nomor urut yang LEBIH PANJANG dari lebar yang diminta tidak dipotong. `LPAD` Oracle
// memotongnya dari kanan, sehingga kunci yang dihasilkan akan bertabrakan dengan kunci
// lain — diam-diam. Pada lebar sepuluh digit batas itu tidak akan tersentuh pemakaian yang
// wajar, tetapi sequence yang dipakai ulang atau direset tinggi tidak menuntut sepuluh
// miliar baris untuk sampai ke sana. Di sini ia dibiarkan tumbuh: kuncinya menjadi lebih
// panjang, dan itu terlihat, alih-alih salah tanpa terlihat.
func ComposeID(site string, sequence int64, width int) string {
	number := strconv.FormatInt(sequence, 10)
	if pad := width - len(number); pad > 0 {
		number = strings.Repeat("0", pad) + number
	}
	return strings.TrimSpace(site) + number
}

// Repo adalah seam ke penyimpanan master sparepart SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat
// kueri (ADR-0030 Opsi 1).
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring.
	List(ctx context.Context, filter Filter) ([]Sparepart, error)

	// Get mengembalikan satu baris; ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (Sparepart, error)

	// FindByName mencari baris menurut NAMA_SPART-nya; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/ValidationMasterSparepartName-SQL.xml`, yang mencocokkan
	// `upper(NAMA_SPART)` — perlakuan yang ditiru apa adanya.
	FindByName(ctx context.Context, name string) (Sparepart, error)

	// FindByNumber mencari baris menurut NO_SPART-nya; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/ValidationMasterSparepartNo-SQL.xml`.
	FindByNumber(ctx context.Context, number string) (Sparepart, error)

	// FindByCode mencari baris menurut KODE_SPART-nya; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/ValidationMasterSparepartCode-SQL.xml`.
	FindByCode(ctx context.Context, code string) (Sparepart, error)

	// Insert menyisipkan baris baru.
	//
	// Pemeriksaan KETIGA kunci alami berada DI DALAM operasi repo, bukan dipecah menjadi
	// "cek" lalu "sisip" di lapisan aplikasi. Sistem lama memecahnya —
	// `ValidateMasterSparepart` dipanggil dari layar, `UpdateSparepartHE_act` menyimpan
	// jauh sesudahnya — dan jarak di antara keduanya adalah lubang balapan yang tidak
	// dijaga apa pun.
	//
	// KETERBATASAN YANG DISADARI. Tanpa constraint unik pada ketiga kolom itu, lubang itu
	// hanya dipersempit, tidak ditutup. Penutupnya adalah constraint di basis data, dan
	// itu menunggu DDL (`R-08`) beserta prosedur perubahan skema (`D-63`).
	Insert(ctx context.Context, s Sparepart) error

	// Update menyimpan perubahan pada baris yang sudah ada; ErrNotFound bila barisnya
	// hilang di antara pemuatan layar dan penyimpanan.
	Update(ctx context.Context, s Sparepart) error

	// SetStatus menetapkan APPROVAL sejumlah baris sekaligus.
	//
	// Ia terpisah dari Update karena di sistem lama pun terpisah, dan bentuknya memang
	// borongan: `Activity/SetApprovalAllMaster` menelusuri baris yang
	// `.pySelected=="true"` lalu menetapkan `APPROVAL := Param.approval` pada
	// masing-masing — tanpa menyentuh satu pun kolom lain.
	//
	// TANPA alasan, berbeda dari Master Panel: `POOLDATA.SPAREPART_HE` tidak punya kolom
	// penampungnya. Lihat usecase.Service.Decide.
	//
	// Yang dikembalikan adalah jumlah baris yang benar-benar berubah, supaya pemanggil
	// dapat membedakan "tidak ada yang dipilih" dari "yang dipilih sudah tidak ada".
	SetStatus(ctx context.Context, id []string, status ApprovalStatus) (int, error)
}

// RepoSelector memilih Store milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Store, error)

// Store menyatukan ketiga seam yang dipakai layanan modul ini.
//
// Ketiganya tetap DIDEKLARASIKAN terpisah — Repo untuk tabel master, LookupRepo untuk dua
// tabel acuan yang hanya dibaca, IDSource untuk penomoran — karena ketiganya menjawab
// pertanyaan yang berbeda dan dapat berubah sendiri-sendiri. Yang disatukan hanyalah CARA
// MEMILIHNYA: ketiganya selalu berasal dari koneksi entitas yang sama, sehingga tiga
// pemilih terpisah hanya akan membuka kemungkinan ketiganya menunjuk entitas berbeda.
type Store interface {
	Repo
	LookupRepo
	IDSource
}
