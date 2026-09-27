package memory

import "claim-pnc/internal/daftartipedokumen"

// SampleList adalah isi awal untuk pengembangan dan pengujian tanpa basis data.
//
// # PERINGATAN — INI BUKAN DATA PRODUKSI
//
// Isi sebenarnya POOLDATA.LST_DOC_TYPE tidak ada di export: tidak ada berkas CSV-nya di
// `Database/` seperti halnya `v_sts_claim.csv` dan `m_portal_pnc.csv`, tidak ada satu pun
// nama tipe dokumen yang tertulis di rule Pega mana pun, dan DDL tabelnya pun belum
// diterima (`R-08`).
//
// Nama di bawah karena itu SUSUNAN SENDIRI — dipilih sekadar agar layar, pengurutan, dan
// paginasi dapat dicoba. Ia TIDAK BOLEH dipakai sebagai dasar uji kesetaraan gerbang 1,
// dan harus diganti isi tabel yang sebenarnya begitu DBA mengirimkannya.
//
// Perlakuan yang sama dipakai `masterdokumentravel/repo/memory.SampleList` dan
// `masterstatusprogres/repo/memory.SampleList`, dengan alasan yang sama.
//
// # Kenapa sebagian ProcessStatus kosong, dan sebagian tidak
//
// Itu disengaja. STS_PROSES adalah catatan teks bebas tanpa kewajiban isi (lihat kepala
// paket `daftartipedokumen`), sehingga layar harus tetap terbaca baik ketika kolom itu
// terisi maupun ketika tidak. Contoh yang seluruhnya terisi akan menyembunyikan cacat
// tampilan pada baris yang kosong — dan baris kosong justru yang paling mungkin ada di
// tabel sebenarnya.
//
// ID-nya mengikuti bentuk yang benar — kode situs "1" disambung nomor urut EMPAT digit —
// supaya penambahan pertama pada pengembangan melanjutkan deret yang masuk akal dan
// bentuk yang tampil di layar sama dengan bentuk yang kelak datang dari Oracle.
func SampleList() []daftartipedokumen.DocumentType {
	return []daftartipedokumen.DocumentType{
		{ID: "10001", Type: "Dokumen Registrasi", ProcessStatus: "Register"},
		{ID: "10002", Type: "Dokumen Survey", ProcessStatus: "Survey"},
		{ID: "10003", Type: "Dokumen Komite", ProcessStatus: "Komite"},
		{ID: "10004", Type: "Dokumen Salvage", ProcessStatus: ""},
		{ID: "10005", Type: "Dokumen Pembayaran", ProcessStatus: "Payment"},
		{ID: "10006", Type: "Dokumen Pendukung Lainnya", ProcessStatus: ""},
	}
}
