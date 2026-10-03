package plapdf

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

func samplePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func sampleDoc() registrasi.PLADocument {
	day := time.Date(2026, 6, 5, 5, 0, 0, 0, time.UTC)
	return registrasi.PLADocument{
		PLA: registrasi.PLA{
			Number: "J261000000000000001", Recipient: "ANGGOTA SATU", Date: day, Note: "Estimation only",
			Amount: []registrasi.PLAAmount{
				{Currency: "IDR", Reserve: registrasi.Rupiah(50_000_000), Base: registrasi.Rupiah(50_000_000), Share: 200_000, Result: registrasi.Rupiah(10_000_000)},
				{Currency: "USD"},
			},
		},
		BusinessName: "Property", PolicyNumber: "POL-1", ClaimNumber: "PNCN.26.1", Insured: "TERTANGGUNG",
		Interest: "Gedung", SumInsured: registrasi.FaceSheetAmount{Currency: "IDR", Value: registrasi.Rupiah(500_000_000)},
		PeriodStart: day, PeriodEnd: day.AddDate(1, 0, 0), PolicyCondition: "AS PER ORIGINAL POLICY",
		DateOfLoss: day, NatureOfLoss: "Kebakaran", LossLocation: "Jakarta", SignerName: "KOMITE",
	}
}

func pages(out []byte) int { return strings.Count(string(out), "/Type /Page\n") }

// PLA membentuk PDF satu halaman untuk setiap entitas, dengan dan tanpa kop serta tanda tangan.
func TestRenderPLAVariants(t *testing.T) {
	logo := samplePNG(t)

	for _, entity := range []string{"ASM", "ASI", "TIDAK-DIKENAL"} {
		doc := sampleDoc()
		doc.Entity = entity
		plain, err := Renderer{}.Render(doc)
		require.NoError(t, err, entity)
		require.True(t, bytes.HasPrefix(plain, []byte("%PDF")), entity)
		require.Equal(t, 1, pages(plain), entity)

		doc.Signature = logo
		doc.Place = "Surabaya"
		doc.Interest = " "
		withImages, err := Renderer{Logo: map[string][]byte{"ASM": logo, "ASI": logo}}.Render(doc)
		require.NoError(t, err, entity)
		require.Equal(t, 1, pages(withImages), entity)
		// Gambar kop dan tanda tangan ikut tertanam, sehingga dokumennya lebih besar.
		require.Greater(t, len(withImages), len(plain), entity)
	}

	// Gambar yang tidak terbaca dilewati; dokumen tetap terbentuk.
	doc := sampleDoc()
	doc.Signature = []byte("bukan png")
	out, err := Renderer{Logo: map[string][]byte{"ASM": []byte("rusak")}}.Render(doc)
	require.NoError(t, err)
	require.Equal(t, 1, pages(out))
}

func TestFormatHelpers(t *testing.T) {
	require.Equal(t, "", moment(time.Time{}))
	require.Equal(t, "", date(time.Time{}))
	at := time.Date(2026, 6, 5, 18, 30, 0, 0, time.UTC)
	require.Equal(t, "06/06/26 01:30", moment(at))
	require.Equal(t, "06/06/2026", date(at))
	require.Equal(t, time.Date(2026, 6, 6, 1, 30, 0, 0, clock.ZoneWIB).Format("02/01/2006"), date(at))

	require.True(t, strings.HasPrefix(amount("IDR", registrasi.Rupiah(1_000)), "IDR "))
	require.Equal(t, strings.TrimSpace(amount("", registrasi.Rupiah(1))), amount("", registrasi.Rupiah(1)))

	require.Nil(t, normalizePNG(nil))
	require.Nil(t, normalizePNG([]byte("x")))
	normalized := normalizePNG(samplePNG(t))
	require.True(t, bytes.HasPrefix(normalized, []byte("\x89PNG")))
}
