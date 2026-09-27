package memory

import "claim-pnc/internal/masterdokumentravel"

// SampleList adalah isi awal untuk pengembangan dan pengujian tanpa basis data.
//
// # PERINGATAN — INI BUKAN DATA PRODUKSI
//
// Isi sebenarnya POOLDATA.M_DOCTRAVEL tidak ada di export: tidak ada berkas CSV-nya di
// `Database/` seperti halnya `v_sts_claim.csv` dan `m_portal_pnc.csv`, tidak ada satu
// pun judul dokumen yang tertulis di rule Pega mana pun, dan DDL tabelnya pun belum
// diterima (`R-08`).
//
// Judul di bawah karena itu SUSUNAN SENDIRI — dipilih sekadar agar layar, penyaringan,
// dan pengurutan dapat dicoba. Ia TIDAK BOLEH dipakai sebagai dasar uji kesetaraan
// gerbang 1, dan harus diganti isi tabel yang sebenarnya begitu DBA mengirimkannya.
//
// Perlakuan yang sama dipakai `masterstatusprogres/repo/memory.SampleList`, dengan
// alasan yang sama.
//
// DOCID-nya mengikuti bentuk yang benar — kode situs "1" disambung nomor urut lima
// digit — supaya penambahan pertama pada pengembangan melanjutkan deret yang masuk akal
// dan bentuk yang tampil di layar sama dengan bentuk yang kelak datang dari Oracle.
func SampleList() []masterdokumentravel.TravelDocument {
	return []masterdokumentravel.TravelDocument{
		{ID: "100001", Name: "Paspor"},
		{ID: "100002", Name: "Tiket Perjalanan"},
		{ID: "100003", Name: "Boarding Pass"},
		{ID: "100004", Name: "Laporan Kehilangan Bagasi"},
		{ID: "100005", Name: "Kuitansi Biaya Pengobatan"},
		{ID: "100006", Name: "Surat Keterangan Maskapai"},
	}
}
