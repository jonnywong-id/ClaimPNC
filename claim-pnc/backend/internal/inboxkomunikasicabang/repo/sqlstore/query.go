// Package sqlstore memenuhi seam inboxkomunikasicabang.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
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

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxkomunikasicabang/sqlstore")

// getQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func getQuery(name string) string {
	return sqlfile.MustGet(queries, "inboxkomunikasicabang/sqlstore", name)
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
