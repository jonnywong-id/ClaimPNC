package memory

import (
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// Data contoh untuk mode pengembangan tanpa Oracle.
//
// # Yang sengaja TIDAK ada di sini
//
// Tidak ada nomor polis, nama tertanggung, nomor klaim, NPWP, maupun nomor rekening yang
// berasal dari data nyata. Seluruhnya dikarang, dan bentuknya sengaja dibuat terlihat
// dikarang — "Tertanggung Contoh A", bukan nama orang yang masuk akal.
//
// `D-69` melarang data nasabah ditulis ke berkas yang di-commit. Larangan itu berlaku pada
// berkas contoh persis seperti pada dokumen: yang membuatnya berbahaya bukan tempatnya,
// melainkan isinya.
//
// # Kenapa contohnya mencakup kelima lini bisnis
//
// Supaya penyaring lini bisnis dapat dicoba tanpa basis data, dan supaya salah satunya —
// Group Panel `009` — memperlihatkan perilaku yang mudah terlewat: ia tidak tampil pada
// pilihan mana pun selain "Semua Lini Bisnis". Itu perilaku sistem lama yang direplikasi
// (`P-5`), dan menaruhnya di data contoh membuatnya terlihat saat dicoba, bukan hanya
// tertulis di komentar.
func sampleTime(day int) time.Time {
	return time.Date(2026, time.August, day, 3, 0, 0, 0, time.UTC)
}

func sampleDate(day int) *time.Time {
	moment := time.Date(2026, time.July, day, 0, 0, 0, 0, time.UTC)
	return &moment
}

// SampleOutstanding adalah klaim berjalan contoh.
func SampleOutstanding() []ClaimRecord {
	return []ClaimRecord{
		{
			GroupPanel: "003", BusinessGroupID: "10001",
			Row: dashboardclaim.ClaimRow{
				ClaimID: "CONTOH-BERJALAN-1", ClaimNumber: "PNCN.26.0001",
				PolicyNumber: "POLIS-CONTOH-0001", InsuredName: "Tertanggung Contoh A",
				BusinessName: "Aneka", BusinessSource: "Cabang", BranchName: "Jakarta",
				TechnicalPIC: "PIC Contoh 1", AdminPNC: "Admin Contoh 1",
				ClaimStatusCode: "1147", ProcessStatus: "New",
				LossDate: sampleDate(4), RegisteredAt: sampleTime(5),
			},
		},
		{
			GroupPanel: "002", BusinessGroupID: "10002",
			Row: dashboardclaim.ClaimRow{
				ClaimID: "CONTOH-BERJALAN-2", ClaimNumber: "PNCN.26.0002",
				PolicyNumber: "POLIS-CONTOH-0002", InsuredName: "Tertanggung Contoh B",
				BusinessName: "Personal Accident", BusinessSource: "Broker", BranchName: "Surabaya",
				TechnicalPIC: "PIC Contoh 2", AdminPNC: "Admin Contoh 2",
				ClaimStatusCode: "1149", ProcessStatus: "Pending",
				LossDate: sampleDate(6), RegisteredAt: sampleTime(7),
			},
		},
		{
			GroupPanel: "005", BusinessGroupID: "10003",
			Row: dashboardclaim.ClaimRow{
				ClaimID: "CONTOH-BERJALAN-3", ClaimNumber: "PNCN.26.0003",
				PolicyNumber: "POLIS-CONTOH-0003", InsuredName: "Tertanggung Contoh C",
				BusinessName: "Travel", BusinessSource: "Agen", BranchName: "Denpasar",
				TechnicalPIC: "PIC Contoh 3", AdminPNC: "Admin Contoh 3",
				ClaimStatusCode: "1147", ProcessStatus: "New",
				LossDate: sampleDate(8), RegisteredAt: sampleTime(9),
			},
		},
		{
			GroupPanel: "004", BusinessGroupID: "10008",
			Row: dashboardclaim.ClaimRow{
				ClaimID: "CONTOH-BERJALAN-4", ClaimNumber: "PNCN.26.0004",
				PolicyNumber: "POLIS-CONTOH-0004", InsuredName: "Tertanggung Contoh D",
				BusinessName: "Bonding", BusinessSource: "Cabang", BranchName: "Medan",
				TechnicalPIC: "PIC Contoh 4", AdminPNC: "Admin Contoh 4",
				ClaimStatusCode: "1149", ProcessStatus: "New",
				LossDate: sampleDate(10), RegisteredAt: sampleTime(11),
			},
		},
		{
			// Group Panel 009 — varian Aneka yang TIDAK termasuk Non-MBU di sistem lama.
			GroupPanel: "009", BusinessGroupID: "10004",
			Row: dashboardclaim.ClaimRow{
				ClaimID: "CONTOH-BERJALAN-5", ClaimNumber: "PNCN.26.0005",
				PolicyNumber: "POLIS-CONTOH-0005", InsuredName: "Tertanggung Contoh E",
				BusinessName: "Aneka Varian", BusinessSource: "Cabang", BranchName: "Semarang",
				TechnicalPIC: "PIC Contoh 5", AdminPNC: "Admin Contoh 5",
				ClaimStatusCode: "1147", ProcessStatus: "New",
				LossDate: sampleDate(12), RegisteredAt: sampleTime(13),
			},
		},
	}
}

// SampleClosed adalah klaim tutup contoh.
func SampleClosed() []ClaimRecord {
	return []ClaimRecord{
		{
			GroupPanel: "003", BusinessGroupID: "10001",
			Row: dashboardclaim.ClaimRow{
				ClaimID: "CONTOH-TUTUP-1", ClaimNumber: "PNCN.26.0101",
				PolicyNumber: "POLIS-CONTOH-0101", InsuredName: "Tertanggung Contoh F",
				BusinessName: "Aneka", BusinessSource: "Cabang", BranchName: "Jakarta",
				TechnicalPIC: "PIC Contoh 1", AdminPNC: "Admin Contoh 1",
				ClaimStatusCode: "1163", ProcessStatus: "Resolved-Completed",
				LossDate: sampleDate(1), RegisteredAt: sampleTime(2),
			},
		},
		{
			GroupPanel: "002", BusinessGroupID: "10002",
			Row: dashboardclaim.ClaimRow{
				ClaimID: "CONTOH-TUTUP-2", ClaimNumber: "PNCN.26.0102",
				PolicyNumber: "POLIS-CONTOH-0102", InsuredName: "Tertanggung Contoh G",
				BusinessName: "Personal Accident", BusinessSource: "Broker", BranchName: "Surabaya",
				TechnicalPIC: "PIC Contoh 2", AdminPNC: "Admin Contoh 2",
				ClaimStatusCode: "1142", ProcessStatus: "Resolved-Rejected",
				LossDate: sampleDate(3), RegisteredAt: sampleTime(4),
			},
		},
	}
}

// SampleSurveys adalah survei contoh untuk kedua tile survei.
func SampleSurveys() []SurveyRecord {
	return []SurveyRecord{
		{
			Kind: dashboardclaim.SurveyorAdjuster, GroupPanel: "003", BusinessGroupID: "10001",
			Row: dashboardclaim.SurveyRow{
				SurveyID: "CONTOH-SURVEI-1", SurveyNumber: "SRV-CONTOH-0001",
				ClaimNumber: "PNCN.26.0001", PolicyNumber: "POLIS-CONTOH-0001",
				InsuredName: "Tertanggung Contoh A", ReferenceNumber: "REF-CONTOH-01",
				SurveyorName: "Adjuster Contoh 1", TechnicalPIC: "PIC Contoh 1",
				SurveyLocation: "Jakarta", SurveyStatus: "Assigned",
				ProcessStatus: "Open", AdjusterPIC: "PIC Adjuster Contoh 1",
				AssignedAt: sampleTime(6),
			},
		},
		{
			Kind: dashboardclaim.SurveyorAdjuster, GroupPanel: "002", BusinessGroupID: "10002",
			Row: dashboardclaim.SurveyRow{
				SurveyID: "CONTOH-SURVEI-2", SurveyNumber: "SRV-CONTOH-0002",
				ClaimNumber: "PNCN.26.0002", PolicyNumber: "POLIS-CONTOH-0002",
				InsuredName: "Tertanggung Contoh B", ReferenceNumber: "REF-CONTOH-02",
				SurveyorName: "Adjuster Contoh 2", TechnicalPIC: "PIC Contoh 2",
				SurveyLocation: "Surabaya", SurveyStatus: "On Progress",
				ProcessStatus: "Open", AdjusterPIC: "PIC Adjuster Contoh 2",
				AssignedAt: sampleTime(8),
			},
		},
		{
			Kind: dashboardclaim.SurveyorInternal, GroupPanel: "003", BusinessGroupID: "10001",
			Row: dashboardclaim.SurveyRow{
				SurveyID: "CONTOH-SURVEI-3", SurveyNumber: "SRV-CONTOH-0003",
				ClaimNumber: "PNCN.26.0001", PolicyNumber: "POLIS-CONTOH-0001",
				InsuredName: "Tertanggung Contoh A", ReferenceNumber: "REF-CONTOH-03",
				SurveyorName: "Surveyor Contoh 1", TechnicalPIC: "PIC Contoh 1",
				SurveyLocation: "Jakarta", SurveyStatus: "Scheduled",
				ProcessStatus: "Open", AdjusterPIC: "Surveyor Contoh 1",
				ScheduledAt: sampleDate(20), AssignedAt: sampleTime(10),
			},
		},
		{
			Kind: dashboardclaim.SurveyorInternal, GroupPanel: "005", BusinessGroupID: "10003",
			Row: dashboardclaim.SurveyRow{
				SurveyID: "CONTOH-SURVEI-4", SurveyNumber: "SRV-CONTOH-0004",
				ClaimNumber: "PNCN.26.0003", PolicyNumber: "POLIS-CONTOH-0003",
				InsuredName: "Tertanggung Contoh C", ReferenceNumber: "REF-CONTOH-04",
				SurveyorName: "Surveyor Contoh 2", TechnicalPIC: "PIC Contoh 3",
				SurveyLocation: "Denpasar", SurveyStatus: "Scheduled",
				ProcessStatus: "Open", AdjusterPIC: "Surveyor Contoh 2",
				ScheduledAt: sampleDate(22), AssignedAt: sampleTime(12),
			},
		},
	}
}
