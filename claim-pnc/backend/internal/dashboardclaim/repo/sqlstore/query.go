// Package sqlstore memenuhi seam dashboardclaim.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata pada layar INI, bukan cacat teoretis. Keempat kueri lama
// menyisipkan potongan klausa WHERE dari properti klipboard:
//
//	GcnmBrowseCase_SQL        enam penanda {ASIS:…}, tiga di antaranya dari isian pengguna
//	BrowseLossAdjuster        {ASIS:TempView.pyNote}, {ASIS:TempSearch.Province}
//	BrowseInternalSurveyor    {ASIS:TempView.pyNote}, {ASIS:TempSearch.AlasanTerlambat}
//	Get_Count*                dua penanda masing-masing
//
// Ditambah paginasinya sendiri: `rn >= {ASIS:Pagination.FirstRow}` dan
// `rn <= {ASIS:Pagination.LastRow}` — batas halaman pun dirangkai sebagai teks.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "dashboardclaim/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "dashboardclaim/sqlstore", name) }

// scanner adalah bagian *sql.Row dan *sql.Rows yang dipakai pemindai baris.
//
// Dinyatakan supaya satu fungsi pemindai melayani keduanya, dan supaya pemindainya dapat
// diuji tanpa basis data.
type scanner interface {
	Scan(dest ...any) error
}
