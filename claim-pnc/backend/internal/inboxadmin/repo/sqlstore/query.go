// Package sqlstore memenuhi seam inboxadmin.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis. Ketujuh kueri lama menyusun
// klausa WHERE-nya di activity lalu menyisipkannya sebagai POTONGAN SQL lewat tujuh titik
// `{ASIS:…}` — termasuk klausa paginasinya sendiri. Larangan itu TIDAK ikut dikecualikan
// oleh keputusan Work Owner "replikasi apa adanya": yang direplikasi adalah perilaku
// bisnis, bukan celah injeksi.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxadmin/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxadmin/sqlstore", name) }

// resultColumns adalah ke-29 alias yang dikembalikan SETIAP kueri tab.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxadmin.sql dan dengan urutan pemindai
// scanWorkItem. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go.
var resultColumns = []string{
	"CASE_ID", "REFERENCE", "POLICY_NUMBER", "INSURED_NAME",
	"BUSINESS_NAME", "BUSINESS_SOURCE", "BRANCH_NAME", "CLAIM_BRANCH", "CREATOR",
	"LOSS_DATE", "REPORT_DATE", "INPUT_DATE", "LOD_DATE",
	"NOTE", "CLAIM_POSITION", "CLAIM_STATUS", "LOD_STATUS",
	"REQUEST_DATE", "POLICY_BRANCH", "SURVEY_BRANCH", "TECHNICAL_PIC",
	"SURVEYOR", "SURVEY_NUMBER",
	"INBOX_DATE", "ANALYST_NOTE", "RCL_PUCL_STATUS", "LETTER_PRINT_DATE",
	"CLAIM_AGE", "EXPIRY_STATUS",
}
