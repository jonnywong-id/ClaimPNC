package memory

import "claim-pnc/internal/masterxol"

// SampleMaster adalah contoh isi Master XOL yang MENIRU BENTUK data produksi.
//
// # Kenapa bentuknya ditiru, bukan dikarang rapi
//
// Penyimpanan memori dipakai mengembangkan dan menguji layar. Bila contohnya rapi
// sementara produksinya tidak, layar akan tampak benar saat dikembangkan lalu pecah saat
// menyentuh data sebenarnya. Keempat keanehan berikut karena itu SENGAJA ikut, dan
// keempatnya nyata — diperiksa langsung ke portal ASM pada 2026-09-20:
//
//  1. **Nomor induk berlubang.** Produksi memuat 10001, 10002, lalu 10004 — 10003 pernah
//     dihapus. Layar tidak boleh mengandaikan nomornya berurutan.
//  2. **TYPEXOL kosong.** Dua dari delapan induk menyimpan NULL. Dropdown-nya harus dapat
//     menampilkan keadaan "belum dipilih" tanpa memaksanya menjadi salah satu kode.
//  3. **Lapisan tanpa isi.** Lapisan `10017` milik induk `10008` bernama kosong, limit
//     dan excess-nya NULL. Grid harus tetap terbaca.
//  4. **Total share bukan 100%.** Lapisan `10004` milik induk `10002` tidak punya satu
//     pun reasuradur, sehingga totalnya 0 — dan ia tetap tersimpan. Inilah bukti bahwa
//     aturan 100% adalah peringatan, bukan penolakan (lihat masterxol.ShareWarning).
//
// # Yang TIDAK disalin
//
// Tidak ada data nasabah di tabel ini — isinya struktur treaty dan nama perusahaan
// reasuransi, bukan nomor polis maupun nama tertanggung (`D-69`). Nama induk pun berupa
// label teknis seperti "Section 1".
func SampleMaster() []masterxol.Master {
	return []masterxol.Master{
		{
			ID:              "10001",
			Name:            "Section 1",
			Year:            "2018",
			ExchangeRate:    13500,
			Type:            masterxol.TypeProperty,
			PIC:             "MARIATRIELSA",
			CommitteeStatus: masterxol.CommitteePending,
			Committee:       "NOVERHALOMOAN",
			RemarkPIC:       "Revisi",
			RemarkCommittee: "ok",
			Business: []masterxol.Business{
				{ID: "10004", Name: "MOTOR VEHICLE"},
				{ID: "10009", Name: "ENGINEERING"},
				{ID: "10013", Name: "FIRE"},
				{ID: "10033", Name: "AVIATION HULL"},
			},
			Layer: []masterxol.Layer{
				{
					ID: "10001", Name: "Sub Layer", Limit: 1095000, Excess: 1155000,
					Reinsurer: []masterxol.Reinsurer{
						{ID: "10036309", Name: "REASURANSI NASIONAL INDONESIA", Share: 5},
						{ID: "10036322", Name: "SWISS RE", Share: 75},
						{ID: "10036329", Name: "TUGU REASURANSI INDONESIA", Share: 4},
						{ID: "10038311", Name: "REASURANSI INDONESIA UTAMA", Share: 10},
						{ID: "10038851", Name: "MASKAPAI REASURANSI INDONESIA", Share: 2},
						{ID: "10043934", Name: "REASURANSI NUSANTARA MAKMUR", Share: 4},
					},
				},
				{
					ID: "10002", Name: "Layer 1", Limit: 2750000, Excess: 2250000,
					Reinsurer: []masterxol.Reinsurer{
						{ID: "10050067", Name: "SIMAS REINSURANCE BROKERS", Share: 25},
						{ID: "10051584", Name: "WILLIS INSURANCE BROKERS CO. LTD", Share: 75},
					},
				},
				{
					ID: "10003", Name: "Layer 2", Limit: 5000000, Excess: 5000000,
					Reinsurer: []masterxol.Reinsurer{
						{ID: "10050067", Name: "SIMAS REINSURANCE BROKERS", Share: 25},
						{ID: "10051584", Name: "WILLIS INSURANCE BROKERS CO. LTD", Share: 75},
					},
				},
			},
		},
		{
			// TYPEXOL kosong, dan lapisannya tanpa satu pun reasuradur — keanehan 2 dan 4.
			ID:           "10002",
			Name:         "Section 1",
			Year:         "2017",
			ExchangeRate: 14500,
			Type:         masterxol.TypeUnknown,
			Business: []masterxol.Business{
				{ID: "10009", Name: "ENGINEERING"},
				{ID: "10012", Name: "MOTOR CYCLE"},
				{ID: "10013", Name: "FIRE"},
				{ID: "11111", Name: masterxol.TreatyInwardName},
			},
			Layer: []masterxol.Layer{
				{ID: "10004", Name: "Sub Layer", Limit: 1095000, Excess: 1155000},
				{
					ID: "10005", Name: "Main Layer", Limit: 2750000, Excess: 2250000,
					Reinsurer: []masterxol.Reinsurer{
						{ID: "10038290", Name: "SIMAS REINSURANCE BROKER", Share: 25},
						{ID: "10036359", Name: "WILLIS LIMITED", Share: 75},
					},
				},
			},
		},
		{
			// Nomor 10003 sengaja dilewati — keanehan 1.
			ID:              "10004",
			Name:            "TESTING",
			Year:            "2022",
			ExchangeRate:    14000,
			Type:            masterxol.TypeUnknown,
			PIC:             "NOVERHALOMOAN",
			CommitteeStatus: masterxol.CommitteeApproved,
			Committee:       "NOVERHALOMOAN",
			RemarkPIC:       "Pengajuan Master XOL",
			RemarkCommittee: "ok",
			Business: []masterxol.Business{
				{ID: "10012", Name: "MOTOR CYCLE"},
				{ID: "10013", Name: "FIRE"},
			},
			Layer: []masterxol.Layer{
				{
					ID: "10011", Name: "Layer 1", Limit: 1000000, Excess: 20000000,
					Reinsurer: []masterxol.Reinsurer{
						{ID: "10038693", Name: "AON REINSURANCE BROKERS INDONESIA", Share: 100},
					},
				},
			},
		},
		{
			// Lapisan tanpa nama, limit, dan excess — keanehan 3.
			ID:              "10008",
			Name:            "Section 6",
			Year:            "2016",
			ExchangeRate:    14000,
			Type:            masterxol.TypeProperty,
			PIC:             "MARIATRIELSA",
			CommitteeStatus: masterxol.CommitteePending,
			RemarkPIC:       "baru",
			Business: []masterxol.Business{
				{ID: "10006", Name: "PA"},
				{ID: "10012", Name: "MOTOR CYCLE"},
			},
			Layer: []masterxol.Layer{
				{ID: "10017"},
			},
		},
	}
}

// SampleYear meniru isi POOLDATA.M_TREATYYEAR yang dipakai dropdown Tahun.
//
// Produksi memuat 36 tahun, 1995 sampai 2030. Contoh ini memuat rentang yang jauh lebih
// pendek tetapi mencakup seluruh tahun yang benar-benar dipakai SampleMaster, supaya
// dropdown di layar pengembangan tidak pernah menampilkan tahun yang tidak ada
// pilihannya.
func SampleYear() []string {
	return []string{
		"2030", "2029", "2028", "2027", "2026", "2025", "2024", "2023",
		"2022", "2021", "2020", "2019", "2018", "2017", "2016", "2015",
	}
}

// sampleBusinessGroup meniru pilihan grup bisnis beserta nama grup treaty induknya.
//
// Kolom treatyTag adalah POOLDATA.PROPORTIONALARRG.TREATYGROUPNAME yang menentukan sebuah
// grup muncul pada Type XOL yang mana. Nilainya disalin dari hasil penelusuran ke portal
// ASM pada 2026-09-20, termasuk akibatnya yang janggal:
//
//   - Type 2 (`%PA%` / `%GA%`) hanya mengembalikan **AVIATION HULL**, dan sebabnya
//     kebetulan belaka: grup treaty induknya bernama "AVIATION & AEROSPACE", dan kata
//     AERO**SPA**CE memuat potongan "PA". PA dan GA (OTHERS) — dua grup yang justru
//     dimaksud penyaring ini — tidak muncul, karena keduanya bernaung di bawah
//     "GENERAL ACCIDENT" yang tidak memuat "PA" maupun "GA" secara berurutan.
//   - HEAVY EQUIPMENT muncul pada Type **1**, bukan Type 3, karena grup treaty induknya
//     ENGINEERING.
//
// Keduanya ditiru apa adanya (keputusan Work Owner 2026-09-20). Contoh ini dibuat supaya
// uji dapat membuktikan kejanggalan itu memang direproduksi, bukan diam-diam diperbaiki.
func sampleBusinessGroup() []businessGroupRow {
	return []businessGroupRow{
		{masterxol.Business{ID: "10004", Name: "MOTOR VEHICLE"}, "MOTOR VEHICLE"},
		{masterxol.Business{ID: "10012", Name: "MOTOR CYCLE"}, "MOTOR VEHICLE"},
		{masterxol.Business{ID: "10013", Name: "FIRE"}, "PROPERTY"},
		{masterxol.Business{ID: "10030", Name: "ASURANSI SIMAS SATELIT"}, "PROPERTY"},
		{masterxol.Business{ID: "10009", Name: "ENGINEERING"}, "ENGINEERING"},
		{masterxol.Business{ID: "10014", Name: "HEAVY EQUIPMENT"}, "ENGINEERING"},
		{masterxol.Business{ID: "10033", Name: "AVIATION HULL"}, "AVIATION & AEROSPACE"},
		{masterxol.Business{ID: "10002", Name: "MARINE CARGO"}, "MARINE CARGO"},
		{masterxol.Business{ID: "10003", Name: "MARINE HULL"}, "MARINE HULL"},
		{masterxol.Business{ID: "10006", Name: "PA"}, "GENERAL ACCIDENT"},
		{masterxol.Business{ID: "10005", Name: "GA (OTHERS)"}, "GENERAL ACCIDENT"},
		{masterxol.Business{ID: "10007", Name: "HEALTH"}, "HOSPITAL"},
		{masterxol.Business{ID: "10015", Name: "SURETY BOND"}, "SURETY BOND"},
	}
}
