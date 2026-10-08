package usecase_test

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

func printPLA(task registrasi.Task) usecase.PLACommand {
	return usecase.PLACommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1}
}

func coinsLeader(l environment) {
	l.pla.Coins[firePolicy] = []registrasi.PLACoinsMember{
		{ID: "C0", Name: "ASURANSI SINAR MAS - KANTOR PUSAT", Leader: true, Share: 700_000, HasShare: true},
		{ID: "C1", Name: "ANGGOTA SATU", Share: 200_000, HasShare: true},
		{ID: "C2", Name: "ANGGOTA DUA", Share: 100_000, HasShare: true},
		{ID: "C3", Name: "ANGGOTA TERHAPUS", Share: 50_000, HasShare: true, Deleted: true},
		{ID: "C4", Name: "ANGGOTA NOL", Share: 0, HasShare: true},
	}
}

// Print PLA menunggu Claim Face Sheet (`!isCFS`).
func TestPLANeedsFaceSheet(t *testing.T) {
	l := setup(t)
	coinsLeader(l)
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(context.Background(), oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)

	_, err = l.service.PrintPLA(context.Background(), printPLA(task), l.caller)
	violation(t, err, registrasi.ViolationPLANeedsFaceSheet)
}

// Tanpa CoinsList (`IsNoCoins`) dan bila Sinar Mas bukan leader, tidak ada PLA.
func TestPLARequiresCoinsuranceLedByUs(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	_, err = l.service.PrintPLA(ctx, printPLA(task), l.caller)
	violation(t, err, registrasi.ViolationPLANoCoins)

	l.pla.Coins[firePolicy] = []registrasi.PLACoinsMember{
		{ID: "C0", Name: "ASURANSI SINAR MAS", Share: 300_000, HasShare: true},
		{ID: "C1", Name: "LEADER LAIN", Leader: true, Share: 700_000, HasShare: true},
	}
	_, err = l.service.PrintPLA(ctx, printPLA(task), l.caller)
	violation(t, err, registrasi.ViolationPLANotLeader)
}

// Satu PLA per anggota (bukan Sinar Mas, bukan terhapus, share > 0); dicetak ulang tanpa
// nomor baru; revisi CFS berikutnya menerbitkan PLA baru dengan catatan rujukan.
func TestPLAIssuesOnePerMemberAndReprints(t *testing.T) {
	l := setup(t)
	coinsLeader(l)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	first, err := l.service.PrintPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 2, first.Issued)
	require.Equal(t, "application/zip", first.ContentType)
	zr, err := zip.NewReader(bytes.NewReader(first.Content), int64(len(first.Content)))
	require.NoError(t, err)
	require.Len(t, zr.File, 2)
	require.True(t, strings.HasPrefix(zr.File[0].Name, "PLACOINSJ261"))

	require.Len(t, l.pla.Saved, 2)
	one := l.pla.Saved[0]
	require.Equal(t, "ANGGOTA SATU", one.Recipient)
	require.Equal(t, 0, one.Revision)
	require.Equal(t, registrasi.Rupiah(50_000_000), one.Amount[0].Base)
	require.Equal(t, registrasi.Rupiah(10_000_000), one.Amount[0].Result)
	require.Equal(t, registrasi.Rupiah(5_000_000), l.pla.Saved[1].Amount[0].Result)
	require.Contains(t, one.Note, "Estimation only")

	again, err := l.service.PrintPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 0, again.Issued)
	require.Len(t, l.pla.Saved, 2)

	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 2), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	next, err := l.service.PrintPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 2, next.Issued)
	revised := l.pla.Saved[2]
	require.Equal(t, 1, revised.Revision)
	require.Equal(t, registrasi.Rupiah(100_000_000), revised.Amount[0].Reserve)
	require.Contains(t, revised.Note, "Please see our PLA No.:"+one.Number)
}

// Layar PrintPLA_dtl: daftar menerbitkan sekali, REMARKS tersimpan, Print PLA per nomor
// mengembalikan satu PDF.
func TestPLAListNotesAndSinglePrint(t *testing.T) {
	l := setup(t)
	coinsLeader(l)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	list, err := l.service.ListPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 2, list.Issued)
	again, err := l.service.ListPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 0, again.Issued)
	require.Len(t, again.PLA, 2)

	number := list.PLA[0].Number
	saved, err := l.service.SavePLANotes(ctx, printPLA(task), map[string]string{number: "Catatan baru"}, l.caller)
	require.NoError(t, err)
	require.Equal(t, "Catatan baru", saved.PLA[0].Note)
	_, err = l.service.SavePLANotes(ctx, printPLA(task), map[string]string{"BUKAN-MILIK": "x"}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)

	// Isian Email (`.pyEmailAddress`) dapat diubah bersama Remarks; dipangkas, dan dibatasi
	// panjang kolom EMAILPLA.
	withEmail, err := l.service.SavePLADetails(ctx, printPLA(task), nil,
		map[string]string{number: "  klaim@contoh.co.id; re@contoh.co.id  "}, l.caller)
	require.NoError(t, err)
	require.Equal(t, "klaim@contoh.co.id; re@contoh.co.id", withEmail.PLA[0].Info.Email)
	reread, err := l.service.ListPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, "klaim@contoh.co.id; re@contoh.co.id", reread.PLA[0].Info.Email)
	_, err = l.service.SavePLADetails(ctx, printPLA(task), nil, map[string]string{number: strings.Repeat("a", 1001)}, l.caller)
	violation(t, err, registrasi.ViolationPLAEmailTooLong)
	_, err = l.service.SavePLADetails(ctx, printPLA(task), nil, map[string]string{"BUKAN-MILIK": "x@contoh.co.id"}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)

	command := printPLA(task)
	command.Number = number
	one, err := l.service.PrintPLA(ctx, command, l.caller)
	require.NoError(t, err)
	require.Equal(t, "application/pdf", one.ContentType)
	require.Equal(t, "PLACOINS"+number+".pdf", one.FileName)
}

// Jaminan dengan spreading FAC OUT mendapat PLA FACOUT per reasuradur FacOffer — juga bila
// polis tidak berkoasuransi (DownloadFireLossAdvice_act langkah 23 dan 25). Nilainya
// ShareOffered × reserve / (TSISublimit × percentASM).
func TestPLAFacOutWithoutCoinsurance(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)

	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.InsuredItem[0].Coverage[0].Spreading[0].TreatyKind = registrasi.TreatyFacOut
	require.NoError(t, l.store.Save(ctx, claim))
	objectID := claim.InsuredItem[0].ID
	l.dla.Cases[registrasi.TreatyFacOut] = "3" // REINSURANCETYPE.TYPE fakultatif
	l.pla.Offers[firePolicy] = []registrasi.FacOffer{{
		ReinsurerName: "REASURANSI CONTOH", ReinsurerID: "R9",
		Property: []registrasi.FacObject{{ObjectNo: objectID, Coverage: []registrasi.FacCoverage{
			{Code: claim.InsuredItem[0].Coverage[0].ID, TSISublimit: "1000000000", ShareOffered: "250000000"},
		}}},
	}}

	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(40_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	list, err := l.service.ListPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Len(t, list.PLA, 1)
	p := list.PLA[0]
	require.Equal(t, registrasi.PLATypeFacOut, p.Type)
	require.Equal(t, "REASURANSI CONTOH", p.Recipient)
	require.True(t, strings.HasPrefix(p.Number, registrasi.PLACodeFacOut))
	require.Equal(t, registrasi.Rupiah(10_000_000), p.Amount[0].Result, "250 jt / 1 M × 40 jt")

	printed, err := l.service.PrintPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, "PLAFACOFFER"+p.Number+".pdf", printed.FileName)
}

// GeneratePLAList langkah 33: PLA yang terbit mengubah Status Klaim menjadi 1138 (PLA Report).
func TestPLAIssueSetsStatusPLAReport(t *testing.T) {
	l := setup(t)
	coinsLeader(l)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	_, err = l.service.ListPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, registrasi.StatusClaimPLAReport, stored.ClaimStatus)
}

// GeneratePLAList langkah 6: klaim Ex Gratia tidak menerbitkan PLA.
func TestPLANotIssuedForExGratia(t *testing.T) {
	l := setup(t)
	coinsLeader(l)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.ExGratia = true
	require.NoError(t, l.store.Save(ctx, claim))

	_, err = l.service.ListPLA(ctx, printPLA(task), l.caller)
	violation(t, err, registrasi.ViolationPLAExGratia)
}

// DownloadAllDocumentPLA: PLA dengan REMARKS kosong tidak dicetak.
func TestPLAPrintNeedsRemarks(t *testing.T) {
	l := setup(t)
	coinsLeader(l)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	list, err := l.service.ListPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	number := list.PLA[0].Number
	_, err = l.service.SavePLANotes(ctx, printPLA(task), map[string]string{number: "  "}, l.caller)
	require.NoError(t, err)

	command := printPLA(task)
	command.Number = number
	_, err = l.service.PrintPLA(ctx, command, l.caller)
	violation(t, err, registrasi.ViolationPLARemarksEmpty)
}

// Spreading BPPDAN menerbitkan PLA BPPDAN (PLABPPDAN_Act): penerima BPPDAN, huruf H, nilai
// reserve × persen bagian; dokumennya PLABPPDAN<nomor>.pdf.
func TestPLABPPDAN(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)

	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.InsuredItem[0].Coverage[0].Spreading[0].TreatyKind = "10010"
	claim.InsuredItem[0].Coverage[0].Spreading[0].Share = 25_000 // 2,5%
	require.NoError(t, l.store.Save(ctx, claim))

	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(40_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	list, err := l.service.ListPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Len(t, list.PLA, 1)
	p := list.PLA[0]
	require.Equal(t, registrasi.PLATypeBPPDAN, p.Type)
	require.Equal(t, "10038311", p.RecipientCode)
	require.True(t, strings.HasPrefix(p.Number, registrasi.PLACodeBPPDAN))
	require.Equal(t, registrasi.Rupiah(1_000_000), p.Amount[0].Result, "2,5% × 40 jt")

	printed, err := l.service.PrintPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, "PLABPPDAN"+p.Number+".pdf", printed.FileName)
}
