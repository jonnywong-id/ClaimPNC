package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
)

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

func TestFindQueryReturnsTheExpectedAliases(t *testing.T) {
	// Satu pemindai membaca kueri ini. Begitu aliasnya berbeda atau urutannya bergeser,
	// pemindai itu memasukkan nilai ke isian yang salah — dan akibatnya BUKAN galat,
	// melainkan dokumen klaim yang terbaca sebagai status alur kerja.
	alias := regexp.MustCompile(`(?im)\bAS\s+([A-Z_]+)\s*,?\s*$`)

	found := []string{}
	for _, match := range alias.FindAllStringSubmatch(query("find_claim"), -1) {
		found = append(found, strings.ToUpper(match[1]))
	}
	require.Equal(t, resultColumns, found)
}

func TestQueryReadsTheDocumentColumnThatActuallyHoldsData(t *testing.T) {
	// `POOLDATA.JSON_KLAIM` punya DUA kolom JSON, dan yang berisi dokumen klaim treaty
	// non-proporsional adalah `DATA_JSONBLOB`. Itu DIUKUR: pada seluruh 14 klaim `CLMNP-%`
	// yang punya baris di tabel itu, `DATA_JSON` berpanjang nol.
	//
	// Kueri inbox non-prop membaca `DATA_JSON` dan karena itu selalu mengembalikan NULL.
	// Uji ini yang mencegah berkas ini "diseragamkan" dengan mereka dan berakhir kosong.
	text := strings.ToUpper(query("find_claim"))
	require.Contains(t, text, "B.DATA_JSONBLOB",
		"kueri rincian tidak membaca kolom yang benar-benar berisi dokumen")
	require.NotRegexp(t, `B\.DATA_JSON\s`, text,
		"kueri rincian membaca DATA_JSON, yang kosong pada klaim CLMNP")
}

func TestQueryAnchorsTheNonPropClaimPrefix(t *testing.T) {
	// `PC_ASM_FW_GCNMFW_WORK` memuat objek kerja SELURUH jenis klaim. Tanpa penyaring ini,
	// alamat layar dapat diisi nomor klaim PNC biasa dan layar akan menggambarnya dengan
	// susunan akseptasi treaty — ~50 isian yang hampir seluruhnya kosong.
	//
	// Jangkar depannya ikut dijaga: `CLMP` milik layar Prop adalah awalan dari `CLMNP` pada
	// tiga huruf pertamanya, sehingga penyaring longgar akan mencampur kedua layar treaty.
	want := "'" + inputacceptation.ClaimPrefix + "%'"
	require.Contains(t, strings.ToUpper(query("find_claim")), want)
}

func TestQueriesNeverWrite(t *testing.T) {
	// SELURUH tabel yang dibaca modul ini milik sistem lama. Menulis satu saja melanggar
	// `P-1`, dan akibatnya bukan galat melainkan dua sistem yang saling menimpa.
	//
	// Uji ini yang menjaga jalur Submit tidak diam-diam diberi `UPDATE` sebelum kepemilikan
	// tabelnya benar-benar berpindah (`D-63`).
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

func TestNoValueIsConcatenatedIntoSQL(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %s memuat pola {ASIS:…} warisan", name)
		require.NotContainsf(t, text, "{Inputdata", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "%s", "kueri %s tampak dirangkai lewat fmt", name)
		require.NotContainsf(t, text, "||", "kueri %s merangkai teks di dalam SQL", name)
	}
}

func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	// Daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3.
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

func TestQueriesTouchOnlyTheExpectedTables(t *testing.T) {
	// Daftar tabel yang boleh disentuh ditulis tegas. Tanpa ini, satu gabungan tambahan yang
	// ditambahkan kemudian dapat menarik data dari tabel yang belum pernah ditinjau
	// kepemilikannya (`P-1`) maupun kewenangan bacanya.
	allowed := []string{
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
	for name, text := range queries {
		require.NotContainsf(t, text, "@ASMD", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, text, "@SIMASNET", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, text, "@SMI", "kueri %s menembus DB Link", name)
	}
}

func TestJoinStaysLeftJoin(t *testing.T) {
	// Klaim yang belum punya baris di JSON_KLAIM tetap DAPAT DIBUKA, dengan isian kosong dan
	// nomor klaimnya terbaca. Menggantinya dengan INNER akan membuat klaim seperti itu
	// dijawab "tidak ditemukan" — padahal ia ada, hanya isinya yang belum tersalin.
	require.Contains(t, strings.ToUpper(query("find_claim")), "LEFT JOIN POOLDATA.JSON_KLAIM")
}
