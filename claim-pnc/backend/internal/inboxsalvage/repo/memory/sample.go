package memory

// Data contoh modul Inbox Salvage.
//
// # Aturan yang mengikat berkas ini
//
// TIDAK ADA data nasabah nyata di sini. Nomor klaim, nama barang, dan nama PIC seluruhnya
// karangan, dan tidak satu pun disalin dari produksi maupun dari export (`D-69`).
//
// # Apa yang ia harus buktikan
//
// Bukan "layarnya terisi" melainkan setiap aturan yang dapat salah:
//
//   - Ketiga belas daftar punya isi, sehingga tab yang kosong berarti penyaringnya salah,
//     bukan datanya habis.
//   - Daftar Salvage Outstanding dan pencacah "Outstanding" menghitung populasi yang
//     BERBEDA — selisih yang direplikasi (`P-5`) dan harus terlihat di layar.
//   - Daftar Request Balai Lelang berisi baris milik dua PIC yang berbeda, sehingga
//     penyaring "hanya milik saya" dapat gagal dengan terlihat.
//   - Satu baris ber-`NILAIAKSEP` NOL, supaya "Belum Terjual" tidak dapat lolos hanya
//     karena kolomnya kosong.
//   - Satu baris ber-catatan request terisi dan satu yang kosong, supaya kedua nilai
//     "Tipe Pengajuan" tergambar.

// SampleCallerPIC adalah PIC yang memiliki baris pada daftar Request Balai Lelang.
//
// Ia dipakai uji untuk memastikan penyaring "hanya milik saya" benar-benar menyaring:
// daftar itu berisi baris milik PIC ini DAN milik PIC lain, sehingga penyaring yang lupa
// dipasang akan terlihat sebagai baris tambahan, bukan sebagai daftar kosong.
const SampleCallerPIC = "SITIRAHAYU"

// sampleOtherPIC memiliki baris pada daftar yang sama, dan tidak boleh terlihat oleh
// SampleCallerPIC.
const sampleOtherPIC = "BUDISANTOSO"

// SampleToday adalah tanggal yang dianggap "hari ini" oleh data contoh.
//
// Ia TETAP, bukan jam mesin, supaya kolom "Aging" menghasilkan angka yang sama pada setiap
// pembukaan — kalau tidak, satu-satunya uji yang dapat ditulis adalah uji yang tidak
// memeriksa angkanya.
const SampleToday = "2026-09-25"

// NewSampleStore membentuk penyimpanan berisi data contoh.
func NewSampleStore() *Store {
	store := NewStore()
	store.SetNow(func() string { return SampleToday })
	store.Seed(sampleClaims(), sampleSalvages())
	return store
}

// sampleClaims memasok keluarga A dan B.
//
// Sebaran `STSSALVAGE`-nya disengaja:
//
//	kosong  2 baris  -> daftar Salvage Outstanding
//	"3"     2 baris  -> daftar Ekonomis, DAN pencacah "Outstanding"
//	"5"     1 baris  -> daftar TBA, DAN pencacah "Outstanding"
//	"4"     1 baris  -> daftar Tidak Ekonomis
//	"1"     1 baris  -> daftar Tidak Ada Salvage
//
// Dengan sebaran itu pencacah "Outstanding" menyebut 3 sementara daftarnya menampilkan 2 —
// selisih yang memang ada di Pega, dan yang sekarang dapat dilihat alih-alih dipercaya.
func sampleClaims() []Claim {
	return []Claim{
		{
			ClaimNo:      "PNC-2041",
			PIC:          SampleCallerPIC,
			BusinessName: "Property All Risk",
			LossDate:     "2026-08-14",
			ObjectName:   "Gudang Blok C",
			WorkStatus:   "Open",
		},
		{
			ClaimNo:      "PNC-2042",
			PIC:          sampleOtherPIC,
			BusinessName: "Marine Cargo",
			LossDate:     "2026-08-29",
			ObjectName:   "Kontainer 20 ft",
			WorkStatus:   "Pending-Survey",
		},
		{
			// Sudah selesai — TIDAK boleh muncul di daftar Salvage Outstanding, karena
			// hanya daftar itu yang menyaring status kerja.
			ClaimNo:      "PNC-2043",
			PIC:          SampleCallerPIC,
			BusinessName: "Aneka",
			LossDate:     "2026-07-02",
			ObjectName:   "Mesin Pendingin",
			WorkStatus:   "Resolved-Completed",
		},
		{
			ClaimNo:       "PNC-2044",
			PIC:           SampleCallerPIC,
			BusinessName:  "Property All Risk",
			LossDate:      "2026-09-01",
			ObjectName:    "Panel Listrik",
			SalvageStatus: "3",
			WorkStatus:    "Open",
		},
		{
			ClaimNo:       "PNC-2045",
			PIC:           sampleOtherPIC,
			BusinessName:  "Aneka",
			LossDate:      "2026-09-04",
			ObjectName:    "Rak Penyimpanan",
			SalvageStatus: "3",
			WorkStatus:    "Open",

			// Klaim yang sama juga punya nilai salvage di adjustment, sehingga ia muncul
			// di daftar Ekonomis DAN Salvage Buyback. Itu memang mungkin: kedua
			// penyaringnya tidak saling meniadakan.
			HasBuybackValue: true,
		},
		{
			ClaimNo:       "PNC-2046",
			PIC:           SampleCallerPIC,
			BusinessName:  "Marine Cargo",
			LossDate:      "2026-09-08",
			ObjectName:    "Drum Kimia",
			SalvageStatus: "5",
			WorkStatus:    "Open",
		},
		{
			ClaimNo:       "PNC-2047",
			PIC:           sampleOtherPIC,
			BusinessName:  "Fire",
			LossDate:      "2026-09-11",
			ObjectName:    "Atap Baja Ringan",
			SalvageStatus: "4",
			WorkStatus:    "Open",
		},
		{
			ClaimNo:       "PNC-2048",
			PIC:           SampleCallerPIC,
			BusinessName:  "Aneka",
			LossDate:      "2026-09-15",
			ObjectName:    "Perangkat Jaringan",
			SalvageStatus: "1",
			WorkStatus:    "Open",
		},
	}
}

// sampleSalvages memasok keluarga C.
//
// Sebaran `STSTRANSFER`-nya menyentuh setiap daftar keluarga C:
//
//	"1"  Balai Lelang           2 baris, satu terjual dan satu belum
//	"3"  Checker / Diterima     2 baris, satu ber-request dan satu tidak
//	"4"  Rejected Checker       1 baris
//	"5"  Salvage Ditolak        1 baris  <- di Pega daftarnya tidak pernah dijalankan
//	"7"  Request Balai Lelang   2 baris, PIC berbeda
//	"6"                         1 baris  <- hanya terhitung pencacah "Histori Salvage"
func sampleSalvages() []Salvage {
	return []Salvage{
		{
			SalvageID:      "101",
			ClaimNo:        "PNC-2044",
			InputDate:      "2026-09-02",
			PIC:            SampleCallerPIC,
			SalvageType:    "Besi Tua",
			Location:       "Gudang Cakung",
			Quantity:       "12",
			EstimateValue:  "4500000",
			Email:          "salvage.pusat@contoh.internal",
			Remark:         "Menunggu penilaian ulang",
			TransferStatus: "1",

			// Sudah laku — kolom "Status Lelang" harus berbunyi "Terjual".
			AcceptedValue: "5100000",
		},
		{
			SalvageID:      "102",
			ClaimNo:        "PNC-2045",
			InputDate:      "2026-09-03",
			PIC:            sampleOtherPIC,
			SalvageType:    "Kayu Bekas",
			Location:       "Gudang Bekasi",
			Quantity:       "40",
			EstimateValue:  "2750000",
			Email:          "salvage.pusat@contoh.internal",
			Remark:         "Lelang pertama tidak ada penawar",
			TransferStatus: "1",

			// NOL, bukan kosong. "Belum Terjual" di sini membuktikan aturannya memeriksa
			// nilainya, bukan sekadar keberadaannya.
			AcceptedValue: "0",
		},
		{
			SalvageID:      "103",
			ClaimNo:        "PNC-2046",
			InputDate:      "2026-09-09",
			PIC:            SampleCallerPIC,
			SalvageType:    "Drum Kosong",
			Location:       "Gudang Cakung",
			Quantity:       "25",
			EstimateValue:  "1800000",
			Email:          "salvage.pusat@contoh.internal",
			Remark:         "Menunggu keputusan checker",
			TransferStatus: "3",

			// Tanpa catatan request -> "Tipe Pengajuan" berbunyi "Pengajuan Baru".
			RequestNote:  "",
			RequestValue: "",
		},
		{
			SalvageID:      "104",
			ClaimNo:        "PNC-2047",
			InputDate:      "2026-09-12",
			PIC:            sampleOtherPIC,
			SalvageType:    "Baja Ringan",
			Location:       "Gudang Serpong",
			Quantity:       "8",
			EstimateValue:  "9200000",
			Email:          "salvage.pusat@contoh.internal",
			Remark:         "Nilai pengajuan di atas pasar",
			TransferStatus: "3",

			// Dengan catatan request -> "Request Balai Lelang".
			RequestNote:  "Balai lelang menawar lebih rendah",
			RequestValue: "7400000",
		},
		{
			SalvageID:      "105",
			ClaimNo:        "PNC-2043",
			InputDate:      "2026-09-16",
			PIC:            SampleCallerPIC,
			SalvageType:    "Mesin Rusak",
			Location:       "Gudang Cakung",
			Quantity:       "1",
			EstimateValue:  "15000000",
			Email:          "salvage.pusat@contoh.internal",
			Remark:         "Dikembalikan untuk dilengkapi foto",
			TransferStatus: "4",
		},
		{
			SalvageID:      "106",
			ClaimNo:        "PNC-2042",
			InputDate:      "2026-09-18",
			PIC:            sampleOtherPIC,
			SalvageType:    "Kontainer Bekas",
			Location:       "Depo Tanjung Priok",
			Quantity:       "2",
			EstimateValue:  "32000000",
			Email:          "salvage.pusat@contoh.internal",
			Remark:         "Ditolak, barang sudah dilepas tertanggung",
			TransferStatus: "5",
		},
		{
			SalvageID:      "107",
			ClaimNo:        "PNC-2041",
			InputDate:      "2026-09-20",
			PIC:            SampleCallerPIC,
			SalvageType:    "Panel Surya",
			Location:       "Gudang Cakung",
			Quantity:       "6",
			EstimateValue:  "11500000",
			Email:          "salvage.pusat@contoh.internal",
			Remark:         "Menunggu jawaban balai lelang",
			TransferStatus: "7",
			RequestNote:    "Minta penurunan nilai minimum",
			RequestValue:   "9000000",
		},
		{
			// PIC BERBEDA — baris ini tidak boleh terlihat oleh SampleCallerPIC pada
			// daftar Request Balai Lelang maupun pada pencacahnya.
			SalvageID:      "108",
			ClaimNo:        "PNC-2048",
			InputDate:      "2026-09-21",
			PIC:            sampleOtherPIC,
			SalvageType:    "Kabel Tembaga",
			Location:       "Gudang Bekasi",
			Quantity:       "150",
			EstimateValue:  "6300000",
			Email:          "salvage.pusat@contoh.internal",
			Remark:         "Menunggu jawaban balai lelang",
			TransferStatus: "7",
			RequestNote:    "Minta tambahan foto",
			RequestValue:   "5800000",
		},
		{
			// `STSTRANSFER` 6 tidak punya daftar sendiri. Ia ADA supaya pencacah "Histori
			// Salvage" — yang menghitung 1 atau 6 — menyebut angka yang berbeda dari
			// jumlah baris daftarnya, persis seperti di Pega.
			SalvageID:      "109",
			ClaimNo:        "PNC-2044",
			InputDate:      "2026-08-30",
			PIC:            SampleCallerPIC,
			SalvageType:    "Besi Tua",
			Location:       "Gudang Cakung",
			Quantity:       "3",
			EstimateValue:  "900000",
			Email:          "salvage.pusat@contoh.internal",
			Remark:         "Pengajuan lama",
			AcceptanceNo:   "AKS-2026-0091",
			TransferStatus: "6",
			AcceptedValue:  "1000000",
		},
	}
}
