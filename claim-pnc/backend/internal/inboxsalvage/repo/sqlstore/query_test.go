package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// Uji di berkas ini TIDAK menyentuh basis data. Yang diperiksa adalah kesesuaian antara
// teks SQL, daftar alias di query.go, dan pemindai di inboxsalvage.go — tiga tempat yang
// mudah bergeser satu sama lain tanpa ada yang gagal sampai kuerinya benar-benar dijalankan
// terhadap Oracle.

func TestEveryQueryNamedInTheCodeExists(t *testing.T) {
	names := []string{
		"list_claim", "list_claim_search",
		"list_claim_object", "list_claim_object_search",
		"list_claim_buyback", "list_claim_buyback_search",
		"list_salvage",
		"count_claim_status", "count_claim_outstanding", "count_claim_buyback",
		"count_salvage_status", "count_salvage_detail_unsold",
		"next_salvage_id", "insert_salvage", "update_salvage",
		"count_salvage_detail_for", "insert_salvage_detail",
		"check_salvage_columns",
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
func TestAliasOrderInSQLMatchesTheListsInGo(t *testing.T) {
	cases := []struct {
		queries []string
		aliases []string
	}{
		{[]string{"list_claim", "list_claim_search"}, claimColumns},
		{
			[]string{
				"list_claim_object", "list_claim_object_search",
				"list_claim_buyback", "list_claim_buyback_search",
			},
			claimObjectColumns,
		},
		{[]string{"list_salvage"}, salvageColumns},
	}

	for _, testCase := range cases {
		for _, name := range testCase.queries {
			require.Equal(t, testCase.aliases, aliasesOf(query(name)),
				"urutan alias kueri %q bergeser", name)
		}
	}
}

// Setiap kueri daftar WAJIB mengembalikan jumlah seluruh baris lewat `COUNT(*) OVER ()`.
//
// Tanpa itu, paginasi kehilangan totalnya dan layar menggambar "halaman 1 dari 1" pada
// daftar yang sebenarnya punya ribuan baris.
func TestEveryListQueryReturnsTheTotalRowCount(t *testing.T) {
	for _, name := range append(listQueries, "list_salvage") {
		require.Contains(t, query(name), "COUNT(*) OVER ()",
			"kueri %q tidak menghitung total barisnya", name)
		require.Contains(t, query(name), "TOTAL_ROWS")
	}
}

// Paginasi dipotong DI BASIS DATA, bukan setelah baris sampai ke aplikasi.
func TestEveryListQueryPagesInTheDatabase(t *testing.T) {
	for _, name := range append(listQueries, "list_salvage") {
		text := query(name)
		require.Contains(t, text, "OFFSET", "kueri %q tidak memotong halaman", name)
		require.Contains(t, text, "FETCH NEXT", "kueri %q tidak membatasi halaman", name)
	}
}

// Penyaring lini bisnis WAJIB ada di setiap kueri berbasis klaim.
//
// Kueri yang kehilangannya akan menampilkan klaim Personal Accident dan Travel — lini yang
// tidak mengenal salvage sama sekali — tanpa satu pun galat.
func TestClaimQueriesAlwaysFilterTheBusinessLines(t *testing.T) {
	for _, name := range businessFilterQueries {
		text := query(name)
		for _, panel := range inboxsalvage.GroupPanels {
			require.Contains(t, text, "'"+panel+"'",
				"kueri %q kehilangan Group Panel %s", name, panel)
		}
		for _, code := range inboxsalvage.ExcludedBizCode {
			require.Contains(t, text, "'"+code+"'",
				"kueri %q kehilangan pengecualian %s", name, code)
		}
	}
}

// TIDAK ADA nilai yang dirangkai ke dalam teks SQL.
//
// Sistem lama menyisipkan isi kotak pencarian langsung ke dalam kueri lewat pola
// `{ASIS:…}` di LIMA tempat. Uji ini menjaga polanya tidak pernah terbawa.
func TestNoQueryCarriesTheLegacyStringSplicingPattern(t *testing.T) {
	for name, text := range queries {
		require.NotContains(t, text, "{ASIS",
			"kueri %q membawa pola perangkaian SQL sistem lama", name)
		require.NotContains(t, text, "||'%'||",
			"kueri %q merangkai pola pencarian di dalam SQL", name)
	}
}

// Setiap pencarian memakai `ESCAPE`, supaya `%` yang DIKETIK pengguna tidak berlaku sebagai
// wildcard — terutama pada ketiga daftar yang pencariannya cocok persis.
func TestEverySearchQueryDeclaresAnEscapeCharacter(t *testing.T) {
	searchQueries := []string{
		"list_claim_search", "list_claim_object_search",
		"list_claim_buyback_search", "list_salvage",
	}

	for _, name := range searchQueries {
		text := query(name)
		likeCount := strings.Count(text, "LIKE")
		escapeCount := strings.Count(text, "ESCAPE")
		require.Equal(t, likeCount, escapeCount,
			"kueri %q punya %d LIKE tetapi %d ESCAPE", name, likeCount, escapeCount)
	}
}

// Kedua cabang pencarian keluarga C digabung dengan ATAU, bukan DAN.
//
// Penyaring tab Checker di sistem lama berbunyi `and (a.noklaim = '…' or a.pic = '…')` —
// satu kotak mencari dua kolom. Menggantinya dengan DAN akan membuat pencarian nomor klaim
// hanya menemukan baris yang PIC-nya kebetulan sama dengan nomor klaim itu, yakni tidak
// pernah.
func TestTheTwoSearchBranchesAreJoinedWithOr(t *testing.T) {
	text := query("list_salvage")
	require.Regexp(t, regexp.MustCompile(`\(\s*UPPER\(a\.NOKLAIM\)[\s\S]*?OR[\s\S]*?a\.PIC`),
		text, "kedua cabang pencarian tidak digabung dengan OR")
}

// Kolom bertipe NUMBER yang masuk ke `COALESCE` bersama sentinel teks WAJIB dibungkus
// `TO_CHAR`.
//
// Tanpa itu Oracle menjawab `ORA-00932: inconsistent datatypes: expected NUMBER got CHAR`,
// dan galat itu menjatuhkan KETUJUH daftar keluarga C sekaligus — sementara pencacah tetap
// berjalan, sehingga layar tampak separuh hidup dan sebabnya tidak terbaca dari gejalanya.
//
// Uji ini menjaga bentuknya, bukan menebak tipenya: `TO_CHAR` aman untuk kolom teks maupun
// angka, sehingga ia benar apa pun tipe sebenarnya — yang memang belum diketahui, karena
// DDL `PNC_SALVAGE` belum pernah diterima (`R-08`).
func TestNumericColumnsAreMadeTypeSafeBeforeCoalesceWithText(t *testing.T) {
	text := query("list_salvage")

	require.Contains(t, text, "COALESCE(TO_CHAR(a.STSTRANSFER), '~')",
		"STSTRANSFER bertipe NUMBER; COALESCE dengan sentinel teks menuntut TO_CHAR")

	require.NotContains(t, text, "COALESCE(a.STSTRANSFER",
		"bentuk tanpa TO_CHAR menjatuhkan ketujuh daftar keluarga C")
}

// Kueri pembaruan TIDAK BOLEH menulis `IDSALVAGE`.
//
// Inilah butir 12 daftar perbaikan `P-5` (`D-49` #9): procedure lama menimpa kunci barisnya
// sendiri dengan nilai kosong pada cabang pembaruan, sehingga barisnya hilang dari seluruh
// daftar. Uji ini menjaga perbaikannya tetap berlaku.
func TestUpdateNeverWritesTheSalvageIDColumn(t *testing.T) {
	text := query("update_salvage")

	setClause := text
	if index := strings.Index(text, "WHERE"); index >= 0 {
		setClause = text[:index]
	}

	require.NotContains(t, setClause, "IDSALVAGE =",
		"pembaruan tidak boleh menimpa kunci barisnya sendiri")
	require.Contains(t, text, "WHERE IDSALVAGE = :10",
		"kunci pencarinya tetap ID salvage")
}

// Bind pada pernyataan simpan WAJIB sejumlah argumen yang disusun insertArgs. Selisih satu
// saja menghasilkan galat bind yang menyebut nomor, bukan menyebut isian mana yang hilang.
func TestInsertArgumentCountMatchesTheBindsInBothStatements(t *testing.T) {
	form := inboxsalvage.Form{Caller: inboxsalvage.Caller{Login: "SITIRAHAYU"}}
	args := insertArgs(form, "101")

	require.Len(t, args, 22, "insertArgs menyusun 22 bind")

	for _, name := range []string{"insert_salvage", "update_salvage"} {
		require.Equal(t, 22, highestBind(query(name)),
			"kueri %q memakai nomor bind tertinggi yang berbeda", name)
	}
}

// Nomor bind harus BERURUTAN tanpa lompatan. Lompatan berarti satu argumen tidak pernah
// dipakai, dan argumen yang tidak dipakai biasanya berarti isian yang tidak tersimpan.
func TestBindNumbersAreContiguousInEveryStatement(t *testing.T) {
	for name, text := range queries {
		highest := highestBind(text)
		if highest == 0 {
			continue
		}

		// `update_salvage` memang melewati satu nomor di klausa SET — `:10` dipakai di
		// WHERE — sehingga seluruh nomor tetap terpakai, hanya letaknya berbeda.
		for number := 1; number <= highest; number++ {
			require.Contains(t, text, ":"+strconv.Itoa(number),
				"kueri %q melewatkan bind :%d", name, number)
		}
	}
}

// Awalan kunci Pega DIIKAT, bukan ditulis di dalam SQL — supaya terbaca sebagai nilai yang
// kelak berubah (`D-22`, `D-71`), bukan sebagai bagian kuerinya.
func TestThePegaWorkKeyPrefixIsBoundNotWrittenIntoTheSQL(t *testing.T) {
	require.NotContains(t, query("list_salvage"), "ASM-FW-GCNMFW-WORK",
		"awalan kunci Pega tidak boleh tertanam di dalam teks SQL")

	// Dan SPASI di ujungnya adalah bagian nilainya. Tanpa spasi itu gabungan tidak pernah
	// cocok, dan daftarnya kosong tanpa satu pun galat.
	require.True(t, strings.HasSuffix(PegaWorkKeyPrefix, " "),
		"awalan kunci Pega wajib berakhir dengan spasi")
}

// Tidak satu pun kueri memakai konstruksi khas Oracle yang `D-20` larang.
func TestNoQueryUsesTheForbiddenOraclePatterns(t *testing.T) {
	forbidden := []string{"NVL(", "ROWNUM", "SYSDATE", "DECODE(", "SELECT *"}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContains(t, upper, pattern,
				"kueri %q memakai pola terlarang %q", name, pattern)
		}
	}
}

// aliasesOf membaca urutan alias pada klausa SELECT TERLUAR sebuah kueri.
//
// # Kenapa kedalaman kurung dihitung
//
// Karena dua kueri di berkas ini memuat sub-kueri berkolom sendiri — nama objek pertama
// klaim, dan kedua agregat detail salvage. Berhenti pada `FROM` yang PERTAMA ditemukan akan
// berhenti di dalam sub-kueri itu, sehingga separuh alias terluar tidak pernah terbaca dan
// ujinya gagal karena alasan yang bukan yang sedang diujinya.
//
// Yang dicari adalah `FROM` pada kedalaman NOL, dan alias yang dikumpulkan hanyalah yang
// berada pada kedalaman itu pula — alias di dalam sub-kueri bukan kolom hasil.
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

// Panel Detail Salvage memakai dua kueri yang dijalankan berurutan atas satu ID.
func TestDetailQueriesExist(t *testing.T) {
	for _, name := range []string{"detail_header", "detail_items"} {
		require.NotPanics(t, func() { query(name) }, "kueri %q tidak ada", name)
		require.NotEmpty(t, strings.TrimSpace(query(name)))
	}
}

// Alias GANDA pada kueri kepala panel dipisah, bukan dibawa apa adanya.
//
// `RDB List/GcnmSetSalvageData_SQL-SQL.xml` mengaliaskan DUA kolom yang berbeda menjadi
// `"AreaClaimId"` — jumlah barang, dan penanda pengajuan sebelum 17 Juli 2023. Oracle
// menerimanya; yang membaca hasilnya tidak, karena satu nama hanya dapat menunjuk satu
// kolom dan yang terbaca adalah yang kebetulan terakhir.
//
// Ini kelas cacat yang sama dengan alias ganda `UserName` di §48.4, dan seperti di sana ia
// DIPERBAIKI, bukan direplikasi: keduanya digambar di layar sebagai isian yang berbeda,
// sehingga membawanya apa adanya berarti salah satunya pasti salah.
//
// Uji ini gagal bila seseorang menyalin ulang alias Pega ke dalam kueri kita.
func TestDetailHeaderHasNoDuplicateAlias(t *testing.T) {
	seen := map[string]int{}
	for _, alias := range aliasesOf(query("detail_header")) {
		seen[strings.ToUpper(alias)]++
	}

	for alias, count := range seen {
		require.Equalf(t, 1, count,
			"alias %q muncul %d kali pada detail_header — satu nama hanya dapat "+
				"menunjuk satu kolom", alias, count)
	}

	// Kedua penggantinya memang ada, sehingga uji di atas tidak dapat lulus hanya dengan
	// membuang salah satu kolomnya.
	text := strings.ToUpper(query("detail_header"))
	require.Contains(t, text, "AS QUANTITY")
	require.Contains(t, text, "AS LEGACY_FLAG")
	require.NotContains(t, text, "AREACLAIMID")
}

// Kedua kueri panel menyaring dengan parameter, bukan dengan nilai yang dirangkai.
func TestDetailQueriesBindTheirParameters(t *testing.T) {
	for _, name := range []string{"detail_header", "detail_items"} {
		text := query(name)
		require.Contains(t, text, ":1", "kueri %q tidak mengikat parameter", name)
		require.Contains(t, text, ":2", "kueri %q tidak mengikat parameter", name)
	}
}

// Panel rincian yang dibuka dari baris KLAIM memakai dua kueri tambahan.
func TestClaimKeyedDetailQueriesExist(t *testing.T) {
	for _, name := range []string{"claim_header", "latest_salvage_of_claim"} {
		require.NotPanics(t, func() { query(name) }, "kueri %q tidak ada", name)

		text := query(name)
		require.NotEmpty(t, strings.TrimSpace(text))
		require.Contains(t, text, ":1", "kueri %q tidak mengikat parameter", name)
	}
}

// Pencarian pengajuan terakhir memakai MAX, bukan urutan lalu ambil satu.
//
// Bedanya nyata: `IDSALVAGE` numerik, dan `ORDER BY` atas kolom yang terbaca sebagai teks
// akan menempatkan 9 di atas 10. Kueri lama pun memakai `max(IDSALVAGE)`
// (`SetDataDetailSalvage_act` langkah 18), dan uji ini menjaganya tidak berubah menjadi
// pengurutan.
func TestLatestSalvageOfClaimUsesMaxNotOrdering(t *testing.T) {
	text := strings.ToUpper(query("latest_salvage_of_claim"))

	require.Contains(t, text, "MAX(")
	require.NotContains(t, text, "ORDER BY")
	require.Contains(t, text, "DETAIL_PNC_SALVAGE")
}

// Kepala panel klaim TIDAK membawa penyaring populasi daftar Salvage Outstanding.
//
// `GcnmSalvageData_OS_SQL` menyaring STATUSWORK, GROUPPANEL, dan BUSINESSCODE untuk
// menyusun populasi SATU daftar. Panel rincian dibuka dari KEENAM daftar berbasis klaim,
// sehingga membawa penyaring itu akan membuat baris yang tampil di daftar lain menjawab
// "tidak ditemukan" — padahal barisnya baru saja digambar di layar yang sama.
//
// Selisih ini disengaja, dan uji ini yang menjaganya tidak "diperbaiki" kembali.
func TestClaimHeaderDoesNotCarryTheOutstandingPopulationFilter(t *testing.T) {
	text := strings.ToUpper(query("claim_header"))

	require.NotContains(t, text, "STATUSWORK")
	require.NotContains(t, text, "GROUPPANEL")
	require.NotContains(t, text, "BUSINESSCODE")
	require.Contains(t, text, "T_CLAIM_PNC")
}

// Grid "Detail History Salvage" punya kueri tersendiri, terikat parameter.
func TestSalvageHistoryQueryExistsAndBindsItsParameter(t *testing.T) {
	require.NotPanics(t, func() { query("salvage_history_of_claim") })

	text := query("salvage_history_of_claim")
	require.Contains(t, text, ":1")
	require.Contains(t, strings.ToUpper(text), "PNC_SALVAGE")
	require.Contains(t, strings.ToUpper(text), "ORDER BY")
}

// Penerjemahan `STSTRANSFER` menjadi kalimat TIDAK dilakukan di dalam SQL.
//
// Kueri lama menempuh `CASE WHEN` yang menghasilkan "Sudah Aksep Checker", "Salvage
// Waive", dan seterusnya. Pemetaan itu dipindahkan ke Go, tempat ia dapat diuji tanpa
// basis data — dan tempat pertentangannya dengan dua pemetaan lain atas kolom yang sama
// tercatat.
//
// Uji ini gagal bila seseorang menyalin kembali `CASE WHEN`-nya ke dalam kueri.
func TestHistoryQueryDoesNotTranslateStatusIntoSentences(t *testing.T) {
	text := strings.ToUpper(query("salvage_history_of_claim"))

	for _, sentence := range []string{
		"SUDAH AKSEP", "SUDAH AKSEPTASI", "CREATE AJD",
		"DITERIMA DI KOMITE", "TOLAK DI KOMITE", "SALVAGE WAIVE", "OTHER",
	} {
		require.NotContainsf(t, text, sentence,
			"kalimat %q harus dipetakan di Go, bukan di dalam SQL", sentence)
	}

	// Yang dikirim adalah kodenya, ditambah penanda terisi-tidaknya nomor akseptasi —
	// karena kode 2 bercabang menurut kolom itu.
	require.Contains(t, text, "AS TRANSFER_STATUS")
	require.Contains(t, text, "AS HAS_ACCEPTANCE_NO")
}

// Kedua kueri pilihan form "Menambahkan Data Salvage" ada dan terikat parameternya.
//
// Keduanya salinan langkah 2 dan 3 `Activity/GetDataSalavageCovCurObj_act-Act.xml`.
func TestClaimChoiceQueriesExistAndBindTheirParameters(t *testing.T) {
	for _, name := range []string{"claim_objects", "claim_coverages"} {
		require.NotPanicsf(t, func() { query(name) }, "kueri %q hilang", name)

		text := query(name)
		require.Containsf(t, text, ":1", "kueri %q tidak mengikat awalan kunci Pega", name)
		require.Containsf(t, text, ":2", "kueri %q tidak mengikat nomor klaim", name)
	}

	require.Contains(t, strings.ToUpper(query("claim_objects")), "T_CLAIM_OBJECTLIST")
	require.Contains(t, strings.ToUpper(query("claim_coverages")), "T_CLAIM_OBJECTCOVERAGE")
}

// Pilihan "Nama Coverage" memakai `COVERAGEID`, BUKAN `OBJECTCOVERAGEID`.
//
// # Kenapa ini perlu dijaga uji
//
// Karena kekeliruan ke arah sebaliknya sudah pernah terjadi, dan alasannya masuk akal:
// tujuh kueri PLA/DLA memperlakukan "IDCoverage" sebagai `OBJECTCOVERAGEID`
// (`GetDataDLA-SQL.xml:49`, `GetDataPLA-SQL.xml:108`, `GetDataPreDLA-SQL.xml:47`).
// Siapa pun yang membaca kueri-kueri itu lebih dulu akan menyimpulkan hal yang sama.
//
// Yang berlaku di layar ini adalah kuerinya sendiri:
// `RDB List/GetDataSalvageObjectCoverageForOs-SQL.xml` menyebut `COVERAGEID` apa adanya.
//
// Uji ini gagal bila seseorang "memperbaikinya" kembali.
func TestCoverageChoiceCarriesTheCoverageTypeCode(t *testing.T) {
	text := strings.ToUpper(query("claim_coverages"))

	require.Contains(t, text, "COVERAGEID",
		"pilihan coverage harus membawa COVERAGEID, kolom yang dipakai kueri layar ini")
	require.NotContains(t, text, "OBJECTCOVERAGEID",
		"OBJECTCOVERAGEID adalah kolom yang dipakai layar PLA/DLA, bukan layar ini")
}

// Pilihan coverage TIDAK terikat objek, sama seperti di layar lama.
//
// Kueri aslinya tidak mengambil `OBJECTID` sama sekali â€” daftar coverage ditarik per
// KLAIM. Mempersempitnya menurut objek terdengar masuk akal dan pernah dicoba, tetapi itu
// menyembunyikan pilihan yang di Pega tersedia (`P-5`).
func TestCoverageChoiceIsNotScopedToAnObject(t *testing.T) {
	text := strings.ToUpper(query("claim_coverages"))

	require.NotContains(t, text, "OBJECTID",
		"kueri coverage layar lama tidak mengambil OBJECTID; daftarnya per klaim")
}

// Kedua kueri pilihan TIDAK menyentuh kolom yang lahir dari migrasi.
//
// # Kenapa ini perlu dijaga uji
//
// Karena penyaring `DIHAPUS_PADA IS NULL` sempat dipasang di sini — atas dasar `D-66`,
// yang memang menuntut setiap pembaca membuang baris bertanda terhapus — dan itu MEMUTUS
// layar. Kolom itu bukan bawaan kedua tabel warisan: ia ditambahkan migrasi
// `0008_klaim_anak_dan_spreading`, sehingga basis data portal yang belum menjalankannya
// menolak kuerinya dengan `ORA-00904`, dan yang gagal bukan hanya daftar pilihan
// melainkan seluruh permintaan rincian klaim.
//
// Kedua kueri ini karena itu dibatasi pada kolom yang sudah ada di tabel warisan sejak
// semula — kolom yang sama yang dibaca kueri Pega aslinya.
func TestClaimChoiceQueriesTouchOnlyLegacyColumns(t *testing.T) {
	for _, name := range []string{"claim_objects", "claim_coverages"} {
		require.NotContainsf(t, strings.ToUpper(query(name)), "DIHAPUS_PADA",
			"kueri %q tidak boleh bergantung pada kolom hasil migrasi 0008 — "+
				"portal yang belum bermigrasi akan menolaknya dengan ORA-00904", name)
	}
}

// Kedua kueri pilihan menyatakan urutannya.
//
// Kueri aslinya tidak, sehingga urutan daftar pilihan tidak ditentukan dan dapat berbeda
// pada setiap pembukaan. Urutannya menurut NAMA â€” satu-satunya isian yang dibaca pengguna
// pada autocomplete.
func TestClaimChoiceQueriesOrderByTheNameThatIsRead(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("claim_objects")), "ORDER BY O.OBJECTNAME")
	require.Contains(t, strings.ToUpper(query("claim_coverages")), "ORDER BY C.COVERAGENAME")
}

// ============================================================================
// UNGGAHAN DOKUMEN
// ============================================================================

// Keenam kueri rantai unggahan ada, dan masing-masing mengikat parameternya.
//
// Rantai ini dipanggil berurutan di dalam SATU transaksi; satu nama yang salah ketik
// baru terlihat ketika dokumen nasabah pertama diunggah di produksi.
func TestAttachmentChainQueriesExistAndBindTheirParameters(t *testing.T) {
	for _, name := range []string{
		"attachment_name_prefix",
		"attachment_sequence_of_claim",
		"attachment_content",
		"attachment_meta",
		"attachment_history",
		"attachment_by_image",
		"salvage_document_link",
	} {
		require.NotPanicsf(t, func() { query(name) }, "kueri %q tidak ada", name)
		require.Containsf(t, query(name), ":1", "kueri %q tidak mengikat parameter", name)
	}
}

// Penyimpan ISI menerima satu bind LEBIH BANYAK daripada penyimpan KETERANGAN.
//
// Selisih satu itu adalah isi berkasnya, dan ia satu-satunya pembeda kedua procedure.
// Memanggil salah satu dengan jumlah argumen milik yang lain ditolak Oracle dengan
// PLS-00306 — galat yang baru muncul saat kuerinya benar-benar dijalankan.
func TestContentStoreTakesExactlyOneMoreBindThanTheMetadataStore(t *testing.T) {
	require.Equal(t, 12, highestBind(query("attachment_content")))
	require.Equal(t, 11, highestBind(query("attachment_meta")))
}

// `IMAGEID` tidak dibangkitkan di dalam SQL.
//
// `GenerateimageID` menyusunnya dengan `STANDARD_HASH` atas `SYSTIMESTAMP`; keduanya khas
// Oracle dan `D-20` melarangnya. Pembangkitannya pindah ke `inboxsalvage.NewImageID`, dan
// uji ini menjaganya supaya tidak diam-diam kembali ke SQL.
func TestNoAttachmentQueryGeneratesTheImageKeyInOracle(t *testing.T) {
	for name, text := range queries {
		upper := strings.ToUpper(text)
		require.NotContainsf(t, upper, "STANDARD_HASH",
			"kueri %q membangkitkan kunci di Oracle; itu milik Go", name)
		require.NotContainsf(t, upper, "SYSTIMESTAMP",
			"kueri %q memakai SYSTIMESTAMP; `D-20` melarangnya", name)
	}
}

// Penaut salvage memakai DATAID yang dikembalikan `attachment_by_image`.
//
// Kedua tabel lampiran membangkitkan DATAID sendiri-sendiri dan keduanya BERBEDA. Yang
// menautkan keduanya `IMAGEID`. Penaut salvage yang memakai DATAID milik tabel isi akan
// menunjuk baris yang tidak ada di tabel keterangan — dan tidak ada galat yang muncul,
// hanya dokumen yang tidak dapat dibuka dari layar salvage.
func TestTheImageLookupIsWhatBridgesTheTwoAttachmentTables(t *testing.T) {
	upper := strings.ToUpper(query("attachment_by_image"))

	require.Contains(t, upper, "DATA_ATTACHFILE")
	require.Contains(t, upper, "IMAGEID = :1")
	require.NotContains(t, upper, "TEMP_DATA_ATTACHFILE",
		"yang dibaca tabel KETERANGAN, bukan tabel isi")
}

// Nama tabel yang SALAH KETIK dibawa apa adanya.
//
// `SALAVAGEDOCUMENT` dan kolom `CATRGORY` memang bernama begitu di basis data. Membetulkan
// ejaannya di sini berarti menunjuk objek yang tidak ada; membetulkannya di basis data
// menempuh `D-63`.
func TestLegacyMisspellingsAreCarriedUnchanged(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("attachment_sequence_of_claim")),
		"SALAVAGEDOCUMENT")
	require.Contains(t, strings.ToUpper(query("attachment_history")), "CATRGORY")
}

// Penanda salvage daftar Outstanding di SQL WAJIB sama dengan yang dipakai pencacahnya.
//
// # Kenapa penjaganya perlu ada
//
// Keduanya ditulis di tempat berbeda dan dengan bentuk berbeda: daftar menuliskannya
// sebagai nilai TETAP di dalam teks SQL, pencacah mengikatnya dari
// `inboxsalvage.OutstandingSalvageStatuses`. Dua sumber kebenaran untuk satu aturan — dan
// persis kelas cacat yang menghabiskan 2026-10-08: angka ringkasan dan jumlah baris
// daftarnya berasal dari predikat yang berbeda, tanpa satu pun uji yang menolaknya.
//
// Nilainya tetap ditulis TETAP di SQL, dengan alasan yang sama seperti `GROUPPANEL` dan
// `BUSINESSCODE`: ia domain tertutup milik aturan bisnis, bukan masukan pengguna. Yang
// ditambahkan di sini bukan bind, melainkan penjaga yang menolak keduanya bergeser.
func TestOutstandingListFiltersTheSameSalvageMarksAsItsCounter(t *testing.T) {
	for _, name := range []string{"list_claim", "list_claim_search"} {
		text := query(name)

		require.Contains(t, text, "c.STSSALVAGE IN (",
			"kueri %q tidak lagi menyaring STSSALVAGE sama sekali", name)

		for _, status := range inboxsalvage.OutstandingSalvageStatuses {
			require.Contains(t, text, "'"+status+"'",
				"kueri %q tidak memuat penanda %q yang dipakai pencacahnya", name, status)
		}
	}
}
