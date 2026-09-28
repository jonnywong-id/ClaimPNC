package registrasi

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// # Claim Face Sheet
//
// Tombol "Download Claim Face Sheet" pada baris jaminan tahap Input Estimasi. Isinya
// dicetak rule HTML `ClaimFaceSheetHTML` (kelas Work-PNC). Activity yang MENGISI halaman
// sementaranya (TempDate, TempDownload, tempEstimation, tempSpreadingResult, TempReas1,
// tempCoins) TIDAK ADA di export, sehingga isi tiap baris diturunkan dari tiga sumber:
//
//   - template itu sendiri: urutan, judul, dan properti yang dirujuk;
//   - satu contoh PDF produksi yang diserahkan Work Owner (2026-09-27);
//   - activity yang menghitung halaman serupa untuk survey dan komite:
//     `CountTempOfEstimationBaseOnCurrency` (reserve per mata uang, hanya estimasi tipe
//     "1") dan `CalculationCoasSpreadingSurvey` (bagian anggota = share × nilai; share ASM
//     hanya dikenakan bila Sinar Mas leader).
//
// Menekan tombol juga MENGUNCI estimasi jaminan itu (PrintFaceClaim = 1) dan mencatat
// satu revisi CFSList — seperti `ObjectCoverageList().CFSList()` pada JSON klaim Pega.

// FaceSheetAmount adalah satu nilai dalam satu mata uang.
type FaceSheetAmount struct {
	Currency string // nama mata uang, mis. IDR
	Value    Money
}

// FaceSheetShare adalah satu baris pembagian: spreading, reasuradur fac out, atau anggota
// koasuransi.
type FaceSheetShare struct {
	Name     string
	Share    Percent
	Currency string
	Value    Money
}

// FaceSheet adalah seluruh isi satu Claim Face Sheet.
type FaceSheet struct {
	Title        string // Quotation.BusinessName, atau BusinessType bila kosong
	Date         time.Time
	AdminName    string
	Insured      string
	PolicyNumber string
	RiskLocation string
	PeriodStart  time.Time
	PeriodEnd    time.Time
	Class        string
	Coverage     string
	Object       string
	SumInsured   FaceSheetAmount
	Broker       string

	TechnicalPIC     string
	ClaimNumber      string
	NotificationDate time.Time
	DateOfLoss       time.Time
	RegistrationDate time.Time
	NatureOfLoss     string

	// Travel menukar label baris Object menjadi PARTICIPANT(S) dan menghilangkan Nature
	// of Loss, seperti cabang `BusinessType='Travel'` pada template.
	Travel bool

	// Payments adalah status pembayaran premi. Sumbernya (TempDownload.M_COL_ID) diisi
	// activity yang hilang; sampai ada, kosong — lebih baik kosong daripada menebak.
	Payments string

	Reserve   []FaceSheetAmount
	Spreading []FaceSheetShare
	Reinsurer []FaceSheetShare
	CoMember  []FaceSheetShare
}

// FacReinsurer adalah satu penawaran fakultatif keluar polis — FacOfferList.
type FacReinsurer struct {
	Name     string  // ReinsurerName
	Share    Percent // PctShareForAllObj
	HasShare bool
}

// FaceSheetRevision adalah satu baris CFSList — POOLDATA.TC_PNC_CFS beserta
// TC_PNC_CFS_ESTIMASI.
type FaceSheetRevision struct {
	ClaimID     string
	ObjectID    string
	CoverageSeq int
	Revision    int
	Date        time.Time
	FileName    string
	Reserve     []FaceSheetAmount
}

// FaceSheetSource adalah seam ke data pendamping Claim Face Sheet dan penyimpanan
// revisinya.
type FaceSheetSource interface {
	// CauseOfLossName membaca uraian penyebab kerugian (V_D_CAUSE_OF_LOSS.DESCRIPTION).
	CauseOfLossName(ctx context.Context, id string) (string, error)

	// OperatorName membaca nama operator (M_LOGIN_PNC.LOGIN_NAME); kosong bila tidak ada.
	OperatorName(ctx context.Context, operatorID string) (string, error)

	// Coinsurance dan FacReinsurers membaca CoinsList dan FacOfferList dokumen polis.
	Coinsurance(ctx context.Context, policyNumber string) ([]CoinsuranceRow, error)
	FacReinsurers(ctx context.Context, policyNumber string) ([]FacReinsurer, error)

	// LastRevision mengembalikan nomor revisi terakhir sebuah jaminan.
	LastRevision(ctx context.Context, claimID, objectID string, coverageSeq int) (int, bool, error)

	// SaveRevision menulis satu revisi. Dipanggil di dalam UnitOfWork.
	SaveRevision(ctx context.Context, r FaceSheetRevision) error
}

// FaceSheetRenderer mengubah Claim Face Sheet menjadi dokumen yang diunduh.
type FaceSheetRenderer interface {
	Render(f FaceSheet) ([]byte, error)
}

// Pesan aturan Claim Face Sheet.
const (
	// msgFaceSheetMissing adalah pesan `ValidateInputEstimate_act` langkah 9.
	msgFaceSheetMissing = "Tidak Bisa Tambah Estimate, Belum Claim Face Sheet"

	msgFaceSheetNothingNew = "Tidak ada estimasi baru untuk dibuatkan Claim Face Sheet."
)

// FaceSheetFileName meniru nama berkas CFSList Pega: `ClaimFaceSheetHTML1#1/Revisi0.pdf`.
func FaceSheetFileName(objectIndex, coverageIndex, revision int) string {
	return fmt.Sprintf("ClaimFaceSheetHTML%d#%d/Revisi%d.pdf", objectIndex, coverageIndex, revision)
}

// CheckNewEstimates menegakkan aturan kedua `ValidateInputEstimate_act`: estimasi baru
// pada sebuah item hanya boleh ditambahkan bila estimasi sebelumnya sudah dibuatkan
// Claim Face Sheet.
//
// Yang diperiksa hanya estimasi BARU (indeks >= stored). Estimasi lama yang tersimpan
// sebelum aturan ini dibawa tidak ikut ditolak.
func CheckNewEstimates(item ObjectItem, stored int) error {
	for m := stored; m < len(item.Estimation); m++ {
		if m > 0 && !item.Estimation[m-1].FaceSheet {
			return &ValidationError{Violation: []Violation{{
				Code: ViolationFaceSheetMissing, Field: "estimasi", Message: msgFaceSheetMissing,
			}}}
		}
	}
	return nil
}

// Share menghitung bagian sebuah nilai menurut persentase, dibulatkan ke sen terdekat.
func (u Money) Share(p Percent) Money {
	result := new(big.Rat).SetFrac(
		new(big.Int).Mul(big.NewInt(int64(u)), big.NewInt(int64(p))),
		big.NewInt(int64(PercentFull)),
	)
	half := big.NewRat(1, 2)
	if result.Sign() < 0 {
		half.Neg(half)
	}
	result.Add(result, half)
	return Money(new(big.Int).Quo(result.Num(), result.Denom()).Int64())
}

// FaceSheetInput adalah bahan penyusun satu Claim Face Sheet.
type FaceSheetInput struct {
	Claim         Claim
	ObjectIndex   int // 0-based
	CoverageIndex int // 0-based
	Policy        Policy
	CoMember      []CoinsuranceRow
	Reinsurer     []FacReinsurer
	CauseOfLoss   string
	AdminName     string
	CurrencyName  map[string]string
	Now           time.Time
}

// BuildFaceSheet menyusun isi Claim Face Sheet satu jaminan.
func BuildFaceSheet(in FaceSheetInput) FaceSheet {
	k := in.Claim
	object := k.InsuredItem[in.ObjectIndex]
	coverage := object.Coverage[in.CoverageIndex]
	currency := func(code string) string {
		if name := in.CurrencyName[code]; name != "" {
			return name
		}
		return code
	}

	f := FaceSheet{
		Title:            firstNonEmpty(k.Policy.BusinessName, in.Policy.BusinessName, k.Policy.BusinessType),
		Date:             in.Now,
		AdminName:        firstNonEmpty(in.AdminName, k.CreatedBy),
		Insured:          firstNonEmpty(k.Policy.InsuredName, in.Policy.InsuredName),
		PolicyNumber:     k.Policy.Number,
		RiskLocation:     strings.ToUpper(k.Location),
		PeriodStart:      in.Policy.CoverageStart,
		PeriodEnd:        in.Policy.CoverageEnd,
		Coverage:         firstNonEmpty(coverage.Name, coverage.ID),
		Object:           object.Name,
		SumInsured:       FaceSheetAmount{Currency: currency(k.Policy.Currency), Value: coverage.TSI},
		Broker:           firstNonEmpty(k.Policy.SourceOfBusinessName, in.Policy.SourceOfBusinessName),
		TechnicalPIC:     k.TechnicalPIC,
		ClaimNumber:      k.Number,
		NotificationDate: k.ReportDate,
		DateOfLoss:       k.DateOfLoss,
		RegistrationDate: k.CreatedAt,
		NatureOfLoss:     in.CauseOfLoss,
		Travel:           k.Policy.Line == LineTravel,
	}
	if f.Travel {
		f.Class = strings.ToUpper(f.Title)
	} else {
		f.Class = strings.ToUpper(firstNonEmpty(k.Policy.BusinessType, in.Policy.BusinessType))
	}

	f.Reserve = FaceSheetReserve(coverage, currency)

	// Share ASM hanya dikenakan bila Sinar Mas leader (`CalculationCoasSpreadingSurvey`:
	// `@if(Local.leader=="true", sharePercentageASM/100, 1)`). Polis tanpa CoinsList
	// dianggap 100%.
	asm := PercentFull
	if c := k.Policy.Coinsurance; c.Role == "LEADER" && c.HasShare {
		asm = c.ShareASM
	}

	var facOut Percent
	for _, s := range coverage.Spreading {
		if !s.Removed && s.TreatyKind == TreatyFacOut {
			facOut += s.Share
		}
	}
	var facTotal Percent
	for _, r := range in.Reinsurer {
		if r.HasShare {
			facTotal += r.Share
		}
	}

	for _, reserve := range f.Reserve {
		base := reserve.Value.Share(asm)
		for _, s := range coverage.Spreading {
			if s.Removed {
				continue
			}
			f.Spreading = append(f.Spreading, FaceSheetShare{
				Name: strings.ToUpper(s.Name), Share: s.Share, Currency: reserve.Currency, Value: base.Share(s.Share),
			})
		}
		// Bagian tiap reasuradur fac out: share Fac Out dibagi menurut PctShareForAllObj.
		// Dengan satu reasuradur hasilnya sama dengan share Fac Out — seperti contoh PDF.
		if facOut > 0 && facTotal > 0 {
			for _, r := range in.Reinsurer {
				if !r.HasShare || r.Share <= 0 {
					continue
				}
				share := Percent(new(big.Int).Quo(
					new(big.Int).Mul(big.NewInt(int64(facOut)), big.NewInt(int64(r.Share))),
					big.NewInt(int64(facTotal))).Int64())
				f.Reinsurer = append(f.Reinsurer, FaceSheetShare{
					Name: r.Name, Share: share, Currency: reserve.Currency, Value: base.Share(share),
				})
			}
		}
		for _, c := range in.CoMember {
			if c.CoinsName == "" || !c.HasShare {
				continue
			}
			f.CoMember = append(f.CoMember, FaceSheetShare{
				Name: c.CoinsName, Share: c.PercentShare, Currency: reserve.Currency, Value: reserve.Value.Share(c.PercentShare),
			})
		}
	}
	return f
}

// FaceSheetReserve menjumlahkan estimasi klaim (tipe "1") sebuah jaminan per mata uang,
// menurut urutan kemunculannya — `CountTempOfEstimationBaseOnCurrency`. Nilai nol tidak
// dicetak (`$THIS.DeskripsiObject>'0'`).
func FaceSheetReserve(c Coverage, currency func(string) string) []FaceSheetAmount {
	var out []FaceSheetAmount
	index := map[string]int{}
	for _, item := range c.Item {
		for _, e := range item.Estimation {
			if e.Type != EstimateClaim {
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
	result := out[:0]
	for _, r := range out {
		if r.Value > 0 {
			result = append(result, r)
		}
	}
	return result
}

// HasUnprintedEstimate menyatakan apakah jaminan punya estimasi yang belum dibuatkan
// Claim Face Sheet — pemeriksaan "cek apakah ada estimasi baru" pada
// `CountTempOfEstimationBaseOnCurrency`.
func HasUnprintedEstimate(c Coverage) bool {
	for _, item := range c.Item {
		for _, e := range item.Estimation {
			if !e.FaceSheet {
				return true
			}
		}
	}
	return false
}

// LockEstimates menandai seluruh estimasi jaminan sudah dibuatkan Claim Face Sheet.
func LockEstimates(c *Coverage, at time.Time) {
	for i := range c.Item {
		for j := range c.Item[i].Estimation {
			e := &c.Item[i].Estimation[j]
			if !e.FaceSheet {
				e.FaceSheet = true
				e.FaceSheetDate = at
			}
		}
	}
}

// ErrFaceSheetNothingNew adalah pelanggaran saat seluruh estimasi sudah dibuatkan CFS.
func ErrFaceSheetNothingNew() error {
	return &ValidationError{Violation: []Violation{{
		Code: ViolationFaceSheetNothingNew, Field: "estimasi", Message: msgFaceSheetNothingNew,
	}}}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
