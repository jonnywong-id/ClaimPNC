// Package sqlstore memenuhi seam inboxcompliance.Repo dengan SQL terhadap Oracle.
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
var queries = sqlfile.MustLoad(queryFiles, "inboxcompliance/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxcompliance/sqlstore", name) }

// Alias yang dikembalikan kueri daftar tiap tab.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxcompliance.sql dan dengan urutan
// pemindainya. Keduanya ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go.
//
// # Kenapa dua daftar, bukan satu yang dipakai bersama
//
// Karena kedua tab membaca TABEL YANG BERBEDA, bukan tabel yang sama dengan penyaring
// berbeda — tab Compliance membaca tabel warisan Pega, tab Post Audit membaca tabel datar
// `POOLDATA.T_CLAIM_COMPLIANCE_H`. Kolom yang tidak berlaku pun berbeda sifatnya: tab
// Compliance tidak punya Compliance Remarks karena tabelnya tidak menyimpannya, sedangkan
// tab Post Audit tidak punya Nama Bisnis karena tabelnya memang hanya enam kolom.
//
// Memaksakan satu daftar bersama menuntut setiap kueri menyebut kolom `NULL` untuk isian
// yang tidak berlaku baginya — cara yang dipakai modul Inbox Admin karena di sana ketujuh
// kuerinya memang membaca satu keluarga tabel yang sama. Di sini pemaksaan itu hanya akan
// menambah enam `CAST(NULL AS …)` pada dua kueri tanpa satu pun manfaat.
var (
	complianceColumns = []string{
		"CASE_ID", "REFERENCE", "POLICY_NUMBER", "INSURED_NAME",
		"BUSINESS_NAME", "BRANCH_NAME", "ADMIN_NAME", "COMPLIANCE_SENT_DATE",
	}

	postAuditColumns = []string{
		"CASE_ID", "CLAIM_NUMBER", "INSURED_NAME", "POLICY_NUMBER",
		"COMPLIANCE_REMARKS", "POST_AUDIT_SENT_DATE",
	}
)
