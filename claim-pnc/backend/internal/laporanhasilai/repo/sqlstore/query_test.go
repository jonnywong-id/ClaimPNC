package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// usedQueries adalah seluruh nama kueri yang dipanggil kode modul ini.
var usedQueries = []string{
	"report_list",
	"report_count",
	"report_summary",
	"report_check_table",
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

// TestQueriesTargetBothTables membuktikan seluruhnya menembak kedua tabel yang benar.
//
// Satu kueri yang lupa menggabungkan `T_CLAIM_KOMITE_LIST` akan menampilkan penilaian AI
// tanpa keputusan komitenya — data yang masuk akal dari tempat yang salah, dan itu kelas
// cacat yang tidak terlihat sebagai galat.
func TestQueriesTargetBothTables(t *testing.T) {
	for _, name := range usedQueries {
		text := strings.ToUpper(getQuery(name))
		require.Containsf(t, text, "POOLDATA.T_CLAIM_DATA_RESULTS_AI",
			"kueri %q tidak menembak tabel penilaian AI", name)
		require.Containsf(t, text, "POOLDATA.T_CLAIM_KOMITE_LIST",
			"kueri %q tidak menembak tabel keputusan komite", name)
	}
}

// TestModuleNeverWrites membuktikan modul ini benar-benar baca-saja.
//
// Kedua tombol layar lama memanggil activity yang sama, dan activity itu tidak memuat satu
// pun langkah tulis. Uji ini yang menjaga jalur tulis tidak masuk diam-diam lewat kueri
// baru — yang akan melanggar `P-1`, karena kedua tabel ini masih ditulis Pega.
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
// `TO_CHAR` dan `SYSDATE` disebut khusus karena kueri Pega aslinya MEMAKAI keduanya —
// sekali untuk memformat `TANGGALKOMITE`, dan sekali lagi untuk kolom mati
// `TO_CHAR(SYSDATE-7,'dd/mm/yyyy')`. Menyalinnya apa adanya akan membuat modul ini gagal di
// PostgreSQL, dan kegagalannya baru terlihat saat cutover.
//
// `TRUNC(` juga dilarang: kueri lama memakainya pada kolom tanggal, yang mematikan index
// sekaligus tidak portabel.
func TestNoForbiddenSQLPattern(t *testing.T) {
	forbidden := []string{"ROWNUM", "NVL(", "SYSDATE", "DECODE(", "TO_CHAR(", "TRUNC(", "FROM DUAL"}
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

// TestNoClipboardStringConcatenation membuktikan pola `{Asis:…}` tidak ikut terbawa.
//
// Kueri lama menempelkan SELURUH syarat tanggalnya sebagai teks yang disusun activity dari
// dua isian yang diketik pengguna. Itu celah SQL injection, dan larangan perangkaian tidak
// dikecualikan oleh keputusan mana pun.
func TestNoClipboardStringConcatenation(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, text, "{Asis:",
			"kueri %q masih memakai perangkaian klipboard", name)
		require.NotContainsf(t, text, "{ASIS:",
			"kueri %q masih memakai perangkaian klipboard", name)
		require.NotContainsf(t, text, "TempDatalaporanAI",
			"kueri %q masih menyebut halaman klipboard Pega", name)
	}
}

// TestPaginationUsesBoundParameters membuktikan jendela halaman memakai parameter terikat.
func TestPaginationUsesBoundParameters(t *testing.T) {
	text := strings.ToUpper(getQuery("report_list"))
	require.Contains(t, text, "OFFSET :3 ROWS", "report_list tidak memaginasi")
	require.Contains(t, text, "FETCH NEXT :4 ROWS ONLY", "report_list tidak membatasi halaman")
}

// TestFilteringQueriesShareTheSameFilter membuktikan ketiga kueri menyaring hal yang sama.
//
// Ini yang paling menentukan di modul ini. Ada TIGA kueri yang harus sepakat:
//
//	report_list      isi grid
//	report_count     penggerak paginator
//	report_summary   angka di grid ringkasan
//
// Bila salah satunya menyaring berbeda, layar akan menampilkan ringkasan yang tidak dapat
// dicocokkan dengan barisnya, atau paginator yang melaporkan halaman kosong — dan keduanya
// baru terlihat setelah datanya banyak.
func TestFilteringQueriesShareTheSameFilter(t *testing.T) {
	want := whereClauseOf(getQuery("report_list"))
	require.NotEmpty(t, want, "report_list tidak punya klausa WHERE")

	for _, name := range []string{"report_count", "report_summary"} {
		require.Equalf(t, want, whereClauseOf(getQuery(name)),
			"penyaring kueri %q berbeda dari report_list", name)
	}
}

// whereClauseOf mengambil klausa WHERE sebuah kueri, tanpa ORDER BY dan paginasinya.
func whereClauseOf(text string) string {
	upper := strings.ToUpper(text)
	start := strings.Index(upper, "\n WHERE")
	if start < 0 {
		return ""
	}
	clause := upper[start:]
	if stop := strings.Index(clause, "ORDER BY"); stop >= 0 {
		clause = clause[:stop]
	}
	return strings.Join(strings.Fields(clause), " ")
}

// TestDateRangeIsHalfOpen membuktikan batas atas memakai `<`, bukan `<=`.
//
// Kueri lama menulis `trunc(TANGGALKOMITE) <= to_date(akhir)`. Penggantinya rentang
// setengah terbuka, yang memilih baris yang sama persis tetapi tetap dapat memakai index.
// Bila seseorang kelak mengembalikannya menjadi `<=`, hasilnya akan MELEBIHI satu hari —
// karena pemanggil sudah menambahkan satu hari lewat `Filter.ToExclusive`.
func TestDateRangeIsHalfOpen(t *testing.T) {
	for _, name := range []string{"report_list", "report_count", "report_summary"} {
		text := strings.ToUpper(getQuery(name))
		require.Containsf(t, text, "B.TANGGALKOMITE >= :1",
			"kueri %q tidak memakai batas bawah terikat", name)
		require.Containsf(t, text, "B.TANGGALKOMITE < :2",
			"kueri %q tidak memakai batas atas EKSKLUSIF", name)
		require.NotContainsf(t, text, "TANGGALKOMITE <= :",
			"kueri %q memakai batas atas inklusif — hasilnya akan lebih satu hari", name)
	}
}

// TestResolvedCaseGateIsPreserved membuktikan syarat `Resolved-Completed` tidak hilang.
//
// Ia satu-satunya penyaring yang membedakan "komite sudah selesai memutuskan" dari "komite
// masih berjalan". Menghilangkannya akan memasukkan kasus yang belum tuntas ke dalam
// laporan, dan ringkasannya akan mencacah keputusan yang belum ada.
func TestResolvedCaseGateIsPreserved(t *testing.T) {
	for _, name := range []string{"report_list", "report_count", "report_summary"} {
		text := getQuery(name)
		require.Containsf(t, text, "'Resolved-Completed'",
			"kueri %q kehilangan penyaring kasus selesai", name)
		require.Containsf(t, strings.ToUpper(text), "EXISTS",
			"kueri %q tidak memakai EXISTS", name)
		require.Containsf(t, strings.ToUpper(text), "B.TANGGALKOMITE IS NOT NULL",
			"kueri %q kehilangan syarat TANGGALKOMITE tidak NULL", name)
	}
}

// TestReaderQueriesShareColumnOrder membuktikan kedua pembaca menyebut kolom yang sama pada
// urutan yang sama — syarat agar satu scanRow cukup untuk keduanya.
func TestReaderQueriesShareColumnOrder(t *testing.T) {
	want := []string{
		"B.KOMITE_ID",
		"B.KOMITEKE",
		"B.NO_KLAIM",
		"B.STATUSAPPROVE",
		"B.TANGGALKOMITE",
		"A.OBJECTID",
		"A.COVERAGEID",
		"A.RESULTAI",
		"A.TGLAI",
	}
	for _, name := range []string{"report_list", "report_check_table"} {
		require.Equalf(t, want, selectedColumns(getQuery(name)),
			"urutan kolom kueri %q berbeda dari yang dibaca scanRow", name)
	}
}

// TestFiveColumnsStayUnselected mengunci keputusan Work Owner 2026-09-26.
//
// Kelima kolom ini SENGAJA tidak dipilih, sehingga kelima kolom layar yang membacanya
// tergambar kosong — persis seperti Pega. Uji ini bukan untuk melarang selamanya; ia untuk
// memastikan perubahannya DISENGAJA. Siapa pun yang menambahkan salah satunya akan melihat
// uji ini gagal beserta alasannya, lalu menghapusnya secara sadar bersama keputusan barunya.
func TestFiveColumnsStayUnselected(t *testing.T) {
	unselected := []string{
		"OBJECTNAME",
		"NOTETERIMA",
		"NOTETOLAK",
		"COVERAGE_AI_FINAL",
		"KATEGORI_KRONOLOGI",
	}
	for _, name := range []string{"report_list", "report_check_table"} {
		columns := strings.Join(selectedColumns(getQuery(name)), ",")
		for _, column := range unselected {
			require.NotContainsf(t, columns, column,
				"kueri %q memilih %s — keputusan Work Owner 2026-09-26 berubah? "+
					"perbarui doc paket laporanhasilai dan uji ini", name, column)
		}
	}
}

// TestSummaryCountsBothSides membuktikan ringkasan mencacah AI DAN komite.
//
// Keduanya harus datang dari SATU kueri atas himpunan baris yang sama; itulah yang membuat
// kedua barisnya dapat dibandingkan satu sama lain — dan membandingkannya adalah seluruh
// alasan laporan ini ada.
func TestSummaryCountsBothSides(t *testing.T) {
	text := strings.ToUpper(getQuery("report_summary"))
	require.Contains(t, text, "A.RESULTAI = 'DITERIMA'")
	require.Contains(t, text, "A.RESULTAI = 'DITOLAK'")
	require.Contains(t, text, "B.STATUSAPPROVE = '1'")
	require.Contains(t, text, "B.STATUSAPPROVE = '2'")
	require.Contains(t, text, "COUNT(*)")
}

// TestSummaryComparesStatusAsText membuktikan STATUSAPPROVE dibandingkan sebagai teks.
//
// Tipe kolomnya belum diketahui (`R-08`). Membandingkannya dengan ANGKA berbahaya: pada
// kolom teks, Oracle akan mengonversi KOLOMNYA dan gagal pada baris yang tidak numerik —
// dan kegagalan itu baru muncul di produksi, pada baris yang kebetulan kotor.
func TestSummaryComparesStatusAsText(t *testing.T) {
	text := strings.ToUpper(getQuery("report_summary"))
	require.NotContains(t, text, "B.STATUSAPPROVE = 1",
		"STATUSAPPROVE dibandingkan sebagai angka")
	require.NotContains(t, text, "B.STATUSAPPROVE = 2",
		"STATUSAPPROVE dibandingkan sebagai angka")
}

// selectedColumns mengambil daftar kolom pada klausa SELECT sebuah kueri.
func selectedColumns(text string) []string {
	upper := strings.ToUpper(text)
	start := strings.Index(upper, "SELECT")
	stop := strings.Index(upper, "\n  FROM")
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

// TestClaimNumberBlankingFollowsCommitteeStep mengunci aturan pengosongan nomor klaim.
//
// Padanan `CASE WHEN B.KOMITEKE = '1' THEN B.NO_KLAIM ELSE '' END`, yang Work Owner
// putuskan untuk ditiru pada 2026-09-26.
func TestClaimNumberBlankingFollowsCommitteeStep(t *testing.T) {
	require.Equal(t, "PNCN.26.0001", claimNumberOf("1", "PNCN.26.0001"))
	require.Equal(t, "PNCN.26.0001", claimNumberOf(" 1 ", " PNCN.26.0001 "),
		"spasi pada kolom CHAR tidak boleh membatalkan aturannya")
	require.Equal(t, "", claimNumberOf("2", "PNCN.26.0001"))
	require.Equal(t, "", claimNumberOf("", "PNCN.26.0001"))
}

// TestRowKeySeparatesParts membuktikan kunci baris tidak dapat bertabrakan.
//
// Tanpa pemisah, ("1","23") dan ("12","3") menghasilkan kunci yang sama — dan dua baris
// berkunci sama membuat tabel di layar menggambar salah satunya dua kali.
func TestRowKeySeparatesParts(t *testing.T) {
	require.NotEqual(t,
		rowKey("KMT-1", "1", "23", "C"),
		rowKey("KMT-1", "12", "3", "C"),
	)
	require.Equal(t, "KMT-1|1|OBJ|CVG", rowKey("KMT-1", "1", " OBJ ", " CVG "))
}
