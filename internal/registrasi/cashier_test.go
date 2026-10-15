package registrasi

import (
	"testing"
	"time"
)

// Tombol: bukan salvage, sudah bernomor akseptasi, dan belum pernah ditransfer.
func TestCanTransferCashier(t *testing.T) {
	ok := SettlementLine{PaymentType: PaymentFinal, AcceptedNo: "A26"}
	if err := CanTransferCashier(ok); err != nil {
		t.Fatalf("baris sah ditolak: %v", err)
	}
	for name, line := range map[string]SettlementLine{
		"salvage":         {PaymentType: PaymentSalvage, AcceptedNo: "A26"},
		"belum akseptasi": {PaymentType: PaymentFinal},
		"sudah transfer":  {PaymentType: PaymentFinal, AcceptedNo: "A26", CashierTransferredAt: time.Now()},
		"sudah case id":   {PaymentType: PaymentFinal, AcceptedNo: "A26", CashierCaseID: "ECR-1"},
	} {
		if err := CanTransferCashier(line); err == nil {
			t.Errorf("%s: harus ditolak", name)
		}
	}
}

// Nett: nilai adjustment, atau GrossValue bila kita leader koasuransi (kecuali TYPEOFCOINS 1).
func TestCashierNett(t *testing.T) {
	line := SettlementLine{Gross: Rupiah(1000), Value: Rupiah(600)}
	leader := []PLACoinsMember{{ID: "1", Name: "PT ASURANSI SINAR MAS", Leader: true}}
	if got := CashierNett("ASM", line, "2", leader); got != Rupiah(1000) {
		t.Fatalf("leader = %d", got)
	}
	if got := CashierNett("ASM", line, "1", leader); got != Rupiah(600) {
		t.Fatalf("TYPEOFCOINS 1 = %d", got)
	}
	if got := CashierNett("ASM", line, "F", leader); got != Rupiah(600) {
		t.Fatalf("fac in = %d", got)
	}
	if got := CashierNett("ASM", line, "2", nil); got != Rupiah(600) {
		t.Fatalf("tanpa koasuransi = %d", got)
	}
}

// Layanan: syariah, lalu KASIRIVEST untuk klaim >= Rp 500 juta (bukan portal ASI).
func TestCashierService(t *testing.T) {
	if got := CashierService("ASM", true, Rupiah(1), ExchangeRate(10_000)); got != CashierServicePaidSyariah {
		t.Fatalf("syariah = %s", got)
	}
	if got := CashierService("ASM", false, Rupiah(500_000_000), ExchangeRate(10_000)); got != CashierServiceInvest {
		t.Fatalf("besar = %s", got)
	}
	if got := CashierService("ASI", false, Rupiah(500_000_000), ExchangeRate(10_000)); got != CashierServicePaid {
		t.Fatalf("ASI = %s", got)
	}
	if got := CashierService("ASM", false, Rupiah(1_000), ExchangeRate(10_000)); got != CashierServicePaid {
		t.Fatalf("biasa = %s", got)
	}
}

// Urutan validasi mengikuti TransferToKasir_act.
func TestValidateCashierOrder(t *testing.T) {
	full := CashierCheck{
		Line:     SettlementLine{AcceptedNo: "A26", Currency: "IDR", Acceptance: Acceptance{AcceptedAt: time.Now()}},
		Receiver: CashierReceiver{Name: "X", AccountNo: "1", Email: "e", BankName: "B"},
		Nett:     Rupiah(1), BankFound: true, DLAPrinted: true,
	}
	if err := ValidateCashier(full); err != nil {
		t.Fatalf("lengkap ditolak: %v", err)
	}
	c := full
	c.Receiver.AccountNo, c.Receiver.Email = "", ""
	if msg := firstMessage(ValidateCashier(c)); msg != "Harap Isi No Rekening Terlebih Dahulu" {
		t.Fatalf("rekening lebih dulu, dapat %q", msg)
	}
	c = full
	c.BankFound = false
	if msg := firstMessage(ValidateCashier(c)); msg != "Kode Bank Tidak Ditemukan di Daftar Bank" {
		t.Fatalf("bank = %q", msg)
	}
	c = full
	c.DLAPrinted = false
	if msg := firstMessage(ValidateCashier(c)); msg != "Harap Print DLA Terlebih Dahulu" {
		t.Fatalf("DLA = %q", msg)
	}
}

func firstMessage(err error) string {
	v, ok := err.(*ValidationError)
	if !ok {
		return ""
	}
	p, _ := v.First()
	return p.Message
}

// Muatan: tanggal DD-MM-YYYY, email berpemisah titik koma, jurnal memorial bila nett negatif.
func TestBuildCashierPayload(t *testing.T) {
	day := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	line := SettlementLine{AcceptedNo: "A26", Currency: "10001", PaymentType: PaymentFinal,
		Acceptance: Acceptance{AcceptedAt: day}}
	p := BuildCashierPayload(line, CashierReceiver{Name: "X", AccountNo: "1", Email: "a@x,b@x"}, Rupiah(-5),
		CashierFacts{Portal: "ASI", PICEmail: "c@x", ClaimNumber: "PNCN.26.0001"}, day).TAllPaymentData[0]
	if p.TglAksep != "30-09-2026" || p.Email != "a@x;b@x;c@x" || p.Nett != "-5.00" || p.IsJurnalMemorial != "1" ||
		p.CompanyName != "SIMASNET" || p.NoKlaim != "PNCN.26.0001" {
		t.Fatalf("muatan salah: %+v", p)
	}
}

// Kalimat Pre_AlertTransferkasir: investasi → INVESTMENT, selain itu KASIR.
func TestCashierConfirmation(t *testing.T) {
	if got := CashierConfirmation("A26", CashierServicePaid); got != "Apakah Anda Yakin Akseptasi : A26 DiTransfer Ke KASIR?" {
		t.Fatalf("kasir = %q", got)
	}
	if got := CashierConfirmation("A26", CashierServiceInvest); got != "Apakah Anda Yakin Akseptasi : A26 DiTransfer Ke INVESTMENT?" {
		t.Fatalf("investasi = %q", got)
	}
}

// Fac-out "tidak dibayar": hanya Join Placement/Fronting, hanya DLA FAC OUT adjustment itu,
// dan tiap pilihan menjadi baris tambahan bernilai negatif.
func TestUnpaidFacOut(t *testing.T) {
	available := CashierFacOuts([]DLA{
		{Number: "H26-1", Type: DLATypeFacOut, Recipient: "REAS A", Value: "1500.25"},
		{Number: "J26-1", Type: DLATypeCoins, Recipient: "KOAS B", Value: "10"},
	})
	if len(available) != 1 {
		t.Fatalf("FAC OUT = %+v", available)
	}
	if got, err := ChooseUnpaidFacOut("1", []string{"H26-1"}, available); err != nil || got != nil {
		t.Fatalf("Pembayaran Biasa mengabaikan centang: %v %v", got, err)
	}
	if _, err := ChooseUnpaidFacOut("2", []string{"J26-1"}, available); err == nil {
		t.Fatal("DLA COINS tidak boleh dipilih")
	}
	got, err := ChooseUnpaidFacOut("3", []string{"H26-1"}, available)
	if err != nil || len(got) != 1 {
		t.Fatalf("Fronting: %v %v", got, err)
	}
	p := CashierPayload{TAllPaymentData: []CashierPayment{{NoTrans: "A26", Nett: "100.00", AccountNo: "1"}}}
	AddUnpaidFacOut(&p, got)
	if len(p.TAllPaymentData) != 2 {
		t.Fatalf("baris = %d", len(p.TAllPaymentData))
	}
	row := p.TAllPaymentData[1]
	if row.NoTrans != "H26-1" || row.Nett != "-1500.25" || row.AccountNo != "1" {
		t.Fatalf("baris fac-out = %+v", row)
	}
}
