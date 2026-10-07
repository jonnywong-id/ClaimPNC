package memory

import (
	_ "embed"

	"claim-pnc/internal/masterdokumentravel"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

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
	return sampledata.Must[[]masterdokumentravel.TravelDocument](sampleJSON, "SampleList")
}
