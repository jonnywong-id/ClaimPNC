// Package sqlstore memenuhi seam inboxpladlapredla.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis — dan di modul ini cacatnya ada
// di ENAM tempat, seluruhnya menyentuh isian yang DIKETIK PENGGUNA.
// `Activity/GetSearchPNCList_DLA-Act.xml` menyusun potongan klausa SQL dari isi kotak
// pencarian dan kedua kotak tanggal, lalu menyisipkannya lewat pola `{ASIS:…}`:
//
//	"AND B.CLAIMID LIKE '%" + Local.NOKLAIM + "%'"
//	"and trunc(a.tgldla)>=to_date('" + PLADLA.AnalystTransferDate + "','dd/mm/yyyy') …"
//
// Satu tanda kutip yang diketik pengguna sudah cukup mengubah arti kueri di sana. Dan
// karena kotak tanggalnya pun dirangkai sebagai teks, isian kosong menghasilkan potongan
// berikut — yang ditolak Oracle, bukan diabaikan:
//
//	to_date('','dd/mm/yyyy')
package sqlstore

import (
	"embed"
	"strings"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxpladlapredla/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxpladlapredla/sqlstore", name) }

// listColumns adalah kesembilan alias yang dikembalikan ketiga kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxpladlapredla.sql dan dengan urutan
// pemindai di List. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go.
var listColumns = []string{
	"CLAIM_KEY", "CLAIM_NO", "POLICY_NO", "INSURED", "REGISTER_DATE",
	"LOSS_DATE", "PIC_TEKNIK", "ADVICE_DATE", "TOTAL_ROWS",
}

// documentColumns adalah kesebelas alias yang dikembalikan kedua kueri rincian.
//
// Keduanya mengembalikan senarai yang SAMA meski dua kolomnya hanya ada di salah satu
// tabel — yang tidak ada diisi `NULL`. Itu yang membuat satu pemindai melayani keduanya,
// dan yang membuat pergeseran kolom pada salah satunya terlihat sebagai uji yang gagal.
var documentColumns = []string{
	"ADVICE_NO", "REINSURER", "ADVICE_TYPE", "REVISION", "ADVICE_DATE",
	"SENT", "SENT_DATE", "RECEIVED_DATE", "NOTES", "EMAIL", "ACCEPTANCE_NO",
}

// printColumns adalah keenam alias yang dikembalikan kueri panel "Print Pre DLA".
//
// Senarainya BUKAN bagian dari documentColumns meski keduanya menyangkut dokumen. Panel
// ini membaca tabel lain (`T_PREDLALIST`), digabung ke kedua tabel lampiran, dan hanya
// mengambil kolom yang benar-benar digambar — menyeragamkannya dengan documentColumns
// berarti membawa lima kolom `NULL` yang tidak berguna.
var printColumns = []string{
	"ADVICE_NO", "REINSURER", "ADVICE_TYPE", "SENT_DATE", "SENT", "ATTACHMENT_KEY",
}

// writeQueries adalah kueri yang MENULIS. Sengaja didaftarkan tersendiri.
//
// Modul ini nyaris seluruhnya membaca, dan satu-satunya kueri tulis mudah bertambah tanpa
// disadari. Senarai ini dipakai query_test.go untuk memastikan tidak ada kueri tulis lain
// yang tersisip — setiap penulisan ke tabel milik Pega menyentuh `P-1` dan menuntut
// keputusan, bukan sekadar kode.
var writeQueries = []string{
	"mark_pre_dla_sent",
	"mark_advice_sent_pla",
	"mark_advice_sent_dla",
	"update_reinsurer_email",
}

// listQueries adalah nama ketiga kueri daftar, dipakai uji kesesuaian alias.
var listQueries = []string{"list_pla", "list_dla", "list_pre_dla"}

// documentQueries adalah nama kedua kueri rincian, dipakai uji kesesuaian alias.
var documentQueries = []string{"documents_pla", "documents_dla"}

// sendQueries adalah kedua kueri pembaca dokumen yang akan dikirim.
var sendQueries = []string{"advice_for_sending_pla", "advice_for_sending_dla"}

// businessFilterQueries adalah kueri yang WAJIB mengecualikan Personal Accident dan
// Travel.
//
// Kesesuaiannya dijaga query_test.go, dan itu bukan kerapian: kueri yang kehilangan
// penyaring itu akan menampilkan klaim PA dan Travel — dua lini yang layar ini memang
// tidak layani — tanpa satu pun galat.
var businessFilterQueries = []string{"list_pla", "list_dla", "list_pre_dla"}

// escapeLike melepaskan karakter wildcard dari kata kunci yang diketik pengguna.
//
// Tanpa ini, `%` yang diketik pengguna akan berperilaku sebagai wildcard SQL dan mencocoki
// seluruh baris. Ia dipasangkan dengan `ESCAPE '\'` pada kuerinya.
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return value
}
