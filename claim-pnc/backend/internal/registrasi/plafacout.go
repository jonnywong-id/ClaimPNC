package registrasi

import (
	"math/big"
	"strings"
)

// # PLA reasuransi: FAC OUT dan BPPDAN / EQ POOL
//
// Sumbernya rantai pra-proses flow action PrintPLA:
//
//	GeneratePLAListObject → GeneratePLAList → PLAFacout_Act / PLABPPDAN_Act / PLATreaty_Act
//	                                        → PLACoins_Act
//
// `GeneratePLAList` menyiapkan bahan bersama (langkah 10–24): reserve per mata uang
// (TempEstimasi.CoinsList.TSIShare, lihat PLAReserve), bagian ASM percentASM dan isASMLeader
// (langkah 15–17, termasuk FAC IN), dan HandingFee = TSIShare × percentASM bila ASM leader,
// selain itu TSIShare (langkah 19). Lalu SETIAP baris spreading jaminan digolongkan lewat
// REINSURANCETYPE (`GetTypeReinsuranceSpreading`, langkah 26): BPPDAN/EQ POOL (10010/10023)
// ke PLABPPDAN_Act, fakultatif (tipe 3) ke PLAFacout_Act, treaty (tipe 2) ke PLATreaty_Act.
//
// Langkah berlabel `//` pada activity-activity itu adalah langkah yang DIKOMENTARI di Pega
// (label `//` mematikan langkah) — setiap langkah berlabel `//` di sana punya pasangan versi
// baru tanpa label tepat di sebelahnya. Yang dibawa di sini adalah versi tanpa label.

// Tipe dan huruf nomor PLA reasuransi — `PLAFacout_Act` 4.3–4.4, `PLABPPDAN_Act` 3.2 dan
// 3.7–3.8.
const (
	PLATypeFacOut = "FACOUT"
	PLATypeBPPDAN = "BPPDAN"
	PLATypeEQPool = "EQPOOL"

	PLACodeFacOut    = "H"
	PLACodeFacOutSRB = "M" // FACOBSRB
	PLACodeBPPDAN    = "H"
)

// PLAReinsuranceInput adalah bahan PLA reasuransi satu baris spreading jaminan klaim.
type PLAReinsuranceInput struct {
	TreatyType  string // TreatyType baris spreading
	GroupPanel  string
	ObjectID    string // ObjectList.ObjectID — local.IDObj
	ObjectName  string // ObjectList.ObjectName
	Coverage    string // CoverageOldID — param.coverageOldID
	CoverageSeq int    // param.objectcoverageid, berbasis 1
	PercentASM  *big.Rat
	Policy      DLAPolicy
	Offers      []FacOffer
}

// PLARecipient adalah satu penerima PLA reasuransi beserta bagian dan dasar pembaginya.
type PLARecipient struct {
	Type, Code string
	Name, ID   string
	Share      *big.Rat // FAC OUT: Name (nilai); BPPDAN: persen bagian
	Base       *big.Rat // FAC OUT: ShareTSI (nilai); BPPDAN: tidak dipakai
}

// FacOutPLARecipients adalah `PLAFacout_Act` langkah 2–3: penerima PLA fakultatif keluar
// beserta bagian (Name) dan dasar pembaginya (ShareTSI).
//
// Perilaku Pega yang dibawa apa adanya:
//   - FACOBSRB (10061) menghasilkan satu penerima, SIMAS REINSURANCE BROKER, dengan bagian
//     TSISpreaded baris FACOBSRB polis dan dasar SumOfTSI polis × percentASM (langkah 2).
//   - FacOffer ber-FlagDelete 1 dilewati (langkah 3).
//   - ShareTSI TIDAK dikosongkan di antara FacOffer: FacOffer yang objeknya tidak cocok
//     memakai dasar FacOffer sebelumnya (`local.ShareTSI` satu untuk seluruh perulangan).
//   - Group Panel 002/005 (PA, Travel) tidak punya cabang: bagiannya 0.
//
// Penyimpangan sadar: baris T_FACOFFER tanpa JSONDATA (FacOffer.FlatShare) memakai
// PCT_SHAREREAS persen dengan dasar 100 — Pega tidak punya jalur ini karena FacOfferList
// selalu terisi di halaman polisnya.
func FacOutPLARecipients(in PLAReinsuranceInput) []PLARecipient {
	pct := in.PercentASM
	if pct == nil {
		pct = big.NewRat(1, 1)
	}
	caseASM := strings.Contains(in.Policy.CaseID, "ASM")
	sumOfTSI := in.Policy.SumOfTSI
	if sumOfTSI == nil {
		sumOfTSI = new(big.Rat)
	}

	if strings.TrimSpace(in.TreatyType) == treatyFacOBSRB {
		share := in.Policy.TSISpreaded[treatyFacOBSRB]
		if share == nil {
			share = new(big.Rat)
		}
		return []PLARecipient{{Type: PLATypeFacOut, Code: PLACodeFacOutSRB,
			Name: "SIMAS REINSURANCE BROKER", ID: "10038290", Share: share, Base: mul(sumOfTSI, pct)}}
	}

	gp := strings.TrimSpace(in.GroupPanel)
	base := new(big.Rat) // local.ShareTSI — tidak dikosongkan di antara FacOffer
	var out []PLARecipient
	for _, offer := range in.Offers {
		if offer.Deleted {
			continue
		}
		r := PLARecipient{Type: PLATypeFacOut, Code: PLACodeFacOut,
			Name: offer.ReinsurerName, ID: strings.TrimSpace(offer.ReinsurerID), Share: new(big.Rat)}

		if strings.TrimSpace(offer.FlatShare) != "" {
			r.Share, r.Base = decimalOf(offer.FlatShare), big.NewRat(100, 1)
			out = append(out, r)
			continue
		}

		keep := true
		switch gp {
		case "006":
			// Langkah 3.1: objek PropertyList yang ObjectNo-nya sama, coverage yang kodenya
			// sama dengan CoverageOldID jaminan.
			for _, o := range offer.Property {
				if strings.TrimSpace(o.ObjectNo) != strings.TrimSpace(in.ObjectID) {
					continue
				}
				for _, c := range o.Coverage {
					if strings.TrimSpace(c.Code) != strings.TrimSpace(in.Coverage) {
						continue
					}
					factor := big.NewRat(1, 1)
					if strings.TrimSpace(c.Sublimit) != "" {
						factor = round(div(decimalOf(c.Sublimit), big.NewRat(100, 1)), 5)
					}
					r.Share = mul(decimalOf(c.ShareOffered), factor)
					tsi := c.TSISublimit
					if strings.TrimSpace(tsi) == "" {
						tsi = c.TSI
					}
					base = mul(decimalOf(tsi), pct)
				}
			}
		case "003", "009":
			// Langkah 3.2: objek AnekaList yang ObjectName-nya sama, coverage urutan jaminan.
			for _, o := range offer.Aneka {
				if strings.TrimSpace(o.ObjectName) != strings.TrimSpace(in.ObjectName) {
					continue
				}
				var c FacCoverage
				if in.CoverageSeq >= 1 && in.CoverageSeq <= len(o.Coverage) {
					c = o.Coverage[in.CoverageSeq-1]
				}
				offered := c.ShareOffered
				r.Share = decimalOf(offered)
				tsiShare := round(div(mul(decimalOf(c.TSI), decimalOf(c.Percent)), big.NewRat(100, 1)), 4)
				if decimalOf(offered).Cmp(tsiShare) == 0 {
					base = mul(decimalOf(c.TSI), pct)
				} else {
					base = decimalOf(c.SumTSISpreaded)
				}
				// Langkah 3.2.3–3.2.4: cadangan dari FacOffer pertama bila kosong.
				if strings.TrimSpace(offered) == "" {
					r.Share = decimalOf(firstAnekaCoverage(in.Offers).ShareOffered)
				}
				if base.Sign() == 0 {
					if caseASM {
						base = mul(decimalOf(firstAnekaCoverage(in.Offers).TSI), pct)
					} else {
						base = mul(sumOfTSI, pct)
					}
				}
			}
		case "004":
			// Langkah 3.3: cargo yang GoodNote dan IndexObject-nya sama; bagian nol
			// mengeluarkan reasuradur itu dari daftar.
			for _, o := range offer.Cargo {
				if strings.TrimSpace(o.GoodNote) != strings.TrimSpace(in.ObjectName) ||
					strings.TrimSpace(o.IndexObject) != strings.TrimSpace(in.ObjectID) {
					continue
				}
				var c FacCoverage
				if in.CoverageSeq >= 1 && in.CoverageSeq <= len(o.Coverage) {
					c = o.Coverage[in.CoverageSeq-1]
				}
				offered := c.ShareOffered
				if len(offered) > 38 {
					offered = offered[:38]
				}
				share := round(decimalOf(offered), 4)
				if share.Sign() == 0 {
					keep = false
					break
				}
				r.Share = share
				if caseASM {
					base = decimalOf(c.TSI)
				} else {
					base = mul(sumOfTSI, pct)
				}
			}
		}
		if !keep {
			continue
		}
		r.Base = base
		out = append(out, r)
	}
	return out
}

// firstAnekaCoverage adalah FacOfferList(1).AnekaList(1).CoverageList(1) — cadangan
// `PLAFacout_Act` langkah 3.2.3–3.2.4.
func firstAnekaCoverage(offers []FacOffer) FacCoverage {
	if len(offers) > 0 && len(offers[0].Aneka) > 0 && len(offers[0].Aneka[0].Coverage) > 0 {
		return offers[0].Aneka[0].Coverage[0]
	}
	return FacCoverage{}
}

// BPPDANPLARecipient adalah `PLABPPDAN_Act`: satu penerima per baris spreading BPPDAN
// (10010 → BPPDAN, 10038311) atau EQ POOL (10023 → REASURANSI MAIPARK INDONESIA, 10038726),
// dengan bagian SharePercentage baris itu (persen).
func BPPDANPLARecipient(treatyType string, share Percent) (PLARecipient, bool) {
	r := PLARecipient{Code: PLACodeBPPDAN, Share: percentRat(share)}
	switch strings.TrimSpace(treatyType) {
	case treatyBPPDAN:
		r.Type, r.Name, r.ID = PLATypeBPPDAN, "BPPDAN", "10038311"
	case treatyEQPool:
		r.Type, r.Name, r.ID = PLATypeEQPool, "REASURANSI MAIPARK INDONESIA", "10038726"
	default:
		return PLARecipient{}, false
	}
	return r, true
}

// PLAReinsuranceAmounts menghitung EstimasiList PLA reasuransi seorang penerima.
//
// FAC OUT (`PLAFacout_Act` 4.8): PercentBrokerage = round2(Name), PercentPPN =
// round2(HandingFee), PercentHandlingFee = round2(ShareTSI), PercentPPH = round2(Brokerage ×
// PPN / HandlingFee) — tercatat EstimationValue = PPN, SharePLA = Brokerage, PercentPLA =
// HandlingFee, ResultPLA = PPH. Dasar pembagi nol menghasilkan nol, bukan galat.
//
// BPPDAN/EQ POOL (`PLABPPDAN_Act` 3.5 dan 3.9): hasil = HandingFee × round5(persen/100) —
// tercatat EstimationValue = round2(HandingFee), SharePLA = round2(persen), ResultPLA =
// round2(hasil). Cabang "TSI objek > 20 M" berlabel `//` (mati) dan tidak dibawa.
func PLAReinsuranceAmounts(reserve []FaceSheetAmount, currencyID map[string]string, r PLARecipient,
	percentASM *big.Rat, asmLeader bool) []PLAAmount {
	pct := percentASM
	if pct == nil {
		pct = big.NewRat(1, 1)
	}
	out := make([]PLAAmount, 0, len(reserve))
	for _, x := range reserve {
		fee := moneyRat(x.Value)
		if asmLeader {
			fee = mul(fee, pct)
		}
		a := PLAAmount{Currency: x.Currency, CurrencyID: currencyID[x.Currency], Reserve: x.Value,
			ASMCount: moneyOf(mul(moneyRat(x.Value), pct))}
		switch r.Type {
		case PLATypeBPPDAN, PLATypeEQPool:
			share := round(div(r.Share, big.NewRat(100, 1)), 5)
			a.Base = moneyOf(fee)
			a.FacShare = moneyOf(r.Share)
			a.Result = moneyOf(mul(fee, share))
		default:
			brokerage := round(r.Share, 2)
			ppn := round(fee, 2)
			base := new(big.Rat)
			if r.Base != nil {
				base = round(r.Base, 2)
			}
			a.Base, a.FacShare, a.FacBase = moneyOf(ppn), moneyOf(brokerage), moneyOf(base)
			if base.Sign() != 0 {
				a.Result = moneyOf(round(div(mul(brokerage, ppn), base), 2))
			}
		}
		out = append(out, a)
	}
	return out
}

// msgPLANoRecipient adalah pesan saat tidak satu pun penerima PLA ditemukan.
const msgPLANoRecipient = "Tidak ada penerima PLA: polis tidak punya anggota koasuransi maupun Fac Offer untuk jaminan ini."

// ErrPLANoRecipient adalah pelanggaran saat PLA koasuransi maupun FAC OUT tidak punya penerima.
func ErrPLANoRecipient() error { return plaViolation(ViolationPLANoRecipient, msgPLANoRecipient) }

// StatusClaimPLAReport adalah Status Klaim setelah PLA dibuat — `GeneratePLAList` langkah 33
// (`pyWorkPage.ClaimData.StatusClaim := "1138"`, master: "PLA Report").
const StatusClaimPLAReport ClaimStatus = "1138"

// ErrPLAExGratia: klaim Ex Gratia tidak menerbitkan PLA — `GeneratePLAList` langkah 6
// ("jika exgratia exit act"). Pega keluar diam-diam; di sini alasannya disebut.
func ErrPLAExGratia() error {
	return plaViolation(ViolationPLAExGratia, "Klaim Ex Gratia tidak menerbitkan PLA.")
}

// ErrPLARemarksEmpty: PLA tidak dicetak selama REMARKS-nya kosong — `DownloadAllDocumentPLA`
// (`Local.erremark := "Remarks kosong, isi terlebih dahulu."`).
func ErrPLARemarksEmpty() error {
	return plaViolation(ViolationPLARemarksEmpty, "Remarks kosong, isi terlebih dahulu.")
}

// ASMCoinsShare adalah `DownloadFireLossAdvice_act` langkah 14: bagian dan status leader
// perusahaan sendiri pada CoinsList. Tanpa koasuransi: 1 dan bukan leader.
func ASMCoinsShare(portal string, members []PLACoinsMember) (*big.Rat, bool) {
	own := OwnCompanyOf(portal)
	pct, leader := big.NewRat(1, 1), false
	for _, m := range members {
		if strings.TrimSpace(m.ID) == "" || !strings.Contains(strings.ToUpper(m.Name), own) {
			continue
		}
		if m.HasShare {
			pct = round(div(percentRat(m.Share), big.NewRat(100, 1)), 4)
		}
		leader = m.Leader
	}
	return pct, leader
}

// moneyOf membulatkan nilai rupiah ke sen terdekat (setengah menjauhi nol).
func moneyOf(r *big.Rat) Money {
	if r == nil {
		return 0
	}
	cents := new(big.Rat).Mul(r, big.NewRat(100, 1))
	half := big.NewRat(1, 2)
	if cents.Sign() < 0 {
		half.Neg(half)
	}
	cents.Add(cents, half)
	return Money(new(big.Int).Quo(cents.Num(), cents.Denom()).Int64())
}

// PLAReserve adalah reserve PLA per mata uang — `GeneratePLAList` langkah 18 dan 20:
//
//   - estimasi yang dijumlahkan adalah yang SUDAH dibuatkan Claim Face Sheet (PrintFaceClaim),
//     APA PUN tipenya — berbeda dari reserve CFS (FaceSheetReserve) yang hanya Estimasi Klaim;
//   - mata uang yang totalnya nol dibuang (langkah 20, `Page-Remove` bila TSIShare == 0);
//   - jaminan PA PHK (coverage 10010, 10023, 10018): mata uang terakhir memakai ProposeValue
//     adjustment terakhir (langkah 18.2).
func PLAReserve(c Coverage, currency func(string) string) []FaceSheetAmount {
	var out []FaceSheetAmount
	index := map[string]int{}
	for _, item := range c.Item {
		for _, e := range item.Estimation {
			if !e.FaceSheet {
				continue
			}
			name := currency(e.Currency)
			i, ok := index[name]
			if !ok {
				i = len(out)
				index[name] = i
				out = append(out, FaceSheetAmount{Currency: name})
			}
			out[i].Value += e.Value
		}
	}
	switch strings.TrimSpace(c.ID) {
	case "10010", "10023", "10018":
		if len(out) > 0 && len(c.Settlement) > 0 {
			out[len(out)-1].Value = c.Settlement[len(c.Settlement)-1].Submitted
		}
	}
	result := out[:0]
	for _, r := range out {
		if r.Value != 0 {
			result = append(result, r)
		}
	}
	return result
}

// PLAShareLabel adalah TempPLASpreading.CedingCoName — akhiran label "Your Share" dokumen
// PLA (`DownloadPLA` langkah 13 dan 19–23, dijalankan berurutan sehingga yang terakhir
// menang): tipe PLA, lalu "FACOBLIG" bila nomornya memuat M, "FAC-OUT" bila memuat H (juga
// EQ POOL, yang bernomor H), dan "BPPDAN" untuk tipe BPPDAN.
func PLAShareLabel(p PLA) string {
	label := p.Type
	if strings.Contains(p.Number, "M") {
		label = "FACOBLIG"
	}
	if strings.Contains(p.Number, "H") {
		label = "FAC-OUT"
	}
	if p.Type == PLATypeBPPDAN {
		label = "BPPDAN"
	}
	return label
}

// RupiahOf membulatkan nilai desimal ke Money (sen terdekat).
func RupiahOf(r *big.Rat) Money { return moneyOf(r) }
