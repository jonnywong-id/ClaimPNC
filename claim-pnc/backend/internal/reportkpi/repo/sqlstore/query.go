// Package sqlstore memenuhi seam reportkpi.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis — dan di modul ini KETIGA
// penyaringnya dirangkai dari isian layar. `Activity/PNCReportKPIAdjuster_act-Act.xml`
// menyusun ketiganya sebagai potongan teks:
//
//	"and adjuster='" + TempAdjComp.NameOfBank + "'"
//	"and trunc(TANGGAL)>=to_date('" + local.awal + "','dd/mm/yyyy') …"
//
// lalu menyisipkannya lewat `{ASIS:...}`. Nama adjuster datang dari dropdown, tetapi
// kedua tanggal datang dari isian yang DIKETIK pengguna.
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
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func query(name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf(
			"reportkpi/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// scoreColumns adalah kesembilan alias nilai, DALAM URUTAN KOLOM LAYAR.
//
// Urutannya WAJIB sama dengan urutan SELECT di reportkpi.sql, dengan urutan pemindai di
// reportkpi.go, dan dengan urutan reportkpi.Components(). Ketiganya diuji kesesuaiannya di
// query_test.go — dan itu bukan kerapian: satu alias yang tertukar menaruh nilai
// "TANGGAPAN KOMUNIKASI" di bawah judul "UPDATE PROGRESS", dan tidak ada satu pun galat
// yang menandakannya.
var scoreColumns = []string{
	"SURVEY", "IMMEDIATE_ADVICE", "PRELIMINARY_ADVICE", "INTERIM_REPORT",
	"PROGRESS", "COMMUNICATION", "PROPOSE", "FINAL_REPORT", "TOTAL_SCORE",
}

// summaryColumns adalah ke-11 alias yang dikembalikan kueri Summary.
var summaryColumns = append([]string{"ADJUSTER", "REPORT_TYPE"}, scoreColumns...)

// detailColumns adalah ke-14 alias yang dikembalikan kueri Detail.
//
// Ia membawa dua kolom yang TIDAK ada di Summary — `CASE_ID` dan `SCORED_ON` — karena
// barisnya satu kasus, bukan satu adjuster. Dan ia membawa `TOTAL_ROWS` karena ia
// dipaginasi; Summary tidak.
var detailColumns = append(
	append([]string{"ADJUSTER", "CASE_ID", "REPORT_TYPE", "SCORED_ON"}, scoreColumns...),
	"TOTAL_ROWS",
)

// filteredQueries adalah kueri yang penyaringnya WAJIB sama persis.
//
// Summary dan Detail menjawab pertanyaan yang sama dengan bentuk yang berbeda; satu
// penyaring yang tertinggal di salah satunya membuat rata-rata dan rinciannya tidak dapat
// dicocokkan pengguna — tanpa satu pun galat.
var filteredQueries = []string{"summary", "detail"}

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("reportkpi/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("reportkpi/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("reportkpi/sqlstore: nama kueri ganda: " + name)
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
