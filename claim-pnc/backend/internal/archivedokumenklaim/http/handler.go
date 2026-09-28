package archivedokumenklaimhttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/archivedokumenklaim"
	"claim-pnc/internal/archivedokumenklaim/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login mengisi kolom USERINPUT.
	Login string

	// Position menentukan lini bisnis yang tampak pada daftar kirim ke cabang.
	Position string

	// BranchCode mengisi kolom KODECABANG. Lihat archivedokumenklaim.Draft.BranchCode.
	BranchCode string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// maxRequestBody adalah batas panjang badan permintaan penyimpanan.
//
// Formulirnya dua belas isian pendek, sehingga 64 KiB sudah jauh melampaui kebutuhannya.
// Batasnya ada untuk menahan yang tidak wajar, bukan untuk membatasi yang wajar.
const maxRequestBody = 64 << 10

// Handler melayani permintaan modul Archive Dokumen Klaim.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	logger     *slog.Logger
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan
	// diimpor dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa
	// menariknya serta.
	GetCaller CallerReader

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
	// galat portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Archive Dokumen Klaim.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Open menangani GET /arsip-dokumen/buka.
//
// Ia GET, berbeda dari layar View History Claim yang pembukaannya POST. Alasannya nyata,
// bukan selera: membuka layar ini tidak mengubah apa pun dan tidak memakai jatah apa pun,
// sehingga mengulang permintaannya aman — dan itulah yang dijanjikan GET.
func (h *Handler) Open(w http.ResponseWriter, r *http.Request) {
	active, caller, ok := h.begin(w, r)
	if !ok {
		return
	}

	opened, err := h.service.Open(r.Context(), active.Alias, toDomainCaller(caller))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toOpenResponse(opened, active.Alias))
}

// Search menangani GET /arsip-dokumen.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	active, _, ok := h.begin(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	from, readable := parseDate(query.Get("tanggal_dari"))
	if !readable {
		h.writeBadDate(w, r, archivedokumenklaim.FieldFrom)
		return
	}

	until, readable := parseDate(query.Get("tanggal_sampai"))
	if !readable {
		h.writeBadDate(w, r, archivedokumenklaim.FieldTo)
		return
	}

	found, err := h.service.Search(
		r.Context(),
		active.Alias,
		archivedokumenklaim.CriteriaInput{
			Mode:    query.Get("mode"),
			Keyword: query.Get("kata_kunci"),
			From:    from,
			To:      until,
		},
		archivedokumenklaim.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, SearchResponse{
		Files:      toArchiveFileListDTO(found.Page.Files),
		Pagination: toPaginationDTO(found.Page),
		Portal:     active.Alias,
	})
}

// SearchClaims menangani GET /arsip-dokumen/klaim.
func (h *Handler) SearchClaims(w http.ResponseWriter, r *http.Request) {
	active, _, ok := h.begin(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	claims, err := h.service.SearchClaims(
		r.Context(), active.Alias, query.Get("tipe"), query.Get("nilai"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, ClaimSearchResponse{
		Claims: toClaimListDTO(claims),
		Portal: active.Alias,
	})
}

// FillingCodes menangani GET /arsip-dokumen/kode-filling.
func (h *Handler) FillingCodes(w http.ResponseWriter, r *http.Request) {
	active, _, ok := h.begin(w, r)
	if !ok {
		return
	}

	codes, err := h.service.FillingCodes(
		r.Context(), active.Alias, r.URL.Query().Get("kata_kunci"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, FillingCodeResponse{
		Codes:         toFillingCodeListDTO(codes),
		Reconstructed: true,
		Portal:        active.Alias,
	})
}

// Save menangani POST /arsip-dokumen.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, caller, ok := h.begin(w, r)
	if !ok {
		return
	}

	var body SaveRequest
	if err := decodeJSON(r, &body); err != nil {
		h.writeBadRequest(w, r, "Badan permintaan tidak dapat dibaca sebagai JSON.")
		return
	}

	lossDate, readable := parseDate(valueOf(body.LossDate))
	if !readable {
		h.writeBadDate(w, r, archivedokumenklaim.FieldClaimNumber)
		return
	}

	receivedDate, readable := parseDate(valueOf(body.DocumentReceivedDate))
	if !readable {
		h.writeBadDate(w, r, archivedokumenklaim.FieldReceivedDate)
		return
	}

	saved, err := h.service.Save(
		r.Context(),
		active.Alias,
		toDomainCaller(caller),
		caller.BranchCode,
		archivedokumenklaim.DraftInput{
			ID:                   body.ID,
			ClaimNumber:          body.ClaimNumber,
			PolicyNumber:         body.PolicyNumber,
			InsuredName:          body.InsuredName,
			LossDate:             lossDate,
			TechnicalPIC:         body.TechnicalPIC,
			GroupPanel:           body.GroupPanel,
			DocumentReceivedDate: receivedDate,
			SheetCount:           body.SheetCount,
			DocumentTypeCode:     body.DocumentTypeCode,
			DocumentKindCode:     body.DocumentKindCode,
			BoxName:              body.BoxName,
			FillingCode:          body.FillingCode,
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	status := http.StatusOK
	message := "Berkas arsip diperbarui."
	if saved.Created {
		status = http.StatusCreated
		message = "Berkas arsip tersimpan."
	}

	// Keadaan pengiriman ikut disebut di pesannya. Pengguna perlu tahu bedanya
	// "tersimpan dan terkirim" dari "tersimpan tetapi belum terkirim" — yang kedua
	// menuntut tindakan lanjutan dari tab Kirim ke Cabang.
	switch {
	case saved.Sent:
		message += " Berkas juga dikirim ke sistem Arsip."
	case saved.SendError != "":
		message += " Pengirimannya ke sistem Arsip GAGAL; berkasnya tetap tersimpan " +
			"dan dapat dikirim ulang dari tab Kirim ke Cabang."
	}

	h.writeJSON(w, r, status, SaveResponse{
		ID:          saved.ID,
		Created:     saved.Created,
		Message:     message,
		Sent:        saved.Sent,
		ServiceCode: saved.ServiceCode,
		ServiceNote: saved.ServiceNote,
		SendError:   saved.SendError,
		Portal:      active.Alias,
	})
}

// Pending menangani GET /arsip-dokumen/kirim-cabang.
func (h *Handler) Pending(w http.ResponseWriter, r *http.Request) {
	active, caller, ok := h.begin(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	pending, err := h.service.PendingBranch(
		r.Context(),
		active.Alias,
		toDomainCaller(caller),
		archivedokumenklaim.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, PendingResponse{
		Files:      toArchiveFileListDTO(pending.Page.Files),
		Pagination: toPaginationDTO(pending.Page),
		Scope:      toBranchScopeDTO(pending.Scope),
		Portal:     active.Alias,
	})
}

// Send menangani POST /arsip-dokumen/{id}/kirim-cabang.
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	active, caller, ok := h.begin(w, r)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
	if err != nil || id <= 0 {
		h.writeBadRequest(w, r, "Nomor berkas arsip tidak dapat dibaca.")
		return
	}

	sent, err := h.service.SendToBranch(r.Context(), active.Alias, toDomainCaller(caller), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, SendResponse{
		ID:          sent.ID,
		ServiceCode: sent.Code,
		ServiceNote: sent.Note,
		Message:     "Berkas dikirim ke sistem Arsip.",
		Portal:      active.Alias,
	})
}

// begin membaca portal aktif dan identitas pemanggil, atau menuliskan galatnya.
//
// Keduanya dibutuhkan SETIAP rute, dan menyalin pemeriksaannya ke enam handler adalah cara
// paling mudah melewatkannya di satu tempat tanpa ada yang menyadarinya.
func (h *Handler) begin(
	w http.ResponseWriter,
	r *http.Request,
) (portal.Portal, Caller, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, archivedokumenklaim.ErrCallerUnknown)
		return portal.Portal{}, Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
//
// Ia mengembalikan tipe transport, bukan tipe domain, karena kode cabang hanya dipakai
// satu operasi — memasukkannya ke tipe domain akan membawanya ke lima operasi lain yang
// tidak membutuhkannya.
func (h *Handler) readCaller(r *http.Request) (Caller, bool) {
	if h.caller == nil {
		return Caller{}, false
	}

	caller, exists := h.caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return Caller{}, false
	}
	return caller, true
}

// toDomainCaller memetakan identitas pemanggil ke tipe modul.
func toDomainCaller(caller Caller) archivedokumenklaim.Caller {
	return archivedokumenklaim.Caller{
		Login:    caller.Login,
		Position: caller.Position,
	}
}

// writeBadDate menjawab tanggal yang tidak dapat dibaca.
//
// 422 dengan penunjuk isian, bukan 400: bentuk permintaannya benar — parameternya ada dan
// bertipe teks — hanya isinya yang tidak dapat dibaca sebagai tanggal. Layar dapat
// menandai isian yang salah alih-alih menampilkan galat umum.
func (h *Handler) writeBadDate(w http.ResponseWriter, r *http.Request, field string) {
	h.writeError(w, r, archivedokumenklaim.NewValidationError([]archivedokumenklaim.Violation{{
		Field:   field,
		Message: "Tanggal tidak dapat dibaca. Bentuknya YYYY-MM-DD.",
	}}))
}

// writeBadRequest menjawab permintaan yang bentuknya salah.
func (h *Handler) writeBadRequest(w http.ResponseWriter, r *http.Request, message string) {
	h.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
		Code:    CodeBadRequest,
		Message: message,
	})
}

// decodeJSON membaca badan permintaan.
//
// Field yang tidak dikenal DITOLAK. Klien yang mengirim `jumlah_lembaran` alih-alih
// `jumlah_lembar` akan menerima galat alih-alih menyimpan berkas tanpa jumlah lembar —
// dan yang kedua tidak meninggalkan satu pun jejak bahwa ada yang salah.
func decodeJSON(r *http.Request, target any) error {
	// Badan permintaan dibatasi panjangnya. Tanpa batas, satu permintaan bercanda
	// sepanjang gigabyte cukup untuk menjatuhkan aplikasi.
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxRequestBody))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

// parseDate membaca tanggal berbentuk YYYY-MM-DD.
//
// Teks kosong menghasilkan nil DAN dinyatakan terbaca: isian tanggal yang tidak diisi
// bukan isian yang salah, dan aturan wajib-tidaknya ditegakkan lapisan domain.
func parseDate(raw string) (*time.Time, bool) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, true
	}

	moment, err := time.Parse(dateLayout, clean)
	if err != nil {
		return nil, false
	}
	return &moment, true
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan Pagination.Normalize membetulkannya
// menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan membuat layar
// gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman pertama adalah
// jawaban yang selalu masuk akal.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// valueOf membaca pointer teks yang boleh kosong.
func valueOf(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
