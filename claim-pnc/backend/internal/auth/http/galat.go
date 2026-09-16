package authhttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan.
const (
	KodeKredensialSalah    = "kredensial_salah"
	KodePenggunaTidakAktif = "pengguna_tidak_aktif"
	KodeIdentitasPutus     = "sistem_identitas_tidak_terhubung"
	KodeSesiTidakSah       = "sesi_tidak_sah"
	KodeSesiKedaluwarsa    = "sesi_kedaluwarsa"
	KodePermintaanCacat    = "permintaan_cacat"
	KodeGalatInternal      = "galat_internal"
)

// pesanKredensialSalah sengaja sama untuk pengguna yang tidak ada dan kata sandi yang
// salah. Membedakan keduanya memberi tahu siapa saja yang punya akun di sistem ini.
const pesanKredensialSalah = "Nama pengguna atau kata sandi salah."

// PenulisGalat menuliskan galat dalam bentuk respons HTTP.
type PenulisGalat func(w http.ResponseWriter, r *http.Request, err error)

// TulisGalat memetakan galat menjadi respons HTTP.
//
// Galat yang tidak dikenali dijawab 500 dengan pesan umum, dan rinciannya hanya masuk
// log — rincian galat internal tidak pernah dikirim ke peramban.
func TulisGalat(logger *slog.Logger) PenulisGalat {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, badan := petakanGalat(err)
		if status >= http.StatusInternalServerError {
			logging.Dari(r.Context(), logger).Error("permintaan gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
		}
		TulisJSON(w, r, status, badan, logger)
	}
}

func petakanGalat(err error) (int, ResponsGalat) {
	var profilBolong *auth.GalatProfilTidakLengkap

	switch {
	case errors.Is(err, auth.ErrKredensialSalah):
		return http.StatusUnauthorized, ResponsGalat{
			Kode:  KodeKredensialSalah,
			Pesan: pesanKredensialSalah,
		}

	case errors.Is(err, auth.ErrPenggunaTidakAktif):
		return http.StatusForbidden, ResponsGalat{
			Kode:  KodePenggunaTidakAktif,
			Pesan: "Akun Anda tidak aktif. Hubungi administrator Claim PNC.",
		}

	case errors.Is(err, auth.ErrSistemTidakTerhubung):
		// 503, bukan 401: ini bukan kesalahan pengguna, dan mencoba berulang kali
		// justru membanjiri sistem yang sedang bermasalah.
		return http.StatusServiceUnavailable, ResponsGalat{
			Kode:  KodeIdentitasPutus,
			Pesan: "Sistem identitas sedang tidak dapat dihubungi. Coba beberapa saat lagi.",
		}

	case errors.As(err, &profilBolong):
		// Sistem identitas menjawab dengan profil yang tidak lengkap. Meneruskannya
		// berarti pengguna masuk tetapi tidak dikenali data klaimnya sendiri.
		return http.StatusBadGateway, ResponsGalat{
			Kode:  KodeIdentitasPutus,
			Pesan: "Profil pengguna dari sistem identitas tidak lengkap. Hubungi administrator Claim PNC.",
		}

	case errors.Is(err, auth.ErrSesiKedaluwarsa):
		// Dibedakan dari sesi tidak sah supaya frontend dapat menyelamatkan isian yang
		// belum tersimpan, bukan sekadar melempar pengguna ke layar masuk.
		return http.StatusUnauthorized, ResponsGalat{
			Kode:  KodeSesiKedaluwarsa,
			Pesan: "Sesi Anda sudah berakhir. Silakan masuk kembali.",
		}

	case errors.Is(err, auth.ErrSesiTidakDitemukan), errors.Is(err, auth.ErrSesiDicabut):
		return http.StatusUnauthorized, ResponsGalat{
			Kode:  KodeSesiTidakSah,
			Pesan: "Sesi tidak sah. Silakan masuk kembali.",
		}

	default:
		return http.StatusInternalServerError, ResponsGalat{
			Kode:  KodeGalatInternal,
			Pesan: "Terjadi kesalahan pada sistem.",
		}
	}
}

// TulisJSON menuliskan badan respons.
func TulisJSON(w http.ResponseWriter, r *http.Request, status int, badan any, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Respons autentikasi tidak boleh disinggahi cache mana pun di jalur.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if badan == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(badan); err != nil {
		logging.Dari(r.Context(), logger).Error("gagal menulis respons",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
}
