package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrecovery"
	"claim-pnc/internal/masterrecovery/usecase"
)

// issuerStub adalah penerbit VA yang jawabannya dipatok.
type issuerStub struct {
	va  masterrecovery.VirtualAccount
	err error
}

func (s issuerStub) Issue(context.Context, string, masterrecovery.VirtualAccountRequest) (masterrecovery.VirtualAccount, error) {
	return s.va, s.err
}

// repoStub membungkus Repo dan menimpa sebagian jawabannya.
type repoStub struct {
	masterrecovery.Repo
	saveDocumentID  string
	findPrincipal   error
	savePrincipal   error
	insertErr       error
	lookupPolicyErr error
}

func (r repoStub) SaveDocument(context.Context, masterrecovery.Document) (string, error) {
	return r.saveDocumentID, nil
}

func (r repoStub) FindPrincipal(ctx context.Context, clientID, name string) (masterrecovery.Principal, error) {
	if r.findPrincipal != nil {
		return masterrecovery.Principal{}, r.findPrincipal
	}
	return r.Repo.FindPrincipal(ctx, clientID, name)
}

func (r repoStub) SavePrincipal(ctx context.Context, p masterrecovery.Principal) error {
	if r.savePrincipal != nil {
		return r.savePrincipal
	}
	return r.Repo.SavePrincipal(ctx, p)
}

func (r repoStub) Insert(ctx context.Context, rec masterrecovery.Recovery) (masterrecovery.Recovery, error) {
	if r.insertErr != nil {
		return masterrecovery.Recovery{}, r.insertErr
	}
	return r.Repo.Insert(ctx, rec)
}

func (r repoStub) LookupPolicy(ctx context.Context, policyNo string) (masterrecovery.PolicyReference, error) {
	if r.lookupPolicyErr != nil {
		return masterrecovery.PolicyReference{}, r.lookupPolicyErr
	}
	return r.Repo.LookupPolicy(ctx, policyNo)
}

func serviceWith(t *testing.T, repo masterrecovery.Repo, issuer masterrecovery.VirtualAccountIssuer) *usecase.Service {
	t.Helper()
	s, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterrecovery.Repo, error) { return repo, nil },
		Issuer:       issuer,
	})
	require.NoError(t, err)
	return s
}

func validRecovery() masterrecovery.Recovery {
	return masterrecovery.Recovery{
		PrincipalName: "PT A", Year: "2026", Remark: "r", CasePosition: "p", ClaimAmount: 10,
	}
}

var errBroken = errors.New("rusak")

func TestNewServiceRequiresIssuer(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterrecovery.Repo, error) { return nil, nil },
	})
	require.ErrorContains(t, err, "Issuer wajib diisi")
}

func TestUnknownPortalRejectedByEveryOperation(t *testing.T) {
	service, _, _ := bangun(t)
	ctx := context.Background()

	_, err := service.NextBatch(ctx, "LAIN")
	require.ErrorContains(t, err, "portal tidak tersedia")
	_, err = service.Principals(ctx, "LAIN")
	require.ErrorContains(t, err, "portal tidak tersedia")
	_, _, err = service.List(ctx, "LAIN", masterrecovery.ListFilter{})
	require.ErrorContains(t, err, "portal tidak tersedia")
	_, err = service.Document(ctx, "LAIN", "x")
	require.ErrorContains(t, err, "portal tidak tersedia")
	_, err = service.LookupPolicy(ctx, "LAIN", "x")
	require.ErrorContains(t, err, "portal tidak tersedia")
	_, err = service.IssueVirtualAccount(ctx, "LAIN", masterrecovery.VirtualAccountRequest{})
	require.ErrorContains(t, err, "portal tidak tersedia")
	_, err = service.SaveDocument(ctx, "LAIN", masterrecovery.Document{})
	require.ErrorContains(t, err, "portal tidak tersedia")
	_, _, err = service.ReadClaimLine("LAIN", strings.NewReader(""))
	require.ErrorContains(t, err, "portal tidak tersedia")

	err = service.EnsurePortalReady("LAIN")
	require.ErrorContains(t, err, `portal "LAIN" tidak dapat dilayani`)
	require.NoError(t, service.EnsurePortalReady(portalUji))
}

func TestNextBatchAndPrincipals(t *testing.T) {
	service, repo, _ := bangun(t)
	ctx := context.Background()

	batch, err := service.NextBatch(ctx, portalUji)
	require.NoError(t, err)
	require.Equal(t, int64(1), batch)

	list, err := service.Principals(ctx, portalUji)
	require.NoError(t, err)
	require.Len(t, list, 2)

	repo.SetError(errBroken)
	_, err = service.NextBatch(ctx, portalUji)
	require.ErrorIs(t, err, errBroken)
	require.ErrorContains(t, err, "nomor batch berikutnya")
	_, err = service.Principals(ctx, portalUji)
	require.ErrorContains(t, err, "daftar principal")
}

func TestListGroupsByPrincipalAndClampsPaging(t *testing.T) {
	service, repo, _ := bangun(t)
	ctx := context.Background()

	for _, rec := range []masterrecovery.Recovery{
		{PrincipalName: "PT B", Year: "2026", Remark: "r", CasePosition: "p", ClaimAmount: 5},
		{PrincipalName: "PT A", Year: "2026", Remark: "r", CasePosition: "p", ClaimAmount: 10},
		{PrincipalName: "PT A", Year: "2026", Remark: "r", CasePosition: "p", ClaimAmount: 20},
	} {
		_, err := service.Save(ctx, portalUji, rec)
		require.NoError(t, err)
	}

	// Limit negatif → baku; offset negatif → nol.
	groups, total, err := service.List(ctx, portalUji, masterrecovery.ListFilter{Limit: -1, Offset: -5})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, groups, 2)
	require.Equal(t, "PT A", groups[0].Name)
	require.Len(t, groups[0].Batch, 2)
	// Latest adalah batch terakhir, bukan jumlah.
	require.Equal(t, groups[0].Batch[1], groups[0].Latest)
	require.Equal(t, "PT B", groups[1].Name)

	// Limit di atas batas → dipotong ke MaxListLimit (tetap menampung semua).
	groups, _, err = service.List(ctx, portalUji, masterrecovery.ListFilter{Limit: 1000})
	require.NoError(t, err)
	require.Len(t, groups, 2)

	repo.SetError(errBroken)
	_, _, err = service.List(ctx, portalUji, masterrecovery.ListFilter{})
	require.ErrorContains(t, err, "daftar batch recovery")
}

func TestDocumentTrimsIDAndPassesNotFound(t *testing.T) {
	service, repo, _ := bangun(t)
	ctx := context.Background()

	id, err := repo.SaveDocument(ctx, masterrecovery.Document{Name: "a.pdf", Content: []byte("x")})
	require.NoError(t, err)

	doc, err := service.Document(ctx, portalUji, "  "+id+"  ")
	require.NoError(t, err)
	require.Equal(t, "a.pdf", doc.Name)

	_, err = service.Document(ctx, portalUji, "tidak-ada")
	require.ErrorIs(t, err, masterrecovery.ErrDocumentNotFound)
}

func TestLookupPolicy(t *testing.T) {
	service, repo, _ := bangun(t)
	ctx := context.Background()

	ref, err := service.LookupPolicy(ctx, portalUji, " CONTOH-POLIS-0002 ")
	require.NoError(t, err)
	require.Equal(t, "02", ref.BusinessID)

	_, err = service.LookupPolicy(ctx, portalUji, "   ")
	require.ErrorIs(t, err, masterrecovery.ErrPolicyNotFound)

	_, err = service.LookupPolicy(ctx, portalUji, "TIDAK-ADA")
	require.ErrorIs(t, err, masterrecovery.ErrPolicyNotFound)

	repo.SetError(errBroken)
	_, err = service.LookupPolicy(ctx, portalUji, "X1")
	require.ErrorIs(t, err, errBroken)
	require.ErrorContains(t, err, `mencari polis "X1"`)
}

func TestIssueVirtualAccountFailurePaths(t *testing.T) {
	ctx := context.Background()
	request := masterrecovery.VirtualAccountRequest{ClientID: "NEW", PrincipalName: "PT NEW", Email: "a@b.c"}

	t.Run("validation", func(t *testing.T) {
		service, _, _ := bangun(t)
		_, err := service.IssueVirtualAccount(ctx, portalUji, masterrecovery.VirtualAccountRequest{})
		var verr *masterrecovery.ValidationError
		require.ErrorAs(t, err, &verr)
		require.Len(t, verr.Violation, 3)
	})
	t.Run("find principal fails", func(t *testing.T) {
		_, repo, _ := bangun(t)
		service := serviceWith(t, repoStub{Repo: repo, findPrincipal: errBroken}, issuerStub{})
		_, err := service.IssueVirtualAccount(ctx, portalUji, request)
		require.ErrorIs(t, err, errBroken)
		require.ErrorContains(t, err, "memeriksa principal")
	})
	t.Run("issuer known error passes through", func(t *testing.T) {
		_, repo, _ := bangun(t)
		service := serviceWith(t, repo, issuerStub{err: masterrecovery.ErrIssuerUnconfigured})
		_, err := service.IssueVirtualAccount(ctx, portalUji, request)
		require.Equal(t, masterrecovery.ErrIssuerUnconfigured, err)
	})
	t.Run("issuer unknown error wrapped", func(t *testing.T) {
		_, repo, _ := bangun(t)
		service := serviceWith(t, repo, issuerStub{err: errBroken})
		_, err := service.IssueVirtualAccount(ctx, portalUji, request)
		require.ErrorIs(t, err, errBroken)
		require.ErrorContains(t, err, "menerbitkan virtual account")
	})
	t.Run("issuer returns empty number", func(t *testing.T) {
		_, repo, _ := bangun(t)
		service := serviceWith(t, repo, issuerStub{va: masterrecovery.VirtualAccount{Number: "  "}})
		_, err := service.IssueVirtualAccount(ctx, portalUji, request)
		require.ErrorIs(t, err, masterrecovery.ErrIssuerRejected)
	})
	t.Run("save principal fails", func(t *testing.T) {
		_, repo, _ := bangun(t)
		service := serviceWith(t, repoStub{Repo: repo, savePrincipal: errBroken}, issuerStub{va: masterrecovery.VirtualAccount{Number: "1"}})
		_, err := service.IssueVirtualAccount(ctx, portalUji, request)
		require.ErrorIs(t, err, errBroken)
		require.ErrorContains(t, err, "mencatat virtual account")
	})
}

func TestSaveDocumentFailurePaths(t *testing.T) {
	ctx := context.Background()
	doc := masterrecovery.Document{Name: "a.pdf", Content: []byte("x")}

	service, repo, _ := bangun(t)
	_, err := service.SaveDocument(ctx, portalUji, masterrecovery.Document{})
	var verr *masterrecovery.ValidationError
	require.ErrorAs(t, err, &verr)

	repo.SetError(errBroken)
	_, err = service.SaveDocument(ctx, portalUji, doc)
	require.ErrorIs(t, err, masterrecovery.ErrDocumentNotSaved)
	require.ErrorContains(t, err, "rusak")

	_, repo2, _ := bangun(t)
	service = serviceWith(t, repoStub{Repo: repo2, saveDocumentID: " "}, issuerStub{})
	_, err = service.SaveDocument(ctx, portalUji, doc)
	require.Equal(t, masterrecovery.ErrDocumentNotSaved, err)
}

func TestSaveInsertFailures(t *testing.T) {
	ctx := context.Background()

	_, repo, _ := bangun(t)
	service := serviceWith(t, repoStub{Repo: repo, insertErr: masterrecovery.ErrBatchTaken}, issuerStub{})
	_, err := service.Save(ctx, portalUji, validRecovery())
	require.Equal(t, masterrecovery.ErrBatchTaken, err)

	service = serviceWith(t, repoStub{Repo: repo, insertErr: errBroken}, issuerStub{})
	_, err = service.Save(ctx, portalUji, validRecovery())
	require.ErrorIs(t, err, errBroken)
	require.ErrorContains(t, err, "menyimpan batch recovery")

	// Kegagalan mencari polis tidak membatalkan penyimpanan; identitasnya kosong.
	service = serviceWith(t, repoStub{Repo: repo, lookupPolicyErr: errBroken}, issuerStub{})
	rec := validRecovery()
	rec.PolicyNo = "CONTOH-POLIS-0001"
	saved, err := service.Save(ctx, portalUji, rec)
	require.NoError(t, err)
	require.Empty(t, saved.BusinessID)
}
