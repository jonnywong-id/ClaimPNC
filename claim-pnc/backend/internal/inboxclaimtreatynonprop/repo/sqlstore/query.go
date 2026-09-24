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
		panic(fmt.Sprintf(
			"inboxclaimtreatynonprop/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// resultColumns adalah ke-16 alias yang dikembalikan SETIAP kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxclaimtreatynonprop.sql dan dengan urutan
// pemindai scanWorkItem. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat
// diuji kesesuaiannya di query_test.go.
var resultColumns = []string{
	"REFERENCE", "CLAIM_ID", "ASSIGNED_OPERATOR",
	"MASTER_ID", "JSON_MASTER_ID", "POLICY_NUMBER", "LOSS_DATE",
	"BUSINESS_NAME", "BUSINESS_SOURCE", "CEDING_COMPANY", "INSURED_NAME",
	"STATUS", "AGING_DAYS", "CREATE_OPERATOR", "LAST_UPDATE_OPERATOR",
	"TOTAL_ROWS",
}

// listQueries adalah nama kelima kueri daftar, dipakai uji kesesuaian alias.
var listQueries = []string{
	"list_admin", "list_admin_all", "list_admin_tba", "list_admin_all_tba",
	"list_technical",
}

// adminQueries adalah keempat kueri tab Admin.
//
// Dipisah dari listQueries karena hanya keempat ini yang WAJIB memuat teks status
// "Estimation"; kueri Teknik memakai teks yang berbeda. Kesesuaiannya dengan konstanta
// domain dijaga query_test.go.
var adminQueries = []string{
	"list_admin", "list_admin_all", "list_admin_tba", "list_admin_all_tba",
}

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxclaimtreatynonprop/sqlstore: tidak dapat membaca berkas kueri: " +
			err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxclaimtreatynonprop/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxclaimtreatynonprop/sqlstore: nama kueri ganda: " + name)
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
			// Komentar kepala berkas dan komentar penjelas tiap kueri tidak ikut
			// dikirim: yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
			continue
		}

		body = append(body, line)
	}

	flush()
	return result
}
