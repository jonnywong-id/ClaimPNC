// Package sqlstore memenuhi seam inboxsurvey.Repo dan inboxsurvey.Directory dengan SQL
// terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya dapat
//     dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam teks
//     SQL — termasuk daftar nama surveyor, yang dikirim sebagai SATU teks berpembatas dan
//     diperiksa `INSTR`, bukan dirangkai menjadi klausa `IN`.
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
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan masukan
// pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat saat pertama
// dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func query(name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf(
			"inboxsurvey/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// taskColumns adalah ke-16 alias yang dikembalikan kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxsurvey.sql dan dengan urutan pemindai
// scanTask. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go — satu kolom yang bergeser akan memindahkan nomor polis ke
// kolom nama tertanggung tanpa menghasilkan galat apa pun.
//
// `APPOINTMENT_NUMBER` dan `REFERENCE_NUMBER` TIDAK ada di sini: kolom asalnya belum ada di
// `POOLDATA.T_SURVEYORLIST`. Keduanya tetap menjadi field pada inboxsurvey.SurveyTask supaya
// kolomnya tetap tergambar di layar sebagai isian yang belum terbawa — lihat kepala
// inboxsurvey.sql bagian C.
var taskColumns = []string{
	"SURVEY_ID", "CLAIM_ID", "SURVEY_INDEX", "REFERENCE_NUMBER",
	"CLAIM_NUMBER", "POLICY_NUMBER", "INSURED_NAME", "CLASS_OF_BUSINESS", "CAUSE_OF_LOSS",
	"LOCATION", "TECHNICAL_PIC", "ADJUSTER_PIC", "DATE_OF_LOSS", "CREATED_AT",
	"ASM_STATUS", "SURVEYOR_TYPE", "TOTAL_ROWS",
}

// countColumnsFull adalah alias `count_tabs_full`, dalam urutan tampil KETUJUH tab.
//
// Urutannya WAJIB sama dengan inboxsurvey.Tabs(). Satu kolom yang bergeser akan menukar jumlah
// tab Invoice dengan tab Close — dua angka yang sama-sama masuk akal.
var countColumnsFull = []string{
	"COUNT_OUTSTANDING", "COUNT_INVOICE", "COUNT_CLOSE", "COUNT_ALL",
	"COUNT_NOT_ANSWERED", "COUNT_NOT_REPLIED", "COUNT_REPLIED",
}

// countColumns adalah alias kueri penghitung tab, dalam urutan tampil tab.
//
// HANYA tab yang dapat dihitung ada di sini — keempat tab yang bergantung pada kolom adjuster
// tidak dihitung sama sekali, karena angka nol akan terbaca sebagai "tab ini kosong" alih-alih
// "tab ini belum dapat dihitung".
//
// Urutannya WAJIB sama dengan urutan tab tersedia pada inboxsurvey.Tabs(). Satu kolom yang
// bergeser akan menukar jumlah tab "belum dibalas ASM" dengan "sudah dibalas ASM" — dua angka
// yang sama-sama masuk akal, sehingga tertukarnya tidak akan disadari siapa pun.
var countColumns = []string{
	"COUNT_NOT_ANSWERED", "COUNT_NOT_REPLIED", "COUNT_REPLIED",
}

// kpiColumns adalah kesepuluh alias kueri KPI.
//
// Kesembilan angkanya berpasangan satu-satu dengan judul kapital pada
// `Section/InboxSurvey_section-Section.xml`; urutannya mengikuti urutan kolom di sana.
// kpiKeyColumns adalah kelima kolom KUNCI di depan setiap kueri KPI.
//
// Kolom yang tidak berlaku pada sebuah bentuk diisi NULL di SQL, bukan dihilangkan — kelima
// kueri karena itu berbentuk sama dan dibaca scanKPI yang satu.
var kpiKeyColumns = []string{"GROUP_KEY", "STATUS_KEY", "QUARTER_KEY", "MONTH_KEY", "CASE_KEY"}

var kpiColumns = []string{
	"GROUP_KEY", "STATUS_KEY", "QUARTER_KEY", "MONTH_KEY", "CASE_KEY", "SURVEY_SCHEDULING", "IMMEDIATE_ADVICE", "PRELIMINARY_ADVICE",
	"INTERIM_REPORT", "PROGRESS_UPDATE", "COMMUNICATION_RESPONSE", "PROPOSE_ADJUSTMENT",
	"FINAL_REPORT", "VALUE_SCORE",
}

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxsurvey/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxsurvey/sqlstore: tidak dapat membaca " + entry.Name() +
				": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxsurvey/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}

	return result
}

// splitByName memisahkan isi berkas menjadi pernyataan bernama, membuang baris komentar
// supaya yang dikirim ke basis data hanyalah SQL-nya.
//
// # Kenapa carriage return dibuang lebih dulu
//
// Repository ini ber-`core.autocrlf=true`, sehingga berkas `.sql` yang sama berisi LF di satu
// mesin dan CRLF di mesin lain — tergantung checkout, bukan tergantung isinya. Tanpa pembuangan
// ini, setiap baris SQL berakhir dengan `\r` yang ikut terkirim ke Oracle.
//
// Oracle memperlakukannya sebagai spasi putih sehingga kuerinya tetap jalan, dan justru itu
// yang membuatnya berbahaya: ia tidak pernah gagal, hanya muncul saat SQL dicetak ke log atau
// dibandingkan dengan teks yang diharapkan. Uji urutan `ORDER BY` modul ini sempat merah karena
// persis itu — pada mesin yang checkout-nya CRLF, bukan karena kuerinya berubah.
func splitByName(content string) map[string]string {
	const marker = "-- name:"

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
			// Komentar kepala berkas dan komentar penjelas tiap kueri tidak ikut dikirim:
			// yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
			continue
		}

		body = append(body, line)
	}

	flush()
	return result
}
