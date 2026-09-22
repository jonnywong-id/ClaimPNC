package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
)

// sampleQuery adalah permintaan lengkap, dipakai memanggil penyusun argumen tiap tab.
func sampleQuery(tab inboxadmin.Tab) inboxadmin.Query {
	return inboxadmin.Query{
		Tab:      tab,
		Business: inboxadmin.BusinessNonMBU,
		Keyword:  "PNC-1",
		Caller:   inboxadmin.Caller{Login: "ADMINKLAIM"},
	}
}

// tabQueries mengembalikan teks SQL setiap tab.
func tabQueries(t *testing.T) map[string]string {
	t.Helper()

	result := map[string]string{}
	for _, tab := range inboxadmin.Tabs() {
		selected, known := plans[tab.Code]
		require.Truef(t, known, "tab %s (%s) belum punya kueri", tab.Code, tab.Name)
		result[tab.Code] = query(selected.name)
	}
	return result
}

func TestEveryTabHasQuery(t *testing.T) {
	for _, tab := range inboxadmin.Tabs() {
		_, known := plans[tab.Code]
		require.Truef(t, known, "tab %s (%s) tidak ada di plans", tab.Code, tab.Name)
	}
}

func TestDisabledTabsHaveNoQuery(t *testing.T) {
	// Ketiga tab Komunikasi tidak dibangun. Kueri yang menganggur untuk tab yang tidak ada
	// adalah kode mati yang kelak dikira siap dipakai.
	for _, disabled := range inboxadmin.DisabledTabs {
		_, exists := plans[disabled.Code]
		require.Falsef(t, exists, "tab %s (%s) tidak dibangun, kuerinya tidak boleh ada",
			disabled.Code, disabled.Name)
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

func TestEveryTabQueryReturnsTheSameAliases(t *testing.T) {
	// Satu pemindai melayani ketujuh kueri. Begitu satu kueri memakai alias yang berbeda
	// atau urutannya bergeser, pemindai itu memasukkan nilai ke isian yang salah — dan
	// akibatnya BUKAN galat, melainkan kolom yang tertukar di layar.
	// Hanya alias di UJUNG baris kolom yang dihitung. Tanpa syarat itu, `CAST(NULL AS DATE)`
	// ikut terbaca sebagai alias bernama "DATE" — dan kueri yang sebetulnya benar akan
	// dilaporkan salah.
	alias := regexp.MustCompile(`(?im)\bAS\s+([A-Z_]+)\s*,?\s*$`)

	for code, text := range tabQueries(t) {
		found := []string{}
		for _, match := range alias.FindAllStringSubmatch(text, -1) {
			found = append(found, strings.ToUpper(match[1]))
		}
		require.Equalf(t, resultColumns, found,
			"alias kueri tab %s berbeda dari resultColumns", code)
	}
}

func TestBindCountMatchesSuppliedArguments(t *testing.T) {
	// Argumen yang kurang menghasilkan ORA-01008 saat permintaan pertama datang; argumen
	// yang berlebih menghasilkan ORA-01036. Keduanya hanya terlihat di produksi bila tidak
	// diuji di sini, karena kuerinya tidak pernah dijalankan saat kompilasi.
	bind := regexp.MustCompile(`:(\d+)`)

	for _, tab := range inboxadmin.Tabs() {
		selected := plans[tab.Code]
		text := query(selected.name)

		highest := 0
		for _, match := range bind.FindAllStringSubmatch(text, -1) {
			index, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			if index > highest {
				highest = index
			}
		}

		args := selected.args(sampleQuery(tab))
		require.Lenf(t, args, highest,
			"tab %s (%s) memakai bind tertinggi :%d tetapi menyiapkan %d argumen",
			tab.Code, tab.Name, highest, len(args))
	}
}

func TestNoValueIsConcatenatedIntoSQL(t *testing.T) {
	// Inilah cacat terbesar layar lama: tujuh titik `{ASIS:…}` yang menyisipkan POTONGAN
	// SQL, bukan nilai. Keputusan "replikasi apa adanya" berlaku pada perilaku bisnis,
	// bukan pada celah injeksi (`08-TECHNICAL-STRATEGY.md` §4.3).
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %s memuat pola {ASIS:…} warisan", name)
		require.NotContainsf(t, text, "%s", "kueri %s tampak dirangkai lewat fmt", name)

		// Perangkaian `||` hanya sah pada pola LIKE, tempat literal `'%'` disambung ke
		// BIND — bukan ke nilai. Di luar baris LIKE, `||` berarti ada teks yang disusun
		// di dalam SQL, dan itulah yang dilarang.
		for _, line := range strings.Split(text, "\n") {
			if !strings.Contains(line, "||") {
				continue
			}
			require.Containsf(t, strings.ToUpper(line), " LIKE ",
				"kueri %s merangkai teks di luar pola LIKE: %s", name, strings.TrimSpace(line))
		}
	}
}

func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	// Daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3. TO_CHAR ikut dilarang di sini
	// karena kueri lama memakainya untuk PEMFORMATAN TAMPILAN — yang membuat pengurutan
	// tanggal menjadi pengurutan teks.
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
				"kueri %s tampak menulis (%s); modul ini hanya membaca", name, strings.TrimSpace(verb))
		}
	}
}

func TestNoQueryPaginatesInSQL(t *testing.T) {
	// Keputusan Work Owner 2026-09-20: paginasi direplikasi apa adanya, yaitu seluruh
	// baris ditarik lalu dipotong di aplikasi. Kueri yang diam-diam memaginasi akan
	// membuat jumlah "Total Data" di layar menjadi jumlah SATU HALAMAN.
	for name, text := range queries {
		upper := strings.ToUpper(text)
		require.NotContainsf(t, upper, "FETCH NEXT",
			"kueri %s memaginasi di SQL; pemotongan halaman milik inboxadmin.Slice", name)
		require.NotContainsf(t, upper, "OFFSET ",
			"kueri %s memaginasi di SQL; pemotongan halaman milik inboxadmin.Slice", name)
	}
}

func TestEveryLikeUsesEscape(t *testing.T) {
	// Tanpa ESCAPE, satu tanda `%` yang diketik pengguna berubah menjadi pola dan
	// mengembalikan seluruh isi antrean.
	for name, text := range queries {
		for _, line := range strings.Split(text, "\n") {
			upper := strings.ToUpper(line)
			if !strings.Contains(upper, " LIKE ") {
				continue
			}
			// `NOT LIKE '%FixCorrespondence%'` membandingkan dengan pola tetap milik
			// kueri, bukan dengan isian pengguna — ia tidak butuh ESCAPE.
			if strings.Contains(upper, "NOT LIKE") {
				continue
			}
			require.Containsf(t, upper, "ESCAPE",
				"kueri %s memakai LIKE tanpa ESCAPE pada baris: %s", name, strings.TrimSpace(line))
		}
	}
}

func TestBranchFilterIsAbsentEverywhere(t *testing.T) {
	// Penyaring cabang dan korwil MENUNGGU API pengganti DB Link HRD (`R-03`), keputusan
	// Work Owner 2026-09-20. Uji ini menjaga agar ia tidak masuk diam-diam lewat DB Link —
	// yang akan menembus batas yang sengaja belum dilewati.
	for name, text := range queries {
		upper := strings.ToUpper(text)
		require.NotContainsf(t, upper, "@ASMD", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, upper, "V_HRD_MST", "kueri %s membaca master HRD", name)
		require.NotContainsf(t, upper, "LST_USER_ASURANSI", "kueri %s membaca master user asuransi", name)
		require.NotContainsf(t, upper, "BASTERRITORY", "kueri %s menyaring menurut korwil", name)
	}
}

func TestOnlyOneQueryServesBothUnregisteredTabs(t *testing.T) {
	// Kedua tab memang dilayani SATU RDB rule di Pega; yang membedakannya hanya saringan
	// kurir. Memecahnya menjadi dua kueri berarti dua tempat yang harus berubah
	// berpasangan saat aturannya berubah.
	require.Equal(t,
		plans[inboxadmin.TabUnregisteredRCV].name,
		plans[inboxadmin.TabRCVOnline].name,
	)

	normal := plans[inboxadmin.TabUnregisteredRCV].args(sampleQuery(inboxadmin.Tab{}))
	online := plans[inboxadmin.TabRCVOnline].args(sampleQuery(inboxadmin.Tab{}))
	require.Equal(t, "NORMAL", normal[len(normal)-1])
	require.Equal(t, "ONLINE", online[len(online)-1])
}

func TestEmptyKeywordIsSentAsNull(t *testing.T) {
	tab, found := inboxadmin.FindTab(inboxadmin.TabAll)
	require.True(t, found)

	empty := inboxadmin.Query{Tab: tab, Business: inboxadmin.BusinessAll}
	require.Nil(t, plans[tab.Code].args(empty)[0])

	filled := inboxadmin.Query{Tab: tab, Business: inboxadmin.BusinessAll, Keyword: "PNC"}
	require.Equal(t, "PNC", plans[tab.Code].args(filled)[0])
}
