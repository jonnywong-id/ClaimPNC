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

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxsurvey/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string { return sqlfile.MustGet(queries, "inboxsurvey/sqlstore", name) }

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
	"SURVEY_ID", "CLAIM_ID", "SURVEY_INDEX",
	"CLAIM_NUMBER", "POLICY_NUMBER", "INSURED_NAME", "CLASS_OF_BUSINESS", "CAUSE_OF_LOSS",
	"LOCATION", "TECHNICAL_PIC", "ADJUSTER_PIC", "DATE_OF_LOSS", "CREATED_AT",
	"ASM_STATUS", "SURVEYOR_TYPE", "TOTAL_ROWS",
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
var kpiColumns = []string{
	"GROUP_KEY", "SURVEY_SCHEDULING", "IMMEDIATE_ADVICE", "PRELIMINARY_ADVICE",
	"INTERIM_REPORT", "PROGRESS_UPDATE", "COMMUNICATION_RESPONSE", "PROPOSE_ADJUSTMENT",
	"FINAL_REPORT", "VALUE_SCORE",
}
