package masterpicteknikhttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpicteknik"
)

// batasBadanSimpan membatasi ukuran badan permintaan simpan.
//
// Form ini hanya mengirim beberapa field pendek; apa pun yang lebih besar bukan
// permintaan yang wajar, dan menolaknya lebih awal menjaga memori tidak dihabiskan badan
// permintaan yang dikarang.
const batasBadanSimpan = 8 << 10

// Layanan adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Layanan interface {
	Daftar(ctx context.Context) ([]masterpicteknik.PICTeknik, error)
	Ambil(ctx context.Context, idOperator string) (masterpicteknik.PICTeknik, error)
	Tambah(ctx context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error)
	Ubah(ctx context.Context, idOperator string, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error)
}

// Handler melayani permintaan Master PIC Teknik.
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

// HandlerBaru membentuk handler modul Master PIC Teknik.
func HandlerBaru(o Opsi) *Handler {
	return &Handler{
		layanan:     o.Layanan,
		logger:      o.Logger,
		tulisRespon: o.TulisRespon,
		tulisGalat:  TulisGalat(o.Logger, o.TulisRespon, o.TulisGalatCadangan),
	}
}

// Daftar menangani GET /api/master/pic-teknik.
//
// Menggantikan Report Definition `BrowseVMstUserTeknis_RD` yang mengisi grid layar
// `UserTeknisInbox`.
func (h *Handler) Daftar(w http.ResponseWriter, r *http.Request) {
	daftar, err := h.layanan.Daftar(r.Context())
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}

	isi := make([]PICTeknikDTO, 0, len(daftar))
	for _, p := range daftar {
		isi = append(isi, keDTO(p))
	}
	h.tulisRespon(w, r, http.StatusOK, ResponsDaftar{PICTeknik: isi, Total: len(isi)})
}

// Ambil menangani GET /api/master/pic-teknik/{id}.
//
// Menggantikan `SetMstUserTeknisValue_act`, yang menjalankan `GetMasterPICTeknis` lalu
// menyalin hasilnya ke halaman `TempDcol` untuk diisi ke form.
func (h *Handler) Ambil(w http.ResponseWriter, r *http.Request) {
	p, err := h.layanan.Ambil(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	h.tulisRespon(w, r, http.StatusOK, ResponsSatu{PICTeknik: keDTO(p)})
}

// Tambah menangani POST /api/master/pic-teknik.
//
// Menggantikan `CNMInsertMstUserTeknis_act`, termasuk penolakan bila ID operatornya
// tidak ditemukan di direktori operator.
func (h *Handler) Tambah(w http.ResponseWriter, r *http.Request) {
	permintaan, terbaca := h.bacaPermintaan(w, r)
	if !terbaca {
		return
	}

	p, err := h.layanan.Tambah(r.Context(), dariPermintaan(permintaan.IDOperator, permintaan))
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	// 201, bukan 200: sumber daya baru terbentuk.
	h.tulisRespon(w, r, http.StatusCreated, ResponsSatu{PICTeknik: keDTO(p)})
}

// Ubah menangani PUT /api/master/pic-teknik/{id}.
//
// ID diambil dari jalur URL, tidak pernah dari badan permintaan.
func (h *Handler) Ubah(w http.ResponseWriter, r *http.Request) {
	permintaan, terbaca := h.bacaPermintaan(w, r)
	if !terbaca {
		return
	}

	id := chi.URLParam(r, "id")
	p, err := h.layanan.Ubah(r.Context(), id, dariPermintaan(id, permintaan))
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}
	h.tulisRespon(w, r, http.StatusOK, ResponsSatu{PICTeknik: keDTO(p)})
}

// bacaPermintaan membaca badan permintaan simpan. Nilai kedua false berarti jawabannya
// sudah ditulis dan pemanggil harus berhenti.
func (h *Handler) bacaPermintaan(w http.ResponseWriter, r *http.Request) (PermintaanSimpan, bool) {
	var permintaan PermintaanSimpan
	r.Body = http.MaxBytesReader(w, r.Body, batasBadanSimpan)

	if err := json.NewDecoder(r.Body).Decode(&permintaan); err != nil {
		// Isi badan permintaan tidak ikut dikembalikan: memantulkan masukan mentah ke
		// peramban adalah kebiasaan yang tidak layak dimulai di satu tempat pun.
		h.tulisRespon(w, r, http.StatusBadRequest, ResponsGalat{
			Kode:  KodePermintaanCacat,
			Pesan: "Permintaan tidak dapat dibaca.",
		})
		return PermintaanSimpan{}, false
	}
	return permintaan, true
}

// dariPermintaan menyusun bentuk domain dari isian form.
//
// Nama dan GrupPanel sengaja dibiarkan kosong: keduanya tidak pernah datang dari klien.
// Nama diisi usecase dari direktori operator; GrupPanel dipertahankan dari baris yang
// sudah ada.
func dariPermintaan(idOperator string, p PermintaanSimpan) masterpicteknik.PICTeknik {
	return masterpicteknik.PICTeknik{
		IDOperator: idOperator,
		Email:      p.Email,
		LiniBisnis: p.LiniBisnis,
		Grup:       p.Grup,
		Atasan:     p.Atasan,
		Kuota:      p.Kuota,
		KuotaLuar:  p.KuotaLuar,
		Aktif:      p.Aktif,
	}
}

func keDTO(p masterpicteknik.PICTeknik) PICTeknikDTO {
	return PICTeknikDTO{
		IDOperator: p.IDOperator,
		Nama:       p.Nama,
		Email:      p.Email,
		LiniBisnis: p.LiniBisnis,
		Grup:       p.Grup,
		Atasan:     p.Atasan,
		Kuota:      p.Kuota,
		KuotaLuar:  p.KuotaLuar,
		GrupPanel:  p.GrupPanel,
		Aktif:      p.Aktif,
	}
}
