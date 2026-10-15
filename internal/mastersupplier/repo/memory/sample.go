package memory

import "claim-pnc/internal/mastersupplier"

// Isi contoh untuk pengembangan tanpa Oracle.
//
// SELURUHNYA KARANGAN. Tidak ada satu pun nama supplier, alamat, NPWP, nomor rekening,
// maupun alamat surel yang disalin dari data nyata — `D-69` melarang data nasabah dan
// alamat surel ditulis di berkas yang di-commit, dan supplier adalah pihak ketiga yang
// datanya diperlakukan sama.
//
// Yang ditiru dari data nyata hanyalah BENTUKNYA: panjang kode, bentuk ID, dan sebaran
// status — supaya layar yang dicoba saat pengembangan berperilaku seperti layar yang
// dipakai di produksi.

// SampleSite adalah kode situs contoh.
//
// Di produksi ia dibaca dari `POOLDATA.M_SITE_DATABASE` dan panjangnya tidak diketahui
// (R-08). Dua digit dipilih supaya ID yang dihasilkan menjadi tiga belas karakter —
// sebelas digit nomor urut ditambah dua digit situs, persis bentuk
// `PEGA_M_SUPPLIER.prc:21`.
const SampleSite = "01"

// SampleSequence adalah nomor urut terakhir yang dianggap sudah dipakai.
//
// Bukan nol, supaya ID yang diterbitkan saat pengembangan tidak tampak seperti nomor urut
// pertama dan tidak bertabrakan dengan baris contoh di bawah.
const SampleSequence int64 = 3

// SampleList adalah supplier contoh.
//
// Ketiganya sengaja berbeda pada hal yang paling menentukan perilaku layar:
//
//	baris 1  aktif, rekanan, supplier Heavy Equipment
//	baris 2  aktif, bukan HE — memperlihatkan pasangan JENIS_STATUS/SUPPLIER_HE yang "0"
//	baris 3  TIDAK aktif — satu-satunya jalur yang menyimpan tanpa meminta persetujuan
//
// Baris ketiga yang paling mudah terlupa diuji, dan justru ia yang mengubah keadaan tanpa
// melewati antrean siapa pun.
func SampleList() []mastersupplier.Supplier {
	return []mastersupplier.Supplier{
		{
			ID:              "0100000000001",
			Name:            "Supplier Contoh Utama",
			Address:         "Jalan Contoh Nomor 1",
			City:            "Jakarta Pusat",
			BranchName:      "Cabang Contoh Pusat",
			PostalCode:      "10110",
			Country:         "Indonesia",
			Phone:           "021-0000001",
			Fax:             "021-0000002",
			Email:           "kontak@contoh-supplier.example",
			TaxNumber:       "00.000.000.0-000.000",
			ContactPerson:   "Contoh Narahubung",
			PartnerStatus:   "1",
			SupplyType:      mastersupplier.SupplyTypeHeavyEquipment,
			HeavyEquipment:  mastersupplier.SupplyTypeHeavyEquipment,
			TermOfPayment:   "30",
			TermOfDelivery:  "14",
			Note:            "Kerja sama aktif.",
			Bank:            "Bank Contoh Satu",
			AccountNumber:   "1000000001",
			AccountName:     "Supplier Contoh Utama",
			BankBranch:      "KCP Contoh Pusat",
			SupplierType:    "1",
			ActiveRequested: mastersupplier.ActiveYes,
			Active:          mastersupplier.ActiveYes,
			AutoPayment:     mastersupplier.ActiveYes,
			UpdatedBy:       "CONTOH.PETUGAS",
			UpdatedAt:       "01/01/2026",
		},
		{
			ID:              "0100000000002",
			Name:            "Supplier Contoh Aneka",
			Address:         "Jalan Contoh Nomor 2",
			City:            "Bandung",
			BranchName:      "Cabang Contoh Bandung",
			PostalCode:      "40111",
			Country:         "Indonesia",
			Phone:           "022-0000001",
			Email:           "kontak@contoh-aneka.example",
			ContactPerson:   "Contoh Narahubung Dua",
			PartnerStatus:   "1",
			SupplyType:      mastersupplier.SupplyTypeOther,
			HeavyEquipment:  mastersupplier.SupplyTypeOther,
			TermOfPayment:   "45",
			TermOfDelivery:  "7",
			Bank:            "Bank Contoh Dua",
			AccountNumber:   "2000000002",
			BankBranch:      "KCP Contoh Bandung",
			SupplierType:    "2",
			ActiveRequested: mastersupplier.ActiveYes,
			Active:          mastersupplier.ActiveYes,
			AutoPayment:     mastersupplier.ActiveNo,
			UpdatedBy:       "CONTOH.PETUGAS",
			UpdatedAt:       "02/01/2026",
		},
		{
			ID:              "0100000000003",
			Name:            "Supplier Contoh Nonaktif",
			Address:         "Jalan Contoh Nomor 3",
			City:            "Surabaya",
			BranchName:      "Cabang Contoh Surabaya",
			Country:         "Indonesia",
			Phone:           "031-0000001",
			ContactPerson:   "Contoh Narahubung Tiga",
			PartnerStatus:   "0",
			SupplyType:      mastersupplier.SupplyTypeOther,
			HeavyEquipment:  mastersupplier.SupplyTypeOther,
			TermOfPayment:   "0",
			TermOfDelivery:  "0",
			Note:            "Kerja sama dihentikan sementara.",
			Bank:            "Bank Contoh Satu",
			AccountNumber:   "3000000003",
			SupplierType:    "2",
			ActiveRequested: mastersupplier.ActiveNo,
			Active:          mastersupplier.ActiveNo,
			AutoPayment:     mastersupplier.ActiveNo,
			UpdatedBy:       "CONTOH.PETUGAS",
			UpdatedAt:       "03/01/2026",
		},
	}
}

// SampleBranches adalah cabang contoh untuk dropdown Cabang.
func SampleBranches() []mastersupplier.Branch {
	return []mastersupplier.Branch{
		{ID: "001", Name: "Cabang Contoh Pusat"},
		{ID: "002", Name: "Cabang Contoh Bandung"},
		{ID: "003", Name: "Cabang Contoh Surabaya"},
	}
}

// SampleCities adalah kota contoh untuk lookup Kota.
//
// Kodenya meniru bentuk kode wilayah empat digit, sama seperti yang dipakai contoh Master
// Bengkel — supaya keduanya terlihat berasal dari tabel CITY yang sama.
func SampleCities() []mastersupplier.City {
	return []mastersupplier.City{
		{ID: "3171", Name: "Jakarta Pusat"},
		{ID: "3273", Name: "Bandung"},
		{ID: "3578", Name: "Surabaya"},
		{ID: "3374", Name: "Semarang"},
		{ID: "1271", Name: "Medan"},
	}
}

// SampleCountries adalah negara contoh untuk isian Negara.
func SampleCountries() []mastersupplier.Country {
	return []mastersupplier.Country{
		{ID: "ID", Name: "Indonesia"},
		{ID: "SG", Name: "Singapura"},
		{ID: "MY", Name: "Malaysia"},
		{ID: "TL", Name: "Timor-Leste"},
	}
}

// SampleBanks adalah bank contoh untuk dropdown Bank.
func SampleBanks() []mastersupplier.Bank {
	return []mastersupplier.Bank{
		{Code: "001", Name: "Bank Contoh Satu"},
		{Code: "002", Name: "Bank Contoh Dua"},
		{Code: "003", Name: "Bank Contoh Tiga"},
	}
}
