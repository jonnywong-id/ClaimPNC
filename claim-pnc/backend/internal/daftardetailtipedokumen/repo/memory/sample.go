package memory

import (
	_ "embed"

	"claim-pnc/internal/daftardetailtipedokumen"
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
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Satu baris TANPA lini bisnis, supaya keadaan itu benar-benar terlihat saat
// dicoba — ia sah, dan grid utama pun tetap menampilkannya utuh.
// Satu baris dengan dokumen TIDAK WAJIB dan jumlah minimum lebih dari satu,
// supaya kedua isian itu tidak selalu bernilai sama di data contoh.
// Baris yang RUJUKANNYA TIDAK ADA di master mana pun — tipe dokumen, penyebab
// kerugian, objek dokumen, dan bisnisnya semua menunjuk kode yang tidak
// terdaftar.
//
// Ia ada dengan sengaja: di Oracle baris seperti ini dihasilkan LEFT JOIN
// dengan keterangan KOSONG, dan ia harus TETAP TAMPIL supaya petugas dapat
// memperbaikinya. Kueri lama memakai INNER JOIN dan menyembunyikannya — itulah
// penyimpangan yang dicatat pada `detail_business_list`, dan baris inilah yang
// membuktikan penyimpangan itu bekerja.
func SampleList() []daftardetailtipedokumen.DetailType {
	return sampledata.Must[[]daftardetailtipedokumen.DetailType](sampleJSON, "SampleList")
}

// SampleDocumentTypeList adalah pilihan ID Tipe Dokumen untuk pengembangan.
//
// Sama dengan isi `daftartipedokumen/repo/memory.SampleList`, dan itu disengaja —
// keduanya membaca POOLDATA.V_LST_DOC_TYPE yang sama. Bukan data produksi.
func SampleDocumentTypeList() []daftardetailtipedokumen.DocumentTypeOption {
	return sampledata.Must[[]daftardetailtipedokumen.DocumentTypeOption](sampleJSON, "SampleDocumentTypeList")
}

// SampleCauseOfLossList adalah pilihan Dokumen kolom ID untuk pengembangan.
//
// Sama dengan isi `masterpenyebabkerugian/repo/memory.SampleList`. Bukan data produksi.
//
// Baris `1007` sengaja berketerangan KOSONG, mengikuti data contoh modul itu: ia
// memperlihatkan bahwa pilihan tanpa keterangan tetap dapat dipilih, dan layar harus
// menanganinya tanpa menampilkan baris yang tampak rusak.
func SampleCauseOfLossList() []daftardetailtipedokumen.CauseOfLossOption {
	return sampledata.Must[[]daftardetailtipedokumen.CauseOfLossOption](sampleJSON, "SampleCauseOfLossList")
}

// SampleObjectDocumentList adalah pilihan Objek Dokumen untuk pengembangan.
//
// Sama dengan isi `daftarobjekdokumen/repo/memory.SampleList`. Bukan data produksi.
func SampleObjectDocumentList() []daftardetailtipedokumen.ObjectDocumentOption {
	return sampledata.Must[[]daftardetailtipedokumen.ObjectDocumentOption](sampleJSON, "SampleObjectDocumentList")
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
	return sampledata.Must[[]daftardetailtipedokumen.Business](sampleJSON, "SampleBusinessList")
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
