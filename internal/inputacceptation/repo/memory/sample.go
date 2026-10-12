package memory

import (
	"claim-pnc/internal/inputacceptation"
)

// SampleDetails adalah akseptasi contoh untuk pengembangan lokal dan pengujian.
//
// # Seluruh isinya KARANGAN, dan itu disengaja
//
// Tidak satu pun nomor polis, nama tertanggung, atau nama Ceding Co di bawah berasal dari data
// nyata. `D-69` melarang data nasabah ditulis ke berkas yang di-commit, dan larangan itu
// berlaku penuh pada data contoh — berkas contoh justru yang paling mudah tersalin ke tempat
// lain.
//
// # Yang TIDAK dikarang adalah bentuknya
//
// Isian dan grid di bawah dirakit DARI KATALOG (`section.go`), bukan diketik satu per satu.
// Akibatnya contoh ini tidak dapat tertinggal saat katalognya berubah: isian baru langsung
// ikut terisi, dan isian yang dihapus langsung hilang dari sini.
//
// Nilai yang tidak disebut `sampleValues` diisi penanda `—` supaya layar tetap menggambar
// seluruh isiannya. Sel kosong dan sel yang memang belum diisi tidak dapat dibedakan di layar,
// dan pada layar contoh perbedaan itu tidak penting.
//
// # Nomor klaimnya SAMA dengan contoh Inbox Claim Treaty Non Prop
//
// Itu bukan kebetulan. Satu-satunya pintu ke layar ini adalah nomor klaim di antrean, dan
// contoh yang nomornya tidak cocok membuat setiap tautan di layar antrean berakhir "klaim
// tidak ditemukan" saat `PENYIMPANAN=memori`. Keempatnya diambil dari
// `inboxclaimtreatynonprop/repo/memory/sample.go`:
//
//	CLMNP-1001  antrean Admin — akseptasi yang isinya lengkap
//	CLMNP-1002  antrean Admin — klaim yang ADA tetapi dokumennya kosong; ia membuktikan
//	            layar tetap terbuka dengan nomor klaim terbaca, bukan dijawab
//	            "tidak ditemukan"
//	CLMNP-2001  antrean Teknik — lengkap, supaya tab Teknik ikut dapat ditelusuri
//	CLMNP-2002  antrean Teknik — lengkap
func SampleDetails() []inputacceptation.Detail {
	lengkap := func(claimID, statusWork string) inputacceptation.Detail {
		detail, err := inputacceptation.NewDetail(
			claimID,
			"ASM-FW-GCNMFW-WORK-CLAIMTREATYNONPROP "+claimID,
			statusWork,
			"ADMINNONPROP1",
			filledValues(claimID),
			filledGrids(),
		)
		if err != nil {
			// Galat di sini berarti katalog dan perakit contoh berselisih — cacat
			// pemrograman yang harus terlihat saat aplikasi start, bukan saat pengguna
			// membuka layar.
			panic("inputacceptation/memory: contoh tidak cocok dengan katalog: " +
				err.Error())
		}
		return detail
	}

	// Klaim yang dokumennya kosong. Ia TIDAK boleh dijawab "tidak ditemukan" — lihat
	// catatan LEFT JOIN pada kuerinya.
	empty, err := inputacceptation.NewDetail(
		"CLMNP-1002",
		"ASM-FW-GCNMFW-WORK-CLAIMTREATYNONPROP CLMNP-1002",
		"New",
		"ADMINNONPROP1",
		map[string]string{},
		map[string][]inputacceptation.GridRow{},
	)
	if err != nil {
		panic("inputacceptation/memory: contoh kosong tidak cocok dengan katalog: " +
			err.Error())
	}

	return []inputacceptation.Detail{
		lengkap("CLMNP-1001", "Pending-Acceptation"),
		empty,
		lengkap("CLMNP-2001", "Pending-Acceptation"),
		lengkap("CLMNP-2002", "Pending-Acceptation"),
	}
}

// sampleValues adalah nilai contoh untuk isian yang layak terbaca sebagai kalimat.
//
// Isian yang tidak disebut di sini tetap digambar, dengan penanda `—`.
var sampleValues = map[string]string{
	"treaty_id":         "TNP-2026-01",
	"treaty_name":       "XOL Property 2026 Layer 1",
	"treaty_year":       "2026",
	"treaty_start_date": "2026-01-01",
	"treaty_end_date":   "2026-12-31",

	"policy_no":        "99.002.2026.00001",
	"policy_no_ceding": "CED/2026/000123",
	"date_of_loss":     "2026-08-14",
	"report_date":      "2026-08-16",
	"received_date":    "2026-08-20",
	"insured_name":     "PT Contoh Sejahtera",
	"reporter_name":    "Contoh Pelapor",
	"reporter_phone":   "021-5550000",
	"reporter_address": "Jalan Contoh No. 1, Jakarta",
	"pla_no_ceding":    "PLA/2026/0001",
	"claim_no_ceding":  "CLM/2026/0001",
	"dla_no_ceding":    "DLA/2026/0001",
	"policy_start":     "2026-01-01",
	"policy_end":       "2026-12-31",
	"cause_of_loss":    "Kebakaran",
	"report_status":    "Diterima",
	"report_type":      "Laporan Awal",
	"location_of_loss": "Gudang Contoh, Bekasi",
	"report_description": "Kerugian akibat kebakaran pada gudang penyimpanan. " +
		"Seluruh isi contoh ini karangan.",
	"appointed_adj":       "Adjuster Contoh",
	"consultant_name":     "Konsultan Contoh",
	"circumtances":        "Kronologi contoh untuk pengembangan lokal.",
	"supporting_document": "Berita acara, foto lokasi, laporan surveyor",

	"total_sum_insured_idr": "25.000.000.000,00",
	"insured_interest":      "Bangunan dan isinya",

	"share_ceding":        "40,0000",
	"deductible_type":     "Persentase",
	"form_type":           "Excess of Loss",
	"currency_deductible": "IDR",
	"deductible_value":    "500.000.000,00",
	"deductible_percent":  "2,5000",
	"type_deductible":     "TSI",
	"tsi_deductible":      "25.000.000.000,00",
}

// filledValues merakit seluruh isian yang tidak terhalang dari katalog.
//
// Nomor klaimnya diterima sebagai parameter, bukan diambil dari sampleValues: isian "Claim No"
// wajib sama dengan nomor klaim yang dibuka, dan contoh yang menampilkan nomor berbeda dari
// alamatnya adalah contoh yang menyesatkan.
func filledValues(claimID string) map[string]string {
	values := map[string]string{"claim_no": claimID}
	for _, field := range inputacceptation.Fields() {
		if field.Key == "claim_no" {
			continue
		}
		if field.Blocked {
			// Isian terhalang TIDAK diisi contoh. Mengisinya akan membuat layar tampak
			// berfungsi di pengembangan lalu kosong di staging — persis kesalahpahaman yang
			// penanda terhalang ada untuk mencegahnya.
			continue
		}
		if value, listed := sampleValues[field.Key]; listed {
			values[field.Key] = value
			continue
		}
		values[field.Key] = "—"
	}
	return values
}

// sampleRows adalah baris contoh tiap grid, dikunci kode grid.
//
// Nilainya disebut per KOLOM supaya pembacanya dapat mencocokkannya dengan katalog; kolom yang
// tidak disebut diisi penanda.
var sampleRows = map[string][]map[string]string{
	inputacceptation.GridInterest: {
		{"object_name": "Bangunan Gudang", "currency": "IDR",
			"value_idr": "15.000.000.000,00", "value": "15.000.000.000,00"},
		{"object_name": "Mesin dan Peralatan", "currency": "IDR",
			"value_idr": "10.000.000.000,00", "value": "10.000.000.000,00"},
	},
	inputacceptation.GridInterestTotal: {
		{"currency": "IDR", "value": "25.000.000.000,00"},
	},
	inputacceptation.GridClaimAmount: {
		{"currency": "IDR", "rate_of_exchange": "1,0000",
			"claim_amount": "3.000.000.000,00", "tpl": "0,00",
			"adjuster_fee": "25.000.000,00", "salvage": "50.000.000,00",
			"fee": "5.000.000,00", "proportion_pct": "100,0000",
			"claim_amount_idr":    "3.000.000.000,00",
			"claim_amount_cedant": "3.000.000.000,00"},
	},
	inputacceptation.GridSpreadLoss: {
		{"currency": "IDR", "treaty_name": "OR", "share_pct": "60,0000",
			"claim_amount": "1.800.000.000,00", "adjuster_fee": "15.000.000,00",
			"salvage": "30.000.000,00", "fee": "3.000.000,00", "to_xol": "false"},
		{"currency": "IDR", "treaty_name": "XOL Layer 1", "share_pct": "40,0000",
			"claim_amount": "1.200.000.000,00", "adjuster_fee": "10.000.000,00",
			"salvage": "20.000.000,00", "fee": "2.000.000,00", "to_xol": "true"},
	},
	inputacceptation.GridSpreadingRisk: {
		{"currency": "IDR", "treaty_name": "OR", "claim_amount": "1.800.000.000,00",
			"asm_share_pct": "25,0000", "claim_amount_asm": "450.000.000,00",
			"adjuster_fee": "3.750.000,00", "salvage": "7.500.000,00",
			"fee": "750.000,00"},
	},
	inputacceptation.GridTotalEstimation: {
		{"currency": "IDR", "claim_amount": "3.000.000.000,00",
			"claim_amount_asm": "450.000.000,00", "adjuster_fee": "3.750.000,00",
			"salvage": "7.500.000,00", "fee": "750.000,00"},
	},
	inputacceptation.GridReinstatement: {
		{"layer": "1", "currency": "IDR", "claim_amount": "1.200.000.000,00",
			"adjuster_fee": "10.000.000,00", "salvage": "20.000.000,00",
			"limit": "5.000.000.000,00", "premi_mdp": "150.000.000,00",
			"reinstatement_pct":         "100,0000",
			"reinstatement_premium":     "36.000.000,00",
			"reinstatement_premium_asm": "9.000.000,00"},
	},
	inputacceptation.GridSpreadingClaim: {
		{"treaty_type": "OR", "share_pct": "60,0000", "currency": "IDR",
			"claim_spreded": "1.800.000.000,00", "adjuster_fee": "15.000.000,00",
			"salvage": "30.000.000,00", "fee": "3.000.000,00"},
	},
	inputacceptation.GridAdjustment: {
		{"type": "Final", "acceptation_no": "AKS/TNP/2026/0001",
			"acceptation_date": "2026-09-25", "status": "1"},
	},
	inputacceptation.GridSpreadAdjust: {
		{"treaty_type": "OR", "share_pct": "60,0000", "currency": "IDR",
			"claim_spreaded": "1.800.000.000,00", "adjuster_fee": "15.000.000,00",
			"salvage": "30.000.000,00", "fee": "3.000.000,00",
			"total_claim":           "1.848.000.000,00",
			"reinstatement_premium": "36.000.000,00"},
	},
	inputacceptation.GridSuggest: {
		{"confirmed": "true", "name": "Contoh Petugas Ceding",
			"date": "2026-09-26", "noted": "Disetujui sesuai perhitungan."},
	},
}

// filledGrids merakit seluruh grid dari katalog.
//
// Grid yang tidak punya baris contoh tetap ADA dengan nol baris — bukan dihilangkan. Bedanya
// bermakna: grid yang ada tetapi kosong berarti "tidak ada isinya", grid yang tidak ada sama
// sekali berarti "belum dapat dibaca".
func filledGrids() map[string][]inputacceptation.GridRow {
	result := map[string][]inputacceptation.GridRow{}

	for _, grid := range inputacceptation.GridList() {
		if grid.Blocked {
			continue
		}

		rows := make([]inputacceptation.GridRow, 0, len(sampleRows[grid.Code]))
		for _, sample := range sampleRows[grid.Code] {
			row := inputacceptation.GridRow{}
			for _, column := range grid.Columns {
				if value, listed := sample[column.Key]; listed {
					row[column.Key] = value
					continue
				}
				row[column.Key] = "—"
			}
			rows = append(rows, row)
		}
		result[grid.Code] = rows
	}

	return result
}
