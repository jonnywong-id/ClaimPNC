// Package sqlstore memenuhi seam inboxmanagerreceivepucl.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis.
// `RDB List/GetDataPUCLRCLForDailyReport-SQL.xml` menyisipkan
// `{TempRCLPUCLReport.AlasanKlaim}` dan `{TempRCLPUCLReport.NoteKasir}` langsung ke teks
// SQL-nya sebagai batas tanggal.
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
			"inboxmanagerreceivepucl/sqlstore: kueri %q tidak ditemukan di berkas .sql",
			name))
	}
	return text
}

// resultColumns adalah ke-18 alias yang dikembalikan SETIAP kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxmanagerreceivepucl.sql dan dengan urutan
// pemindai scanWorkItem. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat
// diuji kesesuaiannya di query_test.go.
var resultColumns = []string{
	"REFERENCE", "CASE_ID", "POLICY_NUMBER", "CLAIM_NUMBER", "INSURED_NAME",
	"LOSS_DATE", "GROUP_PANEL", "SENDER_NAME", "DOCUMENT_RECEIVED_DATE",
	"DOCUMENT_SHEET_COUNT", "INBOX_ENTRY_AT", "ANALYST_NOTE",
	"TRACK", "TRACK_STATUS", "LETTER_PRINTED_AT", "CLAIM_AGE", "EXPIRY_STATUS",
	"TOTAL_ROWS",
}

// listQueries adalah nama kedua kueri daftar, dipakai uji kesesuaian alias.
var listQueries = []string{"list_receive", "list_rclpucl"}

// receiveQueries adalah kueri tab Receive.
//
// Dipisah dari listQueries karena hanya ia yang WAJIB menyaring kelas objek kerja berkas
// penerimaan dokumen dan menggabung tabel penugasan per orang; kueri RCL/PUCL menyaring
// kelas dan tabel yang berbeda. Kesesuaiannya dijaga query_test.go.
//
// Ia tetap senarai meski isinya satu: kedua grid Pega digabung menjadi satu kueri pada
// 2026-09-30, dan bentuk senarai membuat pemecahannya kembali — bila kelak diputuskan — tidak
// menuntut perubahan bentuk uji.
var receiveQueries = []string{"list_receive"}

// documentColumns adalah ke-23 alias yang dikembalikan kueri LAYAR KERJA.
//
// Urutannya WAJIB sama dengan urutan kolom `detail_receive_document` di
// inboxmanagerreceivepucl.sql dan dengan urutan pemindai scanDocument. Ia ditulis lengkap di
// sini pula supaya ketiga tempat itu dapat diuji kesesuaiannya di query_test.go — pemindai
// yang urutannya bergeser satu kolom tidak menghasilkan satu pun galat, ia hanya menaruh
// kronologi kejadian di kolom nomor polis.
var documentColumns = []string{
	"REFERENCE", "CASE_ID", "CLAIM_NUMBER", "GROUP_PANEL", "WORK_STATUS",
	"CREATED_AT", "RECEIVED_AT", "SENDER_NAME", "SENDER_EMAIL", "SENDER_PHONE",
	"COURIER_NAME", "INSURED_NAME", "POLICY_NUMBER", "LOSS_DATE", "REFERENCE_NUMBER",
	"INSURED_EMAIL", "LOSS_LOCATION", "DRIVER_LICENCE", "CHRONOLOGY", "DAMAGE_DETAIL",
	"TRANSFER_REASON", "EMAIL_SUBJECT", "NOT_REGISTERED_NOTE",
}

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxmanagerreceivepucl/sqlstore: tidak dapat membaca berkas kueri: " +
			err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxmanagerreceivepucl/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxmanagerreceivepucl/sqlstore: nama kueri ganda: " + name)
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
			// Komentar kepala berkas dan komentar penjelas tiap kueri tidak ikut
			// dikirim: yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
			continue
		}

		body = append(body, line)
	}

	flush()
	return result
}
