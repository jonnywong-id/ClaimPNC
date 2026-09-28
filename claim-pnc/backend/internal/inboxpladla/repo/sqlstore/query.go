// Package sqlstore memenuhi seam inboxpladla.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata di modul ini, dan cacatnya menyentuh dua hal sekaligus.
// `Activity/SetDataPLADLA-Act.xml` merangkai kata kunci pencarian
// (`"and (b.claimid like '%" + TempPLA.pyID + "%')"`) DAN merangkai nama login reasuradur
// ke dalam kueri XOL. Yang pertama adalah isian pengguna; yang kedua adalah nilai yang
// menentukan data MILIK SIAPA yang terbaca.
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
// saat pertama dijalankan.
func query(name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf(
			"inboxpladla/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// listColumns adalah kedua belas alias yang dikembalikan ketiga kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxpladla.sql dan dengan urutan pemindai
// di List. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go.
var listColumns = []string{
	"CLAIM_KEY", "CLAIM_NO", "POLICY_NO", "INSURED", "BUSINESS_NAME",
	"REGISTER_DATE", "LOSS_DATE", "PIC_TEKNIK", "STATUS_CODE", "STATUS_LABEL",
	"ADVICE_NO", "CLOSE_NOTE", "TOTAL_ROWS",
}

// countColumns adalah ketiga alias yang dikembalikan kueri tabel ringkas.
var countColumns = []string{"STATUS_CODE", "STATUS_LABEL", "TOTAL_ROWS"}

// adviceListQueries adalah ketiga kueri daftar PEMBERITAHUAN.
//
// Hanya ketiganya yang mengecualikan lini Personal Accident dan Travel — ketiga daftar
// komunikasi tidak. Pemisahannya di sini yang membuat uji pengecualian itu tetap dapat
// memeriksa "setiap kueri yang SEHARUSNYA mengecualikan", bukan "setiap kueri".
var adviceListQueries = []string{"list_pla", "list_dla", "list_close"}

// adviceCountQueries adalah ketiga kueri tabel ringkas daftar pemberitahuan.
var adviceCountQueries = []string{"count_pla", "count_dla", "count_close"}

// communicationListQueries adalah kedua kueri daftar KOMUNIKASI.
//
// Dua, bukan tiga: tab "Terkirim — Belum Dijawab" dan "Terkirim — Sudah Dijawab" memakai
// kueri yang sama dengan nilai bind status yang berbeda.
var communicationListQueries = []string{
	"list_komunikasi_recipient", "list_komunikasi_sender",
}

// communicationCountQueries adalah kedua kueri tabel ringkas daftar komunikasi.
var communicationCountQueries = []string{
	"count_komunikasi_recipient", "count_komunikasi_sender",
}

// listQueries adalah SELURUH kueri daftar — pemberitahuan maupun komunikasi.
var listQueries = append(
	append([]string{}, adviceListQueries...), communicationListQueries...)

// countQueries adalah SELURUH kueri tabel ringkas.
var countQueries = append(
	append([]string{}, adviceCountQueries...), communicationCountQueries...)

// detailQueries adalah kueri layar RINCIAN.
//
// `detail_reply` TIDAK ada di sini: ia satu-satunya pernyataan yang MENULIS, dan uji yang
// memeriksa bentuk kueri baca — jumlah kolom, paginasi, urutan alias — tidak berlaku
// padanya. Ia diuji tersendiri.
var detailQueries = []string{
	"detail_claim_header",
	"detail_advices_pla",
	"detail_advices_dla",
	"detail_documents",
	"detail_document_content",
	"detail_conversations",
	"detail_conversation_exists",
}

// reinsurerScopedQueries adalah kueri yang WAJIB menyaring menurut reasuradur pemanggil.
//
// SELURUHNYA, tanpa perkecualian — termasuk pernyataan yang menulis. Kueri yang kehilangan
// penyaring itu akan menampilkan data satu mitra kepada mitra lain, dan di layar yang
// memang dibaca pihak luar itu bukan cacat tampilan melainkan kebocoran.
//
// Cara masing-masing menyaring BERBEDA, dan perbedaannya disengaja:
//
//	daftar pemberitahuan  kode reasuradur dari `T_REINSURER`
//	daftar komunikasi     login pada kedua sisi percakapan
//	layar rincian         KEDUANYA — kode untuk pemberitahuan, login untuk percakapan
//
// Yang diuji karena itu bukan "memuat `T_REINSURER`" melainkan "memuat sekurang-kurangnya
// satu batas yang berangkat dari pemanggil".
var reinsurerScopedQueries = func() []string {
	all := append([]string{}, listQueries...)
	all = append(all, countQueries...)
	all = append(all, detailQueries...)
	return append(all, "xol_summary", "detail_reply")
}()

// loadQueries membaca seluruh berkas .sql yang disematkan dan memecahnya per nama.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxpladla/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxpladla/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxpladla/sqlstore: nama kueri ganda: " + name)
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
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return value
}
