package sqlstore

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// allQueryNames adalah SELURUH kueri modul ini, disebut lengkap.
//
// Ia ditulis tangan, bukan dibaca dari map `queries`: daftar yang dihitung sendiri oleh uji
// akan ikut menyusut bila sebuah kueri terhapus, dan uji yang menyusut bersama cacatnya tidak
// menjaga apa pun.
//
// CELAH YANG PERNAH ADA DI SINI. Sampai 2026-09-24 daftar ini memuat kueri BACA saja,
// sehingga uji urutan penanda bind — dua uji tepat di bawahnya — tidak pernah memeriksa satu
// pun pernyataan tulis. Penjagaan yang dibangun untuk menangkap cacat bind justru melewatkan
// pernyataan yang paling berbahaya bila bind-nya tertukar.
//
// Itu celah yang sama bentuknya dengan yang ditemukan pada branchFilteredQueries di hari yang
// sama, dan pelajarannya satu: daftar penjaga harus diperiksa setiap kali KATEGORI baru masuk,
// bukan hanya saat anggotanya bertambah.
var allQueryNames = []string{
	"list_not_answered", "list_answered",
	"count_answered", "count_not_answered",
	"detail_header", "detail_thread", "detail_attachments",
	"check_table", "check_attachment_table", "check_history_table",
	"branch_of_login",
	"reply_update", "reply_history_insert", "finish_update",
	"branch_options", "message_insert", "message_max_id", "message_history_insert",
}

func TestEveryLoadedQueryIsListedInAllQueryNames(t *testing.T) {
	// Penjagaan atas penjagaannya sendiri. Kueri baru yang lupa didaftarkan di atas akan lolos
	// dari SELURUH uji di berkas ini tanpa satu pun tanda — persis yang terjadi pada ketiga
	// pernyataan tulis sampai 2026-09-24.
	loaded := make([]string, 0, len(queries))
	for name := range queries {
		loaded = append(loaded, name)
	}

	require.ElementsMatch(t, loaded, allQueryNames,
		"setiap kueri yang termuat harus terdaftar di allQueryNames, dan sebaliknya")
}

func TestEveryNamedQueryIsLoadable(t *testing.T) {
	for _, name := range allQueryNames {
		require.NotEmptyf(t, getQuery(name), "kueri %q kosong", name)
	}
}

func TestBindMarkersAppearInAscendingOrder(t *testing.T) {
	// Oracle mengikat argumen menurut URUTAN KEMUNCULAN penanda di dalam teks kueri, BUKAN
	// menurut angka pada `:n`.
	//
	// `docs/catatan-pengembangan.md` §19.14 mencatat cacat nyata yang lahir dari melupakannya
	// pada modul Inbox Laporan Klaim: penyaring cabang menerima NULL sementara pencacah
	// menerima kode cabang — tanpa satu pun galat, karena setiap bind tetap terisi sesuatu.
	//
	// Uji ini memeriksa SETIAP kueri, bukan hanya yang sedang dicurigai.
	marker := regexp.MustCompile(`:(\d+)`)

	for _, name := range allQueryNames {
		found := marker.FindAllStringSubmatch(getQuery(name), -1)

		previous := 0
		for _, match := range found {
			value, err := strconv.Atoi(match[1])
			require.NoError(t, err)

			require.GreaterOrEqualf(t, value, previous,
				"kueri %q: penanda :%d muncul setelah :%d — urutan kemunculan harus menaik",
				name, value, previous)
			previous = value
		}
	}
}

func TestBindMarkersStartAtOneAndSkipNothing(t *testing.T) {
	// Penanda yang melompat (`:1`, `:3`) berarti satu argumen tidak pernah terpakai — dan
	// pemanggil yang tetap mengirimkannya akan menggeser SELURUH sisanya.
	marker := regexp.MustCompile(`:(\d+)`)

	for _, name := range allQueryNames {
		seen := map[int]bool{}
		for _, match := range marker.FindAllStringSubmatch(getQuery(name), -1) {
			value, _ := strconv.Atoi(match[1])
			seen[value] = true
		}
		if len(seen) == 0 {
			continue
		}

		numbers := make([]int, 0, len(seen))
		for value := range seen {
			numbers = append(numbers, value)
		}
		sort.Ints(numbers)

		require.Equalf(t, 1, numbers[0], "kueri %q tidak mulai dari :1", name)
		for i, value := range numbers {
			require.Equalf(t, i+1, value, "kueri %q melompati penanda bind", name)
		}
	}
}

func TestEveryQueryTouchingDataFiltersByBranch(t *testing.T) {
	// Kueri yang lupa menyaring cabang TIDAK menghasilkan satu pun galat — ia hanya
	// menampilkan percakapan cabang lain. Itu kelas cacat yang `R-20` catat, dan satu-satunya
	// yang dapat menangkapnya sebelum produksi adalah uji seperti ini.
	//
	// Daftarnya diambil dari branchFilteredQueries supaya penambahan kueri kelak ikut
	// terperiksa begitu namanya didaftarkan.
	for _, name := range branchFilteredQueries {
		text := getQuery(name)

		require.Containsf(t, text, "COMMUNICATE_TO",
			"kueri %q tidak menyaring kolom tujuan", name)
		require.Containsf(t, text, "COMMUNICATE_FROM",
			"kueri %q tidak menyaring kolom asal", name)

		// `OR`, bukan `AND`. Menukarnya mengosongkan seluruh layar, karena tidak ada
		// percakapan yang asal dan tujuannya sama.
		//
		// Awalan aliasnya dibiarkan bebas: kueri baca memakai alias `k.`, sementara kedua
		// pernyataan tulis TIDAK boleh beralias sama sekali (PostgreSQL menolak awalan alias
		// pada klausa SET). Yang diperiksa adalah operatornya, bukan ejaan aliasnya.
		require.Regexpf(t, `OR\s+(k\.)?COMMUNICATE_FROM`, text,
			"kueri %q menyambung kedua sisi batas cabang dengan operator yang salah", name)
	}
}

func TestBranchFilteredQueriesCoverEveryDataQuery(t *testing.T) {
	// Penjagaan atas penjagaannya sendiri: sebuah kueri baru yang menyentuh data tetapi lupa
	// didaftarkan di branchFilteredQueries akan lolos dari uji di atas tanpa ketahuan.
	dataQueries := []string{
		"list_not_answered", "list_answered",
		"count_answered", "count_not_answered",
		"detail_header", "detail_thread", "detail_attachments",
		"reply_update", "finish_update",
	}

	require.ElementsMatch(t, dataQueries, branchFilteredQueries)
}

func TestWriteStatementsCarryNoTableAlias(t *testing.T) {
	// Oracle mengizinkan `UPDATE tabel alias SET alias.kolom = …`; PostgreSQL TIDAK.
	//
	// Cacatnya tidak akan terlihat sampai cutover PostgreSQL (`D-24`), dan pada saat itu
	// memperbaikinya jauh lebih mahal daripada menahannya sekarang. Uji ini yang menahannya.
	for _, name := range writeQueries {
		text := getQuery(name)

		require.NotRegexpf(t, `(?i)UPDATE\s+\S+\s+\w+\s+SET`, text,
			"pernyataan %q memberi alias pada tabelnya; PostgreSQL menolaknya", name)
		require.NotContainsf(t, text, "SET k.",
			"pernyataan %q memberi awalan alias pada klausa SET", name)
	}
}

func TestEveryWriteStatementRefusesAClosedConversation(t *testing.T) {
	// Kedua pernyataan tulis WAJIB menyaring `CASEID`.
	//
	// Pada finish_update ia mencegah penutupan ganda; pada reply_update ia mencegah balasan
	// atas percakapan yang sudah ditutup — yang tanpa syarat ini akan "berhasil" lalu dijawab
	// layar dengan kalimat tentang perpindahan tab yang tidak terjadi.
	//
	// Keduanya diperiksa di satu tempat supaya pernyataan tulis yang ditambahkan kelak ikut
	// terperiksa begitu namanya didaftarkan di writeQueries.
	for _, name := range writeQueries {
		require.Containsf(t, getQuery(name), "CASEID =",
			"pernyataan %q tidak menyaring kanal percakapan", name)
	}
}

func TestBothListQueriesReturnTheSameAliasesInTheSameOrder(t *testing.T) {
	// Satu pemindai Go melayani KEDUA kueri daftar. Kolom yang tertukar di salah satunya
	// tidak menghasilkan galat apa pun — seluruh kolomnya bertipe teks, sehingga nilainya
	// hanya berpindah kolom di layar.
	for _, name := range listQueries {
		require.Equalf(t, listColumns, aliasesOf(getQuery(name)),
			"alias kueri %q tidak sesuai urutan pemindainya", name)
	}
}

func TestThreadQueryReturnsExactlyTheThreeColumnsTheSectionDraws(t *testing.T) {
	// `Section/BalasKomunikasiCabang-Section.xml` menggambar TIGA kolom: Tanggal, Pengirim,
	// Pesan. Tidak lebih.
	//
	// Versi sebelumnya mengembalikan tujuh, karena ia membaca tabel percakapan dan membawa
	// serta balasan beserta penjawabnya. Sejak utas dibaca dari tabel riwayat, balasan BUKAN
	// isian pada sebuah ucapan — ia ucapan tersendiri.
	require.Equal(t, threadColumns, aliasesOf(getQuery("detail_thread")))
	require.Len(t, threadColumns, 3)
	require.NotEqual(t, listColumns, threadColumns)
}

func TestTheThreadIsReadFromTheHistoryTableNotTheConversationTable(t *testing.T) {
	// Keterangan Work Owner 2026-09-24. Tabel percakapan menyimpan pesan dan balasan
	// TERAKHIR saja; tabel riwayat menyimpan setiap ucapan sebagai barisnya sendiri.
	//
	// Membacanya dari tabel percakapan menghasilkan utas yang SELALU berisi tepat satu baris,
	// betapapun panjang percakapannya — dan itu tidak terlihat sebagai galat.
	text := getQuery("detail_thread")

	require.Contains(t, text, "M_KOMUNIKASI_CABANG")
	require.Contains(t, text, "JOIN",
		"batas cabang hanya dapat ditegakkan lewat gabungan: tabel riwayat tidak memuat "+
			"kolom asal maupun tujuan")
}

func TestTheHeaderQueryExistsSoAnEmptyThreadIsNotMistakenForAMissingConversation(t *testing.T) {
	// Percakapan yang riwayatnya belum pernah ditulis punya kepala tanpa utas. Tanpa
	// pemeriksaan terpisah, ia akan dijawab "tidak ditemukan" — dan itu akan dilaporkan
	// sebagai kerusakan.
	require.Equal(t, headerColumns, aliasesOf(getQuery("detail_header")))
	require.Contains(t, getQuery("detail_header"), "M_KOMUNIKASI_PNC")
}

func TestAttachmentQueryReturnsItsOwnAliasSet(t *testing.T) {
	require.Equal(t, attachmentColumns, aliasesOf(getQuery("detail_attachments")))
}

func TestListQueriesDifferOnlyInTheReplyFilterAndTheOrdering(t *testing.T) {
	// Kedua tab dibedakan SATU penyaring. Bila keduanya menyaring hal yang sama, isi kedua
	// tab menjadi identik — dan tidak ada apa pun di layar yang menandainya.
	notAnswered := getQuery("list_not_answered")
	answered := getQuery("list_answered")

	require.Contains(t, notAnswered, "k.REPLYMESSAGE IS NULL")
	require.Contains(t, answered, "k.REPLYMESSAGE IS NOT NULL")

	// Urutannya BERLAWANAN, dan itu perilaku sistem lama apa adanya.
	require.Contains(t, notAnswered, "ORDER BY k.CREATEDDATE ASC")
	require.Contains(t, answered, "ORDER BY k.CREATEDATEREPLY DESC")
}

func TestCounterQueriesCheckTwoReplyColumnsWhileListQueriesCheckOne(t *testing.T) {
	// Selisih SATU KOLOM yang ada di sistem lama, dan yang paling mudah "dirapikan" oleh
	// pembaca berikutnya. Menyeragamkannya mengubah angka yang dilihat pengguna hari ini.
	for _, name := range countQueries {
		require.Containsf(t, getQuery(name), "k.REPLYFROM",
			"pencacah %q harus memeriksa kolom penjawab pula", name)
	}

	for _, name := range listQueries {
		require.NotContainsf(t, getQuery(name), "REPLYFROM IS",
			"kueri daftar %q TIDAK boleh memeriksa kolom penjawab", name)
	}
}

func TestCounterQueriesDoNotFilterSenderOrMessage(t *testing.T) {
	// Kedua kueri pencacah tidak memuat penyaring itu, padahal kedua kueri grid memilikinya.
	// Dibawa apa adanya (`P-5`).
	for _, name := range countQueries {
		text := getQuery(name)
		require.NotContainsf(t, text, "k.SENDER IS NOT NULL",
			"pencacah %q tidak boleh menyaring pengirim", name)
		require.NotContainsf(t, text, "k.MESSAGE IS NOT NULL",
			"pencacah %q tidak boleh menyaring pesan", name)
	}
}

func TestListQueriesFilterSenderAndMessage(t *testing.T) {
	for _, name := range listQueries {
		text := getQuery(name)
		require.Containsf(t, text, "k.SENDER IS NOT NULL", "kueri %q", name)
		require.Containsf(t, text, "k.MESSAGE IS NOT NULL", "kueri %q", name)
	}
}

func TestListAndCounterQueriesCompareTheChannelExactly(t *testing.T) {
	// `CABANG SELESAI` BERAWALAN `CABANG`. Penyaring berbasis `LIKE` akan meloloskan
	// percakapan yang justru sudah ditutup.
	for _, name := range append(append([]string{}, listQueries...), countQueries...) {
		text := getQuery(name)
		require.Containsf(t, text, "k.CASEID = :1", "kueri %q", name)
		require.NotContainsf(t, text, "LIKE", "kueri %q tidak boleh memakai LIKE", name)
	}
}

func TestDetailThreadDoesNotFilterTheChannel(t *testing.T) {
	// Percakapan yang sudah ditutup tetap dapat dibaca utasnya bila nomornya diketahui.
	// Menutupnya adalah aturan yang TIDAK dapat dibaca dari export mana pun — kueri pemasok
	// layar detail hilang (`R-16`), sehingga menambahkannya berarti mengarang.
	require.NotContains(t, getQuery("detail_thread"), "CASEID")
}

func TestNoQueryUsesForbiddenOracleOnlyConstructs(t *testing.T) {
	// `09-DATABASE-STRATEGY.md` §4 melarang seluruhnya demi portabilitas ke PostgreSQL.
	// `SELECT *` dilarang pula: kolom baru di basis data tidak boleh diam-diam mengubah
	// perilaku aplikasi.
	//
	// `SYSDATE` diperiksa sebagai kata utuh supaya nama kolom yang kebetulan memuatnya tidak
	// ikut tertangkap.
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bNVL\s*\(`),
		regexp.MustCompile(`(?i)\bSYSDATE\b`),
		regexp.MustCompile(`(?i)\bDECODE\s*\(`),
		regexp.MustCompile(`(?i)\bROWNUM\b`),
		regexp.MustCompile(`(?i)\bTO_CHAR\s*\(`),
		regexp.MustCompile(`(?i)\bTRUNC\s*\(`),
		regexp.MustCompile(`SELECT\s+\*`),
	}

	for _, name := range allQueryNames {
		text := getQuery(name)
		for _, pattern := range forbidden {
			require.Falsef(t, pattern.MatchString(text),
				"kueri %q memakai konstruksi terlarang %s", name, pattern)
		}
	}
}

func TestNoQueryConcatenatesValuesIntoItsText(t *testing.T) {
	// Sistem lama merangkai batas cabangnya sebagai teks lalu menyisipkannya lewat `{ASIS:}`
	// ke enam kueri. Yang direplikasi adalah perilaku bisnis, bukan polanya.
	for _, name := range allQueryNames {
		text := getQuery(name)
		require.NotContainsf(t, text, "{ASIS", "kueri %q", name)
		require.NotContainsf(t, text, "||", "kueri %q merangkai teks", name)
	}
}

func TestListQueriesPaginateInTheDatabaseAndCountInOnePass(t *testing.T) {
	for _, name := range listQueries {
		text := getQuery(name)
		require.Containsf(t, text, "OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY", "kueri %q", name)
		require.Containsf(t, text, "COUNT(*) OVER ()", "kueri %q", name)
	}
}

func TestListQueriesOrderByATieBreakerSoPagesDoNotShuffle(t *testing.T) {
	// Kueri lama mengurutkan hanya menurut tanggal. Dua baris bertanggal sama karena itu
	// dapat berpindah urutan di antara dua permintaan — dan pada daftar yang dipaginasi,
	// urutan yang tidak tetap membuat sebuah baris terlewat di halaman satu lalu muncul dua
	// kali di halaman dua.
	for _, name := range listQueries {
		require.Containsf(t, getQuery(name), "k.KOMUNIKASIID", "kueri %q", name)
	}
}

func TestAttachmentQueryKeepsInnerJoinsToTheDocumentMasters(t *testing.T) {
	// Lampiran yang jenisnya tidak ada di master TIDAK muncul sama sekali. Bentuk `INNER`
	// dipertahankan (`P-5`): `LEFT JOIN` akan MENAMBAH baris yang di Pega tidak pernah
	// terlihat.
	text := getQuery("detail_attachments")

	require.Contains(t, text, "INNER JOIN POOLDATA.V_LST_DOC_TYPE")
	require.Contains(t, text, "INNER JOIN POOLDATA.V_LST_DET_TYPE_DOC")
	require.NotContains(t, text, "LEFT JOIN")
}

func TestQueryNamesAndDomainConstantsStayInStep(t *testing.T) {
	// Penjagaan kecil yang menangkap satu kekeliruan besar: kanal percakapan dikirim sebagai
	// argumen dari kode, bukan ditulis di dalam SQL, sehingga konstanta domainnya harus
	// benar-benar dipakai.
	require.Equal(t, "CABANG", inboxkomunikasicabang.CaseOpen)
	require.Equal(t, "CABANG SELESAI", inboxkomunikasicabang.CaseClosed)

	for _, name := range listQueries {
		require.NotContainsf(t, getQuery(name), "'CABANG'",
			"kueri %q menuliskan kanal sebagai literal alih-alih parameter", name)
	}
}

// aliasesOf membaca alias kolom dari klausa SELECT sebuah kueri.
//
// Ia sengaja sederhana dan hanya mengenali bentuk `... AS ALIAS`, karena itulah satu-satunya
// bentuk yang dipakai berkas .sql modul ini — dan keseragaman itu sendiri patut dijaga.
func aliasesOf(text string) []string {
	pattern := regexp.MustCompile(`(?i)\sAS\s+([A-Z_][A-Z0-9_]*)`)

	selectPart := text
	if idx := strings.Index(strings.ToUpper(text), "\nFROM"); idx > 0 {
		selectPart = text[:idx]
	}

	found := pattern.FindAllStringSubmatch(selectPart, -1)
	aliases := make([]string, 0, len(found))
	for _, match := range found {
		aliases = append(aliases, strings.ToUpper(match[1]))
	}
	return aliases
}

// ── Pernyataan pembuatan percakapan baru ─────────────────────────────────────

func TestEveryInsertBindsItsValuesInsteadOfConcatenatingThem(t *testing.T) {
	// INSERT tidak punya klausa WHERE, sehingga tidak ada batas cabang maupun kanal yang
	// dapat menjaganya. Yang tersisa sebagai penjaga adalah parameter binding — dan
	// pemeriksaan ini yang memastikannya masih ada.
	for _, name := range insertQueries {
		text := getQuery(name)

		require.Containsf(t, text, ":1", "pernyataan %q tidak mengikat satu pun nilai", name)
		require.NotContainsf(t, text, "||",
			"pernyataan %q merangkai nilai ke dalam teksnya sendiri", name)
	}
}

func TestTheNewMessageInsertNamesEverySevenColumnItFills(t *testing.T) {
	// `InsertMessageCABANG_PNC` mengisi TUJUH kolom, dan tidak satu pun boleh hilang: yang
	// hilang akan diisi basis data dengan nilai bawaan, dan dua di antaranya — COMMUNICATE_TO
	// dan COMMUNICATE_FROM — menentukan siapa melihat percakapannya.
	text := getQuery("message_insert")

	for _, column := range []string{
		"CASEID", "SENDER", "MESSAGE", "SENDERNAME", "KOMUNIKASISTATUS",
		"COMMUNICATE_TO", "COMMUNICATE_FROM",
	} {
		require.Containsf(t, text, column, "kolom %q tidak diisi", column)
	}

	// Tujuh kolom, tujuh penanda bind.
	require.Contains(t, text, ":7")
	require.NotContains(t, text, ":8")
}

func TestTheNewMessageInsertLeavesTheCreatedDateToTheDatabase(t *testing.T) {
	// `InsertMessageCABANG_PNC` TIDAK menyebut CREATEDDATE, sehingga tanggalnya diisi basis
	// data. Itu direplikasi apa adanya: kolom itu DASAR PENGURUTAN tab "Belum Dijawab", dan
	// baris baru yang tanggalnya diisi aplikasi akan berperilaku berbeda dari seluruh baris
	// yang sudah ada.
	require.NotContains(t, getQuery("message_insert"), "CREATEDDATE")
}

func TestTheBranchListQueryRemovesDuplicates(t *testing.T) {
	// `V_D_SURVEYORS` adalah view SURVEYOR, bukan view cabang: satu cabang hampir pasti
	// muncul sekali per surveyor. Tanpa DISTINCT, pemilih tujuan menampilkan nama cabang yang
	// sama berulang-ulang.
	text := getQuery("branch_options")

	require.Contains(t, text, "DISTINCT")
	require.Contains(t, text, "V_D_SURVEYORS")
	require.Contains(t, text, "ORDER BY")
}

func TestTheBranchListQueryTakesNoParameters(t *testing.T) {
	// Ia TIDAK disaring menurut cabang pemanggil — yang dibatasi adalah percakapan, bukan
	// daftar cabang. Menyaringnya akan mengosongkan pemilihnya bagi setiap petugas cabang,
	// sehingga tidak seorang pun dapat mengirim pesan ke mana pun.
	require.NotContains(t, getQuery("branch_options"), ":1")
}

func TestTheMaxIdQueryReadsTheSameTableTheInsertWrote(t *testing.T) {
	// Nomornya ditemukan kembali lewat `MAX()` karena INSERT-nya tidak menyebut KOMUNIKASIID.
	// Bila keduanya membaca tabel yang berbeda, nomor yang dikembalikan milik percakapan
	// orang lain — dan riwayatnya tertaut ke tempat yang salah.
	require.Contains(t, getQuery("message_max_id"), "M_KOMUNIKASI_PNC")
	require.Contains(t, getQuery("message_insert"), "M_KOMUNIKASI_PNC")
	require.Contains(t, getQuery("message_max_id"), "MAX(KOMUNIKASIID)")
}
