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

// listColumns adalah ke-12 alias yang dikembalikan SETIAP kueri daftar.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxrclpucl.sql dan dengan urutan pemindai
// scanWorkItem. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di query_test.go.
//
// `CREATED_AT` adalah kolom yang MENGURUTKAN ketiga kueri (`ORDER BY TGL_CREATE_PUCL DESC`).
// Ia ditambahkan 2026-09-30 atas keputusan Work Owner supaya tabelnya tidak lagi terbaca
// acak; urutannya sendiri tidak berubah sedikit pun. Lihat WorkItem.CreatedAt.
//
// `REFERENCE` DAN `CASE_ID` KINI BERNILAI SAMA, keduanya `TC_PNC_PUCL.CLAIMID`. Tabel Pega
// memisahkan keduanya — `PZINSKEY` kunci teknis, `PYID` nomor case — dan tabel datar hanya
// menyimpan yang kedua. Kedua alias dipertahankan supaya kontrak ke layar tidak berubah;
// yang berubah hanyalah isi `REFERENCE`. Lihat catatan pada kueri `list_cetak_surat`.
var listColumns = []string{
	"REFERENCE", "CASE_ID", "POLICY_NUMBER", "INSURED_NAME", "INBOX_ENTRY_AT",
	"ANALYST_NOTE", "TRACK_CODE", "LETTER_PRINTED_AT", "CLAIM_AGE", "EXPIRY_STATUS",
	"CREATED_AT", "TOTAL_ROWS",
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

// detailColumns adalah alias yang dikembalikan kueri layar kerja.
//
// Jumlahnya SENGAJA tidak ditulis di sini. Ia sudah dua kali bertambah — keempat isian surat
// 2026-10-01, lalu kedua kolom penentu tombol pada hari yang sama — dan angka di komentar
// tidak ikut berubah bersamanya. Yang menjaga kelengkapannya adalah uji, bukan kalimat ini.
//
// Ia BERBEDA dari listColumns, dan perbedaannya bukan kelalaian — ia mengikuti SECTION,
// bukan grid:
//
//   - membawa DUA isian TURUNAN dari anak klaim (`FIRST_OBJECT_NAME`,
//     `FIRST_PROPOSE_VALUE`) yang tidak ada di grid mana pun. `FIRST_OBJECT_NAME` mengisi
//     DUA isian sekaligus — "Nama Peserta" dan "UP" — karena begitulah Pega mengisinya;
//   - TIDAK membawa "Lama Klaim" maupun "Status Kadaluarsa", yang hanya dipakai grid;
//   - TIDAK membawa tanggal cetak surat maupun tanggal kirim RCL/PUCL. Keduanya sempat
//     dibawa ke sini dan itu KELIRU: `Section/SectionLampiranSuratPUCL-Section.xml` tidak
//     memuat satu pun dari keduanya. Layar kerja menggambar apa yang digambar section-nya,
//     bukan apa yang kebetulan sudah ada di tangan (`D-13`).

var detailColumns = []string{
	// `INSURED_PARTY` (`QQ_NAME`) disisipkan tepat SESUDAH `REFERENCE`, sesuai urutannya di
	// berkas .sql. Ia penerima surat — baris "Kepada Yth." pada templat `SuratPUCL`.
	"REFERENCE", "INSURED_PARTY",
	"CLAIM_NUMBER", "TRACK_CODE", "ANALYST_NOTE", "POLICY_NUMBER",
	"LOSS_DATE", "PUCL_NOTE",

	// Keempat isian surat, ditambahkan 2026-10-01 setelah kolomnya ditemukan ADA di
	// `TC_PNC_PUCL` yang berjalan — meski tidak ada di `Database/CREATE_TABLE_3.SQL`.
	// Sebelumnya keempatnya digambar bertanda "di clipboard Pega".
	"SUBJECT", "OPENING_NOTE", "BODY_NOTE", "CLOSING_NOTE",

	// Kedua kolom ini TIDAK digambar sebagai isian — keduanya menentukan TOMBOL mana yang
	// muncul. Ditambahkan 2026-10-01; lihat ClaimDetail.Buttons.
	"MSIG_FLAG", "GROUP_PANEL",

	// Kedua isian tab Penerimaan Dokumen yang sebelumnya bertanda "di clipboard Pega".
	// Ditambahkan 2026-10-01 atas penetapan Work Owner; lihat catatan di kueri `detail`.
	"DOCUMENT_COMPLETE_AT",

	// Ketiga parameter tersembunyi tombol tindakan. Ditambahkan 2026-10-01 setelah kolomnya
	// dibuat Work Owner; sebelumnya ketiganya memakai nilai penampung.
	//
	// Urutannya WAJIB sama dengan urutan kolom di berkas .sql dan dengan urutan pemindai
	// scanDetail — ketiganya disisipkan SEBELUM INSURED_EMAIL, bukan sesudahnya.
	"ACTION_ID_OBJECT", "ACTION_ID_COVERAGE", "ACTION_ID_ADJUSTMENT",

	"INSURED_EMAIL",

	// Baris PERTAMA grid "Tanggal Terima Dokumen". Daftarnya page list tanpa tabel; hanya
	// baris pertamanya yang diekspos sebagai kolom. Lihat catatan di kueri `detail`.
	"RECEIVED_DATE_FIRST", "RECEIVED_NOTE_FIRST",

	"FIRST_OBJECT_NAME", "FIRST_PROPOSE_VALUE",
}

// letterColumnsBeyondTheSharedDDL adalah kolom yang dipakai layar kerja tetapi TIDAK ada di
// `Database/CREATE_TABLE_3.SQL`.
//
// Ia ditulis terpisah supaya selisih antara tabel yang berjalan dan DDL yang dibagikan punya
// satu tempat yang menyebutkannya, dan supaya uji dapat menuntut keempatnya ikut diperiksa
// `check_columns`. Begitu DDL-nya disamakan, daftar ini dikosongkan — bukan dihapus diam-diam.
var letterColumnsBeyondTheSharedDDL = []string{
	"PERIHAL", "KETERANGAN1", "KETERANGAN2", "KETERANGAN3",

	// Ketiganya dibuat Work Owner 2026-10-01 dan BELUM masuk `CREATE_TABLE_3.SQL` pula.
	// Portal yang tabelnya dibuat dari berkas itu akan gagal ORA-00904 pada klaim pertama
	// yang dibuka — bukan saat build.
	"ID_OBJECT", "ID_COVERAGE", "ID_ADJUSTMENT",
}

// listQueries adalah nama ketiga kueri daftar, dipakai uji kesesuaian alias.
var listQueries = []string{
	"list_cetak_surat", "list_kelengkapan_dokumen", "list_klaim_msig",
}

// flatTableQueries adalah kueri yang membaca POOLDATA.TC_PNC_PUCL.
//
// Ia ketiga kueri daftar DITAMBAH layar kerja. Dipisah dari listQueries karena yang diuji
// berbeda: listQueries diuji kesesuaian alias dan penyaringnya, daftar ini diuji bahwa tidak
// satu pun dari keempatnya tertinggal membaca tabel Pega atau memakai nama kolom bentuk lama.
//
// Nama kolom tabel datar BERBEDA dari nama kolom Pega yang digantikannya — akhiran `_1`
// dibuang dan kata dipisah garis bawah. Satu nama yang tertinggal pada bentuk lama gagal
// dengan ORA-00904 pada permintaan pertama di produksi, bukan saat build.
var flatTableQueries = []string{
	"list_cetak_surat", "list_kelengkapan_dokumen", "list_klaim_msig", "detail",
}

// pegaReadsAllowedIn menyebut kueri tabel datar yang BOLEH tetap membaca tabel Pega, beserta
// apa yang dibacanya.
//
// # Kenapa pengecualian BERNAMA, bukan penjaga yang dilonggarkan
//
// Karena penjaganya menangkap kelas cacat yang nyata: kueri tabel datar yang tertinggal
// membaca tabel Pega TETAP berjalan dan tetap mengembalikan baris yang terbaca masuk akal,
// sehingga tidak ada apa pun yang menandakannya. Melonggarkan penjaganya menghapus
// perlindungan itu untuk seluruh kueri; menyebut pengecualiannya satu per satu hanya
// menghapusnya untuk yang memang diputuskan.
//
// Isiannya KOSONG sejak 2026-10-08. Sebelumnya `detail` membaca DUA kolom hasil ekspos baris
// pertama page list "Tanggal Terima Dokumen" (`RECEIVEDDATE_1`, `KETERANGAN_1`) dari tabel
// objek kerja Pega. Work Owner memutuskan tabel itu tidak dipakai lagi: tanggalnya kini dari
// `T_CLAIM_PNC.RECEIVEDATE`, catatannya NULL — lihat catatan pada kueri `detail`.
//
// Petanya dipertahankan supaya pengecualian berikutnya, bila memang diputuskan, tetap harus
// disebut satu per satu.
var pegaReadsAllowedIn = map[string][]string{}

// documentColumns adalah alias kueri daftar dokumen.
//
// Ia TIDAK masuk flatTableQueries dan itu disengaja: `documents` tidak membaca `TC_PNC_PUCL`
// sama sekali — ia menerjemahkan nomor case menjadi kunci lampiran lewat `T_CLAIM_PNC` dan
// (sejak 2026-10-08, menggantikan tabel objek kerja Pega) kunci berprefix yang dirangkai dari
// nomor case. Memasukkannya ke daftar penjaga akan membuat uji menuntut hal yang mustahil.
var documentColumns = []string{
	"DOCUMENT_ID", "DOCUMENT_NAME", "MIME_TYPE",
	"CATEGORY_NAME", "SUBCATEGORY_NAME", "UPLOADED_AT", "UPLOADED_BY",

	// Penanda apakah barisnya tergambar di layar lampiran Pega — lihat kueri `documents`.
	"PEGA_VISIBLE",
}

// legacyColumnNames adalah nama kolom Pega yang TIDAK boleh lagi muncul di kueri tabel
// datar.
//
// Seluruhnya punya padanan bernama lain di `TC_PNC_PUCL`, dan seluruhnya tetap sah di
// `daily_report` — yang memang masih membaca tabel Pega. Karena itu uji penjaganya berjalan
// atas flatTableQueries, bukan atas seluruh kueri.
var legacyColumnNames = []string{
	"PZINSKEY", "PYID", "PXOBJCLASS", "PXCREATEDATETIME", "PYSTATUSWORK",
	"PXASSIGNEDOPERATORID", "POLICYNO", "QQNAME",
	"TANGGALKIRIMPUCL_1", "TANGGALCETAKDOKUMENPUCL_1", "KOMENTARANALISATOR_1",
	"KOMENTARPUCL_1", "RCL_PUCL_1", "LAMAKLAIM_1", "STATUSKLAIM_1", "STATUSCASE_1",
	"PUCLAPPROVE_1", "MSIG_1", "DATEOFLOSS_1",
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
