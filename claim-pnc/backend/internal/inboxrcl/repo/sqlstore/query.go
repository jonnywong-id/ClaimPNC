// Package sqlstore memenuhi seam inboxrcl.Repo dengan SQL terhadap Oracle.
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
var queries = sqlfile.MustLoad(queryFiles, "inboxrcl/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxrcl/sqlstore", name) }

// taskColumns adalah ke-11 alias yang dikembalikan kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxrcl.sql dan dengan urutan pemindai
// scanTask. Ketiganya dijaga query_test.go — satu kolom yang bergeser akan memindahkan nomor
// polis ke kolom nama tertanggung tanpa menghasilkan galat apa pun.
var taskColumns = []string{
	"REFERENCE", "CASE_ID", "POLICY_NUMBER", "INSURED_NAME", "SENT_TO_RCL_AT",
	"ANALYST_NOTE", "RCL_DOCTOR", "REGISTERED_AT", "PROCESS_STATUS",
	"ASSIGNED_OPERATOR", "TOTAL_ROWS",
}
