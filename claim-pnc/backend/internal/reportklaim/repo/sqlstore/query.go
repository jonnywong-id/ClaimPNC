package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = sqlfile.MustLoad(queryFiles, "reportklaim/sqlstore", sqlfile.KeepEmpty())

// getQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func getQuery(name string) string { return sqlfile.MustGet(query, "reportklaim/sqlstore", name) }

// hasQuery menyatakan apakah sebuah kueri sudah ada di berkas .sql.
//
// # Kenapa ada, padahal getQuery sudah panik bila tidak ada
//
// Karena tidak semua ketiadaan adalah cacat pemrograman. Sebuah laporan boleh berada di
// katalog sementara kuerinya belum ditulis — dan keadaan itu harus terbaca sebagai
// PENOLAKAN YANG MENYEBUTKAN SEBABNYA, bukan sebagai panik yang menjatuhkan permintaan,
// dan bukan pula sebagai hasil kosong yang terbaca "tidak ada data".
//
// **Sejak 2026-09-25 seluruh 26 kueri sudah ada**, sehingga jalur penolakan itu tidak lagi
// terpakai. Ia tidak dihapus: yang dijaganya bukan keadaan hari ini melainkan perilaku
// ketika laporan berikutnya ditambahkan ke katalog sebelum kuerinya ditulis. Lihat
// Repo.Stream, NotPorted, dan TestKueriBelumDipindahkanDitolakDenganSebabnya.
func hasQuery(name string) bool {
	_, existing := query[name]
	return existing
}
