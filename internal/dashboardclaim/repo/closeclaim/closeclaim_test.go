package closeclaim_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/repo/closeclaim"
	"claim-pnc/internal/inboxcloseclaim"
)

// fakeClosedRepo merekam penyaring yang diterimanya dan mengembalikan halaman tetap.
type fakeClosedRepo struct {
	got  []inboxcloseclaim.Filter
	page inboxcloseclaim.Page
	err  error
}

func (f *fakeClosedRepo) List(_ context.Context, filter inboxcloseclaim.Filter) (inboxcloseclaim.Page, error) {
	f.got = append(f.got, filter)
	return f.page, f.err
}

func (f *fakeClosedRepo) ClosedClaimNumber(context.Context, string) (string, error) {
	return "", errors.New("tidak dipakai dashboard")
}

func selectorFor(repo inboxcloseclaim.Repo, err error) inboxcloseclaim.RepoSelector {
	return func(string) (inboxcloseclaim.Repo, error) { return repo, err }
}

func TestNewReaderRequiresSelector(t *testing.T) {
	reader, err := closeclaim.NewReader(nil)
	require.Nil(t, reader)
	require.EqualError(t, err, "dashboardclaim/closeclaim: pemilih repo klaim tutup wajib diisi")
}

// Count meminta satu baris saja, dari awal, dengan penyaring yang sudah dinormalisasi.
func TestCountAsksForOneRow(t *testing.T) {
	repo := &fakeClosedRepo{page: inboxcloseclaim.Page{Total: 42}}
	reader, err := closeclaim.NewReader(selectorFor(repo, nil))
	require.NoError(t, err)

	count, err := reader.Count(context.Background(), "ASM", dashboardclaim.Filter{
		Business: dashboardclaim.BusinessPA, Search: " polis ", Limit: 50, Offset: 30,
	})
	require.NoError(t, err)
	require.Equal(t, 42, count)
	require.Equal(t, []inboxcloseclaim.Filter{{
		Search: "polis", Business: inboxcloseclaim.BusinessLine("PA"), Limit: 1, Offset: 0,
	}}, repo.got)
}

func TestCountErrors(t *testing.T) {
	boom := errors.New("portal belum siap")
	reader, err := closeclaim.NewReader(selectorFor(nil, boom))
	require.NoError(t, err)
	_, err = reader.Count(context.Background(), "ASM", dashboardclaim.Filter{})
	require.ErrorIs(t, err, boom)

	failing := &fakeClosedRepo{err: errors.New("ora-00942")}
	reader, err = closeclaim.NewReader(selectorFor(failing, nil))
	require.NoError(t, err)
	_, err = reader.Count(context.Background(), "ASM", dashboardclaim.Filter{})
	require.EqualError(t, err, "dashboardclaim/closeclaim: menghitung klaim tutup: ora-00942")
}

// List memetakan setiap klaim tutup menjadi baris dashboard tanpa kolom tambahan.
func TestListAdaptsRows(t *testing.T) {
	loss := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	registered := time.Date(2026, time.September, 2, 3, 0, 0, 0, time.UTC)
	closed := time.Date(2026, time.September, 9, 0, 0, 0, 0, time.UTC)

	repo := &fakeClosedRepo{page: inboxcloseclaim.Page{Total: 9, Claims: []inboxcloseclaim.ClosedClaim{{
		ClaimID: "ID1", ClaimNumber: "PNCN.26.0101", PolicyNumber: "POL", InsuredName: "Tertanggung",
		BusinessName: "Bisnis", BusinessSource: "Sumber", BranchName: "Cabang",
		TechnicalPIC: "PIC", AdminPNC: "Admin", ClaimStatusCode: "1163",
		ProcessStatus: "Resolved-Completed", LossDate: &loss, RegisteredAt: registered,
		ClosedAt: &closed, TransferredToCashier: true,
	}}}}
	reader, err := closeclaim.NewReader(selectorFor(repo, nil))
	require.NoError(t, err)

	page, err := reader.List(context.Background(), "ASM", dashboardclaim.Filter{Limit: 10, Offset: 20})
	require.NoError(t, err)
	require.Equal(t, 9, page.Total)
	require.Equal(t, []dashboardclaim.ClaimRow{{
		ClaimID: "ID1", ClaimNumber: "PNCN.26.0101", PolicyNumber: "POL", InsuredName: "Tertanggung",
		BusinessName: "Bisnis", BusinessSource: "Sumber", BranchName: "Cabang",
		TechnicalPIC: "PIC", AdminPNC: "Admin", ClaimStatusCode: "1163",
		ProcessStatus: "Resolved-Completed", LossDate: &loss, RegisteredAt: registered,
	}}, page.Rows)
	require.Equal(t, []inboxcloseclaim.Filter{{
		Business: inboxcloseclaim.BusinessLine("ALL"), Limit: 10, Offset: 20,
	}}, repo.got)
}

func TestListErrors(t *testing.T) {
	boom := errors.New("portal belum siap")
	reader, err := closeclaim.NewReader(selectorFor(nil, boom))
	require.NoError(t, err)
	_, err = reader.List(context.Background(), "ASM", dashboardclaim.Filter{})
	require.ErrorIs(t, err, boom)

	failing := &fakeClosedRepo{err: errors.New("ora-01017")}
	reader, err = closeclaim.NewReader(selectorFor(failing, nil))
	require.NoError(t, err)
	_, err = reader.List(context.Background(), "ASM", dashboardclaim.Filter{})
	require.EqualError(t, err, "dashboardclaim/closeclaim: membaca daftar klaim tutup: ora-01017")
}

// Halaman kosong tetap senarai, bukan nil.
func TestListEmptyPage(t *testing.T) {
	reader, err := closeclaim.NewReader(selectorFor(&fakeClosedRepo{}, nil))
	require.NoError(t, err)

	page, err := reader.List(context.Background(), "ASM", dashboardclaim.Filter{})
	require.NoError(t, err)
	require.NotNil(t, page.Rows)
	require.Empty(t, page.Rows)
}
