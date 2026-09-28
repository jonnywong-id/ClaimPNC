package sqlstore

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// namedQueries adalah keempat kueri yang wajib ada di berkas .sql.
var namedQueries = []string{"legacy_operator_for", "list_tasks", "check_tables", "check_columns"}

func TestSetiapKueriBernamaAda(t *testing.T) {
	for _, name := range namedQueries {
		require.NotEmptyf(t, query(name), "kueri %s kosong atau tidak ada", name)
	}
}

func TestTidakAdaKueriTakTerpakaiDiBerkasSQL(t *testing.T) {
	require.Len(t, queries, len(namedQueries))
}

// TestAliasKueriSamaDenganDaftarDanUrutannya menjaga kueri, taskColumns, dan scanTask sepadan.
func TestAliasKueriSamaDenganDaftarDanUrutannya(t *testing.T) {
	aliases := regexp.MustCompile(`(?i)\bAS\s+([A-Z_]+)`).FindAllStringSubmatch(query("list_tasks"), -1)
	require.Len(t, aliases, len(taskColumns))

	for i, match := range aliases {
		require.Equalf(t, taskColumns[i], strings.ToUpper(match[1]),
			"alias ke-%d tidak sepadan dengan taskColumns", i+1)
	}
}

// TestKeempatPenyaringReportDefinitionAda mengunci `pyFilterLogic = "A AND B AND C AND D"`.
func TestKeempatPenyaringReportDefinitionAda(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "UPPER(TRIM(K.PXASSIGNEDOPERATORID)) = UPPER(:1)", "penyaring A")
	require.Contains(t, text, "K.PYSTATUSWORK <> :2", "penyaring B — `!=`, bukan `=`")
	require.Contains(t, text, "K.TANGGALANALYSTSENDRCL_1 IS NOT NULL", "penyaring C")
	require.Contains(t, text, "UPPER(TRIM(K.NAMADOKTERRCL_1)) = UPPER(:3)", "penyaring D")
}

// TestResolvedRejectedTidakIkutDikecualikan — hanya `Resolved-Completed` yang keluar.
func TestResolvedRejectedTidakIkutDikecualikan(t *testing.T) {
	require.NotContains(t, query("list_tasks"), "Resolved-Rejected")
}

func TestKueriDaftarMenyaringKelasObjekKerja(t *testing.T) {
	require.Contains(t, query("list_tasks"), "'ASM-FW-GCNMFW-Work-PNC'")
}

// TestTidakAdaTabelPegaYangDibaca mengunci keputusan Work Owner 2026-09-27: modul ini
// membaca POOLDATA.T_CLAIMLIST_ADMIN, tidak lagi DATAPEGA — di kueri MANA PUN, termasuk
// kueri `-periksa`.
func TestTidakAdaTabelPegaYangDibaca(t *testing.T) {
	for _, name := range namedQueries {
		require.NotContainsf(t, strings.ToUpper(query(name)), "DATAPEGA.",
			"kueri %s masih membaca tabel Pega", name)
	}
}

// TestKueriDaftarMembacaTabelDatarTanpaGabungan — satu baris per klaim, sehingga
// gabungan ke worklist (dan klaim yang tampil dua kali karenanya) hilang.
func TestKueriDaftarMembacaTabelDatarTanpaGabungan(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "FROM POOLDATA.T_CLAIMLIST_ADMIN K")
	require.NotContains(t, text, "JOIN")
	require.NotContains(t, text, "EXISTS")
}

// TestUrutanMenurunDenganPemutusSeriPYID — `pySortOrder = 2` jatuh pada `.pyID`, bukan
// `.pzInsKey` seperti di Inbox Analyst Doctor.
func TestUrutanMenurunDenganPemutusSeriPYID(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")),
		"ORDER BY K.PXCREATEDATETIME DESC, K.PYID DESC")
}

func TestPaginasiDikerjakanBasisData(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY")
	require.Contains(t, text, "COUNT(*) OVER ()")
}

func TestPencarianDimatikanSaatKataKunciNULL(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")), ":4 IS NULL")
}

// TestIdentitasLamaMenyaringKetigaGrupAkses mengunci langkah pertama
// `GetpyUserIdentifierFromTable` — tanpanya pengguna yang di Pega tidak punya
// `TempOperator.City` akan memperoleh antrean.
func TestIdentitasLamaMenyaringKetigaGrupAkses(t *testing.T) {
	text := strings.ToUpper(query("legacy_operator_for"))

	require.Contains(t, text, "POOLDATA.T_ACCESS_GROUP_PNC")
	require.Contains(t, text, "G.STS_AKTIF = '1'")
	require.Contains(t, text, "G.ACCESS_GROUP IN (:2, :3, :4)")
	require.Contains(t, text, "G.ACCESS_GROUP <> :5")
	require.Contains(t, text, "MAX(", "deterministik, bukan pxResults(1)")
}

// TestSeluruhNilaiLewatParameterBinding adalah uji keamanan, bukan uji gaya.
func TestSeluruhNilaiLewatParameterBinding(t *testing.T) {
	for _, name := range namedQueries {
		text := query(name)
		require.NotContainsf(t, strings.ToUpper(text), "{ASIS",
			"kueri %s: perangkaian gaya Pega tidak boleh terbawa", name)
		require.NotContainsf(t, text, "||",
			"kueri %s: tidak boleh ada perangkaian apa pun ke teks SQL", name)
	}
}

// TestSetiapPenandaBindMunculTepatSekaliDanBerurutan mengunci cacat yang ditemukan terhadap
// Oracle 2026-09-27: godror mengikat parameter menurut URUTAN KEMUNCULAN, bukan nomornya,
// sehingga `:4` yang dipakai tiga kali gagal dengan ORA-01008. Uji memori tidak pernah
// menyentuh jalur itu — hanya uji ini yang menjaganya.
func TestSetiapPenandaBindMunculTepatSekaliDanBerurutan(t *testing.T) {
	expected := map[string]int{"list_tasks": 8, "legacy_operator_for": 5}

	for name, count := range expected {
		markers := regexp.MustCompile(`:\d+`).FindAllString(query(name), -1)

		want := make([]string, 0, count)
		for i := 1; i <= count; i++ {
			want = append(want, fmt.Sprintf(":%d", i))
		}
		require.Equalf(t, want, markers,
			"kueri %s: penanda harus :1..:%d, masing-masing tepat sekali dan berurutan", name, count)
	}
}

// TestPolaPencarianDiEscape — "100%" tidak boleh menjadi pola yang cocok dengan semuanya.
func TestPolaPencarianDiEscape(t *testing.T) {
	require.Equal(t, "", searchPattern("   "))
	require.Equal(t, "%PNCN.26%", searchPattern(" pncn.26 "))
	require.Equal(t, `%100\%%`, searchPattern("100%"))
	require.Equal(t, `%A\_B%`, searchPattern("a_b"))
	require.Contains(t, query("list_tasks"), `ESCAPE '\'`)
}

func TestTidakAdaPernyataanYangMenulis(t *testing.T) {
	for _, name := range namedQueries {
		text := strings.ToUpper(query(name))
		for _, forbidden := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "} {
			require.NotContainsf(t, text, forbidden,
				"kueri %s memuat pernyataan yang menulis: %s", name, forbidden)
		}
	}
}

func TestKueriPeriksaTidakMembacaSatuBarisPun(t *testing.T) {
	for _, name := range []string{"check_tables", "check_columns"} {
		require.Containsf(t, query(name), "1 = 0", "kueri %s harus menolak seluruh baris", name)
	}
}

func TestKueriPeriksaKolomMenyebutKetigaKolomMigrasi0012(t *testing.T) {
	text := strings.ToUpper(query("check_columns"))

	require.Contains(t, text, "TANGGALANALYSTSENDRCL_1")
	require.Contains(t, text, "NAMADOKTERRCL_1")
	require.Contains(t, text, "KOMENTARANALISATOR_1")
	require.Contains(t, text, "POOLDATA.T_CLAIMLIST_ADMIN")
}

func TestKomentarTidakIkutDikirimKeBasisData(t *testing.T) {
	for _, name := range namedQueries {
		require.NotContainsf(t, query(name), "--", "kueri %s masih memuat baris komentar", name)
	}
}
