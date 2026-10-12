package daftartipedokumenbisnis_test

import (
	"errors"
	"testing"

	"claim-pnc/internal/daftartipedokumenbisnis"
)

func TestCleanMemangkasSpasiSetiapIsian(t *testing.T) {
	clean := daftartipedokumenbisnis.Input{
		DocumentTypeID:  "  20001  ",
		ObjectDocID:     "\t30001\n",
		DetailTypeDocID: " 40001 ",
		DetailDocument:  "  Laporan Kerugian  ",
		Mandatory:       true,
		MinDocument:     2,
	}.Clean()

	if clean.DocumentTypeID != "20001" {
		t.Fatalf("DocumentTypeID = %q, ingin %q", clean.DocumentTypeID, "20001")
	}
	if clean.ObjectDocID != "30001" {
		t.Fatalf("ObjectDocID = %q, ingin %q", clean.ObjectDocID, "30001")
	}
	if clean.DetailTypeDocID != "40001" {
		t.Fatalf("DetailTypeDocID = %q, ingin %q", clean.DetailTypeDocID, "40001")
	}
	if clean.DetailDocument != "Laporan Kerugian" {
		t.Fatalf("DetailDocument = %q, ingin %q", clean.DetailDocument, "Laporan Kerugian")
	}
}

// Isian kosong TETAP DITERIMA. Layar Pega tidak memuat satu pun `pyRequired` bernilai true
// dan tidak memasang Validate rule, sehingga menolaknya di sini akan menghalangi
// penyimpanan yang di sistem lama berhasil (`P-5`).
func TestCleanTidakMenolakIsianKosong(t *testing.T) {
	clean := daftartipedokumenbisnis.Input{DetailDocument: "   "}.Clean()

	if clean.DetailDocument != "" {
		t.Fatalf("DetailDocument = %q, ingin kosong", clean.DetailDocument)
	}
}

// MIN_DOC negatif tidak punya arti yang dapat dipenuhi maupun dilanggar. Ia DIRATAKAN
// menjadi nol, bukan ditolak, supaya perlakuannya tetap sejalan dengan "tanpa validasi".
func TestCleanMerataknMinDokumenNegatifMenjadiNol(t *testing.T) {
	clean := daftartipedokumenbisnis.Input{MinDocument: -5}.Clean()

	if clean.MinDocument != 0 {
		t.Fatalf("MinDocument = %d, ingin 0", clean.MinDocument)
	}
}

// Bisnis yang sama dipilih dua kali dipadatkan menjadi satu. Tanpa itu, petugas memperoleh
// dua baris kembar yang tidak dapat dibedakan di grid mana pun.
func TestBatchCleanMembuangBisnisKembar(t *testing.T) {
	clean := daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"001", " 001 ", "004", "", "   "},
		Rules:       []daftartipedokumenbisnis.Input{{DetailDocument: "Laporan"}},
	}.Clean()

	if len(clean.BusinessIDs) != 2 {
		t.Fatalf("jumlah bisnis = %d, ingin 2: %v", len(clean.BusinessIDs), clean.BusinessIDs)
	}
	if clean.BusinessIDs[0] != "001" || clean.BusinessIDs[1] != "004" {
		t.Fatalf("bisnis = %v, ingin [001 004]", clean.BusinessIDs)
	}
}

// Baris grid yang baru ditambahkan tetapi belum diisi dibuang. Menyimpannya berarti
// menulis aturan dokumen tanpa dokumen.
func TestBatchCleanMembuangBarisYangSeluruhnyaKosong(t *testing.T) {
	clean := daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"001"},
		Rules: []daftartipedokumenbisnis.Input{
			{DetailDocument: "Laporan Kerugian"},
			{},
			{DocumentTypeID: "  "},
		},
	}.Clean()

	if len(clean.Rules) != 1 {
		t.Fatalf("jumlah baris = %d, ingin 1", len(clean.Rules))
	}
}

// Baris yang HANYA bertanda wajib tetap disimpan: menandai dokumen wajib tanpa mengisi
// namanya adalah isian yang janggal, tetapi ia bukan baris kosong — dan sistem lama pun
// menyimpannya.
func TestBatchCleanMempertahankanBarisYangHanyaBertandaWajib(t *testing.T) {
	clean := daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"001"},
		Rules:       []daftartipedokumenbisnis.Input{{Mandatory: true}},
	}.Clean()

	if len(clean.Rules) != 1 {
		t.Fatalf("jumlah baris = %d, ingin 1", len(clean.Rules))
	}
}

// Satu-satunya aturan isian yang ditiru dari layar lama
// (`InsertDetailTypeDocumentBusiness_act:208`).
func TestValidateMenolakTanpaBisnis(t *testing.T) {
	err := daftartipedokumenbisnis.BatchInput{
		Rules: []daftartipedokumenbisnis.Input{{DetailDocument: "Laporan"}},
	}.Clean().Validate()

	if !errors.Is(err, daftartipedokumenbisnis.ErrBusinessRequired) {
		t.Fatalf("galat = %v, ingin ErrBusinessRequired", err)
	}
}

// Penyimpanan TANPA baris dokumen DITERIMA, dan tidak menyimpan apa pun.
//
// Ini meniru Pega: perulangan atas baris dokumen berputar nol kali, lalu layar tertutup
// tanpa pesan apa pun. Versi pertama modul ini menolaknya sebagai galat — penambahan saya
// sendiri, dicabut atas keputusan Work Owner 2026-09-23.
//
// Uji ini menjaga pencabutan itu: bila kelak penolakannya dipasang kembali, sebuah
// penyimpanan yang di sistem lama berhasil akan mulai ditolak.
func TestValidateMenerimaTanpaBarisDokumen(t *testing.T) {
	err := daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"001"},
	}.Clean().Validate()

	if err != nil {
		t.Fatalf("galat = %v, ingin nil — Pega pun menerimanya tanpa menyimpan apa pun", err)
	}
}

func TestValidateMenerimaBisnisDanBarisYangTerisi(t *testing.T) {
	err := daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"001", "004"},
		Rules:       []daftartipedokumenbisnis.Input{{DetailDocument: "Laporan"}},
	}.Clean().Validate()

	if err != nil {
		t.Fatalf("galat = %v, ingin nil", err)
	}
}

// Empat digit, dibaca dari `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:22`:
//
//	id_site || lpad(to_Char(LST_TYPE_DOC_BUSINESS_SEQ.nextval),4,'0')
func TestFormatIDMengikutiBentukProcedure(t *testing.T) {
	cases := []struct {
		site     string
		sequence int64
		want     string
	}{
		{"1", 1, "10001"},
		{"1", 28, "10028"},
		{"1", 9999, "19999"},
		{"2", 7, "20007"},
	}

	for _, c := range cases {
		if got := daftartipedokumenbisnis.FormatID(c.site, c.sequence); got != c.want {
			t.Fatalf("FormatID(%q, %d) = %q, ingin %q", c.site, c.sequence, got, c.want)
		}
	}
}

// LPAD Oracle tidak memotong nilai yang lebih panjang dari lebarnya, dan FormatID pun
// tidak. Penyisipan ke-10000 karena itu menghasilkan ID satu karakter lebih panjang —
// keadaan yang perlu diketahui DBA sebelum terjadi, dan dicatat di migrasi 0010.
func TestFormatIDTidakMemotongNomorYangMelampauiPadding(t *testing.T) {
	if got := daftartipedokumenbisnis.FormatID("1", 10000); got != "110000" {
		t.Fatalf("FormatID = %q, ingin %q", got, "110000")
	}
}
