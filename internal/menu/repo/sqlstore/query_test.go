package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEveryUsedQueryExists(t *testing.T) {
	used := []string{
		"menu_list",
		"menu_app_exists",
		"menu_groups_of_login",
		"menu_authorized_ids",
		"menu_check_table",
	}
	for _, name := range used {
		require.NotPanics(t, func() { _ = getQuery(name) }, "kueri %q dipakai kode tetapi tidak ada", name)
		require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = getQuery("kueri_yang_tidak_pernah_ada") })
}

// D-20: satu set SQL yang berjalan di Oracle 19c DAN PostgreSQL 17+. Pola di bawah
// punya padanan portabel, dan memakainya di sini akan memecah janji itu tanpa satu pun
// tanda sampai cutover.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"NVL(":      "pakai COALESCE",
		"SYSDATE":   "pakai CURRENT_TIMESTAMP",
		"DECODE(":   "pakai CASE WHEN",
		"ROWNUM":    "pakai OFFSET … FETCH NEXT … ROWS ONLY",
		"INSTR(":    "pakai POSITION",
		"LISTAGG(":  "pakai STRING_AGG",
		"TO_CHAR(":  "pemformatan dilakukan di Go",
		"FROM DUAL": "tidak ada padanannya di PostgreSQL",
		"(+)":       "pakai LEFT JOIN gaya ANSI",
		"SELECT *":  "kolom selalu disebut namanya",
	}

	for name, text := range query {
		upper := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, upper, strings.ToUpper(pattern),
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
	}
}

// Celah `{ASIS:...}` warisan ditutup dengan parameter binding tanpa perkecualian
// (`03-CURRENT-ARCHITECTURE.md` §4.5). Setiap kueri yang menerima nilai harus memakai
// penanda parameter, bukan nilai yang dirangkai ke dalam teksnya.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterised := []string{"menu_list", "menu_app_exists", "menu_groups_of_login"}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1", "kueri %q tidak berparameter", name)
	}
}

// expandSubjects menyisipkan PENANDA parameter, tidak pernah nilainya. Uji ini yang
// menjaga pernyataan itu tetap benar bila fungsinya kelak disunting.
func TestExpandSubjectsInsertsPlaceholdersNeverValues(t *testing.T) {
	statement, args := expandSubjects(getQuery("menu_authorized_ids"), "CLAIM PNC", []string{"IT", "JONNY"})

	require.Contains(t, statement, "IN (:2, :3)")
	require.NotContains(t, statement, subjectsMarker)

	// Tidak satu pun nilai boleh muncul di dalam teks SQL-nya.
	require.NotContains(t, statement, "JONNY")
	require.NotContains(t, statement, "CLAIM PNC")

	require.Equal(t, []any{"CLAIM PNC", "IT", "JONNY"}, args)
}

// Perbandingannya UPPER(TRIM(...)) di sisi SQL. Bila sisi Go tidak ikut diseragamkan,
// baris yang sah tidak pernah cocok dan menunya hanya kosong — kegagalan yang diam.
func TestExpandSubjectsNormalisesValues(t *testing.T) {
	_, args := expandSubjects(getQuery("menu_authorized_ids"), "CLAIM PNC", []string{" it ", "jonny"})

	require.Equal(t, []any{"CLAIM PNC", "IT", "JONNY"}, args)
}

// Aplikasi dan subjek sama-sama dibandingkan setelah dipangkas dan diseragamkan
// hurufnya, di KEDUA sisi.
func TestComparisonsAreCaseAndSpaceInsensitive(t *testing.T) {
	for _, name := range []string{"menu_list", "menu_app_exists", "menu_groups_of_login", "menu_authorized_ids"} {
		require.Containsf(t, strings.ToUpper(getQuery(name)), "UPPER(TRIM(",
			"kueri %q membandingkan teks tanpa menyeragamkannya", name)
	}
}

// Modul ini hanya membaca. Pernyataan yang menulis akan membuatnya menjadi penulis
// kedua atas tabel yang belum punya pemilik tunggal (`P-1`).
func TestNoQueryWrites(t *testing.T) {
	for name, text := range query {
		upper := strings.ToUpper(text)
		for _, verb := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE "} {
			require.NotContainsf(t, upper, verb, "kueri %q menulis (%s)", name, strings.TrimSpace(verb))
		}
	}
}

// Pemeriksaan tabel tidak boleh mengambil satu baris pun, supaya aman dijalankan
// terhadap produksi.
func TestCheckTableFetchesNoRows(t *testing.T) {
	require.Contains(t, strings.ReplaceAll(getQuery("menu_check_table"), " ", ""), "1=0")
}
