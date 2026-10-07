// Package sqlstore memenuhi seam penyimpanan modul registrasi dengan SQL.
//
// Empat aturan mengikat seluruh berkas di sini:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//   - `SELECT *` dilarang. Kolom disebut namanya.
//   - Tidak ada `DELETE` atas data bernilai bisnis (`ADR-0012`). Baris yang tidak lagi
//     terpakai DITANDAI, tidak dihapus.
//
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

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "registrasi/sqlstore", sqlfile.KeepComments())

// loadQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func loadQuery(name string) string { return sqlfile.MustGet(queries, "registrasi/sqlstore", name) }

// splitByName memecah isi satu berkas .sql dengan aturan yang sama seperti pemuat di atas.
func splitByName(content string) map[string]string {
	return sqlfile.Split(content, sqlfile.KeepComments())
}
