// Package sqlstore memenuhi seam inboxbandinghargasalvage.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis. `Activity/SetReqSalvage_Act-Act.xml`
// langkah 5 dan 11 menyusun penyaring pencariannya sebagai
// `"and noklaim='" + Param.noklaim + "' "` lalu menyisipkannya lewat `{Asis:…}` — apa yang
// diketik pengguna, langsung ke dalam teks kueri. Larangan itu TIDAK ikut dikecualikan oleh
// `P-5`: yang direplikasi adalah perilaku bisnis, bukan celah injeksi.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxbandinghargasalvage/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string {
	return sqlfile.MustGet(queries, "inboxbandinghargasalvage/sqlstore", name)
}

// requestColumns adalah ke-11 kolom yang dikembalikan list_request.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxbandinghargasalvage.sql dan dengan urutan
// pemindai scanRequestRow. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat
// diuji kesesuaiannya di query_test.go.
var requestColumns = []string{
	"CLAIM_NO", "REQUEST_DATE", "DETAIL_OBJECT", "ITEM_NAME",
	"ITEM_PRICE", "REQUEST_PRICE", "REQUEST_NOTE", "AGING_DAYS",
	"CHECKER_NOTE", "SALVAGE_ID", "COMMITTEE_NAME",
}

// historyColumns adalah keempat kolom yang dikembalikan list_history.
var historyColumns = []string{
	"CLAIM_NO", "SALVAGE_TYPE", "SALVAGE_LOCATION", "PIC",
}

// decisionColumns adalah ketujuh kolom yang dikembalikan list_decisions.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxbandinghargasalvage.sql dan dengan urutan
// pemindai scanDecision.
var decisionColumns = []string{
	"APPROVED_AT", "DETAIL_OBJECT", "ITEM_NAME", "ITEM_PRICE",
	"REQUEST_PRICE", "DECISION_STATUS", "COMMITTEE_NAME",
}
