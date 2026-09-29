// Package sqlstore memenuhi seam inboxosclaimpercabang.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya dapat
//     dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam teks
//     SQL.
//
// Aturan kedua menutup cacat nyata di layar INI, bukan cacat teoretis. Ketiga kueri sumbernya
// menyisipkan nilai langsung ke teks SQL-nya — `GetDataOutstandingperCabang` dan
// `GetDataOutstandingperCabangExport` menyisipkan `{OperatorID.pyTelephone}`, dan
// `GetProgress1Sama` menyisipkan `{TempCari.CARI1}`. Yang pertama adalah batas datanya sendiri:
// nilai yang disisipkan ke sana menentukan cabang siapa yang terlihat.
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
			"inboxosclaimpercabang/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// listColumns adalah ke-20 alias yang dikembalikan kueri `list`, dalam urutannya.
//
// Urutannya WAJIB sama dengan pemindai scanWorkItem. Ia ditulis lengkap di sini supaya kedua
// tempat itu dapat diuji kesesuaiannya di query_test.go.
var listColumns = []string{
	"BRANCH_NAME", "BRANCH_CODE", "BUSINESS_SOURCE", "BUSINESS_NAME",
	"POLICY_NUMBER", "CLAIM_NUMBER", "REGISTER_DATE", "LOSS_DATE",
	"REMARK_RECOMMENDATION", "ESTIMATION_VALUE", "LAST_PROGRESS_AT",
	"PROGRESS_STATUS_1", "PROGRESS_STATUS_2", "TECHNICAL_PIC", "PROGRESS_NOTE",
	"ADJUSTER_NAME", "CAUSE_OF_LOSS", "CHRONOLOGY", "PROGRESS_STALLED",
	"TOTAL_ROWS",
}

// exportOnlyColumns adalah alias yang HANYA dikembalikan `list_export`, dalam urutannya.
//
// Ia disebut terpisah supaya dua hal dapat diuji sekaligus: bahwa `list_export` memuat seluruh
// alias `list` sebagai AWALAN persis, dan bahwa tambahannya tepat yang disebut di sini. Tanpa
// yang kedua, menghapus satu kolom treaty tetap lolos uji awalan.
var exportOnlyColumns = []string{
	"CLAIM_KEY", "POLICY_BUSINESS_NAME", "INSURED_NAME",
	"RESERVE_CLAIM_FULL", "RESERVE_CLAIM_ASM", "COINSURANCE",
	"SHARE_OR", "SHARE_FACOUT", "SHARE_FACOB", "SHARE_QS",
	"SHARE_FSPL", "SHARE_SSPL", "SHARE_ER1", "SHARE_ER2",
	"SHARE_BPPDAN", "SHARE_PSRQS", "SHARE_PSRSPL", "SHARE_ORS",
	"SHARE_XL", "SHARE_PSROR", "SHARE_QSOR", "SHARE_PSS",
	"SHARE_PRGBI", "SHARE_PFRA", "SHARE_FSPLNSRI", "SHARE_PSPLNSRI",
	"SHARE_FSPLNSOR", "SHARE_PSPLNSOR", "SHARE_FACOBSRB", "SHARE_FACOBINDT",
}

// loadQueries membaca setiap berkas .sql dan memecahnya pada penanda "-- name: <nama>",
// sehingga satu berkas dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu
// kesatuan saat di-review.
func loadQueries() map[string]string {
	result := map[string]string{}

	entries, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxosclaimpercabang/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}

	for _, entry := range entries {
		content, err := queryFiles.ReadFile(entry.Name())
		if err != nil {
			panic("inboxosclaimpercabang/sqlstore: tidak dapat membaca " +
				entry.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxosclaimpercabang/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}

	return result
}

// splitByName memisahkan isi berkas menjadi pernyataan bernama, membuang baris komentar supaya
// yang dikirim ke basis data hanyalah SQL-nya.
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
			// Komentar kepala berkas dan komentar penjelas tiap kueri tidak ikut dikirim:
			// yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
			continue
		}

		body = append(body, line)
	}

	flush()
	return result
}
