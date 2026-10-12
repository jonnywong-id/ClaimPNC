package dashboardclaimhttp

import (
	"encoding/csv"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/usecase"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// exportBatchSize adalah banyaknya baris yang dibaca sekali jalan saat mengunduh.
//
// Ia JAUH di atas batas halaman layar (100) dengan sengaja: unduhan tidak dipaginasi untuk
// dibaca manusia, ia mengalir sampai habis. Tetapi ia tetap TERBATAS — membaca seluruh
// tabel sekaligus akan menahan satu koneksi sambil menumpuk puluhan ribu baris di memori
// (`D-10`: puluhan juta baris).
const exportBatchSize = 500

// exportMaxRows adalah pagar atas jumlah baris yang diunduh.
//
// # Kenapa ada pagar sama sekali
//
// Tanpa pagar, satu permintaan unduh pada penyaring yang longgar dapat berjalan berjam-jam
// sambil memegang koneksi dari pool yang hanya 20 per instance. Pagar membuat kegagalannya
// TERLIHAT dan cepat, bukan menggantung.
//
// # Kenapa angkanya bukan 500
//
// Sistem lama memasang `pyMaxRecords=500` pada 54 dari 56 laporannya, sehingga unduhan di
// sana DIAM-DIAM terpotong di 500 baris. Angka itu tidak ditiru: ia bukan kebutuhan bisnis
// melainkan batas bawaan yang tidak pernah diputuskan siapa pun.
//
// Berapa baris yang WAJIB dilayani satu unduhan belum ditetapkan secara terukur, dan tidak
// dapat ditetapkan dari data historis karena angkanya selama ini selalu terpotong
// (`ADR-0011`). 50.000 dipilih sebagai angka kerja yang jauh di atas pemakaian nyata dan
// masih aman untuk satu permintaan — dan saat tercapai, layar MENGATAKANNYA lewat baris
// terakhir, bukan diam seperti sistem lama.
const exportMaxRows = 50_000

// ExportTile menjawab GET /dashboard-claim/{tile}/unduh.
//
// Menggantikan tombol "Export to Excel" pada keempat grid layar lama. Bentuknya CSV, bukan
// XLSX: `D-11` menetapkan pembuatan berkas dikerjakan sendiri, dan CSV terbaca Excel tanpa
// pustaka tambahan sekaligus dapat dialirkan baris demi baris tanpa menahan memori.
func (h *Handler) ExportTile(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	tile, known := dashboardclaim.ParseTile(chi.URLParam(r, "tile"))
	if !known {
		h.writeError(w, r, dashboardclaim.ErrTileNotFound)
		return
	}

	filter, err := readFilter(r.URL.Query())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Halaman pertama dibaca SEBELUM header respons ditulis.
	//
	// Setelah badan respons mulai mengalir, status HTTP tidak dapat diubah lagi — galat yang
	// terjadi sesudahnya tidak dapat dijawab 500. Membaca sekali di muka memastikan kegagalan
	// yang paling mungkin (portal belum siap, kueri salah) tetap terjawab sebagai galat.
	filter.Offset = 0
	filter.Limit = exportBatchSize

	first, err := h.service.List(r.Context(), usecase.ListQuery{
		PortalAlias: active.Alias, Tile: tile, Filter: filter,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.beginCSV(w, "dashboard-"+string(tile))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	survey := first.Shape == dashboardclaim.ShapeSurvey
	header := claimExportHeader()
	if survey {
		header = surveyExportHeader(tile)
	}
	if err := writer.Write(header); err != nil {
		h.logStreamFailure(r, err)
		return
	}

	now := h.now()
	written := 0
	page := first

	for {
		if survey {
			for _, row := range page.Surveys.Rows {
				if h.stopAtCap(writer, &written) {
					return
				}
				if err := writer.Write(surveyExportRow(row, tile, now, h.location)); err != nil {
					h.logStreamFailure(r, err)
					return
				}
			}
			if len(page.Surveys.Rows) < filter.Limit {
				return
			}
		} else {
			for _, row := range page.Claims.Rows {
				if h.stopAtCap(writer, &written) {
					return
				}
				if err := writer.Write(claimExportRow(row, tile, now, h.location)); err != nil {
					h.logStreamFailure(r, err)
					return
				}
			}
			if len(page.Claims.Rows) < filter.Limit {
				return
			}
		}

		// Halaman berikutnya. Galat di tengah aliran TIDAK dapat dijawab sebagai status HTTP;
		// yang dapat dilakukan adalah berhenti dan mencatatnya. Berkas yang diterima pengguna
		// karena itu dapat kurang — dan itu sebabnya baris penutup ditulis di bawah.
		filter.Offset += filter.Limit
		writer.Flush()

		page, err = h.service.List(r.Context(), usecase.ListQuery{
			PortalAlias: active.Alias, Tile: tile, Filter: filter,
		})
		if err != nil {
			h.logStreamFailure(r, err)
			_ = writer.Write([]string{"-- UNDUHAN TERPOTONG: sisa baris tidak dapat dibaca --"})
			return
		}
	}
}

// TIDAK ADA ExportHolding.
//
// Tab Inbox Tampungan PIC tidak punya tombol unduh di layar lama, dan rute
// GET /dashboard-claim/tampungan/unduh beserta handler ini DICABUT 2026-10-07.
//
// Dibuktikan dua cara sebelum dicabut: Section/DashboardClaim_Section2-Section.xml memuat
// NOL aksi openUrlInWindow — cara keempat ekspor tile dipanggil, yang muncul 4x di
// DashboardClaim_Section1 — dan tidak ada satu pun activity Export* di export yang
// menyentuh klaim ber-USERTEKNIS_1 IS NULL.

// beginCSV menuliskan header respons unduhan.
func (h *Handler) beginCSV(w http.ResponseWriter, prefix string) {
	filename := prefix + "-" + h.now().In(h.location).Format("20060102-150405") + ".csv"

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
}

// stopAtCap menuliskan baris penutup bila pagar tercapai, dan menyatakan unduhan berhenti.
//
// Pagar yang tercapai DIUMUMKAN di dalam berkasnya sendiri. Sistem lama memotong di 500
// baris tanpa satu pun tanda, dan pengguna tidak punya cara mengetahui bahwa yang
// diterimanya kurang.
func (h *Handler) stopAtCap(writer *csv.Writer, written *int) bool {
	if *written >= exportMaxRows {
		_ = writer.Write([]string{
			"-- UNDUHAN DIHENTIKAN pada " + strconv.Itoa(exportMaxRows) +
				" baris. Persempit penyaring untuk mengunduh sisanya. --",
		})
		return true
	}
	*written++
	return false
}

// logStreamFailure mencatat kegagalan yang terjadi setelah badan respons mulai mengalir.
//
// Ia TIDAK dapat dijawab sebagai status HTTP — header sudah terkirim. Yang tersisa adalah
// mencatatnya, dan kegagalan yang tidak dicatat tidak terlihat oleh siapa pun.
func (h *Handler) logStreamFailure(r *http.Request, err error) {
	logging.From(r.Context(), h.logger).Error("unduhan dashboard terhenti di tengah",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
