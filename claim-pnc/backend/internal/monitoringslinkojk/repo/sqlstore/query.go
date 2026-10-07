// Package sqlstore memenuhi seam monitoringslinkojk.Repo dengan SQL terhadap Oracle.
//
// Tiga aturan mengikat seluruh berkas di sini:
//
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL — dan di modul ini itu menutup cacat NYATA, bukan cacat teoretis: kedua
//     kueri lama menerima penyaringnya sebagai potongan teks SQL yang dirangkai dari
//     isian layar lalu disisipkan mentah dengan `{ASIS:…}`.
//   - **Tidak ada satu pun pernyataan yang menulis.** Lihat kepala monitoringslinkojk.sql.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "monitoringslinkojk/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "monitoringslinkojk/sqlstore", name) }
