package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestAllUsedQueriesExist(t *testing.T) {
	used := []string{
		"reas_list",
		"reas_list_search",
		"reas_check_table",
		"reas_count_all",
		"reas_count_duplicate_key",
		"reas_count_shared_login",
		"reas_count_empty_login",
		"reas_count_missing_email",
		"reas_count_without_fallback",
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

// Disiplin SQL portabel (`D-20`) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *":  "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":      "pakai COALESCE",
		"SYSDATE":   "pakai CURRENT_TIMESTAMP",
		"DECODE(":   "pakai CASE WHEN",
		"ROWNUM":    "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":    "pakai POSITION",
		"LISTAGG(":  "pakai STRING_AGG",
		"TO_CHAR(":  "pemformatan tanggal dan angka dilakukan di Go",
		"LPAD(":     "pemformatan angka dilakukan di Go",
		"FROM DUAL": "tidak ada kueri modul ini yang membutuhkannya",
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

// Modul ini HANYA MEMBACA, dan uji ini yang menjaganya tetap begitu.
//
// Keputusannya berdasar bukti: satu-satunya penulis POOLDATA.T_REINSURER di sistem lama
// adalah alur PLA/DLA lewat Database/UPDATEREAS.prc, dipanggil Activity/UpdateDetailPLA2 dan
// Activity/UpdateDetailDLA2 — bukan layar master ini.
//
// Bila kelak terbukti layar lamanya punya tombol simpan (section gridnya memang tidak ada di
// export, `R-16`), uji ini yang akan gagal lebih dulu. Itu memang yang diinginkan:
// penambahan jalur tulis harus menjadi keputusan yang disadari, bukan yang menyelinap.
func TestNoQueryWrites(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		for _, statement := range []string{
			"INSERT", "UPDATE", "DELETE", "MERGE", "TRUNCATE", "DROP ", "LOCK TABLE",
		} {
			require.NotContainsf(t, uppercase, statement,
				"kueri %q memuat %q; modul ini hanya membaca", name, statement)
		}
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL adalah
// celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}, 538 kemunculan, dan yang
// pada tabel ini dirangkai langsung di Activity/SetDataPLADLA:
//
//	"... where login='" + Local.loginreas + "'"
func TestSearchQueryUsesParameterBinding(t *testing.T) {
	text := getQuery("reas_list_search")
	for _, placeholder := range []string{":1", ":2", ":3", ":4"} {
		require.Containsf(t, text, placeholder,
			"kueri pencarian harus memakai parameter %s", placeholder)
	}
	require.NotContains(t, text, "'%",
		"pola LIKE disusun pemanggil, bukan dirangkai di dalam teks SQL")
}

// COUNTRYID sengaja TIDAK di-SELECT.
//
// Ia ditulis Database/UPDATEREAS.prc tetapi tidak dibaca satu pun rule di seluruh export.
// Mengambilnya berarti membawa kolom yang tidak ada pembacanya, dan menampilkannya berarti
// mengarang kegunaan yang tidak dapat ditunjukkan.
func TestCountryIDNotSelected(t *testing.T) {
	for _, name := range []string{"reas_list", "reas_list_search", "reas_check_table"} {
		require.NotContainsf(t, strings.ToUpper(getQuery(name)), "COUNTRYID",
			"kueri %q mengambil COUNTRYID; tidak ada satu pun rule yang membacanya", name)
	}
}

// Urutan kolom pada ketiga kueri baca WAJIB sama — scanRow dipakai untuk ketiganya, dan
// urutan yang berbeda akan menaruh surel di kolom nama tanpa satu pun galat.
func TestReadQueriesShareColumnOrder(t *testing.T) {
	expected := []string{
		"REINSURERID", "REINSURERNAME", "LOGIN", "EMAIL", "COUNTRY", "TYPE",
	}

	for _, name := range []string{"reas_list", "reas_list_search", "reas_check_table"} {
		text := strings.ToUpper(getQuery(name))
		position := -1
		for _, column := range expected {
			at := strings.Index(text, column)
			require.GreaterOrEqualf(t, at, 0, "kueri %q tidak menyebut kolom %s", name, column)
			require.Greaterf(t, at, position,
				"kueri %q menyebut %s di luar urutan; scanRow memakai urutan yang sama untuk ketiganya",
				name, column)
			position = at
		}
	}
}

// Daftar WAJIB terurut. Tanpa ORDER BY, basis data bebas mengembalikan baris dalam urutan
// yang berbeda setiap kali — dan pada daftar yang dipaginasi di layar, itu membuat satu
// baris muncul di dua halaman sekaligus sementara yang lain tidak muncul sama sekali.
func TestListQueriesAreOrdered(t *testing.T) {
	for _, name := range []string{"reas_list", "reas_list_search"} {
		require.Containsf(t, strings.ToUpper(getQuery(name)),
			"ORDER BY REINSURERNAME, TYPE, REINSURERID",
			"kueri %q harus terurut sama dengan repo memori", name)
	}
}

// likePattern meloloskan karakter khusus LIKE, dan urutannya menentukan.
//
// Garis miring terbalik harus diloloskan LEBIH DULU. Membaliknya akan meloloskan garis
// miring yang baru saja ditambahkan, dan pola yang dihasilkan tidak cocok dengan apa pun.
func TestLikePatternEscapesSpecialCharacters(t *testing.T) {
	require.Equal(t, `%ANDALAS%`, likePattern("andalas"))
	require.Equal(t, `%100\%%`, likePattern("100%"),
		"tanpa pelolosan, mengetik % akan mencocokkan SELURUH baris tanpa satu pun tanda")
	require.Equal(t, `%RE\_001%`, likePattern("re_001"),
		"tanpa pelolosan, _ mencocokkan sembarang satu karakter")
	require.Equal(t, `%A\\B%`, likePattern(`a\b`),
		"garis miring terbalik diloloskan lebih dulu; membalik urutannya merusak polanya")
	require.Equal(t, `%ANDALAS%`, likePattern("  andalas  "),
		"spasi ujung dipangkas sebelum pola disusun")
}
