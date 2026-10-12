package inboxosclaimpercabang

import (
	"sort"
	"time"

	"claim-pnc/internal/platform/money"
)

// Berkas ini memuat panel ringkasan di atas grid — kartu angka, sebaran umur, dan rincian
// per COB maupun per sumber bisnis.
//
// # Ia TIDAK ada di Pega
//
// Layar lama hanya punya judul, tombol ekspor, dan grid. Panel ini diminta Work Owner
// (2026-10-08), sehingga ia **tidak punya pembanding untuk uji kesetaraan** — keadaan yang
// sama dengan `F-3` dan `S-5` (`D-56`). Yang dapat diuji hanyalah bahwa angkanya konsisten
// dengan grid di bawahnya: jumlah berkas pada kartu WAJIB sama dengan jumlah baris grid, dan
// jumlah nilainya sama dengan jumlah kolom "Reserve Claim ASM Share".
//
// # Dua nilai uang, dan keduanya memang berbeda
//
// Work Owner meminta total porsi treaty **OR** per cabang. Ia dibawa apa adanya, tetapi tidak
// berdiri sendiri: diukur langsung, `treaty_loss@asmd` hanya memuat **19 baris** berawalan
// `PNC-` dari 10.152 baris, sehingga **49 dari 50 cabang** akan menampilkan Rp 0.
//
// Karena itu kartu nilai estimasi ikut ditampilkan di sebelahnya. Tanpa itu panel ini akan
// terbaca sebagai layar yang rusak, padahal angkanya memang nol — dan nol itu sudah terjadi
// di Pega juga, karena rumus dan penyaringnya disalin persis dari kueri ekspornya.

// AgeBucket adalah satu pita umur pada batang sebaran.
type AgeBucket struct {
	// Label adalah teks yang dibaca pengguna, mis. "Sampai 6 bulan".
	Label string

	// Claims adalah jumlah berkas pada pita ini.
	Claims int

	// Value adalah jumlah nilai estimasi berkas pada pita ini.
	Value money.Money
}

// SummaryGroup adalah satu baris rincian — per COB atau per sumber bisnis.
type SummaryGroup struct {
	// Name adalah nama kelompoknya. Kosong diganti teks penanda oleh pemanggil, bukan di
	// sini: domain tidak memutuskan bagaimana sesuatu digambar.
	Name string

	Claims int
	Value  money.Money

	// OverTwoYears adalah berapa di antaranya sudah lewat dua tahun.
	//
	// Ia ada supaya rincian per COB dapat menunjuk kelompok mana yang menua, bukan hanya
	// mana yang besar nilainya. Keduanya pertanyaan yang berbeda.
	OverTwoYears int
}

// Summary adalah seluruh isi panel ringkasan satu cabang.
type Summary struct {
	// AsOf adalah kapan angkanya dibaca, dipakai baris "Posisi <tanggal>".
	//
	// Ia waktu PEMBACAAN, bukan tanggal data. Tidak ada tanggal posisi pada data ini: baris
	// klaim berubah kapan saja, dan tidak ada snapshot harian yang dapat dirujuk.
	AsOf time.Time

	// TotalClaims WAJIB sama dengan jumlah seluruh baris grid cabang ini.
	TotalClaims int

	// EstimationTotal adalah jumlah nilai estimasi, dihitung sama persis dengan kolom grid.
	EstimationTotal money.Money

	// ReserveOR adalah total porsi treaty OR, rumusnya disalin dari kueri ekspor Pega.
	ReserveOR money.Money

	// ReserveORAvailable bernilai false ketika DB Link `@asmd` tidak dapat dihubungi.
	//
	// Dibedakan dari "nol" dengan sengaja: Rp 0 adalah jawaban yang sah dan memang lazim di
	// sini, sedangkan tidak terbaca adalah keadaan lain yang menuntut tindakan lain. Layar
	// yang menampilkan keduanya sebagai "Rp 0" menyembunyikan gangguan jaringan.
	ReserveORAvailable bool

	// OverTwoYears adalah jumlah berkas yang umurnya melewati dua tahun.
	OverTwoYears int

	// Buckets adalah keempat pita umur, selalu lengkap dan selalu berurutan.
	//
	// Pita kosong TETAP disertakan. Batang sebaran yang pitanya hilang akan berubah bentuk
	// dari hari ke hari, dan pembacanya tidak dapat tahu apakah pita itu nol atau tidak ada.
	Buckets []AgeBucket

	// ByCOB dan BySource terurut menurun menurut nilai, lalu menurut nama.
	ByCOB    []SummaryGroup
	BySource []SummaryGroup
}

// SummaryRow adalah satu baris mentah yang diringkas.
//
// Ia sengaja sempit: panel hanya butuh empat hal per klaim, dan mengambil baris grid utuh
// berarti membayar seluruh gabungannya untuk angka yang tidak digambar.
type SummaryRow struct {
	RegisterDate    *time.Time
	BusinessName    string
	BusinessSource  string
	EstimationValue money.Money
}

// Ambang pita umur, dalam hari kalender WIB.
//
// Dinyatakan sebagai konstanta, bukan angka di tengah kode, supaya label dan ambangnya tidak
// dapat berselisih. "Dua tahun" di sini 730 hari — bukan dua kali 365 yang dihitung ulang di
// tempat lain.
const (
	halfYearDays = 183
	oneYearDays  = 365
	TwoYearsDays = 730
)

// Label pita umur. Teksnya dipakai layar apa adanya.
const (
	BucketUpToSixMonths = "Sampai 6 bulan"
	BucketSixToTwelve   = "6–12 bulan"
	BucketOneToTwoYears = "1–2 tahun"
	BucketOverTwoYears  = "Di atas 2 tahun"
)

// Summarize meringkas baris mentah menjadi isi panel.
//
// # Kenapa peringkasan ada di Go, bukan di SQL
//
// Karena pengelompokan umur menuntut pemotongan tanggal, dan padanan portabel `TRUNC` tidak
// memangkas jam di Oracle — alasan yang sama yang membuat kolom Aging dihitung di sini
// (AgingDaysSince). Mengelompokkan di SQL berarti menulis aturan umur untuk kedua kalinya,
// dengan bentuk yang tidak dapat dibuat sama persis.
//
// Biayanya kecil dan terukur: outstanding per cabang paling banyak 83 baris, rerata 9,2.
func Summarize(rows []SummaryRow, now time.Time) Summary {
	summary := Summary{
		AsOf:        now,
		TotalClaims: len(rows),
		Buckets: []AgeBucket{
			{Label: BucketUpToSixMonths},
			{Label: BucketSixToTwelve},
			{Label: BucketOneToTwoYears},
			{Label: BucketOverTwoYears},
		},
	}

	perCOB := map[string]*SummaryGroup{}
	perSource := map[string]*SummaryGroup{}

	for _, row := range rows {
		age := AgingDaysSince(row.RegisterDate, now)
		slot := bucketOf(age)

		summary.EstimationTotal += row.EstimationValue
		summary.Buckets[slot].Claims++
		summary.Buckets[slot].Value += row.EstimationValue

		old := age > TwoYearsDays
		if old {
			summary.OverTwoYears++
		}

		tally(perCOB, row.BusinessName, row.EstimationValue, old)
		tally(perSource, row.BusinessSource, row.EstimationValue, old)
	}

	summary.ByCOB = sorted(perCOB)
	summary.BySource = sorted(perSource)
	return summary
}

// bucketOf memilih pita umur sebuah klaim.
//
// Batasnya TERTUTUP di atas: 183 hari masih "sampai 6 bulan", 184 sudah bukan. Ditulis
// berurutan dari yang termuda supaya tidak ada umur yang jatuh ke dua pita sekaligus.
func bucketOf(ageDays int) int {
	switch {
	case ageDays <= halfYearDays:
		return 0
	case ageDays <= oneYearDays:
		return 1
	case ageDays <= TwoYearsDays:
		return 2
	default:
		return 3
	}
}

// tally menambahkan satu klaim ke kelompoknya.
func tally(into map[string]*SummaryGroup, name string, value money.Money, old bool) {
	group, exists := into[name]
	if !exists {
		group = &SummaryGroup{Name: name}
		into[name] = group
	}

	group.Claims++
	group.Value += value
	if old {
		group.OverTwoYears++
	}
}

// sorted mengurutkan kelompok: nilai terbesar lebih dulu, lalu nama.
//
// Nama dipakai sebagai pemisah ketika nilainya sama. Tanpa itu urutannya mengikuti urutan
// penelusuran map Go, yang BERBEDA setiap kali dijalankan — dan panel yang barisnya berpindah
// sendiri pada setiap muat ulang tidak dapat dipercaya pembacanya.
func sorted(groups map[string]*SummaryGroup) []SummaryGroup {
	result := make([]SummaryGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Value != result[j].Value {
			return result[i].Value > result[j].Value
		}
		return result[i].Name < result[j].Name
	})

	return result
}
