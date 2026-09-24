package memory

import (
	"time"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

// SampleOperator adalah pemilik seluruh kasus contoh.
//
// Ia sengaja sama dengan salah satu penyetuju pada SampleThresholds — ELLENSUPRIYATI,
// jenjang terendah Non-MBU pita 1. Dengan begitu contoh inbox dan contoh master ambang
// bercerita tentang orang yang sama, dan seseorang yang membuka kedua layar tanpa basis
// data melihat satu keadaan yang utuh alih-alih dua dunia yang tidak berhubungan.
const SampleOperator = "ELLENSUPRIYATI"

// SampleCases mengembalikan kasus komite contoh.
//
// # Ini data KARANGAN, dan itu harus dinyatakan terang
//
// Berbeda dari SampleThresholds — yang setiap barisnya diturunkan dari
// `Database/emailkomite.csv` — isi inbox tidak pernah diserahkan kepada kami. Tidak ada
// satu pun ekstrak `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` di repositori ini.
//
// Karena itu nilai di bawah dipilih untuk MENUTUPI SELURUH KEADAAN yang layarnya harus
// tangani, bukan untuk tampak seperti data sungguhan:
//
//	K-2601  menunggu, sudah lama     → menguji Aging dan urutan "terlama di atas"
//	K-2602  menunggu, baru masuk     → menguji Aging kecil
//	K-2603  menunggu, TANPA nilai AI → menguji OUTER JOIN: kasus tanpa penilaian AI
//	                                   TETAP muncul, tidak hilang dari inbox
//	K-2604  sudah diterima di PEGA   → menguji riwayat warisan pada kotak Diterima
//	K-2605  sudah ditolak di PEGA    → menguji riwayat warisan pada kotak Ditolak
//	K-2606  milik operator LAIN      → menguji inbox tidak bocor antar orang
//	K-2607  sudah Resolved-Completed → menguji penyaring PYSTATUSWORK
//
// Tidak ada nomor polis, nama tertanggung, maupun nomor klaim sungguhan di sini — `D-69`
// melarang data nasabah ditulis ke artefak yang di-commit, tanpa perkecualian.
func SampleCases() []komite.CommitteeCase {
	// Tanggal acuan dibekukan, bukan diambil dari jam berjalan. Contoh yang bergantung
	// pada "hari ini" membuat pengujian berubah hasil tanpa satu baris pun disunting.
	base := time.Date(2026, 9, 20, 2, 0, 0, 0, time.UTC)

	return []komite.CommitteeCase{
		{
			CaseID:           "K-2601",
			ClaimNumber:      "PNCN.26.0101",
			PolicyNumber:     "POL-FIRE-0001",
			InsuredName:      "Tertanggung Contoh Satu",
			BusinessName:     "Property All Risk",
			SourceOfBusiness: "Direct",
			BranchName:       "Jakarta Pusat",
			GroupPanel:       "006",
			ClaimPIC:         "PICTEKNIK1",
			AssignedOperator: SampleOperator,
			CommitteeDate:    base.AddDate(0, 0, -9),
			CreatedAt:        base.AddDate(0, 0, -9),
			WorkStatus:       "Open",
			CommitteeKind:    komite.CommitteeKindOf("2", "5"),
			ClaimValue:       money.FromRupiah(45_000_000),
			ASMShareValue:    money.FromRupiah(31_500_000),
			ORValue:          money.FromRupiah(22_500_000),
			LegacyOutcome:    komite.OutcomePending,
			LegacyTier:       1,

			AIResult:        "DITERIMA",
			AINoteAccepted:  "Dokumen lengkap dan kronologi sesuai polis.",
			AIAssessedAt:    base.AddDate(0, 0, -9),
			HasAIAssessment: true,
		},
		{
			CaseID:           "K-2602",
			ClaimNumber:      "PNCN.26.0102",
			PolicyNumber:     "POL-MARINE-0007",
			InsuredName:      "Tertanggung Contoh Dua",
			BusinessName:     "Marine Cargo",
			SourceOfBusiness: "Broker",
			BranchName:       "Surabaya",
			GroupPanel:       "004",
			ClaimPIC:         "PICTEKNIK2",
			AssignedOperator: SampleOperator,
			CommitteeDate:    base.AddDate(0, 0, -1),
			CreatedAt:        base.AddDate(0, 0, -1),
			WorkStatus:       "Open",
			CommitteeKind:    komite.CommitteeKindOf("2", "2"),
			ClaimValue:       money.FromRupiah(18_750_000),
			ASMShareValue:    money.FromRupiah(18_750_000),
			ORValue:          money.FromRupiah(9_375_000),
			LegacyOutcome:    komite.OutcomePending,
			LegacyTier:       1,

			AIResult:        "DITOLAK",
			AINoteRejected:  "Tanggal lapor melewati batas tujuh hari.",
			AIAssessedAt:    base.AddDate(0, 0, -1),
			HasAIAssessment: true,
		},
		{
			// Tanpa penilaian AI sama sekali. Ia WAJIB tetap muncul: kueri lama
			// menyambungkan `T_CLAIM_DATA_RESULTS_AI` dengan OUTER JOIN
			// (`A.pyID = AI.KOMITE(+)`), dan mengubahnya menjadi INNER akan membuat
			// pekerjaan menghilang tanpa satu pun tanda.
			CaseID:           "K-2603",
			ClaimNumber:      "PNCN.26.0103",
			PolicyNumber:     "POL-ANEKA-0022",
			InsuredName:      "Tertanggung Contoh Tiga",
			BusinessName:     "Aneka",
			SourceOfBusiness: "Direct",
			BranchName:       "Bandung",
			GroupPanel:       "003",
			ClaimPIC:         "PICTEKNIK1",
			AssignedOperator: SampleOperator,
			CommitteeDate:    base.AddDate(0, 0, -4),
			CreatedAt:        base.AddDate(0, 0, -4),
			WorkStatus:       "Open",
			CommitteeKind:    komite.CommitteeKindOf("1", ""),
			ClaimValue:       money.FromRupiah(7_200_000),
			ASMShareValue:    money.FromRupiah(7_200_000),
			LegacyOutcome:    komite.OutcomePending,
			LegacyTier:       1,
			HasAIAssessment:  false,
		},
		{
			CaseID:           "K-2604",
			ClaimNumber:      "PNCN.26.0104",
			PolicyNumber:     "POL-PA-0310",
			InsuredName:      "Tertanggung Contoh Empat",
			BusinessName:     "Personal Accident",
			SourceOfBusiness: "Bancassurance",
			BranchName:       "Medan",
			GroupPanel:       "002",
			ClaimPIC:         "PICTEKNIK3",
			AssignedOperator: SampleOperator,
			CommitteeDate:    base.AddDate(0, 0, -21),
			CreatedAt:        base.AddDate(0, 0, -21),
			WorkStatus:       komite.StatusResolved,
			CommitteeKind:    komite.CommitteeKindOf("2", "1"),
			ClaimValue:       money.FromRupiah(9_500_000),
			ASMShareValue:    money.FromRupiah(9_500_000),
			CommitteeNote:    "Disetujui sesuai hasil survei.",
			LegacyOutcome:    komite.OutcomeApproved,
			LegacyTier:       1,

			AIResult:        "DITERIMA",
			AINoteAccepted:  "Sesuai manfaat polis.",
			AIAssessedAt:    base.AddDate(0, 0, -21),
			HasAIAssessment: true,
		},
		{
			CaseID:           "K-2605",
			ClaimNumber:      "PNCN.26.0105",
			PolicyNumber:     "POL-TRAVEL-0088",
			InsuredName:      "Tertanggung Contoh Lima",
			BusinessName:     "Travel",
			SourceOfBusiness: "Online",
			BranchName:       "Denpasar",
			GroupPanel:       "005",
			ClaimPIC:         "PICTEKNIK2",
			AssignedOperator: SampleOperator,
			CommitteeDate:    base.AddDate(0, 0, -30),
			CreatedAt:        base.AddDate(0, 0, -30),
			WorkStatus:       komite.StatusResolved,
			CommitteeKind:    komite.CommitteeKindOf("2", "6"),
			ClaimValue:       money.FromRupiah(3_400_000),
			ASMShareValue:    money.FromRupiah(3_400_000),
			CommitteeNote:    "Kejadian di luar periode pertanggungan.",
			LegacyOutcome:    komite.OutcomeRejected,
			LegacyTier:       1,
			HasAIAssessment:  false,
		},
		{
			// Milik orang lain. Ia ada supaya setiap pengujian inbox membuktikan
			// batasnya, bukan sekadar mengandalkan penyaring yang kebetulan benar.
			CaseID:           "K-2606",
			ClaimNumber:      "PNCN.26.0106",
			PolicyNumber:     "POL-FIRE-0444",
			InsuredName:      "Tertanggung Contoh Enam",
			BusinessName:     "Property All Risk",
			SourceOfBusiness: "Direct",
			BranchName:       "Semarang",
			GroupPanel:       "006",
			ClaimPIC:         "PICTEKNIK4",
			AssignedOperator: "INDRAGUNAWAN",
			CommitteeDate:    base.AddDate(0, 0, -12),
			CreatedAt:        base.AddDate(0, 0, -12),
			WorkStatus:       "Open",
			CommitteeKind:    komite.CommitteeKindOf("2", "5"),
			ClaimValue:       money.FromRupiah(320_000_000),
			ASMShareValue:    money.FromRupiah(160_000_000),
			LegacyOutcome:    komite.OutcomePending,
			LegacyTier:       2,
			HasAIAssessment:  false,
		},
		{
			// Sudah tuntas di Pega tetapi tidak punya keputusan tercatat sama sekali.
			// Ia TIDAK boleh muncul di kotak mana pun: bukan Outstanding karena
			// PYSTATUSWORK sudah Resolved-Completed, dan bukan riwayat karena tidak ada
			// keputusan yang dapat ditampilkan.
			CaseID:           "K-2607",
			ClaimNumber:      "PNCN.26.0107",
			PolicyNumber:     "POL-ANEKA-0555",
			InsuredName:      "Tertanggung Contoh Tujuh",
			BusinessName:     "Aneka",
			SourceOfBusiness: "Direct",
			BranchName:       "Makassar",
			GroupPanel:       "009",
			ClaimPIC:         "PICTEKNIK1",
			AssignedOperator: SampleOperator,
			CommitteeDate:    base.AddDate(0, 0, -45),
			CreatedAt:        base.AddDate(0, 0, -45),
			WorkStatus:       komite.StatusResolved,
			CommitteeKind:    komite.CommitteeKindOf("4", ""),
			ClaimValue:       money.FromRupiah(1_100_000),
			LegacyOutcome:    komite.OutcomePending,
			HasAIAssessment:  false,
		},
	}
}
