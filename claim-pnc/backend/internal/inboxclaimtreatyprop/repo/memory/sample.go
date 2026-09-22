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
//	dua baris milik `ADMINTREATY1`      tab Work List tanpa "See All" harus memberi dua
//	satu baris milik `ADMINTREATY2`     baris itu muncul HANYA dengan "See All"
//	dua baris di workbasket teknik      tab Teknik harus memberi dua, tab lain nol
//	satu baris worklist tanpa `CLMP`    harus tersaring di SELURUH tab
//	satu baris workbasket bukan teknik  harus tersaring di tab Teknik
func SampleRows() []Row {
	at := func(day int) time.Time {
		return time.Date(2026, time.September, day, 3, 0, 0, 0, time.UTC)
	}

	return []Row{
		{
			AssignedAt: at(18),
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:          "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1001",
				Reference:        "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1001!FLOW",
				ClaimID:          "CLMP-1001",
				AssignedOperator: "ADMINTREATY1",
				MasterID:         "TRP-2026-01",
				PolicyNumber:     "99.001.2026.00001",
				LossDate:         "2026-08-14",
				BusinessName:     "Property All Risk",
				BusinessSource:   "Treaty Inward",
				CedingCompany:    "Asuransi Contoh Pertama",
				InsuredName:      "PT Contoh Sejahtera",
			},
		},
		{
			AssignedAt: at(20),
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:          "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1002",
				Reference:        "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1002!FLOW",
				ClaimID:          "CLMP-1002",
				AssignedOperator: "ADMINTREATY1",
				MasterID:         "TRP-2026-02",
				PolicyNumber:     "99.001.2026.00002",
				LossDate:         "2026-09-02",
				BusinessName:     "Marine Cargo",
				BusinessSource:   "Treaty Inward",
				CedingCompany:    "Asuransi Contoh Kedua",
				InsuredName:      "PT Contoh Bahari",
			},
		},
		{
			AssignedAt: at(19),
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:          "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1003",
				Reference:        "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1003!FLOW",
				ClaimID:          "CLMP-1003",
				AssignedOperator: "ADMINTREATY2",
				MasterID:         "TRP-2026-03",
				PolicyNumber:     "99.001.2026.00003",
				LossDate:         "2026-08-28",
				BusinessName:     "Engineering",
				BusinessSource:   "Treaty Inward",
				CedingCompany:    "Asuransi Contoh Ketiga",
				InsuredName:      "PT Contoh Konstruksi",
			},
		},

		// Dua baris antrean teknik. Keduanya membawa Subjectivity, karena hanya kueri
		// Teknik yang mengambilnya.
		{
			AssignedAt:     at(21),
			FromWorkbasket: true,
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:          "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-2001",
				Reference:        "ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-2001!FLOW",
				ClaimID:          "CLMP-2001",
				AssignedOperator: inboxclaimtreatyprop.TechnicalWorkbasket,
				MasterID:         "TRP-2026-04",
				PolicyNumber:     "99.001.2026.00004",
				LossDate:         "2026-09-09",
				BusinessName:     "Property All Risk",
				BusinessSource:   "Treaty Inward",
				CedingCompany:    "Asuransi Contoh Keempat",
				InsuredName:      "PT Contoh Industri",
				Subjectivity:     "1",
			},
		},
		{
			AssignedAt:     at(17),
			FromWorkbasket: true,
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:          "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-2002",
				Reference:        "ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-2002!FLOW",
				ClaimID:          "CLMP-2002",
				AssignedOperator: inboxclaimtreatyprop.TechnicalWorkbasket,
				MasterID:         "TRP-2026-05",
				PolicyNumber:     "99.001.2026.00005",
				LossDate:         "2026-08-30",
				BusinessName:     "Marine Hull",
				BusinessSource:   "Treaty Inward",
				CedingCompany:    "Asuransi Contoh Kelima",
				InsuredName:      "PT Contoh Pelayaran",
				Subjectivity:     "0",
			},
		},

		// Baris worklist yang kunci objek kerjanya TANPA penanda CLMP. Ia harus tersaring
		// di seluruh tab — inilah yang membuktikan penyaring `LIKE '%CLMP%'` benar-benar
		// berjalan, bukan sekadar tertulis.
		{
			AssignedAt: at(22),
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:          "ASM-FW-GCNMFW-WORK-PNC PNC-9001",
				Reference:        "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK-PNC PNC-9001!FLOW",
				ClaimID:          "PNC-9001",
				AssignedOperator: "ADMINTREATY1",
				PolicyNumber:     "99.001.2026.09001",
				InsuredName:      "PT Contoh Bukan Treaty",
			},
		},

		// Baris workbasket milik antrean LAIN. Ia harus tersaring di tab Teknik —
		// membuktikan penyaring nama akun antrean berjalan, bukan sekadar "ada di
		// workbasket".
		{
			AssignedAt:     at(22),
			FromWorkbasket: true,
			Item: inboxclaimtreatyprop.WorkItem{
				WorkKey:          "ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-3001",
				Reference:        "ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-3001!FLOW",
				ClaimID:          "CLMP-3001",
				AssignedOperator: "AntreanLain",
				PolicyNumber:     "99.001.2026.00006",
				InsuredName:      "PT Contoh Antrean Lain",
			},
		},
	}
}
