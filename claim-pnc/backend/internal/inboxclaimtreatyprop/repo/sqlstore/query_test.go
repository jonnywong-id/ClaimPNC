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
		{"worklist_milik_sendiri", inboxclaimtreatyprop.TabWorkList, false},
		{"worklist_lihat_semua", inboxclaimtreatyprop.TabWorkList, true},
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

func TestSeeAllPicksADifferentQuery(t *testing.T) {
	// Kedua keadaan checkbox dilayani kueri yang BERBEDA, bukan satu kueri ber-penyaring
	// yang dapat dimatikan. Kalau keduanya sama, penyaring kepemilikan pasti hilang di
	// salah satunya — dan itu kebocoran antrean, bukan sekadar hasil yang lebih banyak.
	plans := allPlans(t)
	require.NotEqual(t,
		plans["worklist_milik_sendiri"].plan.name,
		plans["worklist_lihat_semua"].plan.name,
	)
}

func TestOnlyScopedQueryBindsTheCallerLogin(t *testing.T) {
	plans := allPlans(t)
	page := samplePage()

	scoped := plans["worklist_milik_sendiri"]
	require.Equal(t, "ADMINTREATY1", scoped.plan.args(scoped.query, page)[0],
		"kueri antrean milik sendiri wajib mengikat login pemanggil")

	all := plans["worklist_lihat_semua"]
	for _, arg := range all.plan.args(all.query, page) {
		require.NotEqual(t, "ADMINTREATY1", arg,
			"kueri lihat semua tidak boleh mengikat login pemanggil")
	}
}

func TestTechnicalQueryBindsTheWorkbasketAccount(t *testing.T) {
	// Nama akun antrean dikirim sebagai BIND, bukan ditulis di dalam SQL. Nilainya karena
	// itu hanya hidup di satu tempat, dan penyimpanan memori membaca konstanta yang sama.
	plans := allPlans(t)
	technical := plans["antrean_teknik"]

	require.Equal(t,
		inboxclaimtreatyprop.TechnicalWorkbasket,
		technical.plan.args(technical.query, samplePage())[0],
	)
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

func TestOnlyTheTechnicalQueryReadsSubjectivity(t *testing.T) {
	// Hanya `GetClaimTreatyTeknik_SQL` yang membawa Subjectivity. Kalau kueri worklist
	// diam-diam ikut membacanya, kolomnya akan terisi di tab yang di sistem lama tidak
	// pernah memilikinya — selisih yang tidak ada di daftar P-5.
	require.Contains(t, query("list_workbasket"), "$.IsSubjectivity")
	require.NotContains(t, query("list_worklist"), "$.IsSubjectivity")
	require.NotContains(t, query("list_worklist_all"), "$.IsSubjectivity")
}

func TestEveryListQueryReadsTheLossDate(t *testing.T) {
	// Inilah perbaikan P-5 yang disetujui Work Owner 2026-09-21. Di sistem lama kueri
	// antrean teknik menaruh Tanggal Kejadian di CARI13 sementara gridnya membaca CARI10,
	// sehingga kolomnya selalu kosong. Uji ini menjaga agar ketiganya benar-benar
	// membawanya, dan dengan alias yang sama.
	for _, name := range listQueries {
		require.Containsf(t, query(name), "$.DateOfLoss",
			"kueri %s tidak membawa Tanggal Kejadian", name)
		require.Containsf(t, query(name), "AS LOSS_DATE",
			"kueri %s tidak mengaliaskan Tanggal Kejadian ke LOSS_DATE", name)
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

func TestEveryLikeComparesAFixedPattern(t *testing.T) {
	// Satu-satunya LIKE di modul ini membandingkan dengan pola TETAP milik kueri
	// (`'%CLMP%'`), bukan dengan isian pengguna — sehingga ia tidak butuh ESCAPE. Uji ini
	// menjaga agar tidak ada LIKE lain yang masuk diam-diam dengan bind di kanannya.
	for name, text := range queries {
		for _, line := range strings.Split(text, "\n") {
			upper := strings.ToUpper(line)
			if !strings.Contains(upper, " LIKE ") {
				continue
			}
			require.Containsf(t, upper, "'%CLMP%'",
				"kueri %s memakai LIKE dengan pola selain penanda klaim treaty: %s",
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
