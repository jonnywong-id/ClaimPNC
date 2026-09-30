package usecase_test

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/repo/memory"
	"claim-pnc/internal/registrasi/usecase"
)

// approvedClaim membawa adjustment 1 sampai disetujui kedua jenjang komite, dengan penerima
// yang rekeningnya lengkap.
func (l environment) approvedClaim(t *testing.T) (registrasi.Task, registrasi.Claim) {
	t.Helper()
	ctx := context.Background()
	task, result := l.transferredClaim(t)
	_, err := l.service.DecideCommittee(ctx, decide(result.Committee.ID, registrasi.DecisionApprove, ""), committee1)
	require.NoError(t, err)
	_, err = l.service.DecideCommittee(ctx, decide(result.Committee.ID, registrasi.DecisionApprove, ""), committee2)
	require.NoError(t, err)
	_, err = l.service.SaveReceiver(ctx, usecase.ReceiverCommand{
		ClaimID: task.ClaimID, TaskID: task.ID, ReceiverID: "1", AccountNo: "9876543210", Email: "a@contoh.internal",
	}, l.caller)
	require.NoError(t, err)
	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	// Jenis dokumen unggahan diperiksa terhadap checklist kode bisnis polis.
	claim.Policy.BusinessCode = "10140"
	require.NoError(t, l.store.Save(ctx, claim))
	return task, claim
}

func acceptance(task registrasi.Task, lod string) usecase.AcceptanceCommand {
	day := time.Date(2026, time.June, 9, 0, 0, 0, 0, clock.ZoneWIB)
	return usecase.AcceptanceCommand{
		ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1,
		Form: registrasi.AcceptanceForm{
			LODStatus: lod, ReceiveDate: day, PayableDate: day, PrintDate: day,
			LODValue: registrasi.Rupiah(5_000_000), HasLODValue: true, ReceiverID: "1",
			CommitteeName: "KOMITE02", Remark: "OK", MinutesNote: "Berita acara",
		},
		Files: []usecase.AcceptanceFile{{DocumentTypeID: "14904", FileName: "lod.pdf", Content: []byte("%PDF"), Note: "LOD"}},
	}
}

// Setuju: nomor A<yy><site><15 digit>, isian tersimpan, Status Klaim 1161, riwayat, dan dua
// progres 014/60 Done (`SetAdjustmentAcceptation` langkah 62–96).
func TestAcceptSettlementIssuesNumberAndMarksClaim(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, claim := l.approvedClaim(t)
	// Persetujuan komite terakhir membuka posisi AKSEPTASI (KomitePost_Adjustment 19–21).
	require.Len(t, l.acceptance.Started, 1)
	require.Equal(t, registrasi.ProgressStart{
		ClaimNumber: claim.Number, CaseID: claim.Number, Position: registrasi.PositionAcceptance,
		Note: "Auto Create AKSEPTASI", Progress1: "014", Progress2: "60",
		User: l.acceptance.Started[0].User, At: l.acceptance.Started[0].At,
	}, l.acceptance.Started[0])
	opened := l.acceptance.Positions[memory.PositionKey(claim.Number, registrasi.PositionAcceptance)]
	require.NotEmpty(t, opened)

	saved, err := l.service.AcceptSettlement(ctx, acceptance(task, registrasi.LODAgreed), l.caller)
	require.NoError(t, err)
	line := saved.InsuredItem[0].Coverage[0].Settlement[0]
	require.True(t, strings.HasPrefix(line.AcceptedNo, "A2610"), line.AcceptedNo)
	require.Len(t, line.AcceptedNo, 1+2+2+15)
	require.Equal(t, registrasi.LODAgreed, line.AcceptanceLODStatus)
	require.Equal(t, "KOMITE02", line.Acceptance.Form.CommitteeName)
	require.NotEmpty(t, line.Acceptance.ReceiverName)
	require.False(t, line.Acceptance.AcceptedAt.IsZero())
	require.Equal(t, registrasi.StatusClaimAccepted, saved.ClaimStatus)

	require.Len(t, l.acceptance.History, 1)
	require.Equal(t, "Claim Accepted with No "+line.AcceptedNo, l.acceptance.History[0].Note)
	require.Equal(t, claim.Keys().Prefixed, l.acceptance.History[0].CaseID)
	require.Len(t, l.acceptance.Progress, 2)
	require.Equal(t, "Auto FInish Akseptasi", l.acceptance.Progress[0].Note)
	require.Equal(t, "Auto Akseptasi By LOD", l.acceptance.Progress[1].Note)
	require.Equal(t, "014", l.acceptance.Progress[0].Progress1)
	require.Equal(t, "60", l.acceptance.Progress[0].Progress2)
	require.Equal(t, opened, l.acceptance.Progress[0].PositionID, "POSISIID = posisi AKSEPTASI terbuka")
	require.Equal(t, opened, l.acceptance.Progress[1].PositionID)
	require.Empty(t, l.acceptance.Positions, "posisi AKSEPTASI tertutup Done")
	require.Len(t, l.uploader.Files, 1, "berkas Unggah Dokumen Persetujuan LOD terkirim")

	_, err = l.service.AcceptSettlement(ctx, acceptance(task, registrasi.LODAgreed), l.caller)
	violation(t, err, registrasi.ViolationAcceptanceNumbered)
}

// Tidak setuju: nomor tetap terbit (sama seperti Pega), tetapi Status Klaim tidak menjadi 1161
// dan tanggal akseptasi kosong.
func TestAcceptSettlementDisagreedStillNumbers(t *testing.T) {
	l := setup(t)
	task, _ := l.approvedClaim(t)
	saved, err := l.service.AcceptSettlement(context.Background(), acceptance(task, registrasi.LODDisagreed), l.caller)
	require.NoError(t, err)
	line := saved.InsuredItem[0].Coverage[0].Settlement[0]
	require.NotEmpty(t, line.AcceptedNo)
	require.Equal(t, registrasi.LODDisagreed, line.AcceptanceLODStatus)
	require.True(t, line.Acceptance.AcceptedAt.IsZero())
	require.NotEqual(t, registrasi.StatusClaimAccepted, saved.ClaimStatus)
}

// CheckAttachmentLOD, nilai LOD, dan isian wajib.
func TestAcceptSettlementValidatesForm(t *testing.T) {
	l := setup(t)
	task, _ := l.approvedClaim(t)
	cmd := acceptance(task, "")
	cmd.Files = nil
	cmd.Form.LODValue = registrasi.Rupiah(900_000_000)
	_, err := l.service.AcceptSettlement(context.Background(), cmd, l.caller)
	violation(t, err, registrasi.ViolationAcceptanceRequired)
	violation(t, err, registrasi.ViolationAcceptanceAttachment)
	violation(t, err, registrasi.ViolationAcceptanceLODValue)
	require.Empty(t, l.acceptance.History, "tidak ada yang ditulis")
}

// Premi belum lunas diblokir kecuali ada Open Protection premi yang disetujui.
func TestAcceptSettlementBlocksUnpaidPremium(t *testing.T) {
	l := setup(t)
	task, claim := l.approvedClaim(t)
	l.premium.Statements[claim.Policy.Number] = registrasi.PremiumStatement{AgingAmount: big.NewRat(5, 1)}

	_, err := l.service.AcceptSettlement(context.Background(), acceptance(task, registrasi.LODAgreed), l.caller)
	violation(t, err, registrasi.ViolationAcceptancePremium)

	l.acceptance.OpenProtection = map[string]bool{claim.Policy.Number: true}
	_, err = l.service.AcceptSettlement(context.Background(), acceptance(task, registrasi.LODAgreed), l.caller)
	require.NoError(t, err, "Open Protection premi yang disetujui meloloskan akseptasi")
}

// Layanan premi yang gagal MENAHAN akseptasi, bukan meloloskannya.
func TestAcceptSettlementHoldsWhenPremiumUnavailable(t *testing.T) {
	l := setup(t)
	task, _ := l.approvedClaim(t)
	l.premium.Err = context.DeadlineExceeded
	_, err := l.service.AcceptSettlement(context.Background(), acceptance(task, registrasi.LODAgreed), l.caller)
	require.ErrorIs(t, err, usecase.ErrPremiumUnavailable)
}

// Print LOD mencatat tanggal cetak dan jenis LOD pada baris; keduanya tampil baca saja di form
// Persetujuan / Akseptasi (`.PrintDateLOD`, `.PDFType`).
func TestPrintLODRecordsPrintDateAndType(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, _ := l.approvedClaim(t)

	_, err := l.service.PrintLOD(ctx, usecase.LODCommand{
		ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1, Type: "4",
	}, l.caller)
	require.NoError(t, err)
	require.Len(t, l.acceptance.LODPrint, 1)
	for _, p := range l.acceptance.LODPrint {
		require.Equal(t, "4", p.Type)
		require.False(t, p.PrintedAt.IsZero())
	}
}

// Klaim yang disetujui komite sebelum posisi AKSEPTASI dibuka aplikasi ini tidak punya
// posisinya; POSISIID tidak dapat diisi (NOT NULL): progres akseptasi dilewati, Submit berhasil.
func TestAcceptSettlementSkipsProgressWithoutAcceptancePosition(t *testing.T) {
	l := setup(t)
	task, _ := l.approvedClaim(t)
	l.acceptance.Positions = nil

	saved, err := l.service.AcceptSettlement(context.Background(), acceptance(task, registrasi.LODAgreed), l.caller)
	require.NoError(t, err)
	require.NotEmpty(t, saved.InsuredItem[0].Coverage[0].Settlement[0].AcceptedNo)
	require.Empty(t, l.acceptance.Progress)
	require.Len(t, l.acceptance.History, 1)
}

// Tolak (KomitePost_Adjustment 61–62): posisi KOMITE terbuka ditutup "Auto Reject KOMITE"
// 006/24 Done, dan posisi AKSEPTASI tidak dibuka.
func TestCommitteeRejectClosesCommitteePosition(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, result := l.transferredClaim(t)
	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	l.acceptance.Positions = map[string]string{memory.PositionKey(claim.Number, registrasi.PositionCommittee): "7"}

	_, err = l.service.DecideCommittee(ctx, decide(result.Committee.ID, registrasi.DecisionReject, "tidak layak"), committee1)
	require.NoError(t, err)
	require.Empty(t, l.acceptance.Started)
	require.Len(t, l.acceptance.Progress, 1)
	require.Equal(t, "Auto Reject KOMITE", l.acceptance.Progress[0].Note)
	require.Equal(t, "7", l.acceptance.Progress[0].PositionID)
	require.Equal(t, "006", l.acceptance.Progress[0].Progress1)
	require.Equal(t, "24", l.acceptance.Progress[0].Progress2)
	require.Equal(t, registrasi.AcceptanceProgressDone, l.acceptance.Progress[0].Position)
}
