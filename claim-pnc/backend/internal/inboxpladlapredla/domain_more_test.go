package inboxpladlapredla_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
)

// Offset dihitung dari paginasi yang sudah dibetulkan, bukan dari isian mentah.
func TestOffsetIsComputedFromTheNormalizedPagination(t *testing.T) {
	require.Equal(t, 20, inboxpladlapredla.Pagination{Page: 3, Size: 10}.Offset())
	require.Equal(t, 0, inboxpladlapredla.Pagination{Page: 0, Size: 0}.Offset())
	require.Equal(t, 100, inboxpladlapredla.Pagination{Page: 2, Size: 500}.Offset())
}

// Jumlah halaman membulatkan sisa ke atas.
func TestTotalPagesRoundsTheRemainderUp(t *testing.T) {
	page := inboxpladlapredla.Page{
		Total: 25, Pagination: inboxpladlapredla.Pagination{Page: 1, Size: 10},
	}
	require.Equal(t, 3, page.TotalPages())

	page.Total = 20
	require.Equal(t, 2, page.TotalPages())
}

func rowsNumbered(n int) []inboxpladlapredla.Row {
	rows := make([]inboxpladlapredla.Row, 0, n)
	for i := 1; i <= n; i++ {
		rows = append(rows, inboxpladlapredla.Row{ClaimNo: string(rune('A' + i - 1))})
	}
	return rows
}

// Slice memotong halaman tengah, halaman terakhir yang tidak penuh, dan halaman di luar
// jangkauan.
func TestSliceCutsTheRequestedPage(t *testing.T) {
	all := rowsNumbered(5)

	middle := inboxpladlapredla.Slice(all, inboxpladlapredla.Pagination{Page: 2, Size: 2})
	require.Equal(t, 5, middle.Total)
	require.Equal(t, []string{"C", "D"}, claimNos(middle.Items))

	last := inboxpladlapredla.Slice(all, inboxpladlapredla.Pagination{Page: 3, Size: 2})
	require.Equal(t, []string{"E"}, claimNos(last.Items))

	beyond := inboxpladlapredla.Slice(all, inboxpladlapredla.Pagination{Page: 9, Size: 2})
	require.Empty(t, beyond.Items)
	require.NotNil(t, beyond.Items)
	require.Equal(t, 5, beyond.Total)
	require.Equal(t, 9, beyond.Pagination.Page)
}

func claimNos(rows []inboxpladlapredla.Row) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ClaimNo)
	}
	return out
}

// Pencarian yang terisi mencocokkan kunci kerja, dan baris yang tidak memuatnya ditolak.
func TestMatchesRejectsARowWithoutTheKeyword(t *testing.T) {
	query, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{Search: "pnc-1001"}, caller())
	require.NoError(t, err)

	require.True(t, query.Matches(inboxpladlapredla.Row{
		ClaimKey: "ASM-FW-GCNMFW-WORK PNC-1001"}))
	require.False(t, query.Matches(inboxpladlapredla.Row{
		ClaimKey: "ASM-FW-GCNMFW-WORK PNC-2002"}))

	// Tanpa kata kunci, setiap baris lolos.
	open, err := inboxpladlapredla.NewQuery(inboxpladlapredla.QueryInput{}, caller())
	require.NoError(t, err)
	require.True(t, open.Matches(inboxpladlapredla.Row{ClaimKey: "apa saja"}))
}

// Pesan log penolakan tombol menyebut tindakannya, atau menyebut "tidak dikenal".
func TestNotAvailableErrorMessageNamesTheAction(t *testing.T) {
	known := inboxpladlapredla.NewNotAvailable("kirim")
	require.Equal(t, "inboxpladlapredla: tindakan kirim belum tersedia", known.Error())

	unknown := inboxpladlapredla.NewNotAvailable("entah")
	require.Equal(t,
		"inboxpladlapredla: tindakan tidak dikenal belum tersedia", unknown.Error())
}

// Pesan log galat validasi merangkai seluruh pelanggaran.
func TestValidationErrorMessageJoinsEveryViolation(t *testing.T) {
	err := inboxpladlapredla.NewValidationError([]inboxpladlapredla.Violation{
		{Field: "dari", Message: "salah"},
		{Field: "sampai", Message: "terbalik"},
	})
	require.EqualError(t, err, "inboxpladlapredla: dari: salah; sampai: terbalik")

	empty := &inboxpladlapredla.ValidationError{}
	require.Equal(t, "inboxpladlapredla: isian tidak sah", empty.Error())
}

// Dokumen beralamat dan belum terkirim boleh dikirim.
func TestAnUnsentAdviceWithAnEmailCanBeSent(t *testing.T) {
	advice := inboxpladlapredla.SendableAdvice{Email: "reas@contoh.example", Sent: "0"}
	require.NoError(t, advice.CanBeSent())
}
