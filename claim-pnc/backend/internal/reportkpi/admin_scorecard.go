package reportkpi

// AdminTotals adalah angka MENTAH kartu skor, apa adanya dari basis data.
//
// # Kenapa ia terpisah dari AdminScorecard
//
// Supaya penyusunan kartunya — urutan metrik, labelnya, bentuk angkanya, dan kesimpulan
// tercapai atau tidak — hidup di lapisan domain dan dapat diuji TANPA basis data. Pengisi
// seam hanya mengisi angka; ia tidak memutuskan bagaimana kartunya tersusun.
//
// Isian yang tidak berlaku pada sebuah kelompok dibiarkan kosong. Kosong di sini berarti
// "tidak dihitung", dan itu berbeda dari nol — pada kartu skor, nol berarti tidak satu pun
// klaim melewati SLA, yaitu hasil terbaik.
type AdminTotals struct {
	// Keenam isian NON-MBU.
	LeaderOverSLA  Score
	LeaderTotal    Score
	LeaderPercent  Score
	LeaderScore    Score
	LeaderSubtotal Score

	MemberOverSLA  Score
	MemberTotal    Score
	MemberPercent  Score
	MemberScore    Score
	MemberSubtotal Score

	QuantitativeTotal Score
	AchievementRatio  Score

	// Keenam isian PA.
	RegisterOverSLA Score
	RegisterTotal   Score
	RegisterScore   Score
	PaymentOverSLA  Score
	PaymentTotal    Score
	PaymentScore    Score
}

// BuildScorecard menyusun kartu skor dari angka mentah.
//
// Urutan metriknya mengikuti urutan kolom kueri lama, yang sama dengan urutan baris pada
// kartu skor layar lama. Ia ditulis di sini — bukan di lapisan transport — supaya berkas
// ekspor dan layar tidak dapat menyusunnya dengan urutan yang berbeda.
func BuildScorecard(group AdminGroup, rng DateRange, totals AdminTotals) AdminScorecard {
	card := AdminScorecard{
		Group:       group,
		Identity:    AdminIdentityFor(group),
		EffectiveOn: effectiveOn(rng),
	}

	if group == AdminGroupPA {
		card.Metrics = paMetrics(totals)
		// Kartu skor PA TIDAK punya baris kesimpulan, dan itu bukan kelalaian: kuerinya
		// memang tidak menghitung rasio pencapaian maupun teks tercapai/tidak. Mengarangnya
		// akan menampilkan penilaian yang tidak pernah dibuat Pega.
		return card
	}

	card.Metrics = nonMBUMetrics(totals)
	card.Achievement = achievementOf(totals.AchievementRatio)
	return card
}

// nonMBUMetrics menyusun keempat belas baris kartu skor NON-MBU.
//
// Urutannya berpasangan — leader lebih dulu, lalu member, lalu gabungannya — persis seperti
// kartu skor layar lama membacanya dari kiri ke kanan.
func nonMBUMetrics(t AdminTotals) []Metric {
	return []Metric{
		{MetricLeaderOverSLA, "PROSES REGISTRASI KLAIM NON MBU LEADER > SLA", t.LeaderOverSLA, FormatCount},
		{MetricLeaderTotal, "TOTAL KLAIM LEADER", t.LeaderTotal, FormatCount},
		{MetricLeaderPercent, "PERSENTASE LEADER", t.LeaderPercent, FormatPercent},
		{MetricLeaderScore, "NILAI LEADER SLA", t.LeaderScore, FormatScore},
		{MetricLeaderWeight, "BOBOT LEADER", NewScore(AdminLeaderWeight), FormatDecimal},
		{MetricLeaderSubtotal, "SUBTOTAL LEADER", t.LeaderSubtotal, FormatPercent},

		{MetricMemberOverSLA, "PROSES REGISTRASI KLAIM NON MBU MEMBER > SLA", t.MemberOverSLA, FormatCount},
		{MetricMemberTotal, "TOTAL KLAIM MEMBER", t.MemberTotal, FormatCount},
		{MetricMemberPercent, "PERSENTASE MEMBER", t.MemberPercent, FormatPercent},
		{MetricMemberScore, "NILAI MEMBER SLA", t.MemberScore, FormatScore},
		{MetricMemberWeight, "BOBOT MEMBER", NewScore(AdminMemberWeight), FormatDecimal},
		{MetricMemberSubtotal, "SUBTOTAL MEMBER", t.MemberSubtotal, FormatPercent},

		{MetricQuantitativeSum, "TOTAL KUANTITATIF", t.QuantitativeTotal, FormatPercent},
		{MetricAchievementRatio, "PENCAPAIAN KUANTITATIF", t.AchievementRatio, FormatDecimal},
	}
}

// paMetrics menyusun keenam baris kartu skor PA.
//
// Perhatikan kedua pembaginya BERBEDA, dan itu bukan kekeliruan pembacaan:
//
//	persentase registrasi  regist_klaim_pa     / total_klaim
//	persentase pembayaran  pembayaran_klaim_pa / total_klaim      ← pembagi yang SAMA
//
// sementara `total_pembayaran_pa` — yang tampak seperti pembagi yang seharusnya dipakai
// baris kedua — TIDAK dipakai menghitung apa pun. Ia hanya ditampilkan sebagai
// "TOTAL KLAIM BAYAR", dan rentang tanggalnya terkunci pada 2023. Lihat reportkpi_admin.sql.
func paMetrics(t AdminTotals) []Metric {
	return []Metric{
		{MetricRegisterOverSLA, "REGIST KLAIM > SLA", t.RegisterOverSLA, FormatCount},
		{MetricRegisterTotal, "TOTAL KLAIM REGIST", t.RegisterTotal, FormatCount},
		{MetricRegisterScore, "PENCAPAIAN KUANTITATIF REGIST", t.RegisterScore, FormatScore},

		{MetricPaymentOverSLA, "PEMBAYARAN KLAIM > SLA", t.PaymentOverSLA, FormatCount},
		{MetricPaymentTotal, "TOTAL KLAIM BAYAR", t.PaymentTotal, FormatCount},
		{MetricPaymentScore, "PENCAPAIAN KAUNTITATIF PEMBAYARAN", t.PaymentScore, FormatScore},
	}
}

// achievementOf menerjemahkan rasio pencapaian menjadi kesimpulan.
//
// Ambangnya `< 1`, persis `CASE` pada `GetDataKPIAdmin-SQL.xml`. Teksnya pun apa adanya,
// termasuk salah ketik yang mungkin ada di dalamnya (`D-13`).
//
// Rasio yang TIDAK dapat dihitung — karena tidak ada satu pun klaim pada periode itu —
// menghasilkan kesimpulan KOSONG, bukan "TIDAK TERCAPAI TARGET". Keduanya berbeda: yang
// pertama berarti belum ada yang dapat dinilai, yang kedua berarti dinilai dan gagal.
// Di Pega perbedaan itu tidak ada, karena kuerinya gagal lebih dulu dengan ORA-01476.
func achievementOf(ratio Score) string {
	if !ratio.Present {
		return ""
	}
	if ratio.Value < 1 {
		return AchievementMissed
	}
	return AchievementReached
}

// effectiveOn menyusun teks TANGGAL EFEKTIF dari rentang periode.
//
// Bentuknya `dd/mm/yyyy - dd/mm/yyyy`, meniru perangkaian di dalam SELECT kueri lama yang
// menggabungkan dua tanggal berformat `dd/MM/yyyy` dengan " - ".
//
// Perangkaiannya pindah ke Go, dan itu disengaja: melakukannya di SQL berarti pemformatan
// tanggal kembali ke basis data, yang `D-20` justru keluarkan dari sana.
func effectiveOn(rng DateRange) string {
	from, to := displayDate(rng.From), displayDate(rng.To)
	if from == "" || to == "" {
		return ""
	}
	return from + " - " + to
}

// displayDate mengubah `YYYY-MM-DD` menjadi `dd/mm/yyyy`.
//
// Ia memotong dengan posisi karakter, bukan mengurai tanggal, karena masukannya sudah
// dipastikan sah oleh NewAdminQuery — dan mengurai ulang di sini hanya menambah jalur galat
// yang tidak dapat terjadi.
func displayDate(iso string) string {
	if len(iso) != len("2006-01-02") {
		return ""
	}
	return iso[8:10] + "/" + iso[5:7] + "/" + iso[0:4]
}
