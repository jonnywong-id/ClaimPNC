// Package sqlstore memenuhi seam archivedokumenklaim.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis:
// `Activity/SearchDataArchiveFilling-Act.xml` menyusun SELURUH klausa WHERE-nya sebagai
// teks dari kata kunci yang diketik pengguna, lalu menempelkannya lewat
// `{ASIS:TempClaimAttach.NoteKasir}`. Satu petik tunggal sudah cukup untuk mengubah arti
// kuerinya.
//
// Larangan itu TIDAK ikut dikecualikan oleh keputusan Work Owner "replikasi apa adanya"
// atas penomoran ID: yang direplikasi adalah perilaku bisnis, bukan celah injeksi.
package sqlstore

import (
	"embed"
	"fmt"
	"strconv"
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
			"archivedokumenklaim/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// resultColumns adalah kedua puluh satu alias yang dikembalikan SETIAP kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di archivedokumenklaim.sql dan dengan urutan
// pemindai scanFile. Ia ditulis lengkap, bukan `SELECT *`: kolom disebut namanya tanpa
// perkecualian (`08-TECHNICAL-STRATEGY.md` §4.3), dan menyebutkannya di sini pula yang
// membuat ketiga tempat itu dapat diuji kesesuaiannya di query_test.go.
const resultColumns = `ARCHIVE_ID, CLAIM_NUMBER, POLICY_NUMBER, INSURED_NAME, LOSS_DATE,
       TECHNICAL_PIC, RECEIVED_DATE, INPUT_DATE, SHEET_COUNT,
       DOCUMENT_TYPE_CODE, DOCUMENT_TYPE_NAME, DOCUMENT_KIND_CODE, DOCUMENT_KIND_NAME,
       BOX_NAME, FILLING_CODE, INPUT_USER, SENT_DATE,
       GROUP_PANEL, BRANCH_STATUS, SERVICE_CODE, SERVICE_NOTE`

// listQueryParams adalah jumlah parameter yang dipakai TUBUH setiap kueri daftar.
//
// Ia dibutuhkan karena paginasi menambahkan dua parameter di belakangnya, dan nomor
// penandanya harus melanjutkan yang sudah terpakai. Menyimpannya sebagai peta — bukan
// menghitung `:n` dari teks kuerinya — membuat kesalahan terbaca saat kompilasi berupa
// nama yang tidak terdaftar, alih-alih berupa galat bind saat dijalankan.
var listQueryParams = map[string]int{
	"search_keyword":      3,
	"search_input_date":   2,
	"pending_all":         0,
	"pending_exclude_one": 1,
	"pending_exclude_two": 2,
}

// paged membungkus satu kueri daftar menjadi satu halaman hasil.
//
// # Kenapa dirangkai, bukan ditulis dua kali per kueri
//
// Kelima kueri daftar masing-masing butuh dua bentuk: penghitung seluruh baris yang
// cocok, dan pengambil satu halaman. Menuliskan keduanya di berkas .sql berarti sepuluh
// blok yang harus berubah berpasangan, dan satu yang tertinggal akan membuat jumlah di
// layar tidak sesuai dengan isinya — cacat yang tidak menghasilkan galat, hanya angka
// yang salah.
//
// # Kenapa ini BUKAN perangkaian SQL yang dilarang
//
// Yang dirangkai adalah teks kueri milik kita sendiri dari berkas .sql, seluruhnya
// konstanta saat kompilasi. Tidak ada satu pun nilai dari pengguna yang menyentuhnya —
// kata kunci, rentang tanggal, offset, dan ukuran halaman semuanya tetap lewat parameter
// binding.
//
// Urutannya menurun menurut ARCHIVE_ID: berkas terbaru lebih dulu, dan karena ID-nya naik
// monoton ia juga urutan pengarsipan. Tanpa urutan yang ditetapkan, paginasi terhadap
// hasil yang urutannya diserahkan ke basis data membuat satu baris muncul di dua halaman
// sekaligus hilang dari halaman lain.
func paged(name string) string {
	used := listParamCount(name)
	return "SELECT " + resultColumns + "\n" +
		"  FROM (" + query(name) + ")\n" +
		" ORDER BY ARCHIVE_ID DESC\n" +
		" OFFSET :" + strconv.Itoa(used+1) + " ROWS FETCH NEXT :" + strconv.Itoa(used+2) + " ROWS ONLY"
}

// counted membungkus satu kueri daftar menjadi penghitung baris.
func counted(name string) string {
	return "SELECT COUNT(*) FROM (" + query(name) + ")"
}

// listParamCount mengembalikan jumlah parameter tubuh sebuah kueri daftar.
func listParamCount(name string) int {
	count, listed := listQueryParams[name]
	if !listed {
		panic(fmt.Sprintf(
			"archivedokumenklaim/sqlstore: kueri daftar %q belum terdaftar di listQueryParams",
			name))
	}
	return count
}

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("archivedokumenklaim/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("archivedokumenklaim/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("archivedokumenklaim/sqlstore: nama kueri ganda: " + name)
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

		// Komentar kepala berkas dan komentar penjelas tiap kueri tidak ikut dikirim ke
		// basis data. Penjelasannya panjang dengan sengaja — ia yang menjaga alasan tiap
		// perbedaan terhadap kueri lama tetap terbaca di sebelah kuerinya.
		if name == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		body = append(body, line)
	}

	flush()
	return result
}
