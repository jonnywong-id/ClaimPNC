package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya — sebagai
// panik, di produksi.
func TestAllUsedQueriesExist(t *testing.T) {
	used := []string{
		"masking_list",
		"masking_get",
		"masking_find_pair",
		"masking_next_id",
		"masking_insert",
		"masking_update",
		"masking_set_active",
		"branch_list",
		"branch_exists",
	}
	for _, name := range used {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = getQuery("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (D-20) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *": "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":     "pakai COALESCE",
		"SYSDATE":  "pakai CURRENT_TIMESTAMP, atau isi waktunya dari seam Clock",
		"DECODE(":  "pakai CASE WHEN",
		"ROWNUM":   "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":   "pakai POSITION",
		"LISTAGG(": "pakai STRING_AGG",
		"TO_CHAR(": "pemformatan tanggal dan angka dilakukan di Go",
		"LPAD(":    "pemformatan angka dilakukan di Go",
	}

	for name, text := range query {
		uppercase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, uppercase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, uppercase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// Tidak ada satu pun nilai yang dirangkai ke dalam teks SQL.
//
// Uji ini menjaga hal yang paling mudah terlupakan saat menambah kueri kelak. Layar lama
// merangkai kata kunci pencarian langsung ke pernyataannya lewat `{ASIS:InputSearch.CARI1}`
// (`Activity/SearchDataMasking-Act.xml`), dan itu membuat setiap kata yang diketik pengguna
// menjadi bagian dari SQL-nya sendiri. Seluruh nilai di sini WAJIB lewat parameter binding.
func TestNoStringInterpolationMarkers(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, text, "{ASIS", "kueri %q membawa penanda perangkaian Pega", name)
		require.NotContainsf(t, text, "||'%", "kueri %q merangkai pola LIKE di dalam SQL", name)
		require.NotContainsf(t, text, `|| '%`, "kueri %q merangkai pola LIKE di dalam SQL", name)
	}
}

// Setiap penanda bind muncul TEPAT SEKALI di kueri yang memakainya.
//
// Ini bukan kerapian: `masking_list` sengaja mengikat kata kunci dua kali sebagai :2 dan :3
// alih-alih mengulang satu penanda, supaya kueri tidak bergantung pada perilaku driver
// terhadap penanda berulang. Uji ini menjaga keputusan itu tidak batal saat kueri disunting
// kelak — penanda yang berulang akan membuat jumlah argumen yang dikirim Go tidak lagi
// cocok, dan galatnya hanya muncul saat dijalankan terhadap Oracle sungguhan.
func TestBindMarkersAppearOnce(t *testing.T) {
	for name, text := range query {
		seen := map[string]int{}
		for i := 1; i <= 20; i++ {
			marker := ":" + itoa(i)
			seen[marker] = countMarker(text, marker)
		}
		for marker, total := range seen {
			require.LessOrEqualf(t, total, 1,
				"kueri %q memakai penanda %s sebanyak %d kali; setiap penanda harus muncul sekali",
				name, marker, total)
		}
	}
}

// countMarker menghitung kemunculan penanda bind, dan TIDAK menghitung `:1` sebagai bagian
// dari `:10`. Tanpa pembedaan itu, kueri yang memakai sepuluh bind akan selalu dilaporkan
// melanggar.
func countMarker(text, marker string) int {
	total := 0
	for i := 0; i+len(marker) <= len(text); i++ {
		if text[i:i+len(marker)] != marker {
			continue
		}
		next := i + len(marker)
		if next < len(text) && text[next] >= '0' && text[next] <= '9' {
			continue // sebenarnya penanda yang lebih panjang, mis. :10
		}
		total++
	}
	return total
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
