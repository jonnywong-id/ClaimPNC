package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastersurveyors/account"
	"claim-pnc/internal/mastersurveyors/committee"
	"claim-pnc/internal/mastersurveyors/repo/memory"
	"claim-pnc/internal/mastersurveyors/usecase"
)

var errRepo = errors.New("repo rusak")

// brokenRepo menggagalkan operasi yang ditandai dan meneruskan sisanya ke repo memori.
type brokenRepo struct {
	*memory.Repo
	fail     map[string]error
	byName   []mastersurveyors.Surveyor
	byLogin  []mastersurveyors.Surveyor
	override bool
}

func (b *brokenRepo) List(ctx context.Context, f mastersurveyors.Filter) ([]mastersurveyors.Surveyor, int, error) {
	if err := b.fail["List"]; err != nil {
		return nil, 0, err
	}
	return b.Repo.List(ctx, f)
}

func (b *brokenRepo) Get(ctx context.Context, id string) (mastersurveyors.Surveyor, error) {
	if err := b.fail["Get"]; err != nil {
		return mastersurveyors.Surveyor{}, err
	}
	return b.Repo.Get(ctx, id)
}

func (b *brokenRepo) FindByNameKey(ctx context.Context, key string) ([]mastersurveyors.Surveyor, error) {
	if err := b.fail["FindByNameKey"]; err != nil {
		return nil, err
	}
	if b.override {
		return b.byName, nil
	}
	return b.Repo.FindByNameKey(ctx, key)
}

func (b *brokenRepo) FindByAppLogin(ctx context.Context, login string) ([]mastersurveyors.Surveyor, error) {
	if err := b.fail["FindByAppLogin"]; err != nil {
		return nil, err
	}
	if b.override {
		return b.byLogin, nil
	}
	return b.Repo.FindByAppLogin(ctx, login)
}

func (b *brokenRepo) Insert(ctx context.Context, s mastersurveyors.Surveyor) (mastersurveyors.Surveyor, error) {
	if err := b.fail["Insert"]; err != nil {
		return mastersurveyors.Surveyor{}, err
	}
	return b.Repo.Insert(ctx, s)
}

func (b *brokenRepo) Update(ctx context.Context, s mastersurveyors.Surveyor) error {
	if err := b.fail["Update"]; err != nil {
		return err
	}
	return b.Repo.Update(ctx, s)
}

type failingCommittee struct{ err error }

func (f failingCommittee) Resolve(context.Context, string, mastersurveyors.Surveyor) (string, error) {
	return "", f.err
}

type failingAccounts struct{ err error }

func (f failingAccounts) Register(context.Context, string, mastersurveyors.AccountRequest) error {
	return f.err
}

type parts struct {
	committee mastersurveyors.CommitteeResolver
	accounts  mastersurveyors.AccountRegistrar
}

func brokenService(t *testing.T, repo *brokenRepo, p parts) *usecase.Service {
	t.Helper()
	if p.committee == nil {
		p.committee = committee.Fixed{Identity: "KOMITEUJI"}
	}
	if p.accounts == nil {
		p.accounts = account.NewRecorder(nil)
	}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastersurveyors.Repo, error) { return repo, nil },
		Committee:    p.committee,
		Accounts:     p.accounts,
	})
	require.NoError(t, err)
	return service
}

func newBroken(fail map[string]error) *brokenRepo {
	if fail == nil {
		fail = map[string]error{}
	}
	return &brokenRepo{Repo: memory.NewRepo(), fail: fail}
}

func TestListDanGet(t *testing.T) {
	service := brokenService(t, newBroken(nil), parts{})
	ctx := context.Background()

	rows, total, err := service.List(ctx, portalAlias, mastersurveyors.Filter{})
	require.NoError(t, err)
	require.Equal(t, len(rows), total)
	require.NotZero(t, total)

	got, err := service.Get(ctx, portalAlias, " 1000001 ")
	require.NoError(t, err)
	require.Equal(t, "1000001", got.ID)

	_, err = service.Get(ctx, portalAlias, "999")
	require.ErrorIs(t, err, mastersurveyors.ErrNotFound)

	require.NoError(t, service.EnsurePortalReady(portalAlias))
}

func TestPortalGagalDiteruskanSetiapOperasi(t *testing.T) {
	errPortal := errors.New("portal belum siap")
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastersurveyors.Repo, error) { return nil, errPortal },
		Committee:    committee.Fixed{},
		Accounts:     account.NewRecorder(nil),
	})
	require.NoError(t, err)
	ctx := context.Background()

	require.ErrorIs(t, service.EnsurePortalReady(portalAlias), errPortal)
	_, err = service.Get(ctx, portalAlias, "1")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Submit(ctx, portalAlias, submission(), submitter())
	require.ErrorIs(t, err, errPortal)
	_, err = service.Update(ctx, portalAlias, "1", submission(), submitter())
	require.ErrorIs(t, err, errPortal)
	_, err = service.Decide(ctx, portalAlias, "1",
		usecase.Decision{Status: mastersurveyors.StatusApproved}, usecase.Committee{})
	require.ErrorIs(t, err, errPortal)
}

func TestGalatRepoDibungkus(t *testing.T) {
	ctx := context.Background()
	decision := usecase.Decision{Status: mastersurveyors.StatusApproved}
	internal := submission()
	internal.TypeCode = mastersurveyors.InternalTypeCode
	internal.AppLogin = "LOGINBARU"

	cases := []struct {
		fail    string
		message string
		call    func(s *usecase.Service) error
	}{
		{"List", "membaca daftar surveyor", func(s *usecase.Service) error {
			_, _, err := s.List(ctx, portalAlias, mastersurveyors.Filter{})
			return err
		}},
		{"Get", "membaca surveyor", func(s *usecase.Service) error {
			_, err := s.Get(ctx, portalAlias, "1000001")
			return err
		}},
		{"Get", "membaca surveyor", func(s *usecase.Service) error {
			_, err := s.Update(ctx, portalAlias, "1000002", submission(), submitter())
			return err
		}},
		{"Get", "membaca surveyor", func(s *usecase.Service) error {
			_, err := s.Decide(ctx, portalAlias, "1000002", decision, usecase.Committee{})
			return err
		}},
		{"FindByNameKey", "memeriksa nama ganda", func(s *usecase.Service) error {
			_, err := s.Submit(ctx, portalAlias, submission(), submitter())
			return err
		}},
		{"FindByAppLogin", "memeriksa login ganda", func(s *usecase.Service) error {
			_, err := s.Submit(ctx, portalAlias, internal, submitter())
			return err
		}},
		{"Insert", "menyimpan surveyor", func(s *usecase.Service) error {
			_, err := s.Submit(ctx, portalAlias, submission(), submitter())
			return err
		}},
		{"Update", "mengubah surveyor", func(s *usecase.Service) error {
			_, err := s.Update(ctx, portalAlias, "1000002", submission(), submitter())
			return err
		}},
		{"Update", "menyimpan keputusan komite", func(s *usecase.Service) error {
			_, err := s.Decide(ctx, portalAlias, "1000002", decision,
				usecase.Committee{Identity: "KOMITECONTOH"})
			return err
		}},
	}

	for _, c := range cases {
		t.Run(c.fail+"/"+c.message, func(t *testing.T) {
			service := brokenService(t, newBroken(map[string]error{c.fail: errRepo}), parts{})
			err := c.call(service)
			require.ErrorIs(t, err, errRepo)
			require.ErrorContains(t, err, c.message)
		})
	}
}

// Galat domain dari repo diteruskan utuh, bukan dibungkus.
func TestGalatDomainDariRepoDiteruskanUtuh(t *testing.T) {
	ctx := context.Background()

	for _, domainErr := range []error{
		mastersurveyors.ErrNameTaken, mastersurveyors.ErrLoginTaken,
		mastersurveyors.ErrNotFound, mastersurveyors.ErrNoSite,
	} {
		service := brokenService(t, newBroken(map[string]error{"Insert": domainErr}), parts{})
		_, err := service.Submit(ctx, portalAlias, submission(), submitter())
		require.Equal(t, domainErr, err)
	}

	service := brokenService(t,
		newBroken(map[string]error{"Get": mastersurveyors.ErrNotFound}), parts{})
	_, err := service.Update(ctx, portalAlias, "1", submission(), submitter())
	require.Equal(t, mastersurveyors.ErrNotFound, err)
	_, err = service.Decide(ctx, portalAlias, "1",
		usecase.Decision{Status: mastersurveyors.StatusRejected}, usecase.Committee{})
	require.Equal(t, mastersurveyors.ErrNotFound, err)
}

func TestSubmitGalatKomiteDanAkun(t *testing.T) {
	ctx := context.Background()
	internal := submission()
	internal.TypeCode = mastersurveyors.InternalTypeCode
	internal.AppLogin = "LOGINBARU"

	service := brokenService(t, newBroken(nil), parts{committee: failingCommittee{err: errRepo}})
	_, err := service.Submit(ctx, portalAlias, submission(), submitter())
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "menetapkan komite")

	service = brokenService(t, newBroken(nil), parts{accounts: failingAccounts{err: errRepo}})
	_, err = service.Submit(ctx, portalAlias, internal, submitter())
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "meminta akun aplikasi untuk \"LOGINBARU\"")

	internal.Name = "Nama Lain"
	service = brokenService(t, newBroken(nil),
		parts{accounts: failingAccounts{err: mastersurveyors.ErrLoginTaken}})
	_, err = service.Submit(ctx, portalAlias, internal, submitter())
	require.Equal(t, mastersurveyors.ErrLoginTaken, err)
}

// Login yang dipakai surveyor lain ditolak sebelum menyimpan.
func TestSubmitMenolakLoginGanda(t *testing.T) {
	service := brokenService(t, newBroken(nil), parts{})
	internal := submission()
	internal.TypeCode = mastersurveyors.InternalTypeCode
	internal.AppLogin = "surveyorsatu"

	_, err := service.Submit(context.Background(), portalAlias, internal, submitter())
	require.ErrorIs(t, err, mastersurveyors.ErrLoginTaken)
}

func TestUpdateValidasiDanBentrok(t *testing.T) {
	ctx := context.Background()

	service := brokenService(t, newBroken(nil), parts{})
	invalid := submission()
	invalid.Email = ""
	_, err := service.Update(ctx, portalAlias, "1000002", invalid, submitter())
	var validation *mastersurveyors.ValidationError
	require.ErrorAs(t, err, &validation)

	clash := submission()
	clash.Name = "Surveyor Contoh Satu"
	_, err = service.Update(ctx, portalAlias, "1000002", clash, submitter())
	require.ErrorIs(t, err, mastersurveyors.ErrNameTaken)

	loginClash := submission()
	loginClash.AppLogin = "SURVEYORSATU"
	_, err = service.Update(ctx, portalAlias, "1000002", loginClash, submitter())
	require.ErrorIs(t, err, mastersurveyors.ErrLoginTaken)
}

// Akun diminta bila login BARU terisi saat menyunting, dan galatnya diteruskan.
func TestUpdateMemintaAkunSaatLoginBerganti(t *testing.T) {
	ctx := context.Background()
	changed := submission()
	changed.AppLogin = "LOGINDUA"

	recorder := account.NewRecorder(nil)
	service := brokenService(t, newBroken(nil), parts{accounts: recorder})
	updated, err := service.Update(ctx, portalAlias, "1000002", changed, submitter())
	require.NoError(t, err)
	require.Equal(t, "LOGINDUA", updated.AppLogin)
	require.True(t, recorder.Taken("LOGINDUA"))

	service = brokenService(t, newBroken(nil), parts{accounts: failingAccounts{err: errRepo}})
	_, err = service.Update(ctx, portalAlias, "1000002", changed, submitter())
	require.ErrorIs(t, err, errRepo)
}

// Baris dengan ID sendiri tidak dianggap bentrok dengan dirinya.
func TestPemeriksaanGandaMengecualikanDiriSendiri(t *testing.T) {
	repo := newBroken(nil)
	repo.override = true
	repo.byName = []mastersurveyors.Surveyor{{ID: "1000002"}}
	repo.byLogin = []mastersurveyors.Surveyor{{ID: "1000002"}}

	service := brokenService(t, repo, parts{})
	changed := submission()
	changed.AppLogin = "LOGINDUA"
	_, err := service.Update(context.Background(), portalAlias, "1000002", changed, submitter())
	require.NoError(t, err)
}
