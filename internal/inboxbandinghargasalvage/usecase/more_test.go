package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
	"claim-pnc/internal/inboxbandinghargasalvage/repo/memory"
	"claim-pnc/internal/inboxbandinghargasalvage/usecase"
)

// repoRusak menjawab setiap pembacaan dengan galat.
type repoRusak struct{ err error }

func (r repoRusak) List(
	context.Context, inboxbandinghargasalvage.Query, inboxbandinghargasalvage.Pagination,
) (inboxbandinghargasalvage.Page, error) {
	return inboxbandinghargasalvage.Page{}, r.err
}

func (r repoRusak) Count(context.Context, inboxbandinghargasalvage.Query) (int, error) {
	return 0, r.err
}

func (r repoRusak) ListDecisions(
	context.Context, inboxbandinghargasalvage.DecisionQuery,
) ([]inboxbandinghargasalvage.Decision, error) {
	return nil, r.err
}

// layananLengkap merakit layanan dengan pembaca, penulis, dan pembaca dokumen di atas data
// contoh, beserta penyangga log.
func layananLengkap(t *testing.T) (*usecase.Service, *bytes.Buffer) {
	t.Helper()

	store := memory.NewSampleStore()
	writer := memory.NewWriter(store)
	documents := memory.NewSampleDocumentStore(store)
	log := &bytes.Buffer{}

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxbandinghargasalvage.Repo, error) {
			if alias != portalUtama {
				return nil, errPortalTidakDikenal
			}
			return store, nil
		},
		WriterSelector: func(string) (inboxbandinghargasalvage.Writer, error) {
			return writer, nil
		},
		DocumentSelector: func(alias string) (inboxbandinghargasalvage.DocumentReader, error) {
			if alias != portalUtama {
				return nil, errPortalTidakDikenal
			}
			return documents, nil
		},
		Logger: slog.New(slog.NewTextHandler(log, nil)),
	})
	require.NoError(t, err)
	return service, log
}

// Galat penyimpanan dibungkus dengan tab atau klaim yang sedang dibaca.
func TestReadFailuresAreWrappedWithWhatWasBeingRead(t *testing.T) {
	ctx := context.Background()
	failure := errors.New("oracle mati")
	service := layanan(t, repoRusak{err: failure})
	caller := komite(memory.SampleOwner)

	_, err := service.List(ctx, portalUtama, caller,
		inboxbandinghargasalvage.QueryInput{}, inboxbandinghargasalvage.Pagination{})
	require.ErrorIs(t, err, failure)
	require.Contains(t, err.Error(), "mengambil isi tab")

	_, err = service.Summary(ctx, portalUtama, caller)
	require.ErrorIs(t, err, failure)
	require.Contains(t, err.Error(), "menghitung tab")

	_, err = service.Decisions(ctx, portalUtama, caller, "PNCN.26.0451")
	require.ErrorIs(t, err, failure)
	require.Contains(t, err.Error(), "mengambil keputusan klaim PNCN.26.0451")
}

// Ringkasan dan panel rincian menolak portal tak dikenal dan pemanggil tanpa identitas.
func TestSummaryAndDecisionsRejectBadCallsBeforeReading(t *testing.T) {
	ctx := context.Background()
	service := layanan(t, memory.NewSampleStore())

	_, err := service.Summary(ctx, "ENTAH", komite(memory.SampleOwner))
	require.ErrorIs(t, err, errPortalTidakDikenal)

	_, err = service.Summary(ctx, portalUtama, komite(" "))
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)

	_, err = service.Decisions(ctx, portalUtama, komite(""), "PNCN.26.0451")
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)

	_, err = service.Decisions(ctx, "ENTAH", komite(memory.SampleOwner), "PNCN.26.0451")
	require.ErrorIs(t, err, errPortalTidakDikenal)
}

// Membuka antrean komite lain atas aturan bernama orang DICATAT.
func TestOpeningADelegatedQueueIsLogged(t *testing.T) {
	service, log := layananLengkap(t)

	listed, err := service.List(context.Background(), portalUtama, komite("MARIATRIELSA"),
		inboxbandinghargasalvage.QueryInput{}, inboxbandinghargasalvage.Pagination{})
	require.NoError(t, err)
	require.True(t, listed.Query.Reviewer.Delegated)

	require.Contains(t, log.String(), "antrean banding harga dibuka atas nama komite lain")
	require.Contains(t, log.String(), "antrean_milik=BAMBANGSETIADJIGUNAWAN")
}

// Pembukaan antrean sendiri tidak dicatat.
func TestOpeningOwnQueueIsNotLogged(t *testing.T) {
	service, log := layananLengkap(t)

	_, err := service.List(context.Background(), portalUtama, komite(memory.SampleOwner),
		inboxbandinghargasalvage.QueryInput{}, inboxbandinghargasalvage.Pagination{})
	require.NoError(t, err)
	require.Empty(t, log.String())
}

// Keputusan yang tersimpan dicatat beserta langkah yang benar-benar berjalan.
func TestASavedDecisionIsLogged(t *testing.T) {
	service, log := layananLengkap(t)

	result, err := service.Decide(context.Background(), portalUtama,
		komite(memory.SampleOwner), isian("PNC-0451/1", "451", false))
	require.NoError(t, err)
	require.True(t, result.Recorded)

	require.Contains(t, log.String(), "keputusan banding harga salvage dicatat")
	require.Contains(t, log.String(), "disetujui=false")
	require.Contains(t, log.String(), "id_detail_salvage=PNC-0451/1")
}

// Galat penulis diteruskan apa adanya.
func TestDecideReturnsTheWriterError(t *testing.T) {
	failure := errors.New("penulis rusak")
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxbandinghargasalvage.Repo, error) {
			return memory.NewSampleStore(), nil
		},
		WriterSelector: func(string) (inboxbandinghargasalvage.Writer, error) {
			return nil, failure
		},
	})
	require.NoError(t, err)

	_, err = service.Decide(context.Background(), portalUtama, komite(memory.SampleOwner),
		isian("PNC-0451/1", "451", true))
	require.ErrorIs(t, err, failure)
}

// Dialog Lihat File menyerahkan daftar dan isi dokumen milik komite pemanggil.
func TestDocumentsAndContentForTheOwnerCommittee(t *testing.T) {
	service, _ := layananLengkap(t)
	ctx := context.Background()
	caller := komite(memory.SampleOwner)

	rows, err := service.Documents(ctx, portalUtama, caller, "PNC-0451/1", "451")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "9001", rows[0].ID)

	content, err := service.DocumentContent(ctx, portalUtama, caller,
		"PNC-0451/1", "451", "9001")
	require.NoError(t, err)
	require.Equal(t, "penawaran-balai-lelang.pdf", content.Name)
	require.Equal(t, "application/pdf", content.MIMEType)
}

func TestDocumentsRejectBadCalls(t *testing.T) {
	service, _ := layananLengkap(t)
	ctx := context.Background()
	caller := komite(memory.SampleOwner)

	_, err := service.Documents(ctx, portalUtama, komite(""), "PNC-0451/1", "451")
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)

	_, err = service.Documents(ctx, "ENTAH", caller, "PNC-0451/1", "451")
	require.ErrorIs(t, err, errPortalTidakDikenal)

	_, err = service.DocumentContent(ctx, portalUtama, komite(""), "PNC-0451/1", "451", "9001")
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)

	_, err = service.DocumentContent(ctx, portalUtama, caller, "PNC-0451/1", "451", "  ")
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrDocumentNotFound)

	_, err = service.DocumentContent(ctx, "ENTAH", caller, "PNC-0451/1", "451", "9001")
	require.ErrorIs(t, err, errPortalTidakDikenal)

	// Tanpa pembaca dokumen terpasang, dialognya menjawab dengan alasan.
	readOnly := layanan(t, memory.NewSampleStore())
	_, err = readOnly.Documents(ctx, portalUtama, caller, "PNC-0451/1", "451")
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrWriteNotAvailable)
	_, err = readOnly.DocumentContent(ctx, portalUtama, caller, "PNC-0451/1", "451", "9001")
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrWriteNotAvailable)
}
