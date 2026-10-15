package dlapdf

import (
	"strings"
	"testing"
	"time"

	"claim-pnc/internal/registrasi"
)

// Keempat template DLA membentuk PDF satu halaman.
func TestRenderEveryDLAType(t *testing.T) {
	for _, kind := range []string{
		registrasi.DLATypeCoins, registrasi.DLATypeFacOut, registrasi.DLATypeBPPDAN,
		registrasi.DLATypeEQPool, registrasi.DLATypeTreaty, registrasi.DLATypeFacOblig,
	} {
		doc := registrasi.DLADocument{
			DLA: registrasi.DLA{
				Number: "J261000000000000001", Type: kind, Recipient: "PENERIMA UJI", Date: time.Now(),
				Value: "2500000.00", Percent: "25.00", ShareSpread: "40.000", ClaimAmount: "10000000.00",
				Currency: "IDR", QSRI: "0",
			},
			Entity: "ASM", BusinessName: "Property", PolicyNumber: "POL-UJI", ClaimNumber: "PNCN.26.0001",
			Insured: "TERTANGGUNG UJI", SumInsured: registrasi.FaceSheetAmount{Currency: "IDR", Value: registrasi.Rupiah(500_000_000)},
			PolicyCondition: "AS PER ORIGINAL POLICY", PaymentType: registrasi.PaymentFinal,
			Gross: registrasi.Rupiah(10_000_000), ShareUnder: "FAC-OUT", DueTotal: "2500000.000",
			Remarks: "Catatan\n- Final Payment", SignerName: "KOMITE UJI",
		}
		out, err := Renderer{}.Render(doc)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if !strings.HasPrefix(string(out), "%PDF") {
			t.Fatalf("%s: bukan PDF", kind)
		}
		if n := strings.Count(string(out), "/Type /Page\n"); n != 1 {
			t.Fatalf("%s: %d halaman", kind, n)
		}
	}
}

func TestDecimalAddsThousandSeparators(t *testing.T) {
	for in, want := range map[string]string{"1234567.890": "1,234,567.890", "100": "100", "": "0", "-12345.5": "-12,345.5"} {
		if got := decimal(in); got != want {
			t.Errorf("decimal(%q) = %q, ingin %q", in, got, want)
		}
	}
}
