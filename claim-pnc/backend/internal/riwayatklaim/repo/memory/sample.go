package memory

import (
	"time"

	"claim-pnc/internal/riwayatklaim"
)

// SELURUH ISI BERKAS INI KARANGAN.
//
// Tidak ada satu pun nomor polis, nama tertanggung, nomor klaim, atau tanggal lahir yang
// berasal dari data nyata. `D-69` melarang data nasabah masuk ke berkas yang di-commit,
// dan larangan itu tidak mengenal pengecualian "hanya untuk contoh".
//
// Nomor klaimnya sengaja memuat KEDUA format yang hidup berdampingan selama masa paralel:
// `PNC-xxxx` terbitan Pega dan `PNCN.YY.xxxx` terbitan sistem baru (`D-71`). Dengan begitu
// pencarian No Klaim dapat dicoba terhadap keduanya — dan perubahan pada kueri tipe 7
// (lihat riwayatklaim.sql) terbukti bekerja, bukan hanya diyakini.

// SampleClaims mengembalikan riwayat klaim contoh.
//
// Isinya dipilih supaya KESEBELAS tipe pencarian yang tersedia dapat dicoba tanpa Oracle:
// ada baris ber-nomor PLA, ber-nomor DLA, ber-ID balai lelang, ber-nomor survei, dan satu
// baris Personal Accident yang membawa peserta beserta tanggal lahirnya.
func SampleClaims() []Claim {
	lossDate := func(year int, month time.Month, day int) *time.Time {
		moment := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		return &moment
	}

	return []Claim{
		{
			History: riwayatklaim.ClaimHistory{
				Reference:     "ASM-FW-GCNMFW-WORK PNC-9001",
				Number:        "PNC-9001",
				PolicyNumber:  "POL-CONTOH-0001",
				InsuredName:   "PT SUMBER CONTOH SENTOSA",
				LossDate:      lossDate(2026, time.March, 12),
				BusinessName:  "Fire / Property",
				BranchName:    "Cabang Contoh Pusat",
				WorkStatus:    "Resolved-Completed",
				ClaimPosition: "Paid",
				CloseDate:     lossDate(2026, time.May, 2),
				CloseNote:     "Klaim selesai dibayar penuh.",
				TechnicalPIC:  "PIC Teknik Contoh A",
			},
			ObjectName:       "Gudang Contoh Blok A",
			PLANumber:        "PLA-CONTOH-0001",
			DLANumber:        "DLA-CONTOH-0001",
			AcceptanceNumber: "AKS-CONTOH-0001",
			SurveyNumber:     "SRV-CONTOH-0001",
		},
		{
			History: riwayatklaim.ClaimHistory{
				Reference:     "ASM-FW-GCNMFW-WORK PNC-9002",
				Number:        "PNC-9002",
				PolicyNumber:  "POL-CONTOH-0002",
				InsuredName:   "CV MITRA CONTOH ABADI",
				LossDate:      lossDate(2026, time.March, 12),
				BusinessName:  "Marine Cargo",
				BranchName:    "Cabang Contoh Timur",
				WorkStatus:    "Open",
				ClaimPosition: "Claim Committee",
				TechnicalPIC:  "PIC Teknik Contoh B",
			},
			ObjectName:   "Kontainer Contoh 20 Kaki",
			PLANumber:    "PLA-CONTOH-0002",
			SurveyNumber: "SRV-CONTOH-0002",
		},
		{
			History: riwayatklaim.ClaimHistory{
				Reference:     "ASM-FW-GCNMFW-WORK PNC-9003",
				Number:        "PNC-9003",
				PolicyNumber:  "POL-CONTOH-0003",
				InsuredName:   "PT SALVAGE CONTOH JAYA",
				LossDate:      lossDate(2025, time.November, 4),
				BusinessName:  "Aneka",
				BranchName:    "Cabang Contoh Barat",
				WorkStatus:    "Resolved-Completed",
				ClaimPosition: "Close Claim for this object",
				CloseDate:     lossDate(2026, time.January, 20),
				CloseNote:     "Ditutup setelah lelang barang sisa.",
				TechnicalPIC:  "PIC Teknik Contoh A",
			},
			ObjectName:       "Mesin Contoh Produksi",
			AcceptanceNumber: "AKS-CONTOH-0003",
			AuctionID:        "BL-CONTOH-77",
			DLANumber:        "DLA-CONTOH-0003",
		},
		{
			History: riwayatklaim.ClaimHistory{
				Reference:     "ASM-FW-GCNMFW-WORK PNC-9004",
				Number:        "PNC-9004",
				PolicyNumber:  "POL-CONTOH-0004",
				InsuredName:   "YAYASAN CONTOH PEDULI",
				LossDate:      lossDate(2026, time.February, 9),
				BusinessName:  "Personal Accident",
				BranchName:    "Cabang Contoh Pusat",
				WorkStatus:    "Open",
				ClaimPosition: "Analyst",
				TechnicalPIC:  "Analyst Doctor Contoh",
			},
			ObjectName: "Peserta Contoh Kelompok 1",
			// Satu-satunya baris yang membawa peserta. Ia ada supaya pencarian Tanggal
			// Lahir PUNYA sasaran yang seharusnya ditemukan — dan uji dapat membuktikan
			// bahwa yang membuatnya tidak ditemukan adalah cacat yang direplikasi, bukan
			// data contoh yang kebetulan kosong.
			PersonName:      "Peserta Contoh Satu",
			PersonBirthDate: lossDate(1990, time.July, 17),
			SurveyNumber:    "SRV-CONTOH-0004",
		},
		{
			History: riwayatklaim.ClaimHistory{
				Reference: "PNCN.26.0001",
				// Nomor terbitan SISTEM BARU — tanpa awalan Pega pada Reference-nya,
				// persis yang ditetapkan `D-22` dan `D-71`.
				Number:        "PNCN.26.0001",
				PolicyNumber:  "POL-CONTOH-0005",
				InsuredName:   "PT PENDATANG CONTOH BARU",
				LossDate:      lossDate(2026, time.September, 1),
				BusinessName:  "Travel",
				BranchName:    "Cabang Contoh Selatan",
				WorkStatus:    "Open",
				ClaimPosition: "Register",
				TechnicalPIC:  "PIC Teknik Contoh C",
			},
			ObjectName: "Perjalanan Contoh Asia",
		},
	}
}

// SampleProtections mengembalikan baris proteksi data contoh.
//
// # Kenapa ini ada, dan kenapa ia penting
//
// Tanpa baris proteksi, gerbang menolak SETIAP pengguna dan layar tidak dapat dibuka
// sama sekali — itulah perilaku sistem lama, dan Work Owner memutuskan 2026-09-20 ia
// dibangun penuh. Di Oracle, barisnya didaftarkan lewat Master Proteksi Data milik sistem
// lama. Di memori, tidak ada yang mendaftarkannya, sehingga contohnya disediakan di sini.
//
// Ketiga baris di bawah sengaja berbeda keadaan supaya ketiga jalur gerbang dapat dicoba:
// lolos, hampir habis, dan sudah habis. Pengguna tiruan yang TIDAK disebut di sini —
// `profilbolong` — mewakili jalur keempat: belum terdaftar sama sekali.
func SampleProtections() []riwayatklaim.Protection {
	return []riwayatklaim.Protection{
		{
			Login:       "adminpnc",
			SearchQuota: 50,
			ViewQuota:   50,
			SubModules:  []string{riwayatklaim.ModuleKey},
		},
		{
			// Jatahnya tinggal satu: membuka layar sekali lagi menghabiskannya, dan
			// pembukaan berikutnya ditolak. Jalur "jatah habis" karena itu dapat dicoba
			// tanpa menunggu lima puluh kali percobaan.
			Login:       "pictekniks",
			SearchQuota: 1,
			ViewQuota:   5,
			SubModules:  []string{riwayatklaim.ModuleKey},
			MaskPhone:   true,
			MaskIDCard:  true,
		},
		{
			Login:       "penggunanonaktif",
			SearchQuota: 0,
			ViewQuota:   0,
			SubModules:  []string{riwayatklaim.ModuleKey},
		},
	}
}
