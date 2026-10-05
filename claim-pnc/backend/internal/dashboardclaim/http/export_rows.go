package dashboardclaimhttp

import (
	"strconv"
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// Judul kolom unduhan mengikuti judul kolom LAYAR, bukan nama kolom basis data.
//
// Yang membuka berkasnya adalah orang yang sama yang membaca layarnya, dan ia mencari judul
// yang dikenalnya (`D-13`). Nama kolom Oracle justru menyulitkan: `USERTEKNIS_1` tidak
// berarti apa-apa bagi pembacanya.

// claimExportHeader adalah judul kolom unduhan kedua tile bertipe klaim.
//
// Tile Close Claim tidak menggambar Report Date dan Claim status di layar, tetapi KEDUANYA
// tetap ada di berkas unduhan — dengan sel kosong. Berkas CSV dengan jumlah kolom yang
// berbeda-beda menurut tile akan menyulitkan siapa pun yang menggabungkan keduanya, dan sel
// kosong sudah menyatakan bahwa datanya memang tidak ada.
func claimExportHeader() []string {
	return []string{
		"No Klaim",
		"No Polis",
		"Nama Tertanggung",
		"Nama Bisnis",
		"Sumber Bisnis",
		"Nama Cabang",
		"Tanggal Pendaftaran",
		"Report Date",
		"Lama Waktu Klaim (hari)",
		"PIC Teknik",
		"Admin PNC",
		"Claim status",
	}
}

// claimExportRow menuliskan satu baris klaim.
func claimExportRow(
	row dashboardclaim.ClaimRow,
	tile dashboardclaim.Tile,
	now time.Time,
	loc *time.Location,
) []string {
	// Kedua kolom ini dikosongkan pada tile Close Claim karena datanya memang tidak dibaca
	// kueri tile itu — bukan karena kebetulan kosong.
	reportDate := ""
	status := ""
	if tile != dashboardclaim.TileCloseClaim {
		reportDate = formatDate(row.ReportDate, loc)
		status = row.ClaimStatusLabel
	}

	return []string{
		row.ClaimNumber,
		row.PolicyNumber,
		row.InsuredName,
		row.BusinessName,
		row.BusinessSource,
		row.BranchName,
		formatDate(&row.RegisteredAt, loc),
		reportDate,
		strconv.Itoa(row.AgeInDays(now, loc)),
		row.TechnicalPIC,
		row.AdminPNC,
		status,
	}
}

// surveyExportHeader adalah judul kolom unduhan kedua tile survei.
//
// Judulnya berbeda menurut tile, sama seperti di layar: grid Loss Adjuster berjudul bahasa
// Inggris, grid Internal Surveyor berjudul bahasa Indonesia. Keduanya ditiru apa adanya.
func surveyExportHeader(tile dashboardclaim.Tile) []string {
	if tile == dashboardclaim.TileLossAdjuster {
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
			"Aging (hari)",
		}
	}
	return []string{
		"Nomor Case",
		"No Klaim",
		"No Polis",
		"Nama Tertanggung",
		"Aging (hari)",
		"Tanggal Survey",
		"Lokasi Survey",
		"Status Survey",
		"PIC ASM",
		"Surveyor",
	}
}

// surveyExportRow menuliskan satu baris survei, dalam urutan kolom tile-nya sendiri.
func surveyExportRow(
	row dashboardclaim.SurveyRow,
	tile dashboardclaim.Tile,
	now time.Time,
	loc *time.Location,
) []string {
	umur := strconv.Itoa(row.AgeInDays(now, loc))

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
			umur,
		}
	}
	return []string{
		row.SurveyNumber,
		row.ClaimNumber,
		row.PolicyNumber,
		row.InsuredName,
		umur,
		formatDate(row.ScheduledAt, loc),
		row.SurveyLocation,
		row.SurveyStatus,
		row.TechnicalPIC,
		row.SurveyorName,
	}
}

// holdingExportHeader adalah judul kolom unduhan tab Inbox Tampungan PIC.
func holdingExportHeader() []string {
	return []string{
		"No Klaim",
		"No Polis",
		"Nama Tertanggung",
		"Nama Bisnis",
		"Sumber Bisnis",
		"Nama Cabang",
		"Tanggal Pendaftaran",
		"Admin PNC",
	}
}

// holdingExportRow menuliskan satu baris penampungan.
func holdingExportRow(row dashboardclaim.HoldingRow, loc *time.Location) []string {
	return []string{
		row.ClaimNumber,
		row.PolicyNumber,
		row.InsuredName,
		row.BusinessName,
		row.BusinessSource,
		row.BranchName,
		formatDate(&row.RegisteredAt, loc),
		row.AdminPNC,
	}
}
