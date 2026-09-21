package pelaporanklaimhttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/pelaporanklaim"
	"claim-pnc/internal/pelaporanklaim/usecase"
)

// maxSaveBodyBytes membatasi ukuran badan permintaan simpan.
//
// Form ini memuat tiga kolom teks panjang — kronologi, rincian kerusakan, dan alasan —
// yang masing-masing dibatasi 4.000 karakter oleh domain. 64 KiB memberi ruang lapang di
// atasnya sekaligus menolak badan permintaan yang dikarang sebelum ia menghabiskan memori.
const maxSaveBodyBytes = 64 << 10

// Service adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor sebagai tipe
// konkret, supaya handler dapat diuji tanpa membentuk seluruh layanan beserta
// penyimpanannya.
type Service interface {
	List(ctx context.Context, f pelaporanklaim.Filter) (usecase.ListResult, error)
	Get(ctx context.Context, number string) (pelaporanklaim.ClaimReport, error)
	Record(ctx context.Context, report pelaporanklaim.ClaimReport, by usecase.Recorder) (pelaporanklaim.ClaimReport, error)
	Update(ctx context.Context, number string, report pelaporanklaim.ClaimReport) (pelaporanklaim.ClaimReport, error)
	Transfer(ctx context.Context, number string) (pelaporanklaim.ClaimReport, error)
	LinkClaim(ctx context.Context, number, claimNumber string) (pelaporanklaim.ClaimReport, error)
}

// Caller adalah identitas pengguna yang sedang masuk, sejauh yang dibutuhkan modul ini.
//
// Ia sengaja hanya memuat dua field. Modul ini tidak perlu tahu apa pun tentang bentuk
// sesi, dan modul auth tidak perlu tahu modul ini ada — jembatannya dipasang di
// cmd/claimpnc, satu-satunya berkas yang memang tahu keduanya.
type Caller struct {
	Login      string
	BranchCode string
}

// GetCaller membaca identitas pengguna dari konteks permintaan.
type GetCaller func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan Pelaporan Klaim.
type Handler struct {
	service     Service
	getCaller   GetCaller
	logger      *slog.Logger
	writeJSON   JSONWriter
	writeErrorF ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service   Service
	GetCaller GetCaller
	Logger    *slog.Logger

	// WriteJSON dan FallbackErrorWriter dipasok dari luar supaya seluruh modul menuliskan
	// respons dan galat sesi dengan cara yang sama.
	WriteJSON           JSONWriter
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Pelaporan Klaim.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:     o.Service,
		getCaller:   o.GetCaller,
		logger:      o.Logger,
		writeJSON:   o.WriteJSON,
		writeErrorF: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// List menangani GET /api/pelaporan-klaim.
//
// Menggantikan ketiga kueri inbox sistem lama sekaligus — `ViewTableBrowseRCVInProcess`,
// `ViewTableBrowseRCVAcc`, dan `ViewTableBrowseRCVReject` — yang di sana menjadi tiga rule
// terpisah karena penyaringnya dirangkai ke dalam teks SQL. Dengan penyaring sebagai
// parameter, ketiganya menjadi satu kueri dan satu endpoint.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := pelaporanklaim.Filter{
		Stage:      pelaporanklaim.Stage(trimSpace(q.Get("tahap"))),
		Search:     q.Get("cari"),
		BranchCode: q.Get("cabang"),
		Limit:      intParam(q.Get("batas"), 0),
		Offset:     intParam(q.Get("lewati"), 0),
	}

	result, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}

	items := make([]ReportDTO, 0, len(result.Page.Reports))
	for _, report := range result.Page.Reports {
		items = append(items, toDTO(report))
	}

	normalized := filter.Normalize()
	h.writeJSON(w, r, http.StatusOK, ListResponse{
		Reports: items,
		Total:   result.Page.Total,
		Limit:   normalized.Limit,
		Offset:  normalized.Offset,
		Summary: toSummaryDTO(result.Summary),
	})
}

// Get menangani GET /api/pelaporan-klaim/{nomor}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.Get(r.Context(), chi.URLParam(r, "nomor"))
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, SingleResponse{Report: toDTO(report)})
}

// Record menangani POST /api/pelaporan-klaim.
//
// Menggantikan `Activity/CreateNewCaseRCV-Act.xml`, yang membuat case RCV lalu memanggil
// `rcv_InsertRecivedDocumentClaim`.
func (h *Handler) Record(w http.ResponseWriter, r *http.Request) {
	req, ok := h.readSaveRequest(w, r)
	if !ok {
		return
	}
	if !h.datesParseable(w, r, req) {
		return
	}

	caller, ok := h.caller(w, r)
	if !ok {
		return
	}

	report, err := h.service.Record(r.Context(), toDomain(req), usecase.Recorder{
		Login:      caller.Login,
		BranchCode: caller.BranchCode,
	})
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}
	// 201, bukan 200: laporan baru terbentuk dan nomornya baru diketahui di sini.
	h.writeJSON(w, r, http.StatusCreated, SingleResponse{Report: toDTO(report)})
}

// Update menangani PUT /api/pelaporan-klaim/{nomor}.
//
// PUT, bukan PATCH: seluruh isi yang boleh diubah dikirim setiap kali, sehingga
// permintaannya menggantikan dan idempoten. Mengirim permintaan yang sama dua kali
// menghasilkan keadaan akhir yang sama.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	req, ok := h.readSaveRequest(w, r)
	if !ok {
		return
	}
	if !h.datesParseable(w, r, req) {
		return
	}

	report, err := h.service.Update(r.Context(), chi.URLParam(r, "nomor"), toDomain(req))
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, SingleResponse{Report: toDTO(report)})
}

// Transfer menangani POST /api/pelaporan-klaim/{nomor}/transfer.
//
// Aksi bisnis dimodelkan sebagai sub-sumber daya, bukan sebagai PATCH yang menyetel satu
// field (`10-API-STRATEGY.md` §2). Alasannya bukan gaya: transfer punya invarian sendiri —
// tidak boleh terjadi dua kali, dan tidak boleh terjadi setelah registrasi — dan pembaruan
// field generik akan melewatkan keduanya.
func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.Transfer(r.Context(), chi.URLParam(r, "nomor"))
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, SingleResponse{Report: toDTO(report)})
}

// LinkClaim menangani POST /api/pelaporan-klaim/{nomor}/klaim.
//
// Menggantikan `Activity/UpdateRCVCase-Act.xml`, yang di sistem lama dijalankan dari sisi
// KLAIM: klaim membuka case laporan lalu menulis ke dalamnya.
func (h *Handler) LinkClaim(w http.ResponseWriter, r *http.Request) {
	var req LinkClaimRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxSaveBodyBytes)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeBadRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return
	}

	report, err := h.service.LinkClaim(r.Context(), chi.URLParam(r, "nomor"), req.ClaimNumber)
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, SingleResponse{Report: toDTO(report)})
}

// readSaveRequest membaca badan permintaan catat dan ubah. Nilai kedua false berarti
// jawabannya sudah ditulis dan pemanggil harus berhenti.
func (h *Handler) readSaveRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var req SaveRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxSaveBodyBytes)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Isi badan permintaan TIDAK ikut dikembalikan. Form ini memuat nama tertanggung,
		// nomor polis, dan alamat surel; memantulkan masukan mentah ke peramban akan
		// membuat data nasabah ikut tercatat di log proxy dan riwayat peramban.
		h.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeBadRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest{}, false
	}
	return req, true
}

// datesParseable memastikan kedua kolom tanggal berbentuk YYYY-MM-DD bila diisi.
//
// Pemeriksaannya ada di sini, bukan di domain, karena yang salah adalah BENTUK TEKS-nya —
// dan begitu teks itu diubah menjadi tanggal, kesalahannya sudah hilang tanpa jejak.
// Tanpa pemeriksaan ini, tanggal yang salah ketik akan tersimpan diam-diam sebagai kosong,
// dan pengguna baru menyadarinya saat laporan tidak muncul pada penyaringan periode.
func (h *Handler) datesParseable(w http.ResponseWriter, r *http.Request, req SaveRequest) bool {
	var violations []ViolationDTO

	check := func(field, value, label string) {
		value = trimSpace(value)
		if value == "" {
			return
		}
		if _, err := time.ParseInLocation(DateFormat, value, time.UTC); err != nil {
			violations = append(violations, ViolationDTO{
				Field:   field,
				Message: label + " harus berbentuk TTTT-BB-HH, misalnya 2026-09-18.",
			})
		}
	}
	check("tanggal_kejadian", req.LossDate, "Tanggal kejadian")
	check("tanggal_terima_dokumen", req.DocumentReceivedDate, "Tanggal terima dokumen")

	if len(violations) == 0 {
		return true
	}
	h.writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
		Code:    CodeValidationFail,
		Message: "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
		Details: violations,
	})
	return false
}

// caller membaca identitas pengguna, atau menjawab 401 bila tidak ada.
//
// Seharusnya tidak pernah terjadi: rute modul ini berada di balik middleware sesi. Ia ada
// supaya kegagalan perakitan — jembatan pemanggil yang lupa dipasang di cmd — terlihat
// sebagai penolakan, bukan sebagai laporan yang tercatat tanpa nama pencatat.
func (h *Handler) caller(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	if h.getCaller == nil {
		h.writeJSON(w, r, http.StatusUnauthorized, ErrorResponse{
			Code:    CodeBadRequest,
			Message: "Sesi tidak dikenali.",
		})
		return Caller{}, false
	}
	caller, ok := h.getCaller(r.Context())
	if !ok {
		h.writeJSON(w, r, http.StatusUnauthorized, ErrorResponse{
			Code:    CodeBadRequest,
			Message: "Sesi tidak dikenali.",
		})
		return Caller{}, false
	}
	return caller, true
}

// toSummaryDTO menyusun lencana tiap tab dalam URUTAN TETAP.
//
// Urutannya mengikuti perjalanan laporan, bukan urutan map — map Go tidak punya urutan,
// dan membiarkannya membuat tab berpindah tempat setiap kali halaman dimuat.
func toSummaryDTO(summary pelaporanklaim.StageSummary) []SummaryDTO {
	order := []pelaporanklaim.Stage{
		pelaporanklaim.StageNotTransferred,
		pelaporanklaim.StageNotRegistered,
		pelaporanklaim.StageRegistered,
		pelaporanklaim.StageAccepted,
		pelaporanklaim.StageRejected,
	}

	result := make([]SummaryDTO, 0, len(order))
	for _, stage := range order {
		result = append(result, SummaryDTO{
			Stage: string(stage),
			Label: stage.Label(),
			Count: summary[stage],
		})
	}
	return result
}

// intParam membaca parameter kueri berupa bilangan, atau mengembalikan nilai baku bila
// kosong maupun tidak terbaca.
//
// Nilai yang tidak terbaca TIDAK menjadi galat: parameter paginasi yang salah ketik lebih
// baik jatuh ke halaman pertama daripada menolak seluruh permintaan.
func intParam(text string, fallback int) int {
	text = trimSpace(text)
	if text == "" {
		return fallback
	}
	n, err := strconv.Atoi(text)
	if err != nil {
		return fallback
	}
	return n
}

// trimSpace membuang spasi tepi.
func trimSpace(s string) string { return strings.TrimSpace(s) }
