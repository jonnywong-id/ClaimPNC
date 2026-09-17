package sqlstore

import "testing"

// Dua uji berikut mengunci SANDI KOLOM, bukan perilaku aplikasi.
//
// Keduanya ada karena saya sempat menebaknya salah: STS_AKTIF saya sandikan "1"/"0"
// padahal isinya "Ya"/"Tidak". Salah sandi di sini tidak membuat apa pun gagal — ia
// hanya membuat setiap rekening terbaca sebagai NONAKTIF, sehingga petugas mengira
// rekening yang sah tidak dapat dipakai membayar klaim. Kegagalan diam seperti itulah
// yang menuntut uji, bukan kegagalan yang berisik.
//
// Sumbernya activity SetTipeRekening, yang mengisi kedua daftar pilihan layar lama.

func TestSandiAktifMemakaiYaDanTidak(t *testing.T) {
	if got := sandiAktif(true); got != "Ya" {
		t.Errorf("rekening aktif ditulis sebagai %q, seharusnya \"Ya\"", got)
	}
	if got := sandiAktif(false); got != "Tidak" {
		t.Errorf("rekening nonaktif ditulis sebagai %q, seharusnya \"Tidak\"", got)
	}
}

func TestBacaAktifMenerimaEjaanYangSudahAdaDiTabel(t *testing.T) {
	// Kolom ini sudah dipakai bertahun-tahun oleh beberapa rule. Baris lama yang
	// ejaannya berbeda tetap harus terbaca aktif.
	aktif := []string{"Ya", "ya", "YA", " Ya ", "1", "Y", "Aktif"}
	for _, nilai := range aktif {
		if !bacaAktif(nilai) {
			t.Errorf("STS_AKTIF %q seharusnya terbaca AKTIF", nilai)
		}
	}

	nonaktif := []string{"Tidak", "tidak", "0", "", "N", "apa saja"}
	for _, nilai := range nonaktif {
		if bacaAktif(nilai) {
			t.Errorf("STS_AKTIF %q seharusnya terbaca NONAKTIF", nilai)
		}
	}
}

func TestPulangPergiStatusAktifTetapUtuh(t *testing.T) {
	for _, aktif := range []bool{true, false} {
		if bacaAktif(sandiAktif(aktif)) != aktif {
			t.Errorf("status aktif %v berubah setelah ditulis lalu dibaca kembali", aktif)
		}
	}
}
