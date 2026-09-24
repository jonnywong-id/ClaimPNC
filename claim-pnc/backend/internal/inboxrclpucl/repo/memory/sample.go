package memory

import (
	"time"

	"claim-pnc/internal/inboxrclpucl"
)

// SampleRows adalah baris contoh untuk pengembangan lokal dan uji.
//
// # Seluruhnya KARANGAN
//
// Tidak satu pun nomor polis, nama tertanggung, nomor case, maupun nama petugas di bawah
// berasal dari data nyata. `D-69` melarang menulis data nasabah ke berkas yang di-commit,
// dan larangan itu tidak mengenal pengecualian untuk "data contoh".
//
// Namanya sengaja dibuat jelas-jelas karangan — "Tertanggung Contoh", "CONTOH-…" — supaya
// tidak ada yang mengira ia salinan produksi bila kelak terbaca di layar pengembangan.
//
// # Apa yang dibuktikan susunan ini
//
// Sebelas baris, dipilih supaya SETIAP penyaring modul ini punya baris yang LOLOS dan baris
// yang TERTOLAK olehnya. Uji yang seluruh barisnya lolos tidak membuktikan penyaringnya
// bekerja — yang membuktikannya adalah baris yang seharusnya tidak muncul dan memang tidak
// muncul.
//
//	baris  membuktikan
//	-----  -----------------------------------------------------------------------------
//	1, 2   tab Cetak Surat berisi klaim tanpa tanggal cetak dan ber-STATUSCASE_1 '0'
//	3      STATUSCASE_1 selain '0' TIDAK muncul di tab Cetak Surat meski suratnya belum
//	       dicetak — penyaring kedua tab itu benar-benar dipakai
//	4, 5   tab Kelengkapan Dokumen berisi klaim bersurat, belum disetujui, bukan MSIG
//	6      klaim yang SUDAH disetujui tidak muncul di tab mana pun
//	7      klaim ber-PUCLAPPROVE_1 KOSONG tidak muncul di tab Kelengkapan Dokumen —
//	       meniru `NULL <> '1'` yang bernilai UNKNOWN, bukan TRUE
//	8      tab Klaim MSIG berisi klaim berpenanda MSIG, dan klaim itu TIDAK bocor ke tab
//	       Kelengkapan Dokumen
//	9      klaim SELESAI tidak muncul di tab mana pun
//	10     klaim di antrean bersama LAIN tidak muncul di tab mana pun
//	11     klaim Personal Accident di LUAR antrean RCL/PUCL — tidak muncul di tab mana pun,
//	       tetapi IKUT di laporan harian lewat cabang kedua UNION
func SampleRows() []Row {
	// Waktu dasar dibuat tetap, bukan `time.Now()`. Urutan baris pada uji karena itu tidak
	// berubah menurut hari, dan uji yang memeriksanya tidak gagal esok hari tanpa ada yang
	// menyentuh kode.
	base := time.Date(2026, time.September, 22, 8, 0, 0, 0, time.UTC)
	claim := inboxrclpucl.WorkClassClaim
	basket := inboxrclpucl.RCLPUCLWorkbasket

	// sent menyusun pasangan waktu kirim dan teksnya sekaligus.
	//
	// Keduanya sengaja berasal dari satu sumber: yang satu dipakai menyaring dan
	// mengurutkan laporan, yang lain digambar layar, dan membiarkannya menyimpang akan
	// membuat baris contoh menceritakan dua hal yang berbeda.
	sent := func(day int) (time.Time, string) {
		at := time.Date(2026, time.September, day, 9, 30, 0, 0, time.UTC)
		return at, at.Format("2006-01-02 15:04:05")
	}

	rows := []Row{}

	// ---- Tab Cetak Surat: surat BELUM dicetak, STATUSCASE_1 = '0' -------------------
	at1, text1 := sent(10)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:    "ASM-FW-GCNMFW-WORK PNC-700001",
			CaseID:       "PNC-700001",
			PolicyNumber: "CONTOH-RCL-0001",
			InsuredName:  "Tertanggung Contoh Satu",
			InboxEntryAt: text1,
			AnalystNote:  "Dokumen pendukung tidak lengkap, diteruskan ke jalur RCL.",
			// Kosong — inilah yang menempatkannya di tab Cetak Surat.
			LetterPrintedAt: "",
			ClaimAge:        "12",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodeRCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       "Open",
		ExpiryCaseStatus: inboxrclpucl.ExpiryStatusActive,
		ClaimStatus:      "1142",
		GroupPanel:       "006",
		CreatedAt:        base.Add(-3 * time.Hour),
		SentAt:           at1,
	})

	at2, text2 := sent(11)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700002",
			CaseID:          "PNC-700002",
			PolicyNumber:    "CONTOH-PUCL-0002",
			InsuredName:     "Tertanggung Contoh Dua",
			InboxEntryAt:    text2,
			AnalystNote:     "Permintaan proses ulang dari cabang.",
			LetterPrintedAt: "",
			ClaimAge:        "5",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodePUCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       "Open",
		ExpiryCaseStatus: inboxrclpucl.ExpiryStatusActive,
		ClaimStatus:      "1164",
		GroupPanel:       "004",
		CreatedAt:        base.Add(-2 * time.Hour),
		SentAt:           at2,
	})

	// ---- TERTOLAK tab Cetak Surat: STATUSCASE_1 bukan '0' --------------------------
	//
	// Suratnya belum dicetak, tetapi penanda kasusnya berbeda. Ia tidak muncul di tab mana
	// pun — dan itulah yang membuktikan penyaring STATUSCASE_1 benar-benar dipakai, bukan
	// sekadar ikut tertulis.
	at3, text3 := sent(12)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700003",
			CaseID:          "PNC-700003",
			PolicyNumber:    "CONTOH-RCL-0003",
			InsuredName:     "Tertanggung Contoh Tiga",
			InboxEntryAt:    text3,
			AnalystNote:     "Penanda kasus berbeda; tidak masuk antrean cetak surat.",
			LetterPrintedAt: "",
			ClaimAge:        "30",
			ExpiryStatus:    "Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodeRCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       "Open",
		ExpiryCaseStatus: "1",
		ClaimStatus:      "1142",
		GroupPanel:       "003",
		CreatedAt:        base.Add(-90 * time.Minute),
		SentAt:           at3,
	})

	// ---- Tab Kelengkapan Dokumen: bersurat, belum disetujui, bukan MSIG ------------
	at4, text4 := sent(13)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700004",
			CaseID:          "PNC-700004",
			PolicyNumber:    "CONTOH-RCL-0004",
			InsuredName:     "Tertanggung Contoh Empat",
			InboxEntryAt:    text4,
			AnalystNote:     "Surat penolakan sudah dikirim, menunggu tanggapan.",
			LetterPrintedAt: "2026-09-15",
			ClaimAge:        "8",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodeRCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       "Open",
		ExpiryCaseStatus: inboxrclpucl.ExpiryStatusActive,
		PUCLApprove:      "0",
		ClaimStatus:      "1142",
		GroupPanel:       "006",
		CreatedAt:        base.Add(-75 * time.Minute),
		SentAt:           at4,
	})

	at5, text5 := sent(14)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700005",
			CaseID:          "PNC-700005",
			PolicyNumber:    "CONTOH-PUCL-0005",
			InsuredName:     "Tertanggung Contoh Lima",
			InboxEntryAt:    text5,
			AnalystNote:     "Menunggu kelengkapan dokumen dari tertanggung.",
			LetterPrintedAt: "2026-09-16",
			ClaimAge:        "3",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodePUCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       "Open",
		ExpiryCaseStatus: "1",
		PUCLApprove:      "0",
		ClaimStatus:      "1164",
		GroupPanel:       "005",
		CreatedAt:        base.Add(-60 * time.Minute),
		SentAt:           at5,
	})

	// ---- TERTOLAK: sudah disetujui -------------------------------------------------
	at6, text6 := sent(15)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700006",
			CaseID:          "PNC-700006",
			PolicyNumber:    "CONTOH-PUCL-0006",
			InsuredName:     "Tertanggung Contoh Enam",
			InboxEntryAt:    text6,
			AnalystNote:     "Sudah disetujui; keluar dari antrean.",
			LetterPrintedAt: "2026-09-17",
			ClaimAge:        "2",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodePUCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       "Open",
		ExpiryCaseStatus: "1",
		PUCLApprove:      inboxrclpucl.PUCLApproved,
		ClaimStatus:      "1163",
		GroupPanel:       "006",
		CreatedAt:        base.Add(-50 * time.Minute),
		SentAt:           at6,
	})

	// ---- TERTOLAK: PUCLAPPROVE_1 KOSONG --------------------------------------------
	//
	// Baris paling penting di berkas ini. Di Oracle, `NULL <> '1'` bernilai UNKNOWN — bukan
	// TRUE — sehingga baris ini TIDAK muncul di tab Kelengkapan Dokumen meski suratnya
	// sudah dicetak dan ia jelas belum disetujui.
	//
	// Perilakunya terasa keliru, dan memang mungkin keliru. Ia tetap ditiru karena yang
	// diuji adalah kesetaraan dengan Pega, bukan kebenaran aturannya — dan perbaikannya
	// bukan wewenang berkas ini.
	at7, text7 := sent(16)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700007",
			CaseID:          "PNC-700007",
			PolicyNumber:    "CONTOH-RCL-0007",
			InsuredName:     "Tertanggung Contoh Tujuh",
			InboxEntryAt:    text7,
			AnalystNote:     "Penanda persetujuan belum pernah diisi.",
			LetterPrintedAt: "2026-09-18",
			ClaimAge:        "9",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodeRCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       "Open",
		ExpiryCaseStatus: "1",
		PUCLApprove:      "",
		ClaimStatus:      "1142",
		GroupPanel:       "003",
		CreatedAt:        base.Add(-45 * time.Minute),
		SentAt:           at7,
	})

	// ---- Tab Klaim MSIG ------------------------------------------------------------
	//
	// Di produksi kolom penandanya tampaknya tidak pernah terisi, sehingga tab ini
	// kemungkinan selalu kosong. Di sini ia sengaja diisi supaya tabnya punya sesuatu untuk
	// dibuktikan — dan supaya terbukti pula ia TIDAK bocor ke tab Kelengkapan Dokumen.
	at8, text8 := sent(17)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700008",
			CaseID:          "PNC-700008",
			PolicyNumber:    "CONTOH-MSIG-0008",
			InsuredName:     "Tertanggung Contoh Delapan",
			InboxEntryAt:    text8,
			AnalystNote:     "Klaim jalur MSIG.",
			LetterPrintedAt: "2026-09-19",
			ClaimAge:        "4",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodePUCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       "Open",
		ExpiryCaseStatus: "1",
		PUCLApprove:      "0",
		MSIG:             inboxrclpucl.MSIGMarker,
		ClaimStatus:      "1164",
		GroupPanel:       "006",
		CreatedAt:        base.Add(-40 * time.Minute),
		SentAt:           at8,
	})

	// ---- TERTOLAK: klaim sudah selesai ---------------------------------------------
	at9, text9 := sent(18)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700009",
			CaseID:          "PNC-700009",
			PolicyNumber:    "CONTOH-RCL-0009",
			InsuredName:     "Tertanggung Contoh Sembilan",
			InboxEntryAt:    text9,
			AnalystNote:     "Sudah tuntas; tidak lagi menunggu tindakan.",
			LetterPrintedAt: "",
			ClaimAge:        "1",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodeRCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       inboxrclpucl.WorkStatusCompleted,
		ExpiryCaseStatus: inboxrclpucl.ExpiryStatusActive,
		ClaimStatus:      "1163",
		GroupPanel:       "006",
		CreatedAt:        base.Add(-30 * time.Minute),
		SentAt:           at9,
	})

	// ---- TERTOLAK: antrean bersama LAIN --------------------------------------------
	at10, text10 := sent(19)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700010",
			CaseID:          "PNC-700010",
			PolicyNumber:    "CONTOH-LAIN-0010",
			InsuredName:     "Tertanggung Contoh Sepuluh",
			InboxEntryAt:    text10,
			AnalystNote:     "Berada di antrean komite, bukan RCL/PUCL.",
			LetterPrintedAt: "",
			ClaimAge:        "6",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodeRCL,
		WorkClass:        claim,
		AssignedOperator: "KOMITEPNC",
		WorkStatus:       "Open",
		ExpiryCaseStatus: inboxrclpucl.ExpiryStatusActive,
		ClaimStatus:      "1149",
		GroupPanel:       "006",
		CreatedAt:        base.Add(-20 * time.Minute),
		SentAt:           at10,
	})

	// ---- Hanya untuk LAPORAN HARIAN: klaim PA di luar antrean RCL/PUCL -------------
	//
	// Ia TIDAK muncul di tab mana pun — antreannya bukan RCLPUCL. Tetapi ia IKUT di laporan
	// harian lewat cabang kedua `UNION`, yang mengambil seluruh klaim ber-Group Panel '002'
	// pada rentang tanggal yang sama tanpa melihat antreannya sama sekali.
	//
	// Inilah satu-satunya baris yang membuktikan laporan dan tabel memang berbeda isinya.
	at11, text11 := sent(13)
	rows = append(rows, Row{
		Item: inboxrclpucl.WorkItem{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-700011",
			CaseID:          "PNC-700011",
			PolicyNumber:    "CONTOH-PA-0011",
			InsuredName:     "Tertanggung Contoh Sebelas",
			InboxEntryAt:    text11,
			AnalystNote:     "Klaim Personal Accident di luar antrean RCL/PUCL.",
			LetterPrintedAt: "2026-09-14",
			ClaimAge:        "7",
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodePUCL,
		WorkClass:        claim,
		AssignedOperator: "PNCADMIN",
		WorkStatus:       "Open",
		ExpiryCaseStatus: "1",
		PUCLApprove:      "0",
		ClaimStatus:      "1147",
		GroupPanel:       inboxrclpucl.GroupPanelPA,
		CreatedAt:        base.Add(-10 * time.Minute),
		SentAt:           at11,
	})

	return rows
}
