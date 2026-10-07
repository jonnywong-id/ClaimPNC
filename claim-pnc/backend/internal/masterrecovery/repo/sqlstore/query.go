// Package sqlstore memenuhi seam masterrecovery.Repo dengan SQL terhadap Oracle.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (`ADR-0030` Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL — dan di modul ini aturan itu menutup celah yang benar-benar ada di sistem
//     lama, yaitu `{ASIS:GeneratedVARecovery.NoteKomite}` pada
//     `RDB List/GetDataVAbyValidasiVA-SQL.xml`.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL, dikunci dengan namanya.
var query = sqlfile.MustLoad(queryFiles, "masterrecovery/sqlstore")

// getQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func getQuery(name string) string { return sqlfile.MustGet(query, "masterrecovery/sqlstore", name) }

// splitByName memecah isi satu berkas .sql dengan aturan yang sama seperti pemuat di atas.
func splitByName(content string) map[string]string { return sqlfile.Split(content) }
