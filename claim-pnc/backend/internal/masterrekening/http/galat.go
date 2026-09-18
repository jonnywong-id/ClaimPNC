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
	KodeTidakDitemukan  = "rekening_tidak_ditemukan"
	KodeSudahAda        = "nomor_rekening_sudah_ada"
	KodeSudahDiputuskan = "keputusan_sudah_diambil"
	KodeIsianTidakSah   = "isian_tidak_sah"
	KodePermintaanCacat = "permintaan_cacat"
	KodeGalatInternal   = "galat_internal"
)

// TulisGalat memetakan galat menjadi respons HTTP.
//
// Galat yang tidak dikenali dijawab 500 dengan pesan umum, dan rinciannya hanya masuk
// log — rincian galat internal tidak pernah dikirim ke peramban.
func TulisGalat(logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, badan := petakanGalat(err)
		if status >= http.StatusInternalServerError {
			logging.Dari(r.Context(), logger).Error("permintaan master rekening gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
		}
		TulisJSON(w, r, status, badan, logger)
	}
}

func petakanGalat(err error) (int, ResponsGalat) {
	var validasi *masterrekening.GalatValidasi

	switch {
	case errors.As(err, &validasi):
		// 422, bukan 400: permintaannya terbaca dengan benar, isinya yang belum
		// memenuhi aturan bisnis. Membedakan keduanya membuat layar tahu kapan harus
		// menandai kolom dan kapan harus melaporkan cacat pemrograman.
		return http.StatusUnprocessableEntity, ResponsGalat{
			Kode:   KodeIsianTidakSah,
			Pesan:  "Ada isian yang belum benar. Periksa kolom yang ditandai.",
			Detail: pelanggaranDari(validasi),
		}

	case errors.Is(err, masterrekening.ErrTidakDitemukan):
		return http.StatusNotFound, ResponsGalat{
			Kode:  KodeTidakDitemukan,
			Pesan: "Rekening tidak ditemukan.",
		}

	case errors.Is(err, masterrekening.ErrSudahAda):
		return http.StatusConflict, ResponsGalat{
			Kode: KodeSudahAda,
			Pesan: "Nomor rekening ini sudah terdaftar dan belum ditolak komite. " +
				"Gunakan data yang sudah ada, atau tunggu keputusan komite.",
		}

	case errors.Is(err, masterrekening.ErrSudahDiputuskan):
		return http.StatusConflict, ResponsGalat{
			Kode: KodeSudahDiputuskan,
			Pesan: "Keputusan komite atas rekening ini sudah pernah diambil. " +
				"Ajukan rekening baru bila datanya perlu diubah.",
		}

	case errors.Is(err, masterrekening.ErrStatusTidakDikenal):
		return http.StatusBadRequest, ResponsGalat{
			Kode:  KodePermintaanCacat,
			Pesan: "Keputusan harus berupa menyetujui atau menolak.",
		}

	default:
		return http.StatusInternalServerError, ResponsGalat{
			Kode:  KodeGalatInternal,
			Pesan: "Terjadi kesalahan pada sistem.",
		}
	}
}

// pelanggaranDari mengubah galat validasi domain menjadi daftar untuk klien.
//
// Diurutkan menurut nama field supaya jawaban atas permintaan yang sama selalu identik.
// Tanpa itu, urutannya mengikuti iterasi map Go — yang sengaja acak — sehingga uji
// kontrak menjadi rapuh dan log sulit dibandingkan.
func pelanggaranDari(g *masterrekening.GalatValidasi) []PelanggaranDTO {
	nama := make([]string, 0, len(g.Field))
	for f := range g.Field {
		nama = append(nama, f)
	}
	sort.Strings(nama)

	hasil := make([]PelanggaranDTO, 0, len(nama))
	for _, f := range nama {
		hasil = append(hasil, PelanggaranDTO{Field: f, Pesan: g.Field[f]})
	}
	return hasil
}

// TulisJSON menuliskan badan respons.
func TulisJSON(w http.ResponseWriter, r *http.Request, status int, badan any, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Master rekening memuat data pihak ketiga dan tidak boleh disinggahi cache mana
	// pun di jalur.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if badan == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(badan); err != nil {
		logging.Dari(r.Context(), logger).Error("gagal menulis respons master rekening",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
}
