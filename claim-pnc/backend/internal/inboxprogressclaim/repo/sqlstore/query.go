// Package sqlstore memenuhi seam inboxprogressclaim.Repo dengan SQL terhadap Oracle.
//
// Dua aturan mengikat seluruh berkas di sini, sama dengan sqlstore modul lain:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke dalam
//     teks SQL.
//
// Aturan kedua menutup cacat nyata, bukan cacat teoretis. Ketiga kueri lama menyusun
// klausa WHERE-nya di activity lalu menyisipkannya sebagai POTONGAN SQL lewat lima titik
// `{ASIS:…}` — dan yang paling terbuka adalah kotak cari, yang merangkai isian pengguna apa
// adanya menjadi `like '%"+TempRefresh.ClaimNo+"%'`. Larangan itu TIDAK ikut dikecualikan
// oleh keputusan Work Owner "replikasi apa adanya": yang direplikasi adalah perilaku
// bisnis, bukan celah injeksi.
package sqlstore

import (
	"embed"
	"strconv"
	"strings"

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxprogressclaim/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxprogressclaim/sqlstore", name) }

// claimColumns adalah ke-10 alias yang dikembalikan KEDUA kueri daftar klaim.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxprogressclaim.sql dan dengan urutan
// pemindai scanClaimRow. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat
// diuji kesesuaiannya di query_test.go.
var claimColumns = []string{
	"CLAIM_NUMBER", "POLICY_NUMBER", "INSURED_NAME",
	"REGISTER_DATE", "LOSS_DATE", "LGB_NOTE", "TECHNICAL_PIC",
	"EARLIEST_FOLLOW_UP", "PROCESS_DATE", "PROD_KE",
}

// positionColumns adalah alias kueri posisi.
var positionColumns = []string{
	"CLAIM_NUMBER", "POSITION_NAME", "PROGRESS_STATUS_1", "PROGRESS_STATUS_2",
	"NEXT_FOLLOW_UP",
}

// picColumns adalah alias kueri rekap per PIC.
var picColumns = []string{
	"PIC", "CLAIM_COUNT", "UPDATE_COUNT", "DUE_TODAY_COUNT", "ON_TIME_COUNT", "LATE_COUNT",
}

// inList menyusun daftar PENANDA bind untuk sebuah klausa IN, bukan nilainya.
//
// # Kenapa ini bukan pelanggaran larangan merangkai SQL
//
// Yang dirangkai hanyalah teks `:7, :8, :9` — deret penanda yang panjangnya ditentukan
// JUMLAH baris, bukan ISI baris. Tidak ada satu pun nilai yang menyentuh teks SQL; seluruh
// nomor klaim tetap dikirim sebagai argumen bind. `query_test.go` menguji tepat itu:
// keluaran fungsi ini hanya boleh terdiri atas penanda bind dan pemisahnya.
//
// # Kenapa daftarnya perlu dinamis sama sekali
//
// Karena kueri posisi dijalankan untuk satu HALAMAN klaim sekaligus. Alternatifnya
// memanggilnya sekali per baris, dan itu kueri di dalam perulangan — yang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 larang. Alternatif lain adalah menyiapkan
// penanda sebanyak MaxPageSize lalu mengisi sisanya NULL, yang membuat setiap kueri
// membawa 100 bind meski halamannya berisi 15 baris.
//
// `start` adalah nomor bind pertama, yaitu jumlah bind yang sudah terpakai ditambah satu.
func inList(start, count int) string {
	if count < 1 {
		// Klausa `IN ()` tidak sah di SQL mana pun. `NULL` menghasilkan klausa yang sah
		// dan tidak pernah cocok — tepat seperti yang dimaksud oleh "tidak ada satu pun
		// nomor klaim untuk dicari".
		return "NULL"
	}

	markers := make([]string, 0, count)
	for i := 0; i < count; i++ {
		markers = append(markers, ":"+strconv.Itoa(start+i))
	}
	return strings.Join(markers, ", ")
}
