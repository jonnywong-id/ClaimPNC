package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Uji di berkas ini TIDAK menyentuh basis data. Yang diperiksa adalah kesesuaian antara
// teks SQL, daftar alias di query.go, dan pemindai di inboxpladlapredla.go — tiga tempat
// yang mudah bergeser satu sama lain tanpa ada yang gagal sampai kuerinya benar-benar
// dijalankan terhadap Oracle.

func TestEveryQueryNamedInTheCodeExists(t *testing.T) {
	names := []string{
		"list_pla", "list_dla", "list_pre_dla",
		"documents_pla", "documents_dla",
		"claim_exists", "print_pre_dla",
	}

	for _, name := range names {
		require.NotPanics(t, func() { query(name) }, "kueri %q tidak ada", name)
		require.NotEmpty(t, strings.TrimSpace(query(name)))
	}
}

// Komentar TIDAK ikut dikirim ke basis data: yang dibaca DBA adalah berkasnya, bukan jejak
// di jurnal basis data.
func TestCommentsAreStrippedFromTheStatementsSent(t *testing.T) {
	for name, text := range queries {
		for _, line := range strings.Split(text, "\n") {
			require.False(t, strings.HasPrefix(strings.TrimSpace(line), "--"),
				"kueri %q masih membawa baris komentar", name)
		}
	}
}

// Urutan alias di SQL WAJIB sama dengan daftar di query.go, dan daftar itu pula yang
// menjadi urutan pemindai. Satu kolom yang tersisip di tengah akan menggeser SELURUH isi
// baris tanpa satu pun galat — kelas cacat yang tidak menghasilkan tanda apa pun.
//
// Di modul ini bahayanya nyata: ketiga kueri daftar mengembalikan kolom yang sama persis,
// sehingga satu kolom yang tergeser pada SALAH SATUNYA tidak akan terlihat dari
// membandingkan hasilnya dengan yang lain.
func TestAliasOrderInSQLMatchesTheListsInGo(t *testing.T) {
	for _, name := range listQueries {
		require.Equal(t, listColumns, aliasesOf(query(name)),
			"urutan alias kueri daftar %q bergeser", name)
	}

	for _, name := range documentQueries {
		require.Equal(t, documentColumns, aliasesOf(query(name)),
			"urutan alias kueri rincian %q bergeser", name)
	}

	require.Equal(t, printColumns, aliasesOf(query("print_pre_dla")),
		"urutan alias kueri panel Print Pre DLA bergeser")
}

// Setiap kueri daftar WAJIB mengembalikan jumlah seluruh baris lewat `COUNT(*) OVER ()`.
//
// Tanpa itu, paginasi kehilangan totalnya dan layar menggambar "halaman 1 dari 1" pada
// antrean yang sebenarnya punya ribuan baris.
//
// Ia sekaligus menggantikan ketiga kueri penghitung terpisah Pega (`CountPNCList_PLA` dan
// saudaranya), yang harus dijaga tetap sejalan dengan kueri daftarnya — dua tempat untuk
// satu penyaring.
func TestEveryListQueryReturnsTheTotalRowCount(t *testing.T) {
	for _, name := range listQueries {
		require.Contains(t, strings.ToUpper(query(name)), "COUNT(*) OVER ()",
			"kueri %q tidak mengembalikan jumlah baris", name)
	}
}

// Setiap kueri daftar WAJIB memotong halamannya DI BASIS DATA.
//
// Menariknya ke aplikasi lebih dulu berarti seluruh baris yang cocok melewati memori,
// dan itu persis yang dilarang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2.
func TestEveryListQueryPaginatesInTheDatabase(t *testing.T) {
	for _, name := range listQueries {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "OFFSET", "kueri %q tidak memotong halaman", name)
		require.Contains(t, text, "FETCH NEXT", "kueri %q tidak membatasi baris", name)
	}
}

// Ketiga kueri daftar WAJIB mengecualikan Personal Accident dan Travel.
//
// Kueri yang kehilangan penyaring ini akan menampilkan klaim dari dua lini yang layar ini
// memang tidak layani — tanpa satu pun galat, dan tanpa apa pun yang membedakannya dari
// baris yang sah.
func TestEveryListQueryExcludesPersonalAccidentAndTravel(t *testing.T) {
	for _, name := range businessFilterQueries {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "GROUPPANEL <> '002'",
			"kueri %q tidak mengecualikan Personal Accident", name)
		require.Contains(t, text, "GROUPPANEL <> '005'",
			"kueri %q tidak mengecualikan Travel", name)
	}
}

// Dua penyaring tambahan tab DLA HANYA boleh ada di kueri DLA.
//
// Menyalinnya ke tab lain akan MENGHILANGKAN baris dari antrean PLA dan Pre DLA — dan
// baris yang hilang dari antrean tidak menghasilkan keluhan sampai seseorang menyadari
// pekerjaannya tidak pernah muncul.
func TestBranchAndBusinessGroupFiltersBelongToTheDLAListOnly(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_dla")), "ASNET",
		"kueri DLA kehilangan pengecualian cabang ASNET")
	require.Contains(t, strings.ToUpper(query("list_dla")), "10008",
		"kueri DLA kehilangan pengecualian kelompok bisnis 10008")

	for _, name := range []string{"list_pla", "list_pre_dla"} {
		text := strings.ToUpper(query(name))
		require.NotContains(t, text, "ASNET",
			"kueri %q tidak boleh mengecualikan cabang ASNET", name)
		require.NotContains(t, text, "10008",
			"kueri %q tidak boleh mengecualikan kelompok bisnis 10008", name)
	}
}

// Tab Pre DLA menyaring `NOAKSEP IS NULL`, BUKAN `ISKIRIM`.
//
// Ini penyaring yang paling mudah "diseragamkan" menjadi salah. Yang mengeluarkan sebuah
// Pre-DLA dari antrean adalah terbitnya Nomor Akseptasi, bukan terkirimnya surat — dan
// menyamakannya dengan dua tab lain mengubah isi antrean tanpa satu pun galat.
func TestPreDLAListFiltersOnAcceptanceNumberNotOnSentFlag(t *testing.T) {
	text := strings.ToUpper(query("list_pre_dla"))

	require.Contains(t, text, "A.NOAKSEP IS NULL",
		"kueri Pre DLA kehilangan penyaring Nomor Akseptasi")
	require.NotContains(t, text, "A.ISKIRIM",
		"kueri Pre DLA tidak boleh menyaring keanggotaan dengan ISKIRIM")
}

// Kedua tab lain menyaring keanggotaan dengan `ISKIRIM`, dan menerima `'0'` DI SAMPING
// `NULL`.
//
// Menerima `NULL` saja akan menghilangkan sebagian antrean: kedua nilai itu benar-benar
// ada di data — lihat klaim PNC-1008 pada data contoh penyimpanan memori.
func TestPLAAndDLAListsAcceptBothUnsentMarkers(t *testing.T) {
	for _, name := range []string{"list_pla", "list_dla"} {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "ISKIRIM IS NULL OR",
			"kueri %q tidak menerima ISKIRIM NULL sebagai belum terkirim", name)
		require.Contains(t, text, "= '0'",
			"kueri %q tidak menerima ISKIRIM '0' sebagai belum terkirim", name)
	}
}

// Hanya tab PLA yang menuntut kode reasuradur sudah terisi.
func TestOnlyThePLAListRequiresAReinsurerCode(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_pla")), "REINSCODE IS NOT NULL",
		"kueri PLA kehilangan syarat kode reasuradur")

	for _, name := range []string{"list_dla", "list_pre_dla"} {
		require.NotContains(t, strings.ToUpper(query(name)), "REINSCODE",
			"kueri %q tidak boleh menuntut kode reasuradur", name)
	}
}

// Penyaring rentang tanggal dibandingkan terhadap KOLOM APA ADANYA, bukan terhadap
// `TRUNC(kolom)`.
//
// `TRUNC` pada kolom membuat index atas kolom itu tidak terpakai. Pada tabel dokumen yang
// tumbuh terus, itu pemindaian penuh setiap kali panel pencarian dipakai — dan kueri lama
// melakukannya persis begitu.
func TestDateRangeDoesNotWrapTheColumnInTrunc(t *testing.T) {
	for _, name := range listQueries {
		require.NotContains(t, strings.ToUpper(query(name)), "TRUNC(",
			"kueri %q membungkus kolom tanggal dengan TRUNC", name)
	}
}

// Batas atas rentang bersifat EKSKLUSIF.
//
// Ia dipasangkan dengan penggeseran satu hari di domain (lihat Query.To). Bila kuerinya
// memakai `<=` sementara domainnya tetap menggeser, rentangnya diam-diam melebar satu
// hari — selisih yang tidak menghasilkan galat dan hanya terlihat pada baris di tepi.
func TestUpperBoundOfTheDateRangeIsExclusive(t *testing.T) {
	for _, name := range listQueries {
		text := query(name)
		require.NotContains(t, text, "<= :6",
			"kueri %q memakai batas atas inklusif", name)
		require.Contains(t, text, "< :6",
			"kueri %q kehilangan batas atas eksklusif", name)
	}
}

// Kata kunci pencarian DIIKAT, tidak pernah dirangkai.
//
// Sistem lama menyisipkannya ke dalam teks SQL lewat `{ASIS:TempPLA.ASMSurveyor}` — apa
// yang diketik pengguna, langsung ke dalam kueri.
func TestSearchKeywordIsBoundAndEscaped(t *testing.T) {
	for _, name := range listQueries {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "LIKE :2 ESCAPE",
			"kueri %q tidak melepaskan wildcard kata kunci", name)
		require.NotContains(t, text, "{ASIS",
			"kueri %q masih memuat pola perangkaian Pega", name)
	}
}

// Penanda kehadiran penyaring TIDAK boleh bertipe tanggal.
//
// Bind yang HANYA muncul di dalam `IS NULL` tidak punya konteks tipe di dalam kueri,
// sehingga tipenya ditentukan driver. Di sini penandanya selalu bind teks (`:1`, `:3`,
// `:5`) dan bind tanggal (`:4`, `:6`) selalu muncul di dalam perbandingan — uji ini
// menjaga pasangan itu tetap begitu.
func TestOnlyTextBindsAreTestedForNull(t *testing.T) {
	pattern := regexp.MustCompile(`:(\d+)\s+IS NULL`)

	for _, name := range listQueries {
		for _, match := range pattern.FindAllStringSubmatch(query(name), -1) {
			number, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			require.Contains(t, []int{1, 3, 5}, number,
				"kueri %q menguji IS NULL pada bind :%d, yang bukan penanda teks",
				name, number)
		}
	}
}

// Jumlah bind yang dipakai kueri daftar WAJIB delapan — sama dengan jumlah yang disusun
// listArgs. Selisih satu saja menghasilkan galat bind yang menyebut NOMOR, bukan menyebut
// kueri mana yang rusak.
func TestListQueriesUseExactlyEightBinds(t *testing.T) {
	for _, name := range listQueries {
		require.Equal(t, 8, highestBind(query(name)),
			"kueri %q memakai jumlah bind yang berbeda dari listArgs", name)
	}
}

// Kueri rincian memakai tepat satu bind: kunci klaimnya.
func TestDocumentQueriesUseExactlyOneBind(t *testing.T) {
	names := append([]string{"claim_exists", "print_pre_dla"}, documentQueries...)
	for _, name := range names {
		require.Equal(t, 1, highestBind(query(name)),
			"kueri %q memakai jumlah bind yang tidak diharapkan", name)
	}
}

// Kueri rincian WAJIB punya `ORDER BY`.
//
// `GetPLAList` dan `GetDLAList` tidak punya satu pun, sehingga urutan barisnya di Pega
// tidak ditentukan. Urutan yang tidak ditentukan bukan perilaku yang layak ditiru.
func TestDocumentQueriesAreOrdered(t *testing.T) {
	for _, name := range documentQueries {
		require.Contains(t, strings.ToUpper(query(name)), "ORDER BY",
			"kueri %q tidak menentukan urutan barisnya", name)
	}
}

// Panel "Print Pre DLA" WAJIB tetap menyaring lewat kedua tabel lampiran.
//
// Gabungannya mudah terbaca sebagai hiasan — panelnya toh tidak menggambar satu pun kolom
// lampiran kecuali kuncinya. Padahal gabungan itulah yang MENENTUKAN baris mana yang
// muncul: Pre-DLA tanpa lampiran berkategori `DLA` tidak masuk panel sama sekali.
//
// Menghapusnya membuat panel menampilkan LEBIH BANYAK baris daripada Pega, tanpa satu pun
// galat — dan baris tambahannya justru Pre-DLA yang dokumennya belum ada.
func TestPrintPanelKeepsTheAttachmentFilter(t *testing.T) {
	text := strings.ToUpper(query("print_pre_dla"))

	require.Contains(t, text, "PC_LINK_ATTACHMENT")
	require.Contains(t, text, "PC_DATA_WORKATTACH")
	require.Contains(t, text, "'DLA'",
		"penyaring kategori lampiran hilang")
	require.Contains(t, text, "SUBSTR(PXATTACHNAME, -15, 11)",
		"pencocokan nama berkas hilang")
}

// `PXATTACHNAME` sengaja TIDAK dikualifikasi nama tabelnya, persis seperti kueri Pega.
//
// Kolom itu ada di salah satu dari dua tabel lampiran, dan export tidak memuat DDL
// keduanya (`R-08`). Menebak tabelnya berisiko memilih yang salah — dan yang salah tidak
// menghasilkan galat, hanya panel kosong. Uji ini menjaga tebakan itu tidak masuk diam-diam
// saat seseorang merapikan kuerinya.
func TestAttachmentNameColumnStaysUnqualified(t *testing.T) {
	text := strings.ToUpper(query("print_pre_dla"))

	require.NotContains(t, text, "A.PXATTACHNAME")
	require.NotContains(t, text, "B.PXATTACHNAME")
}

// Panel ini HANYA MEMBACA.
//
// `GetPreDLAList` membawa pernyataan simpan yang menandai Pre-DLA sebagai terkirim, dan
// pernyataan itu TIDAK dibawa: `T_PREDLALIST` masih dimiliki Pega selama masa paralel
// (`P-1`). Uji ini menjaga jalur tulisnya tidak tersisip belakangan tanpa keputusan.
func TestPrintPanelNeverWrites(t *testing.T) {
	text := strings.ToUpper(query("print_pre_dla"))

	for _, verb := range []string{"UPDATE ", "INSERT ", "DELETE ", "MERGE "} {
		require.NotContains(t, text, verb,
			"kueri panel membawa jalur tulis %q", strings.TrimSpace(verb))
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

		// Batas kata di kiri, supaya `… AS` di dalam nama kolom tidak ikut tertangkap.
		if index > 0 && !isWordBoundary(text[index-1]) {
			continue
		}

		rest := text[index:]
		if match := pattern.FindStringSubmatch(rest); match != nil {
			found = append(found, strings.ToUpper(match[1]))
			index += len(match[0]) - 1
			continue
		}

		// `FROM` pada kedalaman nol menutup klausa SELECT terluar.
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
