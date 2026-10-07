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

	"claim-pnc/internal/platform/sqlfile"
)

//go:embed *.sql
var queryFiles embed.FS

// queries memuat seluruh pernyataan SQL, dikunci dengan namanya.
var queries = sqlfile.MustLoad(queryFiles, "inboxmanagerreceivepucl/sqlstore")

// query mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func query(name string) string {
	return sqlfile.MustGet(queries, "inboxmanagerreceivepucl/sqlstore", name)
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
