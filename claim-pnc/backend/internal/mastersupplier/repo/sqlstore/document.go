package sqlstore

import (
	"encoding/json"
	"fmt"
	"strings"

	"claim-pnc/internal/mastersupplier"
)

// # Bentuk dokumen JSON, dan dari mana setiap namanya berasal
//
// Kedua puluh delapan kunci di bawah dibaca langsung dari
// `RDB List/GetDataEditMasterSupller-SQL.xml:90-118` dan dari kedua activity penyimpan.
// Tidak ada satu pun yang dikarang:
//
//	25 kunci  dibaca kueri di atas satu per satu (`A.JSONDATA.NAMA`, …)
//	 2 kunci  USERKLAIMID dan TGL_INSERT — ditulis CreateNewMasterSupplier_post step 6
//	          dan EditMasterSupplier_post step 7, tidak pernah dibaca kembali
//	 1 kunci  ID — ditulis PEGA_M_SUPPLIER.prc:24 lewat penggantian teks 'UnknownID'
//
// Nama kuncinya PERSIS seperti di sana, termasuk besar-kecil hurufnya: jalur JSON bersifat
// case-sensitive, dan satu huruf yang berbeda membuat layar Pega membaca kosong tanpa satu
// pun galat.
//
// # Kunci yang tidak dikenal TIDAK dibuang
//
// Lihat mergeDocument. Ini yang membedakannya dari Pega, yang menimpa seluruh dokumen
// dengan apa pun yang kebetulan ada di halaman klipboardnya.
const (
	keyID   = "ID"
	keyName = "NAMA"

	keyAddress    = "ALAMAT"
	keyCity       = "KOTA"
	keyBranchName = "NAMA_CABANG"
	keyPostalCode = "KODE_POS"
	keyCountry    = "NEGARA"

	keyPhone = "TELEPON"
	keyFax   = "FAX"
	keyEmail = "EMAIL"

	keyTaxNumber     = "NPWP"
	keyContactPerson = "CONTACT_PERSON"

	keyPartnerStatus  = "STS_REKANAN"
	keySupplyType     = "JENIS_STATUS"
	keyHeavyEquipment = "SUPPLIER_HE"

	keyTermOfPayment  = "TOP"
	keyTermOfDelivery = "TOD"
	keyNote           = "KETERANGAN"

	keyBank          = "BANK"
	keyAccountNumber = "ACCOUNT_NO"
	keyAccountName   = "ACCOUNT_NAME"
	keyBankBranch    = "BANK_BRANCH"

	keySupplierType    = "JENIS_SUPPLIER"
	keyActiveRequested = "STS_AKTIF_PROMLIST"
	keyActive          = "STS_AKTIF"
	keyAutoPayment     = "STS_AUTOPAYMENT"

	keyUpdatedBy = "USERKLAIMID"
	keyUpdatedAt = "TGL_INSERT"
)

// documentValue memetakan satu baris supplier menjadi pasangan kunci dan nilai.
//
// Ia satu-satunya tempat yang mengetahui nama kunci, dan ia dipakai jalur penambahan,
// jalur penyimpanan, DAN penyalinan ke baris permintaan persetujuan — sehingga ketiganya
// tidak pernah dapat menghasilkan bentuk dokumen yang berbeda.
func documentValue(s mastersupplier.Supplier) map[string]string {
	return map[string]string{
		keyID:   s.ID,
		keyName: s.Name,

		keyAddress:    s.Address,
		keyCity:       s.City,
		keyBranchName: s.BranchName,
		keyPostalCode: s.PostalCode,
		keyCountry:    s.Country,

		keyPhone: s.Phone,
		keyFax:   s.Fax,
		keyEmail: s.Email,

		keyTaxNumber:     s.TaxNumber,
		keyContactPerson: s.ContactPerson,

		keyPartnerStatus:  s.PartnerStatus,
		keySupplyType:     s.SupplyType,
		keyHeavyEquipment: s.HeavyEquipment,

		keyTermOfPayment:  s.TermOfPayment,
		keyTermOfDelivery: s.TermOfDelivery,
		keyNote:           s.Note,

		keyBank:          s.Bank,
		keyAccountNumber: s.AccountNumber,
		keyAccountName:   s.AccountName,
		keyBankBranch:    s.BankBranch,

		keySupplierType:    s.SupplierType,
		keyActiveRequested: s.ActiveRequested,
		keyActive:          s.Active,
		keyAutoPayment:     s.AutoPayment,

		keyUpdatedBy: s.UpdatedBy,
		keyUpdatedAt: s.UpdatedAt,
	}
}

// buildDocument merangkai dokumen JSON baru dari sebuah baris supplier.
//
// Dipakai jalur penambahan dan penyalinan ke baris permintaan persetujuan — keduanya tidak
// punya dokumen sebelumnya untuk digabung.
func buildDocument(s mastersupplier.Supplier) (string, error) {
	return mergeDocument("", s)
}

// mergeDocument menulis ulang kunci yang dikenal DI ATAS dokumen yang sudah ada.
//
// # Kenapa menggabung, bukan menimpa
//
// `Database/PEGA_M_SUPPLIER.prc:36` mengganti seluruh dokumen (`SET JSONDATA = DataPega`),
// dan yang dikirim Pega hanyalah kunci yang kebetulan ada di halaman klipboardnya. Kunci
// apa pun yang tidak dibaca `GetDataEditMasterSupller` karena itu LENYAP pada setiap
// penyimpanan — termasuk kunci yang ditulis jalur lain, dan termasuk kunci yang belum
// diketahui siapa pun.
//
// Di sini dokumen tersimpan dibaca lebih dulu, kunci yang dikenal ditulis ulang, dan
// sisanya dibiarkan. Perlakuannya sama dengan DOKUMENID pada Master Bengkel: jalur yang
// menghapus datanya sendiri tidak ikut dibawa.
//
// # Dokumen lama yang tidak dapat diurai tidak menghentikan penyimpanan
//
// Baris yang JSONDATA-nya rusak — bukan JSON, atau bukan objek — TIDAK boleh membuat
// petugas terkunci tanpa cara memperbaikinya. Dalam keadaan itu dokumennya ditulis ulang
// dari nol, dan yang hilang hanyalah kunci yang memang sudah tidak terbaca siapa pun.
//
// Nilai yang kosong TETAP ditulis sebagai teks kosong, bukan dihilangkan dari dokumen:
// menghilangkannya membuat JSON_VALUE menjawab NULL, dan pembaca tidak punya cara
// membedakan "dikosongkan petugas" dari "kunci ini belum pernah ada".
func mergeDocument(existing string, s mastersupplier.Supplier) (string, error) {
	document := map[string]any{}
	if clean := strings.TrimSpace(existing); clean != "" {
		if err := json.Unmarshal([]byte(clean), &document); err != nil {
			// Sengaja tidak dikembalikan sebagai galat; lihat doc comment.
			document = map[string]any{}
		}
	}

	for key, value := range documentValue(s) {
		document[key] = value
	}

	encoded, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("mastersupplier/sqlstore: merangkai dokumen %q: %w", s.ID, err)
	}
	return string(encoded), nil
}
