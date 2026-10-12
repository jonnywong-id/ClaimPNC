package konversicoins

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func doc(body string) PolicyDoc {
	return PolicyDoc{IDPega: "ASM-FW-GISFW-WORK POL-1", PolicyNo: "P-01", ProdKe: "2", Document: []byte(body)}
}

// Bentuk dokumen LIVE (terukur 2026-10-08): CoinsList di akar dokumen polis, nilai berupa teks.
func TestCoinsListDiuraiDenganPemetaanEksplisit(t *testing.T) {
	rows, warnings, err := Convert(doc(`{"PolicyNo":"X","CoinsList":[
	  {"CoinsID":"A01","CoinsName":"ASURANSI SATU","Leader":"true","PercentShare":"60",
	   "TSIShare":"600","PremiShare":"6","DiscountShare":"0","Brokerage":"1","PercentBrokerage":"5",
	   "HandingFee":"2","PercentHandlingFee":"3","PPH":"0.1","PPN":"0.2","pxObjClass":"x",
	   "OldCoins":[{"CoinsID":"A01","PercentShare":"50"}]},
	  {"CoinsID":"B02","Leader":"false","PercentShare":40,"FlagDelete":"1"}
	]}`))
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, rows, 2)

	first := rows[0]
	require.Equal(t, "ASM-FW-GISFW-WORK POL-1", first[ColIDPega], "IDPEGA dari baris JSON_POLIS")
	require.Equal(t, "P-01", first[ColPolicyNo])
	require.Equal(t, "2", first[ColProdKe])
	require.Equal(t, "A01", first["COINSID"])
	require.Equal(t, "true", first["LEADER"])
	require.Equal(t, "60", first["PERCENT_SHARE"], "PercentShare → PERCENT_SHARE")
	require.Equal(t, "2", first["HANDLINGFEE"], "HandingFee → HANDLINGFEE")
	require.Equal(t, "0", first["FLAGDELETE"], "tanpa FlagDelete ditulis '0', sama dengan Pega")
	_, hasPctFee := first["PERCENTHANDLINGFEE"]
	require.False(t, hasPctFee, "PERCENTHANDLINGFEE dibiarkan NULL, sama dengan Pega di LIVE")
	_, hasOld := first["OLDPERCENT_SHARE"]
	require.False(t, hasOld, "kolom OLD* tidak diisi")

	require.Equal(t, json.Number("40"), rows[1]["PERCENT_SHARE"])
	require.Equal(t, "1", rows[1]["FLAGDELETE"])
}

func TestTanpaKoasuransiTidakMenghasilkanBaris(t *testing.T) {
	for _, body := range []string{"", `{"PolicyNo":"X"}`, `{"CoinsList":null}`, `{"CoinsList":[]}`} {
		rows, _, err := Convert(doc(body))
		require.NoError(t, err, body)
		require.Empty(t, rows, body)
	}
}

func TestAnggotaTanpaCoinsIDDiperingatkan(t *testing.T) {
	rows, warnings, err := Convert(doc(`{"CoinsList":[{"CoinsName":"X"}]}`))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Len(t, warnings, 1)
}

func TestDokumenRusakDitolak(t *testing.T) {
	_, _, err := Convert(doc(`{rusak`))
	require.Error(t, err)
	_, _, err = Convert(doc(`{"CoinsList":{"CoinsID":"A"}}`))
	require.Error(t, err)
}

func TestCoerce(t *testing.T) {
	v, w := Coerce("1,500.25", Column{Name: "PPH", DataType: "NUMBER"})
	require.Equal(t, 1500.25, v)
	require.Empty(t, w)
	v, w = Coerce("abc", Column{Name: "PPH", DataType: "NUMBER"})
	require.Nil(t, v)
	require.NotEmpty(t, w)
	v, w = Coerce("ABCDEFGHIJKL", Column{Name: "LEADER", DataType: "VARCHAR2", MaxBytes: 10})
	require.Equal(t, "ABCDEFGHIJ", v)
	require.NotEmpty(t, w)
}

func TestParsePolicyList(t *testing.T) {
	require.Equal(t, []string{"A", "B", "C"}, ParsePolicyList("A\nB, C;A\tB"))
}
