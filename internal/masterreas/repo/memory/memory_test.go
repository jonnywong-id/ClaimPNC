package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterreas"
)

// Urutannya meniru ORDER BY REINSURERNAME, TYPE, REINSURERID pada adapter SQL.
func TestListSortsByNameTypeAndID(t *testing.T) {
	r := NewRepo(Options{Rows: []masterreas.Member{
		{ReinsurerID: "B", ReinsurerName: "Zeta", Type: "1"},
		{ReinsurerID: "C", ReinsurerName: "Alfa", Type: "2"},
		{ReinsurerID: "B", ReinsurerName: "Alfa", Type: "1"},
		{ReinsurerID: "A", ReinsurerName: "Alfa", Type: "1"},
	}})

	got, err := r.List(context.Background(), masterreas.Filter{})
	require.NoError(t, err)
	var order []string
	for _, one := range got {
		order = append(order, one.ReinsurerName+"/"+one.Type+"/"+one.ReinsurerID)
	}
	require.Equal(t, []string{"Alfa/1/A", "Alfa/1/B", "Alfa/2/C", "Zeta/1/B"}, order)
}

// Kata kunci dicari pada keempat kolom tanpa memandang besar-kecil huruf; Country tidak ikut.
func TestListMatchesKeywordOnFourColumns(t *testing.T) {
	r := NewSampleRepo()
	ctx := context.Background()

	byName, err := r.List(ctx, masterreas.Filter{Keyword: " andalas "})
	require.NoError(t, err)
	require.Len(t, byName, 2)

	byID, err := r.List(ctx, masterreas.Filter{Keyword: "re-004"})
	require.NoError(t, err)
	require.Len(t, byID, 1)
	require.Equal(t, "Cakrawala Re Asia", byID[0].ReinsurerName)

	byEmail, err := r.List(ctx, masterreas.Filter{Keyword: "XOL.NUSANTARA"})
	require.NoError(t, err)
	require.Len(t, byEmail, 1)
	require.Equal(t, "3", byEmail[0].Type)

	byLogin, err := r.List(ctx, masterreas.Filter{Keyword: "bahterareinsuranceltd"})
	require.NoError(t, err)
	require.Len(t, byLogin, 1)

	byCountry, err := r.List(ctx, masterreas.Filter{Keyword: "Singapura"})
	require.NoError(t, err)
	require.Empty(t, byCountry, "Country tidak termasuk kolom yang dicari")

	all, err := r.List(ctx, masterreas.Filter{})
	require.NoError(t, err)
	require.Len(t, all, len(SampleList()))
}

// NewRepo menyalin baris, sehingga mengubah slice asal tidak mengubah isi repo.
func TestNewRepoCopiesRows(t *testing.T) {
	rows := []masterreas.Member{{ReinsurerID: "A", ReinsurerName: "Alfa"}}
	r := NewRepo(Options{Rows: rows})
	rows[0].ReinsurerName = "Diubah"

	got, err := r.List(context.Background(), masterreas.Filter{})
	require.NoError(t, err)
	require.Equal(t, "Alfa", got[0].ReinsurerName)
}

func TestSetErrorFailsList(t *testing.T) {
	r := NewSampleRepo()
	boom := errors.New("boom")
	r.SetError(boom)

	_, err := r.List(context.Background(), masterreas.Filter{})
	require.ErrorIs(t, err, boom)
}
