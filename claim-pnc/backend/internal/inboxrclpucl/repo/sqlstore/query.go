// Package sqlstore memenuhi seam inboxrclpucl.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis — dan di modul ini cacatnya
// menyentuh isian yang DIKETIK PENGGUNA. `RDB List/GetDataPUCLRCLForDailyReport-SQL.xml`
// menyisipkan `{TempRCLPUCLReport.AlasanKlaim}` dan `{TempRCLPUCLReport.NoteKasir}` — kedua
// isian tanggal laporan harian — langsung ke teks SQL-nya.
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
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat saat
// pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func query(name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf(
			"inboxrclpucl/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// listColumns adalah ke-11 alias yang dikembalikan SETIAP kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxrclpucl.sql dan dengan urutan pemindai
// scanWorkItem. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go.
var listColumns = []string{
	"REFERENCE", "CASE_ID", "POLICY_NUMBER", "INSURED_NAME", "INBOX_ENTRY_AT",
	"ANALYST_NOTE", "TRACK_CODE", "LETTER_PRINTED_AT", "CLAIM_AGE", "EXPIRY_STATUS",
	"TOTAL_ROWS",
}

// reportColumns adalah ke-10 alias yang dikembalikan kueri laporan harian.
//
// Ia BERBEDA dari listColumns, dan perbedaannya bukan kelalaian: laporan memuat
// `CLAIM_STATUS` yang tidak ada di grid mana pun, dan TIDAK memuat `CLAIM_AGE` yang ada di
// setiap grid. Lihat catatan pada inboxrclpucl.DailyReportRow.
var reportColumns = []string{
	"REFERENCE", "CASE_ID", "POLICY_NUMBER", "INSURED_NAME", "SENT_AT",
	"ANALYST_NOTE", "LETTER_PRINTED_AT", "TRACK_CODE", "CLAIM_STATUS", "TOTAL_ROWS",
}

// listQueries adalah nama ketiga kueri daftar, dipakai uji kesesuaian alias.
var listQueries = []string{
	"list_cetak_surat", "list_kelengkapan_dokumen", "list_klaim_msig",
}

// printedQueries adalah kedua kueri yang menyaring surat SUDAH dicetak.
//
// Dipisah dari listQueries karena hanya keduanya yang wajib memuat `IS NOT NULL` dan
// penyaring penanda persetujuan; kueri tab Cetak Surat menyaring kebalikannya. Kesesuaiannya
// dijaga query_test.go — dan itu bukan kerapian: satu tanda yang tertukar di sana menukar
// isi dua tab tanpa satu pun galat.
var printedQueries = []string{"list_kelengkapan_dokumen", "list_klaim_msig"}

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxrclpucl/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxrclpucl/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxrclpucl/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}

	return result
}

// splitByName memisahkan isi berkas menjadi pernyataan bernama, membuang baris komentar
// supaya yang dikirim ke basis data hanyalah SQL-nya.
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
			// Komentar kepala berkas dan komentar penjelas tiap kueri tidak ikut
			// dikirim: yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
			continue
		}

		body = append(body, line)
	}

	flush()
	return result
}
