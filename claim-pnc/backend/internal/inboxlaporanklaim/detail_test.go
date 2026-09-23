package inboxlaporanklaim_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"claim-pnc/internal/inboxlaporanklaim"
)

func TestFormCarriesEveryFieldItCollects(t *testing.T) {
	// Berkas yang isiannya dibaca lalu dituliskan kembali harus utuh. Satu field yang
	// terlewat di DetailOf atau di Apply tidak menghasilkan galat — ia hanya membuat
	// isian itu hilang setiap kali petugas menekan Simpan.
	received := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	loss := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)

	filled := inboxlaporanklaim.Detail{
		ReceivedDate:      received,
		DateOfLoss:        loss,
		ReporterName:      "Pelapor Contoh",
		ReporterEmail:     "pelapor@example.invalid",
		ReporterPhone:     "0800000000",
		CourierName:       "Kurir Contoh",
		PolicyNumber:      "POL-CONTOH-1",
		InsuredName:       "Tertanggung Contoh",
		BusinessName:      "Aneka",
		ReferenceNumber:   "REF-CONTOH-1",
		EstimateValue:     inboxlaporanklaim.Rupiah(2_500_000),
		LossLocation:      "Gudang Contoh",
		EmailSubject:      "Laporan kerugian",
		Chronology:        "Kronologis contoh.",
		DamageDetail:      "Rincian contoh.",
		Reason:            "Menunggu dokumen.",
		NotRegisteredNote: "Polis belum ditemukan.",
		DocumentCount:     7,
	}

	roundTrip := inboxlaporanklaim.DetailOf(filled.Apply(inboxlaporanklaim.ClaimReport{}))
	if roundTrip != filled {
		t.Fatalf("isian berubah setelah ditulis lalu dibaca ulang:\n  ingin %+v\n  dapat %+v", filled, roundTrip)
	}
}

func TestFormNeverTouchesTheReportHeader(t *testing.T) {
	// Form MENGISI berkas; ia tidak memindahkannya. Keenam nilai di bawah adalah kepala
	// berkas, dan seluruhnya harus lolos tanpa tersentuh.
	before := inboxlaporanklaim.ClaimReport{
		ID:          "RCVN.26.0001",
		ClaimNumber: "PNC-1801",
		BranchCode:  "1001",
		Transferred: true,
		Origin:      inboxlaporanklaim.OriginNew,
		CreatedBy:   "adminpnc",
		CreatedAt:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	after := inboxlaporanklaim.Detail{ReporterName: "Pelapor Baru"}.Apply(before)

	switch {
	case after.ID != before.ID:
		t.Fatalf("nomor berkas berubah menjadi %q", after.ID)
	case after.ClaimNumber != before.ClaimNumber:
		t.Fatalf("nomor klaim berubah menjadi %q", after.ClaimNumber)
	case after.BranchCode != before.BranchCode:
		t.Fatalf("cabang berubah menjadi %q — itu batas data", after.BranchCode)
	case after.Transferred != before.Transferred:
		t.Fatal("penanda diserahkan berubah; perpindahan tahap bukan tindakan form")
	case after.Origin != before.Origin:
		t.Fatalf("asal berkas berubah menjadi %q", after.Origin)
	case after.CreatedBy != before.CreatedBy || !after.CreatedAt.Equal(before.CreatedAt):
		t.Fatal("jejak pembuatan ditulis ulang")
	}

	if after.ReporterName != "Pelapor Baru" {
		t.Fatalf("isian justru tidak tersimpan: nama pelapor = %q", after.ReporterName)
	}
}

func TestFormAcceptsEverythingEmptyBecausePegaDoesToo(t *testing.T) {
	// Flow action `InputReceiveDocument` tidak punya satu pun validate rule maupun isian
	// bertanda wajib — diperiksa langsung ke berkasnya. Berkas laporan memang dapat masuk
	// sebelum isinya diketahui, dan menolak form kosong akan menolak berkas yang di Pega
	// diterima (`P-5`).
	if err := (inboxlaporanklaim.Detail{}).Clean().Check(); err != nil {
		t.Fatalf("form kosong ditolak: %v", err)
	}
}

func TestFormHasNoDateOrderRuleBecauseThatBelongsToRegistration(t *testing.T) {
	// Aturan "Tanggal Lapor <= DOL + 7 hari" dan saudara-saudaranya milik `B-2`
	// (`02-BUSINESS-UNDERSTANDING.md` §3.1). Berkas laporan justru sering masuk sebelum
	// tanggal kejadiannya dipastikan, dan urutan yang aneh di sini bukan alasan menolak.
	backwards := inboxlaporanklaim.Detail{
		ReceivedDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		DateOfLoss:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	}
	if err := backwards.Clean().Check(); err != nil {
		t.Fatalf("urutan tanggal ditolak, padahal aturannya milik B-2: %v", err)
	}
}

func TestFormTrimsBeforeItChecks(t *testing.T) {
	// Isian sepanjang batas ditambah spasi harus LOLOS: yang tersimpan adalah nilai yang
	// sudah dipangkas. Memeriksa sebelum memangkas akan menolaknya.
	atLimit := strings.Repeat("a", inboxlaporanklaim.MaxNameLength)
	detail := inboxlaporanklaim.Detail{ReporterName: "  " + atLimit + "  "}.Clean()

	if detail.ReporterName != atLimit {
		t.Fatalf("nama pelapor tidak dipangkas: panjang %d", len(detail.ReporterName))
	}
	if err := detail.Check(); err != nil {
		t.Fatalf("isian sepanjang batas ditolak: %v", err)
	}
}

func TestFormRejectsTextLongerThanItsColumn(t *testing.T) {
	// Ini penjaga PENYIMPANAN, bukan aturan bisnis. Tanpa ia, Oracle menolaknya dengan
	// ORA-12899 yang tidak menyebut isian mana yang terlalu panjang.
	detail := inboxlaporanklaim.Detail{
		ReporterName: strings.Repeat("a", inboxlaporanklaim.MaxNameLength+1),
	}

	var failure *inboxlaporanklaim.ValidationError
	if err := detail.Clean().Check(); !errors.As(err, &failure) {
		t.Fatalf("isian melebihi lebar kolom justru diterima: %v", err)
	}
	if len(failure.Violation) != 1 || failure.Violation[0].Field != "nama_pelapor" {
		t.Fatalf("pelanggaran = %+v", failure.Violation)
	}
}

func TestFormReportsEveryViolationAtOnce(t *testing.T) {
	// Form ini memuat tujuh belas isian. Mengembalikan satu galat per percobaan akan
	// membuat pengguna menebak isian mana lagi yang salah (`P-5`).
	detail := inboxlaporanklaim.Detail{
		ReporterName:  strings.Repeat("a", inboxlaporanklaim.MaxNameLength+1),
		ReporterEmail: strings.Repeat("b", inboxlaporanklaim.MaxEmailLength+1),
		PolicyNumber:  strings.Repeat("c", inboxlaporanklaim.MaxPolicyLength+1),
		EstimateValue: -1,
		DocumentCount: -1,
	}

	var failure *inboxlaporanklaim.ValidationError
	if err := detail.Clean().Check(); !errors.As(err, &failure) {
		t.Fatalf("isian cacat justru diterima: %v", err)
	}
	if len(failure.Violation) != 5 {
		t.Fatalf("pelanggaran dilaporkan %d, ingin 5: %+v", len(failure.Violation), failure.Violation)
	}

	// Setiap pelanggaran menunjuk isiannya, supaya layar dapat menyorotnya di tempatnya
	// alih-alih menampilkan satu pesan di atas form.
	for _, v := range failure.Violation {
		if v.Field == "" || v.Message == "" {
			t.Fatalf("pelanggaran tanpa isian atau tanpa pesan: %+v", v)
		}
	}
}

func TestFormCountsLengthInCharactersNotBytes(t *testing.T) {
	// Satu huruf beraksen memakan dua byte. Menghitungnya sebagai dua karakter akan
	// menolak isian yang sebenarnya pendek — dan nama orang Indonesia memuatnya.
	detail := inboxlaporanklaim.Detail{
		ReporterName: strings.Repeat("é", inboxlaporanklaim.MaxNameLength),
	}
	if err := detail.Clean().Check(); err != nil {
		t.Fatalf("isian %d karakter beraksen ditolak: %v", inboxlaporanklaim.MaxNameLength, err)
	}
}

func TestFormRejectsAbsurdDocumentCount(t *testing.T) {
	// Bukan aturan bisnis — sistem lama tidak punya batas. Ia penjaga terhadap salah
	// ketik: angka enam digit pada "Total Jumlah Dokumen" hampir pasti bukan jumlah
	// dokumen.
	detail := inboxlaporanklaim.Detail{DocumentCount: inboxlaporanklaim.MaxDocumentCount + 1}

	var failure *inboxlaporanklaim.ValidationError
	if err := detail.Clean().Check(); !errors.As(err, &failure) {
		t.Fatalf("jumlah dokumen di luar batas justru diterima: %v", err)
	}
	if failure.Violation[0].Field != "jumlah_dokumen" {
		t.Fatalf("pelanggaran menunjuk isian %q", failure.Violation[0].Field)
	}
}

func TestMoneyIsStoredInCents(t *testing.T) {
	// `ADR-0016` menuntut nilai uang presisi penuh. Rp 1.000 = 100.000 sen.
	if got := inboxlaporanklaim.Rupiah(1_000); got != 100_000 {
		t.Fatalf("Rp 1.000 = %d sen, ingin 100000", got)
	}
}
