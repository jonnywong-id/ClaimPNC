package memory

import "claim-pnc/internal/masterpanel"

// Isi contoh untuk pengembangan tanpa Oracle.
//
// SELURUHNYA KARANGAN. Yang ditiru dari data nyata hanyalah BENTUKNYA: bentuk ID_PANEL,
// sebaran status, dan kelima nama lokasi yang memang ditanam di
// `Activity/SetLokasiSisiPanel-Act.xml` — supaya layar yang dicoba saat pengembangan
// berperilaku seperti layar yang dipakai di produksi.
//
// Nilai kesembilan penanda STS_* sengaja dibuat BERAGAM — ada "1", "0", dan "Y" — bukan
// seragam. Daftar nilai sahnya tidak ada di export (`R-16`), dan contoh yang seragam akan
// membuat layar tampak seolah domainnya sudah diketahui.

// SamplePanelSite adalah kode situs contoh.
//
// Di produksi ia dibaca dari `POOLDATA.M_SITE_DATABASE` dan panjangnya tidak diketahui
// (R-08). Dua digit dipilih supaya ID_PANEL yang dihasilkan menjadi delapan karakter —
// kode situs ditambah enam digit nomor urut, persis bentuk `PEGA_M_PANEL_HE.prc:21`.
const SamplePanelSite = "01"

// SamplePanelSequence adalah nomor urut terakhir yang dianggap sudah dipakai.
//
// Bukan nol, supaya ID yang diterbitkan saat pengembangan tidak tampak seperti nomor urut
// pertama dan tidak bertabrakan dengan baris contoh di bawah.
const SamplePanelSequence int64 = 3

// SampleList adalah panel contoh, satu per status persetujuan.
//
// Ketiga tab layar karena itu terisi tanpa perlu menambah apa pun lebih dulu — termasuk
// tab Reject, yang paling mudah terlupa diuji.
//
// Ketiganya sengaja punya jumlah lokasi yang BERBEDA — dua, satu, dan nol — supaya layar
// yang menggambar daftar lokasi teruji pada ketiga keadaannya, termasuk panel tanpa
// lokasi sama sekali yang merupakan keadaan yang sah.
func SampleList() []masterpanel.Panel {
	return []masterpanel.Panel{
		{
			ID:                  "01000001",
			Name:                "Pintu Depan",
			RepairStatus:        "1",
			EditQuantityStatus:  "1",
			PremiumRepairStatus: "0",
			ShatterStatus:       "0",
			StickerStatus:       "1",
			SideStatus:          "1",
			SevereDamageStatus:  "0",
			ActiveStatus:        "1",
			ExclusionC:          "0",
			ApprovalMark:        "1",
			DocumentID:          "",
			Status:              masterpanel.StatusApproved,
			Location: []masterpanel.PanelLocation{
				{Name: "KIRI", Side: masterpanel.SideLeft},
				{Name: "KANAN", Side: masterpanel.SideRight},
			},
		},
		{
			ID:                  "01000002",
			Name:                "Kaca Depan",
			RepairStatus:        "0",
			EditQuantityStatus:  "0",
			PremiumRepairStatus: "0",
			ShatterStatus:       "1",
			StickerStatus:       "0",
			SideStatus:          "-",
			SevereDamageStatus:  "1",
			ActiveStatus:        "1",
			ExclusionC:          "Y",
			ApprovalMark:        "",
			DocumentID:          "",
			Status:              masterpanel.StatusPending,
			Location: []masterpanel.PanelLocation{
				{Name: "DEPAN", Side: masterpanel.SideNone},
			},
		},
		{
			ID:                  "01000003",
			Name:                "Bumper Belakang",
			RepairStatus:        "1",
			EditQuantityStatus:  "1",
			PremiumRepairStatus: "1",
			ShatterStatus:       "0",
			StickerStatus:       "0",
			SideStatus:          "-",
			SevereDamageStatus:  "1",
			ActiveStatus:        "0",
			ExclusionC:          "0",
			ApprovalMark:        "",
			// Alasan penolakan terisi, supaya layar yang menampilkannya teruji tanpa perlu
			// menolak satu baris lebih dulu.
			RejectReason: "Nama panel bertabrakan dengan panel yang sudah ada.",
			DocumentID:   "",
			Status:       masterpanel.StatusRejected,
		},
	}
}
