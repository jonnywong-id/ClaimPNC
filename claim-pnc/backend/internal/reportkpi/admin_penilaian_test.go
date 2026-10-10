package reportkpi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// Tangga nilai NON-MBU, termasuk keanehan yang DIREPLIKASI.
//
// Persentase di sini adalah persentase PELANGGARAN SLA — makin kecil makin baik, sehingga
// tangganya menurun: 0% memberi nilai 5.
func TestTanggaNilaiAdminNonMBU(t *testing.T) {
	// over, total -> nilai yang diharapkan
	kasus := []struct {
		nama        string
		over, total int
		nilai       float64
		persen      float64
		adaPersen   bool
	}{
		{"tanpa pelanggaran", 0, 1000, 5, 0, true},
		{"0,4% — masih di bawah 0,5", 4, 1000, 5, 0.4, true},
		{"tepat 0,5%", 5, 1000, 4, 0.5, true},
		{"tepat 1% — TETAP 4, bukan 3", 10, 1000, 4, 1, true},
		{"1,2%", 12, 1000, 2, 1.2, true},
		{"tepat 1,5%", 15, 1000, 2, 1.5, true},
		{"1,8%", 18, 1000, 1, 1.8, true},
		{"tepat 2%", 20, 1000, 1, 2, true},
		{"di atas 2%", 21, 1000, 0, 2.1, true},
		{"tidak ada yang diukur", 0, 0, 0, 0, false},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil := reportkpi.BuildAdminTotalsNonMBU(reportkpi.AdminCountsNonMBU{
				LeaderOverSLA: k.over, LeaderTotal: k.total,
			})

			require.Equal(t, k.nilai, hasil.LeaderScore.Value)
			require.Equal(t, k.adaPersen, hasil.LeaderPercent.Present,
				"pembagi nol harus menghasilkan persentase KOSONG, bukan nol")
			if k.adaPersen {
				require.InDelta(t, k.persen, hasil.LeaderPercent.Value, 0.0001)
			}
		})
	}
}

// Nilai 3 TIDAK PERNAH keluar dari tangga ini.
//
// Cabang `= 1` pada kueri lama tidak terjangkau: `BETWEEN 0.5 AND 1` di atasnya inklusif,
// sehingga nilai 1 sudah tertangkap lebih dulu. Keanehan itu direplikasi (`P-5`), dan uji
// ini yang membuatnya disengaja — bukan kelalaian penyalinan.
//
// Bila kelak Work Owner memutuskan memperbaikinya, uji inilah yang pertama gagal, dan
// pembacanya akan menemukan alasannya di sini.
func TestNilaiTigaTidakPernahTercapai(t *testing.T) {
	for over := 0; over <= 300; over++ {
		hasil := reportkpi.BuildAdminTotalsNonMBU(reportkpi.AdminCountsNonMBU{
			LeaderOverSLA: over, LeaderTotal: 10000,
		})
		require.NotEqualf(t, float64(3), hasil.LeaderScore.Value,
			"nilai 3 muncul pada %d/10000 — cabang `= 1` seharusnya tidak terjangkau", over)
	}
}

// Bobot dan rasio pencapaian disalin apa adanya dari kueri lama.
func TestBobotDanRasioPencapaianAdmin(t *testing.T) {
	// Keduanya sempurna: nilai 5 dan 5.
	hasil := reportkpi.BuildAdminTotalsNonMBU(reportkpi.AdminCountsNonMBU{
		LeaderOverSLA: 0, LeaderTotal: 100,
		MemberOverSLA: 0, MemberTotal: 100,
	})

	// (5/5) * 0,45 * 100 = 45 · (5/5) * 0,40 * 100 = 40
	require.InDelta(t, 45, hasil.LeaderSubtotal.Value, 0.0001)
	require.InDelta(t, 40, hasil.MemberSubtotal.Value, 0.0001)
	require.InDelta(t, 85, hasil.QuantitativeTotal.Value, 0.0001)

	// ROUND(85 / ((3/5)*90), 2) = ROUND(85 / 54, 2) = 1,57
	require.InDelta(t, 1.57, hasil.AchievementRatio.Value, 0.0001)
}

// Kartu skor PA memakai `claim_total` sebagai pembagi KEDUA tangga.
//
// Termasuk untuk pembayaran — bukan `payment_total`. Itu yang dilakukan kueri lama, dan
// menukarnya akan mengubah nilai tanpa ada yang meminta.
func TestTanggaNilaiAdminPAMemakaiClaimTotal(t *testing.T) {
	hasil := reportkpi.BuildAdminTotalsPA(reportkpi.AdminCountsPA{
		RegisterOverSLA: 0,
		PaymentOverSLA:  30, // 30/1000 = 3% -> di atas 2% -> nilai 0
		ClaimTotal:      1000,
		PaymentTotal:    3, // SENGAJA kecil: bila ia yang jadi pembagi, nilainya berbeda
	})

	require.Equal(t, float64(5), hasil.RegisterScore.Value)
	require.Equal(t, float64(0), hasil.PaymentScore.Value,
		"pembaginya claim_total, bukan payment_total")

	// PaymentTotal tetap dilaporkan apa adanya meski tidak dipakai sebagai pembagi.
	require.Equal(t, float64(3), hasil.PaymentTotal.Value)
	require.Equal(t, float64(1000), hasil.RegisterTotal.Value)
}
