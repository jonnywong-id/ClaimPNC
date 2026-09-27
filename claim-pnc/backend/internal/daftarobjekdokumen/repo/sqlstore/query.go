package sqlstore

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat saat
// pertama dijalankan — bukan menjadi galat runtime yang terlihat pengguna.
func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf("daftarobjekdokumen/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("daftarobjekdokumen/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("daftarobjekdokumen/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("daftarobjekdokumen/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
//
// Komentar dibuang, bukan dibiarkan, karena isinya panjang dan seluruhnya ikut terkirim
// setiap kali kueri dijalankan — dan ikut muncul di jejak basis data saat DBA menelusuri
// kueri yang lambat.
func splitByName(content string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		var statement []string
		for _, row := range body {
			if strings.HasPrefix(strings.TrimSpace(row), "--") {
				continue
			}
			statement = append(statement, row)
		}
		if text := strings.TrimSpace(strings.Join(statement, "\n")); text != "" {
			result[name] = text
		}
	}

	for _, row := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(row); strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		body = append(body, row)
	}
	save()
	return result
}
