package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/archivedokumenklaim"
	"claim-pnc/internal/archivedokumenklaim/gateway"
	"claim-pnc/internal/archivedokumenklaim/repo/memory"
	"claim-pnc/internal/archivedokumenklaim/usecase"
)

var errRepo = errors.New("repo rusak")

// brokenRepo menggagalkan operasi yang ditandai, sisanya diteruskan ke repo memori.
type brokenRepo struct {
	archivedokumenklaim.Repo
	fail map[string]bool
}

func (b *brokenRepo) Search(ctx context.Context, c archivedokumenklaim.Criteria,
	p archivedokumenklaim.Pagination) (archivedokumenklaim.ArchivePage, error) {
	if b.fail["Search"] {
		return archivedokumenklaim.ArchivePage{}, errRepo
	}
	return b.Repo.Search(ctx, c, p)
}

func (b *brokenRepo) SearchClaims(ctx context.Context,
	c archivedokumenklaim.ClaimCriteria) ([]archivedokumenklaim.ClaimCandidate, error) {
	if b.fail["SearchClaims"] {
		return nil, errRepo
	}
	return b.Repo.SearchClaims(ctx, c)
}

func (b *brokenRepo) Save(ctx context.Context, d archivedokumenklaim.Draft) (int64, error) {
	if b.fail["Save"] {
		return 0, errRepo
	}
	return b.Repo.Save(ctx, d)
}

func (b *brokenRepo) DocumentTypes(ctx context.Context) ([]archivedokumenklaim.DocumentTypeOption, error) {
	if b.fail["DocumentTypes"] {
		return nil, errRepo
	}
	return b.Repo.DocumentTypes(ctx)
}

func (b *brokenRepo) DocumentKinds(ctx context.Context) ([]archivedokumenklaim.DocumentKindOption, error) {
	if b.fail["DocumentKinds"] {
		return nil, errRepo
	}
	return b.Repo.DocumentKinds(ctx)
}

func (b *brokenRepo) FillingCodes(ctx context.Context,
	k string) ([]archivedokumenklaim.FillingCodeOption, error) {
	if b.fail["FillingCodes"] {
		return nil, errRepo
	}
	return b.Repo.FillingCodes(ctx, k)
}

func (b *brokenRepo) PendingBranch(ctx context.Context, s archivedokumenklaim.BranchScope,
	p archivedokumenklaim.Pagination) (archivedokumenklaim.ArchivePage, error) {
	if b.fail["PendingBranch"] {
		return archivedokumenklaim.ArchivePage{}, errRepo
	}
	return b.Repo.PendingBranch(ctx, s, p)
}

func (b *brokenRepo) FindByID(ctx context.Context, id int64) (archivedokumenklaim.ArchiveFile, bool, error) {
	if b.fail["FindByID"] {
		return archivedokumenklaim.ArchiveFile{}, false, errRepo
	}
	return b.Repo.FindByID(ctx, id)
}

func (b *brokenRepo) MarkSent(ctx context.Context, r archivedokumenklaim.Receipt) error {
	if b.fail["MarkSent"] {
		return errRepo
	}
	return b.Repo.MarkSent(ctx, r)
}

func (b *brokenRepo) StoreReceipt(ctx context.Context, r archivedokumenklaim.Receipt) error {
	if b.fail["StoreReceipt"] {
		return errRepo
	}
	return b.Repo.StoreReceipt(ctx, r)
}

// zeroTimeGateway menjawab tanpa waktu kirim, sehingga layanan memakai waktu permintaan.
type zeroTimeGateway struct{}

func (zeroTimeGateway) Send(_ context.Context, _ string,
	s archivedokumenklaim.Shipment) (archivedokumenklaim.Receipt, error) {
	return archivedokumenklaim.Receipt{ID: s.ID, Code: "201", Note: "masuk"}, nil
}

var errSelector = errors.New("portal belum siap")

func buildBroken(t *testing.T, fail ...string) (*usecase.Service, *brokenRepo) {
	t.Helper()
	repo := &brokenRepo{Repo: memory.NewSampleRepo(), fail: map[string]bool{}}
	for _, name := range fail {
		repo.fail[name] = true
	}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (archivedokumenklaim.Repo, error) { return repo, nil },
		Gateway:      gateway.NewRecorder(),
		Clock:        fixedClock{at: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)},
	})
	require.NoError(t, err)
	return service, repo
}

func TestNewServiceMenyebutSeamYangKurang(t *testing.T) {
	selector := func(string) (archivedokumenklaim.Repo, error) { return nil, nil }

	_, err := usecase.NewService(usecase.Options{RepoSelector: selector})
	require.ErrorContains(t, err, "Gateway")

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: selector, Gateway: gateway.NewRecorder(),
	})
	require.ErrorContains(t, err, "Clock")
}

// Setiap operasi meneruskan galat pemilih portal apa adanya.
func TestGalatPemilihPortalDiteruskan(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (archivedokumenklaim.Repo, error) { return nil, errSelector },
		Gateway:      gateway.NewRecorder(),
		Clock:        fixedClock{},
	})
	require.NoError(t, err)

	ctx := context.Background()
	who := caller("PA")

	_, err = service.Open(ctx, portalAlias, who)
	require.ErrorIs(t, err, errSelector)

	_, err = service.Search(ctx, portalAlias, archivedokumenklaim.CriteriaInput{Keyword: "X"},
		archivedokumenklaim.Pagination{})
	require.ErrorIs(t, err, errSelector)

	_, err = service.SearchClaims(ctx, portalAlias, "no_klaim", "X")
	require.ErrorIs(t, err, errSelector)

	_, err = service.FillingCodes(ctx, portalAlias, "")
	require.ErrorIs(t, err, errSelector)

	_, err = service.Save(ctx, portalAlias, who, "01", validDraftInput())
	require.ErrorIs(t, err, errSelector)

	_, err = service.PendingBranch(ctx, portalAlias, who, archivedokumenklaim.Pagination{})
	require.ErrorIs(t, err, errSelector)

	_, err = service.SendToBranch(ctx, portalAlias, who, 1)
	require.ErrorIs(t, err, errSelector)
}

func TestGalatRepoDibungkusKonteks(t *testing.T) {
	ctx := context.Background()
	who := caller("PA")

	cases := []struct {
		fail    string
		message string
		call    func(s *usecase.Service) error
	}{
		{"DocumentTypes", "membaca tipe dokumen", func(s *usecase.Service) error {
			_, err := s.Open(ctx, portalAlias, who)
			return err
		}},
		{"DocumentKinds", "membaca jenis dokumen", func(s *usecase.Service) error {
			_, err := s.Open(ctx, portalAlias, who)
			return err
		}},
		{"Search", "membaca berkas arsip", func(s *usecase.Service) error {
			_, err := s.Search(ctx, portalAlias, archivedokumenklaim.CriteriaInput{Keyword: "X"},
				archivedokumenklaim.Pagination{})
			return err
		}},
		{"SearchClaims", "mencari klaim", func(s *usecase.Service) error {
			_, err := s.SearchClaims(ctx, portalAlias, "no_klaim", "X")
			return err
		}},
		{"FillingCodes", "membaca kode filling", func(s *usecase.Service) error {
			_, err := s.FillingCodes(ctx, portalAlias, "")
			return err
		}},
		{"Save", "menyimpan berkas arsip", func(s *usecase.Service) error {
			_, err := s.Save(ctx, portalAlias, who, "01", validDraftInput())
			return err
		}},
		{"FindByID", "memeriksa berkas arsip", func(s *usecase.Service) error {
			input := validDraftInput()
			input.ID = 1
			_, err := s.Save(ctx, portalAlias, who, "01", input)
			return err
		}},
		{"PendingBranch", "belum dikirim", func(s *usecase.Service) error {
			_, err := s.PendingBranch(ctx, portalAlias, who, archivedokumenklaim.Pagination{})
			return err
		}},
		{"FindByID", "membaca berkas arsip", func(s *usecase.Service) error {
			_, err := s.SendToBranch(ctx, portalAlias, who, 1)
			return err
		}},
		{"MarkSent", "menyimpan jawaban layanan Arsip", func(s *usecase.Service) error {
			_, err := s.SendToBranch(ctx, portalAlias, who, 1)
			return err
		}},
	}

	for _, c := range cases {
		t.Run(c.fail+"/"+c.message, func(t *testing.T) {
			service, _ := buildBroken(t, c.fail)
			err := c.call(service)
			require.ErrorIs(t, err, errRepo)
			require.ErrorContains(t, err, c.message)
		})
	}
}

func TestValidasiDitolakSebelumMenyentuhRepo(t *testing.T) {
	service, _ := buildBroken(t)
	ctx := context.Background()

	_, err := service.Search(ctx, portalAlias, archivedokumenklaim.CriteriaInput{},
		archivedokumenklaim.Pagination{})
	var validation *archivedokumenklaim.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = service.SearchClaims(ctx, portalAlias, "", "X")
	require.ErrorAs(t, err, &validation)

	_, err = service.Save(ctx, portalAlias, caller("PA"), "01", archivedokumenklaim.DraftInput{})
	require.ErrorAs(t, err, &validation)

	_, err = service.PendingBranch(ctx, portalAlias, archivedokumenklaim.Caller{},
		archivedokumenklaim.Pagination{})
	require.ErrorIs(t, err, archivedokumenklaim.ErrCallerUnknown)

	_, err = service.SendToBranch(ctx, portalAlias, archivedokumenklaim.Caller{}, 1)
	require.ErrorIs(t, err, archivedokumenklaim.ErrCallerUnknown)
}

// Jawaban layanan yang gagal dicatat menjadi keterangan, bukan galat penyimpanan.
func TestSimpanJawabanGagalDicatatSebagaiKeterangan(t *testing.T) {
	service, repo := buildBroken(t, "StoreReceipt")

	saved, err := service.Save(context.Background(), portalAlias, caller("PA"), "01",
		validDraftInput())
	require.NoError(t, err)
	require.False(t, saved.Sent)
	require.Equal(t, errRepo.Error(), saved.SendError)

	_, exists, err := repo.FindByID(context.Background(), saved.ID)
	require.NoError(t, err)
	require.True(t, exists)
}

// Jawaban tanpa waktu kirim memakai waktu permintaan dari Clock.
func TestJawabanTanpaWaktuMemakaiWaktuPermintaan(t *testing.T) {
	repo := memory.NewSampleRepo()
	at := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (archivedokumenklaim.Repo, error) { return repo, nil },
		Gateway:      zeroTimeGateway{},
		Clock:        fixedClock{at: at},
	})
	require.NoError(t, err)

	saved, err := service.Save(context.Background(), portalAlias, caller("PA"), "01",
		validDraftInput())
	require.NoError(t, err)
	require.Equal(t, "201", saved.ServiceCode)
	require.Equal(t, "masuk", saved.ServiceNote)

	file, _, err := repo.FindByID(context.Background(), saved.ID)
	require.NoError(t, err)
	require.Equal(t, at, *file.SentDate)

	sent, err := service.SendToBranch(context.Background(), portalAlias, caller("PA"), 1)
	require.NoError(t, err)
	require.Equal(t, usecase.Sent{ID: 1, Code: "201", Note: "masuk"}, sent)

	file, _, err = repo.FindByID(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, at, *file.SentDate)
}
