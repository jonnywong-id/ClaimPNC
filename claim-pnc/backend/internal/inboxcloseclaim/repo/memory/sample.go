package memory

import (
	"time"

	"claim-pnc/internal/inboxcloseclaim"
)

// SampleClaims adalah data contoh untuk menjalankan aplikasi tanpa Oracle.
//
// # SELURUHNYA KARANGAN
//
// Tidak ada satu pun nomor polis, nama tertanggung, atau nomor klaim nyata di sini.
// `D-69` menetapkan data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit — dan
// berkas contoh adalah tempat yang paling mudah melanggarnya tanpa disadari, karena data
// nyata membuat layarnya terlihat lebih meyakinkan.
//
// # Yang sengaja diwakili
//
// Barisnya dipilih supaya setiap cabang penyaring punya sekurangnya satu contoh, dan
// supaya keadaan yang mudah keliru terlihat langsung di layar:
//
//	PNCN.26.0001  Non-MBU · sudah transfer · LUNAS
//	PNCN.26.0002  Non-MBU · belum transfer · belum lunas
//	PNCN.26.0003  Bonding · kelompok bisnis Bonding, untuk menguji arah `IN`
//	PNCN.26.0004  PA      · ditolak, bukan selesai — kolom status menampilkan "Reject"
//	PNCN.26.0005  Travel  · TANPA kode status, untuk memperlihatkan baris yang HILANG
//	              saat penyaring "BELUM LUNAS" dipakai
//	PNC-9001      klaim warisan tanpa tanggal tutup — Lama Waktu Klaim jatuh ke cadangan
func SampleClaims() []inboxcloseclaim.ClosedClaim {
	daftar := time.Date(2026, 1, 12, 3, 0, 0, 0, time.UTC)

	return []inboxcloseclaim.ClosedClaim{
		{
			ClaimID:              "ASM-FW-GCNMFW-WORK PNCN.26.0001",
			ClaimNumber:          "PNCN.26.0001",
			PolicyNumber:         "POL-CONTOH-0001",
			InsuredName:          "Tertanggung Contoh Satu",
			BusinessName:         "Property All Risk",
			BusinessSource:       "Cabang",
			BranchName:           "JAKARTA",
			GroupPanel:           "006",
			BusinessGroupID:      "10001",
			RegisteredAt:         daftar,
			LossDate:             ptrTime(daftar.AddDate(0, 0, -5)),
			ClosedAt:             ptrTime(daftar.AddDate(0, 0, 41)),
			ProcessStatus:        inboxcloseclaim.StatusKerjaSelesai,
			ClaimStatusCode:      inboxcloseclaim.KodeStatusLunas,
			ClaimStatusLabel:     "Paid",
			TechnicalPIC:         "PIC TEKNIK CONTOH",
			AdminPNC:             "ADMIN CONTOH",
			TransferredToCashier: true,
		},
		{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0002",
			ClaimNumber:      "PNCN.26.0002",
			PolicyNumber:     "POL-CONTOH-0002",
			InsuredName:      "Tertanggung Contoh Dua",
			BusinessName:     "Marine Cargo",
			BusinessSource:   "Broker",
			BranchName:       "SURABAYA",
			GroupPanel:       "004",
			BusinessGroupID:  "10002",
			RegisteredAt:     daftar.AddDate(0, 0, 3),
			LossDate:         ptrTime(daftar.AddDate(0, 0, -1)),
			ClosedAt:         ptrTime(daftar.AddDate(0, 0, 20)),
			ProcessStatus:    inboxcloseclaim.StatusKerjaSelesai,
			ClaimStatusCode:  "1143",
			ClaimStatusLabel: "Close Claim for this object",
			TechnicalPIC:     "PIC TEKNIK CONTOH",
			AdminPNC:         "ADMIN CONTOH DUA",
		},
		{
			ClaimID:              "ASM-FW-GCNMFW-WORK PNCN.26.0003",
			ClaimNumber:          "PNCN.26.0003",
			PolicyNumber:         "POL-CONTOH-0003",
			InsuredName:          "Tertanggung Contoh Tiga",
			BusinessName:         "Bonding",
			BusinessSource:       "Cabang",
			BranchName:           "BANDUNG",
			GroupPanel:           "003",
			BusinessGroupID:      "10008",
			RegisteredAt:         daftar.AddDate(0, 0, 6),
			ClosedAt:             ptrTime(daftar.AddDate(0, 0, 90)),
			ProcessStatus:        inboxcloseclaim.StatusKerjaSelesai,
			ClaimStatusCode:      "1163",
			ClaimStatusLabel:     "Paid",
			TechnicalPIC:         "PIC BONDING CONTOH",
			AdminPNC:             "ADMIN CONTOH",
			TransferredToCashier: true,
		},
		{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0004",
			ClaimNumber:      "PNCN.26.0004",
			PolicyNumber:     "POL-CONTOH-0004",
			InsuredName:      "Tertanggung Contoh Empat",
			BusinessName:     "Personal Accident",
			BusinessSource:   "Agen",
			BranchName:       "MEDAN",
			GroupPanel:       "002",
			BusinessGroupID:  "10003",
			RegisteredAt:     daftar.AddDate(0, 0, 9),
			ClosedAt:         ptrTime(daftar.AddDate(0, 0, 15)),
			ProcessStatus:    inboxcloseclaim.StatusKerjaDitolak,
			ClaimStatusCode:  "1142",
			ClaimStatusLabel: "Rejected Claim",
			TechnicalPIC:     "PIC PA CONTOH",
			AdminPNC:         "ADMIN CONTOH TIGA",
		},
		{
			// Kode statusnya sengaja KOSONG. Baris ini HILANG saat penyaring "BELUM LUNAS"
			// dipakai — perilaku `<>` terhadap NULL di Oracle, yang ditiru apa adanya.
			// Lihat unpaidMatches.
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0005",
			ClaimNumber:     "PNCN.26.0005",
			PolicyNumber:    "POL-CONTOH-0005",
			InsuredName:     "Tertanggung Contoh Lima",
			BusinessName:    "Travel",
			BusinessSource:  "Online",
			BranchName:      "DENPASAR",
			GroupPanel:      "005",
			BusinessGroupID: "10004",
			RegisteredAt:    daftar.AddDate(0, 0, 11),
			ClosedAt:        ptrTime(daftar.AddDate(0, 0, 18)),
			ProcessStatus:   inboxcloseclaim.StatusKerjaSelesai,
			TechnicalPIC:    "PIC TRAVEL CONTOH",
			AdminPNC:        "ADMIN CONTOH",
		},
		{
			// Klaim warisan: bernomor `PNC-xxxx` (`D-22`) dan TANPA tanggal tutup.
			// Kolom Lama Waktu Klaim jatuh ke PYRESOLVEDTIMESTAMP — cadangan pertama.
			ClaimID:          "ASM-FW-GCNMFW-WORK PNC-9001",
			ClaimNumber:      "PNC-9001",
			PolicyNumber:     "POL-CONTOH-9001",
			InsuredName:      "Tertanggung Contoh Warisan",
			BusinessName:     "Property All Risk",
			BusinessSource:   "Cabang",
			BranchName:       "SEMARANG",
			GroupPanel:       "006",
			BusinessGroupID:  "10001",
			RegisteredAt:     daftar.AddDate(-1, 0, 0),
			ResolvedAt:       ptrTime(daftar.AddDate(-1, 0, 64)),
			ProcessStatus:    inboxcloseclaim.StatusKerjaSelesai,
			ClaimStatusCode:  "1144",
			ClaimStatusLabel: "Cancelled Claim",
			TechnicalPIC:     "PIC TEKNIK CONTOH",
			AdminPNC:         "ADMIN CONTOH",
		},
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
