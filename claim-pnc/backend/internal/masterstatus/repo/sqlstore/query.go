// Package sqlstore memenuhi seam masterstatus.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke
//     dalam teks SQL.
package sqlstore

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL, dikunci dengan namanya.
var query = loadAllQueries()

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func getQuery(name string) string {
	text, existing := query[name]
	if !existing {
		panic(fmt.Sprintf("masterstatus/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// loadAllQueries membaca setiap berkas .sql dan memecahnya pada penanda
// "-- name: <nama>", sehingga satu berkas dapat memuat beberapa pernyataan dan tetap
// terbaca sebagai satu kesatuan saat di-review.
func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("masterstatus/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, files := range list {
		content, err := queryFiles.ReadFile(files.Name())
		if err != nil {
			panic("masterstatus/sqlstore: tidak dapat membaca " + files.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, bentrok := result[name]; bentrok {
				panic("masterstatus/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memisahkan isi berkas menjadi pernyataan bernama, membuang baris komentar
// supaya yang dikirim ke basis data hanyalah SQL-nya.
func splitByName(content string) map[string]string {
	const penanda = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name != "" {
			if text := strings.TrimSpace(strings.Join(body, "\n")); text != "" {
				result[name] = text
			}
		}
	}
	for _, rows := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(rows)
		if strings.HasPrefix(trimmed, penanda) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, penanda))
			body = nil
			continue
		}
		if name == "" || strings.HasPrefix(trimmed, "--") {
			// Komentar kepala berkas dan komentar penjelas tiap kueri tidak ikut
			// dikirim: yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
			continue
		}
		body = append(body, rows)
	}
	save()
	return result
}
