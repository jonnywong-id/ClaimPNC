package masterstatushttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterstatus"
)

// batasBadanSimpan membatasi ukuran badan permintaan simpan.
//
// Form ini hanya mengirim satu field pendek; apa pun yang lebih besar dari ini bukan
// permintaan yang wajar, dan menolaknya lebih awal menjaga memori tidak dihabiskan
// badan permintaan yang dikarang.
const batasBadanSimpan = 4 << 10

// Layanan adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Layanan interface {
	Daftar(ctx context.Context) ([]masterstatus.StatusKlaim, error)
	Ambil(ctx context.Context, kode string) (masterstatus.StatusKlaim, error)
	Tambah(ctx context.Context, label string) (masterstatus.StatusKlaim, error)
	Ubah(ctx context.Context, kode, label string) (masterstatus.StatusKlaim, error)
}

// Handler melayani permintaan Master Status Klaim.
type Handler struct {
	layanan     Layanan
	logger      *slog.Logger
	tulisRespon PenulisJSON
	tulisGalat  PenulisGalat
}

// Opsi adalah bahan pembentuk Handler.
type Opsi struct {
	Layanan Layanan
	Logger  *slog.Logger

	// TulisRespon dan TulisGalatCadangan dipasok dari luar supaya seluruh modul
	// menuliskan respons dan galat sesi dengan cara yang sama.
	TulisRespon        PenulisJSON
	TulisGalatCadangan PenulisGalat
}

// HandlerBaru membentuk handler modul Master Status Klaim.
func HandlerBaru(o Opsi) *Handler {
	return &Handler{
		layanan:     o.Layanan,
		logger:      o.Logger,
		tulisRespon: o.TulisRespon,
		tulisGalat:  TulisGalat(o.Logger, o.TulisRespon, o.TulisGalatCadangan),
	}
}

// Daftar menangani GET /api/master/status-klaim.
//
// Menggantikan Report Definition BrowseVStsClaim_RD yang mengisi grid layar
// StatusClaimInbox.
func (h *Handler) Daftar(w http.ResponseWriter, r *http.Request) {
	daftar, err := h.layanan.Daftar(r.Context())
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}

	isi := make([]StatusKlaimDTO, 0, len(daftar))
	for _, s := range daftar {
		isi = append(isi, keDTO(s))
	}
	h.tulisRespon(w, r, http.StatusOK, ResponsDaftar{StatusKlaim: isi, Total: len(isi)})
}

// Ambil menangani GET /api/master/status-klaim/{kode}.
//
// Menggantikan SetStsClaimValue_act(lscid), yang menjalankan SelectVStsClaim_RD lalu
// menyalin hasilnya ke halaman TempStsClaim untuk diisi ke form.
func (h *Handler) Ambil(w http.ResponseWriter, r *http.Request) {
	status, err := h.layanan.Ambil(r.Context(), chi.URLParam(r, "kode"))
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	h.tulisRespon(w, r, http.StatusOK, ResponsSatu{StatusKlaim: keDTO(status)})
}

// Tambah menangani POST /api/master/status-klaim.
//
// Menggantikan tombol Tambah pada harness, yang mengirim sentinel "UnknownID" supaya
// procedure membentuk kodenya. Di sini kodenya tidak pernah ikut di badan permintaan
// sama sekali — sentinel itu tidak dibawa.
func (h *Handler) Tambah(w http.ResponseWriter, r *http.Request) {
	permintaan, terbaca := h.bacaPermintaan(w, r)
	if !terbaca {
		return
	}

	status, err := h.layanan.Tambah(r.Context(), permintaan.Label)
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	// 201, bukan 200: sumber daya baru terbentuk dan kodenya baru diketahui di sini.
	h.tulisRespon(w, r, http.StatusCreated, ResponsSatu{StatusKlaim: keDTO(status)})
}

// Ubah menangani PUT /api/master/status-klaim/{kode}.
//
// Menggantikan tombol Simpan pada baris yang sedang diubah. Kode diambil dari jalur URL,
// tidak pernah dari badan permintaan.
func (h *Handler) Ubah(w http.ResponseWriter, r *http.Request) {
	permintaan, terbaca := h.bacaPermintaan(w, r)
	if !terbaca {
		return
	}

	status, err := h.layanan.Ubah(r.Context(), chi.URLParam(r, "kode"), permintaan.Label)
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	h.tulisRespon(w, r, http.StatusOK, ResponsSatu{StatusKlaim: keDTO(status)})
}

// bacaPermintaan membaca badan permintaan simpan. Nilai kedua false berarti jawabannya
// sudah ditulis dan pemanggil harus berhenti.
func (h *Handler) bacaPermintaan(w http.ResponseWriter, r *http.Request) (PermintaanSimpan, bool) {
	var permintaan PermintaanSimpan
	r.Body = http.MaxBytesReader(w, r.Body, batasBadanSimpan)

	if err := json.NewDecoder(r.Body).Decode(&permintaan); err != nil {
		// Isi badan permintaan tidak ikut dikembalikan. Di modul ini ia tidak memuat
		// rahasia, tetapi memantulkan masukan mentah ke peramban adalah kebiasaan yang
		// tidak layak dimulai di satu tempat pun.
		h.tulisRespon(w, r, http.StatusBadRequest, ResponsGalat{
			Kode:  KodePermintaanCacat,
			Pesan: "Permintaan tidak dapat dibaca.",
		})
		return PermintaanSimpan{}, false
	}
	return permintaan, true
}

func keDTO(s masterstatus.StatusKlaim) StatusKlaimDTO {
	return StatusKlaimDTO{Kode: s.Kode, Label: s.Label, KodeLama: s.KodeLama}
}
