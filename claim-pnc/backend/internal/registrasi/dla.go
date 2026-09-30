package registrasi

import (
	"context"
	"math/big"
	"strings"
	"time"
)

// # Definite Loss Advice (DLA)
//
// Tombol "Print DLA" pada baris adjustment (`Section/PNCTombolDLA_sect.xml`) membuka flow
// action `PrintDLA`, yang pra-prosesnya `GenerateDLAListAdjustment` → `GenerateDLAList`
// dan layarnya `PrintDLA`. Saat dialog dibuka, DLA diterbitkan satu per penerima bila
// adjustment itu belum punya DLA; PRINT per baris (`DownloadDLA`) membentuk dokumennya.
//
// Empat pengolah penerima, dibaca dari rule masing-masing:
//
//   - `DLACoins_act`   — koasuransi, huruf J, TIPEDLA "COINS";
//   - `DLAFacout_act`  — fakultatif keluar, huruf H (M untuk FACOBSRB), TIPEDLA "FAC OUT";
//   - `DLABPPDAN_act`  — BPPDAN dan EQ POOL, huruf H, TIPEDLA "BPPDAN"/"EQPOOL";
//   - `DLATreaty_Act`  — treaty, huruf L (M untuk FAC-OBLIG/FACOBLIGINDT).
//
// Nilai tersimpan sebagai TEKS desimal seperti kolom VARCHAR2 T_DLALIST, dengan jumlah
// desimal yang sama dengan `@divide(x,1,n)` masing-masing rule. Hitungannya memakai
// bilangan rasional supaya pembulatan antara (`@divide(...,5)`) tidak bergantung pada
// pecahan biner.

// Tipe DLA — TIPEDLA.
const (
	DLATypeCoins        = "COINS"
	DLATypeFacOut       = "FAC OUT"
	DLATypeBPPDAN       = "BPPDAN"
	DLATypeEQPool       = "EQPOOL"
	DLATypeTreaty       = "TREATY"
	DLATypeFacOblig     = "FAC-OBLIG"
	DLATypeFacObligIndt = "FACOBLIGINDT"
)

// Huruf nomor DLA — `PLA_DLA.prc` KODE.
const (
	DLACodeCoins     = "J"
	DLACodeFacOut    = "H"
	DLACodeFacOutSRB = "M"
	DLACodeTreaty    = "L"
	DLACodeTreatyFac = "M"
)

// Kode treaty yang dikenali pengolah DLA.
const (
	treatyFacOblig     = "10002"
	treatyBPPDAN       = "10010"
	treatyEQPool       = "10023"
	treatyFacObligIndt = "10024"
	treatyPFRA         = "10029"
	treatyFacOBSRB     = "10061"
	treatyQS           = "10003"
	treatyPQS          = "10019"
)

// CaseID jenis reasuransi (REINSURANCETYPE.TYPE): 2 treaty, 3 fakultatif.
const (
	reinsCaseTreaty = "2"
	reinsCaseFac    = "3"
)

// DLA adalah satu baris POOLDATA.T_DLALIST.
type DLA struct {
	ClaimID       string
	ObjectID      string
	CoverageSeq   int
	AdjustmentSeq int

	Number        string
	Type          string
	Recipient     string
	RecipientCode string
	Date          time.Time
	Note          string

	// Nilai sebagai teks desimal — NILAIDLA, PERCENTDLA, SHARESPREADING, KLAIMAMOUNT.
	Value       string
	Percent     string
	ShareSpread string
	ClaimAmount string

	Currency       string // nama mata uang adjustment
	PolicyCurrency string // nama mata uang polis
	QSPQS          string
	QSRI           string

	AcceptedNo   string
	AcceptedDate time.Time

	Info PLARecipientInfo

	Printed bool // ISDLA = '1'
	Sent    bool // ISKIRIM = '1'
}

// ZeroValue menyatakan NILAIDLA nol — `DownloadDLA` menolak mencetak bila ada.
func (d DLA) ZeroValue() bool {
	v := decimalOf(d.Value)
	return v.Sign() == 0
}

// DLAShare adalah hasil hitungan satu penerima sebelum bernomor.
type DLAShare struct {
	Code          string // huruf nomor
	Type          string
	Recipient     string
	RecipientCode string
	Value         string
	Percent       string
	ShareSpread   string
	ClaimAmount   string
	QSPQS         string
	QSRI          string
}

// DLAPrevious adalah dokumen terakhir kepada seorang penerima pada klaim yang sama.
type DLAPrevious struct {
	Number string
	Date   time.Time
}

// DLANote memilih catatan DLA baru — `INSERT_PLADLA.prc` cabang DLA:
//
//   - belum ada DLA kepada penerima itu: rujuk PLA terakhirnya, bila ada;
//   - sudah ada: rujuk DLA terakhirnya. Tanggalnya dari DATE tanpa format (NLS bawaan
//     `DD-MON-RR`), seperti baris produksi "… with DD: 04-DEC-23".
func DLANote(pla PLAPrevious, hasPLA bool, dla DLAPrevious, hasDLA bool) string {
	if hasDLA {
		if dla.Number == "" {
			return ""
		}
		return "- Please see our DLA No.:" + dla.Number + " with DD: " + strings.ToUpper(dla.Date.Format("02-Jan-06"))
	}
	if hasPLA && pla.Number != "" {
		return PLAPreviousNote(pla.Number, pla.Date)
	}
	return ""
}

// DLAPrintNote menyusun REMARKS yang tercetak dan disimpan pada PRINT pertama — `DownloadDLA`
// langkah 31: remark + "\n- " + tipe pembayaran. Baris "Status Claim" (dropdown Status Klaim
// DLA) belum dibawa.
func DLAPrintNote(remark, paymentType string) string {
	return remark + "\n- " + PaymentTypeName(paymentType)
}

// DLASource adalah seam ke data DLA.
type DLASource interface {
	// Issued membaca DLA revisi 0 satu adjustment (`GetDataDLA`), terurut tanggal.
	Issued(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int) ([]DLA, error)
	// PreDLA membaca Pre DLA adjustment itu (`GetDataPreDLA`).
	PreDLA(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int) ([]DLA, error)
	// Previous membaca DLA terakhir kepada seorang penerima pada klaim yang sama.
	Previous(ctx context.Context, claimID, recipientCode string) (DLAPrevious, bool, error)

	// Policy membaca bahan DLA dari dokumen polis.
	Policy(ctx context.Context, policyNumber string) (DLAPolicy, error)
	// ReinsuranceCase membaca REINSURANCETYPE.TYPE per kode treaty.
	ReinsuranceCase(ctx context.Context) (map[string]string, error)
	// Treaty membaca pengaturan treaty (`GetTreatyGroupID`, `SelectTreatyReinsurer`,
	// `SelectProportionalArrg`, `searchQSReins*`).
	Treaty(ctx context.Context, businessCode string, year int, treatyType string) (TreatyArrangement, error)

	// NextNumber menerbitkan nomor DLA (DLA_SEQ + baris POOLDATA.DLA). Di dalam UnitOfWork.
	NextNumber(ctx context.Context, code string, year int) (string, error)
	// Save menulis satu baris T_DLALIST. Di dalam UnitOfWork.
	Save(ctx context.Context, d DLA) error
	// MarkPrinted mengisi NOTES dan ISDLA = '1' — PRINT pertama.
	MarkPrinted(ctx context.Context, claimID, number, note string) error
}

// DLARenderer mengubah DLA menjadi dokumen yang diunduh.
type DLARenderer interface {
	Render(d DLADocument) ([]byte, error)
}

// DLADocument adalah isi satu dokumen DLA.
type DLADocument struct {
	DLA
	Entity          string // ASM atau ASI
	BusinessName    string
	PolicyNumber    string
	ClaimNumber     string
	Insured         string
	SumInsured      FaceSheetAmount
	LossLocation    string
	PolicyCondition string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	DateOfLoss      time.Time
	NatureOfLoss    string
	PaymentType     string
	Gross           Money  // GrossValue adjustment — baris "Total …"
	ShareUnder      string // CedingCoName — label "Your Share Under"
	DueTotal        string // Σ NILAIDLA dengan nomor yang sama
	Remarks         string
	SignerName      string
	Signature       []byte
}

// DLAPolicy adalah bahan DLA dari dokumen polis.
type DLAPolicy struct {
	Coins           []PLACoinsMember
	CaseID          string
	SumOfTSI        *big.Rat
	TypeOfCoins     string
	OfferFacInShare *big.Rat // OfferFacIn.PercentShare (persen)
	Syariah         bool
	BusinessCode    string
	StartYear       int
	FacOffer        []FacOffer
	// TSISpreaded per kode treaty dari SpreadingList polis — bahan FACOBSRB.
	TSISpreaded map[string]*big.Rat
}

// FacOffer adalah satu baris FacOfferList polis.
type FacOffer struct {
	ReinsurerName  string
	ReinsurerID    string
	Deleted        bool
	OfferedMissing bool // TotalOffered "" atau "0"
	Property       []FacObject
	Aneka          []FacObject
	Cargo          []FacObject
}

// FacObject adalah satu baris PropertyList / AnekaList / CargoList sebuah FacOffer.
type FacObject struct {
	ObjectNo    string // PropertyList.ObjectNo
	ObjectName  string // AnekaList.ObjectName
	SumTSIAneka string // AnekaList.SumTSIObjectAneka
	GoodID      string // CargoList.GoodID
	GoodNote    string // CargoList.GoodNote
	IndexObject string // CargoList.IndexObject
	Coverage    []FacCoverage
}

// FacCoverage adalah satu baris CoverageList beserta FacOutObjectList(1)-nya.
type FacCoverage struct {
	Code         string
	TSI          string
	SumTSI       string
	TSISublimit  string
	ShareOffered string
	Percent      string
}

// TreatyReinsurer adalah satu baris `SelectTreatyReinsurer`.
type TreatyReinsurer struct {
	Name     string
	ID       string
	PctShare string // TSI (pctshare, persen)
	TypeName string // REINSTYPENAME
}

// TreatyArrangement adalah pengaturan treaty satu kode pada tahun polis.
type TreatyArrangement struct {
	Reinsurers []TreatyReinsurer
	Limit      string // proportionalarrg RP (treatydescid 10003); kosong bila tidak ada
	QSPct      string // searchQSReins2 PCT baris terakhir (pecahan); kosong bila tidak ada
}

// ---- gerbang ---------------------------------------------------------------------------

// OwnFacOutASO adalah CoinsID yang menandai polis ASO (`InputRegister_act`: FlagASO := 1).
const coinsASO = "10038996"

// FlagASO menyatakan polis ASO — ada anggota CoinsList berkode 10038996.
func FlagASO(members []PLACoinsMember) bool {
	for _, m := range members {
		if strings.TrimSpace(m.ID) == coinsASO {
			return true
		}
	}
	return false
}

// DLASkipsExGratia adalah gerbang `GenerateDLAList` langkah 14: klaim ex gratia non-ASO
// tidak mendapat DLA.
func DLASkipsExGratia(exGratia, aso bool) bool { return exGratia && !aso }

// DLAAmountBase adalah `adjList`: nilai adjustment terpilih menurut tipe pembayarannya —
// AdjusterFeeValue (4/7), SalvageValue (3), AdjustmentValue (1/2/5). Pada model ini
// ketiganya tersimpan di ASM_SHARE_VALUE (`SettlementLine.Value`).
func DLAAmountBase(line SettlementLine) Money { return line.Value }

// ---- COINS -----------------------------------------------------------------------------

// coinsExcluded adalah CoinsID yang tidak pernah mendapat DLA koasuransi (`DLACoins_act`).
const coinsExcluded = "10000000262"

// CoinsDLAShares menghitung DLA koasuransi — `DLACoins_act`. Hanya bila perusahaan sendiri
// leader; tiap anggota lain yang share-nya > 0 dan tidak bertanda hapus mendapat satu DLA:
// nilai = round2(GrossValue × share / 100), klaim = GrossValue.
func CoinsDLAShares(portal string, members []PLACoinsMember, gross Money) []DLAShare {
	own := OwnCompanyOf(portal)
	leader := false
	for _, m := range members {
		if m.ID != "" && m.Leader && strings.Contains(strings.ToUpper(m.Name), own) {
			leader = true
		}
	}
	if !leader {
		return nil
	}
	base := moneyRat(gross)
	var out []DLAShare
	for _, m := range members {
		id := strings.TrimSpace(m.ID)
		if id == "" || m.Deleted || !m.HasShare || m.Share <= 0 || id == coinsExcluded {
			continue
		}
		if strings.Contains(strings.ToUpper(m.Name), own) {
			continue
		}
		share := percentRat(m.Share)
		value := mul(base, div(share, big.NewRat(100, 1)))
		out = append(out, DLAShare{
			Code: DLACodeCoins, Type: DLATypeCoins, Recipient: m.Name, RecipientCode: id,
			Value: decText(value, 2), Percent: decText(share, 2), ClaimAmount: decText(base, 2),
		})
	}
	return out
}

// ---- BPPDAN / EQ POOL ------------------------------------------------------------------

// BPPDANDLAShare menghitung DLA BPPDAN (10010) atau EQ POOL (10023) — `DLABPPDAN_act`.
// Penerimanya tetap per kode; nilai = round5(share × round5(adjList / 100)). Ambang
// 20 miliar dan share ASM dikomentari di rule, sehingga tidak dibawa.
func BPPDANDLAShare(treatyType string, share Percent, amount Money) (DLAShare, bool) {
	var d DLAShare
	switch strings.TrimSpace(treatyType) {
	case treatyBPPDAN:
		d = DLAShare{Recipient: "BPPDAN", RecipientCode: "10038311", Type: DLATypeBPPDAN}
	case treatyEQPool:
		d = DLAShare{Recipient: "REASURANSI MAIPARK INDONESIA", RecipientCode: "10038726", Type: DLATypeEQPool}
	default:
		return DLAShare{}, false
	}
	adj := moneyRat(amount)
	pct := percentRat(share)
	value := round(mul(pct, round(div(adj, big.NewRat(100, 1)), 5)), 5)
	d.Code = DLACodeFacOut
	d.Value = decText(value, 5)
	d.Percent = decText(pct, 5)
	d.ShareSpread = decText(pct, 5)
	d.ClaimAmount = decText(adj, 5)
	return d, true
}

// ---- TREATY ----------------------------------------------------------------------------

// TreatyDLAShares menghitung DLA treaty satu baris spreading — `DLATreaty_Act`.
//
// totalConvert adalah `local.totalConvert` GenerateDLAList (nilai IDR); di sini dikalikan
// share spreading. Penerima treaty (bukan FAC-OBLIG/FACOBLIGINDT) hanya mendapat DLA bila
// nilai itu MELAMPAUI limit RP.
func TreatyDLAShares(treatyType string, share Percent, amount Money, totalConvert *big.Rat, arr TreatyArrangement) []DLAShare {
	if len(arr.Reinsurers) == 0 {
		return nil
	}
	facType := treatyType == treatyFacOblig || treatyType == treatyFacObligIndt
	if !facType && strings.TrimSpace(arr.Limit) == "" {
		return nil
	}
	pctTreaty := percentRat(share)
	converted := mul(totalConvert, div(pctTreaty, big.NewRat(100, 1)))
	adj := moneyRat(amount)
	qs := decimalOf(arr.QSPct)
	hasQS := strings.TrimSpace(arr.QSPct) != "" && qs.Sign() != 0
	limit := decimalOf(arr.Limit)

	var out []DLAShare
	for _, r := range arr.Reinsurers {
		typeName := strings.TrimSpace(r.TypeName)
		isFac := typeName == DLATypeFacOblig || typeName == DLATypeFacObligIndt
		if !isFac && converted.Cmp(limit) <= 0 {
			continue
		}
		pct := decimalOf(r.PctShare)
		var sum *big.Rat
		asmShare := new(big.Rat)
		switch {
		case isFac:
			sum = mul(div(adj, big.NewRat(100, 1)), pctTreaty)
		case hasQS:
			sum = mul(mul(mul(round(div(pct, big.NewRat(100, 1)), 5), qs), adj), round(div(pctTreaty, big.NewRat(100, 1)), 5))
			asmShare = mul(qs, big.NewRat(100, 1))
		default:
			sum = mul(mul(round(div(pct, big.NewRat(100, 1)), 5), adj), round(div(pctTreaty, big.NewRat(100, 1)), 5))
		}
		code, kind := DLACodeTreaty, DLATypeTreaty
		if isFac {
			code = DLACodeTreatyFac
			kind = typeName
		}
		out = append(out, DLAShare{
			Code: code, Type: kind, Recipient: r.Name, RecipientCode: strings.TrimSpace(r.ID),
			Value: decText(sum, 2), ShareSpread: decText(pct, 3), Percent: decText(pctTreaty, 3),
			ClaimAmount: decText(adj, 3), QSPQS: typeName, QSRI: plainDecimal(asmShare),
		})
	}
	return out
}

// ---- FAC OUT ---------------------------------------------------------------------------

// FacOutInput adalah bahan `DLAFacout_act` satu baris spreading.
type FacOutInput struct {
	Portal      string
	TreatyType  string
	GroupPanel  string
	Policy      DLAPolicy
	ObjectID    string // ObjectList(objID).ObjectID — objectLocation
	ObjectName  string
	ContractNo  string // tidak tersimpan pada model klaim ini — kosong
	Coverage    string // CoverageOldID — kode jaminan polis
	CoverageSeq int    // param.cvgID, berbasis 1
	Amount      Money  // adjList

	// InitialBase adalah IMProfit2 yang sudah ditetapkan GenerateDLAList sebelum
	// pengolah dipanggil (langkah jaminan: SumTSI jaminan atau SumOfTSI polis).
	InitialBase *big.Rat
}

type facSlot struct {
	name, id, jurisdiction string
	share                  *big.Rat
}

// FacOutPercentASM adalah `local.percentASM` DLAFacout_act langkah 2–3.
func FacOutPercentASM(portal string, p DLAPolicy) *big.Rat {
	pct := big.NewRat(1, 1)
	if len(p.Coins) > 0 {
		for _, m := range p.Coins {
			if strings.TrimSpace(m.ID) == "" {
				continue
			}
			name := strings.ToUpper(m.Name)
			match := false
			switch {
			case strings.EqualFold(portal, "ASI"):
				match = strings.Contains(name, "ASURANSI SIMAS INSURTECH")
			case p.Syariah:
				match = strings.Contains(name, "ASURANSI SINAR MAS") && strings.Contains(name, "SYARIAH")
			default:
				match = strings.Contains(name, "ASURANSI SINAR MAS") && !strings.Contains(name, "SYARIAH")
			}
			if match && m.HasShare {
				pct = round(div(percentRat(m.Share), big.NewRat(100, 1)), 4)
			}
		}
	}
	if strings.TrimSpace(p.TypeOfCoins) == "F" && p.OfferFacInShare != nil {
		pct = mul(pct, round(div(p.OfferFacInShare, big.NewRat(100, 1)), 10))
	}
	return pct
}

// FacOutDLAShares menghitung DLA fakultatif keluar — `DLAFacout_act`.
//
// Bagian penerima adalah JUMLAH yang ditawarkan (ShareOffered), dan dasar pembaginya satu
// nilai TSI (IMProfit2) untuk seluruh penerima: nilai = round4(ShareOffered × adjList /
// IMProfit2), disimpan round3.
func FacOutDLAShares(in FacOutInput) []DLAShare {
	percentASM := FacOutPercentASM(in.Portal, in.Policy)
	base := in.InitialBase
	if base == nil {
		base = new(big.Rat)
	}
	caseASM := strings.Contains(in.Policy.CaseID, "ASM")
	sumOfTSI := in.Policy.SumOfTSI
	if sumOfTSI == nil {
		sumOfTSI = new(big.Rat)
	}

	var slots []facSlot
	if in.TreatyType == treatyFacOBSRB {
		share := in.Policy.TSISpreaded[treatyFacOBSRB]
		if share == nil {
			share = new(big.Rat)
		}
		slots = append(slots, facSlot{name: "SIMAS REINSURANCE BROKER", id: "10038290", jurisdiction: "FACOBSRB", share: share})
		base = mul(sumOfTSI, percentASM)
	} else {
		gp := strings.TrimSpace(in.GroupPanel)
		for _, offer := range in.Policy.FacOffer {
			if offer.Deleted || offer.OfferedMissing {
				continue
			}
			slot := facSlot{name: offer.ReinsurerName, id: offer.ReinsurerID, share: new(big.Rat)}
			keep := true
			switch gp {
			case "006":
				for idx, prop := range offer.Property {
					if strings.TrimSpace(prop.ObjectNo) != strings.TrimSpace(in.ObjectID) {
						continue
					}
					for _, c := range prop.Coverage {
						if strings.TrimSpace(c.Code) != strings.TrimSpace(in.Coverage) {
							continue
						}
						offered := c.ShareOffered
						if strings.TrimSpace(offered) == "" && idx < len(offer.Property) && len(offer.Property[idx].Coverage) > 0 {
							offered = offer.Property[idx].Coverage[0].ShareOffered
						}
						slot.share = decimalOf(offered)
						// GroupPanel 006 menimpa dasar CaseID dengan TSISublimit, atau SumTSI
						// bila sublimit kosong (`DLAFacout_act` :3347).
						value := decimalOf(c.SumTSI)
						if strings.TrimSpace(c.TSISublimit) != "" {
							value = decimalOf(c.TSISublimit)
						}
						base = mul(value, percentASM)
					}
				}
			case "003", "009":
				for _, an := range offer.Aneka {
					if strings.TrimSpace(an.ObjectName) != strings.TrimSpace(in.ObjectName) {
						continue
					}
					if in.CoverageSeq < 1 || in.CoverageSeq > len(an.Coverage) {
						continue
					}
					c := an.Coverage[in.CoverageSeq-1]
					offered := decimalOf(c.ShareOffered)
					slot.share = offered
					shareFac := round(div(decimalOf(c.Percent), big.NewRat(100, 1)), 2)
					value := sumOfTSI
					if caseASM {
						value = decimalOf(c.TSI)
					}
					value = anekaBase(offered, shareFac, value, decimalOf(an.SumTSIAneka), decimalOf(c.SumTSI), decimalOf(c.TSI))
					base = mul(value, percentASM)
					break
				}
			case "004":
				for _, cg := range offer.Cargo {
					if strings.TrimSpace(cg.GoodID) != strings.TrimSpace(in.ContractNo) ||
						strings.TrimSpace(cg.GoodNote) != strings.TrimSpace(in.ObjectName) ||
						strings.TrimSpace(cg.IndexObject) != strings.TrimSpace(in.ObjectID) {
						continue
					}
					if in.CoverageSeq < 1 || in.CoverageSeq > len(cg.Coverage) {
						continue
					}
					c := cg.Coverage[in.CoverageSeq-1]
					offered := c.ShareOffered
					if len(offered) > 38 {
						offered = offered[:38]
					}
					share := round(decimalOf(offered), 4)
					if share.Sign() == 0 {
						keep = false
						break
					}
					slot.share = share
					value := sumOfTSI
					if caseASM {
						value = decimalOf(c.TSI)
					}
					base = mul(value, percentASM)
				}
			}
			if keep {
				slots = append(slots, slot)
			}
		}
	}
	if base.Sign() == 0 || len(slots) == 0 {
		return nil
	}
	adj := moneyRat(in.Amount)
	var out []DLAShare
	for _, s := range slots {
		sum := round(div(mul(s.share, adj), base), 4)
		code := DLACodeFacOut
		if s.jurisdiction == "FACOBSRB" {
			code = DLACodeFacOutSRB
		}
		out = append(out, DLAShare{
			Code: code, Type: DLATypeFacOut, Recipient: s.name, RecipientCode: strings.TrimSpace(s.id),
			Value: decText(sum, 3), Percent: decText(base, 3), ShareSpread: decText(s.share, 3),
			ClaimAmount: decText(adj, 3),
		})
	}
	return out
}

// anekaBase memilih dasar TSI fakultatif Aneka (`DLAFacout_act` langkah 7.4.1): TSI mana
// yang rasio ShareOffered-nya sama dengan persen FacOut.
func anekaBase(offered, shareFac, first, sumAneka, sumTSI, tsi *big.Rat) *big.Rat {
	for _, candidate := range []*big.Rat{first, sumAneka, sumTSI} {
		if candidate.Sign() != 0 && round(div(offered, candidate), 2).Cmp(shareFac) == 0 {
			return candidate
		}
	}
	return tsi
}

// ---- penggolongan baris spreading -------------------------------------------------------

// DLARoute menyatakan pengolah sebuah baris spreading — `GenerateDLAList` langkah 27.4.7.
type DLARoute int

// Pengolah baris spreading.
const (
	DLARouteNone DLARoute = iota
	DLARouteFacOut
	DLARouteBPPDAN
	DLARouteTreaty
)

// RouteSpreading menggolongkan satu kode treaty menurut REINSURANCETYPE.TYPE. FAC-OBLIG
// (10002) dilewati ("kalau fac oblig tidak usah"); 10024 dipaksa treaty; BPPDAN dan EQ
// POOL ke pengolahnya sendiri; PFRA (10029) tidak menghasilkan DLA.
func RouteSpreading(treatyType, reinsCase string) DLARoute {
	switch strings.TrimSpace(treatyType) {
	case treatyFacOblig, treatyPFRA:
		return DLARouteNone
	case treatyBPPDAN, treatyEQPool:
		return DLARouteBPPDAN
	case treatyFacObligIndt:
		return DLARouteTreaty
	}
	switch strings.TrimSpace(reinsCase) {
	case reinsCaseFac:
		return DLARouteFacOut
	case reinsCaseTreaty:
		return DLARouteTreaty
	}
	return DLARouteNone
}

// TreatyUsesQS menyatakan kode treaty yang mencari share QS (`DLATreaty_Act` 13–17).
func TreatyUsesQS(treatyType string) bool {
	return treatyType == treatyQS || treatyType == treatyPQS
}

// DLAShareUnder memilih label "Your Share Under" (CedingCoName) — `DownloadDLA`
// langkah 24–30: huruf pada nomor lalu tipe DLA.
func DLAShareUnder(number, dlaType, qsPQS, treatyName string) string {
	label := ""
	if strings.Contains(number, "M") {
		label = "FACOBLIG"
	}
	if strings.Contains(number, "H") {
		label = "FAC-OUT"
	}
	if strings.Contains(number, "L") {
		label = treatyName
	}
	switch dlaType {
	case DLATypeFacObligIndt, "FACOSRB", DLATypeBPPDAN:
		label = dlaType
	}
	if dlaType == DLATypeTreaty && qsPQS != "" {
		label = qsPQS
	}
	if label == "" {
		label = dlaType
	}
	return label
}

// ---- bilangan --------------------------------------------------------------------------

func moneyRat(m Money) *big.Rat     { return big.NewRat(int64(m), 100) }
func percentRat(p Percent) *big.Rat { return big.NewRat(int64(p), 10_000) }

func mul(a, b *big.Rat) *big.Rat { return new(big.Rat).Mul(a, b) }

func div(a, b *big.Rat) *big.Rat {
	if b.Sign() == 0 {
		return new(big.Rat)
	}
	return new(big.Rat).Quo(a, b)
}

// round membulatkan ke n desimal, setengah menjauhi nol — FloatString melakukannya.
func round(r *big.Rat, n int) *big.Rat {
	out, _ := new(big.Rat).SetString(r.FloatString(n))
	return out
}

func decText(r *big.Rat, n int) string { return r.FloatString(n) }

// DecimalOf membaca teks desimal kolom (spasi dan koma pemisah ribuan diabaikan).
func DecimalOf(s string) *big.Rat { return decimalOf(s) }

func decimalOf(s string) *big.Rat {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return new(big.Rat)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return new(big.Rat)
	}
	return r
}

// plainDecimal menulis rasional tanpa nol di belakang koma — "0", "25", "12.5".
func plainDecimal(r *big.Rat) string {
	s := r.FloatString(10)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

// ---- pelanggaran -----------------------------------------------------------------------

// Pesan aturan DLA. Yang dari Pega disalin apa adanya; tambahan berbahasa Inggris (`D-80`).
const (
	msgDLANotAllowed = "Print DLA is only available once the insured has responded to the LOD on an accepted adjustment."
	msgDLAZeroShare  = "Ada DLA Share yang nilainya 0" // DownloadDLA langkah 11
)

func dlaViolation(code ViolationCode, message string) error {
	return &ValidationError{Violation: []Violation{{Code: code, Field: "dla", Message: message}}}
}

// CanPrintDLA adalah aturan tombol Print DLA (`ShowAdjustment_sect` / `PNCTombolDLA_sect`):
// adjustment sudah diakseptasi komite (STATUSAKSEPTASI 1), dan Persetujuan Tertanggung
// terisi. Tombolnya mati bila persetujuan bukan "1" atau status akseptasi "2".
func CanPrintDLA(line SettlementLine) error {
	if strings.TrimSpace(line.AcceptanceStatus) != "1" || strings.TrimSpace(line.AcceptanceLODStatus) != LODAgreed {
		return dlaViolation(ViolationDLANotAllowed, msgDLANotAllowed)
	}
	return nil
}

// ErrDLAZeroShare adalah penolakan cetak bila ada DLA bernilai nol.
func ErrDLAZeroShare() error { return dlaViolation(ViolationDLAZeroShare, msgDLAZeroShare) }
