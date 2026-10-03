package memory

import "claim-pnc/internal/outstandingclaim"

// SampleDetails adalah rincian contoh untuk pengembangan lokal dan pengujian.
//
// # Seluruh isinya KARANGAN, dan itu disengaja
//
// Tidak satu pun nomor polis, nama tertanggung, nama Ceding Co, atau nilai uang di bawah
// berasal dari data nyata. `D-69` melarang data nasabah ditulis ke berkas yang di-commit, dan
// larangan itu berlaku penuh pada data contoh — berkas contoh justru yang paling mudah
// tersalin ke tempat lain.
//
// Yang TIDAK dikarang adalah BENTUKNYA: nomor klaimnya memakai penanda `CLMP`, kunci
// isiannya diambil dari section.go, dan kolom gridnya mengikuti susunan grid yang sama.
//
// # Apa yang sengaja diuji oleh susunan di bawah
//
//	CLMP-1001  klaim terisi penuh — seluruh kelompok punya isi, seluruh grid punya baris
//	CLMP-1002  klaim yang dokumennya belum tersalin — hanya keadaan objek kerja yang ada,
//	           meniru gabungan LEFT JOIN yang tidak menemukan baris di JSON_KLAIM
func SampleDetails() []outstandingclaim.Detail {
	return []outstandingclaim.Detail{
		sampleFullDetail(),

		// Klaim tanpa dokumen. Ia TETAP dapat dibuka — itulah yang dijamin LEFT JOIN — dan
		// layarnya menggambar nomor serta statusnya, dengan seluruh isian kosong.
		outstandingclaim.NewDetail(
			"CLMP-1002",
			"ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1002",
			"New",
			"ADMINTREATY1",
			map[string]string{},
			map[string][]outstandingclaim.GridRow{},
		),
	}
}

// sampleFullDetail menyusun satu klaim yang seluruh bagiannya terisi.
func sampleFullDetail() outstandingclaim.Detail {
	values := map[string]string{
		// ── Treaty Information ──
		//
		// Keenam isian yang terhalang (`ri_type`, `ceding_name`, `sob_name`, `bordeaux`,
		// `bordereaux_note`, `accounting_mode`, `teritorial_scope`, `asm_share`) SENGAJA
		// tidak diisi di sini. NewDetail memang membuangnya, tetapi menuliskannya tetap
		// akan menyesatkan pembaca berkas ini: ia akan tampak seolah kedelapan isian itu
		// punya sumber.
		"id_master":         "TRP-2026-01",
		"treaty_name":       "Treaty Contoh Proporsional 2026",
		"class_of_business": "Property All Risk",
		"year_of_account":   "2026",
		"start_date_treaty": "2026-01-01",
		"end_date_treaty":   "2026-12-31",
		"treaty_group_id":   "TG-001",
		"treaty_group_name": "Property",

		// ── Claim Information ──
		"policy_no":                   "99.001.2026.00001",
		"quater":                      "3",
		"year_of_quartal":             "2026",
		"treaty_year":                 "2026",
		"policy_no_ceding":            "CED-2026-0001",
		"insured_name":                "PT Contoh Sejahtera",
		"pla_no_ceding":               "PLA-2026-0001",
		"policy_start_ceding":         "2026-01-01",
		"policy_end_ceding":           "2026-12-31",
		"date_of_loss":                "2026-08-14",
		"report_date":                 "2026-08-16",
		"received_date":               "2026-08-18",
		"reporter_name":               "Contoh Pelapor",
		"reporter_email":              "contoh.pelapor@example.invalid",
		"cause_of_loss":               "Kebakaran",
		"report_status":               "Reported",
		"report_type":                 "Email",
		"insured_relationship_others": "",
		"report_address":              "Jalan Contoh Nomor 1",
		"appointed_adj":               "Adjuster Contoh",
		"consultant_name":             "Konsultan Contoh",
		"report_description":          "Kerugian akibat kebakaran di gudang contoh.",
		"location_of_loss":            "Gudang Contoh",
		"province":                    "DKI Jakarta",
		"zip_code":                    "12345",

		// ── Insured Interest ──
		"total_sum_insured_idr": "15000000000",
		"insured_interest":      "Bangunan gudang beserta isinya",

		// ── Deductible ──
		"share_ceding":         "25",
		"deductible_type":      "true",
		"form_type":            "Percentage",
		"currency_deductible":  "IDR",
		"deductible_value":     "50000000",
		"deductible_percent":   "10",
		"type_deductible":      "Claim Amount",
		"tsi_deductible":       "15000000000",
		"net_deductible_value": "50000000",

		// ── Estimation ──
		"total_gross_estimate_idr": "2000000000",
		"total_estimasi_idr":       "500000000",
		"ibnr_idr":                 "25000000",
	}

	gridRows := map[string][]outstandingclaim.GridRow{
		outstandingclaim.GridInterest: {
			{
				"object_name":    "Bangunan Gudang Contoh",
				"currency":       "IDR",
				"kurs":           "1",
				"tsi_per_object": "10000000000",
			},
			{
				"object_name":    "Isi Gudang Contoh",
				"currency":       "IDR",
				"kurs":           "1",
				"tsi_per_object": "5000000000",
			},
		},
		outstandingclaim.GridInterestTotal: {
			{"currency": "IDR", "value": "15000000000"},
		},
		outstandingclaim.GridClaimAmount: {
			{
				"currency":            "IDR",
				"claim_amount":        "2000000000",
				"net_deductible":      "50000000",
				"claim_amount_ceding": "1950000000",
				"claim_amount_idr":    "1950000000",
			},
		},
		outstandingclaim.GridSpreadingRisk: {
			{
				"currency":         "IDR",
				"treaty_type":      "Quota Share",
				"share_percentage": "25",
				"result_claim":     "487500000",
				"ibnr":             "12500000",
				"result_claim_idr": "487500000",
			},
			{
				"currency":         "IDR",
				"treaty_type":      "Surplus",
				"share_percentage": "5",
				"result_claim":     "97500000",
				"ibnr":             "12500000",
				"result_claim_idr": "97500000",
			},
		},
		outstandingclaim.GridEstimation: {
			{
				"type_loss":          "Partial Loss",
				"estimation_date":    "2026-08-20",
				"type":               "Initial",
				"currency":           "IDR",
				"kurs":               "1",
				"gross_estimation":   "2000000000",
				"estimation_asm":     "500000000",
				"ibnr":               "25000000",
				"estimation_asm_idr": "500000000",
			},
		},
		outstandingclaim.GridEstimationTotal: {
			{"currency": "IDR", "gross_estimate": "2000000000", "estimation_asm": "500000000"},
		},
		outstandingclaim.GridSpreadingClaim: {
			{
				"treaty_type":      "Quota Share",
				"share_percentage": "25",
				"currency":         "IDR",
				"claim_spreaded":   "487500000",
			},
		},
		outstandingclaim.GridSpreadingBreak: {
			{
				"treaty_type":      "Quota Share Break",
				"share_percentage": "12.5",
				"currency":         "IDR",
				"claim_spreaded":   "243750000",
			},
		},
		outstandingclaim.GridSuggestion: {
			{
				"name":  "PICTREATY1",
				"date":  "2026-08-21",
				"noted": "Menunggu laporan adjuster.",
			},
		},

		// Grid lampiran TIDAK diisi: ia terhalang, dan NewDetail memang membuangnya.
		// Mengisinya di sini akan membuat penghalangnya tampak sudah hilang.
	}

	return outstandingclaim.NewDetail(
		"CLMP-1001",
		"ASM-FW-GCNMFW-WORK-CLAIMTREATY CLMP-1001",
		"Pending-Teknik",
		"PICTREATY1",
		values,
		gridRows,
	)
}
