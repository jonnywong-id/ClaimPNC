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
					InsuredName:     "PT CONTOH SATU",
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
				ReserveClaimFull:   money.FromRupiah(250_000),
				ReserveClaimASM:    money.FromRupiah(200_000),
				Coinsurance:        money.FromRupiah(50_000),
				TreatyShares: inboxosclaimpercabang.TreatyShares{
					OR: money.FromRupiah(150_000), QS: money.FromRupiah(50_000),
				},
			},
			// Baris ini membawa isi popup yang LENGKAP: dua objek, dua catatan progres, dan
			// dua pesan — satu internal, satu dari adjuster luar. Pasangan internal/eksternal
			// itu yang membuktikan penandanya benar-benar dibaca, bukan selalu bernilai sama.
			detail: inboxosclaimpercabang.Detail{
				Occupation:      "PERKANTORAN",
				TotalSumInsured: money.FromRupiah(5_000_000),
				// Objek PERTAMA punya dua coverage, objek kedua tidak punya sama sekali.
				// Perbedaan itu disengaja: panel yang terbuka harus terbukti menampilkan
				// coverage milik objeknya sendiri, dan harus terbukti menangani objek yang
				// memang kosong tanpa menampilkan panel yang menyesatkan.
				Objects: []inboxosclaimpercabang.DetailObject{
					{
						ID:       "OBJ-9001-1",
						Name:     "GUDANG A",
						Location: "JL CONTOH NO 1, CILEGON",
						Coverages: []inboxosclaimpercabang.DetailCoverage{
							{
								ObjectID: "OBJ-9001-1", ID: "CVG-1",
								Currency: "IDR",
								SumTSI:   money.FromRupiah(3_000_000),
								Name:     "Kebakaran bangunan",
								// Satu item dengan DUA estimasi yang saling meniadakan —
								// bentuk yang benar-benar ada di data nyata, dan yang
								// membuktikan nilai negatif ikut digambar.
								Items: []inboxosclaimpercabang.DetailItem{{
									ObjectID: "OBJ-9001-1", CoverageID: "CVG-1", ID: "1",
									Name:        "BUILDINGS",
									Description: "Bangunan gudang utama",
									Estimations: []inboxosclaimpercabang.DetailEstimation{
										{
											ObjectID: "OBJ-9001-1", CoverageID: "CVG-1",
											ItemID: "1", Sequence: "1",
											RecordedAt: moment(2024, time.January, 20, 9),
											Type:       "Claim",
											Currency:   "IDR",
											Rate:       money.FromRupiah(1),
											Value:      money.FromRupiah(3_000_000),
										},
										{
											ObjectID: "OBJ-9001-1", CoverageID: "CVG-1",
											ItemID: "1", Sequence: "2",
											RecordedAt: moment(2024, time.January, 21, 10),
											Type:       "Claim",
											Currency:   "IDR",
											Rate:       money.FromRupiah(1),
											Value:      money.FromRupiah(-500_000),
										},
									},
								}},
							},
							{
								ObjectID: "OBJ-9001-1", ID: "CVG-2",
								Currency: "USD",
								SumTSI:   money.FromRupiah(2_000_000),
								Name:     "Gempa bumi",
								Items:    []inboxosclaimpercabang.DetailItem{},
							},
						},
					},
					{
						ID:        "OBJ-9001-2",
						Name:      "GUDANG B",
						Location:  "JL CONTOH NO 2, CILEGON",
						Coverages: []inboxosclaimpercabang.DetailCoverage{},
					},
				},
				ProgressHistory: []inboxosclaimpercabang.DetailProgress{
					{
						RecordedAt:     moment(2024, time.February, 1, 9),
						ClaimNumber:    "PNC-9001",
						Status1:        "SURVEY",
						Status2:        "HASIL SURVEY BELUM ADA",
						EnteredBy:      "PICCONTOHSATU",
						NextFollowUpAt: day(2024, time.February, 8),
						Status:         "OPEN",
						Note:           "Menunggu hasil survei",
					},
					{
						RecordedAt:  moment(2024, time.January, 16, 10),
						ClaimNumber: "PNC-9001",
						Status1:     "REGISTER",
						EnteredBy:   "ADMINCONTOH",
						Note:        "Klaim diregistrasi",
					},
				},
				AdjusterMessages: []inboxosclaimpercabang.DetailMessage{
					{
						SenderName: "PICCONTOHSATU",
						SentAt:     moment(2024, time.February, 2, 8),
						Message:    "Mohon kirimkan laporan survei.",
						RepliedAt:  moment(2024, time.February, 3, 11),
						Reply:      "Laporan sedang disusun.",
						Internal:   true,
					},
					{
						SenderName: "PT ADJUSTER CONTOH",
						SentAt:     moment(2024, time.February, 5, 15),
						Message:    "Dokumen pendukung belum lengkap.",
						Internal:   false,
					},
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
					InsuredName:     "PT CONTOH DUA",
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
				ReserveClaimFull:   money.FromRupiah(1_500_000),
				ReserveClaimASM:    money.FromRupiah(1_500_000),
				Coinsurance:        money.Zero,
				TreatyShares:       inboxosclaimpercabang.TreatyShares{OR: money.FromRupiah(1_500_000)},
			},
			// Baris PA sengaja memakai kolom objek yang BERBEDA — nama peserta, status,
			// KTP/Paspor, tanggal lahir — supaya varian kolom kedua ikut terbukti bekerja.
			// Bila kedua baris contoh memakai kolom yang sama, uji tidak pernah membuktikan
			// layar memilih variannya.
			detail: inboxosclaimpercabang.Detail{
				Occupation:      "",
				TotalSumInsured: money.FromRupiah(50_000),
				Objects: []inboxosclaimpercabang.DetailObject{
					{
						ID:                "OBJ-9002-1",
						Name:              "BUDI CONTOH",
						DateOfBirth:       day(1990, time.March, 17),
						IDCard:            "3200000000000001",
						ParticipantStatus: "KARYAWAN",
						Job:               "TEKNISI",
						Coverages: []inboxosclaimpercabang.DetailCoverage{
							{
								ObjectID: "OBJ-9002-1", ID: "CVG-3",
								Currency: "IDR",
								SumTSI:   money.FromRupiah(50_000),
								Name:     "Kecelakaan diri",
								Items:    []inboxosclaimpercabang.DetailItem{},
							},
						},
					},
				},
				ProgressHistory: []inboxosclaimpercabang.DetailProgress{
					{
						RecordedAt:  moment(2026, time.September, 25, 14),
						ClaimNumber: "PNC-9002",
						Status1:     "ACCEPTATION",
						Status2:     "AUTO PROGRESS",
						EnteredBy:   "PICCONTOHDUA",
						Note:        "Progres tidak berubah tiga kali berturut-turut",
					},
				},
				// Tanpa satu pun pesan. Grid kosong harus terbaca sebagai "belum ada
				// komunikasi", bukan sebagai kegagalan memuat.
				AdjusterMessages: []inboxosclaimpercabang.DetailMessage{},
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
