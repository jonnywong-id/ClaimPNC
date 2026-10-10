package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
)

// countQueries dan listQueries adalah keenam kueri modul ini, dikelompokkan menurut
// perannya supaya uji dapat memperlakukan keduanya berbeda pada paginasi.
var (
	countQueries = []string{"outstanding_count", "loss_adjuster_count", "internal_surveyor_count"}
	listQueries  = []string{"outstanding_list", "loss_adjuster_list", "internal_surveyor_list"}
)

// TestMissingQueryPanics memastikan nama kueri yang salah ketik terlihat segera.
func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

// TestEveryQueryExists memastikan keenam kueri yang dipakai kode benar-benar ada.
//
// Tanpa uji ini, kueri yang namanya salah ketik baru ketahuan saat permintaan pertama
// datang — dan yang dilihat pengguna adalah panik, bukan galat yang dapat dibaca.
func TestEveryQueryExists(t *testing.T) {
	for _, name := range append(append([]string{}, countQueries...), listQueries...) {
		require.NotEmptyf(t, query(name), "kueri %s kosong", name)
	}
}

// TestQueryNamesCoverEverySurveyorType memastikan kedua jenis surveyor punya pasangan
// kueri, dan jenis yang tidak dikenal DITOLAK.
//
// Butir terakhir yang menentukan: mengembalikan kueri bawaan untuk jenis tak dikenal akan
// menjalankan kueri surveyor internal untuk loss adjuster, dan hasilnya tampak sah
// seluruhnya salah.
func TestQueryNamesCoverEverySurveyorType(t *testing.T) {
	for _, kind := range []dashboardclaim.SurveyorType{
		dashboardclaim.SurveyorAdjuster,
		dashboardclaim.SurveyorInternal,
	} {
		count, list, err := queryNamesFor(kind)
		require.NoErrorf(t, err, "jenis surveyor %q seharusnya dikenal", kind)
		require.NotEmpty(t, query(count))
		require.NotEmpty(t, query(list))
	}

	_, _, err := queryNamesFor(dashboardclaim.SurveyorType("9"))
	require.Error(t, err, "jenis surveyor tak dikenal wajib ditolak, bukan jatuh ke kueri bawaan")
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
	filter := dashboardclaim.Filter{}.Normalize()

	require.Len(t, filterArgs(filter), filterBindCount)
	require.Len(t, outstandingFilterArgs(filter), outstandingBindCount,
		"Outstanding menyiapkan argumen panel penyaring: Nopolis, No Klaim, PIC, "+
			"Status Transfer, dan Status Pembayaran")

	// Jumlahnya BERBEDA per keluarga sejak panel penyaring dibangun, jadi uji memilih
	// konstanta yang benar menurut kuerinya — bukan memakai satu angka untuk semuanya.
	bind := func(name string) int {
		if strings.HasPrefix(name, "outstanding_") {
			return outstandingBindCount
		}
		return filterBindCount
	}

	for _, name := range countQueries {
		require.Equalf(t, bind(name), highestBind(t, query(name)),
			"kueri hitung %s memakai nomor bind tertinggi yang tidak sesuai", name)
	}

	for _, name := range listQueries {
		require.Equalf(t, bind(name)+2, highestBind(t, query(name)),
			"kueri daftar %s harus memakai dua penanda tambahan: offset dan limit", name)
	}
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
	for name := range queries {

		// Probe '-periksa' juga dikecualikan: ia sengaja TIDAK berparameter. Yang
		// diperiksanya parse pernyataan, bukan hasilnya. Bentuknya dijaga TestProbesAreInert.
		if isProbe(name) {
			continue
		}

		numbers := bindNumbers(t, query(name))
		require.NotEmptyf(t, numbers, "kueri %s tidak punya penanda parameter", name)

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

// TestOutstandingListAndCountShareTheSameWhere menjaga tile yang MEMANG harus sejalan.
//
// Pada tile Outstanding, kueri hitung dan kueri daftar membaca tabel yang sama dengan syarat
// yang sama, sehingga angka kartu dan jumlah baris telusurnya wajib cocok. Penyimpangannya
// TIDAK menghasilkan galat apa pun: pengguna membaca satu angka lalu menemukan jumlah baris
// yang lain.
func TestOutstandingListAndCountShareTheSameWhere(t *testing.T) {
	require.Equal(t,
		whereClause(t, query("outstanding_count")),
		whereClause(t, query("outstanding_list")),
		"syarat WHERE outstanding_count dan outstanding_list berbeda — totalnya akan menyimpang dari barisnya")
}

// TestCountAndListMayDifferForSurveyTiles menjaga keputusan Work Owner 2026-09-26.
//
// Pada kedua tile survei, kueri hitung dan kueri daftar MEMANG membaca tabel yang berbeda:
// yang menghitung berjalan atas KLAIM, yang mendaftar berjalan atas SURVEI. Itu bentuk Pega
// apa adanya, dan ketiga butir terbuka diputuskan mengikutinya (`P-5`).
//
// Uji ini sengaja menuntut keduanya BERBEDA, bukan sekadar membiarkannya. Dengan begitu,
// siapa pun yang kelak menyeragamkannya harus menghapus uji ini lebih dulu — dan penghapusan
// itu yang akan menanyakan alasannya. Tanpa penjaga, penyeragaman diam-diam akan lolos
// review sebagai "perbaikan" dan memunculkan selisih tak terjelaskan pada gerbang 1.
func TestCountAndListMayDifferForSurveyTiles(t *testing.T) {
	for _, pair := range []struct{ count, list string }{
		{"loss_adjuster_count", "loss_adjuster_list"},
		{"internal_surveyor_count", "internal_surveyor_list"},
	} {
		require.NotEqualf(t, whereClause(t, query(pair.count)), whereClause(t, query(pair.list)),
			"%s dan %s kini sejalan — bila itu disengaja, keputusan Work Owner 2026-09-26 perlu dicabut lebih dulu",
			pair.count, pair.list)
	}
}

// TestSurveyCountsRunOverClaimTable menjaga bentuk kueri hitung terhadap sumbernya.
//
// Keduanya berjalan atas tabel KLAIM, bukan atas tabel survei. Itulah yang membuat angkanya
// setara dengan sistem lama.
//
// DIPERBARUI 2026-10-08: syarat "harus menggabung ke PC_ASSIGN_WORKLIST" diganti "tidak boleh
// menyentuh DATAPEGA sama sekali". Kedua tabel Pega menyatu di POOLDATA.T_CLAIMLIST_ADMIN,
// sehingga join itu hilang — dan bersamanya hilang pula penggandaan baris pada klaim yang
// memegang lebih dari satu penugasan.
func TestSurveyCountsRunOverClaimTable(t *testing.T) {
	for _, name := range []string{"loss_adjuster_count", "internal_surveyor_count"} {
		statement := query(name)

		require.Containsf(t, statement, "'ASM-FW-GCNMFW-Work-PNC'",
			"kueri %s harus berjalan atas tabel klaim", name)
		require.Containsf(t, statement, "POOLDATA.T_CLAIMLIST_ADMIN",
			"kueri %s harus berjalan atas tabel datar, bukan tabel Pega", name)
		require.NotContainsf(t, statement, "DATAPEGA.",
			"kueri %s masih menyentuh skema DATAPEGA, yang sudah tidak dipakai lagi", name)
		require.Containsf(t, statement, "COALESCE(SUM(",
			"kueri %s harus memakai SUM(CASE …) seperti kueri lamanya, dibungkus COALESCE", name)
	}
}

// TestSurveyorTypeIsPinnedPerQuery memastikan setiap kueri survei menyaring jenis yang benar.
//
// Nilainya tertanam di dalam teks SQL dan bukan dikirim sebagai parameter, karena kedua
// kueri memang berbeda struktur — bukan satu kueri berparameter. Uji ini yang menjaga
// keduanya tidak tertukar saat disunting.
func TestSurveyorTypeIsPinnedPerQuery(t *testing.T) {
	for _, name := range []string{"loss_adjuster_count", "loss_adjuster_list"} {
		require.Containsf(t, query(name), "SURVEYORTYPE_1 = '2'",
			"kueri %s harus menyaring loss adjuster", name)
	}
	for _, name := range []string{"internal_surveyor_count", "internal_surveyor_list"} {
		require.Containsf(t, query(name), "SURVEYORTYPE_1 = '1'",
			"kueri %s harus menyaring surveyor internal", name)
	}
}

// TestBusinessLineRepeatedFiveTimes menjaga bahwa nilai lini bisnis benar-benar dikirim pada
// kelima kemunculannya.
//
// Satu saja yang terlewat membuat cabang itu tidak pernah benar — dan akibatnya BUKAN galat
// melainkan daftar yang kehilangan seluruh baris lini tersebut.
func TestBusinessLineRepeatedFiveTimes(t *testing.T) {
	filter := dashboardclaim.Filter{Business: dashboardclaim.BusinessBonding}.Normalize()
	value := string(dashboardclaim.BusinessBonding)

	// Pada kueri survei lini bisnis menempati :1…:5.
	for index := 0; index < 5; index++ {
		require.Equalf(t, value, filterArgs(filter)[index],
			"penanda :%d tidak membawa nilai lini bisnis", index+1)
	}

	// Pada kueri outstanding ia menempati :4…:8, karena pencarian ditulis lebih dulu.
	for index := 3; index < 8; index++ {
		require.Equalf(t, value, outstandingFilterArgs(filter)[index],
			"penanda :%d tidak membawa nilai lini bisnis", index+1)
	}
}

// TestEmptyFilterSendsNulls memastikan penyaring yang tidak diisi dikirim sebagai NULL.
//
// Pola "NULL berarti tidak menyaring" itulah yang membuat satu teks SQL melayani seluruh
// kombinasi. Mengirim teks kosong akan membuat `LIKE '%%'` — yang kebetulan juga cocok
// dengan semuanya, tetapi memaksa basis data memindai setiap baris.
func TestEmptyFilterSendsNulls(t *testing.T) {
	filter := dashboardclaim.Filter{}.Normalize()

	for _, index := range []int{5, 6, 7} {
		require.Nilf(t, filterArgs(filter)[index],
			"penanda :%d seharusnya NULL saat kotak cari kosong", index+1)
	}
	for _, index := range []int{0, 1, 2} {
		require.Nilf(t, outstandingFilterArgs(filter)[index],
			"penanda :%d seharusnya NULL saat kotak cari kosong", index+1)
	}

	// Lini bisnis TIDAK pernah NULL: kosong berarti 'ALL', dan kueri membandingkannya
	// dengan teks 'ALL', bukan dengan NULL.
	require.Equal(t, string(dashboardclaim.BusinessAll), filterArgs(filter)[0])
}

// TestCountFilterDropsPaging memastikan kueri hitung tidak pernah membawa paginasi.
//
// Kueri hitung menghitung SELURUH baris yang cocok. Membiarkan LIMIT ikut terbawa akan
// membuat angka pada kartu berhenti di 25 berapa pun isinya — dan itu tampak seperti angka
// yang sah.
func TestCountFilterDropsPaging(t *testing.T) {
	filter := dashboardclaim.Filter{Limit: 50, Offset: 100}.Normalize().CountFilter()

	require.Zero(t, filter.Limit)
	require.Zero(t, filter.Offset)
}

// TestWithPagingDoesNotMutateFilters menjaga bahwa kueri hitung dan kueri daftar dapat
// memakai irisan argumen yang sama dalam satu permintaan.
//
// Keduanya memang menerimanya berurutan di ListOutstanding dan ListSurvey. Menambahkan ke
// irisan yang sama akan membuat kueri hitung ikut membawa paginasi pada pemanggilan
// berikutnya — dan angkanya berubah tanpa ada yang mengubah apa pun.
func TestWithPagingDoesNotMutateFilters(t *testing.T) {
	filter := dashboardclaim.Filter{}.Normalize()
	filters := filterArgs(filter)

	first := withPaging(filters, filter)
	second := withPaging(filters, filter)

	require.Len(t, filters, filterBindCount, "irisan penyaring ikut berubah setelah withPaging")
	require.Len(t, first, filterBindCount+2)
	require.Len(t, second, filterBindCount+2)
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

// TestWritersAreNamed memagari daftar kueri yang boleh MENULIS.
//
// Uji ini sebelumnya bernama TestOnlyTransferWrites dan menegaskan satu penulis saja, yang
// menyasar tabel milik aplikasi sendiri — penjabaran `P-1`: satu tabel satu penulis, dan
// selama masa paralel tabel klaim milik Pega.
//
// # Kenapa daftarnya bertambah, dan kenapa itu BUKAN pelonggaran
//
// Work Owner memutuskan (2026-10-06) tombol Assign mengikuti Pega: PIC Teknik dipindahkan
// langsung, bukan dicatat sebagai permintaan. Keputusan itu berdiri di atas satu kenyataan —
// antrean permintaan tidak punya pelaksana, sehingga permintaan menumpuk dan tidak pernah
// dijalankan.
//
// Akibatnya modul ini kini menulis `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, dan itu tulisan PERTAMA
// aplikasi ini ke skema Pega. Justru karena itu penjaganya diperketat, bukan dilonggarkan:
// setiap penulis disebut NAMANYA beserta sasarannya, sehingga penulis berikutnya — atau
// penulis lama yang berpindah sasaran — menuntut uji ini disunting lebih dulu.
func TestWritersAreNamed(t *testing.T) {
	// Setiap penulis beserta satu-satunya sasaran yang boleh disentuhnya.
	penulis := map[string]string{
		// Tabel milik PEGA. Satu kolom — lihat pindahpic.sql.
		//
		// Keduanya menulis tabel yang sama: yang satu satu baris, yang satu seluruh klaim
		// milik satu petugas.
		"pindah_pic":        "POOLDATA.T_CLAIMLIST_ADMIN",
		"pindah_pic_massal": "POOLDATA.T_CLAIMLIST_ADMIN",

		// "Select All" lintas halaman. Syaratnya salinan persis outstanding_count, dijaga
		// TestSelectAllMovesExactlyWhatTheListShows.
		"pindah_pic_saring": "POOLDATA.T_CLAIMLIST_ADMIN",

		// Master dan ringkasan milik POOLDATA, ditulis Pega lewat PNC_ReassignPNCTeknik.
		// `pencacah_turun` DICABUT 2026-10-07: kueri Pega-nya menurunkan `TOTAL_JOB` saja,
		// dan kolom itu tidak ada pada tabel ini — ia milik view `V_MST_USER_TEKNIS`.
		// Lihat pindahpic.sql.
		"pencacah_naik": "POOLDATA.MST_USER_TEKNIK",
		"dashboard_pic": "POOLDATA.PEGA_DASHBOARDPNC",
	}

	for name, sasaran := range penulis {
		statement, known := queries[name]
		require.Truef(t, known, "kueri penulis %s tidak ada", name)
		require.Containsf(t, strings.ToUpper(statement), sasaran,
			"kueri %s tidak lagi menyasar %s — penulis yang berpindah sasaran tidak "+
				"menghasilkan galat apa pun sampai ia menulis ke tempat yang salah",
			name, sasaran)
	}

	// SATU-SATUNYA kueri yang boleh menyentuh skema Pega adalah pindah_pic. Sisanya, baik
	// penulis maupun pembaca-yang-berubah-menjadi-penulis, tidak.
	for name, statement := range queries {
		upper := strings.ToUpper(statement)

		isPenulis := false
		for known := range penulis {
			if name == known {
				isPenulis = true
				break
			}
		}
		if isPenulis {
			continue
		}

		// Probe berbentuk UPDATE, dan itu DISENGAJA — lihat pindahpic.sql. Ia tidak
		// terdaftar sebagai penulis karena ia tidak menulis: TestProbesAreInert
		// membuktikan setiap probe memuat 'WHERE 1 = 0', sehingga pengecualian ini
		// tidak dapat dipakai menyelundupkan penulis sungguhan.
		if isProbe(name) {
			continue
		}

		for _, forbidden := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "} {
			require.NotContainsf(t, upper, forbidden,
				"kueri %s memuat %q tetapi tidak terdaftar sebagai penulis — "+
					"tambahkan namanya di uji ini lebih dulu",
				name, strings.TrimSpace(forbidden))
		}
	}
}

// TestOnlyOneQueryWritesPegaSchema memagari batas yang paling mudah dilanggar diam-diam.
//
// Menulis skema `DATAPEGA` menuntut hak basis data tersendiri dan menyentuh tabel yang
// dibaca 116 rule Pega. Satu kueri boleh melakukannya; penambahan berikutnya harus menempuh
// permintaan tertulis dan persetujuan Work Owner (`D-63`), bukan sekadar satu baris SQL.
func TestOnlyPICMovesWritePegaSchema(t *testing.T) {
	diizinkan := map[string]bool{
		"pindah_pic":        true,
		"pindah_pic_massal": true,
		"pindah_pic_saring": true,

		// Probe hak akses. Ia berbentuk UPDATE dengan sengaja — hak SELECT pada tabel ini
		// sudah dimiliki, sehingga probe SELECT akan lulus sementara tombol Transfer tetap
		// menjawab 503. Ia tidak mengubah baris: TestProbesAreInert membuktikannya.
		"pindah_pic_check": true,
	}

	for name, statement := range queries {
		upper := strings.ToUpper(statement)
		if !strings.Contains(upper, "DATAPEGA.") {
			continue
		}

		menulis := false
		for _, kata := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "} {
			if strings.Contains(upper, kata) {
				menulis = true
				break
			}
		}
		if !menulis {
			continue
		}

		require.Truef(t, diizinkan[name],
			"kueri %s menulis skema DATAPEGA — hanya pindah_pic dan pindah_pic_massal yang "+
				"boleh, dan penambahannya menempuh D-63",
			name)
	}
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
		"TRUNC(":    "pakai CAST(… AS DATE)",
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
// Kedalaman kurung ikut dihitung karena keempat kueri memuat `WHERE` DI DALAM subkueri —
// EXISTS pada kueri survei, dan pencarian nomor klaim induk pada kueri daftarnya. Mencari
// kemunculan "WHERE" yang pertama akan menemukan salah satu dari keduanya, bukan syarat yang
// hendak dibandingkan.
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

// TestHoldingUsesItsOwnBindCount menjaga perbedaan yang mudah terlewat.
//
// Tab Inbox Tampungan PIC memakai DUA argumen penyaring, bukan delapan: kueri lamanya tidak
// punya penanda lini bisnis sama sekali.
//
// Memakai ulang filterArgs di sana akan mengirim enam argumen yang tidak punya penanda, dan
// Oracle menolaknya dengan ORA-01008 pada permintaan pertama yang datang — bukan saat kode
// ditulis, dan bukan pada mesin pengembang yang tidak punya Oracle.
func TestHoldingUsesItsOwnBindCount(t *testing.T) {
	args := holdingFilterArgs(dashboardclaim.Filter{}.Normalize())
	require.Len(t, args, 2)

	require.Equal(t, 2, highestBind(t, query("holding_count")))
	require.Equal(t, 4, highestBind(t, query("holding_list")),
		"kueri daftar tampungan harus memakai dua penanda tambahan: offset dan limit")
}

// TestHoldingListAndCountShareTheSameWhere menjaga agar totalnya tidak menyimpang dari
// barisnya.
func TestHoldingListAndCountShareTheSameWhere(t *testing.T) {
	require.Equal(t,
		whereClause(t, query("holding_count")),
		whereClause(t, query("holding_list")),
		"syarat WHERE holding_count dan holding_list berbeda — totalnya akan menyimpang dari barisnya")
}

// TestHoldingKeepsTheThreeConditions menjaga ketiga syarat yang mendefinisikan "tampungan".
//
// Tidak satu pun cukup sendirian, dan menghapus salah satunya mengubah isi tab menjadi
// populasi yang lain tanpa satu pun galat.
func TestHoldingKeepsTheThreeConditions(t *testing.T) {
	for _, name := range []string{"holding_count", "holding_list"} {
		statement := query(name)

		require.Containsf(t, statement, "'ServicePNC'",
			"kueri %s kehilangan syarat akun penampung", name)
		require.Containsf(t, statement, "USERTEKNIS_1 IS NULL",
			"kueri %s kehilangan syarat belum punya PIC Teknik", name)
		require.Containsf(t, statement, "POLICYNO IS NOT NULL",
			"kueri %s kehilangan syarat polis sudah terisi", name)
	}
}

// TestPendingMarkersExpandSafely menjaga satu-satunya tempat di modul ini yang MERANGKAI
// teks SQL.
//
// Yang dirangkai adalah PENANDA (`:1, :2, …`), bukan nilainya — jumlahnya memang berubah
// tiap permintaan. Uji ini memastikan batas itu tidak bergeser: nilainya tidak pernah

// isProbe mengenali kueri `-periksa` dari namanya.
//
// Konvensinya satu: akhiran `_check`. Ia dipakai TIGA penjaga sekaligus sebagai pengecualian,
// dan karena itu TestProbesAreInert menjaga konvensinya sendiri — tanpa itu, cukup menamai
// satu kueri `..._check` untuk melewati ketiganya.
func isProbe(name string) bool { return strings.HasSuffix(name, "_check") }

// TestProbesAreInert menjaga pintu yang dibuka tiga penjaga lain.
//
// `isProbe` dipakai sebagai pengecualian di TestBindMarkersAreUniqueAndAscending,
// TestWritersAreNamed, dan TestOnlyPICMovesWritePegaSchema. Tanpa uji ini, pengecualian itu
// menjadi lubang: satu kueri bernama `hapus_semua_check` akan melewati ketiganya sekaligus —
// termasuk penjaga yang melindungi skema milik Pega.
//
// Syaratnya dua, dan keduanya membuat probe tidak mungkin mengubah data:
//
//  1. memuat `WHERE 1 = 0` — nol baris tersentuh, sementara Oracle tetap memeriksa hak akses
//     dan keberadaan kolom saat mem-parse-nya;
//  2. tanpa penanda parameter — tidak ada nilai dari luar yang dapat masuk.
func TestProbesAreInert(t *testing.T) {
	jumlah := 0

	for name, statement := range queries {
		if !isProbe(name) {
			continue
		}
		jumlah++

		rapat := strings.Join(strings.Fields(strings.ToUpper(statement)), " ")
		require.Containsf(t, rapat, "WHERE 1 = 0",
			"probe %s tidak memuat `WHERE 1 = 0` — tanpa itu ia dapat menyentuh baris, "+
				"sementara namanya sudah melewatkannya dari tiga penjaga", name)

		require.Emptyf(t, bindNumbers(t, statement),
			"probe %s punya penanda parameter — probe tidak menerima nilai dari luar", name)
	}

	// Nol probe berarti penjaga ini diam-diam tidak memeriksa apa pun. Itu persis kelas uji
	// yang lulus tanpa menguji, dan modul ini sudah dua kali menemukannya.
	require.NotZero(t, jumlah, "tidak ada satu pun kueri probe — apakah akhirannya berubah?")
}

// TestProbesNameEveryColumnTheyGuard menutup lubang yang ditemukan 2026-10-08.
//
// Ketiga probe semula hanya menyebut kolom yang di-SET, tidak pernah kolom di klausa WHERE.
// Oracle memvalidasi SELURUH rujukan kolom saat mem-parse, jadi probe yang menyebut keduanya
// membuktikan keduanya ada — sementara probe yang menyebut satu hanya membuktikan satu.
//
// Akibat lubang itu TIDAK terlihat sebagai kegagalan uji maupun sebagai probe merah: `-periksa`
// menjawab hijau, lalu kueri nyatanya gagal `ORA-00904` saat tombol ditekan pertama kali.
//
// Yang paling berisiko `dashboard_pic`: `POOLDATA.PEGA_DASHBOARDPNC` satu-satunya tabel yang
// modul ini tulis TANPA PERNAH BACA, sehingga tidak ada kueri lain yang membuktikan `NOKLAIM`
// ada. Pasangannya ditulis tangan di sini — dan memang harus, karena yang dijaga adalah
// hubungan antara dua pernyataan, bukan bentuk salah satunya.
func TestProbesNameEveryColumnTheyGuard(t *testing.T) {
	pasangan := map[string]struct {
		dijaga string
		kolom  []string
	}{
		"pindah_pic_check":    {dijaga: "pindah_pic", kolom: []string{"USERTEKNIS_1", "PZINSKEY"}},
		"pencacah_check":      {dijaga: "pencacah_naik", kolom: []string{"COUNTER_QUOTA", "OPERATOR_ID"}},
		"dashboard_pic_check": {dijaga: "dashboard_pic", kolom: []string{"PIC", "NOKLAIM"}},

		// Sudah benar sejak awal, dan menjadi contoh polanya: ia menyebut SELURUH kolom
		// yang dibaca `pic_teknik_list`, termasuk `STS_AKTIF` yang hanya muncul di WHERE.
		// Komentarnya menjelaskan sebabnya — kueri itu sempat membaca `TOTAL_JOB`, kolom
		// yang tidak ada pada tabel itu, dan probe `SELECT 1` akan lulus terhadapnya.
		"pic_teknik_check": {dijaga: "pic_teknik_list", kolom: []string{
			"OPERATOR_ID", "MCL_NAME", "EMAIL", "TEAM_GROUP", "COUNTER_QUOTA", "STS_AKTIF",
		}},

		// `IDPEGA` ikut didaftarkan meski ia milik tabel seberang: kolom gabung yang tidak
		// ada membuat kueri gagal sama saja dengan kolom yang dibaca.
		"klaim_rincian_check": {dijaga: "klaim_rincian", kolom: []string{
			"PYID", "PZINSKEY", "PYSTATUSWORK", "STATUSCLAIM_1", "USERTEKNIS_1",
			"PXCREATEOPNAME", "PXCREATEDATETIME", "DATA_JSON", "IDPEGA", "PXOBJCLASS",
		}},
	}

	for probe, p := range pasangan {
		statement, ada := queries[probe]
		require.Truef(t, ada, "probe %s tidak ada lagi — pasangannya di uji ini ikut diperbarui?", probe)

		dijaga, ada := queries[p.dijaga]
		require.Truef(t, ada, "kueri %s yang dijaga %s tidak ada lagi", p.dijaga, probe)

		for _, kolom := range p.kolom {
			require.Containsf(t, strings.ToUpper(dijaga), kolom,
				"kueri %s tidak lagi menyebut %s — pasangan di uji ini sudah usang", p.dijaga, kolom)
			require.Containsf(t, strings.ToUpper(statement), kolom,
				"probe %s tidak menyebut %s, padahal %s memakainya. Probe yang tidak menyebut "+
					"sebuah kolom tidak membuktikan kolom itu ada, dan ia akan hijau "+
					"sementara fiturnya gagal ORA-00904", probe, kolom, p.dijaga)
		}
	}

	// Setiap probe wajib punya pasangan. Menambah probe baru tanpa mendaftarkannya di sini
	// mengembalikan lubang yang sama persis.
	for name := range queries {
		if !isProbe(name) {
			continue
		}
		_, terdaftar := pasangan[name]
		require.Truef(t, terdaftar,
			"probe %s belum terdaftar di TestProbesNameEveryColumnTheyGuard — daftarkan "+
				"beserta kueri yang dijaganya dan seluruh kolom yang dipakai kueri itu", name)
	}
}

// TestSelectAllMovesExactlyWhatTheListShows menjaga penjodohan yang paling mahal bila meleset.
//
// `pindah_pic_saring` memindahkan klaim yang cocok dengan penyaring layar. Bila syaratnya
// menyimpang satu baris saja dari `outstanding_count`, tombol "Select All" memindahkan
// himpunan yang BERBEDA dari yang tercentang — dan tidak ada galat yang muncul. Yang terjadi
// hanya klaim berpindah tanpa ada yang memintanya, ditemukan entah kapan.
//
// Satu perbedaan DIIZINKAN dan diperhitungkan di sini secara eksplisit: penandanya bergeser
// satu, karena `:1` dipakai operator tujuan. Pergeseran itu dibalik lebih dulu, bukan
// dimaafkan dengan melonggarkan perbandingannya.
func TestSelectAllMovesExactlyWhatTheListShows(t *testing.T) {
	hitung := query("outstanding_count")
	pindah := query("pindah_pic_saring")

	// Badan kueri hitung: dari FROM sampai habis.
	awal := strings.Index(hitung, "FROM")
	require.GreaterOrEqual(t, awal, 0, "outstanding_count tidak punya klausa FROM")
	harapan := geserPenanda(t, rapatkan(hitung[awal:]), -1)

	// Badan subkueri pada pernyataan pindah: dari FROM sampai kurung penutupnya.
	mulai := strings.Index(pindah, "FROM")
	require.GreaterOrEqual(t, mulai, 0, "pindah_pic_saring tidak punya subkueri")
	nyata := rapatkan(strings.TrimSuffix(strings.TrimSpace(pindah[mulai:]), ")"))

	require.Equal(t, harapan, nyata,
		"syarat pindah_pic_saring menyimpang dari outstanding_count — "+
			"Select All akan memindahkan klaim yang tidak terlihat di layar")
}

// rapatkan menyeragamkan spasi supaya indentasi subkueri tidak dibaca sebagai perbedaan.
func rapatkan(s string) string { return strings.Join(strings.Fields(s), " ") }

// geserPenanda menggeser setiap `:n` sebanyak delta.
func geserPenanda(t *testing.T, s string, delta int) string {
	t.Helper()

	return regexp.MustCompile(`:(\d+)`).ReplaceAllStringFunc(s, func(m string) string {
		n, err := strconv.Atoi(m[1:])
		require.NoError(t, err)
		return ":" + strconv.Itoa(n-delta)
	})
}

// TestSkemaDATAPEGATidakDipakaiLagi menjaga keputusan Work Owner 2026-10-08.
//
// Seluruh kueri modul ini — baca maupun tulis — berpindah dari `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`
// dan `DATAPEGA.PC_ASSIGN_WORKLIST` ke satu tabel datar `POOLDATA.T_CLAIMLIST_ADMIN`.
//
// Uji ini ada karena kembalinya DATAPEGA **tidak menghasilkan galat apa pun**: kueri lama tetap
// sah, tabelnya masih ada, dan hasilnya masih masuk akal. Yang berubah hanya dari mana angkanya
// berasal — dan itu tidak terlihat di layar.
func TestSkemaDATAPEGATidakDipakaiLagi(t *testing.T) {
	for name, statement := range queries {
		require.NotContainsf(t, strings.ToUpper(statement), "DATAPEGA.",
			"kueri %s menyentuh skema DATAPEGA; modul ini seluruhnya berjalan atas "+
				"POOLDATA.T_CLAIMLIST_ADMIN sejak 2026-10-08", name)
	}
}
