package statusprogreshttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/portal"
	"claim-pnc/internal/statusprogres"
	"claim-pnc/internal/statusprogres/usecase"

	portalhttp "claim-pnc/internal/portal/http"
)

// batasBadanPermintaan membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat dua isian pendek; 64 KiB sudah jauh lebih dari cukup.
// Batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum memakan memori,
// bukan setelah.
const batasBadanPermintaan = 64 << 10

// Handler melayani permintaan master status progres.
type Handler struct {
	layanan     *usecase.Layanan
	logger      *slog.Logger
	tulisRespon PenulisRespon
	tulisGalat  PenulisGalat
}

// Opsi adalah bahan pembentuk Handler.
type Opsi struct {
	Layanan *usecase.Layanan
	Logger  *slog.Logger

	// TulisRespon dan TulisGalat disuntikkan dari cmd, bukan diimpor dari modul auth.
	// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul
	// dapat dipindahkan tanpa menariknya serta.
	TulisRespon PenulisRespon
	TulisGalat  PenulisGalat
}

// HandlerBaru membentuk handler modul master status progres.
func HandlerBaru(o Opsi) (*Handler, error) {
	if o.Layanan == nil {
		return nil, errors.New("statusprogres/http: Layanan wajib diisi")
	}
	if o.TulisRespon == nil || o.TulisGalat == nil {
		return nil, errors.New("statusprogres/http: TulisRespon dan TulisGalat wajib diisi")
	}
	return &Handler{
		layanan:     o.Layanan,
		logger:      o.Logger,
		tulisRespon: o.TulisRespon,
		tulisGalat:  o.TulisGalat,
	}, nil
}

// Daftar menangani GET /master/status-progres-1.
func (h *Handler) Daftar(w http.ResponseWriter, r *http.Request) {
	aktif, ada := portalhttp.PortalAktifDari(r.Context())
	if !ada {
		h.tulisGalatModul(w, r, portal.ErrTidakDisebut)
		return
	}

	daftar, err := h.layanan.Daftar(r.Context(), aktif.Alias)
	if err != nil {
		h.tulisGalatModul(w, r, err)
		return
	}

	h.tulisRespon(w, r, http.StatusOK, ResponsDaftar{
		StatusProgres: keDaftarDTO(daftar),
		Portal:        aktif.Alias,
	})
}

// Tambah menangani POST /master/status-progres-1.
func (h *Handler) Tambah(w http.ResponseWriter, r *http.Request) {
	aktif, ada := portalhttp.PortalAktifDari(r.Context())
	if !ada {
		h.tulisGalatModul(w, r, portal.ErrTidakDisebut)
		return
	}

	permintaan, terbaca := h.bacaPermintaan(w, r)
	if !terbaca {
		return
	}

	tersimpan, err := h.layanan.Tambah(r.Context(), aktif.Alias, statusprogres.Isian{
		Nama:       permintaan.Nama,
		KodePosisi: permintaan.KodePosisi,
	})
	if err != nil {
		h.tulisGalatModul(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan beserta ID-nya. ID
	// diterbitkan server, sehingga layar tidak punya cara lain mengetahuinya.
	h.tulisRespon(w, r, http.StatusCreated, ResponsSatu{
		StatusProgres: keDTO(tersimpan),
		Portal:        aktif.Alias,
	})
}

// Ubah menangani PUT /master/status-progres-1/{id}.
func (h *Handler) Ubah(w http.ResponseWriter, r *http.Request) {
	aktif, ada := portalhttp.PortalAktifDari(r.Context())
	if !ada {
		h.tulisGalatModul(w, r, portal.ErrTidakDisebut)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.tulisGalatModul(w, r, statusprogres.ErrTidakDitemukan)
		return
	}

	permintaan, terbaca := h.bacaPermintaan(w, r)
	if !terbaca {
		return
	}

	tersimpan, err := h.layanan.Ubah(r.Context(), aktif.Alias, id, statusprogres.Isian{
		Nama:       permintaan.Nama,
		KodePosisi: permintaan.KodePosisi,
	})
	if err != nil {
		h.tulisGalatModul(w, r, err)
		return
	}

	h.tulisRespon(w, r, http.StatusOK, ResponsSatu{
		StatusProgres: keDTO(tersimpan),
		Portal:        aktif.Alias,
	})
}

// Posisi menangani GET /master/posisi-klaim.
//
// Rutenya TIDAK dipasangi PortalAktif: keempat posisi klaim adalah daftar milik
// aplikasi, bukan isi basis data entitas mana pun (lihat statusprogres/posisi.go).
// Menuntut portal di sini akan membuat dropdown gagal justru saat pengguna belum memilih
// portal — padahal tidak ada satu baris data entitas pun yang dibacanya.
func (h *Handler) Posisi(w http.ResponseWriter, r *http.Request) {
	daftar := h.layanan.Posisi()

	isi := make([]PosisiDTO, 0, len(daftar))
	for _, p := range daftar {
		isi = append(isi, PosisiDTO{Kode: p.Kode, Nama: p.Nama})
	}
	h.tulisRespon(w, r, http.StatusOK, ResponsPosisi{Posisi: isi})
}

// bacaPermintaan membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) bacaPermintaan(w http.ResponseWriter, r *http.Request) (PermintaanSimpan, bool) {
	var permintaan PermintaanSimpan

	pembaca := http.MaxBytesReader(w, r.Body, batasBadanPermintaan)
	penyandi := json.NewDecoder(pembaca)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama
	// field akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa
	// satu pun tanda bahwa ada yang salah.
	penyandi.DisallowUnknownFields()

	if err := penyandi.Decode(&permintaan); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan
		// badan permintaan.
		h.tulisRespon(w, r, http.StatusBadRequest, ResponsGalat{
			Kode:  KodePermintaanCacat,
			Pesan: "Permintaan tidak dapat dibaca.",
		})
		return PermintaanSimpan{}, false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := penyandi.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.tulisRespon(w, r, http.StatusBadRequest, ResponsGalat{
			Kode:  KodePermintaanCacat,
			Pesan: "Permintaan tidak dapat dibaca.",
		})
		return PermintaanSimpan{}, false
	}

	return permintaan, true
}

// Pasang mendaftarkan rute modul master status progres.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// Middleware PortalAktif dipasang di sini, hanya pada rute yang menyentuh basis data
// entitas. Rute daftar posisi sengaja berada di luarnya (lihat Handler.Posisi).
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Pasang(r chi.Router, h *Handler, bahanPortal portalhttp.BahanPortalAktif) {
	r.Get("/master/posisi-klaim", h.Posisi)

	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.PortalAktif(bahanPortal))

		perPortal.Get("/master/status-progres-1", h.Daftar)
		perPortal.Post("/master/status-progres-1", h.Tambah)
		perPortal.Put("/master/status-progres-1/{id}", h.Ubah)
	})
}
