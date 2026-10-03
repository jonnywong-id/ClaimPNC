package registrasi

import (
	"strconv"
	"strings"
	"time"
)

// Receiver adalah satu penerima klaim — `.ClaimData.ReceiverClaim(n)`, disalin ke
// POOLDATA.T_CLAIM_RECEIVER (CLAIMID, IDRECEIVER, NAME, NAMEOFBANK, NOACCOUNT, ADDRESS).
type Receiver struct {
	ID        string // IDRECEIVER
	Name      string // NAME
	Address   string // ADDRESS
	BankName  string // NAMEOFBANK
	AccountNo string // NOACCOUNT
}

// DefaultReceiver adalah penerima yang dibentuk `InputRegister_act` saat Input Register
// disubmit: daftar ReceiverClaim dikosongkan (Page-Remove) lalu diisi satu penerima —
// Name = `.Policy.QQName`, Address = `.Policy.DeliveryAddressList(1).ASMAddress`,
// IDReceiver = "1". PaymentType "2" yang ikut diisi tidak punya kolom di T_CLAIM_RECEIVER.
//
// Penyimpangan yang disadari: bila QQName kosong, nama tertanggung (TheInsured) dipakai.
// Pega membiarkannya kosong — 50 baris T_CLAIM_RECEIVER tersimpan tanpa nama — dan QQName
// kosong pada sebagian besar dokumen polis terbaru.
func DefaultReceiver(p Policy) Receiver {
	name := strings.TrimSpace(p.QQName)
	if name == "" {
		name = strings.TrimSpace(p.InsuredName)
	}
	return Receiver{ID: "1", Name: name, Address: strings.TrimSpace(p.DeliveryAddress)}
}

// BankAccount adalah satu rekening Master Rekening (POOLDATA.LST_ACCOUNT) sebagaimana
// dibaca isian No Rekening penerima klaim (`Section/InputReceiver_sect.xml`).
//
// Pega mengisinya lewat activity `GetDataBankMaster(norekening)` — TIDAK ada di export.
// Pemetaannya direkonstruksi dari data: pada 362 penerima di JSON_KLAIM yang rekeningnya ada
// di master, NameOfBank sama 319/351, BranchOfBank 215/222, IDBank 208/210, Address 217/253,
// EmailReceiver 193/256, Telephone 128/152 (2026-09-28). Tanggal approve disalin saat
// rekening dipilih; master yang berubah sesudahnya membuat sebagian tidak lagi sama.
type BankAccount struct {
	Number    string // ACCOUNT_NO
	Name      string // ACCOUNT_NAME
	BankName  string // BANK_NAME
	Branch    string // BANK_BRANCH
	Address   string // BANK_ADDRESS
	BankID    string // BANKID
	Email     string // EMAIL
	Telephone string // TELP
	// Approval adalah APPROVAL: "1" disetujui, "0" sedang proses approval, "2" lainnya.
	// `GetDataPenerimaKlaim` hanya memakai rekening "1".
	Approval string

	// CashierApprovedAt dan CommitteeApprovedAt adalah TANGGALAPPROVEKASIR dan
	// TANGGALAPPROVEKOMITE; nol bila belum diisi.
	CashierApprovedAt   time.Time
	CommitteeApprovedAt time.Time
}

// AccountApproved menyatakan rekening dapat dipakai sebagai penerima klaim (APPROVAL = '1').
func (a BankAccount) AccountApproved() bool { return strings.TrimSpace(a.Approval) == "1" }

// ApplyAccount mengisi penerima dari rekening master: No Rekening, Nama, Nama Bank, dan
// Alamat — keempatnya baca-saja di InputReceiver, jadi hanya master yang mengisinya.
func (r *Receiver) ApplyAccount(a BankAccount) {
	r.AccountNo = strings.TrimSpace(a.Number)
	r.Name = strings.TrimSpace(a.Name)
	r.BankName = strings.TrimSpace(a.BankName)
	r.Address = strings.TrimSpace(a.Address)
}

// NextReceiverID adalah IDRECEIVER penerima baru (tombol Tambah): satu lebih besar dari
// IDRECEIVER angka terbesar. Pega menomori ReceiverClaim berurutan dari 1; T_CLAIM_RECEIVER
// memuat sampai 5 penerima per klaim.
func NextReceiverID(receivers []Receiver) string {
	highest := 0
	for _, r := range receivers {
		if n, err := strconv.Atoi(strings.TrimSpace(r.ID)); err == nil && n > highest {
			highest = n
		}
	}
	return strconv.Itoa(highest + 1)
}
