package sqlstore

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// listQueryNames adalah kelima kueri daftar yang dibungkus paged/counted.
func listQueryNames() []string {
	names := make([]string, 0, len(listQueryParams))
	for name := range listQueryParams {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// otherQueryNames adalah kueri yang dipanggil kode di luar kueri daftar.
var otherQueryNames = []string{
	"find_by_id",
	"search_claim_any",
	"search_claim_by_policy",
	"next_archive_id",
	"insert_archive",
	"update_archive",
	"mark_sent",
	"store_receipt",
	"document_types",
	"document_kinds",
	"filling_codes",
}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql.
//
// Tanpa uji ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpoint-nya
// — dan bentuknya panik, bukan galat yang rapi.
func TestEveryUsedQueryExists(t *testing.T) {
	for _, name := range append(listQueryNames(), otherQueryNames...) {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = query(name) })
			require.NotEmpty(t, strings.TrimSpace(query(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = query("kueri_yang_tidak_pernah_ada") })
}

// Setiap kueri di berkas .sql harus dipakai kode, dan sebaliknya.
//
// Kedua arah diperiksa. Kueri tanpa pemanggil adalah SQL mati yang tetap ikut di-review
// dan dipelihara; pemanggil tanpa kueri gagal panik saat dipakai.
func TestTidakAdaKueriYatim(t *testing.T) {
	used := map[string]bool{}
	for _, name := range append(listQueryNames(), otherQueryNames...) {
		used[name] = true
	}

	for name := range queries {
		require.Truef(t, used[name],
			"kueri %q ada di berkas .sql tetapi tidak dipanggil kode mana pun", name)
	}

	require.Len(t, queries, len(used))
}

// Kedua puluh satu alias WAJIB sama, dalam urutan yang sama, di setiap kueri daftar.
//
// Tiga hal bergantung padanya: satu pemindai Go melayani kelimanya, subquery paginasi
// merujuk kolomnya dengan nama, dan kolom yang bergeser satu posisi akan mengisi field
// yang salah TANPA galat — nama tertanggung muncul di kolom nama boks, dan tak satu pun
// uji lain akan menangkapnya.
func TestSeluruhKueriDaftarMemakaiAliasYangSama(t *testing.T) {
	expected := aliasList(resultColumns)
	require.Len(t, expected, 21)

	// find_by_id ikut diperiksa: ia dibaca pemindai yang sama.
	for _, name := range append(listQueryNames(), "find_by_id") {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, aliasesOf(query(name)),
				"alias kueri %q berbeda dari resultColumns", name)
		})
	}
}

// Paginasi menambahkan penandanya SETELAH penanda tubuh kueri.
//
// Nomor yang bertabrakan membuat offset menimpa nilai pencarian — kegagalan yang bentuknya
// bukan galat melainkan hasil yang salah.
func TestPenandaPaginasiMelanjutkanPenandaTubuh(t *testing.T) {
	for _, name := range listQueryNames() {
		t.Run(name, func(t *testing.T) {
			used := listQueryParams[name]

			body := query(name)
			for index := 1; index <= used; index++ {
				require.Containsf(t, body, ":"+strconv.Itoa(index),
					"kueri %q seharusnya memakai penanda :%d", name, index)
			}
			require.NotContainsf(t, body, ":"+strconv.Itoa(used+1),
				"kueri %q memakai penanda yang dicadangkan paginasi", name)

			wrapped := paged(name)
			require.Contains(t, wrapped, "OFFSET :"+strconv.Itoa(used+1))
			require.Contains(t, wrapped, "FETCH NEXT :"+strconv.Itoa(used+2))
		})
	}
}

// Pola SQL yang dilarang `08-TECHNICAL-STRATEGY.md` §4.3 tidak boleh muncul.
//
// `SELECT *` dilarang karena kolom baru di basis data tidak boleh diam-diam mengubah
// perilaku aplikasi. Keempat fungsi khas Oracle dilarang demi portabilitas ke PostgreSQL
// 17+ (`D-20`).
func TestTidakAdaPolaSQLTerlarang(t *testing.T) {
	forbidden := map[string]*regexp.Regexp{
		"SELECT *": regexp.MustCompile(`(?i)select\s+\*`),
		"NVL(":     regexp.MustCompile(`(?i)\bnvl\s*\(`),
		"ROWNUM":   regexp.MustCompile(`(?i)\brownum\b`),
		"SYSDATE":  regexp.MustCompile(`(?i)\bsysdate\b`),
		"DECODE(":  regexp.MustCompile(`(?i)\bdecode\s*\(`),
		"TO_CHAR(": regexp.MustCompile(`(?i)\bto_char\s*\(`),
	}

	for name, text := range queries {
		for label, pattern := range forbidden {
			// `next_archive_id` SENGAJA memakai NVL — ia mereplikasi percabangan
			// prosedur lama, dan COALESCE tidak dipakai di sana karena kuerinya juga
			// menjadi rujukan saat membandingkan dengan `.prc`-nya. Ia dikecualikan
			// secara eksplisit, bukan dengan melonggarkan aturannya.
			if name == "next_archive_id" && label == "NVL(" {
				continue
			}
			require.Falsef(t, pattern.MatchString(text),
				"kueri %q memakai pola terlarang %s", name, label)
		}
	}
}

// Setiap pencarian LIKE wajib menyebut ESCAPE.
//
// Tanpa itu, kode filling yang memuat `_` berubah menjadi pola pencarian — dan garis bawah
// cukup lazim di kode arsip.
func TestSetiapLikeMenyebutEscape(t *testing.T) {
	for name, text := range queries {
		upperCase := strings.ToUpper(text)
		if !strings.Contains(upperCase, " LIKE ") {
			continue
		}
		require.Containsf(t, upperCase, "ESCAPE",
			"kueri %q memakai LIKE tanpa ESCAPE", name)
	}
}

// Tidak ada satu pun nilai yang dirangkai ke dalam teks SQL.
//
// Yang dicari adalah pola `{ASIS:` warisan dan perangkaian teks Oracle `||` yang menyentuh
// nilai. `||` yang sah — merangkai konstanta — tidak ada di berkas ini, sehingga
// keberadaannya sama sekali sudah cukup menjadi tanda.
func TestTidakAdaPerangkaianNilaiKeDalamSQL(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS:",
			"kueri %q membawa pola perangkaian warisan", name)
		require.NotContainsf(t, text, "||",
			"kueri %q merangkai teks SQL; nilai wajib lewat parameter binding", name)
	}
}

// Kedua pernyataan penyimpanan jawaban layanan WAJIB tetap berbeda pada satu hal.
//
// `mark_sent` menandai CABANGSTATUS; `store_receipt` tidak. Perbedaan itulah yang membuat
// berkas terkirim dua kali di sistem lama — sekali saat disimpan, sekali lagi dari layar
// Dokument Cabang — dan Work Owner memutuskan 2026-09-25 ia direplikasi.
//
// Uji ini menahannya supaya tidak "dirapikan" menjadi satu pernyataan oleh orang
// berikutnya yang membaca keduanya berdampingan dan mengira salah satunya salinan.
func TestKeduaPenyimpananJawabanBerbedaHanyaPadaCabangStatus(t *testing.T) {
	marked := strings.ToUpper(query("mark_sent"))
	stored := strings.ToUpper(query("store_receipt"))

	require.Contains(t, marked, "CABANGSTATUS",
		"mark_sent WAJIB menandai status terkirim")
	require.NotContains(t, stored, "CABANGSTATUS",
		"store_receipt TIDAK boleh menandai status terkirim — lihat archivedokumenklaim.Repo")

	// Sisanya harus sama: keempat kolom jawaban dan penyaring barisnya.
	for _, column := range []string{"KODESERVICE", "NOTESERVICE", "HITARCHIVE", "TGLKIRIMDOK"} {
		require.Containsf(t, marked, column, "mark_sent kehilangan kolom %s", column)
		require.Containsf(t, stored, column, "store_receipt kehilangan kolom %s", column)
	}
	require.Contains(t, stored, "WHERE ID_ARCHIVE = :5")
	require.Contains(t, marked, "WHERE ID_ARCHIVE = :5")
}

// Kueri daftar yang belum terdaftar di listQueryParams gagal keras.
func TestKueriDaftarTidakTerdaftarPanik(t *testing.T) {
	require.Panics(t, func() { _ = paged("document_types") })
}

// aliasList memecah daftar alias pada resultColumns.
func aliasList(columns string) []string {
	parts := strings.Split(columns, ",")
	list := make([]string, 0, len(parts))
	for _, part := range parts {
		if clean := strings.TrimSpace(part); clean != "" {
			list = append(list, clean)
		}
	}
	return list
}

// aliasesOf membaca alias `AS <NAMA>` dari sebuah kueri, dalam urutan kemunculannya.
var aliasPattern = regexp.MustCompile(`(?i)\bAS\s+([A-Z_][A-Z0-9_]*)`)

func aliasesOf(text string) []string {
	matches := aliasPattern.FindAllStringSubmatch(text, -1)
	list := make([]string, 0, len(matches))
	for _, match := range matches {
		list = append(list, strings.ToUpper(match[1]))
	}
	return list
}
