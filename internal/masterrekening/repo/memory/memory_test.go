package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterrekening/repo/memory"
)

var base = time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

func sample() []masterrekening.Account {
	return []masterrekening.Account{
		{Number: "111", BankCode: "014", OwnerName: "PT Satu", BankName: "BANK BCA",
			Status: masterrekening.StatusPending, CommitteeApproval: "Budi", CreatedAt: base},
		{Number: "222", BankCode: "002", OwnerName: "PT Dua", BankName: "BANK BRI",
			Status: masterrekening.StatusApproved, CreatedAt: base.Add(time.Hour)},
		{Number: "111", BankCode: "002", OwnerName: "PT Satu", BankName: "BANK BRI",
			Status: masterrekening.StatusRejected, CreatedAt: base},
	}
}

func numbers(rows []masterrekening.Account) []string {
	out := []string{}
	for _, r := range rows {
		out = append(out, r.Number+"/"+r.BankCode)
	}
	return out
}

func TestListSortsNewestFirstAndFilters(t *testing.T) {
	repo := memory.NewRepo(sample()...)
	ctx := context.Background()

	rows, total, err := repo.List(ctx, masterrekening.Filter{})
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, "222/002", numbers(rows)[0], "terbaru lebih dulu")
	// Tanggal sama diurutkan menurut nomor; keduanya bernomor 111.
	require.ElementsMatch(t, []string{"111/014", "111/002"}, numbers(rows)[1:])

	rows, total, err = repo.List(ctx, masterrekening.Filter{Status: masterrekening.StatusApproved})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, []string{"222/002"}, numbers(rows))

	rows, _, _ = repo.List(ctx, masterrekening.Filter{Number: " 11 ", BankName: "bri"})
	require.Equal(t, []string{"111/002"}, numbers(rows))

	rows, _, _ = repo.List(ctx, masterrekening.Filter{OwnerName: "dua"})
	require.Equal(t, []string{"222/002"}, numbers(rows))

	rows, _, _ = repo.List(ctx, masterrekening.Filter{OwnerName: "tidak ada"})
	require.Empty(t, rows)

	rows, _, _ = repo.List(ctx, masterrekening.Filter{MyCommitteeOnly: true, CommitteeIdentity: " budi "})
	require.Equal(t, []string{"111/014"}, numbers(rows))

	// Tanpa identitas komite, penyaring "milik saya" diabaikan.
	_, total, _ = repo.List(ctx, masterrekening.Filter{MyCommitteeOnly: true})
	require.Equal(t, 3, total)
}

func TestListPaginates(t *testing.T) {
	repo := memory.NewRepo(sample()...)
	ctx := context.Background()

	rows, total, _ := repo.List(ctx, masterrekening.Filter{Limit: 1, Offset: -5})
	require.Equal(t, 3, total)
	require.Equal(t, []string{"222/002"}, numbers(rows))

	rows, _, _ = repo.List(ctx, masterrekening.Filter{Limit: 5, Offset: 1})
	require.Len(t, rows, 2)

	rows, total, _ = repo.List(ctx, masterrekening.Filter{Offset: 3})
	require.Equal(t, 3, total)
	require.Empty(t, rows)
}

func TestGetFindSaveUpdate(t *testing.T) {
	repo := memory.NewRepo(sample()...)
	ctx := context.Background()

	got, err := repo.Get(ctx, masterrekening.Key{Number: "222", BankCode: "002"})
	require.NoError(t, err)
	require.Equal(t, "PT Dua", got.OwnerName)

	_, err = repo.Get(ctx, masterrekening.Key{Number: "999"})
	require.ErrorIs(t, err, masterrekening.ErrNotFound)

	found, err := repo.FindByNumber(ctx, " 111 ")
	require.NoError(t, err)
	require.Equal(t, []string{"111/002", "111/014"}, numbers(found), "urut menurut kode bank")

	err = repo.Save(ctx, masterrekening.Account{Number: "222", BankCode: "002"})
	require.ErrorIs(t, err, masterrekening.ErrAlreadyExists)
	require.NoError(t, repo.Save(ctx, masterrekening.Account{Number: "333", BankCode: "009"}))

	err = repo.Update(ctx, masterrekening.Account{Number: "444", BankCode: "009"})
	require.ErrorIs(t, err, masterrekening.ErrNotFound)
	require.NoError(t, repo.Update(ctx, masterrekening.Account{
		Number: "333", BankCode: "009", OwnerName: "Baru",
	}))
	got, _ = repo.Get(ctx, masterrekening.Key{Number: "333", BankCode: "009"})
	require.Equal(t, "Baru", got.OwnerName)
}

func TestClearRejectedOnlyRemovesRejectedAccounts(t *testing.T) {
	repo := memory.NewRepo(sample()...)
	ctx := context.Background()

	err := repo.ClearRejected(ctx, masterrekening.Key{Number: "999"})
	require.ErrorIs(t, err, masterrekening.ErrNotFound)

	err = repo.ClearRejected(ctx, masterrekening.Key{Number: "222", BankCode: "002"})
	require.ErrorIs(t, err, masterrekening.ErrAlreadyDecided)

	require.NoError(t, repo.ClearRejected(ctx, masterrekening.Key{Number: "111", BankCode: "002"}))
	_, err = repo.Get(ctx, masterrekening.Key{Number: "111", BankCode: "002"})
	require.ErrorIs(t, err, masterrekening.ErrNotFound)
}

func TestBankRepoReturnsACopy(t *testing.T) {
	banks := memory.SampleBanks()
	require.NotEmpty(t, banks)

	repo := memory.NewBankRepo(banks...)
	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, banks, list)

	list[0].Name = "DIUBAH"
	again, _ := repo.List(context.Background())
	require.Equal(t, banks[0].Name, again[0].Name, "salinan, bukan rujukan")
}
