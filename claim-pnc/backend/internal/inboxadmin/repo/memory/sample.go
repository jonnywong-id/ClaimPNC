package memory

import (
	"time"

	"claim-pnc/internal/inboxadmin"
)

// SampleOwner adalah login pemilik baris contoh pada tab yang ScopedToCaller.
//
// Ia dipakai adapter memori saja. Pada basis data sungguhan, pemiliknya adalah pengguna
// yang benar-benar membuat atau ditugasi barisnya.
const SampleOwner = "ADMINKLAIM"

// day membentuk tanggal UTC tanpa jam, supaya contoh terbaca dan hitungan Aging-nya
// mudah diperiksa dengan mata.
func day(year int, month time.Month, date int) *time.Time {
	at := time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
	return &at
}

// SampleRows adalah antrean contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa nomor polis dan nama tertanggungnya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama tertanggung, dan nomor klaim di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": tiap tab memperoleh baris yang membuat penyaringnya
// dapat dibuktikan bekerja.
//
//   - Tab ALL memuat empat lini bisnis sekaligus, sehingga dropdown Business dapat diuji.
//   - Kedua tab Unregistered RCV memuat baris ber-KURIR dan tanpa KURIR, sehingga
//     pemisahannya terlihat.
//   - Tab yang ScopedToCaller memuat satu baris MILIK ORANG LAIN, sehingga penyaring
//     kepemilikan yang lupa dipasang akan langsung terlihat sebagai baris yang bocor.
//   - Tanggalnya berjarak supaya kolom Aging tidak semuanya bernilai sama.
func SampleRows() []Row {
	return []Row{
		// ---------------------------------------------------------------- tab ALL
		{
			Tab:             inboxadmin.TabAll,
			GroupPanel:      "003",
			BusinessGroupID: "10001",
			Item: inboxadmin.WorkItem{
				CaseID:         "PNC-8801",
				Reference:      "ASM-FW-GCNMFW-WORK PNC-8801",
				PolicyNumber:   "16.001.2026.00101",
				InsuredName:    "PT Contoh Aneka Sejahtera",
				BusinessName:   "Aneka",
				BusinessSource: "Broker",
				BranchName:     "Jakarta Pusat",
				ClaimBranch:    "Jakarta Pusat",
				Creator:        SampleOwner,
				LossDate:       day(2026, time.August, 2),
				ReportDate:     day(2026, time.August, 5),
				InputDate:      day(2026, time.August, 6),
				ClaimPosition:  "On Progress",
				ClaimStatus:    "Register",
			},
		},
		{
			Tab:             inboxadmin.TabAll,
			GroupPanel:      "002",
			BusinessGroupID: "10002",
			Item: inboxadmin.WorkItem{
				CaseID:         "PNC-8802",
				Reference:      "ASM-FW-GCNMFW-WORK PNC-8802",
				PolicyNumber:   "16.002.2026.00120",
				InsuredName:    "Contoh Peserta PA",
				BusinessName:   "Personal Accident",
				BusinessSource: "Agen",
				BranchName:     "Surabaya",
				ClaimBranch:    "Surabaya",
				Creator:        "ADMINPA",
				LossDate:       day(2026, time.August, 20),
				ReportDate:     day(2026, time.August, 21),
				InputDate:      day(2026, time.August, 22),
				ClaimPosition:  "On Progress",
				ClaimStatus:    "Claim Committee",
			},
		},
		{
			Tab:             inboxadmin.TabAll,
			GroupPanel:      "005",
			BusinessGroupID: "10003",
			Item: inboxadmin.WorkItem{
				CaseID:         "PNCN.26.0007",
				Reference:      "PNCN.26.0007",
				PolicyNumber:   "16.005.2026.00033",
				InsuredName:    "Contoh Tertanggung Travel",
				BusinessName:   "Travel",
				BusinessSource: "Direct",
				BranchName:     "Bandung",
				ClaimBranch:    "Bandung",
				Creator:        SampleOwner,
				LossDate:       day(2026, time.September, 1),
				ReportDate:     day(2026, time.September, 2),
				InputDate:      day(2026, time.September, 2),
				ClaimPosition:  "On Progress",
				ClaimStatus:    "Register",
			},
		},
		{
			// Bonding — satu-satunya baris yang lolos saringan BONDING, dan satu-satunya
			// yang DIBUANG saringan NONMBU meski Group Panel-nya termasuk keempatnya.
			Tab:             inboxadmin.TabAll,
			GroupPanel:      "009",
			BusinessGroupID: "10015",
			Item: inboxadmin.WorkItem{
				CaseID:         "PNC-8804",
				Reference:      "ASM-FW-GCNMFW-WORK PNC-8804",
				PolicyNumber:   "16.009.2026.00077",
				InsuredName:    "PT Contoh Penjaminan",
				BusinessName:   "Bonding",
				BusinessSource: "Broker",
				BranchName:     "Jakarta Pusat",
				ClaimBranch:    "Jakarta Pusat",
				Creator:        "ADMINBOND",
				LossDate:       day(2026, time.July, 11),
				ReportDate:     day(2026, time.July, 14),
				InputDate:      day(2026, time.July, 15),
				ClaimPosition:  "On Progress",
				ClaimStatus:    "Register",
			},
		},

		// ------------------------------------------- tab Unregistered RCV dan RCV Online
		{
			Tab:             inboxadmin.TabUnregisteredRCV,
			GroupPanel:      "004",
			BusinessGroupID: "10004",
			Courier:         "JNE",
			Item: inboxadmin.WorkItem{
				CaseID:         "RCV-2201",
				Reference:      "ASM-FW-GCNMFW-WORK RCV-2201",
				PolicyNumber:   "16.004.2026.00210",
				InsuredName:    "PT Contoh Marine Cargo",
				BusinessName:   "Marine Cargo",
				BusinessSource: "Broker",
				BranchName:     "Semarang",
				ClaimBranch:    "Semarang",
				Creator:        "ADMINRCV",
				LossDate:       day(2026, time.August, 28),
				InputDate:      day(2026, time.August, 30),
				Note:           "Menunggu kelengkapan surat jalan.",
			},
		},
		{
			Tab:             inboxadmin.TabRCVOnline,
			GroupPanel:      "006",
			BusinessGroupID: "10005",
			Courier:         "Auto Service",
			Item: inboxadmin.WorkItem{
				CaseID:         "RCV-2202",
				Reference:      "ASM-FW-GCNMFW-WORK RCV-2202",
				PolicyNumber:   "16.006.2026.00301",
				InsuredName:    "PT Contoh Properti",
				BusinessName:   "Fire",
				BusinessSource: "Direct",
				BranchName:     "Medan",
				ClaimBranch:    "Medan",
				Creator:        "ADMINRCV",
				LossDate:       day(2026, time.September, 4),
				InputDate:      day(2026, time.September, 5),
				Note:           "Masuk lewat Auto Service.",
			},
		},

		// ------------------------------------------------------- tab Request Survey
		{
			Tab:   inboxadmin.TabRequestSurvey,
			Owner: SampleOwner,
			Item: inboxadmin.WorkItem{
				CaseID:       "PNC-8801",
				Reference:    "ASM-FW-GCNMFW-WORK PNC-8801",
				PolicyNumber: "16.001.2026.00101",
				RequestDate:  day(2026, time.September, 8),
				PolicyBranch: "Jakarta Pusat",
				SurveyBranch: "Jakarta Selatan",
				TechnicalPIC: "PICTEKNIK1",
				Surveyor:     "Surveyor Contoh",
				SurveyNumber: "SRV-000123",
			},
		},
		{
			// Milik ORANG LAIN. Ia tidak boleh muncul bagi SampleOwner — bila muncul,
			// penyaring kepemilikan tidak terpasang.
			Tab:   inboxadmin.TabRequestSurvey,
			Owner: "ADMINLAIN",
			Item: inboxadmin.WorkItem{
				CaseID:       "PNC-8899",
				Reference:    "ASM-FW-GCNMFW-WORK PNC-8899",
				PolicyNumber: "16.001.2026.00999",
				RequestDate:  day(2026, time.September, 9),
				PolicyBranch: "Denpasar",
				SurveyBranch: "Denpasar",
				TechnicalPIC: "PICTEKNIK9",
				Surveyor:     "Surveyor Lain",
				SurveyNumber: "SRV-000999",
			},
		},

		// ------------------------------------------------------ tab Request Dokumen
		{
			Tab:   inboxadmin.TabRequestDocument,
			Owner: SampleOwner,
			Item: inboxadmin.WorkItem{
				CaseID:         "PNC-8802",
				Reference:      "ASM-FW-GCNMFW-WORK PNC-8802",
				PolicyNumber:   "16.002.2026.00120",
				InsuredName:    "Contoh Peserta PA",
				BusinessName:   "Personal Accident",
				BusinessSource: "Agen",
				LossDate:       day(2026, time.August, 20),
				ReportDate:     day(2026, time.August, 21),
				InputDate:      day(2026, time.August, 25),
			},
		},

		// ------------------------------------------------------- tab All Case Admin
		{
			Tab:   inboxadmin.TabAllCaseAdmin,
			Owner: SampleOwner,
			Item: inboxadmin.WorkItem{
				CaseID:         "PNC-8801",
				Reference:      "ASM-FW-GCNMFW-WORK PNC-8801",
				PolicyNumber:   "16.001.2026.00101",
				InsuredName:    "PT Contoh Aneka Sejahtera",
				BusinessName:   "Aneka",
				BusinessSource: "Broker",
				LossDate:       day(2026, time.August, 2),
				ReportDate:     day(2026, time.August, 5),
				InputDate:      day(2026, time.August, 6),
			},
		},
		{
			Tab:   inboxadmin.TabAllCaseAdmin,
			Owner: SampleOwner,
			Item: inboxadmin.WorkItem{
				CaseID:         "PNCN.26.0007",
				Reference:      "PNCN.26.0007",
				PolicyNumber:   "16.005.2026.00033",
				InsuredName:    "Contoh Tertanggung Travel",
				BusinessName:   "Travel",
				BusinessSource: "Direct",
				LossDate:       day(2026, time.September, 1),
				ReportDate:     day(2026, time.September, 2),
				InputDate:      day(2026, time.September, 2),
			},
		},
		{
			Tab:   inboxadmin.TabAllCaseAdmin,
			Owner: "ADMINLAIN",
			Item: inboxadmin.WorkItem{
				CaseID:         "PNC-8899",
				Reference:      "ASM-FW-GCNMFW-WORK PNC-8899",
				PolicyNumber:   "16.001.2026.00999",
				InsuredName:    "PT Contoh Milik Orang Lain",
				BusinessName:   "Aneka",
				BusinessSource: "Broker",
				LossDate:       day(2026, time.September, 3),
				ReportDate:     day(2026, time.September, 4),
				InputDate:      day(2026, time.September, 4),
			},
		},

		// --------------------------------------------------------- tab Branch Claim
		{
			Tab:             inboxadmin.TabBranchClaim,
			GroupPanel:      "002",
			BusinessGroupID: "10002",
			Item: inboxadmin.WorkItem{
				CaseID:         "PNC-8802",
				Reference:      "ASM-FW-GCNMFW-WORK PNC-8802",
				PolicyNumber:   "16.002.2026.00120",
				InsuredName:    "Contoh Peserta PA",
				BusinessSource: "Agen",
				BranchName:     "Surabaya",
				ClaimBranch:    "Surabaya",
				Creator:        "ADMINPA",
				LossDate:       day(2026, time.August, 20),
				ReportDate:     day(2026, time.August, 21),
				InputDate:      day(2026, time.August, 22),
				LODDate:        day(2026, time.September, 1),
				ClaimPosition:  "On Progress",
				LODStatus:      "Belum Upload",
			},
		},

		// ------------------------------------------------------ tab Status RCL/PUCL
		{
			Tab: inboxadmin.TabRCLPUCL,
			Item: inboxadmin.WorkItem{
				CaseID:          "PNC-8804",
				Reference:       "ASM-FW-GCNMFW-WORK PNC-8804",
				PolicyNumber:    "16.009.2026.00077",
				InsuredName:     "PT Contoh Penjaminan",
				InboxDate:       day(2026, time.August, 18),
				AnalystNote:     "Dokumen pendukung belum lengkap, menunggu tanggapan tertanggung.",
				RCLPUCLStatus:   "PUCL",
				LetterPrintDate: day(2026, time.August, 19),
				ClaimAge:        "45",
				ExpiryStatus:    "Belum Kadaluarsa",
			},
		},
	}
}
