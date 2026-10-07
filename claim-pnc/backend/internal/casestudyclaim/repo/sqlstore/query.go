// Package sqlstore memenuhi seam casestudyclaim.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata pada layar INI, bukan cacat teoretis: kueri lama
// menyisipkan DUA potongan klausa WHERE dari properti klipboard — `{ASIS:TempView.City}`
// dan `{ASIS:TempSearch.UserTeknis}` — dan kedua potongan itulah yang menentukan status
// dan lini bisnis mana yang ditampilkan.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "casestudyclaim/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "casestudyclaim/sqlstore", name) }
