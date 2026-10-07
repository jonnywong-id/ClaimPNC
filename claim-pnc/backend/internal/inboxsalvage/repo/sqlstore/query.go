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
	"strings"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxsalvage/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxsalvage/sqlstore", name) }

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
