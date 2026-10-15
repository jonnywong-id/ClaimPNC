package inboxlaporanklaim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxlaporanklaim"
)

// Kategori yang tidak dikenal tidak mengarang judul maupun kode lama.
func TestUnknownCategoryFallsBackToItsOwnCode(t *testing.T) {
	unknown := inboxlaporanklaim.Category("entah")
	require.Equal(t, "entah", unknown.Title())
	require.Empty(t, unknown.LegacyCode())
	require.Empty(t, unknown.LegacyQuery())
}

func TestLegacyCodeMatchesTheOldScreenTabNumbers(t *testing.T) {
	require.Equal(t, "1", inboxlaporanklaim.CategoryOutstanding.LegacyCode())
	require.Equal(t, "9", inboxlaporanklaim.CategoryUnregistered.LegacyCode())
	require.Equal(t, "0", inboxlaporanklaim.CategoryAll.LegacyCode())
}

// Tanggal acuan yang jatuh setelah "sekarang" tidak menghasilkan umur negatif.
func TestAgingInTheFutureIsZero(t *testing.T) {
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	report := inboxlaporanklaim.ClaimReport{AgingAt: now.AddDate(0, 0, 3)}
	require.Zero(t, report.AgingDays(now))
}

func TestValidationErrorListsEveryViolation(t *testing.T) {
	err := &inboxlaporanklaim.ValidationError{Violation: []inboxlaporanklaim.Violation{
		{Field: "nama_pelapor", Message: "terlalu panjang"},
		{Field: "jumlah_dokumen", Message: "negatif"},
	}}
	require.Equal(t,
		"inboxlaporanklaim: isian tidak sah (nama_pelapor: terlalu panjang; jumlah_dokumen: negatif)",
		err.Error())
}

func TestBusinessLinesListedInScreenOrder(t *testing.T) {
	list := inboxlaporanklaim.ListBusinessLines()
	require.Equal(t, []inboxlaporanklaim.BusinessLineInfo{
		{Line: inboxlaporanklaim.BusinessLineAll, Name: "Semua bisnis"},
		{Line: inboxlaporanklaim.BusinessLineNonMBU, Name: "Non-MBU (Aneka, Marine, Fire)"},
		{Line: inboxlaporanklaim.BusinessLineSpecialGroup, Name: "Kelompok bisnis khusus"},
		{Line: inboxlaporanklaim.BusinessLinePA, Name: "Personal Accident"},
		{Line: inboxlaporanklaim.BusinessLineTravel, Name: "Travel"},
	}, list)
}

func TestBusinessLineNameAndCriteriaForUnknownLine(t *testing.T) {
	require.Equal(t, "Travel", inboxlaporanklaim.BusinessLineTravel.Name())

	unknown := inboxlaporanklaim.BusinessLine("entah")
	require.Equal(t, "entah", unknown.Name())
	panel, in, notIn := unknown.Criteria()
	require.Nil(t, panel)
	require.Nil(t, in)
	require.Nil(t, notIn)
}

func TestFilterAndCallerCleanTrimSpaces(t *testing.T) {
	filter := inboxlaporanklaim.Filter{
		Category:     inboxlaporanklaim.CategoryAll,
		RegionCode:   " 01 ",
		BranchCode:   " 1001 ",
		BusinessLine: inboxlaporanklaim.BusinessLinePA,
		Keyword:      " RCV-1 ",
		Operator:     " op ",
	}
	require.Equal(t, inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryAll, RegionCode: "01", BranchCode: "1001",
		BusinessLine: inboxlaporanklaim.BusinessLinePA, Keyword: "RCV-1", Operator: "op",
	}, filter.Clean())

	caller := inboxlaporanklaim.Caller{Login: " adminpnc ", Name: " Admin "}
	require.Equal(t, inboxlaporanklaim.Caller{Login: "adminpnc", Name: "Admin"}, caller.Clean())
}

func TestSummaryCountOfEveryCountedTab(t *testing.T) {
	summary := inboxlaporanklaim.Summary{
		Total: 1, NotTransferred: 2, Unregistered: 3, Outstanding: 4, Accepted: 5,
		MessageUnanswered: 6, MessageWaiting: 7, MessageReplied: 8,
	}
	want := map[inboxlaporanklaim.Category]int{
		inboxlaporanklaim.CategoryAll:               1,
		inboxlaporanklaim.CategoryNotTransferred:    2,
		inboxlaporanklaim.CategoryUnregistered:      3,
		inboxlaporanklaim.CategoryOutstanding:       4,
		inboxlaporanklaim.CategoryAccepted:          5,
		inboxlaporanklaim.CategoryMessageUnanswered: 6,
		inboxlaporanklaim.CategoryMessageWaiting:    7,
		inboxlaporanklaim.CategoryMessageReplied:    8,
	}
	for category, count := range want {
		got, counted := summary.CountOf(category)
		require.Truef(t, counted, "tab %q", category)
		require.Equalf(t, count, got, "tab %q", category)
	}
}

func TestNormalizePolicyNumberDropsDotsAndUppercases(t *testing.T) {
	require.Equal(t, "126ABC01", inboxlaporanklaim.NormalizePolicyNumber(" 126.abc.01 "))
}

func TestPolicyNoticesAndBlocking(t *testing.T) {
	notFound := inboxlaporanklaim.Notices(inboxlaporanklaim.Policy{}, false)
	require.Equal(t, []inboxlaporanklaim.PolicyNotice{
		{Code: inboxlaporanklaim.PolicyNotFound, Message: "Nomor Polis tidak tersedia"},
	}, notFound)
	require.False(t, inboxlaporanklaim.Blocked(notFound), "polis tidak ditemukan hanya memberi tahu")

	// Polis Syariah sekaligus bukan PNC menghasilkan dua pesan yang sama-sama memblokir.
	both := inboxlaporanklaim.Notices(inboxlaporanklaim.Policy{Syariah: true, GroupPanel: " 007 "}, true)
	require.Len(t, both, 2)
	require.Equal(t, inboxlaporanklaim.PolicySyariah, both[0].Code)
	require.Equal(t, inboxlaporanklaim.PolicyNotPNC, both[1].Code)
	require.True(t, inboxlaporanklaim.Blocked(both))

	require.Empty(t, inboxlaporanklaim.Notices(inboxlaporanklaim.Policy{GroupPanel: "003"}, true))
	require.False(t, inboxlaporanklaim.Blocked(nil))
}
