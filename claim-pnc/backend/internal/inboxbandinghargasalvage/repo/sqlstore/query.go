// Package sqlstore memenuhi seam inboxbandinghargasalvage.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya dapat
//     dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam teks
//     SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis. `Activity/SetReqSalvage_Act-Act.xml`
// langkah 5 dan 11 menyusun penyaring pencariannya sebagai
// `"and noklaim='" + Param.noklaim + "' "` lalu menyisipkannya lewat `{Asis:…}` — apa yang
// diketik pengguna, langsung ke dalam teks kueri. Larangan itu TIDAK ikut dikecualikan oleh
// `P-5`: yang direplikasi adalah perilaku bisnis, bukan celah injeksi.
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
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan masukan
// pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat saat pertama
// dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func query(name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf(
			"inboxbandinghargasalvage/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
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

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxbandinghargasalvage/sqlstore: tidak dapat membaca berkas kueri: " +
			err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxbandinghargasalvage/sqlstore: tidak dapat membaca " + entry.Name() +
				": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxbandinghargasalvage/sqlstore: nama kueri ganda: " + name)
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
