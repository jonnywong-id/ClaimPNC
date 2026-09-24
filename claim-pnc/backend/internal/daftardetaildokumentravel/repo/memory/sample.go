package memory

import "claim-pnc/internal/daftardetaildokumentravel"

// SampleList adalah isi awal untuk pengembangan dan pengujian tanpa basis data.
//
// # PERINGATAN — INI BUKAN DATA PRODUKSI
//
// Isi sebenarnya V_LST_DOC_TRAVEL tidak ada di export: tidak ada berkas CSV-nya di
// `Database/` seperti halnya `v_sts_claim.csv` dan `m_portal_pnc.csv`, tidak ada satu pun
// aturan dokumen yang tertulis di rule Pega mana pun, dan DDL tabelnya pun belum
// diterima (`R-08`).
//
// Isi di bawah karena itu SUSUNAN SENDIRI — dipilih sekadar agar layar, penyaringan,
// pengurutan, dan grid coverage dapat dicoba. Ia TIDAK BOLEH dipakai sebagai dasar uji
// kesetaraan gerbang 1, dan harus diganti isi tabel yang sebenarnya begitu DBA
// mengirimkannya.
//
// Perlakuan yang sama dipakai `masterdokumentravel/repo/memory.SampleList` dan
// `masterstatusprogres/repo/memory.SampleList`, dengan alasan yang sama.
//
// DOCID-nya sengaja SAMA PERSIS dengan `masterdokumentravel/repo/memory.SampleList`.
// Keduanya memang merujuk tabel yang sama, dan memakai kode yang berbeda pada data
// pengembangan akan menampilkan isian ID Dokumen yang tidak pernah cocok dengan
// daftarnya sendiri — kelas kebingungan yang tidak ada di produksi.
func SampleList() []daftardetaildokumentravel.Detail {
	return []daftardetaildokumentravel.Detail{
		{
			ID:           "00001",
			DocumentID:   "100001",
			DocumentName: "Paspor",
			Mandatory:    true,
			MinUpload:    1,
		},
		{
			ID:           "00002",
			DocumentID:   "100002",
			DocumentName: "Tiket Perjalanan",
			Mandatory:    true,
			MinUpload:    1,
		},
		{
			// Satu baris dengan pembatasan plan dan jaminan, supaya grid coverage pada
			// form benar-benar terisi saat dibuka di pengembangan — bukan selalu kosong.
			ID:           "00003",
			DocumentID:   "100004",
			DocumentName: "Laporan Kehilangan Bagasi",
			Mandatory:    false,
			MinUpload:    2,
			Coverages: []daftardetaildokumentravel.Coverage{
				{
					ID:           "00005",
					PlanID:       "TP01",
					PlanName:     "Travel Plan Silver",
					CoverageID:   "TC02",
					CoverageName: "Kehilangan Bagasi",
				},
				{
					ID:           "00006",
					PlanID:       "TP02",
					PlanName:     "Travel Plan Gold",
					CoverageID:   "TC02",
					CoverageName: "Kehilangan Bagasi",
				},
			},
		},
		{
			ID:           "00004",
			DocumentID:   "100005",
			DocumentName: "Kuitansi Biaya Pengobatan",
			Mandatory:    true,
			MinUpload:    1,
			Coverages: []daftardetaildokumentravel.Coverage{
				{
					ID:           "00007",
					PlanID:       "TP02",
					PlanName:     "Travel Plan Gold",
					CoverageID:   "TC01",
					CoverageName: "Biaya Pengobatan Darurat",
				},
			},
		},
	}
}

// SampleDocumentList adalah pilihan ID Dokumen untuk pengembangan.
//
// Sama dengan isi `masterdokumentravel/repo/memory.SampleList`, dan itu disengaja —
// keduanya membaca POOLDATA.M_DOCTRAVEL yang sama. Bukan data produksi.
func SampleDocumentList() []daftardetaildokumentravel.Document {
	return []daftardetaildokumentravel.Document{
		{ID: "100001", Name: "Paspor"},
		{ID: "100002", Name: "Tiket Perjalanan"},
		{ID: "100003", Name: "Boarding Pass"},
		{ID: "100004", Name: "Laporan Kehilangan Bagasi"},
		{ID: "100005", Name: "Kuitansi Biaya Pengobatan"},
		{ID: "100006", Name: "Surat Keterangan Maskapai"},
	}
}

// SamplePlanList adalah pilihan Nama Plan untuk pengembangan.
//
// BUKAN data produksi. Isi POOLDATA.M_PLANTRAVEL tidak ada di export, dan tabel itu pun
// dimiliki GISFW (`D-03`) sehingga isinya memang tidak berada di tangan tim ini.
func SamplePlanList() []daftardetaildokumentravel.Plan {
	return []daftardetaildokumentravel.Plan{
		{ID: "TP01", Name: "Travel Plan Silver"},
		{ID: "TP02", Name: "Travel Plan Gold"},
		{ID: "TP03", Name: "Travel Plan Platinum"},
	}
}

// SampleCoverageList adalah pilihan Nama Jaminan untuk pengembangan.
//
// BUKAN data produksi. PlanID setiap barisnya menunjuk SamplePlanList, supaya
// penyaringan jaminan menurut plan benar-benar terlihat bekerja saat dicoba.
func SampleCoverageList() []daftardetaildokumentravel.CoverageOption {
	return []daftardetaildokumentravel.CoverageOption{
		{ID: "TC01", Name: "Biaya Pengobatan Darurat", PlanID: "TP01"},
		{ID: "TC02", Name: "Kehilangan Bagasi", PlanID: "TP01"},
		{ID: "TC01", Name: "Biaya Pengobatan Darurat", PlanID: "TP02"},
		{ID: "TC02", Name: "Kehilangan Bagasi", PlanID: "TP02"},
		{ID: "TC03", Name: "Keterlambatan Penerbangan", PlanID: "TP02"},
		{ID: "TC03", Name: "Keterlambatan Penerbangan", PlanID: "TP03"},
		{ID: "TC04", Name: "Pembatalan Perjalanan", PlanID: "TP03"},
	}
}
