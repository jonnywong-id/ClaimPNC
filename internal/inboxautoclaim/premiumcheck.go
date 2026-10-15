package inboxautoclaim

import (
	"errors"
	"strings"
)

// # Tab Cek Premi
//
// Tab keempat layar lama (`InboxAutoClaim-Harness.xml` :50640). Petugas memilih Nama
// Bisnis dan Sumber Bisnis, lalu tombol Cek Premi (`InboxAutoClaim/CekPremi-Act.xml`)
// menampilkan dua angka:
//
//   - Total Premi — `TotalPremiumPaid` dari layanan REST `GetPremiumPaid_SPK`
//     (`/getPaymentDataSumbis`, parameter SourceOfBizCode dan BizCode);
//   - Total Klaim — jumlah nilai klaim Kredit yang sudah Sukses Klaim untuk pasangan yang
//     sama (`GetTotalKlaimCreditValue_API`).
//
// Kolom ketiga layar lama, "Max Premi (%)" (`TempPremi.DistrictID`), TIDAK PERNAH diisi
// activity mana pun di export, sehingga di Pega selalu kosong. Ia tidak dibawa.

// Choice adalah satu pilihan isian: kode yang dikirim, nama yang tampil.
type Choice struct {
	Code string
	Name string
}

// PremiumCheckChoices adalah isi kedua isian tab Cek Premi.
type PremiumCheckChoices struct {
	// Business dari POOLDATA.BUSINESS (Report Definition `BrowseBusiness_RD`).
	Business []Choice

	// SourceOfBusiness dari Master Auto Claim yang disetujui (`BrowseAutoKlaim`,
	// approval = '1'): kodenya INISIALID, namanya NAMA_PENERIMA.
	SourceOfBusiness []Choice
}

// PremiumCheckQuery adalah pasangan yang diperiksa.
type PremiumCheckQuery struct {
	// BusinessCode dikirim sebagai BizCode dan dicocokkan ke T_GENERAL.BUSINESSCODE.
	BusinessCode string

	// SourceOfBusiness dikirim sebagai SourceOfBizCode dan dicocokkan ke
	// T_GENERAL.SOURCEOFBUSINESS.
	SourceOfBusiness string
}

// Clean merapikan spasi dan melaporkan isian yang kosong sekaligus.
func (q PremiumCheckQuery) Clean() (PremiumCheckQuery, error) {
	clean := PremiumCheckQuery{
		BusinessCode:     strings.TrimSpace(q.BusinessCode),
		SourceOfBusiness: strings.TrimSpace(q.SourceOfBusiness),
	}
	var violation []Violation
	if clean.BusinessCode == "" {
		violation = append(violation, Violation{Field: "kode_bisnis", Message: "Nama Bisnis wajib dipilih."})
	}
	if clean.SourceOfBusiness == "" {
		violation = append(violation, Violation{Field: "kode_sumber_bisnis", Message: "Sumber Bisnis wajib dipilih."})
	}
	if len(violation) > 0 {
		return clean, &ValidationError{Violation: violation}
	}
	return clean, nil
}

// PremiumCheckResult adalah kedua angka yang ditampilkan.
//
// Keduanya teks angka apa adanya (presisi penuh, I-12); pemformatan ribuan terjadi di
// layar, seperti `NumberAddSeparator` pada activity aslinya.
type PremiumCheckResult struct {
	PremiumPaid string
	ClaimTotal  string
}

// ErrPremiumServiceUnavailable: layanan total premi tidak dapat dihubungi atau jawabannya
// tidak terbaca. Dibedakan dari galat basis data supaya layar dapat menyebut sebabnya.
var ErrPremiumServiceUnavailable = errors.New("inboxautoclaim: layanan cek premi tidak dapat dihubungi")
