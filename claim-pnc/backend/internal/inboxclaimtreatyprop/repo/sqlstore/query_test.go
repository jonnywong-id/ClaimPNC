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

func TestSubjectivityIsNeverGuessed(t *testing.T) {
	// `IsSubjectivity` adalah satu-satunya properti pada Report Definition yang nama kolom
	// tereksposnya tidak dapat ditemukan di export. Ia dikirim NULL.
	//
	// Menebak namanya menggagalkan SELURUH kueri dengan ORA-00904 — tab yang tidak dapat
	// dibuka sama sekali, alih-alih satu kolom yang kosong. Uji ini yang menahan tebakan
	// itu masuk.
	for _, name := range listQueries {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, upper, "AS SUBJECTIVITY",
			"kueri %s tidak lagi mengirim kolom SUBJECTIVITY", name)
		require.Containsf(t, upper, "CAST(NULL AS VARCHAR2(100))",
			"kueri %s tidak lagi mengirim SUBJECTIVITY sebagai NULL", name)
		require.NotContainsf(t, upper, "W.ISSUBJECTIVITY",
			"kueri %s menebak nama kolom IsSubjectivity", name)
	}
}

func TestEveryListQueryReadsTheLossDateFromTheWorkObject(t *testing.T) {
	// `DATEOFLOSS_1`, BUKAN `DATEOFLOSS`. Pada seluruh rule Pega yang aliasnya menunjuk
	// tabel objek kerja, yang dipakai selalu bentuk ber-`_1` (19 berkas) dan tidak pernah
	// yang tanpa (0 berkas).
	//
	// Salah memilih di antara keduanya tidak menghasilkan galat bila kolom tanpa `_1`
	// kebetulan ada — hanya tanggal milik properti yang berbeda.
	for _, name := range listQueries {
		require.Containsf(t, query(name), "w.DATEOFLOSS_1",
			"kueri %s tidak membaca Tanggal Kejadian dari tabel objek kerja", name)
		require.Containsf(t, query(name), "AS LOSS_DATE",
			"kueri %s tidak mengaliaskan Tanggal Kejadian ke LOSS_DATE", name)
	}
}

func TestNoQueryReadsTheClaimJSONDocument(t *testing.T) {
	// Sumber kolom bisnis BERPINDAH: kedua Report Definition membacanya dari kolom
	// terekspos pada objek kerja, bukan dari dokumen JSON. Kedua sumber dapat berbeda
	// isinya, dan membaca yang salah menampilkan nilai milik salinan yang usang tanpa satu
	// pun galat.
	for name, text := range queries {
		upper := strings.ToUpper(text)
		require.NotContainsf(t, upper, "JSON_VALUE",
			"kueri %s masih memetik dari dokumen JSON", name)
		require.NotContainsf(t, upper, "JSON_KLAIM",
			"kueri %s masih menyentuh POOLDATA.JSON_KLAIM", name)
	}
}

func TestEveryListQueryReadsTheWorkObjectColumns(t *testing.T) {
	// "Last update" dan "Status Claim ID" disebut Report Definition sebagai
	// `WorkPage.pxUpdateOperator` dan `WorkPage.pyStatusWork` — keduanya milik objek kerja.
	// Uji ini memastikan kedua kueri membacanya dari kolom yang SAMA; satu kueri yang
	// mengambilnya dari tempat lain akan menampilkan arti berbeda pada tab yang berbeda,
	// tanpa satu pun galat.
	for _, name := range listQueries {
		text := query(name)
		require.Containsf(t, text, "w.PXUPDATEOPERATOR",
			"kueri %s tidak membaca operator pengubah", name)
		require.Containsf(t, text, "w.PYSTATUSWORK",
			"kueri %s tidak membaca status objek kerja", name)
		require.Containsf(t, text, "INNER JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w",
			"kueri %s menggabungkan tabel objek kerja bukan dengan INNER JOIN", name)
	}
}

func TestTheWorkObjectJoinIsInner(t *testing.T) {
	// Report Definition menyatakan `JOIN type=INNER`. Versi sebelumnya di sini memakai LEFT
	// dengan alasan "penugasan yang objek kerjanya tidak terbaca tetap muncul"; alasan itu
	// DITARIK setelah RD-nya dibaca — yang di-LEFT-join dulu adalah JSON_KLAIM, salinan yang
	// memang boleh belum ada, sedangkan ini objek kerja yang DITUNJUK penugasan itu sendiri.
	//
	// Memakai LEFT di sini akan menampilkan baris yang Pega produksi buang, dan barisnya
	// kosong di SELURUH kolom bisnis — terbaca sebagai data hilang.
	for _, name := range listQueries {
		require.Containsf(t, strings.ToUpper(query(name)),
			"INNER JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
			"kueri %s tidak menggabungkan tabel objek kerja dengan INNER JOIN", name)
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
	for _, name := range listQueries {
		require.Containsf(t, query(name), "w.PXOBJCLASS = :1",
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

		// Tabel objek kerja — sumber SELURUH kolom bisnis sejak layar ini dipasok Report
		// Definition. `POOLDATA.JSON_KLAIM` sengaja TIDAK ada di daftar ini lagi.
		"DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
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
