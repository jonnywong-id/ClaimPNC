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
		return at, at.Format(inboxrclpucl.DisplayTimeLayout)
	}

	// claimAge menyusun isi kolom "Lama Klaim".
	//
	// # Ia TANGGAL, bukan angka — dan baris contoh ini sempat menyatakan sebaliknya
	//
	// Sampai 2026-09-30 kesebelas baris di bawah mengisinya dengan bilangan ("12", "5", …),
	// mengikuti judul kolomnya. Judul itu menyesatkan sejak di Pega: Work Owner menjelaskan
	// isinya **tanggal kirim untuk proses PUCL**, dan kolomnya terverifikasi bertipe
	// `TIMESTAMP(6)` di Oracle pada hari yang sama.
	//
	// Baris contoh yang bentuknya berbeda dari produksi meloloskan uji yang tidak akan
	// lolos di produksi, sehingga bilangan itu diganti tanggal.
	//
	// Ia sengaja dibuat satu detik LEBIH AWAL daripada "Tanggal Masuk Inbox", bukan sama
	// persis. Di produksi kedua kolom memang terpaut milidetik — keduanya ditulis pada
	// langkah yang sama — tetapi baris contoh yang membuatnya identik akan meloloskan
	// tertukarnya kedua isian tanpa ketahuan.
	claimAge := func(day int) string {
		at, _ := sent(day)
		return at.Add(-time.Second).Format(inboxrclpucl.DisplayTimeLayout)
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
			ClaimAge:        claimAge(10),
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

		// Isian layar kerja. Ketiganya HANYA dipakai Detail, bukan oleh grid mana pun.
		//
		// FirstObjectName mengisi DUA isian sekaligus di layar kerja — "Nama Peserta" dan
		// "UP" — karena activity penyusun lampiran memang menunjuk ekspresi yang sama
		// untuk keduanya.
		LossDate:          "2026-09-01",
		FirstObjectName:   "Objek Contoh Satu",
		FirstProposeValue: "15000000",
		PUCLNote:          "Menunggu kelengkapan dari cabang.",
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
			ClaimAge:        claimAge(11),
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

		// SENGAJA tanpa objek dan tanpa adjustment: subkueri yang tidak mengembalikan
		// baris menghasilkan isian turunan KOSONG, dan itu keadaan yang sah — bukan
		// kegagalan. Uji layar kerja memakai baris ini untuk membuktikannya.
		LossDate: "2026-09-02",
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
			ClaimAge:        claimAge(12),
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
			ClaimAge:        claimAge(13),
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

		LossDate:          "2026-09-04",
		FirstObjectName:   "Objek Contoh Empat",
		FirstProposeValue: "8750000",
		PUCLNote:          "Surat sudah dikirim ke tertanggung.",
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
			ClaimAge:        claimAge(14),
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
			ClaimAge:        claimAge(15),
			ExpiryStatus:    "Belum Kadaluarsa",
		},
		TrackCode:        inboxrclpucl.TrackCodePUCL,
		WorkClass:        claim,
		AssignedOperator: basket,
		WorkStatus:       "Open",
		ExpiryCaseStatus: "1",
		PUCLApprove:      inboxrclpucl.PUCLReturnedToAnalyst,
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
			ClaimAge:        claimAge(16),
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
			ClaimAge:        claimAge(17),
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
			ClaimAge:        claimAge(18),
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
			ClaimAge:        claimAge(19),
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
			ClaimAge:        claimAge(13),
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
