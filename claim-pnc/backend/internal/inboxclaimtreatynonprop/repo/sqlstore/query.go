// Package sqlstore memenuhi seam inboxclaimtreatynonprop.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis. `GetKlaimNonPropAdmin_SQL`
// menyisipkan `{Inputdata.CARI10}` langsung ke teks SQL-nya, dan `GetWorkCNP_Act` bahkan
// merangkai potongan klausa `WHERE` dari string.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxclaimtreatynonprop/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string {
	return sqlfile.MustGet(queries, "inboxclaimtreatynonprop/sqlstore", name)
}

// splitByName memecah isi satu berkas .sql dengan aturan yang sama seperti pemuat di atas.
func splitByName(content string) map[string]string { return sqlfile.Split(content) }

// resultColumns adalah ke-15 alias yang dikembalikan SETIAP kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxclaimtreatynonprop.sql dan dengan urutan
// pemindai scanWorkItem. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat
// diuji kesesuaiannya di query_test.go.
var resultColumns = []string{
	"REFERENCE", "CLAIM_ID", "ASSIGNED_OPERATOR",
	"MASTER_ID", "JSON_MASTER_ID", "POLICY_NUMBER", "LOSS_DATE",
	"BUSINESS_NAME", "BUSINESS_SOURCE", "CEDING_COMPANY", "INSURED_NAME",
	"WORK_CREATED_AT", "CREATE_OPERATOR", "LAST_UPDATE_OPERATOR",
	"TOTAL_ROWS",
}

// listQueries adalah nama kelima kueri daftar, dipakai uji kesesuaian alias.
var listQueries = []string{
	"list_admin", "list_admin_all", "list_admin_tba", "list_admin_all_tba",
	"list_technical",
}

// adminQueries adalah keempat kueri tab Admin.
//
// Dipisah dari listQueries karena hanya keempat ini yang membaca ID Master dari blob JSON;
// kueri Teknik memakai kolom objek kerja. Kesesuaiannya dijaga query_test.go.
var adminQueries = []string{
	"list_admin", "list_admin_all", "list_admin_tba", "list_admin_all_tba",
}
