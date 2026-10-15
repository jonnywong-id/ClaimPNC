package monitoringslinkojk

import "strings"

// Muatan yang dikirim tombol "SLIK OJK".
//
// ============================================================================
// KONTRAKNYA TERBACA 2026-10-08 — dan ia BUKAN panggilan ke OJK
// ============================================================================
//
// Catatan lama di write.go menyatakan Connect-REST-nya "nol kemunculan di export".
// **Itu keliru.** Berkasnya ada, dan setelah Work Owner menarik rule sisi penerimanya,
// kontraknya terbaca utuh:
//
//	Connect REST/Rest_SendDataClientBasedDebitur-ConnectREST.xml   pemanggil
//	Service REST/ASMRequestServiceCreateClient-REST.xml            penerima
//	Activity/ASMRequestServiceCreateClient_Act -Act.xml            yang memprosesnya
//
// Tujuannya **aplikasi Pega lain** (`ASMFWInternalWork`), bukan OJK: ia mendaftarkan
// klien/CIF. Nama tombolnya menyesatkan dan dipertahankan apa adanya (`D-13`).
//
// Satu akibat yang perlu disadari: `ASMFWInternalWork` **tidak ikut bermigrasi** (`D-03`),
// sehingga ketergantungan ke Pega di jalur ini **tetap ada setelah Tahap 7** — berbeda dari
// seluruh jalur lain di modul ini.

// Nilai `CustomerType`, dibaca dari precondition `ASMRequestServiceCreateClient_Act`.
const (
	// CustomerPerson memilih cabang perorangan — `Customer_P` yang dibaca.
	CustomerPerson = "1"

	// CustomerCompany memilih cabang perusahaan — `Customer_C` yang dibaca.
	CustomerCompany = "2"
)

// Debtor adalah data debitur yang menyusun badan permintaan.
//
// # Bentuknya mengikuti halaman klipboard Pega, bukan selera kami
//
// Sistem lama mengirim halaman `RequestService.NBWorkPage` yang diserialisasi utuh. Nama
// isiannya karena itu dipertahankan apa adanya di adapter (lihat paket pegaslik) — ia
// **kontrak dengan sistem lain**, dan menamainya ulang hanya menambah satu terjemahan yang
// dapat salah di antara dua pihak.
//
// Di dalam domain, namanya tetap bahasa Inggris yang bermakna (`D-80`). Penerjemahannya
// terjadi satu kali, di adapter.
type Debtor struct {
	// CustomerType memilih cabang. Penerimanya membaca `Customer_P` pada CustomerPerson
	// dan `Customer_C` pada CustomerCompany — tidak keduanya.
	CustomerType string

	// TransactionID adalah kunci idempotensi, dan ini bagian terpenting dari kontraknya.
	//
	// Penerimanya memeriksanya lewat `CheckJSONServiceLogExist_SQL` SEBELUM membuat
	// klien. Pengiriman ulang dengan nilai yang sama karena itu **tidak menggandakan**
	// data di sistem tujuan — itulah yang membuat tombol ini aman ditekan dua kali.
	//
	// Diisi nomor urut pengiriman (Submission.ID), bukan angka acak: nomor itu sudah
	// tercatat di baris pengirimannya, sehingga satu nilai dapat ditelusuri dari kedua
	// sisi.
	TransactionID string

	// --- Cabang perorangan ---

	// FirstName WAJIB bila CustomerType = CustomerPerson.
	//
	// Penerimanya menolak dengan `ErrMsg = "Data Client tidak boleh kosong"` bila kosong
	// (`pyFirstName==""`). Lihat ErrEmptyClient.
	FirstName string
	LastName  string

	IDCardNo    string // ASMIDCard
	Gender      string // ASMGender
	DateOfBirth string // ASMDateOfBirth
	NPWP        string // ASMNPWP
	MotherName  string // ASMMotherName

	// --- Cabang perusahaan ---

	// CompanyName WAJIB bila CustomerType = CustomerCompany.
	//
	// Penerimanya menolak dengan `ErrMsg = "Data Perusahaan tidak boleh kosong"`
	// (`pyCompany==""`).
	CompanyName string
	CompanyID   string // ASMComID

	Addresses []DebtorAddress
}

// DebtorAddress adalah satu baris `AddressList`.
type DebtorAddress struct {
	Street     string // ASMAddress
	District   string // ASMDistrict
	City       string // ASMCity
	PostalCode string // ASMZipCode

	// Primary mengisi `IsPrimary`. Alamat pertama ditandai utama.
	Primary bool

	Phones []DebtorPhone
}

// DebtorPhone adalah satu baris `ASMTelfax`.
type DebtorPhone struct {
	Type     string // TelfaxType
	AreaCode string // TelFaxCode
	Number   string // TelfaxNumber
}

// RequiredFieldMissing memberitahu apakah penerimanya PASTI menolak muatan ini.
//
// # Kenapa diperiksa di sini, bukan dibiarkan ditolak penerimanya
//
// Karena penolakannya tetap meninggalkan baris pengiriman di tabel kita — sudah tercatat
// sebelum dikirim, mengikuti `InsertDataSlinkOJKIndividu`. Baris tanpa `id_transaction`
// berarti "tidak sampai", dan muatan yang sejak awal pasti ditolak bukan itu artinya.
//
// Yang diperiksa HANYA yang penerimanya sendiri periksa — satu field per cabang, dibaca
// dari precondition-nya. Tidak ditambah-tambahi: aturan yang tidak ada di penerimanya akan
// menolak muatan yang sebenarnya diterima.
func (d Debtor) RequiredFieldMissing() bool {
	switch d.CustomerType {
	case CustomerCompany:
		return strings.TrimSpace(d.CompanyName) == ""
	default:
		return strings.TrimSpace(d.FirstName) == ""
	}
}
