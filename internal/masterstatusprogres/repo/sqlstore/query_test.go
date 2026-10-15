package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji
// ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"progress_status_list",
		"progress_status_get",
		"progress_status_list_id_locked",
		"progress_status_insert",
		"progress_status_update",
		"progress_status_check_table",
	}
	for _, name := range usedNames {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = getQuery("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (D-20) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *":  "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":      "pakai COALESCE",
		"SYSDATE":   "pakai CURRENT_TIMESTAMP",
		"DECODE(":   "pakai CASE WHEN",
		"ROWNUM":    "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":    "pakai POSITION",
		"LISTAGG(":  "pakai STRING_AGG",
		"TO_CHAR(":  "pemformatan tanggal dan angka dilakukan di Go",
		"FROM DUAL": "tidak ada padanannya di PostgreSQL",
	}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, upperCase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, upperCase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterised := []string{
		"progress_status_get",
		"progress_status_insert",
		"progress_status_update",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// ID_PROGRESS bertipe CHAR berlebar tetap (ditetapkan Work Owner 2026-09-17), sehingga
// nilainya dipadatkan spasi: "01" tersimpan sebagai "01 ".
//
// Oracle membandingkan CHAR dengan CHAR memakai blank-padded comparison — spasi di ujung
// diabaikan. Tetapi parameter binding bertipe VARCHAR2, dan CHAR lawan VARCHAR2 memakai
// NON-padded comparison: "01 " tidak sama dengan "01". Tanpa TRIM, kedua kueri ini tidak
// pernah menemukan barisnya.
//
// Uji ini menjaga TRIM tidak hilang saat seseorang "merapikan" kueri kelak, karena
// gejalanya sangat menyesatkan: tidak ada galat basis data sama sekali — penyuntingan
// hanya melaporkan "tidak ditemukan" untuk setiap baris yang sebenarnya ada.
func TestIDFilterHandlesCHARPadding(t *testing.T) {
	filtersID := []string{
		"progress_status_get",
		"progress_status_update",
	}
	for _, name := range filtersID {
		t.Run(name, func(t *testing.T) {
			text := strings.ToUpper(getQuery(name))
			require.Contains(t, text, "TRIM(ID_PROGRESS)",
				"penyaring ID wajib memangkas spasi padatan kolom CHAR")
			require.NotRegexp(t, `WHERE\s+ID_PROGRESS\s*=`, text,
				"perbandingan langsung tanpa TRIM tidak akan menemukan baris apa pun")
		})
	}
}

// Kolom ID_PROGRESS tidak boleh ikut di-SET saat memperbarui: ia kunci baris, dirujuk
// GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1 dan GCNM_MST_PROGRESS.ID_PROGRESS pada data klaim
// yang sudah berjalan.
func TestUpdateNeverChangesRowKey(t *testing.T) {
	text := strings.ToUpper(getQuery("progress_status_update"))
	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	require.NotContains(t, setClause, "ID_PROGRESS",
		"ID_PROGRESS hanya boleh menyaring di WHERE, tidak pernah di-SET")
}

// Penurunan nomor baru harus menyerialkan diri; tanpa FOR UPDATE, dua penambahan
// bersamaan dapat menerima nomor yang sama.
func TestIDFetcherLocksRows(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("progress_status_list_id_locked")), "FOR UPDATE")
}

// Kueri pemeriksa tabel tidak boleh mengambil satu baris pun — ia dijalankan terhadap
// produksi pada mode periksa.
func TestCheckTableFetchesNoRows(t *testing.T) {
	require.Contains(t, getQuery("progress_status_check_table"), "1 = 0")
}
