package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri tingkat 2 yang dipanggil kode harus benar-benar ada di berkas .sql.
func TestEveryUsedQuery2Exists(t *testing.T) {
	usedNames := []string{
		"progress_status2_list",
		"progress_status2_get",
		"progress_status2_list_id_locked",
		"progress_status2_insert",
		"progress_status2_check_table",
	}
	for _, name := range usedNames {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

// Tabel yang benar. Ini uji yang paling penting di berkas ini.
//
// Nama kedua tabel nyaris sama, dan yang berakhiran `_KLAIM` adalah tingkat SATU.
// Tertukar sekali saja berarti layar tingkat 2 membaca — atau lebih buruk, MENULIS — ke
// tabel induknya, dan tidak ada galat basis data apa pun yang muncul: kedua tabel punya
// kolom ID_PROGRESS.
func TestQueries2TargetTheChildTable(t *testing.T) {
	for name, text := range query {
		if !strings.HasPrefix(name, "progress_status2_") {
			continue
		}
		upperCase := strings.ToUpper(text)
		require.Containsf(t, upperCase, "POOLDATA.GCNM_MST_PROGRESS",
			"kueri %q harus menyentuh tabel tingkat 2", name)
		require.NotContainsf(t, upperCase, "GCNM_MST_PROGRESS_KLAIM",
			"kueri %q menyentuh tabel tingkat SATU — periksa ulang nama tabelnya", name)
	}
}

// Tingkat 2 tidak punya UPDATE maupun DELETE, dan itu bukan pekerjaan yang belum selesai.
//
// Seluruh export Pega tidak memuat satu pun pernyataan yang mengubah isi tabel ini setelah
// barisnya tersimpan; alasan lengkapnya ada pada doc comment masterstatusprogres.Repo2.
// Uji ini menjaga keduanya tidak muncul diam-diam kelak tanpa keputusan yang menyertainya.
func TestQueries2NeverChangeStoredRows(t *testing.T) {
	for name, text := range query {
		if !strings.HasPrefix(name, "progress_status2_") {
			continue
		}
		upperCase := strings.ToUpper(text)
		require.NotContainsf(t, upperCase, "UPDATE ", "kueri %q mengubah baris tersimpan", name)
		require.NotContainsf(t, upperCase, "DELETE ", "kueri %q menghapus baris", name)
	}
}

// Nilai selalu lewat parameter binding.
func TestQueries2UseParameterBinding(t *testing.T) {
	for _, name := range []string{"progress_status2_get", "progress_status2_insert"} {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Penyaring ID memangkas spasi padatan kolom CHAR.
//
// Alasannya sama dengan tingkat 1: parameter binding bertipe VARCHAR2, dan perbandingan
// CHAR dengan VARCHAR2 memakai non-padded comparison. Gejala hilangnya TRIM sangat
// menyesatkan — tidak ada galat sama sekali, hanya "tidak ditemukan" untuk setiap baris
// yang sebenarnya ada.
func TestID2FilterHandlesCHARPadding(t *testing.T) {
	text := strings.ToUpper(getQuery("progress_status2_get"))
	require.Contains(t, text, "TRIM(ID_MST)")
	require.NotRegexp(t, `WHERE\s+ID_MST\s*=`, text)
}

// Penurunan nomor baru harus menyerialkan diri.
//
// Tanpa FOR UPDATE, dua penambahan bersamaan dapat menerima ID_MST yang sama — lubang yang
// memang terbuka di sistem lama, karena MAX(ID_MST)+1 di sana dijalankan sebagai kueri
// lepas beberapa langkah sebelum INSERT-nya.
func TestID2FetcherLocksRows(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("progress_status2_list_id_locked")), "FOR UPDATE")
}

// Penyisipan TIDAK menyebut kolom TIPE.
//
// Sistem lama pun tidak menyebutnya, sehingga basis data mengisinya dengan default
// kolomnya sendiri. Menyebutkannya dengan nilai tebakan berarti mengarang; artinya tidak
// diketahui dan DDL-nya belum diterima (R-08).
func TestInsert2NeverWritesKind(t *testing.T) {
	text := strings.ToUpper(getQuery("progress_status2_insert"))
	columns := text[strings.Index(text, "("):strings.Index(text, "VALUES")]
	require.NotContains(t, columns, "TIPE",
		"TIPE dibaca apa adanya dan tidak pernah ditulis")
}

// Kueri pemeriksa tabel tidak mengambil satu baris pun — ia dijalankan terhadap produksi
// pada mode periksa.
func TestCheckTable2FetchesNoRows(t *testing.T) {
	require.Contains(t, getQuery("progress_status2_check_table"), "1 = 0")
}
