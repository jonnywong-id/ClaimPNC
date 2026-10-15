package inboxmanager_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

var supervisor = inboxmanager.Caller{Login: "MGR"}

func violationsOf(t *testing.T, err error) []inboxmanager.Violation {
	t.Helper()
	var validation *inboxmanager.ValidationError
	require.ErrorAs(t, err, &validation)
	return validation.Violations
}

func TestVerdictValue(t *testing.T) {
	require.Equal(t, inboxmanager.StatusApproved, inboxmanager.VerdictApprove.Value())
	require.Equal(t, inboxmanager.StatusRejected, inboxmanager.VerdictReject.Value())
	require.Equal(t, inboxmanager.StatusRejected, inboxmanager.Verdict("lain").Value())
}

func TestNewDecisionOrderOfChecks(t *testing.T) {
	_, err := inboxmanager.NewDecision(inboxmanager.TabMasterBengkel, inboxmanager.VerdictApprove,
		[]string{"A"}, "", inboxmanager.Caller{Login: " "})
	require.ErrorIs(t, err, inboxmanager.ErrCallerUnknown)

	_, err = inboxmanager.NewDecision("99", inboxmanager.VerdictApprove, []string{"A"}, "", supervisor)
	require.Equal(t, []inboxmanager.Violation{{Field: inboxmanager.FieldTab, Message: "Tab tidak dikenal."}},
		violationsOf(t, err))

	// Nomor Rangka dibatasi NONMBU.
	_, err = inboxmanager.NewDecision(inboxmanager.TabNomorRangka, inboxmanager.VerdictApprove,
		[]string{"A"}, "", inboxmanager.Caller{Login: "MGR", LineBusiness: inboxmanager.LinePA})
	require.ErrorIs(t, err, inboxmanager.ErrTabNotAllowed)

	_, err = inboxmanager.NewDecision(inboxmanager.TabApprovalMaster, inboxmanager.VerdictApprove,
		[]string{"A"}, "", supervisor)
	require.ErrorIs(t, err, inboxmanager.ErrQueueNotDecidable)

	_, err = inboxmanager.NewDecision(inboxmanager.TabPaymentAkseptasi, inboxmanager.VerdictApprove,
		[]string{"A"}, "", supervisor)
	require.ErrorIs(t, err, inboxmanager.ErrApproveBlocked)
}

// Seluruh pelanggaran isi dikumpulkan sekaligus.
func TestNewDecisionCollectsAllViolations(t *testing.T) {
	_, err := inboxmanager.NewDecision(inboxmanager.TabMasterBengkel, inboxmanager.Verdict("hapus"),
		[]string{" ", ""}, "", supervisor)
	require.Equal(t, []inboxmanager.Violation{
		{Field: inboxmanager.FieldVerdict, Message: "Keputusan hanya boleh \"setujui\" atau \"tolak\"."},
		{Field: inboxmanager.FieldKeys, Message: "Pilih dulu baris yang hendak diputuskan."},
	}, violationsOf(t, err))

	many := make([]string, 0, inboxmanager.MaxDecisionKeys+1)
	for i := 0; i <= inboxmanager.MaxDecisionKeys; i++ {
		many = append(many, fmt.Sprintf("K%d", i))
	}
	_, err = inboxmanager.NewDecision(inboxmanager.TabMasterBengkel, inboxmanager.VerdictReject,
		many, "", supervisor)
	require.Equal(t, []inboxmanager.Violation{
		{Field: inboxmanager.FieldKeys, Message: "Terlalu banyak baris dalam satu permintaan. " +
			"Putuskan paling banyak 100 baris sekaligus."},
		{Field: inboxmanager.FieldReason, Message: "Alasan penolakan wajib diisi."},
	}, violationsOf(t, err))
}

func TestValidationErrorMessage(t *testing.T) {
	require.Equal(t, "inboxmanager: isian tidak sah", inboxmanager.NewValidationError(nil).Error())
	require.Equal(t, "inboxmanager: tab: salah; baris: kosong", inboxmanager.NewValidationError(
		[]inboxmanager.Violation{{Field: "tab", Message: "salah"}, {Field: "baris", Message: "kosong"}},
	).Error())
}

func TestCallerWithLineBusinessReturnsCopy(t *testing.T) {
	original := inboxmanager.Caller{Login: "A", OrgUnit: "X"}
	filled := original.WithLineBusiness("PA")
	require.Equal(t, inboxmanager.Caller{Login: "A", OrgUnit: "X", LineBusiness: "PA"}, filled)
	require.Empty(t, original.LineBusiness)
}

func TestQueuePagination(t *testing.T) {
	rows := make([]inboxmanager.QueueRow, 0, 5)
	for i := 0; i < 5; i++ {
		rows = append(rows, inboxmanager.QueueRow{Key: fmt.Sprintf("K%d", i)})
	}

	page := inboxmanager.SliceQueue(rows, inboxmanager.Pagination{Page: 2, Size: 2})
	require.Equal(t, []inboxmanager.QueueRow{{Key: "K2"}, {Key: "K3"}}, page.Rows)
	require.Equal(t, 5, page.Total)
	require.Equal(t, 3, page.TotalPages())

	last := inboxmanager.SliceQueue(rows, inboxmanager.Pagination{Page: 3, Size: 2})
	require.Equal(t, []inboxmanager.QueueRow{{Key: "K4"}}, last.Rows)

	exact := inboxmanager.SliceQueue(rows[:4], inboxmanager.Pagination{Page: 1, Size: 2})
	require.Equal(t, 2, exact.TotalPages())
}

func TestNewQueryRejectsUnknownAndForbiddenTab(t *testing.T) {
	_, err := inboxmanager.NewQuery(inboxmanager.QueryInput{Tab: "99"}, supervisor)
	require.Equal(t, inboxmanager.FieldTab, violationsOf(t, err)[0].Field)

	_, err = inboxmanager.NewQuery(inboxmanager.QueryInput{Tab: inboxmanager.TabNomorRangka},
		inboxmanager.Caller{Login: "MGR", LineBusiness: inboxmanager.LineTravel})
	require.ErrorIs(t, err, inboxmanager.ErrTabNotAllowed)

	_, err = inboxmanager.NewQuery(inboxmanager.QueryInput{}, inboxmanager.Caller{})
	require.ErrorIs(t, err, inboxmanager.ErrCallerUnknown)

	q, err := inboxmanager.NewQuery(inboxmanager.QueryInput{Page: inboxmanager.Pagination{Size: 999}},
		inboxmanager.Caller{Login: " MGR ", LineBusiness: " PA "})
	require.NoError(t, err)
	require.Equal(t, inboxmanager.DefaultTab, q.Tab.Code)
	require.Equal(t, inboxmanager.MaxPageSize, q.Page.Size)
	require.Equal(t, "PA", q.LineBusiness)
}

func periodQuery(t *testing.T, input inboxmanager.PeriodInput) (inboxmanager.Query, error) {
	t.Helper()
	return inboxmanager.NewQuery(inboxmanager.QueryInput{Tab: inboxmanager.TabKlaim, Period: input}, supervisor)
}

func TestPeriodParsingBranches(t *testing.T) {
	t.Run("kosong berarti seluruh periode", func(t *testing.T) {
		q, err := periodQuery(t, inboxmanager.PeriodInput{Mode: inboxmanager.PeriodMonth})
		require.NoError(t, err)
		require.True(t, q.Period.Empty())
	})

	t.Run("bentuk tanpa mode menebak rentang dari dari/sampai", func(t *testing.T) {
		q, err := periodQuery(t, inboxmanager.PeriodInput{From: "2026-01-10", Until: "2026-01-10"})
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), q.Period.From)
		require.Equal(t, time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC), q.Period.Until)
	})

	t.Run("bentuk tanpa mode menebak bulan", func(t *testing.T) {
		q, err := periodQuery(t, inboxmanager.PeriodInput{Month: " 2026-03 "})
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), q.Period.From)
	})

	t.Run("mode tidak dikenal", func(t *testing.T) {
		_, err := periodQuery(t, inboxmanager.PeriodInput{Mode: "tahun", Month: "2026"})
		require.Equal(t, []inboxmanager.Violation{{
			Field: inboxmanager.FieldPeriod, Message: "Bentuk periode hanya boleh \"bulan\" atau \"rentang\".",
		}}, violationsOf(t, err))
	})

	t.Run("mode bulan tanpa bulan", func(t *testing.T) {
		_, err := periodQuery(t, inboxmanager.PeriodInput{Mode: inboxmanager.PeriodMonth, From: "2026-01-01"})
		require.Equal(t, "Bulan & Tahun wajib diisi.", violationsOf(t, err)[0].Message)
	})

	t.Run("rentang dengan kedua tanggal salah", func(t *testing.T) {
		_, err := periodQuery(t, inboxmanager.PeriodInput{
			Mode: inboxmanager.PeriodRange, From: "kemarin", Until: "  ",
		})
		messages := []string{}
		for _, v := range violationsOf(t, err) {
			messages = append(messages, v.Message)
		}
		require.Equal(t, []string{
			"Tanggal \"Dari\" tidak dikenali. Contoh yang benar: 2026-09-01.",
			"Tanggal \"Sampai\" tidak dikenali. Contoh yang benar: 2026-09-30.",
		}, messages)
	})

	t.Run("sampai mendahului dari", func(t *testing.T) {
		_, err := periodQuery(t, inboxmanager.PeriodInput{
			Mode: inboxmanager.PeriodRange, From: "2026-09-30", Until: "2026-09-01",
		})
		require.Equal(t, "Tanggal \"Sampai\" tidak boleh mendahului tanggal \"Dari\".",
			violationsOf(t, err)[0].Message)
	})
}

func TestQueueTabsAndDefaultTab(t *testing.T) {
	queues := inboxmanager.QueueTabs()
	require.Len(t, queues, 9)
	for _, tab := range queues {
		require.Equal(t, inboxmanager.KindQueue, tab.Kind)
	}
	require.Equal(t, inboxmanager.TabMasterBengkel, queues[0].Code)

	require.Equal(t, inboxmanager.TabOutstanding, inboxmanager.DefaultTabFor(supervisor))

	_, known := inboxmanager.FindTab("99")
	require.False(t, known)

	// Kode tak dikenal diurutkan paling akhir.
	sorted := inboxmanager.SortCounters([]inboxmanager.Counter{
		{TabCode: "99"}, {TabCode: inboxmanager.TabKlaim},
	})
	require.Equal(t, inboxmanager.TabKlaim, sorted[0].TabCode)
	require.Equal(t, "99", sorted[1].TabCode)
}
