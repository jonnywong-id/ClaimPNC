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

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "archivedokumenklaim/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "archivedokumenklaim/sqlstore", name) }

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
