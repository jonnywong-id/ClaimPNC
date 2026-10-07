package sqlstore

import (
	"embed"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = sqlfile.MustLoad(queryFiles, "inboxlaporanklaim/sqlstore")

// getQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func getQuery(name string) string { return sqlfile.MustGet(query, "inboxlaporanklaim/sqlstore", name) }

// sourceName adalah nama fragmen WITH yang dipakai bersama seluruh kueri pembaca.
const sourceName = "claim_report_source"

// sourced menyambung fragmen sumber dengan salah satu badan kueri.
//
// # Kenapa disambung, bukan ditulis utuh sembilan kali
//
// Fragmen sumbernya lima puluh baris dan memuat SELURUH aturan gabungan kedua tabel —
// termasuk cara Position, accepted, dan rejected diturunkan. Menuliskannya ulang di
// setiap badan berarti enam salinan yang harus berubah bersamaan, dan satu yang
// tertinggal akan membuat satu tab menyaring atas dasar yang berbeda dari tab lain.
// Cacat seperti itu tidak menghasilkan galat; ia menghasilkan daftar yang tampak wajar.
//
// # Kenapa ini BUKAN perangkaian SQL yang dilarang
//
// Yang disambung adalah dua teks dari berkas `.sql` milik kita sendiri, keduanya
// konstanta saat kompilasi. Tidak ada satu pun nilai dari pengguna yang menyentuhnya —
// seluruh nilai tetap lewat parameter binding. Yang dilarang `08-TECHNICAL-STRATEGY.md`
// §4.3 adalah merangkai NILAI ke dalam teks SQL, persis pola `{ASIS:…}` warisan.
func sourced(bodyName string) string {
	return getQuery(sourceName) + "\n" + getQuery(bodyName)
}
