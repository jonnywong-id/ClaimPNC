package lodpdf_test

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/lodpdf"
)

// Format angka contoh PDF LOD: tiga desimal, dibulatkan, nol di belakang dibuang.
func TestFormatAngkaSepertiContohLOD(t *testing.T) {
	t.Parallel()
	require.Equal(t, "77.187.648,3", lodpdf.Amount(7_718_764_830))
	require.Equal(t, "500.000", lodpdf.Amount(50_000_000))
	require.Equal(t, "44.382.897,773", lodpdf.Decimal(big.NewRat(443_828_977_725, 10_000), 3))
	require.Equal(t, "2.508.598,57", lodpdf.Decimal(big.NewRat(250_859_856_975, 100_000), 3))
	require.Equal(t, "11.578.147,245", lodpdf.Decimal(big.NewRat(11_578_147_245, 1_000), 3))
	require.Equal(t, "57.5", lodpdf.Percent(575_000))
	require.Equal(t, "3", lodpdf.Percent(30_000))
	require.Equal(t, "4.75", lodpdf.Percent(47_500))
	require.Equal(t, "100", lodpdf.Percent(registrasi.PercentFull))
	wib := time.Date(2026, 9, 16, 1, 0, 0, 0, time.UTC)
	require.Equal(t, "16 September 2026", lodpdf.LongDate(wib))
	require.Equal(t, "16/09/26", lodpdf.ShortDate(wib))
}

// Setiap jenis yang punya contoh PDF tercetak dengan jumlah halaman contohnya: surat satu
// halaman ditambah Indeks Kepuasan Pelanggan; jenis 3 (catatan co member di halaman kedua)
// dan 6 (dua salinan surat) tiga halaman; Marine Hull empat. Jenis 16 tanpa template
// ditolak. Seluruh data KARANGAN (D-69). LOD_PDF_OUT=<folder> menyimpan hasilnya.
func TestRenderSetiapJenisSesuaiJumlahHalamanContoh(t *testing.T) {
	t.Parallel()
	pages := map[string]int{"3": 3, "6": 3, "11": 4}
	line := registrasi.SettlementLine{Gross: 7_718_764_830}
	// Seperti contohnya: jenis 3 atas polis sembilan anggota koasuransi, jenis lain atas
	// polis tanpa CoinsList (tabelnya satu baris PT. Asuransi Sinar Mas 100 %).
	nine := make([]registrasi.PLACoinsMember, 9)
	for i := range nine {
		nine[i] = registrasi.PLACoinsMember{Name: "ANGGOTA CONTOH " + string(rune('A'+i)) + " - KANTOR PUSAT", Share: 111_111}
	}
	facts := registrasi.LODFacts{
		Currency: "IDR", PolicyNumber: "POLIS-UJI", DateOfLoss: time.Date(2025, 11, 26, 17, 0, 0, 0, time.UTC),
		LossLocation: "JL. CONTOH NO. 1, KOTA CONTOH", InsuredName: "PT CONTOH QQ TERTANGGUNG CONTOH",
		InsuredAddress: "JL. ALAMAT CONTOH NO. 2, KOTA CONTOH",
	}
	out := os.Getenv("LOD_PDF_OUT")
	for _, ty := range registrasi.LODTypesFor(registrasi.Policy{Line: registrasi.LineFire}) {
		var members []registrasi.PLACoinsMember
		if ty.ID == "3" {
			members = nine
		}
		doc := registrasi.BuildLOD(line, ty.ID, members, facts, time.Date(2026, 9, 16, 1, 0, 0, 0, time.UTC))
		pdf, err := lodpdf.Renderer{}.Render(doc)
		if !registrasi.LODTemplateReady(ty.ID) {
			require.Error(t, err, ty.ID)
			continue
		}
		require.NoError(t, err, ty.ID)
		require.Equal(t, "%PDF", string(pdf[:4]))
		want := pages[ty.ID]
		if want == 0 {
			want = 2
		}
		assert.Equal(t, want, bytes.Count(pdf, []byte("/Type /Page\n")), "jenis %s: %s", ty.ID, ty.Name)
		if out != "" {
			require.NoError(t, os.WriteFile(filepath.Join(out, "LOD_"+ty.ID+".pdf"), pdf, 0o600))
		}
	}
}
