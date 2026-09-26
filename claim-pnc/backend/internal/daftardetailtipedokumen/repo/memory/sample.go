package memory

import "claim-pnc/internal/daftardetailtipedokumen"

// SampleList adalah isi awal untuk pengembangan dan pengujian tanpa basis data.
//
// # PERINGATAN — INI BUKAN DATA PRODUKSI
//
// Isi sebenarnya V_LST_DET_TYPE_DOC tidak ada di export: tidak ada berkas CSV-nya di
// `Database/` seperti halnya `v_sts_claim.csv` dan `m_portal_pnc.csv`, tidak satu pun
// rincian dokumen tertulis di rule Pega mana pun, dan DDL tabelnya pun belum diterima
// (`R-08`).
//
// Isi di bawah karena itu SUSUNAN SENDIRI — dipilih sekadar agar layar, penyaringan,
// pengurutan, dan grid bisnisnya dapat dicoba. Ia TIDAK BOLEH dipakai sebagai dasar uji
// kesetaraan gerbang 1, dan harus diganti isi tabel yang sebenarnya begitu DBA
// mengirimkannya.
//
// Perlakuan yang sama dipakai `daftarobjekdokumen`, `daftardetaildokumentravel`, dan
// `masterstatusprogres`, dengan alasan yang sama.
//
// # Kode rujukannya sengaja SAMA dengan data contoh modul lain
//
//	DOC_TYPE_ID  -> daftartipedokumen/repo/memory.SampleList          10001..10006
//	DOC_COL_ID   -> masterpenyebabkerugian/repo/memory.SampleList     1001..1010
//	OBJ_DOC      -> daftarobjekdokumen/repo/memory.SampleList         10001..10004
//	DFT_BISNIS_ID-> SampleBusinessList di bawah                       002..006
//
// Ketiganya memang merujuk tabel yang sama di produksi, dan memakai kode yang berbeda
// pada data pengembangan akan menampilkan isian yang tidak pernah cocok dengan daftarnya
// sendiri — kelas kebingungan yang tidak ada di produksi.
//
// Kedua keterangan yang TERSIMPAN — `CauseOfLossDescription` dan
// `ObjectDocumentDescription` — diisi di sini, karena di Oracle pun keduanya kolom pada
// barisnya sendiri.
//
// Yang sengaja TIDAK diisi hanyalah `DocumentTypeName` dan nama bisnis pada baris anak:
// keduanya hasil join, dan di adapter memori diisikan `Repo.withReferences`.
// Menuliskannya di sini akan menyembunyikan cacat bila pencariannya kelak rusak.
func SampleList() []daftardetailtipedokumen.DetailType {
	return []daftardetailtipedokumen.DetailType{
		{
			ID:                        "100001",
			DocumentTypeID:            "10001",
			Detail:                    "Formulir Laporan Kerugian",
			InsuredStatus:             "Tertanggung",
			CauseOfLossID:             "1001",
			CauseOfLossDescription:    "Contoh Golongan A",
			ObjectDocumentID:          "10002",
			ObjectDocumentDescription: "Polis Asli",
			Risk:                      "0",
			Businesses: []daftardetailtipedokumen.BusinessRule{
				{BusinessID: "003", Mandatory: true, MinDocument: 1},
				{BusinessID: "006", Mandatory: true, MinDocument: 1},
			},
		},
		{
			ID:                        "100002",
			DocumentTypeID:            "10001",
			Detail:                    "Fotokopi KTP Tertanggung",
			InsuredStatus:             "Tertanggung",
			CauseOfLossID:             "1002",
			CauseOfLossDescription:    "Contoh Golongan B",
			ObjectDocumentID:          "10001",
			ObjectDocumentDescription: "KTP Tertanggung",
			Risk:                      "0",
			Businesses: []daftardetailtipedokumen.BusinessRule{
				{BusinessID: "002", Mandatory: true, MinDocument: 1},
			},
		},
		{
			// Satu baris TANPA lini bisnis, supaya keadaan itu benar-benar terlihat saat
			// dicoba — ia sah, dan grid utama pun tetap menampilkannya utuh.
			ID:                        "100003",
			DocumentTypeID:            "10002",
			Detail:                    "Foto Kerusakan",
			InsuredStatus:             "",
			CauseOfLossID:             "1003",
			CauseOfLossDescription:    "Contoh Golongan C",
			ObjectDocumentID:          "",
			ObjectDocumentDescription: "",
			Risk:                      "",
		},
		{
			// Satu baris dengan dokumen TIDAK WAJIB dan jumlah minimum lebih dari satu,
			// supaya kedua isian itu tidak selalu bernilai sama di data contoh.
			ID:                        "100004",
			DocumentTypeID:            "10002",
			Detail:                    "Surat Keterangan Dokter",
			InsuredStatus:             "Tertanggung",
			CauseOfLossID:             "1004",
			CauseOfLossDescription:    "Contoh Golongan D",
			ObjectDocumentID:          "10003",
			ObjectDocumentDescription: "Surat Keterangan Dokter",
			Risk:                      "1",
			Businesses: []daftardetailtipedokumen.BusinessRule{
				{BusinessID: "002", Mandatory: false, MinDocument: 2},
				{BusinessID: "005", Mandatory: true, MinDocument: 1},
			},
		},
		{
			// Baris yang RUJUKANNYA TIDAK ADA di master mana pun — tipe dokumen, penyebab
			// kerugian, objek dokumen, dan bisnisnya semua menunjuk kode yang tidak
			// terdaftar.
			//
			// Ia ada dengan sengaja: di Oracle baris seperti ini dihasilkan LEFT JOIN
			// dengan keterangan KOSONG, dan ia harus TETAP TAMPIL supaya petugas dapat
			// memperbaikinya. Kueri lama memakai INNER JOIN dan menyembunyikannya — itulah
			// penyimpangan yang dicatat pada `detail_business_list`, dan baris inilah yang
			// membuktikan penyimpangan itu bekerja.
			ID:                        "100005",
			DocumentTypeID:            "19999",
			Detail:                    "Dokumen Warisan Tanpa Master",
			InsuredStatus:             "",
			CauseOfLossID:             "1999",
			CauseOfLossDescription:    "Keterangan Warisan Tanpa Master",
			ObjectDocumentID:          "19999",
			ObjectDocumentDescription: "Objek Warisan Tanpa Master",
			Risk:                      "0",
			Businesses: []daftardetailtipedokumen.BusinessRule{
				{BusinessID: "099", Mandatory: true, MinDocument: 1},
			},
		},
	}
}

// SampleDocumentTypeList adalah pilihan ID Tipe Dokumen untuk pengembangan.
//
// Sama dengan isi `daftartipedokumen/repo/memory.SampleList`, dan itu disengaja —
// keduanya membaca POOLDATA.V_LST_DOC_TYPE yang sama. Bukan data produksi.
func SampleDocumentTypeList() []daftardetailtipedokumen.DocumentTypeOption {
	return []daftardetailtipedokumen.DocumentTypeOption{
		{ID: "10001", Name: "Dokumen Registrasi"},
		{ID: "10002", Name: "Dokumen Survey"},
		{ID: "10003", Name: "Dokumen Komite"},
		{ID: "10004", Name: "Dokumen Salvage"},
		{ID: "10005", Name: "Dokumen Pembayaran"},
		{ID: "10006", Name: "Dokumen Pendukung Lainnya"},
	}
}

// SampleCauseOfLossList adalah pilihan Dokumen kolom ID untuk pengembangan.
//
// Sama dengan isi `masterpenyebabkerugian/repo/memory.SampleList`. Bukan data produksi.
//
// Baris `1007` sengaja berketerangan KOSONG, mengikuti data contoh modul itu: ia
// memperlihatkan bahwa pilihan tanpa keterangan tetap dapat dipilih, dan layar harus
// menanganinya tanpa menampilkan baris yang tampak rusak.
func SampleCauseOfLossList() []daftardetailtipedokumen.CauseOfLossOption {
	return []daftardetailtipedokumen.CauseOfLossOption{
		{ID: "1001", Description: "Contoh Golongan A"},
		{ID: "1002", Description: "Contoh Golongan B"},
		{ID: "1003", Description: "Contoh Golongan C"},
		{ID: "1004", Description: "Contoh Golongan D"},
		{ID: "1005", Description: "Contoh Golongan E"},
		{ID: "1006", Description: "Contoh Golongan F"},
		{ID: "1007", Description: ""},
		{ID: "1008", Description: "Contoh Golongan H"},
		{ID: "1009", Description: "Contoh Golongan I"},
		{ID: "1010", Description: "Contoh Golongan J"},
	}
}

// SampleObjectDocumentList adalah pilihan Objek Dokumen untuk pengembangan.
//
// Sama dengan isi `daftarobjekdokumen/repo/memory.SampleList`. Bukan data produksi.
func SampleObjectDocumentList() []daftardetailtipedokumen.ObjectDocumentOption {
	return []daftardetailtipedokumen.ObjectDocumentOption{
		{ID: "10001", Description: "KTP Tertanggung"},
		{ID: "10002", Description: "Polis Asli"},
		{ID: "10003", Description: "Surat Keterangan Dokter"},
		{ID: "10004", Description: "Bill of Lading"},
	}
}

// SampleBusinessList adalah pilihan ID Bisnis untuk pengembangan.
//
// Sama dengan isi `daftarobjekdokumen/repo/memory.SampleBusinessList` dan
// `mastercolsimasonline/repo/memory.SampleBusinessList` — ketiganya membaca
// POOLDATA.BUSINESS yang sama, tabel milik GISFW (`D-03`).
//
// Kodenya mengikuti Group Panel yang terbaca di `CONTEXT.md`, bukan dikarang: `002`
// Personal Accident, `003` Aneka, `004` Marine Cargo, `005` Travel, `006` Fire/Property.
// Bukan data produksi — isi POOLDATA.BUSINESS tidak ada di export.
func SampleBusinessList() []daftardetailtipedokumen.Business {
	return []daftardetailtipedokumen.Business{
		{ID: "002", Name: "PERSONAL ACCIDENT"},
		{ID: "003", Name: "ANEKA"},
		{ID: "004", Name: "MARINE CARGO"},
		{ID: "005", Name: "TRAVEL"},
		{ID: "006", Name: "FIRE / PROPERTY"},
	}
}

// NewSampleReferenceRepo membentuk pembaca master rujukan berisi keempat daftar contoh.
//
// Ia ada supaya perakitan di cmd dan di uji tidak perlu menyebut keempat daftar itu satu
// per satu — dan supaya keempatnya tidak pernah terpasang setengah.
func NewSampleReferenceRepo() *ReferenceRepo {
	return NewReferenceRepo(
		SampleDocumentTypeList(),
		SampleCauseOfLossList(),
		SampleObjectDocumentList(),
		SampleBusinessList(),
	)
}
