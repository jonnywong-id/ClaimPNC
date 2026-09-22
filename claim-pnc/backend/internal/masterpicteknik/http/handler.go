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
const maxSaveBodyBytes = 8 << 10

// Layanan adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context) ([]masterpicteknik.PICTeknik, error)
	Get(ctx context.Context, operatorID string) (masterpicteknik.PICTeknik, error)
	Create(ctx context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error)
	Update(ctx context.Context, operatorID string, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error)
}

// Handler melayani permintaan Master PIC Teknik.
type Handler struct {
	service     Service
	logger      *slog.Logger
	writeResponse JSONWriter
	writeError  ErrorWriter
}

// Opsi adalah bahan pembentuk Handler.
type Options struct {
	Service Service
	Logger  *slog.Logger

	// TulisRespon dan TulisGalatCadangan dipasok dari luar supaya seluruh modul
	// menuliskan respons dan galat sesi dengan cara yang sama.
	WriteResponse        JSONWriter
	FallbackErrorWriter ErrorWriter
}

// HandlerBaru membentuk handler modul Master PIC Teknik.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:     o.Service,
		logger:      o.Logger,
		writeResponse: o.WriteResponse,
		writeError:  WriteError(o.Logger, o.WriteResponse, o.FallbackErrorWriter),
	}
}

// Daftar menangani GET /api/master/pic-teknik.
//
// Menggantikan Report Definition `BrowseVMstUserTeknis_RD` yang mengisi grid layar
// `UserTeknisInbox`.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.List(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	content := make([]PICTeknikDTO, 0, len(list))
	for _, p := range list {
		content = append(content, toDTO(p))
	}
	h.writeResponse(w, r, http.StatusOK, ListResponse{PICTeknik: content, Total: len(content)})
}

// Ambil menangani GET /api/master/pic-teknik/{id}.
//
// Menggantikan `SetMstUserTeknisValue_act`, yang menjalankan `GetMasterPICTeknis` lalu
// menyalin hasilnya ke halaman `TempDcol` untuk diisi ke form.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p, err := h.service.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{PICTeknik: toDTO(p)})
}

// Tambah menangani POST /api/master/pic-teknik.
//
// Menggantikan `CNMInsertMstUserTeknis_act`, termasuk penolakan bila ID operatornya
// tidak ditemukan di direktori operator.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	request, readable := h.readRequest(w, r)
	if !readable {
		return
	}

	p, err := h.service.Create(r.Context(), fromRequest(request.OperatorID, request))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// 201, bukan 200: sumber daya baru terbentuk.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{PICTeknik: toDTO(p)})
}

// Ubah menangani PUT /api/master/pic-teknik/{id}.
//
// ID diambil dari jalur URL, tidak pernah dari badan permintaan.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	request, readable := h.readRequest(w, r)
	if !readable {
		return
	}

	id := chi.URLParam(r, "id")
	p, err := h.service.Update(r.Context(), id, fromRequest(id, request))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{PICTeknik: toDTO(p)})
}

// bacaPermintaan membaca badan permintaan simpan. Nilai kedua false berarti jawabannya
// sudah ditulis dan pemanggil harus berhenti.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxSaveBodyBytes)

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		// Isi badan permintaan tidak ikut dikembalikan: memantulkan masukan mentah ke
		// peramban adalah kebiasaan yang tidak layak dimulai di satu tempat pun.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:  CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest{}, false
	}
	return request, true
}

// dariPermintaan menyusun bentuk domain dari isian form.
//
// Nama dan GrupPanel sengaja dibiarkan kosong: keduanya tidak pernah datang dari klien.
// Nama diisi usecase dari direktori operator; GrupPanel dipertahankan dari baris yang
// sudah ada.
func fromRequest(operatorID string, p SaveRequest) masterpicteknik.PICTeknik {
	return masterpicteknik.PICTeknik{
		OperatorID: operatorID,
		Email:      p.Email,
		BusinessLine: p.BusinessLine,
		Group:       p.Group,
		Supervisor:     p.Supervisor,
		Quota:      p.Quota,
		ExternalQuota:  p.ExternalQuota,
		Active:      p.Active,
	}
}

func toDTO(p masterpicteknik.PICTeknik) PICTeknikDTO {
	return PICTeknikDTO{
		OperatorID: p.OperatorID,
		Name:       p.Name,
		Email:      p.Email,
		BusinessLine: p.BusinessLine,
		Group:       p.Group,
		Supervisor:     p.Supervisor,
		Quota:      p.Quota,
		ExternalQuota:  p.ExternalQuota,
		GrupPanel:  p.GrupPanel,
		Active:      p.Active,
	}
}
