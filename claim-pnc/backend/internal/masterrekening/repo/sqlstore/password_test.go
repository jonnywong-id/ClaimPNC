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

func TestActiveFlagUsesYesAndNo(t *testing.T) {
	if got := activeFlag(true); got != "Ya" {
		t.Errorf("rekening aktif ditulis sebagai %q, seharusnya \"Ya\"", got)
	}
	if got := activeFlag(false); got != "Tidak" {
		t.Errorf("rekening nonaktif ditulis sebagai %q, seharusnya \"Tidak\"", got)
	}
}

func TestReadActiveAcceptsSpellingsAlreadyInTable(t *testing.T) {
	// Kolom ini sudah dipakai bertahun-tahun oleh beberapa rule. Baris lama yang
	// ejaannya berbeda tetap harus terbaca aktif.
	active := []string{"Ya", "ya", "YA", " Ya ", "1", "Y", "Aktif"}
	for _, value := range active {
		if !readActive(value) {
			t.Errorf("STS_AKTIF %q seharusnya terbaca AKTIF", value)
		}
	}

	inactive := []string{"Tidak", "tidak", "0", "", "N", "apa saja"}
	for _, value := range inactive {
		if readActive(value) {
			t.Errorf("STS_AKTIF %q seharusnya terbaca NONAKTIF", value)
		}
	}
}

func TestActiveFlagSurvivesRoundTrip(t *testing.T) {
	for _, active := range []bool{true, false} {
		if readActive(activeFlag(active)) != active {
			t.Errorf("status aktif %v berubah setelah ditulis lalu dibaca kembali", active)
		}
	}
}
