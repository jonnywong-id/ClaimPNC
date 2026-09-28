package inboxmanageradmin_test

import (
	"testing"
	"time"

	"claim-pnc/internal/inboxmanageradmin"
)

// Uji di berkas ini menjaga bentuk kolom "Lama Waktu Klaim" supaya tidak menyimpang dari
// `inboxcompliance.FormatElapsed`, yang menulis ulang format bawaan Pega yang sama.
//
// Kasusnya sengaja memuat satu yang BENAR-BENAR TERAMATI di layar Pega dan beberapa yang
// masih rekonstruksi — keduanya ditandai, supaya orang berikutnya tahu mana yang boleh
// dijadikan acuan dan mana yang menunggu verifikasi.

func TestLamaWaktuKlaimMengikutiBentukYangTeramatiDiPega(t *testing.T) {
	// TERAMATI LANGSUNG: baris bertanggal 22/04/25 13:46 menampilkan "1 year 5 months ago".
	//
	// Dari satu contoh itu dua hal dapat dipastikan, dan keduanya diuji di sini: dua satuan
	// ditampilkan sekaligus begitu melewati satu tahun, dan bentuk jamaknya benar PER
	// SATUAN — "1 year" tunggal berdampingan dengan "5 months" jamak.
	from := time.Date(2025, time.April, 22, 13, 46, 0, 0, time.UTC)
	now := time.Date(2026, time.September, 24, 10, 0, 0, 0, time.UTC)

	if got := inboxmanageradmin.FormatElapsed(&from, now); got != "1 year 5 months ago" {
		t.Errorf("hasil %q, seharusnya %q", got, "1 year 5 months ago")
	}
}

func TestLamaWaktuKlaimMenurunkanSatuanSesuaiRentangnya(t *testing.T) {
	// REKONSTRUKSI: bunyi cabang bulan, hari, jam, dan menit mengikuti bentuk baku
	// `DateTime-Frame` dan BELUM diverifikasi terhadap layar Pega — tidak ada baris yang
	// cukup baru di sana untuk dibandingkan.
	//
	// Diuji tetap, supaya bentuknya tidak berubah diam-diam sebelum verifikasinya datang.
	now := time.Date(2026, time.September, 26, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		from time.Time
		want string
	}{
		{"dua tahun tepat, sisa bulan tidak ditulis",
			time.Date(2024, time.September, 26, 12, 0, 0, 0, time.UTC), "2 years ago"},
		{"beberapa bulan",
			time.Date(2026, time.June, 26, 12, 0, 0, 0, time.UTC), "3 months ago"},
		{"satu bulan",
			time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC), "1 month ago"},
		{"beberapa hari",
			time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC), "5 days ago"},
		{"satu hari",
			time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC), "1 day ago"},
		{"beberapa jam",
			time.Date(2026, time.September, 26, 9, 0, 0, 0, time.UTC), "3 hours ago"},
		{"satu jam",
			time.Date(2026, time.September, 26, 11, 0, 0, 0, time.UTC), "1 hour ago"},
		{"beberapa menit",
			time.Date(2026, time.September, 26, 11, 45, 0, 0, time.UTC), "15 minutes ago"},
		{"satu menit",
			time.Date(2026, time.September, 26, 11, 59, 0, 0, time.UTC), "1 minute ago"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			from := c.from
			if got := inboxmanageradmin.FormatElapsed(&from, now); got != c.want {
				t.Errorf("hasil %q, seharusnya %q", got, c.want)
			}
		})
	}
}

func TestLamaWaktuKlaimKosongUntukTanggalYangTidakBerlaku(t *testing.T) {
	now := time.Date(2026, time.September, 26, 12, 0, 0, 0, time.UTC)
	future := now.Add(48 * time.Hour)
	var zero time.Time

	cases := []struct {
		name string
		from *time.Time
	}{
		{"tanggal kosong", nil},
		{"tanggal nol", &zero},
		// Tanggal di masa depan memang ada di data warisan, dan "in 2 days" bukan bentuk
		// yang pernah ditulis kolom ini. Kosong lebih jujur daripada kalimat yang salah.
		{"tanggal di masa depan", &future},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := inboxmanageradmin.FormatElapsed(c.from, now); got != "" {
				t.Errorf("hasil %q, seharusnya kosong", got)
			}
		})
	}
}

func TestLamaWaktuKlaimDihitungTerhadapWIB(t *testing.T) {
	// Waktu disimpan UTC dan diubah ke WIB di satu tempat saja (`F-5`). Kalau pengubahan
	// itu terlewat, baris yang masuk pukul 23.30 WIB terbaca sebagai hari sebelumnya, dan
	// kolom ini melaporkan satu hari lebih tua.
	//
	// 2026-09-25 17:00 UTC adalah 2026-09-26 00:00 WIB — awal hari yang sama dengan `now`
	// di bawah, sehingga selisihnya kurang dari sehari.
	from := time.Date(2026, time.September, 25, 17, 30, 0, 0, time.UTC)
	now := time.Date(2026, time.September, 25, 20, 30, 0, 0, time.UTC)

	if got := inboxmanageradmin.FormatElapsed(&from, now); got != "3 hours ago" {
		t.Errorf("hasil %q, seharusnya %q", got, "3 hours ago")
	}
}
