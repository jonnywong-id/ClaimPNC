package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastercolsimasonline"
)

// Daftar diurutkan sebagai TEKS dan tidak membawa pemetaan bisnis.
func TestListSortsCodesAsTextWithoutTheBusinessMapping(t *testing.T) {
	repo := NewRepo(
		mastercolsimasonline.CauseOfLoss{Code: "109", Description: "B",
			Businesses: []mastercolsimasonline.Business{{ID: "003", Name: "ANEKA"}}},
		mastercolsimasonline.CauseOfLoss{Code: "1010", Description: "A"},
	)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []mastercolsimasonline.CauseOfLoss{
		{Code: "1010", Description: "A"},
		{Code: "109", Description: "B"},
	}, list)
}

// Get memangkas kode yang dicari dan menyalin pemetaan bisnisnya.
func TestGetReturnsACopyOfTheRow(t *testing.T) {
	repo := NewRepo(SampleList()...)

	row, err := repo.Get(context.Background(), " 1001 ")
	require.NoError(t, err)
	require.Equal(t, "KEBAKARAN", row.Description)
	require.Len(t, row.Businesses, 2)

	// Mengubah hasilnya tidak mengubah isi yang tersimpan.
	row.Businesses[0].Name = "DIUBAH"
	again, err := repo.Get(context.Background(), "1001")
	require.NoError(t, err)
	require.Equal(t, "FIRE / PROPERTY", again.Businesses[0].Name)

	_, err = repo.Get(context.Background(), "9999")
	require.ErrorIs(t, err, mastercolsimasonline.ErrNotFound)
}

// Kode baru melanjutkan deret yang tersimpan: situs "1" + tiga digit.
func TestInsertContinuesTheStoredSequence(t *testing.T) {
	repo := NewRepo(SampleList()...)

	saved, err := repo.Insert(context.Background(), mastercolsimasonline.SaveData{
		Description: "BANJIR",
		Businesses:  []mastercolsimasonline.Business{{ID: "006", Name: "FIRE / PROPERTY"}},
	})
	require.NoError(t, err)
	require.Equal(t, "1005", saved.Code)

	stored, err := repo.Get(context.Background(), "1005")
	require.NoError(t, err)
	require.Equal(t, "BANJIR", stored.Description)
	require.Equal(t, []mastercolsimasonline.Business{{ID: "006", Name: "FIRE / PROPERTY"}},
		stored.Businesses)
}

// Kode satu karakter dan kode yang bukan angka tidak ikut menentukan nomor urut.
func TestInsertIgnoresCodesWithoutASequenceNumber(t *testing.T) {
	repo := NewRepo(
		mastercolsimasonline.CauseOfLoss{Code: "9"},
		mastercolsimasonline.CauseOfLoss{Code: "1ABC"},
	)

	saved, err := repo.Insert(context.Background(), mastercolsimasonline.SaveData{})
	require.NoError(t, err)
	require.Equal(t, "1001", saved.Code)
}

// Penyuntingan mengganti deskripsi dan seluruh pemetaan, tidak menggabungkannya.
func TestUpdateReplacesTheBusinessMapping(t *testing.T) {
	repo := NewRepo(SampleList()...)

	saved, err := repo.Update(context.Background(), " 1001", mastercolsimasonline.SaveData{
		Description: "KEBAKARAN BESAR",
		Businesses:  []mastercolsimasonline.Business{{Name: "BEBAS"}},
	})
	require.NoError(t, err)
	require.Equal(t, mastercolsimasonline.CauseOfLoss{
		Code: "1001", Description: "KEBAKARAN BESAR",
		Businesses: []mastercolsimasonline.Business{{Name: "BEBAS"}},
	}, saved)

	_, err = repo.Update(context.Background(), "9999", mastercolsimasonline.SaveData{})
	require.ErrorIs(t, err, mastercolsimasonline.ErrNotFound)
}

// Galat yang dipasang dijawab oleh setiap operasi.
func TestSetErrorFailsEveryOperation(t *testing.T) {
	failure := errors.New("oracle mati")
	repo := NewRepo(SampleList()...)
	repo.SetError(failure)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, failure)
	_, err = repo.Get(ctx, "1001")
	require.ErrorIs(t, err, failure)
	_, err = repo.Insert(ctx, mastercolsimasonline.SaveData{})
	require.ErrorIs(t, err, failure)
	_, err = repo.Update(ctx, "1001", mastercolsimasonline.SaveData{})
	require.ErrorIs(t, err, failure)
}

// Master bisnis diurutkan menurut nama, dan galatnya dapat dipasang.
func TestBusinessRepoListsByName(t *testing.T) {
	business := NewBusinessRepo(SampleBusinessList()...)

	list, err := business.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "ANEKA", list[0].Name)
	require.Equal(t, "TRAVEL", list[len(list)-1].Name)

	failure := errors.New("gisfw mati")
	business.SetError(failure)
	_, err = business.List(context.Background())
	require.ErrorIs(t, err, failure)
}

// Nama bisnis dicari menurut ID tanpa memandang huruf besar-kecil maupun spasi tepi.
func TestBusinessNameIsFoundByItsID(t *testing.T) {
	business := NewBusinessRepo(mastercolsimasonline.Business{ID: "abc", Name: "Lini ABC"})
	repo := NewRepo().WithBusiness(business)
	require.Same(t, business, repo.business)

	name, found := business.nameOf(" ABC ")
	require.True(t, found)
	require.Equal(t, "Lini ABC", name)

	_, found = business.nameOf("xyz")
	require.False(t, found)
}

// Bilangan di atas 999 dibiarkan apa adanya, sama seperti LPAD Oracle.
func TestThreeDigitsPadsButNeverTruncates(t *testing.T) {
	require.Equal(t, "007", threeDigits(7))
	require.Equal(t, "1000", threeDigits(1000))
}
