package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxoutstanding/usecase"
)

// failingRepo gagal pada method yang galatnya diisi; sisanya menjawab kosong.
type failingRepo struct {
	legacyErr  error
	listErr    error
	summaryErr error
	lineErr    error
	exportErr  error
}

func (f failingRepo) LegacyOperatorFor(context.Context, string) (string, error) {
	return "", f.legacyErr
}

func (f failingRepo) SummarizeDocumentStatus(context.Context, inboxoutstanding.Filter) (inboxoutstanding.Summary, error) {
	return inboxoutstanding.Summary{}, f.summaryErr
}

func (f failingRepo) List(context.Context, inboxoutstanding.Filter) (inboxoutstanding.Page, error) {
	return inboxoutstanding.Page{}, f.listErr
}

func (f failingRepo) Export(context.Context, inboxoutstanding.ExportFilter) (inboxoutstanding.Page, error) {
	return inboxoutstanding.Page{}, f.exportErr
}

func (f failingRepo) LineBusinessFor(context.Context, string) (inboxoutstanding.LineBusiness, error) {
	return inboxoutstanding.LineUnknown, f.lineErr
}

func newFailingService(t *testing.T, repo failingRepo) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(selectorFor(repo))
	require.NoError(t, err)
	return service
}

var errRepo = errors.New("koneksi putus")

func TestListWrapsALegacyIdentityFailure(t *testing.T) {
	service := newFailingService(t, failingRepo{legacyErr: errRepo})

	_, err := service.List(context.Background(), usecase.Query{LoginID: "budi", PortalAlias: testPortal})
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "membaca identitas lama pemanggil")
}

func TestListWrapsAListFailure(t *testing.T) {
	service := newFailingService(t, failingRepo{listErr: errRepo})

	_, err := service.List(context.Background(), usecase.Query{LoginID: "budi", PortalAlias: testPortal})
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "membaca daftar klaim")
}

func TestSummaryStopsWhenThePortalCannotBeSelected(t *testing.T) {
	selectorErr := errors.New("portal belum siap")
	service, err := usecase.NewService(func(string) (inboxoutstanding.Repo, error) {
		return nil, selectorErr
	})
	require.NoError(t, err)

	_, err = service.Summary(context.Background(), usecase.Query{LoginID: "budi", PortalAlias: testPortal})
	require.ErrorIs(t, err, selectorErr)

	_, err = service.Export(context.Background(), usecase.ExportQuery{LoginID: "budi", PortalAlias: testPortal})
	require.ErrorIs(t, err, selectorErr)
}

func TestSummaryWrapsALegacyIdentityFailure(t *testing.T) {
	service := newFailingService(t, failingRepo{legacyErr: errRepo})

	_, err := service.Summary(context.Background(), usecase.Query{LoginID: "budi", PortalAlias: testPortal})
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "membaca identitas lama pemanggil")
}

func TestSummaryWrapsASummaryFailure(t *testing.T) {
	service := newFailingService(t, failingRepo{summaryErr: errRepo})

	_, err := service.Summary(context.Background(), usecase.Query{LoginID: "budi", PortalAlias: testPortal})
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "meringkas status dokumen")
}

func TestExportRejectsAnEmptyIdentity(t *testing.T) {
	service := newFailingService(t, failingRepo{})

	_, err := service.Export(context.Background(), usecase.ExportQuery{PortalAlias: testPortal})
	require.ErrorContains(t, err, "identitas pemanggil kosong")
}

func TestExportWrapsALineBusinessFailure(t *testing.T) {
	service := newFailingService(t, failingRepo{lineErr: errRepo})

	_, err := service.Export(context.Background(), usecase.ExportQuery{LoginID: "budi", PortalAlias: testPortal})
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "membaca lini bisnis pemanggil")
}

func TestExportWrapsAnExportFailure(t *testing.T) {
	service := newFailingService(t, failingRepo{exportErr: errRepo})

	_, err := service.Export(context.Background(), usecase.ExportQuery{LoginID: "budi", PortalAlias: testPortal})
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "membaca baris export")
}
