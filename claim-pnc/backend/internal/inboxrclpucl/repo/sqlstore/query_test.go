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

func TestEveryFlatTableQueryReadsTheFlatTable(t *testing.T) {
	// Ketiga tab dan layar kerja membaca POOLDATA.TC_PNC_PUCL, bukan tabel Pega. Satu kueri
	// yang tertinggal akan tetap berjalan — tabel Pega masih ada — dan mengembalikan baris
	// yang terbaca masuk akal, sehingga tidak ada apa pun yang menandakannya.
	for _, name := range flatTableQueries {
		text := query(name)
		require.Containsf(t, text, "POOLDATA.TC_PNC_PUCL",
			"kueri %s tidak membaca tabel datar", name)
		require.NotContainsf(t, text, "DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
			"kueri %s masih membaca tabel objek kerja Pega", name)
		require.NotContainsf(t, text, "PC_ASSIGN_WORKBASKET",
			"kueri %s masih menggabung tabel penugasan Pega", name)
		require.NotContainsf(t, text, "PC_ASSIGN_WORKLIST",
			"kueri %s membaca tabel penugasan per orang — tabel yang salah", name)
	}
}

func TestNoFlatTableQueryUsesALegacyColumnName(t *testing.T) {
	// Nama kolom tabel datar BERBEDA dari nama kolom Pega yang digantikannya: akhiran `_1`
	// dibuang dan kata dipisah garis bawah (`Database/CREATE_TABLE_3.SQL`).
	//
	// Satu nama yang tertinggal pada bentuk lama TIDAK terlihat saat build — ia gagal dengan
	// ORA-00904 pada permintaan pertama di produksi. Uji ini yang menangkapnya lebih dulu.
	for _, name := range flatTableQueries {
		text := query(name)
		for _, column := range legacyColumnNames {
			require.NotContainsf(t, text, column,
				"kueri %s memakai nama kolom Pega %s; tabel datar menamainya lain",
				name, column)
		}
	}
}

func TestEveryListQueryFiltersItsSharedWorkbasket(t *testing.T) {
	// Gabungan ke PC_ASSIGN_WORKBASKET digantikan kolom pada baris klaimnya sendiri.
	// Penyaringnya tetap wajib: tanpanya, ketiga tab menampilkan seluruh isi tabel datar
	// tanpa melihat antrean mana pun.
	for _, name := range listQueries {
		require.Containsf(t, query(name), "p.ASSIGNED_OPERATOR_ID = :1",
			"kueri %s tidak menyaring akun antreannya", name)
	}
}

func TestEveryListQueryExcludesCompletedWork(t *testing.T) {
	// Penyaringnya `<>`, bukan `=`. Satu tanda yang salah membalik seluruh isi layar: yang
	// tampil menjadi klaim yang sudah tuntas.
	for _, name := range listQueries {
		require.Containsf(t, query(name), "p.STATUS_WORK <> :2",
			"kueri %s tidak mengecualikan pekerjaan yang selesai", name)
	}
}

func TestTheLetterFilterIsInvertedBetweenTabs(t *testing.T) {
	// Inilah penyaring yang memisahkan tab pertama dari dua tab lainnya. Membaliknya
	// menukar isi tab tanpa satu pun galat — dan kedua tab punya kolom yang IDENTIK,
	// sehingga tidak ada apa pun di layar yang menandakannya.
	cetak := query("list_cetak_surat")
	require.Contains(t, cetak, "p.TGL_CETAK_DOKUMEN_PUCL IS NULL")
	require.NotContains(t, cetak, "TGL_CETAK_DOKUMEN_PUCL IS NOT NULL")

	for _, name := range printedQueries {
		require.Containsf(t, query(name), "p.TGL_CETAK_DOKUMEN_PUCL IS NOT NULL",
			"kueri %s seharusnya menyaring surat yang SUDAH dicetak", name)
	}
}

func TestOnlyTheCetakSuratQueryFiltersTheCaseStatus(t *testing.T) {
	// `STATUS_CASE = '0'` hanya ada di Report Definition tab pertama. Menambahkannya di
	// tab lain akan menyembunyikan baris yang di Pega terlihat.
	require.Contains(t, query("list_cetak_surat"), "p.STATUS_CASE = :3")

	for _, name := range printedQueries {
		require.NotContainsf(t, query(name), "STATUS_CASE",
			"kueri %s tidak seharusnya menyaring STATUS_CASE", name)
	}
}

func TestOnlyThePrintedQueriesFilterTheApprovalFlag(t *testing.T) {
	require.NotContains(t, query("list_cetak_surat"), "PUCL_APPROVE")

	for _, name := range printedQueries {
		require.Containsf(t, query(name), "p.PUCL_APPROVE <> :3",
			"kueri %s tidak menyaring penanda persetujuan", name)
	}
}

func TestTheMSIGFilterIsInvertedBetweenTheTwoPrintedTabs(t *testing.T) {
	// SATU-SATUNYA perbedaan antara kedua kueri bersurat. Tertukarnya membuat seluruh
	// klaim non-MSIG muncul di tab Klaim MSIG dan sebaliknya.
	require.Contains(t, query("list_kelengkapan_dokumen"), "p.MSIG IS NULL")
	require.NotContains(t, query("list_kelengkapan_dokumen"), "p.MSIG = ")

	require.Contains(t, query("list_klaim_msig"), "p.MSIG = :4")
	require.NotContains(t, query("list_klaim_msig"), "p.MSIG IS NULL")

	// Tab pertama tidak menyaringnya sama sekali — Report Definition-nya memang tidak
	// punya penyaring itu, sehingga klaim jalur MSIG yang suratnya belum dicetak TETAP
	// muncul di sana.
	require.NotContains(t, query("list_cetak_surat"), "p.MSIG")
}

func TestEveryListQueryKeepsTheLegacySortOrder(t *testing.T) {
	// `ORDER BY 5 DESC, 7 DESC` pada SQL hasil generate Pega = PXCREATEDATETIME lalu PYID,
	// keduanya MENURUN — di tabel datar bernama TGL_CREATE_PUCL dan CLAIMID. Pemutus seri
	// yang menaik akan menukar urutan baris berwaktu sama di antara dua halaman.
	for _, name := range listQueries {
		require.Containsf(t, query(name),
			"ORDER BY p.TGL_CREATE_PUCL DESC, p.CLAIMID DESC",
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
	// Dua bind pertama sama di ketiga kueri. Urutan yang tertukar di salah satunya
	// menyaring akun antrean dengan nama status kerja — dan hasilnya nol baris, bukan galat.
	for label, entry := range allPlans(t) {
		args := entry.plan.args(samplePage())
		require.Equalf(t, inboxrclpucl.RCLPUCLWorkbasket, args[0], "%s bind :1", label)
		require.Equalf(t, inboxrclpucl.WorkStatusCompleted, args[1], "%s bind :2", label)
	}
}

func TestNoListPlanBindsTheWorkClass(t *testing.T) {
	// `TC_PNC_PUCL` tidak punya `PXOBJCLASS`, sehingga tidak ada bind untuknya. Bind yang
	// tertinggal akan membuat jumlah argumen tidak cocok dengan jumlah `:n` — tetapi di
	// Oracle, bind BERLEBIH yang tidak terpakai tidak selalu ditolak, sehingga ia dapat
	// menggeser seluruh penyaring satu posisi tanpa satu pun galat.
	for label, entry := range allPlans(t) {
		for i, arg := range entry.plan.args(samplePage()) {
			require.NotEqualf(t, inboxrclpucl.WorkClassClaim, arg,
				"%s masih mengikat kelas objek kerja pada bind :%d", label, i+1)
		}
	}
}

func TestTheMSIGPlanBindsTheMarkerBeforePagination(t *testing.T) {
	// Satu bind lebih banyak daripada tab bersaudaranya, dan ia disisipkan SEBELUM
	// paginasi. Menaruhnya sesudah akan mengirim offset sebagai penanda jalur dan penanda
	// jalur sebagai offset.
	entry := allPlans(t)["klaim_msig"]
	args := entry.plan.args(samplePage())

	require.Len(t, args, 6)
	require.Equal(t, inboxrclpucl.PUCLReturnedToAnalyst, args[2])
	require.Equal(t, inboxrclpucl.MSIGMarker, args[3])
	require.Equal(t, 100, args[4], "offset halaman ketiga berukuran 50")
	require.Equal(t, 50, args[5])
}

func TestTheCetakSuratPlanBindsTheCaseStatus(t *testing.T) {
	entry := allPlans(t)["cetak_surat"]
	args := entry.plan.args(samplePage())

	require.Len(t, args, 5)
	require.Equal(t, inboxrclpucl.ExpiryStatusActive, args[2])
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
	for _, name := range []string{
		"check_rclpucl", "check_columns", "check_detail", "check_laporan",
	} {
		require.Containsf(t, query(name), "WHERE 1 = 0",
			"kueri %s harus tidak menyentuh satu baris pun", name)
	}
}

func TestColumnProbeNamesEveryFilteringColumn(t *testing.T) {
	// Kolom yang TIDAK ADA dan kolom yang ada tetapi KOSONG menghasilkan layar yang
	// sama-sama kosong, dan hanya yang pertama yang merupakan kerusakan. Pemeriksaan ini
	// yang membedakan keduanya — terutama untuk MSIG, yang paling diragukan.
	//
	// Ia pula yang menangkap nama kolom bentuk LAMA yang tertinggal: tabel datar menamai
	// kelimanya berbeda dari tabel Pega.
	text := query("check_columns")
	for _, column := range []string{
		"TGL_CETAK_DOKUMEN_PUCL", "STATUS_CASE",
		"PUCL_APPROVE", "MSIG", "TGL_KIRIM_PUCL",
	} {
		require.Containsf(t, text, column, "kolom %s tidak ikut diperiksa", column)
	}
	require.Contains(t, text, "POOLDATA.TC_PNC_PUCL",
		"pemeriksaan kolom harus menyebut tabel datar, bukan tabel Pega")
}

func TestTheReportProbeChecksThePegaTables(t *testing.T) {
	// Laporan harian masih membaca kedua tabel Pega, dan kegagalannya BERBEDA akibatnya dari
	// kegagalan tabel datar: yang mati hanya tombol unduh tab "Cetak Surat", bukan layarnya.
	// Pemeriksaannya terpisah supaya galatnya menyebut hal yang benar.
	text := query("check_laporan")
	require.Contains(t, text, "DATAPEGA.PC_ASM_FW_GCNMFW_WORK")
	require.Contains(t, text, "DATAPEGA.PC_ASSIGN_WORKBASKET")
}

// ---------------------------------------------------------------------------
// Layar kerja satu klaim
// ---------------------------------------------------------------------------

func TestDetailReturnsItsOwnAliases(t *testing.T) {
	// Layar kerja TIDAK memakai alias yang sama dengan grid: ia membawa dua isian TURUNAN
	// dari anak klaim, dan tidak membawa "Lama Klaim" maupun "Status Kadaluarsa".
	require.Equal(t, detailColumns, aliasesOf(query("detail")))
}

func TestDetailIsKeyedByTheCaseNumber(t *testing.T) {
	// Kuncinya kini `CLAIMID` tabel datar, yang berisi NOMOR CASE — bukan `PZINSKEY`
	// berbentuk `ASM-FW-GCNMFW-WORK PNC-1865`.
	//
	// Nilainya datang dari `WorkItem.Reference`, yang ketiga kueri daftar pasok dari kolom
	// yang sama persis. Memakai kolom lain di salah satu tempat membuat setiap pembukaan
	// layar kerja mengembalikan "tidak ditemukan" — bukan galat.
	text := query("detail")
	require.Contains(t, text, "p.CLAIMID = :1")
	require.NotContains(t, text, ":2",
		"layar kerja hanya punya satu bind sejak penyaring kelas objek kerja hilang")
}

func TestDetailDoesNotFilterByQueueOrWorkStatus(t *testing.T) {
	// Layar kerja dibuka dengan KUNCI, bukan lewat antrean — begitu pula di Pega, tempat
	// tautannya mengirim `inskey` tanpa satu pun penyaring antrean. Klaim yang sudah
	// berpindah antrean sejak daftarnya dimuat tetap harus dapat dibuka.
	text := query("detail")
	require.NotContains(t, text, "ASSIGNED_OPERATOR_ID")
	require.NotContains(t, text, "STATUS_WORK")
}

func TestDetailReachesTheChildTablesThroughTClaimPNC(t *testing.T) {
	// INI JALUR YANG PALING MUDAH SALAH, dan salahnya TIDAK menghasilkan galat.
	//
	// Kolom bernama `CLAIMID` di tabel datar berisi NOMOR CASE; kolom bernama sama di
	// `T_CLAIM_OBJECTLIST` dan `T_CLAIM_ADJUSTMENT` berisi KUNCI TEKNIS berprefix.
	// Menggabungkan keduanya langsung selalu mengembalikan nol baris — dan nol baris di sini
	// terbaca persis seperti klaim yang memang tidak punya objek.
	//
	// `T_CLAIM_PNC` jembatannya: `CLAIMNO` sama dengan nomor case, `CLAIMID` sama dengan
	// kunci teknis.
	text := query("detail")

	require.Contains(t, text, "POOLDATA.T_CLAIM_PNC",
		"tanpa jembatan ini, kedua isian turunan tidak akan pernah ketemu")
	require.Contains(t, text, "c.CLAIMNO = p.CLAIMID")
	require.Contains(t, text, "ON c.CLAIMID = o.CLAIMID")

	// Prefix TIDAK boleh dirangkai sendiri. Ia benar untuk klaim warisan dan GAGAL DIAM-DIAM
	// untuk klaim ber-nomor `PNCN.YY.xxxx`, yang `D-22` dan `D-71` bebaskan dari prefix itu.
	require.NotContains(t, text, "ASM-FW-GCNMFW-WORK",
		"kunci teknis dibaca dari T_CLAIM_PNC, bukan dirangkai dari literal")
}

func TestDetailDerivesBothLetterFieldsFromTheFirstObject(t *testing.T) {
	// `SetDataLampiranSuratRCLPUCL_Act` memakai indeks `(1)` di setiap tingkat. "Pertama"
	// di sini ditetapkan tegas dengan `ORDER BY`, karena urutan page list klipboard Pega
	// tidak terbaca dari export mana pun — dan urutan yang tidak ditetapkan membuat isian
	// surat berubah-ubah antar pemanggilan.
	text := query("detail")

	require.Contains(t, text, "POOLDATA.T_CLAIM_OBJECTLIST")
	require.Contains(t, text, "POOLDATA.T_CLAIM_ADJUSTMENT")
	require.Contains(t, text, "ORDER BY o.OBJECTID")

	// `OBJECTCOVERAGEID`, BUKAN `COVERAGEID`.
	//
	// Nama pendeknya ditebak saat modul ini ditulis — DDL-nya tidak pernah diterima
	// (`R-08`) — dan tebakan itu membuat SETIAP pembukaan layar kerja gagal dengan
	// ORA-00904. Nama yang benar dibaca dari `ALL_TAB_COLUMNS` pada 2026-09-30.
	//
	// Uji ini menyebut nama kolomnya, bukan sekadar "ada ORDER BY": yang salah dulu adalah
	// namanya, dan uji yang hanya memeriksa bentuk kuerinya akan lolos dengan nama apa pun.
	require.Contains(t, text, "ORDER BY j.OBJECTCOVERAGEID, j.ADJUSTMENTID")
	require.NotContains(t, text, "j.COVERAGEID",
		"kolomnya bernama OBJECTCOVERAGEID; nama pendeknya tidak ada di tabel itu")

	// Adjustment-nya WAJIB terikat pada objek pertama, bukan sekadar adjustment pertama
	// klaim — jalur Pega-nya `ObjectList(1).ObjectCoverageList(1).AdjustmentList(1)`.
	require.Contains(t, text, "j.OBJECTID = (SELECT o2.OBJECTID")
}

func TestDetailCarriesOnlyWhatTheSectionDraws(t *testing.T) {
	// Layar kerja menggambar `Section/SectionLampiranSuratPUCL-Section.xml`, bukan grid.
	//
	// Kedua kolom di bawah ADA di tabel yang sama dan sudah dipakai ketiga kueri daftar,
	// sehingga membawanya ke sini nyaris tanpa biaya — dan itulah jebakannya. Section-nya
	// tidak memuat satu pun dari keduanya: seluruh properti yang dirujuknya adalah
	// `RCL_PUCL`, `KomentarAnalisator`, `UP`, `NIK`, `Policy.PolicyNo`, `BusinessUnitSeksi`,
	// `NamaPeserta`, `JumlahTagihan`, `Perihal`, `DateOfLoss`, dan `Keterangan1..3`.
	//
	// Versi pertama modul ini membawa keduanya karena keduanya "sudah di tangan". Uji ini
	// menahan pengulangannya: layar kerja mengikuti section-nya (`D-13`), bukan apa yang
	// kebetulan mudah diambil.
	text := query("detail")

	require.NotContains(t, text, "LETTER_PRINTED_AT")
	require.NotContains(t, text, "SENT_AT")
	require.NotContains(t, text, "TGL_CETAK_DOKUMEN_PUCL")
	require.NotContains(t, text, "TGL_KIRIM_PUCL")

	// Yang memang milik section tetap ada.
	require.Contains(t, text, "p.RCL_PUCL")
	require.Contains(t, text, "p.KOMENTAR_ANALISATOR")
	require.Contains(t, text, "p.DATE_OF_LOSS")
	require.Contains(t, text, "p.KOMENTAR_PUCL")
}

func TestDetailFillsInsuredNameAndSumInsuredFromOneSubquery(t *testing.T) {
	// "Nama Peserta" dan "UP" diisi dari SATU subkueri — `ObjectList(1).ObjectName` —
	// karena begitulah `SetDataLampiranSuratRCLPUCL_Act` mengisinya, dan Work Owner
	// menegaskan 2026-09-24 bahwa itu memang benar.
	//
	// Pernah ditambahkan subkueri kedua ke `T_CLAIM_OBJECTCOVERAGE` untuk mengambil
	// `SUMTSI` sebagai "UP". Itu dicabut. Uji ini menahannya lahir kembali: tabel coverage
	// TIDAK disentuh kueri ini sama sekali.
	text := query("detail")

	require.Contains(t, text, "FIRST_OBJECT_NAME")
	require.NotContains(t, text, "FIRST_SUM_TSI")
	require.NotContains(t, text, "SUMTSI")
	require.NotContains(t, text, "T_CLAIM_OBJECTCOVERAGE")
}

func TestDetailUsesProposeValueNotAdjustmentValue(t *testing.T) {
	// `PROPOSE_VALUE` dan `ADJUSTMENTVALUE` keduanya ada di tabel yang sama, dan keduanya
	// nilai uang yang masuk akal. Yang ditunjuk `.ProposeValue` adalah yang pertama;
	// menukarnya menampilkan jumlah tagihan yang salah TANPA satu pun galat.
	text := query("detail")
	require.Contains(t, text, "j.PROPOSE_VALUE")
	require.NotContains(t, text, "ADJUSTMENTVALUE")
}

func TestDetailProbeTouchesEveryChildTable(t *testing.T) {
	// Tanpa ketiga tabel, layar kerja tetap terbuka tetapi "Nama Peserta", "UP", dan
	// "Jumlah Tagihan" diam-diam kosong — dan kosong adalah keadaan yang sah bagi klaim
	// tanpa objek, sehingga tidak dapat dibedakan dari kerusakan.
	//
	// `T_CLAIM_PNC` ikut karena ia JEMBATAN kuncinya. Tanpanya kedua isian turunan tidak
	// dapat dicapai sama sekali, meski kedua tabel anaknya ada dan terbaca.
	text := query("check_detail")
	require.Contains(t, text, "POOLDATA.T_CLAIM_PNC")
	require.Contains(t, text, "POOLDATA.T_CLAIM_OBJECTLIST")
	require.Contains(t, text, "POOLDATA.T_CLAIM_ADJUSTMENT")
	require.Contains(t, text, "WHERE 1 = 0")
}

// ---------------------------------------------------------------------------
// Empat isian surat yang pindah dari clipboard ke kolom (2026-10-01)
// ---------------------------------------------------------------------------

func TestDetailReadsTheFourLetterColumnsFromTheFlatTable(t *testing.T) {
	// `PERIHAL`, `KETERANGAN1`, `KETERANGAN2`, `KETERANGAN3` ADA di `TC_PNC_PUCL` yang
	// berjalan dan terisi — dibaca langsung 2026-10-01, isinya cocok kata demi kata dengan
	// layar Pega.
	//
	// Keempatnya dibaca dari TABEL DATAR, bukan dari tabel objek kerja Pega: `detail` ada
	// di flatTableQueries, dan uji di berkas ini sudah menuntut kueri itu tidak menyentuh
	// tabel Pega mana pun.
	text := query("detail")

	for _, column := range letterColumnsBeyondTheSharedDDL {
		require.Containsf(t, text, "p."+column,
			"kueri detail tidak membaca kolom %s", column)
	}
}

func TestCheckColumnsProbesTheColumnsMissingFromTheSharedDDL(t *testing.T) {
	// Keempat kolom itu TIDAK ada di `Database/CREATE_TABLE_3.SQL` — berkas itu
	// mendefinisikan 26 kolom, tabel yang berjalan punya 30.
	//
	// Akibatnya nyata: portal yang tabelnya dibuat dari berkas itu kehilangan keempatnya,
	// dan `detail` gagal ORA-00904 pada klaim PERTAMA yang dibuka — bukan saat build, dan
	// bukan pada seluruh layar, melainkan hanya saat seseorang mengklik sebuah baris.
	//
	// `-periksa` karena itu wajib menyentuhnya, supaya keadaan itu terbaca lebih dulu.
	text := query("check_columns")

	for _, column := range letterColumnsBeyondTheSharedDDL {
		require.Containsf(t, text, "COUNT(p."+column+")",
			"check_columns tidak memeriksa kolom %s", column)
	}
}

func TestDetailStillDerivesTheThreeFieldsTheFlatTableDoesNotHold(t *testing.T) {
	// "Nama Peserta", "UP", dan "Jumlah Tagihan" TIDAK ada di `TC_PNC_PUCL`. Ketiganya tetap
	// diturunkan dari anak klaim, dan itu bukan sisa migrasi yang terlewat — tabel datarnya
	// memang tidak menyimpannya.
	//
	// Uji ini ada supaya penghapusannya menempuh keputusan: menghapus subkuerinya akan
	// mengosongkan tiga isian surat tanpa satu pun galat.
	text := query("detail")

	require.Contains(t, text, "POOLDATA.T_CLAIM_OBJECTLIST")
	require.Contains(t, text, "POOLDATA.T_CLAIM_ADJUSTMENT")
	require.Contains(t, text, "AS FIRST_OBJECT_NAME")
	require.Contains(t, text, "AS FIRST_PROPOSE_VALUE")
}
