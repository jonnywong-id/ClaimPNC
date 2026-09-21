// Package mastersupplier adalah inti modul Master Supplier.
//
// # Apa yang dimodelkan di sini
//
// Daftar **supplier rekanan** beserta syarat dagangnya: ke mana pembayarannya dikirim,
// berapa tenggat bayar dan tenggat kirimnya, ia rekanan atau bukan, dan apakah ia
// termasuk supplier Heavy Equipment. Sama seperti Master Bengkel, ia master **komersial**
// — sebagian kolomnya menentukan ke rekening siapa uang berpindah.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/MasterSupplier-Harness.xml             layar, tombol New Supplier · Edit · Refresh
//	Section/InboxMasterSupplier-Section.xml        grid 8 kolom + pencacah
//	Section/CreateMasterSupplier_Sec-Section.xml   form 24 isian, dipakai Create DAN Edit
//	Section/DataCountMasterSupllier-Section.xml    "Total Data :"
//	Flow Action/CreateMasterSupplier-FlowAction.xml   section + post CreateNewMasterSupplier_post
//	Flow Action/EditMasterSupplier-FlowAction.xml     pre GetDataSupplier_pre + post EditMasterSupplier_post
//	Activity/CreateNewMasterSupplier_post-Act.xml  urutan langkah penambahan
//	Activity/EditMasterSupplier_post-Act.xml       urutan langkah penyimpanan
//	Activity/GetDataSupplier_pre-Act.xml           pemuatan ke form, dan penurunan JENIS_STATUS
//	RDB List/GetDataEditMasterSupller-SQL.xml      25 kunci JSON, terbaca satu per satu
//	RDB List/KonversiMasterSupplier_SQL-SQL.xml    pemanggilan procedure penyimpanan
//	RDB List/InsertProteksiKlaimMBU_SQL-SQL.xml    baris permintaan persetujuan
//	Database/PEGA_M_SUPPLIER.prc                   pembentukan ID dan penyimpanan JSON
//	Report Definition/BrowseCity_RD-RD.xml         lookup Kota
//	Report Definition/BrowseBranchForGKM_RD-RD.xml lookup Cabang
//	Report Definition/BrowseCountry_RD-RD.xml      lookup Negara
//	Report Definition/BrowseBankGroup-RD.xml       lookup Bank
//
// # Perbedaan terbesar dari Master Bengkel: tabelnya HANYA JSON
//
// `POOLDATA.BENGKEL_HE` punya empat puluh kolom bernama, sehingga Master Bengkel dapat
// menulis kolom. `M_SUPPLIER` **tidak punya kembaran relasional sama sekali** —
// `Database/PEGA_M_SUPPLIER.prc:24,36` membuktikan ia hanya `ID` dan `JSONDATA`, ditambah
// `OLDID` yang terbaca dari kueri bacanya.
//
// Di Master Bengkel, menulis dokumen JSON DITOLAK karena nama kuncinya tidak diketahui:
// `@GCNM.GetPageJSONString()` hanya ada tanda tangannya di export. Di sini keadaannya
// berbeda dan itu yang menentukan keputusannya: `GetDataEditMasterSupller-SQL.xml:90-118`
// membaca setiap kunci satu per satu lewat notasi titik Oracle, sehingga **kedua puluh
// lima kuncinya terbaca lengkap tanpa satu pun tebakan**.
//
// Yang TIDAK dibawa tetap sama: procedure-nya tidak dipanggil (`D-02`). Go menyusun
// dokumen JSON-nya sendiri dan menulisnya dengan pernyataan biasa.
//
// # Satu kolom yang memikul dua arti, dan artinya TIDAK dibawa
//
// `NO_KLAIM` pada `pooldata.proteksi_klaimmbu` diisi **ID supplier**, bukan nomor klaim
// (`RDB List/InsertProteksiKlaimMBU_SQL-SQL.xml`). Persis bentuk utang yang
// `03-CURRENT-ARCHITECTURE.md` §4.2 catat. Barisnya tetap ditulis apa adanya supaya
// antrean persetujuan yang berjalan hari ini tidak putus, tetapi di dalam modul ini nilai
// itu bernama apa adanya — ID supplier. Lihat approval.go.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package mastersupplier

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// SupplyTypeHeavyEquipment adalah nilai JENIS_STATUS yang berarti supplier Heavy Equipment.
//
// Ia satu-satunya nilai pada modul ini yang artinya terbaca dari export, dan terbacanya
// dari percabangan — bukan dari label:
//
//	Activity/CreateNewMasterSupplier_post step 7  @equals(MasterSupplier.JENIS_STATUS,"1")
//	                                              -> MasterSupplier.SUPPLIER_HE := "1"
//	Activity/GetDataSupplier_pre step 6.3         @String.equals(.SUPPLIER_HE,"1")
//	                                              -> MasterSupplier.JENIS_STATUS := "1"
//	                                              selain itu "0"
//
// Kedua arah itu saling membalik, dan itulah yang membuat JENIS_STATUS dan SUPPLIER_HE
// terbukti dua wajah dari satu hal. Lihat Supplier.SupplyType.
const SupplyTypeHeavyEquipment = "1"

// SupplyTypeOther adalah nilai JENIS_STATUS selain Heavy Equipment.
//
// Asalnya cabang "selain itu" pada `GetDataSupplier_pre` yang menetapkan `"0"`.
const SupplyTypeOther = "0"

// ActiveYes dan ActiveNo adalah nilai STS_AKTIF dan STS_AKTIF_PROMLIST.
//
// Keduanya terbaca dari percabangan, bukan dari daftar nilai — daftar nilainya ada di
// rule Field Value yang tidak ikut di export (`R-16`):
//
//	CreateNewMasterSupplier_post step 6   MasterSupplier.STS_AKTIF := "0"
//	                                      supplier baru lahir TIDAK aktif
//	EditMasterSupplier_post step 12       @String.equals(STS_AKTIF_PROMLIST,"1") || ...,""
//	                                      hanya yang aktif meminta persetujuan
const (
	ActiveNo  = "0"
	ActiveYes = "1"
)

// Supplier adalah satu baris master supplier — satu baris `M_SUPPLIER`.
//
// # Kenapa seluruhnya bertipe teks
//
// Termasuk yang namanya jelas tenggat waktu, TOP dan TOD, dan yang namanya jelas tanggal,
// TGL_INSERT. Alasannya dua, dan keduanya mengikat:
//
//  1. Isinya memang teks. `JSONDATA` adalah dokumen JSON yang ditulis
//     `@ASM.GetPageJSONString()` dari properti klipboard Pega, dan seluruh properti pada
//     `Section/CreateMasterSupplier_Sec` dirender sebagai `pxTextInput`, `pxDropdown`,
//     atau `pxTextArea` — tidak satu pun bertipe angka maupun tanggal.
//  2. Dokumennya DIBACA BERSAMA Pega selama masa paralel (`D-21`). Mengubah bentuk nilai
//     yang tersimpan berarti layar Pega membaca sesuatu yang berbeda dari yang ditulisnya
//     sendiri.
type Supplier struct {
	// ID adalah kolom ID — kunci baris ini.
	//
	// Ia TIDAK diketik pengguna: `Section/CreateMasterSupplier_Sec` memasang isiannya
	// `Read-only` dan menyembunyikannya sampai terisi (`pyVisible: NOTBLANK`), sehingga ia
	// hanya tampak saat menyunting. `Database/PEGA_M_SUPPLIER.prc:21` menerbitkannya
	// sebagai kode situs ditambah nomor urut **sebelas** digit; lihat IDSource.
	ID string

	// OldID adalah kolom OLDID.
	//
	// Ia KOLOM, bukan kunci di dalam JSON — `GetDataEditMasterSupller-SQL.xml:91` membaca
	// `A.OLDID`, bukan `A.JSONDATA.OLDID`. Modul ini membacanya dan tidak pernah
	// menulisinya: tidak ada satu pun rule di export yang mengisinya, sehingga menuliskan
	// nilai apa pun ke sana berarti menebak apa gunanya.
	OldID string

	// Name adalah kunci NAMA. Ia kunci alami layar: setelah baris tersimpan, isiannya
	// menjadi read-only (`pyReadOnlyCondition: MasterSupplier.ID != ''`) sehingga nama
	// supplier TIDAK dapat diubah. Lihat ErrNameLocked.
	Name string

	Address    string // ALAMAT
	City       string // KOTA
	BranchName string // NAMA_CABANG
	PostalCode string // KODE_POS
	Country    string // NEGARA

	Phone string // TELEPON
	Fax   string // FAX
	Email string // EMAIL

	// TaxNumber adalah kunci NPWP.
	TaxNumber string

	// ContactPerson adalah kunci CONTACT_PERSON.
	ContactPerson string

	// PartnerStatus adalah kunci STS_REKANAN, berlabel "Status Rekanan" di layar.
	//
	// NILAI SAHNYA TIDAK DIKETAHUI. Isiannya `pxDropdown` bersumber `associated`, dan
	// daftar pilihannya ada di rule Field Value yang tidak ikut di export (`R-16`).
	// Diperlakukan sebagai teks apa adanya — bukan enum — sampai daftarnya diterima.
	PartnerStatus string

	// SupplyType adalah kunci JENIS_STATUS, berlabel "Status Supply" di layar.
	//
	// Ia dan HeavyEquipment adalah DUA WAJAH DARI SATU HAL; lihat SupplyTypeHeavyEquipment.
	SupplyType string

	// HeavyEquipment adalah kunci SUPPLIER_HE — turunan SupplyType, bukan isian.
	//
	// Ia TIDAK ada di form mana pun. Yang menuliskannya adalah
	// `CreateNewMasterSupplier_post` step 7, dan yang membacanya kembali adalah
	// `GetDataSupplier_pre` step 6.3 yang justru menurunkan SupplyType darinya. Lihat
	// DeriveHeavyEquipment.
	HeavyEquipment string

	// TermOfPayment dan TermOfDelivery adalah kunci TOP dan TOD — tenggat bayar dan
	// tenggat kirim, berlabel "Term of Payment" dan "Term of Delivery".
	//
	// SATUANNYA TIDAK DISEBUT di mana pun dalam export, dan isiannya `pxTextInput` bebas.
	// Karena itu keduanya tidak pernah diperiksa berbentuk angka di sini — menolak "30
	// hari" pada isian yang hari ini menerimanya adalah selisih yang tidak diminta siapa
	// pun.
	TermOfPayment  string
	TermOfDelivery string

	// Note adalah kunci KETERANGAN.
	//
	// Ia BUKAN hanya catatan: `InsertProteksiKlaimMBU_SQL` menyalinnya ke kolom
	// ALASAN_REQ pada baris permintaan persetujuan, sehingga isinya ikut terbaca pihak
	// yang memutuskan.
	Note string

	// Bank, AccountNumber, AccountName, dan BankBranch adalah kunci BANK, ACCOUNT_NO,
	// ACCOUNT_NAME, dan BANK_BRANCH.
	//
	// Perhatikan: BANK menyimpan NAMA bank, bukan kodenya. Tidak ada kunci BANK_ID di
	// seluruh dokumen — berbeda dari `BENGKEL_HE` yang punya kolom kodenya sendiri.
	Bank          string
	AccountNumber string
	AccountName   string
	BankBranch    string

	// SupplierType adalah kunci JENIS_SUPPLIER, berlabel "Jenis Supplier".
	//
	// Nilai sahnya tidak diketahui, alasannya sama dengan PartnerStatus.
	SupplierType string

	// ActiveRequested adalah kunci STS_AKTIF_PROMLIST, berlabel "Status Aktif" di layar.
	//
	// Ia yang DIISI pengguna. Pasangannya, Active, tidak pernah tampil di form.
	ActiveRequested string

	// Active adalah kunci STS_AKTIF — status aktif yang sebenarnya berlaku.
	//
	// Ia BUKAN isian melainkan akibat, dan kedua jalur memperlakukannya berbeda:
	//
	//	CreateNewMasterSupplier_post step 6   STS_AKTIF := "0"
	//	EditMasterSupplier_post step 7        STS_AKTIF := STS_AKTIF_PROMLIST
	//
	// Artinya supplier baru selalu lahir tidak aktif berapa pun yang dipilih di layar,
	// dan baru mengikuti pilihan itu saat disunting. Ditiru persis.
	Active string

	// AutoPayment adalah kunci STS_AUTOPAYMENT. Satu-satunya isian ber-`pxCheckbox` di
	// form, tetapi tersimpan sebagai teks seperti yang lain.
	AutoPayment string

	// UpdatedBy dan UpdatedAt adalah kunci USERKLAIMID dan TGL_INSERT.
	//
	// Keduanya ditulis `CreateNewMasterSupplier_post` step 6 dan `EditMasterSupplier_post`
	// step 7, dan KEDUANYA TIDAK PERNAH DIBACA KEMBALI — `GetDataEditMasterSupller` tidak
	// menyebut satu pun. Modul ini membacanya supaya jejaknya terlihat di layar; itu
	// tambahan terhadap Pega, dan tidak mengubah apa pun yang tersimpan.
	//
	// UpdatedAt berbentuk TEKS `dd/MM/yyyy` zona Asia/Jakarta, bukan waktu. Itu persis
	// utang yang `F-5` larang di kode baru, dan ia tetap ditulis begitu karena dokumennya
	// dibaca bersama Pega. Dicatat sebagai utang, bukan diperbaiki diam-diam.
	UpdatedBy string
	UpdatedAt string
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Dua puluh tiga isian, mengikuti `Section/CreateMasterSupplier_Sec-Section.xml`. Yang ADA
// di dokumen tersimpan tetapi TIDAK di sini, karena seluruhnya diturunkan sistem:
//
//	ID           diterbitkan saat penambahan; lihat IDSource
//	OLDID        kolom warisan yang tidak pernah ditulis rule mana pun
//	SUPPLIER_HE  diturunkan dari JENIS_STATUS
//	STS_AKTIF    "0" saat menambah, menyusul STS_AKTIF_PROMLIST saat menyimpan
//	USERKLAIMID  identitas pemanggil
//	TGL_INSERT   waktu penyimpanan
type Input struct {
	Name       string
	Address    string
	City       string
	BranchName string
	PostalCode string
	Country    string

	Phone string
	Fax   string
	Email string

	TaxNumber     string
	ContactPerson string

	PartnerStatus string
	SupplyType    string

	TermOfPayment  string
	TermOfDelivery string
	Note           string

	Bank          string
	AccountNumber string
	AccountName   string
	BankBranch    string

	SupplierType    string
	ActiveRequested string
	AutoPayment     string
}

// Panjang maksimum isian teks.
//
// SELURUHNYA ASUMSI YANG DISADARI, bukan angka yang diterima Work Owner maupun dibaca
// dari DDL: `M_SUPPLIER` tidak ada DDL-nya di export (`R-08`), dan sistem lama tidak
// memeriksa panjang satu pun isian.
//
// Di sini batasnya punya alasan tambahan yang tidak dimiliki modul lain: nilainya masuk ke
// dalam **satu kolom JSONDATA**, sehingga isian yang sangat panjang tidak ditolak basis
// data melainkan diterima diam-diam sampai dokumennya melampaui batas kolomnya sendiri —
// dan yang gagal saat itu adalah SELURUH baris, bukan isian yang kepanjangan.
//
// Angka yang sama diulang di `SupplierForm.tsx`. Bila berubah, KEDUA tempat harus ikut
// berubah — utang yang disadari dari menduplikasi sebuah angka, dijaga terlihat oleh uji
// di mastersupplier_test.go.
const (
	MaxNameLength    = 100
	MaxAddressLength = 250
	MaxCityLength    = 100
	MaxPostalLength  = 10
	MaxPhoneLength   = 30
	MaxEmailLength   = 100
	MaxTaxIDLength   = 30
	MaxCodeLength    = 20
	MaxShortLength   = 50
	MaxAccountLength = 30
	MaxNoteLength    = 500
)

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("mastersupplier: supplier tidak ditemukan")

	// ErrNameTaken: NAMA yang akan disisipkan sudah dipakai baris lain.
	//
	// # Pemeriksaan ini DITAMBAHKAN terhadap sistem lama
	//
	// Berbeda dari Master Bengkel yang punya `ValidationMasterBengkel`, tidak ada satu pun
	// rule di export yang memeriksa nama supplier ganda. Ia ditambahkan karena layar
	// sendiri memperlakukan nama sebagai kunci alami: sekali tersimpan, isiannya menjadi
	// read-only dan tidak dapat diperbaiki lagi (lihat ErrNameLocked).
	//
	// Gabungan keduanya berarti dua supplier bernama sama di sistem lama akan hidup
	// selamanya tanpa satu pun cara membedakannya dari layar. SELISIH YANG DIRENCANAKAN:
	// penambahan yang dulu diterima kini ditolak, dan baris lama dibaca apa adanya.
	ErrNameTaken = errors.New("mastersupplier: nama supplier sudah dipakai")

	// ErrNameLocked: nama supplier diubah pada baris yang sudah tersimpan.
	//
	// Asalnya `Section/CreateMasterSupplier_Sec-Section.xml`, yang memasang
	// `pyReadOnlyCondition: MasterSupplier.ID != ''` pada isian NAMA — satu-satunya isian
	// di form itu yang punya syarat read-only.
	//
	// Layar Pega menegakkannya dengan mengunci isiannya, dan hanya itu. Di sini server
	// ikut memeriksanya: penguncian di antarmuka adalah kenyamanan tampilan, dan
	// permintaan yang tidak datang dari layar itu tidak melewatinya sama sekali.
	ErrNameLocked = errors.New("mastersupplier: nama supplier tidak dapat diubah")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kunci JSON —
	// layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (`P-5`). Form ini punya lima belas isian wajib —
// terbanyak di antara seluruh modul master yang sudah dibangun — sehingga mengembalikan
// satu galat per percobaan berarti lima belas kali bolak-balik untuk satu form kosong.
type ValidationError struct {
	Violation []Violation
}

// OneViolation membungkus satu pelanggaran menjadi ValidationError.
//
// Dipakai lapisan aplikasi untuk pemeriksaan yang menuntut pembacaan basis data — keunikan
// nama — supaya galatnya sampai ke layar dalam bentuk yang SAMA dengan pelanggaran isian
// lain, dan menempel pada isiannya.
func OneViolation(field, message string) error {
	return &ValidationError{Violation: []Violation{{Field: field, Message: message}}}
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "mastersupplier: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
func (i Input) Clean() Input {
	trim := strings.TrimSpace
	return Input{
		Name:            trim(i.Name),
		Address:         trim(i.Address),
		City:            trim(i.City),
		BranchName:      trim(i.BranchName),
		PostalCode:      trim(i.PostalCode),
		Country:         trim(i.Country),
		Phone:           trim(i.Phone),
		Fax:             trim(i.Fax),
		Email:           trim(i.Email),
		TaxNumber:       trim(i.TaxNumber),
		ContactPerson:   trim(i.ContactPerson),
		PartnerStatus:   trim(i.PartnerStatus),
		SupplyType:      trim(i.SupplyType),
		TermOfPayment:   trim(i.TermOfPayment),
		TermOfDelivery:  trim(i.TermOfDelivery),
		Note:            trim(i.Note),
		Bank:            trim(i.Bank),
		AccountNumber:   trim(i.AccountNumber),
		AccountName:     trim(i.AccountName),
		BankBranch:      trim(i.BankBranch),
		SupplierType:    trim(i.SupplierType),
		ActiveRequested: trim(i.ActiveRequested),
		AutoPayment:     trim(i.AutoPayment),
	}
}

// requiredField adalah kelima belas isian yang layar Pega tandai `pyRequired=true`.
//
// Daftarnya dibaca langsung dari `Section/CreateMasterSupplier_Sec-Section.xml`, dan
// bukan ditentukan sendiri. Itu penting: mewajibkan lebih banyak akan menolak penambahan
// yang hari ini diterima, dan mewajibkan lebih sedikit akan meloloskan baris yang layar
// lamanya tolak — keduanya selisih yang tidak diminta siapa pun.
//
// Isian ber-`pyRequired=false` yang sengaja TIDAK diwajibkan: KODE_POS, FAX, EMAIL, NPWP,
// KETERANGAN, ACCOUNT_NAME, BANK_BRANCH, dan STS_AUTOPAYMENT.
func (i Input) requiredField() []struct {
	field, label, value string
	max                 int
} {
	return []struct {
		field, label, value string
		max                 int
	}{
		{"nama", "Nama", i.Name, MaxNameLength},
		{"alamat", "Alamat", i.Address, MaxAddressLength},
		{"kota", "Kota", i.City, MaxCityLength},
		{"nama_cabang", "Cabang", i.BranchName, MaxNameLength},
		{"negara", "Negara", i.Country, MaxCityLength},
		{"telepon", "Telepon", i.Phone, MaxPhoneLength},
		{"contact_person", "Contact Person", i.ContactPerson, MaxNameLength},
		{"status_rekanan", "Status Rekanan", i.PartnerStatus, MaxCodeLength},
		{"status_supply", "Status Supply", i.SupplyType, MaxCodeLength},
		{"term_of_payment", "Term of Payment", i.TermOfPayment, MaxShortLength},
		{"term_of_delivery", "Term of Delivery", i.TermOfDelivery, MaxShortLength},
		{"bank", "Bank", i.Bank, MaxNameLength},
		{"no_account", "No Account", i.AccountNumber, MaxAccountLength},
		{"jenis_supplier", "Jenis Supplier", i.SupplierType, MaxCodeLength},
		{"status_aktif", "Status Aktif", i.ActiveRequested, MaxCodeLength},
	}
}

// optionalField adalah isian yang boleh kosong; hanya panjangnya yang diperiksa.
func (i Input) optionalField() []struct {
	field, label, value string
	max                 int
} {
	return []struct {
		field, label, value string
		max                 int
	}{
		{"kode_pos", "Kode Pos", i.PostalCode, MaxPostalLength},
		{"fax", "Fax", i.Fax, MaxPhoneLength},
		{"email", "Email", i.Email, MaxEmailLength},
		{"npwp", "NPWP", i.TaxNumber, MaxTaxIDLength},
		{"keterangan", "Keterangan", i.Note, MaxNoteLength},
		{"account_name", "Account Name", i.AccountName, MaxNameLength},
		{"bank_branch", "Bank Branch", i.BankBranch, MaxNameLength},
		{"status_autopayment", "Status Autopayment", i.AutoPayment, MaxCodeLength},
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Yang TIDAK diperiksa di sini, dan kenapa
//
//	NAMA ganda      menuntut pembacaan basis data; dikerjakan lapisan aplikasi
//	NAMA berubah    menuntut baris yang tersimpan; dikerjakan lapisan aplikasi
//	TOP dan TOD     satuannya tidak disebut di mana pun; lihat Supplier.TermOfPayment
//	nilai dropdown  daftar nilai sahnya ada di rule Field Value yang tidak ikut di
//	                export (R-16). Memeriksanya berarti menebak, dan tebakan yang salah
//	                menolak supplier yang sah tanpa satu pun cara membetulkannya.
func (i Input) Check() error {
	var violation []Violation

	for _, r := range i.requiredField() {
		violation = append(violation, checkRequired(r.field, r.label, r.value, r.max)...)
	}
	for _, r := range i.optionalField() {
		violation = append(violation, checkLength(r.field, r.label, r.value, r.max)...)
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
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

// DeriveHeavyEquipment menurunkan SUPPLIER_HE dari JENIS_STATUS.
//
// Ia berada di paket domain, bukan di lapisan aplikasi, karena hubungan kedua nilai itu
// adalah ATURAN — bukan urutan langkah. Ia harus sama persis pada adapter SQL maupun
// adapter memori, dan satu uji yang menjaganya.
//
// Meniru `Activity/CreateNewMasterSupplier_post` step 7:
//
//	WHEN @equals(MasterSupplier.JENIS_STATUS,"1")  ->  MasterSupplier.SUPPLIER_HE := "1"
//
// # Satu cacat sistem lama yang TIDAK dibawa
//
// Pega hanya punya cabang "bila", tanpa cabang "selain itu": ketika JENIS_STATUS BUKAN
// "1", SUPPLIER_HE tidak pernah ditulis sama sekali dan tetap bernilai apa pun isinya
// sebelumnya. Akibatnya supplier HE yang diubah menjadi bukan-HE **tetap tersimpan
// sebagai HE**, dan saat dimuat kembali `GetDataSupplier_pre` mengembalikan JENIS_STATUS
// menjadi "1" — perubahannya hilang tanpa satu pun tanda di layar.
//
// Di sini kedua arah selalu ditulis. SELISIH YANG DIRENCANAKAN, dan satu-satunya yang
// terlihat pengguna: supplier yang diubah menjadi bukan-HE kini benar-benar berubah.
func DeriveHeavyEquipment(supplyType string) string {
	if strings.TrimSpace(supplyType) == SupplyTypeHeavyEquipment {
		return SupplyTypeHeavyEquipment
	}
	return SupplyTypeOther
}

// DeriveSupplyType menurunkan JENIS_STATUS kembali dari SUPPLIER_HE.
//
// Arah kebalikan DeriveHeavyEquipment, meniru `Activity/GetDataSupplier_pre` step 6.3.
//
// Ia dipakai saat MEMBACA, bukan saat menulis: dokumen lama dapat memuat JENIS_STATUS
// yang tidak sejalan dengan SUPPLIER_HE justru karena cacat yang dicatat di atas, dan
// yang menentukan perilaku sistem hilir adalah SUPPLIER_HE. Membacanya dari sanalah yang
// membuat layar menampilkan keadaan yang sebenarnya berlaku.
func DeriveSupplyType(heavyEquipment string) string {
	if strings.TrimSpace(heavyEquipment) == SupplyTypeHeavyEquipment {
		return SupplyTypeHeavyEquipment
	}
	return SupplyTypeOther
}

// Filter menyaring daftar yang dibaca layar.
//
// Sistem lama TIDAK punya penyaring apa pun di layar ini: `Section/InboxMasterSupplier`
// hanya punya tombol New Supplier, Edit, dan Refresh, dan gridnya membaca
// `ListMasterSupllier.pxResults` — sebuah page list yang **tidak ada satu pun rule di
// export yang mengisinya** (`R-16`).
//
// Keyword karena itu DITAMBAHKAN. Tanpa kueri lamanya, tidak ada yang dapat ditiru; yang
// dapat dilakukan adalah membuat daftar yang dapat dipersempit alih-alih daftar yang
// panjangnya tidak diketahui siapa pun.
type Filter struct {
	// Keyword mempersempit daftar pada nama, kota, atau contact person. Kosong berarti
	// tanpa penyaring.
	Keyword string
}

// IDSource menerbitkan ID supplier baru.
//
// Ia seam tersendiri, bukan method pada Repo, karena isinya bukan urusan master supplier
// melainkan urusan **penomoran**: kode situs dan pola yang sama dipakai lima belas
// procedure `PEGA_M_*` lain pada basis data yang sama.
//
// # Bentuknya, dibaca dari Database/PEGA_M_SUPPLIER.prc:12,21
//
//	SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
//	id_supplier_ins := id_site || lpad(to_Char(supplier_seq.nextval),11,'0');
//
// Ditiru persis, termasuk pembandingnya yang berupa TEKS '1' dan bukan angka.
//
// PERHATIKAN LEBARNYA: **sebelas** digit, bukan sepuluh seperti `PEGA_M_BENGKEL_HE.prc`.
// Menyalin lebar dari modul tetangga akan menerbitkan kunci yang berbeda bentuk dari
// seluruh baris yang sudah ada.
type IDSource interface {
	// NextID mengembalikan ID supplier berikutnya.
	NextID(ctx context.Context) (string, error)
}

// SequenceWidth adalah lebar nomor urut pada ID supplier.
//
// Ia di paket domain supaya adapter SQL dan adapter memori memakai angka yang SAMA tanpa
// saling mengimpor — memory tidak boleh bergantung pada sqlstore, yang menyeret driver
// basis data ke dalam uji yang justru dibuat agar tidak membutuhkannya.
const SequenceWidth = 11

// ComposeID merangkai ID supplier dari kode situs dan nomor urut.
//
// Ia berada di paket domain karena BENTUK KUNCI adalah aturan domain: ia yang menentukan
// bagaimana sebuah supplier dikenali, dan ia harus sama persis pada kedua adapter.
//
// Nomor urut yang LEBIH PANJANG dari lebar yang diminta tidak dipotong. `LPAD` Oracle
// memotongnya dari kanan, sehingga urutan yang melampaui sebelas digit akan menghasilkan
// kunci yang bertabrakan dengan urutan lain — diam-diam. Di sini ia dibiarkan tumbuh:
// kuncinya menjadi lebih panjang, dan itu terlihat, alih-alih salah tanpa terlihat.
func ComposeID(site string, sequence int64, width int) string {
	number := strconv.FormatInt(sequence, 10)
	if pad := width - len(number); pad > 0 {
		number = strings.Repeat("0", pad) + number
	}
	return strings.TrimSpace(site) + number
}

// Repo adalah seam ke penyimpanan master supplier SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat
// kueri (ADR-0030 Opsi 1).
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring.
	List(ctx context.Context, filter Filter) ([]Supplier, error)

	// Get mengembalikan satu baris; ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (Supplier, error)

	// FindByName mencari baris menurut NAMA-nya; ErrNotFound bila tidak ada.
	FindByName(ctx context.Context, name string) (Supplier, error)

	// Insert menyisipkan baris baru.
	//
	// Pemeriksaan nama ganda berada DI DALAM operasi repo, bukan dipecah menjadi "cek"
	// lalu "sisip" di lapisan aplikasi — jarak di antara keduanya adalah lubang balapan
	// yang tidak dijaga apa pun.
	//
	// KETERBATASAN YANG DISADARI. Tanpa constraint unik pada nama, lubang itu hanya
	// dipersempit, tidak ditutup. Penutupnya adalah constraint di basis data, dan itu
	// menunggu DDL (`R-08`) beserta prosedur perubahan skema (`D-63`). Di sini bahkan
	// lebih sempit lagi kemungkinannya: namanya tersimpan di dalam dokumen JSON, sehingga
	// constraint unik atasnya menuntut index berbasis fungsi lebih dulu.
	Insert(ctx context.Context, s Supplier) error

	// Update menyimpan perubahan pada baris yang sudah ada; ErrNotFound bila barisnya
	// hilang di antara pemuatan layar dan penyimpanan.
	//
	// # Kunci yang TIDAK dikenal modul ini ikut dipertahankan
	//
	// `Database/PEGA_M_SUPPLIER.prc:36` mengganti SELURUH dokumen
	// (`SET JSONDATA = DataPega`), dan yang dikirim Pega hanyalah kunci yang ada di
	// halaman klipboardnya — sehingga kunci yang tidak dibaca `GetDataEditMasterSupller`
	// LENYAP pada setiap penyimpanan.
	//
	// Adapter SQL modul ini menggabungkan, bukan menimpa: dokumen tersimpan dibaca lebih
	// dulu, kunci yang dikenal ditulis ulang, dan sisanya dibiarkan. Perlakuannya sama
	// dengan DOKUMENID pada Master Bengkel — jalur yang menghapus datanya sendiri tidak
	// ikut dibawa.
	Update(ctx context.Context, s Supplier) error
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

// Store menyatukan keempat seam yang dipakai layanan modul ini.
//
// Keempatnya tetap DIDEKLARASIKAN terpisah — Repo untuk tabel master, LookupRepo untuk
// empat tabel acuan yang hanya dibaca, ApprovalRepo untuk antrean persetujuan, IDSource
// untuk penomoran — karena keempatnya menjawab pertanyaan yang berbeda dan dapat berubah
// sendiri-sendiri. Yang disatukan hanyalah CARA MEMILIHNYA: keempatnya selalu berasal dari
// koneksi entitas yang sama, sehingga empat pemilih terpisah hanya akan membuka
// kemungkinan keempatnya menunjuk entitas yang berbeda.
type Store interface {
	Repo
	LookupRepo
	ApprovalRepo
	IDSource
}
