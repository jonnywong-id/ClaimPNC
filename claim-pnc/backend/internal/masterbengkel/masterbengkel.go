// Package masterbengkel adalah inti modul Master Bengkel.
//
// # Apa yang dimodelkan di sini
//
// Daftar **bengkel rekanan** beserta syarat kerja samanya: ke mana pembayarannya
// dikirim, berapa diskon jasa dan sparepart yang berlaku, pajak apa yang dipotong,
// berapa SLA-nya, dan kanal mana saja yang boleh dipakai bengkel itu (e-klaim, auto
// aksep, auto payment, order). Ia master **komersial**, bukan sekadar daftar alamat —
// sebagian kolomnya menentukan hasil hitungan uang pada klaim yang memakai bengkel itu.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/BengkelHE-Harness.xml              layar
//	Section/MasterBengkelHE-Section.xml        judul "MASTER BENGKEL HE"
//	Section/BrowseMasterHE-Section.xml         3 tab: APPROVE · WAITING APPROVAL · REJECT
//	Section/BrowseMasterHEApprove-Section.xml  grid + form, 33 isian berlabel
//	Section/BrowseMasterHEApproval-Section.xml tab menunggu
//	Section/BrowseMasterHEReject-Section.xml   tab ditolak
//	Section/ApprovalMasterBengkelHE-Section.xml persetujuan borongan di Inbox Manager
//	Report Definition/BrowseBengkelHE_RD-RD.xml 40 kolom POOLDATA.BENGKEL_HE
//	RDB List/ValidationMasterBengkel-SQL.xml   tolak nama bengkel ganda
//	RDB List/UpdateBengkelHE-SQL.xml           simpan lewat PEGA_M_BENGKEL_HE
//	RDB List/BrowseCabangBengkelHE-SQL.xml     lookup Cabang
//	RDB List/CountMasterBengkelManagee-SQL.xml pencacah antrean persetujuan
//	RDB List/GetIDDokumenBengkel-SQL.xml       id dokumen lampiran
//	Database/PEGA_M_BENGKEL_HE.prc             pembentukan ID dan penyimpanan JSON
//	Activity/UpdateBengkelHE_act-Act.xml       urutan langkah simpan
//	Activity/ValidateMasterBengkel-Act.xml     pesan galat nama ganda
//	Activity/ValidationLoginBengkel_act-Act.xml pemeriksaan login dan pembuatan operator
//	Activity/SetApprovalAllMaster-Act.xml      persetujuan borongan
//
// # Dua kolom yang memikul dua arti, dan keduanya TIDAK dibawa
//
// Kedua-duanya persis bentuk utang yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat:
//
//   - `ACCOUNT_ID` adalah kolom rekening pada `BENGKEL_HE`, TETAPI pada
//     `RDB List/UpdateBengkelHE-SQL.xml` properti klipboard bernama sama dipakai sebagai
//     **pembawa seluruh dokumen JSON** (`Datapega := {InputBengkel.ACCOUNT_ID}`). Satu
//     nama, dua isi yang sama sekali tidak berhubungan.
//   - `ALASAN_STS_BGKL` adalah alasan status bengkel, TETAPI pada
//     `Activity/SetApprovalAllMaster-Act.xml` ia dipakai membawa **nama tabel**
//     (`"M_BENGKEL_HE"`, `"M_PANEL_HE"`, `"M_SPAREPART_HE"`).
//
// Di modul ini keduanya hanya berarti satu hal: kolomnya sendiri. Pembawa JSON menjadi
// argumen tersendiri pada adapter, dan pemilihan tabel menjadi modul yang berbeda.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterbengkel

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ApprovalStatus adalah posisi sebuah baris dalam alur persetujuan.
//
// Nilainya tetap "0", "1", "2" seperti kolom `APPROVAL` pada `POOLDATA.BENGKEL_HE`:
// tabelnya masih dibaca sistem lama selama masa paralel (ADR-0004), sehingga mengubah
// sandi nilainya akan membuat kedua sistem membaca baris yang sama secara berbeda.
//
// Sandinya kebetulan sama dengan Master Auto Claim dan Master Rekening. Ketiganya
// sengaja TIDAK dipakai bersama: tabelnya berbeda, dan tipe bersama membuat perubahan di
// satu master menyeret master lain.
type ApprovalStatus string

const (
	// StatusPending — diajukan, belum diputuskan. Nilai lahir setiap baris baru DAN
	// setiap baris yang disunting: `Activity/UpdateBengkelHE_act` step 7 menetapkan
	// `TempBengkel.APPROVAL := "0"`.
	StatusPending ApprovalStatus = "0"

	// StatusApproved — disetujui. Hanya baris berstatus ini yang dipakai sistem hilir.
	StatusApproved ApprovalStatus = "1"

	// StatusRejected — ditolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
//
// Teksnya mengikuti caption tab pada `Section/BrowseMasterHE-Section.xml` apa adanya —
// "Approve", "Waiting Approval", "Reject" — supaya petugas membaca kata yang sama dengan
// yang dibacanya di Pega hari ini (D-13).
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

// PartnerStatusNonPartner adalah nilai STATUS_REKANAN yang berarti **bukan rekanan**.
//
// Ia satu-satunya nilai kolom status pada modul ini yang artinya terbaca dari export,
// dan terbacanya bukan dari label melainkan dari percabangan:
// `Activity/ValidationLoginBengkel_act` melompat keluar pada prasyarat
// `Local.STS_REKANAN=='0'` — langkah yang dilewatinya adalah pembuatan login aplikasi.
//
// Artinya: bengkel non-rekanan TIDAK diberi login aplikasi. Selebihnya — apa saja nilai
// sah kolom ini, dan apa arti masing-masing — tidak ada di export. Lihat Workshop.
const PartnerStatusNonPartner = "0"

// Workshop adalah satu baris master bengkel — satu baris `POOLDATA.BENGKEL_HE`.
//
// Keempat puluh kolomnya dibaca dari `Report Definition/BrowseBengkelHE_RD-RD.xml`, dan
// label yang dipakai layar dibaca dari caption `Section/BrowseMasterHEApprove-Section.xml`.
//
// # Kenapa hampir semuanya bertipe teks
//
// Termasuk yang namanya jelas angka — PPN, diskon, persen material, SLA — dan yang
// namanya jelas tanggal, TGL_STATUS. Alasannya satu: **DDL tabelnya belum diterima**
// (`R-08`), sehingga tipe kolom yang sebenarnya belum diketahui. Mengubahnya menjadi
// angka atau waktu di sini berarti memutuskan presisi, pembulatan, dan zona waktu tanpa
// dasar — dan nilai uang maupun persentase tidak boleh ditebak (`D-51`, `F-5`).
//
// Baris lama karena itu dibaca apa adanya; yang baru diperiksa berbentuk angka lewat
// Input.Check tanpa pernah diubah bentuknya saat disimpan.
type Workshop struct {
	// ID adalah kolom ID_BENGKEL — kunci baris ini.
	//
	// Ia TIDAK diketik pengguna. `Database/PEGA_M_BENGKEL_HE.prc:19` menerbitkannya
	// sebagai kode situs ditambah nomor urut sepuluh digit; lihat IDSource.
	ID string

	// Name adalah kolom NAMA_BENGKEL. Ia kunci alami: penambahan ditolak bila namanya
	// sudah dipakai baris lain (`RDB List/ValidationMasterBengkel-SQL.xml`).
	Name string

	Address string // ALM_BENGKEL
	Phone   string // TELP_BENGKEL
	Mobile  string // NOHP_BENGKEL
	Email   string // MAIL

	// WorkOrderEmail adalah kolom MAIL_WO — surel tujuan perintah kerja, terpisah dari
	// surel umum bengkel.
	WorkOrderEmail string

	// BranchID dan BranchName adalah kolom CABANG_ID dan NAMA_CABANG. Keduanya dipilih
	// berpasangan dari lookup Cabang.
	BranchID   string
	BranchName string

	// CityID dan CityName adalah kolom CITY_ID dan NAMA_KABUPATEN.
	//
	// Perhatikan namanya: kolom bernama NAMA_KABUPATEN diberi label **"NAMA KOTA"** di
	// layar Pega, dan sumbernya adalah `BrowseCity_RD` atas kelas
	// `ASM-FW-GISFW-Int-CITY`. Label layar yang diikuti (D-13); nama kolom tetap apa
	// adanya karena tabelnya dimiliki bersama Pega (D-21).
	CityID   string
	CityName string

	// PartnerStatus adalah kolom STATUS_REKANAN. Lihat PartnerStatusNonPartner.
	PartnerStatus string

	// WorkshopStatus, StatusReason, dan StatusDate adalah kolom STS_BENGKEL,
	// ALASAN_STS_BGKL, dan TGL_STATUS — status operasional bengkel beserta alasan dan
	// tanggal berlakunya.
	WorkshopStatus string
	StatusReason   string
	StatusDate     string

	// Login adalah kolom LOGIN_APLIKASI.
	//
	// Ia DIKETIK pengguna, bukan diterbitkan sistem: `ValidationLoginBengkel_act` step 4
	// menyalinnya dari `Param.LOGIN` apa adanya. Yang diterbitkan sistem di Pega adalah
	// AKUNNYA, dan akun itu tidak dibawa — lihat catatan pada ErrLoginTaken.
	Login string

	// BankID, BankName, AccountNumber, dan AccountName adalah kolom BANK_ID, NAMA_BANK,
	// NO_ACCOUNT, dan NAMA_ACCOUNT.
	//
	// Hanya NAMA_BANK dan NO_ACCOUNT yang punya caption di layar Pega ("NAMA BANK" dan
	// "NO REKENING"); dua sisanya ada di tabel tetapi tidak pernah ditampilkan. Keduanya
	// tetap dibaca dan ditulis kembali apa adanya supaya penyimpanan lewat layar baru
	// tidak mengosongkan kolom yang tidak dilihat siapa pun.
	BankID        string
	BankName      string
	AccountNumber string
	AccountName   string

	// AccountID adalah kolom ACCOUNT_ID — dan HANYA kolom itu.
	//
	// Properti klipboard bernama sama di Pega dipakai membawa dokumen JSON; lihat
	// catatan paket. Di sini ia tidak pernah berarti apa pun selain isinya sendiri.
	AccountID string

	// TaxName, TaxNumber, TaxAddress, dan IncomeTaxType adalah kolom NAMA_NPWP, NO_NPWP,
	// ALM_NPWP, dan JENIS_PPH.
	TaxName       string
	TaxNumber     string
	TaxAddress    string
	IncomeTaxType string

	// Keempat nilai berikut adalah persentase, disimpan sebagai TEKS presisi penuh
	// (`D-51`): PPN, DISC_JASA, DISC_SPART, PERSEN_MATERIAL.
	ValueAddedTax   string
	ServiceDiscount string
	PartDiscount    string
	MaterialPercent string

	// PriceListGapPercent adalah kolom PCT_SELISIH_PL — persentase selisih terhadap
	// harga price list. Ada di tabel, tidak pernah ditampilkan layar Pega.
	PriceListGapPercent string

	// SLA adalah kolom SLA — janji waktu penyelesaian. Satuannya tidak disebut di mana
	// pun dalam export.
	SLA string

	// Ketujuh penanda berikut adalah kolom STS_SUPPLY, STS_EKLAIM, STS_AUTO_AKSEP,
	// STS_PAYMENT, STS_AUTOPAYMENT, STS_TEKNO, dan STS_ORDER.
	//
	// NILAI SAHNYA TIDAK DIKETAHUI. Ketujuhnya dirender `pxRadioButtons` atau
	// `pxDropdown` di Pega, dan daftar pilihannya ada di rule Field Value yang **tidak
	// ikut di export** (`R-16`). Keduanya karena itu diperlakukan sebagai teks apa adanya
	// — bukan boolean, bukan enum — sampai daftar nilainya diterima.
	SuppliedByASM     string
	EClaimStatus      string
	AutoAcceptStatus  string
	PaymentStatus     string
	AutoPaymentStatus string
	TeknoStatus       string
	OrderStatus       string

	// Supplier adalah kolom SUPPLIER. Ada di tabel, tanpa caption di layar Pega.
	Supplier string

	// DocumentID adalah kolom DOKUMENID — lampiran yang ditautkan ke baris ini.
	//
	// Ia diisi `Activity/UpdateBengkelHE_act` step 4 dari hasil `PNCSaveAttachmentToDB`.
	// Unggah lampirannya TIDAK dibawa modul ini — lihat catatan pada usecase.Service.
	DocumentID string

	// Status adalah kolom APPROVAL.
	Status ApprovalStatus
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Tiga puluh tiga isian, mengikuti caption `Section/BrowseMasterHEApprove-Section.xml`.
// Yang ADA di tabel tetapi TIDAK di sini, karena ketiganya diturunkan sistem:
//
//	ID_BENGKEL  diterbitkan saat penambahan; lihat IDSource
//	APPROVAL    selalu StatusPending pada penyimpanan lewat layar ini
//	DOKUMENID   hasil unggah lampiran, jalur yang tidak dibawa modul ini
type Input struct {
	Name    string
	Address string
	Phone   string
	Mobile  string
	Email   string

	WorkOrderEmail string

	BranchID   string
	BranchName string
	CityID     string
	CityName   string

	PartnerStatus  string
	WorkshopStatus string
	StatusReason   string
	StatusDate     string

	Login string

	BankID        string
	BankName      string
	AccountNumber string
	AccountName   string
	AccountID     string

	TaxName       string
	TaxNumber     string
	TaxAddress    string
	IncomeTaxType string

	ValueAddedTax   string
	ServiceDiscount string
	PartDiscount    string
	MaterialPercent string

	PriceListGapPercent string
	SLA                 string

	SuppliedByASM     string
	EClaimStatus      string
	AutoAcceptStatus  string
	PaymentStatus     string
	AutoPaymentStatus string
	TeknoStatus       string
	OrderStatus       string

	Supplier string
}

// Panjang maksimum isian teks.
//
// SELURUHNYA ASUMSI YANG DISADARI, bukan angka yang diterima dari Work Owner maupun
// dibaca dari DDL: `POOLDATA.BENGKEL_HE` tidak ada DDL-nya di export (`R-08`), dan
// sistem lama tidak memeriksa panjang satu pun isian.
//
// Batasnya tetap dipasang karena tanpa itu penolakan datang dari basis data sebagai
// ORA-12899 — galat teknis yang tidak menuntun pengguna ke mana pun.
//
// Angka yang sama diulang di `WorkshopForm.tsx`. Bila berubah, KEDUA tempat harus ikut
// berubah — utang yang disadari dari menduplikasi sebuah angka, dijaga terlihat oleh uji
// di masterbengkel_test.go.
const (
	MaxNameLength    = 100
	MaxAddressLength = 250
	MaxPhoneLength   = 30
	MaxEmailLength   = 100
	MaxLoginLength   = 32
	MaxBankLength    = 100
	MaxAccountLength = 30
	MaxTaxIDLength   = 30
	MaxCodeLength    = 20
	MaxShortLength   = 50
	MaxReasonLength  = 250
)

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("masterbengkel: bengkel tidak ditemukan")

	// ErrNameTaken: NAMA_BENGKEL yang akan disisipkan sudah dipakai baris lain.
	//
	// Padanan langsung pesan `Activity/ValidateMasterBengkel`:
	// "Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain."
	ErrNameTaken = errors.New("masterbengkel: nama bengkel sudah dipakai")

	// ErrLoginTaken: LOGIN_APLIKASI yang dikirim sudah dipakai.
	//
	// # Yang DIPERIKSA berbeda dari sistem lama, dan itu tidak terhindarkan
	//
	// `ValidationLoginBengkel_act` step 5 memeriksanya terhadap daftar operator Pega
	// (`Data-Admin-Operator-ID` lewat report `GCNMGetListOfOperators`), lalu step 9
	// MEMBUAT operator baru bila bengkelnya bukan non-rekanan.
	//
	// Sistem baru tidak punya operator Pega, dan kontrak identitasnya sendiri belum ada
	// (`F-3`, `R-14`). Pembuatan akun karena itu TIDAK DIBAWA, dan pemeriksaan
	// keunikannya dialihkan ke tempat yang tersedia: kolom LOGIN_APLIKASI pada
	// `POOLDATA.BENGKEL_HE` itu sendiri.
	//
	// Satu hal yang TIDAK BOLEH dibawa apa pun keadaannya: `ValidationLoginBengkel_act`
	// step 2 menetapkan `Local.password := "123456"` — kata sandi yang sama untuk setiap
	// bengkel yang pernah dibuatkan akun. Mereplikasinya berarti menerbitkan akun dengan
	// kata sandi yang sudah diketahui siapa pun yang pernah membaca rule ini.
	ErrLoginTaken = errors.New("masterbengkel: login aplikasi sudah dipakai")

	// ErrUnknownStatus: status persetujuan di luar "0", "1", "2".
	ErrUnknownStatus = errors.New("masterbengkel: status persetujuan tidak dikenal")
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
// Ini kesetaraan perilaku, bukan selera (`P-5`). Pega pun menampilkan pesannya lewat
// `Page-Set-Messages` yang menempel pada isiannya masing-masing; yang berbeda hanyalah
// di sini seluruhnya dikirim dalam satu jawaban.
type ValidationError struct {
	Violation []Violation
}

// OneViolation membungkus satu pelanggaran menjadi ValidationError.
//
// Dipakai lapisan aplikasi untuk pemeriksaan yang menuntut pembacaan basis data —
// keunikan nama dan login — supaya galatnya sampai ke layar dalam bentuk yang SAMA
// dengan pelanggaran isian lain, dan menempel pada isiannya.
func OneViolation(field, message string) error {
	return &ValidationError{Violation: []Violation{{Field: field, Message: message}}}
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "masterbengkel: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
func (i Input) Clean() Input {
	trim := strings.TrimSpace
	return Input{
		Name:                trim(i.Name),
		Address:             trim(i.Address),
		Phone:               trim(i.Phone),
		Mobile:              trim(i.Mobile),
		Email:               trim(i.Email),
		WorkOrderEmail:      trim(i.WorkOrderEmail),
		BranchID:            trim(i.BranchID),
		BranchName:          trim(i.BranchName),
		CityID:              trim(i.CityID),
		CityName:            trim(i.CityName),
		PartnerStatus:       trim(i.PartnerStatus),
		WorkshopStatus:      trim(i.WorkshopStatus),
		StatusReason:        trim(i.StatusReason),
		StatusDate:          trim(i.StatusDate),
		Login:               trim(i.Login),
		BankID:              trim(i.BankID),
		BankName:            trim(i.BankName),
		AccountNumber:       trim(i.AccountNumber),
		AccountName:         trim(i.AccountName),
		AccountID:           trim(i.AccountID),
		TaxName:             trim(i.TaxName),
		TaxNumber:           trim(i.TaxNumber),
		TaxAddress:          trim(i.TaxAddress),
		IncomeTaxType:       trim(i.IncomeTaxType),
		ValueAddedTax:       trim(i.ValueAddedTax),
		ServiceDiscount:     trim(i.ServiceDiscount),
		PartDiscount:        trim(i.PartDiscount),
		MaterialPercent:     trim(i.MaterialPercent),
		PriceListGapPercent: trim(i.PriceListGapPercent),
		SLA:                 trim(i.SLA),
		SuppliedByASM:       trim(i.SuppliedByASM),
		EClaimStatus:        trim(i.EClaimStatus),
		AutoAcceptStatus:    trim(i.AutoAcceptStatus),
		PaymentStatus:       trim(i.PaymentStatus),
		AutoPaymentStatus:   trim(i.AutoPaymentStatus),
		TeknoStatus:         trim(i.TeknoStatus),
		OrderStatus:         trim(i.OrderStatus),
		Supplier:            trim(i.Supplier),
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Yang diwajibkan, dan dari mana asalnya
//
// Sistem lama **tidak punya satu pun prasyarat "wajib diisi"** pada layar ini — tidak
// seperti Master Auto Claim yang menolak delapan isian kosong sekaligus. Yang ada
// hanyalah dua pemeriksaan keunikan (nama bengkel dan login aplikasi), dan keduanya
// menuntut pembacaan basis data sehingga dikerjakan lapisan aplikasi.
//
// Yang diwajibkan di sini karena itu dibatasi pada **tiga isian yang tanpanya baris ini
// tidak dapat dipakai siapa pun**, dan ketiganya dapat dibenarkan dari export:
//
//	NAMA_BENGKEL    ia kunci alami — ValidationMasterBengkel mencari baris DENGAN nama
//	                itu, sehingga baris tanpa nama tidak akan pernah tertangkap
//	                pemeriksaan ganda, dan dua baris tanpa nama akan lolos berdampingan
//	STATUS_REKANAN  ia yang menentukan apakah bengkel diberi login (ValidationLoginBengkel_act)
//	LOGIN_APLIKASI  wajib HANYA bila bengkelnya rekanan; lihat checkLogin
//
// Selebihnya boleh kosong, persis seperti hari ini. Mewajibkan lebih banyak akan
// menolak penambahan yang sekarang diterima — dan itu selisih perilaku yang tidak
// diminta siapa pun.
//
// # Yang DITAMBAHKAN terhadap sistem lama
//
// Keempat persentase dan PCT_SELISIH_PL diperiksa berbentuk angka 0–100 bila diisi.
// Sistem lama menerima teks apa pun, termasuk "abc", dan akibatnya baru muncul jauh di
// hilir pada perhitungan yang memakainya — tanpa satu pun tanda bahwa asalnya dari baris
// master ini. SELISIH YANG DIRENCANAKAN; isian yang dulu lolos kini ditolak, dan baris
// lama tetap dibaca apa adanya.
func (i Input) Check() error {
	var violation []Violation

	violation = append(violation, checkRequired("nama_bengkel", "Nama bengkel", i.Name, MaxNameLength)...)
	violation = append(violation, checkRequired("status_rekanan", "Status rekanan", i.PartnerStatus, MaxCodeLength)...)
	violation = append(violation, i.checkLogin()...)

	for _, r := range []struct {
		field, label, value string
		max                 int
	}{
		{"alamat_bengkel", "Alamat bengkel", i.Address, MaxAddressLength},
		{"telp_bengkel", "Telepon bengkel", i.Phone, MaxPhoneLength},
		{"nohp_bengkel", "No HP bengkel", i.Mobile, MaxPhoneLength},
		{"email", "Email", i.Email, MaxEmailLength},
		{"email_wo", "Email WO", i.WorkOrderEmail, MaxEmailLength},
		{"id_cabang", "Kode cabang", i.BranchID, MaxCodeLength},
		{"nama_cabang", "Nama cabang", i.BranchName, MaxNameLength},
		{"id_kota", "Kode kota", i.CityID, MaxCodeLength},
		{"nama_kota", "Nama kota", i.CityName, MaxNameLength},
		{"status_bengkel", "Status bengkel", i.WorkshopStatus, MaxCodeLength},
		{"alasan_status_bengkel", "Alasan status bengkel", i.StatusReason, MaxReasonLength},
		{"tanggal_status", "Tanggal status", i.StatusDate, MaxShortLength},
		{"id_bank", "Kode bank", i.BankID, MaxCodeLength},
		{"nama_bank", "Nama bank", i.BankName, MaxBankLength},
		{"no_rekening", "No rekening", i.AccountNumber, MaxAccountLength},
		{"nama_rekening", "Nama rekening", i.AccountName, MaxNameLength},
		{"id_rekening", "ID rekening", i.AccountID, MaxCodeLength},
		{"nama_npwp", "Nama NPWP", i.TaxName, MaxNameLength},
		{"no_npwp", "No NPWP", i.TaxNumber, MaxTaxIDLength},
		{"alamat_npwp", "Alamat NPWP", i.TaxAddress, MaxAddressLength},
		{"jenis_pph", "Jenis PPh", i.IncomeTaxType, MaxShortLength},
		{"sla", "SLA", i.SLA, MaxShortLength},
		{"supplier", "Supplier", i.Supplier, MaxNameLength},
		{"status_disupply_asm", "Status disupply ASM", i.SuppliedByASM, MaxCodeLength},
		{"status_eklaim", "Status e-klaim", i.EClaimStatus, MaxCodeLength},
		{"status_auto_aksep", "Status auto aksep", i.AutoAcceptStatus, MaxCodeLength},
		{"status_payment", "Status payment", i.PaymentStatus, MaxCodeLength},
		{"status_autopayment", "Status autopayment", i.AutoPaymentStatus, MaxCodeLength},
		{"status_tekno", "Status Tekno", i.TeknoStatus, MaxCodeLength},
		{"status_order", "Status order", i.OrderStatus, MaxCodeLength},
	} {
		violation = append(violation, checkLength(r.field, r.label, r.value, r.max)...)
	}

	for _, r := range []struct {
		field, label, value string
	}{
		{"ppn", "PPN", i.ValueAddedTax},
		{"diskon_jasa", "Discount jasa", i.ServiceDiscount},
		{"diskon_sparepart", "Discount sparepart", i.PartDiscount},
		{"persen_material", "Persen material", i.MaterialPercent},
		{"pct_selisih_pl", "PCT selisih price list", i.PriceListGapPercent},
	} {
		violation = append(violation, checkPercent(r.field, r.label, r.value)...)
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// checkLogin memeriksa LOGIN_APLIKASI.
//
// Ia wajib HANYA bila bengkelnya rekanan, dan syarat itu bukan karangan: Pega melompati
// seluruh urusan login pada prasyarat `Local.STS_REKANAN=='0'`
// (`ValidationLoginBengkel_act`). Bengkel non-rekanan memang tidak pernah punya login,
// sehingga mewajibkannya akan menolak baris yang hari ini sah.
//
// Kebalikannya juga dijaga: bengkel rekanan TANPA login akan tersimpan diam-diam tanpa
// pernah dapat masuk, dan tidak ada satu pun layar yang menjelaskan kenapa.
func (i Input) checkLogin() []Violation {
	if i.PartnerStatus == PartnerStatusNonPartner {
		// Non-rekanan: login boleh kosong, tetapi bila diisi tetap diperiksa panjangnya.
		return checkLength("login_aplikasi", "Login aplikasi", i.Login, MaxLoginLength)
	}
	return checkRequired("login_aplikasi", "Login aplikasi", i.Login, MaxLoginLength)
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

// checkPercent memeriksa satu isian persentase yang boleh kosong.
//
// Nilainya TIDAK dipakai untuk apa pun selain pemeriksaan — yang tersimpan tetap teks
// apa adanya, sehingga pembulatan tidak pernah terjadi di jalur ini (`D-51`).
func checkPercent(field, label, value string) []Violation {
	if value == "" {
		return nil
	}

	number, err := parsePercent(value)
	if err != nil {
		return []Violation{{
			Field:   field,
			Message: label + " harus berupa angka, misalnya 11 atau 12,5.",
		}}
	}
	if number < 0 || number > 100 {
		return []Violation{{
			Field:   field,
			Message: label + " harus di antara 0 dan 100.",
		}}
	}
	return nil
}

// parsePercent membaca sebuah persentase sebagai angka.
//
// Koma DAN titik keduanya diterima sebagai pemisah desimal: petugas Indonesia mengetik
// "12,5" sementara nilai yang tersimpan di basis data memakai "12.5". Menolak salah
// satunya berarti menolak isian yang benar hanya karena papan ketiknya.
//
// ParseFloat dipakai, bukan Sscanf: Sscanf berhenti pada karakter pertama yang tidak
// cocok dan TETAP melapor sukses, sehingga "12abc" akan lolos sebagai 12.
func parsePercent(text string) (float64, error) {
	number, err := strconv.ParseFloat(strings.ReplaceAll(text, ",", "."), 64)
	if err != nil {
		return 0, fmt.Errorf("masterbengkel: %q bukan angka: %w", text, err)
	}
	// NaN dan Inf lolos ParseFloat lewat teks "NaN" dan "Inf". Keduanya bukan
	// persentase, dan perbandingan rentang di pemanggil tidak menangkapnya: setiap
	// perbandingan dengan NaN bernilai false.
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("masterbengkel: %q bukan angka", text)
	}
	return number, nil
}

// Filter menyaring daftar yang dibaca layar.
//
// Ia cerminan ketiga tab `Section/BrowseMasterHE-Section.xml`, yang ketiganya membaca
// `BrowseBengkelHE_RD` yang sama dan hanya berbeda pada nilai APPROVAL-nya.
//
// Keyword DITAMBAHKAN terhadap sistem lama. Alasannya bukan kelengkapan: `pyMaxRecords`
// pada report definition-nya bernilai **500** (`BrowseBengkelHE_RD-RD.xml`), sehingga
// daftar Pega memang terpotong di 500 baris tanpa satu pun cara mempersempitnya dari
// layar. Pencarian di sini yang menggantikan pemotongan itu.
type Filter struct {
	// Status wajib salah satu dari tiga yang dikenal.
	Status ApprovalStatus

	// Keyword mempersempit daftar pada nama, kota, atau cabang bengkel. Kosong berarti
	// tanpa penyaring.
	Keyword string
}

// IDSource menerbitkan ID_BENGKEL baru.
//
// Ia seam tersendiri, bukan method pada Repo, karena isinya bukan urusan master bengkel
// melainkan urusan **penomoran**: kode situs dan sequence yang sama dipakai lima belas
// procedure `PEGA_M_*` lain pada basis data yang sama.
//
// # Bentuknya, dibaca dari Database/PEGA_M_BENGKEL_HE.prc:11,19
//
//	SELECT ID INTO id_site FROM M_SITE_DATABASE WHERE CURRENT_SITE = '1';
//	id_bengkel := id_site || lpad(to_Char(BENGKEL_HE_SEQ.nextval),10,'0');
//
// Ditiru persis, termasuk pembandingnya yang berupa TEKS '1' dan bukan angka, dan
// termasuk lebar sepuluh digit. Deretnya karena itu melanjutkan deret yang sudah ada dan
// tidak pernah bertabrakan dengan ID yang pernah diterbitkan Pega.
//
// Perlakuan yang sama dipakai Master Status Klaim (`claim_status_site` dan
// `claim_status_next_sequence`) atas keluarga procedure yang sama.
type IDSource interface {
	// NextID mengembalikan ID_BENGKEL berikutnya.
	NextID(ctx context.Context) (string, error)
}

// ComposeID merangkai ID_BENGKEL dari kode situs dan nomor urut.
//
// Ia berada di paket domain, bukan di adapter, karena BENTUK KUNCI adalah aturan
// domain: ia yang menentukan bagaimana sebuah bengkel dikenali, dan ia harus sama persis
// pada adapter SQL maupun adapter memori. Satu tempat, satu bentuk — dan satu uji yang
// menjaganya.
//
// Meniru `Database/PEGA_M_BENGKEL_HE.prc:19`:
//
//	id_bengkel := id_site || lpad(to_Char(BENGKEL_HE_SEQ.nextval),10,'0');
//
// Nomor urut yang LEBIH PANJANG dari lebar yang diminta tidak dipotong. `LPAD` Oracle
// memotongnya dari kanan, sehingga urutan ke-10.000.000.000 akan menghasilkan kunci yang
// bertabrakan dengan urutan lain — diam-diam. Di sini ia dibiarkan tumbuh: kuncinya
// menjadi lebih panjang, dan itu terlihat, alih-alih salah tanpa terlihat.
func ComposeID(site string, sequence int64, width int) string {
	number := strconv.FormatInt(sequence, 10)
	if pad := width - len(number); pad > 0 {
		number = strings.Repeat("0", pad) + number
	}
	return strings.TrimSpace(site) + number
}

// Repo adalah seam ke penyimpanan master bengkel SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat
// kueri (ADR-0030 Opsi 1).
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring.
	List(ctx context.Context, filter Filter) ([]Workshop, error)

	// Get mengembalikan satu baris; ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (Workshop, error)

	// FindByName mencari baris menurut NAMA_BENGKEL-nya; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/ValidationMasterBengkel-SQL.xml`, yang mencocokkan
	// `upper(trim(nama_bengkel))` — perlakuan yang ditiru apa adanya.
	FindByName(ctx context.Context, name string) (Workshop, error)

	// FindByLogin mencari baris menurut LOGIN_APLIKASI-nya; ErrNotFound bila tidak ada.
	//
	// Sistem lama memeriksanya terhadap daftar operator Pega; lihat ErrLoginTaken untuk
	// alasan pengalihannya ke kolom tabel ini.
	FindByLogin(ctx context.Context, login string) (Workshop, error)

	// Insert menyisipkan baris baru.
	//
	// Pemeriksaan nama dan login ganda berada DI DALAM operasi repo, bukan dipecah
	// menjadi "cek" lalu "sisip" di lapisan aplikasi. Sistem lama memecahnya —
	// `ValidateMasterBengkel` dipanggil dari layar, `UpdateBengkelHE_act` menyimpan jauh
	// sesudahnya — dan jarak di antara keduanya adalah lubang balapan yang tidak dijaga
	// apa pun.
	//
	// KETERBATASAN YANG DISADARI. Tanpa constraint unik pada NAMA_BENGKEL dan
	// LOGIN_APLIKASI, lubang itu hanya dipersempit, tidak ditutup. Penutupnya adalah
	// constraint di basis data, dan itu menunggu DDL (`R-08`) beserta prosedur perubahan
	// skema (`D-63`).
	Insert(ctx context.Context, w Workshop) error

	// Update menyimpan perubahan pada baris yang sudah ada; ErrNotFound bila barisnya
	// hilang di antara pemuatan layar dan penyimpanan.
	Update(ctx context.Context, w Workshop) error

	// SetStatus menetapkan APPROVAL sejumlah baris sekaligus.
	//
	// Ia terpisah dari Update karena di sistem lama pun terpisah, dan bentuknya memang
	// borongan: `Activity/SetApprovalAllMaster` menelusuri baris yang
	// `.pySelected=="true"` lalu menetapkan `APPROVAL := Param.approval` pada
	// masing-masing — tanpa menyentuh satu pun kolom lain.
	//
	// Yang dikembalikan adalah jumlah baris yang benar-benar berubah, supaya pemanggil
	// dapat membedakan "tidak ada yang dipilih" dari "yang dipilih sudah tidak ada".
	SetStatus(ctx context.Context, id []string, status ApprovalStatus) (int, error)
}

// RepoSelector memilih Store milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada
// saat permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Store, error)

// Store menyatukan ketiga seam yang dipakai layanan modul ini.
//
// Ketiganya tetap DIDEKLARASIKAN terpisah — Repo untuk tabel master, LookupRepo untuk
// tiga tabel acuan yang hanya dibaca, IDSource untuk penomoran — karena ketiganya
// menjawab pertanyaan yang berbeda dan dapat berubah sendiri-sendiri. Yang disatukan
// hanyalah CARA MEMILIHNYA: ketiganya selalu berasal dari koneksi entitas yang sama,
// sehingga tiga pemilih terpisah hanya akan membuka kemungkinan ketiganya menunjuk
// entitas yang berbeda.
type Store interface {
	Repo
	LookupRepo
	IDSource
}
