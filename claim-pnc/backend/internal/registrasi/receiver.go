package registrasi

import "strings"

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
