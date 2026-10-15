package inboxmanagerhttp

import (
	"encoding/csv"
	"net/http"
	"strings"

	"claim-pnc/internal/inboxmanager"
)

// ExportDetail menangani GET /api/inbox-manager/ekspor-detail — blok "Export Data Detail
// Klaim" pada tab Outstanding.
//
// Parameter: `dari` dan `sampai` (YYYY-MM-DD), keduanya kosong untuk seluruh periode.
//
// # Kapan header berkas ditulis
//
// Header HTTP dan baris judul CSV ditulis TEPAT SEBELUM baris pertama — atau di akhir bila
// tidak ada satu baris pun. Penolakan (tanggal tidak sah, tab tidak boleh dilihat) dan
// kegagalan kueri karena itu tetap tiba sebagai JSON yang dapat dibaca layar, bukan sebagai
// berkas CSV berisi pesan galat.
//
// Setelah baris pertama terkirim galat tidak dapat lagi dijawab sebagai JSON; yang dilakukan
// hanyalah mencatatnya dan berhenti — sama dengan ekspor antrean.
func (h *Handler) ExportDetail(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()
	input := inboxmanager.DetailExportInput{
		From:  strings.TrimSpace(query.Get("dari")),
		Until: strings.TrimSpace(query.Get("sampai")),
	}

	writer := csv.NewWriter(w)
	started := false
	start := func() error {
		if started {
			return nil
		}
		started = true

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition",
			`attachment; filename="`+inboxmanager.DetailExportFilename+`"`)
		// Berkas ini memuat nama tertanggung dan nilai klaim; ia tidak boleh mengendap di
		// cache perantara.
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)

		header := make([]string, 0, len(inboxmanager.DetailExportColumns))
		for _, column := range inboxmanager.DetailExportColumns {
			header = append(header, column.Title)
		}
		return writer.Write(header)
	}

	written := 0
	err := h.service.DetailExport(r.Context(), active.Alias, input, caller,
		func(row inboxmanager.DetailExportRow) error {
			if err := start(); err != nil {
				return err
			}

			record := make([]string, 0, len(inboxmanager.DetailExportColumns))
			for _, column := range inboxmanager.DetailExportColumns {
				record = append(record, row[column.Property])
			}
			if err := writer.Write(record); err != nil {
				return err
			}

			// Dialirkan per potongan supaya peramban mulai menerima sementara kueri yang
			// lebih dari satu menit masih berjalan.
			written++
			if written%500 == 0 {
				writer.Flush()
				if flusher, able := w.(http.Flusher); able {
					flusher.Flush()
				}
			}
			return writer.Error()
		})

	if err != nil {
		if !started {
			h.writeError(w, r, err)
			return
		}
		h.logExportFailure(r, err)
		writer.Flush()
		return
	}

	if err := start(); err != nil {
		h.logExportFailure(r, err)
		return
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		h.logExportFailure(r, err)
	}
}
