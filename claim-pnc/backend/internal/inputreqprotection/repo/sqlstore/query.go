// Package sqlstore memenuhi seam inputreqprotection.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL (`03-CURRENT-ARCHITECTURE.md` §4.5).
//
// # Tabelnya BARU, dan kolomnya dibaca dari katalog
//
// `POOLDATA.T_CLAIM_OPENPROTECTION` dibuat Work Owner pada 2026-09-23 dan berisi 16 kolom.
// Nama kolom di sini **dibaca langsung dari `ALL_TAB_COLUMNS`**, bukan disalin dari usulan
// — usulan dan kenyataan sempat berbeda, dan yang berlaku adalah kenyataan.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inputreqprotection/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inputreqprotection/sqlstore", name) }

// splitByName memecah isi satu berkas .sql dengan aturan yang sama seperti pemuat di atas.
func splitByName(content string) map[string]string { return sqlfile.Split(content) }
