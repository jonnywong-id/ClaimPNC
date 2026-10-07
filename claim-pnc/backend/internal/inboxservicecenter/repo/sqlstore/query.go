// Package sqlstore memenuhi seam inboxservicecenter.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis. Kueri lama menyusun klausa WHERE-nya
// di activity lalu menyisipkannya sebagai POTONGAN SQL lewat tiga titik `{ASIS:...}` —
// termasuk kata kunci yang diketik pengguna. Larangan itu TIDAK ikut dikecualikan oleh `P-5`:
// yang direplikasi adalah perilaku bisnis, bukan celah injeksi.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxservicecenter/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxservicecenter/sqlstore", name) }

// resultColumns adalah ke-13 kolom yang dikembalikan list_claims.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxservicecenter.sql dan dengan urutan
// pemindai scanClaim. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go.
var resultColumns = []string{
	"ID", "REPAIRID", "CLAIMNO", "NOPOLIS", "QQNAME", "TYPE", "PIC",
	"INPUTDATE", "IMEI", "STS_APPROVAL", "STATUS", "LOGIN", "KOMITEAPPROVE",
}
