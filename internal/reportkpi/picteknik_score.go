package reportkpi

import "math"

// Perhitungan nilai tab KPI PIC Teknik.
//
// Seluruh rumus di berkas ini MENIRU langkah activity lama termasuk pembulatannya. Itu bukan
// kerewelan: `@divide(x,y,n)` pada Pega membulatkan ke `n` desimal SETIAP KALI dipanggil, dan
// rumus pembobotan memanggilnya tiga kali berturut-turut. Menyederhanakannya menjadi satu
// ekspresi mengubah angka pada digit terakhir — cukup untuk membuat uji kesetaraan melaporkan
// selisih yang tidak akan pernah dapat dijelaskan.

// PercentOf menghitung persentase yang dipakai mencari pita.
//
// Meniru `@if(total==0, 0, @divide(tercapai, total, 3) * 100)` pada keempat sub-activity.
//
// Perhatikan urutannya: pembulatan terjadi SEBELUM dikalikan 100, bukan sesudah. Jadi 2 dari
// 3 menjadi `0,667 × 100 = 66,7`, bukan `66,667`. Membalik urutannya menggeser hasilnya tepat
// di titik batas pita — dan titik batas itulah yang menentukan nilai seseorang naik atau
// turun satu angka.
//
// Total nol menghasilkan 0, bukan nilai kosong. Itu perilaku lama, dan ia punya akibat yang
// perlu disadari: pada pita MENURUN, tidak ada data sama sekali menghasilkan nilai TERTINGGI.
func PercentOf(total, achieved float64) float64 {
	if total == 0 {
		return 0
	}
	return roundTo(achieved/total, 3) * 100
}

// WeightedScoreOf menghitung nilai berbobot, meniru `GetBobotNilaiKPIPNC` langkah demi langkah.
//
//	langkah 1  nilai := round(round(tercapai/total, 2) * 100 / 20, 2)
//	langkah 2  bobot := round(bobot/5, 2)
//	langkah 3  hasil := round(nilai * bobot, 2)
//
// Hanya komponen Progress yang melewatinya, dengan bobot 15 — sehingga hasilnya berskala 0–15,
// BUKAN 1–5 seperti nilai pita. Keduanya hidup berdampingan pada baris yang sama di sistem
// lama, disimpan di dua kolom berbeda, dan keduanya dibawa ke sini.
//
// Pembagian nol dijaga: rule lama akan melemparkan galat di sana.
func WeightedScoreOf(total, achieved, weight float64) Score {
	if total == 0 {
		return EmptyScore()
	}
	value := roundTo(roundTo(achieved/total, 2)*100/20, 2)
	factor := roundTo(weight/5, 2)
	return NewScore(roundTo(value*factor, 2))
}

// ScoreRow merakit satu baris penilaian dari cacahnya dan pita yang berlaku.
//
// # Satu tambalan dari sistem lama yang ikut dibawa
//
// Pada komponen Progress, activity lama memaksa hasilnya ketika persentasenya mencapai 100:
//
//	ClaimNo     := @if(bobotprogress >= 100, 99, bobotprogress)   // persen DITULIS 99
//	ClaimAmount := @if(bobotprogress >= 100, 5,  <nilai pita>)    // nilai DIPAKSA 5
//
// Tambalan itu hanya masuk akal bila pencarian pitanya sendiri memberi jawaban yang keliru di
// sana — dan memang begitu: pita Progress MENURUN, sehingga 100% tepat waktu jatuh ke pita
// 35–100 dan bernilai 1. Keduanya direplikasi apa adanya (`P-5`).
//
// Akibatnya dua PIC dapat sama-sama tampil "99%" dengan nilai 5 dan 1. Itu bukan cacat di
// sini; itu perilaku yang sedang berjalan hari ini, dan ia dinyatakan di
// `PICTeknikPlannedDifferences`.
func ScoreRow(component PICComponent, pic string, total, achieved float64, bands []Band) PICRow {
	percent := PercentOf(total, achieved)

	row := PICRow{
		PIC:       pic,
		Component: component.Code,
		Label:     component.Label,
		Total:     total,
		Achieved:  achieved,
	}

	capped := component.Code == PICComponentProgress && percent >= 100
	if capped {
		row.Percent = NewScore(99)
		row.Value = NewScore(5)
		return row
	}

	row.Percent = NewScore(percent)
	if band, found := BandFor(bands, percent); found {
		row.Value = NewScore(band.Value)
	} else {
		// Pita tidak ditemukan berarti nilainya KOSONG, bukan nol.
		//
		// Di sistem lama `GetNilai.pxResults(1)` pada hasil kosong menghasilkan properti yang
		// tidak ter-set — yang kemudian tampil sebagai sel kosong, bukan sebagai angka 0.
		// Membedakan keduanya penting: 0 bukan salah satu nilai yang sah pada skala 1–5,
		// sehingga menggambarnya justru menyamarkan pita yang berlubang.
		row.Value = EmptyScore()
	}
	return row
}

// AggregateLeader merakit baris Leader dari seluruh kartu skor PIC.
//
// # Bagaimana sistem lama melakukannya
//
// Tiap sub-activity MENJUMLAHKAN ke satu baris `TempKPILeader` yang sama:
//
//	Country     += total       CountryID   += tercapai
//	ClaimNo     += persen      ClaimAmount += nilai
//
// lalu `PNCReportKPI_act` membaginya dengan jumlah PIC dan membatasinya:
//
//	ClaimNo     := @if(jumlah/pic >= 100, 100, jumlah/pic)
//	ClaimAmount := @round(@if(jumlah/pic >= 5, 5, jumlah/pic))
//
// Perhatikan dua hal yang berbeda antara keduanya: persen dibatasi di 100 dan TIDAK
// dibulatkan, sedangkan nilai dibatasi di 5 DAN dibulatkan ke bilangan bulat. Keduanya
// direplikasi apa adanya.
//
// # Yang TIDAK dirakit di sini
//
// Sistem lama masih punya satu lapis agregasi lagi — per kelompok tim, dengan pembagi 2, 4,
// dan 5 pada tempat yang berbeda. Syarat kapan masing-masing pembagi berlaku tidak dapat
// ditentukan dari export, sehingga lapis itu **tidak dibangun** alih-alih ditebak. Lihat
// `PICTeknikPlannedDifferences`.
func AggregateLeader(scorecards []PICScorecard) PICScorecard {
	leader := PICScorecard{PIC: "Leader", Leader: true}
	count := float64(len(scorecards))

	for _, component := range picComponents {
		row := PICRow{
			PIC:       leader.PIC,
			Component: component.Code,
			Label:     component.Label,
		}

		var percentSum, valueSum float64
		var anyPercent, anyValue bool

		for _, card := range scorecards {
			for _, source := range card.Rows {
				if source.Component != component.Code {
					continue
				}
				row.Total += source.Total
				row.Achieved += source.Achieved
				if source.Percent.Present {
					percentSum += source.Percent.Value
					anyPercent = true
				}
				if source.Value.Present {
					valueSum += source.Value.Value
					anyValue = true
				}
			}
		}

		// Tanpa satu pun PIC, tidak ada yang dapat dirata-ratakan. Membagi dengan nol di sini
		// akan menghasilkan angka yang tampak sah, dan itu lebih buruk daripada sel kosong.
		if count == 0 || !anyPercent {
			row.Percent = EmptyScore()
		} else {
			row.Percent = NewScore(math.Min(percentSum/count, 100))
		}

		if count == 0 || !anyValue {
			row.Value = EmptyScore()
		} else {
			row.Value = NewScore(math.Round(math.Min(valueSum/count, 5)))
		}

		leader.Rows = append(leader.Rows, row)
	}

	return leader
}

// roundTo membulatkan ke `digits` desimal, meniru `@divide(...,digits)` pada Pega.
//
// Pega membulatkan setengah MENJAUHI nol, sama seperti math.Round di Go — bukan setengah ke
// genap seperti sebagian pustaka lain. Perbedaan itu hanya muncul pada angka yang tepat di
// tengah, dan tepat di sanalah nilai seseorang berpindah pita.
func roundTo(value float64, digits int) float64 {
	factor := math.Pow(10, float64(digits))
	return math.Round(value*factor) / factor
}
