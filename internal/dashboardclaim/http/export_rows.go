package dashboardclaimhttp

import (
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// ============================================================================
// SUSUNAN KOLOM BERKAS UNDUHAN — EMPAT TILE, EMPAT SUSUNAN
// ============================================================================
//
// Keempatnya DISALIN dari berkas unduhan Pega, bukan dikarang: judul kolomnya, urutannya,
// jumlahnya, dan bahasanya. Sumber tiap susunan ada di parameter `pxConvertResultsToCSV`
// pada activity-nya, tempat `CSVPropHeaders` dan `CSVProperties` berpasangan satu-satu —
// dan pasangan itulah yang membuat pemetaannya tidak perlu ditebak.
//
//	Outstanding        Activity/ExportOutstanding_Act-Act.xml   "Laporan Outstanding "
//	Close Claim        Activity/ExportDataCloseClaim-Act.xml    "Data Close Claim"
//	Loss Adjuster      Activity/ExportSurveyorClaim-Act.xml     "Data Lost Adjuster"
//	Internal Surveyor  Activity/ExportSurveyorClaim-Act.xml     "Data Internal Surveyor"
//
// Kedua tile survei dilayani SATU activity dengan dua cabang; itu sebabnya judulnya berbeda
// bahasa — Loss Adjuster berbahasa Inggris, Internal Surveyor berbahasa Indonesia. Ditiru
// apa adanya (`D-13`): yang membuka berkasnya mencari judul yang sudah dikenalnya.
//
// ----------------------------------------------------------------------------
// TIGA HAL YANG SENGAJA TIDAK DISALIN, DAN ALASANNYA
// ----------------------------------------------------------------------------
//
//  1. SPASI DI DEPAN JUDUL. Pega menulis `"No Klaim, No Polis,TSI,…"` — sebagian judul
//     berawal spasi, sebagian tidak, tanpa pola. Excel menampilkannya sebagai " No Polis".
//     Itu kelalaian pengetikan, bukan isi; judulnya dirapikan.
//
//  2. TANGGAL MENTAH PEGA. Berkas Close Claim dan Internal Surveyor memuat
//     `20171227T081806.037 GMT` — nilai internal Pega yang lolos tanpa diformat, dan dalam
//     GMT. Excel tidak mengenalinya sebagai tanggal, dan pembacanya di Jakarta membaca jam
//     yang meleset tujuh jam. Berkas Outstanding Pega sendiri sudah memformat tanggalnya
//     `dd/mm/yyyy` lewat `to_char`, jadi yang dipakai di sini adalah bentuk yang Pega pakai
//     saat ia memformat sama sekali — diterapkan ke keempat berkas, dalam WIB (`F-5`).
//
//  3. AWALAN KELAS PEGA PADA NOMOR KLAIM. Berkas Internal Surveyor Pega menulis
//     `ASM-FW-GCNMFW-WORK PNC-116625`, karena kuerinya mengambil `a.CASEID_1` mentah
//     (`RDB List/BrowseInternalSurveyor-SQL.xml`) sementara berkas Loss Adjuster mengambil
//     `pyid` induknya sehingga bersih. `D-22` menetapkan awalan itu tidak dibawa ke sistem
//     baru, dan grid di layar pun sudah menampilkannya tanpa awalan. Yang ditulis di sini
//     `PNC-116625`.
//
// Ketiganya perbedaan yang DISENGAJA dan dinyatakan, bukan ditemukan belakangan pada
// gerbang 1. Bila Work Owner menghendaki bentuk Pega apa adanya, ketiganya masing-masing
// satu baris untuk dikembalikan.

// exportHeader adalah judul kolom berkas unduhan satu tile.
func exportHeader(tile dashboardclaim.Tile) []string {
	switch tile {
	case dashboardclaim.TileOutstanding:
		// `CSVPropHeaders` ExportOutstanding_Act, 16 kolom.
		return []string{
			"No Klaim",
			"No Polis",
			"TSI",
			"EstimasiValue(IDR)",
			"Nama Tertanggung",
			"Nama Bisnis",
			"Sumber Bisnis",
			"Nama Cabang",
			"Nama Admin",
			"Tgl Kejadian",
			"Tgl Register",
			"Tgl Transfer PIC",
			"Posisi Klaim",
			"Login Terakhir",
			"PIC Teknik",
			"Report Date",
		}

	case dashboardclaim.TileCloseClaim:
		// `CSVPropHeaders` ExportDataCloseClaim, 9 kolom.
		//
		// Susunannya BERBEDA dari Outstanding, dan itu memang begitu di Pega: keduanya
		// berkas yang berbeda dari activity yang berbeda. Menyatukan keduanya menjadi satu
		// susunan — yang pernah dilakukan di sini, dengan sel kosong pada kolom yang tidak
		// berlaku — membuat KEDUANYA tidak sama dengan berkas Pega mana pun.
		return []string{
			"No Klaim",
			"No Polis",
			"Nama Tertanggung",
			"Nama Bisnis",
			"Sumber Bisnis",
			"Nama Cabang",
			"Tanggal Pendaftaran",
			"PIC Teknik",
			"Admin PNC",
		}

	case dashboardclaim.TileLossAdjuster:
		// `CSVPropHeaders` ExportSurveyorClaim cabang adjuster, 9 kolom.
		return []string{
			"Appointment No",
			"Reference No",
			"Claim No",
			"Policy No",
			"Adjuster",
			"PIC Adjuster",
			"Insured Name",
			"PIC ASM",
			"Status",
		}

	default:
		// `CSVPropHeaders` ExportSurveyorClaim cabang internal, 7 kolom.
		//
		// "Reschedule", bukan "Survey": kolomnya memang `RescheduleDate_1` dan
		// `RescheduleLocation_1`, dan itu nama yang dibaca pengguna di berkasnya.
		return []string{
			"Nomor Case",
			"No Klaim",
			"No Polis",
			"Nama Tertanggung",
			"Tanggal Reschedule",
			"Lokasi Reschedule",
			"Status",
		}
	}
}

// claimExportRow menuliskan satu baris tile bertipe klaim, dalam susunan tile-nya sendiri.
func claimExportRow(
	row dashboardclaim.ClaimRow,
	tile dashboardclaim.Tile,
	loc *time.Location,
) []string {
	if tile == dashboardclaim.TileCloseClaim {
		return []string{
			row.ClaimNumber,
			row.PolicyNumber,
			row.InsuredName,
			row.BusinessName,
			row.BusinessSource,
			row.BranchName,
			formatTanggalUnduhan(&row.RegisteredAt, loc),
			row.TechnicalPIC,
			row.AdminPNC,
		}
	}

	// Outstanding. Urutannya mengikuti `CSVProperties` Pega satu-satu — termasuk TSI dan
	// EstimasiValue yang duduk di kolom ketiga dan keempat, sebelum nama tertanggung.
	return []string{
		row.ClaimNumber,
		row.PolicyNumber,
		row.TSI,
		row.EstimateIDR,
		row.InsuredName,
		row.BusinessName,
		row.BusinessSource,
		row.BranchName,
		row.OriginUser,
		formatTanggalUnduhan(row.LossDate, loc),
		formatTanggalUnduhan(&row.RegisteredAt, loc),
		formatTanggalUnduhan(row.TransferredToPICAt, loc),
		row.ClaimStatusLabel,
		row.AssignedOperator,
		row.TechnicalPIC,
		formatTanggalUnduhan(row.ReportDate, loc),
	}
}

// surveyExportRow menuliskan satu baris survei, dalam susunan tile-nya sendiri.
func surveyExportRow(
	row dashboardclaim.SurveyRow,
	tile dashboardclaim.Tile,
	loc *time.Location,
) []string {
	if tile == dashboardclaim.TileLossAdjuster {
		return []string{
			row.SurveyNumber,
			row.ReferenceNumber,
			row.ClaimNumber,
			row.PolicyNumber,
			row.SurveyorName,
			row.AdjusterPIC,
			row.InsuredName,
			row.TechnicalPIC,
			row.SurveyStatus,
		}
	}

	return []string{
		row.SurveyNumber,
		row.ClaimNumber,
		row.PolicyNumber,
		row.InsuredName,
		formatTanggalUnduhan(row.ScheduledAt, loc),
		row.SurveyLocation,
		row.SurveyStatus,
	}
}

// formatTanggalUnduhan menuliskan tanggal dalam bentuk yang dipakai berkas unduhan Pega.
//
// `dd/mm/yyyy` diambil dari `to_char(…,'dd/mm/yyyy')` pada
// `RDB List/BrowseInboxOutstanding1-SQL.xml` — bentuk yang Pega pakai saat ia memformat
// tanggalnya sama sekali. Bentuk ini SENGAJA BERBEDA dari `formatDate`, yang melayani JSON
// layar dan karena itu memakai ISO `2006-01-02` agar dapat diurutkan di sisi klien.
//
// Berkas unduhan dibaca manusia di Excel, bukan diurutkan mesin; dan menyeragamkan keduanya
// berarti salah satu dari dua pembacanya dilayani dengan bentuk yang bukan untuknya.
func formatTanggalUnduhan(t *time.Time, loc *time.Location) string {
	if t == nil || t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("02/01/2006")
}
