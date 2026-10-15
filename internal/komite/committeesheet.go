package komite

import (
	"math/big"
	"strings"
	"time"

	"claim-pnc/internal/platform/money"
)

// Isi kolom kanan dan tab tambahan layar "Lihat Detail Transfer" — bagian
// `Section/ShowTransferDetail` yang dulu belum dibangun.
//
// ============================================================================
// DARI MANA ANGKANYA DI PEGA
// ============================================================================
//
// Pega tidak membaca kolom kanan ini dari basis data. Ia MENGHITUNGNYA di clipboard, saat
// case komite dibentuk, lewat `Activity/CalculatedSpredingForClaimKomite`:
//
//	nilai komite   langkah 5   AdjusterFeeValue bila Type 4/7, SalvageValue bila Type 3,
//	                           AdjustmentValue selain itu
//	SpreadingList  langkah 11  KmtSpred  = nilai komite × SharePercentage / 100
//	CoMember       langkah 12  TSIShare  = PercentShare × GrossValue / 100 — hanya bila
//	                           Type bukan 3 DAN CoinsList polis berisi lebih dari satu baris
//
// Sedangkan LEADER (`tempCvg.ObjectSurveyor` / `ObjectSurveyorMarine`) diisi
// `SetListComiteeClaimPerObjAdj`: nama dan PercentShare baris CoinsList yang Leader-nya
// "true", atau "ASURANSI SINAR MAS" 100 bila polis tanpa koasuransi.
//
// Rumus-rumus itu disalin di sini apa adanya dan dihitung ulang setiap kali layar dibuka,
// dari tabel yang memuat bahan yang sama — `T_CLAIM_ADJUSTMENT`, `T_CLAIM_SPREADING`, dan
// CoinsList dokumen polis di `JSON_POLIS`.
//
// ============================================================================
// YANG SENGAJA TIDAK DIHITUNG
// ============================================================================
//
//   - Cabang Quota Share (langkah 11.3–11.7) membaca tabel treaty lewat dua RDB-List yang
//     tidak ada di export. Baris spreading QS karena itu ditampilkan dengan rumus umum.
//   - "Spreding (%)" dan "Result Value" Fac-Out (langkah 19–20) diturunkan dari
//     `FacOutObjectList.ShareOffered`, `TSISublimit`, dan `LimitOfLiability` per coverage
//     polis, dengan tiga cabang bersyarat yang syaratnya tidak terbaca di export. Yang
//     ditampilkan adalah nama reasuradur dan share pada dokumen polis — bukan hasil hitung
//     Pega — dan layar menyatakannya.
//   - Polis Fac In (`TypeOfCoins` "F") memakai `OfferFacIn.PercentShare` sebagai share
//     leader; bagian penawaran fakultatifnya belum dibaca modul ini.

// OwnCompany adalah nama bawaan leader bila polis tanpa koasuransi —
// `SetListComiteeClaimPerObjAdj`, cabang `TempGetApp.LSC_ID != "SIMASNET"`.
//
// Pada portal Simas Insurtech Pega memakai "ASURANSI SIMAS INSURTECH". Portal belum
// dibedakan modul ini (`D-75`), jadi yang dipakai nama ASM.
const OwnCompany = "ASURANSI SINAR MAS"

// PolicyFacts adalah bagian dokumen polis (`JSON_POLIS`) yang dibaca layar ini.
type PolicyFacts struct {
	Start time.Time
	End   time.Time

	// Coinsurance adalah CoinsList, urut seperti di dokumen.
	Coinsurance []CoinsuranceShare

	// FacOffers adalah FacOfferList — reasuradur fakultatif keluar.
	FacOffers []FacOffer
}

// CoinsuranceShare adalah satu baris CoinsList polis.
type CoinsuranceShare struct {
	Name    string
	Leader  bool
	Percent string // PercentShare apa adanya, mis. "65" atau "6.0945"
}

// FacOffer adalah satu baris FacOfferList polis.
type FacOffer struct {
	ReinsurerName string
	Percent       string // PctShareForAllObj pada dokumen polis — BUKAN hasil hitung Pega
}

// SpreadingShare adalah satu baris `POOLDATA.T_CLAIM_SPREADING`.
type SpreadingShare struct {
	ObjectID   string
	CoverageID string
	TreatyType string

	// TreatyName adalah NOTE master `REINSURANCETYPE` — yang ditampilkan dropdown Pega
	// (`BrowseReinsuranceType_RD`, prompt `.Note`, nilai `.ID`).
	TreatyName string
	Percent    string
}

// Attachment adalah satu baris `POOLDATA.DATA_ATTACHFILE` milik klaimnya.
type Attachment struct {
	ID       string
	Name     string
	Note     string
	Category string
	InputBy  string
	InputAt  time.Time
}

// Leader adalah baris "LEADER" kolom kanan.
type Leader struct {
	Name    string
	Percent string
}

// Leader menurunkan leader koasuransi seperti `SetListComiteeClaimPerObjAdj`.
func (d TransferDetail) Leader() Leader {
	for _, c := range d.Policy.Coinsurance {
		if c.Leader {
			return Leader{Name: c.Name, Percent: c.Percent}
		}
	}
	return Leader{Name: OwnCompany, Percent: "100"}
}

// FullSpreading menyatakan apakah blok spreading tampil dalam bentuk empat kolom
// (Spreading, Currency, Share, Result Value) beserta blok Fac-Out.
//
// `ShowTransferDetail` memilihnya dengan perbandingan PERSIS
// `tempCvg.ObjectSurveyor == 'ASURANSI SINAR MAS'`. Nama leader dari CoinsList
// ("ASURANSI SINAR MAS - KANTOR PUSAT") tidak sama persis, sehingga bentuk ringkas hanya
// muncul pada polis tanpa koasuransi. Ditiru apa adanya (`P-5`).
func (d TransferDetail) FullSpreading() bool {
	return d.Leader().Name != OwnCompany
}

// CommitteeValue adalah "nilai komite" satu baris — `Local.nilaiklaim` langkah 5.
func (l AdjustmentLine) CommitteeValue() money.Money {
	switch kodeRingkas(l.PaymentType) {
	case "4", "7":
		return l.AdjusterFee
	case "3":
		return l.SalvageValue
	default:
		return l.ASMShareValue
	}
}

// SpreadingRow adalah satu baris "List Spreading".
type SpreadingRow struct {
	TreatyType string
	TreatyName string
	Currency   string
	Percent    string

	// Value adalah KmtSpred. HasValue salah bila share-nya tidak dapat dibaca sebagai
	// angka — nol di sana tidak dapat dibedakan dari "tidak dihitung".
	Value    money.Money
	HasValue bool
}

// SpreadingFor menyusun "List Spreading" satu baris adjustment.
//
// Spreading-nya milik coverage yang sama (OBJECTID, OBJECTCOVERAGEID) — di Pega ia
// `AdjustmentList().SpreadingList` yang disalin dari coverage-nya.
func (d TransferDetail) SpreadingFor(l AdjustmentLine) []SpreadingRow {
	var rows []SpreadingRow
	base := l.CommitteeValue()
	for _, s := range d.Spreading {
		if !sameKey(s.ObjectID, l.ObjectID) || !sameKey(s.CoverageID, l.CoverageID) {
			continue
		}
		value, ok := ShareOf(base, s.Percent)
		rows = append(rows, SpreadingRow{
			TreatyType: s.TreatyType,
			TreatyName: s.TreatyName,
			Currency:   l.CurrencyCode,
			Percent:    s.Percent,
			Value:      value,
			HasValue:   ok,
		})
	}
	return rows
}

// CoMemberRow adalah satu baris "CO MEMBER".
type CoMemberRow struct {
	Name     string
	Currency string
	Percent  string
	Value    money.Money
	HasValue bool
}

// CoMembersFor menyusun "CO MEMBER" satu baris adjustment — langkah 12, dengan syarat
// `Local.cekcoas`: Type bukan 3 dan CoinsList lebih dari satu baris.
func (d TransferDetail) CoMembersFor(l AdjustmentLine) []CoMemberRow {
	if kodeRingkas(l.PaymentType) == "3" || len(d.Policy.Coinsurance) <= 1 {
		return nil
	}
	rows := make([]CoMemberRow, 0, len(d.Policy.Coinsurance))
	for _, c := range d.Policy.Coinsurance {
		value, ok := ShareOf(l.GrossValue, c.Percent)
		rows = append(rows, CoMemberRow{
			Name:     c.Name,
			Currency: l.CurrencyCode,
			Percent:  c.Percent,
			Value:    value,
			HasValue: ok,
		})
	}
	return rows
}

// ShareOf menghitung `nilai × persen / 100`, dibulatkan setengah menjauhi nol ke satuan
// terkecil (sen).
//
// Dihitung dengan bilangan rasional, bukan float: persennya dapat berdesimal empat
// ("6.0945"), dan float membuat hasil yang sama berbeda satu sen antar mesin (`I-12`).
// Persen yang tidak terbaca mengembalikan ok salah.
func ShareOf(value money.Money, percent string) (money.Money, bool) {
	p, ok := new(big.Rat).SetString(strings.TrimSpace(percent))
	if !ok {
		return 0, false
	}
	r := new(big.Rat).SetInt64(value.MinorUnits())
	r.Mul(r, p)
	r.Quo(r, big.NewRat(100, 1))

	num := new(big.Int).Set(r.Num())
	den := r.Denom()
	neg := num.Sign() < 0
	num.Abs(num)
	q, m := new(big.Int).QuoRem(num, den, new(big.Int))
	if new(big.Int).Mul(m, big.NewInt(2)).Cmp(den) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if neg {
		q.Neg(q)
	}
	if !q.IsInt64() {
		return 0, false
	}
	return money.FromMinorUnits(q.Int64()), true
}

// sameKey membandingkan kunci objek/coverage tanpa peduli spasi maupun nol di depan.
func sameKey(a, b string) bool { return kodeRingkas(a) == kodeRingkas(b) }
