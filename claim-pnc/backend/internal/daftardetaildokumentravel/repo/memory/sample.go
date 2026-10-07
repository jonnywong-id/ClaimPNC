package memory

import (
	_ "embed"

	"claim-pnc/internal/daftardetaildokumentravel"
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
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Satu baris dengan pembatasan plan dan jaminan, supaya grid coverage pada
// form benar-benar terisi saat dibuka di pengembangan — bukan selalu kosong.
func SampleList() []daftardetaildokumentravel.Detail {
	return sampledata.Must[[]daftardetaildokumentravel.Detail](sampleJSON, "SampleList")
}

// SampleDocumentList adalah pilihan ID Dokumen untuk pengembangan.
//
// Sama dengan isi `masterdokumentravel/repo/memory.SampleList`, dan itu disengaja —
// keduanya membaca POOLDATA.M_DOCTRAVEL yang sama. Bukan data produksi.
func SampleDocumentList() []daftardetaildokumentravel.Document {
	return sampledata.Must[[]daftardetaildokumentravel.Document](sampleJSON, "SampleDocumentList")
}

// SamplePlanList adalah pilihan Nama Plan untuk pengembangan.
//
// BUKAN data produksi. Isi POOLDATA.M_PLANTRAVEL tidak ada di export, dan tabel itu pun
// dimiliki GISFW (`D-03`) sehingga isinya memang tidak berada di tangan tim ini.
func SamplePlanList() []daftardetaildokumentravel.Plan {
	return sampledata.Must[[]daftardetaildokumentravel.Plan](sampleJSON, "SamplePlanList")
}

// SampleCoverageList adalah pilihan Nama Jaminan untuk pengembangan.
//
// BUKAN data produksi. PlanID setiap barisnya menunjuk SamplePlanList, supaya
// penyaringan jaminan menurut plan benar-benar terlihat bekerja saat dicoba.
func SampleCoverageList() []daftardetaildokumentravel.CoverageOption {
	return sampledata.Must[[]daftardetaildokumentravel.CoverageOption](sampleJSON, "SampleCoverageList")
}
