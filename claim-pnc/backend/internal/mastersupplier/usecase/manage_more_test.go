package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersupplier/repo/memory"
	"claim-pnc/internal/mastersupplier/usecase"
	"claim-pnc/internal/platform/clock"
)

// stubStore membungkus repo memori dan dapat menimpa penerbitan ID serta permintaan
// persetujuan, untuk keadaan yang tidak dapat dibentuk repo memori sendiri.
type stubStore struct {
	*memory.Repo
	nextID      string
	nextIDError error
	approvalErr error
	codes       *mastersupplier.CodeSet
}

func (s *stubStore) ListCodes(ctx context.Context) (mastersupplier.CodeSet, error) {
	if s.codes != nil {
		return *s.codes, nil
	}
	return s.Repo.ListCodes(ctx)
}

// Sandi tersimpan tanpa label diberi label sandinya sendiri; sandi kosong dan kembar dibuang,
// dan label yang terbukti tidak ditimpa.
func TestListCodesFillsMissingLabelAndDropsBlankAndDuplicate(t *testing.T) {
	store := &stubStore{Repo: memory.NewRepo(memory.Options{}), codes: &mastersupplier.CodeSet{
		PartnerStatus: []mastersupplier.CodeOption{{Value: " 7 ", Label: " "}, {Value: "", Label: "kosong"}},
		Active:        []mastersupplier.CodeOption{{Value: "1", Label: "mentah"}},
	}}
	set, err := buildWithStore(t, store).ListCodes(context.Background(), portalAlias)
	require.NoError(t, err)
	require.Equal(t, []mastersupplier.CodeOption{{Value: "7", Label: "7"}}, set.PartnerStatus)
	require.Equal(t, []mastersupplier.CodeOption{
		{Value: "1", Label: "Aktif"}, {Value: "0", Label: "Tidak aktif"},
	}, set.Active)
}

func (s *stubStore) NextID(ctx context.Context) (string, error) {
	if s.nextIDError != nil || s.nextID != "" {
		return s.nextID, s.nextIDError
	}
	return s.Repo.NextID(ctx)
}

func (s *stubStore) RequestApproval(ctx context.Context, r mastersupplier.ApprovalRequest) error {
	if s.approvalErr != nil {
		return s.approvalErr
	}
	return s.Repo.RequestApproval(ctx, r)
}

func buildWithStore(t *testing.T, store mastersupplier.Store) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (mastersupplier.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal")
			}
			return store, nil
		},
		Clock: clock.FixedAt(fixedNow),
	})
	require.NoError(t, err)
	return service
}

// Setiap operasi menolak portal yang tidak dikenal.
func TestEveryOperationRejectsUnknownPortal(t *testing.T) {
	service, _ := build(t)
	ctx := context.Background()

	_, err := service.Get(ctx, "entah", "x")
	require.ErrorContains(t, err, "portal tidak dikenal")
	_, err = service.Save(ctx, "entah", "x", validInput(), actor(), nil)
	require.ErrorContains(t, err, "portal tidak dikenal")
	_, err = service.ListBranches(ctx, "entah")
	require.ErrorContains(t, err, "portal tidak dikenal")
	_, err = service.SearchCities(ctx, "entah", "jakarta")
	require.ErrorContains(t, err, "portal tidak dikenal")
	_, err = service.ListCountries(ctx, "entah")
	require.ErrorContains(t, err, "portal tidak dikenal")
	_, err = service.ListBanks(ctx, "entah")
	require.ErrorContains(t, err, "portal tidak dikenal")
	_, err = service.ListCodes(ctx, "entah")
	require.ErrorContains(t, err, "portal tidak dikenal")
}

// ID kosong pada penyimpanan dijawab "tidak ditemukan"; isian tidak sah ditolak sebelum dibaca.
func TestSaveRejectsBlankIDAndInvalidInput(t *testing.T) {
	service, _ := build(t)
	_, err := service.Save(context.Background(), portalAlias, "  ", validInput(), actor(), nil)
	require.ErrorIs(t, err, mastersupplier.ErrNotFound)

	_, err = service.Save(context.Background(), portalAlias, "0100000000001", mastersupplier.Input{}, actor(), nil)
	var validation *mastersupplier.ValidationError
	require.ErrorAs(t, err, &validation)
}

// Galat penyimpanan dan pembacaan sandi diteruskan apa adanya.
func TestSaveAndListCodesForwardStorageFailure(t *testing.T) {
	service, repo := build(t)
	stored, err := service.Get(context.Background(), portalAlias, "0100000000001")
	require.NoError(t, err)

	boom := errors.New("basis data mati")
	repo.SetError(boom)

	input := validInput()
	input.Name = stored.Name
	_, err = service.Save(context.Background(), portalAlias, "0100000000001", input, actor(), nil)
	require.ErrorIs(t, err, boom)
	_, err = service.ListCodes(context.Background(), portalAlias)
	require.ErrorIs(t, err, boom)
}

// Galat penerbitan ID dibungkus, dan ID kosong ditolak terang-terangan.
func TestCreateRejectsFailedOrBlankIdentifier(t *testing.T) {
	store := &stubStore{Repo: memory.NewSampleRepo(), nextIDError: errors.New("sequence mati")}
	_, err := buildWithStore(t, store).Create(context.Background(), portalAlias, validInput(), actor(), nil)
	require.ErrorContains(t, err, "menerbitkan ID supplier")
	require.ErrorContains(t, err, "sequence mati")

	blank := &stubStore{Repo: memory.NewSampleRepo(), nextID: "   "}
	_, err = buildWithStore(t, blank).Create(context.Background(), portalAlias, validInput(), actor(), nil)
	require.ErrorContains(t, err, "ID supplier yang diterbitkan kosong")
}

// Bila antrean persetujuan gagal, pesannya membedakan bahwa supplier sudah tersimpan.
func TestApprovalFailureIsDistinguished(t *testing.T) {
	store := &stubStore{Repo: memory.NewSampleRepo(), approvalErr: errors.New("antrean mati")}
	service := buildWithStore(t, store)

	_, err := service.Create(context.Background(), portalAlias, validInput(), actor(), nil)
	require.ErrorContains(t, err, "tersimpan tetapi permintaan persetujuannya gagal")

	stored, err := service.Get(context.Background(), portalAlias, "0100000000001")
	require.NoError(t, err)
	input := validInput()
	input.Name = stored.Name
	_, err = service.Save(context.Background(), portalAlias, stored.ID, input, actor(), nil)
	require.ErrorContains(t, err, "tersimpan tetapi permintaan persetujuannya gagal")
}

// Penyimpanan yang berhasil dicatat beserta penanda persetujuannya.
func TestSavedSupplierIsLogged(t *testing.T) {
	service, _ := build(t)
	buffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buffer, nil))

	saved, err := service.Create(context.Background(), portalAlias, validInput(), actor(), logger)
	require.NoError(t, err)
	require.Contains(t, buffer.String(), `"peristiwa":"supplier baru ditambahkan"`)
	require.Contains(t, buffer.String(), `"id_supplier":"`+saved.ID+`"`)
	require.Contains(t, buffer.String(), `"persetujuan_diminta":true`)
	require.Contains(t, buffer.String(), `"oleh":"PETUGAS.UJI"`)
}
