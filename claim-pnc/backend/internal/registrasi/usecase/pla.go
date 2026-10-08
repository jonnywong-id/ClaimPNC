package usecase

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// PLACommand adalah permintaan tombol "Print PLA" pada satu baris jaminan. Object dan
// Coverage berbasis 1.
type PLACommand struct {
	ClaimID  string
	TaskID   string
	Object   int
	Coverage int

	// Number memilih satu PLA untuk dicetak (tombol Print PLA per baris). Kosong berarti
	// seluruhnya (Print All PLA).
	Number string
}

// PLAList adalah isi layar PrintPLA_dtl: PLA revisi CFS terakhir jaminan itu.
type PLAList struct {
	Revision int
	PLA      []registrasi.PLA
	Issued   int // jumlah PLA yang baru terbit saat daftar dibuka
}

// plaScope adalah jaminan yang sedang dikerjakan beserta revisi CFS terakhirnya.
type plaScope struct {
	claim    registrasi.Claim
	object   registrasi.InsuredItem
	coverage registrasi.Coverage
	seq      int
	revision int
	names    map[string]string
	ids      map[string]string
}

// PLAResult adalah dokumen yang diunduh: satu PDF bila penerimanya satu, ZIP bila lebih.
type PLAResult struct {
	FileName    string
	ContentType string
	Content     []byte
	Issued      int // jumlah PLA yang baru terbit (0 bila hanya dicetak ulang)
}

// plaSigner memetakan entitas ke penanda tangan PLA.
//
// PENGECUALIAN `D-15` yang disadari: kunci tanda tangan diambil dari
// `SetSignatureNonMBU` (BranchID "BAMBANGSG" / "DHARMANTO" untuk kategori PLA) dan namanya
// tertulis mati di HTML `PLAHTML`, karena POOLDATA.MTTD.NAME kosong. Begitu DBA mengisi
// MTTD.NAME, nama dibaca dari sana dan nilai cadangan ini tidak lagi dipakai.
var plaSigner = map[string]struct{ ID, Name string }{
	"ASM": {ID: "BAMBANGSG", Name: "BAMBANG S. GUNAWAN"},
	"ASI": {ID: "DHARMANTO", Name: "DHARMANTO RAHARDJO"},
}

func plaEntity(portal string) string {
	if portal == "ASI" {
		return "ASI"
	}
	return "ASM"
}

// plaScopeOf memeriksa tugas, jaminan, dan isCFS, lalu mengembalikan cakupannya.
func (l *Service) plaScopeOf(ctx context.Context, p PLACommand, by Caller) (plaScope, error) {
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID, action: registrasi.ActionInputEstimate, alsoAction: registrasi.ActionInputSurveyor})
	if err != nil {
		return plaScope{}, err
	}
	if claim.ID != p.ClaimID {
		return plaScope{}, fmt.Errorf("%w: tugas %s bukan milik klaim %s", registrasi.ErrInvalidAction, p.TaskID, p.ClaimID)
	}
	if !l.canWork(task, by) {
		return plaScope{}, registrasi.ErrNotTaskOwner
	}
	if p.Object < 1 || p.Object > len(claim.InsuredItem) ||
		p.Coverage < 1 || p.Coverage > len(claim.InsuredItem[p.Object-1].Coverage) {
		return plaScope{}, fmt.Errorf("%w: objek %d jaminan %d tidak ada", registrasi.ErrInvalidAction, p.Object, p.Coverage)
	}
	object := claim.InsuredItem[p.Object-1]

	// isCFS: PLA dibentuk dari revisi Claim Face Sheet terakhir jaminan ini.
	revision, found, err := l.faceSheet.LastRevision(ctx, claim.ID, object.ID, p.Coverage)
	if err != nil {
		return plaScope{}, err
	}
	if !found {
		return plaScope{}, registrasi.ErrPLANeedsFaceSheet()
	}
	names, ids, err := l.currencyNames(ctx)
	if err != nil {
		return plaScope{}, err
	}
	return plaScope{claim: claim, object: object, coverage: object.Coverage[p.Coverage-1], seq: p.Coverage,
		revision: revision, names: names, ids: ids}, nil
}

// ListPLA membuka layar PrintPLA_dtl: menerbitkan PLA revisi CFS terakhir bila belum ada
// (pra-proses GeneratePLAListObject), lalu mengembalikan daftarnya.
func (l *Service) ListPLA(ctx context.Context, p PLACommand, by Caller) (PLAList, error) {
	sc, err := l.plaScopeOf(ctx, p, by)
	if err != nil {
		return PLAList{}, err
	}
	list, issued, err := l.issuedOrNew(ctx, sc, by)
	if err != nil {
		return PLAList{}, err
	}
	return PLAList{Revision: sc.revision, PLA: list, Issued: issued}, nil
}

func (l *Service) issuedOrNew(ctx context.Context, sc plaScope, by Caller) ([]registrasi.PLA, int, error) {
	list, err := l.pla.Issued(ctx, sc.claim.ID, sc.object.ID, sc.seq, sc.revision)
	if err != nil || len(list) > 0 {
		return list, 0, err
	}
	list, err = l.issuePLA(ctx, sc.claim, sc.object, sc.coverage, sc.seq, sc.revision, sc.names, sc.ids, by)
	return list, len(list), err
}

// SavePLANotes menyimpan isian REMARKS PLA yang sudah terbit, dikunci nomor PLA.
func (l *Service) SavePLANotes(ctx context.Context, p PLACommand, notes map[string]string, by Caller) (PLAList, error) {
	return l.SavePLADetails(ctx, p, notes, nil, by)
}

// PLAEmailMax adalah panjang maksimum isian Email PLA: T_PLALIST.EMAILPLA VARCHAR2(1000),
// terukur dari ALL_TAB_COLUMNS 2026-10-08.
const PLAEmailMax = 1000

// SavePLADetails menyimpan isian REMARKS dan Email PLA yang sudah terbit (layar PrintPLA_dtl:
// `.PLARemarks` dan `.pyEmailAddress`, keduanya Editable), dikunci nomor PLA. Email disimpan
// apa adanya setelah dipangkas — Pega tidak memeriksa formatnya, dan satu isian dapat memuat
// lebih dari satu alamat.
func (l *Service) SavePLADetails(ctx context.Context, p PLACommand, notes, emails map[string]string, by Caller) (PLAList, error) {
	for number, email := range emails {
		if utf8.RuneCountInString(strings.TrimSpace(email)) > PLAEmailMax {
			return PLAList{}, &registrasi.ValidationError{Violation: []registrasi.Violation{{
				Code: registrasi.ViolationPLAEmailTooLong, Field: "email",
				Message: "Email PLA " + number + " is limited to 1000 characters.",
			}}}
		}
	}
	sc, err := l.plaScopeOf(ctx, p, by)
	if err != nil {
		return PLAList{}, err
	}
	list, err := l.pla.Issued(ctx, sc.claim.ID, sc.object.ID, sc.seq, sc.revision)
	if err != nil {
		return PLAList{}, err
	}
	known := map[string]int{}
	for i, x := range list {
		known[x.Number] = i
	}
	now := l.clock.Now().UTC()
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		changed := 0
		for number, note := range notes {
			i, ok := known[number]
			if !ok {
				return fmt.Errorf("%w: PLA %s bukan milik jaminan ini", registrasi.ErrInvalidAction, number)
			}
			if list[i].Note == note {
				continue
			}
			if err := l.pla.UpdateNote(ctx, sc.claim.ID, number, sc.revision, note); err != nil {
				return err
			}
			list[i].Note = note
			changed++
		}
		for number, email := range emails {
			i, ok := known[number]
			if !ok {
				return fmt.Errorf("%w: PLA %s bukan milik jaminan ini", registrasi.ErrInvalidAction, number)
			}
			email = strings.TrimSpace(email)
			if list[i].Info.Email == email {
				continue
			}
			if err := l.pla.UpdateEmail(ctx, sc.claim.ID, number, sc.revision, email); err != nil {
				return err
			}
			list[i].Info.Email = email
			changed++
		}
		if changed == 0 {
			return nil
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: sc.claim.ID, ClaimNumber: sc.claim.Number, Event: "PLA_CATATAN", Actor: by.Identity, At: now,
			Note: "Remarks/Email " + strconv.Itoa(changed) + " PLA diubah",
		})
	})
	if err != nil {
		return PLAList{}, err
	}
	return PLAList{Revision: sc.revision, PLA: list}, nil
}

// PrintPLA mencetak PLA satu jaminan (koasuransi dan fac out) — satu nomor, atau seluruhnya. PLA yang
// belum terbit diterbitkan lebih dulu.
func (l *Service) PrintPLA(ctx context.Context, p PLACommand, by Caller) (PLAResult, error) {
	sc, err := l.plaScopeOf(ctx, p, by)
	if err != nil {
		return PLAResult{}, err
	}
	list, issued, err := l.issuedOrNew(ctx, sc, by)
	if err != nil {
		return PLAResult{}, err
	}
	if p.Number != "" {
		var one []registrasi.PLA
		for _, x := range list {
			if x.Number == p.Number {
				one = append(one, x)
			}
		}
		if len(one) == 0 {
			return PLAResult{}, fmt.Errorf("%w: PLA %s bukan milik jaminan ini", registrasi.ErrInvalidAction, p.Number)
		}
		list = one
	}
	// DownloadAllDocumentPLA: PLA tidak dicetak selama REMARKS-nya kosong.
	for _, x := range list {
		if strings.TrimSpace(x.Note) == "" {
			return PLAResult{}, registrasi.ErrPLARemarksEmpty()
		}
	}
	docs, err := l.plaDocuments(ctx, sc.claim, sc.object, sc.coverage, sc.names, list)
	if err != nil {
		return PLAResult{}, err
	}
	return packPLA(docs, issued)
}

// pendingPLA adalah satu PLA yang siap diterbitkan: penerima dan nilainya, tanpa nomor.
type pendingPLA struct {
	code, kind string
	name, id   string
	amount     []registrasi.PLAAmount
}

// issuePLA menerbitkan PLA satu jaminan dalam satu transaksi — `DownloadFireLossAdvice_act`:
// satu PLA per anggota koasuransi (COINS, huruf J) dan satu per reasuradur fakultatif keluar
// bila jaminan punya spreading FAC OUT (FACOUT, huruf H, langkah 23 dan 25).
//
// Koasuransi yang tidak berlaku (polis tanpa anggota, atau Sinar Mas bukan leader) tidak
// menggagalkan FAC OUT: keduanya penerima yang berbeda. Galat koasuransi baru dikembalikan
// bila tidak ada penerima FAC OUT sama sekali, supaya pesannya tetap menyebut alasannya.
func (l *Service) issuePLA(ctx context.Context, claim registrasi.Claim, object registrasi.InsuredItem,
	coverage registrasi.Coverage, coverageSeq, revision int, names, ids map[string]string, by Caller) ([]registrasi.PLA, error) {
	if claim.ExGratia {
		return nil, registrasi.ErrPLAExGratia()
	}
	members, err := l.pla.CoinsMembers(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return nil, err
	}

	// Reserve PLA: estimasi ber-CFS per mata uang (GeneratePLAList langkah 18–20).
	reserve := registrasi.PLAReserve(coverage, func(code string) string {
		if n := names[code]; n != "" {
			return n
		}
		return code
	})
	if len(reserve) == 0 {
		return nil, registrasi.ErrPLANoReserve()
	}

	var pending []pendingPLA

	// Koasuransi.
	coins, coinsErr := registrasi.CoinsPLARecipients(claim.Portal, members)
	if coinsErr == nil {
		asm := registrasi.PercentFull
		if c := claim.Policy.Coinsurance; c.HasShare {
			asm = c.ShareASM
		}
		for _, r := range coins {
			pending = append(pending, pendingPLA{
				code: registrasi.PLACodeCoins, kind: registrasi.PLATypeCoins, name: r.Name, id: r.ID,
				amount: registrasi.CoinsPLAAmounts(reserve, ids, r.Share, asm),
			})
		}
	}

	// Reasuransi: setiap baris spreading digolongkan lewat REINSURANCETYPE (GeneratePLAList
	// langkah 26) — BPPDAN/EQ POOL ke PLABPPDAN_Act, fakultatif ke PLAFacout_Act. Treaty
	// (PLATreaty_Act) belum dibangun.
	reinsurance, err := l.reinsurancePLA(ctx, claim, object, coverage, coverageSeq, members, reserve, ids)
	if err != nil {
		return nil, err
	}
	pending = append(pending, reinsurance.pending...)

	if len(pending) == 0 {
		if coinsErr != nil && !reinsurance.routed {
			return nil, coinsErr
		}
		return nil, registrasi.ErrPLANoRecipient()
	}

	infos := make([]registrasi.PLARecipientInfo, len(pending))
	for i, r := range pending {
		if infos[i], err = l.pla.Recipient(ctx, r.id, r.name); err != nil {
			return nil, err
		}
	}

	now := l.clock.Now().UTC()
	// Tahun pada nomor: tahun registrasi klaim (cocok 1.313 dari 1.406 PLA di T_PLALIST).
	year := clock.DateWIB(claim.CreatedAt).Year()
	if claim.CreatedAt.IsZero() {
		year = clock.DateWIB(now).Year()
	}
	policyCurrency := names[claim.Policy.Currency]
	if policyCurrency == "" {
		policyCurrency = claim.Policy.Currency
	}

	var out []registrasi.PLA
	counts := map[string]int{}
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		for i, r := range pending {
			number, err := l.pla.NextNumber(ctx, r.code, year)
			if err != nil {
				return err
			}
			previous, had, err := l.pla.Previous(ctx, claim.ID, r.id)
			if err != nil {
				return err
			}
			p := registrasi.PLA{
				ClaimID: claim.ID, ObjectID: object.ID, CoverageSeq: coverageSeq, Number: number,
				Type: r.kind, Recipient: r.name, RecipientCode: r.id, Revision: revision,
				Date: now, Note: registrasi.PLANote(previous, had), PolicyCurrency: policyCurrency,
				Amount: r.amount, Info: infos[i],
			}
			if err := l.pla.Save(ctx, p); err != nil {
				return err
			}
			out = append(out, p)
			counts[r.kind]++
		}
		reinsCount := len(out) - counts[registrasi.PLATypeCoins]
		// GeneratePLAList langkah 33: Status Klaim menjadi PLA Report.
		claim.ClaimStatus = registrasi.StatusClaimPLAReport
		claim.UpdatedBy = by.Identity
		claim.UpdatedAt = now
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "PLA_TERBIT", Actor: by.Identity, At: now,
			Note: "PLA objek " + object.ID + " jaminan " + strconv.Itoa(coverageSeq) +
				" revisi CFS " + strconv.Itoa(revision) + ": " + strconv.Itoa(counts[registrasi.PLATypeCoins]) +
				" koasuransi, " + strconv.Itoa(reinsCount) + " reasuransi",
		})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// reinsurancePLA adalah hasil penggolongan spreading jaminan untuk PLA reasuransi.
type reinsurancePLA struct {
	pending []pendingPLA
	routed  bool // ada baris spreading yang punya pengolah PLA reasuransi
}

// reinsurancePLA menyusun PLA reasuransi satu jaminan — `GeneratePLAList` langkah 15–19 dan
// 26: bagian ASM (termasuk FAC IN) dari dokumen polis, lalu setiap baris spreading ke
// pengolahnya. Fac Offer dibaca dari T_FACOFFER dengan cadangan kolom datar.
func (l *Service) reinsurancePLA(ctx context.Context, claim registrasi.Claim, object registrasi.InsuredItem,
	coverage registrasi.Coverage, coverageSeq int, members []registrasi.PLACoinsMember,
	reserve []registrasi.FaceSheetAmount, ids map[string]string) (reinsurancePLA, error) {
	var out reinsurancePLA
	var live []registrasi.Spreading
	for _, s := range coverage.Spreading {
		if !s.Removed {
			live = append(live, s)
		}
	}
	if len(live) == 0 {
		return out, nil
	}
	cases, err := l.dla.ReinsuranceCase(ctx)
	if err != nil {
		return out, err
	}
	policy, err := l.dla.Policy(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return out, err
	}
	policy.Coins = members
	pct := registrasi.FacOutPercentASM(claim.Portal, policy)
	_, leader := registrasi.ASMCoinsShare(claim.Portal, members)

	var offers []registrasi.FacOffer
	var flatFacOut, flatTotal *big.Rat
	offersRead := false
	for _, s := range live {
		var recipients []registrasi.PLARecipient
		switch registrasi.RouteSpreading(s.TreatyKind, cases[s.TreatyKind]) {
		case registrasi.DLARouteBPPDAN:
			if r, ok := registrasi.BPPDANPLARecipient(s.TreatyKind, s.Share); ok {
				recipients = append(recipients, r)
			}
		case registrasi.DLARouteFacOut:
			if !offersRead {
				if offers, err = l.pla.FacOffers(ctx, claim.Policy.Number, claim.Policy.ProdKe); err != nil {
					return out, err
				}
				offersRead = true
				if hasFlatOffer(offers) {
					flatFacOut, flatTotal, err = l.pla.SpreadingTSI(ctx, registrasi.SpreadingTSIQuery{
						PolicyNumber: claim.Policy.Number, ProdKe: claim.Policy.ProdKe,
						GroupPanel: string(claim.Policy.Line), ObjectID: object.ID, Coverage: coverage.ID,
					})
					if err != nil {
						return out, err
					}
				}
			}
			recipients = registrasi.FacOutPLARecipients(registrasi.PLAReinsuranceInput{
				TreatyType: s.TreatyKind, GroupPanel: string(claim.Policy.Line),
				ObjectID: object.ID, ObjectName: object.Name, Coverage: coverage.ID, CoverageSeq: coverageSeq,
				PercentASM: pct, Policy: policy, Offers: offers,
				FlatFacOut: flatFacOut, FlatTotal: flatTotal,
			})
		default:
			continue
		}
		out.routed = true
		for _, r := range recipients {
			out.pending = append(out.pending, pendingPLA{
				code: r.Code, kind: r.Type, name: r.Name, id: r.ID,
				amount: registrasi.PLAReinsuranceAmounts(reserve, ids, r, pct, leader),
			})
		}
	}
	return out, nil
}

// hasFlatOffer menyatakan ada Fac Offer tanpa JSONDATA — hanya saat itu T_SPREADINGLIST
// dibaca.
func hasFlatOffer(offers []registrasi.FacOffer) bool {
	for _, o := range offers {
		if strings.TrimSpace(o.FlatShare) != "" {
			return true
		}
	}
	return false
}

type plaFile struct {
	name    string
	content []byte
}

// plaDocuments membentuk dokumen setiap PLA.
func (l *Service) plaDocuments(ctx context.Context, claim registrasi.Claim, object registrasi.InsuredItem,
	coverage registrasi.Coverage, names map[string]string, list []registrasi.PLA) ([]plaFile, error) {
	policy, err := l.policy.Get(ctx, claim.Policy.Number)
	if err != nil {
		return nil, err
	}
	cause := ""
	if coverage.CauseOfLoss != "" {
		if cause, err = l.faceSheet.CauseOfLossName(ctx, coverage.CauseOfLoss); err != nil {
			return nil, err
		}
	}
	entity := plaEntity(claim.Portal)
	signer := plaSigner[entity]
	signerName, signature, err := l.pla.Signature(ctx, signer.ID)
	if err != nil {
		return nil, err
	}
	if signerName == "" {
		signerName = signer.Name
	}
	currency := names[claim.Policy.Currency]
	if currency == "" {
		currency = claim.Policy.Currency
	}
	// Total Sum Insured dokumen PLA adalah SumOfTSI POLIS (DownloadPLA langkah 13:
	// tempObjeCvg.AgingAmount := pyWorkPage.Policy.SumOfTSI), bukan TSI jaminan; TSI jaminan
	// hanya cadangan bila dokumen polis tidak memuatnya.
	sumInsured := coverage.TSI
	if doc, err := l.dla.Policy(ctx, claim.Policy.Number, claim.Policy.ProdKe); err != nil {
		return nil, err
	} else if doc.SumOfTSI != nil && doc.SumOfTSI.Sign() != 0 {
		sumInsured = registrasi.RupiahOf(doc.SumOfTSI)
	}

	files := make([]plaFile, 0, len(list))
	for _, p := range list {
		content, err := l.plaRenderer.Render(registrasi.PLADocument{
			PLA: p, Entity: entity,
			BusinessName:    firstText(claim.Policy.BusinessName, policy.BusinessName),
			PolicyNumber:    claim.Policy.Number,
			ClaimNumber:     claim.Number,
			Insured:         firstText(claim.Policy.InsuredName, policy.InsuredName),
			Interest:        object.Name,
			SumInsured:      registrasi.FaceSheetAmount{Currency: currency, Value: sumInsured},
			ShareLabel:      registrasi.PLAShareLabel(p),
			PeriodStart:     policy.CoverageStart,
			PeriodEnd:       policy.CoverageEnd,
			PolicyCondition: firstText(coverage.Name, coverage.ID),
			DateOfLoss:      claim.DateOfLoss,
			NatureOfLoss:    cause,
			LossLocation:    claim.Location,
			SignerName:      signerName,
			Signature:       signature,
		})
		if err != nil {
			return nil, fmt.Errorf("registrasi/usecase: membentuk PLA %s: %w", p.Number, err)
		}
		// Nama berkas Pega (DownloadPLA langkah 26–32): FAC OUT "PLAFACOFFER", selainnya
		// "PLA" + tipe (PLACOINS, PLABPPDAN, PLAEQPOOL).
		name := "PLA" + p.Type + p.Number + ".pdf"
		if p.Type == registrasi.PLATypeFacOut {
			name = "PLAFACOFFER" + p.Number + ".pdf"
		}
		files = append(files, plaFile{name: name, content: content})
	}
	return files, nil
}

// packPLA mengembalikan satu PDF, atau ZIP bila penerimanya lebih dari satu.
func packPLA(files []plaFile, issued int) (PLAResult, error) {
	if len(files) == 1 {
		return PLAResult{FileName: files[0].name, ContentType: "application/pdf", Content: files[0].content, Issued: issued}, nil
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			return PLAResult{}, err
		}
		if _, err := w.Write(f.content); err != nil {
			return PLAResult{}, err
		}
	}
	if err := zw.Close(); err != nil {
		return PLAResult{}, err
	}
	return PLAResult{FileName: "PLA.zip", ContentType: "application/zip", Content: buf.Bytes(), Issued: issued}, nil
}

// currencyNames mengembalikan peta kode → nama dan nama → kode mata uang.
func (l *Service) currencyNames(ctx context.Context) (map[string]string, map[string]string, error) {
	list, err := l.currency.Currencies(ctx)
	if err != nil {
		return nil, nil, err
	}
	names, ids := map[string]string{}, map[string]string{}
	for _, c := range list {
		names[c.ID] = c.Name
		ids[c.Name] = c.ID
	}
	return names, ids, nil
}

func firstText(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
