// Package sqlstore memenuhi seam dokumenpenunjang.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL (`03-CURRENT-ARCHITECTURE.md` §4.5).
//
// # Tabelnya ada di basis data LAIN
//
// Ketiga tabel metadata — `T_FOLDER_STORAGE`, `T_STORAGE_IMAGE`, `GCP_IMAGE` — berada di
// skema `GENERAL` yang **tidak ada** di basis data Claim PNC, dan dicapai lewat DB link.
// Alasannya, buktinya, dan hubungannya dengan `D-25` ada di kepala berkas .sql.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "dokumenpenunjang/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "dokumenpenunjang/sqlstore", name) }

// namaKueri menyebutkan seluruh kueri modul ini, untuk dipagari uji.
//
// Didaftar tangan, bukan diambil dari map: daftar yang menghitung dirinya sendiri akan ikut
// menyusut ketika sebuah kueri terhapus, dan ujinya tetap lulus.
var namaKueri = []string{
	"folder_aplikasi",
	"catat_akses_unggah",
	"simpan_metadata",
	"dokumen_per_klaim",
	"dokumen_menurut_imageid",
}
