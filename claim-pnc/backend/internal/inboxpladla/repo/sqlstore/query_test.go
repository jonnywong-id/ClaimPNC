package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
)

// Uji di berkas ini TIDAK menyentuh basis data. Yang diperiksa adalah kesesuaian antara
// teks SQL, daftar alias di query.go, dan pemindai di inboxpladla.go.

func TestEveryQueryNamedInTheCodeExists(t *testing.T) {
	names := []string{"reinsurer_codes", "detail_reply"}
	names = append(names, listQueries...)
	names = append(names, countQueries...)
	names = append(names, detailQueries...)

	for _, name := range names {
		require.NotPanics(t, func() { query(name) }, "kueri %q tidak ada", name)
		require.NotEmpty(t, strings.TrimSpace(query(name)))
	}
}

// Penanda bind WAJIB muncul dalam urutan MENAIK.
//
// # Ini uji yang paling mudah diremehkan, dan ia sudah pernah menggigit
//
// Oracle mengikat argumen menurut urutan KEMUNCULAN penanda di dalam teks kueri, BUKAN
// menurut angka pada `:n`. Penomoran `:1 … :7` yang tersebar tidak berurutan terbaca benar
// oleh manusia dan SALAH oleh Oracle.
//
// Kegagalannya tidak menghasilkan galat sama sekali — setiap bind tetap terisi sesuatu.
// Pada modul ini akibatnya: kolom "PLA No" selalu kosong, dan pencarian tidak pernah
// menemukan apa pun. Cacat yang sama pernah lolos di modul lain dan baru ketahuan setelah
// Oracle hidup (`catatan-pengembangan.md` §19.14).
//
// Ketiga kueri daftar di modul ini MEMANG rusak begitu sampai 2026-09-28, dan modul ini
// belum pernah dijalankan terhadap Oracle — sehingga tidak ada yang menyadarinya.
func TestBindMarkersAppearInAscendingOrder(t *testing.T) {
	pattern := regexp.MustCompile(`:(\d+)`)

	for name, text := range queries {
		seen := []int{}
		for _, match := range pattern.FindAllStringSubmatch(text, -1) {
			number, err := strconv.Atoi(match[1])
			require.NoError(t, err)

			if len(seen) > 0 && seen[len(seen)-1] == number {
				// Penanda yang sama berturut-turut tidak mungkin: satu penanda yang
				// dirujuk dua kali membuat jumlah kemunculan tidak lagi sama dengan
				// jumlah argumen. Ia tetap dicatat supaya urutannya diperiksa.
				continue
			}
			seen = append(seen, number)
		}

		for index := 1; index < len(seen); index++ {
			require.Equal(t, seen[index-1]+1, seen[index],
				"kueri %q: penanda bind muncul tidak berurutan (%v) — "+
					"Oracle mengikat menurut urutan KEMUNCULAN, bukan menurut nomor",
				name, seen)
		}
	}
}

// Komentar TIDAK ikut dikirim ke basis data.
func TestCommentsAreStrippedFromTheStatementsSent(t *testing.T) {
	for name, text := range queries {
		for _, line := range strings.Split(text, "\n") {
			require.False(t, strings.HasPrefix(strings.TrimSpace(line), "--"),
				"kueri %q masih membawa baris komentar", name)
		}
	}
}

// Urutan alias di SQL WAJIB sama dengan daftar di query.go.
//
// Ketiga kueri daftar mengembalikan kolom yang sama persis, sehingga satu kolom yang
// tergeser pada SALAH SATUNYA tidak akan terlihat dari membandingkan hasilnya dengan yang
// lain.
func TestAliasOrderInSQLMatchesTheListsInGo(t *testing.T) {
	for _, name := range listQueries {
		require.Equal(t, listColumns, aliasesOf(query(name)),
			"urutan alias kueri daftar %q bergeser", name)
	}

	for _, name := range countQueries {
		require.Equal(t, countColumns, aliasesOf(query(name)),
			"urutan alias kueri ringkas %q bergeser", name)
	}
}

// SETIAP kueri WAJIB menyaring menurut reasuradur pemanggil.
//
// Ini uji terpenting di berkas ini. Kueri yang kehilangan rantai itu akan menampilkan data
// satu mitra kepada mitra lain — dan di layar yang memang dibaca pihak luar, itu bukan
// cacat tampilan melainkan kebocoran yang tidak menghasilkan satu pun galat.
func TestEveryQueryIsScopedToTheCallersReinsurerCode(t *testing.T) {
	for _, name := range reinsurerScopedQueries {
		text := strings.ToUpper(query(name))

		// Cara menyaringnya BERBEDA menurut apa yang dibaca kuerinya, dan perbedaannya
		// disengaja:
		//
		//	pemberitahuan  kode reasuradur dari `T_REINSURER`
		//	komunikasi     login pada salah satu sisi percakapan
		//	dokumen        `T_DOC_REAS.LOGIN`, ditambah rantai pemberitahuannya
		//
		// Yang diuji karena itu bukan "memuat T_REINSURER" melainkan "memuat
		// sekurang-kurangnya satu batas yang berangkat dari pemanggil".
		viaReinsurer := strings.Contains(text, "POOLDATA.T_REINSURER")
		viaConversation := strings.Contains(text, "COMMUNICATE_TO") ||
			strings.Contains(text, "K.SENDER") ||
			strings.Contains(text, "(SENDER)")
		viaDocumentOwner := strings.Contains(text, "R.LOGIN")

		require.True(t, viaReinsurer || viaConversation || viaDocumentOwner,
			"kueri %q tidak menyaring menurut pemanggil sama sekali", name)

		// Kueri yang menyaring lewat `T_REINSURER` WAJIB mencocokkan kolom LOGIN-nya.
		// Yang menyaring lewat percakapan tidak punya kolom itu — identitas pemanggil di
		// sana berada pada `COMMUNICATE_TO` atau `SENDER`.
		if viaReinsurer {
			require.Contains(t, text, "LOGIN",
				"kueri %q membaca T_REINSURER tanpa mencocokkan login", name)
		}
	}
}

// Setiap kueri RINCIAN WAJIB memagari kunci klaimnya sekaligus pemanggilnya.
//
// Kunci klaim saja tidak cukup: bentuknya `ASM-FW-GCNMFW-WORK PNC-xxxx` — pola yang dapat
// ditebak — sehingga kueri yang hanya menyaring kunci akan menyerahkan nilai uang,
// dokumen, dan isi percakapan milik mitra lain kepada siapa pun yang menebaknya.
func TestEveryDetailQueryIsScopedToTheCaller(t *testing.T) {
	for _, name := range append(append([]string{}, detailQueries...), "detail_reply") {
		text := strings.ToUpper(query(name))

		// Kunci klaim tersimpan di kolom yang BERBEDA menurut tabelnya:
		//
		//	T_CLAIM_PNC · T_PLALIST · T_DLALIST · T_DOC_REAS   CLAIMID
		//	M_KOMUNIKASI_PNC                                    CASEID
		//
		// Kolom bernama `CASEID` yang berisi kunci klaim adalah perangkap penamaan Pega
		// yang sudah tercatat: di modul `inboxkomunikasicabang` kolom yang SAMA berisi
		// nama kanal. Uji ini menerima keduanya, bukan menuntut satu nama.
		require.True(t,
			strings.Contains(text, "CLAIMID") || strings.Contains(text, "CASEID"),
			"kueri rincian %q tidak memagari kunci klaim", name)

		// Identitas pemanggil pun tersimpan di kolom yang berbeda: `LOGIN` pada master
		// reasuransi dan `T_DOC_REAS`, `COMMUNICATE_TO`/`SENDER` pada percakapan.
		require.True(t,
			strings.Contains(text, "LOGIN") ||
				strings.Contains(text, "COMMUNICATE_TO") ||
				strings.Contains(text, "SENDER"),
			"kueri rincian %q tidak memagari pemanggil", name)
	}
}

// Kedua kueri pemberitahuan pada layar rincian WAJIB menuntut dokumennya SUDAH terkirim.
//
// Ia selisih terhadap Pega yang disengaja: grid Pega dimuat dari objek kerja klaim dan
// memuat seluruh baris apa pun keadaannya. Dokumen yang belum dikirim belum menjadi milik
// penerimanya.
func TestDetailAdviceGridsOnlyShowSentAdvices(t *testing.T) {
	for _, name := range []string{"detail_advices_pla", "detail_advices_dla"} {
		require.Contains(t, strings.ToUpper(query(name)), "ISKIRIM = '1'",
			"kueri %q menampilkan pemberitahuan yang belum terkirim", name)
	}
}

// Pernyataan balasan WAJIB memagari ketiganya sekaligus.
//
// Yang ketiga paling penting: `REPLYMESSAGE IS NULL` menutup balasan kedua yang akan
// MENIMPA balasan pertama pada kolom yang sama, dan yang pertama tidak dapat dipulihkan
// dari mana pun. Pemeriksaan terpisah sebelum menulis tidak menutupnya — dua permintaan
// yang datang bersamaan akan sama-sama lolos.
func TestTheReplyStatementGuardsAgainstOverwriting(t *testing.T) {
	text := strings.ToUpper(query("detail_reply"))

	require.Contains(t, text, "REPLYMESSAGE IS NULL",
		"balasan kedua akan menimpa balasan pertama tanpa dapat dipulihkan")
	require.Contains(t, text, "CASEID = :6",
		"percakapan milik klaim lain dapat dibalas lewat alamat klaim ini")
	require.Contains(t, text, "KOMUNIKASIID = :9",
		"pernyataan balasan kehilangan kunci percakapannya")
}

// Pernyataan balasan TIDAK memberi alias pada tabelnya.
//
// Oracle mengizinkan `UPDATE tabel alias SET alias.kolom = …`; PostgreSQL menolak awalan
// alias pada klausa SET. Bentuk tanpa alias sah di keduanya, dan `D-20` menuntut satu set
// SQL yang berjalan di keduanya.
func TestTheReplyStatementIsPortable(t *testing.T) {
	text := strings.ToUpper(query("detail_reply"))
	require.Contains(t, text, "UPDATE POOLDATA.M_KOMUNIKASI_PNC\n   SET",
		"tabel pada pernyataan UPDATE tidak boleh diberi alias")
}

// Waktu balasan DIIKAT, tidak pernah diisi `sysdate`.
//
// `09-DATABASE-STRATEGY.md` §4 menuntutnya. Pada tulisan pihak luar alasannya nyata: waktu
// yang lahir di basis data tidak dapat diuji secara deterministik, dan balasan pihak luar
// adalah hal yang paling mungkin dipersoalkan kelak.
func TestTheReplyTimeIsBoundNotTakenFromTheDatabase(t *testing.T) {
	text := strings.ToUpper(query("detail_reply"))
	require.NotContains(t, text, "SYSDATE",
		"waktu balasan harus lahir di lapisan aplikasi, bukan di basis data")
	require.NotContains(t, text, "CURRENT_TIMESTAMP",
		"waktu balasan harus lahir di lapisan aplikasi, bukan di basis data")
}

// Isi dokumen dibaca APA ADANYA, tidak dibungkus procedure basis data.
//
// `GetAttachmentFromDB_Sql` membungkusnya `pooldata.base64encode(attachfile)`. Itu
// memanggil procedure — yang `D-02` larang — dan membesarkan muatan sepertiga tanpa satu
// pun manfaat, karena isinya diserahkan sebagai berkas, bukan sebagai teks di dalam JSON.
func TestDocumentContentIsReadWithoutCallingAProcedure(t *testing.T) {
	require.NotContains(t, strings.ToUpper(query("detail_document_content")),
		"BASE64ENCODE",
		"isi dokumen tidak boleh dibungkus procedure basis data")
}

// Login DIIKAT, tidak pernah dirangkai.
//
// Kueri XOL lama merangkainya: `"... where login='" + Local.loginreas + "'"` — dan nilai
// yang dirangkai itu ditulis TETAP ke satu nama reasuradur.
func TestTheLoginIsBoundNeverConcatenated(t *testing.T) {
	for _, name := range reinsurerScopedQueries {
		text := query(name)
		require.NotContains(t, text, "TUGU",
			"kueri %q masih memuat nama reasuradur yang ditulis tetap", name)
		require.NotContains(t, text, "{ASIS",
			"kueri %q masih memuat pola perangkaian Pega", name)
	}
}

// Setiap kueri daftar WAJIB mengembalikan jumlah seluruh baris lewat `COUNT(*) OVER ()`.
func TestEveryListQueryReturnsTheTotalRowCount(t *testing.T) {
	for _, name := range listQueries {
		require.Contains(t, strings.ToUpper(query(name)), "COUNT(*) OVER ()",
			"kueri %q tidak mengembalikan jumlah baris", name)
	}
}

// Setiap kueri daftar WAJIB memotong halamannya DI BASIS DATA.
//
// Ketiga kueri Pega tidak berpaginasi sama sekali — mereka memuat SELURUH baris yang cocok
// ke memori. Itu yang dilarang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2.
func TestEveryListQueryPaginatesInTheDatabase(t *testing.T) {
	for _, name := range listQueries {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "OFFSET", "kueri %q tidak memotong halaman", name)
		require.Contains(t, text, "FETCH NEXT", "kueri %q tidak membatasi baris", name)
	}
}

// Kueri ringkas TIDAK boleh berpaginasi.
//
// Ia menghitung seluruh populasi; memotongnya berarti angka yang lebih kecil daripada
// jumlah baris yang sebenarnya cocok — dan tidak ada yang akan menyadarinya.
func TestCountQueriesDoNotPaginate(t *testing.T) {
	for _, name := range countQueries {
		require.NotContains(t, strings.ToUpper(query(name)), "OFFSET",
			"kueri ringkas %q tidak boleh memotong halaman", name)
	}
}

// Ketiga daftar PEMBERITAHUAN mengecualikan PA dan Travel.
func TestEveryAdviceQueryExcludesPersonalAccidentAndTravel(t *testing.T) {
	names := append(append([]string{}, adviceListQueries...), adviceCountQueries...)
	for _, name := range names {
		require.Contains(t, strings.ToUpper(query(name)), "NOT IN ('002', '005')",
			"kueri %q tidak mengecualikan Personal Accident dan Travel", name)
	}
}

// Ketiga daftar KOMUNIKASI justru TIDAK mengecualikan keduanya.
//
// Ia perbedaan yang mudah "diperbaiki" tanpa sengaja oleh orang yang menyeragamkan keenam
// kueri. `BrowseCommunicationReas` memang tidak memuat satu pun syarat `grouppanel`, dan
// menambahkannya akan MENGHILANGKAN klaim Personal Accident dari daftar komunikasi —
// hilang tanpa galat, dan tanpa ada yang menyadarinya sampai seseorang membandingkannya
// dengan Pega.
func TestCommunicationQueriesDoNotExcludeAnyBusinessLine(t *testing.T) {
	names := append(
		append([]string{}, communicationListQueries...), communicationCountQueries...)
	for _, name := range names {
		require.NotContains(t, strings.ToUpper(query(name)), "GROUPPANEL",
			"kueri %q mengecualikan lini bisnis, sementara Pega tidak", name)
	}
}

// Ketiga daftar komunikasi disaring PERCAKAPAN, bukan dokumen pemberitahuan.
//
// Sebuah klaim masuk daftar itu karena ada percakapan — bukan karena ada PLA maupun DLA.
// Menambahkan syarat dokumen akan mengosongkan daftar bagi mitra yang berkomunikasi
// sebelum satu pun pemberitahuan dikirimkan kepadanya.
func TestCommunicationQueriesFilterOnConversationsNotOnAdvices(t *testing.T) {
	for _, name := range communicationListQueries {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "POOLDATA.M_KOMUNIKASI_PNC",
			"kueri %q tidak menyaring percakapan sama sekali", name)
		require.Contains(t, text, "KOMUNIKASISTATUS = :4",
			"kueri %q tidak mengikat status percakapan", name)
	}
}

// Kedua sisi percakapan dipisahkan menjadi DUA kueri, bukan satu dengan `OR`.
//
// Bentuk `(:n = 'penerima' AND … OR :n = 'pengirim' AND …)` akan membuat Oracle kehilangan
// index pada kedua kolomnya sekaligus — dan tabel percakapan tumbuh seiring SELURUH klaim,
// bukan seiring klaim satu mitra.
func TestEachCommunicationSideHasItsOwnQuery(t *testing.T) {
	recipient := strings.ToUpper(query("list_komunikasi_recipient"))
	sender := strings.ToUpper(query("list_komunikasi_sender"))

	require.Contains(t, recipient, "K.COMMUNICATE_TO")
	require.NotContains(t, recipient, "K.SENDER",
		"daftar komunikasi masuk tidak boleh ikut mencocokkan pengirim")

	require.Contains(t, sender, "K.SENDER")
	require.NotContains(t, sender, "K.COMMUNICATE_TO",
		"daftar komunikasi terkirim tidak boleh ikut mencocokkan penerima")
}

// Daftar Close disaring T_PLALIST, BUKAN T_DLALIST.
//
// Ia mudah dikira salah ketik — daftar klaim selesai yang disaring pemberitahuan ESTIMASI
// — tetapi itu memang bentuk `GetPNCList_PLADLAClose`. Uji ini yang akan gagal bila
// seseorang "memperbaikinya".
func TestTheCloseListIsFilteredByPLANotByDLA(t *testing.T) {
	for _, name := range []string{"list_close", "count_close"} {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "POOLDATA.T_PLALIST",
			"kueri %q seharusnya disaring T_PLALIST", name)
		require.NotContains(t, text, "POOLDATA.T_DLALIST",
			"kueri %q tidak menyaring T_DLALIST di Pega", name)
	}
}

// Daftar Close memakai `IN` atas seluruh kode; kedua daftar lain memakai `=`.
//
// Satu kueri, dua aturan berbeda: kolom "No PLA" pada daftar Close tetap memakai kode
// TERTINGGI, sementara penyaring keanggotaannya memakai semuanya.
func TestOnlyTheCloseListMatchesEveryReinsurerCode(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_close")), "REINSCODE IN (SELECT",
		"daftar Close seharusnya mencocokkan SELURUH kode reasuradur")

	for _, name := range []string{"list_pla", "list_dla"} {
		require.NotContains(t, strings.ToUpper(query(name)), "REINSCODE IN (SELECT",
			"kueri %q seharusnya mencocokkan satu kode saja", name)
		require.Contains(t, strings.ToUpper(query(name)), "REINSCODE = (SELECT",
			"kueri %q kehilangan pencocokan kode tertinggi", name)
	}
}

// Hanya daftar DLA yang mengganti kode status menjadi `1139`.
func TestOnlyTheDLAListReplacesTheStatusWith1139(t *testing.T) {
	for _, name := range []string{"list_dla", "count_dla"} {
		require.Contains(t, query(name), "'1139'",
			"kueri %q kehilangan penggantian status menunggu penutupan", name)
	}

	for _, name := range []string{"list_pla", "list_close", "count_pla", "count_close"} {
		require.NotContains(t, query(name), "'1139'",
			"kueri %q tidak boleh mengganti kode status", name)
	}
}

// Hanya daftar DLA yang menggabungkan tabel kerja Pega secara INNER.
//
// Menyeragamkan ketiganya menjadi INNER akan MENGHILANGKAN baris dari dua daftar lain —
// klaim yang belum punya baris tabel kerja. Baris yang hilang dari antrean tidak
// menghasilkan keluhan sampai seseorang menyadari klaimnya tidak pernah muncul.
func TestOnlyTheDLAListJoinsTheWorkTableInner(t *testing.T) {
	require.NotContains(t, strings.ToUpper(query("list_dla")),
		"LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
		"daftar DLA membaca ISPENDINGCLOSE, sehingga gabungannya harus INNER")

	for _, name := range []string{"list_pla", "list_close"} {
		require.Contains(t, strings.ToUpper(query(name)),
			"LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
			"kueri %q harus memakai LEFT JOIN, setara sub-kueri di Pega", name)
	}
}

// Dokumen dianggap terkirim hanya bila KETIGA syaratnya terpenuhi.
//
// `iskirim = '1'` saja tidak cukup: kueri lama menuntut tanggal kirim dan alamat surel
// ikut terisi. Memeriksa yang pertama saja akan memasukkan dokumen yang ditandai terkirim
// tanpa pernah benar-benar dikirim.
func TestAnAdviceCountsAsSentOnlyWhenAllThreeMarkersAreSet(t *testing.T) {
	names := append(append([]string{}, adviceListQueries...), adviceCountQueries...)
	for _, name := range names {
		text := strings.ToUpper(query(name))
		require.Contains(t, text, "ISKIRIM = '1'", "kueri %q", name)
		require.Contains(t, text, "TGLKIRIM IS NOT NULL", "kueri %q", name)
	}
}

// Kata kunci pencarian DIIKAT dan dilepaskan wildcard-nya.
//
// NOMOR bind-nya berbeda antara kueri daftar dan kueri ringkas — daftar mengambil kolom
// "PLA No" yang binding-nya muncul lebih dulu di klausa SELECT, ringkas tidak. Yang diuji
// karena itu bentuknya, bukan nomornya.
func TestSearchKeywordIsBoundAndEscaped(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)LIKE :\d+ ESCAPE`)

	for _, name := range append(append([]string{}, listQueries...), countQueries...) {
		require.Regexp(t, pattern, query(name),
			"kueri %q tidak melepaskan wildcard kata kunci", name)
	}
}

// Tanggal dikembalikan sebagai TANGGAL, bukan dibungkus `TO_CHAR`.
//
// Kueri lama membungkus keduanya, sehingga pengurutan tanggal menjadi pengurutan TEKS dan
// penyaringan rentang tidak dapat memakai index (`09-DATABASE-STRATEGY.md` §3.2).
func TestDatesAreReturnedAsDatesNotAsFormattedText(t *testing.T) {
	for _, name := range listQueries {
		require.NotContains(t, strings.ToUpper(query(name)), "TO_CHAR(",
			"kueri %q masih memformat tanggal di dalam SQL", name)
	}
}

// Jumlah bind tiap kueri WAJIB sama dengan yang disusun filterArgs dan countArgs.
//
// Selisih satu saja menghasilkan galat bind yang menyebut NOMOR, bukan menyebut kueri
// mana yang rusak.
func TestEachQueryUsesTheExpectedNumberOfBinds(t *testing.T) {
	expected := map[string]int{
		"reinsurer_codes": 1,
		"list_pla":        7, // 3 login + 2 penyaring + offset + ukuran
		"list_dla":        6, // 2 login + 2 penyaring + offset + ukuran
		"list_close":      6, // 2 login + 2 penyaring + offset + ukuran
		"count_pla":       4, // 2 penyaring + 2 login
		"count_dla":       3, // 2 penyaring + 1 login
		"count_close":     3, // 2 penyaring + 1 login

		"list_komunikasi_recipient":  7, // login + 2 penyaring + status + login + halaman
		"list_komunikasi_sender":     7,
		"count_komunikasi_recipient": 4, // 2 penyaring + status + login
		"count_komunikasi_sender":    4,

		"detail_claim_header":        5, // kunci klaim + 4 login
		"detail_advices_pla":         2, // kunci klaim + login
		"detail_advices_dla":         2,
		"detail_documents":           5, // kunci + nomor + jenis + 2 login
		"detail_document_content":    4, // id + kunci + 2 login
		"detail_conversations":       3, // kunci klaim + 2 login
		"detail_conversation_exists": 4, // kunci + percakapan + 2 login
		"detail_reply":               9, // 5 nilai tulis + kunci + 2 login + percakapan
	}

	// Setiap kueri yang ADA wajib disebut di sini. Tanpa itu, kueri baru dapat lolos
	// tanpa satu pun pemeriksaan jumlah bind — dan selisih satu bind menghasilkan galat
	// yang menyebut NOMOR, bukan menyebut kueri mana yang rusak.
	for name := range queries {
		_, listed := expected[name]
		require.True(t, listed,
			"kueri %q belum disebut di daftar jumlah bind", name)
	}

	for name, count := range expected {
		require.Equal(t, count, highestBind(query(name)),
			"jumlah bind kueri %q berbeda dari yang disusun di Go", name)
	}
}

// Jumlah argumen yang DISUSUN GO wajib sama dengan jumlah bind pada SQL-nya.
//
// # Kenapa uji ini berbeda dari yang di atasnya
//
// `TestEachQueryUsesTheExpectedNumberOfBinds` memeriksa SQL terhadap angka yang ditulis
// tangan. Uji ini memeriksa SQL terhadap kode yang benar-benar menyusun argumennya —
// sehingga penambahan satu rantai reasuradur di SQL yang lupa diikuti di `filterArgs` akan
// gagal di sini, bukan di Oracle.
//
// Selisih satu argumen menghasilkan `ORA-01008: not all variables bound` yang menyebut
// NOMOR, bukan menyebut tab mana yang rusak — dan pada modul yang belum pernah dijalankan
// terhadap Oracle, ia tidak akan terlihat sampai seorang mitra membuka layarnya.
func TestGoBuildsAsManyArgumentsAsTheSQLBinds(t *testing.T) {
	for _, tab := range inboxpladla.Tabs() {
		list, count, err := queriesFor(tab)
		require.NoError(t, err, "%s", tab.Code)

		query := inboxpladla.Query{Tab: tab, Caller: inboxpladla.Caller{Login: "UJI"}}

		// Kueri DAFTAR menerima paginasi di ujungnya — dua bind yang tidak disusun
		// filterArgs melainkan ditambahkan List.
		require.Equal(t, highestBind(this(list)), len(filterArgs(query))+2,
			"jumlah argumen kueri daftar %q tidak cocok dengan bind-nya", list)

		require.Equal(t, highestBind(this(count)), len(countArgs(query)),
			"jumlah argumen kueri ringkas %q tidak cocok dengan bind-nya", count)
	}
}

// this mengambil teks kueri; namanya sependek mungkin supaya baris require tetap terbaca.
func this(name string) string { return query(name) }

// Kueri ringkas WAJIB mengelompokkan menurut kode status.
func TestCountQueriesGroupByStatus(t *testing.T) {
	for _, name := range countQueries {
		require.Contains(t, strings.ToUpper(query(name)), "GROUP BY",
			"kueri ringkas %q tidak mengelompokkan apa pun", name)
	}
}

// aliasesOf mengambil alias kolom pada klausa SELECT terluar, berurutan.
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

		if index > 0 && !isWordBoundary(text[index-1]) {
			continue
		}

		rest := text[index:]
		if match := pattern.FindStringSubmatch(rest); match != nil {
			found = append(found, strings.ToUpper(match[1]))
			index += len(match[0]) - 1
			continue
		}

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
