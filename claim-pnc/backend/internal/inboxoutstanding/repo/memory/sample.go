package memory

import (
	"time"

	"claim-pnc/internal/inboxoutstanding"
)

// sampleClaims adalah klaim contoh untuk pengembangan lokal.
//
// # Seluruhnya KARANGAN
//
// Nomor polis, nama tertanggung, dan nama petugas di bawah tidak merujuk siapa pun. `D-69`
// melarang data nasabah nyata ditulis di berkas yang di-commit, dan `D-64` yang
// mengizinkan data produksi hanya berlaku untuk isi lingkungan staging — bukan isi kode.
//
// # Yang sengaja diwakili
//
// Daftar ini bukan sekadar "beberapa baris supaya layar tidak kosong". Ia dipilih agar
// setiap keadaan yang dapat membuat layar salah gambar benar-benar terjadi:
//
//   - klaim yang BELUM BERNOMOR — nomor baru terbit setelah Input Register lolos
//   - tugas di Workbasket yang BELUM BERTUAN — CurrentHolder kosong
//   - tanggal lapor yang KOSONG — klaim dibuka sebelum tanggalnya pasti
//   - kelima Group Panel, supaya batas data per lini dapat diuji sungguhan
//   - umur klaim yang berbeda-beda, termasuk yang didaftarkan hari ini
func sampleClaims() []inboxoutstanding.OutstandingClaim {
	// Waktu acuan dibuat relatif terhadap hari ini supaya kolom "Total Aging" selalu
	// masuk akal, berapa lama pun berkas ini tidak disentuh. Data contoh bertanggal tetap
	// akan tampak berumur bertahun-tahun setelah beberapa bulan.
	now := time.Now().UTC()
	daysAgo := func(n int) time.Time { return now.AddDate(0, 0, -n) }
	dateOf := func(t time.Time) *time.Time { return &t }
	agingOf := func(n int) *int { return &n }

	return []inboxoutstanding.OutstandingClaim{
		{
			ClaimID:        "KLM-0001",
			ClaimNumber:    "PNCN.26.0001",
			PolicyNumber:   "POL-FIRE-0001",
			InsuredName:    "PT Bumi Contoh Sentosa",
			BusinessName:   "Fire",
			BusinessSource: "Broker",
			BranchName:     "JKT",
			GroupPanel:     "006",
			RCVID:          "RCV-0001",
			RegisteredAt:   daysAgo(12),
			LossDate:       dateOf(daysAgo(16)),
			ReportDate:     dateOf(daysAgo(14)),
			ProcessStatus:  statusKerjaProduksi,
			ClaimStatus:    "1147",
			AgingDays:      agingOf(5),
			ProgressStatus: "On Progress",
			TechnicalPIC:   "BUDISANTOSO",
			RecordedBy:     "ADMINPNC",
			CurrentHolder:  "BUDISANTOSO",
			CurrentStage:   "Input Estimasi",
		},
		{
			ClaimID:        "KLM-0002",
			ClaimNumber:    "", // belum lolos Input Register — nomor belum terbit
			PolicyNumber:   "POL-MARINE-0042",
			InsuredName:    "CV Samudra Contoh",
			BusinessName:   "Marine Cargo",
			BusinessSource: "Direct",
			BranchName:     "SBY",
			GroupPanel:     "004",
			RCVID:          "RCV-0002",
			RegisteredAt:   daysAgo(1),
			LossDate:       nil,
			ReportDate:     nil, // tanggal lapor belum pasti
			ProcessStatus:  statusKerjaProduksi,
			ClaimStatus:    "1147",
			AgingDays:      nil,
			ProgressStatus: "On Progress",
			TechnicalPIC:   "SITIRAHAYU",
			RecordedBy:     "ADMINPNC",
			CurrentHolder:  "SITIRAHAYU",
			CurrentStage:   "Input Register",
		},
		{
			ClaimID:        "KLM-0003",
			ClaimNumber:    "PNCN.26.0003",
			PolicyNumber:   "POL-PA-0107",
			InsuredName:    "Rahmat Contoh Pratama",
			BusinessName:   "Personal Accident",
			BusinessSource: "Agen",
			BranchName:     "BDG",
			GroupPanel:     "002",
			RCVID:          "RCV-0003",
			RegisteredAt:   daysAgo(30),
			LossDate:       dateOf(daysAgo(35)),
			ReportDate:     dateOf(daysAgo(33)),
			ProcessStatus:  statusKerjaProduksi,
			ClaimStatus:    "1151",
			AgingDays:      agingOf(12),
			ProgressStatus: "On Progress",
			TechnicalPIC:   "DEWILESTARI",
			RecordedBy:     "ADMINPA",
			CurrentHolder:  "", // di Workbasket, belum diambil siapa pun
			CurrentStage:   "Analyst Doctor",
		},
		{
			ClaimID:        "KLM-0004",
			ClaimNumber:    "PNCN.26.0004",
			PolicyNumber:   "POL-TRAVEL-0088",
			InsuredName:    "Maya Contoh Anggraini",
			BusinessName:   "Travel",
			BusinessSource: "Online",
			BranchName:     "JKT",
			GroupPanel:     "005",
			RCVID:          "RCV-0004",
			RegisteredAt:   daysAgo(5),
			LossDate:       dateOf(daysAgo(8)),
			ReportDate:     dateOf(daysAgo(6)),
			ProcessStatus:  statusKerjaProduksi,
			ClaimStatus:    "1147",
			AgingDays:      agingOf(0),
			ProgressStatus: "On Progress",
			TechnicalPIC:   "AGUNGWIJAYA",
			RecordedBy:     "ADMINTRAVEL",
			CurrentHolder:  "AGUNGWIJAYA",
			CurrentStage:   "Send To PIC Teknik",
		},
		{
			ClaimID:        "KLM-0005",
			ClaimNumber:    "PNCN.26.0005",
			PolicyNumber:   "POL-ANEKA-0311",
			InsuredName:    "PT Aneka Contoh Makmur",
			BusinessName:   "Aneka",
			BusinessSource: "Direct",
			BranchName:     "MDN",
			GroupPanel:     "003",
			RCVID:          "RCV-0008",
			RegisteredAt:   daysAgo(0), // didaftarkan hari ini — umur nol hari
			LossDate:       dateOf(daysAgo(2)),
			ReportDate:     dateOf(daysAgo(0)),
			ProcessStatus:  statusKerjaProduksi,
			ClaimStatus:    "1147",
			AgingDays:      agingOf(0),
			ProgressStatus: "On Progress",
			TechnicalPIC:   "BUDISANTOSO",
			RecordedBy:     "ADMINPNC",
			CurrentHolder:  "",
			CurrentStage:   "Compliance",
		},
		{
			ClaimID:        "KLM-0006",
			ClaimNumber:    "PNCN.26.0006",
			PolicyNumber:   "POL-FIRE-0209",
			InsuredName:    "PT Graha Contoh Abadi",
			BusinessName:   "Fire",
			BusinessSource: "Broker",
			BranchName:     "SBY",
			GroupPanel:     "006",
			RCVID:          "RCV-0001",
			RegisteredAt:   daysAgo(64),
			LossDate:       dateOf(daysAgo(68)),
			ReportDate:     dateOf(daysAgo(66)),
			ProcessStatus:  statusKerjaProduksi,
			ClaimStatus:    "1149",
			AgingDays:      agingOf(40),
			ProgressStatus: "On Progress",
			TechnicalPIC:   "SITIRAHAYU",
			RecordedBy:     "ADMINPNC",
			CurrentHolder:  "SITIRAHAYU",
			CurrentStage:   "Komite",
		},
		{
			ClaimID:        "KLM-0007",
			ClaimNumber:    "PNCN.26.0007",
			PolicyNumber:   "POL-PA-0450",
			InsuredName:    "Intan Contoh Permata",
			BusinessName:   "Personal Accident",
			BusinessSource: "Agen",
			BranchName:     "BDG",
			GroupPanel:     "002",
			RCVID:          "RCV-0003",
			RegisteredAt:   daysAgo(3),
			LossDate:       dateOf(daysAgo(5)),
			ReportDate:     dateOf(daysAgo(3)),
			ProcessStatus:  statusKerjaProduksi,
			ClaimStatus:    "1147",
			AgingDays:      agingOf(2),
			ProgressStatus: "On Progress",
			TechnicalPIC:   "DEWILESTARI",
			RecordedBy:     "ADMINPA",
			CurrentHolder:  "DEWILESTARI",
			CurrentStage:   "Input Estimasi",
		},
		// Baris kedelapan DIBENTUK DARI SATU BARIS PRODUKSI yang diserahkan Work Owner
		// pada 2026-09-21.
		//
		// # Nilai pengenalnya dikarang, bentuknya tidak
		//
		// `D-69` melarang nomor polis, nama tertanggung, dan nomor klaim asli ditulis di
		// berkas yang di-commit. Keempatnya diganti. Yang DIPERTAHANKAN adalah hal-hal
		// yang membuat baris itu berharga — empat keadaan yang tidak terpikir dikarang,
		// dan tidak satu pun terwakili tujuh baris di atas:
		//
		//  1. Tanggal kejadian SESUDAH tanggal pendaftaran. Melanggar invarian `I-2`,
		//     tetapi ada di produksi. Layar harus tetap menggambarkannya, bukan gagal —
		//     modul ini memantau, ia tidak memvalidasi.
		//  2. Aging 618 hari, jauh melampaui umur sejak pendaftaran. Bukti bahwa `AGING`
		//     BUKAN turunan tanggal pendaftaran, dan menguatkan keputusan membacanya apa
		//     adanya alih-alih menghitungnya ulang.
		//  3. "Status ASM" KOSONG. Baris aslinya tidak menyertakan `STATUSLOCK_1` sama
		//     sekali, sehingga kolom itu tampil kosong pada data produksi yang nyata.
		//  4. `BUSINESSGROUPID` di luar daftar yang dikecualikan, dengan Group Panel 006.
		//     Ia karena itu terlihat oleh NONMBU maupun BONDING — satu-satunya baris
		//     contoh yang menguji kedua lini sekaligus.
		{
			ClaimID:        "ASM-FW-GCNMFW-WORK PNC-0008", // kunci warisan berprefix Pega
			ClaimNumber:    "PNC-0008",                    // format LAMA, bukan PNCN.YY.xxxx
			PolicyNumber:   "POL-FIRE-0008",
			InsuredName:    "PT Cilegon Contoh Utama",
			BusinessName:   "FIRE",
			BusinessSource: "Contoh Sumber Bisnis",
			BranchName:     "CILEGON",
			GroupPanel:     "006",
			RCVID:          "RCV-0008",
			RegisteredAt:   daysAgo(640),
			LossDate:       dateOf(daysAgo(376)), // SESUDAH pendaftaran, apa adanya
			ReportDate:     dateOf(daysAgo(640)),
			ProcessStatus:  statusKerjaProduksi,
			ClaimStatus:    "", // STATUSLOCK_1 tidak terisi — "Status ASM" kosong
			AgingDays:      agingOf(618),
			ProgressStatus: "ACCEPTATION",
			TechnicalPIC:   "PICTEKNIKS",
			RecordedBy:     "ADMINPNC",
			CurrentHolder:  "PICTEKNIKS",
			CurrentStage:   "Choose Surveyor",
		},
	}
}

// statusKerjaProduksi adalah nilai `PYSTATUSWORK` yang benar-benar terlihat pada baris
// produksi: `"New"`, bukan `"BERJALAN"`.
//
// Data contoh sempat memakai `"BERJALAN"` — nilai tabel yang salah pakai — dan seluruh uji
// tetap lulus karena kode diperiksa terhadap data yang dikarang dari kode itu sendiri.
// Konstanta ini menamai nilai yang benar supaya kekeliruan itu tidak kembali diam-diam.
const statusKerjaProduksi = "New"
