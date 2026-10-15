package inboxclaimtreatynonprop_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatynonprop"
)

// Pesan galat validasi menyebut setiap pelanggaran, dan tetap terbaca bila daftarnya kosong.
func TestValidationErrorMessage(t *testing.T) {
	empty := inboxclaimtreatynonprop.NewValidationError(nil)
	require.Equal(t, "inboxclaimtreatynonprop: isian tidak sah", empty.Error())

	full := inboxclaimtreatynonprop.NewValidationError([]inboxclaimtreatynonprop.Violation{
		{Field: "tab", Message: "salah"},
		{Field: "lain", Message: "juga salah"},
	})
	require.Equal(t, "inboxclaimtreatynonprop: tab: salah; lain: juga salah", full.Error())
}

// Paginasi di luar rentang dibetulkan, bukan ditolak.
func TestPaginationNormalize(t *testing.T) {
	require.Equal(t,
		inboxclaimtreatynonprop.Pagination{Page: 1, Size: inboxclaimtreatynonprop.DefaultPageSize},
		inboxclaimtreatynonprop.Pagination{Page: -3, Size: 0}.Normalize())
	require.Equal(t,
		inboxclaimtreatynonprop.Pagination{Page: 2, Size: inboxclaimtreatynonprop.MaxPageSize},
		inboxclaimtreatynonprop.Pagination{Page: 2, Size: 5000}.Normalize())
	require.Equal(t, 50, inboxclaimtreatynonprop.Pagination{Page: 3, Size: 25}.Offset())
}

// Jumlah halaman dibulatkan ke atas, dan pembagian pas tidak menambah halaman.
func TestTotalPagesRounding(t *testing.T) {
	exact := inboxclaimtreatynonprop.Page{
		Total: 50, Pagination: inboxclaimtreatynonprop.Pagination{Page: 1, Size: 25},
	}
	require.Equal(t, 2, exact.TotalPages())

	over := inboxclaimtreatynonprop.Page{
		Total: 51, Pagination: inboxclaimtreatynonprop.Pagination{Page: 1, Size: 25},
	}
	require.Equal(t, 3, over.TotalPages())
}

// Halaman di luar jangkauan menghasilkan senarai kosong, bukan nil, dengan total yang utuh.
func TestSliceBeyondLastPage(t *testing.T) {
	all := []inboxclaimtreatynonprop.WorkItem{{ClaimID: "A"}, {ClaimID: "B"}, {ClaimID: "C"}}

	beyond := inboxclaimtreatynonprop.Slice(all, inboxclaimtreatynonprop.Pagination{Page: 5, Size: 2})
	require.NotNil(t, beyond.Items)
	require.Empty(t, beyond.Items)
	require.Equal(t, 3, beyond.Total)

	last := inboxclaimtreatynonprop.Slice(all, inboxclaimtreatynonprop.Pagination{Page: 2, Size: 2})
	require.Equal(t, []inboxclaimtreatynonprop.WorkItem{{ClaimID: "C"}}, last.Items)
}

// Tab yang tidak dikenal ditolak sebagai galat validasi pada isian tab.
func TestNewQueryUnknownTab(t *testing.T) {
	_, err := inboxclaimtreatynonprop.NewQuery(
		inboxclaimtreatynonprop.QueryInput{Tab: "99"},
		inboxclaimtreatynonprop.Caller{Login: "ADMIN"},
	)

	var validation *inboxclaimtreatynonprop.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Equal(t, []inboxclaimtreatynonprop.Violation{
		{Field: inboxclaimtreatynonprop.FieldTab, Message: "Tab tidak dikenal."},
	}, validation.Violations)

	_, found := inboxclaimtreatynonprop.FindTab("99")
	require.False(t, found)
}

// Tab kosong jatuh ke tab bawaan, dan login dipangkas spasinya.
func TestNewQueryDefaultTabAndCleanCaller(t *testing.T) {
	q, err := inboxclaimtreatynonprop.NewQuery(
		inboxclaimtreatynonprop.QueryInput{Tab: "  ", SeeAll: true, TBAOnly: true},
		inboxclaimtreatynonprop.Caller{Login: "  ADMIN  "},
	)
	require.NoError(t, err)
	require.Equal(t, inboxclaimtreatynonprop.DefaultTab, q.Tab.Code)
	require.Equal(t, "ADMIN", q.Caller.Login)
	require.True(t, q.SeeAll)
	require.True(t, q.TBAOnly)
	require.False(t, q.ScopedToCaller(), "See All melepas penyaring kepemilikan")
}

// Tabs mengembalikan salinan: menulisi hasilnya tidak mengubah daftar tab modul.
func TestTabsReturnsCopy(t *testing.T) {
	first := inboxclaimtreatynonprop.Tabs()
	first[0].Name = "DIUBAH"

	again := inboxclaimtreatynonprop.Tabs()
	require.Equal(t, "Treaty-In Admin", again[0].Name)
	require.Len(t, again, 3)
}
