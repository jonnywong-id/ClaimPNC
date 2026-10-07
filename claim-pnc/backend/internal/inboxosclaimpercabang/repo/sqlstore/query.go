// Package sqlstore memenuhi seam inboxosclaimpercabang.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata di layar INI, bukan cacat teoretis. Ketiga kueri sumbernya
// menyisipkan nilai langsung ke teks SQL-nya — `GetDataOutstandingperCabang` dan
// `GetDataOutstandingperCabangExport` menyisipkan `{OperatorID.pyTelephone}`, dan
// `GetProgress1Sama` menyisipkan `{TempCari.CARI1}`. Yang pertama adalah batas datanya sendiri:
// nilai yang disisipkan ke sana menentukan cabang siapa yang terlihat.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxosclaimpercabang/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string {
	return sqlfile.MustGet(queries, "inboxosclaimpercabang/sqlstore", name)
}

// splitByName memecah isi satu berkas .sql dengan aturan yang sama seperti pemuat di atas.
func splitByName(content string) map[string]string { return sqlfile.Split(content) }

// listColumns adalah ke-20 alias yang dikembalikan kueri `list`, dalam urutannya.
//
// Urutannya WAJIB sama dengan pemindai scanWorkItem. Ia ditulis lengkap di sini supaya kedua
// tempat itu dapat diuji kesesuaiannya di query_test.go.
var listColumns = []string{
	"BRANCH_NAME", "BRANCH_CODE", "BUSINESS_SOURCE", "BUSINESS_NAME",
	"POLICY_NUMBER", "INSURED_NAME", "CLAIM_NUMBER", "REGISTER_DATE", "LOSS_DATE",
	"REMARK_RECOMMENDATION", "ESTIMATION_VALUE", "LAST_PROGRESS_AT",
	"PROGRESS_STATUS_1", "PROGRESS_STATUS_2", "TECHNICAL_PIC", "PROGRESS_NOTE",
	"ADJUSTER_NAME", "CAUSE_OF_LOSS", "CHRONOLOGY", "PROGRESS_STALLED",
	"TOTAL_ROWS",
}

// exportOnlyColumns adalah alias yang HANYA dikembalikan `list_export`, dalam urutannya.
//
// Ia disebut terpisah supaya dua hal dapat diuji sekaligus: bahwa `list_export` memuat seluruh
// alias `list` sebagai AWALAN persis, dan bahwa tambahannya tepat yang disebut di sini. Tanpa
// yang kedua, menghapus satu kolom treaty tetap lolos uji awalan.
var exportOnlyColumns = []string{
	"CLAIM_KEY", "POLICY_BUSINESS_NAME",
	"RESERVE_CLAIM_FULL", "RESERVE_CLAIM_ASM", "COINSURANCE",
	"SHARE_OR", "SHARE_FACOUT", "SHARE_FACOB", "SHARE_QS",
	"SHARE_FSPL", "SHARE_SSPL", "SHARE_ER1", "SHARE_ER2",
	"SHARE_BPPDAN", "SHARE_PSRQS", "SHARE_PSRSPL", "SHARE_ORS",
	"SHARE_XL", "SHARE_PSROR", "SHARE_QSOR", "SHARE_PSS",
	"SHARE_PRGBI", "SHARE_PFRA", "SHARE_FSPLNSRI", "SHARE_PSPLNSRI",
	"SHARE_FSPLNSOR", "SHARE_PSPLNSOR", "SHARE_FACOBSRB", "SHARE_FACOBINDT",
}
