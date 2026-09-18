// Package sqlstore memenuhi seam penyimpanan Master Rekening dengan SQL.
//
// Dua aturan mengikat seluruh berkas di sini:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke
//     dalam teks SQL — itu tepat kegagalan yang diwarisi rule lama, yang menyusun
//     klausa WHERE-nya dari properti klipboard lewat {ASIS:...}.
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
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func getQuery(name string) string {
	text, existing := query[name]
	if !existing {
		panic(fmt.Sprintf("masterrekening/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("masterrekening/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, files := range list {
		content, err := queryFiles.ReadFile(files.Name())
		if err != nil {
			panic("masterrekening/sqlstore: tidak dapat membaca " + files.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, bentrok := result[name]; bentrok {
				panic("masterrekening/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>". Satu berkas karena
// itu dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu kesatuan saat
// di-review.
//
// Komentar yang berdiri sendiri SETELAH sebuah pernyataan — penjelasan kenapa kueri
// itu ditulis begitu — dibuang dari teks yang dikirim ke basis data. Penjelasannya
// tetap terbaca di berkas .sql tempatnya berguna, tanpa ikut terbawa ke jaringan pada
// setiap pemanggilan.
func splitByName(content string) map[string]string {
	const penanda = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		if text := strings.TrimSpace(strings.Join(stripTrailingComment(body), "\n")); text != "" {
			result[name] = text
		}
	}
	for _, rows := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(rows); strings.HasPrefix(trimmed, penanda) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, penanda))
			body = nil
			continue
		}
		body = append(body, rows)
	}
	save()
	return result
}

// stripTrailingComment membuang baris kosong dan baris komentar di ujung sebuah kueri.
//
// Komentar di TENGAH pernyataan dibiarkan — ia menjelaskan baris di sekitarnya dan
// membuangnya akan merusak pernyataan yang memuat komentar sebaris.
func stripTrailingComment(rows []string) []string {
	end := len(rows)
	for end > 0 {
		trimmed := strings.TrimSpace(rows[end-1])
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			end--
			continue
		}
		break
	}
	return rows[:end]
}
