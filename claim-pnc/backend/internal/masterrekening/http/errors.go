package masterrekeninghttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
const (
	CodeNotFound         = "rekening_tidak_ditemukan"
	CodeAlreadyExists    = "nomor_rekening_sudah_ada"
	CodeAlreadyDecided   = "keputusan_sudah_diambil"
	CodeInvalidInput     = "isian_tidak_sah"
	CodeMalformedRequest = "permintaan_cacat"
	CodeInternalError    = "galat_internal"
)

// WriteError memetakan galat menjadi respons HTTP.
//
// Galat yang tidak dikenali dijawab 500 dengan pesan umum, dan rinciannya hanya masuk
// log — rincian galat internal tidak pernah dikirim ke peramban.
func WriteError(logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body := mapError(err)
		if status >= http.StatusInternalServerError {
			logging.From(r.Context(), logger).Error("permintaan master rekening gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
		}
		WriteJSON(w, r, status, body, logger)
	}
}

func mapError(err error) (int, ErrorResponse) {
	var validasi *masterrekening.ValidationError

	switch {
	case errors.As(err, &validasi):
		// 422, bukan 400: permintaannya terbaca dengan benar, isinya yang belum
		// memenuhi aturan bisnis. Membedakan keduanya membuat layar tahu kapan harus
		// menandai kolom dan kapan harus melaporkan cacat pemrograman.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeInvalidInput,
			Message: "Ada isian yang belum benar. Periksa kolom yang ditandai.",
			Detail:  violationsFrom(validasi),
		}

	case errors.Is(err, masterrekening.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Rekening tidak ditemukan.",
		}

	case errors.Is(err, masterrekening.ErrAlreadyExists):
		return http.StatusConflict, ErrorResponse{
			Code: CodeAlreadyExists,
			Message: "Nomor rekening ini sudah terdaftar dan belum ditolak komite. " +
				"Gunakan data yang sudah ada, atau tunggu keputusan komite.",
		}

	case errors.Is(err, masterrekening.ErrAlreadyDecided):
		return http.StatusConflict, ErrorResponse{
			Code: CodeAlreadyDecided,
			Message: "Decision komite atas rekening ini sudah pernah diambil. " +
				"Submit rekening baru bila datanya perlu diubah.",
		}

	case errors.Is(err, masterrekening.ErrUnknownStatus):
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Decision harus berupa menyetujui atau menolak.",
		}

	default:
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeInternalError,
			Message: "Terjadi kesalahan pada sistem.",
		}
	}
}

// violationsFrom mengubah galat validasi domain menjadi daftar untuk klien.
//
// Diurutkan menurut nama field supaya jawaban atas permintaan yang sama selalu identik.
// Tanpa itu, urutannya mengikuti iterasi map Go — yang sengaja acak — sehingga uji
// kontrak menjadi rapuh dan log sulit dibandingkan.
func violationsFrom(e *masterrekening.ValidationError) []PelanggaranDTO {
	names := make([]string, 0, len(e.Field))
	for f := range e.Field {
		names = append(names, f)
	}
	sort.Strings(names)

	result := make([]PelanggaranDTO, 0, len(names))
	for _, f := range names {
		result = append(result, PelanggaranDTO{Field: f, Pesan: e.Field[f]})
	}
	return result
}

// WriteJSON menuliskan badan respons.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, body any, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Master rekening memuat data pihak ketiga dan tidak boleh disinggahi cache mana
	// pun di jalur.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logging.From(r.Context(), logger).Error("gagal menulis respons master rekening",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
}
