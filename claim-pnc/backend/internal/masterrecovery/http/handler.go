package masterrecoveryhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterrecovery"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Batas ukuran badan permintaan.
//
// Keduanya berbeda jauh karena isinya berbeda jauh: form simpan hanya berisi isian pendek
// ditambah daftar klaim, sementara unggahan membawa berkas. Menolaknya lebih awal menjaga
// memori tidak dihabiskan badan permintaan yang dikarang.
//
// Batas unggahan diberi kelonggaran di atas batas berkasnya sendiri untuk menampung
// pembungkus multipart — tanpa itu, berkas yang tepat sebesar batas akan ditolak sebelum
// pesan yang menjelaskan batasnya sempat terbentuk.
const (
	maxSaveBodyBytes   = 2 << 20
	maxUploadBodyBytes = masterrecovery.MaxDocumentBytes + (1 << 20)
)

// Service adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	NextBatch(ctx context.Context, portalAlias string) (int64, error)
	Years(ctx context.Context, portalAlias string) []string
	Principals(ctx context.Context, portalAlias string) ([]masterrecovery.Principal, error)
	LookupPolicy(ctx context.Context, portalAlias, policyNo string) (masterrecovery.PolicyReference, error)
	IssueVirtualAccount(ctx context.Context, portalAlias string, request masterrecovery.VirtualAccountRequest) (masterrecovery.VirtualAccount, error)
	SaveDocument(ctx context.Context, portalAlias string, document masterrecovery.Document) (string, error)
	ReadClaimLine(portalAlias string, source io.Reader) ([]masterrecovery.ClaimLine, []masterrecovery.Violation, error)
	Save(ctx context.Context, portalAlias string, recovery masterrecovery.Recovery) (masterrecovery.Recovery, error)
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

// Handler melayani permintaan Master Recovery.
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

	// Caller dipakai mengisi kolom USERNAME. Wajib: batch recovery mencatat nilai uang,
	// dan baris tanpa pencatat tidak dapat ditelusuri siapa pun kelak. `D-59` menjadikan
	// jejak siapa-mengerjakan-apa sebagai satu-satunya kontrol pengimbang yang tersisa.
	Caller CallerReader

	Logger *slog.Logger

	// WriteResponse dan WriteError dipasok dari luar supaya seluruh modul menuliskan
	// respons dan galat dengan cara yang sama. WriteError yang disuntikkan cmd sudah
	// dibungkus portalhttp.WithPortalError, sehingga galat portal terpetakan seragam.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Master Recovery.
//
// Ia menolak bahan yang tidak lengkap saat perakitan di cmd, bukan saat permintaan
// pertama datang: rakitan yang setengah jadi harus gagal saat start.
func NewHandler(o Options) (*Handler, error) {
	switch {
	case o.Service == nil:
		return nil, errors.New("masterrecovery/http: Service wajib diisi")
	case o.Caller == nil:
		return nil, errors.New("masterrecovery/http: Caller wajib diisi")
	case o.WriteResponse == nil || o.WriteError == nil:
		return nil, errors.New("masterrecovery/http: WriteResponse dan WriteError wajib diisi")
	}

	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// Form menangani GET /api/master/recovery/form.
//
// Menggantikan `Activity/GetIDMasterRecoveryKlaim-Act.xml` beserta pengisian dropdown
// Tahun, yang di sistem lama dijalankan saat harness dibuka.
func (h *Handler) Form(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	batch, err := h.service.NextBatch(r.Context(), active)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, FormResponse{
		NextBatch: batch,
		Year:      h.service.Years(r.Context(), active),
		Portal:    active,
	})
}

// Principals menangani GET /api/master/recovery/principal.
//
// Menggantikan pra-aktivitas `getDataAllMSTVA` yang mengisi autocomplete "Nama Principal".
func (h *Handler) Principals(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	list, err := h.service.Principals(r.Context(), active)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := make([]PrincipalDTO, 0, len(list))
	for _, p := range list {
		content = append(content, PrincipalDTO{
			ClientID:             p.ClientID,
			Name:                 p.Name,
			VirtualAccountNumber: p.VirtualAccountNumber,
			Email:                p.Email,
		})
	}
	h.writeResponse(w, r, http.StatusOK, PrincipalListResponse{
		Principal: content,
		Total:     len(content),
		Portal:    active,
	})
}

// Policy menangani GET /api/master/recovery/polis/{nomor}.
//
// Menggantikan `RDB List/GetRecoveryClaimData-SQL.xml`. Di sistem lama pencarian ini
// berjalan sebagai bagian dari aksi simpan, sehingga nomor polis yang salah ketik baru
// ketahuan setelah batch tersimpan dengan keempat identitas kosong. Di sini ia rute
// tersendiri supaya layar dapat memperlihatkan hasilnya lebih dulu.
func (h *Handler) Policy(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	policyNo := strings.TrimSpace(chi.URLParam(r, "nomor"))
	reference, err := h.service.LookupPolicy(r.Context(), active, policyNo)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, PolicyReferenceResponse{
		PolicyNo:    policyNo,
		BusinessID:  reference.BusinessID,
		BranchID:    reference.BranchID,
		AgentID:     reference.AgentID,
		MarketingID: reference.MarketingID,
		Portal:      active,
	})
}

// IssueVirtualAccount menangani POST /api/master/recovery/virtual-account.
//
// Menggantikan tombol **Generated VA** beserta `Activity/GeneratedVAClaimRecovery-Act.xml`.
func (h *Handler) IssueVirtualAccount(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	var request VirtualAccountRequestDTO
	if !h.readJSON(w, r, maxSaveBodyBytes, &request) {
		return
	}

	issued, err := h.service.IssueVirtualAccount(r.Context(), active, masterrecovery.VirtualAccountRequest{
		ClientID:      request.ClientID,
		PrincipalName: request.PrincipalName,
		Email:         request.Email,
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 200, bukan 201, ketika nomornya dipakai ulang: tidak ada yang baru terbentuk. Pada
	// penerbitan baru, 201 menyatakan sebaliknya.
	status := http.StatusCreated
	if issued.Reused {
		status = http.StatusOK
	}
	h.writeResponse(w, r, status, VirtualAccountResponse{
		Number:  issued.Number,
		Status:  issued.Status,
		Message: issued.Message,
		Reused:  issued.Reused,
		Portal:  active,
	})
}

// UploadDocument menangani POST /api/master/recovery/bukti-bayar.
//
// Menggantikan tombol **Upload Document** beserta `Call PNCSaveAttachmentToDB`.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	caller, known := h.caller(r.Context())
	if !known {
		// Tidak boleh terjadi — rute ini di balik middleware sesi. Bila terjadi, ia cacat
		// perakitan, dan menyimpan lampiran tanpa pengunggah jauh lebih buruk daripada
		// menolaknya.
		h.writeError(w, r, errors.New("masterrecovery/http: identitas pemanggil tidak tersedia"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBodyBytes)
	if err := r.ParseMultipartForm(maxUploadBodyBytes); err != nil {
		h.writeResponse(w, r, http.StatusRequestEntityTooLarge, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Berkas tidak dapat dibaca atau melebihi 5 MB.",
		})
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, header, err := r.FormFile("berkas")
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak memuat berkas pada bagian bernama \"berkas\".",
		})
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(io.LimitReader(file, masterrecovery.MaxDocumentBytes+1))
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Berkas tidak dapat dibaca.",
		})
		return
	}

	id, err := h.service.SaveDocument(r.Context(), active, masterrecovery.Document{
		// Nama berkas dibersihkan dari jalur: peramban pada sebagian sistem mengirimkan
		// jalur lengkap, dan menyimpannya apa adanya membocorkan susunan folder pengguna.
		Name:       filepath.Base(header.Filename),
		MimeType:   mimeTypeOf(header.Header.Get("Content-Type"), header.Filename),
		Note:       strings.TrimSpace(r.FormValue("keterangan")),
		Content:    content,
		UploadedBy: caller.Identity,
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusCreated, DocumentResponse{
		DocumentID: id,
		Name:       filepath.Base(header.Filename),
		Portal:     active,
	})
}

// ReadClaimLine menangani POST /api/master/recovery/baris-klaim.
//
// Menggantikan tombol **Upload Data Klaim**. Ia TIDAK menyimpan apa pun — hasilnya
// dikembalikan untuk ditampilkan sebagai grid, lalu dikirim kembali bersama permintaan
// simpan, persis seperti sistem lama menyusunnya di klipboard.
func (h *Handler) ReadClaimLine(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBodyBytes)
	if err := r.ParseMultipartForm(maxUploadBodyBytes); err != nil {
		h.writeResponse(w, r, http.StatusRequestEntityTooLarge, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Berkas tidak dapat dibaca atau terlalu besar.",
		})
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("berkas")
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak memuat berkas pada bagian bernama \"berkas\".",
		})
		return
	}
	defer func() { _ = file.Close() }()

	line, rejected, err := h.service.ReadClaimLine(active, file)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := make([]ClaimLineDTO, 0, len(line))
	for _, l := range line {
		content = append(content, ClaimLineDTO{PolicyNo: l.PolicyNo, ClaimAmount: int64(l.ClaimAmount)})
	}
	reject := make([]ViolationDTO, 0, len(rejected))
	for _, v := range rejected {
		reject = append(reject, ViolationDTO{Field: v.Field, Message: v.Message})
	}

	h.writeResponse(w, r, http.StatusOK, ClaimLineResponse{
		ClaimLine:        content,
		Total:            len(content),
		TotalClaimAmount: int64(masterrecovery.TotalClaimAmount(line)),
		Rejected:         reject,
		Portal:           active,
	})
}

// Template menangani GET /api/master/recovery/format-unggahan.
//
// Menggantikan tautan **Format File** beserta rule `DownloadFileCSVFormaatter`, yang tidak
// ada di export — lihat masterrecovery.ClaimLineTemplate.
func (h *Handler) Template(w http.ResponseWriter, r *http.Request) {
	if _, ready := h.activePortal(w, r); !ready {
		return
	}

	// Ditulis langsung, tidak lewat writeResponse: yang dikirim bukan JSON, dan header
	// Content-Disposition-lah yang membuat peramban mengunduhnya alih-alih menampilkannya.
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="format-data-klaim-recovery.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, masterrecovery.ClaimLineTemplate())
}

// Save menangani POST /api/master/recovery.
//
// Menggantikan tombol **Transfer Recovery** beserta
// `Activity/Insert_mst_recoveryKlaimASM-Act.xml`.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	caller, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New("masterrecovery/http: identitas pemanggil tidak tersedia"))
		return
	}

	var request SaveRequest
	if !h.readJSON(w, r, maxSaveBodyBytes, &request) {
		return
	}

	line := make([]masterrecovery.ClaimLine, 0, len(request.ClaimLine))
	for _, l := range request.ClaimLine {
		line = append(line, masterrecovery.ClaimLine{
			PolicyNo:    l.PolicyNo,
			ClaimAmount: masterrecovery.Amount(l.ClaimAmount),
		})
	}

	saved, err := h.service.Save(r.Context(), active, masterrecovery.Recovery{
		PrincipalName:        request.PrincipalName,
		ClientID:             request.ClientID,
		VirtualAccountNumber: request.VirtualAccountNumber,
		Year:                 request.Year,
		ClaimAmount:          masterrecovery.Amount(request.ClaimAmount),
		PreviousPayment:      masterrecovery.Amount(request.PreviousPayment),
		Payment:              masterrecovery.Amount(request.Payment),
		Remark:               request.Remark,
		CasePosition:         request.CasePosition,
		DocumentID:           request.DocumentID,
		// Diambil dari sesi, TIDAK PERNAH dari badan permintaan: kolom USERNAME menyatakan
		// siapa yang mencatat, dan nilai yang datang dari klien tidak membuktikan apa pun.
		InputBy:      caller.Identity,
		ServiceLogID: request.ServiceLogID,
		PolicyNo:     request.PolicyNo,
		ClaimLine:    line,
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusCreated, SaveResponse{
		Recovery:       toDTO(saved),
		PolicyResolved: saved.PolicyNo == "" || saved.BusinessID != "" || saved.BranchID != "",
		Portal:         active,
	})
}

// activePortal membaca portal aktif. Nilai kedua false berarti jawabannya sudah ditulis
// dan pemanggil harus berhenti.
func (h *Handler) activePortal(w http.ResponseWriter, r *http.Request) (string, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return "", false
	}
	return active.Alias, true
}

// readJSON membaca badan permintaan JSON. Nilai kembalian false berarti jawabannya sudah
// ditulis dan pemanggil harus berhenti.
func (h *Handler) readJSON(w http.ResponseWriter, r *http.Request, limit int64, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah. Ia juga yang menolak klien yang mencoba mengirim
	// `nomor_batch`, `sisa`, maupun keempat identitas polis — seluruhnya milik server.
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

// mimeTypeOf memilih jenis isi berkas.
//
// Yang dikirim peramban dipakai lebih dulu; bila ia kosong atau terlalu umum, jenisnya
// disimpulkan dari akhiran nama berkas. Kolom ATTACHMIMETYPE hanya VARCHAR2(30), sehingga
// parameter tambahan pada nilai seperti `text/plain; charset=utf-8` dibuang — yang
// dibutuhkan hanyalah jenisnya.
func mimeTypeOf(declared, filename string) string {
	if parsed, _, err := mime.ParseMediaType(declared); err == nil && parsed != "" && parsed != "application/octet-stream" {
		return parsed
	}
	if byExtension := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))); byExtension != "" {
		if parsed, _, err := mime.ParseMediaType(byExtension); err == nil {
			return parsed
		}
	}
	return "application/octet-stream"
}

func toDTO(r masterrecovery.Recovery) RecoveryDTO {
	line := make([]ClaimLineDTO, 0, len(r.ClaimLine))
	for _, l := range r.ClaimLine {
		line = append(line, ClaimLineDTO{PolicyNo: l.PolicyNo, ClaimAmount: int64(l.ClaimAmount)})
	}

	return RecoveryDTO{
		Batch:                r.Batch,
		PrincipalName:        r.PrincipalName,
		ClientID:             r.ClientID,
		VirtualAccountNumber: r.VirtualAccountNumber,
		Year:                 r.Year,
		ClaimAmount:          int64(r.ClaimAmount),
		PreviousPayment:      int64(r.PreviousPayment),
		Payment:              int64(r.Payment),
		Remainder:            int64(r.Remainder),
		Remark:               r.Remark,
		CasePosition:         r.CasePosition,
		DocumentID:           r.DocumentID,
		InputBy:              r.InputBy,
		PolicyNo:             r.PolicyNo,
		BusinessID:           r.BusinessID,
		BranchID:             r.BranchID,
		AgentID:              r.AgentID,
		MarketingID:          r.MarketingID,
		ClaimLine:            line,
	}
}
