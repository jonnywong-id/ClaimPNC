package detailpenyebab_test

import (
	"errors"
	"testing"

	"claim-pnc/internal/detailpenyebab"
)

// Nama uji menyebut ATURANNYA, bukan nama fungsinya — sesuai `14-TESTING-STRATEGY.md` §3.2,
// supaya daftar uji terbaca sebagai daftar aturan yang selalu mutakhir.

func TestStatusAktifKosongDiterimaKarenaIsiannyaTidakWajib(t *testing.T) {
	input := detailpenyebab.Input{Description: "Kebakaran"}.Clean()

	if err := input.Check(); err != nil {
		t.Fatalf("Status Aktif kosong seharusnya diterima, tetapi ditolak: %v", err)
	}
}

func TestSeluruhIsianKosongDiterimaKarenaPegaTidakMemeriksaApaPun(t *testing.T) {
	// Ini bukan kelalaian melainkan kesetaraan perilaku (P-5).
	// `CNMInsertDetailCauseOfLoss_act` tidak punya satu pun Property-Set-Messages maupun
	// precondition — lihat doc comment Input.Check.
	if err := (detailpenyebab.Input{}).Clean().Check(); err != nil {
		t.Fatalf("baris kosong seharusnya diterima seperti di Pega, tetapi ditolak: %v", err)
	}
}

func TestStatusAktifDiLuarPilihannyaDitolak(t *testing.T) {
	input := detailpenyebab.Input{Active: "9"}.Clean()

	err := input.Check()
	if err == nil {
		t.Fatal("Status Aktif di luar daftar pilihannya seharusnya ditolak")
	}

	var validationError *detailpenyebab.ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("galat seharusnya ValidationError, tetapi %T", err)
	}
	if len(validationError.Violation) != 1 {
		t.Fatalf("pelanggaran seharusnya satu, tetapi %d", len(validationError.Violation))
	}
	if field := validationError.Violation[0].Field; field != "status_aktif" {
		t.Fatalf("isian yang disorot seharusnya status_aktif, tetapi %q", field)
	}
}

func TestKeduaNilaiStatusAktifYangSahDiterima(t *testing.T) {
	for _, code := range []string{detailpenyebab.ActiveYes, detailpenyebab.ActiveNo} {
		if err := (detailpenyebab.Input{Active: code}).Clean().Check(); err != nil {
			t.Fatalf("Status Aktif %q seharusnya diterima, tetapi ditolak: %v", code, err)
		}
	}
}

func TestCleanMemangkasSpasiDiKeduaUjungSetiapIsian(t *testing.T) {
	clean := detailpenyebab.Input{
		LegacyID:    "  COL-1  ",
		MasterID:    "  9001 ",
		Description: "  Kebakaran  ",
		LossCode:    "  FIRE-01 ",
		Active:      "  1 ",
	}.Clean()

	if clean.LegacyID != "COL-1" || clean.MasterID != "9001" ||
		clean.Description != "Kebakaran" || clean.LossCode != "FIRE-01" ||
		clean.Active != "1" {
		t.Fatalf("isian belum dipangkas: %+v", clean)
	}
}

func TestBarisBisnisYangSeluruhnyaKosongDibuang(t *testing.T) {
	// Selisih terencana terhadap Pega; alasannya pada doc comment Input.Clean.
	clean := detailpenyebab.Input{
		Business: []detailpenyebab.Business{
			{ID: "003", Name: "Aneka"},
			{ID: "  ", Name: "   "},
			{ID: "", Name: ""},
		},
	}.Clean()

	if len(clean.Business) != 1 {
		t.Fatalf("baris bisnis seharusnya tersisa satu, tetapi %d", len(clean.Business))
	}
	if clean.Business[0].ID != "003" {
		t.Fatalf("baris yang tersisa seharusnya 003, tetapi %q", clean.Business[0].ID)
	}
}

func TestBarisBisnisYangHanyaBernamaTetapDipertahankan(t *testing.T) {
	// Autocomplete layar lama membolehkan ketikan bebas, sehingga nama tanpa kode adalah
	// keadaan yang sah — bukan baris kosong yang perlu dibuang.
	clean := detailpenyebab.Input{
		Business: []detailpenyebab.Business{{ID: "", Name: "Aneka"}},
	}.Clean()

	if len(clean.Business) != 1 {
		t.Fatalf("baris bisnis bernama tanpa kode seharusnya dipertahankan, tetapi terbuang")
	}
}

func TestFormatIDMemadatkanNomorUrutMenjadiEmpatDigit(t *testing.T) {
	// Asal: Database/PEGA_D_CAUSE_OF_LOSS.prc:19
	//   id_dcol_ins := id_site || lpad(to_Char(D_CAUSE_SEQ.nextval),4,'0');
	cases := []struct {
		site     string
		sequence int64
		want     string
	}{
		{"10", 1, "100001"},
		{"10", 42, "100042"},
		{"10", 9999, "109999"},
		{"99", 7, "990007"},
	}

	for _, one := range cases {
		if got := detailpenyebab.FormatID(one.site, one.sequence); got != one.want {
			t.Fatalf("FormatID(%q, %d) = %q, seharusnya %q",
				one.site, one.sequence, got, one.want)
		}
	}
}

func TestNomorUrutLebihDariEmpatDigitTidakDipotong(t *testing.T) {
	// `lpad` pada Oracle pun tidak memotong — ia hanya berhenti memadatkan. ID yang lebih
	// panjang lebih baik daripada ID yang terpotong dan bertabrakan dengan baris lain.
	if got := detailpenyebab.FormatID("10", 123456); got != "10123456" {
		t.Fatalf("FormatID tidak boleh memotong nomor urut, tetapi menghasilkan %q", got)
	}
}

func TestHanyaNilaiSatuYangBerartiAktif(t *testing.T) {
	// Seluruh perbandingan STS_AKTIF di export berbunyi `= '1'`; apa pun selain itu sudah
	// berperilaku sebagai tidak aktif.
	if !detailpenyebab.IsActive("1") {
		t.Fatal(`"1" seharusnya berarti aktif`)
	}
	for _, code := range []string{"0", "", "2", "Y", "ya"} {
		if detailpenyebab.IsActive(code) {
			t.Fatalf("%q seharusnya TIDAK berarti aktif", code)
		}
	}
}

func TestStatusAktifKosongDibedakanDariTidakAktifSaatDitampilkan(t *testing.T) {
	// Keduanya sama bagi penyaring, tetapi berbeda artinya bagi petugas — lihat
	// ActiveLabel.
	if label := detailpenyebab.ActiveLabel(""); label != "Belum diisi" {
		t.Fatalf(`label status kosong seharusnya "Belum diisi", tetapi %q`, label)
	}
	if label := detailpenyebab.ActiveLabel("0"); label != "Tidak Aktif" {
		t.Fatalf(`label status "0" seharusnya "Tidak Aktif", tetapi %q`, label)
	}
	if label := detailpenyebab.ActiveLabel("1"); label != "Aktif" {
		t.Fatalf(`label status "1" seharusnya "Aktif", tetapi %q`, label)
	}
}

func TestNilaiStatusAktifWarisanYangTidakDikenalTetapTerbaca(t *testing.T) {
	// Baris lama bernilai lain tidak boleh gagal dibaca; di Pega ia hanya tidak ikut
	// tersaring.
	if label := detailpenyebab.ActiveLabel("X"); label != "Tidak Aktif" {
		t.Fatalf("nilai warisan yang tidak dikenal seharusnya terbaca Tidak Aktif, tetapi %q", label)
	}
}

func TestPilihanStatusAktifTidakDapatDiubahDariLuar(t *testing.T) {
	// ActiveOptions adalah fungsi, bukan variabel paket, supaya satu pemanggil tidak dapat
	// mengubah pilihan pemanggil lain.
	first := detailpenyebab.ActiveOptions()
	first[0].Label = "Diubah"

	if second := detailpenyebab.ActiveOptions(); second[0].Label != "Aktif" {
		t.Fatalf("daftar pilihan ikut berubah dari luar: %q", second[0].Label)
	}
}
