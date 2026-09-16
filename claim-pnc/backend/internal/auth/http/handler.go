package authhttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/usecase"
)

// batasBadanMasuk membatasi ukuran badan permintaan masuk. Layar masuk hanya mengirim
// dua field pendek; apa pun yang lebih besar dari ini bukan permintaan yang wajar.
const batasBadanMasuk = 4 << 10

// Layanan adalah bagian usecase yang dipakai handler ini.
type Layanan interface {
	Masuk(ctx context.Context, k auth.Kredensial) (usecase.Hasil, error)
	Perpanjang(ctx context.Context, token auth.Token) (auth.Sesi, error)
	Keluar(ctx context.Context, token auth.Token) error
}

// Handler memuat handler masuk, keluar, identitas pemanggil, dan perpanjangan sesi.
type Handler struct {
	layanan    Layanan
	logger     *slog.Logger
	tulisGalat PenulisGalat
}

// HandlerBaru membentuk handler modul auth.
func HandlerBaru(layanan Layanan, logger *slog.Logger) *Handler {
	return &Handler{layanan: layanan, logger: logger, tulisGalat: TulisGalat(logger)}
}

// Masuk menangani POST /api/masuk.
func (h *Handler) Masuk(w http.ResponseWriter, r *http.Request) {
	var permintaan PermintaanMasuk
	r.Body = http.MaxBytesReader(w, r.Body, batasBadanMasuk)
	if err := json.NewDecoder(r.Body).Decode(&permintaan); err != nil {
		// Isi badan permintaan tidak ikut dicatat maupun dikembalikan: di situlah kata
		// sandi berada.
		TulisJSON(w, r, http.StatusBadRequest, ResponsGalat{
			Kode:  KodePermintaanCacat,
			Pesan: "Permintaan tidak dapat dibaca.",
		}, h.logger)
		return
	}

	hasil, err := h.layanan.Masuk(r.Context(), auth.Kredensial{
		NamaPengguna: permintaan.NamaPengguna,
		KataSandi:    permintaan.KataSandi,
	})
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}

	TulisJSON(w, r, http.StatusOK, ResponsMasuk{
		Token:         string(hasil.Token),
		TipeToken:     "Bearer",
		BerlakuSampai: hasil.Sesi.BerlakuSampai,
		Pengguna:      kePenggunaDTO(hasil.Pengguna),
	}, h.logger)
}

// Saya menangani GET /api/saya — dipakai peramban untuk memulihkan keadaan setelah
// muat ulang halaman tanpa harus menyimpan profil pengguna sendiri.
func (h *Handler) Saya(w http.ResponseWriter, r *http.Request) {
	konteks, ada := KonteksPengguna(r.Context())
	if !ada {
		h.tulisGalat(w, r, auth.ErrSesiTidakDitemukan)
		return
	}
	TulisJSON(w, r, http.StatusOK, ResponsSaya{
		Pengguna:      kePenggunaDTO(konteks.Pengguna),
		BerlakuSampai: konteks.Sesi.BerlakuSampai,
	}, h.logger)
}

// Perpanjang menangani POST /api/sesi/perpanjang.
func (h *Handler) Perpanjang(w http.ResponseWriter, r *http.Request) {
	token, ada := TokenDariPermintaan(r)
	if !ada {
		h.tulisGalat(w, r, auth.ErrSesiTidakDitemukan)
		return
	}
	diperpanjang, err := h.layanan.Perpanjang(r.Context(), token)
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	TulisJSON(w, r, http.StatusOK, ResponsPerpanjang{
		BerlakuSampai: diperpanjang.BerlakuSampai,
	}, h.logger)
}

// Keluar menangani POST /api/keluar.
//
// Ia mencabut sesi di server, bukan sekadar meminta peramban melupakan tokennya —
// token lama harus ditolak sejak permintaan berikutnya.
func (h *Handler) Keluar(w http.ResponseWriter, r *http.Request) {
	token, ada := TokenDariPermintaan(r)
	if !ada {
		// Keluar tanpa token bukan kegagalan yang perlu diperlihatkan: hasil akhirnya
		// sama-sama tidak ada sesi.
		TulisJSON(w, r, http.StatusNoContent, nil, h.logger)
		return
	}
	if err := h.layanan.Keluar(r.Context(), token); err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	TulisJSON(w, r, http.StatusNoContent, nil, h.logger)
}

func kePenggunaDTO(p auth.Pengguna) PenggunaDTO {
	return PenggunaDTO{
		Identitas:  p.Identitas,
		Nama:       p.Nama,
		Jenis:      string(p.Jenis),
		Login:      p.Login,
		Email:      p.Email,
		Perusahaan: p.Perusahaan,
	}
}
