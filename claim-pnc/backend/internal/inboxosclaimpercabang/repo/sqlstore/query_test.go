package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/money"
)

// listQueries adalah kedua kueri yang mengembalikan baris berhalaman.
var listQueries = []string{"list", "list_export"}

// aliasesOf mengambil alias hasil sebuah kueri, berurut sebagaimana tertulis.
//
// # Kenapa ia menghitung kedalaman kurung, bukan sekadar mencocokkan pola
//
// Karena kedua kueri di modul ini punya delapan subkueri yang juga memberi alias —
// `MAX(sts_progress1) AS sts_progress1`, `SUM(estimationvalue) AS reserves`, dan seterusnya.
// Pencocokan pola polos akan ikut menangkapnya, sehingga uji kesesuaian alias membandingkan
// daftar yang memuat kolom yang tidak pernah dikembalikan ke pemanggil.
//
// Yang dihitung hanyalah alias pada kedalaman NOL, yakni daftar SELECT terluar — persis yang
// dilihat pemindai.
func aliasesOf(name string) []string {
	alias := regexp.MustCompile(`(?i)\bAS\s+([A-Z_0-9]+)\s*,?\s*$`)

	found := []string{}
	for _, line := range strings.Split(maskNested(query(name)), "\n") {
		if match := alias.FindStringSubmatch(strings.TrimRight(line, " \t")); match != nil {
			found = append(found, strings.ToUpper(match[1]))
		}
	}

	return found
}

// maskNested mengganti setiap karakter di dalam kurung dengan spasi.
//
// Baris barunya DIPERTAHANKAN, sehingga susunan barisnya tidak berubah dan pencocokan
// per-baris tetap menunjuk baris yang sama dengan berkas aslinya.
//
// Yang tersisa sesudahnya adalah daftar SELECT terluar apa adanya — termasuk alias pada baris
// yang diawali penutup subkueri, seperti `FETCH FIRST 1 ROW ONLY) AS POLICY_BUSINESS_NAME`.
func maskNested(text string) string {
	var masked strings.Builder
	masked.Grow(len(text))

	depth := 0
	for _, char := range text {
		switch {
		case char == '\n':
			masked.WriteRune(char)
		case char == '(':
			depth++
			masked.WriteRune(' ')
		case char == ')':
			depth--
			masked.WriteRune(' ')
		case depth > 0:
			masked.WriteRune(' ')
		default:
			masked.WriteRune(char)
		}
	}

	return masked.String()
}

func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	// Daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3.
	//
	// Empat di antaranya dipakai kueri lama dan sengaja diganti — `NVL`, `TRUNC`, `LISTAGG`,
	// dan `SELECT *` di dalam subkueri treaty. Penggantinya beserta alasannya ada di kepala
	// berkas .sql.
	forbidden := []string{
		"NVL(", "SYSDATE", "DECODE(", "ROWNUM", "INSTR(", "LISTAGG(", "STRING_AGG(",
		"FROM DUAL", "ADD_MONTHS(", "MONTHS_BETWEEN(", "TO_CHAR(", "TO_NUMBER(",
		"TRUNC(", "SELECT *", "PIVOT", " KEEP (",
	}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContainsf(t, upper, pattern,
				"kueri %s memakai %s yang tidak portabel", name, pattern)
		}
	}
}

func TestQueriesNeverInterpolateValues(t *testing.T) {
	// Ketiga kueri sumbernya menyisipkan nilai LANGSUNG ke teks SQL —
	// `{OperatorID.pyTelephone}` pada kedua kueri daftar dan `{TempCari.CARI1}` pada
	// penanda progres. Yang pertama adalah batas datanya sendiri: nilai yang disisipkan ke
	// sana menentukan cabang siapa yang terlihat.
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %s memuat pola {ASIS:…} warisan", name)
		require.NotContainsf(t, text, "{Operator", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "{Temp", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "%s", "kueri %s tampak dirangkai lewat fmt", name)
		require.NotContainsf(t, text, "||", "kueri %s merangkai teks di dalam SQL", name)
	}
}

func TestQueriesNeverWrite(t *testing.T) {
	// SELURUH tabel yang dibaca modul ini milik sistem lama. Menulis satu saja melanggar
	// `P-1`, dan akibatnya bukan galat melainkan dua sistem yang saling menimpa.
	writing := []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, verb := range writing {
			require.NotContainsf(t, upper, verb,
				"kueri %s tampak menulis (%s); modul ini hanya membaca",
				name, strings.TrimSpace(verb))
		}
	}
}

func TestListQueriesPaginateOrderAndCount(t *testing.T) {
	// Kueri yang lupa memaginasi akan menarik seluruh klaim cabang ke memori aplikasi, dan
	// kueri yang lupa mengurutkan membuat satu baris muncul di dua halaman sekaligus.
	for _, name := range listQueries {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, upper, "OFFSET ", "kueri %s tidak memaginasi", name)
		require.Containsf(t, upper, "FETCH NEXT", "kueri %s tidak membatasi jumlah baris", name)
		require.Containsf(t, upper, "ORDER BY", "kueri %s tidak menetapkan urutan", name)
		require.Containsf(t, upper, "COUNT(*) OVER ()",
			"kueri %s tidak membawa jumlah seluruh baris", name)
	}
}

func TestEveryQueryThatReadsClaimsFiltersByBranch(t *testing.T) {
	// Batas data layar ini SELURUHNYA cabang. Kueri yang lupa menyaringnya tidak
	// menghasilkan satu pun galat — ia menampilkan klaim seluruh cabang, terisi dan tampak
	// wajar.
	for _, name := range []string{"list", "list_export", "dominant_factors"} {
		require.Containsf(t, query(name), "c.branchcode = :1",
			"kueri %s tidak menyaring menurut kode cabang", name)
	}
}

func TestListQueriesKeepTheOutstandingFilter(t *testing.T) {
	// "Outstanding" di layar ini adalah SATU penyaring, bukan status akseptasi. Kehilangannya
	// membuat klaim yang sudah tutup ikut tampil — dan kolomnya terlihat sama wajarnya.
	for _, name := range append(append([]string{}, listQueries...), "dominant_factors") {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, upper, "'RESOLVED-REJECTED'",
			"kueri %s kehilangan penyaring outstanding", name)
		require.Containsf(t, upper, "'RESOLVED-COMPLETED'",
			"kueri %s kehilangan penyaring outstanding", name)
	}
}

func TestListAliasesMatchTheScanner(t *testing.T) {
	require.Equal(t, listColumns, aliasesOf("list"),
		"alias kueri list berbeda dari listColumns; pemindai akan salah kolom")
}

func TestExportAliasesExtendTheListAliases(t *testing.T) {
	// Ke-19 kolom pertama berkas WAJIB sama dengan ke-19 kolom pertama grid. Itulah yang
	// membuat berkas dan layar dapat dicocokkan baris per baris.
	//
	// TOTAL_ROWS tidak ikut: pada `list` ia kolom ke-20, pada `list_export` ia pindah ke
	// ujung supaya kolom tambahan dapat disisipkan tanpa menggeser awalannya.
	base := listColumns[:len(listColumns)-1]

	want := append(append([]string{}, base...), exportOnlyColumns...)
	want = append(want, "TOTAL_ROWS")

	require.Equal(t, want, aliasesOf("list_export"),
		"susunan alias list_export bukan awalan list ditambah kolom ekspor")
}

func TestExportCarriesExactlyTwentyFourTreatyColumns(t *testing.T) {
	// Judul, nilai, dan alias SQL-nya hidup di tiga tempat. Uji ini yang menjaga ketiganya
	// tidak pernah berbeda panjang — satu pergeseran kolom pada berkas berisi angka uang
	// tidak menghasilkan satu pun galat.
	require.Len(t, inboxosclaimpercabang.ExportTreatyColumns, 24,
		"jumlah judul kolom treaty berubah")

	require.Len(t, inboxosclaimpercabang.TreatyShares{}.TreatyValues(), 24,
		"jumlah nilai treaty berbeda dari jumlah judulnya")

	shares := 0
	for _, alias := range exportOnlyColumns {
		if strings.HasPrefix(alias, "SHARE_") {
			shares++
		}
	}
	require.Equal(t, 24, shares, "jumlah alias SHARE_* pada kueri ekspor bukan 24")
}

func TestTreatyValuesFollowTheAliasOrder(t *testing.T) {
	// Setiap nilai diberi angka yang berbeda lalu dibaca kembali berurut. Uji ini menangkap
	// dua field yang tertukar — kesalahan yang tidak dapat dilihat dari jumlahnya saja.
	sen := money.FromMinorUnits
	shares := inboxosclaimpercabang.TreatyShares{
		OR: sen(1), FacOut: sen(2), FacOB: sen(3), QS: sen(4), FSPL: sen(5),
		SSPL: sen(6), ER1: sen(7), ER2: sen(8), BPPDAN: sen(9), PSRQS: sen(10),
		PSRSPL: sen(11), ORS: sen(12), XL: sen(13), PSROR: sen(14), QSOR: sen(15),
		PSS: sen(16), PRGBI: sen(17), PFRA: sen(18), FSPLNSRI: sen(19),
		PSPLNSRI: sen(20), FSPLNSOR: sen(21), PSPLNSOR: sen(22), FacOBSRB: sen(23),
		FacOBIndt: sen(24),
	}

	for index, value := range shares.TreatyValues() {
		require.Equalf(t, money.FromMinorUnits(int64(index+1)), value,
			"nilai treaty ke-%d (%s) tertukar", index+1,
			inboxosclaimpercabang.ExportTreatyColumns[index])
	}
}

func TestOnlyExportTouchesTheDatabaseLink(t *testing.T) {
	// DB Link `@asmd` adalah ketergantungan ke basis data lain, dan `D-25` menetapkan ia
	// kelak diganti API. Selama itu belum tiba, ia hanya boleh disentuh EKSPOR — layar tidak
	// boleh berhenti bekerja karena link yang sedang padam.
	allowed := map[string]bool{"list_export": true, "check_export_tables": true}

	for name, text := range queries {
		if allowed[name] {
			continue
		}
		require.NotContainsf(t, strings.ToLower(text), "@asmd",
			"kueri %s menyentuh DB Link; hanya ekspor yang boleh", name)
	}
}

func TestStalledMarkerAsksForThreeIdenticalProgressEntries(t *testing.T) {
	// Penanda inilah yang membuat sebuah baris digambar merah. Aturannya — tiga catatan
	// progres terakhir bernilai sama, dan tak satu pun NULL — hidup hanya di dalam SQL,
	// sehingga ia dijaga di sini.
	//
	// `COUNT(status_progress1) = 3` bukan `COUNT(*) = 3`: yang pertama mengabaikan NULL, dan
	// itulah yang meniru perilaku lama. Pada PIVOT, `r1 = r2` bernilai UNKNOWN begitu salah
	// satunya NULL, sehingga klaim itu tidak pernah ditandai.
	for _, name := range listQueries {
		text := query(name)
		require.Containsf(t, text, "rn <= 3",
			"kueri %s tidak membatasi pada tiga catatan progres terakhir", name)
		require.Containsf(t, text, "COUNT(status_progress1) = 3",
			"kueri %s memakai COUNT(*) alih-alih COUNT(kolom); baris NULL akan ikut", name)
		require.Containsf(t, text, "COUNT(DISTINCT status_progress1) = 1",
			"kueri %s tidak memeriksa ketiganya bernilai sama", name)
	}
}

func TestBothPolicySubqueriesUseTheSameOrder(t *testing.T) {
	// Nama bisnis polis dan nama tertanggung diambil dari perpanjangan polis yang SAMA.
	// Mengubah urutan salah satunya akan memasangkan keduanya dari perpanjangan berbeda —
	// tanpa satu pun gejala.
	order := "ORDER BY CAST(TRIM(t.prodke) AS NUMERIC) DESC"
	require.Equal(t, 2, strings.Count(query("list_export"), order),
		"kedua subkueri T_GENERAL tidak memakai urutan yang identik")
}
