package memory

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Uji langsung penyimpanan memori. Penyimpanan ini dipakai pengembangan lokal dan seluruh uji
// usecase, sehingga perilakunya — termasuk rollback transaksi — harus sama dengan janjinya.

var bg = context.Background()

var at = time.Date(2026, 6, 9, 3, 0, 0, 0, time.UTC)

func sampleClaim(id, number string) registrasi.Claim {
	return registrasi.Claim{
		ID: id, Number: number, Location: "Gudang  A",
		Policy: registrasi.Policy{Number: "POL-1"},
		InsuredItem: []registrasi.InsuredItem{{
			ID: "OBJ1",
			Coverage: []registrasi.Coverage{{
				ID: "C1", CauseOfLoss: "12002",
				Spreading:  []registrasi.Spreading{{TreatyKind: "10001", Share: registrasi.PercentFull}},
				Settlement: []registrasi.SettlementLine{{PaymentType: "1"}},
			}},
		}},
		Receiver: []registrasi.Receiver{{ID: "1"}},
	}
}

// Klaim tersimpan sebagai salinan: mengubah hasil Get tidak mengubah isi penyimpanan.
func TestStoreSaveGetCopies(t *testing.T) {
	s := NewStore()
	deleted := at
	k := sampleClaim("K1", "PNCN.26.1")
	k.DeletedAt = &deleted
	k.ClaimStatus = "1161"
	require.NoError(t, s.Save(bg, k))

	got, err := s.Get(bg, "K1")
	require.NoError(t, err)
	require.Equal(t, registrasi.ClaimStatusNames["1161"], got.ClaimStatusName)
	got.InsuredItem[0].Coverage[0].Spreading[0].Share = 1
	got.Receiver[0].ID = "X"
	*got.DeletedAt = time.Time{}

	again, err := s.Get(bg, "K1")
	require.NoError(t, err)
	require.Equal(t, registrasi.PercentFull, again.InsuredItem[0].Coverage[0].Spreading[0].Share)
	require.Equal(t, "1", again.Receiver[0].ID)
	require.Equal(t, at, *again.DeletedAt)

	_, err = s.Get(bg, "TIDAK")
	require.ErrorIs(t, err, registrasi.ErrClaimNotFound)

	byNumber, err := s.GetByNumber(bg, "PNCN.26.1")
	require.NoError(t, err)
	require.Equal(t, "K1", byNumber.ID)
	_, err = s.GetByNumber(bg, "PNCN.26.9")
	require.ErrorIs(t, err, registrasi.ErrClaimNotFound)
}

// Transaksi yang gagal mengembalikan klaim, tugas, audit, dan pemberitahuan seperti semula.
func TestStoreRunRollsBack(t *testing.T) {
	s := NewStore()
	require.NoError(t, s.Save(bg, sampleClaim("K1", "PNCN.26.1")))
	boom := errors.New("gagal")

	err := s.Run(bg, func(ctx context.Context) error {
		require.NoError(t, s.Save(ctx, sampleClaim("K2", "PNCN.26.2")))
		require.NoError(t, s.SaveTask(ctx, registrasi.Task{ID: "T1", ClaimID: "K2"}))
		require.NoError(t, s.Record(ctx, registrasi.AuditTrail{Event: "X"}))
		require.NoError(t, s.Send(ctx, registrasi.Notification{Kind: "Y"}))
		return boom
	})
	require.ErrorIs(t, err, boom)
	_, err = s.Get(bg, "K2")
	require.ErrorIs(t, err, registrasi.ErrClaimNotFound)
	_, err = s.ClaimTask(bg, "T1")
	require.ErrorIs(t, err, registrasi.ErrTaskNotFound)
	require.Empty(t, s.AuditTrail())
	require.Empty(t, s.Notification())

	require.NoError(t, s.Run(bg, func(ctx context.Context) error {
		require.NoError(t, s.Record(ctx, registrasi.AuditTrail{Event: "X"}))
		return s.Send(ctx, registrasi.Notification{Kind: "Y"})
	}))
	require.Len(t, s.AuditTrail(), 1)
	require.Len(t, s.Notification(), 1)
}

// Duplikat: polis, objek, lokasi (spasi dan huruf diabaikan), dan penyebab kerugian.
func TestStoreFindDuplicates(t *testing.T) {
	s := NewStore()
	require.NoError(t, s.Save(bg, sampleClaim("K1", "PNCN.26.1")))
	require.NoError(t, s.Save(bg, sampleClaim("K2", "")))
	gone := sampleClaim("K3", "PNCN.26.3")
	gone.DeletedAt = &at
	require.NoError(t, s.Save(bg, gone))

	key := registrasi.DuplicateKey{PolicyNumber: "POL-1", InsuredItemID: "OBJ1", Location: "gudang a"}
	got, err := s.FindDuplicates(bg, []registrasi.DuplicateKey{key, key}, "")
	require.NoError(t, err)
	require.Equal(t, []registrasi.DuplicateClaim{{Number: "PNCN.26.1", InsuredItem: "OBJ1"}}, got)

	got, err = s.FindDuplicates(bg, []registrasi.DuplicateKey{key}, "K1")
	require.NoError(t, err)
	require.Empty(t, got)

	cause := registrasi.DuplicateKey{PolicyNumber: "POL-1", InsuredItemID: "OBJ1", CauseOfLoss: "12002"}
	got, err = s.FindDuplicates(bg, []registrasi.DuplicateKey{cause}, "")
	require.NoError(t, err)
	require.Len(t, got, 1)

	for _, miss := range []registrasi.DuplicateKey{
		{PolicyNumber: "POL-2", InsuredItemID: "OBJ1"},
		{PolicyNumber: "POL-1", InsuredItemID: "OBJ1", Location: "Gudang B"},
		{PolicyNumber: "POL-1", InsuredItemID: "OBJ9"},
		{PolicyNumber: "POL-1", InsuredItemID: "OBJ1", CauseOfLoss: "99999"},
	} {
		got, err = s.FindDuplicates(bg, []registrasi.DuplicateKey{miss}, "")
		require.NoError(t, err)
		require.Empty(t, got, "%+v", miss)
	}
}

// Inbox: worklist milik sendiri, workbasket bebas yang diizinkan, workbasket yang diambil
// sendiri, dan worklist tahap grup; tugas selesai disaring.
func TestStoreTasksAndInbox(t *testing.T) {
	s := NewStore()
	repo := s.TaskRepo()
	done := at
	tasks := []registrasi.Task{
		{ID: "T1", ClaimID: "K1", Queue: registrasi.QueueWorklist, Owner: "NIK1", CreatedAt: at},
		{ID: "T2", ClaimID: "K1", Queue: registrasi.QueueWorkbasket, Workbasket: "WB", CreatedAt: at.Add(time.Hour)},
		{ID: "T3", ClaimID: "K2", Queue: registrasi.QueueWorkbasket, Workbasket: "WB", Owner: "NIK1", CreatedAt: at.Add(2 * time.Hour)},
		{ID: "T4", ClaimID: "K3", Queue: registrasi.QueueWorklist, Owner: "LAIN", Stage: "ST", CreatedAt: at.Add(3 * time.Hour)},
		{ID: "T5", ClaimID: "K3", Queue: registrasi.QueueWorklist, Owner: "NIK1", CompletedAt: &done, CreatedAt: at},
		{ID: "T6", ClaimID: "K4", Queue: registrasi.QueueWorkbasket, Workbasket: "WB-LAIN", CreatedAt: at},
		{ID: "T0", ClaimID: "K1", Queue: registrasi.QueueWorklist, Owner: "X", CreatedAt: at},
	}
	for _, task := range tasks {
		require.NoError(t, repo.Save(bg, task))
	}

	got, err := repo.Inbox(bg, "NIK1", []string{"WB"}, []string{"ST"})
	require.NoError(t, err)
	ids := []string{}
	for _, task := range got {
		ids = append(ids, task.ID)
	}
	require.Equal(t, []string{"T1", "T2", "T3", "T4"}, ids)

	task, err := repo.Get(bg, "T2")
	require.NoError(t, err)
	require.Equal(t, "WB", task.Workbasket)

	open, err := repo.OpenTaskForClaim(bg, "K1")
	require.NoError(t, err)
	require.Equal(t, "T0", open.ID, "urut waktu lalu ID")
	_, err = repo.OpenTaskForClaim(bg, "K9")
	require.ErrorIs(t, err, registrasi.ErrTaskNotFound)
}

func TestPolicyStoreAndSamples(t *testing.T) {
	policies := SamplePolicies(at)
	require.Len(t, policies, 5)
	p := NewPolicyStore(policies...)
	got, err := p.Get(bg, " pol-fire-0001 ")
	require.NoError(t, err)
	require.Equal(t, registrasi.LineFire, got.Line)
	require.True(t, got.CoverageStart.Before(at))
	require.True(t, got.CoverageEnd.After(at))
	spk, err := p.Get(bg, "POL-SPK-0004")
	require.NoError(t, err)
	require.True(t, spk.CreditGuarantee)
	_, err = p.Get(bg, "POL-TIDAK")
	require.ErrorIs(t, err, registrasi.ErrPolicyNotFound)
}

// Nomor klaim berurut per tahun WIB.
func TestNumberIssuerSequencePerYear(t *testing.T) {
	n := NewNumberIssuer()
	a, _ := n.Issue(bg, at)
	b, _ := n.Issue(bg, at)
	c, _ := n.Issue(bg, time.Date(2026, 12, 31, 18, 0, 0, 0, time.UTC))
	require.Equal(t, []string{"PNCN.26.1", "PNCN.26.2", "PNCN.27.1"}, []string{a, b, c})
}

func TestParameterSetters(t *testing.T) {
	p := NewParameter()
	v, err := p.LargeLossThreshold(bg)
	require.NoError(t, err)
	require.Equal(t, registrasi.Rupiah(1_000_000_000), v)
	p.SetThreshold(5)
	v, _ = p.LargeLossThreshold(bg)
	require.Equal(t, registrasi.Money(5), v)

	p.SetGeneral([]string{"z@x"})
	p.SetRecipients(registrasi.LineTravel, []string{"a@x"})
	got, err := p.LargeLossRecipients(bg, registrasi.LineTravel)
	require.NoError(t, err)
	require.Equal(t, []string{"a@x", "z@x"}, got)
}

func TestExchangeRateSource(t *testing.T) {
	s := NewExchangeRateSource()
	rate, err := s.Find(bg, " idr ", at)
	require.NoError(t, err)
	require.Equal(t, registrasi.ExchangeRateOne, rate)
	_, err = s.Find(bg, "USD", at)
	require.ErrorIs(t, err, registrasi.ErrExchangeRateNotFound)
	s.Set(" usd ", 150_000_000)
	rate, err = s.Find(bg, "USD", at)
	require.NoError(t, err)
	require.Equal(t, registrasi.ExchangeRate(150_000_000), rate)
}

// Penugasan memori: aturan yang sama dengan penugas SQL, lalu tim dengan beban teringan.
func TestAssignerMemory(t *testing.T) {
	a := NewAssigner(SampleTeams())
	got, _ := a.Assign(bg, registrasi.Stage{Queue: registrasi.QueueWorkbasket, Workbasket: "WB"}, registrasi.Claim{}, "NIK1")
	require.Equal(t, registrasi.Assignee{Workbasket: "WB"}, got)
	got, _ = a.Assign(bg, registrasi.Stage{Operator: "DOKTER"}, registrasi.Claim{}, "NIK1")
	require.Equal(t, registrasi.Assignee{Operator: "DOKTER"}, got)
	got, _ = a.Assign(bg, registrasi.Stage{Router: registrasi.RouterCurrentOperator}, registrasi.Claim{}, "NIK1")
	require.Equal(t, registrasi.Assignee{Operator: "NIK1"}, got)
	got, _ = a.Assign(bg, registrasi.Stage{Router: registrasi.RouterPNCAdmin}, registrasi.Claim{CreatedBy: "ADMIN"}, "NIK1")
	require.Equal(t, registrasi.Assignee{Operator: "ADMIN"}, got)
	got, _ = a.Assign(bg, registrasi.Stage{Router: registrasi.RouterPNCAdmin}, registrasi.Claim{}, "NIK1")
	require.Equal(t, registrasi.Assignee{Operator: "NIK1"}, got)
	got, _ = a.Assign(bg, registrasi.Stage{Router: registrasi.RouterPNCTechnical}, registrasi.Claim{TechnicalPIC: "PIC"}, "NIK1")
	require.Equal(t, registrasi.Assignee{Operator: "PIC"}, got)

	tech := registrasi.Stage{Router: registrasi.RouterPNCTechnical}
	first, _ := a.Assign(bg, tech, registrasi.Claim{}, "NIK1")
	second, _ := a.Assign(bg, tech, registrasi.Claim{}, "NIK1")
	third, _ := a.Assign(bg, tech, registrasi.Claim{}, "NIK1")
	require.Equal(t, []string{"TEKNIK01", "TEKNIK02", "TEKNIK01"}, []string{first.Operator, second.Operator, third.Operator})
	require.Equal(t, map[string]int{"TEKNIK01": 2, "TEKNIK02": 1}, a.Load())

	got, _ = a.Assign(bg, registrasi.Stage{Router: "TanpaTim"}, registrasi.Claim{}, "NIK1")
	require.Equal(t, registrasi.Assignee{Operator: "NIK1"}, got)
}

func TestIDGeneratorAndSameText(t *testing.T) {
	a, b := IDGenerator{}.New(), IDGenerator{}.New()
	require.Len(t, a, 32)
	require.NotEqual(t, a, b)
	require.True(t, sameText(" Gudang   A ", "gudang a"))
	require.False(t, sameText("Gudang A", "Gudang B"))
}

// Laporan: nomor klaim hanya terpasang setelah diserahkan; potret memakai nomor terpasang.
func TestClaimReportLinkMemory(t *testing.T) {
	l := NewClaimReportLink()
	require.ErrorContains(t, l.AttachClaimNumber(bg, "RCV1", "PNCN.26.1"), "belum ditandai diserahkan")
	require.False(t, l.HandedOver("RCV1"))

	require.NoError(t, l.MarkHandedOver(bg, "RCV1", at))
	require.NoError(t, l.MarkHandedOver(bg, "RCV1", at.Add(time.Hour)))
	require.True(t, l.HandedOver("RCV1"))
	require.NoError(t, l.AttachClaimNumber(bg, "RCV1", "PNCN.26.1"))
	require.Equal(t, "PNCN.26.1", l.ClaimNumber("RCV1"))

	empty, err := l.Snapshot(bg, "RCV2")
	require.NoError(t, err)
	require.Equal(t, registrasi.ClaimReportSnapshot{}, empty)
	l.SetSnapshot("RCV1", registrasi.ClaimReportSnapshot{Location: "Gudang", ClaimNumber: "LAMA"})
	snap, err := l.Snapshot(bg, "RCV1")
	require.NoError(t, err)
	require.Equal(t, "Gudang", snap.Location)
	require.Equal(t, "PNCN.26.1", snap.ClaimNumber)
}

func TestAcceptanceMemory(t *testing.T) {
	s := NewAcceptance()
	n1, _ := s.NextNumber(bg, 2026)
	n2, _ := s.NextNumber(bg, 2026)
	require.Equal(t, registrasi.PLANumber(registrasi.AcceptanceCode, 2026, "10", 1), n1)
	require.NotEqual(t, n1, n2)

	line := registrasi.SettlementLine{Acceptance: registrasi.Acceptance{Form: registrasi.AcceptanceForm{Remark: "OK"}}}
	require.NoError(t, s.Save(bg, "K1", "O", 1, 1, line))
	var read registrasi.SettlementLine
	require.NoError(t, s.Fields(bg, "K1", "O", 1, 1, &read))
	require.Equal(t, "OK", read.Acceptance.Form.Remark)
	var none registrasi.SettlementLine
	require.NoError(t, s.Fields(bg, "K1", "O", 1, 2, &none))
	require.Empty(t, none.Acceptance.Form.Remark)

	require.NoError(t, s.RecordLODPrint(bg, "K1", "O", 1, 1, at, "4"))
	require.NoError(t, s.RecordLODPrint(bg, "K1", "O", 1, 1, at.Add(time.Hour), "5"))
	require.Equal(t, LODPrint{PrintedAt: at, Type: "5"}, s.LODPrint["K1/O/1/1"])

	s.DLA["K1"] = []registrasi.AcceptanceDLAState{{Number: "D1"}}
	dla, err := s.OtherDLA(bg, "K1", "O", 1, 1)
	require.NoError(t, err)
	require.Len(t, dla, 1)

	require.NoError(t, s.AddHistory(bg, "C1", "catat", "NIK1", at))
	require.Equal(t, []AcceptanceHistory{{CaseID: "C1", Note: "catat", User: "NIK1", At: at}}, s.History)

	id, err := s.OpenPosition(bg, "PNCN.26.1", registrasi.PositionAcceptance)
	require.NoError(t, err)
	require.Empty(t, id)
	id, err = s.StartProgress(bg, registrasi.ProgressStart{ClaimNumber: "PNCN.26.1", Position: registrasi.PositionAcceptance})
	require.NoError(t, err)
	require.Equal(t, "1", id)
	open, _ := s.OpenPosition(bg, "PNCN.26.1", registrasi.PositionAcceptance)
	require.Equal(t, "1", open)

	// Progres yang tidak Done tidak menutup posisi; yang Done menutupnya.
	require.NoError(t, s.AddProgress(bg, registrasi.ProgressUpdate{ClaimNumber: "PNCN.26.1", PositionID: "1", Position: "On Progress"}))
	open, _ = s.OpenPosition(bg, "PNCN.26.1", registrasi.PositionAcceptance)
	require.Equal(t, "1", open)
	require.NoError(t, s.AddProgress(bg, registrasi.ProgressUpdate{ClaimNumber: "PNCN.26.1", PositionID: "1", Position: registrasi.AcceptanceProgressDone}))
	open, _ = s.OpenPosition(bg, "PNCN.26.1", registrasi.PositionAcceptance)
	require.Empty(t, open)
	require.Len(t, s.Progress, 2)

	s.CaseIDs = map[string]string{"POL": "CASE"}
	s.OpenProtection = map[string]bool{"POL": true}
	s.TravelClients = map[string]string{"POL": "AGEN"}
	caseID, _ := s.PolicyCaseID(bg, "POL")
	approved, _ := s.OpenProtectionApproved(bg, "POL", "K1", "N1")
	client, _ := s.TravelClientName(bg, "POL")
	require.Equal(t, "CASE", caseID)
	require.True(t, approved)
	require.Equal(t, "AGEN", client)
}

func TestCashierMemory(t *testing.T) {
	c := NewCashier()
	c.Banks["BANK A"] = "001"
	id, ok, err := c.BankGroupID(bg, "BANK A", "001")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "001", id)
	_, ok, _ = c.BankGroupID(bg, "BANK A", "002")
	require.False(t, ok)

	require.NoError(t, c.Log(bg, registrasi.CashierLog{AcceptedNo: "A1"}))
	require.NoError(t, c.MarkTransferred(bg, "K1", "O", 1, 2, at, "ECR"))
	require.Equal(t, []CashierMark{{"K1", "O", 1, 2, at, "ECR"}}, c.Marked)

	c.Reply = registrasi.CashierReply{ResponseMessage: "SUCCESS"}
	reply, err := c.Transfer(bg, "ASM", "PAID", registrasi.CashierPayload{})
	require.NoError(t, err)
	require.Equal(t, "SUCCESS", reply.ResponseMessage)
	c.Err = errors.New("timeout")
	_, err = c.Transfer(bg, "ASM", "PAID", registrasi.CashierPayload{})
	require.EqualError(t, err, "timeout")
	require.Equal(t, []string{"PAID", "PAID"}, c.Services)
	require.Len(t, c.Logs, 1)
}

func TestCommitteesMemory(t *testing.T) {
	c := NewCommittees()
	id1, err := c.NextCaseID(bg, at)
	require.NoError(t, err)
	id2, _ := c.NextCaseID(bg, at)
	want1, _ := registrasi.FormatCommitteeCaseID(2026, 1)
	want2, _ := registrasi.FormatCommitteeCaseID(2026, 2)
	require.Equal(t, want1, id1)
	require.Equal(t, want2, id2)

	k := registrasi.NewCommitteeCase(id1, "PNCN.26.1",
		[]registrasi.CommitteeApprover{{OperatorID: "KOMITE01"}, {OperatorID: "KOMITE02"}},
		registrasi.SettlementLine{PaymentType: "1"}, 900, at)
	require.NoError(t, c.Save(bg, k))
	got, err := c.Get(bg, " "+id1+" ")
	require.NoError(t, err)
	require.Len(t, got.Members, 2)
	_, err = c.Get(bg, "TIDAK")
	require.ErrorIs(t, err, registrasi.ErrCommitteeNotFound)

	pending, err := c.Pending(bg, " komite01 ")
	require.NoError(t, err)
	require.Len(t, pending, 1)
	pending, _ = c.Pending(bg, "KOMITE02")
	require.Empty(t, pending)

	tiers := NewCommitteeTiering()
	route, err := tiers.Route(bg, "NONMBU", 1, " komite01 ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.CommitteeApprover{{OperatorID: "KOMITE02", Name: "KOMITE CONTOH 2"}}, route.Approvers)
	custom := NewCommitteeTiering(registrasi.CommitteeApprover{OperatorID: "A"})
	require.Len(t, custom.List, 1)
}

func TestRecordsAndUploaderMemory(t *testing.T) {
	r := SampleClaimRecords()
	types, err := r.DocumentTypes(bg, "")
	require.NoError(t, err)
	require.Nil(t, types)
	types, _ = r.DocumentTypes(bg, "10140")
	require.Len(t, types, 6)

	require.NoError(t, r.AddAttachment(bg, registrasi.NewAttachment{
		ClaimKey: "ASM-FW-GCNMFW-WORK PNCN.26.1", Name: "a.pdf", Extension: "pdf", ImageID: "IMG-1", By: "NIK1", At: at,
	}))
	require.NoError(t, r.AddAttachment(bg, registrasi.NewAttachment{ClaimKey: "PNCN.26.1", Name: "b.pdf"}))
	keys := registrasi.RecordKeys{Number: "PNCN.26.1"}
	files, err := r.Attachments(bg, keys)
	require.NoError(t, err)
	require.Len(t, files, 2)
	require.Equal(t, "1", files[0].ID)
	require.Equal(t, "2", files[1].ID)
	require.Equal(t, "IMG-1", files[0].ImageID)

	r.Survey = map[string][]registrasi.Survey{"PNCN.26.1": {{CaseID: "S"}}}
	r.ProgressEntry = map[string][]registrasi.ProgressEntry{"PNCN.26.1": {{Seq: 1}}}
	r.Communication = map[string][]registrasi.Communication{"PNCN.26.1": {{ID: "1"}}}
	s, _ := r.Surveys(bg, keys)
	p, _ := r.Progress(bg, keys)
	c, _ := r.Communications(bg, keys)
	require.Len(t, s, 1)
	require.Len(t, p, 1)
	require.Len(t, c, 1)

	u := &DocumentUploader{}
	id, err := u.Upload(bg, registrasi.DocumentFile{ClaimNumber: "PNCN.26.1"})
	require.NoError(t, err)
	require.Equal(t, "IMG-1", id)
	id, _ = u.Upload(bg, registrasi.DocumentFile{})
	require.Equal(t, "IMG-2", id)
}

func TestFaceSheetMemory(t *testing.T) {
	f := NewFaceSheet()
	f.CauseOfLoss["1"] = "API"
	f.Operator["NIK1"] = "Budi"
	f.CoMember["POL"] = []registrasi.CoinsuranceRow{{CoinsName: "A"}}
	f.Reinsurer["POL"] = []registrasi.FacReinsurer{{Name: "R"}}
	name, _ := f.CauseOfLossName(bg, "1")
	op, _ := f.OperatorName(bg, "NIK1")
	co, _ := f.Coinsurance(bg, "POL")
	re, _ := f.FacReinsurers(bg, "POL")
	require.Equal(t, "API", name)
	require.Equal(t, "Budi", op)
	require.Len(t, co, 1)
	require.Len(t, re, 1)

	_, found, err := f.LastRevision(bg, "K1", "O", 1)
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, f.SaveRevision(bg, registrasi.FaceSheetRevision{ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, Revision: 0}))
	require.NoError(t, f.SaveRevision(bg, registrasi.FaceSheetRevision{ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, Revision: 2}))
	require.NoError(t, f.SaveRevision(bg, registrasi.FaceSheetRevision{ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, Revision: 1}))
	require.ErrorIs(t, f.SaveRevision(bg, registrasi.FaceSheetRevision{ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, Revision: 2}), registrasi.ErrInvalidAction)
	last, found, _ := f.LastRevision(bg, "K1", "O", 1)
	require.True(t, found)
	require.Equal(t, 2, last)
}

func TestPLAMemory(t *testing.T) {
	s := NewPLA()
	s.Coins["POL"] = []registrasi.PLACoinsMember{{ID: "C1"}}
	s.Recipients["C1"] = registrasi.PLARecipientInfo{Email: "e@x"}
	coins, _ := s.CoinsMembers(bg, "POL")
	info, _ := s.Recipient(bg, "C1", "")
	require.Len(t, coins, 1)
	require.Equal(t, "e@x", info.Email)

	n1, _ := s.NextNumber(bg, "J", 2026)
	require.Equal(t, registrasi.PLANumber("J", 2026, "1", 1), n1)
	require.NoError(t, s.Save(bg, registrasi.PLA{ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, Number: "P1", RecipientCode: "C1", Date: at}))
	require.NoError(t, s.Save(bg, registrasi.PLA{ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, Number: "P2", RecipientCode: "C1", Date: at.Add(time.Hour)}))
	require.ErrorContains(t, s.Save(bg, registrasi.PLA{Number: "P1"}), "nomor PLA P1 ganda")
	// PLA tanpa tanggal diberi tanggal penyimpanan.
	require.NoError(t, s.Save(bg, registrasi.PLA{ClaimID: "K2", Number: "P3"}))
	require.False(t, s.Saved[2].Date.IsZero())

	prev, found, _ := s.Previous(bg, "K1", "C1")
	require.True(t, found)
	require.Equal(t, "P2", prev.Number)
	_, found, _ = s.Previous(bg, "K1", "C9")
	require.False(t, found)

	issued, _ := s.Issued(bg, "K1", "O", 1, 0)
	require.Len(t, issued, 2)
	require.NoError(t, s.UpdateNote(bg, "K1", "P1", 0, "catatan"))
	require.Equal(t, "catatan", s.Saved[0].Note)
	require.ErrorContains(t, s.UpdateNote(bg, "K1", "P9", 0, "x"), "PLA P9 tidak ada")

	name, png, err := s.Signature(bg, "ASM")
	require.NoError(t, err)
	require.Empty(t, name)
	require.Nil(t, png)

	s.InsuredEmails = map[string]string{"PNCN.26.1": "t@x"}
	s.PICEmails = map[string]string{"PIC": "p@x"}
	insured, pic, err := s.LODEmails(bg, "PNCN.26.1", "PIC")
	require.NoError(t, err)
	require.Equal(t, "t@x", insured)
	require.Equal(t, "p@x", pic)
}

func TestDLAMemory(t *testing.T) {
	s := NewDLA()
	s.Policies["POL"] = registrasi.DLAPolicy{CaseID: "C"}
	s.Cases["10008"] = "2"
	s.Treaties["10008"] = registrasi.TreatyArrangement{Limit: "1"}
	s.Pre = []registrasi.DLA{{ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, AdjustmentSeq: 1, Number: "PRE"}, {ClaimID: "K2"}}

	p, _ := s.Policy(bg, "POL")
	require.Equal(t, "C", p.CaseID)
	cases, _ := s.ReinsuranceCase(bg)
	cases["10008"] = "diubah"
	again, _ := s.ReinsuranceCase(bg)
	require.Equal(t, "2", again["10008"], "peta dikembalikan sebagai salinan")
	treaty, _ := s.Treaty(bg, "B", 2026, "10008")
	require.Equal(t, "1", treaty.Limit)
	pre, _ := s.PreDLA(bg, "K1", "O", 1, 1)
	require.Len(t, pre, 1)

	n, _ := s.NextNumber(bg, "H", 2026)
	require.Equal(t, registrasi.PLANumber("H", 2026, "1", 1), n)
	require.NoError(t, s.Save(bg, registrasi.DLA{ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, AdjustmentSeq: 1, Number: "D1", RecipientCode: "77", Date: at}))
	require.NoError(t, s.Save(bg, registrasi.DLA{ClaimID: "K1", ObjectID: "O", CoverageSeq: 1, AdjustmentSeq: 2, Number: "D2", RecipientCode: "77", Date: at.Add(time.Hour)}))
	require.ErrorContains(t, s.Save(bg, registrasi.DLA{ClaimID: "K1", Number: "D1"}), "nomor DLA D1 ganda")

	issued, _ := s.Issued(bg, "K1", "O", 1, 1)
	require.Len(t, issued, 1)
	prev, found, _ := s.Previous(bg, "K1", "77")
	require.True(t, found)
	require.Equal(t, "D2", prev.Number)
	_, found, _ = s.Previous(bg, "K1", "88")
	require.False(t, found)

	require.NoError(t, s.MarkPrinted(bg, "K1", "D1", "dicetak"))
	require.True(t, s.Saved[0].Printed)
	require.Equal(t, "dicetak", s.Saved[0].Note)
	require.False(t, s.Saved[1].Printed)
}

func TestPremiumMemory(t *testing.T) {
	s := NewPremium()
	st, err := s.Statement(bg, "ASM", registrasi.PremiumQuery{PolicyNumber: "POL"})
	require.NoError(t, err)
	require.Equal(t, 0, st.AgingAmount.Sign())

	s.Statements["POL"] = registrasi.PremiumStatement{AgingAmount: big.NewRat(5, 1)}
	st, _ = s.Statement(bg, "ASM", registrasi.PremiumQuery{PolicyNumber: "POL"})
	require.Equal(t, big.NewRat(5, 1), st.AgingAmount)

	s.Err = errors.New("layanan mati")
	_, err = s.Statement(bg, "ASM", registrasi.PremiumQuery{PolicyNumber: "POL"})
	require.EqualError(t, err, "layanan mati")
}

func TestInboxEntriesMemory(t *testing.T) {
	s := NewInboxEntries()
	require.NoError(t, s.Mirror(bg, registrasi.InboxEntry{}))
	_, ok := s.Get("")
	require.False(t, ok)

	task := registrasi.Task{ID: "T1"}
	entry := registrasi.InboxEntry{Claim: registrasi.Claim{Number: "PNCN.26.1"}, Task: &task}
	require.NoError(t, s.Mirror(bg, entry))
	task.ID = "DIUBAH"
	got, ok := s.Get("PNCN.26.1")
	require.True(t, ok)
	require.Equal(t, "T1", got.Task.ID, "tugas disalin saat dicerminkan")
}

func TestPolicyItemsAndCurrenciesMemory(t *testing.T) {
	p := NewPolicyItems(SamplePolicyItems())
	items, err := p.Items(bg, registrasi.Policy{Number: "POL-FIRE-0001"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "100819", items[0].Coverage[0].Code)

	opts, _ := p.ItemOptions(bg, registrasi.Policy{Line: registrasi.LineFire}, "1")
	require.Len(t, opts, 2)
	opts, _ = p.ItemOptions(bg, registrasi.Policy{Line: registrasi.LineTravel}, "1")
	require.Nil(t, opts)

	cur, err := CurrencyDirectory{}.Currencies(bg)
	require.NoError(t, err)
	require.Equal(t, "IDR", cur[0].Name)
}

func TestAreaGroupsAndAccountsMemory(t *testing.T) {
	d := NewAreaDirectory()
	countries, err := d.Options(bg, registrasi.AreaCountry, "abaikan")
	require.NoError(t, err)
	require.Len(t, countries, 2)
	cities, _ := d.Options(bg, registrasi.AreaCity, " 10012 ")
	require.Len(t, cities, 2)
	_, err = d.Options(bg, registrasi.AreaLevel("planet"), "")
	require.ErrorIs(t, err, registrasi.ErrUnknownAreaLevel)

	g := Groups{"NIK1": {"PNCADMIN"}}
	groups, _ := g.GroupsOf(bg, " nik1 ")
	require.Equal(t, []string{"PNCADMIN"}, groups)
	groups, _ = g.GroupsOf(bg, "LAIN")
	require.Nil(t, groups)

	a := NewAccounts()
	acc, err := a.FindAccount(bg, " 1234567890 ")
	require.NoError(t, err)
	require.Equal(t, "PT CONTOH PENERIMA", acc.Name)
	_, err = a.FindAccount(bg, "000")
	require.ErrorIs(t, err, registrasi.ErrAccountNotFound)
	a.Add(registrasi.BankAccount{Number: " 000 ", Name: "BARU"})
	acc, _ = a.FindAccount(bg, "000")
	require.Equal(t, "BARU", acc.Name)
}
