package memory

import "claim-pnc/internal/inboxxol"

// NewSampleRepo membentuk penyimpanan berisi contoh yang mencakup SELURUH jalur layar.
//
// # Seluruh isinya karangan
//
// Tidak satu pun nomor, nama, atau nilai di berkas ini berasal dari data produksi
// (`D-69`: data nasabah tidak pernah ditulis ke berkas yang di-commit). Yang ditiru
// adalah BENTUK dan KEADAANNYA, bukan isinya.
//
// # Enam keadaan yang sengaja dibuat berbeda
//
// Supaya setiap cabang layar dapat dicoba tanpa Oracle — termasuk empat yang paling mudah
// terlewat kalau contohnya "semua normal":
//
//  1. Perjanjian dengan dua group business dan klaim di dua tanggal → jalur utama.
//  2. Perjanjian yang salah satu group business-nya TIDAK punya nama di master → menguji
//     penggantian menjadi "TREATY INWARD" (`GET_GROUPBUSINESS_XOL`).
//  3. Perjanjian TANPA group business sama sekali → menguji jalur daftar kosong yang
//     BUKAN galat, yaitu master yang baru dibuat dan belum diisi.
//  4. Baris treaty inward yang kursnya TIDAK ditemukan → menguji RateMissing, pengganti
//     `RETURN 1` yang `D-49` butir 5 perbaiki.
//  5. Pemberitahuan yang sudah direvisi → menguji perakitan "nomor / revisi".
//  6. Pemberitahuan yang belum disetujui di kedua tipe → menguji antrean tab Komite.
func NewSampleRepo() *Repo {
	return NewRepo(
		WithMasters(SampleMasters()...),
		WithCauseOfLoss(SampleCauseOfLoss()...),
		WithSummaries("2024", SampleSummaries2024()...),
		WithSummaries("2023", SampleSummaries2023()...),
		WithBreakdown("12/03/2024", "BANJIR", SampleBreakdownBanjir()...),
		WithTreatyInward("12/03/2024", "BANJIR", SampleTreatyBanjir()...),
		WithBreakdown("28/07/2024", "KEBAKARAN", SampleBreakdownKebakaran()...),
		WithTreatyInward("28/07/2024", "KEBAKARAN", SampleTreatyKebakaran()...),
		WithAdvices(SampleAdvices()...),
	)
}

// SampleMasters adalah tiga perjanjian XOL contoh.
//
// Nilai kursnya dibuat berbeda supaya pembagian ke mata uang perjanjian benar-benar
// terlihat: kalau kursnya sama, kekeliruan membagi dua kali tidak akan terbaca.
func SampleMasters() []inboxxol.MasterXOL {
	return []inboxxol.MasterXOL{
		{
			ID:           "XOL-001",
			Name:         "XOL Property Treaty",
			Year:         "2024",
			ExchangeRate: 15500,
			Type:         "PROPERTY",
			// Sudah disetujui komite — tidak muncul di antrean tab Komite.
			CommitteeStatus: "1",
			CommitteeNote:   "Disetujui rapat komite.",
			PIC:             "PICTEKNIK01",
			PICEmail:        "pic.teknik01@contoh.invalid",
			PICNote:         "Pengajuan tahunan.",
			BusinessGroups: []inboxxol.BusinessGroup{
				{ID: "10", Name: "Fire"},
				{ID: "20", Name: "Marine Cargo"},
			},
		},
		{
			ID:              "XOL-002",
			Name:            "XOL Aneka Treaty",
			Year:            "2023",
			ExchangeRate:    15000,
			Type:            "ANEKA",
			CommitteeStatus: "1",
			PIC:             "PICTEKNIK02",
			PICEmail:        "pic.teknik02@contoh.invalid",
			BusinessGroups: []inboxxol.BusinessGroup{
				{ID: "30", Name: "Aneka"},
				// Nama kosong: di sistem lama GET_GROUPBUSINESS_XOL menggantinya dengan
				// "TREATY INWARD". Di sini penggantian itu terjadi di Go.
				{ID: "99", Name: ""},
			},
		},
		{
			ID:   "XOL-003",
			Name: "XOL Engineering Treaty",
			Year: "2025",
			// Kurs sudah diisi, group business belum. Perjanjian seperti ini NYATA —
			// ia baru dibuat — dan layarnya harus menampilkan daftar kosong, bukan galat.
			ExchangeRate:    16200,
			Type:            "ENGINEERING",
			CommitteeStatus: "0",
			PIC:             "PICTEKNIK01",
			PICEmail:        "pic.teknik01@contoh.invalid",
			PICNote:         "Menunggu persetujuan komite.",
			BusinessGroups:  nil,
		},
	}
}

// SampleCauseOfLoss adalah isi dropdown Penyebab Kerugian.
//
// Yang tersimpan di kolom CAUSEOFLOSS pada tabel klaim adalah DESKRIPSI-nya, bukan
// kodenya — karena itu deskripsi di sini sama persis dengan yang dipakai kunci contoh
// rincian di bawah.
func SampleCauseOfLoss() []inboxxol.CauseOfLoss {
	return []inboxxol.CauseOfLoss{
		{ID: "12001", Description: "BANJIR"},
		{ID: "12002", Description: "KECELAKAAN DIRI"},
		{ID: "12003", Description: "KEBAKARAN"},
		{ID: "12004", Description: "GEMPA BUMI"},
	}
}

// SampleSummaries2024 adalah akumulasi klaim perjanjian 2024, MASIH DALAM RUPIAH.
//
// Nilainya sengaja dibiarkan rupiah persis seperti yang dikembalikan basis data:
// pembagian dengan kurs adalah tugas usecase, dan mengisi contoh ini dengan nilai yang
// sudah terbagi akan menyembunyikan kekeliruan bila pembagiannya kelak terlewat.
func SampleSummaries2024() []inboxxol.ClaimSummary {
	return []inboxxol.ClaimSummary{
		{LossDate: "12/03/2024", CauseOfLoss: "BANJIR", OutstandingValue: 4_650_000_000, AcceptedValue: 3_100_000_000},
		{LossDate: "28/07/2024", CauseOfLoss: "KEBAKARAN", OutstandingValue: 9_300_000_000, AcceptedValue: 7_750_000_000},
	}
}

// SampleSummaries2023 adalah akumulasi klaim perjanjian 2023.
func SampleSummaries2023() []inboxxol.ClaimSummary {
	return []inboxxol.ClaimSummary{
		{LossDate: "05/11/2023", CauseOfLoss: "GEMPA BUMI", OutstandingValue: 1_500_000_000, AcceptedValue: 1_200_000_000},
	}
}

// SampleBreakdownBanjir adalah rincian klaim milik sendiri, masih dalam rupiah.
func SampleBreakdownBanjir() []inboxxol.BusinessBreakdown {
	return []inboxxol.BusinessBreakdown{
		{
			BusinessGroup:    "Fire",
			BusinessGroupID:  "10",
			ClaimCount:       12,
			OutstandingValue: 3_100_000_000,
			AcceptedValue:    2_170_000_000,
			Source:           inboxxol.SourceOwnBusiness,
		},
		{
			BusinessGroup:    "Marine Cargo",
			BusinessGroupID:  "20",
			ClaimCount:       4,
			OutstandingValue: 1_550_000_000,
			AcceptedValue:    930_000_000,
			Source:           inboxxol.SourceOwnBusiness,
		},
	}
}

// SampleTreatyBanjir adalah baris treaty inward yang kursnya LENGKAP.
//
// Nilainya SUDAH dalam mata uang perjanjian — baris treaty inward dikonversi di kuerinya
// sendiri dan tidak ikut dibagi kurs di usecase.
func SampleTreatyBanjir() []inboxxol.BusinessBreakdown {
	return []inboxxol.BusinessBreakdown{{
		BusinessGroup:    "Treaty Inward",
		ClaimCount:       3,
		OutstandingValue: 48_500,
		AcceptedValue:    31_200,
		Source:           inboxxol.SourceTreatyInward,
	}}
}

// SampleBreakdownKebakaran adalah rincian klaim milik sendiri untuk tanggal kedua.
func SampleBreakdownKebakaran() []inboxxol.BusinessBreakdown {
	return []inboxxol.BusinessBreakdown{{
		BusinessGroup:    "Fire",
		BusinessGroupID:  "10",
		ClaimCount:       27,
		OutstandingValue: 9_300_000_000,
		AcceptedValue:    7_750_000_000,
		Source:           inboxxol.SourceOwnBusiness,
	}}
}

// SampleTreatyKebakaran adalah baris treaty inward yang KURSNYA TIDAK DITEMUKAN.
//
// Inilah keadaan yang di sistem lama menghasilkan angka salah tanpa satu pun tanda:
// `GETCURRENCYSTANDARD` mengembalikan `1`, sehingga nilai valuta asing diperlakukan satu
// banding satu terhadap rupiah. Di sini nilainya nol dan barisnya ditandai, sehingga
// layar menyatakan kursnya tidak tersedia alih-alih menampilkan angka yang salah.
func SampleTreatyKebakaran() []inboxxol.BusinessBreakdown {
	return []inboxxol.BusinessBreakdown{{
		BusinessGroup: "Treaty Inward",
		ClaimCount:    2,
		Source:        inboxxol.SourceTreatyInward,
		RateMissing:   true,
	}}
}

// SampleAdvices adalah pemberitahuan PLA dan DLA contoh.
//
// Dua di antaranya belum disetujui (`ApprovalStatus` `"0"`) pada tipe yang berbeda,
// sehingga antrean tab Komite berisi dua baris — satu PLA, satu DLA.
func SampleAdvices() []inboxxol.Advice {
	return []inboxxol.Advice{
		{
			Number:         "PLA/XOL/2024/0001",
			Revision:       "0",
			ReinsurerID:    "R-001",
			ReinsurerName:  "Reasuransi Contoh Pertama",
			LayerID:        "L1",
			LayerName:      "Layer 1",
			Year:           "2024",
			CauseOfLoss:    "BANJIR",
			ExchangeRate:   15500,
			SharePercent:   35.5,
			Limit:          "10.000.000.000",
			ApprovalStatus: "0",
			MasterID:       "XOL-001",
			InputBy:        "PICTEKNIK01",
			InputByEmail:   "pic.teknik01@contoh.invalid",
			Remark:         "Menunggu konfirmasi reasuradur.",
			Email:          "reas.pertama@contoh.invalid",
			Country:        "SINGAPORE",
			Type:           inboxxol.AdvicePLA,
			IssuedOn:       "15/03/2024",
		},
		{
			// Sudah direvisi sekali — nomor yang dibaca pengguna menjadi
			// "PLA/XOL/2024/0002 / 1". Perakitannya terjadi di Go, bukan di SQL.
			Number:         "PLA/XOL/2024/0002",
			Revision:       "1",
			ReinsurerID:    "R-002",
			ReinsurerName:  "Reasuransi Contoh Kedua",
			LayerID:        "L2",
			LayerName:      "Layer 2",
			Year:           "2024",
			CauseOfLoss:    "BANJIR",
			ExchangeRate:   15500,
			SharePercent:   64.5,
			Limit:          "25.000.000.000",
			ApprovalStatus: "1",
			MasterID:       "XOL-001",
			InputBy:        "PICTEKNIK01",
			InputByEmail:   "pic.teknik01@contoh.invalid",
			ApprovalNote:   "Disetujui komite.",
			// Surel kosong pada baris pemberitahuan; di SQL ia jatuh ke
			// T_REINSURER.EMAIL lewat COALESCE. Di contoh ini hasil jatuhnya sudah
			// terisi, karena penyimpanan memori tidak punya tabel reasuradur.
			Email:    "reas.kedua@contoh.invalid",
			Country:  "MALAYSIA",
			Type:     inboxxol.AdvicePLA,
			IssuedOn: "18/03/2024",
		},
		{
			Number:         "DLA/XOL/2024/0001",
			Revision:       "0",
			ReinsurerID:    "R-001",
			ReinsurerName:  "Reasuransi Contoh Pertama",
			LayerID:        "L1",
			LayerName:      "Layer 1",
			Year:           "2024",
			CauseOfLoss:    "KEBAKARAN",
			ExchangeRate:   15500,
			SharePercent:   35.5,
			Limit:          "10.000.000.000",
			ApprovalStatus: "0",
			MasterID:       "XOL-001",
			InputBy:        "PICTEKNIK02",
			InputByEmail:   "pic.teknik02@contoh.invalid",
			PICNote:        "Nilai akseptasi final.",
			Email:          "reas.pertama@contoh.invalid",
			Country:        "SINGAPORE",
			Type:           inboxxol.AdviceDLA,
			IssuedOn:       "02/08/2024",
		},
	}
}
