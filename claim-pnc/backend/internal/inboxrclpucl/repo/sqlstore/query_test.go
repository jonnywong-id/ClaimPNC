package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
)

// samplePage adalah paginasi lengkap, dipakai memanggil penyusun argumen tiap kueri.
func samplePage() inboxrclpucl.Pagination {
	return inboxrclpucl.Pagination{Page: 3, Size: 50}
}

// sampleQuery menyusun permintaan untuk sebuah tab.
func sampleQuery(t *testing.T, code string) inboxrclpucl.Query {
	t.Helper()

	tab, found := inboxrclpucl.FindTab(code)
	require.Truef(t, found, "tab %s tidak ditemukan", code)

	return inboxrclpucl.Query{
		Tab:    tab,
		Caller: inboxrclpucl.Caller{Login: "PETUGASCONTOH"},
	}
}

type planCase struct {
	plan  plan
	query inboxrclpucl.Query
}

// allPlans mengembalikan ketiga rencana kueri beserta permintaannya.
//
// Ketiganya disebut lengkap, bukan disimpulkan dari daftar tab: daftar yang dihitung sendiri
// oleh uji akan ikut salah bila pemilihan kuerinya salah.
func allPlans(t *testing.T) map[string]planCase {
	t.Helper()

	cases := []struct {
		label string
		tab   string
	}{
		{"cetak_surat", inboxrclpucl.TabCetakSurat},
		{"kelengkapan_dokumen", inboxrclpucl.TabKelengkapanDokumen},
		{"klaim_msig", inboxrclpucl.TabKlaimMSIG},
	}

	result := map[string]planCase{}
	for _, c := range cases {
		q := sampleQuery(t, c.tab)
		selected, err := planFor(q)
		require.NoErrorf(t, err, "%s belum punya kueri", c.label)
		result[c.label] = planCase{plan: selected, query: q}
	}

	return result
}

func TestEverySelectableTabHasQuery(t *testing.T) {
	for _, tab := range inboxrclpucl.Tabs() {
		if tab.Blocked {
			continue
		}
		_, err := planFor(inboxrclpucl.Query{Tab: tab})
		require.NoErrorf(t, err, "tab %s (%s) belum punya kueri", tab.Code, tab.Name)
	}
}

func TestEveryTabPicksItsOwnQuery(t *testing.T) {
	// Ketiga tab dilayani kueri yang BERBEDA-BEDA. Kalau ada dua yang sama, salah satu
	// penyaringnya pasti hilang — dan di layar ini yang paling mungkin hilang adalah
	// penyaring `MSIG_1`, yang berarti isi dua tab tertukar tanpa satu pun galat.
	plans := allPlans(t)

	seen := map[string]string{}
	for label, entry := range plans {
		if previous, clash := seen[entry.plan.name]; clash {
			t.Fatalf("%s dan %s memakai kueri yang sama (%s)",
				previous, label, entry.plan.name)
		}
		seen[entry.plan.name] = label
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

// aliasesOf membaca alias di UJUNG setiap baris kolom sebuah kueri.
//
// Syarat "di ujung baris" penting: tanpanya, potongan seperti
// `TO_DATE(:3, 'YYYY-MM-DD')` dan nama tabel ikut terbaca sebagai alias.
func aliasesOf(text string) []string {
	alias := regexp.MustCompile(`(?im)\bAS\s+([A-Z_]+)\s*,?\s*$`)

	found := []string{}
	for _, match := range alias.FindAllStringSubmatch(text, -1) {
		found = append(found, strings.ToUpper(match[1]))
	}
	return found
}

func TestEveryListQueryReturnsTheSameAliases(t *testing.T) {
	// Satu pemindai melayani ketiga kueri. Begitu satu kueri memakai alias yang berbeda
	// atau urutannya bergeser, pemindai itu memasukkan nilai ke isian yang salah — dan
	// akibatnya BUKAN galat, melainkan kolom yang tertukar di layar.
	//
	// Di layar ini bahayanya khusus: dua kolom terakhir adalah `STATUSKLAIM_1` dan
	// pasangan judul yang bersilangan dengan modul lain. Pergeseran satu posisi
	// menampilkan status jalur di kolom "Status Kadaluarsa" dan sebaliknya — keduanya
	// teks yang masuk akal, sehingga tidak ada yang terlihat rusak.
	for _, name := range listQueries {
		require.Equalf(t, listColumns, aliasesOf(query(name)),
			"alias kueri %s berbeda dari listColumns", name)
	}
}

func TestDailyReportReturnsItsOwnAliases(t *testing.T) {
	// Laporan TIDAK memakai alias yang sama dengan grid, dan itu disengaja: ia punya
	// CLAIM_STATUS yang tidak ada di grid, dan tidak punya CLAIM_AGE yang ada di grid.
	//
	// Hanya alias pada SELECT terluar yang dihitung; subkueri di dalamnya memakai alias
	// yang sama sehingga tidak menambah apa pun yang berbeda.
	found := aliasesOf(query("daily_report"))

	for _, wanted := range reportColumns {
		require.Containsf(t, found, wanted,
			"alias %s hilang dari kueri daily_report", wanted)
	}
}

func TestEveryListQueryFiltersTheWorkClass(t *testing.T) {
	// Satu tabel Pega menampung beberapa kelas objek kerja, dan semuanya punya PYID,
	// POLICYNO, serta QQNAME. Kueri yang lupa menyaringnya mencampur klaim dengan berkas
	// lain — dan hasilnya tidak menghasilkan satu pun galat.
	for _, name := range listQueries {
		require.Containsf(t, query(name), "w.PXOBJCLASS = :1",
			"kueri %s tidak menyaring kelas objek kerja", name)
	}
}

func TestEveryListQueryJoinsTheSharedWorkbasket(t *testing.T) {
	// Ketiga tab membaca antrean BERSAMA (PC_ASSIGN_WORKBASKET), bukan penugasan per
	// orang (PC_ASSIGN_WORKLIST). Membaca tabel yang salah mengembalikan nol baris tanpa
	// satu pun galat.
	for _, name := range listQueries {
		text := query(name)
		require.Containsf(t, text, "DATAPEGA.PC_ASSIGN_WORKBASKET",
			"kueri %s tidak menggabung antrean bersama", name)
		require.NotContainsf(t, text, "PC_ASSIGN_WORKLIST",
			"kueri %s membaca tabel penugasan per orang — tabel yang salah", name)
		require.Containsf(t, text, "b.PXASSIGNEDOPERATORID = :2",
			"kueri %s tidak menyaring akun antreannya", name)
	}
}

func TestEveryListQueryExcludesCompletedWork(t *testing.T) {
	// Penyaringnya `<>`, bukan `=`. Satu tanda yang salah membalik seluruh isi layar: yang
	// tampil menjadi klaim yang sudah tuntas.
	for _, name := range listQueries {
		require.Containsf(t, query(name), "w.PYSTATUSWORK <> :3",
			"kueri %s tidak mengecualikan pekerjaan yang selesai", name)
	}
}

func TestTheLetterFilterIsInvertedBetweenTabs(t *testing.T) {
	// Inilah penyaring yang memisahkan tab pertama dari dua tab lainnya. Membaliknya
	// menukar isi tab tanpa satu pun galat — dan kedua tab punya kolom yang IDENTIK,
	// sehingga tidak ada apa pun di layar yang menandakannya.
	cetak := query("list_cetak_surat")
	require.Contains(t, cetak, "w.TANGGALCETAKDOKUMENPUCL_1 IS NULL")
	require.NotContains(t, cetak, "TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL")

	for _, name := range printedQueries {
		require.Containsf(t, query(name), "w.TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL",
			"kueri %s seharusnya menyaring surat yang SUDAH dicetak", name)
	}
}

func TestOnlyTheCetakSuratQueryFiltersTheCaseStatus(t *testing.T) {
	// `STATUSCASE_1 = '0'` hanya ada di Report Definition tab pertama. Menambahkannya di
	// tab lain akan menyembunyikan baris yang di Pega terlihat.
	require.Contains(t, query("list_cetak_surat"), "w.STATUSCASE_1 = :4")

	for _, name := range printedQueries {
		require.NotContainsf(t, query(name), "STATUSCASE_1",
			"kueri %s tidak seharusnya menyaring STATUSCASE_1", name)
	}
}

func TestOnlyThePrintedQueriesFilterTheApprovalFlag(t *testing.T) {
	require.NotContains(t, query("list_cetak_surat"), "PUCLAPPROVE_1")

	for _, name := range printedQueries {
		require.Containsf(t, query(name), "w.PUCLAPPROVE_1 <> :4",
			"kueri %s tidak menyaring penanda persetujuan", name)
	}
}

func TestTheMSIGFilterIsInvertedBetweenTheTwoPrintedTabs(t *testing.T) {
	// SATU-SATUNYA perbedaan antara kedua kueri bersurat. Tertukarnya membuat seluruh
	// klaim non-MSIG muncul di tab Klaim MSIG dan sebaliknya.
	require.Contains(t, query("list_kelengkapan_dokumen"), "w.MSIG_1 IS NULL")
	require.NotContains(t, query("list_kelengkapan_dokumen"), "w.MSIG_1 = ")

	require.Contains(t, query("list_klaim_msig"), "w.MSIG_1 = :5")
	require.NotContains(t, query("list_klaim_msig"), "w.MSIG_1 IS NULL")

	// Tab pertama tidak menyaringnya sama sekali — Report Definition-nya memang tidak
	// punya penyaring itu, sehingga klaim jalur MSIG yang suratnya belum dicetak TETAP
	// muncul di sana.
	require.NotContains(t, query("list_cetak_surat"), "MSIG_1")
}

func TestEveryListQueryKeepsTheLegacySortOrder(t *testing.T) {
	// `ORDER BY 5 DESC, 7 DESC` pada SQL hasil generate Pega = PXCREATEDATETIME lalu PYID,
	// keduanya MENURUN. Pemutus seri yang menaik akan menukar urutan baris berwaktu sama
	// di antara dua halaman.
	for _, name := range listQueries {
		require.Containsf(t, query(name),
			"ORDER BY w.PXCREATEDATETIME DESC, w.PYID DESC",
			"kueri %s tidak memakai urutan Report Definition", name)
	}
}

func TestEveryListQueryPaginatesInTheDatabase(t *testing.T) {
	for _, name := range listQueries {
		text := query(name)
		require.Containsf(t, text, "OFFSET", "kueri %s tidak dipaginasi", name)
		require.Containsf(t, text, "FETCH NEXT", "kueri %s tidak dipaginasi", name)
		require.Containsf(t, text, "COUNT(*) OVER ()",
			"kueri %s tidak menghitung jumlah seluruh baris", name)
	}
}

func TestNoQueryUsesForbiddenSQLPatterns(t *testing.T) {
	// `SELECT *` dan `TRUNC` pada kolom keduanya dilarang
	// (`08-TECHNICAL-STRATEGY.md` §4.3, `09-DATABASE-STRATEGY.md` §4). `TRUNC` patut
	// diperhatikan khusus di modul ini: kueri laporan lama memakainya pada kolom tanggal,
	// dan penggantinya adalah rentang setengah terbuka.
	for name, text := range queries {
		upper := strings.ToUpper(text)
		require.NotContainsf(t, upper, "SELECT *", "kueri %s memakai SELECT *", name)
		require.NotContainsf(t, upper, "TRUNC(", "kueri %s memakai TRUNC", name)
		require.NotContainsf(t, upper, "NVL(", "kueri %s memakai NVL", name)
		require.NotContainsf(t, upper, "SYSDATE", "kueri %s memakai SYSDATE", name)
	}
}

func TestNoQueryInlinesAFilterValue(t *testing.T) {
	// Nilai penyaring WAJIB lewat bind. Kueri lama menyisipkan kedua tanggal laporan —
	// isian yang diketik pengguna — langsung ke teks SQL-nya.
	for name, text := range queries {
		require.NotContainsf(t, text, "'RCLPUCL'",
			"kueri %s menuliskan akun antrean sebagai literal", name)
		require.NotContainsf(t, text, "'Resolved-Completed'",
			"kueri %s menuliskan status kerja sebagai literal", name)
		require.NotContainsf(t, text, "{",
			"kueri %s memuat penanda perangkaian gaya Pega", name)
	}
}

// ---------------------------------------------------------------------------
// Susunan bind
// ---------------------------------------------------------------------------

func TestEveryPlanBindsAsManyArgumentsAsItsQueryUses(t *testing.T) {
	// Jumlah bind yang tidak cocok dengan jumlah penanda `:n` menghasilkan galat basis
	// data yang baru muncul saat kueri dijalankan — di produksi, bukan saat build.
	marker := regexp.MustCompile(`:(\d+)`)

	for label, entry := range allPlans(t) {
		text := query(entry.plan.name)

		highest := 0
		for _, match := range marker.FindAllStringSubmatch(text, -1) {
			n := len(match[1])
			_ = n
			value := 0
			for _, digit := range match[1] {
				value = value*10 + int(digit-'0')
			}
			if value > highest {
				highest = value
			}
		}

		args := entry.plan.args(samplePage())
		require.Equalf(t, highest, len(args),
			"%s: kueri memakai %d bind, argumennya %d", label, highest, len(args))
	}
}

func TestEveryPlanBindsTheSharedFiltersInTheSameOrder(t *testing.T) {
	// Tiga bind pertama sama di ketiga kueri. Urutan yang tertukar di salah satunya
	// menyaring akun antrean dengan nama kelas objek kerja — dan hasilnya nol baris, bukan
	// galat.
	for label, entry := range allPlans(t) {
		args := entry.plan.args(samplePage())
		require.Equalf(t, inboxrclpucl.WorkClassClaim, args[0], "%s bind :1", label)
		require.Equalf(t, inboxrclpucl.RCLPUCLWorkbasket, args[1], "%s bind :2", label)
		require.Equalf(t, inboxrclpucl.WorkStatusCompleted, args[2], "%s bind :3", label)
	}
}

func TestTheMSIGPlanBindsTheMarkerBeforePagination(t *testing.T) {
	// Satu bind lebih banyak daripada tab bersaudaranya, dan ia disisipkan SEBELUM
	// paginasi. Menaruhnya sesudah akan mengirim offset sebagai penanda jalur dan penanda
	// jalur sebagai offset.
	entry := allPlans(t)["klaim_msig"]
	args := entry.plan.args(samplePage())

	require.Len(t, args, 7)
	require.Equal(t, inboxrclpucl.PUCLApproved, args[3])
	require.Equal(t, inboxrclpucl.MSIGMarker, args[4])
	require.Equal(t, 100, args[5], "offset halaman ketiga berukuran 50")
	require.Equal(t, 50, args[6])
}

func TestTheCetakSuratPlanBindsTheCaseStatus(t *testing.T) {
	entry := allPlans(t)["cetak_surat"]
	args := entry.plan.args(samplePage())

	require.Len(t, args, 6)
	require.Equal(t, inboxrclpucl.ExpiryStatusActive, args[3])
}

func TestUnknownTabHasNoQuery(t *testing.T) {
	_, err := planFor(inboxrclpucl.Query{Tab: inboxrclpucl.Tab{Code: "99"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "99")
}

// ---------------------------------------------------------------------------
// Laporan harian
// ---------------------------------------------------------------------------

func TestDailyReportKeepsBothUnionBranches(t *testing.T) {
	// `UNION`, bukan `UNION ALL`: klaim Personal Accident yang berada di antrean RCL/PUCL
	// memenuhi KEDUA cabang, dan dengan `UNION ALL` ia muncul dua kali di berkas.
	text := query("daily_report")

	require.Contains(t, text, "UNION")
	require.NotContains(t, strings.ToUpper(text), "UNION ALL")
	require.Contains(t, text, "w.GROUPPANEL_1 = :6",
		"cabang kedua harus menyaring Group Panel Personal Accident")
}

func TestDailyReportSecondBranchDoesNotJoinTheWorkbasket(t *testing.T) {
	// Cabang kedua sengaja TIDAK menggabung antrean bersama — ia mengambil seluruh klaim
	// PA pada rentang itu, termasuk yang tidak pernah masuk antrean RCL/PUCL. Itulah
	// sebabnya isi laporan berbeda dari isi tabel.
	text := query("daily_report")
	require.Equal(t, 1, strings.Count(text, "DATAPEGA.PC_ASSIGN_WORKBASKET"),
		"hanya cabang pertama yang menggabung antrean bersama")
}

func TestDailyReportUsesAHalfOpenDateRange(t *testing.T) {
	// Pengganti `TRUNC(kolom) BETWEEN …`, yang dilarang dan mematikan index. Batas atas
	// setengah terbuka memilih baris yang sama persis, termasuk yang punya komponen jam.
	text := query("daily_report")
	require.Equal(t, 2, strings.Count(text, "+ INTERVAL '1' DAY"),
		"kedua cabang harus memakai rentang setengah terbuka")
}

func TestDailyReportBindsEachRangeSeparately(t *testing.T) {
	// Kedua cabang menyaring rentang yang sama, tetapi penandanya berbeda — menulis `:3`
	// dua kali akan bergantung pada cara driver menafsirkan penanda berulang.
	text := query("daily_report")
	for _, marker := range []string{":3", ":4", ":7", ":8"} {
		require.Containsf(t, text, marker, "penanda %s hilang", marker)
	}
}

func TestDailyReportIsOrderedAndPaginated(t *testing.T) {
	// Kueri lama tidak punya ORDER BY sama sekali. Tanpa urutan, satu baris dapat muncul
	// di dua halaman sekaligus hilang dari halaman lain begitu hasilnya dipotong.
	text := query("daily_report")
	require.Contains(t, text, "ORDER BY r.SENT_AT DESC, r.CASE_ID DESC")
	require.Contains(t, text, "OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY")
}

func TestProbeQueriesTouchNoRows(t *testing.T) {
	for _, name := range []string{"check_rclpucl", "check_columns"} {
		require.Containsf(t, query(name), "WHERE 1 = 0",
			"kueri %s harus tidak menyentuh satu baris pun", name)
	}
}

func TestColumnProbeNamesEveryFilteringColumn(t *testing.T) {
	// Kolom yang TIDAK ADA dan kolom yang ada tetapi KOSONG menghasilkan layar yang
	// sama-sama kosong, dan hanya yang pertama yang merupakan kerusakan. Pemeriksaan ini
	// yang membedakan keduanya — terutama untuk MSIG_1, yang paling diragukan.
	text := query("check_columns")
	for _, column := range []string{
		"TANGGALCETAKDOKUMENPUCL_1", "STATUSCASE_1",
		"PUCLAPPROVE_1", "MSIG_1", "TANGGALKIRIMPUCL_1",
	} {
		require.Containsf(t, text, column, "kolom %s tidak ikut diperiksa", column)
	}
}
