package registrasi

import (
	"time"

	"claim-pnc/internal/platform/clock"
)

// PAReportDescription adalah Deskripsi Laporan klaim PA yang baru dibuka —
// `Activity/CallActivityInputRegister-Act.xml` langkah 44 "Auto Deskripsi Laporan PA" (When
// `isPA_PNC`: Group Panel 002):
//
//	"Berdasarkan surat keterangan kematian dari ... menyatakan bahwa benar telah meninggal dunia
//	 tertanggung atas nama ... pada tanggal " + DD/MM/YYYY tanggal kejadian +
//	 " yang diakibatkan oleh karena ..." + .ClaimData.Kronologis
//
// Kalimat disalin apa adanya, termasuk tiga titik yang diisi petugas dan tidak adanya spasi
// sebelum Kronologis. Tanggal kejadian yang belum diketahui menghasilkan "//", sama dengan
// `@substring` Pega atas teks kosong.
func PAReportDescription(dateOfLoss time.Time, chronology string) string {
	date := "//"
	if !dateOfLoss.IsZero() {
		date = dateOfLoss.In(clock.ZoneWIB).Format("02/01/2006")
	}
	return "Berdasarkan surat keterangan kematian dari ... menyatakan bahwa benar telah meninggal dunia " +
		"tertanggung atas nama ... pada tanggal " + date + " yang diakibatkan oleh karena ..." + chronology
}
