package inboxsurveyhttp

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"

	"claim-pnc/internal/inboxsurvey"
	"claim-pnc/internal/inboxsurvey/usecase"
)

// ExportKPI menangani GET /api/inbox-survey/kpi/ekspor — tombol **Export Data** pada panel
// KPI layar lama.
//
// # Padanannya di Pega
//
// `Activity/ExportKPILoginAdjuster-Act.xml`, yang menjalankan kueri yang SAMA dengan tombol
// Cari lalu melewatkan hasilnya ke `pxConvertResultsToCSV`. Yang diekspor karena itu adalah
// isi tabel yang sedang dilihat — bukan laporan lain.
//
// # Kenapa dibuat di Go, bukan di peramban
//
// `D-11` menetapkan pembuatan PDF, Excel, dan CSV dilakukan aplikasi, bukan engine lain.
// Menyusunnya di peramban juga berarti hanya baris yang sudah terunduh yang ikut, dan itu
// membuat berkasnya diam-diam berbeda dari yang diminta.
//
// # Yang TIDAK dilakukan di sini
//
// Tidak ada penomoran halaman: ringkasan KPI dikelompokkan `GROUP BY` — satu baris per
// adjuster atau per tahun — sehingga jumlah barisnya sebanyak orang atau sebanyak tahun,
// bukan sebanyak klaim. Memotongnya berhalaman akan menambah kerumitan untuk himpunan yang
// memang kecil.
//
// Isian wajib tetap diperiksa lebih dulu. Mengekspor "apa adanya" tanpa Status Survey akan
// menghasilkan berkas yang isinya tidak dapat dijelaskan siapa pun yang membukanya nanti.
func (h *Handler) ExportKPI(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.begin(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()

	scored, err := h.service.KPI(
		r.Context(),
		active.Alias,
		caller,
		inboxsurvey.KPIFilter{
			Status:  inboxsurvey.SurveyStatus(query.Get("status_survei")),
			Report:  inboxsurvey.ReportType(query.Get("tipe_report")),
			Quarter: query.Get("kuartal"),
			Year:    query.Get("tahun"),
		},
	)
	if err != nil {
		// Galat ditulis SEBELUM satu byte pun badan terkirim. Sesudah header terkirim,
		// kegagalan hanya dapat memutus berkas di tengah — dan berkas yang terpotong
		// terbuka tanpa satu pun tanda bahwa ia tidak lengkap.
		h.writeError(w, r, err)
		return
	}

	nama := fmt.Sprintf("kpi-adjuster-%s.csv", active.Alias)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nama+`"`)

	tulis := csv.NewWriter(w)
	defer tulis.Flush()

	// Judul kolomnya diambil dari sumber yang sama dengan layar, bukan ditulis ulang di
	// sini. Dua daftar judul yang terpisah akan berbeda pada perubahan berikutnya, dan
	// berkas ekspor yang judulnya berbeda dari layar tidak dapat dicocokkan siapa pun.
	header := []string{"KELOMPOK"}
	for _, c := range usecase.KPIColumns() {
		header = append(header, c.Title)
	}
	if err := tulis.Write(header); err != nil {
		return
	}

	for _, row := range scored.Rows {
		record := []string{row.Group}
		for _, nilai := range []float64{
			row.SurveyScheduling, row.ImmediateAdvice, row.PreliminaryAdvice,
			row.InterimReport, row.ProgressUpdate, row.CommunicationResponse,
			row.ProposeAdjustment, row.FinalReport, row.Value,
		} {
			record = append(record, strconv.FormatFloat(nilai, 'f', 2, 64))
		}
		if err := tulis.Write(record); err != nil {
			return
		}
	}
}
