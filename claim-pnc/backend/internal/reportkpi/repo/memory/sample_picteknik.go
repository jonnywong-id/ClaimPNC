package memory

import (
	"time"

	"claim-pnc/internal/reportkpi"
)

// Bahan contoh tab KPI PIC Teknik.
//
// Seluruh nama dan angkanya KARANGAN — `D-69` melarang data nasabah maupun identitas nyata
// ditulis di berkas yang di-commit, dan larangan itu berlaku untuk data contoh sama seperti
// untuk dokumen.
//
// Isi TANGGA NILAI-nya bukan karangan: ia disalin apa adanya dari `POOLDATA.M_KPI_PNC`,
// karena justru tangga itulah yang menentukan hasil dan yang perlu dapat diperiksa.

// sampleDay membentuk satu tanggal contoh pada Maret 2026.
func sampleDay(day int) time.Time {
	return time.Date(2026, 3, day, 0, 0, 0, 0, time.UTC)
}

// SamplePICTeknik menyusun bahan contoh tab KPI PIC Teknik.
//
// # Hasil yang dapat diperiksa dengan tangan
//
// Periode 1–31 Maret 2026, lini NONMBU, dua petugas:
//
//	CONTOHPICSATU — progres 10 pembaruan, 9 tepat waktu
//	  persen 90  → tangga MENURUN → pita 35–100 → nilai 1
//	  berbobot   → round(round(0,9)*100/20, 2) = 4,5 → ×3 = 13,5
//
//	CONTOHPICDUA  — progres 10 pembaruan, 2 tepat waktu
//	  persen 20  → pita 0–20 → nilai 5
//
// Perhatikan yang rajin bernilai 1 dan yang jarang bernilai 5. Itu BUKAN kekeliruan contoh
// ini; itu perilaku Pega yang direplikasi — lihat `PICTeknikPlannedDifferences`.
func SamplePICTeknik() PICTeknikData {
	return PICTeknikData{
		PICs: []reportkpi.PICProfile{
			{OperatorID: "CONTOHPICSATU", Leader: true},
			{OperatorID: "CONTOHPICDUA"},
		},

		Bands: sampleBands(),

		Thresholds: []ThresholdRow{
			{Job: reportkpi.JobAnalysis, Days: 10},
			{Job: reportkpi.JobAcceptance, Days: 1},
			{Job: reportkpi.JobSLA, Note: reportkpi.TeamLeader, Days: 390},
			{Job: reportkpi.JobSLA, Note: reportkpi.TeamMember, Days: 399},
		},

		// Satu hari libur di tengah periode, pada hari kerja.
		Holidays: []time.Time{sampleDay(18)},

		Progress: sampleProgress(),

		Analysis: []SpanRow{
			// Selesai dalam 3 hari kerja — di bawah ambang 10, jadi tepat waktu.
			{
				PIC: "CONTOHPICSATU", Line: reportkpi.LineNonMBU,
				Registered: sampleDay(2), Start: sampleDay(2), End: sampleDay(5),
			},
			// Selesai dalam 15 hari kerja — melewati ambang.
			{
				PIC: "CONTOHPICDUA", Line: reportkpi.LineNonMBU,
				Registered: sampleDay(2), Start: sampleDay(2), End: sampleDay(24),
			},
		},

		Acceptance: []AcceptanceRow{
			// LEADER memakai pasangan ReceiveLOD → Accepted: 1 hari kerja, tepat waktu.
			{
				PIC: "CONTOHPICSATU", Line: reportkpi.LineNonMBU, Team: reportkpi.TeamLeader,
				ReceiveLOD: sampleDay(9), Committee: sampleDay(2), Accepted: sampleDay(10),
			},
			// MEMBER memakai pasangan Committee → Accepted: 8 hari kerja, terlambat.
			{
				PIC: "CONTOHPICDUA", Line: reportkpi.LineNonMBU, Team: reportkpi.TeamMember,
				ReceiveLOD: sampleDay(9), Committee: sampleDay(2), Accepted: sampleDay(12),
			},
		},

		Closure: []ClosureRow{
			{
				PIC: "CONTOHPICSATU", Line: reportkpi.LineNonMBU, Team: reportkpi.TeamLeader,
				Registered: sampleDay(2), Closed: sampleDay(20),
			},
			{
				PIC: "CONTOHPICDUA", Line: reportkpi.LineNonMBU, Team: reportkpi.TeamMember,
				Registered: sampleDay(3), Closed: sampleDay(25),
			},
		},
	}
}

// sampleBands menyalin tangga nilai PIC dari `POOLDATA.M_KPI_PNC` apa adanya.
//
// Dua tangga pertama MENURUN dan dua berikutnya MENAIK. Perbedaan arah itu sengaja dibiarkan
// terlihat di sini: ia inti dari temuan yang dicatat sebagai selisih terencana.
func sampleBands() []reportkpi.Band {
	return []reportkpi.Band{
		// UPDATE PROGRESS KLAIM — MENURUN.
		{Job: reportkpi.JobProgress, Value: 5, Bottom: 0, Top: 20},
		{Job: reportkpi.JobProgress, Value: 4, Bottom: 20, Top: 25},
		{Job: reportkpi.JobProgress, Value: 3, Bottom: 25, Top: 30},
		{Job: reportkpi.JobProgress, Value: 2, Bottom: 30, Top: 35},
		{Job: reportkpi.JobProgress, Value: 1, Bottom: 35, Top: 100},

		// ANALISA KLAIM — MENAIK.
		{Job: reportkpi.JobAnalysis, Value: 5, Bottom: 80, Top: 100},
		{Job: reportkpi.JobAnalysis, Value: 4, Bottom: 75, Top: 80},
		{Job: reportkpi.JobAnalysis, Value: 3, Bottom: 70, Top: 75},
		{Job: reportkpi.JobAnalysis, Value: 2, Bottom: 65, Top: 70},
		{Job: reportkpi.JobAnalysis, Value: 1, Bottom: 0, Top: 65},

		// AKSEPTASI KLAIM — MENAIK.
		{Job: reportkpi.JobAcceptance, Value: 5, Bottom: 80, Top: 100},
		{Job: reportkpi.JobAcceptance, Value: 4, Bottom: 75, Top: 80},
		{Job: reportkpi.JobAcceptance, Value: 3, Bottom: 70, Top: 75},
		{Job: reportkpi.JobAcceptance, Value: 2, Bottom: 65, Top: 70},
		{Job: reportkpi.JobAcceptance, Value: 1, Bottom: 0, Top: 65},

		// SLA KLAIM — MENURUN. Kolom NOTE hanya terisi pada dua baris teratas.
		{Job: reportkpi.JobSLA, Value: 5, Bottom: 0, Top: 20, Note: reportkpi.TeamLeader},
		{Job: reportkpi.JobSLA, Value: 4, Bottom: 20, Top: 25, Note: reportkpi.TeamMember},
		{Job: reportkpi.JobSLA, Value: 3, Bottom: 25, Top: 30},
		{Job: reportkpi.JobSLA, Value: 2, Bottom: 30, Top: 35},
		{Job: reportkpi.JobSLA, Value: 1, Bottom: 35, Top: 100},
	}
}

// sampleProgress menyusun pembaruan progres contoh: 9 dari 10 tepat waktu, lalu 2 dari 10.
func sampleProgress() []ProgressRow {
	rows := make([]ProgressRow, 0, 20)

	for i := 0; i < 10; i++ {
		rows = append(rows, ProgressRow{
			PIC: "CONTOHPICSATU", Line: reportkpi.LineNonMBU,
			At: sampleDay(2 + i), OnTime: i < 9,
		})
	}
	for i := 0; i < 10; i++ {
		rows = append(rows, ProgressRow{
			PIC: "CONTOHPICDUA", Line: reportkpi.LineNonMBU,
			At: sampleDay(2 + i), OnTime: i < 2,
		})
	}

	return rows
}
