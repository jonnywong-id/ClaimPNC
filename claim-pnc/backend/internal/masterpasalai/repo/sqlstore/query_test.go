package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasalai"
)

// usedQueries adalah seluruh nama kueri yang dipanggil kode modul ini.
var usedQueries = []string{
	"clause_count",
	"clause_count_search",
	"clause_list",
	"clause_list_search",
	"clause_check_table",
}

// TestEveryUsedQueryExists menangkap salah ketik nama kueri saat uji, bukan saat runtime.
func TestEveryUsedQueryExists(t *testing.T) {
	for _, name := range usedQueries {
		require.NotPanicsf(t, func() { getQuery(name) }, "kueri %q tidak ada", name)
		require.NotEmptyf(t, getQuery(name), "kueri %q kosong", name)
	}
}

// TestNoStrayQuery membuktikan tidak ada kueri yang ditulis lalu tidak pernah dipakai.
//
// Kueri yang tidak dipanggil siapa pun tidak pernah diuji terhadap basis data, sehingga ia
// tetap tampak benar sampai seseorang memakainya.
func TestNoStrayQuery(t *testing.T) {
	known := map[string]bool{}
	for _, name := range usedQueries {
		known[name] = true
	}
	for name := range query {
		require.Truef(t, known[name], "kueri %q ada di berkas .sql tetapi tidak dipakai", name)
	}
}

// TestQueriesTargetTheRightTable membuktikan seluruhnya menembak tabel yang benar.
//
// Satu kueri yang menembak tabel lain akan menampilkan data yang masuk akal dari tempat yang
// salah — kelas cacat yang tidak terlihat sebagai galat.
func TestQueriesTargetTheRightTable(t *testing.T) {
	for _, name := range usedQueries {
		require.Containsf(t, strings.ToUpper(getQuery(name)), "POOLDATA.MST_PASAL_AI",
			"kueri %q tidak menembak POOLDATA.MST_PASAL_AI", name)
	}
}

// TestModuleNeverWrites membuktikan modul ini benar-benar baca-saja.
//
// Layar lamanya tidak punya satu pun jalur tulis (`pyEditingMode = readOnly`), dan tidak ada
// rule di export yang menulisi tabelnya. Uji ini yang menjaga jalur tulis tidak masuk
// diam-diam lewat kueri baru.
func TestModuleNeverWrites(t *testing.T) {
	forbidden := []string{"INSERT", "UPDATE", "DELETE", "MERGE", "TRUNCATE"}
	for name, text := range query {
		upper := strings.ToUpper(text)
		for _, word := range forbidden {
			require.NotContainsf(t, upper, word,
				"kueri %q memuat %s — modul ini baca-saja", name, word)
		}
	}
}

// TestNoForbiddenSQLPattern menjaga janji SQL portabel (`D-20`).
//
// `ROWNUM` disebut khusus karena kueri Pega aslinya MEMAKAINYA — tiga tingkat subquery
// dengan `RN >= FirstRow AND RN <= LastRow`. Menyalinnya apa adanya akan membuat modul ini
// gagal di PostgreSQL, dan kegagalannya baru terlihat saat cutover.
func TestNoForbiddenSQLPattern(t *testing.T) {
	forbidden := []string{"ROWNUM", "NVL(", "SYSDATE", "DECODE(", "TO_CHAR(", "FROM DUAL"}
	for name, text := range query {
		upper := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContainsf(t, upper, pattern,
				"kueri %q memuat %s — tidak portabel (D-20)", name, pattern)
		}
		require.NotContainsf(t, text, "SELECT *",
			"kueri %q memakai SELECT * — kolom harus disebut namanya", name)
	}
}

// TestNoClipboardStringConcatenation membuktikan pola `{ASIS:…}` tidak ikut terbawa.
//
// Kueri lama menempelkan SELURUH klausa WHERE sebagai teks yang disusun activity
// (`{ASIS:TempQuery.AlasanKlaim}`), dan juga menempelkan `{Pagination.FirstRow}` serta
// `{Pagination.LastRow}`. Ketiganya adalah celah SQL injection.
func TestNoClipboardStringConcatenation(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, text, "{ASIS:",
			"kueri %q masih memakai perangkaian klipboard", name)
		require.NotContainsf(t, text, "Pagination.",
			"kueri %q masih menempel nilai paginasi sebagai teks", name)
	}
}

// TestPaginationUsesBoundParameters membuktikan jendela halaman memakai parameter terikat.
func TestPaginationUsesBoundParameters(t *testing.T) {
	for _, name := range []string{"clause_list", "clause_list_search"} {
		text := strings.ToUpper(getQuery(name))
		require.Containsf(t, text, "OFFSET", "kueri %q tidak memaginasi", name)
		require.Containsf(t, text, "FETCH NEXT", "kueri %q tidak membatasi halaman", name)
	}
}

// TestBothListQueriesOrderByTheSameColumn membuktikan urutannya tidak berbeda antarkueri.
//
// `ORDER BY WP_ID` disalin dari kueri lama. Bila satu kueri mengurutkan dan yang lain tidak,
// hasil pencarian dan hasil tanpa pencarian akan berurutan berbeda — dan paginasinya menjadi
// tidak dapat dipercaya pada salah satunya.
func TestBothListQueriesOrderByTheSameColumn(t *testing.T) {
	for _, name := range []string{"clause_list", "clause_list_search"} {
		require.Containsf(t, strings.ToUpper(getQuery(name)), "ORDER BY WP_ID",
			"kueri %q tidak mengurutkan menurut WP_ID", name)
	}
}

// TestCountAndListShareTheSameFilter membuktikan penyaring cacah sama dengan penyaring daftar.
//
// Bila keduanya berbeda, paginator akan melaporkan halaman yang isinya kosong — atau
// menyembunyikan baris yang sebenarnya cocok. Keduanya cacat yang hanya muncul setelah
// datanya banyak.
func TestCountAndListShareTheSameFilter(t *testing.T) {
	require.Equal(t,
		whereClauseOf(getQuery("clause_count_search")),
		whereClauseOf(getQuery("clause_list_search")),
		"penyaring cacah dan penyaring daftar berbeda")
}

// whereClauseOf mengambil klausa WHERE sebuah kueri, tanpa ORDER BY dan paginasinya.
func whereClauseOf(text string) string {
	upper := strings.ToUpper(text)
	start := strings.Index(upper, "WHERE")
	if start < 0 {
		return ""
	}
	clause := upper[start:]
	if stop := strings.Index(clause, "ORDER BY"); stop >= 0 {
		clause = clause[:stop]
	}
	return strings.Join(strings.Fields(clause), " ")
}

// TestSearchSpansAllThreeColumns membuktikan ketiga kolom ikut dicari, digabung OR.
//
// Padanan `Activity/GetListPasalAI_act.xml:548`. Melewatkan satu kolom akan membuat
// pencarian menemukan lebih sedikit daripada layar lama, tanpa satu pun tanda.
func TestSearchSpansAllThreeColumns(t *testing.T) {
	for _, name := range []string{"clause_count_search", "clause_list_search"} {
		text := strings.ToUpper(getQuery(name))
		for _, column := range []string{"WP_PASAL", "WP_AYAT", "WP_KEJADIAN"} {
			require.Containsf(t, text, "UPPER("+column+") LIKE",
				"kueri %q tidak mencari di kolom %s", name, column)
		}
		require.Equalf(t, 2, strings.Count(text, " OR "),
			"kueri %q seharusnya menggabungkan ketiga kolom dengan OR", name)
	}
}

// TestSearchIsCaseInsensitiveOnBothSides membuktikan UPPER dipasang di kedua sisi.
//
// Kata kunci dinaikkan di Go (likePattern) dan kolomnya dinaikkan di SQL. Bila hanya satu
// sisi, pencarian menjadi peka huruf di Oracle — dan adapter memori yang tidak peka huruf
// akan meluluskan uji yang gagal di produksi.
func TestSearchIsCaseInsensitiveOnBothSides(t *testing.T) {
	require.Equal(t, "%KEBAKARAN%", likePattern(" kebakaran "))
	require.Contains(t, strings.ToUpper(getQuery("clause_list_search")), "UPPER(WP_PASAL)")
}

// TestLikePatternEscapesWildcard membuktikan `%` dan `_` yang diketik pengguna diloloskan.
func TestLikePatternEscapesWildcard(t *testing.T) {
	require.Equal(t, `%100\%%`, likePattern("100%"))
	require.Equal(t, `%A\_B%`, likePattern("a_b"))
	require.Equal(t, `%C\\D%`, likePattern(`c\d`))

	// `ESCAPE '\'` wajib disebut eksplisit: Oracle tidak punya karakter pelolos bawaan.
	for _, name := range []string{"clause_count_search", "clause_list_search"} {
		require.Containsf(t, getQuery(name), `ESCAPE '\'`,
			"kueri %q tidak menyebut ESCAPE", name)
	}
}

// TestReaderQueriesShareColumnOrder membuktikan ketiga pembaca menyebut kolom yang sama pada
// urutan yang sama — syarat agar satu scanRow cukup untuk ketiganya.
func TestReaderQueriesShareColumnOrder(t *testing.T) {
	want := []string{"WP_ID", "WP_PASAL", "WP_AYAT", "WP_KEJADIAN"}
	for _, name := range []string{"clause_list", "clause_list_search", "clause_check_table"} {
		require.Equalf(t, want, selectedColumns(getQuery(name)),
			"urutan kolom kueri %q berbeda dari yang dibaca scanRow", name)
	}
}

// selectedColumns mengambil daftar kolom pada klausa SELECT sebuah kueri.
func selectedColumns(text string) []string {
	upper := strings.ToUpper(text)
	start := strings.Index(upper, "SELECT")
	stop := strings.Index(upper, "FROM")
	if start < 0 || stop < 0 {
		return nil
	}
	body := upper[start+len("SELECT") : stop]

	var result []string
	for _, part := range strings.Split(body, ",") {
		if clean := strings.TrimSpace(part); clean != "" {
			result = append(result, clean)
		}
	}
	return result
}

// TestPageSizeIsNotHardcodedInSQL membuktikan ukuran halaman datang dari domain.
//
// Menanamkan 25 di dalam teks SQL berarti angkanya hidup di dua tempat, dan keduanya akan
// berpisah begitu salah satunya disunting.
func TestPageSizeIsNotHardcodedInSQL(t *testing.T) {
	number := regexp.MustCompile(`\b25\b`)
	for name, text := range query {
		require.Falsef(t, number.MatchString(text),
			"kueri %q menanamkan ukuran halaman; ia milik masterpasalai.PageSize", name)
	}
	require.Equal(t, 25, masterpasalai.PageSize)
}
