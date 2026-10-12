package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/monitoringslinkojk/repo/memory"
)

// Uji di berkas ini melengkapi memory_test.go: penyaring Business Name, paginasi, dan
// seluruh aksi tulis penyimpanan memori.

func cancelledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func claimsOf(page monitoringslinkojk.Page) []string {
	out := make([]string, 0, len(page.Rows))
	for _, r := range page.Rows {
		out = append(out, r.Get("no_klaim")+"|"+r.Get("contract_no"))
	}
	return out
}

func TestBusinessScopeFilterInvertsForSuretyBond(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	kredit, err := repo.Search(ctx, monitoringslinkojk.SegmentD01,
		monitoringslinkojk.Filter{BusinessScope: monitoringslinkojk.ScopeCreditInsurance}.Normalize())
	require.NoError(t, err)
	require.Equal(t, 3, kredit.Total)

	// SURETY BOND berarti seluruh lini SELAIN Asuransi Kredit.
	surety, err := repo.Search(ctx, monitoringslinkojk.SegmentD01,
		monitoringslinkojk.Filter{BusinessScope: monitoringslinkojk.ScopeSuretyBond}.Normalize())
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0207|KTR-2026-0207"}, claimsOf(surety))
}

func TestSearchPaginationAndTieBreakOnRowKey(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	page, err := repo.Search(ctx, monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 4, page.Total)
	require.Equal(t, []string{"PNCN.26.0101|KTR-2026-0102", "PNCN.26.0033|KTR-2026-0033"}, claimsOf(page))

	// Halaman di luar jangkauan: kosong, total tetap.
	page, err = repo.Search(ctx, monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{Page: 9, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 4, page.Total)
	require.NotNil(t, page.Rows)
	require.Empty(t, page.Rows)

	// Halaman terakhir yang tidak penuh.
	page, err = repo.Search(ctx, monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{Page: 2, Size: 3})
	require.NoError(t, err)
	require.Len(t, page.Rows, 1)
}

func TestStreamStopsOnEmitErrorAndRespectsContext(t *testing.T) {
	repo := memory.NewSampleRepo()
	errStop := errors.New("berhenti")

	var seen int
	err := repo.Stream(context.Background(), monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{},
		func(row monitoringslinkojk.Row) error {
			seen++
			require.Equal(t, "L", row.Get("jenis_kelamin"))
			return errStop
		})
	require.ErrorIs(t, err, errStop)
	require.Equal(t, 1, seen)

	err = repo.Stream(cancelledCtx(), monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{},
		func(monitoringslinkojk.Row) error { return nil })
	require.ErrorIs(t, err, context.Canceled)
}

func TestReportsInsertAndCount(t *testing.T) {
	repo := memory.NewRepo()
	ctx := context.Background()

	n, err := repo.CountReport(ctx, "PNCN.26.0001")
	require.NoError(t, err)
	require.Zero(t, n)

	require.NoError(t, repo.InsertReport(ctx, monitoringslinkojk.ReportEntry{ClaimID: " PNCN.26.0001 ", ContractNo: " K "}))
	require.NoError(t, repo.InsertReport(ctx, monitoringslinkojk.ReportEntry{ClaimID: "PNCN.26.0001", ContractNo: "K2"}))
	require.NoError(t, repo.InsertReport(ctx, monitoringslinkojk.ReportEntry{ClaimID: "LAIN"}))

	n, err = repo.CountReport(ctx, " PNCN.26.0001 ")
	require.NoError(t, err)
	require.Equal(t, 2, n)

	reports := repo.Reports()
	require.Len(t, reports, 3)
	// Isian dirapikan saat disimpan.
	require.Equal(t, "PNCN.26.0001", reports[0].ClaimID)
	require.Equal(t, "K", reports[0].ContractNo)

	// Salinan: mengubahnya tidak menyentuh penyimpanan.
	reports[0].ClaimID = "DIUBAH"
	require.Equal(t, "PNCN.26.0001", repo.Reports()[0].ClaimID)
}

func TestWriteOperationsRespectCancelledContext(t *testing.T) {
	repo := memory.NewRepo()
	ctx := cancelledCtx()

	_, err := repo.CountReport(ctx, "A")
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, repo.InsertReport(ctx, monitoringslinkojk.ReportEntry{}), context.Canceled)
	require.ErrorIs(t, repo.StreamSource(ctx, monitoringslinkojk.Filter{},
		func(monitoringslinkojk.ReportEntry) error { return nil }), context.Canceled)
	_, err = repo.NextSubmissionID(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, repo.RecordSubmission(ctx, monitoringslinkojk.Submission{}), context.Canceled)
	require.ErrorIs(t, repo.CompleteSubmission(ctx, monitoringslinkojk.Submission{},
		monitoringslinkojk.SubmissionResult{}), context.Canceled)
}

func TestStreamSourceDerivesEntriesFromD01Rows(t *testing.T) {
	repo := memory.NewSampleRepo()

	var entries []monitoringslinkojk.ReportEntry
	err := repo.StreamSource(context.Background(),
		monitoringslinkojk.Filter{BusinessScope: monitoringslinkojk.ScopeSuretyBond},
		func(e monitoringslinkojk.ReportEntry) error {
			entries = append(entries, e)
			return nil
		})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	require.Equal(t, "PNCN.26.0207", e.ClaimID)
	require.Equal(t, "KTR-2026-0207", e.ContractNo)
	require.Equal(t, "9900000207", e.FacilityAccountNo)
	require.Equal(t, "CIF0000207", e.DebtorCIF)
	require.Equal(t, "15/02/2026", e.PolicyStart)
	require.Equal(t, "500000000", e.Obligation)
	require.Equal(t, "POL-CONTOH-0207", e.PolicyNo)
	// Kolom yang di produksi tidak tersedia tetap kosong.
	require.Empty(t, e.ArrearsDays)
	require.Empty(t, e.IDCardNo)
	require.Empty(t, e.ClientID)

	errStop := errors.New("berhenti")
	err = repo.StreamSource(context.Background(), monitoringslinkojk.Filter{},
		func(monitoringslinkojk.ReportEntry) error { return errStop })
	require.ErrorIs(t, err, errStop)
}

func TestSubmissionsLifecycle(t *testing.T) {
	repo := memory.NewRepo()
	ctx := context.Background()

	id, err := repo.NextSubmissionID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), id)

	require.NoError(t, repo.RecordSubmission(ctx, monitoringslinkojk.Submission{ID: 5, ClaimID: "A", ContractNo: "K"}))
	require.NoError(t, repo.RecordSubmission(ctx, monitoringslinkojk.Submission{ID: 2, ClaimID: "B"}))

	// max + 1, bukan jumlah + 1.
	id, err = repo.NextSubmissionID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(6), id)

	result := monitoringslinkojk.SubmissionResult{ClientID: "C1", TransactionID: "T1"}
	require.NoError(t, repo.CompleteSubmission(ctx, monitoringslinkojk.Submission{ID: 5, ClaimID: "A"}, result))
	// Pasangan yang tidak cocok diabaikan tanpa galat.
	require.NoError(t, repo.CompleteSubmission(ctx, monitoringslinkojk.Submission{ID: 2, ClaimID: "A"}, result))

	subs := repo.Submissions()
	require.Len(t, subs, 2)
	require.True(t, subs[0].Completed)
	require.Equal(t, result, subs[0].Result)
	require.Equal(t, "K", subs[0].ContractNo)
	require.False(t, subs[1].Completed)
}
