package memory

import "claim-pnc/internal/detailpenyebab"

// NewSampleRepo membentuk penyimpanan berisi baris contoh.
//
// # Isinya DIKARANG, dan itu dinyatakan terang-terangan
//
// Tidak ada satu pun baris `POOLDATA.D_CAUSE_OF_LOSS` di repository ini — yang diterima
// dari DBA sejauh ini hanyalah `v_sts_claim.csv`, `emailkomite.csv`, dan berkas master
// menu. Baris di bawah karena itu **bukan salinan data produksi**; ia disusun agar
// layarnya dapat dijalankan dan diuji.
//
// Dua hal yang dijaga supaya contoh ini tetap berguna dan tidak menyesatkan:
//
//  1. **Bentuknya benar.** ID mengikuti aturan `PEGA_D_CAUSE_OF_LOSS.prc:19` — kode situs
//     disambung empat digit berpadding nol — sehingga cacat penyusunan ID akan terlihat di
//     sini juga.
//  2. **Isinya tidak menyerupai data nyata.** Kode situsnya `99`, yang tidak dipakai
//     entitas mana pun, dan tidak ada nomor polis, nama tertanggung, maupun nilai uang di
//     dalamnya (`D-69`).
//
// # Keragaman yang sengaja dimasukkan
//
// Baris contohnya tidak seragam, karena keseragaman menyembunyikan cacat:
//
//   - satu baris **tidak aktif** — menguji kolom Status Aktif benar-benar terbaca
//   - satu baris ber-Status Aktif **kosong** — meniru baris yang lahir sebelum isiannya
//     ada; lihat detailpenyebab.ActiveLabel
//   - satu baris **tanpa induk** — baris yatim, yang mungkin ada karena tidak ada foreign
//     key yang diketahui (`R-08`)
//   - satu baris **tanpa lini bisnis**, dan satu baris dengan **tiga** lini bisnis
//   - satu baris **tanpa Kode Kehilangan**
//
// Kelima keadaan itu sah di sistem lama, karena jalur simpannya tidak memeriksa apa pun —
// lihat detailpenyebab.Input.Check.
func NewSampleRepo() *Repo {
	repo := NewRepo()
	repo.seed(sampleRows(), sampleMaster(), sampleBusiness())
	return repo
}

// sampleMaster adalah Master Penyebab Kerugian contoh — induk dari baris di bawah.
//
// Tabel aslinya `POOLDATA.V_M_CAUSE_OF_LOSS`, dan modul ini hanya MEMBACA-nya. Modul yang
// mengelolanya — Master Penyebab Kerugian, MENU_ID 20 `CauseOfLossInbox` — belum dibangun.
func sampleMaster() []detailpenyebab.MasterOption {
	return []detailpenyebab.MasterOption{
		{ID: "9001", Label: "Kebakaran"},
		{ID: "9002", Label: "Kecelakaan Diri"},
		{ID: "9003", Label: "Pencurian dan Perampokan"},
		{ID: "9004", Label: "Kerusakan Pengangkutan"},
		{ID: "9005", Label: "Bencana Alam"},
	}
}

// sampleBusiness adalah lini bisnis contoh.
//
// Nama-namanya mengikuti Group Panel yang tercatat di `02-BUSINESS-UNDERSTANDING.md` §1,
// supaya istilah yang muncul di layar adalah istilah yang memang dipakai petugas.
func sampleBusiness() []detailpenyebab.Business {
	return []detailpenyebab.Business{
		{ID: "002", Name: "Personal Accident"},
		{ID: "003", Name: "Aneka"},
		{ID: "004", Name: "Marine Cargo"},
		{ID: "005", Name: "Travel"},
		{ID: "006", Name: "Fire / Property"},
		{ID: "009", Name: "Aneka (varian lain)"},
	}
}

// sampleRows adalah baris Detail Penyebab Kerugian contoh.
func sampleRows() []detailpenyebab.CauseOfLossDetail {
	return []detailpenyebab.CauseOfLossDetail{
		{
			ID:          "990001",
			LegacyID:    "COL-0001",
			MasterID:    "9001",
			Description: "Kebakaran akibat hubungan arus pendek",
			LossCode:    "FIRE-01",
			Active:      detailpenyebab.ActiveYes,
			Business: []detailpenyebab.Business{
				{ID: "006", Name: "Fire / Property"},
			},
		},
		{
			ID:          "990002",
			LegacyID:    "COL-0002",
			MasterID:    "9001",
			Description: "Kebakaran akibat petir",
			LossCode:    "FIRE-02",
			Active:      detailpenyebab.ActiveYes,
			Business: []detailpenyebab.Business{
				{ID: "006", Name: "Fire / Property"},
				{ID: "003", Name: "Aneka"},
			},
		},
		{
			// Baris TANPA lini bisnis — sah, karena isiannya tidak wajib.
			ID:          "990003",
			LegacyID:    "COL-0003",
			MasterID:    "9002",
			Description: "Kecelakaan lalu lintas saat perjalanan dinas",
			LossCode:    "PA-11",
			Active:      detailpenyebab.ActiveYes,
		},
		{
			// Baris dengan TIGA lini bisnis.
			ID:          "990004",
			LegacyID:    "COL-0004",
			MasterID:    "9003",
			Description: "Pencurian dengan pemberatan",
			LossCode:    "THEFT-01",
			Active:      detailpenyebab.ActiveYes,
			Business: []detailpenyebab.Business{
				{ID: "003", Name: "Aneka"},
				{ID: "004", Name: "Marine Cargo"},
				{ID: "006", Name: "Fire / Property"},
			},
		},
		{
			// Baris TIDAK AKTIF.
			ID:          "990005",
			LegacyID:    "COL-0005",
			MasterID:    "9003",
			Description: "Kehilangan tanpa unsur pemaksaan",
			LossCode:    "THEFT-02",
			Active:      detailpenyebab.ActiveNo,
			Business: []detailpenyebab.Business{
				{ID: "003", Name: "Aneka"},
			},
		},
		{
			// Baris ber-Status Aktif KOSONG — meniru baris yang lahir sebelum isiannya ada.
			ID:          "990006",
			LegacyID:    "",
			MasterID:    "9004",
			Description: "Kerusakan kemasan selama pengangkutan laut",
			LossCode:    "MC-07",
			Active:      "",
			Business: []detailpenyebab.Business{
				{ID: "004", Name: "Marine Cargo"},
			},
		},
		{
			// Baris TANPA Kode Kehilangan.
			ID:          "990007",
			LegacyID:    "COL-0007",
			MasterID:    "9005",
			Description: "Banjir dan genangan air",
			LossCode:    "",
			Active:      detailpenyebab.ActiveYes,
			Business: []detailpenyebab.Business{
				{ID: "006", Name: "Fire / Property"},
			},
		},
		{
			// Baris YATIM — MasterID menunjuk induk yang tidak ada di sampleMaster.
			// Sebutan induknya akan tampil kosong di layar, persis seperti di Pega.
			ID:          "990008",
			LegacyID:    "COL-0008",
			MasterID:    "9099",
			Description: "Pembatalan perjalanan karena sakit mendadak",
			LossCode:    "TRV-03",
			Active:      detailpenyebab.ActiveYes,
			Business: []detailpenyebab.Business{
				{ID: "005", Name: "Travel"},
			},
		},
	}
}
