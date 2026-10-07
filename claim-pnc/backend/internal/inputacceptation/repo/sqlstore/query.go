// Package sqlstore memenuhi seam inputacceptation.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inputacceptation/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inputacceptation/sqlstore", name) }

// resultColumns adalah kelima alias yang dikembalikan kueri rincian.
//
// Urutannya WAJIB sama dengan urutan kolom di inputacceptation.sql dan dengan urutan pemindai
// di inputacceptation.go. Ketiganya dijaga query_test.go.
var resultColumns = []string{
	"CLAIM_ID", "REFERENCE", "STATUS_WORK", "LAST_UPDATE_OPERATOR", "CLAIM_DOCUMENT",
}
