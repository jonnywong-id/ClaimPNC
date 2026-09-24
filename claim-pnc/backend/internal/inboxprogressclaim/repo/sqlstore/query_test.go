package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxprogressclaim"
)

// claimViews adalah kedua region yang berbentuk baris klaim.
var claimViews = []string{
	inboxprogressclaim.ViewOutstanding,
	inboxprogressclaim.ViewNextFollowUp,
}

// sampleClaimQuery adalah permintaan lengkap, dipakai memanggil penyusun argumen.
func sampleClaimQuery(view string) inboxprogressclaim.ClaimQuery {
	return inboxprogressclaim.ClaimQuery{
		View:    view,
		Keyword: "PNCN.26",
		Today:   time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC),
	}
}

func TestEveryClaimViewHasQueryPair(t *testing.T) {
	for _, view := range claimViews {
		selected, known := plans[view]
		require.Truef(t, known, "bagian %s tidak ada di plans", view)
		require.NotEmptyf(t, selected.list, "bagian %s tidak punya kueri daftar", view)
		require.NotEmptyf(t, selected.count, "bagian %s tidak punya kueri pencacah", view)
	}
}

func TestNonClaimViewsHaveNoClaimQuery(t *testing.T) {
	// Rekap per PIC dan region Evaluasi tidak dilayani `plans`. Kueri yang menganggur
	// untuk region yang tidak memakainya adalah kode mati yang kelak dikira siap dipakai.
	for _, view := range []string{
		inboxprogressclaim.ViewPerPIC,
		inboxprogressclaim.ViewEvaluation,
	} {
		_, exists := plans[view]
		require.Falsef(t, exists, "bagian %s bukan region klaim, kuerinya tidak boleh ada", view)
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

// aliasesOf mengambil alias di UJUNG baris kolom.
//
// Syarat "di ujung baris" itu perlu: tanpanya, ekspresi seperti `CAST(x AS DATE)` ikut
// terbaca sebagai alias bernama "DATE", dan kueri yang sebetulnya benar akan dilaporkan
// salah.
func aliasesOf(text string) []string {
	alias := regexp.MustCompile(`(?im)\bAS\s+([A-Z_0-9]+)\s*,?\s*$`)

	found := []string{}
	for _, match := range alias.FindAllStringSubmatch(text, -1) {
		found = append(found, strings.ToUpper(match[1]))
	}
	return found
}

func TestBothClaimQueriesReturnTheSameAliases(t *testing.T) {
	// Satu pemindai melayani kedua kueri daftar. Begitu salah satunya memakai alias yang
	// berbeda atau urutannya bergeser, pemindai itu memasukkan nilai ke isian yang salah —
	// dan akibatnya BUKAN galat, melainkan kolom yang tertukar di layar.
	for _, view := range claimViews {
		text := query(plans[view].list)
		require.Equalf(t, claimColumns, aliasesOf(text),
			"alias kueri daftar bagian %s berbeda dari claimColumns", view)
	}
}

func TestPositionAndPICQueriesMatchTheirScanners(t *testing.T) {
	require.Equal(t, positionColumns, aliasesOf(query("positions")))
	require.Equal(t, picColumns, aliasesOf(query("pic_summary")))
}

func TestBindCountMatchesSuppliedArguments(t *testing.T) {
	// Argumen yang kurang menghasilkan ORA-01008 saat permintaan pertama datang; argumen
	// yang berlebih menghasilkan ORA-01036. Keduanya hanya terlihat di produksi bila tidak
	// diuji di sini, karena kuerinya tidak pernah dijalankan saat kompilasi.
	for _, view := range claimViews {
		selected := plans[view]
		filters := selected.filterArgs(sampleClaimQuery(view))

		// Pencacah memakai penyaringnya saja.
		require.Lenf(t, filters, highestBind(query(selected.count)),
			"kueri pencacah bagian %s tidak sepadan dengan argumennya", view)

		// Daftar memakai penyaring yang sama DITAMBAH dua bind paginasi.
		require.Equalf(t, len(filters)+2, highestBind(query(selected.list)),
			"kueri daftar bagian %s tidak sepadan dengan argumennya", view)
	}

	// Rekap per PIC: lini bisnis, login, hari ini, dan sepasang batas tanggal.
	require.Equal(t, 5, highestBind(query("pic_summary")))
}

// highestBind mencari nomor bind tertinggi di sebuah kueri.
func highestBind(text string) int {
	bind := regexp.MustCompile(`:(\d+)`)

	highest := 0
	for _, match := range bind.FindAllStringSubmatch(text, -1) {
		index, _ := strconv.Atoi(match[1])
		if index > highest {
			highest = index
		}
	}
	return highest
}

func TestCountQueryFiltersExactlyLikeItsListQuery(t *testing.T) {
	// Bila keduanya berbeda, layar menampilkan jumlah halaman yang tidak sesuai isinya —
	// dan halaman terakhir menjadi kosong tanpa sebab yang terlihat. Yang dibandingkan
	// adalah klausa WHERE-nya, dipotong sebelum ORDER BY supaya paginasi tidak ikut
	// terbawa.
	for _, view := range claimViews {
		selected := plans[view]
		require.Equalf(t,
			whereClauseOf(query(selected.count)),
			whereClauseOf(query(selected.list)),
			"penyaring daftar dan pencacah bagian %s tidak sama", view)
	}
}

// whereClauseOf mengambil klausa WHERE sebuah kueri, tanpa ORDER BY dan paginasinya.
func whereClauseOf(text string) string {
	upper := strings.ToUpper(text)

	start := strings.Index(upper, "\n WHERE ")
	if start < 0 {
		return ""
	}

	clause := text[start:]
	if end := strings.Index(strings.ToUpper(clause), "\n ORDER BY "); end >= 0 {
		clause = clause[:end]
	}
	return strings.TrimSpace(clause)
}

func TestNoValueIsConcatenatedIntoSQL(t *testing.T) {
	// Inilah cacat terbesar layar lama: lima titik `{ASIS:…}` yang menyisipkan POTONGAN
	// SQL, bukan nilai — dan yang paling terbuka adalah kotak cari, yang merangkai isian
	// pengguna apa adanya. Keputusan "replikasi apa adanya" berlaku pada perilaku bisnis,
	// bukan pada celah injeksi (`08-TECHNICAL-STRATEGY.md` §4.3).
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %s memuat pola {ASIS:…} warisan", name)

		// Satu-satunya kueri yang menerima sisipan adalah `positions`, dan yang
		// disisipkan hanyalah DERET PENANDA bind — diuji tersendiri di
		// TestInListProducesOnlyBindMarkers.
		if name != "positions" {
			require.NotContainsf(t, text, "%s", "kueri %s tampak dirangkai lewat teks", name)
		}

		// Perangkaian `||` hanya sah pada pola LIKE, tempat literal `'%'` disambung ke
		// BIND — bukan ke nilai. Di luar baris LIKE, `||` berarti ada teks yang disusun
		// di dalam SQL, dan itulah yang dilarang.
		for _, line := range strings.Split(text, "\n") {
			if !strings.Contains(line, "||") {
				continue
			}
			require.Containsf(t, strings.ToUpper(line), " LIKE ",
				"kueri %s merangkai teks di luar pola LIKE: %s", name, strings.TrimSpace(line))
		}
	}
}

func TestInListProducesOnlyBindMarkers(t *testing.T) {
	// Inilah yang membuat sisipan pada kueri `positions` tetap sah: yang disusun adalah
	// PENANDA bind, yang panjangnya ditentukan JUMLAH baris — bukan ISI baris. Tidak ada
	// satu pun nilai yang dapat menyentuh teks SQL.
	markers := regexp.MustCompile(`^:\d+(, :\d+)*$`)

	for _, count := range []int{1, 2, 15, 100} {
		produced := inList(1, count)
		require.Truef(t, markers.MatchString(produced),
			"inList(1, %d) menghasilkan teks di luar penanda bind: %s", count, produced)
		require.Equal(t, count, strings.Count(produced, ":"))
	}

	require.Equal(t, ":3, :4", inList(3, 2), "nomor bind harus melanjutkan bind sebelumnya")

	// Halaman kosong tidak boleh menghasilkan `IN ()`, yang tidak sah di SQL mana pun.
	require.Equal(t, "NULL", inList(1, 0))
}

func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	// Daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3. TO_CHAR ikut dilarang karena kueri
	// lama memakainya untuk MEMBANDINGKAN tanggal — yang membuang index dan memaksa
	// pemindaian penuh. ROW_NUMBER ikut dilarang karena itulah cara kueri lama memaginasi,
	// dan `09-DATABASE-STRATEGY.md` §3.3 menggantinya dengan OFFSET ... FETCH NEXT.
	forbidden := []string{
		"NVL(", "SYSDATE", "DECODE(", "ROWNUM", "INSTR(", "LISTAGG(", "STRING_AGG(",
		"FROM DUAL", "ADD_MONTHS(", "MONTHS_BETWEEN(", "TO_CHAR(", "SELECT *",
		"ROW_NUMBER(", "TRUNC(",
	}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContainsf(t, upper, pattern,
				"kueri %s memakai %s yang tidak portabel", name, pattern)
		}
	}
}

func TestQueriesNeverCallStoredRoutines(t *testing.T) {
	// `D-02` menetapkan tidak ada pemanggilan stored procedure maupun function dari
	// aplikasi. Yang paling mudah terbawa kembali adalah GET_POSISI_PROGRESS_PNC, karena
	// satu panggilan menggantikan seluruh kueri `positions`.
	routines := []string{
		"GET_POSISI_PROGRESS_PNC", "GET_POSISI_PROGRESS2", "PROGRESS_CLAIM_PNC",
		"GETSELISIHJAM", "GETCURRENCYSTANDARD", "BEGIN ",
	}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, routine := range routines {
			require.NotContainsf(t, upper, routine,
				"kueri %s memanggil rutin basis data %s (D-02)", name, routine)
		}
	}
}

func TestQueriesNeverWrite(t *testing.T) {
	// SELURUH tabel yang dibaca modul ini milik sistem lama. Menulis satu saja melanggar
	// `P-1`, dan akibatnya bukan galat melainkan dua sistem yang saling menimpa. Region
	// yang memang menulis di layar lama sengaja berada di luar lingkup.
	//
	// Batas kata dipakai, bukan pencocokan teks biasa: kolom `ID_UPDATE` memuat kata
	// "UPDATE" di dalamnya, dan pencocokan polos akan melaporkan kueri baca yang
	// sepenuhnya sah sebagai kueri tulis.
	writing := regexp.MustCompile(`\b(INSERT|UPDATE|DELETE|MERGE|TRUNCATE|COMMIT)\b`)

	for name, text := range queries {
		found := writing.FindString(strings.ToUpper(text))
		require.Emptyf(t, found,
			"kueri %s tampak menulis (%s); modul ini hanya membaca", name, found)
	}
}

func TestOnlyClaimListQueriesPaginateInSQL(t *testing.T) {
	// Layar ini BENAR-BENAR memaginasi di basis data, berbeda dari Inbox Admin. Yang harus
	// dijaga adalah pasangannya: pencacah yang ikut memaginasi akan melaporkan jumlah SATU
	// HALAMAN sebagai jumlah seluruh baris.
	paginating := map[string]bool{}
	for _, view := range claimViews {
		paginating[plans[view].list] = true
	}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		hasPaging := strings.Contains(upper, "FETCH NEXT") || strings.Contains(upper, "OFFSET ")

		require.Equalf(t, paginating[name], hasPaging,
			"kueri %s: paginasi di SQL hanya boleh ada pada kueri daftar klaim", name)
	}
}

func TestEveryLikeUsesEscape(t *testing.T) {
	// Tanpa ESCAPE, satu tanda `%` yang diketik pengguna berubah menjadi pola dan
	// mengembalikan seluruh isi daftar.
	for name, text := range queries {
		for _, line := range strings.Split(text, "\n") {
			upper := strings.ToUpper(line)
			if !strings.Contains(upper, " LIKE ") {
				continue
			}
			require.Containsf(t, upper, "ESCAPE",
				"kueri %s memakai LIKE tanpa ESCAPE pada baris: %s",
				name, strings.TrimSpace(line))
		}
	}
}

func TestBranchFilterIsAbsentEverywhere(t *testing.T) {
	// Penyaring cabang MENUNGGU API pengganti DB Link HRD (`R-03`). Uji ini menjaga agar
	// ia tidak masuk diam-diam lewat DB Link — yang akan menembus batas yang sengaja belum
	// dilewati.
	for name, text := range queries {
		upper := strings.ToUpper(text)
		require.NotContainsf(t, upper, "@ASMD", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, upper, "V_HRD_MST", "kueri %s membaca master HRD", name)
		require.NotContainsf(t, upper, "LST_USER_ASURANSI",
			"kueri %s membaca master user asuransi", name)
	}
}

func TestClosedClaimsAreExcludedEverywhere(t *testing.T) {
	// `STSKLAIM NOT IN ('1','2','3')` adalah yang membuat layar ini Inbox: barisnya hilang
	// begitu klaimnya selesai, ditolak, atau dibatalkan. Kueri yang melewatkannya akan
	// menampilkan klaim yang sudah tutup sebagai pekerjaan yang menunggu.
	for _, name := range []string{
		plans[inboxprogressclaim.ViewOutstanding].list,
		plans[inboxprogressclaim.ViewOutstanding].count,
		plans[inboxprogressclaim.ViewNextFollowUp].list,
		plans[inboxprogressclaim.ViewNextFollowUp].count,
		"pic_summary",
	} {
		require.Containsf(t, query(name), "STSKLAIM NOT IN ('1', '2', '3')",
			"kueri %s tidak membuang klaim yang sudah tutup", name)
	}
}

func TestNextFollowUpQueryFiltersByDueDate(t *testing.T) {
	// Satu-satunya hal yang membedakan region ini dari Outstanding. Bila saringannya
	// hilang, kedua region menampilkan isi yang sama persis dan tidak ada yang menyadarinya
	// sampai seseorang menghitung barisnya.
	for _, name := range []string{
		plans[inboxprogressclaim.ViewNextFollowUp].list,
		plans[inboxprogressclaim.ViewNextFollowUp].count,
	} {
		text := query(name)
		require.Contains(t, text, "b.USER_INPUT = a.PIC",
			"saringan jatuh tempo harus melihat catatan PIC klaim itu sendiri")
		require.Contains(t, text, "MAX(b.NEXT_FOLLOWUP)")
	}

	// Outstanding TIDAK boleh membawanya.
	require.NotContains(t, query(plans[inboxprogressclaim.ViewOutstanding].list),
		"b.USER_INPUT")
}

func TestEmptyKeywordIsSentAsNull(t *testing.T) {
	empty := inboxprogressclaim.ClaimQuery{View: inboxprogressclaim.ViewOutstanding}
	require.Nil(t, plans[inboxprogressclaim.ViewOutstanding].filterArgs(empty)[0])

	filled := inboxprogressclaim.ClaimQuery{
		View: inboxprogressclaim.ViewOutstanding, Keyword: "PNCN",
	}
	require.Equal(t, "PNCN", plans[inboxprogressclaim.ViewOutstanding].filterArgs(filled)[0])
}

func TestKeywordSearchesAllThreeColumns(t *testing.T) {
	// Kolom PIC mudah terlewat saat membaca kuerinya, dan justru ia yang membuat mengetik
	// nama seorang petugas memunculkan seluruh klaim yang ia tangani.
	for _, view := range claimViews {
		text := query(plans[view].list)
		for _, column := range []string{"a.NOKLAIM", "a.NOPOLIS", "a.PIC"} {
			require.Containsf(t, text, "UPPER("+column+") LIKE",
				"kueri daftar bagian %s tidak menelusuri %s", view, column)
		}
	}
}

func TestPICSummaryCarriesAllFourBusinessLines(t *testing.T) {
	// Keempat cabang berasal dari Activity/GetProgressPerPIC-Act.xml. Cabang yang hilang
	// berarti satu lini bisnis diam-diam mengembalikan nol baris.
	text := query("pic_summary")
	for _, line := range inboxprogressclaim.BusinessLines() {
		require.Containsf(t, text, "'"+string(line)+"'",
			"kueri pic_summary tidak memuat cabang lini bisnis %s", line)
	}
}

func TestNullableDateBecomesNil(t *testing.T) {
	require.Nil(t, nullableDate(nil))

	at := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, at, nullableDate(&at))
}
