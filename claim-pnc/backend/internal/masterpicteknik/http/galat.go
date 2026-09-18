package masterpicteknikhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
const (
	KodeTidakDitemukan     = "pic_teknik_tidak_ditemukan"
	KodeValidasiGagal      = "validasi_gagal"
	KodeSudahAda           = "id_operator_sudah_terdaftar"
	KodeDirektoriTerputus  = "direktori_operator_tidak_terhubung"
	KodePermintaanCacat    = "permintaan_cacat"
	KodeGalatInternal      = "galat_internal"
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
// dengan kode yang sudah dikenal frontend. Bila tidak ada cadangan, atau cadangannya pun
// tidak mengenalinya, jawabannya 500 dengan pesan umum dan rinciannya hanya masuk log.
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
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini, supaya pemanggil dapat
// membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang lain".
func petakanGalat(err error) (int, ResponsGalat, bool) {
	var validasi *masterpicteknik.GalatValidasi

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

	case errors.Is(err, masterpicteknik.ErrSudahAda):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan
		// penyimpanan saat ini — mungkin karena orang lain baru saja mendaftarkannya.
		return http.StatusConflict, ResponsGalat{
			Kode:  KodeSudahAda,
			Pesan: "ID operator itu sudah terdaftar. Buka datanya lalu ubah di sana.",
		}, true

	case errors.Is(err, masterpicteknik.ErrTidakDitemukan):
		return http.StatusNotFound, ResponsGalat{
			Kode:  KodeTidakDitemukan,
			Pesan: "PIC teknik tidak ditemukan.",
		}, true

	case errors.Is(err, masterpicteknik.ErrDirektoriTidakTerhubung):
		// 503, bukan 500: ini bukan cacat aplikasi melainkan sumber luar yang sedang
		// tidak dapat dihubungi, dan tindak lanjutnya menunggu — bukan melapor.
		return http.StatusServiceUnavailable, ResponsGalat{
			Kode:  KodeDirektoriTerputus,
			Pesan: "Direktori operator sedang tidak dapat dihubungi. Coba beberapa saat lagi.",
		}, true

	default:
		return 0, ResponsGalat{}, false
	}
}
