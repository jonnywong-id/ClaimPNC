package registrasi

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// # Preliminary Loss Advice (PLA) koasuransi
//
// Tombol "Print PLA" pada baris jaminan (`Section/ObjectCoverage_sect.xml`) membuka flow
// action `PrintPLA`, yang pra-prosesnya `GeneratePLAListObject` dan layarnya `PrintPLA_dtl`
// — keduanya TIDAK ada di export. Tombolnya mati bila `IsNoCoins || !isCFS`, sehingga PLA
// dari tombol ini adalah PLA KOASURANSI (huruf J).
//
// Aturannya dibaca dari sumber yang ada:
//   - penerima: padanan DLA-nya, `DLACoins_act` — PLA hanya bila Sinar Mas (atau Simas
//     Insurtech pada portal ASI) LEADER; setiap anggota lain dengan share > 0 dan tidak
//     bertanda hapus mendapat satu PLA; Sinar Mas sendiri tidak pernah (0 dari 942 baris
//     T_PLALIST COINS);
//   - nilai: `JSON_PLA.EstimasiList` pada T_PLALIST — ResultPLA = reserve × % anggota;
//   - nomor: `Database/PLA_DLA.prc` — KODE || TAHUN || ID_SITE || LPAD(PLA_SEQ,15,'0');
//   - catatan: `Database/INSERT_PLADLA.prc` — "Estimation only…" untuk PLA pertama kepada
//     penerima itu, "- Please see our PLA No.:… with DD: …" sesudahnya;
//   - dokumen: HTML `PLAHTML` dan empat contoh PDF dari Work Owner.

// PLATypeCoins adalah TIPEPLA untuk PLA koasuransi; PLACodeCoins hurufnya.
const (
	PLATypeCoins = "COINS"
	PLACodeCoins = "J"
)

// Catatan PLA — `INSERT_PLADLA.prc`.
const plaFirstNote = "Estimation only. Only when we have detailed information, we would revise accordingly."

// PLAPreviousNote menyusun catatan untuk PLA berikutnya kepada penerima yang sama.
func PLAPreviousNote(number string, date time.Time) string {
	return "- Please see our PLA No.:" + number + " with DD: " + date.Format("02/01/2006")
}

// PLANumber menyusun nomor PLA seperti `PLA_DLA.prc`.
func PLANumber(code string, year int, site string, counter int64) string {
	return fmt.Sprintf("%s%02d%s%015d", code, year%100, site, counter)
}

// PLACoinsMember adalah satu baris CoinsList polis yang dibaca untuk PLA.
type PLACoinsMember struct {
	ID       string // CoinsID — REINSCODE
	Name     string
	Leader   bool
	Share    Percent
	HasShare bool
	Deleted  bool // FlagDelete == "1"
}

// PLARecipientInfo adalah data penerima dari master T_REINSURER — disalin ke T_PLALIST
// seperti `INSERT_PLADLA.prc`.
type PLARecipientInfo struct {
	Login, Country, Email string
}

// PLAAmount adalah satu baris EstimasiList sebuah PLA.
type PLAAmount struct {
	Currency   string // nama, mis. IDR
	CurrencyID string // kode POOLDATA.CURRENCY
	Reserve    Money  // EstimastionReserve
	Base       Money  // EstimationValue — dasar perkalian
	Share      Percent
	Result     Money // ResultPLA
	ASMCount   Money // reserve × share ASM
}

// PLA adalah satu PLA untuk satu penerima.
type PLA struct {
	ClaimID        string
	ObjectID       string
	CoverageSeq    int
	Number         string
	Type           string
	Recipient      string
	RecipientCode  string
	Revision       int
	Date           time.Time
	Note           string
	PolicyCurrency string
	Amount         []PLAAmount
	Info           PLARecipientInfo
}

// PLAPrevious adalah PLA terakhir kepada seorang penerima pada klaim yang sama.
type PLAPrevious struct {
	Number string
	Date   time.Time
}

// PLADocument adalah isi satu dokumen PLA.
type PLADocument struct {
	PLA
	Entity          string // ASM atau ASI — kop, penanda tangan, dan alamat kaki
	BusinessName    string
	PolicyNumber    string
	ClaimNumber     string
	Insured         string
	Interest        string
	SumInsured      FaceSheetAmount
	PeriodStart     time.Time
	PeriodEnd       time.Time
	PolicyCondition string
	DateOfLoss      time.Time
	NatureOfLoss    string
	LossLocation    string
	Place           string
	SignerName      string
	Signature       []byte // PNG
}

// PLASource adalah seam ke data PLA: CoinsList, master penerima, nomor, dan T_PLALIST.
type PLASource interface {
	CoinsMembers(ctx context.Context, policyNumber, prodKe string) ([]PLACoinsMember, error)
	Recipient(ctx context.Context, code, name string) (PLARecipientInfo, error)
	Previous(ctx context.Context, claimID, recipientCode string) (PLAPrevious, bool, error)

	// Issued mengembalikan PLA yang sudah terbit untuk satu revisi CFS sebuah jaminan.
	Issued(ctx context.Context, claimID, objectID string, coverageSeq, revision int) ([]PLA, error)

	// NextNumber menerbitkan nomor PLA (PLA_SEQ + baris POOLDATA.PLA). Di dalam UnitOfWork.
	NextNumber(ctx context.Context, code string, year int) (string, error)
	// Save menulis satu baris T_PLALIST. Di dalam UnitOfWork.
	Save(ctx context.Context, p PLA) error
	// UpdateNote mengganti catatan PLA yang sudah terbit.
	UpdateNote(ctx context.Context, claimID, number string, revision int, note string) error

	// Signature membaca penanda tangan PLA sebuah entitas (POOLDATA.MTTD).
	Signature(ctx context.Context, entity string) (name string, png []byte, err error)

	// LODEmails membaca bahan isian Email LOD: email tertanggung dari pengkinian data
	// klaim dan email PIC teknik. Kosong bila tidak ada barisnya.
	LODEmails(ctx context.Context, claimNumber, technicalPIC string) (insured, pic string, err error)
}

// PLARenderer mengubah PLA menjadi dokumen yang diunduh.
type PLARenderer interface {
	Render(d PLADocument) ([]byte, error)
}

// Pesan aturan PLA.
const (
	msgPLANeedsFaceSheet = "PLA hanya dapat dicetak setelah Claim Face Sheet jaminan ini dibuat."
	msgPLANoCoins        = "Polis ini tidak berkoasuransi; tidak ada PLA koasuransi yang diterbitkan."
	msgPLANotLeader      = "PLA koasuransi hanya diterbitkan bila Sinar Mas menjadi leader koasuransi."
	msgPLANoReserve      = "Belum ada nilai estimasi klaim untuk dicantumkan pada PLA."
)

func plaViolation(code ViolationCode, message string) error {
	return &ValidationError{Violation: []Violation{{Code: code, Field: "pla", Message: message}}}
}

// OwnCompanyOf mengembalikan nama perusahaan sendiri pada CoinsList menurut portal.
func OwnCompanyOf(portal string) string {
	if strings.EqualFold(portal, "ASI") {
		return "ASURANSI SIMAS INSURTECH"
	}
	return OwnCompany
}

// CoinsPLARecipients memilih penerima PLA koasuransi — `DLACoins_act`.
func CoinsPLARecipients(portal string, members []PLACoinsMember) ([]PLACoinsMember, error) {
	own := OwnCompanyOf(portal)
	listed := false
	leader := false
	for _, m := range members {
		if m.ID == "" {
			continue
		}
		listed = true
		if m.Leader && strings.Contains(strings.ToUpper(m.Name), own) {
			leader = true
		}
	}
	if !listed {
		return nil, plaViolation(ViolationPLANoCoins, msgPLANoCoins)
	}
	if !leader {
		return nil, plaViolation(ViolationPLANotLeader, msgPLANotLeader)
	}
	var out []PLACoinsMember
	for _, m := range members {
		if m.ID == "" || m.Deleted || !m.HasShare || m.Share <= 0 {
			continue
		}
		if strings.Contains(strings.ToUpper(m.Name), own) {
			continue
		}
		out = append(out, m)
	}
	if len(out) == 0 {
		return nil, plaViolation(ViolationPLANoCoins, msgPLANoCoins)
	}
	return out, nil
}

// CoinsPLAAmounts menghitung EstimasiList PLA koasuransi seorang anggota: dasar = reserve
// penuh, hasil = reserve × % anggota; ASMCount = reserve × share ASM.
func CoinsPLAAmounts(reserve []FaceSheetAmount, currencyID map[string]string, member Percent, asm Percent) []PLAAmount {
	out := make([]PLAAmount, 0, len(reserve))
	for _, r := range reserve {
		out = append(out, PLAAmount{
			Currency: r.Currency, CurrencyID: currencyID[r.Currency],
			Reserve: r.Value, Base: r.Value, Share: member,
			Result: r.Value.Share(member), ASMCount: r.Value.Share(asm),
		})
	}
	return out
}

// PLANote memilih catatan PLA — `INSERT_PLADLA.prc`.
func PLANote(previous PLAPrevious, found bool) string {
	if found && previous.Number != "" {
		return PLAPreviousNote(previous.Number, previous.Date)
	}
	return plaFirstNote
}

// ErrPLANeedsFaceSheet adalah pelanggaran `!isCFS`.
func ErrPLANeedsFaceSheet() error {
	return plaViolation(ViolationPLANeedsFaceSheet, msgPLANeedsFaceSheet)
}

// ErrPLANoReserve adalah pelanggaran saat tidak ada reserve.
func ErrPLANoReserve() error { return plaViolation(ViolationPLANoReserve, msgPLANoReserve) }
