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
	"fmt"
	"strings"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = loadQueries()

// query mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func query(name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf(
			"inboxpladlapredla/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

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

// loadQueries membaca seluruh berkas .sql yang disematkan dan memecahnya per nama.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxpladlapredla/sqlstore: tidak dapat membaca berkas kueri: " +
			err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxpladlapredla/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxpladlapredla/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}

	return result
}

// splitByName memecah satu berkas .sql menjadi beberapa kueri bernama.
//
// Penanda namanya `-- name: <nama>`, dan seluruh baris komentar TIDAK ikut dikirim ke
// basis data: yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
func splitByName(content string) map[string]string {
	const marker = "-- name:"

	// Carriage return dibuang lebih dulu: core.autocrlf=true membuat berkas .sql yang
	// sama berisi LF di satu mesin dan CRLF di mesin lain. Tanpa ini setiap baris SQL
	// berakhir `\r` yang ikut terkirim ke Oracle -- yang menerimanya sebagai spasi putih,
	// sehingga kuerinya tidak pernah gagal dan selisihnya hanya muncul saat SQL dicetak
	// ke log atau dibandingkan dengan teks yang diharapkan.
	content = strings.ReplaceAll(content, "\r\n", "\n")

	result := map[string]string{}
	name := ""
	var body []string

	flush := func() {
		if name != "" {
			if text := strings.TrimSpace(strings.Join(body, "\n")); text != "" {
				result[name] = text
			}
		}
	}

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, marker) {
			flush()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		if name == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		body = append(body, line)
	}

	flush()
	return result
}

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
