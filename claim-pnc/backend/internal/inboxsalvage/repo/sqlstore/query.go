// Package sqlstore memenuhi seam inboxsalvage.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis — dan di modul ini cacatnya ada
// di LIMA tempat sekaligus, seluruhnya menyentuh isian yang DIKETIK PENGGUNA.
// `Activity/SetDataSalavage_act-Act.xml` langkah 7 sampai 11 menyusun potongan klausa SQL
// dari isi kotak pencarian, lalu menyisipkannya ke dalam kueri lewat pola `{ASIS:…}`:
//
//	"and a.NOKLAIM like '%" + TempCariData.NoKTP + "%'"
//	"and (a.noklaim = '" + TempCariData.Province + "' or a.pic = '…')"
//
// Satu tanda kutip yang diketik pengguna sudah cukup mengubah arti kueri di sana.
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
			"inboxsalvage/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// claimColumns adalah kelima alias yang dikembalikan kueri keluarga A.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxsalvage.sql dan dengan urutan pemindai
// scanClaimRow. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go.
var claimColumns = []string{
	"CLAIM_NO", "PIC", "BUSINESS_NAME", "LOSS_DATE", "TOTAL_ROWS",
}

// claimObjectColumns adalah kelima alias yang dikembalikan kueri keluarga B.
//
// Ia BERBEDA dari claimColumns pada kolom keempat: keluarga B menggambar nama objek, bukan
// tanggal kejadian. Keduanya dipisah alih-alih dijadikan satu kueri berkolom enam, karena
// kolom yang tidak pernah digambar tetap menambah biaya sub-kueri per baris.
var claimObjectColumns = []string{
	"CLAIM_NO", "PIC", "BUSINESS_NAME", "OBJECT_NAME", "TOTAL_ROWS",
}

// salvageColumns adalah ke-16 alias yang dikembalikan kueri keluarga C.
var salvageColumns = []string{
	"SALVAGE_ID", "CLAIM_NO", "INPUT_DATE", "PIC", "SALVAGE_TYPE",
	"SALVAGE_LOCATION", "QUANTITY", "ESTIMATE_VALUE", "EMAIL", "REMARK",
	"ACCEPTANCE_NO", "TRANSFER_STATUS", "ACCEPTED_VALUE", "REQUEST_VALUE", "REQUEST_NOTE",
	"TOTAL_ROWS",
}

// listQueries adalah nama keenam kueri daftar, dipakai uji kesesuaian alias.
var listQueries = []string{
	"list_claim", "list_claim_search",
	"list_claim_object", "list_claim_object_search",
	"list_claim_buyback", "list_claim_buyback_search",
}

// businessFilterQueries adalah kueri yang WAJIB memuat penyaring lini bisnis.
//
// Kesesuaiannya dijaga query_test.go, dan itu bukan kerapian: kueri yang kehilangan
// penyaring itu akan menampilkan klaim Personal Accident dan Travel — lini yang tidak
// mengenal salvage sama sekali — tanpa satu pun galat.
var businessFilterQueries = []string{
	"list_claim", "list_claim_search",
	"list_claim_object", "list_claim_object_search",
	"list_claim_buyback", "list_claim_buyback_search",
	"count_claim_status", "count_claim_buyback",
}

// escapeLike melepaskan karakter wildcard yang DIKETIK pengguna.
//
// Tanpa ini, pengguna yang mengetik `%` pada kotak pencarian tab Checker akan mencocokkan
// SELURUH baris — padahal kueri lama membandingkan dengan `=`, yang memperlakukan `%`
// sebagai huruf biasa.
//
// Urutannya penting: garis miring terbalik dilepaskan LEBIH DULU. Membaliknya akan
// melepaskan garis miring yang baru saja disisipkan, dan menghasilkan pola yang salah.
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return value
}

// patternNever adalah pola yang TIDAK PERNAH cocok dengan baris mana pun.
//
// Ia dipakai mematikan cabang pencarian PIC pada tab yang hanya mencari nomor klaim.
// Bentuknya sengaja memuat karakter yang tidak dapat muncul di kolom mana pun dan tidak
// mengandung wildcard sama sekali.
const patternNever = `~~tidak-pernah-cocok~~`

// patternAll adalah pola yang cocok dengan baris mana pun yang nilainya TIDAK kosong.
const patternAll = "%"

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxsalvage/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxsalvage/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxsalvage/sqlstore: nama kueri ganda: " + name)
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
