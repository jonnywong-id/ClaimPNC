package dlapdf

import (
	"bytes"
	"image"
	"image/png"
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

// Logo kop DLA dipilih menurut Portal dokumen, bukan entitas alamat kaki.
func TestDLALogoFollowsPortal(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewGray(image.Rect(0, 0, 4, 4)) // tanpa alfa: satu objek gambar, tanpa masker
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	r := Renderer{Logo: map[string][]byte{"ASM": buf.Bytes()}}
	images := func(portal string) int {
		out, err := r.Render(registrasi.DLADocument{
			DLA:    registrasi.DLA{Number: "H1", Type: registrasi.DLATypeBPPDAN, Date: time.Now(), Currency: "IDR"},
			Entity: "ASM", Portal: portal,
		})
		if err != nil {
			t.Fatal(err)
		}
		return strings.Count(string(out), "/Subtype /Image")
	}
	if n := images("ASM"); n != 1 {
		t.Fatalf("ASM: %d gambar, ingin 1", n)
	}
	for _, portal := range []string{"SMAS", "SMI", "SPK", "SPKS", "ASI", ""} {
		if n := images(portal); n != 0 {
			t.Fatalf("%s: %d gambar, ingin tanpa logo", portal, n)
		}
	}
}

// Footer (kop bawah) per portal: ASM, ASI, dan SMI punya alamat dari template Pega; portal
// lain tidak punya dan tidak boleh meminjam alamat badan hukum lain.
func TestFooterFollowsPortal(t *testing.T) {
	want := map[string]string{
		"ASM": "PT Asuransi Sinar Mas",
		"ASI": "PT ASURANSI SIMAS INSURTECH",
		"SMI": "Sinar Mas Insurance, S.A.",
	}
	for portal, name := range want {
		if footer[portal][0] != name {
			t.Fatalf("%s: footer %q, ingin %q", portal, footer[portal][0], name)
		}
	}
	for _, portal := range []string{"SMAS", "SPK", "SPKS"} {
		if _, ok := footer[portal]; ok {
			t.Fatalf("%s tidak boleh punya footer tanpa sumber", portal)
		}
	}
}
