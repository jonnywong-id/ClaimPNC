package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcloseclaim"
)

// TestMissingQueryPanics memastikan nama kueri yang salah ketik terlihat segera.
func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

// TestEveryQueryExists memastikan ketiga kueri yang dipakai kode benar-benar ada.
//
// Tanpa uji ini, kueri yang namanya salah ketik baru ketahuan saat permintaan pertama
// datang — dan yang dilihat pengguna adalah panik, bukan galat yang dapat dibaca.
func TestEveryQueryExists(t *testing.T) {
	for _, name := range []string{"close_claim_list", "close_claim_count", "close_claim_exists"} {
		require.NotEmptyf(t, query(name), "kueri %s kosong", name)
	}
}

// TestFilterArgsMatchesBindCount adalah uji yang paling menentukan di berkas ini.
//
// Ia menghubungkan tiga hal yang mudah menyimpang diam-diam: banyaknya penanda di dalam SQL,
// nilai konstanta filterBindCount, dan banyaknya argumen yang benar-benar disiapkan.
//
// Argumen yang kurang menghasilkan **ORA-01008 not all variables bound** pada permintaan
// pertama yang datang — bukan saat kode ditulis, dan bukan pada mesin pengembang yang tidak
// punya Oracle.
func TestFilterArgsMatchesBindCount(t *testing.T) {
	args := filterArgs(inboxcloseclaim.Filter{}.Normalize())
	require.Len(t, args, filterBindCount,
		"filterArgs menyiapkan %d argumen, filterBindCount menyebut %d", len(args), filterBindCount)

	require.Equal(t, filterBindCount, highestBind(t, query("close_claim_count")),
		"close_claim_count memakai nomor bind tertinggi yang tidak sama dengan filterBindCount")

	// Kueri daftar memakai dua penanda tambahan untuk paginasi.
	require.Equal(t, filterBindCount+2, highestBind(t, query("close_claim_list")),
		"close_claim_list harus memakai dua penanda tambahan: offset dan limit")
}

// TestBindMarkersAreUniqueAndAscending menjaga aturan penomoran yang membuat kueri ini
// bekerja pada penafsiran driver mana pun.
//
// Setiap KEMUNCULAN penanda punya nomornya sendiri, dan nomornya menaik sesuai urutan
// kemunculannya di dalam teks. Dengan begitu, driver yang mengikat menurut nomor dan driver
// yang mengikat menurut urutan kemunculan menghasilkan pengikatan yang SAMA.
//
// Melanggarnya tidak menghasilkan galat kompilasi maupun uji yang gagal di tempat lain —
// hanya nilai yang masuk ke penyaring yang salah, dan itu terlihat sebagai daftar yang
// isinya keliru tanpa satu pun pesan.
func TestBindMarkersAreUniqueAndAscending(t *testing.T) {
	for _, name := range []string{"close_claim_list", "close_claim_count"} {
		numbers := bindNumbers(t, query(name))
		require.NotEmpty(t, numbers, "kueri %s tidak punya penanda parameter", name)

		seen := map[int]bool{}
		previous := 0
		for _, number := range numbers {
			require.Falsef(t, seen[number],
				"kueri %s memakai :%d lebih dari sekali — setiap kemunculan wajib punya nomornya sendiri",
				name, number)
			require.Greaterf(t, number, previous,
				"kueri %s memakai :%d setelah :%d — nomor wajib menaik sesuai urutan kemunculan",
				name, number, previous)
			seen[number] = true
			previous = number
		}
	}
}

// TestListAndCountShareTheSameWhere menjaga keputusan Work Owner 2026-09-23.
//
// Kueri hitung Pega kehilangan penyaring Status Bayar, sehingga totalnya tidak cocok dengan
// barisnya. Di sini keduanya disamakan — dan uji inilah yang menjaganya tetap sama, karena
// penyimpangannya TIDAK menghasilkan galat apa pun: pengguna membaca "247 baris cocok" lalu
// menemukan jumlah yang lain.
func TestListAndCountShareTheSameWhere(t *testing.T) {
	list := whereClause(t, query("close_claim_list"))
	count := whereClause(t, query("close_claim_count"))

	require.Equal(t, count, list,
		"syarat WHERE close_claim_list dan close_claim_count berbeda — totalnya akan menyimpang dari barisnya")
}

// TestPaymentFilterCarriesTheStatusCode memastikan kode '1163' datang dari konstanta domain,
// bukan tertanam di dalam teks SQL.
//
// `D-15` menetapkan tidak ada nilai bisnis yang boleh di-hardcode berulang kali. Di sini ia
// juga soal ketelusuran: bila kelak kode "lunas" berubah, yang disunting satu konstanta —
// bukan empat tempat di dalam dua kueri.
func TestPaymentFilterCarriesTheStatusCode(t *testing.T) {
	args := filterArgs(inboxcloseclaim.Filter{
		Payment: inboxcloseclaim.PaymentPaid,
	}.Normalize())

	require.Equal(t, inboxcloseclaim.KodeStatusLunas, args[19], "penanda :20 harus membawa kode lunas")
	require.Equal(t, inboxcloseclaim.KodeStatusLunas, args[21], "penanda :22 harus membawa kode lunas")

	require.NotContains(t, query("close_claim_list"), "'1163'",
		"kode status lunas tidak boleh tertanam di dalam teks SQL")
}

// TestBusinessLineRepeatedFiveTimes menjaga bahwa nilai lini bisnis benar-benar dikirim pada
// kelima kemunculannya.
//
// Satu saja yang terlewat membuat cabang itu tidak pernah benar — dan akibatnya BUKAN galat
// melainkan daftar yang kehilangan seluruh baris lini tersebut.
func TestBusinessLineRepeatedFiveTimes(t *testing.T) {
	args := filterArgs(inboxcloseclaim.Filter{
		Business: inboxcloseclaim.BusinessBonding,
	}.Normalize())

	for _, index := range []int{9, 10, 11, 12, 13} {
		require.Equalf(t, string(inboxcloseclaim.BusinessBonding), args[index],
			"penanda :%d tidak membawa nilai lini bisnis", index+1)
	}
}

// TestEmptyFilterSendsNulls memastikan penyaring yang tidak diisi dikirim sebagai NULL.
//
// Pola "NULL berarti tidak menyaring" itulah yang membuat satu teks SQL melayani seluruh
// kombinasi. Mengirim teks kosong akan membuat `LIKE '%%'` — yang kebetulan juga cocok
// dengan semuanya, tetapi memaksa basis data memindai setiap baris.
func TestEmptyFilterSendsNulls(t *testing.T) {
	args := filterArgs(inboxcloseclaim.Filter{}.Normalize())

	for _, index := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 14, 17} {
		require.Nilf(t, args[index], "penanda :%d seharusnya NULL saat penyaringnya kosong", index+1)
	}

	// Lini bisnis TIDAK pernah NULL: kosong berarti 'ALL', dan kueri membandingkannya
	// dengan teks 'ALL', bukan dengan NULL.
	require.Equal(t, string(inboxcloseclaim.BusinessAll), args[9])
}

// TestLikePatternEscapesWildcards memastikan pencarian "100%" tidak berubah menjadi pola
// yang mencocokkan apa saja.
func TestLikePatternEscapesWildcards(t *testing.T) {
	require.Equal(t, `%100\%%`, likePattern("100%"))
	require.Equal(t, `%A\_B%`, likePattern("a_b"))
	require.Equal(t, "", likePattern("   "))

	// Diseragamkan huruf besar karena sisi SQL memakai UPPER(...).
	require.Equal(t, "%PNC-1%", likePattern("pnc-1"))
}

// TestNoForbiddenSQLPatterns menjaga aturan §4.3 dan §6 Technical Strategy.
//
// Pemeriksaan pola SQL terlarang wajib berjalan di CI dan memblokir merge. Uji ini
// menjalankannya pada berkas modul ini, sehingga pelanggarannya tertangkap di sini lebih
// dulu — bukan di CI setelah review selesai.
func TestNoForbiddenSQLPatterns(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *":  "sebutkan nama kolom",
		"NVL(":      "pakai COALESCE",
		"SYSDATE":   "pakai CURRENT_TIMESTAMP",
		"DECODE(":   "pakai CASE WHEN",
		"ROWNUM":    "pakai OFFSET … FETCH NEXT",
		"TO_CHAR(":  "format tanggal dan angka di Go",
		"INSTR(":    "pakai POSITION",
		"LISTAGG(":  "pakai STRING_AGG",
		"FROM DUAL": "hilangkan klausa FROM",
		"{ASIS:":    "perangkaian SQL dilarang — pakai parameter binding",
	}

	for name, statement := range queries {
		upper := strings.ToUpper(statement)
		for pattern, advice := range forbidden {
			require.NotContainsf(t, upper, strings.ToUpper(pattern),
				"kueri %s memakai %q — %s", name, pattern, advice)
		}
	}
}

// highestBind mengembalikan nomor penanda tertinggi di dalam sebuah pernyataan.
func highestBind(t *testing.T, statement string) int {
	t.Helper()

	highest := 0
	for _, number := range bindNumbers(t, statement) {
		if number > highest {
			highest = number
		}
	}
	return highest
}

// bindNumbers mengembalikan nomor setiap penanda sesuai urutan kemunculannya.
func bindNumbers(t *testing.T, statement string) []int {
	t.Helper()

	pattern := regexp.MustCompile(`:(\d+)`)
	matches := pattern.FindAllStringSubmatch(statement, -1)

	result := make([]int, 0, len(matches))
	for _, match := range matches {
		number, err := strconv.Atoi(match[1])
		require.NoError(t, err)
		result = append(result, number)
	}
	return result
}

// whereClause memotong bagian WHERE TERLUAR sebuah pernyataan, membuang ORDER BY dan
// paginasi.
//
// # Kenapa kedalaman kurung ikut dihitung
//
// Kedua kueri memuat `WHERE` DI DALAM subkueri — label status klaim dicari ke
// `V_STS_CLAIM`, dan penanda transfer kasir memeriksa `T_CLAIM_ADJUSTMENT`. Mencari
// kemunculan "WHERE" yang pertama akan menemukan salah satu dari keduanya, bukan syarat
// yang hendak dibandingkan.
//
// Versi pertama helper ini melakukan persis itu, dan ujinya gagal — bukan karena SQL-nya
// berbeda, melainkan karena helper-nya salah membaca. Kegagalan itu dibiarkan mengajari:
// yang diperbaiki helper-nya, bukan ujinya dilonggarkan.
//
// Spasi diseragamkan supaya perbandingannya menguji SYARATNYA, bukan tata letaknya.
func whereClause(t *testing.T, statement string) string {
	t.Helper()

	start := outerKeyword(statement, "WHERE")
	require.GreaterOrEqual(t, start, 0, "pernyataan tanpa klausa WHERE di kedalaman nol")

	clause := statement[start:]
	if end := outerKeyword(clause, "ORDER BY"); end >= 0 {
		clause = clause[:end]
	}
	return strings.Join(strings.Fields(clause), " ")
}

// outerKeyword mencari kata kunci pada kedalaman kurung NOL, atau -1 bila tidak ada.
func outerKeyword(statement, keyword string) int {
	upper := strings.ToUpper(statement)
	target := strings.ToUpper(keyword)

	depth := 0
	for i := 0; i < len(upper); i++ {
		switch upper[i] {
		case '(':
			depth++
			continue
		case ')':
			depth--
			continue
		}
		if depth != 0 {
			continue
		}
		if strings.HasPrefix(upper[i:], target) {
			return i
		}
	}
	return -1
}
