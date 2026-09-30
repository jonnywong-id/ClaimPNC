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
	"strconv"
	"strings"
	"time"

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
	List(ctx context.Context, portalAlias string, filter masterrecovery.ListFilter) ([]masterrecovery.PrincipalGroup, int, error)
	Document(ctx context.Context, portalAlias, id string) (masterrecovery.Document, error)
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

// List menangani GET /api/master/recovery.
//
// Mengisi tab **Outstanding** pada `Section/OutstandingMasterRecovery-Section.xml`.
//
// # Kenapa rute ini akhirnya ada
//
// Ia sempat sengaja TIDAK didaftarkan, atas kesimpulan saya bahwa layar lama adalah form
// entri tanpa daftar. Kesimpulan itu KELIRU, dan dasarnya keliru: saya menyimpulkannya
// dari tidak adanya kueri pembaca di export, padahal export itu sendiri tidak lengkap
// (`R-16`). Grid-nya ada, jelas terbaca di section, dan berisi data di layar Pega yang
// berjalan. Yang hilang adalah rule pemuatnya, bukan fiturnya.
//
// # Paginasi
//
// `limit` dan `lewati` diterima dari klien tetapi TIDAK dipercaya: keduanya dirapikan di
// lapisan usecase, yang dilewati setiap pemanggil. Nilai yang bukan angka diperlakukan
// sebagai tidak disebutkan, bukan ditolak — pemotongan halaman bukan aturan bisnis, dan
// menolak permintaan karena satu parameter tampilan salah ketik tidak menolong siapa pun.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()
	filter := masterrecovery.ListFilter{
		PrincipalName: strings.TrimSpace(query.Get("cari")),
		Year:          strings.TrimSpace(query.Get("tahun")),
		Limit:         atoiOrZero(query.Get("limit")),
		Offset:        atoiOrZero(query.Get("lewati")),
	}

	groups, total, err := h.service.List(r.Context(), active, filter)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := make([]RecoveryGroupDTO, 0, len(groups))
	for _, group := range groups {
		batch := make([]RecoveryRowDTO, 0, len(group.Batch))
		for _, row := range group.Batch {
			batch = append(batch, toRowDTO(row))
		}
		content = append(content, RecoveryGroupDTO{
			PrincipalName:   group.Name,
			ClaimAmount:     int64(group.Latest.ClaimAmount),
			PreviousPayment: int64(group.Latest.PreviousPayment),
			Payment:         int64(group.Latest.Payment),
			Remainder:       int64(group.Latest.Remainder),
			Batch:           batch,
		})
	}

	h.writeResponse(w, r, http.StatusOK, RecoveryListResponse{
		Principal: content,
		Total:     total,
		Portal:    active,
	})
}

func toRowDTO(row masterrecovery.Recovery) RecoveryRowDTO {
	// Waktu nol dikirim sebagai teks KOSONG, bukan sebagai "0001-01-01T00:00:00Z".
	// Tanggal tahun satu yang muncul di layar jauh lebih membingungkan daripada kolom
	// kosong, dan baris warisan tanpa INSERTDATE memang ada.
	inputDate := ""
	if !row.InputDate.IsZero() {
		inputDate = row.InputDate.Format(time.RFC3339)
	}

	var attachment *AttachmentDTO
	if row.Attachment != nil {
		uploadedAt := ""
		if !row.Attachment.UploadedAt.IsZero() {
			uploadedAt = row.Attachment.UploadedAt.Format(time.RFC3339)
		}
		attachment = &AttachmentDTO{
			ID:         row.Attachment.ID,
			Name:       row.Attachment.Name,
			UploadedBy: row.Attachment.UploadedBy,
			UploadedAt: uploadedAt,
		}
	}

	return RecoveryRowDTO{
		Attachment:           attachment,
		Batch:                row.Batch,
		PrincipalName:        row.PrincipalName,
		Year:                 row.Year,
		InputDate:            inputDate,
		ServiceLogID:         row.ServiceLogID,
		ClaimAmount:          int64(row.ClaimAmount),
		PreviousPayment:      int64(row.PreviousPayment),
		Payment:              int64(row.Payment),
		Remainder:            int64(row.Remainder),
		Remark:               row.Remark,
		CasePosition:         row.CasePosition,
		VirtualAccountNumber: row.VirtualAccountNumber,
		PolicyNo:             row.PolicyNo,
		DocumentID:           strings.TrimSpace(row.DocumentID),
	}
}

// Document menangani GET /api/master/recovery/bukti-bayar/{id}.
//
// Melayani tombol **View Document** pada grid dalam.
//
// # Isinya dialirkan mentah, bukan sebagai JSON ber-base64
//
// `RDB List/GetAttachmentFromDB_Sql-SQL.xml` mengembalikannya sebagai teks base64 lewat
// `pooldata.base64encode`. Di sini tidak: bytenya dikirim apa adanya beserta jenis isinya,
// sehingga peramban dapat membukanya sendiri, ukurannya tidak membesar sepertiga, dan
// tidak ada yang perlu diurai ulang di sisi mana pun.
//
// # `inline` HANYA untuk jenis yang memang dapat ditampilkan — diperbaiki 2026-09-29
//
// Versi pertama menyajikan seluruh berkas `inline` dengan jenis isi apa adanya dari kolom
// `ATTACHMIMETYPE`. Itu keliru dua kali:
//
//  1. **Keluarannya kacau.** Lampiran warisan yang jenisnya tidak tercatat, atau tercatat
//     keliru, membuat peramban menggambar isi biner sebagai teks — berhalaman-halaman
//     karakter acak alih-alih berkas. Terlihat langsung pada lampiran XLSX di portal ASM.
//  2. **Ia celah keamanan.** Jenis isi itu datang dari BASIS DATA, bukan dari kode. Satu
//     baris ber-`ATTACHMIMETYPE` `text/html` akan dijalankan peramban sebagai halaman di
//     origin aplikasi ini — skrip di dalamnya membaca sesi pengguna yang membukanya.
//
// Karena itu `inline` hanya diberikan kepada **daftar jenis yang memang aman ditampilkan**
// (`inlineSafe`). Selebihnya dikirim sebagai `attachment`, sehingga peramban mengunduhnya
// alih-alih menggambarnya. Ditambah `X-Content-Type-Options: nosniff`, supaya peramban
// tidak menebak sendiri jenisnya dan membatalkan pembedaan di atas.
//
// Tombolnya tetap bernama **View** Document dan tetap jujur: yang dapat dilihat, dilihat;
// yang tidak, diunduh — dan tidak ada yang berakhir sebagai layar penuh karakter acak.
func (h *Handler) Document(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeError(w, r, masterrecovery.ErrDocumentNotFound)
		return
	}

	document, err := h.service.Document(r.Context(), active, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	mimeType := strings.ToLower(strings.TrimSpace(document.MimeType))
	if mimeType == "" {
		// Jenis isi yang tidak tercatat TIDAK ditebak dari nama berkas: menebak salah
		// membuat peramban menampilkan isi yang keliru. Oktet mentah membuatnya diunduh,
		// dan berkasnya tetap utuh.
		mimeType = "application/octet-stream"
	}

	name := strings.TrimSpace(document.Name)
	if name == "" {
		name = "bukti-bayar"
	}

	// Ditampilkan hanya bila jenisnya ada di daftar aman; selebihnya diunduh.
	disposition := "attachment"
	if inlineSafe[mimeType] {
		disposition = "inline"
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Length", strconv.Itoa(len(document.Content)))
	// Melarang peramban menebak sendiri jenis isinya. Tanpa ini, berkas ber-jenis
	// application/octet-stream yang isinya kebetulan menyerupai HTML masih dapat
	// digambar sebagai halaman — dan pembedaan di atas menjadi tidak berarti.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{
		"filename": name,
	}))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(document.Content); err != nil {
		h.logger.WarnContext(r.Context(), "bukti bayar gagal dialirkan",
			slog.String("modul", "masterrecovery"), slog.Any("galat", err))
	}
}

// inlineSafe adalah jenis isi yang boleh DIGAMBAR peramban, bukan diunduh.
//
// Daftar, bukan aturan — dan pendek dengan sengaja. Yang masuk hanyalah jenis yang tidak
// dapat menjalankan skrip di origin aplikasi ini:
//
//   - PDF dan gambar raster: digambar peramban, tidak dieksekusi.
//   - `text/plain`: digambar apa adanya; peramban TIDAK menafsirkan tag di dalamnya.
//
// Yang sengaja TIDAK masuk, meski tampak tidak berbahaya:
//
//   - `image/svg+xml` — SVG dapat memuat `<script>`, dan dijalankan saat dibuka langsung.
//   - `text/html`, `application/xhtml+xml`, `application/xml` — jelas dapat menjalankan
//     skrip.
//   - Seluruh bentuk Office — peramban tidak dapat menampilkannya, dan memaksanya inline
//     justru menghasilkan layar penuh karakter acak. Inilah yang terjadi pada lampiran
//     XLSX di portal ASM.
var inlineSafe = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
	"image/jpeg":      true,
	"image/jpg":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/bmp":       true,
	"image/tiff":      true,
	"text/plain":      true,
}

// atoiOrZero membaca angka desimal dan mengembalikan 0 bila tidak dapat dibaca.
//
// Nol berarti "tidak disebutkan" di seluruh pemakaiannya, sehingga nilai cacat jatuh ke
// perilaku baku alih-alih menggagalkan permintaan.
func atoiOrZero(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
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
// Menggantikan tautan **Format File** beserta rule `DownloadFileCSVFormaatter`.
//
// Isi contohnya sengaja berbeda dari yang lama — yang lama hanya memuat satu kolom,
// sehingga tidak dapat diunggah kembali lewat grid yang menuntut dua. Alasannya di
// masterrecovery.ClaimLineTemplate.
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
