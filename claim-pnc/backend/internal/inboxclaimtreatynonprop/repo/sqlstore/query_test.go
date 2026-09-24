package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatynonprop"
)

// samplePage adalah paginasi lengkap, dipakai memanggil penyusun argumen tiap kueri.
func samplePage() inboxclaimtreatynonprop.Pagination {
	return inboxclaimtreatynonprop.Pagination{Page: 3, Size: 25}
}

// sampleQuery menyusun permintaan untuk sebuah tab beserta keadaan kedua checkbox.
func sampleQuery(
	t *testing.T, code string, seeAll, tbaOnly bool,
) inboxclaimtreatynonprop.Query {
	t.Helper()

	tab, found := inboxclaimtreatynonprop.FindTab(code)
	require.Truef(t, found, "tab %s tidak ditemukan", code)

	return inboxclaimtreatynonprop.Query{
		Tab:     tab,
		SeeAll:  seeAll && tab.SupportsSeeAll,
		TBAOnly: tbaOnly && tab.SupportsTBAOnly,
		Caller:  inboxclaimtreatynonprop.Caller{Login: "ADMINNONPROP1"},
	}
}

type planCase struct {
	plan  plan
	query inboxclaimtreatynonprop.Query
}

// allPlans mengembalikan kelima rencana kueri beserta permintaannya.
//
// Kelimanya disebut lengkap, bukan disimpulkan dari kombinasi: kombinasi yang dihitung
// sendiri oleh uji akan ikut salah bila pemilihan kuerinya salah.
func allPlans(t *testing.T) map[string]planCase {
	t.Helper()

	cases := []struct {
		label   string
		tab     string
		seeAll  bool
		tbaOnly bool
	}{
		{"admin_milik_sendiri", inboxclaimtreatynonprop.TabAdmin, false, false},
		{"admin_lihat_semua", inboxclaimtreatynonprop.TabAdmin, true, false},
		{"admin_tba_saja", inboxclaimtreatynonprop.TabAdmin, false, true},
		{"admin_semua_dan_tba", inboxclaimtreatynonprop.TabAdmin, true, true},
		{"antrean_teknik", inboxclaimtreatynonprop.TabTechnical, false, false},
	}

	result := map[string]planCase{}
	for _, c := range cases {
		q := sampleQuery(t, c.tab, c.seeAll, c.tbaOnly)
		selected, err := planFor(q)
		require.NoErrorf(t, err, "%s belum punya kueri", c.label)
		result[c.label] = planCase{plan: selected, query: q}
	}

	return result
}

func TestEverySelectableTabHasQuery(t *testing.T) {
	for _, tab := range inboxclaimtreatynonprop.Tabs() {
		if tab.Blocked {
			continue
		}
		_, err := planFor(inboxclaimtreatynonprop.Query{Tab: tab})
		require.NoErrorf(t, err, "tab %s (%s) belum punya kueri", tab.Code, tab.Name)
	}
}

func TestBlockedTabHasNoQuery(t *testing.T) {
	// Tab komite digambar tetapi belum dapat diisi. Kueri yang menganggur untuk tab yang
	// belum dapat dibaca adalah kode mati yang kelak dikira siap dipakai.
	for _, tab := range inboxclaimtreatynonprop.Tabs() {
		if !tab.Blocked {
			continue
		}
		_, err := planFor(inboxclaimtreatynonprop.Query{Tab: tab})
		require.Errorf(t, err, "tab %s (%s) terhalang, kuerinya tidak boleh ada",
			tab.Code, tab.Name)
	}
}

func TestEveryCheckboxCombinationPicksItsOwnQuery(t *testing.T) {
	// Keempat kombinasi checkbox tab Admin dilayani kueri yang BERBEDA-BEDA, bukan satu
	// kueri ber-penyaring yang dapat dimatikan. Kalau ada dua yang sama, salah satu
	// penyaringnya pasti hilang — dan yang paling mungkin hilang adalah penyaring
	// kepemilikan, yang berarti kebocoran antrean.
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

func TestOnlyScopedQueriesBindTheCallerLogin(t *testing.T) {
	plans := allPlans(t)
	page := samplePage()

	for _, label := range []string{"admin_milik_sendiri", "admin_tba_saja"} {
		scoped := plans[label]
		require.Equalf(t, "ADMINNONPROP1", scoped.plan.args(scoped.query, page)[0],
			"%s wajib mengikat login pemanggil", label)
	}

	for _, label := range []string{"admin_lihat_semua", "admin_semua_dan_tba"} {
		open := plans[label]
		for _, arg := range open.plan.args(open.query, page) {
			require.NotEqualf(t, "ADMINNONPROP1", arg,
				"%s tidak boleh mengikat login pemanggil", label)
		}
	}
}

func TestTechnicalQueryBindsTheWorkbasketAccount(t *testing.T) {
	// Nama akun antrean dikirim sebagai BIND, bukan ditulis di dalam SQL. Nilainya karena
	// itu hanya hidup di satu tempat, dan penyimpanan memori membaca konstanta yang sama.
	plans := allPlans(t)
	technical := plans["antrean_teknik"]

	require.Equal(t,
		inboxclaimtreatynonprop.TechnicalWorkbasket,
		technical.plan.args(technical.query, samplePage())[0],
	)
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

func TestEveryListQueryReturnsTheSameAliases(t *testing.T) {
	// Satu pemindai melayani kelima kueri. Begitu satu kueri memakai alias yang berbeda
	// atau urutannya bergeser, pemindai itu memasukkan nilai ke isian yang salah — dan
	// akibatnya BUKAN galat, melainkan kolom yang tertukar di layar.
	//
	// Hanya alias di UJUNG baris kolom yang dihitung. Tanpa syarat itu,
	// `CAST(NULL AS VARCHAR2(100))` dan `CAST(… AS DATE)` ikut terbaca sebagai alias.
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

func TestStatusTextMatchesTheDomainConstants(t *testing.T) {
	// Teks status adalah LITERAL di dalam kueri, bukan kolom — sehingga ia satu-satunya
	// nilai di modul ini yang hidup di dua tempat sekaligus: di SQL dan di konstanta
	// domain yang dipakai penyimpanan memori. Uji ini yang menjaga keduanya tidak
	// berselisih.
	for _, name := range adminQueries {
		require.Containsf(t, query(name),
			"'"+inboxclaimtreatynonprop.StatusEstimation+"'",
			"kueri %s tidak memakai teks status tab Admin", name)
	}

	require.Contains(t, query("list_technical"),
		"'"+inboxclaimtreatynonprop.StatusAcceptation+"'",
		"kueri antrean teknik tidak memakai teks status tab Teknik")

	require.NotContains(t, query("list_technical"),
		"'"+inboxclaimtreatynonprop.StatusEstimation+"'",
		"kueri antrean teknik memakai teks status tab Admin")
}

func TestOnlyAdminQueriesReadTheJSONMasterID(t *testing.T) {
	// `GetInboxListCNP_SQL` tidak membawa `CARI23` sama sekali. Kalau kueri Teknik
	// diam-diam ikut membacanya, kolomnya akan terisi di tab yang di sistem lama tidak
	// pernah memilikinya — selisih yang tidak ada di daftar P-5.
	for _, name := range adminQueries {
		require.Containsf(t, query(name), "'$.IDMaster'",
			"kueri %s tidak membawa ID Master dari blob JSON", name)
	}
	require.NotContains(t, query("list_technical"), "'$.IDMaster'")
}

func TestOnlyTBAQueriesFilterTheEmptyPolicy(t *testing.T) {
	// `NOPOLIS IS NULL` adalah SATU-SATUNYA yang membedakan kueri TBA. Kalau ia bocor ke
	// kueri lain, tab Admin biasa akan kehilangan seluruh baris yang polisnya sudah
	// terbit — yaitu hampir seluruh isinya.
	tba := []string{"list_admin_tba", "list_admin_all_tba"}
	biasa := []string{"list_admin", "list_admin_all", "list_technical"}

	for _, name := range tba {
		require.Containsf(t, strings.ToUpper(query(name)), "NOPOLIS IS NULL",
			"kueri %s tidak menyaring klaim yang polisnya belum terbit", name)
	}
	for _, name := range biasa {
		require.NotContainsf(t, strings.ToUpper(query(name)), "NOPOLIS IS NULL",
			"kueri %s ikut menyaring polis kosong; ia tidak seharusnya", name)
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
	page := inboxclaimtreatynonprop.Pagination{Page: 3, Size: 10}

	for label, entry := range allPlans(t) {
		args := entry.plan.args(entry.query, page)
		require.Equalf(t, 20, args[len(args)-2], "%s: offset halaman ketiga", label)
		require.Equalf(t, 10, args[len(args)-1], "%s: ukuran halaman", label)
	}
}

func TestNoValueIsConcatenatedIntoSQL(t *testing.T) {
	// `GetKlaimNonPropAdmin_SQL` menyisipkan `{Inputdata.CARI10}` langsung ke teks SQL-nya,
	// dan `GetWorkCNP_Act` bahkan merangkai potongan klausa WHERE dari string. Larangan
	// perangkaian berlaku penuh di sini (`08-TECHNICAL-STRATEGY.md` §4.3).
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %s memuat pola {ASIS:…} warisan", name)
		require.NotContainsf(t, text, "{Inputdata", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "{Operator", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "%s", "kueri %s tampak dirangkai lewat fmt", name)
		require.NotContainsf(t, text, "||", "kueri %s merangkai teks di dalam SQL", name)
	}
}

func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	// Daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3.
	//
	// `SYSDATE` dan `TRUNC(` termasuk di dalamnya, dan keduanya dipakai kueri lama untuk
	// menghitung Aging. Penggantinya ada di catatan 5 pada kepala berkas .sql — hasilnya
	// sama, bentuknya portabel.
	forbidden := []string{
		"NVL(", "SYSDATE", "DECODE(", "ROWNUM", "INSTR(", "LISTAGG(",
		"FROM DUAL", "ADD_MONTHS(", "MONTHS_BETWEEN(", "TO_CHAR(", "TRUNC(", "SELECT *",
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
	// Di sini halaman dipotong basis data. Kueri yang lupa memaginasi akan menarik seluruh
	// antrean ke memori aplikasi, dan kueri yang lupa mengurutkan membuat satu baris
	// muncul di dua halaman sekaligus.
	for _, name := range listQueries {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, upper, "OFFSET ", "kueri %s tidak memaginasi", name)
		require.Containsf(t, upper, "FETCH NEXT", "kueri %s tidak membatasi jumlah baris", name)
		require.Containsf(t, upper, "ORDER BY", "kueri %s tidak menetapkan urutan", name)
		require.Containsf(t, upper, "COUNT(*) OVER ()",
			"kueri %s tidak membawa jumlah seluruh baris", name)
	}
}

func TestEveryLikeComparesTheAnchoredClaimPrefix(t *testing.T) {
	// Satu-satunya LIKE di modul ini membandingkan dengan pola TETAP milik kueri, bukan
	// dengan isian pengguna — sehingga ia tidak butuh ESCAPE.
	//
	// Jangkar depannya ikut dijaga. Tanpa jangkar, `'%CLMNP-%'` akan menangkap nomor yang
	// memuatnya di tengah; dan yang lebih berbahaya, pola milik layar saudaranya
	// (`'%CLMP%'`) tidak boleh muncul di sini sama sekali.
	want := "'" + inboxclaimtreatynonprop.ClaimPrefix + "%'"

	for name, text := range queries {
		for _, line := range strings.Split(text, "\n") {
			upper := strings.ToUpper(line)
			if !strings.Contains(upper, " LIKE ") {
				continue
			}
			require.Containsf(t, upper, want,
				"kueri %s memakai LIKE dengan pola selain awalan klaim non-prop: %s",
				name, strings.TrimSpace(line))
		}
	}
}

func TestQueriesTouchOnlyTheExpectedTables(t *testing.T) {
	// Daftar tabel yang boleh disentuh ditulis tegas. Tanpa ini, satu gabungan tambahan
	// yang ditambahkan kemudian dapat menarik data dari tabel yang belum pernah ditinjau
	// kepemilikannya (`P-1`) maupun kewenangan bacanya.
	allowed := []string{
		"DATAPEGA.PC_ASSIGN_WORKLIST",
		"DATAPEGA.PC_ASSIGN_WORKBASKET",
		"DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
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

func TestQueriesReadTheNonPropJSONColumn(t *testing.T) {
	// `POOLDATA.JSON_KLAIM` punya DUA kolom JSON, dan layar Prop membaca yang BERBEDA.
	// Menukar salah satunya tidak menghasilkan galat apa pun — ia hanya menampilkan
	// tanggal dan ID master milik dokumen yang lain. Lihat catatan "DUA KOLOM JSON" di
	// kepala berkas .sql.
	for _, name := range listQueries {
		require.NotContainsf(t, strings.ToUpper(query(name)), "DATA_JSONBLOB",
			"kueri %s membaca kolom JSON milik layar Prop, bukan milik layar ini", name)
	}
}
