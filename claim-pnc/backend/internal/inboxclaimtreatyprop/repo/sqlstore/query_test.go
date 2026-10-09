package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatyprop"
)

// samplePage adalah paginasi lengkap, dipakai memanggil penyusun argumen tiap kueri.
func samplePage() inboxclaimtreatyprop.Pagination {
	return inboxclaimtreatyprop.Pagination{Page: 3, Size: 25}
}

// sampleQuery menyusun permintaan untuk sebuah tab, dengan atau tanpa "See All Claim".
func sampleQuery(t *testing.T, code string, seeAll bool) inboxclaimtreatyprop.Query {
	t.Helper()

	tab, found := inboxclaimtreatyprop.FindTab(code)
	require.Truef(t, found, "tab %s tidak ditemukan", code)

	return inboxclaimtreatyprop.Query{
		Tab:    tab,
		SeeAll: seeAll && tab.SupportsSeeAll,
		Caller: inboxclaimtreatyprop.Caller{Login: "ADMINTREATY1"},
	}
}

// allPlans mengembalikan ketiga rencana kueri beserta permintaannya.
func allPlans(t *testing.T) map[string]struct {
	plan  plan
	query inboxclaimtreatyprop.Query
} {
	t.Helper()

	result := map[string]struct {
		plan  plan
		query inboxclaimtreatyprop.Query
	}{}

	cases := []struct {
		label  string
		tab    string
		seeAll bool
	}{
		{"antrean_admin", inboxclaimtreatyprop.TabWorkList, false},
		{"antrean_teknik", inboxclaimtreatyprop.TabTechnical, false},
	}

	for _, c := range cases {
		q := sampleQuery(t, c.tab, c.seeAll)
		selected, err := planFor(q)
		require.NoErrorf(t, err, "%s belum punya kueri", c.label)
		result[c.label] = struct {
			plan  plan
			query inboxclaimtreatyprop.Query
		}{plan: selected, query: q}
	}

	return result
}

func TestEverySelectableTabHasQuery(t *testing.T) {
	for _, tab := range inboxclaimtreatyprop.Tabs() {
		if tab.Blocked {
			continue
		}
		_, err := planFor(inboxclaimtreatyprop.Query{Tab: tab})
		require.NoErrorf(t, err, "tab %s (%s) belum punya kueri", tab.Code, tab.Name)
	}
}

func TestBlockedTabHasNoQuery(t *testing.T) {
	// Tab komite digambar tetapi belum dapat diisi. Kueri yang menganggur untuk tab yang
	// belum dapat dibaca adalah kode mati yang kelak dikira siap dipakai.
	for _, tab := range inboxclaimtreatyprop.Tabs() {
		if !tab.Blocked {
			continue
		}
		_, err := planFor(inboxclaimtreatyprop.Query{Tab: tab})
		require.Errorf(t, err, "tab %s (%s) terhalang, kuerinya tidak boleh ada",
			tab.Code, tab.Name)
	}
}

func TestNoQueryBindsTheCallerLogin(t *testing.T) {
	// Kedua Report Definition tidak menyaring menurut petugas. Satu bind berisi login
	// pemanggil yang masuk diam-diam akan mengembalikan penyaring itu tanpa satu pun
	// keputusan — dan tab yang tiba-tiba menyempit terbaca sebagai data hilang.
	page := samplePage()
	for label, entry := range allPlans(t) {
		for _, arg := range entry.plan.args(entry.query, page) {
			require.NotEqualf(t, "ADMINTREATY1", arg,
				"%s mengikat login pemanggil; Report Definition-nya tidak menyaringnya",
				label)
		}
	}
}

func TestEveryQueryBindsTheWorkClass(t *testing.T) {
	// Kelas objek kerja dikirim sebagai BIND, bukan ditulis di dalam SQL. Nilainya karena
	// itu hanya hidup di satu tempat — inboxclaimtreatyprop.WorkClass — dan penyimpanan
	// memori membaca konstanta yang sama.
	//
	// Ia yang menggantikan pola `CLMP` pada kunci objek kerja: JOIN ke sebuah kelas di Pega
	// membatasi barisnya ke kelas itu, sedangkan pola di tengah teks kunci hanya kebetulan
	// bekerja.
	page := samplePage()
	for label, entry := range allPlans(t) {
		require.Equalf(t, inboxclaimtreatyprop.WorkClass,
			entry.plan.args(entry.query, page)[0],
			"%s tidak mengikat kelas objek kerja sebagai argumen pertama", label)
	}
}

func TestNoQueryBindsTheWorkbasketAccount(t *testing.T) {
	// Report Definition antrean teknik tidak menyaring menurut nama antrean. Nama akun yang
	// masuk kembali sebagai bind akan menyembunyikan antrean bersama lain yang di Pega
	// justru tampil.
	page := samplePage()
	for label, entry := range allPlans(t) {
		for _, arg := range entry.plan.args(entry.query, page) {
			require.NotEqualf(t, inboxclaimtreatyprop.TechnicalWorkbasket, arg,
				"%s masih menyaring menurut nama antrean", label)
		}
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

func TestEveryListQueryReturnsTheSameAliases(t *testing.T) {
	// Satu pemindai melayani ketiga kueri. Begitu satu kueri memakai alias yang berbeda
	// atau urutannya bergeser, pemindai itu memasukkan nilai ke isian yang salah — dan
	// akibatnya BUKAN galat, melainkan kolom yang tertukar di layar.
	//
	// Hanya alias di UJUNG baris kolom yang dihitung. Tanpa syarat itu,
	// `CAST(NULL AS VARCHAR2(100))` ikut terbaca sebagai alias.
	alias := regexp.MustCompile(`(?im)\bAS\s+([A-Z_]+)\s*,?\s*$`)

	for _, name := range listQueries {
		found := []string{}
		for _, match := range alias.FindAllStringSubmatch(query(name), -1) {
			found = append(found, strings.ToUpper(match[1]))
		}
		require.Equalf(t, resultColumns, found,
			"alias kueri %s berbeda dari resultColumns", name)
	}
}

func TestSubjectivityComesFromTheClaimDocument(t *testing.T) {
	// `IsSubjectivity` dulu satu-satunya properti Report Definition yang nama kolom
	// tereksposnya pada objek kerja tidak dapat ditemukan, sehingga ia dikirim NULL.
	//
	// Sejak 2026-10-08 objek kerja tidak lagi dibaca, dan properti yang SAMA
	// (`WorkPage.ClaimData.IsSubjectivity`) dipetik langsung dari dokumen ClaimData —
	// jalurnya terbukti ada lewat JSON_DATAGUIDE atas data dev. Uji ini menjaga ia tidak
	// kembali menjadi tebakan nama kolom.
	for _, name := range listQueries {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, query(name),
			"JSON_VALUE(j.DATA_JSONBLOB, '$.IsSubjectivity')",
			"kueri %s tidak memetik IsSubjectivity dari dokumen klaim", name)
		require.NotContainsf(t, upper, "W.ISSUBJECTIVITY",
			"kueri %s menebak nama kolom IsSubjectivity", name)
	}
}

func TestEveryListQueryReadsBusinessColumnsFromTheClaimDocument(t *testing.T) {
	// Keputusan Work Owner 2026-10-08: tabel objek kerja Pega sudah tidak dipakai. Kolom
	// bisnis dipetik dari `POOLDATA.JSON_KLAIM.DATA_JSONBLOB` — dokumen ClaimData — dengan
	// jalur yang mengikuti properti Report Definition (`WorkPage.ClaimData.<X>` -> `$.<X>`).
	//
	// DATA_JSONBLOB, BUKAN DATA_JSON: terukur di dev, DATA_JSON kosong pada seluruh baris
	// klaim treaty. Menukarnya tidak menghasilkan galat apa pun — hanya kolom kosong.
	paths := map[string]string{
		"MASTER_ID":       "'$.IDMaster'",
		"POLICY_NUMBER":   "'$.PolicyData.PolicyNo'",
		"LOSS_DATE":       "'$.DateOfLoss'",
		"BUSINESS_NAME":   "'$.QuotationData.BusinessName'",
		"BUSINESS_SOURCE": "'$.QuotationData.SobName'",
		"CEDING_COMPANY":  "'$.QuotationData.CedingCoName'",
		"INSURED_NAME":    "'$.InsuredName'",
	}
	for _, name := range listQueries {
		text := query(name)
		for alias, path := range paths {
			require.Containsf(t, text, "JSON_VALUE(j.DATA_JSONBLOB, "+path+")",
				"kueri %s tidak memetik %s dari DATA_JSONBLOB", name, alias)
		}
		require.NotContainsf(t, text, "j.DATA_JSON,",
			"kueri %s membaca DATA_JSON, yang kosong untuk klaim treaty", name)
	}
}

func TestColumnsWithoutReplacementAreTypedNull(t *testing.T) {
	// "Last update" (`WorkPage.pxUpdateOperator`) dan "Status Claim ID"
	// (`WorkPage.pyStatusWork`) milik objek kerja yang sudah tidak dipakai, dan tidak punya
	// padanan terbukti di tabel POOLDATA mana pun. Keduanya dikirim NULL bertipe — bukan
	// diganti kolom penugasan yang terbaca masuk akal tetapi berarti lain.
	for _, name := range listQueries {
		text := query(name)
		for _, alias := range []string{"LAST_UPDATE_OPERATOR", "CLAIM_STATUS"} {
			typedNull := regexp.MustCompile(
				`(?i)CAST\(NULL AS VARCHAR2\(100\)\)\s+AS\s+` + alias + `\b`)
			require.Truef(t, typedNull.MatchString(text),
				"kueri %s tidak mengirim %s sebagai NULL bertipe", name, alias)
		}
		require.NotContainsf(t, strings.ToUpper(text), "PXUPDATEOPERATOR",
			"kueri %s mengganti Last update dengan kolom lain", name)
	}
}

func TestTheClaimDocumentJoinIsLeft(t *testing.T) {
	// Dulu gabungannya INNER ke objek kerja yang ditunjuk penugasan — yang selalu ada.
	// Penggantinya, POOLDATA.JSON_KLAIM, memuat hanya 17 dari 70 objek kerja klaim treaty di
	// dev. INNER di sini akan menghilangkan 53 pekerjaan antrean tanpa satu pun galat.
	for _, name := range listQueries {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, upper, "LEFT JOIN POOLDATA.JSON_KLAIM J",
			"kueri %s tidak menggabungkan dokumen klaim dengan LEFT JOIN", name)
		require.NotContainsf(t, upper, "INNER JOIN",
			"kueri %s memakai INNER JOIN yang dapat menghilangkan pekerjaan", name)
	}
}

func TestNoQueryReadsTheRetiredPegaWorkTable(t *testing.T) {
	// Keputusan Work Owner 2026-10-08: DATAPEGA.PC_ASM_FW_GCNMFW_WORK sudah tidak dipakai.
	// Gabungan yang kembali ke sana akan berjalan di dev (tabelnya masih ada) tetapi gagal
	// begitu tabelnya dicabut.
	for name, text := range queries {
		require.NotContainsf(t, strings.ToUpper(text), "PC_ASM_FW_GCNMFW_WORK",
			"kueri %s masih membaca tabel objek kerja Pega", name)
		require.NotContainsf(t, text, "w.",
			"kueri %s masih merujuk alias objek kerja w", name)
	}
}

func TestBindCountMatchesSuppliedArguments(t *testing.T) {
	// Argumen yang kurang menghasilkan ORA-01008 saat permintaan pertama datang; argumen
	// yang berlebih menghasilkan ORA-01036. Keduanya hanya terlihat di produksi bila tidak
	// diuji di sini, karena kuerinya tidak pernah dijalankan saat kompilasi.
	bind := regexp.MustCompile(`:(\d+)`)

	for label, entry := range allPlans(t) {
		text := query(entry.plan.name)

		highest := 0
		for _, match := range bind.FindAllStringSubmatch(text, -1) {
			index, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			if index > highest {
				highest = index
			}
		}

		args := entry.plan.args(entry.query, samplePage())
		require.Lenf(t, args, highest,
			"%s memakai bind tertinggi :%d tetapi menyiapkan %d argumen",
			label, highest, len(args))
	}
}

func TestPaginationArgumentsFollowTheRequestedPage(t *testing.T) {
	// Offset dan ukuran diambil dari paginasi yang SUDAH dinormalkan. Mengambilnya dari
	// yang diminta akan membuat `halaman=0` menghasilkan offset negatif — yang di Oracle
	// bukan galat melainkan halaman pertama, sehingga cacatnya tidak pernah terlihat.
	page := inboxclaimtreatyprop.Pagination{Page: 3, Size: 10}

	for label, entry := range allPlans(t) {
		args := entry.plan.args(entry.query, page)
		require.Equalf(t, 20, args[len(args)-2], "%s: offset halaman ketiga", label)
		require.Equalf(t, 10, args[len(args)-1], "%s: ukuran halaman", label)
	}
}

func TestNoValueIsConcatenatedIntoSQL(t *testing.T) {
	// `GetClaimTreaty_SQL` menyisipkan `{OperatorID.pyUserIdentifier}` langsung ke teks
	// SQL-nya. Larangan perangkaian berlaku penuh di sini
	// (`08-TECHNICAL-STRATEGY.md` §4.3).
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %s memuat pola {ASIS:…} warisan", name)
		require.NotContainsf(t, text, "{Operator", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "%s", "kueri %s tampak dirangkai lewat fmt", name)
		require.NotContainsf(t, text, "||", "kueri %s merangkai teks di dalam SQL", name)
	}
}

func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	// Daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3.
	forbidden := []string{
		"NVL(", "SYSDATE", "DECODE(", "ROWNUM", "INSTR(", "LISTAGG(",
		"FROM DUAL", "ADD_MONTHS(", "MONTHS_BETWEEN(", "TO_CHAR(", "SELECT *",
	}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContainsf(t, upper, pattern,
				"kueri %s memakai %s yang tidak portabel", name, pattern)
		}
	}
}

func TestQueriesNeverWrite(t *testing.T) {
	// SELURUH tabel yang dibaca modul ini milik sistem lama. Menulis satu saja melanggar
	// `P-1`, dan akibatnya bukan galat melainkan dua sistem yang saling menimpa.
	writing := []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, verb := range writing {
			require.NotContainsf(t, upper, verb,
				"kueri %s tampak menulis (%s); modul ini hanya membaca",
				name, strings.TrimSpace(verb))
		}
	}
}

func TestEveryListQueryPaginatesAndOrders(t *testing.T) {
	// Kebalikan dari Inbox Admin, dan itu disengaja: di sini halaman dipotong basis data.
	// Kueri yang lupa memaginasi akan menarik seluruh antrean ke memori aplikasi, dan
	// kueri yang lupa mengurutkan membuat satu baris muncul di dua halaman sekaligus.
	for _, name := range listQueries {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, upper, "OFFSET ", "kueri %s tidak memaginasi", name)
		require.Containsf(t, upper, "FETCH NEXT", "kueri %s tidak membatasi jumlah baris", name)
		require.Containsf(t, upper, "ORDER BY", "kueri %s tidak menetapkan urutan", name)
		require.Containsf(t, upper, "COUNT(*) OVER ()",
			"kueri %s tidak membawa jumlah seluruh baris", name)
	}
}

func TestNoQueryUsesLikeAnyMore(t *testing.T) {
	// Penyaring pola pada kunci objek kerja digantikan pembatas kelas, dan tidak ada LIKE
	// lain yang sah di modul ini.
	//
	// Uji ini lebih tegas daripada "LIKE harus berpola tetap": sebuah LIKE baru dengan bind
	// di kanannya adalah pola yang dirangkai dari masukan pengguna, dan di sini tidak ada
	// satu pun masukan pengguna yang masuk ke kueri.
	for name, text := range queries {
		require.NotContainsf(t, strings.ToUpper(text), " LIKE ",
			"kueri %s memakai LIKE; pembatas jenis klaim kini PXOBJCLASS", name)
	}
}

func TestEveryListQueryRestrictsTheWorkClass(t *testing.T) {
	// Tanpa pembatas ini, kedua tab menampilkan penugasan SELURUH jenis klaim — bukan hanya
	// treaty proporsional — dengan susunan kolom treaty. Tidak ada galat, hanya baris yang
	// seharusnya tidak ada di sana.
	//
	// Sejak 2026-10-08 pembatasnya kelas yang ditunjuk baris penugasan
	// (`a.PXREFOBJECTCLASS`), bukan `w.PXOBJCLASS` objek kerja yang sudah tidak dibaca —
	// keduanya terbukti setara pada data dev (0 penugasan berselisih kelas).
	for _, name := range listQueries {
		require.Containsf(t, query(name), "a.PXREFOBJECTCLASS = :1",
			"kueri %s tidak membatasi kelas objek kerja", name)
	}
}

func TestQueriesTouchOnlyTheExpectedTables(t *testing.T) {
	// Daftar tabel yang boleh disentuh ditulis tegas. Tanpa ini, satu gabungan tambahan
	// yang ditambahkan kemudian dapat menarik data dari tabel yang belum pernah ditinjau
	// kepemilikannya (`P-1`) maupun kewenangan bacanya.
	allowed := []string{
		"DATAPEGA.PC_ASSIGN_WORKLIST",
		"DATAPEGA.PC_ASSIGN_WORKBASKET",

		// Dokumen klaim — sumber SELURUH kolom bisnis sejak 2026-10-08, menggantikan tabel
		// objek kerja Pega (DATAPEGA.PC_ASM_FW_GCNMFW_WORK) yang sudah tidak dipakai.
		"POOLDATA.JSON_KLAIM",
	}

	table := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+([A-Z_]+\.[A-Z_]+)`)

	for name, text := range queries {
		for _, match := range table.FindAllStringSubmatch(text, -1) {
			found := strings.ToUpper(match[1])
			require.Containsf(t, allowed, found,
				"kueri %s menyentuh tabel di luar daftar: %s", name, found)
		}
	}
}

func TestQueriesNeverCrossDBLink(t *testing.T) {
	// DB Link tidak punya padanan di PostgreSQL dan diganti pemanggilan API (`D-25`).
	// Satu `@` yang masuk diam-diam akan menggagalkan perpindahan basis data tanpa satu
	// pun tanda sampai cutover.
	for name, text := range queries {
		require.NotContainsf(t, text, "@ASMD", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, text, "@SIMASNET", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, text, "@SMI", "kueri %s menembus DB Link", name)
	}
}
