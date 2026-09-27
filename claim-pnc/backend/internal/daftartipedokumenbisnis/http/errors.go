package daftartipedokumenbisnishttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/daftartipedokumenbisnis"
	"claim-pnc/internal/platform/logging"
)

// Kode galat modul ini.
//
// Kodenya dibaca mesin dan TIDAK diterjemahkan; pesannya dibaca manusia dan berbahasa
// Indonesia. Klien bercabang pada kode, bukan pada pesan — tanpa itu, memperbaiki satu
// kalimat akan merusak percabangan di layar.
// Hanya DUA kode isian, dan itu cerminan layar lamanya.
//
// `nama_bisnis_belum_diisi` meniru satu-satunya validasi yang benar-benar ada di Pega.
// Selebihnya tidak ada: baris dokumen kosong dan jaminan kosong sama-sama DILEWATI
// diam-diam di sana, bukan ditolak (keputusan Work Owner 2026-09-23 — layar ini mengikuti
// Pega apa adanya).
const (
	CodeNotFound         = "tipe_dokumen_bisnis_tidak_ditemukan"
	CodeBusinessRequired = "nama_bisnis_belum_diisi"
	CodeMalformedRequest = "permintaan_cacat"
)

// JSONWriter menulis jawaban berhasil. Disuntikkan cmd supaya bentuk amplopnya seragam di
// seluruh aplikasi tanpa modul saling mengimpor.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menulis galat yang BUKAN milik modul ini — terutama galat portal.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// writeModuleError menulis galat, memetakannya lebih dulu bila ia milik modul ini.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat portal ditangani penulis dari cmd (`portalhttp.WithPortalError`), bukan di
		// sini. Menyalin pemetaannya ke setiap modul berarti satu perubahan aturan portal
		// harus diingat di dua puluh tempat.
		h.writeError(w, r, err)
		return
	}

	if status >= http.StatusInternalServerError {
		logging.From(r.Context(), h.logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()))
	}
	h.writeResponse(w, r, status, body)
}

// mapError memetakan galat domain menjadi status dan badan jawaban.
//
// Galat yang tidak dikenali dikembalikan dengan penanda false, BUKAN dipaksa menjadi 500.
// Dengan begitu galat portal tetap sampai ke penulis yang memang mengetahuinya.
func mapError(err error) (int, ErrorResponse, bool) {
	switch {
	case errors.Is(err, daftartipedokumenbisnis.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code: CodeNotFound,
			Message: "Aturan dokumen yang dimaksud tidak ditemukan. " +
				"Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	case errors.Is(err, daftartipedokumenbisnis.ErrBusinessRequired):
		// Kalimatnya ditiru dari layar lama apa adanya, termasuk ejaan "di isi" yang
		// terpisah — petugas yang hafal layar lama membaca kalimat yang sama persis
		// (`D-13`). Sumbernya
		// `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:209`.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeBusinessRequired,
			Message: "Nama Bisnis belum di isi.",
		}, true

	}
	return 0, ErrorResponse{}, false
}
