// Package sqlstore memenuhi seam penyimpanan modul registrasi dengan SQL.
//
// Empat aturan mengikat seluruh berkas di sini:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//   - `SELECT *` dilarang. Kolom disebut namanya.
//   - Tidak ada `DELETE` atas data bernilai bisnis (`ADR-0012`). Baris yang tidak lagi
//     terpakai DITANDAI, tidak dihapus.
// # Satu jebakan driver yang sudah menggigit DUA kali
//
// JANGAN menulis pasangan kutip yang MEMBENTANG ANTAR-BARIS di dalam komentar SQL —
// ganda (`"`) maupun TUNGGAL (`'`). go-ora memindai teks pernyataan untuk menemukan bind
// variable dan TIDAK melewati komentar, sehingga kutip yang terbuka di satu baris dan
// tertutup di baris lain membuat penanda bind tidak terbaca. Hasilnya ORA-00900 pada
// pernyataan yang sebenarnya sah.
//
// Kutip yang berpasangan DI DALAM SATU BARIS aman; beberapa kueri di sini memakainya dan
// berjalan normal. Yang berbahaya hanya yang membentang.
//
// Gejalanya menyesatkan: membuang satu baris komentar mana pun TIDAK memperbaikinya,
// karena yang tersisa justru kutip yang tidak berpasangan.
//
// Kali pertama (kueri parameter) penyebabnya kutip ganda, dan catatan ini semula hanya
// menyebut yang ganda. Kali kedua (kueri polis) penyebabnya kutip TUNGGAL — sebuah kalimat
// bahasa Indonesia yang mengutip pesan galat, dan tanda kutipnya jatuh di dua baris yang
// berbeda. Catatan yang hanya menyebut separuh sebab tidak mencegah kejadian kedua.
package sqlstore

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = loadAllQueries()

// loadQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func loadQuery(name string) string {
	text, ok := queries[name]
	if !ok {
		panic(fmt.Sprintf("registrasi/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("registrasi/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		body, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("registrasi/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(body)) {
			if _, conflict := result[name]; conflict {
				panic("registrasi/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

func splitByName(body string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var lines []string

	store := func() {
		if name != "" {
			if text := strings.TrimSpace(strings.Join(lines, "\n")); text != "" {
				result[name] = text
			}
		}
	}
	for _, row := range strings.Split(body, "\n") {
		if trim := strings.TrimSpace(row); strings.HasPrefix(trim, marker) {
			store()
			name = strings.TrimSpace(strings.TrimPrefix(trim, marker))
			lines = nil
			continue
		}
		lines = append(lines, row)
	}
	store()
	return result
}
