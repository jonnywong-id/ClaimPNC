package memory

import (
	"time"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/money"
)

// NewSampleStore membentuk penyimpanan berisi contoh yang mencakup setiap perilaku layar.
//
// Isinya dipilih supaya setiap penyaring dan setiap aturan tampilan terbukti bekerja, bukan
// supaya layar terlihat penuh:
//
//	dua cabang berbeda            membuktikan batas data benar-benar menyaring
//	satu baris sudah selesai      membuktikan penyaring outstanding bekerja
//	satu baris tanpa registerdate membuktikan `registerdate IS NOT NULL` bekerja
//	satu baris berumur > 180 hari membuktikan ambang merah pertama
//	satu baris progres mandek     membuktikan ambang merah kedua, pada klaim yang MASIH MUDA
//	satu baris tanpa progres      membuktikan kolom kosong tidak menjadi tanggal awal zaman
//
// Baris kelima penting justru karena umurnya muda: bila kedua syarat merah selalu muncul
// bersama, uji tidak akan pernah membuktikan syarat kedua benar-benar dibaca.
func NewSampleStore() *Store {
	store := NewStore()

	// Kuncinya kode cabang RINCI (`OLDID`, 3 digit) — itu yang dikirim HCQ — dan nilainya
	// membawa kode yang dipakai klaim (`ID`, 6 digit). Keduanya diambil dari baris nyata di
	// `POOLDATA.BRANCH` supaya contoh ini tidak diam-diam memakai pasangan yang mustahil:
	//
	//	ID 100099  OLDID 078  CILEGON
	//	ID 100059  OLDID 007  BANDUNG
	//
	// Perbedaan panjangnya bukan hiasan. Ia yang membuat uji gagal seketika bila ada kode yang
	// diteruskan tanpa diterjemahkan lebih dulu.
	store.branches = map[string]inboxosclaimpercabang.Branch{
		"078": {Code: "100099", Name: "CILEGON"},
		"007": {Code: "100059", Name: "BANDUNG"},
	}

	// Tanggal contoh dipatok, bukan dihitung dari waktu berjalan. Umur yang bergerak setiap
	// hari akan membuat uji lulus hari ini dan gagal bulan depan tanpa satu baris pun
	// berubah.
	day := func(y int, m time.Month, d int) *time.Time {
		moment := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		return &moment
	}
	moment := func(y int, m time.Month, d, h int) *time.Time {
		value := time.Date(y, m, d, h, 0, 0, 0, time.UTC)
		return &value
	}

	store.factors = map[string][]string{
		// Berurut sesuai `idx_dominanfactor`, sebagaimana kueri mengembalikannya.
		"CLAIM-0002": {"Kelalaian Tertanggung", "Dokumen Tidak Lengkap"},
	}

	store.rows = []row{
		{
			outstanding: true,
			item: inboxosclaimpercabang.ExportRow{
				WorkItem: inboxosclaimpercabang.WorkItem{
					BranchName:      "CILEGON",
					BranchCode:      "100099",
					BusinessSource:  "BANK CONTOH CILEGON",
					BusinessName:    "Aneka",
					PolicyNumber:    "12000000000001",
					ClaimNumber:     "PNC-9001",
					RegisterDate:    day(2024, time.January, 15),
					LossDate:        day(2024, time.January, 10),
					EstimationValue: money.FromRupiah(250_000),
					LastProgressAt:  moment(2024, time.February, 1, 9),
					ProgressStatus1: "SURVEY",
					ProgressStatus2: "HASIL SURVEY BELUM ADA",
					TechnicalPIC:    "PICCONTOHSATU",
					ProgressNote:    "Menunggu hasil survei",
					AdjusterName:    "PT ADJUSTER CONTOH",
					CauseOfLoss:     "FIRE - SHORT CIRCUIT",
					Chronology:      "Terjadi kebakaran pada malam hari.",
					// Umurnya jauh di atas 180 hari terhadap jam uji mana pun yang wajar.
					ProgressStalled: false,
				},
				ClaimKey:           "CLAIM-0001",
				PolicyBusinessName: "PT CONTOH SATU",
				InsuredName:        "PT CONTOH SATU",
				ReserveClaimFull:   money.FromRupiah(250_000),
				ReserveClaimASM:    money.FromRupiah(200_000),
				Coinsurance:        money.FromRupiah(50_000),
				TreatyShares: inboxosclaimpercabang.TreatyShares{
					OR: money.FromRupiah(150_000), QS: money.FromRupiah(50_000),
				},
			},
		},
		{
			outstanding: true,
			item: inboxosclaimpercabang.ExportRow{
				WorkItem: inboxosclaimpercabang.WorkItem{
					BranchName:      "CILEGON",
					BranchCode:      "100099",
					BusinessSource:  "AGEN CONTOH",
					BusinessName:    "PA",
					PolicyNumber:    "12000000000002",
					ClaimNumber:     "PNC-9002",
					RegisterDate:    day(2026, time.September, 20),
					LossDate:        day(2026, time.September, 18),
					EstimationValue: money.FromRupiah(1_500_000),
					LastProgressAt:  moment(2026, time.September, 25, 14),
					ProgressStatus1: "ACCEPTATION",
					ProgressStatus2: "AUTO PROGRESS",
					TechnicalPIC:    "PICCONTOHDUA",
					ProgressNote:    "Progres tidak berubah tiga kali berturut-turut",
					AdjusterName:    "",
					CauseOfLoss:     "ACCIDENTAL DAMAGE",
					Chronology:      "Kecelakaan kerja di lokasi proyek.",
					// Masih muda, tetapi progresnya mandek — satu-satunya sebab ia merah.
					ProgressStalled: true,
				},
				ClaimKey:           "CLAIM-0002",
				PolicyBusinessName: "PT CONTOH DUA",
				InsuredName:        "PT CONTOH DUA",
				ReserveClaimFull:   money.FromRupiah(1_500_000),
				ReserveClaimASM:    money.FromRupiah(1_500_000),
				Coinsurance:        money.Zero,
				TreatyShares:       inboxosclaimpercabang.TreatyShares{OR: money.FromRupiah(1_500_000)},
			},
		},
		{
			outstanding: true,
			item: inboxosclaimpercabang.ExportRow{
				WorkItem: inboxosclaimpercabang.WorkItem{
					BranchName:   "CILEGON",
					BranchCode:   "100099",
					BusinessName: "Travel",
					PolicyNumber: "12000000000003",
					ClaimNumber:  "PNC-9003",
					RegisterDate: day(2026, time.September, 22),
					LossDate:     day(2026, time.September, 21),
					TechnicalPIC: "PICCONTOHDUA",
					// Tanpa satu pun catatan progres: LastProgressAt, kedua status, dan
					// keterangannya semuanya kosong. Baris ini yang membuktikan kolom
					// kosong tidak digambar sebagai tanggal awal zaman.
				},
				ClaimKey: "CLAIM-0003",
			},
		},
		{
			// Cabang LAIN. Ia tidak boleh pernah muncul pada permintaan cabang 100099.
			outstanding: true,
			item: inboxosclaimpercabang.ExportRow{
				WorkItem: inboxosclaimpercabang.WorkItem{
					BranchName:   "BANDUNG",
					BranchCode:   "100059",
					BusinessName: "Marine Cargo",
					PolicyNumber: "12000000000004",
					ClaimNumber:  "PNC-9004",
					RegisterDate: day(2026, time.August, 1),
					LossDate:     day(2026, time.July, 30),
				},
				ClaimKey: "CLAIM-0004",
			},
		},
		{
			// Sudah selesai. Penyaring outstanding yang membuangnya.
			outstanding: false,
			item: inboxosclaimpercabang.ExportRow{
				WorkItem: inboxosclaimpercabang.WorkItem{
					BranchName:   "CILEGON",
					BranchCode:   "100099",
					BusinessName: "Fire",
					PolicyNumber: "12000000000005",
					ClaimNumber:  "PNC-9005",
					RegisterDate: day(2026, time.May, 5),
				},
				ClaimKey: "CLAIM-0005",
			},
		},
		{
			// Tanpa tanggal registrasi. `registerdate IS NOT NULL` yang membuangnya.
			outstanding: true,
			item: inboxosclaimpercabang.ExportRow{
				WorkItem: inboxosclaimpercabang.WorkItem{
					BranchName:   "CILEGON",
					BranchCode:   "100099",
					BusinessName: "Aneka",
					PolicyNumber: "12000000000006",
					ClaimNumber:  "PNC-9006",
					RegisterDate: nil,
				},
				ClaimKey: "CLAIM-0006",
			},
		},
	}

	return store
}
