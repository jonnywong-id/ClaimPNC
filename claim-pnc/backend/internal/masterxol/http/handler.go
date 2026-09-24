package masterxolhttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/masterxol/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxSaveBodyBytes membatasi ukuran badan permintaan simpan.
//
// Nilainya jauh lebih besar daripada modul master lain karena badan di sini memuat
// SELURUH pohon: induk, grup bisnis, lapisan, dan reas tiap lapisan. Induk terbesar di
// produksi — 4 lapisan dengan total 12 baris reas — berukuran sekitar 3 KB sebagai JSON,
// sehingga 256 KB memberi ruang puluhan kali lipat sambil tetap menolak badan yang
// dikarang untuk menghabiskan memori.
const maxSaveBodyBytes = 256 << 10

// Service adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, portalAlias string) ([]masterxol.Master, error)
	Get(ctx context.Context, portalAlias, id string) (masterxol.Master, error)
	Save(ctx context.Context, portalAlias string, master masterxol.Master, caller string) (usecase.SaveResult, error)
	DeleteMaster(ctx context.Context, portalAlias, id string) error
	DeleteBusiness(ctx context.Context, portalAlias, masterID, businessID string) error
	DeleteLayer(ctx context.Context, portalAlias, masterID, layerID string) error
	DeleteReinsurer(ctx context.Context, portalAlias, layerID, reinsurerID string) error
	Form(ctx context.Context, portalAlias string) (usecase.FormOption, error)
	BusinessGroup(ctx context.Context, portalAlias string, t masterxol.Type) ([]masterxol.Business, error)
}

// Caller adalah identitas pemanggil yang sedang bekerja.
//
// Ia dinyatakan di sini sebagai tipe sempit, bukan diimpor dari modul auth, supaya kedua
// modul tetap tidak saling mengimpor. Yang menjembatani keduanya hanyalah berkas
// perakitan di cmd/claimpnc.
type Caller struct {
	Identity string
	Name     string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan Master XOL.
type Handler struct {
	service       Service
	caller        CallerReader
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service Service

	// Caller mengisi kolom PIC saat induk diajukan ke komite. WAJIB: `D-59` menetapkan
	// tidak ada pemisahan tugas formal, sehingga jejak siapa-mengajukan-apa adalah satu-
	// satunya kontrol pengimbang yang tersisa.
	Caller CallerReader

	Logger *slog.Logger

	// WriteResponse dan FallbackErrorWriter dipasok dari luar supaya seluruh modul
	// menuliskan respons dan galat portal dengan cara yang sama.
	WriteResponse       JSONWriter
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Master XOL.
func NewHandler(o Options) (*Handler, error) {
	switch {
	case o.Service == nil:
		return nil, errors.New("masterxol/http: Service wajib diisi")
	case o.Caller == nil:
		return nil, errors.New("masterxol/http: Caller wajib diisi")
	case o.WriteResponse == nil:
		return nil, errors.New("masterxol/http: WriteResponse wajib diisi")
	}

	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    WriteError(o.Logger, o.WriteResponse, o.FallbackErrorWriter),
	}, nil
}

// List menangani GET /api/master/xol.
//
// Menggantikan `RDB List/GetDataMasterXOL-SQL.xml` yang mengisi grid harness
// `DetailMasterXOL`.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, ok := h.activePortal(w, r)
	if !ok {
		return
	}

	list, err := h.service.List(r.Context(), active)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	content := make([]MasterDTO, 0, len(list))
	for _, m := range list {
		content = append(content, toDTO(m))
	}
	h.writeResponse(w, r, http.StatusOK, ListResponse{
		XOL:    content,
		Total:  len(content),
		Portal: active,
	})
}

// Get menangani GET /api/master/xol/{id}.
//
// Menggantikan `Activity/UpdateMasterXOL-Act.xml`, yang memuat induk beserta bisnis,
// lapisan, dan reas-nya sebelum form Ubah dibuka.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, ok := h.activePortal(w, r)
	if !ok {
		return
	}

	master, err := h.service.Get(r.Context(), active, chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{XOL: toDTO(master), Portal: active})
}

// Form menangani GET /api/master/xol/form.
//
// Bekal awal layar: pilihan Tahun dan pilihan Type XOL, dalam satu permintaan.
func (h *Handler) Form(w http.ResponseWriter, r *http.Request) {
	active, ok := h.activePortal(w, r)
	if !ok {
		return
	}

	option, err := h.service.Form(r.Context(), active)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	tipe := make([]TypeOptionDTO, 0, len(option.Type))
	for _, t := range option.Type {
		tipe = append(tipe, TypeOptionDTO{Kode: string(t.Code), Label: t.Label})
	}
	h.writeResponse(w, r, http.StatusOK, FormResponse{
		Tahun:  option.Year,
		Tipe:   tipe,
		Portal: active,
	})
}

// BusinessGroup menangani GET /api/master/xol/bisnis?tipe={kode}.
//
// Menggantikan `Activity/ShowDetailGroupBisnisXol_Act-Act.xml`, yang dijalankan ulang
// setiap kali dropdown Type XOL berubah.
func (h *Handler) BusinessGroup(w http.ResponseWriter, r *http.Request) {
	active, ok := h.activePortal(w, r)
	if !ok {
		return
	}

	// Kode tipe diterima apa adanya lalu diserahkan ke domain, yang memetakannya ke pola
	// pencarian. Kode yang tidak dikenal TIDAK ditolak — ia jatuh ke cabang "tanpa
	// penyaring", sama seperti induk produksi yang TYPEXOL-nya kosong.
	list, err := h.service.BusinessGroup(r.Context(), active, masterxol.Type(r.URL.Query().Get("tipe")))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	content := make([]BusinessDTO, 0, len(list))
	for _, b := range list {
		content = append(content, BusinessDTO{ID: b.ID, Nama: b.Name})
	}
	h.writeResponse(w, r, http.StatusOK, BusinessGroupResponse{
		Bisnis: content,
		Total:  len(content),
		Portal: active,
	})
}

// Create menangani POST /api/master/xol.
//
// Menggantikan tombol **Tambah** pada harness, yang mengosongkan `TempXOL` lewat
// `PageNewXOLMaster` lalu menyerahkan penomorannya ke procedure. Di sini penandanya tidak
// dibawa sama sekali — metode HTTP yang membedakannya.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, "")
}

// Update menangani PUT /api/master/xol/{id}.
//
// ID diambil dari jalur URL, tidak pernah dari badan permintaan.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, chi.URLParam(r, "id"))
}

// save melayani penambahan maupun pengubahan.
//
// Keduanya disatukan karena form-nya memang satu, dan karena penyimpanannya memang satu
// langkah di sistem lama: `InsertUpdateMasterXOL` menangani keduanya, dan yang
// membedakannya hanyalah `TempXOL.BranchID` kosong atau tidak.
func (h *Handler) save(w http.ResponseWriter, r *http.Request, id string) {
	active, ok := h.activePortal(w, r)
	if !ok {
		return
	}

	caller, exists := h.caller(r.Context())
	if !exists {
		// Tidak dapat terjadi di balik middleware sesi, tetapi diperiksa: menyimpan tanpa
		// identitas berarti mengajukan ke komite tanpa jejak siapa yang mengajukan.
		h.writeResponse(w, r, http.StatusUnauthorized, ErrorResponse{
			Code:    ErrCodeBadRequest,
			Message: "Identitas pemanggil tidak terbaca.",
		})
		return
	}

	request, read := h.readRequest(w, r)
	if !read {
		return
	}

	result, err := h.service.Save(r.Context(), active, toMaster(id, request), caller.Identity)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	status := http.StatusOK
	if id == "" {
		// 201, bukan 200: sumber daya baru terbentuk dan nomornya baru diketahui di sini.
		status = http.StatusCreated
	}
	h.writeResponse(w, r, status, SingleResponse{
		XOL:        toDTO(result.Master),
		Portal:     active,
		Peringatan: result.Warning,
	})
}

// DeleteMaster menangani DELETE /api/master/xol/{id}.
func (h *Handler) DeleteMaster(w http.ResponseWriter, r *http.Request) {
	active, ok := h.activePortal(w, r)
	if !ok {
		return
	}

	if err := h.service.DeleteMaster(r.Context(), active, chi.URLParam(r, "id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteBusiness menangani DELETE /api/master/xol/{id}/bisnis/{idBisnis}.
func (h *Handler) DeleteBusiness(w http.ResponseWriter, r *http.Request) {
	active, ok := h.activePortal(w, r)
	if !ok {
		return
	}

	err := h.service.DeleteBusiness(r.Context(), active,
		chi.URLParam(r, "id"), chi.URLParam(r, "idBisnis"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteLayer menangani DELETE /api/master/xol/{id}/layer/{idLayer}.
func (h *Handler) DeleteLayer(w http.ResponseWriter, r *http.Request) {
	active, ok := h.activePortal(w, r)
	if !ok {
		return
	}

	err := h.service.DeleteLayer(r.Context(), active,
		chi.URLParam(r, "id"), chi.URLParam(r, "idLayer"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteReinsurer menangani DELETE /api/master/xol/{id}/layer/{idLayer}/reas/{idReas}.
func (h *Handler) DeleteReinsurer(w http.ResponseWriter, r *http.Request) {
	active, ok := h.activePortal(w, r)
	if !ok {
		return
	}

	err := h.service.DeleteReinsurer(r.Context(), active,
		chi.URLParam(r, "idLayer"), chi.URLParam(r, "idReas"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// activePortal membaca portal aktif. Nilai kedua false berarti jawabannya sudah ditulis
// dan pemanggil harus berhenti.
func (h *Handler) activePortal(w http.ResponseWriter, r *http.Request) (string, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return "", false
	}
	return active.Alias, true
}

// readRequest membaca badan permintaan simpan. Nilai kedua false berarti jawabannya sudah
// ditulis dan pemanggil harus berhenti.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxSaveBodyBytes)

	decoder := json.NewDecoder(r.Body)
	// Field yang tidak dikenal ditolak, bukan diabaikan. Badan permintaan di modul ini
	// memuat kolom komite pada respons tetapi TIDAK pada permintaan — menolaknya membuat
	// klien yang keliru mengirimkannya tahu seketika, alih-alih mengira nilainya
	// tersimpan.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		// Isi badan permintaan tidak ikut dikembalikan. Memantulkan masukan mentah ke
		// peramban adalah kebiasaan yang tidak layak dimulai di satu tempat pun.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    ErrCodeBadRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest{}, false
	}
	return request, true
}

// toMaster mengubah badan permintaan menjadi tipe domain.
//
// LimitIDR pada badan permintaan SENGAJA DIABAIKAN — server yang menghitungnya dari Limit
// dan kurs induk. Begitu pula kolom komite: keempatnya tidak ada di SaveRequest sama
// sekali.
func toMaster(id string, request SaveRequest) masterxol.Master {
	business := make([]masterxol.Business, 0, len(request.Bisnis))
	for _, b := range request.Bisnis {
		business = append(business, masterxol.Business{ID: b.ID, Name: b.Nama})
	}

	layer := make([]masterxol.Layer, 0, len(request.Layer))
	for _, l := range request.Layer {
		reinsurer := make([]masterxol.Reinsurer, 0, len(l.Reas))
		for _, reas := range l.Reas {
			reinsurer = append(reinsurer, masterxol.Reinsurer{
				ID:    reas.ID,
				Name:  reas.Nama,
				Share: masterxol.Share(reas.Share),
			})
		}
		layer = append(layer, masterxol.Layer{
			ID:        l.ID,
			Name:      l.Nama,
			Limit:     masterxol.Amount(l.Limit),
			Excess:    masterxol.Amount(l.Excess),
			Reinsurer: reinsurer,
		})
	}

	return masterxol.Master{
		ID:           id,
		Name:         request.Nama,
		Year:         request.Tahun,
		ExchangeRate: masterxol.Amount(request.Kurs),
		Type:         masterxol.Type(request.Tipe),
		RemarkPIC:    request.RemarkPIC,
		Business:     business,
		Layer:        layer,
	}
}

func toDTO(m masterxol.Master) MasterDTO {
	business := make([]BusinessDTO, 0, len(m.Business))
	for _, b := range m.Business {
		business = append(business, BusinessDTO{ID: b.ID, Nama: b.Name})
	}

	layer := make([]LayerDTO, 0, len(m.Layer))
	for _, l := range m.Layer {
		reas := make([]ReinsurerDTO, 0, len(l.Reinsurer))
		for _, r := range l.Reinsurer {
			reas = append(reas, ReinsurerDTO{ID: r.ID, Nama: r.Name, Share: int64(r.Share)})
		}
		layer = append(layer, LayerDTO{
			ID:       l.ID,
			Nama:     l.Name,
			Limit:    int64(l.Limit),
			Excess:   int64(l.Excess),
			LimitIDR: int64(l.ConvertedLimit),
			Reas:     reas,
		})
	}

	return MasterDTO{
		ID:           m.ID,
		Nama:         m.Name,
		Tahun:        m.Year,
		Kurs:         int64(m.ExchangeRate),
		Tipe:         string(m.Type),
		TipeLabel:    masterxol.TypeLabel(m.Type),
		RemarkPIC:    m.RemarkPIC,
		PIC:          m.PIC,
		StatusKomite: string(m.CommitteeStatus),
		Komite:       m.Committee,
		RemarkKomite: m.RemarkCommittee,
		Bisnis:       business,
		Layer:        layer,
	}
}
