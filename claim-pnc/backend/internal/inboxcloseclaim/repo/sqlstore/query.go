// Package sqlstore memenuhi seam inboxcloseclaim.Repo dan RequestRepo dengan SQL terhadap
// Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
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

	"claim-pnc/internal/platform/sqlfile"
	"claim-pnc/internal/platform/sqlvalue"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxcloseclaim/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxcloseclaim/sqlstore", name) }

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
func nilIfEmpty(value string) any { return sqlvalue.NilIfBlank(value) }
