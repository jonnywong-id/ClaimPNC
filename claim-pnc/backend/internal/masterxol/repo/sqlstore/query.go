// Package sqlstore memenuhi seam masterxol.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL — dan di modul ini aturan itu menutup dua celah nyata sekaligus:
//     pernyataan UPDATE yang dirangkai dari catatan pengguna di
//     `Activity/UpdateStatusMasterKomitexol-Act.xml`, dan potongan klausa WHERE yang
//     disisipkan lewat `{Asis:}` di `RDB List/GetDataBisnisXol_Sql-SQL.xml`.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL, dikunci dengan namanya.
var query = sqlfile.MustLoad(queryFiles, "masterxol/sqlstore")

// getQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func getQuery(name string) string { return sqlfile.MustGet(query, "masterxol/sqlstore", name) }
