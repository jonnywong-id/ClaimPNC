package memory

import (
	"time"

	"claim-pnc/internal/inboxrcl"
)

// SampleLogin adalah login pengembangan lokal (`auth/provider/fake.go`) yang AKTIF di
// M_LOGIN_PNC, sehingga antrean contoh terlihat saat masuk.
const SampleLogin = "adminpnc"

// SampleOperator adalah SampleLogin sebagaimana tersimpan di TC_PNC_PUCL — huruf besar.
const SampleOperator = "ADMINPNC"

// SampleLoginInactive adalah login yang TIDAK AKTIF di M_LOGIN_PNC — antreannya kosong.
const SampleLoginInactive = "pictekniks"

// SampleOtherOperator adalah dokter RCL LAIN, dipakai membuktikan batas kewenangan bekerja.
const SampleOtherOperator = "DOKTERRCL02"

func at(year int, month time.Month, date, hour int) time.Time {
	return time.Date(year, month, date, hour, 0, 0, 0, time.UTC)
}

// SampleTechnicalPIC adalah PIC Teknik contoh — pemilik tugas Send To Analis setelah Dokter
// Tidak Setuju/Back.
const SampleTechnicalPIC = "PICTEKNIK01"

// SampleClaimAdmin adalah admin klaim contoh — pemilik `ASSIGNED_OPERATOR_ID` setelah Dokter
// Setuju (Work Owner, 2026-10-07).
//
// Sengaja BERBEDA dari SampleTechnicalPIC: contoh yang keduanya bernilai sama akan lulus
// baik ketika yang dipakai admin maupun ketika yang dipakai PIC Teknik, sehingga ia tidak
// menguji apa pun.
const SampleClaimAdmin = "ADMINKLAIM01"

// SampleLogins adalah baris contoh `M_LOGIN_PNC`.
var SampleLogins = []LoginRow{
	{LoginID: SampleLogin, Active: true},
	{LoginID: SampleOtherOperator, Active: true},

	// TIDAK AKTIF — loginnya ada, tetapi antreannya tidak boleh dicari.
	{LoginID: SampleLoginInactive, Active: false},
}

// SampleRecords adalah baris contoh `TC_PNC_PUCL`. Nomor polis dan nama tertanggungnya
// karangan (`D-69`).
//
// Apa yang sengaja dibuat pada contohnya:
//
//   - Satu baris RCL dan satu MSIG — kedua mode layar kerja.
//   - Satu baris PUCL milik pemanggil — penyaring D harus menolaknya.
//   - Satu baris milik dokter LAIN — penyaring A harus menolaknya.
//   - Satu baris tanpa TGL_KIRIM_PUCL — penyaring C harus menolaknya.
//   - Satu baris Resolved-Completed (TERTOLAK) dan satu Resolved-Rejected (TETAP MUNCUL).
//   - Dua baris berwaktu daftar SAMA PERSIS, untuk pemutus seri CLAIMID menurun.
var SampleRecords = []Record{
	{
		TechnicalPIC: SampleTechnicalPIC,
		ClaimAdmin:   SampleClaimAdmin,
		RegisteredAt: at(2026, time.October, 5, 1),
		Detail: inboxrcl.RCLDetail{
			ClaimNumber:      "PNCN.26.0412",
			PolicyNumber:     "26.002.2026.00412",
			InsuredName:      "Andi Saputra Contoh",
			Mode:             inboxrcl.ModeRCL,
			AnalystNote:      "Diagnosa tidak termasuk manfaat rawat inap; mohon pertimbangan penolakan.",
			Reason:           "Penyakit tidak dijamin polis.",
			StatusClaim:      "1151",
			ProcessStatus:    "New",
			AssignedOperator: SampleOperator,
			SentToRCLAt:      at(2026, time.October, 5, 1),
		},
	},
	{
		// Berwaktu daftar SAMA PERSIS dengan baris di bawahnya — …0405 harus mendahului …0404.
		TechnicalPIC: SampleTechnicalPIC,
		ClaimAdmin:   SampleClaimAdmin,
		RegisteredAt: at(2026, time.October, 3, 2),
		Detail: inboxrcl.RCLDetail{
			ClaimNumber:      "PNCN.26.0405",
			PolicyNumber:     "26.002.2026.00405",
			InsuredName:      "Maya Kartika Contoh",
			Mode:             inboxrcl.ModeMSIG,
			AnalystNote:      "Klaim co-insurance MSIG.",
			Reason:           "Menunggu konfirmasi leader MSIG.",
			StatusClaim:      "1151",
			ProcessStatus:    "New",
			AssignedOperator: SampleOperator,
			SentToRCLAt:      at(2026, time.October, 3, 2),
		},
	},
	{
		TechnicalPIC: SampleTechnicalPIC,
		ClaimAdmin:   SampleClaimAdmin,
		RegisteredAt: at(2026, time.October, 3, 2),
		Detail: inboxrcl.RCLDetail{
			ClaimNumber:      "PNCN.26.0404",
			PolicyNumber:     "26.005.2026.00404",
			InsuredName:      "Rudi Hartono Contoh",
			Mode:             inboxrcl.ModeRCL,
			StatusClaim:      "1151",
			ProcessStatus:    "New",
			AssignedOperator: SampleOperator,
			SentToRCLAt:      at(2026, time.October, 3, 3),
		},
	},
	{
		// HARUS TETAP MUNCUL — hanya Resolved-Completed yang dikecualikan.
		TechnicalPIC: SampleTechnicalPIC,
		ClaimAdmin:   SampleClaimAdmin,
		RegisteredAt: at(2026, time.September, 28, 4),
		Detail: inboxrcl.RCLDetail{
			ClaimNumber:      "PNCN.26.0380",
			PolicyNumber:     "26.002.2026.00380",
			InsuredName:      "Siti Rahma Contoh",
			Mode:             inboxrcl.ModeRCL,
			AnalystNote:      "Dokumen medis tidak lengkap.",
			ProcessStatus:    "Resolved-Rejected",
			AssignedOperator: SampleOperator,
			SentToRCLAt:      at(2026, time.September, 28, 4),
		},
	},
	{
		// TIDAK BOLEH MUNCUL — tugasnya sudah tuntas (penyaring B).
		TechnicalPIC: SampleTechnicalPIC,
		ClaimAdmin:   SampleClaimAdmin,
		RegisteredAt: at(2026, time.September, 20, 2),
		Detail: inboxrcl.RCLDetail{
			ClaimNumber:      "PNCN.26.0371",
			PolicyNumber:     "26.002.2026.00371",
			InsuredName:      "Agus Salim Contoh",
			Mode:             inboxrcl.ModeRCL,
			ProcessStatus:    inboxrcl.StatusKerjaSelesai,
			AssignedOperator: SampleOperator,
			SentToRCLAt:      at(2026, time.September, 20, 2),
		},
	},
	{
		// TIDAK BOLEH MUNCUL — belum dikirim analis (penyaring C).
		TechnicalPIC: SampleTechnicalPIC,
		ClaimAdmin:   SampleClaimAdmin,
		RegisteredAt: at(2026, time.October, 4, 5),
		Detail: inboxrcl.RCLDetail{
			ClaimNumber:      "PNCN.26.0366",
			PolicyNumber:     "26.002.2026.00366",
			InsuredName:      "Lina Marlina Contoh",
			Mode:             inboxrcl.ModeRCL,
			ProcessStatus:    "New",
			AssignedOperator: SampleOperator,
		},
	},
	{
		// TIDAK BOLEH MUNCUL — jalur PUCL, tidak melewati dokter (penyaring D).
		TechnicalPIC: SampleTechnicalPIC,
		ClaimAdmin:   SampleClaimAdmin,
		RegisteredAt: at(2026, time.October, 4, 6),
		Detail: inboxrcl.RCLDetail{
			ClaimNumber:      "PNCN.26.0359",
			PolicyNumber:     "26.002.2026.00359",
			InsuredName:      "Budi Santoso Contoh",
			Mode:             inboxrcl.ModePUCL,
			ProcessStatus:    "New",
			AssignedOperator: SampleOperator,
			SentToRCLAt:      at(2026, time.October, 4, 6),
		},
	},
	{
		// TIDAK BOLEH MUNCUL bagi SampleOperator — milik dokter lain (penyaring A).
		TechnicalPIC: SampleTechnicalPIC,
		ClaimAdmin:   SampleClaimAdmin,
		RegisteredAt: at(2026, time.October, 4, 7),
		Detail: inboxrcl.RCLDetail{
			ClaimNumber:      "PNCN.26.0350",
			PolicyNumber:     "26.002.2026.00350",
			InsuredName:      "Dian Permata Contoh",
			Mode:             inboxrcl.ModeRCL,
			ProcessStatus:    "New",
			AssignedOperator: SampleOtherOperator,
			SentToRCLAt:      at(2026, time.October, 4, 7),
		},
	},
}

// NewSampleStore membentuk pembaca berisi data contoh — salinan, supaya dua portal tidak
// berbagi senarai yang sama (`R-20`).
func NewSampleStore() *Store {
	logins := make([]LoginRow, len(SampleLogins))
	copy(logins, SampleLogins)
	records := make([]Record, len(SampleRecords))
	copy(records, SampleRecords)
	return NewStore(logins, records)
}
