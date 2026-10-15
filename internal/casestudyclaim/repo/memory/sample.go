package memory

import (
	"time"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/platform/money"
)

// NewSampleStore membentuk penyimpanan berisi baris contoh.
//
// Dipakai saat aplikasi berjalan tanpa Oracle. Seluruh isinya KARANGAN — tidak ada satu
// pun nomor polis, nama tertanggung, atau nomor klaim nyata (`D-69`).
func NewSampleStore() *Store { return NewRepo(SampleRecords()...) }

// SampleRecords adalah sepuluh klaim contoh.
//
// # Lima di antaranya sengaja TIDAK muncul
//
// Penyimpanan contoh yang seluruh barisnya lolos tidak membuktikan apa pun: penyaring yang
// rusak pun akan tampak benar. Kelima baris berikut ada justru supaya penyaringnya dapat
// dilihat bekerja, dan masing-masing tertolak oleh sebab yang BERBEDA:
//
//	STD-0006  nilai settlement terbesarnya di BAWAH ambang
//	STD-0007  totalnya di ATAS ambang, tetapi tidak ada SATU baris pun yang melampauinya
//	STD-0008  tahun registrasinya di luar rentang yang biasa dipakai saat mencoba
//	STD-0009  kode bisnisnya termasuk yang DIKELUARKAN dari cakupan NONMBU
//	STD-0010  Group Panel-nya tidak termasuk cakupan mana pun selain "semua"
//
// STD-0007 yang paling perlu dibaca. Totalnya Rp 8 miliar — di atas ambang — tetapi
// terpecah menjadi dua baris settlement Rp 4 miliar, dan `EXISTS` di kueri lama menguji
// SATU baris, bukan jumlahnya. Tanpa baris seperti ini, keliru membaca ambang sebagai
// "jumlah seluruh settlement" akan lolos dari setiap uji.
func SampleRecords() []Record {
	return []Record{
		{
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:     "STD-0001",
				PolicyNumber:    "POL-CONTOH-0001",
				InsuredName:     "PT Contoh Marine Sejahtera",
				BusinessName:    "Marine Cargo",
				PolicyPeriod:    "2024",
				ClaimMonth:      "03",
				LossDate:        date(2024, 3, 11),
				BusinessSource:  "Broker Contoh",
				ReinsurerRole:   casestudyclaim.ReinsurerRoleOf("1"),
				CauseOfLoss:     "Kerusakan muatan dalam pengangkutan",
				TSI:             rupiah(25_000_000_000),
				ASMSharePercent: percent(400_000), // 40%
				Deductible:      rupiah(500_000_000),
				ASMShareValue:   rupiah(3_200_000_000),
				ClaimValue100:   rupiah(8_000_000_000),
				AdjusterFee:     rupiah(120_000_000),
				NetClaim100:     rupiah(7_380_000_000),
				NetClaimASM:     rupiah(2_952_000_000),
				LackOfDoc:       rupiah(0),
				BranchName:      "Kantor Pusat",
				ClaimStatus:     casestudyclaim.ClaimStatusOf("0"),
				Chronology:      "Kontainer terguling saat bongkar muat di pelabuhan.",
				Remark:          "",
			},
			GroupPanel:        "004",
			BusinessCode:      "10200",
			StatusCode:        "0",
			LargestAdjustment: money.FromRupiah(8_000_000_000),
		},
		{
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:     "STD-0002",
				PolicyNumber:    "POL-CONTOH-0002",
				InsuredName:     "Budi Contoh",
				BusinessName:    "Personal Accident",
				PolicyPeriod:    "2024",
				ClaimMonth:      "07",
				LossDate:        date(2024, 7, 2),
				BusinessSource:  "Agen Contoh",
				ReinsurerRole:   casestudyclaim.ReinsurerRoleOf("2"),
				CauseOfLoss:     "Kecelakaan lalu lintas",
				TSI:             rupiah(10_000_000_000),
				ASMSharePercent: percent(1_000_000), // 100%
				Deductible:      rupiah(0),
				ASMShareValue:   rupiah(6_000_000_000),
				ClaimValue100:   rupiah(6_000_000_000),
				AdjusterFee:     rupiah(75_000_000),
				NetClaim100:     rupiah(5_925_000_000),
				NetClaimASM:     rupiah(5_925_000_000),
				LackOfDoc:       rupiah(0),
				BranchName:      "Cabang Contoh Satu",
				ClaimStatus:     casestudyclaim.ClaimStatusOf("1"),
				Chronology:      "Tertanggung mengalami kecelakaan tunggal.",
				Remark:          "Sudah ditelaah komite, dokumen lengkap.",
			},
			GroupPanel:        "002",
			BusinessCode:      "10300",
			StatusCode:        "1",
			LargestAdjustment: money.FromRupiah(6_000_000_000),
		},
		{
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:    "STD-0003",
				PolicyNumber:   "POL-CONTOH-0003",
				InsuredName:    "Citra Contoh",
				BusinessName:   "Travel",
				PolicyPeriod:   "2025",
				ClaimMonth:     "01",
				LossDate:       date(2025, 1, 18),
				BusinessSource: "Kanal Digital Contoh",
				ReinsurerRole:  casestudyclaim.ReinsurerRoleOf("F"),
				CauseOfLoss:    "Pembatalan perjalanan",
				TSI:            rupiah(9_000_000_000),

				// Ketiga nilai di bawah sengaja KOSONG, bukan nol.
				//
				// Di basis data itu terjadi ketika kolom sumbernya NULL pada seluruh baris
				// settlement: `SUM` atas kolom yang seluruhnya NULL mengembalikan NULL.
				// Layar menggambarnya sebagai tanda hubung, dan berkas CSV sebagai sel
				// kosong — bukan sebagai "Rp 0,00", yang akan ikut terhitung dalam
				// penjumlahan di pengolah angka.
				ASMSharePercent: nil,
				Deductible:      nil,
				LackOfDoc:       nil,

				ASMShareValue: rupiah(5_500_000_000),
				ClaimValue100: rupiah(5_500_000_000),
				AdjusterFee:   rupiah(0),
				NetClaim100:   rupiah(5_500_000_000),
				NetClaimASM:   rupiah(5_500_000_000),
				BranchName:    "Cabang Contoh Dua",
				ClaimStatus:   casestudyclaim.ClaimStatusOf("3"),
				Chronology:    "Perjalanan dibatalkan sepihak oleh penyelenggara.",
				Remark:        "",
			},
			GroupPanel:        "005",
			BusinessCode:      "10400",
			StatusCode:        "3",
			LargestAdjustment: money.FromRupiah(5_500_000_000),
		},
		{
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:     "STD-0004",
				PolicyNumber:    "POL-CONTOH-0004",
				InsuredName:     "PT Contoh Karya Bangun",
				BusinessName:    "Bonding",
				PolicyPeriod:    "2025",
				ClaimMonth:      "05",
				LossDate:        date(2025, 5, 9),
				BusinessSource:  "Bank Contoh",
				ReinsurerRole:   casestudyclaim.ReinsurerRoleOf("1"),
				CauseOfLoss:     "Wanprestasi kontraktor",
				TSI:             rupiah(40_000_000_000),
				ASMSharePercent: percent(600_000), // 60%
				Deductible:      rupiah(1_000_000_000),
				ASMShareValue:   rupiah(7_200_000_000),
				ClaimValue100:   rupiah(12_000_000_000),
				AdjusterFee:     rupiah(250_000_000),
				NetClaim100:     rupiah(10_750_000_000),
				NetClaimASM:     rupiah(6_450_000_000),
				LackOfDoc:       rupiah(0),
				BranchName:      "Kantor Pusat",
				ClaimStatus:     casestudyclaim.ClaimStatusOf("0"),
				Chronology:      "Pekerjaan terhenti dan jaminan pelaksanaan dicairkan.",
				Remark:          "",
			},
			GroupPanel:        "003",
			BusinessCode:      "10076",
			StatusCode:        "0",
			LargestAdjustment: money.FromRupiah(12_000_000_000),
		},
		{
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:     "STD-0005",
				PolicyNumber:    "POL-CONTOH-0005",
				InsuredName:     "PT Contoh Aneka Usaha",
				BusinessName:    "Aneka",
				PolicyPeriod:    "2023",
				ClaimMonth:      "11",
				LossDate:        date(2023, 11, 24),
				BusinessSource:  "Agen Contoh",
				ReinsurerRole:   casestudyclaim.ReinsurerRoleOf("X"), // kode asing -> "-"
				CauseOfLoss:     "Kebakaran gudang",
				TSI:             rupiah(30_000_000_000),
				ASMSharePercent: percent(250_000), // 25%
				Deductible:      rupiah(700_000_000),
				ASMShareValue:   rupiah(1_750_000_000),
				ClaimValue100:   rupiah(7_000_000_000),
				AdjusterFee:     rupiah(90_000_000),
				NetClaim100:     rupiah(6_210_000_000),
				NetClaimASM:     rupiah(1_552_500_000),
				LackOfDoc:       rupiah(0),
				BranchName:      "Cabang Contoh Tiga",
				ClaimStatus:     casestudyclaim.ClaimStatusOf("1"),
				Chronology:      "Api merambat dari bangunan sebelah.",
				Remark:          "Menunggu laporan adjuster final.",
			},
			GroupPanel:        "003",
			BusinessCode:      "10999",
			StatusCode:        "1",
			LargestAdjustment: money.FromRupiah(7_000_000_000),
		},

		// ── Lima baris yang seharusnya TIDAK muncul ─────────────────────────────────

		{
			// Tertolak AMBANG: nilai terbesarnya Rp 4,9 miliar.
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:   "STD-0006",
				PolicyNumber:  "POL-CONTOH-0006",
				InsuredName:   "PT Contoh Properti Jaya",
				BusinessName:  "Fire",
				PolicyPeriod:  "2024",
				ClaimMonth:    "09",
				ClaimValue100: rupiah(4_900_000_000),
				ClaimStatus:   casestudyclaim.ClaimStatusOf("0"),
			},
			GroupPanel:        "006",
			BusinessCode:      "10500",
			StatusCode:        "0",
			LargestAdjustment: money.FromRupiah(4_900_000_000),
		},
		{
			// Tertolak AMBANG meski TOTALNYA Rp 8 miliar: dua baris @ Rp 4 miliar, dan
			// `EXISTS` menguji SATU baris. Inilah saksi yang membedakan pembacaan ambang
			// yang benar dari pembacaan "jumlah seluruh settlement".
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:   "STD-0007",
				PolicyNumber:  "POL-CONTOH-0007",
				InsuredName:   "PT Contoh Dua Termin",
				BusinessName:  "Aneka",
				PolicyPeriod:  "2024",
				ClaimMonth:    "04",
				ClaimValue100: rupiah(8_000_000_000),
				ClaimStatus:   casestudyclaim.ClaimStatusOf("0"),
			},
			GroupPanel:        "009",
			BusinessCode:      "10600",
			StatusCode:        "0",
			LargestAdjustment: money.FromRupiah(4_000_000_000),
		},
		{
			// Tertolak TAHUN pada rentang 2024–2025.
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:   "STD-0008",
				PolicyNumber:  "POL-CONTOH-0008",
				InsuredName:   "PT Contoh Lama",
				BusinessName:  "Marine Cargo",
				PolicyPeriod:  "2022",
				ClaimMonth:    "02",
				ClaimValue100: rupiah(9_000_000_000),
				ClaimStatus:   casestudyclaim.ClaimStatusOf("1"),
			},
			GroupPanel:        "004",
			BusinessCode:      "10700",
			StatusCode:        "1",
			LargestAdjustment: money.FromRupiah(9_000_000_000),
		},
		{
			// Tertolak pada cakupan NONMBU: kode bisnisnya ada di daftar yang DIKELUARKAN.
			// Ia tetap muncul bila Bisnis dibiarkan kosong — dan itulah yang membuatnya
			// berguna: perbedaan antara "semua" dan "NONMBU" menjadi terlihat.
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:   "STD-0009",
				PolicyNumber:  "POL-CONTOH-0009",
				InsuredName:   "PT Contoh Dikecualikan",
				BusinessName:  "Aneka",
				PolicyPeriod:  "2024",
				ClaimMonth:    "06",
				ClaimValue100: rupiah(20_000_000_000),
				ClaimStatus:   casestudyclaim.ClaimStatusOf("0"),
			},
			GroupPanel:        "003",
			BusinessCode:      "10145",
			StatusCode:        "0",
			LargestAdjustment: money.FromRupiah(20_000_000_000),
		},
		{
			// Group Panel di luar keempat cakupan. Muncul hanya bila Bisnis dibiarkan
			// kosong — bukti bahwa "semua" benar-benar berarti tanpa penyaring.
			Row: casestudyclaim.CaseStudyRow{
				ClaimNumber:   "STD-0010",
				PolicyNumber:  "POL-CONTOH-0010",
				InsuredName:   "PT Contoh Panel Lain",
				BusinessName:  "Lain-lain",
				PolicyPeriod:  "2025",
				ClaimMonth:    "08",
				ClaimValue100: rupiah(15_000_000_000),
				ClaimStatus:   casestudyclaim.ClaimStatusOf("0"),
			},
			GroupPanel:        "007",
			BusinessCode:      "10800",
			StatusCode:        "0",
			LargestAdjustment: money.FromRupiah(15_000_000_000),
		},
	}
}

// rupiah membentuk penunjuk nilai uang dari angka rupiah utuh.
//
// Ia mengembalikan PENUNJUK karena nil dan nol berbeda artinya pada modul ini — lihat
// catatan pada CaseStudyRow.
func rupiah(amount int64) *money.Money {
	value := money.FromRupiah(amount)
	return &value
}

// percent membentuk penunjuk persentase dikali 10.000.
//
// `percent(1000000)` adalah 100%, `percent(600000)` adalah 60%, dan `percent(4000)`
// adalah 0,4%. Bentuk ini dipakai supaya empat desimalnya utuh tanpa pecahan biner
// (`D-51`).
func percent(e4 int64) *int64 { return &e4 }

// date membentuk penunjuk tanggal UTC.
//
// Tetap, tidak dibaca dari jam sistem: data contoh yang berubah setiap kali dijalankan
// membuat uji yang bergantung padanya gagal pada hari yang tidak terduga.
func date(year int, month time.Month, day int) *time.Time {
	value := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &value
}
