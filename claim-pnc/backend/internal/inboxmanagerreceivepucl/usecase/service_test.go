package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/inboxmanagerreceivepucl/repo/memory"
	"claim-pnc/internal/inboxmanagerreceivepucl/usecase"
)

// failingRepo adalah penyimpanan yang selalu gagal dengan galat yang ditentukan.
type failingRepo struct{ err error }

func (f failingRepo) List(
	context.Context, inboxmanagerreceivepucl.Query, inboxmanagerreceivepucl.Pagination,
) (inboxmanagerreceivepucl.Page, error) {
	return inboxmanagerreceivepucl.Page{}, f.err
}

func (f failingRepo) Document(context.Context, string) (inboxmanagerreceivepucl.ReceiveDocument, error) {
	return inboxmanagerreceivepucl.ReceiveDocument{}, f.err
}

func newService(t *testing.T, repo inboxmanagerreceivepucl.Repo, logs *bytes.Buffer) *usecase.Service {
	t.Helper()
	var logger *slog.Logger
	if logs != nil {
		logger = slog.New(slog.NewJSONHandler(logs, nil))
	}
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxmanagerreceivepucl.Repo, error) {
			if alias != "ASM" {
				return nil, errors.New("portal belum siap")
			}
			return repo, nil
		},
		Logger: logger,
	})
	require.NoError(t, err)
	return svc
}

var caller = inboxmanagerreceivepucl.Caller{Login: "penyelia"}

func TestNewServiceRequiresRepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.ErrorContains(t, err, "RepoSelector wajib diisi")
}

func TestMetadataReturnsTabsDefaultAndPlannedDifferences(t *testing.T) {
	meta := newService(t, memory.NewSampleStore(), nil).Metadata()
	require.Equal(t, inboxmanagerreceivepucl.Tabs(), meta.Tabs)
	require.Equal(t, inboxmanagerreceivepucl.DefaultTab, meta.DefaultTab)
	require.Equal(t, inboxmanagerreceivepucl.PlannedDifferences, meta.PlannedDifferences)
}

func TestListDefaultsToReceiveTabAndLogsOpening(t *testing.T) {
	var logs bytes.Buffer
	listed, err := newService(t, memory.NewSampleStore(), &logs).List(
		context.Background(), "ASM", caller, inboxmanagerreceivepucl.QueryInput{},
		inboxmanagerreceivepucl.Pagination{})
	require.NoError(t, err)
	require.Equal(t, inboxmanagerreceivepucl.TabReceive, listed.Query.Tab.Code)
	require.NotEmpty(t, listed.Page.Items)
	require.Contains(t, logs.String(), "pandangan penyelia antrean receive/RCL-PUCL dibuka")
	require.Contains(t, logs.String(), `"pemanggil":"penyelia"`)
}

func TestListRejectsUnknownCallerAndTab(t *testing.T) {
	svc := newService(t, memory.NewSampleStore(), nil)

	_, err := svc.List(context.Background(), "ASM", inboxmanagerreceivepucl.Caller{},
		inboxmanagerreceivepucl.QueryInput{}, inboxmanagerreceivepucl.Pagination{})
	require.ErrorIs(t, err, inboxmanagerreceivepucl.ErrCallerUnknown)

	_, err = svc.List(context.Background(), "ASM", caller,
		inboxmanagerreceivepucl.QueryInput{Tab: "99"}, inboxmanagerreceivepucl.Pagination{})
	var validation *inboxmanagerreceivepucl.ValidationError
	require.ErrorAs(t, err, &validation)
}

func TestListPropagatesSelectorAndRepoErrors(t *testing.T) {
	_, err := newService(t, memory.NewSampleStore(), nil).List(context.Background(), "ASI", caller,
		inboxmanagerreceivepucl.QueryInput{}, inboxmanagerreceivepucl.Pagination{})
	require.EqualError(t, err, "portal belum siap")

	boom := errors.New("koneksi putus")
	_, err = newService(t, failingRepo{err: boom}, nil).List(context.Background(), "ASM", caller,
		inboxmanagerreceivepucl.QueryInput{Tab: inboxmanagerreceivepucl.TabRCLPUCL},
		inboxmanagerreceivepucl.Pagination{})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "mengambil isi tab 2")
}

func TestDocumentReturnsDetailGroupsActionsAndLogs(t *testing.T) {
	var logs bytes.Buffer
	doc, err := newService(t, memory.NewSampleStore(), &logs).Document(
		context.Background(), "ASM", caller, " ASM-FW-GCNMFW-WORK RCV-900001 ")
	require.NoError(t, err)
	require.Equal(t, "RCV-900001", doc.Detail.CaseID)

	// Bentuk layar disusun UNTUK BERKAS INI, bukan daftar tetap: sebagian isian punya syarat
	// tampil yang bergantung pada Group Panel dan nomor polis berkas yang sedang dibuka.
	require.Equal(t, inboxmanagerreceivepucl.DocumentFieldGroupsFor(doc.Detail), doc.Groups)
	require.Equal(t, inboxmanagerreceivepucl.DocumentWriteActionList(), doc.Actions)
	require.Contains(t, logs.String(), "layar kerja penerimaan dokumen dibuka")
}

func TestDocumentValidatesCallerAndReference(t *testing.T) {
	svc := newService(t, memory.NewSampleStore(), nil)

	_, err := svc.Document(context.Background(), "ASM", inboxmanagerreceivepucl.Caller{}, "K")
	require.ErrorIs(t, err, inboxmanagerreceivepucl.ErrCallerUnknown)

	_, err = svc.Document(context.Background(), "ASM", caller, "  ")
	require.ErrorIs(t, err, inboxmanagerreceivepucl.ErrReferenceRequired)

	_, err = svc.Document(context.Background(), "ASI", caller, "K")
	require.EqualError(t, err, "portal belum siap")
}

func TestDocumentPassesDomainErrorsUnwrappedAndWrapsOthers(t *testing.T) {
	_, err := newService(t, memory.NewSampleStore(), nil).Document(
		context.Background(), "ASM", caller, "TIDAK-ADA")
	require.Equal(t, inboxmanagerreceivepucl.ErrDocumentNotFound, err)

	boom := errors.New("kueri gagal")
	_, err = newService(t, failingRepo{err: boom}, nil).Document(
		context.Background(), "ASM", caller, "K")
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "mengambil berkas penerimaan dokumen")
}
