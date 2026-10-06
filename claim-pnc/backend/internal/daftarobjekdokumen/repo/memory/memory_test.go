package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftarobjekdokumen"
)

var errBoom = errors.New("boom")

// List mengurutkan menurut ID sebagai TEKS dan tidak membawa pemetaan bisnis.
func TestListSortsByIDAsTextWithoutBusinesses(t *testing.T) {
	repo := NewRepo(
		daftarobjekdokumen.DocumentObject{ID: "1009", Description: "b"},
		daftarobjekdokumen.DocumentObject{ID: "10010", Description: "a", OldID: "0007",
			Businesses: []daftarobjekdokumen.Business{{ID: "1", Name: "X"}}},
	)
	got, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftarobjekdokumen.DocumentObject{
		{ID: "10010", Description: "a", OldID: "0007"},
		{ID: "1009", Description: "b"},
	}, got)
}

func TestGetReturnsCopyWithBusinesses(t *testing.T) {
	repo := NewRepo(SampleList()...)
	got, err := repo.Get(context.Background(), " 100002 ")
	require.NoError(t, err)
	require.Equal(t, "Polis Asli", got.Description)
	require.Len(t, got.Businesses, 2)

	// Mengubah hasil tidak boleh mengubah isi tersimpan.
	got.Businesses[0].Name = "DIUBAH"
	again, err := repo.Get(context.Background(), "100002")
	require.NoError(t, err)
	require.Equal(t, "FIRE / PROPERTY", again.Businesses[0].Name)

	_, err = repo.Get(context.Background(), "999999")
	require.ErrorIs(t, err, daftarobjekdokumen.ErrNotFound)
}

// ID baru melanjutkan deret tertinggi, dengan kode situs "1" dan empat digit.
func TestInsertContinuesSequence(t *testing.T) {
	repo := NewRepo(SampleList()...)
	saved, err := repo.Insert(context.Background(), daftarobjekdokumen.SaveData{
		Description: "Baru",
		Businesses:  []daftarobjekdokumen.Business{{ID: "003", Name: "ANEKA"}},
	})
	require.NoError(t, err)
	require.Equal(t, "100005", saved.ID)
	require.Equal(t, []daftarobjekdokumen.Business{{ID: "003", Name: "ANEKA"}}, saved.Businesses)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 5)
}

// Pada repo kosong nomor pertama adalah 10001; ID berbentuk lain dilewati.
func TestInsertOnEmptyAndOddIDs(t *testing.T) {
	saved, err := NewRepo().Insert(context.Background(), daftarobjekdokumen.SaveData{Description: "a"})
	require.NoError(t, err)
	require.Equal(t, "100001", saved.ID)

	// "1" terlalu pendek, "1abc" bukan angka — keduanya tidak memengaruhi deret.
	repo := NewRepo(
		daftarobjekdokumen.DocumentObject{ID: "1"},
		daftarobjekdokumen.DocumentObject{ID: "1abc"},
	)
	saved, err = repo.Insert(context.Background(), daftarobjekdokumen.SaveData{Description: "b"})
	require.NoError(t, err)
	require.Equal(t, "100001", saved.ID)
}

// Nomor urut dibaca dari ID tanpa digit pertama, apa pun digit situsnya; di atas 99999 tidak
// dipotong, sama seperti LPAD.
func TestNextIDReadsSequenceAfterSiteDigit(t *testing.T) {
	repo := NewRepo(daftarobjekdokumen.DocumentObject{ID: "200041"})
	require.Equal(t, "100042", repo.nextID())
	require.Equal(t, "00007", fiveDigits(7))
	require.Equal(t, "100000", fiveDigits(100000))
}

func TestUpdateReplacesDescriptionAndBusinesses(t *testing.T) {
	repo := NewRepo(SampleList()...)
	saved, err := repo.Update(context.Background(), "100003", daftarobjekdokumen.SaveData{
		Description: "Ubah",
		Businesses:  []daftarobjekdokumen.Business{{ID: "10045", Name: "TRAVEL"}},
	})
	require.NoError(t, err)
	// ID dan OldID tidak ikut ditimpa.
	require.Equal(t, daftarobjekdokumen.DocumentObject{
		ID: "100003", Description: "Ubah", OldID: "0007",
		Businesses: []daftarobjekdokumen.Business{{ID: "10045", Name: "TRAVEL"}},
	}, saved)

	_, err = repo.Update(context.Background(), "nope", daftarobjekdokumen.SaveData{})
	require.ErrorIs(t, err, daftarobjekdokumen.ErrNotFound)
}

func TestRepoSetErrorFailsEveryOperation(t *testing.T) {
	repo := NewRepo(SampleList()...)
	repo.SetError(errBoom)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errBoom)
	_, err = repo.Get(ctx, "10001")
	require.ErrorIs(t, err, errBoom)
	_, err = repo.Insert(ctx, daftarobjekdokumen.SaveData{})
	require.ErrorIs(t, err, errBoom)
	_, err = repo.Update(ctx, "10001", daftarobjekdokumen.SaveData{})
	require.ErrorIs(t, err, errBoom)
}

func TestBusinessRepoListSortsByName(t *testing.T) {
	repo := NewBusinessRepo(SampleBusinessList()...)
	got, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "ANEKA", got[0].Name)
	require.Equal(t, "TRAVEL", got[len(got)-1].Name)
	require.Len(t, got, 5)

	repo.SetError(errBoom)
	_, err = repo.List(context.Background())
	require.ErrorIs(t, err, errBoom)
}
