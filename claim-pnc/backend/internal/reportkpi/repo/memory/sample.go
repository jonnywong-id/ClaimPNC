package memory

import "claim-pnc/internal/reportkpi"

// NewSampleStore membentuk penyimpanan berisi contoh.
//
// # Seluruh isinya KARANGAN
//
// Nama adjuster, nomor case, dan nilainya tidak diambil dari produksi. `D-69` melarang
// data nasabah masuk ke berkas yang di-commit, dan nama adjuster adalah nama orang atau
// perusahaan yang nyata di produksi.
//
// # Yang dicakup contoh ini, dan kenapa masing-masing ada
//
// Ia bukan sekadar "beberapa baris supaya layar tidak kosong". Setiap baris mewakili satu
// keadaan yang layar harus tangani benar, termasuk empat yang paling mudah terlewat:
//
//	dua tipe pada SATU adjuster    membuktikan tipe ALL menghasilkan dua baris ringkasan,
//	                               bukan satu baris gabungan
//	komponen yang KOSONG           membuktikan sel kosong tidak menjadi 0
//	komponen BUKAN ANGKA           membuktikan satu nilai rusak tidak menyeret rata-rata
//	                               komponen itu, dan tidak membuang barisnya
//	tanggal di TEPI rentang        membuktikan batas atas ikut terhitung — cacat
//	                               setengah-terbuka yang paling sering lolos uji
func NewSampleStore() *Store {
	return NewStoreWith(SampleRows()).
		WithAdminRows(SampleAdminRows()).
		WithPICTeknik(SamplePICTeknik())
}

// SampleRows adalah baris contoh, terbuka supaya uji dapat memakainya apa adanya.
func SampleRows() []Row {
	return []Row{
		// ── Adjuster pertama: dua tipe sekaligus ────────────────────────────────
		//
		// Ia yang membuktikan perilaku tipe ALL. Pada OUTSTANDING nilainya lebih
		// rendah daripada pada FINAL, sehingga kedua barisnya dapat dibedakan sekilas
		// dan baris yang tertukar akan terlihat.
		{
			Adjuster: "PT ADJUSTER NUSA CONTOH", CaseID: "CONTOH-KPI-0001",
			ReportType: reportkpi.TypeOutstanding, ScoredOn: "2026-03-04",
			Scores: fullScores("3", "4", "3", "2", "4", "3", "3", "2", "3"),
		},
		{
			Adjuster: "PT ADJUSTER NUSA CONTOH", CaseID: "CONTOH-KPI-0002",
			ReportType: reportkpi.TypeOutstanding, ScoredOn: "2026-03-18",
			Scores: fullScores("4", "4", "4", "3", "5", "4", "4", "3", "4"),
		},
		{
			Adjuster: "PT ADJUSTER NUSA CONTOH", CaseID: "CONTOH-KPI-0003",
			ReportType: reportkpi.TypeFinal, ScoredOn: "2026-03-25",
			Scores: fullScores("5", "5", "5", "5", "5", "5", "5", "5", "5"),
		},

		// ── Adjuster kedua: komponen yang belum dinilai ─────────────────────────
		//
		// Tiga komponen sengaja TIDAK ADA di peta, yang berarti kolomnya NULL. Layar
		// harus menggambarnya sebagai tanda hubung, bukan sebagai 0 — dan
		// rata-ratanya harus KOSONG pula, bukan nol.
		{
			Adjuster: "CV SURVEI CONTOH SEJAHTERA", CaseID: "CONTOH-KPI-0004",
			ReportType: reportkpi.TypeOutstanding, ScoredOn: "2026-03-09",
			Scores: map[string]string{
				reportkpi.ComponentSurvey:            "4",
				reportkpi.ComponentImmediateAdvice:   "3",
				reportkpi.ComponentPreliminaryAdvice: "3",
				// INTERIM REPORT, UPDATE PROGRESS, dan TANGGAPAN KOMUNIKASI
				// sengaja tidak ada.
				reportkpi.ComponentPropose:     "4",
				reportkpi.ComponentFinalReport: "4",
				reportkpi.ComponentTotal:       "3.6",
			},
		},

		// ── Adjuster ketiga: satu nilai BUKAN ANGKA ─────────────────────────────
		//
		// Kolomnya bertipe teks di Oracle, sehingga isi seperti ini NYATA-nyata
		// mungkin. Di sini ia membuktikan dua hal sekaligus: komponen itu tidak ikut
		// rata-rata, dan kedelapan komponen lain pada baris yang sama TETAP ikut.
		//
		// Catatan yang perlu diingat saat membaca hasil uji: terhadap Oracle,
		// `TO_NUMBER` pada baris seperti ini MENGGAGALKAN laporan dengan `ORA-01722`
		// — lihat alasannya di reportkpi.sql. Pengisi memori sengaja lebih pemaaf
		// supaya layar dapat diuji; selisih itu ada di penanganan data rusak, bukan
		// pada data yang bersih.
		{
			Adjuster: "PT PENILAI CONTOH PRATAMA", CaseID: "CONTOH-KPI-0005",
			ReportType: reportkpi.TypeFinal, ScoredOn: "2026-03-31",
			Scores: fullScores("5", "4", "N/A", "4", "4", "5", "4", "5", "4.5"),
		},

		// ── Tepi rentang ────────────────────────────────────────────────────────
		//
		// Dua baris pada 1 Maret dan 31 Maret. Rentang 2026-03-01..2026-03-31 WAJIB
		// memuat keduanya; penyaring yang keliru menulis `< sampai` alih-alih
		// `< sampai + 1 hari` akan membuang yang kedua tanpa satu pun galat.
		{
			Adjuster: "PT TEPI CONTOH MANDIRI", CaseID: "CONTOH-KPI-0006",
			ReportType: reportkpi.TypeFinal, ScoredOn: "2026-03-01",
			Scores: fullScores("2", "2", "2", "2", "2", "2", "2", "2", "2"),
		},
		{
			Adjuster: "PT TEPI CONTOH MANDIRI", CaseID: "CONTOH-KPI-0007",
			ReportType: reportkpi.TypeFinal, ScoredOn: "2026-03-31",
			Scores: fullScores("4", "4", "4", "4", "4", "4", "4", "4", "4"),
		},

		// ── DI LUAR rentang contoh ──────────────────────────────────────────────
		//
		// Sehari sebelum dan sehari sesudah Maret. Keduanya ada supaya uji dapat
		// membuktikan penyaring periode BENAR-BENAR menyaring — penyaring yang tidak
		// bekerja sama sekali tidak dapat dibedakan dari penyaring yang bekerja, bila
		// seluruh data kebetulan berada di dalam rentangnya.
		{
			Adjuster: "PT LUAR CONTOH PERIODE", CaseID: "CONTOH-KPI-0008",
			ReportType: reportkpi.TypeFinal, ScoredOn: "2026-02-28",
			Scores: fullScores("1", "1", "1", "1", "1", "1", "1", "1", "1"),
		},
		{
			Adjuster: "PT LUAR CONTOH PERIODE", CaseID: "CONTOH-KPI-0009",
			ReportType: reportkpi.TypeFinal, ScoredOn: "2026-04-01",
			Scores: fullScores("1", "1", "1", "1", "1", "1", "1", "1", "1"),
		},
	}
}

// fullScores menyusun peta kesembilan komponen dalam urutan kolom layar.
//
// Parameternya sengaja tidak bernama satu per satu: urutannya SAMA dengan
// reportkpi.ComponentCodes(), dan menuliskan sembilan nama parameter di sini hanya
// menambah tempat kedua yang harus dijaga kesamaannya.
func fullScores(values ...string) map[string]string {
	codes := reportkpi.ComponentCodes()

	result := make(map[string]string, len(codes))
	for i, code := range codes {
		if i >= len(values) {
			break
		}
		result[code] = values[i]
	}
	return result
}
