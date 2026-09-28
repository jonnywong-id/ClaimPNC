package inboxmanagerhttp

import (
	"encoding/csv"
	"log/slog"
	"net/http"
	"strings"

	"claim-pnc/internal/inboxmanager"
)

// exportLimit adalah batas atas jumlah baris yang ditulis ke berkas ekspor.
//
// Ia jauh di atas kebutuhan nyata — antrean terpanjang berisi 11 baris saat diperiksa
// 2026-09-28 — dan ada sebagai pagar, bukan sebagai penyaring. Berkas yang terpotong
// dinyatakan lewat satu baris penutup, bukan dibiarkan berhenti begitu saja.
const exportLimit = 50000

// Export menangani GET /api/inbox-manager/ekspor.
//
// # Kenapa GET, bukan POST
//
// Karena ia tidak mengubah apa pun. Menjadikannya GET membuat unduhannya dapat dipicu tautan
// biasa, termasuk dibuka ulang dari riwayat peramban dengan penyaring yang sama.
//
// # Kenapa hanya tab ANTREAN yang dapat diekspor
//
// Karena hanya antrean yang berbentuk daftar baris. Grid dashboard adalah ringkasan berdimensi
// yang kolomnya berbeda antar panel; mengekspornya sebagai satu berkas CSV menuntut memilih
// panel mana yang menang, dan itu keputusan yang tidak ada padanannya di layar lama.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	input := readQuery(r)

	// Seluruh baris diambil SEBELUM satu byte pun ditulis. Setelah header terkirim, galat
	// tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa berkas
	// separuh jadi tanpa satu pun keterangan.
	input.Page = inboxmanager.Pagination{Page: 1, Size: inboxmanager.MaxPageSize}

	view, err := h.service.List(r.Context(), active.Alias, input, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if view.Tab.Kind != inboxmanager.KindQueue {
		h.writeError(w, r, inboxmanager.NewValidationError([]inboxmanager.Violation{{
			Field:   inboxmanager.FieldTab,
			Message: "Hanya antrean persetujuan yang dapat diekspor.",
		}}))
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+exportFilename(view.Tab)+`"`)

	// Berkas ini memuat nomor klaim dan nama pengaju; ia tidak boleh mengendap di cache
	// perantara.
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := make([]string, 0, len(view.Tab.Columns))
	for _, column := range view.Tab.Columns {
		header = append(header, column.Title)
	}
	if err := writer.Write(header); err != nil {
		h.logExportFailure(r, err)
		return
	}

	written := 0
	page := 1
	for {
		for _, row := range view.Queue.Rows {
			if written >= exportLimit {
				break
			}

			record := make([]string, 0, len(view.Tab.Columns))
			for _, column := range view.Tab.Columns {
				record = append(record, row.Cells[column.Key])
			}
			if err := writer.Write(record); err != nil {
				h.logExportFailure(r, err)
				return
			}
			written++
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			h.logExportFailure(r, err)
			return
		}
		if flusher, able := w.(http.Flusher); able {
			flusher.Flush()
		}

		if written >= exportLimit || page >= view.Queue.TotalPages() {
			break
		}

		page++
		input.Page = inboxmanager.Pagination{Page: page, Size: inboxmanager.MaxPageSize}

		view, err = h.service.List(r.Context(), active.Alias, input, caller)
		if err != nil {
			// Galat di tengah berkas tidak dapat lagi dijawab sebagai JSON. Yang dapat
			// dilakukan hanyalah mencatatnya dan berhenti — dan berkas yang berhenti di
			// tengah lebih jujur daripada berkas yang diam-diam kehilangan sisanya.
			h.logExportFailure(r, err)
			return
		}
	}

	if written >= exportLimit {
		_ = writer.Write([]string{
			"-- Berkas dipotong pada " + itoa(exportLimit) + " baris. " +
				"Persempit antrean lalu ekspor ulang. --",
		})
	}
}

// exportFilename menyusun nama berkas yang menyebut antreannya.
func exportFilename(tab inboxmanager.Tab) string {
	name := strings.ToLower(tab.Name)
	name = strings.NewReplacer(" ", "-", "/", "-", "\\", "-", "\"", "").Replace(name)
	return "inbox-manager-" + name + ".csv"
}

// logExportFailure mencatat kegagalan yang terjadi setelah header terkirim.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	if h.logger == nil {
		return
	}
	h.logger.Error("ekspor inbox manager gagal di tengah berkas",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}

// itoa mengubah angka menjadi teks tanpa mengimpor strconv ke berkas ini.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
