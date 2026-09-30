package usecase

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// DLACommand adalah permintaan tombol "Print DLA" satu baris adjustment. Object, Coverage,
// dan Adjustment berbasis 1.
type DLACommand struct {
	ClaimID    string
	TaskID     string
	Object     int
	Coverage   int
	Adjustment int

	// Number memilih satu DLA untuk dicetak (PRINT per baris). Kosong berarti seluruhnya
	// (Print All DLA).
	Number string
	// Remarks adalah isian REMARKS per nomor DLA yang belum pernah dicetak.
	Remarks map[string]string
	// AsPerPolicy adalah centang "Coverage AS PER ORIGINAL POLICY" (`.CheckCoverage`).
	AsPerPolicy bool
}

// DLAList adalah isi layar PrintDLA.
type DLAList struct {
	DLA    []registrasi.DLA
	Issued int // jumlah DLA yang baru terbit saat daftar dibuka

	// ExGratia menyatakan klaim ex gratia non-ASO: Pega tidak menerbitkan DLA untuknya.
	ExGratia bool
	// Warning adalah WarningDLAPages — baris tanpa nomor atau kode reasuradur.
	Warning []string
}

// DLAResult adalah dokumen yang diunduh: satu PDF, atau ZIP bila lebih dari satu DLA.
type DLAResult struct {
	FileName    string
	ContentType string
	Content     []byte
}

type dlaScope struct {
	claim    registrasi.Claim
	object   registrasi.InsuredItem
	coverage registrasi.Coverage
	line     registrasi.SettlementLine
	p        DLACommand
}

func (l *Service) dlaScopeOf(ctx context.Context, p DLACommand, by Caller) (dlaScope, error) {
	claim, _, err := l.lodTask(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return dlaScope{}, err
	}
	line, err := settlementAt(&claim, p.Object, p.Coverage, p.Adjustment)
	if err != nil {
		return dlaScope{}, err
	}
	if err := registrasi.CanPrintDLA(*line); err != nil {
		return dlaScope{}, err
	}
	object := claim.InsuredItem[p.Object-1]
	return dlaScope{claim: claim, object: object, coverage: object.Coverage[p.Coverage-1], line: *line, p: p}, nil
}

// ListDLA membuka layar PrintDLA: menerbitkan DLA adjustment itu bila belum ada
// (pra-proses GenerateDLAListAdjustment), lalu mengembalikan daftarnya.
func (l *Service) ListDLA(ctx context.Context, p DLACommand, by Caller) (DLAList, error) {
	sc, err := l.dlaScopeOf(ctx, p, by)
	if err != nil {
		return DLAList{}, err
	}
	return l.dlaIssuedOrNew(ctx, sc, by)
}

func (l *Service) dlaIssuedOrNew(ctx context.Context, sc dlaScope, by Caller) (DLAList, error) {
	list, err := l.dla.Issued(ctx, sc.claim.ID, sc.object.ID, sc.p.Coverage, sc.p.Adjustment)
	if err != nil {
		return DLAList{}, err
	}
	if len(list) > 0 {
		return DLAList{DLA: list, Warning: dlaWarnings(list)}, nil
	}
	policy, err := l.dla.Policy(ctx, sc.claim.Policy.Number)
	if err != nil {
		return DLAList{}, err
	}
	// GenerateDLAList langkah 14: klaim ex gratia non-ASO keluar tanpa DLA.
	if registrasi.DLASkipsExGratia(sc.claim.ExGratia, registrasi.FlagASO(policy.Coins)) {
		return DLAList{ExGratia: true}, nil
	}
	issued, err := l.issueDLA(ctx, sc, policy, by)
	if err != nil {
		return DLAList{}, err
	}
	list, err = l.dla.Issued(ctx, sc.claim.ID, sc.object.ID, sc.p.Coverage, sc.p.Adjustment)
	if err != nil {
		return DLAList{}, err
	}
	return DLAList{DLA: list, Issued: issued, Warning: dlaWarnings(list)}, nil
}

func dlaWarnings(list []registrasi.DLA) []string {
	var out []string
	for _, d := range list {
		if d.Number == "" || d.RecipientCode == "" {
			out = append(out, "- Error Reas DLA "+d.Number+" Kosong")
		}
	}
	return out
}

// issueDLA menerbitkan DLA satu adjustment dalam satu transaksi — GenerateDLAList.
func (l *Service) issueDLA(ctx context.Context, sc dlaScope, policy registrasi.DLAPolicy, by Caller) (int, error) {
	names, _, err := l.currencyNames(ctx)
	if err != nil {
		return 0, err
	}
	currency := firstText(names[sc.line.Currency], sc.line.Currency)
	policyCurrency := firstText(names[sc.claim.Policy.Currency], sc.claim.Policy.Currency)

	// Langkah 15–16: Pre DLA yang sudah ada disalin menjadi DLA (nomor yang sama), lalu
	// perhitungan baru dilewati (label B → END).
	pre, err := l.dla.PreDLA(ctx, sc.claim.ID, sc.object.ID, sc.p.Coverage, sc.p.Adjustment)
	if err != nil {
		return 0, err
	}
	var shares []registrasi.DLAShare
	var copied []registrasi.DLA
	if len(pre) > 0 {
		if sc.line.AcceptanceLODStatus == registrasi.LODAgreed && sc.line.AcceptedNo != "" {
			copied = pre
		}
	} else {
		if shares, err = l.dlaShares(ctx, sc, policy); err != nil {
			return 0, err
		}
	}
	if len(shares) == 0 && len(copied) == 0 {
		return 0, nil
	}

	now := l.clock.Now().UTC()
	year := clock.DateWIB(now).Year() // @CurrentDate("yy","WIB")
	count := 0
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		for _, d := range copied {
			d.ClaimID, d.ObjectID, d.CoverageSeq, d.AdjustmentSeq = sc.claim.ID, sc.object.ID, sc.p.Coverage, sc.p.Adjustment
			d.AcceptedNo, d.AcceptedDate, d.Date = sc.line.AcceptedNo, sc.line.Acceptance.AcceptedAt, now
			d.Printed, d.Sent = false, false
			note, err := l.dlaNote(ctx, sc.claim.ID, d.RecipientCode)
			if err != nil {
				return err
			}
			d.Note = note
			if err := l.dla.Save(ctx, d); err != nil {
				return err
			}
			count++
		}
		for _, s := range shares {
			number, err := l.dla.NextNumber(ctx, s.Code, year)
			if err != nil {
				return err
			}
			info, err := l.pla.Recipient(ctx, s.RecipientCode, s.Recipient)
			if err != nil {
				return err
			}
			note, err := l.dlaNote(ctx, sc.claim.ID, s.RecipientCode)
			if err != nil {
				return err
			}
			d := registrasi.DLA{
				ClaimID: sc.claim.ID, ObjectID: sc.object.ID, CoverageSeq: sc.p.Coverage, AdjustmentSeq: sc.p.Adjustment,
				Number: number, Type: s.Type, Recipient: s.Recipient, RecipientCode: s.RecipientCode, Date: now, Note: note,
				Value: s.Value, Percent: s.Percent, ShareSpread: s.ShareSpread, ClaimAmount: s.ClaimAmount,
				Currency: currency, PolicyCurrency: policyCurrency, QSPQS: s.QSPQS, QSRI: s.QSRI,
				AcceptedNo: sc.line.AcceptedNo, AcceptedDate: sc.line.Acceptance.AcceptedAt, Info: info,
			}
			if err := l.dla.Save(ctx, d); err != nil {
				return err
			}
			count++
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: sc.claim.ID, ClaimNumber: sc.claim.Number, Event: "DLA_TERBIT", Actor: by.Identity, At: now,
			Note: fmt.Sprintf("DLA objek %d jaminan %d adjustment %d: %d penerima", sc.p.Object, sc.p.Coverage, sc.p.Adjustment, count),
		})
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// dlaNote adalah catatan otomatis INSERT_PLADLA.prc cabang DLA.
func (l *Service) dlaNote(ctx context.Context, claimID, code string) (string, error) {
	prevDLA, hasDLA, err := l.dla.Previous(ctx, claimID, code)
	if err != nil {
		return "", err
	}
	prevPLA, hasPLA := registrasi.PLAPrevious{}, false
	if !hasDLA {
		if prevPLA, hasPLA, err = l.pla.Previous(ctx, claimID, code); err != nil {
			return "", err
		}
	}
	return registrasi.DLANote(prevPLA, hasPLA, prevDLA, hasDLA), nil
}

// dlaShares menjalankan pengolah per baris spreading lalu koasuransi — GenerateDLAList
// langkah 27 dan `DLACoins_act`.
func (l *Service) dlaShares(ctx context.Context, sc dlaScope, policy registrasi.DLAPolicy) ([]registrasi.DLAShare, error) {
	amount := registrasi.DLAAmountBase(sc.line)
	var out []registrasi.DLAShare

	var cases map[string]string
	var totalConvert *big.Rat
	for _, s := range sc.coverage.Spreading {
		if s.Removed {
			continue
		}
		if cases == nil {
			var err error
			if cases, err = l.dla.ReinsuranceCase(ctx); err != nil {
				return nil, err
			}
		}
		switch registrasi.RouteSpreading(s.TreatyKind, cases[s.TreatyKind]) {
		case registrasi.DLARouteBPPDAN:
			if d, ok := registrasi.BPPDANDLAShare(s.TreatyKind, s.Share, amount); ok {
				out = append(out, d)
			}
		case registrasi.DLARouteFacOut:
			out = append(out, registrasi.FacOutDLAShares(registrasi.FacOutInput{
				Portal: sc.claim.Portal, TreatyType: s.TreatyKind, GroupPanel: string(sc.claim.Policy.Line),
				Policy: policy, ObjectID: sc.object.ID, ObjectName: sc.object.Name,
				Coverage: sc.coverage.ID, CoverageSeq: sc.p.Coverage, Amount: amount,
				InitialBase: dlaInitialBase(sc, policy),
			})...)
		case registrasi.DLARouteTreaty:
			if totalConvert == nil {
				totalConvert = dlaTotalConvert(sc)
			}
			year := policy.StartYear
			if year == 0 && !sc.claim.Policy.CoverageStart.IsZero() {
				year = clock.DateWIB(sc.claim.Policy.CoverageStart).Year()
			}
			business := firstText(policy.BusinessCode, sc.claim.Policy.BusinessCode)
			arr, err := l.dla.Treaty(ctx, business, year, s.TreatyKind)
			if err != nil {
				return nil, err
			}
			out = append(out, registrasi.TreatyDLAShares(s.TreatyKind, s.Share, amount, totalConvert, arr)...)
		}
	}
	// DLACoins_act selalu dijalankan setelah spreading; dasarnya GrossValue.
	out = append(out, registrasi.CoinsDLAShares(sc.claim.Portal, policy.Coins, sc.line.Gross)...)
	return out, nil
}

// dlaInitialBase adalah IMProfit2 GenerateDLAList langkah jaminan: SumTSI jaminan bila CaseID
// polis memuat "ASM", selain itu SumOfTSI polis; Group Panel 006 memakai TSI jaminan.
func dlaInitialBase(sc dlaScope, policy registrasi.DLAPolicy) *big.Rat {
	tsi := big.NewRat(int64(sc.coverage.TSI), 100)
	if strings.TrimSpace(string(sc.claim.Policy.Line)) == "006" || strings.Contains(policy.CaseID, "ASM") {
		return tsi
	}
	if policy.SumOfTSI != nil {
		return policy.SumOfTSI
	}
	return tsi
}

// dlaTotalConvert adalah `local.totalConvert` GenerateDLAList (IDR): nilai adjustment terpilih
// × kurs, lalu DITAMBAH lagi setiap adjustment jaminan itu yang sudah diakseptasi — sehingga
// adjustment terpilih terhitung dua kali. Perilaku Pega itu dibawa apa adanya (`P-5`).
func dlaTotalConvert(sc dlaScope) *big.Rat {
	total := big.NewRat(int64(sc.line.Value.Convert(sc.line.Rate)), 100)
	for _, a := range sc.coverage.Settlement {
		if strings.TrimSpace(a.AcceptanceStatus) != "1" {
			continue
		}
		if a.AcceptedNo == "" && a.AcceptanceLODStatus != "" {
			continue
		}
		v := big.NewRat(int64(a.Value.Convert(a.Rate)), 100)
		if a.PaymentType == registrasi.PaymentSalvage {
			total.Sub(total, v)
		} else {
			total.Add(total, v)
		}
	}
	return total
}

// PrintDLA mencetak DLA satu adjustment — satu nomor, atau seluruhnya. DLA yang belum terbit
// diterbitkan lebih dulu; PRINT pertama menyimpan REMARKS dan menandai ISDLA.
func (l *Service) PrintDLA(ctx context.Context, p DLACommand, by Caller) (DLAResult, error) {
	sc, err := l.dlaScopeOf(ctx, p, by)
	if err != nil {
		return DLAResult{}, err
	}
	all, err := l.dlaIssuedOrNew(ctx, sc, by)
	if err != nil {
		return DLAResult{}, err
	}
	list := all.DLA
	if p.Number != "" {
		var one []registrasi.DLA
		for _, d := range list {
			if d.Number == p.Number {
				one = append(one, d)
			}
		}
		if len(one) == 0 {
			return DLAResult{}, fmt.Errorf("%w: DLA %s bukan milik adjustment ini", registrasi.ErrInvalidAction, p.Number)
		}
		list = one
	}
	if len(list) == 0 {
		return DLAResult{}, fmt.Errorf("%w: adjustment ini tidak punya DLA", registrasi.ErrInvalidAction)
	}
	for _, d := range list {
		if d.ZeroValue() {
			return DLAResult{}, registrasi.ErrDLAZeroShare()
		}
	}

	policy, err := l.policy.Get(ctx, sc.claim.Policy.Number)
	if err != nil {
		return DLAResult{}, err
	}
	cause := ""
	if sc.coverage.CauseOfLoss != "" {
		if cause, err = l.faceSheet.CauseOfLossName(ctx, sc.coverage.CauseOfLoss); err != nil {
			return DLAResult{}, err
		}
	}
	names, _, err := l.currencyNames(ctx)
	if err != nil {
		return DLAResult{}, err
	}
	entity := plaEntity(sc.claim.Portal)
	signerName := strings.TrimSpace(sc.line.Acceptance.Form.CommitteeName)
	var signature []byte
	if id := dlaSignatureID(signerName); id != "" {
		if _, signature, err = l.pla.Signature(ctx, id); err != nil {
			return DLAResult{}, err
		}
	}
	condition := sc.coverage.Name
	if p.AsPerPolicy {
		condition = "AS PER ORIGINAL POLICY"
	}
	treatyName := ""
	for _, s := range sc.coverage.Spreading {
		if !s.Removed && s.Name != "" {
			treatyName = s.Name
		}
	}

	now := l.clock.Now().UTC()
	files := make([]plaFile, 0, len(list))
	for _, d := range list {
		remarks := d.Note
		if !d.Printed {
			if r, ok := p.Remarks[d.Number]; ok {
				remarks = r
			}
			remarks = registrasi.DLAPrintNote(remarks, sc.line.PaymentType)
		}
		due := new(big.Rat)
		for _, x := range all.DLA {
			if x.Number == d.Number {
				due.Add(due, registrasi.DecimalOf(x.Value))
			}
		}
		content, err := l.dlaRenderer.Render(registrasi.DLADocument{
			DLA: d, Entity: entity,
			BusinessName:    firstText(sc.claim.Policy.BusinessName, policy.BusinessName),
			PolicyNumber:    sc.claim.Policy.Number,
			ClaimNumber:     sc.claim.Number,
			Insured:         firstText(sc.claim.Policy.QQName, sc.claim.Policy.InsuredName, policy.InsuredName),
			SumInsured:      registrasi.FaceSheetAmount{Currency: firstText(d.PolicyCurrency, names[sc.claim.Policy.Currency]), Value: sc.coverage.TSI},
			LossLocation:    sc.claim.Location,
			PolicyCondition: condition,
			PeriodStart:     policy.CoverageStart,
			PeriodEnd:       policy.CoverageEnd,
			DateOfLoss:      sc.claim.DateOfLoss,
			NatureOfLoss:    cause,
			PaymentType:     sc.line.PaymentType,
			Gross:           sc.line.Gross,
			ShareUnder:      registrasi.DLAShareUnder(d.Number, d.Type, d.QSPQS, treatyName),
			DueTotal:        due.FloatString(3),
			Remarks:         remarks,
			SignerName:      signerName,
			Signature:       signature,
		})
		if err != nil {
			return DLAResult{}, fmt.Errorf("registrasi/usecase: membentuk DLA %s: %w", d.Number, err)
		}
		files = append(files, plaFile{name: "DLA" + strings.ReplaceAll(d.Type, " ", "") + d.Number + ".pdf", content: content})
	}

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		printed := 0
		for _, d := range list {
			if d.Printed {
				continue
			}
			remarks := d.Note
			if r, ok := p.Remarks[d.Number]; ok {
				remarks = r
			}
			if err := l.dla.MarkPrinted(ctx, sc.claim.ID, d.Number, registrasi.DLAPrintNote(remarks, sc.line.PaymentType)); err != nil {
				return err
			}
			printed++
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: sc.claim.ID, ClaimNumber: sc.claim.Number, Event: "DLA_CETAK", Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Print DLA objek %d jaminan %d adjustment %d: %d dokumen, %d cetak pertama",
				p.Object, p.Coverage, p.Adjustment, len(list), printed),
		})
	})
	if err != nil {
		return DLAResult{}, err
	}
	return packDLA(files)
}

// dlaSignatureID memilih tanda tangan DLA dari nama komite penyetuju — SetSignatureNonMBU
// kategori DLA, urutan langkahnya dipertahankan (yang terakhir cocok menang).
//
// PENGECUALIAN `D-15` yang disadari, sama dengan plaSigner: aturannya tertanam nama orang
// di rule Pega, dan belum ada master penanda tangan per komite.
func dlaSignatureID(committee string) string {
	name := strings.ToUpper(committee)
	id := ""
	if strings.Contains(name, "BAMBANG") || strings.Contains(name, "LINDA") {
		id = "BAMBANGSG"
	}
	if strings.Contains(name, "DHARMANTO") {
		id = "DHARMANTO"
	}
	if strings.Contains(name, "ELLEN") {
		id = "ELLENSP"
	}
	if strings.Contains(name, "LINDA") {
		id = "LINDANOVA"
	}
	if strings.Contains(name, "INDRA") {
		id = "INDRAGN"
	}
	return id
}

// packDLA mengembalikan satu PDF, atau ZIP bila dokumennya lebih dari satu.
func packDLA(files []plaFile) (DLAResult, error) {
	if len(files) == 1 {
		return DLAResult{FileName: files[0].name, ContentType: "application/pdf", Content: files[0].content}, nil
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			return DLAResult{}, err
		}
		if _, err := w.Write(f.content); err != nil {
			return DLAResult{}, err
		}
	}
	if err := zw.Close(); err != nil {
		return DLAResult{}, err
	}
	return DLAResult{FileName: "DLA.zip", ContentType: "application/zip", Content: buf.Bytes()}, nil
}

