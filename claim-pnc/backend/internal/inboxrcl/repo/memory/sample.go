package memory

import (
	"time"

	"claim-pnc/internal/inboxrcl"
)

// SampleLogin adalah login pengembangan lokal (`auth/provider/fake.go`) yang PUNYA identitas
// lama pada grup yang diizinkan, sehingga antrean contoh terlihat saat masuk.
const SampleLogin = "adminpnc"

// SampleLegacyID adalah identitas lama SampleLogin — padanan `TempOperator.City`.
//
// Karangan. Operator ID sungguhan yang tertanam di sistem lama tidak dipakai sebagai data
// yang dijalankan (`D-15`); `D-69` mengizinkannya hanya di dokumen.
const SampleLegacyID = "DOKTERRCL01"

// SampleLoginNoLegacy adalah login pengembangan lokal yang identitas lamanya HANYA tercatat
// pada grup yang tidak diizinkan — ia harus melihat keadaan "identitas lama tidak ditemukan".
const SampleLoginNoLegacy = "pictekniks"

// SampleOtherLegacyID adalah dokter RCL LAIN, dipakai membuktikan batas kewenangan bekerja.
const SampleOtherLegacyID = "DOKTERRCL02"

func at(year int, month time.Month, date, hour int) time.Time {
	return time.Date(year, month, date, hour, 0, 0, 0, time.UTC)
}

// SampleAccess adalah baris contoh `T_ACCESS_GROUP_PNC`.
//
// Satu orang punya satu baris per grup akses, sama seperti tabel aslinya. Setiap baris yang
// harus TERTOLAK diberi keterangan di tempatnya.
var SampleAccess = []AccessRow{
	{Login: SampleLogin, LegacyID: SampleLegacyID, AccessGroup: "GCNMFW:PNCKomite", Active: true},
	{Login: SampleLogin, LegacyID: SampleLegacyID, AccessGroup: "GCNMFW:ViewClaimPNC", Active: true},

	// TERTOLAK — grupnya bukan salah satu dari tiga yang diizinkan.
	{Login: SampleLoginNoLegacy, LegacyID: "PICTEKNIKLAMA", AccessGroup: "GCNMFW:PncPICTeknik", Active: true},

	// TERTOLAK — barisnya tidak aktif.
	{Login: SampleLoginNoLegacy, LegacyID: "PICTEKNIKLAMA", AccessGroup: "GCNMFW:PNCKomite", Active: false},
}

// SampleTasks adalah antrean contoh. Nomor polis dan nama tertanggungnya karangan (`D-69`).
//
// Apa yang sengaja dibuat pada contohnya:
//
//   - Satu baris milik dokter LAIN — bila penyaring A/D hilang, ia bocor ke layar.
//   - Satu baris yang penugasannya milik pemanggil tetapi dokter RCL-nya orang lain — hanya
//     penyaring D yang menolaknya.
//   - Satu baris tanpa Tanggal Masuk Inbox — hanya penyaring C yang menolaknya.
//   - Satu baris ber-`Resolved-Completed` (TERTOLAK) dan satu ber-`Resolved-Rejected` (TETAP
//     MUNCUL).
//   - Dua baris berwaktu daftar SAMA PERSIS, untuk pemutus seri `pyID` menurun.
var SampleTasks = []inboxrcl.RCLTask{
	{
		ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0412",
		ClaimNumber:      "PNCN.26.0412",
		PolicyNumber:     "26.002.2026.00412",
		InsuredName:      "Andi Saputra Contoh",
		SentToRCLAt:      at(2026, time.September, 22, 3),
		AnalystNote:      "Diagnosa tidak termasuk manfaat rawat inap; mohon pertimbangan penolakan.",
		RCLDoctor:        SampleLegacyID,
		RegisteredAt:     at(2026, time.September, 20, 2),
		ProcessStatus:    "Open",
		AssignedOperator: SampleLegacyID,
	},
	{
		// Berwaktu daftar SAMA PERSIS dengan baris di bawahnya — …0405 harus mendahului …0404.
		ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0405",
		ClaimNumber:      "PNCN.26.0405",
		PolicyNumber:     "26.002.2026.00405",
		InsuredName:      "Maya Kartika Contoh",
		SentToRCLAt:      at(2026, time.September, 19, 7),
		AnalystNote:      "Penyakit bawaan sebelum masa pertanggungan.",
		RCLDoctor:        SampleLegacyID,
		RegisteredAt:     at(2026, time.September, 17, 1),
		ProcessStatus:    "Pending-RCLDokter",
		AssignedOperator: SampleLegacyID,
	},
	{
		ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0404",
		ClaimNumber:      "PNCN.26.0404",
		PolicyNumber:     "26.005.2026.00404",
		InsuredName:      "Rudi Hartono Contoh",
		SentToRCLAt:      at(2026, time.September, 18, 9),
		AnalystNote:      "",
		RCLDoctor:        SampleLegacyID,
		RegisteredAt:     at(2026, time.September, 17, 1),
		ProcessStatus:    "Open",
		AssignedOperator: SampleLegacyID,
	},
	{
		// HARUS TETAP MUNCUL — hanya `Resolved-Completed` yang dikecualikan.
		ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0380",
		ClaimNumber:      "PNCN.26.0380",
		PolicyNumber:     "26.002.2026.00380",
		InsuredName:      "Siti Rahma Contoh",
		SentToRCLAt:      at(2026, time.September, 10, 4),
		AnalystNote:      "Dokumen medis tidak lengkap.",
		RCLDoctor:        SampleLegacyID,
		RegisteredAt:     at(2026, time.September, 8, 3),
		ProcessStatus:    "Resolved-Rejected",
		AssignedOperator: SampleLegacyID,
	},
	{
		// TIDAK BOLEH MUNCUL — tugasnya sudah tuntas (penyaring B).
		ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0371",
		ClaimNumber:      "PNCN.26.0371",
		PolicyNumber:     "26.002.2026.00371",
		InsuredName:      "Agus Salim Contoh",
		SentToRCLAt:      at(2026, time.September, 6, 2),
		RCLDoctor:        SampleLegacyID,
		RegisteredAt:     at(2026, time.September, 5, 2),
		ProcessStatus:    inboxrcl.StatusKerjaSelesai,
		AssignedOperator: SampleLegacyID,
	},
	{
		// TIDAK BOLEH MUNCUL — belum dikirim analis ke dokter RCL (penyaring C).
		ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0366",
		ClaimNumber:      "PNCN.26.0366",
		PolicyNumber:     "26.002.2026.00366",
		InsuredName:      "Lina Marlina Contoh",
		RCLDoctor:        SampleLegacyID,
		RegisteredAt:     at(2026, time.September, 21, 5),
		ProcessStatus:    "Open",
		AssignedOperator: SampleLegacyID,
	},
	{
		// TIDAK BOLEH MUNCUL — penugasan milik pemanggil, tetapi dokter RCL-nya orang lain
		// (penyaring D).
		ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0359",
		ClaimNumber:      "PNCN.26.0359",
		PolicyNumber:     "26.002.2026.00359",
		InsuredName:      "Budi Santoso Contoh",
		SentToRCLAt:      at(2026, time.September, 21, 6),
		RCLDoctor:        SampleOtherLegacyID,
		RegisteredAt:     at(2026, time.September, 21, 5),
		ProcessStatus:    "Open",
		AssignedOperator: SampleLegacyID,
	},
	{
		// TIDAK BOLEH MUNCUL bagi SampleLegacyID — milik dokter lain (penyaring A dan D).
		ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0350",
		ClaimNumber:      "PNCN.26.0350",
		PolicyNumber:     "26.002.2026.00350",
		InsuredName:      "Dian Permata Contoh",
		SentToRCLAt:      at(2026, time.September, 21, 7),
		RCLDoctor:        SampleOtherLegacyID,
		RegisteredAt:     at(2026, time.September, 21, 6),
		ProcessStatus:    "Open",
		AssignedOperator: SampleOtherLegacyID,
	},
}

// NewSampleStore membentuk pembaca berisi data contoh — salinan, supaya dua portal tidak
// berbagi senarai yang sama (`R-20`).
func NewSampleStore() *Store {
	access := make([]AccessRow, len(SampleAccess))
	copy(access, SampleAccess)
	tasks := make([]inboxrcl.RCLTask, len(SampleTasks))
	copy(tasks, SampleTasks)
	return NewStore(access, tasks)
}
