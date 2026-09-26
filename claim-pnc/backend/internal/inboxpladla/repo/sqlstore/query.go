// Package sqlstore memenuhi seam inboxpladla.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata di modul ini, dan cacatnya menyentuh dua hal sekaligus.
// `Activity/SetDataPLADLA-Act.xml` merangkai kata kunci pencarian
// (`"and (b.claimid like '%" + TempPLA.pyID + "%')"`) DAN merangkai nama login reasuradur
// ke dalam kueri XOL. Yang pertama adalah isian pengguna; yang kedua adalah nilai yang
// menentukan data MILIK SIAPA yang terbaca.
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
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan.
func query(name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf(
			"inboxpladla/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// listColumns adalah kedua belas alias yang dikembalikan ketiga kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxpladla.sql dan dengan urutan pemindai
// di List. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go.
var listColumns = []string{
	"CLAIM_KEY", "CLAIM_NO", "POLICY_NO", "INSURED", "BUSINESS_NAME",
	"REGISTER_DATE", "LOSS_DATE", "PIC_TEKNIK", "STATUS_CODE", "STATUS_LABEL",
	"ADVICE_NO", "CLOSE_NOTE", "TOTAL_ROWS",
}

// countColumns adalah ketiga alias yang dikembalikan kueri tabel ringkas.
var countColumns = []string{"STATUS_CODE", "STATUS_LABEL", "TOTAL_ROWS"}

// listQueries adalah nama ketiga kueri daftar.
var listQueries = []string{"list_pla", "list_dla", "list_close"}

// countQueries adalah nama ketiga kueri tabel ringkas.
var countQueries = []string{"count_pla", "count_dla", "count_close"}

// reinsurerScopedQueries adalah kueri yang WAJIB menyaring menurut reasuradur pemanggil.
//
// SELURUHNYA, tanpa perkecualian. Kueri yang kehilangan penyaring itu akan menampilkan
// data satu mitra kepada mitra lain — dan di layar yang memang dibaca pihak luar, itu
// bukan cacat tampilan melainkan kebocoran.
var reinsurerScopedQueries = []string{
	"list_pla", "list_dla", "list_close",
	"count_pla", "count_dla", "count_close",
	"xol_summary",
}

// loadQueries membaca seluruh berkas .sql yang disematkan dan memecahnya per nama.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxpladla/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxpladla/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxpladla/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}

	return result
}

// splitByName memecah satu berkas .sql menjadi beberapa kueri bernama.
//
// Penanda namanya `-- name: <nama>`, dan seluruh baris komentar TIDAK ikut dikirim ke
// basis data: yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
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
			continue
		}

		body = append(body, line)
	}

	flush()
	return result
}

// escapeLike melepaskan karakter wildcard dari kata kunci yang diketik pengguna.
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return value
}
