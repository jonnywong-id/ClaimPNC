// Package sqlstore memenuhi seam inboxmanageradmin.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua berlaku meski sumber layar ini Report Definition, bukan Connect-SQL: nilai
// `OrgUnit` di Pega ditanam section sebagai literal, dan menyalin cara itu berarti
// merangkai nilai ke dalam teks kueri.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxmanageradmin/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxmanageradmin/sqlstore", name) }

// splitByName memecah isi satu berkas .sql dengan aturan yang sama seperti pemuat di atas.
func splitByName(content string) map[string]string { return sqlfile.Split(content) }

// resultColumns adalah kesembilan alias yang dikembalikan kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxmanageradmin.sql dan dengan urutan
// pemindai scanWorkItem. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat
// diuji kesesuaiannya di query_test.go.
var resultColumns = []string{
	"REFERENCE", "CASE_ID", "POLICY_NUMBER", "INSURED_NAME",
	"BUSINESS_NAME", "BUSINESS_SOURCE", "REGISTERED_AT", "ADMIN_NAME",
	"CLAIM_STATUS",
}
