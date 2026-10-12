package komite_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

var sekarang = time.Date(2026, 9, 18, 6, 0, 0, 0, time.UTC)

// tertunda membentuk satu jenjang komite yang masih menunggu: jenjang ke-1 dari 3.
func tertunda(menungguHari float64, nilai int64, jenjang int) komite.PendingCommittee {
	return komite.PendingCommittee{
		ClaimNumber:  "PNCN.26.1",
		CurrentTier:  jenjang,
		TierCount:    3,
		OperatorID:   "UJI",
		Value:        rp(nilai),
		WaitingSince: sekarang.Add(-time.Duration(menungguHari * float64(24*time.Hour))),
	}
}

// Persetujuan otomatis MATI secara bawaan.
//
// Matinya disengaja: job ini melewati seluruh kontrol otorisasi, karena `D-59` menetapkan
// izin bersatuan menu dan job tidak punya pengguna sehingga tidak ada menu yang dapat
// diperiksa. Sesuatu yang menyetujui uang tanpa kontrol tidak boleh menyala hanya karena
// kelalaian menyetelnya.
func TestPersetujuanOtomatisMatiSecaraBawaan(t *testing.T) {
	hasil := komite.EvaluateAuto(tertunda(30, 1_000_000, 1), sekarang, komite.AutoPolicy{})

	require.False(t, hasil.Eligible)
	require.Equal(t, komite.ReasonDisabled, hasil.Reason)
}

// Ambang lama menganggur adalah inti aturannya, dan ia diuji TEPAT DI BATAS.
func TestAmbangLamaMenganggurDiujiTepatDiBatas(t *testing.T) {
	kebijakan := komite.AutoPolicy{Enabled: true}

	kasus := []struct {
		nama  string
		hari  float64
		layak bool
	}{
		{"baru saja", 0, false},
		{"dua hari kurang sedikit", 1.99, false},
		{"dua hari", 2, false},
		{"tepat tiga hari", 3, true},
		{"lebih dari tiga hari", 3.01, true},
		{"tiga puluh hari", 30, true},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil := komite.EvaluateAuto(tertunda(k.hari, 1_000_000, 1), sekarang, kebijakan)
			require.Equal(t, k.layak, hasil.Eligible)
			if !k.layak {
				require.Equal(t, komite.ReasonTooRecent, hasil.Reason)
			}
		})
	}
}

// Ambangnya 72 jam — ditegaskan Work Owner 2026-09-18, menutup ambiguitas `> 2` pada rule
// lama yang dapat dibaca sebagai 48 jam maupun 72 jam.
//
// Ia tetap dapat diubah tanpa menyentuh aturannya, karena `D-15` melarang nilai bisnis
// di-hardcode.
func TestAmbangDapatDiubahTanpaMenyentuhAturan(t *testing.T) {
	duaHari := komite.AutoPolicy{Enabled: true, AgeDays: 2}

	hasil := komite.EvaluateAuto(tertunda(2.5, 1_000_000, 1), sekarang, duaHari)
	require.True(t, hasil.Eligible, "dengan ambang dua hari, 2,5 hari sudah layak")

	hasilBawaan := komite.EvaluateAuto(tertunda(2.5, 1_000_000, 1), sekarang,
		komite.AutoPolicy{Enabled: true})
	require.False(t, hasilBawaan.Eligible, "dengan ambang bawaan tiga hari, belum layak")
}

// TIDAK ADA batas nilai dan tidak ada batas jenjang — Work Owner, 2026-09-18.
//
// Syaratnya hanya dua: sudah menganggur cukup lama, dan jenjangnya belum habis. Klaim
// senilai berapa pun disetujui otomatis bila kedua syarat itu terpenuhi, persis seperti
// sistem lama.
func TestTidakAdaBatasNilaiMaupunJenjang(t *testing.T) {
	kebijakan := komite.AutoPolicy{Enabled: true}

	besar := komite.PendingCommittee{
		ClaimNumber:  "PNCN.26.9",
		CurrentTier:  4,
		TierCount:    4,
		Value:        money.FromRupiah(5_000_000_000),
		WaitingSince: sekarang.Add(-10 * 24 * time.Hour),
	}

	require.True(t, komite.EvaluateAuto(besar, sekarang, kebijakan).Eligible,
		"klaim Rp 5 miliar pada jenjang keempat tetap disetujui otomatis")
}

// Syarat kedua: jenjangnya belum habis — `KomiteCount <= KomiteLoop`.
//
// Ia diambil apa adanya dari `When/IsKomiteLoop-When.xml`, dan Work Owner menegaskan
// inilah satu-satunya syarat selain lama menganggur.
func TestJenjangYangSudahHabisTidakDisetujuiLagi(t *testing.T) {
	kebijakan := komite.AutoPolicy{Enabled: true}

	buat := func(komiteKe, jumlah int) komite.PendingCommittee {
		return komite.PendingCommittee{
			ClaimNumber:  "PNCN.26.2",
			CurrentTier:  komiteKe,
			TierCount:    jumlah,
			WaitingSince: sekarang.Add(-10 * 24 * time.Hour),
		}
	}

	kasus := []struct {
		nama     string
		komiteKe int
		jumlah   int
		layak    bool
	}{
		{"jenjang pertama dari tiga", 1, 3, true},
		{"jenjang terakhir", 3, 3, true},
		{"sudah melewati jenjang terakhir", 4, 3, false},
		{"tidak punya jenjang sama sekali", 1, 0, false},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil := komite.EvaluateAuto(buat(k.komiteKe, k.jumlah), sekarang, kebijakan)
			require.Equal(t, k.layak, hasil.Eligible)
			if !k.layak {
				require.Equal(t, komite.ReasonTiersDone, hasil.Reason)
			}
		})
	}
}

// Yang TIDAK layak ikut dikembalikan beserta alasannya.
//
// Itulah yang membuat job ini dapat diperiksa sebelum dinyalakan: seseorang dapat melihat
// apa yang AKAN disetujui, dan apa yang tidak, tanpa satu pun baris berubah.
func TestSaringMengembalikanYangTidakLayakBesertaAlasannya(t *testing.T) {
	kebijakan := komite.AutoPolicy{Enabled: true}

	hasil := komite.FilterAuto([]komite.PendingCommittee{
		tertunda(10, 1_000_000, 1),
		tertunda(1, 1_000_000, 1),
	}, sekarang, kebijakan)

	require.Len(t, hasil, 2)
	require.True(t, hasil[0].Eligible)
	require.False(t, hasil[1].Eligible)
	require.Equal(t, komite.ReasonTooRecent, hasil[1].Reason)

	// Lama menganggur ikut dilaporkan, supaya yang membaca dapat menilai sendiri seberapa
	// jauh sebuah jenjang dari ambangnya.
	require.Equal(t, 10*24*time.Hour, hasil[0].Idle)
}

// Pelaku dicatat sebagai "SISTEM", bukan dibiarkan kosong.
//
// Sistem lama hanya menitipkan jejaknya pada kalimat di kolom catatan —
// "Auto Accept by PEGA Claim Non MBU" — sehingga satu-satunya cara mengetahui sebuah
// persetujuan itu otomatis adalah mencocokkan teks.
func TestPelakuSistemDinyatakanTegas(t *testing.T) {
	require.Equal(t, "SISTEM", komite.ActorSystem)
}
