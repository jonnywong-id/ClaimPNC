package acceptancenotepdf

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"claim-pnc/internal/registrasi"
)

func TestRenderProducesPDF(t *testing.T) {
	at := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	out, err := Renderer{}.Render(registrasi.AcceptanceNote{
		Entity: "ASM", AcceptedNo: "A26", PaymentLabel: "Interim", PolicyNumber: "P-1", ClaimNumber: "PNCN.26.0001",
		InsuredName: "TERTANGGUNG UJI", PolicyCurrency: "IDR", SumInsured: registrasi.Money(200_000_000),
		PeriodStart: at, PeriodEnd: at, DateOfLoss: at, Currency: "IDR", PaymentType: registrasi.PaymentInterim,
		Gross: registrasi.Money(1_000_00), Own: registrasi.Money(575_00), StatusBusiness: "3",
		Installments: []registrasi.AcceptanceNoteInstallment{{Number: "1", PaidAt: at, Amount: big.NewRat(2061547, 100)}},
		Spread:       []registrasi.AcceptanceNoteSpread{{Name: "qs", Share: registrasi.Percent(72_460), Amount: big.NewRat(41, 1)}},
		QS:           []registrasi.AcceptanceNoteQS{{Name: "QS (OR)", Amount: big.NewRat(10, 1)}},
		SignedAt:     at, SignerName: "KOMITE UJI",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatal("bukan PDF")
	}
}

// Format angka mengikuti contoh: "201.713,8", "2.000.000", "69.591,261".
func TestNumber(t *testing.T) {
	for in, want := range map[string]string{"201713.8": "201.713,8", "2000000": "2.000.000", "69591.2614": "69.591,261", "0": "0"} {
		r, _ := new(big.Rat).SetString(in)
		if got := number(r); got != want {
			t.Errorf("%s = %s, bukan %s", in, got, want)
		}
	}
	if got := longDate(time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)); got != "24 September 2026" {
		t.Fatalf("tanggal = %s", got)
	}
}
