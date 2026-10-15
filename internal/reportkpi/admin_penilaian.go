package reportkpi

import "math"

// Berkas ini memuat PENILAIAN tab KPI Admin: tangga nilai, bobot, dan rasio pencapaian.
//
// # Kenapa ia pindah ke sini (2026-10-09)
//
// Sampai hari itu seluruh hitungan di bawah dikerjakan di dalam teks SQL, bersama dengan
// pencacahannya. Keduanya tidak dapat dipisahkan di sana karena pencacahannya bergantung
// pada `datamining.get_working_hours@asmd` — fungsi lintas DB Link yang **tidak dapat
// dijangkau** dari basis data kita. Terbukti dengan memanggilnya langsung; objek lain pada
// DB Link yang sama terbaca normal.
//
// Karena pencacahannya harus pindah ke Go, penilaiannya ikut — dan tempatnya memang di
// sini: tangga nilai dan bobot adalah ATURAN BISNIS, bukan pengambilan data (`D-50`).
//
// Seluruh angka di bawah disalin kata demi kata dari `reportkpi_admin.sql`, termasuk
// keanehannya.

// adminBand adalah tangga nilai 1–5 atas PERSENTASE pelanggaran SLA.
//
// # Keanehan yang DIREPLIKASI, bukan diperbaiki
//
// `BETWEEN` di Oracle inklusif pada kedua ujungnya, sehingga cabang `= 1` TIDAK PERNAH
// tercapai — cabang `BETWEEN 0.5 AND 1` di atasnya sudah menangkap nilai 1 lebih dulu.
// Nilai 3 karena itu mustahil keluar dari tangga ini.
//
// Itu perilaku Pega hari ini (`P-5`). Memperbaikinya mengubah nilai orang, dan itu
// keputusan Work Owner — bukan keputusan yang diambil sambil memindahkan kode.
//
// Persentase yang TIDAK ADA — karena pembaginya nol — menghasilkan 0, sama seperti
// `NULLIF(total, 0)` yang membuat seluruh perbandingan `CASE` bernilai NULL lalu jatuh ke
// `ELSE 0`.
func adminBand(percent Score) Score {
	if !percent.Present {
		return NewScore(0)
	}

	p := percent.Value
	switch {
	case p < 0.5:
		return NewScore(5)
	case p >= 0.5 && p <= 1:
		return NewScore(4)
	case p > 1 && p <= 1.5:
		return NewScore(2)
	case p > 1.5 && p <= 2:
		return NewScore(1)
	default:
		return NewScore(0)
	}
}

// adminPercent menghitung persentase pelanggaran SLA.
//
// Pembagi nol menghasilkan nilai KOSONG, bukan nol — meniru `NULLIF(total, 0)`. Keduanya
// berbeda artinya: nol berarti "tidak ada yang melanggar", kosong berarti "tidak ada yang
// diukur".
func adminPercent(over, total int) Score {
	if total == 0 {
		return EmptyScore()
	}
	return NewScore(float64(over) / float64(total) * 100)
}

// Bobot kedua kelompok NON-MBU, apa adanya seperti teks kueri lama.
const (
	adminLeaderWeight = 0.45
	adminMemberWeight = 0.40

	// adminMaxBand adalah puncak tangga nilai; subtotal dihitung sebagai bagian darinya.
	adminMaxBand = 5.0

	// adminRatioDivider adalah `(3 / 5) * 90` pada kueri lama, ditulis sebagai hasilnya.
	adminRatioDivider = 54.0

	// AdminAchievementTarget adalah kolom ACHIEVEMENT pada grid "Data KPI".
	//
	// Di kueri lama ia ditulis `3/5*85` dan TIDAK dihitung dari apa pun — sebuah tetapan
	// yang ikut digambar sebagai kolom. Perhatikan pembilangnya 85, sementara pembagi
	// rasio pencapaian memakai 90; kedua angka itu memang berbeda di sana.
	AdminAchievementTarget = 51.0
)

// AdminCountsNonMBU adalah keempat cacahan mentah kelompok NON-MBU.
type AdminCountsNonMBU struct {
	LeaderOverSLA int
	LeaderTotal   int
	MemberOverSLA int
	MemberTotal   int
}

// BuildAdminTotalsNonMBU menyusun angka kartu skor NON-MBU dari cacahan mentah.
func BuildAdminTotalsNonMBU(counts AdminCountsNonMBU) AdminTotals {
	leaderPercent := adminPercent(counts.LeaderOverSLA, counts.LeaderTotal)
	memberPercent := adminPercent(counts.MemberOverSLA, counts.MemberTotal)

	leaderScore := adminBand(leaderPercent)
	memberScore := adminBand(memberPercent)

	leaderSubtotal := leaderScore.Value / adminMaxBand * adminLeaderWeight * 100
	memberSubtotal := memberScore.Value / adminMaxBand * adminMemberWeight * 100
	quantitative := leaderSubtotal + memberSubtotal

	return AdminTotals{
		LeaderOverSLA:  NewScore(float64(counts.LeaderOverSLA)),
		LeaderTotal:    NewScore(float64(counts.LeaderTotal)),
		LeaderPercent:  leaderPercent,
		LeaderScore:    leaderScore,
		LeaderSubtotal: NewScore(leaderSubtotal),

		MemberOverSLA:  NewScore(float64(counts.MemberOverSLA)),
		MemberTotal:    NewScore(float64(counts.MemberTotal)),
		MemberPercent:  memberPercent,
		MemberScore:    memberScore,
		MemberSubtotal: NewScore(memberSubtotal),

		QuantitativeTotal: NewScore(quantitative),

		// TRUNC, bukan ROUND — kueri lama menulis `TRUNC(..., 2)`.
		//
		// Keduanya sepakat pada hampir semua nilai dan berselisih tepat di tengah, misalnya
		// 1,575: TRUNC memberi 1,57 sedangkan ROUND memberi 1,58. Pada kolom yang menentukan
		// tercapai atau tidaknya target, selisih itu tidak boleh diserahkan kepada kebetulan.
		AchievementRatio: NewScore(trunc2(quantitative / adminRatioDivider)),
	}
}

// AdminCountsPA adalah keempat cacahan mentah kelompok PA.
type AdminCountsPA struct {
	RegisterOverSLA int
	PaymentOverSLA  int
	ClaimTotal      int

	// PaymentTotal dihitung atas rentang 2023 yang TERTANAM di kueri lama, bukan atas
	// periode yang dipilih pengguna. Lihat catatan pada kueri `admin_rows_pa_payment_total`.
	PaymentTotal int
}

// BuildAdminTotalsPA menyusun angka kartu skor PA dari cacahan mentah.
//
// Kedua tangga nilainya memakai pembagi yang SAMA — `claim_total` — termasuk untuk
// pembayaran. Itu bukan kekeliruan penyalinan: kueri lama memang membaginya dengan
// `claim_total`, bukan dengan `payment_total`.
func BuildAdminTotalsPA(counts AdminCountsPA) AdminTotals {
	registerPercent := adminPercent(counts.RegisterOverSLA, counts.ClaimTotal)
	paymentPercent := adminPercent(counts.PaymentOverSLA, counts.ClaimTotal)

	return AdminTotals{
		RegisterOverSLA: NewScore(float64(counts.RegisterOverSLA)),
		RegisterTotal:   NewScore(float64(counts.ClaimTotal)),
		RegisterScore:   adminBand(registerPercent),

		PaymentOverSLA: NewScore(float64(counts.PaymentOverSLA)),
		PaymentTotal:   NewScore(float64(counts.PaymentTotal)),
		PaymentScore:   adminBand(paymentPercent),
	}
}

// trunc2 memotong ke dua desimal, meniru `TRUNC(x, 2)` Oracle.
//
// Ia MEMOTONG, bukan membulatkan — dan arah pemotongannya mengikuti tanda bilangannya,
// seperti `TRUNC` Oracle. Nilai di sini tidak pernah negatif, tetapi menuliskannya benar
// lebih murah daripada menjelaskan kelak mengapa ia hanya benar untuk bilangan positif.
func trunc2(value float64) float64 {
	return math.Trunc(value*100) / 100
}
