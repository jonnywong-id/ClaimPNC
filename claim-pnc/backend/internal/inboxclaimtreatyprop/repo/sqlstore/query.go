// Package sqlstore memenuhi seam inboxclaimtreatyprop.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis. `GetClaimTreaty_SQL` menyisipkan
// `{OperatorID.pyUserIdentifier}` langsung ke teks SQL-nya.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxclaimtreatyprop/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string {
	return sqlfile.MustGet(queries, "inboxclaimtreatyprop/sqlstore", name)
}

// splitByName memecah isi satu berkas .sql dengan aturan yang sama seperti pemuat di atas.
func splitByName(content string) map[string]string { return sqlfile.Split(content) }

// resultColumns adalah ke-15 alias yang dikembalikan SETIAP kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxclaimtreatyprop.sql dan dengan urutan
// pemindai scanWorkItem. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat
// diuji kesesuaiannya di query_test.go.
//
// Dua alias terakhir sebelum TOTAL_ROWS datang dari tabel objek kerja, bukan dari tabel
// penugasan maupun JSON_KLAIM — lihat bagian "TABEL KETIGA" di inboxclaimtreatyprop.sql.
var resultColumns = []string{
	"WORK_KEY", "REFERENCE", "CLAIM_ID", "ASSIGNED_OPERATOR",
	"MASTER_ID", "POLICY_NUMBER", "LOSS_DATE",
	"BUSINESS_NAME", "BUSINESS_SOURCE", "CEDING_COMPANY", "INSURED_NAME",
	"SUBJECTIVITY", "LAST_UPDATE_OPERATOR", "CLAIM_STATUS", "TOTAL_ROWS",
}

// listQueries adalah nama kedua kueri daftar, dipakai uji kesesuaian alias.
//
// DUA, bukan tiga: kueri "See All Claim" dihapus ketika sumbernya berpindah ke Report
// Definition — kedua RD tidak punya penyaring operator, sehingga tab Admin sendiri sudah
// tidak menyaring pemanggil dan tidak ada lagi yang perlu ditukar.
var listQueries = []string{"list_worklist", "list_workbasket"}
