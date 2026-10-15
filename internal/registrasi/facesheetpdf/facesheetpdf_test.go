package facesheetpdf

import (
	"bytes"
	"testing"
	"time"

	"claim-pnc/internal/registrasi"
)

// Format angka mengikuti contoh PDF produksi.
func TestNumberFormatFollowsTheSample(t *testing.T) {
	cases := map[string]string{
		FormatMoney(registrasi.Rupiah(610_175_716_556)): "610.175.716.556",
		FormatMoney(registrasi.Money(2_650_000_049)):    "26.500.000",
		FormatMoney(0):         "0",
		FormatPercent(70_175):  "7,018",
		FormatPercent(570_000): "57",
		FormatPercent(47_500):  "4,75",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("mau %q, dapat %q", want, got)
		}
	}
}

func TestRenderProducesAPDF(t *testing.T) {
	out, err := Renderer{}.Render(registrasi.FaceSheet{
		Title: "ALL RISK", ClaimNumber: "PNCN.26.0001", Date: time.Now(), Insured: "Nama – dengan tanda pisah",
		Reserve:  []registrasi.FaceSheetAmount{{Currency: "IDR", Value: registrasi.Rupiah(1)}},
		CoMember: []registrasi.FaceSheetShare{{Name: "ANGGOTA DENGAN NAMA YANG SANGAT PANJANG SEKALI - KANTOR PUSAT", Share: 1, Currency: "IDR"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatalf("bukan PDF")
	}
}
