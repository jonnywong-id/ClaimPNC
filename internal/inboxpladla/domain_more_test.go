package inboxpladla_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
)

func TestAdviceKindsAreRecognizedRegardlessOfCase(t *testing.T) {
	require.True(t, inboxpladla.AdviceKindPLA.Valid())
	require.True(t, inboxpladla.AdviceKindDLA.Valid())
	require.False(t, inboxpladla.AdviceKind("XOL").Valid())

	kind, ok := inboxpladla.ParseAdviceKind(" dla ")
	require.True(t, ok)
	require.Equal(t, inboxpladla.AdviceKindDLA, kind)

	_, ok = inboxpladla.ParseAdviceKind("xol")
	require.False(t, ok)
}

func TestDetailColumnsAreDescribed(t *testing.T) {
	require.NotEmpty(t, inboxpladla.DocumentColumns())
	require.NotEmpty(t, inboxpladla.ConversationColumns())
}

func TestPaginationOffsetSliceAndTotalPages(t *testing.T) {
	require.Equal(t, 20, inboxpladla.Pagination{Page: 3, Size: 10}.Offset())
	require.Equal(t, 0, inboxpladla.Pagination{}.Offset())

	rows := []inboxpladla.Row{{ClaimNo: "1"}, {ClaimNo: "2"}, {ClaimNo: "3"}}

	page := inboxpladla.Slice(rows, inboxpladla.Pagination{Page: 2, Size: 2})
	require.Equal(t, 3, page.Total)
	require.Equal(t, []inboxpladla.Row{{ClaimNo: "3"}}, page.Items)
	require.Equal(t, 2, page.TotalPages())

	page = inboxpladla.Slice(rows, inboxpladla.Pagination{Page: 1, Size: 3})
	require.Len(t, page.Items, 3)
	require.Equal(t, 1, page.TotalPages())

	page = inboxpladla.Slice(rows, inboxpladla.Pagination{Page: 9, Size: 2})
	require.Empty(t, page.Items)

	require.Equal(t, 1, inboxpladla.Page{}.TotalPages(), "hasil kosong tetap satu halaman")
}

func TestDetailScopeCleanTrimsKeyAndLogin(t *testing.T) {
	scope := inboxpladla.DetailScope{
		ClaimKey: " K ", Login: " L ", ReinsurerCodes: []string{"R1"},
	}.Clean()
	require.Equal(t, inboxpladla.DetailScope{
		ClaimKey: "K", Login: "L", ReinsurerCodes: []string{"R1"},
	}, scope)
}

func TestErrorMessagesForLogs(t *testing.T) {
	require.Equal(t, "inboxpladla: tindakan tidak dikenal belum tersedia",
		inboxpladla.NewNotAvailable("  ").Error())
	require.Equal(t, "inboxpladla: tindakan unduh-semua-pla belum tersedia",
		inboxpladla.NewNotAvailable("unduh-semua-pla").Error())

	require.NoError(t, inboxpladla.NewValidationError(nil))
	err := inboxpladla.NewValidationError([]inboxpladla.Violation{
		{Field: "daftar", Message: "salah"}, {Field: "cari", Message: "panjang"},
	})
	require.EqualError(t, err, "inboxpladla: daftar: salah; cari: panjang")
	require.Equal(t, "inboxpladla: isian tidak sah", (&inboxpladla.ValidationError{}).Error())
}

func TestEffectiveCodesAndSearchMatching(t *testing.T) {
	require.Nil(t, inboxpladla.Query{}.EffectiveReinsurerCodes())

	closeTab, _ := inboxpladla.FindTab(inboxpladla.TabClose)
	plaTab, _ := inboxpladla.FindTab(inboxpladla.TabPLA)
	codes := []string{"R901", "R900"}

	require.Equal(t, codes,
		inboxpladla.Query{Tab: closeTab, ReinsurerCodes: codes}.EffectiveReinsurerCodes())
	require.Equal(t, []string{"R901"},
		inboxpladla.Query{Tab: plaTab, ReinsurerCodes: codes}.EffectiveReinsurerCodes())

	row := inboxpladla.Row{ClaimKey: "ASM-FW-GCNMFW-WORK PNC-2001"}
	require.True(t, inboxpladla.Query{}.Matches(row))
	require.True(t, inboxpladla.Query{Search: "pnc-2001"}.Matches(row))
	require.False(t, inboxpladla.Query{Search: "pnc-9"}.Matches(row))
}
