package memory

import "claim-pnc/internal/masterreas"

// SampleList adalah isi contoh untuk pengembangan tanpa Oracle.
//
// # Seluruh nama perusahaan, kode, login, dan surel di sini KARANGAN
//
// Bukan mitra reasuransi nyata, bukan kode nyata, dan bukan alamat nyata. Surelnya memakai
// domain `contoh.invalid` — ranah tingkat atas yang RFC 2606 cadangkan supaya tidak pernah
// dapat diselesaikan DNS, sehingga contoh yang tidak sengaja terkirim tidak akan sampai ke
// siapa pun.
//
// Aturannya sendiri berasal dari `D-69`: alamat surel **selalu disamarkan** di artefak yang
// di-commit. Di sini bahkan tidak ada yang disamarkan — tidak satu pun alamat nyata pernah
// masuk ke berkas ini.
//
// # Yang diwakilinya, dan kenapa persis ini
//
// Tujuh baris, dipilih supaya setiap keadaan yang menentukan perilaku layar dapat dicoba
// tanpa menyiapkan data sendiri:
//
//   - **Satu perusahaan dengan TIGA baris TYPE berbeda** (`RE-001`). Inilah bentuk yang
//     paling mudah disalahpahami di layar ini: satu nama muncul tiga kali, dengan surel yang
//     berbeda-beda, dan yang membedakannya hanya kolom Tipe. Tanpa contoh seperti ini,
//     kolom Tipe tampak seperti keterangan yang tidak berguna.
//   - **Satu perusahaan TANPA baris cadangan** (`RE-004`, hanya TYPE `2`). Ia keadaan yang
//     dicacah `reas_count_without_fallback`: dokumen berjenis lain tidak akan menemukan
//     surel tujuannya sama sekali.
//   - **Dua baris berbagi LOGIN yang sama** (`RE-002` dan `RE-005`). Keadaan yang sistem
//     lama sendiri akui mungkin terjadi — `GetPNCList_PLA1` memilih sembarang satu dengan
//     `order by reinsurerid desc fetch next 1 row only` — dan yang dicacah
//     `reas_count_shared_login`. Ia dibiarkan ada di contoh supaya layar diuji dengan data
//     yang bentuknya seperti produksi, bukan yang sudah dirapikan.
//   - **Satu baris COUNTRY kosong** (`RE-003`). Ia pasti terjadi pada baris yang lahir dari
//     `GetListDataLoginReas`, yang menyisipkan tanpa kolom COUNTRY dan TYPE sama sekali.
//
// # TYPE-nya satu karakter, dan `1` berarti cadangan
//
// Lihat masterreas.FallbackType. Nilai selain `1` di sini — `2` dan `3` — dipilih sekadar
// sebagai karakter yang berbeda; **artinya dalam bahasa bisnis tidak diketahui** (`R-16`),
// dan contoh ini tidak berpura-pura mengetahuinya.
func SampleList() []masterreas.Member {
	return []masterreas.Member{
		// Satu perusahaan, tiga jenis dokumen, tiga surel berbeda.
		{
			ReinsurerID:   "RE-001",
			ReinsurerName: "Reasuransi Nusantara Jaya",
			Login:         "ReasuransiNusantaraJaya",
			Email:         "pla.nusantara@contoh.invalid",
			Country:       "Indonesia",
			Type:          "1",
		},
		{
			ReinsurerID:   "RE-001",
			ReinsurerName: "Reasuransi Nusantara Jaya",
			Login:         "ReasuransiNusantaraJaya",
			Email:         "dla.nusantara@contoh.invalid",
			Country:       "Indonesia",
			Type:          "2",
		},
		{
			ReinsurerID:   "RE-001",
			ReinsurerName: "Reasuransi Nusantara Jaya",
			Login:         "ReasuransiNusantaraJaya",
			Email:         "xol.nusantara@contoh.invalid",
			Country:       "Indonesia",
			Type:          "3",
		},

		// Berbagi LOGIN dengan RE-005; lihat catatan di atas.
		{
			ReinsurerID:   "RE-002",
			ReinsurerName: "Andalas Re",
			Login:         "AndalasRe",
			Email:         "klaim.andalas@contoh.invalid",
			Country:       "Indonesia",
			Type:          "1",
		},

		// COUNTRY kosong — bentuk baris yang lahir dari GetListDataLoginReas.
		{
			ReinsurerID:   "RE-003",
			ReinsurerName: "Bahtera Reinsurance Ltd",
			Login:         "BahteraReinsuranceLtd",
			Email:         "notice.bahtera@contoh.invalid",
			Country:       "",
			Type:          "1",
		},

		// Tanpa baris cadangan: hanya TYPE "2".
		{
			ReinsurerID:   "RE-004",
			ReinsurerName: "Cakrawala Re Asia",
			Login:         "CakrawalaReAsia",
			Email:         "dla.cakrawala@contoh.invalid",
			Country:       "Singapura",
			Type:          "2",
		},

		// Berbagi LOGIN dengan RE-002.
		{
			ReinsurerID:   "RE-005",
			ReinsurerName: "Andalas Re Syariah",
			Login:         "AndalasRe",
			Email:         "syariah.andalas@contoh.invalid",
			Country:       "Indonesia",
			Type:          "1",
		},
	}
}
