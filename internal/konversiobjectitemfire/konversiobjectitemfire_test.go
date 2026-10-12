package konversiobjectitemfire

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func object(doc string) ObjectRow {
	return ObjectRow{IDPega: "ID-1", PolicyNo: "P-01", ProdKe: "3", IndexObject: "2", Document: []byte(doc)}
}

// Bentuk dokumen LIVE (terukur 2026-10-08): {"PropertyItemList":[{...}]}.
func TestItemDiuraiMenjadiBarisDenganKunciDariObjek(t *testing.T) {
	rows, err := Convert(object(`{"PropertyItemList":[
	  {"ItemType":"BUILDING","ItemTypeIndo":"BANGUNAN","PropertyItemGroup":"BUILDING(S)",
	   "TSIObjectItem":"2700000000","FlagDelete":"0","pxObjClass":"ASM-FW-GISFW-Data-PropertyItem",
	   "INDEXOBJECT":"99"},
	  {"ItemType":"CONTENTS","TSIObjectItem":15000000}
	]}`))
	require.NoError(t, err)
	require.Len(t, rows, 2)

	first := rows[0]
	require.Equal(t, "BUILDING", first["ITEMTYPE"])
	require.Equal(t, "BANGUNAN", first["ITEMTYPEINDO"])
	require.Equal(t, "BUILDING(S)", first["PROPERTYITEMGROUP"])
	require.Equal(t, "2", first["INDEXOBJECT"], "kunci objek diambil dari baris objek, bukan dari dokumen")
	require.Equal(t, "P-01", first["NOPOLIS"])
	require.Equal(t, "3", first["PRODKE"])
	require.Equal(t, "ID-1", first["IDPEGA"])
	_, hasClass := first["PXOBJCLASS"]
	require.False(t, hasClass, "pxObjClass tidak punya kolom dan tidak dilaporkan")
	require.Equal(t, json.Number("15000000"), rows[1]["TSIOBJECTITEM"])
}

func TestDokumenKosongAtauNullTidakMenghasilkanBaris(t *testing.T) {
	for _, doc := range []string{"", "   ", `{"PropertyItemList":null}`, `{"PropertyItemList":[]}`} {
		rows, err := Convert(object(doc))
		require.NoError(t, err, doc)
		require.Empty(t, rows, doc)
	}
}

func TestBentukTakDikenalDitolak(t *testing.T) {
	_, err := Convert(object(`{"CoverageList":[]}`))
	require.ErrorIs(t, err, ErrUnknownShape)
	_, err = Convert(object(`{rusak`))
	require.Error(t, err)
}

func TestParsePolicyList(t *testing.T) {
	require.Equal(t, []string{"A", "B", "C"}, ParsePolicyList("A\nB, C;A\tB"))
}

func TestNewFilter(t *testing.T) {
	f, err := NewFilter("  ", "")
	require.NoError(t, err)
	require.True(t, f.Empty(), "kosong berarti seluruh PRODKE dan INDEXOBJECT")

	f, err = NewFilter(" 01 ", "077")
	require.NoError(t, err)
	require.Equal(t, Filter{ProdKe: "1", IndexObject: "77"}, f)
	require.Equal(t, "PRODKE 1, INDEXOBJECT 77", f.Describe())

	_, err = NewFilter("1a", "")
	require.ErrorIs(t, err, ErrInvalidFilter)
}
