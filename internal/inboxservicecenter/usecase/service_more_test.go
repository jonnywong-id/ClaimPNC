package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/inboxservicecenter/repo/memory"
	"claim-pnc/internal/inboxservicecenter/usecase"
)

// brokenRepo menggagalkan operasi yang dipilih, sisanya diteruskan ke penyimpanan contoh.
type brokenRepo struct {
	inboxservicecenter.Repo
	listErr     error
	progressErr error
}

func (b brokenRepo) List(ctx context.Context, q inboxservicecenter.Query, p inboxservicecenter.Pagination) (inboxservicecenter.Page, error) {
	if b.listErr != nil {
		return inboxservicecenter.Page{}, b.listErr
	}
	return b.Repo.List(ctx, q, p)
}

func (b brokenRepo) ListProgress(ctx context.Context, id string) ([]inboxservicecenter.ProgressNote, error) {
	if b.progressErr != nil {
		return nil, b.progressErr
	}
	return b.Repo.ListProgress(ctx, id)
}

func serviceWithLogger(t *testing.T, repo inboxservicecenter.Repo, logs *bytes.Buffer) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxservicecenter.Repo, error) { return repo, nil },
		Logger:       slog.New(slog.NewTextHandler(logs, nil)),
	})
	require.NoError(t, err)
	return service
}

func TestNewServiceRequiresSelector(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{})
	require.Nil(t, service)
	require.EqualError(t, err, "inboxservicecenter/usecase: RepoSelector wajib diisi")
}

func TestUnknownPortalIsRejected(t *testing.T) {
	service := newService(t, memory.NewSampleStore())
	caller := inboxservicecenter.Caller{Login: memory.SampleOwner}

	_, err := service.List(context.Background(), "lain", caller, inboxservicecenter.QueryInput{}, inboxservicecenter.Pagination{})
	require.EqualError(t, err, "portal tidak dikenal: lain")

	_, err = service.Detail(context.Background(), "lain", caller, "SC-000101")
	require.EqualError(t, err, "portal tidak dikenal: lain")
}

func TestListRepoErrorIsWrappedWithTab(t *testing.T) {
	boom := errors.New("putus")
	service := serviceWithLogger(t, brokenRepo{Repo: memory.NewSampleStore(), listErr: boom}, &bytes.Buffer{})

	_, err := service.List(context.Background(), portalUtama,
		inboxservicecenter.Caller{Login: memory.SampleOwner},
		inboxservicecenter.QueryInput{Tab: inboxservicecenter.TabApproved}, inboxservicecenter.Pagination{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "mengambil isi tab "+inboxservicecenter.TabApproved)
}

// Pencarian yang mengembalikan lebih dari ambang dicatat, tanpa memotong hasilnya.
func TestLargeSearchIsLoggedNotTruncated(t *testing.T) {
	claims := make([]inboxservicecenter.ServiceClaim, 0, usecase.LargeSearchWarning+1)
	for i := 0; i <= usecase.LargeSearchWarning; i++ {
		claims = append(claims, inboxservicecenter.ServiceClaim{
			ID: fmt.Sprintf("SC-%05d", i), TechnicalPIC: "PIC", IMEI: "IMEI-X",
		})
	}
	logs := &bytes.Buffer{}
	service := serviceWithLogger(t, memory.NewStore(claims...), logs)

	listed, err := service.List(context.Background(), portalUtama, inboxservicecenter.Caller{Login: "PIC"},
		inboxservicecenter.QueryInput{Keyword: "imei"}, inboxservicecenter.Pagination{})
	require.NoError(t, err)
	require.Len(t, listed.Page.Items, usecase.LargeSearchWarning+1)
	require.Contains(t, logs.String(), "sangat banyak baris")
	require.Contains(t, logs.String(), "jumlah_baris=2001")
}

// Riwayat yang gagal dibaca tidak menggagalkan rincian; ia dicatat di log.
func TestDetailSurvivesProgressFailure(t *testing.T) {
	logs := &bytes.Buffer{}
	service := serviceWithLogger(t,
		brokenRepo{Repo: memory.NewSampleStore(), progressErr: errors.New("tabel progres hilang")}, logs)

	got, err := service.Detail(context.Background(), portalUtama,
		inboxservicecenter.Caller{Login: memory.SampleOwner}, "SC-000101")
	require.NoError(t, err)
	require.Equal(t, "SC-000101", got.Claim.ID)
	require.NotNil(t, got.Progress)
	require.Empty(t, got.Progress)
	require.Contains(t, logs.String(), "riwayat progres Inbox Service Center tidak dapat dibaca")
	require.Contains(t, logs.String(), "tabel progres hilang")

	// Tanpa logger pun hasilnya sama.
	noLog := newService(t, brokenRepo{Repo: memory.NewSampleStore(), progressErr: errors.New("x")})
	got, err = noLog.Detail(context.Background(), portalUtama,
		inboxservicecenter.Caller{Login: memory.SampleOwner}, "SC-000101")
	require.NoError(t, err)
	require.Empty(t, got.Progress)
}
