package memory

import "claim-pnc/internal/masterpenyebabkerugian"

// SampleList adalah daftar contoh untuk pengembangan tanpa basis data.
//
// ============================================================================
// ISI DI BAWAH INI **BUKAN** ISI MASTER YANG SEBENARNYA.
// ============================================================================
//
// Ini berbeda dari SampleList milik modul Master Status Klaim, yang nilainya disalin apa
// adanya dari `Database/v_sts_claim.csv` yang diserahkan Work Owner. Untuk
// `M_CAUSE_OF_LOSS` **tidak ada artefak setara**: export tidak memuat isinya, dan tidak
// ada satu pun CSV master untuk tabel ini.
//
// Deskripsi di bawah karena itu DIKARANG sebagai bahan mencoba layar — bukan sebagai
// pernyataan tentang penyebab kerugian apa yang benar-benar dipakai Asuransi Sinar Mas.
// Jangan mengutipnya ke dokumen mana pun, dan jangan memakainya sebagai dasar uji
// kesetaraan.
//
// Isi yang sebenarnya perlu diminta ke DBA dengan satu kueri:
//
//	SELECT M_COL_ID, OLD_M_COL_ID, COL_DESC
//	  FROM POOLDATA.V_M_CAUSE_OF_LOSS
//	 ORDER BY M_COL_ID;
//
// Permintaan itu dicatat sebagai hal terbuka di `claim-pnc/docs/catatan-pengembangan.md`.
// Begitu hasilnya diterima, berkas ini diganti isinya dan komentar ini dicabut.
//
// # Yang TIDAK dikarang: bentuk ID-nya
//
// Bentuk `1` + tiga digit diturunkan dari dua bukti, bukan dari selera:
//
//   - `Database/PEGA_M_CAUSE_OF_LOSS.prc:20` membentuknya
//     `id_site || lpad(to_char(M_CAUSE_SEQ.nextval), 3, '0')`.
//   - Kode situsnya `1` terbaca dari ID penyebab kerugian yang dikutip aturan duplikasi
//     klaim PA, `12002` — berbentuk situs `1` ditambah empat digit pada tingkat rincian
//     (`docs/Steering/05-DOMAIN-MODEL.md` §2 invarian I-8).
//
// Satu baris sengaja diberi ID LAMA dan satu baris sengaja diberi deskripsi KOSONG.
// Keduanya bukan hiasan: yang pertama membuktikan kolom "ID lama" benar-benar tampil dan
// membedakan diri dari yang kosong, dan yang kedua membuktikan deskripsi kosong memang
// DITERIMA — keputusan Work Owner 2026-09-20 yang tidak dapat dicoba tanpa satu baris
// seperti itu.
func SampleList() []masterpenyebabkerugian.CauseOfLoss {
	return []masterpenyebabkerugian.CauseOfLoss{
		{ID: "1001", LegacyID: "01", Description: "Contoh Golongan A"},
		{ID: "1002", LegacyID: "02", Description: "Contoh Golongan B"},
		{ID: "1003", Description: "Contoh Golongan C"},
		{ID: "1004", Description: "Contoh Golongan D"},
		{ID: "1005", Description: "Contoh Golongan E"},
		{ID: "1006", Description: "Contoh Golongan F"},
		{ID: "1007", Description: ""},
		{ID: "1008", Description: "Contoh Golongan H"},
		{ID: "1009", Description: "Contoh Golongan I"},
		// Sepuluh baris dipilih dengan sengaja supaya urutannya menembus dua digit —
		// itulah yang membuktikan pengurutan benar-benar bekerja, dan bukan kebetulan
		// karena semua nomornya masih satu digit.
		{ID: "1010", Description: "Contoh Golongan J"},
	}
}
