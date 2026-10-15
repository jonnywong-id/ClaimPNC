package konversicoverage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var aneka = Lines[0]

func object(doc string) ObjectRow {
	return ObjectRow{IDPega: "ID-1", PolicyNo: "P-01", ProdKe: "0", IndexObject: "3", Document: []byte(doc)}
}

func TestCoverageDanSpreadingDiuraiDariCoverageList(t *testing.T) {
	doc := `{"CoverageList":[
	  {"Coverage":"FLEXAS","CoverageNote":"Kebakaran","TSI":"1000000","SumTSI":1000000,
	   "IndexObject":"99","SpreadingList":[
	     {"TreatyType":"10001","SharePercentage":"60"},
	     {"TreatyType":"10015","SharePercentage":"40","FlagDelete":"0"}]},
	  {"Coverage":"RSMDCC","IndexCoverage":"7","SpreadingList":[]}
	]}`
	got, err := Convert(object(doc), aneka)
	require.NoError(t, err)
	require.Len(t, got.Coverage, 2)
	require.Len(t, got.Spreading, 2)

	first := got.Coverage[0]
	require.Equal(t, "FLEXAS", first["COVERAGE"])
	require.Equal(t, "3", first["INDEXOBJECT"], "kunci objek diambil dari baris objek, bukan dari dokumen")
	require.Equal(t, "P-01", first["NOPOLIS"])
	require.Equal(t, json.Number("1"), first["INDEXCOVERAGE"], "urutan bila dokumen tidak menyebutnya")
	require.Equal(t, "7", got.Coverage[1]["INDEXCOVERAGE"], "nilai dokumen dipakai bila ada")

	s := got.Spreading[1]
	require.Equal(t, "FLEXAS", s["COVERAGE"])
	require.Equal(t, json.Number("1"), s["INDEXCOVERAGE"])
	require.Equal(t, json.Number("2"), s["INDEXSPREADING"])
	require.Equal(t, "40", s["SHAREPERCENTAGE"])
}

func TestDokumenPersonMemakaiASMCoverageDanCoverageID(t *testing.T) {
	got, err := Convert(object(`{"ASMCoverage":[{"Coverage":"PA01","TSI":"5"}]}`), Lines[3])
	require.NoError(t, err)
	require.Len(t, got.Coverage, 1)
	require.Equal(t, "PA01", got.Coverage[0]["COVERAGEID"])
}

func TestDokumenLarikLangsungDanKosong(t *testing.T) {
	got, err := Convert(object(`[{"Coverage":"A"}]`), aneka)
	require.NoError(t, err)
	require.Len(t, got.Coverage, 1)

	got, err = Convert(ObjectRow{}, aneka)
	require.NoError(t, err)
	require.Empty(t, got.Coverage)
}

func TestDokumenTakDikenaliDitolakDenganKunciAkarnya(t *testing.T) {
	_, err := Convert(object(`{"Lain":1}`), aneka)
	require.ErrorIs(t, err, ErrUnknownShape)
	require.Contains(t, err.Error(), "LAIN")
}

func TestCoerceMengikutiTipeKolom(t *testing.T) {
	v, w := Coerce("1,250.50", Column{Name: "TSI", DataType: "NUMBER"})
	require.Equal(t, 1250.5, v)
	require.Empty(t, w)

	v, w = Coerce("abc", Column{Name: "TSI", DataType: "NUMBER"})
	require.Nil(t, v)
	require.Contains(t, w, "bukan angka")

	v, _ = Coerce("20240115T170000.000 GMT", Column{Name: "BEGINDATE", DataType: "DATE"})
	require.Equal(t, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), v, "GMT dijadikan tanggal-jam WIB")

	v, _ = Coerce("20240115", Column{Name: "BEGINDATE", DataType: "DATE"})
	require.Equal(t, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), v)

	v, w = Coerce("abcdef", Column{Name: "WORDING", DataType: "VARCHAR2", MaxBytes: 4})
	require.Equal(t, "abcd", v)
	require.Contains(t, w, "dipotong")

	v, _ = Coerce(json.Number("12"), Column{Name: "PRODKE", DataType: "VARCHAR2", MaxBytes: 100})
	require.Equal(t, "12", v)

	v, _ = Coerce("", Column{Name: "TSI", DataType: "NUMBER"})
	require.Nil(t, v)
}

func TestDaftarPolisDipecahDanUnik(t *testing.T) {
	require.Equal(t, []string{"A1", "B2", "C3"}, ParsePolicyList("A1\r\nB2, C3;A1\t "))
}

func TestKonfigurasiMenolakLiveSamaDenganTest(t *testing.T) {
	c := Connection{Role: "LIVE", Host: "h", Port: 1521, Service: "s", User: "u", Password: "p"}
	test := c
	test.Role = "TEST"
	require.ErrorIs(t, Config{Live: c, Test: test}.Validate(), ErrSameDatabase)

	test.Service = "lain"
	require.NoError(t, Config{Live: c, Test: test}.Validate())

	err := Config{Live: c, Test: Connection{Role: "TEST"}}.Validate()
	require.ErrorContains(t, err, "KONVERSI_TEST_HOST")
}

func TestCoverageAnekaMembawaIndexTAnekaList(t *testing.T) {
	// T_COVERAGELIST_ANEKA tidak punya INDEXOBJECT; penghubungnya ke objek INDEXTANEKALIST.
	o := object(`{"CoverageList":[{"Coverage":"100371","IndexTAnekaList":"77"}]}`)
	o.ListIndex = "5"
	got, err := Convert(o, aneka)
	require.NoError(t, err)
	require.Equal(t, "5", got.Coverage[0]["INDEXTANEKALIST"], "nilai baris objek menang atas dokumen")

	o.ListIndex = ""
	got, err = Convert(o, aneka)
	require.NoError(t, err)
	require.Equal(t, "3", got.Coverage[0]["INDEXTANEKALIST"], "tanpa INDEXTANEKALIST, INDEXOBJECT dipakai")

	got, err = Convert(object(`{"CoverageList":[{"Coverage":"ICC"}]}`), Lines[1])
	require.NoError(t, err)
	_, has := got.Coverage[0]["INDEXTANEKALIST"]
	require.False(t, has, "lini selain Aneka tidak diberi kolom ini")
}
