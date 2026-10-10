package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

func TestDetailQueryReturnsTheExpectedAliases(t *testing.T) {
	// Pemindainya memasukkan kolom ke isian menurut URUTAN. Begitu satu alias bergeser,
	// yang terjadi BUKAN galat melainkan nomor klaim yang tampil di kolom status.
	alias := regexp.MustCompile(`(?im)\bAS\s+([A-Z_]+)\s*,?\s*$`)

	found := []string{}
	for _, match := range alias.FindAllStringSubmatch(query("find_claim"), -1) {
		found = append(found, strings.ToUpper(match[1]))
	}
	require.Equal(t, resultColumns, found)
}

func TestDetailQueryBindsTheClaimNumber(t *testing.T) {
	// Tiga bind bernilai sama (sejak 2026-10-08: satu per sumber kunci), dan hanya tiga —
	// pemanggil mengirim nomor klaim tiga kali. Bind tambahan yang masuk diam-diam menghasilkan ORA-01008
	// saat permintaan pertama datang di produksi — kuerinya tidak pernah dijalankan saat
	// kompilasi.
	bind := regexp.MustCompile(`:(\d+)`)
	matches := bind.FindAllString(query("find_claim"), -1)
	require.Equal(t, []string{":1", ":2", ":3"}, matches)
}

func TestDetailQueryNeverDropsRowsWithoutADocument(t *testing.T) {
	// Klaim yang belum punya baris di JSON_KLAIM tetap HARUS dapat dibuka. Dengan INNER
	// JOIN ia dijawab "tidak ditemukan" — padahal ia ada, hanya isinya yang belum tersalin.
	upper := strings.ToUpper(query("find_claim"))
	require.Contains(t, upper, "LEFT JOIN POOLDATA.JSON_KLAIM")
	require.NotContains(t, upper, "INNER JOIN POOLDATA.JSON_KLAIM")
}

func TestDetailQueryKeepsTheTreatyGuard(t *testing.T) {
	// Tanpa penyaring ini, alamat layar dapat diisi nomor klaim PNC biasa dan layar
	// menggambarnya dengan susunan treaty — 97 isian yang hampir seluruhnya kosong, tanpa
	// satu pun tanda bahwa yang dibuka adalah jenis klaim yang berbeda.
	require.Contains(t, query("find_claim"), "'%CLMP%'")
}

func TestEveryLikeComparesAFixedPattern(t *testing.T) {
	// Satu-satunya LIKE di modul ini membandingkan dengan pola TETAP milik kueri, bukan
	// dengan isian pengguna — sehingga ia tidak butuh ESCAPE. Uji ini menjaga agar tidak ada
	// LIKE lain yang masuk diam-diam dengan bind di kanannya.
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

func TestNoValueIsConcatenatedIntoSQL(t *testing.T) {
	// Larangan perangkaian berlaku penuh (`08-TECHNICAL-STRATEGY.md` §4.3). Yang
	// direplikasi dari sistem lama adalah perilaku bisnis, bukan celah injeksinya.
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %s memuat pola {ASIS:…} warisan", name)
		require.NotContainsf(t, text, "{Operator", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "%s", "kueri %s tampak dirangkai lewat fmt", name)
		require.NotContainsf(t, text, "||", "kueri %s merangkai teks di dalam SQL", name)
	}
}

func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	// Daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3.
	//
	// `JSON_VALUE` dan `JSON_TABLE` tidak ada di daftar ini karena keduanya memang tidak
	// dipakai: dokumen klaim dibaca utuh lalu diurai di Go — lihat kepala
	// outstandingclaim.sql.
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
	// Flow Action `OutstandingClaim` di Pega MENYIMPAN kembali objek kerjanya. Membawa
	// perilaku itu ke sini melanggar `P-1`, dan akibatnya bukan galat melainkan dua sistem
	// yang saling menimpa isi satu klaim.
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
	// DB Link tidak punya padanan di PostgreSQL dan diganti pemanggilan API (`D-25`). Satu
	// `@` yang masuk diam-diam akan menggagalkan perpindahan basis data tanpa satu pun
	// tanda sampai cutover.
	for name, text := range queries {
		require.NotContainsf(t, text, "@ASMD", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, text, "@SIMASNET", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, text, "@SMI", "kueri %s menembus DB Link", name)
	}
}
