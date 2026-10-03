package registrasi_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Pola Pega @CurrentDate("yyMdhm-sS","Asia/Jakarta") + "-" + jenis + "-" + nama: angka
// tanpa nol di depan, jam 12-an, milidetik di ujung; spasi dan karakter asing dibuang.
func TestUploadFileNameFollowsPega(t *testing.T) {
	at := time.Date(2026, time.October, 1, 7, 5, 9, 42_000_000, time.UTC) // 14:05:09.042 WIB
	require.Equal(t, "2610125-942-14904-InvoiceFaktur1.pdf",
		registrasi.UploadFileName(at, "14904", "Invoice Faktur(1).pdf"))
}

// Dua unggahan berkas yang sama pada saat berbeda mendapat nama berbeda — itu yang membuat
// trigger TBIU_STORAGE_IMAGE tidak menolaknya. Ekstensi tetap di ujung.
func TestUploadFileNameIsUniquePerMoment(t *testing.T) {
	a := registrasi.UploadFileName(time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC), "14904", "lod.pdf")
	b := registrasi.UploadFileName(time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC), "14904", "lod.pdf")
	require.NotEqual(t, a, b)
	require.True(t, strings.HasSuffix(a, "-lod.pdf"))
}
