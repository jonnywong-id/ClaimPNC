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
	require.Len(t, outstandingFilterArgs(filter), filterBindCount,
		"kedua penyusun argumen wajib menyiapkan jumlah yang sama — hanya urutannya yang berbeda")

	for _, name := range countQueries {
		require.Equalf(t, filterBindCount, highestBind(t, query(name)),
			"kueri hitung %s memakai nomor bind tertinggi yang tidak sama dengan filterBindCount", name)
	}

	for _, name := range listQueries {
		require.Equalf(t, filterBindCount+2, highestBind(t, query(name)),
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
		// transfer_pending dikecualikan: penandanya DISISIPKAN saat jalan oleh expandClaims,
		// sebanyak klaim yang diminta. Jumlahnya memang tidak diketahui di dalam teksnya, dan
		// bentuknya dijaga TestPendingMarkersExpandSafely.
		if name == "transfer_pending" {
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
// Keduanya berjalan atas tabel KLAIM dengan gabung ke PC_ASSIGN_WORKLIST — bukan atas tabel
// survei. Itulah yang membuat angkanya setara dengan sistem lama, termasuk penggandaan pada
// klaim yang memegang lebih dari satu penugasan.
func TestSurveyCountsRunOverClaimTable(t *testing.T) {
	for _, name := range []string{"loss_adjuster_count", "internal_surveyor_count"} {
		statement := query(name)

		require.Containsf(t, statement, "'ASM-FW-GCNMFW-Work-PNC'",
			"kueri %s harus berjalan atas tabel klaim", name)
		require.Containsf(t, statement, "PC_ASSIGN_WORKLIST",
			"kueri %s harus menggabung ke PC_ASSIGN_WORKLIST seperti kueri lamanya", name)
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

// TestOnlyTransferWrites memagari dua hal sekaligus.
//
// `P-1` menetapkan satu tabel hanya ditulis satu sistem, dan selama masa paralel SELURUH
// tabel klaim milik Pega. Tombol Transfer karena itu mencatat PERMINTAAN ke tabel milik
// aplikasi sendiri; penugasannya tetap dipindahkan Pega.
//
// Yang dijaga:
//
//  1. tidak ada kueri tulis BARU yang lolos tanpa sengaja — satu-satunya penulis disebut
//     namanya di sini, sehingga penambahan berikutnya harus menyunting uji ini lebih dulu;
//  2. penulis yang satu itu tidak berpindah sasaran ke skema milik Pega.
//
// Butir kedua yang paling mudah terlewat: mengubah nama tabel pada satu kueri tidak
// menghasilkan galat apa pun sampai ia benar-benar menulis ke tempat yang salah.
func TestOnlyTransferWrites(t *testing.T) {
	const penulis = "transfer_insert"

	require.Contains(t, queries[penulis], "POOLDATA.CPNC_PERMINTAAN_TRANSFER",
		"kueri tulis harus menyasar tabel milik aplikasi sendiri, bukan tabel warisan")
	require.NotContains(t, strings.ToUpper(queries[penulis]), "DATAPEGA.",
		"kueri tulis TIDAK boleh menyentuh skema milik Pega")

	for name, statement := range queries {
		if name == penulis {
			continue
		}
		upper := strings.ToUpper(statement)
		for _, forbidden := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "} {
			require.NotContainsf(t, upper, forbidden,
				"kueri %s memuat %q — hanya %s yang boleh menulis",
				name, strings.TrimSpace(forbidden), penulis)
		}
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
// menyentuh teks, dan penandanya tetap unik serta menaik.
func TestPendingMarkersExpandSafely(t *testing.T) {
	require.Contains(t, query("transfer_pending"), "/*CLAIMS*/",
		"penanda yang diganti expandClaims hilang dari kueri")

	ids := []string{"ASM-FW-GCNMFW-WORK PNC-1", "ASM-FW-GCNMFW-WORK PNC-2", "PNC-3"}
	statement, args := expandClaims(query("transfer_pending"), ids)

	require.Len(t, args, len(ids))
	require.NotContains(t, statement, "/*CLAIMS*/", "penanda tidak terganti")

	// Nilainya TIDAK boleh muncul di dalam teks SQL.
	for _, id := range ids {
		require.NotContains(t, statement, id,
			"nilai klaim terangkai ke dalam teks SQL — ia wajib lewat parameter binding")
	}

	numbers := bindNumbers(t, statement)
	require.Len(t, numbers, len(ids))

	previous := 0
	for _, number := range numbers {
		require.Greater(t, number, previous, "penanda wajib menaik sesuai urutan kemunculan")
		previous = number
	}
}

// TestPendingWithoutClaimsNeverRuns memastikan daftar kosong tidak menghasilkan `IN ()`.
//
// `IN ()` bukan SQL yang sah, dan halaman yang kebetulan tidak punya baris akan membuat
// seluruh layar gagal — bukan hanya penandanya yang hilang.
func TestPendingWithoutClaimsNeverRuns(t *testing.T) {
	statement, args := expandClaims(query("transfer_pending"), nil)

	require.Empty(t, args)
	require.Contains(t, statement, "IN ()",
		"bentuk tak sah ini memang dihasilkan — dan karena itu TransferRepo.PendingFor "+
			"wajib menjawab daftar kosong SEBELUM menyentuh basis data")
}
