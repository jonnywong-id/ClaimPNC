package mastermaskinghttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastermasking"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxSaveBodyBytes membatasi ukuran badan permintaan simpan.
//
// Form ini mengirim sembilan isian pendek; apa pun yang lebih besar dari ini bukan
// permintaan yang wajar, dan menolaknya lebih awal menjaga memori tidak dihabiskan badan
// permintaan yang dikarang.
const maxSaveBodyBytes = 8 << 10

// Service adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, portalAlias string, filter mastermasking.Filter) ([]mastermasking.Masking, error)
	Get(ctx context.Context, portalAlias, id string) (mastermasking.Masking, error)
	Create(ctx context.Context, portalAlias string, m mastermasking.Masking, by string) (mastermasking.Masking, error)
	Update(ctx context.Context, portalAlias, id string, m mastermasking.Masking, by string) (mastermasking.Masking, error)
	SetActive(ctx context.Context, portalAlias, id string, active bool, by string) (mastermasking.Masking, error)
	Branches(ctx context.Context, portalAlias, keyword string) ([]mastermasking.Branch, error)
}

// Caller adalah identitas pemanggil yang sedang bekerja.
//
// Ia dinyatakan di sini sebagai tipe sempit, bukan diimpor dari modul auth, supaya kedua
// modul tetap tidak saling mengimpor. Yang menjembatani keduanya hanyalah berkas perakitan
// di cmd/claimpnc.
type Caller struct {
	Identity string
	Name     string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan Master Masking.
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

	// Caller dipakai mengisi kolom USERINPUT.
	//
	// WAJIB, dan pada modul ini alasannya lebih keras daripada sekadar kerapian: yang
	// dicatat adalah siapa yang memberi atau mencabut kewenangan membuka data pribadi
	// nasabah. `D-59` menetapkan tidak ada pemisahan tugas formal, sehingga jejak inilah
	// satu-satunya kontrol pengimbang yang tersisa.
	Caller CallerReader

	Logger *slog.Logger

	// WriteResponse dan WriteError dipasok dari luar supaya seluruh modul menuliskan
	// respons dan galat dengan cara yang sama. WriteError yang disuntikkan cmd sudah
	// dibungkus portalhttp.WithPortalError, sehingga galat portal terpetakan seragam.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Master Masking.
//
// Ia menolak bahan yang tidak lengkap saat perakitan di cmd, bukan saat permintaan pertama
// datang: rakitan yang setengah jadi harus gagal saat start.
func NewHandler(o Options) (*Handler, error) {
	switch {
	case o.Service == nil:
		return nil, errors.New("mastermasking/http: Service wajib diisi")
	case o.Caller == nil:
		return nil, errors.New("mastermasking/http: Caller wajib diisi")
	case o.WriteResponse == nil || o.WriteError == nil:
		return nil, errors.New("mastermasking/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /api/master/masking.
//
// Menggantikan `RDB List/SearchMasking_SQL-SQL.xml` yang mengisi grid layar
// `MasterProteksiVisibilityData`, beserta `Activity/SearchDataMasking-Act.xml` yang
// menyusun penyaringnya.
//
// Penyaringnya datang dari query string, bukan dari badan permintaan: ia permintaan BACA,
// dan menaruhnya di URL membuat hasil pencarian dapat ditandai dan dibagikan.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	filter := mastermasking.Filter{
		By:      mastermasking.SearchBy(r.URL.Query().Get("cari_di")),
		Keyword: r.URL.Query().Get("kata_kunci"),
	}

	if !filter.By.Known() {
		// Tipe pencarian yang tidak dikenal ditolak sebagai permintaan cacat, bukan
		// diam-diam diabaikan. Mengabaikannya akan menampilkan SELURUH baris kepada
		// pengguna yang mengira ia sedang mencari sesuatu yang sempit.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Tipe pencarian tidak dikenal. Pilih pencarian berdasarkan cabang, nama pengguna, atau status aktif.",
		})
		return
	}

	list, err := h.service.List(r.Context(), active.Alias, filter)
	if err != nil {
		if errors.Is(err, mastermasking.ErrStatusNotChosen) {
			// Pesannya diambil apa adanya dari layar lama, yang menyiapkan teks "Pilih
			// Status Aktif" untuk keadaan yang sama persis.
			h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
				Code:    CodeStatusNotChosen,
				Message: "Pilih Status Aktif lebih dulu.",
			})
			return
		}
		h.writeModuleError(w, r, err)
		return
	}

	content := make([]MaskingDTO, 0, len(list))
	for _, m := range list {
		content = append(content, toDTO(m))
	}
	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Masking: content,
		Total:   len(content),
		Portal:  active.Alias,
	})
}

// Branches menangani GET /api/master/masking/cabang.
//
// Menggantikan autocomplete cabang pada layar lama. POOLDATA.BRANCH hanya DIBACA.
func (h *Handler) Branches(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.Branches(r.Context(), active.Alias, r.URL.Query().Get("kata_kunci"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := make([]BranchDTO, 0, len(list))
	for _, b := range list {
		content = append(content, BranchDTO{ID: b.ID, Name: b.Name})
	}
	h.writeResponse(w, r, http.StatusOK, BranchListResponse{
		Branch: content,
		Total:  len(content),
		Portal: active.Alias,
	})
}

// Get menangani GET /api/master/masking/{id}.
//
// Menggantikan aksi EDIT pada `Section/ActionMaskingData_Sec-Section.xml`, yang menyalin
// baris terpilih ke halaman sementara untuk diisi ke form.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	masking, err := h.service.Get(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Masking: toDTO(masking),
		Portal:  active.Alias,
	})
}

// Create menangani POST /api/master/masking.
//
// Menggantikan tombol TAMBAH DATA beserta `Activity/InsermaskingDataKlaimPnc_-Act.xml`
// yang memanggil procedure dengan `T_ACTION = 'insert'`.
//
// Baris baru SELALU aktif. Layar lama pun demikian — tidak ada isian status pada form
// penambahan; status hanya berubah lewat tombol DELETE. Menambah baris yang langsung
// nonaktif adalah kewenangan yang tidak berlaku, dan tidak ada gunanya.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	request, read := h.readSaveRequest(w, r)
	if !read {
		return
	}

	masking := fromRequest(request)
	// Baris baru SELALU aktif, apa pun yang dikirim klien di field `aktif`.
	//
	// Layar lama tidak punya cara membuat baris nonaktif: `T_ACTION = 'insert'` pada
	// procedure menulis `T_STSAKTF` apa adanya, tetapi tombol penonaktifan hanya ada pada
	// baris yang SUDAH tersimpan. Menambah kewenangan yang langsung tidak berlaku tidak
	// ada gunanya, dan membiarkannya hanya membuka satu bentuk keadaan yang membingungkan.
	masking.Active = true

	saved, err := h.service.Create(r.Context(), active.Alias, masking, h.identity(r.Context()))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	// 201, bukan 200: sumber daya baru terbentuk dan ID-nya baru diketahui di sini.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Masking: toDTO(saved),
		Portal:  active.Alias,
	})
}

// Update menangani PUT /api/master/masking/{id}.
//
// Menggantikan tombol SIMPAN pada baris yang sedang diubah, yaitu procedure dengan
// `T_ACTION = 'update'`. ID diambil dari jalur URL, tidak pernah dari badan permintaan.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, mastermasking.ErrNotFound)
		return
	}

	request, read := h.readSaveRequest(w, r)
	if !read {
		return
	}

	saved, err := h.service.Update(r.Context(), active.Alias, id, fromRequest(request), h.identity(r.Context()))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Masking: toDTO(saved),
		Portal:  active.Alias,
	})
}

// SetStatus menangani PUT /api/master/masking/{id}/status.
//
// Menggantikan tombol DELETE pada `Section/ActionMaskingData_Sec-Section.xml`, yang tidak
// pernah membuang baris — ia hanya mengubah STS_AKTF
// (`RDB List/DeleteMstProteksi_SQL-SQL.xml`).
//
// Dibuat sebagai jalur tersendiri, bukan bagian dari form, supaya menyimpan perubahan
// isian tidak dapat mengubah status tanpa sengaja. Menonaktifkan berarti MENCABUT
// kewenangan membuka data pribadi, dan itu harus menjadi tindakan yang disengaja.
func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, mastermasking.ErrNotFound)
		return
	}

	var request StatusRequest
	if !h.decode(w, r, &request) {
		return
	}

	saved, err := h.service.SetActive(r.Context(), active.Alias, id, request.Active, h.identity(r.Context()))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Masking: toDTO(saved),
		Portal:  active.Alias,
	})
}

// readSaveRequest membaca badan permintaan simpan. Nilai kedua false berarti jawabannya
// sudah ditulis dan pemanggil harus berhenti.
func (h *Handler) readSaveRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest
	if !h.decode(w, r, &request) {
		return SaveRequest{}, false
	}
	return request, true
}

// decode membaca satu dokumen JSON dari badan permintaan.
func (h *Handler) decode(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxSaveBodyBytes)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah.
	//
	// Di modul ini ia juga yang menolak klien yang mencoba mengirim `id`, `nama_cabang`,
	// `aktif`, atau `dicatat_oleh` — keempatnya tidak pernah diterima dari luar, dan
	// menolaknya terang-terangan lebih baik daripada mengabaikannya diam-diam.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		// Isi badan permintaan tidak ikut dikembalikan. Memantulkan masukan mentah ke
		// peramban adalah kebiasaan yang tidak layak dimulai di satu tempat pun.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}
	return true
}

// identity mengembalikan penanda pemanggil untuk kolom USERINPUT.
//
// Kosong bila permintaan entah bagaimana tidak melewati middleware Authenticate. Itu
// seharusnya mustahil — seluruh rute modul ini berada di baliknya — dan membiarkannya
// kosong lebih jujur daripada mengisinya dengan tebakan.
func (h *Handler) identity(ctx context.Context) string {
	caller, exists := h.caller(ctx)
	if !exists {
		return ""
	}
	return caller.Identity
}

// fromRequest mengubah isian form menjadi tipe domain.
//
// ID, nama cabang, dan pencatat SENGAJA tidak diisi di sini — ketiganya tidak pernah
// datang dari klien. Yang mengisinya adalah handler (dari sesi dan jalur URL) dan repo
// (nama cabang, dari penggabungan).
//
// Status IKUT dibawa, meniru form layar lama. Pada penambahan ia ditimpa menjadi aktif;
// pada penyuntingan ia dipakai apa adanya.
func fromRequest(request SaveRequest) mastermasking.Masking {
	return mastermasking.Masking{
		BranchID:    request.BranchID,
		Login:       request.Login,
		Module:      request.Module,
		SubModule:   request.SubModule,
		SearchQuota: request.SearchQuota,
		ViewQuota:   request.ViewQuota,
		ViewIDCard:  request.ViewIDCard,
		ViewEmail:   request.ViewEmail,
		ViewPhone:   request.ViewPhone,
		Active:      request.Active,
	}
}

func toDTO(m mastermasking.Masking) MaskingDTO {
	return MaskingDTO{
		ID:          m.ID,
		BranchID:    m.BranchID,
		BranchName:  m.BranchName,
		Login:       m.Login,
		Module:      m.Module,
		SubModule:   m.SubModule,
		SearchQuota: m.SearchQuota,
		ViewQuota:   m.ViewQuota,
		ViewIDCard:  m.ViewIDCard,
		ViewEmail:   m.ViewEmail,
		ViewPhone:   m.ViewPhone,
		Active:      m.Active,
		InputBy:     m.InputBy,
		InputAt:     formatTime(m.InputAt),
	}
}

// formatTime menuliskan waktu dalam RFC 3339 UTC, dan kosong bila waktunya belum terisi.
//
// Waktu nol TIDAK diubah menjadi "0001-01-01T00:00:00Z": tanggal itu akan tampil di layar
// sebagai tanggal sungguhan yang mustahil, dan pengguna tidak punya cara tahu bahwa
// artinya "tidak tercatat".
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
