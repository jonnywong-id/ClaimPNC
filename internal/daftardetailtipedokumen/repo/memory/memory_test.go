package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetailtipedokumen"
	"claim-pnc/internal/daftardetailtipedokumen/repo/memory"
)

var errInjected = errors.New("galat sisipan")

func TestListSortsByIDAndFillsNames(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	repo.UseReferences(memory.NewSampleReferenceRepo())

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 5)
	require.Equal(t, "100001", list[0].ID)
	require.Equal(t, "100005", list[4].ID)
	require.Equal(t, "Dokumen Registrasi", list[0].DocumentTypeName)
	// Tipe dokumen tanpa master: namanya kosong.
	require.Equal(t, "", list[4].DocumentTypeName)
}

func TestGetWithoutReferencesLeavesNamesAsStored(t *testing.T) {
	repo := memory.NewRepo(daftardetailtipedokumen.DetailType{
		ID: " 100009 ", DocumentTypeName: " Tersimpan ",
	})
	got, err := repo.Get(context.Background(), "100009")
	require.NoError(t, err)
	require.Equal(t, "Tersimpan", got.DocumentTypeName)
}

func TestInsertNewIssuesNextIDAndSkipsTakenOnes(t *testing.T) {
	// "100002" sudah ada, "ABC" bukan ber-awalan situs, "10xx" tidak dapat dibaca angka.
	repo := memory.NewRepo(
		daftardetailtipedokumen.DetailType{ID: "100001"},
		daftardetailtipedokumen.DetailType{ID: "ABC"},
		daftardetailtipedokumen.DetailType{ID: "10xx"},
	)
	created, err := repo.InsertNew(context.Background(), daftardetailtipedokumen.Input{
		DocumentTypeID: "10001",
		Detail:         "Kwitansi",
	}, daftardetailtipedokumen.Editor{})
	require.NoError(t, err)
	require.Equal(t, "100002", created.ID)
}

func TestInsertNewContinuesFromHighestSequence(t *testing.T) {
	// Urutan diambil dari ID tertinggi, apa pun urutan barisnya saat dimuat.
	repo := memory.NewRepo(
		daftardetailtipedokumen.DetailType{ID: "100002"},
		daftardetailtipedokumen.DetailType{ID: "100001"},
	)
	created, err := repo.InsertNew(context.Background(), daftardetailtipedokumen.Input{}, daftardetailtipedokumen.Editor{})
	require.NoError(t, err)
	require.Equal(t, "100003", created.ID)
}

func TestUpdateReplacesRow(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	updated, err := repo.Update(context.Background(), " 100002 ", daftardetailtipedokumen.Input{Detail: "Baru"}, daftardetailtipedokumen.Editor{})
	require.NoError(t, err)
	require.Equal(t, "100002", updated.ID)
	require.Equal(t, "Baru", updated.Detail)

	_, err = repo.Update(context.Background(), "nope", daftardetailtipedokumen.Input{}, daftardetailtipedokumen.Editor{})
	require.ErrorIs(t, err, daftardetailtipedokumen.ErrNotFound)
}

func TestRepoSetErrorFailsEveryMethod(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	repo.SetError(errInjected)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errInjected)
	_, err = repo.Get(ctx, "100001")
	require.ErrorIs(t, err, errInjected)
	_, err = repo.InsertNew(ctx, daftardetailtipedokumen.Input{}, daftardetailtipedokumen.Editor{})
	require.ErrorIs(t, err, errInjected)
	_, err = repo.Update(ctx, "100001", daftardetailtipedokumen.Input{}, daftardetailtipedokumen.Editor{})
	require.ErrorIs(t, err, errInjected)
}

func TestReferenceRepoListIsSorted(t *testing.T) {
	repo := memory.NewSampleReferenceRepo()
	ctx := context.Background()

	documentTypes, err := repo.ListDocumentTypes(ctx)
	require.NoError(t, err)
	require.Equal(t, "Dokumen Komite", documentTypes[0].Name)

}

func TestReferenceRepoSetErrorFailsTheList(t *testing.T) {
	repo := memory.NewSampleReferenceRepo()
	repo.SetError(errInjected)
	ctx := context.Background()

	_, err := repo.ListDocumentTypes(ctx)
	require.ErrorIs(t, err, errInjected)
}
