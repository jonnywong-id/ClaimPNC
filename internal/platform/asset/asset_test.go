package asset_test

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/asset"
)

// LogoASM tertanam dan dapat diurai sebagai gambar — bukan berkas kosong atau rusak.
func TestLogoASMIsAnImage(t *testing.T) {
	b, err := asset.Get(asset.LogoASM)
	require.NoError(t, err)
	img, format, err := image.Decode(bytes.NewReader(b))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.Positive(t, img.Bounds().Dx())
}

func TestUnknownIdentityIsAnError(t *testing.T) {
	_, err := asset.Get("LogoForm")
	require.Error(t, err, "identitas lama LogoForm sudah diganti LogoASM")
	require.Panics(t, func() { asset.MustGet("TidakAda") })
}

// Hanya portal ASM yang punya logo; portal lain tidak meminjam logo ASM.
func TestPortalLogosOnlyASM(t *testing.T) {
	logos := asset.PortalLogos()
	require.Equal(t, asset.MustGet(asset.LogoASM), logos["ASM"])
	for _, portal := range []string{"ASI", "SMAS", "SMI", "SPK", "SPKS"} {
		require.Empty(t, logos[portal], portal)
	}
}
