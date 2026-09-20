package memory

import "claim-pnc/internal/masterbengkel"

// Isi contoh untuk pengembangan tanpa Oracle.
//
// SELURUHNYA KARANGAN. Tidak ada satu pun nama bengkel, alamat, nomor rekening, NPWP,
// maupun alamat surel yang disalin dari data nyata — `D-69` melarang data nasabah dan
// alamat surel ditulis di berkas yang di-commit, dan bengkel adalah pihak ketiga yang
// datanya diperlakukan sama.
//
// Yang ditiru dari data nyata hanyalah BENTUKNYA: panjang kode, bentuk ID_BENGKEL, dan
// sebaran status — supaya layar yang dicoba saat pengembangan berperilaku seperti layar
// yang dipakai di produksi.

// SampleSite adalah kode situs contoh.
//
// Di produksi ia dibaca dari `POOLDATA.M_SITE_DATABASE` dan panjangnya tidak diketahui
// (R-08). Dua digit dipilih karena ID_BENGKEL yang dihasilkan menjadi dua belas karakter
// — cukup panjang untuk memperlihatkan bahwa kuncinya bukan angka berurut biasa.
const SampleSite = "01"

// SampleSequence adalah nomor urut terakhir yang dianggap sudah dipakai.
//
// Bukan nol, supaya ID yang diterbitkan saat pengembangan tidak tampak seperti nomor
// urut pertama dan tidak bertabrakan dengan baris contoh di bawah.
const SampleSequence int64 = 3

// SampleList adalah bengkel contoh, satu per status persetujuan.
//
// Ketiga tab layar karena itu terisi tanpa perlu menambah apa pun lebih dulu — termasuk
// tab Reject, yang paling mudah terlupa diuji.
func SampleList() []masterbengkel.Workshop {
	return []masterbengkel.Workshop{
		{
			ID:                  "010000000001",
			Name:                "Bengkel Contoh Utama",
			Address:             "Jalan Contoh Nomor 1",
			Phone:               "021-0000001",
			Mobile:              "0800-0000-001",
			Email:               "kontak@contoh-bengkel.example",
			WorkOrderEmail:      "wo@contoh-bengkel.example",
			BranchID:            "001",
			BranchName:          "Cabang Contoh Pusat",
			CityID:              "3171",
			CityName:            "Jakarta Pusat",
			PartnerStatus:       "1",
			WorkshopStatus:      "1",
			StatusReason:        "Kerja sama aktif.",
			StatusDate:          "01/01/2026",
			Login:               "bengkelcontoh1",
			BankID:              "002",
			BankName:            "Bank Contoh Satu",
			AccountNumber:       "1000000001",
			AccountName:         "Bengkel Contoh Utama",
			AccountID:           "ACC-0001",
			TaxName:             "Bengkel Contoh Utama",
			TaxNumber:           "00.000.000.0-000.001",
			TaxAddress:          "Jalan Contoh Nomor 1",
			IncomeTaxType:       "PPh 23",
			ValueAddedTax:       "11",
			ServiceDiscount:     "10",
			PartDiscount:        "5",
			MaterialPercent:     "100",
			PriceListGapPercent: "0",
			SLA:                 "3",
			SuppliedByASM:       "1",
			Supplier:            "",
			EClaimStatus:        "1",
			AutoAcceptStatus:    "1",
			PaymentStatus:       "1",
			AutoPaymentStatus:   "0",
			TeknoStatus:         "1",
			OrderStatus:         "1",
			DocumentID:          "",
			Status:              masterbengkel.StatusApproved,
		},
		{
			ID:                  "010000000002",
			Name:                "Bengkel Contoh Menunggu",
			Address:             "Jalan Contoh Nomor 2",
			Phone:               "022-0000002",
			Mobile:              "0800-0000-002",
			Email:               "kontak2@contoh-bengkel.example",
			WorkOrderEmail:      "wo2@contoh-bengkel.example",
			BranchID:            "002",
			BranchName:          "Cabang Contoh Bandung",
			CityID:              "3273",
			CityName:            "Bandung",
			PartnerStatus:       "1",
			WorkshopStatus:      "1",
			StatusReason:        "Pengajuan rekanan baru.",
			StatusDate:          "15/01/2026",
			Login:               "bengkelcontoh2",
			BankID:              "008",
			BankName:            "Bank Contoh Dua",
			AccountNumber:       "2000000002",
			AccountName:         "Bengkel Contoh Menunggu",
			AccountID:           "ACC-0002",
			TaxName:             "Bengkel Contoh Menunggu",
			TaxNumber:           "00.000.000.0-000.002",
			TaxAddress:          "Jalan Contoh Nomor 2",
			IncomeTaxType:       "PPh 23",
			ValueAddedTax:       "11",
			ServiceDiscount:     "7,5",
			PartDiscount:        "2,5",
			MaterialPercent:     "100",
			PriceListGapPercent: "0",
			SLA:                 "5",
			SuppliedByASM:       "0",
			EClaimStatus:        "0",
			AutoAcceptStatus:    "0",
			PaymentStatus:       "0",
			AutoPaymentStatus:   "0",
			TeknoStatus:         "0",
			OrderStatus:         "0",
			Status:              masterbengkel.StatusPending,
		},
		{
			// Non-rekanan, dan karena itu TANPA login aplikasi. Ia ada supaya jalur yang
			// paling mudah salah dapat dicoba tanpa basis data: bengkel non-rekanan tidak
			// wajib punya login, dan dua bengkel tanpa login tidak boleh saling menolak.
			ID:             "010000000003",
			Name:           "Bengkel Contoh Ditolak",
			Address:        "Jalan Contoh Nomor 3",
			Phone:          "031-0000003",
			Mobile:         "0800-0000-003",
			Email:          "kontak3@contoh-bengkel.example",
			WorkOrderEmail: "",
			BranchID:       "003",
			BranchName:     "Cabang Contoh Surabaya",
			CityID:         "3578",
			CityName:       "Surabaya",
			PartnerStatus:  masterbengkel.PartnerStatusNonPartner,
			WorkshopStatus: "0",
			StatusReason:   "Dokumen belum lengkap.",
			StatusDate:     "20/01/2026",
			Login:          "",
			BankID:         "",
			BankName:       "",
			AccountNumber:  "",
			AccountName:    "",
			AccountID:      "",
			TaxName:        "",
			TaxNumber:      "",
			TaxAddress:     "",
			IncomeTaxType:  "",
			ValueAddedTax:  "",
			SLA:            "",
			Status:         masterbengkel.StatusRejected,
		},
	}
}

// SampleBranches adalah cabang contoh untuk dropdown Cabang.
func SampleBranches() []masterbengkel.Branch {
	return []masterbengkel.Branch{
		{ID: "001", Name: "Cabang Contoh Pusat"},
		{ID: "002", Name: "Cabang Contoh Bandung"},
		{ID: "003", Name: "Cabang Contoh Surabaya"},
		{ID: "004", Name: "Cabang Contoh Medan"},
	}
}

// SampleCities adalah kota contoh untuk lookup Kota.
//
// Kodenya memakai bentuk kode wilayah empat digit, sama seperti yang tersimpan di kolom
// CITY_ID — supaya panjang isian yang terlihat saat pengembangan tidak menyesatkan.
func SampleCities() []masterbengkel.City {
	return []masterbengkel.City{
		{ID: "3171", Name: "Jakarta Pusat"},
		{ID: "3174", Name: "Jakarta Selatan"},
		{ID: "3273", Name: "Bandung"},
		{ID: "3578", Name: "Surabaya"},
		{ID: "1275", Name: "Medan"},
		{ID: "3374", Name: "Semarang"},
	}
}

// SampleBanks adalah bank contoh untuk dropdown Bank.
func SampleBanks() []masterbengkel.Bank {
	return []masterbengkel.Bank{
		{Code: "002", Name: "Bank Contoh Satu"},
		{Code: "008", Name: "Bank Contoh Dua"},
		{Code: "009", Name: "Bank Contoh Tiga"},
		{Code: "014", Name: "Bank Contoh Empat"},
	}
}
