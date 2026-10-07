// Package sqlstore memenuhi seam inboxmanager.Repo dengan SQL terhadap Oracle.
//
// Tiga aturan mengikat seluruh berkas di sini, dua pertama sama dengan sqlstore modul lain:
//
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya dapat
//     dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam teks
//     SQL — dan di modul ini aturan itu menggantikan sembilan kueri Pega yang menyisipkan
//     potongan SQL lewat pola `{ASIS:…}`.
//   - Setiap penanda bind MUNCUL TEPAT SEKALI dan MENAIK. Driver mengikat argumen menurut
//     urutan kemunculan, bukan menurut nomor penandanya, sehingga penanda berulang menuntut
//     argumen berulang — dan bila lupa, kuerinya gagal ORA-01008 di Oracle sementara seluruh
//     uji memori tetap hijau. Aturan ketiga ini dikunci query_test.go.
package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxmanager/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxmanager/sqlstore", name) }
