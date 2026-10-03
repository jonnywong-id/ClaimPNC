package dokumenpenunjang_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
)

// Pola sama dengan Pega dan dengan registrasi.UploadFileName.
func TestNamaUnggahMengikutiPega(t *testing.T) {
	saat := time.Date(2026, time.October, 1, 7, 5, 9, 42_000_000, time.UTC) // 14:05:09.042 WIB
	require.Equal(t, "2610125-942-14904-InvoiceFaktur1.pdf",
		dokumenpenunjang.NamaUnggah(saat, "14904", "Invoice Faktur(1).pdf"))
	require.Equal(t, "2610125-942--lod.pdf", dokumenpenunjang.NamaUnggah(saat, "", "lod.pdf"))
}
