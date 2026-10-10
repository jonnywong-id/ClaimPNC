package masterstatushttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
const (
	KodeTidakDitemukan  = "status_klaim_tidak_ditemukan"
	KodeValidasiGagal   = "validasi_gagal"
	KodeLabelSudahAda   = "label_status_sudah_dipakai"
	KodeKodeSudahAda    = "kode_status_sudah_dipakai"
	KodePermintaanCacat = "permintaan_cacat"
	KodeGalatInternal   = "galat_internal"
)

// PenulisJSON menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri
// supaya seluruh modul menulis respons dengan cara yang sama, termasuk header
// Cache-Control-nya.
type PenulisJSON func(w http.ResponseWriter, r *http.Request, status int, badan any)

// PenulisGalat menuliskan galat dalam bentuk respons HTTP.
type PenulisGalat func(w http.ResponseWriter, r *http.Request, err error)

// TulisGalat memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke penulisCadangan — yang di cmd diisi
// penulis galat auth, sehingga galat sesi yang lolos dari middleware tetap dijawab
// dengan kode yang sudah dikenal frontend. Bila tidak ada cadangan, atau cadangannya
// pun tidak mengenalinya, jawabannya 500 dengan pesan umum dan rinciannya hanya masuk
// log — rincian galat internal tidak pernah dikirim ke peramban.
func TulisGalat(logger *slog.Logger, tulisJSON PenulisJSON, penulisCadangan PenulisGalat) PenulisGalat {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, badan, dikenali := petakanGalat(err)
		if !dikenali {
			if penulisCadangan != nil {
				penulisCadangan(w, r, err)
				return
			}
			status, badan = http.StatusInternalServerError, ResponsGalat{
				Kode:  KodeGalatInternal,
				Pesan: "Terjadi kesalahan pada sistem.",
			}
		}
		if status >= http.StatusInternalServerError {
			logging.Dari(r.Context(), logger).Error("permintaan gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
		}
		tulisJSON(w, r, status, badan)
	}
}

// petakanGalat menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke
// yang lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
func petakanGalat(err error) (int, ResponsGalat, bool) {
	var validasi *masterstatus.GalatValidasi

	switch {
	case errors.As(err, &validasi):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di kolomnya
		// (docs/Steering/10-API-STRATEGY.md §5).
		detail := make([]PelanggaranDTO, 0, len(validasi.Pelanggaran))
		for _, p := range validasi.Pelanggaran {
			detail = append(detail, PelanggaranDTO{Field: p.Field, Pesan: p.Pesan})
		}
		return http.StatusUnprocessableEntity, ResponsGalat{
			Kode:   KodeValidasiGagal,
			Pesan:  "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
			Detail: detail,
		}, true

	case errors.Is(err, masterstatus.ErrLabelSudahAda):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan
		// penyimpanan saat ini — mungkin karena orang lain baru saja memakai nama itu.
		return http.StatusConflict, ResponsGalat{
			Kode:  KodeLabelSudahAda,
			Pesan: "Status dengan nama itu sudah ada. Pakai nama lain.",
		}, true

	case errors.Is(err, masterstatus.ErrKodeSudahAda):
		return http.StatusConflict, ResponsGalat{
			Kode:  KodeKodeSudahAda,
			Pesan: "Kode status yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.",
		}, true

	case errors.Is(err, masterstatus.ErrTidakDitemukan):
		return http.StatusNotFound, ResponsGalat{
			Kode:  KodeTidakDitemukan,
			Pesan: "Status klaim tidak ditemukan.",
		}, true

	default:
		return 0, ResponsGalat{}, false
	}
}
