// Package sqlstore memenuhi seam inboxxol.Repo dengan SQL terhadap Oracle.
//
// Tiga aturan mengikat seluruh berkas di sini, dua sama dengan sqlstore modul lain dan
// satu khas modul ini:
//
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//   - **Tidak ada satu pun pernyataan yang menulis.** Keputusan Work Owner 2026-09-20:
//     keempat tabel yang ditulis sistem lama tetap dimiliki Pega selama masa paralel
//     (`P-1`). Larangan ini diuji di query_test.go.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis. Di modul ini sistem lama
// merangkai bukan hanya nilai melainkan **nama tabel**: `T_PLA_XOL` dan `T_DLA_XOL`
// dipilih dengan menyusun teks `"from POOLDATA.T_PLA_XOL where TAHUN = '" + dol + "'…"`
// di properti klipboard, lalu menyisipkannya mentah dengan `{ASIS:…}`
// (`Activity/BrowseDataXOLPLADLAGenerated-Act.xml`). Di sini tabelnya dipilih dengan
// MEMILIH KUERI.
package sqlstore

import (
	"embed"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxxol/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxxol/sqlstore", name) }

// idsMarker adalah penanda di berkas .sql yang digantikan deretan placeholder.
const idsMarker = "/*:ids*/"

// expandIDs menggantikan penanda /*:ids*/ dengan deretan placeholder, dan mengembalikan
// kueri beserta seluruh argumennya dalam urutan yang benar.
//
// # Kenapa ini BUKAN perangkaian SQL yang dilarang
//
// Yang dirangkai adalah PLACEHOLDER — `:3, :4, :5` — bukan nilainya. Tidak ada satu pun
// karakter dari pengguna yang menyentuh teks SQL; seluruh kode group business tetap
// dikirim sebagai argumen terpisah. Yang dilarang `08-TECHNICAL-STRATEGY.md` §4.3 adalah
// merangkai NILAI, persis pola `{ASIS:…}` warisan yang modul ini gantikan.
//
// `leading` adalah argumen yang sudah memakai `:1` dan seterusnya di dalam berkas .sql;
// deretan placeholder melanjutkan penomorannya. Salah hitung di sini tidak menghasilkan
// galat kompilasi, karena itu query_test.go memeriksanya.
//
// Senarai kode yang KOSONG ditolak, bukan menghasilkan `IN ()`. Klausa itu tidak sah di
// Oracle, dan di sebagian dialek lain ia mengembalikan seluruh baris — dua perilaku yang
// sama-sama salah, dan yang kedua membocorkan data perjanjian lain.
func expandIDs(name string, leading []any, ids []string) (string, []any, error) {
	text := query(name)
	if !strings.Contains(text, idsMarker) {
		return "", nil, fmt.Errorf("inboxxol/sqlstore: kueri %q tidak memuat penanda %s", name, idsMarker)
	}
	if len(ids) == 0 {
		return "", nil, fmt.Errorf("inboxxol/sqlstore: kueri %q menuntut sekurangnya satu kode group business", name)
	}

	placeholders := make([]string, 0, len(ids))
	arguments := make([]any, 0, len(leading)+len(ids))
	arguments = append(arguments, leading...)
	for i, id := range ids {
		placeholders = append(placeholders, ":"+strconv.Itoa(len(leading)+1+i))
		arguments = append(arguments, id)
	}

	return strings.Replace(text, idsMarker, strings.Join(placeholders, ", "), 1), arguments, nil
}
