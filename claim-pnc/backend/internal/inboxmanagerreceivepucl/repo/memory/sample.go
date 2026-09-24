package memory

import (
	"time"

	"claim-pnc/internal/inboxmanagerreceivepucl"
)

// SampleRows adalah baris contoh untuk pengembangan lokal dan uji.
//
// # Seluruhnya KARANGAN
//
// Tidak satu pun nomor polis, nama tertanggung, nomor case, maupun nama petugas di bawah
// berasal dari data nyata. `D-69` melarang menulis data nasabah ke berkas yang di-commit,
// dan larangan itu tidak mengenal pengecualian untuk "data contoh".
//
// Namanya sengaja dibuat jelas-jelas karangan — "Tertanggung Contoh", "PT Contoh" — supaya
// tidak ada yang mengira ia salinan produksi bila kelak terbaca di layar pengembangan.
//
// # Apa yang dibuktikan susunan ini
//
// Delapan baris, dipilih supaya setiap penyaring modul ini punya baris yang LOLOS dan baris
// yang TERTOLAK olehnya. Uji yang seluruh barisnya lolos tidak membuktikan penyaringnya
// bekerja.
//
//	baris  membuktikan
//	-----  ----------------------------------------------------------------------------
//	1, 2   tab Receive PA berisi berkas ber-Group Panel 002
//	3, 4   tab Receive NONMBU berisi berkas ber-Group Panel lain
//	5      berkas tanpa Group Panel TIDAK muncul di tab mana pun
//	6      klaim (kelas berbeda) TIDAK bocor ke tab Receive
//	7, 8   tab RCL/PUCL berisi klaim di antrean RCLPUCL, satu RCL dan satu PUCL
//	9      klaim SELESAI tidak muncul di tab RCL/PUCL
//	10     klaim di antrean bersama LAIN tidak muncul di tab RCL/PUCL
func SampleRows() []Row {
	// Waktu dasar dibuat tetap, bukan `time.Now()`. Urutan baris pada uji karena itu tidak
	// berubah menurut hari, dan uji yang memeriksanya tidak gagal esok hari tanpa ada yang
	// menyentuh kode.
	base := time.Date(2026, time.September, 22, 8, 0, 0, 0, time.UTC)
	receive := inboxmanagerreceivepucl.WorkClassReceiveDocument
	claim := inboxmanagerreceivepucl.WorkClassClaim

	return []Row{
		{
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:            "ASM-FW-GCNMFW-WORK RCV-900001",
				CaseID:               "RCV-900001",
				PolicyNumber:         "CONTOH-PA-0001",
				ClaimNumber:          "PNCN.26.0001",
				InsuredName:          "Tertanggung Contoh Satu",
				LossDate:             "2026-09-01",
				SenderName:           "Pengirim Contoh Satu",
				DocumentReceivedDate: "03/09/2026",
				InboxEntryAt:         "2026-09-03 09:14:00",
			},
			WorkClass:        receive,
			AssignedOperator: "PETUGASCONTOH1",
			GroupPanel:       inboxmanagerreceivepucl.GroupPanelPA,
			CreatedAt:        base.Add(-2 * time.Hour),
		},
		{
			// Berkas PA yang BELUM diregistrasi menjadi klaim: nomor klaim PNC-nya kosong,
			// dan itu keadaan yang sah — bukan data hilang.
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:            "ASM-FW-GCNMFW-WORK RCV-900002",
				CaseID:               "RCV-900002",
				PolicyNumber:         "CONTOH-PA-0002",
				InsuredName:          "Tertanggung Contoh Dua",
				LossDate:             "2026-09-05",
				SenderName:           "Pengirim Contoh Dua",
				DocumentReceivedDate: "06/09/2026",
				InboxEntryAt:         "2026-09-06 10:02:00",
			},
			WorkClass:        receive,
			AssignedOperator: "PETUGASCONTOH2",
			GroupPanel:       inboxmanagerreceivepucl.GroupPanelPA,
			CreatedAt:        base.Add(-5 * time.Hour),
		},
		{
			// Group Panel 006 — Fire/Property. Ia bukan PA, sehingga muncul di tab NONMBU.
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:            "ASM-FW-GCNMFW-WORK RCV-900003",
				CaseID:               "RCV-900003",
				PolicyNumber:         "CONTOH-FIRE-0003",
				ClaimNumber:          "PNCN.26.0003",
				InsuredName:          "PT Contoh Properti",
				LossDate:             "2026-08-28",
				SenderName:           "Pengirim Contoh Tiga",
				DocumentReceivedDate: "30/08/2026",
				InboxEntryAt:         "2026-08-30 14:20:00",
			},
			WorkClass:        receive,
			AssignedOperator: "PETUGASCONTOH1",
			GroupPanel:       "006",
			CreatedAt:        base.Add(-30 * time.Hour),
		},
		{
			// Berkas NONMBU tanpa pasangan di tabel cermin: Nama Pengirim dan Tanggal
			// Terima Dokumen kosong. Ia SENGAJA ada — `LEFT JOIN` pada kueri aslinya
			// membuat baris seperti ini tetap muncul, dan penyimpanan memori harus
			// menunjukkan hal yang sama.
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK RCV-900004",
				CaseID:       "RCV-900004",
				PolicyNumber: "CONTOH-MARINE-0004",
				InsuredName:  "PT Contoh Kargo",
				LossDate:     "2026-09-10",
				InboxEntryAt: "2026-09-11 08:45:00",
			},
			WorkClass:        receive,
			AssignedOperator: "PETUGASCONTOH3",
			GroupPanel:       "004",
			CreatedAt:        base.Add(-40 * time.Hour),
		},
		{
			// Group Panel KOSONG. Ia tidak muncul di tab Receive mana pun, persis seperti
			// `GROUPPANEL_1 <> '002'` yang tidak menangkap NULL di Oracle.
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK RCV-900005",
				CaseID:       "RCV-900005",
				PolicyNumber: "CONTOH-TANPA-PANEL",
				InsuredName:  "Tertanggung Contoh Lima",
				InboxEntryAt: "2026-09-12 11:30:00",
			},
			WorkClass:        receive,
			AssignedOperator: "PETUGASCONTOH1",
			CreatedAt:        base.Add(-50 * time.Hour),
		},
		{
			// KLAIM, bukan berkas penerimaan dokumen, tetapi berada di tabel penugasan per
			// orang. Ia tidak boleh muncul di tab Receive mana pun — pembedanya semata
			// `PXOBJCLASS`, dan itulah yang dibuktikan baris ini.
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK PNC-800001",
				CaseID:       "PNC-800001",
				PolicyNumber: "CONTOH-KLAIM-8001",
				InsuredName:  "Tertanggung Contoh Enam",
				InboxEntryAt: "2026-09-13 09:00:00",
			},
			WorkClass:        claim,
			AssignedOperator: "PETUGASCONTOH2",
			GroupPanel:       inboxmanagerreceivepucl.GroupPanelPA,
			WorkStatus:       "Open",
			CreatedAt:        base.Add(-60 * time.Hour),
		},
		{
			// Jalur RCL — `RCL_PUCL_1 = '1'`.
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:       "ASM-FW-GCNMFW-WORK PNC-800002",
				CaseID:          "PNC-800002",
				PolicyNumber:    "CONTOH-KLAIM-8002",
				InsuredName:     "Tertanggung Contoh Tujuh",
				InboxEntryAt:    "2026-09-14 13:05:00",
				AnalystNote:     "Dokumen pendukung tidak lengkap; menunggu tanggapan cabang.",
				Track:           inboxmanagerreceivepucl.TrackRCL,
				TrackStatus:     "Menunggu Keputusan",
				LetterPrintedAt: "16/09/2026",
				ClaimAge:        "8",
				ExpiryStatus:    "Belum Kadaluarsa",
			},
			WorkClass:        claim,
			FromWorkbasket:   true,
			AssignedOperator: inboxmanagerreceivepucl.RCLPUCLWorkbasket,
			WorkStatus:       "Open",
			CreatedAt:        base.Add(-70 * time.Hour),
		},
		{
			// Jalur PUCL — `RCL_PUCL_1 = '2'`. Suratnya BELUM dicetak, sehingga
			// `LetterPrintedAt` kosong. Baris seperti ini tidak akan muncul bila ketiga
			// penyaring job pengingat ikut dibawa — dan itulah alasan ketiganya tidak
			// dibawa; lihat catatan di inboxmanagerreceivepucl.sql.
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK PNC-800003",
				CaseID:       "PNC-800003",
				PolicyNumber: "CONTOH-KLAIM-8003",
				InsuredName:  "PT Contoh Ulang",
				InboxEntryAt: "2026-09-15 07:50:00",
				AnalystNote:  "Diajukan proses ulang setelah bukti baru diterima.",
				Track:        inboxmanagerreceivepucl.TrackPUCL,
				TrackStatus:  "Dalam Proses",
				ClaimAge:     "5",
				ExpiryStatus: "Belum Kadaluarsa",
			},
			WorkClass:        claim,
			FromWorkbasket:   true,
			AssignedOperator: inboxmanagerreceivepucl.RCLPUCLWorkbasket,
			WorkStatus:       "Open",
			CreatedAt:        base.Add(-80 * time.Hour),
		},
		{
			// Klaim SELESAI di antrean yang sama. Ia tidak boleh muncul — satu-satunya
			// penyaring Report Definition RCL/PUCL adalah status kerja ini.
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK PNC-800004",
				CaseID:       "PNC-800004",
				PolicyNumber: "CONTOH-KLAIM-8004",
				InsuredName:  "Tertanggung Contoh Sembilan",
				InboxEntryAt: "2026-09-16 15:10:00",
				Track:        inboxmanagerreceivepucl.TrackRCL,
			},
			WorkClass:        claim,
			FromWorkbasket:   true,
			AssignedOperator: inboxmanagerreceivepucl.RCLPUCLWorkbasket,
			WorkStatus:       inboxmanagerreceivepucl.WorkStatusCompleted,
			CreatedAt:        base.Add(-90 * time.Hour),
		},
		{
			// Klaim di antrean bersama LAIN. Ia tidak boleh muncul — dan baris inilah yang
			// membuktikan penyaring antrean benar-benar dipakai, bukan sekadar tertulis.
			// Tanpanya, kueri yang lupa menyaring antrean tetap lulus seluruh uji.
			Item: inboxmanagerreceivepucl.WorkItem{
				Reference:    "ASM-FW-GCNMFW-WORK PNC-800005",
				CaseID:       "PNC-800005",
				PolicyNumber: "CONTOH-KLAIM-8005",
				InsuredName:  "Tertanggung Contoh Sepuluh",
				InboxEntryAt: "2026-09-17 16:40:00",
			},
			WorkClass:        claim,
			FromWorkbasket:   true,
			AssignedOperator: "Compliance",
			WorkStatus:       "Open",
			CreatedAt:        base.Add(-100 * time.Hour),
		},
	}
}
