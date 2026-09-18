package notifikasi

import (
	"strings"
	"testing"

	"claim-pnc/internal/masterrekening"
)

// Uji di berkas ini memeriksa penyusunan surel, bukan pengirimannya. Penyusunannya
// sengaja dipisahkan dari pengiriman supaya dapat diuji tanpa server SMTP.

func peringatanContoh() masterrekening.Peringatan {
	return masterrekening.Peringatan{
		Rekening: masterrekening.Rekening{
			NomorRekening: "1234567890",
			NamaPemilik:   "BENGKEL CONTOH SEJAHTERA",
			NamaBank:      "BANK CONTOH",
			CabangBank:    "JAKARTA PUSAT",
			TipeRekening:  "BIASA",
		},
		Pesan:      "Rekening sudah terdaftar di Kasir.",
		Kode:       "9",
		Diputuskan: masterrekening.Penerima{Nama: "Komite Contoh", Email: "komite@sinarmas.id"},
	}
}

func TestKonfigurasiTanpaPenerimaBelumLengkap(t *testing.T) {
	// Pengirim surel tanpa tujuan bukan setengah aktif — ia tidak aktif. Menyatakannya
	// aktif akan menyembunyikan konfigurasi yang belum selesai di balik pengiriman yang
	// tidak pernah sampai ke siapa pun.
	k := Konfigurasi{Host: "smtp.internal", Port: 25, Dari: "claimpnc@sinarmas.id"}
	if k.Lengkap() {
		t.Error("konfigurasi tanpa penerima seharusnya belum lengkap")
	}

	k.Kepada = []string{"  "}
	if k.Lengkap() {
		t.Error("penerima yang hanya berisi spasi seharusnya tidak dihitung")
	}

	k.Kepada = []string{"timit@sinarmas.id"}
	if !k.Lengkap() {
		t.Error("konfigurasi dengan penerima seharusnya lengkap")
	}
}

func TestSurelDitujukanKeSeluruhMailboxTimIT(t *testing.T) {
	pesan := string(susunSurel(
		"claimpnc@sinarmas.id",
		[]string{"timit@sinarmas.id", "infra@sinarmas.id"},
		peringatanContoh(),
	))

	if !strings.Contains(pesan, "To: timit@sinarmas.id, infra@sinarmas.id\r\n") {
		t.Errorf("kedua alamat Tim IT seharusnya ada di header To; pesan:\n%s", pesan)
	}
	if !strings.Contains(pesan, "From: claimpnc@sinarmas.id\r\n") {
		t.Error("alamat pengirim tidak tertulis")
	}
	if !strings.Contains(pesan, "Content-Type: text/html; charset=UTF-8\r\n") {
		t.Error("badan surel seharusnya dinyatakan sebagai HTML UTF-8")
	}
}

func TestHeaderTidakDapatDisusupiBarisBaru(t *testing.T) {
	// Satu baris baru di dalam alamat cukup untuk menyisipkan header tambahan —
	// termasuk penerima tambahan — ke dalam surel yang dikirim aplikasi.
	pesan := string(susunSurel(
		"claimpnc@sinarmas.id\r\nBcc: penyusup@luar.example",
		[]string{"timit@sinarmas.id"},
		peringatanContoh(),
	))

	kepala, _, _ := strings.Cut(pesan, "\r\n\r\n")
	if strings.Contains(kepala, "Bcc:") {
		t.Errorf("header berhasil disusupi:\n%s", kepala)
	}
}

func TestSubjekMenyebutNomorRekeningYangGagal(t *testing.T) {
	// Tim IT menerima surel ini di antara surel lain; nomor rekeningnya harus terbaca
	// tanpa membuka isinya.
	subjek := Subjek(peringatanContoh())
	if !strings.Contains(subjek, "1234567890") {
		t.Errorf("subjek tidak menyebut nomor rekening: %q", subjek)
	}
	if !strings.Contains(strings.ToUpper(subjek), "GAGAL") {
		t.Errorf("subjek tidak menyatakan kegagalan: %q", subjek)
	}
}

func TestBadanMemuatSeluruhKeteranganTindakLanjut(t *testing.T) {
	badan := BadanHTML(peringatanContoh())

	wajib := []string{
		"1234567890",               // nomor rekening
		"BENGKEL CONTOH SEJAHTERA", // pemilik
		"BANK CONTOH",              // bank
		"Rekening sudah terdaftar", // pesan dari Kasir
		"Komite Contoh",            // siapa yang memutuskan
		"tidak dapat",              // akibatnya terhadap pembayaran
	}
	for _, teks := range wajib {
		if !strings.Contains(badan, teks) {
			t.Errorf("badan surel tidak memuat %q", teks)
		}
	}
}

func TestNilaiDariDataDiEscapeSebelumMasukHTML(t *testing.T) {
	// Nama pemilik rekening dan pesan dari Kasir adalah teks yang dimasukkan pihak
	// lain. Menempelkannya mentah berarti isi basis data dapat menyuntikkan markup ke
	// dalam kotak masuk penerimanya.
	p := peringatanContoh()
	p.Rekening.NamaPemilik = `<script>alert("x")</script>`
	p.Pesan = `<img src=x onerror="curi()">`

	badan := BadanHTML(p)

	if strings.Contains(badan, "<script>") {
		t.Error("nama pemilik tidak di-escape")
	}
	if strings.Contains(badan, "<img src=x") {
		t.Error("pesan dari Kasir tidak di-escape")
	}
	if !strings.Contains(badan, "&lt;script&gt;") {
		t.Error("nama pemilik seharusnya tetap terbaca dalam bentuk ter-escape")
	}
}

func TestBadanTetapTerbacaWalauKeteranganKosong(t *testing.T) {
	// Peringatan yang datanya bolong tetap harus terkirim dan terbaca; kolom kosong
	// ditandai, bukan dibiarkan menjadi baris tanpa isi.
	badan := BadanHTML(masterrekening.Peringatan{
		Rekening: masterrekening.Rekening{NomorRekening: "1"},
	})

	if !strings.Contains(badan, "—") {
		t.Error("kolom kosong seharusnya ditandai dengan em dash")
	}
	if !strings.Contains(badan, "Yth. Tim IT") {
		t.Error("sapaan ke Tim IT hilang")
	}
}

func TestPengirimanDitolakBilaSMTPBelumDikonfigurasi(t *testing.T) {
	// Galat, bukan diam. Konfigurasi yang belum selesai harus terlihat di log, bukan
	// berubah menjadi surel yang tidak pernah dikirim tanpa jejak.
	p := PengirimBaru(Konfigurasi{})
	if err := p.PeringatkanKegagalanKasir(t.Context(), peringatanContoh()); err == nil {
		t.Error("pengiriman tanpa konfigurasi seharusnya menghasilkan galat")
	}
}
