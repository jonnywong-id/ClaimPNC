package memory

import (
	"time"

	"claim-pnc/internal/mastersparepart"
)

// SampleSite adalah kode situs contoh.
//
// Bentuknya meniru `POOLDATA.M_SITE_DATABASE.ID` — kode pendek yang mendahului nomor urut
// pada setiap ID. Nilainya sengaja BUKAN "1" atau "01" supaya salah baca antara kode situs
// dan nomor urut langsung terlihat pada kunci yang diterbitkan.
const SampleSite = "SP"

// SampleSequence adalah nomor urut terakhir yang sudah dipakai contoh di bawah.
//
// Penambahan berikutnya karena itu menerbitkan SP0000000004 — melanjutkan deret, bukan
// menimpa baris yang sudah ada.
const SampleSequence int64 = 3

// sampleTime adalah stempel TGL_UPDATE_HARGA pada baris contoh.
//
// Tetap, bukan time.Now(): daftar contoh yang waktunya bergerak membuat cuplikan layar
// berubah setiap kali aplikasi dijalankan, dan uji yang membandingkannya menjadi rapuh.
func sampleTime(day int) *time.Time {
	at := time.Date(2026, time.September, day, 3, 15, 0, 0, time.UTC)
	return &at
}

// SampleCategories adalah daftar kategori contoh.
//
// Ia meniru `POOLDATA.GCNM_M_SPAREPART_CATEGORY` yang SUDAH DISETUJUI — hanya baris
// ber-APPROVAL '1' yang sampai ke layar; lihat catatan pada mastersparepart/lookup.go.
// Kategori yang masih menunggu karena itu tidak diwakili di sini sama sekali.
func SampleCategories() []mastersparepart.Category {
	return []mastersparepart.Category{
		{ID: "KAT01", Name: "ENGINE"},
		{ID: "KAT02", Name: "HYDRAULIC"},
		{ID: "KAT03", Name: "UNDERCARRIAGE"},
	}
}

// SampleTypes adalah daftar tipe contoh beserta kategori induknya.
//
// Sengaja tidak merata: ENGINE punya dua tipe, HYDRAULIC dua, UNDERCARRIAGE satu. Distribusi
// yang tidak merata membuat penyaringan Tipe menurut Kategori benar-benar terlihat bekerja
// saat layar dicoba — daftar yang setiap kategorinya berisi jumlah yang sama tidak
// membuktikan apa pun.
func SampleTypes() []mastersparepart.PartType {
	return []mastersparepart.PartType{
		{ID: "TIP01", Name: "FILTER", CategoryID: "KAT01"},
		{ID: "TIP02", Name: "PISTON", CategoryID: "KAT01"},
		{ID: "TIP03", Name: "SEAL KIT", CategoryID: "KAT02"},
		{ID: "TIP04", Name: "HOSE", CategoryID: "KAT02"},
		{ID: "TIP05", Name: "TRACK LINK", CategoryID: "KAT03"},
	}
}

// SampleList adalah daftar sparepart contoh untuk pengembangan tanpa Oracle.
//
// Ketiganya sengaja berbeda keadaan, supaya seluruh alur layar dapat dicoba:
//
//	SP0000000001  disetujui, SELURUH kolom terisi
//	SP0000000002  menunggu persetujuan, kolom pilihan KOSONG
//	SP0000000003  ditolak, dan TGL_UPDATE_HARGA-nya nil
//
// Yang kedua penting: kelima kolom yang daftar pilihannya tidak ada di export boleh kosong,
// dan layar harus tetap benar menggambarnya. Yang ketiga menutup keadaan `PriceUpdatedAt`
// nil, yang paling mudah terlupa diuji karena baris pertama selalu mengisinya.
func SampleList() []mastersparepart.Sparepart {
	return []mastersparepart.Sparepart{
		{
			ID:             "SP0000000001",
			Name:           "FILTER OLI MESIN",
			Number:         "1R-0716",
			Code:           "FLT-ENG-001",
			SellingPrice:   "1250000",
			CategoryID:     "KAT01",
			TypeID:         "TIP01",
			Weight:         "850",
			Length:         "12",
			Width:          "12",
			Height:         "18",
			MinStock:       "5",
			MaxStock:       "40",
			OrderQuantity:  "10",
			ProductionDate: "2025-11-04",
			Substitute:     "FILTER OLI MESIN ALT",
			Kind:           "ORIGINAL",
			Unit:           "PCS",
			ActiveStatus:   "1",
			PartStatus:     "READY",
			UpdatedBy:      "INTANHENNY",
			PriceUpdatedAt: sampleTime(12),
			DocumentID:     "DOC-0001",
			Status:         mastersparepart.StatusApproved,
		},
		{
			ID:            "SP0000000002",
			Name:          "SEAL KIT BOOM CYLINDER",
			Number:        "707-99-45600",
			Code:          "SKT-HYD-014",
			SellingPrice:  "4750000",
			CategoryID:    "KAT02",
			TypeID:        "TIP03",
			Weight:        "1200",
			MinStock:      "2",
			MaxStock:      "12",
			OrderQuantity: "4",
			// Kelima kolom penanda sengaja dibiarkan kosong; lihat catatan fungsi ini.
			UpdatedBy:      "INTANHENNY",
			PriceUpdatedAt: sampleTime(18),
			Status:         mastersparepart.StatusPending,
		},
		{
			ID:             "SP0000000003",
			Name:           "TRACK LINK ASSY",
			Number:         "20Y-32-00203",
			Code:           "TRK-UND-007",
			SellingPrice:   "0",
			CategoryID:     "KAT03",
			TypeID:         "TIP05",
			ProductionDate: "2024-07-19",
			Unit:           "SET",
			ActiveStatus:   "0",
			PartStatus:     "DISCONTINUED",
			UpdatedBy:      "INTANHENNY",
			// PriceUpdatedAt sengaja nil: harganya nol, dan baris ini mewakili keadaan yang
			// belum pernah distempel sama sekali.
			Status: mastersparepart.StatusRejected,
		},
	}
}
