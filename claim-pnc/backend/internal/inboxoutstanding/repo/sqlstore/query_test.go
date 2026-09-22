package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
)

// Uji di berkas ini berjalan TANPA basis data.
//
// Yang diperiksa adalah bentuk SQL dan pengikatan parameternya — dua hal yang bila salah
// menghasilkan kegagalan yang mahal: galat pengikatan di produksi, atau lebih buruk,
// penyaring batas data yang tidak berlaku tanpa satu pun galat.

func TestKeduaKueriTerbacaDanMemuatPenandaScope(t *testing.T) {
	for _, name := range []string{"outstanding_list", "outstanding_count"} {
		text := query(name)
		require.NotEmpty(t, text, "kueri %s kosong", name)
		require.Contains(t, text, "/*SCOPE*/", "kueri %s kehilangan penanda scope", name)
	}
}

func TestKomentarTidakIkutDikirimKeBasisData(t *testing.T) {
	// Komentar baris dibuang; komentar blok /*SCOPE*/ HARUS tetap ada karena ia penanda.
	text := query("outstanding_list")
	require.NotContains(t, text, "-- ", "komentar baris tidak boleh ikut terkirim")
	require.Contains(t, text, "/*SCOPE*/")
}

// Inilah uji yang menjaga janji terpenting berkas sqlstore: nilai TIDAK PERNAH dirangkai
// ke dalam teks SQL.
//
// Sistem lama merangkai potongan WHERE dari properti klipboard lewat {ASIS:TempView.pyNote}
// justru pada layar ini, dan potongan itulah yang menentukan batas data. Uji ini yang
// membuat pernyataan "celah itu tertutup" tetap benar.
func TestNilaiTidakPernahMasukKeTeksSQL(t *testing.T) {
	scope := inboxoutstanding.LineScope{GroupPanels: []string{"002", "005"}}
	statement, args := expandScope(query("outstanding_list"), scope, scopeFirstBindList)

	require.NotContains(t, statement, "002", "nilai lini bocor ke teks SQL")
	require.NotContains(t, statement, "005", "nilai lini bocor ke teks SQL")
	require.Equal(t, []any{"002", "005"}, args, "nilainya dikirim terpisah sebagai argumen")
}

func TestScopeTanpaBatasTidakMenambahKlausaApaPun(t *testing.T) {
	scope := inboxoutstanding.LineScope{Unrestricted: true}
	statement, args := expandScope(query("outstanding_list"), scope, scopeFirstBindList)

	require.Empty(t, args)
	require.NotContains(t, statement, "/*SCOPE*/", "penanda harus tergantikan")
	require.NotContains(t, statement, "POLIS_LINI IN", "tanpa batas berarti tanpa penyaring lini")
}

// Scope kosong harus gagal TERTUTUP.
//
// Ia hampir pasti cacat pemrograman, dan meloloskan semuanya akan mengubah cacat itu
// menjadi kebocoran data antar lini yang tidak menghasilkan galat apa pun.
func TestScopeKosongMenghasilkanKlausaYangTidakMeloloskanApaPun(t *testing.T) {
	statement, args := expandScope(query("outstanding_list"), inboxoutstanding.LineScope{}, scopeFirstBindList)

	require.Empty(t, args)
	require.Contains(t, statement, "AND 1 = 0")
}

func TestPenandaScopeDimulaiDariNomorYangBenar(t *testing.T) {
	scope := inboxoutstanding.LineScope{GroupPanels: []string{"003", "004", "006"}}

	listSQL, _ := expandScope(query("outstanding_list"), scope, scopeFirstBindList)
	require.Contains(t, listSQL, ":9, :10, :11")

	countSQL, _ := expandScope(query("outstanding_count"), scope, scopeFirstBindCount)
	require.Contains(t, countSQL, ":7, :8, :9")
}

// Uji yang menjaga konstanta scopeFirstBind* tetap sejalan dengan isi berkas .sql.
//
// Bila seseorang menambah penyaring baru ke SQL tanpa menggeser konstantanya, dua penanda
// akan bertabrakan — dan tabrakannya menghasilkan hasil yang SALAH, bukan galat: nilai
// lini akan terbaca sebagai nilai penyaring lain.
func TestKonstantaNomorScopeSejalanDenganIsiBerkasSQL(t *testing.T) {
	cases := []struct {
		name      string
		firstBind int
	}{
		{"outstanding_list", scopeFirstBindList},
		{"outstanding_count", scopeFirstBindCount},
	}

	for _, c := range cases {
		highest := highestBind(t, query(c.name))
		require.Equal(t, c.firstBind-1, highest,
			"kueri %s memakai :1..:%d, sehingga scope harus mulai dari :%d",
			c.name, highest, highest+1)
	}
}

// highestBind mencari nomor parameter tertinggi yang dipakai sebuah pernyataan.
func highestBind(t *testing.T, statement string) int {
	t.Helper()

	matches := regexp.MustCompile(`:(\d+)`).FindAllStringSubmatch(statement, -1)
	require.NotEmpty(t, matches, "pernyataan tidak memakai satu pun parameter")

	highest := 0
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		require.NoError(t, err)
		if n > highest {
			highest = n
		}
	}
	return highest
}

func TestPolaPencarianDiseragamkanMenjadiHurufBesar(t *testing.T) {
	require.Equal(t, "%PNCN.26%", searchPattern("pncn.26"))
	require.Equal(t, "", searchPattern("   "))
}

// Karakter khusus LIKE di-escape supaya pencarian tidak berubah menjadi pola liar.
//
// Tanpa ini, mengetik "%" akan mencocokkan SELURUH baris — dan pengguna tidak punya cara
// menduga mengapa.
func TestKarakterKhususLIKEDiEscape(t *testing.T) {
	require.Equal(t, `%100\%%`, searchPattern("100%"))
	require.Equal(t, `%A\_B%`, searchPattern("a_b"))
	require.Equal(t, `%C\\D%`, searchPattern(`c\d`))
}

// Escaping hanya bekerja bila SQL menyatakan karakter escape-nya. Keduanya harus cocok:
// Go meng-escape dengan backslash, SQL wajib menyebut ESCAPE '\'.
func TestSQLMenyatakanKarakterEscapeYangSamaDenganGo(t *testing.T) {
	for _, name := range []string{"outstanding_list", "outstanding_count"} {
		text := query(name)
		likeCount := strings.Count(text, "LIKE")
		escapeCount := strings.Count(text, `ESCAPE '\'`)
		require.Equal(t, likeCount, escapeCount,
			"kueri %s: setiap LIKE wajib menyebut ESCAPE '\\'", name)
	}
}

func TestPenyaringKosongDikirimSebagaiNULL(t *testing.T) {
	args := filterArgs(inboxoutstanding.Filter{}.Normalize())

	require.Len(t, args, 6)
	for i, a := range args {
		require.Nil(t, a, "argumen ke-%d harus NULL saat penyaringnya kosong", i+1)
	}
}

func TestPenyaringTahapDanCabangDiseragamkanMenjadiHurufBesar(t *testing.T) {
	args := filterArgs(inboxoutstanding.Filter{
		Stage:      "komite",
		BranchCode: "jkt",
	}.Normalize())

	require.Equal(t, "KOMITE", args[4])
	require.Equal(t, "JKT", args[5])
}

// Kedua kueri wajib punya syarat WHERE yang sama persis selain paginasi.
//
// Bila keduanya menyimpang, pengguna melihat jumlah yang berbeda dari yang dapat
// ditelusurinya — dan tidak ada galat yang muncul.
func TestSyaratOutstandingSamaPadaKeduaKueri(t *testing.T) {
	list := query("outstanding_list")
	count := query("outstanding_count")

	// Kelima penyaring pertama disalin dari `BrowseInboxOutstanding1-SQL.xml:121-128`.
	for _, condition := range []string{
		"k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'",
		"k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')",
		"k.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')",
		"k.PXTASKLABEL NOT IN ('FixCorrespondence')",
		"k.BRANCHNAME <> 'ASNET'",
		"UPPER(TRIM(k.PXTASKLABEL)) = :5",
		"UPPER(TRIM(k.BRANCHNAME)) = :6",
	} {
		require.Contains(t, list, condition, "outstanding_list")
		require.Contains(t, count, condition, "outstanding_count")
	}
}

// Kedua kueri membaca tabel yang benar.
//
// Modul ini sempat dibangun di atas `CPNC_KLAIM` — tabel rancangan yang belum pernah
// dibuat dan modul penulisnya belum dipasang, sehingga layar akan selalu kosong TANPA
// GALAT. Uji ini menjaga kekeliruan itu tidak kembali diam-diam.
func TestKeduaKueriMembacaTabelYangBenar(t *testing.T) {
	for _, name := range []string{"outstanding_list", "outstanding_count"} {
		text := query(name)
		require.Contains(t, text, "POOLDATA.T_CLAIMLIST_ADMIN", "kueri %s", name)
		require.NotContains(t, text, "CPNC_KLAIM", "kueri %s", name)
		require.NotContains(t, text, "CPNC_TUGAS", "kueri %s", name)
	}
}

// Tabel datar berarti TANPA JOIN — dan tanpa join, persoalan INNER versus LEFT gugur.
//
// Di sistem lama kueri ini menempuh empat tabel, dan inner join-nya membuat klaim tanpa
// assignment hilang serta klaim dengan dua assignment muncul dua kali.
func TestKueriTidakMemakaiJoinSamaSekali(t *testing.T) {
	for _, name := range []string{"outstanding_list", "outstanding_count"} {
		text := strings.ToUpper(query(name))
		require.NotContains(t, text, " JOIN ", "kueri %s", name)
		require.Equal(t, 1, strings.Count(text, "FROM "), "kueri %s membaca satu tabel", name)
	}
}

// SELECT * dilarang — `08-TECHNICAL-STRATEGY.md` §4.3.
func TestTidakAdaSelectBintang(t *testing.T) {
	for _, name := range []string{"outstanding_list", "outstanding_count", "line_business_for"} {
		require.NotContains(t, query(name), "SELECT *", "kueri %s", name)
	}
}

// Pola Oracle-khas yang dilarang demi portabilitas ke PostgreSQL (`D-20`).
func TestTidakAdaPolaSQLYangDilarang(t *testing.T) {
	forbidden := []string{"NVL(", "ROWNUM", "SYSDATE", "DECODE(", "FROM DUAL", "(+)", "LISTAGG"}

	for _, name := range []string{"outstanding_list", "outstanding_count", "line_business_for"} {
		text := strings.ToUpper(query(name))
		for _, pattern := range forbidden {
			require.NotContains(t, text, pattern, "kueri %s memakai pola terlarang %s", name, pattern)
		}
	}
}
