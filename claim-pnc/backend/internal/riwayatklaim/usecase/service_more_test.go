package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/riwayatklaim"
	"claim-pnc/internal/riwayatklaim/repo/memory"
	"claim-pnc/internal/riwayatklaim/usecase"
)

// faultyGate adalah gerbang proteksi yang dapat digagalkan per langkah.
type faultyGate struct {
	findErr, countErr, recordErr error
}

func (g faultyGate) Find(context.Context, string, string) (riwayatklaim.Protection, bool, error) {
	return riwayatklaim.Protection{Login: "adminpnc", SearchQuota: 9}, true, g.findErr
}

func (g faultyGate) CountUsage(context.Context, string, string) (int, error) {
	return 0, g.countErr
}

func (g faultyGate) RecordUsage(context.Context, riwayatklaim.Usage) error {
	return g.recordErr
}

// faultyClaims gagal saat mencari.
type faultyClaims struct{ err error }

func (f faultyClaims) Search(context.Context, riwayatklaim.Criteria, riwayatklaim.Pagination) (riwayatklaim.Page, error) {
	return riwayatklaim.Page{}, f.err
}

func build(t *testing.T, gate riwayatklaim.ProtectionRepo, claims riwayatklaim.Repo,
	gateErr, claimsErr error,
) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (riwayatklaim.Repo, error) { return claims, claimsErr },
		ProtectionSelector: func(string) (riwayatklaim.ProtectionRepo, error) {
			return gate, gateErr
		},
		Clock: jamTetap{pada: time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)},
	})
	require.NoError(t, err)
	return service
}

var policySearch = riwayatklaim.CriteriaInput{Type: riwayatklaim.TypePolicyNumber, Text: "POL"}

func TestNewServiceRequiresEveryDependency(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.ErrorContains(t, err, "RepoSelector wajib diisi")

	repoSel := func(string) (riwayatklaim.Repo, error) { return nil, nil }
	_, err = usecase.NewService(usecase.Options{RepoSelector: repoSel})
	require.ErrorContains(t, err, "ProtectionSelector wajib diisi")

	_, err = usecase.NewService(usecase.Options{
		RepoSelector:       repoSel,
		ProtectionSelector: func(string) (riwayatklaim.ProtectionRepo, error) { return nil, nil },
	})
	require.ErrorContains(t, err, "Clock wajib diisi")
}

func TestOpenPropagatesGateFailures(t *testing.T) {
	selectorErr := errors.New("portal belum siap")
	_, err := build(t, nil, nil, selectorErr, nil).Open(context.Background(), "ASM", pemanggil)
	require.Equal(t, selectorErr, err)

	boom := errors.New("ora")
	_, err = build(t, faultyGate{findErr: boom}, nil, nil, nil).
		Open(context.Background(), "ASM", pemanggil)
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "membaca proteksi data")

	_, err = build(t, faultyGate{countErr: boom}, nil, nil, nil).
		Open(context.Background(), "ASM", pemanggil)
	require.ErrorContains(t, err, "menghitung pemakaian jatah")

	_, err = build(t, faultyGate{recordErr: boom}, nil, nil, nil).
		Open(context.Background(), "ASM", pemanggil)
	require.ErrorContains(t, err, "mencatat pemakaian jatah")
}

func TestSearchRejectsUnknownCaller(t *testing.T) {
	_, err := build(t, faultyGate{}, nil, nil, nil).Search(context.Background(), "ASM",
		riwayatklaim.Caller{}, policySearch, riwayatklaim.Pagination{})
	require.ErrorIs(t, err, riwayatklaim.ErrCallerUnknown)
}

func TestSearchPropagatesGateAndRepoFailures(t *testing.T) {
	boom := errors.New("ora")
	claims := memory.NewRepo(memory.SampleClaims()...)
	run := func(s *usecase.Service) error {
		_, err := s.Search(context.Background(), "ASM", pemanggil, policySearch,
			riwayatklaim.Pagination{})
		return err
	}

	require.EqualError(t, run(build(t, nil, claims, errors.New("gerbang"), nil)), "gerbang")
	require.ErrorContains(t, run(build(t, faultyGate{findErr: boom}, claims, nil, nil)),
		"membaca proteksi data")
	require.ErrorContains(t, run(build(t, faultyGate{countErr: boom}, claims, nil, nil)),
		"menghitung pemakaian jatah")
	require.EqualError(t, run(build(t, faultyGate{}, claims, nil, errors.New("repo"))), "repo")
	require.ErrorContains(t, run(build(t, faultyGate{}, faultyClaims{err: boom}, nil, nil)),
		"mencari riwayat klaim")
	require.ErrorContains(t, run(build(t, faultyGate{recordErr: boom}, claims, nil, nil)),
		"mencatat pencarian")
}

func TestSearchByDateRecordsFormattedValue(t *testing.T) {
	gate := memory.NewProtectionRepo(penggunaTerdaftar(5))
	service := build(t, gate, memory.NewRepo(memory.SampleClaims()...), nil, nil)

	lossDay := time.Date(2026, 3, 12, 15, 0, 0, 0, time.UTC)
	found, err := service.Search(context.Background(), "ASM", pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeLossDate, SearchDate: &lossDay},
		riwayatklaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 2, found.Page.Total)

	usage := gate.Usage()
	require.Len(t, usage, 1)
	require.Equal(t, "2026-03-12", usage[0].SearchValue)
	require.Equal(t, riwayatklaim.TypeLossDate, usage[0].SearchTypeCode)

	// Tipe tanggal lahir mengirim isian tanggal pencarian yang selalu kosong.
	birth := time.Date(1990, 7, 17, 0, 0, 0, 0, time.UTC)
	_, err = service.Search(context.Background(), "ASM", pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeBirthDate, BirthDate: &birth},
		riwayatklaim.Pagination{})
	require.NoError(t, err)
	require.Empty(t, gate.Usage()[1].SearchValue)
}
