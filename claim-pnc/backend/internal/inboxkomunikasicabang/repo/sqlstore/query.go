// Package sqlstore memenuhi seam inboxkomunikasicabang.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya dapat
//     dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam teks
//     SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis. `PNCCountKomunikasiCabang_Act`
// merangkai batas cabangnya sebagai TEKS —
//
//	" (COMMUNICATE_TO = '"+Local.IDCABANG+"' OR COMMUNICATE_FROM = '"+Local.IDCABANG+"') "
//
// — lalu menyisipkannya ke enam kueri lewat `{ASIS:TempView.CaseID}`. Nilainya memang berasal
// dari basis data dan bukan dari isian pengguna, tetapi polanya sama persis dengan yang
// `08-TECHNICAL-STRATEGY.md` §4.3 larang tanpa perkecualian.
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

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan masukan
// pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat saat pertama
// dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func getQuery(name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf(
			"inboxkomunikasicabang/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// listColumns adalah ke-11 alias yang dikembalikan KEDUA kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxkomunikasicabang.sql dan dengan urutan
// pemindai scanConversation. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat
// diuji kesesuaiannya di query_test.go.
var listColumns = []string{
	"CONVERSATION_ID", "CREATED_AT", "ORIGIN_CODE", "SENDER_OPERATOR", "MESSAGE",
	"REPLY_MESSAGE", "REPLIER_NAME", "RECIPIENT_CODE", "STATUS", "REPLIED_AT",
	"TOTAL_ROWS",
}

// threadColumns adalah ketujuh alias yang dikembalikan kueri utas layar detail.
//
// Ia BERBEDA dari listColumns, dan perbedaannya bukan kelalaian — layar detail tidak
// menggambar nomor percakapan (ia judulnya), tidak menggambar kode tujuan, dan tidak
// menggambar status. Ketiganya tidak dipilih.
var threadColumns = []string{"CREATED_AT", "SENDER_OPERATOR", "MESSAGE"}

// headerColumns adalah kedua alias kueri kepala percakapan.
var headerColumns = []string{"ORIGIN_CODE", "RECIPIENT_CODE"}

// attachmentColumns adalah kelima alias yang dikembalikan kueri lampiran.
var attachmentColumns = []string{
	"DOCUMENT_ID", "TYPE_NAME", "DETAIL_NAME", "NOTE", "UPLOADED_AT",
}

// listQueries adalah nama kedua kueri daftar, dipakai uji kesesuaian alias.
var listQueries = []string{"list_not_answered", "list_answered"}

// countQueries adalah nama kedua kueri pencacah.
//
// Dipisah dari listQueries karena hanya keduanya yang wajib memeriksa DUA kolom balasan —
// dan justru selisih itu yang paling mudah "dirapikan" oleh pembaca berikutnya. Uji alias
// menjaganya.
var countQueries = []string{"count_answered", "count_not_answered"}

// branchFilteredQueries adalah seluruh kueri yang WAJIB membawa batas cabang.
//
// Ia didaftar di satu tempat supaya uji dapat memeriksa SETIAP satunya, bukan hanya yang
// kebetulan teringat. Kueri yang lupa menyaring cabang tidak menghasilkan satu pun galat —
// ia hanya menampilkan percakapan cabang lain, dan itu kelas cacat yang `R-20` catat.
//
// KEDUA PERNYATAAN TULIS ikut di sini, dan justru merekalah yang paling menuntutnya: batas
// yang hilang pada pembacaan menampilkan percakapan cabang lain, sementara batas yang hilang
// pada penulisan MENGUBAHNYA — dan balasan yang telanjur tersimpan di percakapan cabang lain
// tidak dapat ditarik kembali lewat layar mana pun.
var branchFilteredQueries = []string{
	"list_not_answered", "list_answered",
	"count_answered", "count_not_answered",
	"detail_header", "detail_thread", "detail_attachments",
	"reply_update", "finish_update",
}

// writeQueries adalah pernyataan yang MENGUBAH data.
//
// Ia didaftar terpisah karena dua aturan hanya berlaku padanya: tabelnya tidak boleh diberi
// alias (PostgreSQL menolak awalan alias pada klausa SET, `D-20`), dan seluruhnya wajib
// menyaring kanal percakapan supaya percakapan yang sudah ditutup tidak dapat diubah lagi.
//
// `reply_history_insert` TIDAK di sini meski ia menulis: ia INSERT murni tanpa klausa WHERE,
// sehingga tidak ada yang dapat disaring. Yang menjaganya adalah transaksi — ia hanya ikut
// tersimpan bila reply_update mengenai satu baris yang lolos batas cabang.
var writeQueries = []string{"reply_update", "finish_update"}

// insertQueries adalah pernyataan yang MENYISIPKAN baris.
//
// Ia terpisah dari writeQueries karena aturannya berbeda: INSERT tidak punya klausa WHERE,
// sehingga tidak ada batas cabang maupun kanal yang dapat disaring padanya. Yang menjaganya
// adalah TRANSAKSI dan nilai yang disisipkan — bukan penyaring.
//
// Didaftar supaya uji dapat memastikan tidak satu pun di antaranya merangkai nilai ke dalam
// teksnya sendiri, dan supaya INSERT yang ditambahkan kelak ikut terperiksa.
var insertQueries = []string{
	"reply_history_insert", "message_insert", "message_history_insert",
}

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxkomunikasicabang/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxkomunikasicabang/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxkomunikasicabang/sqlstore: nama kueri ganda: " + name)
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
