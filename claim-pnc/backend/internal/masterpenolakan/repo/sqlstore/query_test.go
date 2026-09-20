package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql.
//
// Nama kueri adalah konstanta di dalam kode, dan ketiadaannya baru terlihat saat kueri itu
// pertama dijalankan — yang bisa jadi berbulan-bulan kemudian pada jalur yang jarang
// dilewati. Uji ini memindahkan kegagalannya ke waktu build.
func TestEveryUsedQueryExists(t *testing.T) {
	for _, name := range []string{
		"rejection_parent_list",
		"rejection_parent_get",
		"rejection_parent_list_id_locked",
		"rejection_parent_insert",
		"rejection_parent_check_table",
		"rejection_list",
		"rejection_get",
		"rejection_list_id_locked",
		"rejection_insert",
		"rejection_update",
		"rejection_check_table",
		"committee_rejection_list",
		"committee_rejection_get",
		"committee_rejection_list_id_locked",
		"committee_rejection_insert",
		"committee_rejection_update",
		"committee_rejection_check_table",
	} {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

// Tabel yang benar. Ini uji yang paling penting di berkas ini.
//
// `MST_PENOLAKAN_KLAIM_1` dan `_2` namanya hanya berbeda satu karakter, dan KEDUANYA punya
// kolom ID_ST dan NOTE_ST. Tertukar sekali saja berarti layar menulis ke tabel yang salah
// tanpa satu pun galat basis data, karena nama kolomnya memang ada di keduanya.
func TestParentQueriesTargetTheParentTable(t *testing.T) {
	for name, text := range query {
		if !strings.HasPrefix(name, "rejection_parent_") {
			continue
		}
		upperCase := strings.ToUpper(text)
		require.Containsf(t, upperCase, "POOLDATA.MST_PENOLAKAN_KLAIM_1",
			"kueri %q harus menyentuh tabel tingkat 1", name)
		require.NotContainsf(t, upperCase, "MST_PENOLAKAN_KLAIM_2",
			"kueri %q menyentuh tabel tingkat DUA — periksa ulang nama tabelnya", name)
	}
}

func TestChildQueriesTargetTheChildTable(t *testing.T) {
	for name, text := range query {
		if !strings.HasPrefix(name, "rejection_") || strings.HasPrefix(name, "rejection_parent_") {
			continue
		}
		upperCase := strings.ToUpper(text)
		require.Containsf(t, upperCase, "POOLDATA.MST_PENOLAKAN_KLAIM_2",
			"kueri %q harus menyentuh tabel tingkat 2", name)
		require.NotContainsf(t, upperCase, "MST_PENOLAKAN_KLAIM_1",
			"kueri %q menyentuh tabel tingkat SATU — periksa ulang nama tabelnya", name)
	}
}

func TestCommitteeQueriesTargetTheirOwnTable(t *testing.T) {
	for name, text := range query {
		if !strings.HasPrefix(name, "committee_rejection_") {
			continue
		}
		upperCase := strings.ToUpper(text)
		require.Containsf(t, upperCase, "POOLDATA.MST_REJECTED_KOMITE",
			"kueri %q harus menyentuh tabel penolakan komite", name)
		require.NotContainsf(t, upperCase, "MST_PENOLAKAN_KLAIM",
			"kueri %q menyentuh tabel tab sebelah", name)
	}
}

// TIDAK ADA DELETE, pada tab mana pun.
//
// Seluruh export tidak memuat satu pun DELETE terhadap ketiga tabel modul ini, layar lama
// tidak punya tombolnya, dan tidak satu pun tabelnya punya kolom penanda terhapus yang
// dapat dipakai `D-66`. Uji ini menjaga DELETE tidak muncul diam-diam kelak tanpa
// keputusan yang menyertainya.
func TestNoQueryDeletesRows(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE ", "kueri %q menghapus baris", name)
	}
}

// Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
//
// Kueri lama justru sebaliknya: `BrowseStatusPenolakanKlaim2-SQL.xml` menyisipkan
// `{ASIS:MasterCheckerPenolakan.RemakApprove}` — potongan teks SQL dari nilai klipboard.
// Itu persis celah yang §4.3 `08-TECHNICAL-STRATEGY.md` tutup.
func TestWriteQueriesUseParameterBinding(t *testing.T) {
	for _, name := range []string{
		"rejection_parent_get",
		"rejection_parent_insert",
		"rejection_get",
		"rejection_insert",
		"rejection_update",
		"committee_rejection_get",
		"committee_rejection_insert",
		"committee_rejection_update",
	} {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Daftar Status Penolakan 2 TIDAK menyaring status.
//
// Potongan `{ASIS:...}` pada kueri lama diisi `"WHERE STATUS='0'"` oleh
// `Activity/BrowseStatusPenolakanKlaim_2-Act.xml`, TETAPI langkah itu berprasyarat
// `Param.master=="1"` yang hanya dikirim layar checker pada Inbox Manager. Dari layar
// Master Penolakan Klaim potongan itu tetap kosong.
//
// Menambahkan penyaring di sini akan menyembunyikan baris yang sudah diputuskan — dan itu
// perubahan perilaku, bukan replikasi.
func TestListShowsEveryRowRegardlessOfApproval(t *testing.T) {
	text := strings.ToUpper(getQuery("rejection_list"))
	require.NotContains(t, text, "WHERE",
		"daftar tidak menyaring apa pun; penyaring status milik layar checker")
	require.NotContains(t, text, "STATUS='0'")
}

// Derivasi label status tidak dilakukan di SQL.
//
// `case when A.STATUS='1' then 'APPROVED' ... end` pada kueri lama adalah pemformatan
// untuk tampilan, dan §4.3 menetapkan pemformatan dilakukan di Go. Yang dibaca adalah
// kolom STATUS apa adanya.
func TestListReadsRawStatusWithoutFormattingIt(t *testing.T) {
	text := strings.ToUpper(getQuery("rejection_list"))
	require.NotContains(t, text, "CASE WHEN")

	// Yang dicari adalah LITERAL bertanda kutip, bukan kata "APPROVED" begitu saja —
	// kolom NOTEAPPROVED dan TANGGAL_APPROVE memang memuatnya sebagai bagian nama.
	require.NotContains(t, text, "'APPROVED'")
	require.NotContains(t, text, "'MENUNGGU'")
}

// Penyaring kunci memangkas spasi padatan kolom CHAR — kecuali pada tabel yang kuncinya
// NUMBER.
//
// Parameter binding bertipe VARCHAR2, dan perbandingan CHAR dengan VARCHAR2 memakai
// non-padded comparison. Gejala hilangnya TRIM sangat menyesatkan: tidak ada galat sama
// sekali, hanya "tidak ditemukan" untuk setiap baris yang sebenarnya ada.
func TestTextKeyFiltersHandleCHARPadding(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("rejection_get")), "TRIM(ID_ND)")
	require.Contains(t, strings.ToUpper(getQuery("rejection_update")), "TRIM(ID_ND)")
	require.Contains(t, strings.ToUpper(getQuery("rejection_parent_get")), "TRIM(ID_ST)")
}

func TestNumericKeyFilterNeedsNoTrim(t *testing.T) {
	// IDMASTER bertipe NUMBER (`INSERTMASTERREJECTEDKOMITE.prc:1`), sehingga tidak ada
	// pemadatan spasi yang perlu ditangani. TRIM di sini justru akan mematikan index-nya
	// tanpa memberi manfaat apa pun.
	require.NotContains(t, strings.ToUpper(getQuery("committee_rejection_get")), "TRIM(")
}

// Penurunan nomor baru harus menyerialkan diri, pada KETIGA tabel.
//
// Tanpa FOR UPDATE, dua penambahan bersamaan dapat menerima nomor yang sama — lubang yang
// memang terbuka di sistem lama, karena MAX(...)+1 di sana dijalankan sebagai pernyataan
// lepas beberapa baris sebelum INSERT-nya.
func TestIDFetchersLockRows(t *testing.T) {
	for _, name := range []string{
		"rejection_parent_list_id_locked",
		"rejection_list_id_locked",
		"committee_rejection_list_id_locked",
	} {
		require.Containsf(t, strings.ToUpper(getQuery(name)), "FOR UPDATE",
			"kueri %q harus mengunci barisnya", name)
	}
}

// Penyisipan TIDAK menyebut ketiga kolom persetujuan.
//
// Sistem lama pun tidak menyebutnya (`MASTERPENOLAKANKLAIM2.prc:9`), sehingga basis data
// mengisinya dengan default kolomnya sendiri. Yang berwenang mengisinya adalah layar
// checker pada Inbox Manager, bukan layar ini.
func TestInsertNeverWritesApprovalColumns(t *testing.T) {
	text := strings.ToUpper(getQuery("rejection_insert"))
	columns := text[strings.Index(text, "("):strings.Index(text, "VALUES")]

	for _, column := range []string{"APPROVEBY", "TANGGAL_APPROVE", "NOTEAPPROVED"} {
		require.NotContainsf(t, columns, column,
			"kolom %s diisi layar checker, bukan layar ini", column)
	}
}

// Pengubahan juga TIDAK menyentuh ketiga kolom persetujuan.
//
// Procedure lama pun membiarkannya (`MASTERPENOLAKANKLAIM2.prc:14`). Akibatnya baris
// berstatus MENUNGGU masih memuat nama penyetuju sebelumnya — jejak keputusan yang pernah
// ada, bukan keadaan yang berlaku.
func TestUpdateLeavesApprovalTraceUntouched(t *testing.T) {
	text := strings.ToUpper(getQuery("rejection_update"))
	for _, column := range []string{"APPROVEBY", "TANGGAL_APPROVE", "NOTEAPPROVED"} {
		require.NotContainsf(t, text, column, "kolom %s tidak boleh disentuh layar ini", column)
	}
}

// Pengubahan MENGEMBALIKAN baris ke antrean persetujuan.
//
// STATUS dan TANGGALKIRIM keduanya di-SET, persis `MASTERPENOLAKANKLAIM2.prc:14`.
// Perilaku itu dipertahankan atas keputusan Work Owner 2026-09-19.
func TestUpdateResetsApprovalState(t *testing.T) {
	text := strings.ToUpper(getQuery("rejection_update"))
	require.Contains(t, text, "STATUS")
	require.Contains(t, text, "TANGGALKIRIM")
}

// Kunci tidak pernah ikut di-SET pada pengubahan mana pun.
func TestUpdateNeverSetsTheKey(t *testing.T) {
	rejection := strings.ToUpper(getQuery("rejection_update"))
	setClause := rejection[strings.Index(rejection, "SET"):strings.Index(rejection, "WHERE")]
	require.NotContains(t, setClause, "ID_ND")

	committee := strings.ToUpper(getQuery("committee_rejection_update"))
	committeeSet := committee[strings.Index(committee, "SET"):strings.Index(committee, "WHERE")]
	require.NotContains(t, committeeSet, "IDMASTER")
}

// Tanpa konstruksi khas Oracle — SQL harus berjalan sama di Oracle 19c dan PostgreSQL 17+
// (`D-20`).
func TestNoQueryUsesOracleOnlyConstructs(t *testing.T) {
	forbidden := []string{"NVL(", "SYSDATE", "DECODE(", "ROWNUM", "TO_CHAR(", "SELECT *", "FROM DUAL"}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContainsf(t, upperCase, pattern,
				"kueri %q memakai %q yang tidak portabel", name, pattern)
		}
	}
}

// Kueri pemeriksa tabel tidak mengambil satu baris pun — ia dijalankan terhadap produksi
// pada mode periksa.
func TestCheckTableFetchesNoRows(t *testing.T) {
	for _, name := range []string{
		"rejection_parent_check_table",
		"rejection_check_table",
		"committee_rejection_check_table",
	} {
		require.Containsf(t, getQuery(name), "1 = 0", "kueri %q harus mengambil nol baris", name)
	}
}
