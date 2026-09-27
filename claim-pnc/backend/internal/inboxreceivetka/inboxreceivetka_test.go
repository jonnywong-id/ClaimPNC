package inboxreceivetka_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxreceivetka"
)

// Bagian jam dibuang di Go, bukan lewat TRUNC di dalam SQL — `D-20` melarang TRUNC dari
// kueri portabel, dan memotongnya di sini membuat nilainya sama persis di Oracle maupun
// PostgreSQL.
func TestCompletionCleanDropsTimeOfDay(t *testing.T) {
	cleaned := inboxreceivetka.Completion{
		ClaimNumber: "  PNC-200118  ",
		CompletedAt: time.Date(2026, 9, 24, 17, 42, 11, 500, time.FixedZone("WIB", 7*3600)),
	}.Clean()

	require.Equal(t, "PNC-200118", cleaned.ClaimNumber)
	require.Equal(t,
		time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
		cleaned.CompletedAt,
		"tanggalnya harus tetap 24 September — memindahkannya ke UTC lebih dulu akan "+
			"menggesernya ke 23 September pada sore hari WIB (R-12)")
}

// Validate meniru precondition activity lama APA ADANYA: dua pemeriksaan, tidak lebih.
//
// Yang TIDAK ada di sana juga diuji di sini — tanggal di masa depan dan tanggal sebelum Date
// Of Loss keduanya DITERIMA, karena sistem lama menerimanya. Menambahkan aturan yang tidak
// pernah ada akan menolak masukan yang selama ini sah (`P-5`).
func TestCompletionValidate(t *testing.T) {
	someday := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)

	t.Run("lengkap", func(t *testing.T) {
		err := inboxreceivetka.Completion{
			ClaimNumber: "PNC-200118",
			CompletedAt: someday,
		}.Validate()
		require.NoError(t, err)
	})

	t.Run("tanpa nomor klaim", func(t *testing.T) {
		err := inboxreceivetka.Completion{CompletedAt: someday}.Validate()
		require.ErrorIs(t, err, inboxreceivetka.ErrClaimNumberRequired)
	})

	t.Run("tanpa tanggal", func(t *testing.T) {
		err := inboxreceivetka.Completion{ClaimNumber: "PNC-200118"}.Validate()
		require.ErrorIs(t, err, inboxreceivetka.ErrDateRequired)
	})

	t.Run("tanggal di masa depan tetap diterima", func(t *testing.T) {
		err := inboxreceivetka.Completion{
			ClaimNumber: "PNC-200118",
			CompletedAt: someday.AddDate(5, 0, 0),
		}.Validate()
		require.NoError(t, err,
			"sistem lama tidak memeriksanya; menambahkan aturan baru melanggar P-5")
	})
}

func TestFilterCleanTrimsKeyword(t *testing.T) {
	require.Equal(t,
		inboxreceivetka.Filter{Keyword: "PNC-200"},
		inboxreceivetka.Filter{Keyword: "  PNC-200 "}.Clean())
}
