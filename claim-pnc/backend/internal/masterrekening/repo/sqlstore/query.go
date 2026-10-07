// Package sqlstore memenuhi seam penyimpanan Master Rekening dengan SQL.
//
// Dua aturan mengikat seluruh berkas di sini:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke
//     dalam teks SQL — itu tepat kegagalan yang diwarisi rule lama, yang menyusun
//     klausa WHERE-nya dari properti klipboard lewat {ASIS:...}.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = sqlfile.MustLoad(queryFiles, "masterrekening/sqlstore", sqlfile.KeepComments(), sqlfile.StripTrailingComments())

// getQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func getQuery(name string) string { return sqlfile.MustGet(query, "masterrekening/sqlstore", name) }
