package registrasi

import (
	"context"
	"strings"
)

// InsuredProfile adalah data tertanggung dari CIF polis (`Policy.CIFData` dokumen POLICYDATA),
// yang ditampilkan blok Alamat · Telephone dan Email pada tab Register
// (`Section/InputAddress_PNC_Klaim`, hanya `IsPA` di `InputRegisterDetail`).
//
// Hanya dibaca: Pega menyalinnya ke `pyWorkPage.AddressList` (`GetDataTertartanggungFromASMTelfFax`)
// untuk ditampilkan, dan tidak ada kolom T_CLAIM_PNC untuknya.
type InsuredProfile struct {
	// IDCard adalah No KTP bawaan: `Customer_C.ASMIDCard`, atau `Customer_P.ASMIDCard` bila
	// kosong — urutan langkah 12 activity itu.
	IDCard string

	Addresses []InsuredAddress
}

// InsuredAddress adalah satu baris `CIFData.AddressList`.
type InsuredAddress struct {
	Type         string
	TypeName     string
	Address      string
	City         string
	CityName     string
	District     string
	DistrictName string
	RW           string
	RWName       string
	ZipCode      string
	Phones       []InsuredPhone
}

// InsuredPhone adalah satu baris `ASMTelfax` — grid "Telephone dan Email".
type InsuredPhone struct {
	Type      string
	TypeName  string
	Code      string
	Number    string
	Extension string
}

// InsuredProfileSource membaca profil tertanggung sebuah polis. Ia dipenuhi penyimpanan polis
// sebagai kemampuan tambahan (lihat Service.InsuredProfile); penyimpanan yang tidak memilikinya
// menghasilkan profil kosong.
type InsuredProfileSource interface {
	InsuredProfile(ctx context.Context, policyNumber, prodKe string) (InsuredProfile, error)
}

// AddressTypeName mengikuti langkah "set data pynote berdasarkan id" pada
// `Activity/GetDataTertartanggungFromASMTelfFax-Act.xml`.
func AddressTypeName(code string) string {
	switch strings.TrimSpace(code) {
	case "1":
		return "Alamat Rumah"
	case "2":
		return "Alamat Kantor"
	case "3":
		return "Alamat Polis"
	case "4":
		return "Alamat Penagihan"
	case "5":
		return "Alamat Kirim Polis"
	case "6":
		return "Alamat Agen"
	case "7":
		return "Alamat Email"
	case "8":
		return "Alamat Korespondensi"
	}
	return ""
}

// PhoneTypeName mengikuti langkah "set pynote berdasarkan type telephone" pada activity yang sama:
// TelfaxType 6 atau nomor yang memuat "@" adalah Email.
func PhoneTypeName(kind, number string) string {
	if strings.TrimSpace(kind) == "6" || strings.Contains(number, "@") {
		return "Email"
	}
	switch strings.TrimSpace(kind) {
	case "1":
		return "Telepon Biasa"
	case "2":
		return "FAX"
	case "3":
		return "Mobile Phone"
	case "4":
		return "Hand Phone"
	case "5":
		return "Telp Kantor"
	}
	return ""
}
