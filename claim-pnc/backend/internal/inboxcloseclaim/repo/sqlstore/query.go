// Package sqlstore memenuhi seam inboxcloseclaim.Repo dan RequestRepo dengan SQL terhadap
// Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya dapat
//     dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata pada layar INI, bukan cacat teoretis: kueri lama
// menyisipkan ENAM potongan klausa WHERE dari properti klipboard sekaligus —
// `{ASIS:TempView.pyNote}`, `{ASIS:TempFilter.CaseID}`, `.City`, `.CityID`, `.District`,
// `.DistrictID` — dan tiga di antaranya dirangkai langsung dari isian yang diketik pengguna
// (`Activity/GCNMGetManagerReopenCase_Act-Act.xml`).
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
		panic(fmt.Sprintf("inboxcloseclaim/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}
	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxcloseclaim/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxcloseclaim/sqlstore: tidak dapat membaca " + entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxcloseclaim/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memisahkan isi berkas menjadi pernyataan bernama, membuang baris komentar
// supaya yang dikirim ke basis data hanyalah SQL-nya.
//
// Komentar BLOK `/*…*/` tidak dibuang — penanda /*CLAIMS*/ justru salah satunya, dan ia
// harus sampai ke expandClaims untuk diganti.
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

// scanner adalah bagian *sql.Row dan *sql.Rows yang dipakai pemindai baris.
//
// Dinyatakan supaya satu fungsi pemindai melayani keduanya, dan supaya pemindainya dapat
// diuji tanpa basis data.
type scanner interface {
	Scan(dest ...any) error
}

// nilIfEmpty mengubah string kosong menjadi NULL.
//
// Dipakai dua tempat dengan alasan yang berbeda, dan keduanya nyata:
//
//   - pada penyaring, supaya pola "NULL berarti tidak menyaring" pada SQL bekerja;
//   - pada penyimpanan, supaya kolom yang memang tidak diisi tersimpan sebagai NULL dan
//     bukan sebagai teks kosong — keduanya terlihat sama di layar, tetapi hanya yang
//     pertama yang terbaca sebagai "tidak diisi" oleh kueri mana pun yang membacanya kelak.
func nilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
