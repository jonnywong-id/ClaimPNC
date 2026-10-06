package sqlstore

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// namedQueries adalah kelima kueri BACA di inboxrcl.sql. Kueri keputusan (decision.sql) —
// satu-satunya yang menulis — dijaga decision_test.go.
var namedQueries = []string{"operator_for", "list_tasks", "claim_detail", "check_tables", "check_columns"}

func TestSetiapKueriBernamaAda(t *testing.T) {
	for _, name := range namedQueries {
		require.NotEmptyf(t, query(name), "kueri %s kosong atau tidak ada", name)
	}
}

func TestTidakAdaKueriTakTerpakaiDiBerkasSQL(t *testing.T) {
	require.Len(t, queries, len(namedQueries)+len(decisionQueries))
}

func aliases(text string) []string {
	result := []string{}
	for _, m := range regexp.MustCompile(`(?i)\bAS\s+([A-Z_]+)`).FindAllStringSubmatch(text, -1) {
		result = append(result, strings.ToUpper(m[1]))
	}
	return result
}

// TestAliasKueriSamaDenganDaftarDanUrutannya menjaga kueri, daftar kolom, dan pemindai sepadan.
func TestAliasKueriSamaDenganDaftarDanUrutannya(t *testing.T) {
	require.Equal(t, taskColumns, aliases(query("list_tasks")))
	require.Equal(t, detailColumns, aliases(query("claim_detail")))
}

// TestSumberDataTC_PNC_PUCL mengunci keputusan Work Owner 2026-10-05: daftar dan layar kerja
// membaca POOLDATA.TC_PNC_PUCL — bukan tabel Pega, bukan T_CLAIMLIST_ADMIN, bukan
// T_ACCESS_GROUP_PNC.
func TestSumberDataTC_PNC_PUCL(t *testing.T) {
	for _, name := range []string{"list_tasks", "claim_detail"} {
		text := strings.ToUpper(query(name))
		require.Containsf(t, text, "FROM POOLDATA.TC_PNC_PUCL P", "kueri %s", name)
		require.NotContainsf(t, text, "JOIN", "kueri %s", name)
	}
	for _, name := range namedQueries {
		text := strings.ToUpper(query(name))
		for _, forbidden := range []string{"DATAPEGA.", "T_CLAIMLIST_ADMIN", "T_ACCESS_GROUP_PNC"} {
			require.NotContainsf(t, text, forbidden, "kueri %s masih membaca %s", name, forbidden)
		}
	}
}

// TestKeempatPenyaringReportDefinitionAda mengunci pemetaan `A AND B AND C AND D`.
func TestKeempatPenyaringReportDefinitionAda(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "UPPER(TRIM(P.ASSIGNED_OPERATOR_ID)) = UPPER(:1)", "penyaring A")
	require.Contains(t, text, "P.STATUS_WORK <> :2", "penyaring B — `!=`, bukan `=`")
	require.Contains(t, text, "P.TGL_KIRIM_PUCL IS NOT NULL", "penyaring C")
	require.Contains(t, text, "TRIM(P.RCL_PUCL) IN (:3, :4)", "penyaring D — RCL dan MSIG")
}

// TestDetailMemakaiPenyaringYangSamaDenganDaftar — layar kerja hanya terbuka bagi klaim yang
// memang tampil di antrean pemanggil; klaim milik dokter lain tidak terbaca.
func TestDetailMemakaiPenyaringYangSamaDenganDaftar(t *testing.T) {
	text := strings.ToUpper(query("claim_detail"))

	require.Contains(t, text, "UPPER(TRIM(P.CLAIMID)) = UPPER(:1)")
	require.Contains(t, text, "UPPER(TRIM(P.ASSIGNED_OPERATOR_ID)) = UPPER(:2)")
	require.Contains(t, text, "P.STATUS_WORK <> :3")
	require.Contains(t, text, "P.TGL_KIRIM_PUCL IS NOT NULL")
	require.Contains(t, text, "TRIM(P.RCL_PUCL) IN (:4, :5)")
	require.Contains(t, text, "P.ALASAN_DOKTER_REJECT_RCL")
}

func TestResolvedRejectedTidakIkutDikecualikan(t *testing.T) {
	require.NotContains(t, query("list_tasks"), "Resolved-Rejected")
}

func TestUrutanMenurunDenganPemutusSeri(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")),
		"ORDER BY P.TGL_CREATE_PUCL DESC, P.CLAIMID DESC")
}

func TestPaginasiDikerjakanBasisData(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "OFFSET :8 ROWS FETCH NEXT :9 ROWS ONLY")
	require.Contains(t, text, "COUNT(*) OVER ()")
}

func TestPencarianDimatikanSaatKataKunciNULL(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")), ":5 IS NULL")
}

// TestOperatorDariMLoginPNC — pemanggil dicari di POOLDATA.M_LOGIN_PNC (login aktif).
func TestOperatorDariMLoginPNC(t *testing.T) {
	text := strings.ToUpper(query("operator_for"))

	require.Contains(t, text, "FROM POOLDATA.M_LOGIN_PNC L")
	require.Contains(t, text, "UPPER(TRIM(L.LOGIN_ID)) = :1")
	require.Contains(t, text, "L.ACTIVE_STATUS = :2")
	require.Contains(t, text, "MAX(")
}

// TestSetiapPenandaBindMunculTepatSekaliDanBerurutan mengunci ORA-01008 (2026-09-27): godror
// mengikat menurut URUTAN KEMUNCULAN, bukan nomornya.
func TestSetiapPenandaBindMunculTepatSekaliDanBerurutan(t *testing.T) {
	expected := map[string]int{"list_tasks": 9, "claim_detail": 5, "operator_for": 2}

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

func TestSeluruhNilaiLewatParameterBinding(t *testing.T) {
	for _, name := range namedQueries {
		text := query(name)
		require.NotContainsf(t, strings.ToUpper(text), "{ASIS", "kueri %s", name)
		require.NotContainsf(t, text, "||", "kueri %s: tidak boleh ada perangkaian", name)
	}
}

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
			require.NotContainsf(t, text, forbidden, "kueri %s: %s", name, forbidden)
		}
	}
}

func TestKueriPeriksaTidakMembacaSatuBarisPun(t *testing.T) {
	for _, name := range []string{"check_tables", "check_columns"} {
		require.Containsf(t, query(name), "1 = 0", "kueri %s", name)
	}
}

func TestKomentarTidakIkutDikirimKeBasisData(t *testing.T) {
	for _, name := range namedQueries {
		require.NotContainsf(t, query(name), "--", "kueri %s masih memuat baris komentar", name)
	}
}
