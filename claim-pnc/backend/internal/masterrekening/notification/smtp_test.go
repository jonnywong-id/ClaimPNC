package notification

import (
	"strings"
	"testing"

	"claim-pnc/internal/masterrekening"
)

// Uji di berkas ini memeriksa penyusunan surel, bukan pengirimannya. Penyusunannya
// sengaja dipisahkan dari pengiriman supaya dapat diuji tanpa server SMTP.

func sampleAlert() masterrekening.Alert {
	return masterrekening.Alert{
		Account: masterrekening.Account{
			Number:      "1234567890",
			OwnerName:   "BENGKEL CONTOH SEJAHTERA",
			BankName:    "BANK CONTOH",
			BankBranch:  "JAKARTA PUSAT",
			AccountType: "BIASA",
		},
		Message:   "Account sudah terdaftar di Cashier.",
		Code:      "9",
		DecidedBy: masterrekening.Recipients{Name: "Committee Contoh", Email: "komite@sinarmas.id"},
	}
}

func TestConfigWithoutRecipientsIsIncomplete(t *testing.T) {
	// Pengirim surel tanpa tujuan bukan setengah aktif — ia tidak aktif. Menyatakannya
	// aktif akan menyembunyikan konfigurasi yang belum selesai di balik pengiriman yang
	// tidak pernah sampai ke siapa pun.
	k := Config{Host: "smtp.internal", Port: 25, From: "claimpnc@sinarmas.id"}
	if k.Complete() {
		t.Error("konfigurasi tanpa penerima seharusnya belum lengkap")
	}

	k.To = []string{"  "}
	if k.Complete() {
		t.Error("penerima yang hanya berisi spasi seharusnya tidak dihitung")
	}

	k.To = []string{"timit@sinarmas.id"}
	if !k.Complete() {
		t.Error("konfigurasi dengan penerima seharusnya lengkap")
	}
}

func TestEmailAddressedToAllITTeamMailboxes(t *testing.T) {
	message := string(composeEmail(
		"claimpnc@sinarmas.id",
		[]string{"timit@sinarmas.id", "infra@sinarmas.id"},
		sampleAlert(),
	))

	if !strings.Contains(message, "To: timit@sinarmas.id, infra@sinarmas.id\r\n") {
		t.Errorf("kedua alamat Tim IT seharusnya ada di header To; pesan:\n%s", message)
	}
	if !strings.Contains(message, "From: claimpnc@sinarmas.id\r\n") {
		t.Error("alamat pengirim tidak tertulis")
	}
	if !strings.Contains(message, "Content-Type: text/html; charset=UTF-8\r\n") {
		t.Error("badan surel seharusnya dinyatakan sebagai HTML UTF-8")
	}
}

func TestHeaderCannotBeInjectedWithNewline(t *testing.T) {
	// Satu baris baru di dalam alamat cukup untuk menyisipkan header tambahan —
	// termasuk penerima tambahan — ke dalam surel yang dikirim aplikasi.
	message := string(composeEmail(
		"claimpnc@sinarmas.id\r\nBcc: penyusup@luar.example",
		[]string{"timit@sinarmas.id"},
		sampleAlert(),
	))

	kepala, _, _ := strings.Cut(message, "\r\n\r\n")
	if strings.Contains(kepala, "Bcc:") {
		t.Errorf("header berhasil disusupi:\n%s", kepala)
	}
}

func TestSubjectNamesTheFailingAccountNumber(t *testing.T) {
	// Tim IT menerima surel ini di antara surel lain; nomor rekeningnya harus terbaca
	// tanpa membuka isinya.
	subjek := Subject(sampleAlert())
	if !strings.Contains(subjek, "1234567890") {
		t.Errorf("subjek tidak menyebut nomor rekening: %q", subjek)
	}
	if !strings.Contains(strings.ToUpper(subjek), "GAGAL") {
		t.Errorf("subjek tidak menyatakan kegagalan: %q", subjek)
	}
}

func TestBodyCarriesAllFollowUpDetails(t *testing.T) {
	body := HTMLBody(sampleAlert())

	wajib := []string{
		"1234567890",               // nomor rekening
		"BENGKEL CONTOH SEJAHTERA", // pemilik
		"BANK CONTOH",              // bank
		"Account sudah terdaftar",  // pesan dari Cashier
		"Committee Contoh",         // siapa yang memutuskan
		"tidak dapat",              // akibatnya terhadap pembayaran
	}
	for _, text := range wajib {
		if !strings.Contains(body, text) {
			t.Errorf("badan surel tidak memuat %q", text)
		}
	}
}

func TestDataValuesAreEscapedBeforeEnteringHTML(t *testing.T) {
	// Nama pemilik rekening dan pesan dari Kasir adalah teks yang dimasukkan pihak
	// lain. Menempelkannya mentah berarti isi basis data dapat menyuntikkan markup ke
	// dalam kotak masuk penerimanya.
	p := sampleAlert()
	p.Account.OwnerName = `<script>alert("x")</script>`
	p.Message = `<img src=x onerror="curi()">`

	body := HTMLBody(p)

	if strings.Contains(body, "<script>") {
		t.Error("nama pemilik tidak di-escape")
	}
	if strings.Contains(body, "<img src=x") {
		t.Error("pesan dari Cashier tidak di-escape")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Error("nama pemilik seharusnya tetap terbaca dalam bentuk ter-escape")
	}
}

func TestBodyStaysReadableEvenWithEmptyDetails(t *testing.T) {
	// Peringatan yang datanya bolong tetap harus terkirim dan terbaca; kolom kosong
	// ditandai, bukan dibiarkan menjadi baris tanpa isi.
	body := HTMLBody(masterrekening.Alert{
		Account: masterrekening.Account{Number: "1"},
	})

	if !strings.Contains(body, "—") {
		t.Error("kolom kosong seharusnya ditandai dengan em dash")
	}
	if !strings.Contains(body, "Yth. Tim IT") {
		t.Error("sapaan ke Tim IT hilang")
	}
}

func TestSendingRejectedWhenSMTPNotConfigured(t *testing.T) {
	// Galat, bukan diam. Konfigurasi yang belum selesai harus terlihat di log, bukan
	// berubah menjadi surel yang tidak pernah dikirim tanpa jejak.
	p := NewSender(Config{})
	if err := p.WarnCashierFailure(t.Context(), sampleAlert()); err == nil {
		t.Error("pengiriman tanpa konfigurasi seharusnya menghasilkan galat")
	}
}
