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
		"reas_update",
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

// Modul ini menulis TEPAT SATU hal: surel, lewat `reas_update`.
//
// Uji ini memagari batas itu. Ia semula melarang SELURUH penulisan — keputusan yang
// berdasar pada bukti yang tersedia saat itu, dan yang **dikoreksi Work Owner 2026-10-05**:
// layar Pega punya kolom Aksi berisi tombol Ubah, dan itu tidak dapat dibaca dari export
// karena section gridnya hilang (`R-16`).
//
// Yang TETAP dilarang, dan alasannya bertahan:
//
//	INSERT    baris baru lahir dari alur PLA/DLA; layar lamanya tanpa tombol Tambah
//	DELETE    tidak satu pun rule menghapus baris tabel ini, dan `D-66` melarangnya
//	MERGE     upsert lewat pintu belakang — ia INSERT yang menyamar
//
// Penambahan jalur tulis berikutnya harus menjadi keputusan yang disadari, bukan yang
// menyelinap: uji ini yang akan gagal lebih dulu.
func TestOnlyUpdateWrites(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)

		for _, statement := range []string{
			"INSERT", "DELETE", "MERGE", "TRUNCATE", "DROP ", "LOCK TABLE",
		} {
			require.NotContainsf(t, uppercase, statement,
				"kueri %q memuat %q; modul ini hanya mengubah surel", name, statement)
		}

		if name != "reas_update" {
			require.NotContainsf(t, uppercase, "UPDATE",
				"hanya reas_update yang boleh menulis; kueri %q tidak", name)
		}
	}
}

// Jalur ubah hanya boleh menyentuh SATU kolom.
//
// `Database/UPDATEREAS.prc` pada baris yang sudah ada hanya menulis `EMAIl`. `LOGIN`,
// `COUNTRY`, dan `COUNTRYID` hanya ditulis pada jalur sisip — yang tidak dibawa modul ini.
//
// `LOGIN` yang paling berakibat bila ikut: ia menentukan klaim mana yang dilihat seorang
// mitra reasuransi (lima kueri inbox menyaringnya), dan mengubahnya dari layar master berarti
// memindahkan visibilitas klaim tanpa satu pun pesan galat (`R-20`).
func TestUpdateTouchesEmailOnly(t *testing.T) {
	text := strings.ToUpper(getQuery("reas_update"))

	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	require.Contains(t, setClause, "EMAIL")

	for _, column := range []string{"LOGIN", "COUNTRY", "COUNTRYID", "REINSURERNAME", "TYPE"} {
		require.NotContainsf(t, setClause, column,
			"klausa SET menyentuh %s; hanya EMAIL yang boleh berubah", column)
	}
}

// Ketiga bagian kunci WAJIB dibungkus COALESCE(TRIM(...), ”).
//
// COALESCE bukan kerapian: `TYPE` benar-benar dapat NULL — `GetListDataLoginReas`
// menyisipkan baris tanpa kolom itu sama sekali. Tanpa COALESCE, baris seperti itu TIDAK
// PERNAH dapat diubah dari layar, dan kegagalannya tampak sebagai "baris tidak ditemukan"
// pada baris yang jelas-jelas tampil.
func TestUpdateKeyHandlesNullAndPadding(t *testing.T) {
	text := strings.ToUpper(getQuery("reas_update"))

	for _, column := range []string{"REINSURERID", "REINSURERNAME", "TYPE"} {
		require.Containsf(t, text, "COALESCE(TRIM("+column+"), '')",
			"bagian kunci %s harus tahan NULL dan spasi padding", column)
	}
	require.NotContains(t, text, "NVL(", "NVL khas Oracle; D-20 menuntut COALESCE")
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
