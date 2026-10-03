package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	mgs "claim-pnc/internal/mastergroupingsparepart"
)

var ctx = context.Background()

func TestListFiltersByStatusAndKeywordAndSortsByID(t *testing.T) {
	r := NewSampleRepo()

	approved, err := r.List(ctx, mgs.Filter{Status: mgs.StatusApproved})
	require.NoError(t, err)
	require.Len(t, approved, 2)
	require.Equal(t, "1", approved[0].ID)
	require.Equal(t, "2", approved[1].ID)

	// Kata kunci mencocoki nama panel tanpa membedakan besar-kecil huruf.
	byPanel, err := r.List(ctx, mgs.Filter{Status: mgs.StatusApproved, Keyword: " bumper "})
	require.NoError(t, err)
	require.Len(t, byPanel, 1)
	require.Equal(t, "2", byPanel[0].ID)

	none, err := r.List(ctx, mgs.Filter{Status: mgs.StatusApproved, Keyword: "TIDAKADA"})
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestGetAndFindByKey(t *testing.T) {
	r := NewSampleRepo()

	g, err := r.Get(ctx, " 3 ")
	require.NoError(t, err)
	require.Equal(t, "SP-1003", g.PartNumber)

	_, err = r.Get(ctx, "99")
	require.ErrorIs(t, err, mgs.ErrNotFound)

	found, err := r.FindByKey(ctx, mgs.NaturalKey{
		PartNumber: " sp-1001", PanelName: "kabin", ChassisNumber: "mhfxw1234k5678901", PanelSide: "1",
	})
	require.NoError(t, err)
	require.Equal(t, "1", found.ID)

	_, err = r.FindByKey(ctx, mgs.NaturalKey{})
	require.ErrorIs(t, err, mgs.ErrNotFound)

	_, err = r.FindByKey(ctx, mgs.NaturalKey{PartNumber: "X"})
	require.ErrorIs(t, err, mgs.ErrNotFound)
}

func TestFindGroupByChassisPicksSmallestGroup(t *testing.T) {
	r := NewRepo(Options{Rows: []mgs.Grouping{
		{ID: "1", ChassisNumber: "RK", GroupNumber: "0005"},
		{ID: "2", ChassisNumber: "rk ", GroupNumber: "0002"},
		{ID: "3", ChassisNumber: "RK", GroupNumber: " "},
		{ID: "4", ChassisNumber: "LAIN", GroupNumber: "0001"},
	}})

	got, err := r.FindGroupByChassis(ctx, " rk")
	require.NoError(t, err)
	require.Equal(t, "0002", got)

	_, err = r.FindGroupByChassis(ctx, " ")
	require.ErrorIs(t, err, mgs.ErrGroupChassisNotFound)

	_, err = r.FindGroupByChassis(ctx, "TIDAK")
	require.ErrorIs(t, err, mgs.ErrGroupChassisNotFound)
}

// Daftar acuan dikembalikan sebagai salinan; mengubahnya tidak mengubah isi repo.
func TestLookupsReturnCopies(t *testing.T) {
	r := NewSampleRepo()

	panels, err := r.ListPanels(ctx)
	require.NoError(t, err)
	require.Equal(t, SamplePanels(), panels)
	panels[0].Name = "DIUBAH"
	again, _ := r.ListPanels(ctx)
	require.Equal(t, "BUMPER DEPAN", again[0].Name)

	sides, err := r.ListSides(ctx, mgs.SideKey{PanelID: " PNL02 "})
	require.NoError(t, err)
	require.Equal(t, []mgs.Side{mgs.SideNone, mgs.SideLeft, mgs.SideRight}, sides)
	sides[0] = "X"
	again2, _ := r.ListSides(ctx, mgs.SideKey{PanelID: "PNL02"})
	require.Equal(t, mgs.SideNone, again2[0])

	unknown, err := r.ListSides(ctx, mgs.SideKey{PanelID: "TIDAK"})
	require.NoError(t, err)
	require.Empty(t, unknown)

	types, err := r.ListVehicleTypes(ctx)
	require.NoError(t, err)
	require.Equal(t, SampleVehicleTypes(), types)
}

func TestFindPart(t *testing.T) {
	r := NewSampleRepo()

	p, err := r.FindPart(ctx, " sp-1002 ")
	require.NoError(t, err)
	require.Equal(t, "SEAL KIT BOOM", p.Name)

	_, err = r.FindPart(ctx, "")
	require.ErrorIs(t, err, mgs.ErrPartNotFound)

	_, err = r.FindPart(ctx, "SP-9999")
	require.ErrorIs(t, err, mgs.ErrPartNotFound)
}

func TestInsertRejectsDuplicateKey(t *testing.T) {
	r := NewSampleRepo()

	dup := SampleList()[0]
	dup.ID = "9"
	dup.PartNumber = " sp-1001 "
	require.ErrorIs(t, r.Insert(ctx, dup), mgs.ErrDuplicate)

	// Kunci kosong tidak diperiksa keunikannya.
	require.NoError(t, r.Insert(ctx, mgs.Grouping{ID: "10"}))
	require.NoError(t, r.Insert(ctx, mgs.Grouping{ID: "11"}))

	fresh := mgs.Grouping{ID: "12", PartNumber: "SP-2", Status: mgs.StatusPending}
	require.NoError(t, r.Insert(ctx, fresh))
	got, err := r.Get(ctx, "12")
	require.NoError(t, err)
	require.Equal(t, fresh, got)
}

// Update tidak memindahkan ID baris yang disimpan.
func TestUpdateKeepsStoredID(t *testing.T) {
	r := NewRepo(Options{Rows: []mgs.Grouping{{ID: "1", Note: "lama"}}})

	require.NoError(t, r.Update(ctx, mgs.Grouping{ID: " 1 ", Note: "baru"}))
	got, err := r.Get(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, "1", got.ID)
	require.Equal(t, "baru", got.Note)

	require.ErrorIs(t, r.Update(ctx, mgs.Grouping{ID: "2"}), mgs.ErrNotFound)
}

func TestSetStatusCountsOnlyChangedRows(t *testing.T) {
	r := NewSampleRepo()

	n, err := r.SetStatus(ctx, []string{" 1 ", "3", "4", "99"}, mgs.StatusApproved)
	require.NoError(t, err)
	// Baris "1" sudah disetujui sehingga tidak dihitung.
	require.Equal(t, 2, n)

	g, _ := r.Get(ctx, "3")
	require.Equal(t, mgs.StatusApproved, g.Status)
}

func TestNextIDAndNextGroupNumber(t *testing.T) {
	r := NewSampleRepo()

	id, err := r.NextID(ctx)
	require.NoError(t, err)
	require.Equal(t, "5", id)
	id, _ = r.NextID(ctx)
	require.Equal(t, "6", id)

	group, err := r.NextGroupNumber(ctx)
	require.NoError(t, err)
	require.Equal(t, mgs.ComposeGroupNumber(4), group)

	// Nilai yang tidak dapat diurai dilewati; perbandingannya secara angka.
	r2 := NewRepo(Options{Rows: []mgs.Grouping{
		{GroupNumber: "0009"}, {GroupNumber: "10"}, {GroupNumber: "RUSAK"},
	}})
	group, err = r2.NextGroupNumber(ctx)
	require.NoError(t, err)
	require.Equal(t, mgs.ComposeGroupNumber(11), group)
}

// NewRepo menyalin isi Options, sehingga mengubah slice asal tidak mengubah repo.
func TestNewRepoCopiesOptions(t *testing.T) {
	sides := map[string][]mgs.Side{"P": {mgs.SideLeft}}
	rows := []mgs.Grouping{{ID: "1"}}
	r := NewRepo(Options{Rows: rows, Side: sides})
	sides["P"][0] = mgs.SideRight
	rows[0].ID = "X"

	got, err := r.ListSides(ctx, mgs.SideKey{PanelID: "P"})
	require.NoError(t, err)
	require.Equal(t, []mgs.Side{mgs.SideLeft}, got)
	_, err = r.Get(ctx, "1")
	require.NoError(t, err)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	boom := errors.New("boom")
	r := NewSampleRepo()
	r.SetError(boom)

	_, err := r.List(ctx, mgs.Filter{})
	require.ErrorIs(t, err, boom)
	_, err = r.Get(ctx, "1")
	require.ErrorIs(t, err, boom)
	_, err = r.FindByKey(ctx, mgs.NaturalKey{PartNumber: "X"})
	require.ErrorIs(t, err, boom)
	_, err = r.FindGroupByChassis(ctx, "X")
	require.ErrorIs(t, err, boom)
	_, err = r.ListPanels(ctx)
	require.ErrorIs(t, err, boom)
	_, err = r.ListSides(ctx, mgs.SideKey{})
	require.ErrorIs(t, err, boom)
	_, err = r.ListVehicleTypes(ctx)
	require.ErrorIs(t, err, boom)
	_, err = r.FindPart(ctx, "X")
	require.ErrorIs(t, err, boom)
	require.ErrorIs(t, r.Insert(ctx, mgs.Grouping{}), boom)
	require.ErrorIs(t, r.Update(ctx, mgs.Grouping{}), boom)
	_, err = r.SetStatus(ctx, []string{"1"}, mgs.StatusApproved)
	require.ErrorIs(t, err, boom)
	_, err = r.NextID(ctx)
	require.ErrorIs(t, err, boom)
	_, err = r.NextGroupNumber(ctx)
	require.ErrorIs(t, err, boom)
}
