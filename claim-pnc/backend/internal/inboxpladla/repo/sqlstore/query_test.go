package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Uji di berkas ini TIDAK menyentuh basis data. Yang diperiksa adalah kesesuaian antara
// teks SQL, daftar alias di query.go, dan pemindai di inboxpladla.go.

func TestEveryQueryNamedInTheCodeExists(t *testing.T) {
	names := []string{
		"reinsurer_codes",
		"list_pla", "list_dla", "list_close",
		"count_pla", "count_dla", "count_close",
		"xol_summary",
	}

	for _, name := range names {
		require.NotPanics(t, func() { query(name) }, "kueri %q tidak ada", name)
		require.NotEmpty(t, strings.TrimSpace(query(name)))
	}
}

// Komentar TIDAK ikut dikirim ke basis data.
func TestCommentsAreStrippedFromTheStatementsSent(t *testing.T) {
	for name, text := range queries {
		for _, line := range strings.Split(text, "\n") {
			require.False(t, strings.HasPrefix(strings.TrimSpace(line), "--"),
				"kueri %q masih membawa baris komentar", name)
		}
	}
}

// Urutan alias di SQL WAJIB sama dengan daftar di query.go.
//
// Ketiga kueri daftar mengembalikan kolom yang sama persis, sehingga satu kolom yang
// tergeser pada SALAH SATUNYA tidak akan terlihat dari membandingkan hasilnya dengan yang
// lain.
func TestAliasOrderInSQLMatchesTheListsInGo(t *testing.T) {
	for _, name := range listQueries {
		require.Equal(t, listColumns, aliasesOf(query(name)),
			"urutan alias kueri daftar %q bergeser", name)
	}

	for _, name := range countQueries {
		require.Equal(t, countColumns, aliasesOf(query(name)),
			"urutan alias kueri ringkas %q bergeser", name)
	}
}

// SETIAP kueri WAJIB menyaring menurut reasuradur pemanggil.
//
// Ini uji terpenting di berkas ini. Kueri yang kehilangan rantai itu akan menampilkan data
// satu mitra kepada mitra lain — dan di layar yang memang dibaca pihak luar, itu bukan
// cacat tampilan melainkan kebocoran yang tidak menghasilkan satu pun galat.
func TestEveryQueryIsScopedToTheCallersReinsurerCode(t *testing.T) {
	for _, name := range reinsurerScopedQueries {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "POOLDATA.T_REINSURER",
			"kueri %q tidak menyaring menurut reasuradur pemanggil", name)
		require.Contains(t, text, "LOGIN",
			"kueri %q tidak mencocokkan login pemanggil", name)
	}
}

// Login DIIKAT, tidak pernah dirangkai.
//
// Kueri XOL lama merangkainya: `"... where login='" + Local.loginreas + "'"` — dan nilai
// yang dirangkai itu ditulis TETAP ke satu nama reasuradur.
func TestTheLoginIsBoundNeverConcatenated(t *testing.T) {
	for _, name := range reinsurerScopedQueries {
		text := query(name)
		require.NotContains(t, text, "TUGU",
			"kueri %q masih memuat nama reasuradur yang ditulis tetap", name)
		require.NotContains(t, text, "{ASIS",
			"kueri %q masih memuat pola perangkaian Pega", name)
	}
}

// Setiap kueri daftar WAJIB mengembalikan jumlah seluruh baris lewat `COUNT(*) OVER ()`.
func TestEveryListQueryReturnsTheTotalRowCount(t *testing.T) {
	for _, name := range listQueries {
		require.Contains(t, strings.ToUpper(query(name)), "COUNT(*) OVER ()",
			"kueri %q tidak mengembalikan jumlah baris", name)
	}
}

// Setiap kueri daftar WAJIB memotong halamannya DI BASIS DATA.
//
// Ketiga kueri Pega tidak berpaginasi sama sekali — mereka memuat SELURUH baris yang cocok
// ke memori. Itu yang dilarang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2.
func TestEveryListQueryPaginatesInTheDatabase(t *testing.T) {
	for _, name := range listQueries {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "OFFSET", "kueri %q tidak memotong halaman", name)
		require.Contains(t, text, "FETCH NEXT", "kueri %q tidak membatasi baris", name)
	}
}

// Kueri ringkas TIDAK boleh berpaginasi.
//
// Ia menghitung seluruh populasi; memotongnya berarti angka yang lebih kecil daripada
// jumlah baris yang sebenarnya cocok — dan tidak ada yang akan menyadarinya.
func TestCountQueriesDoNotPaginate(t *testing.T) {
	for _, name := range countQueries {
		require.NotContains(t, strings.ToUpper(query(name)), "OFFSET",
			"kueri ringkas %q tidak boleh memotong halaman", name)
	}
}

// Ketiga kueri daftar dan ketiga kueri ringkas WAJIB mengecualikan PA dan Travel.
func TestEveryQueryExcludesPersonalAccidentAndTravel(t *testing.T) {
	for _, name := range append(append([]string{}, listQueries...), countQueries...) {
		require.Contains(t, strings.ToUpper(query(name)), "NOT IN ('002', '005')",
			"kueri %q tidak mengecualikan Personal Accident dan Travel", name)
	}
}

// Daftar Close disaring T_PLALIST, BUKAN T_DLALIST.
//
// Ia mudah dikira salah ketik — daftar klaim selesai yang disaring pemberitahuan ESTIMASI
// — tetapi itu memang bentuk `GetPNCList_PLADLAClose`. Uji ini yang akan gagal bila
// seseorang "memperbaikinya".
func TestTheCloseListIsFilteredByPLANotByDLA(t *testing.T) {
	for _, name := range []string{"list_close", "count_close"} {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "POOLDATA.T_PLALIST",
			"kueri %q seharusnya disaring T_PLALIST", name)
		require.NotContains(t, text, "POOLDATA.T_DLALIST",
			"kueri %q tidak menyaring T_DLALIST di Pega", name)
	}
}

// Daftar Close memakai `IN` atas seluruh kode; kedua daftar lain memakai `=`.
//
// Satu kueri, dua aturan berbeda: kolom "No PLA" pada daftar Close tetap memakai kode
// TERTINGGI, sementara penyaring keanggotaannya memakai semuanya.
func TestOnlyTheCloseListMatchesEveryReinsurerCode(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_close")), "REINSCODE IN (SELECT",
		"daftar Close seharusnya mencocokkan SELURUH kode reasuradur")

	for _, name := range []string{"list_pla", "list_dla"} {
		require.NotContains(t, strings.ToUpper(query(name)), "REINSCODE IN (SELECT",
			"kueri %q seharusnya mencocokkan satu kode saja", name)
		require.Contains(t, strings.ToUpper(query(name)), "REINSCODE = (SELECT",
			"kueri %q kehilangan pencocokan kode tertinggi", name)
	}
}

// Hanya daftar DLA yang mengganti kode status menjadi `1139`.
func TestOnlyTheDLAListReplacesTheStatusWith1139(t *testing.T) {
	for _, name := range []string{"list_dla", "count_dla"} {
		require.Contains(t, query(name), "'1139'",
			"kueri %q kehilangan penggantian status menunggu penutupan", name)
	}

	for _, name := range []string{"list_pla", "list_close", "count_pla", "count_close"} {
		require.NotContains(t, query(name), "'1139'",
			"kueri %q tidak boleh mengganti kode status", name)
	}
}

// Hanya daftar DLA yang menggabungkan tabel kerja Pega secara INNER.
//
// Menyeragamkan ketiganya menjadi INNER akan MENGHILANGKAN baris dari dua daftar lain —
// klaim yang belum punya baris tabel kerja. Baris yang hilang dari antrean tidak
// menghasilkan keluhan sampai seseorang menyadari klaimnya tidak pernah muncul.
func TestOnlyTheDLAListJoinsTheWorkTableInner(t *testing.T) {
	require.NotContains(t, strings.ToUpper(query("list_dla")),
		"LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
		"daftar DLA membaca ISPENDINGCLOSE, sehingga gabungannya harus INNER")

	for _, name := range []string{"list_pla", "list_close"} {
		require.Contains(t, strings.ToUpper(query(name)),
			"LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
			"kueri %q harus memakai LEFT JOIN, setara sub-kueri di Pega", name)
	}
}

// Dokumen dianggap terkirim hanya bila KETIGA syaratnya terpenuhi.
//
// `iskirim = '1'` saja tidak cukup: kueri lama menuntut tanggal kirim dan alamat surel
// ikut terisi. Memeriksa yang pertama saja akan memasukkan dokumen yang ditandai terkirim
// tanpa pernah benar-benar dikirim.
func TestAnAdviceCountsAsSentOnlyWhenAllThreeMarkersAreSet(t *testing.T) {
	for _, name := range append(append([]string{}, listQueries...), countQueries...) {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "ISKIRIM = '1'", "kueri %q", name)
		require.Contains(t, text, "TGLKIRIM IS NOT NULL", "kueri %q", name)
	}
}

// Kata kunci pencarian DIIKAT dan dilepaskan wildcard-nya.
func TestSearchKeywordIsBoundAndEscaped(t *testing.T) {
	for _, name := range append(append([]string{}, listQueries...), countQueries...) {
		require.Contains(t, strings.ToUpper(query(name)), "LIKE :2 ESCAPE",
			"kueri %q tidak melepaskan wildcard kata kunci", name)
	}
}

// Tanggal dikembalikan sebagai TANGGAL, bukan dibungkus `TO_CHAR`.
//
// Kueri lama membungkus keduanya, sehingga pengurutan tanggal menjadi pengurutan TEKS dan
// penyaringan rentang tidak dapat memakai index (`09-DATABASE-STRATEGY.md` §3.2).
func TestDatesAreReturnedAsDatesNotAsFormattedText(t *testing.T) {
	for _, name := range listQueries {
		require.NotContains(t, strings.ToUpper(query(name)), "TO_CHAR(",
			"kueri %q masih memformat tanggal di dalam SQL", name)
	}
}

// Jumlah bind tiap kueri WAJIB sama dengan yang disusun filterArgs dan countArgs.
//
// Selisih satu saja menghasilkan galat bind yang menyebut NOMOR, bukan menyebut kueri
// mana yang rusak.
func TestEachQueryUsesTheExpectedNumberOfBinds(t *testing.T) {
	expected := map[string]int{
		"reinsurer_codes": 1,
		"list_pla":        7, // 2 penyaring + 3 login + offset + ukuran
		"list_dla":        6, // 2 penyaring + 2 login + offset + ukuran
		"list_close":      6, // 2 penyaring + 2 login + offset + ukuran
		"count_pla":       4, // 2 penyaring + 2 login
		"count_dla":       3, // 2 penyaring + 1 login
		"count_close":     3, // 2 penyaring + 1 login
		"xol_summary":     2, // login dua kali, satu per bagian union
	}

	for name, count := range expected {
		require.Equal(t, count, highestBind(query(name)),
			"jumlah bind kueri %q berbeda dari yang disusun di Go", name)
	}
}

// Kueri ringkas WAJIB mengelompokkan menurut kode status.
func TestCountQueriesGroupByStatus(t *testing.T) {
	for _, name := range countQueries {
		require.Contains(t, strings.ToUpper(query(name)), "GROUP BY",
			"kueri ringkas %q tidak mengelompokkan apa pun", name)
	}
}

// aliasesOf mengambil alias kolom pada klausa SELECT terluar, berurutan.
func aliasesOf(text string) []string {
	pattern := regexp.MustCompile(`(?i)^AS\s+([A-Za-z_][A-Za-z0-9_]*)`)

	found := []string{}
	depth := 0

	for index := 0; index < len(text); index++ {
		switch text[index] {
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

		if index > 0 && !isWordBoundary(text[index-1]) {
			continue
		}

		rest := text[index:]
		if match := pattern.FindStringSubmatch(rest); match != nil {
			found = append(found, strings.ToUpper(match[1]))
			index += len(match[0]) - 1
			continue
		}

		if len(rest) >= 5 && strings.EqualFold(rest[:4], "FROM") &&
			isWordBoundary(rest[4]) {
			return found
		}
	}

	return found
}

// isWordBoundary menyatakan sebuah bita bukan bagian dari kata.
func isWordBoundary(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '_':
		return false
	default:
		return true
	}
}

// highestBind mengembalikan nomor bind tertinggi yang dipakai sebuah kueri.
func highestBind(text string) int {
	pattern := regexp.MustCompile(`:(\d+)`)

	highest := 0
	for _, match := range pattern.FindAllStringSubmatch(text, -1) {
		if number, err := strconv.Atoi(match[1]); err == nil && number > highest {
			highest = number
		}
	}
	return highest
}
