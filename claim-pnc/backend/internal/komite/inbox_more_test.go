package komite_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
)

func TestCaseNormalizedTrimsEveryText(t *testing.T) {
	c := komite.CommitteeCase{
		CaseID: " K ", ClaimNumber: " C ", PolicyNumber: " P ", InsuredName: " I ",
		BusinessName: " B ", SourceOfBusiness: " S ", BranchName: " R ",
		AssignedOperator: " O ", WorkStatus: " W ",
	}.Normalized()
	require.Equal(t, komite.CommitteeCase{
		CaseID: "K", ClaimNumber: "C", PolicyNumber: "P", InsuredName: "I",
		BusinessName: "B", SourceOfBusiness: "S", BranchName: "R",
		AssignedOperator: "O", WorkStatus: "W",
	}, c)
}

func caseFor(operator string, outcome komite.Outcome, status string, decisions ...komite.Decision) komite.CommitteeCase {
	return komite.CommitteeCase{
		CaseID: "K-1", AssignedOperator: operator, LegacyOutcome: outcome, WorkStatus: status,
		Progress: komite.Progress{Decisions: decisions},
	}
}

func TestInBoxSortsCasesIntoTheThreeBoxes(t *testing.T) {
	const me = "budi"
	approve := komite.Decision{ActorLogin: "BUDI", Kind: komite.DecisionApprove}
	reject := komite.Decision{ActorLogin: "budi", Kind: komite.DecisionReturn}

	pending := caseFor(me, komite.OutcomePending, "Open")
	require.True(t, pending.InBox(komite.InboxOutstanding, me))
	require.False(t, pending.InBox(komite.InboxAccepted, me))
	require.False(t, pending.InBox(komite.InboxRejected, me))
	require.False(t, pending.InBox(komite.InboxOutstanding, "lain"), "milik orang lain")
	require.False(t, pending.InBox(komite.InboxOutstanding, " "))
	require.False(t, pending.InBox(komite.InboxKind("entah"), me))

	resolved := caseFor(me, komite.OutcomePending, komite.StatusResolved)
	require.False(t, resolved.InBox(komite.InboxOutstanding, me))

	approved := caseFor(me, komite.OutcomePending, "Open", approve)
	require.False(t, approved.InBox(komite.InboxOutstanding, me))
	require.True(t, approved.InBox(komite.InboxAccepted, me))
	require.False(t, approved.InBox(komite.InboxRejected, me))

	returned := caseFor(me, komite.OutcomePending, "Open", reject)
	require.True(t, returned.InBox(komite.InboxRejected, me))
	require.False(t, returned.InBox(komite.InboxAccepted, me))

	require.True(t, caseFor(me, komite.OutcomeApproved, "").InBox(komite.InboxAccepted, me))
	require.True(t, caseFor(me, komite.OutcomeRejected, "").InBox(komite.InboxRejected, me))
	require.True(t, caseFor(me, komite.OutcomeReturned, "").InBox(komite.InboxRejected, me))
}

func TestProgressDecidedByAndLastDecision(t *testing.T) {
	progress := komite.Progress{Decisions: []komite.Decision{
		{ID: "1", ActorLogin: "budi"}, {ID: "2", ActorLogin: "siti"},
	}}
	require.True(t, progress.DecidedBy(" BUDI "))
	require.False(t, progress.DecidedBy(""))
	require.False(t, progress.DecidedBy("ani"))

	last, ok := progress.LastDecision()
	require.True(t, ok)
	require.Equal(t, "2", last.ID)

	_, ok = komite.Progress{}.LastDecision()
	require.False(t, ok)
}

func TestSearchAndDateRange(t *testing.T) {
	created := time.Date(2026, 9, 10, 2, 0, 0, 0, time.UTC)
	c := komite.CommitteeCase{CaseID: "K-2601", ClaimNumber: "PNCN.26.0101", CreatedAt: created}

	require.True(t, c.MatchesSearch(" "))
	require.True(t, c.MatchesSearch("k-26"))
	require.True(t, c.MatchesSearch("0101"))
	require.False(t, c.MatchesSearch("9999"))

	require.True(t, c.WithinDateRange(time.Time{}, time.Time{}))
	require.True(t, c.WithinDateRange(created, created))
	require.False(t, c.WithinDateRange(created.AddDate(0, 0, 1), time.Time{}))
	require.False(t, c.WithinDateRange(time.Time{}, created.AddDate(0, 0, -1)))

	undated := komite.CommitteeCase{}
	require.True(t, undated.WithinDateRange(time.Time{}, time.Time{}))
	require.False(t, undated.WithinDateRange(created, time.Time{}))
}

func TestEarliestCreatedAtIsTheStartOfTheYear(t *testing.T) {
	require.Equal(t, time.Date(komite.InboxEarliestYear, 1, 1, 0, 0, 0, 0, time.UTC),
		komite.InboxEarliestCreatedAt())
}

func TestTransferDetailEmptinessAndCoverageFilled(t *testing.T) {
	require.True(t, komite.TransferDetail{}.Empty())
	require.True(t, komite.TransferDetail{}.MoneyEmpty())
	require.False(t, komite.TransferDetail{HasClaim: true}.Empty())
	require.True(t, komite.TransferDetail{HasClaim: true}.MoneyEmpty())
	require.False(t, komite.TransferDetail{HasCommitteeRecord: true}.MoneyEmpty())

	require.False(t, komite.CoverageAnalysis{CoverageName: "Kebakaran"}.Filled())
	require.True(t, komite.CoverageAnalysis{Diagnose: "patah"}.Filled())
}

func TestCommitteeKindCoversEveryCode(t *testing.T) {
	cases := map[[2]string]string{
		{"1", ""}:    "Survey Komite",
		{"3", ""}:    "Ex Gratia",
		{"4", ""}:    "Liable Klaim",
		{"5", ""}:    "Final",
		{"9", ""}:    "Survey Komite",
		{"2", "01"}:  "Final",
		{"2", "2"}:   "Interim",
		{"2", "3"}:   "Salvage",
		{"2", "4"}:   "Adjuster Fee",
		{"2", "5"}:   "Adjustment",
		{"2", "6"}:   "Tolak Klaim",
		{"2", "7"}:   "Collection Fee",
		{"2", "000"}: "Collection Fee",
	}
	for input, want := range cases {
		require.Equal(t, want, komite.CommitteeKindOf(input[0], input[1]), input)
	}
}

func TestFixedRandomizerStaysInRange(t *testing.T) {
	require.Equal(t, 0, komite.FixedRandomizer(5).Pick(0))
	require.Equal(t, 2, komite.FixedRandomizer(5).Pick(3))
	require.Equal(t, 2, komite.FixedRandomizer(-1).Pick(3))
}

func TestValidationErrorMessage(t *testing.T) {
	err := &komite.ValidationError{Violations: []komite.Violation{
		{Field: "a", Message: "x"}, {Field: "b", Message: "y"},
	}}
	require.Equal(t, "komite: validasi gagal — a: x; b: y", err.Error())
}
