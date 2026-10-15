package memory

import (
	"time"

	"claim-pnc/internal/inboxclaimtreatynonprop"
)

// SampleRows adalah antrean contoh untuk pengembangan lokal dan pengujian.
//
// # Seluruh isinya KARANGAN, dan itu disengaja
//
// Tidak satu pun nomor polis, nama tertanggung, atau nama Ceding Co di bawah berasal dari
// data nyata. `D-69` melarang data nasabah ditulis ke berkas yang di-commit, dan larangan
// itu berlaku penuh pada data contoh — berkas contoh justru yang paling mudah tersalin ke
// tempat lain.
//
// Yang TIDAK dikarang adalah BENTUKNYA: susunan kunci objek kerja, awalan `CLMNP-`, nama
// akun antrean teknik, dan pembagian worklist/workbasket seluruhnya mengikuti export.
// Itulah yang membuat penyaring di memory.go benar-benar teruji.
//
// # Apa yang sengaja diuji oleh susunan baris di bawah
//
//	dua baris milik `ADMINNONPROP1`     tab Admin tanpa "See All" harus memberi dua
//	satu baris milik `ADMINNONPROP2`    baris itu muncul HANYA dengan "See All"
//	satu baris TANPA nomor polis        hanya muncul saat "See TBA Claim" tercentang,
//	                                    dan ia MILIK `ADMINNONPROP1` supaya kombinasi
//	                                    "TBA tanpa See All" benar-benar ada isinya —
//	                                    kombinasi yang di sistem lama tidak terlayani
//	dua baris di workbasket teknik      tab Teknik harus memberi dua, tab lain nol
//	satu baris ber-awalan `CLMP-`       harus tersaring di SELURUH tab; ia membuktikan
//	                                    penyaring awalan tidak ikut menangkap klaim
//	                                    treaty PROPORSIONAL milik layar saudaranya
//	satu baris ber-awalan `KMTNP-`      harus tersaring pula; ia objek kerja komite,
//	                                    yang disebut fragmen GetWorkCNP_Act tetapi tidak
//	                                    pernah benar-benar dibaca kueri Admin
//	satu baris workbasket bukan teknik  harus tersaring di tab Teknik
func SampleRows() []Row {
	at := func(day int) time.Time {
		return time.Date(2026, time.September, day, 3, 0, 0, 0, time.UTC)
	}

	return []Row{
		{
			WorkCreatedAt: at(18),
			Item: inboxclaimtreatynonprop.WorkItem{
				Reference:          "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK CLMNP-1001!FLOW",
				ClaimID:            "CLMNP-1001",
				AssignedOperator:   "ADMINNONPROP1",
				MasterID:           "TNP-2026-01",
				JSONMasterID:       "TNP-2026-01",
				PolicyNumber:       "99.002.2026.00001",
				LossDate:           "2026-08-14",
				BusinessName:       "Property All Risk",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Pertama",
				InsuredName:        "PT Contoh Sejahtera",
				CreateOperator:     "ADMINNONPROP1",
				LastUpdateOperator: "ADMINNONPROP1",
			},
		},
		{
			WorkCreatedAt: at(20),
			Item: inboxclaimtreatynonprop.WorkItem{
				Reference:        "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK CLMNP-1002!FLOW",
				ClaimID:          "CLMNP-1002",
				AssignedOperator: "ADMINNONPROP1",
				MasterID:         "TNP-2026-02",
				// Kedua isian master id sengaja BERBEDA pada satu baris.
				//
				// Ia mewakili keadaan yang tidak dapat dikesampingkan: kolom objek kerja
				// dan nilai di dalam blob JSON adalah dua sumber yang berbeda, dan belum
				// ada yang pernah memeriksa apakah keduanya selalu sepakat (`R-08`).
				// Baris ini yang membuat layar ketahuan bila kelak seseorang menyatukan
				// keduanya diam-diam.
				JSONMasterID:       "TNP-2026-02-REV1",
				PolicyNumber:       "99.002.2026.00002",
				LossDate:           "2026-09-02",
				BusinessName:       "Marine Cargo",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Kedua",
				InsuredName:        "PT Contoh Bahari",
				CreateOperator:     "ADMINNONPROP1",
				LastUpdateOperator: "PICNONPROP1",
			},
		},
		{
			WorkCreatedAt: at(19),
			Item: inboxclaimtreatynonprop.WorkItem{
				Reference:          "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK CLMNP-1003!FLOW",
				ClaimID:            "CLMNP-1003",
				AssignedOperator:   "ADMINNONPROP2",
				MasterID:           "TNP-2026-03",
				JSONMasterID:       "TNP-2026-03",
				PolicyNumber:       "99.002.2026.00003",
				LossDate:           "2026-08-28",
				BusinessName:       "Engineering",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Ketiga",
				InsuredName:        "PT Contoh Konstruksi",
				CreateOperator:     "ADMINNONPROP2",
				LastUpdateOperator: "ADMINNONPROP2",
			},
		},

		// Baris TBA: nomor polisnya BELUM ADA.
		//
		// Ia milik `ADMINNONPROP1` dengan sengaja. Di sistem lama kombinasi "TBA tanpa
		// See All" tidak pernah menjalankan kueri apa pun, sehingga tidak ada data contoh
		// yang pernah membuktikannya; baris inilah yang membuat selisih terencana itu
		// benar-benar teruji, bukan hanya dinyatakan.
		{
			WorkCreatedAt: at(21),
			Item: inboxclaimtreatynonprop.WorkItem{
				Reference:        "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK CLMNP-1004!FLOW",
				ClaimID:          "CLMNP-1004",
				AssignedOperator: "ADMINNONPROP1",
				MasterID:         "TNP-2026-04",
				JSONMasterID:     "TNP-2026-04",
				// Kosong, bukan diisi tanda apa pun: di basis data ia NULL.
				PolicyNumber:       "",
				LossDate:           "2026-09-11",
				BusinessName:       "Contractors Plant & Machinery",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Keempat",
				InsuredName:        "PT Contoh Alat Berat",
				CreateOperator:     "ADMINNONPROP1",
				LastUpdateOperator: "ADMINNONPROP1",
			},
		},

		// Dua baris antrean teknik. Keduanya TIDAK membawa JSONMasterID, karena kueri
		// Teknik memang tidak mengambilnya.
		{
			WorkCreatedAt:  at(17),
			FromWorkbasket: true,
			Item: inboxclaimtreatynonprop.WorkItem{
				Reference:          "ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK CLMNP-2001!FLOW",
				ClaimID:            "CLMNP-2001",
				AssignedOperator:   inboxclaimtreatynonprop.TechnicalWorkbasket,
				MasterID:           "TNP-2026-05",
				PolicyNumber:       "99.002.2026.00005",
				LossDate:           "2026-08-30",
				BusinessName:       "Marine Hull",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Kelima",
				InsuredName:        "PT Contoh Pelayaran",
				CreateOperator:     "ADMINNONPROP2",
				LastUpdateOperator: "PICNONPROP2",
			},
		},
		{
			WorkCreatedAt:  at(22),
			FromWorkbasket: true,
			Item: inboxclaimtreatynonprop.WorkItem{
				Reference:          "ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK CLMNP-2002!FLOW",
				ClaimID:            "CLMNP-2002",
				AssignedOperator:   inboxclaimtreatynonprop.TechnicalWorkbasket,
				MasterID:           "TNP-2026-06",
				PolicyNumber:       "99.002.2026.00006",
				LossDate:           "2026-09-09",
				BusinessName:       "Property All Risk",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Keenam",
				InsuredName:        "PT Contoh Industri",
				CreateOperator:     "ADMINNONPROP1",
				LastUpdateOperator: "PICNONPROP2",
			},
		},

		// Klaim treaty PROPORSIONAL, milik layar saudaranya. Ia harus tersaring di seluruh
		// tab — inilah yang membuktikan penyaring awalan `CLMNP-` benar-benar berjalan dan
		// tidak ikut menangkap `CLMP-`.
		{
			WorkCreatedAt: at(23),
			Item: inboxclaimtreatynonprop.WorkItem{
				Reference:        "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1001!FLOW",
				ClaimID:          "CLMP-1001",
				AssignedOperator: "ADMINNONPROP1",
				PolicyNumber:     "99.001.2026.00001",
				InsuredName:      "PT Contoh Treaty Proporsional",
			},
		},

		// Objek kerja KOMITE non-proporsional. `GetWorkCNP_Act` menyebut awalan ini di
		// fragmen penyaringnya, tetapi ketiga kueri Admin yang benar-benar dijalankan
		// hanya menyaring `CLMNP-%`. Baris ini memastikan modul mengikuti kuerinya, bukan
		// fragmen yang tidak pernah terpakai.
		{
			WorkCreatedAt: at(23),
			Item: inboxclaimtreatynonprop.WorkItem{
				Reference:        "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK KMTNP-3001!FLOW",
				ClaimID:          "KMTNP-3001",
				AssignedOperator: "ADMINNONPROP1",
				PolicyNumber:     "99.002.2026.00007",
				InsuredName:      "PT Contoh Komite",
			},
		},

		// Baris workbasket milik antrean LAIN. Ia harus tersaring di tab Teknik —
		// membuktikan penyaring nama akun antrean berjalan, bukan sekadar "ada di
		// workbasket".
		{
			WorkCreatedAt:  at(22),
			FromWorkbasket: true,
			Item: inboxclaimtreatynonprop.WorkItem{
				Reference:        "ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK CLMNP-3002!FLOW",
				ClaimID:          "CLMNP-3002",
				AssignedOperator: "AntreanLain",
				PolicyNumber:     "99.002.2026.00008",
				InsuredName:      "PT Contoh Antrean Lain",
			},
		},
	}
}
