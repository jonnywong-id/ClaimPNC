package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/inboxacceptopenprotection/repo/memory"
	"claim-pnc/internal/inboxacceptopenprotection/usecase"
)

// Uji di berkas ini menjaga cabang galat service: pembaca group yang gagal, pemilih portal
// yang menolak, dan penyimpanan yang gagal. Ketiganya harus diteruskan, bukan ditelan.

var errStore = errors.New("penyimpanan gagal")

type failingGroups struct{}

func (failingGroups) GroupsOf(context.Context, string) ([]string, error) { return nil, errStore }

// failingRepo membungkus repo memory dan menggagalkan operasi tertentu.
type failingRepo struct {
	*memory.Repo
	listErr, decideErr error
}

func (f failingRepo) List(ctx context.Context, flt inboxacceptopenprotection.Filter) (inboxacceptopenprotection.Page, error) {
	if f.listErr != nil {
		return inboxacceptopenprotection.Page{}, f.listErr
	}
	return f.Repo.List(ctx, flt)
}

func (f failingRepo) Decide(ctx context.Context, n string, d inboxacceptopenprotection.Decision, by string, at time.Time) (inboxacceptopenprotection.Protection, error) {
	if f.decideErr != nil {
		return inboxacceptopenprotection.Protection{}, f.decideErr
	}
	return f.Repo.Decide(ctx, n, d, by, at)
}

func serviceWith(t *testing.T, repo inboxacceptopenprotection.Repo, groups inboxacceptopenprotection.GroupReader) *usecase.Service {
	t.Helper()
	s, err := usecase.NewService(usecase.Options{
		Protections: func(alias string) (inboxacceptopenprotection.Repo, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
		Groups: groups,
	})
	require.NoError(t, err)
	return s
}

func TestNewServiceRequiresProtectionSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Groups: memory.NewSampleGroupRepo()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "pemilih repo proteksi wajib diisi")
}

func TestNewServiceDefaultsClockToSystemTime(t *testing.T) {
	// Tanpa Now, waktu akseptasi diambil dari jam sistem — diperiksa hanya bahwa ia terisi
	// dan berada di antara dua titik yang dicatat uji.
	repo := memory.NewRepo()
	repo.Add(lengkap("OPCN.26.0001", "1", time.Now()))
	s := serviceWith(t, repo, memory.NewSampleGroupRepo())

	sebelum := time.Now()
	saved, err := s.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal, Number: "OPCN.26.0001",
		Decision: inboxacceptopenprotection.DecisionApprove, By: loginBerwenang,
	})
	sesudah := time.Now()
	require.NoError(t, err)
	require.NotNil(t, saved.AcceptedAt)
	require.False(t, saved.AcceptedAt.Before(sebelum))
	require.False(t, saved.AcceptedAt.After(sesudah))
}

func TestGroupReaderFailureIsNotForbidden(t *testing.T) {
	s := serviceWith(t, memory.NewRepo(), failingGroups{})
	ctx := context.Background()

	_, err := s.List(ctx, usecase.ListQuery{PortalAlias: portal, Login: "X"})
	require.ErrorIs(t, err, errStore)
	require.NotErrorIs(t, err, inboxacceptopenprotection.ErrForbidden)
	require.Contains(t, err.Error(), "membaca access group pemanggil")

	_, err = s.Queues(ctx, "X")
	require.ErrorIs(t, err, errStore)

	_, err = s.Get(ctx, portal, "A", "X")
	require.ErrorIs(t, err, errStore)

	_, err = s.Decide(ctx, usecase.DecideCommand{
		PortalAlias: portal, Number: "A", Decision: inboxacceptopenprotection.DecisionApprove, By: "X",
	})
	require.ErrorIs(t, err, errStore)
}

func TestQueuesListsAllowedQueues(t *testing.T) {
	s := serviceWith(t, memory.NewRepo(), memory.NewSampleGroupRepo())

	queues, err := s.Queues(context.Background(), loginBerwenang)
	require.NoError(t, err)
	require.Equal(t, []inboxacceptopenprotection.Queue{
		inboxacceptopenprotection.QueueNonPremium, inboxacceptopenprotection.QueuePremium,
	}, queues)
}

func TestListWrapsRepoFailure(t *testing.T) {
	s := serviceWith(t, failingRepo{Repo: memory.NewRepo(), listErr: errStore}, memory.NewSampleGroupRepo())

	_, err := s.List(context.Background(), usecase.ListQuery{PortalAlias: portal, Login: loginBerwenang})
	require.ErrorIs(t, err, errStore)
	require.Contains(t, err.Error(), "membaca antrean akseptasi")
}

func TestUnknownPortalIsRejectedForAuthorizedCaller(t *testing.T) {
	s := serviceWith(t, memory.NewRepo(), memory.NewSampleGroupRepo())
	ctx := context.Background()

	_, err := s.List(ctx, usecase.ListQuery{PortalAlias: "lain", Login: loginBerwenang})
	require.EqualError(t, err, "portal tidak dikenal")

	_, err = s.Get(ctx, "lain", "A", loginBerwenang)
	require.EqualError(t, err, "portal tidak dikenal")

	_, err = s.Decide(ctx, usecase.DecideCommand{
		PortalAlias: "lain", Number: "A", Decision: inboxacceptopenprotection.DecisionApprove, By: loginBerwenang,
	})
	require.EqualError(t, err, "portal tidak dikenal")
}

func TestDecideUnknownProtectionIsNotFound(t *testing.T) {
	s := serviceWith(t, memory.NewRepo(), memory.NewSampleGroupRepo())

	_, err := s.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal, Number: "TIDAK-ADA", Decision: inboxacceptopenprotection.DecisionApprove, By: loginBerwenang,
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrNotFound)
}

func TestDecideForwardsRepoFailure(t *testing.T) {
	base := memory.NewRepo()
	base.Add(lengkap("OPCN.26.0001", "1", time.Now()))
	s := serviceWith(t, failingRepo{Repo: base, decideErr: errStore}, memory.NewSampleGroupRepo())

	_, err := s.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal, Number: "OPCN.26.0001", Decision: inboxacceptopenprotection.DecisionApprove, By: loginBerwenang,
	})
	require.ErrorIs(t, err, errStore)
}
