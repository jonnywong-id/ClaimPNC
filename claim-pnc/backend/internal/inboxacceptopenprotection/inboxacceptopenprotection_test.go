package inboxacceptopenprotection_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxacceptopenprotection"
)

var wib = time.FixedZone("WIB", 7*60*60)

func TestQueueValidAndNormalize(t *testing.T) {
	require.True(t, inboxacceptopenprotection.QueuePremium.Valid())
	require.True(t, inboxacceptopenprotection.QueueNonPremium.Valid())
	require.False(t, inboxacceptopenprotection.Queue("lain").Valid())

	// Antrean yang dikenali dipertahankan; yang asing jatuh ke NON PREMI.
	require.Equal(t, inboxacceptopenprotection.QueuePremium, inboxacceptopenprotection.QueuePremium.Normalize())
	require.Equal(t, inboxacceptopenprotection.QueueNonPremium, inboxacceptopenprotection.Queue("").Normalize())
}

func TestQueueTypeFilter(t *testing.T) {
	v, eq := inboxacceptopenprotection.QueuePremium.TypeFilter()
	require.Equal(t, inboxacceptopenprotection.TypePremium, v)
	require.True(t, eq)

	v, eq = inboxacceptopenprotection.QueueNonPremium.TypeFilter()
	require.Equal(t, inboxacceptopenprotection.TypePremium, v)
	require.False(t, eq)
}

func TestChangeDetailEmpty(t *testing.T) {
	require.True(t, inboxacceptopenprotection.ChangeDetail{ObjectName: "  "}.Empty())

	d := time.Date(2026, 8, 1, 0, 0, 0, 0, wib)
	require.False(t, inboxacceptopenprotection.ChangeDetail{LossDateBefore: &d}.Empty())
	require.False(t, inboxacceptopenprotection.ChangeDetail{LossDateAfter: &d}.Empty())
	require.False(t, inboxacceptopenprotection.ChangeDetail{CauseOfLossBefore: "1"}.Empty())
	require.False(t, inboxacceptopenprotection.ChangeDetail{CauseOfLossAfter: "1"}.Empty())
	require.False(t, inboxacceptopenprotection.ChangeDetail{ObjectName: "x"}.Empty())
	require.False(t, inboxacceptopenprotection.ChangeDetail{BranchName: "x"}.Empty())
}

func TestShowsChangeDetailOnlyForTypesSevenAndEight(t *testing.T) {
	require.True(t, inboxacceptopenprotection.ShowsChangeDetail(" 7 "))
	require.True(t, inboxacceptopenprotection.ShowsChangeDetail("8"))
	require.False(t, inboxacceptopenprotection.ShowsChangeDetail("1"))
	require.False(t, inboxacceptopenprotection.ShowsChangeDetail(""))
}

func TestDecisionValidAndStatus(t *testing.T) {
	require.True(t, inboxacceptopenprotection.DecisionApprove.Valid())
	require.True(t, inboxacceptopenprotection.DecisionReject.Valid())
	require.False(t, inboxacceptopenprotection.Decision("mungkin").Valid())

	require.Equal(t, inboxacceptopenprotection.AcceptApproved, inboxacceptopenprotection.DecisionApprove.Status())
	require.Equal(t, inboxacceptopenprotection.AcceptRejected, inboxacceptopenprotection.DecisionReject.Status())
}

func TestProtectionPendingAndApproved(t *testing.T) {
	p := inboxacceptopenprotection.Protection{AcceptStatus: " "}
	require.True(t, p.Pending())
	require.False(t, p.Approved())

	p.AcceptStatus = " 1 "
	require.False(t, p.Pending())
	require.True(t, p.Approved())

	p.AcceptStatus = "2"
	require.False(t, p.Pending())
	require.False(t, p.Approved())
}

func TestQueueOf(t *testing.T) {
	require.Equal(t, inboxacceptopenprotection.QueuePremium, inboxacceptopenprotection.QueueOf(" 2 "))
	require.Equal(t, inboxacceptopenprotection.QueueNonPremium, inboxacceptopenprotection.QueueOf("7"))
}

func TestFilterNormalize(t *testing.T) {
	f := inboxacceptopenprotection.Filter{Search: "  abc ", Limit: 0, Offset: -3}.Normalize()
	require.Equal(t, "abc", f.Search)
	require.Equal(t, inboxacceptopenprotection.DefaultLimit, f.Limit)
	require.Equal(t, 0, f.Offset)
	require.Equal(t, inboxacceptopenprotection.QueueNonPremium, f.Queue)

	f = inboxacceptopenprotection.Filter{Queue: inboxacceptopenprotection.QueuePremium, Limit: 500, Offset: 4}.Normalize()
	require.Equal(t, inboxacceptopenprotection.MaxLimit, f.Limit)
	require.Equal(t, 4, f.Offset)
	require.Equal(t, inboxacceptopenprotection.QueuePremium, f.Queue)

	f = inboxacceptopenprotection.Filter{Limit: 30}.Normalize()
	require.Equal(t, 30, f.Limit)
}

func TestCanOpenScreen(t *testing.T) {
	require.False(t, inboxacceptopenprotection.CanOpenScreen(nil))
	require.False(t, inboxacceptopenprotection.CanOpenScreen([]string{"PncAdmin"}))
	// Awalan GCNMFW: dan kapitalisasi berbeda tetap dikenali.
	require.True(t, inboxacceptopenprotection.CanOpenScreen([]string{"PncAdmin", " GCNMFW:ADMINISTRATORS "}))
	require.True(t, inboxacceptopenprotection.CanOpenScreen([]string{"pnckomite"}))
}

func TestQueuesForUnionOfGroups(t *testing.T) {
	require.Nil(t, inboxacceptopenprotection.QueuesFor([]string{"PncAdmin"}))
	require.Equal(t,
		[]inboxacceptopenprotection.Queue{inboxacceptopenprotection.QueuePremium},
		inboxacceptopenprotection.QueuesFor([]string{"GCNMFW:PncCollection"}))
	require.Equal(t,
		[]inboxacceptopenprotection.Queue{inboxacceptopenprotection.QueueNonPremium},
		inboxacceptopenprotection.QueuesFor([]string{"CaseManager"}))
	// Urutan mengikuti layar lama: NON PREMI lebih dulu.
	require.Equal(t,
		[]inboxacceptopenprotection.Queue{inboxacceptopenprotection.QueueNonPremium, inboxacceptopenprotection.QueuePremium},
		inboxacceptopenprotection.QueuesFor([]string{"pnccollection", "PncOPCGeneral", "lain"}))
}

func TestCanOpenQueue(t *testing.T) {
	groups := []string{"PncCollection"}
	require.True(t, inboxacceptopenprotection.CanOpenQueue(groups, inboxacceptopenprotection.QueuePremium))
	require.False(t, inboxacceptopenprotection.CanOpenQueue(groups, inboxacceptopenprotection.QueueNonPremium))
	require.False(t, inboxacceptopenprotection.CanOpenQueue(nil, inboxacceptopenprotection.QueuePremium))
}

func TestLossDateToApply(t *testing.T) {
	after := time.Date(2026, 8, 5, 13, 45, 10, 99, wib)
	p := inboxacceptopenprotection.Protection{
		Type:   inboxacceptopenprotection.TypeChangeLossDate,
		Change: inboxacceptopenprotection.ChangeDetail{LossDateAfter: &after},
	}

	// Disetujui: tanggal dipotong ke tengah malam dengan zona dipertahankan.
	got, ok := inboxacceptopenprotection.LossDateToApply(p, inboxacceptopenprotection.DecisionApprove)
	require.True(t, ok)
	require.Equal(t, time.Date(2026, 8, 5, 0, 0, 0, 0, wib), got)
	require.Equal(t, wib, got.Location())

	// Ditolak tidak mengubah klaim.
	_, ok = inboxacceptopenprotection.LossDateToApply(p, inboxacceptopenprotection.DecisionReject)
	require.False(t, ok)

	// Tipe '8' juga memunculkan panel, tetapi tidak mengubah DOL.
	p8 := p
	p8.Type = inboxacceptopenprotection.TypeChangeCauseOfLoss
	_, ok = inboxacceptopenprotection.LossDateToApply(p8, inboxacceptopenprotection.DecisionApprove)
	require.False(t, ok)

	// Tipe lain.
	p1 := p
	p1.Type = "1"
	_, ok = inboxacceptopenprotection.LossDateToApply(p1, inboxacceptopenprotection.DecisionApprove)
	require.False(t, ok)

	// Baris warisan tanpa tanggal baru.
	kosong := inboxacceptopenprotection.Protection{Type: inboxacceptopenprotection.TypeChangeLossDate}
	got, ok = inboxacceptopenprotection.LossDateToApply(kosong, inboxacceptopenprotection.DecisionApprove)
	require.False(t, ok)
	require.True(t, got.IsZero())
}
