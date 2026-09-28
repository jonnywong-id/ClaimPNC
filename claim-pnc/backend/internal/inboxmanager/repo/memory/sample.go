package memory

import (
	"claim-pnc/internal/inboxmanager"
)

// NewSampleStore membentuk penyimpanan berisi data contoh.
//
// Isinya dikarang dan TIDAK berasal dari basis data mana pun. Ia ada supaya layar dapat
// dikembangkan tanpa Oracle, bukan supaya perilaku dapat disimpulkan darinya.
//
// # Satu keadaan nyata ikut dicontohkan, dan itu disengaja
//
// Antrean Master Sparepart ditandai TIDAK DAPAT DIBACA, karena memang begitu keadaannya di
// basis data sejak 2026-09-28: `POOLDATA.SPAREPART_HE` adalah view berstatus INVALID. Tanpa
// mencontohkannya, jalur yang menangani keadaan itu tidak akan pernah terlihat saat
// pengembangan — dan jalur yang tidak pernah terlihat adalah jalur yang tidak pernah diuji
// mata.
func NewSampleStore() *Store {
	store := NewStore()

	// Lini bisnis bawaan supaya cabang penyaring dashboard terlewati saat pengembangan.
	store.SetDefaultLineBusiness(inboxmanager.LineNonMBU)

	store.SetPanel(inboxmanager.TabOutstanding, "pic", []inboxmanager.DashboardRow{
		countRow(inboxmanager.FieldPIC, "ALL", 42),
		countRow(inboxmanager.FieldPIC, "ANDIKA", 18),
		countRow(inboxmanager.FieldPIC, "BUDI", 14),
		countRow(inboxmanager.FieldPIC, "CITRA", 10),
	})
	store.SetPanel(inboxmanager.TabOutstanding, "grup_bisnis", []inboxmanager.DashboardRow{
		countRow(inboxmanager.FieldGrupBisnis, "ANEKA", 21),
		countRow(inboxmanager.FieldGrupBisnis, "FIRE", 13),
		countRow(inboxmanager.FieldGrupBisnis, "MARINE CARGO", 8),
	})

	store.SetPanel(inboxmanager.TabProduktivitas, "grup_bisnis", []inboxmanager.DashboardRow{
		comparisonRow("ANEKA", 21, 17, 9, 7, 3, 2, 9, 8),
		comparisonRow("FIRE", 13, 15, 6, 8, 2, 1, 5, 6),
	})
	store.SetPanel(inboxmanager.TabProduktivitas, "pic", []inboxmanager.DashboardRow{
		comparisonRow("ALL", 34, 32, 15, 15, 5, 3, 14, 14),
		comparisonRow("ANDIKA", 18, 16, 8, 7, 3, 2, 7, 7),
	})

	store.SetPanel(inboxmanager.TabKlaim, "bisnis", []inboxmanager.DashboardRow{
		claimRow("ANEKA", "", 21, 9, 3, 9, "1250000000", "300000000", "875000000"),
		claimRow("FIRE", "", 13, 6, 2, 5, "980000000", "150000000", "420000000"),
	})
	store.SetPanel(inboxmanager.TabKlaim, "penyebab", []inboxmanager.DashboardRow{
		claimRow("ANEKA", "KEBAKARAN", 12, 5, 2, 5, "700000000", "200000000", "500000000"),
		claimRow("ANEKA", "PENCURIAN", 9, 4, 1, 4, "550000000", "100000000", "375000000"),
	})

	store.SetQueue(inboxmanager.TabMasterBengkel, []inboxmanager.QueueRow{
		queueRow("BGK-001", map[string]string{
			inboxmanager.FieldID:         "BGK-001",
			inboxmanager.FieldNama:       "Bengkel Maju Jaya",
			inboxmanager.FieldCabang:     "JAKARTA",
			inboxmanager.FieldKota:       "JAKARTA SELATAN",
			inboxmanager.FieldKeterangan: "Jl. Contoh No. 1",
		}),
		queueRow("BGK-002", map[string]string{
			inboxmanager.FieldID:         "BGK-002",
			inboxmanager.FieldNama:       "Bengkel Sentosa",
			inboxmanager.FieldCabang:     "BANDUNG",
			inboxmanager.FieldKota:       "BANDUNG",
			inboxmanager.FieldKeterangan: "Jl. Contoh No. 2",
		}),
	})

	store.SetQueue(inboxmanager.TabMasterPanel, []inboxmanager.QueueRow{
		queueRow("PNL-001", map[string]string{
			inboxmanager.FieldID:   "PNL-001",
			inboxmanager.FieldNama: "Pintu Depan Kanan",
		}),
	})

	store.SetQueue(inboxmanager.TabNomorRangka, []inboxmanager.QueueRow{
		queueRow("PNC-1001|AVANZA|MHF001|MHF002", map[string]string{
			inboxmanager.FieldNoKlaim:       "PNC-1001",
			inboxmanager.FieldPengirim:      "Bengkel Maju Jaya",
			inboxmanager.FieldMerk:          "TOYOTA",
			inboxmanager.FieldModel:         "AVANZA",
			inboxmanager.FieldTipe:          "1.3 G",
			inboxmanager.FieldRangkaUser:    "MHF001",
			inboxmanager.FieldRangkaBengkel: "MHF002",
		}),
	})

	store.SetQueue(inboxmanager.TabKategoriSparepart, []inboxmanager.QueueRow{
		queueRow("KAT-01", map[string]string{
			inboxmanager.FieldID:   "KAT-01",
			inboxmanager.FieldNama: "Bodi",
		}),
	})

	store.SetQueue(inboxmanager.TabTipeSparepart, []inboxmanager.QueueRow{
		queueRow("TIP-01", map[string]string{
			inboxmanager.FieldID:       "TIP-01",
			inboxmanager.FieldNama:     "Panel Pintu",
			inboxmanager.FieldKategori: "Bodi",
		}),
	})

	store.SetQueue(inboxmanager.TabGroupingSparepart, []inboxmanager.QueueRow{
		queueRow("GRP-01", map[string]string{
			inboxmanager.FieldID:         "GRP-01",
			inboxmanager.FieldRangkaUser: "MHF001",
			inboxmanager.FieldNama:       "Pintu Depan Kanan",
			inboxmanager.FieldKeterangan: "SP-1001",
		}),
	})

	store.SetQueue(inboxmanager.TabPaymentAkseptasi, []inboxmanager.QueueRow{
		queueRow("AKS-2026-0001", map[string]string{
			inboxmanager.FieldNoKlaim:      "PNC-1001",
			inboxmanager.FieldNoAkseptasi:  "AKS-2026-0001",
			inboxmanager.FieldPIC:          "ANDIKA",
			inboxmanager.FieldTanggalInput: "12/09/2026",
		}),
	})

	store.SetQueue(inboxmanager.TabPenolakanKlaim, []inboxmanager.QueueRow{
		queueRow("PEN-01", map[string]string{
			inboxmanager.FieldID:             "PEN-01",
			inboxmanager.FieldStatusPenolak1: "Tidak dijamin polis",
			inboxmanager.FieldStatusPenolak2: "Pasal 4 ayat 2",
			inboxmanager.FieldPetugas:        "BUDI",
		}),
	})

	// Keadaan nyata per 2026-09-28 — lihat catatan di kepala fungsi ini.
	store.SetUnavailable(inboxmanager.TabMasterSparepart,
		"Sumbernya, view POOLDATA.SPAREPART_HE, sedang tidak dapat dibaca basis data.")

	return store
}

// countRow menyusun satu baris grid berbentuk (dimensi, jumlah).
func countRow(dimensionKey, dimension string, total int) inboxmanager.DashboardRow {
	return inboxmanager.DashboardRow{
		Cells: map[string]inboxmanager.DashboardCell{
			dimensionKey:                  {Text: dimension},
			inboxmanager.FieldJumlahKlaim: {Count: total},
		},
	}
}

// comparisonRow menyusun satu baris grid Produktivitas — delapan pencacah, empat keranjang
// dikali dua periode.
func comparisonRow(
	dimension string,
	totalNow, totalPrior, acceptedNow, acceptedPrior,
	rejectedNow, rejectedPrior, outstandingNow, outstandingPrior int,
) inboxmanager.DashboardRow {
	return inboxmanager.DashboardRow{
		Cells: map[string]inboxmanager.DashboardCell{
			inboxmanager.FieldDimensi:         {Text: dimension},
			inboxmanager.FieldTotalPeriodeIni: {Count: totalNow},
			inboxmanager.FieldTotalPeriodeLTY: {Count: totalPrior},
			inboxmanager.FieldAksepPeriodeIni: {Count: acceptedNow},
			inboxmanager.FieldAksepPeriodeLTY: {Count: acceptedPrior},
			inboxmanager.FieldTolakPeriodeIni: {Count: rejectedNow},
			inboxmanager.FieldTolakPeriodeLTY: {Count: rejectedPrior},
			inboxmanager.FieldOSPeriodeIni:    {Count: outstandingNow},
			inboxmanager.FieldOSPeriodeLTY:    {Count: outstandingPrior},
		},
	}
}

// claimRow menyusun satu baris grid Dashboard Klaim.
//
// Nilai uang diberikan sebagai TEKS presisi penuh, sama seperti yang dibaca sqlstore dari
// basis data — bukan sebagai angka pecahan.
func claimRow(
	dimension, cause string,
	total, accepted, rejected, outstanding int,
	acceptedAmount, rejectedAmount, outstandingAmount string,
) inboxmanager.DashboardRow {
	cells := map[string]inboxmanager.DashboardCell{
		inboxmanager.FieldNamaBisnisDK: {Text: dimension},
		inboxmanager.FieldTotalKlaim:   {Count: total},
		inboxmanager.FieldJumlahAksep:  {Count: accepted},
		inboxmanager.FieldJumlahTolak:  {Count: rejected},
		inboxmanager.FieldJumlahOS:     {Count: outstanding},
		inboxmanager.FieldNilaiAksep:   {Amount: acceptedAmount},
		inboxmanager.FieldNilaiTolak:   {Amount: rejectedAmount},
		inboxmanager.FieldNilaiOS:      {Amount: outstandingAmount},
	}
	if cause != "" {
		cells[inboxmanager.FieldPenyebab] = inboxmanager.DashboardCell{Text: cause}
	}
	return inboxmanager.DashboardRow{Cells: cells}
}

// queueRow menyusun satu baris antrean.
func queueRow(key string, cells map[string]string) inboxmanager.QueueRow {
	return inboxmanager.QueueRow{Key: key, Cells: cells}
}
