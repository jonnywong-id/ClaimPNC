package registrasi

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
)

// # Transfer Kasir
//
// Tombol "Transfer Kasir" pada baris adjustment (`Section/InputAdjustment_sect.xml`) membuka
// flow action `ValidasiTransferKasir_dialog` — TIDAK ada di export — yang menjalankan
// `TransferToKasir_act`: validasi, lalu `TransferCashierDataASM_act` mengirim data
// pembayaran ke sistem Kasir (Connect REST `SendDataPaidASMtoCashier_2`). Berhasil bila
// jawaban Kasir memuat CaseIDCashier atau NoTransClaim; baris adjustment lalu mendapat
// TransferCashierDate dan CaseIDCashier, dan Status Klaim menjadi 1162.
//
// Yang tidak dibawa, dan sebabnya:
//   - jalur persetujuan leader (FlagAtasan): hanya terpicu `.RemarkAcceptedLeader`, isian
//     yang tidak ada di aplikasi ini; `DokumentBeforeTFManager` dan SQL
//     `UpdateChasierIDTablePembayaran` juga tidak ada di export;
//   - dua pemeriksaan procedure (`PKG_KONVERSI_JSONKLAIM.Proteksi_PNC_TBI_Kasir`,
//     `Cek_Nilai_Akseptasi_PNC`): paketnya tidak ada di export, dan `D-02` melarang memanggil
//     procedure;
//   - pengiriman berkas ke Kasir (`SendAttachmenttoCashier_2`) — keputusan Work Owner; syarat
//     DLA sudah dicetak tetap diperiksa.

// StatusClaimTransferCashier adalah Status Klaim "Transfered to Cashier".
const StatusClaimTransferCashier ClaimStatus = "1162"

// Kode layanan Kasir pada POOLDATA.GCNM_CONNECT_REST (TYPESERVICE).
const (
	CashierServicePaid        = "KASIRPAID"
	CashierServicePaidSyariah = "KASIRPAIDSYARIAH"
	CashierServiceInvest      = "KASIRIVEST"
)

// cashierMaskingLimit adalah `FlagMaskingNoTelp`: klaim ≥ Rp 500.000.000 dikirim ke layanan
// KASIRIVEST.
var cashierMaskingLimit = big.NewRat(500_000_000, 1)

// Pesan Transfer Kasir — disalin apa adanya dari `TransferToKasir_act` langkah 3 dan
// `SendAttachmenttoCashier_act`; tambahan berbahasa Inggris (`D-80`).
const (
	msgCashierNoAcceptance = "Nomor Akseptasi Kosong"
	msgCashierNoAccount    = "Harap Isi No Rekening Terlebih Dahulu"
	msgCashierNoEmail      = "Harap Isi Email Terlebih Dahulu"
	msgCashierNoAcceptDate = "Tanggal Akseptasi Masih Kosong, Hubungi IT"
	msgCashierNoCurrency   = "Kurs Masih Kosong, Hubungi IT"
	msgCashierNoPayee      = "Tujuan Pembayaran Kosong"
	msgCashierNoNett       = "Nilai Nett Klaim Masih Kosong, Hubungi IT"
	msgCashierNoBank       = "Kode Bank Tidak Ditemukan di Daftar Bank"
	msgCashierNeedsDLA     = "Harap Print DLA Terlebih Dahulu"
	msgCashierTransferred  = "No Akseptasi Sudah Pernah Di Transfer Ke Kasir"
	msgCashierNotAllowed   = "Transfer Kasir is only available on an accepted adjustment that is not a salvage."
)

// Kode pelanggaran Transfer Kasir.
const (
	ViolationCashierNotAllowed  ViolationCode = "kasir_tidak_berlaku"
	ViolationCashierTransferred ViolationCode = "kasir_sudah_transfer"
	ViolationCashierIncomplete  ViolationCode = "kasir_data_kurang"
	ViolationCashierNeedsDLA    ViolationCode = "kasir_dla_belum_cetak"
)

// CashierTransferred menyatakan baris sudah ditransfer ke Kasir.
func (s SettlementLine) CashierTransferred() bool {
	return !s.CashierTransferredAt.IsZero() || strings.TrimSpace(s.CashierCaseID) != ""
}

// CanTransferCashier adalah aturan tombol: tampil bila bukan salvage dan Nomor Akseptasi
// terisi; mati bila sudah ditransfer.
func CanTransferCashier(line SettlementLine) error {
	if line.PaymentType == PaymentSalvage || strings.TrimSpace(line.AcceptedNo) == "" {
		return &ValidationError{Violation: []Violation{{Code: ViolationCashierNotAllowed, Field: "kasir", Message: msgCashierNotAllowed}}}
	}
	if line.CashierTransferred() {
		return &ValidationError{Violation: []Violation{{Code: ViolationCashierTransferred, Field: "kasir", Message: msgCashierTransferred}}}
	}
	return nil
}

// Tipe Transfer Kasir — property `JoinPlacement` (prompt list) pada dialog Transfer Pembayaran.
const (
	CashierTransferRegular       = "1" // Pembayaran Biasa (bawaan)
	CashierTransferJoinPlacement = "2" // Join Placement
	CashierTransferFronting      = "3" // Fronting
)

// CashierTransferTypes adalah pilihan "Tipe Transfer Kasir", urutan property `JoinPlacement`.
var CashierTransferTypes = []LODType{
	{CashierTransferRegular, "Pembayaran Biasa"},
	{CashierTransferJoinPlacement, "Join Placement"},
	{CashierTransferFronting, "Fronting"},
}

// CashierFacOut adalah satu baris tabel "Pilih Fac-out Tidak Dibayar" — `GetdataFacoutJoinPlacement`:
// DLA bertipe FAC OUT pada nomor akseptasi itu (No DLA, Nama Facout, Nilai Bayar).
type CashierFacOut struct {
	Number    string // NODLA — .KomunikasiID
	Reinsurer string // DLAREINSURER — .NameOfBank
	Value     string // NILAIDLA — .AdjusterFeeValue
	Currency  string
}

// CashierFacOuts memilih DLA FAC OUT dari daftar DLA adjustment.
func CashierFacOuts(list []DLA) []CashierFacOut {
	var out []CashierFacOut
	for _, d := range list {
		if strings.TrimSpace(d.Type) == DLATypeFacOut {
			out = append(out, CashierFacOut{Number: d.Number, Reinsurer: d.Recipient, Value: d.Value, Currency: d.Currency})
		}
	}
	return out
}

// Kode pelanggaran pilihan Fac-out.
const ViolationCashierFacOut ViolationCode = "kasir_fac_out"

// ChooseUnpaidFacOut memeriksa pilihan dialog: Tipe Transfer Kasir wajib salah satu pilihan;
// tabel Fac-out hanya berlaku untuk Join Placement dan Fronting (`.JoinPlacement==2 || ==3`),
// dan setiap nomor yang dicentang harus DLA FAC OUT adjustment itu.
func ChooseUnpaidFacOut(kind string, chosen []string, available []CashierFacOut) ([]CashierFacOut, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = CashierTransferRegular
	}
	if kind != CashierTransferRegular && kind != CashierTransferJoinPlacement && kind != CashierTransferFronting {
		return nil, &ValidationError{Violation: []Violation{{Code: ViolationCashierFacOut, Field: "tipe_transfer", Message: "Choose a Tipe Transfer Kasir."}}}
	}
	if kind == CashierTransferRegular {
		return nil, nil
	}
	byNumber := map[string]CashierFacOut{}
	for _, f := range available {
		byNumber[strings.TrimSpace(f.Number)] = f
	}
	var out []CashierFacOut
	for _, n := range chosen {
		f, ok := byNumber[strings.TrimSpace(n)]
		if !ok {
			return nil, &ValidationError{Violation: []Violation{{Code: ViolationCashierFacOut, Field: "fac_out_tidak_dibayar",
				Message: "DLA " + n + " is not a FAC OUT DLA of this adjustment."}}}
		}
		out = append(out, f)
	}
	return out, nil
}

// AddUnpaidFacOut menambahkan satu baris TAllPaymentData per Fac-out yang tidak dibayar —
// `TransferCashierDataASM_act` cabang ListOfPlacement (`.IsDLA=="true"`): baris yang sama
// dengan baris utama, NoTrans = No DLA dan Nett = Nilai Bayar × -1.
func AddUnpaidFacOut(p *CashierPayload, unpaid []CashierFacOut) {
	if len(p.TAllPaymentData) == 0 {
		return
	}
	base := p.TAllPaymentData[0]
	for _, f := range unpaid {
		row := base
		row.NoTrans = strings.TrimSpace(f.Number)
		v := decimalOf(f.Value)
		row.Nett = json.Number(new(big.Rat).Neg(v).FloatString(2))
		row.IsJurnalMemorial = ""
		p.TAllPaymentData = append(p.TAllPaymentData, row)
	}
}

// CashierConfirmation adalah kalimat konfirmasi dialog "Transfer Pembayaran" —
// `Pre_AlertTransferkasir`: TypeTransfer 2 (layanan investasi) → "INVESTMENT", selain itu
// "KASIR". Cabang TypeAtasan = 1 ("… DiTransfer Ke LEADER ?") milik jalur persetujuan atasan
// yang belum dibangun.
func CashierConfirmation(acceptedNo, service string) string {
	target := "KASIR"
	if service == CashierServiceInvest {
		target = "INVESTMENT"
	}
	return "Apakah Anda Yakin Akseptasi : " + strings.TrimSpace(acceptedNo) + " DiTransfer Ke " + target + "?"
}

// CashierReceiver adalah penerima pembayaran beserta data rekening masternya
// (`ReceiverClaim` diisi `GetDataBankMaster`).
type CashierReceiver struct {
	Name      string // .Name — PayableTo
	AccountNo string // .NoAccount
	BankName  string // .NameOfBank
	BankID    string // .IDBank
	Email     string // .EmailReceiver
	Telephone string // .Telephone
}

// CashierNett adalah nilai nett yang dibayarkan (`TransferToKasir_act` :11251–:12739): nilai
// adjustment (AdjustmentValue/AdjusterFeeValue); GrossValue bila perusahaan sendiri LEADER
// koasuransi pada polis bukan fakultatif masuk, kecuali TYPEOFCOINS "1".
func CashierNett(portal string, line SettlementLine, typeOfCoins string, coins []PLACoinsMember) Money {
	own := OwnCompanyOf(portal)
	if strings.TrimSpace(typeOfCoins) != "F" {
		for _, m := range coins {
			if strings.TrimSpace(m.ID) == "" || !strings.Contains(strings.ToUpper(m.Name), own) {
				continue
			}
			if m.Leader && strings.TrimSpace(typeOfCoins) != "1" {
				return line.Gross
			}
			break
		}
	}
	return line.Value
}

// CashierService memilih TYPESERVICE Kasir (`TransferToKasir_act` :9246,
// `TransferCashierDataASM_act` :3480).
func CashierService(portal string, syariah bool, nett Money, rate ExchangeRate) string {
	if syariah {
		return CashierServicePaidSyariah
	}
	claimShare := big.NewRat(int64(nett.Convert(rate)), 100)
	if !strings.EqualFold(portal, "ASI") && claimShare.Cmp(cashierMaskingLimit) >= 0 {
		return CashierServiceInvest
	}
	return CashierServicePaid
}

// CashierCompany adalah `TempGetApp.LSC_ID` (POOLDATA.DB_LINK_PEGA) portal itu.
func CashierCompany(portal string) string {
	if strings.EqualFold(portal, "ASI") {
		return "SIMASNET"
	}
	if strings.TrimSpace(portal) == "" {
		return "ASM"
	}
	return strings.ToUpper(strings.TrimSpace(portal))
}

// CashierCheck adalah bahan validasi `TransferToKasir_act`, sesuai urutan langkahnya.
type CashierCheck struct {
	Line       SettlementLine
	Receiver   CashierReceiver
	Nett       Money
	BankFound  bool
	DLAPrinted bool // seluruh DLA revisi 0 adjustment itu ISDLA = '1'
}

// ValidateCashier menjalankan validasi Transfer Kasir. Pega berhenti pada galat pertama
// (lompat ke ERR); urutannya dipertahankan.
func ValidateCashier(c CashierCheck) error {
	fail := func(code ViolationCode, message string) error {
		return &ValidationError{Violation: []Violation{{Code: code, Field: "kasir", Message: message}}}
	}
	switch {
	case strings.TrimSpace(c.Line.AcceptedNo) == "":
		return fail(ViolationCashierIncomplete, msgCashierNoAcceptance)
	case strings.TrimSpace(c.Receiver.AccountNo) == "":
		return fail(ViolationCashierIncomplete, msgCashierNoAccount)
	case strings.TrimSpace(c.Receiver.Email) == "":
		return fail(ViolationCashierIncomplete, msgCashierNoEmail)
	case c.Line.Acceptance.AcceptedAt.IsZero():
		return fail(ViolationCashierIncomplete, msgCashierNoAcceptDate)
	case strings.TrimSpace(c.Line.Currency) == "":
		return fail(ViolationCashierIncomplete, msgCashierNoCurrency)
	case strings.TrimSpace(c.Receiver.Name) == "":
		return fail(ViolationCashierIncomplete, msgCashierNoPayee)
	case c.Nett == 0:
		return fail(ViolationCashierIncomplete, msgCashierNoNett)
	case !c.BankFound:
		return fail(ViolationCashierIncomplete, msgCashierNoBank)
	case !c.DLAPrinted:
		return fail(ViolationCashierNeedsDLA, msgCashierNeedsDLA)
	}
	return nil
}

// CashierPayment adalah satu baris TAllPaymentData — nama field sama dengan halaman
// BRISurfPNC yang diserialisasi `SetJSONPage` (`TransferCashierDataASM_act` langkah 14).
//
// # Bentuknya mengikuti badan yang DITERIMA Kasir, bukan susunan activity
//
// Badan yang dicatat Pega di POOLDATA.CLAIM_SERVICE_LOG (jenis "Log Kasir", dijawab
// "Success") memuat Nett, Deductible, dan KaliDeduct sebagai ANGKA JSON, dan tidak memuat DOL
// maupun Panel — meski `TransferCashierDataASM_act` mengisi keduanya pada halaman. Kasir
// menolak (HTTP 400) badan yang mengirim ketiga angka itu sebagai teks.
type CashierPayment struct {
	NoTrans          string      `json:"NoTrans"`
	LbuId            string      `json:"LbuId"`
	LdcId            string      `json:"LdcId"`
	AccountNo        string      `json:"AccountNo"`
	LbgID            string      `json:"LbgID"`
	TglAksep         string      `json:"TglAksep"`
	TglLOD           string      `json:"TglLOD"`
	TglBolehBayar    string      `json:"TglBolehBayar"`
	Nett             json.Number `json:"Nett"`
	LkuId            string      `json:"LkuId"`
	LjtdId           string      `json:"LjtdId"`
	NoKlaim          string      `json:"NoKlaim"`
	AcceptType       string      `json:"AcceptType"`
	Deductible       json.Number `json:"Deductible"`
	KaliDeduct       json.Number `json:"KaliDeduct"`
	Kepada           string      `json:"Kepada"`
	Email            string      `json:"Email"`
	StsSyariah       string      `json:"StsSyariah"`
	StsDsa           string      `json:"StsDsa"`
	CompanyName      string      `json:"CompanyName"`
	NoPolis          string      `json:"NoPolis"`
	Note             string      `json:"Note"`
	StsAp            string      `json:"StsAp"`
	NoHP             string      `json:"NoHP"`
	DOL              string      `json:"-"` // diisi activity, tidak sampai ke Kasir
	Panel            string      `json:"-"` // idem
	UserInput        string      `json:"UserInput"`
	IsJurnalMemorial string      `json:"IsJurnalMemorial,omitempty"`
}

// CashierPayload adalah badan permintaan ke Kasir.
type CashierPayload struct {
	TAllPaymentData []CashierPayment `json:"TAllPaymentData"`
}

// CashierFacts adalah bahan muatan dari klaim dan polis.
type CashierFacts struct {
	Portal       string
	ClaimNumber  string
	PolicyNumber string
	BusinessCode string
	BranchCode   string
	DateOfLoss   time.Time
	CauseOfLoss  string
	Syariah      bool
	ExGratia     bool
	User         string
	PICEmail     string
	BankGroupID  string
}

// CashierDate adalah bentuk tanggal Kasir: DD-MM-YYYY (WIB), dipotong dari DateTime Pega.
func CashierDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(clock.ZoneWIB).Format("02-01-2006")
}

// BuildCashierPayload menyusun muatan satu adjustment. DLA placement tidak ikut: halaman
// `ListFacoutFlonting` yang mengisinya tidak pernah dimuat activity itu.
func BuildCashierPayload(line SettlementLine, r CashierReceiver, nett Money, f CashierFacts, now time.Time) CashierPayload {
	email := strings.TrimSpace(r.Email)
	if pic := strings.TrimSpace(f.PICEmail); pic != "" {
		email += ";" + pic
	}
	stsDsa := ""
	if f.ExGratia {
		stsDsa = "1"
	}
	syariah := "0"
	if f.Syariah {
		syariah = "1"
	}
	p := CashierPayment{
		NoTrans: line.AcceptedNo, LbuId: f.BusinessCode, LdcId: f.BranchCode, AccountNo: r.AccountNo,
		LbgID: f.BankGroupID, TglAksep: CashierDate(line.Acceptance.AcceptedAt),
		TglLOD: CashierDate(line.Acceptance.Form.ReceiveDate), TglBolehBayar: CashierDate(now),
		Nett: json.Number(big.NewRat(int64(nett), 100).FloatString(2)), LkuId: line.Currency, LjtdId: "D0031",
		NoKlaim: f.ClaimNumber, AcceptType: line.PaymentType,
		Deductible: json.Number(big.NewRat(int64(line.RiskValue), 100).FloatString(2)),
		KaliDeduct: json.Number(plainDecimal(percentRat(line.RiskPercent))),
		Kepada:     r.Name, Email: strings.ReplaceAll(email, ",", ";"),
		StsSyariah: syariah, StsDsa: stsDsa, CompanyName: CashierCompany(f.Portal), NoPolis: f.PolicyNumber,
		Note: line.Acceptance.Form.MinutesNote, StsAp: "0", NoHP: r.Telephone,
		DOL: CashierDate(f.DateOfLoss), Panel: f.CauseOfLoss, UserInput: f.User,
	}
	if nett < 0 {
		p.IsJurnalMemorial = "1"
	}
	return CashierPayload{TAllPaymentData: []CashierPayment{p}}
}

// CashierReply adalah jawaban Kasir (`ReturnPaymentChas`).
type CashierReply struct {
	ResponseMessage string
	CaseIDCashier   string
	NoTransClaim    string
	Raw             string // dipotong 500 karakter — bahan pesan
	Body            string // jawaban utuh — JSONOUT log layanan
}

// Accepted menyatakan Kasir menerima pembayaran (`TransferToKasir_act` :14863).
func (r CashierReply) Accepted() bool {
	return strings.TrimSpace(r.CaseIDCashier) != "" || strings.TrimSpace(r.NoTransClaim) != ""
}

// CaseID adalah CaseIDCashier, atau NoTransClaim bila kosong.
func (r CashierReply) CaseID() string {
	if c := strings.TrimSpace(r.CaseIDCashier); c != "" {
		return c
	}
	return strings.TrimSpace(r.NoTransClaim)
}

// Logged menyatakan jawaban yang dicatat ke TRF_KASIR_LOG (pesan memuat SUCCESS).
func (r CashierReply) Logged() bool {
	return strings.Contains(strings.ToUpper(r.ResponseMessage), "SUCCESS")
}

// CashierLog adalah satu baris POOLDATA.TRF_KASIR_LOG (`InsertLogKasir_sql`).
type CashierLog struct {
	AcceptedNo  string
	ClaimNumber string
	PIC         string
	Status      string // TypePaid: "2" transfer langsung
	Reason      string
}

// CashierLogTransfer adalah isian log jalur transfer langsung.
const (
	CashierLogStatusTransfer = "2"
	CashierLogReasonTransfer = "Sedang Transfer Kasir"
)

// CashierGateway adalah seam ke sistem Kasir.
type CashierGateway interface {
	Transfer(ctx context.Context, app, service string, payload CashierPayload) (CashierReply, error)
}

// CashierStore adalah seam ke data Transfer Kasir.
type CashierStore interface {
	// BankGroupID mencari LBG_ID GENERAL.LST_BANK_GROUP untuk nama bank, lalu memilih yang
	// sama dengan IDBank penerima.
	BankGroupID(ctx context.Context, bankName, bankID string) (string, bool, error)
	// Log menulis TRF_KASIR_LOG.
	Log(ctx context.Context, entry CashierLog) error
	// CashierHistory membaca POOLDATA.TRF_KASIR_LOG satu nomor akseptasi, urut waktu.
	CashierHistory(ctx context.Context, acceptedNo string) ([]CashierHistoryEntry, error)
	// MarkTransferred mengisi TRANSFER_CASHIER_DATE (bila kosong) dan IDCHASIER.
	MarkTransferred(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, at time.Time, caseID string) error
	// AccountRegistered menyatakan rekening terdaftar aktif di master rekening Kasir
	// (COLLECTION.LST_ACCOUNT@ASMD, berdasarkan ACCOUNT_NO dan LBG_ID).
	AccountRegistered(ctx context.Context, accountNo, bankID string) (bool, error)
	// LogService menulis satu baris POOLDATA.CLAIM_SERVICE_LOG jenis "Log Kasir".
	LogService(ctx context.Context, entry CashierServiceLog) error
}

// CashierServiceLogCategory adalah CATEGORYSERVICE log Kasir (`TransferCashierDataASM_act`:
// Param.jenis := "Log Kasir").
const CashierServiceLogCategory = "Log Kasir"

// CashierServiceLog adalah satu baris POOLDATA.CLAIM_SERVICE_LOG: SERVICEID = nomor klaim
// (pyID), SERVICEREF = nomor akseptasi, JSONIN = badan yang dikirim, JSONOUT = jawaban Kasir
// (atau galatnya bila Kasir tidak menjawab).
//
// Pega menyisipkan JSONIN lebih dulu lalu meng-UPDATE JSONOUT (`InsertUpdateLogService`,
// `UpdateLogServiceClaim`). Di sini satu INSERT setelah Kasir menjawab: Steering §8 menolak
// UPDATE pada tabel log ini, dan isi akhirnya sama.
type CashierServiceLog struct {
	ClaimNumber string
	AcceptedNo  string
	Request     CashierPayload
	Response    string
}

// CashierHistoryEntry adalah satu baris grid "Histori Transfer Kasir" Section/
// InputAdjustment_sect.xml (TempDataLogKasir.pxResults): PIC Teknik (.KomiteAccepted),
// Tanggal Transfer/Reject (.TransferCashierDate), Status Kasir (.StatusBayar), Komentar
// (.Remarks) — kolom PIC, TGL_TRF, STATUS, ALASAN pada POOLDATA.TRF_KASIR_LOG.
type CashierHistoryEntry struct {
	PIC    string
	At     time.Time
	Status string
	Reason string
}
