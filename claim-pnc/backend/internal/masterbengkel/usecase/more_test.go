package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/repo/memory"
	"claim-pnc/internal/masterbengkel/usecase"
)

// storeTerganggu membungkus penyimpanan memori dan menggagalkan operasi tertentu.
type storeTerganggu struct {
	*memory.Repo
	emptyID     bool
	nextIDErr   error
	updateErr   error
	statusErr   error
	findNameErr error
	findLogErr  error
	docIDErr    error
}

func (s *storeTerganggu) NextID(ctx context.Context) (string, error) {
	if s.nextIDErr != nil {
		return "", s.nextIDErr
	}
	if s.emptyID {
		return " ", nil
	}
	return s.Repo.NextID(ctx)
}

func (s *storeTerganggu) Update(ctx context.Context, w masterbengkel.Workshop) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.Repo.Update(ctx, w)
}

func (s *storeTerganggu) SetStatus(
	ctx context.Context, id []string, status masterbengkel.ApprovalStatus,
) (int, error) {
	if s.statusErr != nil {
		return 0, s.statusErr
	}
	return s.Repo.SetStatus(ctx, id, status)
}

func (s *storeTerganggu) FindByName(ctx context.Context, name string) (masterbengkel.Workshop, error) {
	if s.findNameErr != nil {
		return masterbengkel.Workshop{}, s.findNameErr
	}
	return s.Repo.FindByName(ctx, name)
}

func (s *storeTerganggu) FindByLogin(ctx context.Context, login string) (masterbengkel.Workshop, error) {
	if s.findLogErr != nil {
		return masterbengkel.Workshop{}, s.findLogErr
	}
	return s.Repo.FindByLogin(ctx, login)
}

func (s *storeTerganggu) NextDocumentID(ctx context.Context) (string, error) {
	if s.docIDErr != nil {
		return "", s.docIDErr
	}
	return s.Repo.NextDocumentID(ctx)
}

func serviceOver(t *testing.T, store masterbengkel.Store) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterbengkel.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak tersedia")
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return service
}

var errOracle = errors.New("oracle mati")

func TestEveryOperationRejectsAnUnknownPortal(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	_, err := service.Get(ctx, "SMI", "x")
	require.EqualError(t, err, "portal tidak tersedia")
	_, err = service.Create(ctx, "SMI", newInput(), usecase.Actor{}, nil)
	require.EqualError(t, err, "portal tidak tersedia")
	_, err = service.Save(ctx, "SMI", "x", newInput(), usecase.Actor{}, nil)
	require.EqualError(t, err, "portal tidak tersedia")
	_, err = service.Decide(ctx, "SMI", []string{"x"}, masterbengkel.StatusApproved,
		usecase.Actor{}, nil)
	require.EqualError(t, err, "portal tidak tersedia")
	_, err = service.UploadDocument(ctx, "SMI", usecase.Actor{}, masterbengkel.UploadInput{
		WorkshopID: "x", FileName: "a.pdf", Content: []byte("x")})
	require.EqualError(t, err, "portal tidak tersedia")
	_, err = service.Document(ctx, "SMI", "x")
	require.EqualError(t, err, "portal tidak tersedia")
}

// Penerbitan ID yang gagal atau kosong menghentikan penambahan.
func TestCreateStopsWhenTheIDCannotBeIssued(t *testing.T) {
	ctx := context.Background()

	failing := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), nextIDErr: errOracle})
	_, err := failing.Create(ctx, portalAlias, newInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "menerbitkan ID bengkel")

	empty := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), emptyID: true})
	_, err = empty.Create(ctx, portalAlias, newInput(), usecase.Actor{}, nil)
	require.EqualError(t, err, "masterbengkel/usecase: ID bengkel yang diterbitkan kosong")
}

// Penambahan yang berhasil dicatat sebagai pemberitahuan yang tidak dikirim.
func TestCreateLogsTheOmittedNotification(t *testing.T) {
	service, _ := newService(t)
	log := &bytes.Buffer{}

	_, err := service.Create(context.Background(), portalAlias, newInput(),
		usecase.Actor{Login: "penguji"}, slog.New(slog.NewTextHandler(log, nil)))
	require.NoError(t, err)
	require.Contains(t, log.String(), "pemberitahuan master bengkel tidak dikirim")
}

// Penyimpanan berhenti pada ID kosong, isian salah, baris hilang, atau galat penulisan.
func TestSaveStops(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)

	_, err := service.Save(ctx, portalAlias, "  ", newInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)

	bad := newInput()
	bad.Name = ""
	_, err = service.Save(ctx, portalAlias, "010000000001", bad, usecase.Actor{}, nil)
	var validation *masterbengkel.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = service.Save(ctx, portalAlias, "999999999999", newInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)

	broken := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), updateErr: errOracle})
	_, err = broken.Save(ctx, portalAlias, "010000000001", newInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, errOracle)
}

// Pemeriksaan keunikan nama dan login meneruskan galat penyimpanan dengan konteksnya.
func TestSaveUniquenessChecksWrapStorageFailures(t *testing.T) {
	ctx := context.Background()

	byName := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), findNameErr: errOracle})
	_, err := byName.Save(ctx, portalAlias, "010000000001", newInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "memeriksa nama")

	byLogin := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), findLogErr: errOracle})
	_, err = byLogin.Save(ctx, portalAlias, "010000000001", newInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "memeriksa login")

	// Login kosong tidak diperiksa sama sekali.
	noLogin := newInput()
	noLogin.Login = ""
	noLogin.PartnerStatus = "0"
	skipped := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), findLogErr: errOracle})
	_, err = skipped.Save(ctx, portalAlias, "010000000001", noLogin, usecase.Actor{}, nil)
	require.NoError(t, err)
}

// Galat keputusan diteruskan; keputusan sebagian diberi peringatan dan dicatat.
func TestDecideFailuresAndPartialChanges(t *testing.T) {
	ctx := context.Background()

	broken := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), statusErr: errOracle})
	_, err := broken.Decide(ctx, portalAlias, []string{"010000000001"},
		masterbengkel.StatusApproved, usecase.Actor{}, nil)
	require.ErrorIs(t, err, errOracle)

	service, repo := newService(t)
	pending, err := repo.List(ctx, masterbengkel.Filter{Status: masterbengkel.StatusPending})
	require.NoError(t, err)
	require.NotEmpty(t, pending)

	log := &bytes.Buffer{}
	changed, err := service.Decide(ctx, portalAlias, []string{pending[0].ID, "999999999999"},
		masterbengkel.StatusApproved, usecase.Actor{Login: "penyetuju"},
		slog.New(slog.NewTextHandler(log, nil)))
	require.NoError(t, err)
	require.Equal(t, 1, changed)
	require.Contains(t, log.String(), "tidak menyentuh seluruh baris yang dipilih")
	require.Contains(t, log.String(), "keputusan master bengkel tersimpan tanpa pemberitahuan")
}

// Unggahan berhenti pada bengkel yang tidak ada dan pada penerbitan DATAID yang gagal.
func TestUploadDocumentStops(t *testing.T) {
	ctx := context.Background()
	input := masterbengkel.UploadInput{WorkshopID: "010000000001", FileName: "a.pdf",
		Content: []byte("isi")}

	service, _ := newService(t)
	missing := input
	missing.WorkshopID = "999999999999"
	_, err := service.UploadDocument(ctx, portalAlias, usecase.Actor{}, missing)
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)

	broken := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), docIDErr: errOracle})
	_, err = broken.UploadDocument(ctx, portalAlias, usecase.Actor{}, input)
	require.ErrorIs(t, err, errOracle)

	// Tanpa jam yang disetel, waktu unggah diambil dari jam sistem.
	document, err := service.UploadDocument(ctx, portalAlias, usecase.Actor{Login: " penguji "}, input)
	require.NoError(t, err)
	require.False(t, document.UploadedAt.IsZero())
	require.Equal(t, "penguji", document.UploadedBy)
	require.True(t, strings.HasPrefix(document.ID, memory.SampleDocumentYear))
}

// Bengkel yang tidak ada dijawab tidak ditemukan saat lampirannya dibuka.
func TestDocumentOfAMissingWorkshop(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Document(context.Background(), portalAlias, "999999999999")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
}

// storePenyimpanGagal menggagalkan penyimpanan lampiran.
type storePenyimpanGagal struct{ *memory.Repo }

func (storePenyimpanGagal) SaveDocument(context.Context, string, masterbengkel.Document) error {
	return errOracle
}

// Galat penyimpanan lampiran diteruskan, dan Get mengembalikan baris yang diminta.
func TestUploadSaveFailureAndGet(t *testing.T) {
	ctx := context.Background()

	broken := serviceOver(t, storePenyimpanGagal{Repo: memory.NewSampleRepo()})
	_, err := broken.UploadDocument(ctx, portalAlias, usecase.Actor{},
		masterbengkel.UploadInput{WorkshopID: "010000000001", FileName: "a.pdf",
			Content: []byte("isi")})
	require.ErrorIs(t, err, errOracle)

	service, _ := newService(t)
	workshop, err := service.Get(ctx, portalAlias, " 010000000001 ")
	require.NoError(t, err)
	require.Equal(t, "Bengkel Contoh Utama", workshop.Name)
}
