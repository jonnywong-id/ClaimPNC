package memory

import (
	"time"

	"claim-pnc/internal/inboxclaimtreatyprop"
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
// Yang TIDAK dikarang adalah BENTUKNYA: susunan kunci objek kerja, penanda `CLMP`, nama
// akun antrean teknik, dan pembagian worklist/workbasket seluruhnya mengikuti export.
// Itulah yang membuat penyaring di memory.go benar-benar teruji.
//
// # Apa yang sengaja diuji oleh susunan baris di bawah
//
//	tiga baris di worklist              tab Admin harus memberi ketiganya, tanpa
//	                                    memandang siapa petugasnya
//	tiga baris di workbasket            tab Teknik harus memberi ketiganya, tab lain nol
//	satu baris berkelas objek kerja     harus tersaring di SELURUH tab
//	  yang BUKAN klaim treaty
//
// Baris `CLMP-3001` sengaja berada di antrean bernama LAIN, bukan `TreatyinPNCTeknik`. Dulu
// ia harus tersaring; sekarang ia harus MUNCUL — Report Definition antrean teknik tidak
// menyaring menurut nama antrean. Ia satu-satunya baris contoh yang perannya berbalik saat
// sumber data berpindah, dan itu ditulis di sini supaya pembalikannya disengaja.
//
// Satu baris (CLMP-2002) sengaja mengosongkan "Last update" dan "Status Claim ID". Keduanya
// dibaca lewat LEFT JOIN ke tabel objek kerja, sehingga baris yang objek kerjanya tidak
// terbaca memang mengembalikan keduanya NULL — dan layar harus menggambarnya sebagai tanda
// pisah, bukan sebagai sel yang tampak rusak.
func SampleRows() []Row {
	at := func(day int) time.Time {
		return time.Date(2026, time.September, day, 3, 0, 0, 0, time.UTC)
	}

	return []Row{
		{
			AssignedAt: at(18),
			WorkClass:  inboxclaimtreatyprop.WorkClass,
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:            "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1001",
				Reference:          "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1001!FLOW",
				ClaimID:            "CLMP-1001",
				AssignedOperator:   "ADMINTREATY1",
				MasterID:           "TRP-2026-01",
				PolicyNumber:       "99.001.2026.00001",
				LossDate:           "2026-08-14",
				BusinessName:       "Property All Risk",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Pertama",
				InsuredName:        "PT Contoh Sejahtera",
				LastUpdateOperator: "ADMINTREATY1",
				ClaimStatus:        "New",
			},
		},
		{
			AssignedAt: at(20),
			WorkClass:  inboxclaimtreatyprop.WorkClass,
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:            "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1002",
				Reference:          "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1002!FLOW",
				ClaimID:            "CLMP-1002",
				AssignedOperator:   "ADMINTREATY1",
				MasterID:           "TRP-2026-02",
				PolicyNumber:       "99.001.2026.00002",
				LossDate:           "2026-09-02",
				BusinessName:       "Marine Cargo",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Kedua",
				InsuredName:        "PT Contoh Bahari",
				LastUpdateOperator: "PICTREATY1",
				ClaimStatus:        "Pending-Teknik",
			},
		},
		{
			AssignedAt: at(19),
			WorkClass:  inboxclaimtreatyprop.WorkClass,
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:            "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1003",
				Reference:          "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1003!FLOW",
				ClaimID:            "CLMP-1003",
				AssignedOperator:   "ADMINTREATY2",
				MasterID:           "TRP-2026-03",
				PolicyNumber:       "99.001.2026.00003",
				LossDate:           "2026-08-28",
				BusinessName:       "Engineering",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Ketiga",
				InsuredName:        "PT Contoh Konstruksi",
				LastUpdateOperator: "ADMINTREATY2",
				ClaimStatus:        "New",
			},
		},

		// Dua baris antrean teknik. Keduanya TIDAK membawa Subjectivity: kedua kueri
		// mengirimkannya NULL karena nama kolom tereksposnya belum diketahui, dan
		// penyimpanan ini meniru kueri — bukan meniru Report Definition yang memilihnya.
		{
			AssignedAt:     at(21),
			WorkClass:      inboxclaimtreatyprop.WorkClass,
			FromWorkbasket: true,
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:            "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-2001",
				Reference:          "ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-2001!FLOW",
				ClaimID:            "CLMP-2001",
				AssignedOperator:   inboxclaimtreatyprop.TechnicalWorkbasket,
				MasterID:           "TRP-2026-04",
				PolicyNumber:       "99.001.2026.00004",
				LossDate:           "2026-09-09",
				BusinessName:       "Property All Risk",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Keempat",
				InsuredName:        "PT Contoh Industri",
				LastUpdateOperator: "PICTREATY2",
				ClaimStatus:        "Pending-Teknik",
			},
		},
		{
			AssignedAt:     at(17),
			WorkClass:      inboxclaimtreatyprop.WorkClass,
			FromWorkbasket: true,
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:            "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-2002",
				Reference:          "ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-2002!FLOW",
				ClaimID:            "CLMP-2002",
				AssignedOperator:   inboxclaimtreatyprop.TechnicalWorkbasket,
				MasterID:           "TRP-2026-05",
				PolicyNumber:       "99.001.2026.00005",
				LossDate:           "2026-08-30",
				BusinessName:       "Marine Hull",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Kelima",
				InsuredName:        "PT Contoh Pelayaran",
				LastUpdateOperator: "",
				ClaimStatus:        "",
			},
		},

		// Baris worklist yang objek kerjanya berkelas LAIN.
		{
			AssignedAt: at(22),

			// Kelasnya BUKAN klaim treaty. Ia harus tersaring di SELURUH tab — inilah
			// yang membuktikan pembatas `PXOBJCLASS` benar-benar berjalan, bukan sekadar
			// tertulis.
			WorkClass: "ASM-FW-GCNMFW-Work-PNC",

			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:          "ASM-FW-GCNMFW-WORK-PNC PNC-9001",
				Reference:        "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK-PNC PNC-9001!FLOW",
				ClaimID:          "PNC-9001",
				AssignedOperator: "ADMINTREATY1",
				PolicyNumber:     "99.001.2026.09001",
				InsuredName:      "PT Contoh Bukan Treaty",
			},
		},

		// Baris workbasket milik antrean LAIN. Ia harus MUNCUL di tab Teknik —
		// membuktikan tidak ada lagi penyaring nama akun antrean, sesuai Report
		// Definition-nya.
		{
			AssignedAt:     at(22),
			WorkClass:      inboxclaimtreatyprop.WorkClass,
			FromWorkbasket: true,
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:            "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-3001",
				Reference:          "ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-3001!FLOW",
				ClaimID:            "CLMP-3001",
				AssignedOperator:   "AntreanLain",
				MasterID:           "TRP-2026-06",
				PolicyNumber:       "99.001.2026.00006",
				LossDate:           "2026-09-01",
				BusinessName:       "Engineering",
				BusinessSource:     "Treaty Inward",
				CedingCompany:      "Asuransi Contoh Keenam",
				InsuredName:        "PT Contoh Antrean Lain",
				LastUpdateOperator: "PICTREATY2",
				ClaimStatus:        "New",
			},
		},
	}
}
