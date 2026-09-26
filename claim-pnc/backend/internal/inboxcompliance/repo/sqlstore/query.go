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
	"fmt"
	"strings"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = loadQueries()

// query mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat saat
// pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func query(name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf("inboxcompliance/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

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

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxcompliance/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxcompliance/sqlstore: tidak dapat membaca " + entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxcompliance/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}

	return result
}

// splitByName memisahkan isi berkas menjadi pernyataan bernama, membuang baris komentar
// supaya yang dikirim ke basis data hanyalah SQL-nya.
func splitByName(content string) map[string]string {
	const marker = "-- name:"

	result := map[string]string{}
	name := ""
	var body []string

	flush := func() {
		if name != "" {
			if text := strings.TrimSpace(strings.Join(body, "\n")); text != "" {
				result[name] = text
			}
		}
	}

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, marker) {
			flush()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		if name == "" || strings.HasPrefix(trimmed, "--") {
			// Komentar kepala berkas dan komentar penjelas tiap kueri tidak ikut dikirim:
			// yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
			continue
		}

		body = append(body, line)
	}

	flush()
	return result
}
