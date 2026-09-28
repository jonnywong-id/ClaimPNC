package usecase

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strconv"

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
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID, action: registrasi.ActionInputEstimate})
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
	list, err = l.issueCoinsPLA(ctx, sc.claim, sc.object, sc.coverage, sc.seq, sc.revision, sc.names, sc.ids, by)
	return list, len(list), err
}

// SavePLANotes menyimpan isian REMARKS PLA yang sudah terbit, dikunci nomor PLA.
func (l *Service) SavePLANotes(ctx context.Context, p PLACommand, notes map[string]string, by Caller) (PLAList, error) {
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
		if changed == 0 {
			return nil
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: sc.claim.ID, ClaimNumber: sc.claim.Number, Event: "PLA_CATATAN", Actor: by.Identity, At: now,
			Note: "Catatan " + strconv.Itoa(changed) + " PLA diubah",
		})
	})
	if err != nil {
		return PLAList{}, err
	}
	return PLAList{Revision: sc.revision, PLA: list}, nil
}

// PrintPLA mencetak PLA koasuransi satu jaminan — satu nomor, atau seluruhnya. PLA yang
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
	docs, err := l.plaDocuments(ctx, sc.claim, sc.object, sc.coverage, sc.names, list)
	if err != nil {
		return PLAResult{}, err
	}
	return packPLA(docs, issued)
}

// issueCoinsPLA menerbitkan satu PLA per anggota koasuransi dalam satu transaksi.
func (l *Service) issueCoinsPLA(ctx context.Context, claim registrasi.Claim, object registrasi.InsuredItem,
	coverage registrasi.Coverage, coverageSeq, revision int, names, ids map[string]string, by Caller) ([]registrasi.PLA, error) {
	members, err := l.pla.CoinsMembers(ctx, claim.Policy.Number)
	if err != nil {
		return nil, err
	}
	recipients, err := registrasi.CoinsPLARecipients(claim.Portal, members)
	if err != nil {
		return nil, err
	}

	// Reserve PLA adalah reserve Claim Face Sheet: hanya estimasi yang sudah dikunci.
	locked := coverage
	locked.Item = nil
	for _, it := range coverage.Item {
		x := it
		x.Estimation = nil
		for _, e := range it.Estimation {
			if e.FaceSheet {
				x.Estimation = append(x.Estimation, e)
			}
		}
		locked.Item = append(locked.Item, x)
	}
	reserve := registrasi.FaceSheetReserve(locked, func(code string) string {
		if n := names[code]; n != "" {
			return n
		}
		return code
	})
	if len(reserve) == 0 {
		return nil, registrasi.ErrPLANoReserve()
	}
	asm := registrasi.PercentFull
	if c := claim.Policy.Coinsurance; c.HasShare {
		asm = c.ShareASM
	}

	infos := make([]registrasi.PLARecipientInfo, len(recipients))
	for i, r := range recipients {
		if infos[i], err = l.pla.Recipient(ctx, r.ID, r.Name); err != nil {
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
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		for i, r := range recipients {
			number, err := l.pla.NextNumber(ctx, registrasi.PLACodeCoins, year)
			if err != nil {
				return err
			}
			previous, had, err := l.pla.Previous(ctx, claim.ID, r.ID)
			if err != nil {
				return err
			}
			p := registrasi.PLA{
				ClaimID: claim.ID, ObjectID: object.ID, CoverageSeq: coverageSeq, Number: number,
				Type: registrasi.PLATypeCoins, Recipient: r.Name, RecipientCode: r.ID, Revision: revision,
				Date: now, Note: registrasi.PLANote(previous, had), PolicyCurrency: policyCurrency,
				Amount: registrasi.CoinsPLAAmounts(reserve, ids, r.Share, asm), Info: infos[i],
			}
			if err := l.pla.Save(ctx, p); err != nil {
				return err
			}
			out = append(out, p)
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "PLA_TERBIT", Actor: by.Identity, At: now,
			Note: "PLA koasuransi objek " + object.ID + " jaminan " + strconv.Itoa(coverageSeq) +
				" revisi CFS " + strconv.Itoa(revision) + ": " + strconv.Itoa(len(recipients)) + " penerima",
		})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
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

	files := make([]plaFile, 0, len(list))
	for _, p := range list {
		content, err := l.plaRenderer.Render(registrasi.PLADocument{
			PLA: p, Entity: entity,
			BusinessName:    firstText(claim.Policy.BusinessName, policy.BusinessName),
			PolicyNumber:    claim.Policy.Number,
			ClaimNumber:     claim.Number,
			Insured:         firstText(claim.Policy.InsuredName, policy.InsuredName),
			Interest:        object.Name,
			SumInsured:      registrasi.FaceSheetAmount{Currency: currency, Value: coverage.TSI},
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
		files = append(files, plaFile{name: "PLA" + p.Type + p.Number + ".pdf", content: content})
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
