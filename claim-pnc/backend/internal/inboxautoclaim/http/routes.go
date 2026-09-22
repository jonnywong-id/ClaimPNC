package inboxautoclaimhttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/inboxautoclaim/usecase"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxUploadRequest membatasi besar permintaan unggahan.
//
// 12 MiB: delapan untuk berkasnya (batas di inboxautoclaim.ParseUpload) ditambah ruang
// untuk pembungkus multipart. Batasnya ada di sini DAN di domain dengan sengaja — yang
// di sini menolak sebelum seluruh badan permintaan ditarik dari jaringan, yang di domain
// tetap berlaku bila fungsi itu dipanggil dari tempat lain.
const maxUploadRequest = 12 << 20

// uploadFormField adalah nama bagian multipart tempat berkas dikirim.
const uploadFormField = "berkas"

// Caller menyebut pemanggil yang sudah terautentikasi.
//
// Ia dipakai mengisi kolom USERINPUT pada baris yang diunggah — kolom yang menjadi
// "User Upload" di grid. Bentuknya sempit dengan sengaja: modul ini hanya butuh tahu
// SIAPA yang mengunggah, bukan seluruh profil pengguna.
type Caller struct {
	// Login adalah nama pengguna yang diketik saat masuk, bukan NIK.
	//
	// Yang dipakai adalah Login karena itulah yang tertulis di kolom USERINPUT pada baris
	// lama: RDB List/GetHasilAutoClaim-SQL.xml menyaringnya dengan
	// {OperatorID.pyUserIdentifier}, yaitu identitas operator Pega — padanan terdekatnya
	// di sistem baru adalah login, bukan NIK.
	Login string
}

// Handler melayani permintaan Inbox Auto Claim.
type Handler struct {
	service       *usecase.Service
	caller        func(context.Context) (Caller, bool)
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// Caller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan dari cmd, bukan
	// diimpor di sini, supaya kedua modul tetap tidak saling mengimpor.
	Caller func(context.Context) (Caller, bool)

	Logger *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Inbox Auto Claim.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("inboxautoclaim/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("inboxautoclaim/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("inboxautoclaim/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// readSource membaca tab dari parameter ?sumber=.
//
// Tab yang tidak dikenal DITOLAK, bukan dibulatkan ke tab bawaan: membulatkannya berarti
// menampilkan data tabel lain sambil menyorot tab yang diminta, tanpa satu pun tanda di
// layar. Itu kelas kesalahan yang sama dengan jatuh ke portal default (R-20).
func (h *Handler) readSource(w http.ResponseWriter, r *http.Request) (inboxautoclaim.Source, bool) {
	source, err := inboxautoclaim.ParseSource(r.URL.Query().Get("sumber"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return "", false
	}
	return source, true
}

// ListBatch menangani GET /inbox-auto-claim.
func (h *Handler) ListBatch(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	source, ok := h.readSource(w, r)
	if !ok {
		return
	}

	filter := inboxautoclaim.BatchFilter{
		Source:      source,
		CompanyCode: strings.TrimSpace(r.URL.Query().Get("perusahaan")),
		Page:        readPage(r),
	}

	page, err := h.service.ListBatch(r.Context(), active.Alias, filter)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, BatchListResponse{
		Batch:  toBatchListDTO(page.Item),
		Page:   toPageDTO(filter.Page, page.Total),
		Portal: active.Alias,
	})
}

// ListTab menangani GET /inbox-auto-claim/tab.
//
// Rutenya TIDAK dipasangi pemeriksaan portal: daftar tab sama untuk setiap entitas dan
// tidak satu baris data entitas pun dibacanya. Menuntut portal di sini akan membuat
// layar gagal menggambar tabnya justru sebelum pengguna memilih entitas.
func (h *Handler) ListTab(w http.ResponseWriter, r *http.Request) {
	all := inboxautoclaim.AllSource()
	tab := make([]TabDTO, 0, len(all))
	for _, source := range all {
		info, exists := source.Info()
		if !exists {
			// Tidak mungkin terjadi: AllSource hanya mengembalikan yang terdaftar. Bila
			// suatu saat terjadi, tabnya dilewati — lebih baik kehilangan satu tab
			// daripada mengirim tab tanpa tabel yang gridnya pasti gagal dimuat.
			continue
		}
		tab = append(tab, TabDTO{Kode: string(source), Label: info.Label, Tabel: info.Table})
	}
	h.writeResponse(w, r, http.StatusOK, TabListResponse{
		Tab:    tab,
		Bawaan: string(inboxautoclaim.DefaultSource),
	})
}

// ListCompany menangani GET /inbox-auto-claim/perusahaan.
//
// Rutenya TETAP di balik pemeriksaan portal, berbeda dengan daftar posisi klaim pada
// modul master status progres. Sebabnya: daftar ini dibaca dari basis data entitas —
// perusahaan rekanan satu badan hukum bukan urusan badan hukum lain.
func (h *Handler) ListCompany(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.ListCompany(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, CompanyListResponse{
		Perusahaan: toCompanyListDTO(list),
		Portal:     active.Alias,
	})
}

// Summarize menangani GET /inbox-auto-claim/ringkasan.
//
// Rutenya didaftarkan SEBELUM `/{kode}/{batch}` dengan sengaja — sama seperti
// `/perusahaan`. Tanpa urutan itu, chi membaca "ringkasan" sebagai nilai `{kode}` dan
// permintaannya jatuh ke handler rincian batch, yang menjawab galat yang sama sekali
// tidak berhubungan.
func (h *Handler) Summarize(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	source, ok := h.readSource(w, r)
	if !ok {
		return
	}

	summary, err := h.service.SummarizeCompany(r.Context(), active.Alias, source)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SummaryResponse{
		Perusahaan: toSummaryListDTO(summary.Company),
		Total:      summary.Total,
		Portal:     active.Alias,
	})
}

// ListLine menangani GET /inbox-auto-claim/{kode}/{batch}.
func (h *Handler) ListLine(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	source, ok := h.readSource(w, r)
	if !ok {
		return
	}

	companyCode, batchNumber, ok := h.readKey(w, r)
	if !ok {
		return
	}

	result, err := inboxautoclaim.ParseResult(r.URL.Query().Get("hasil"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	query := inboxautoclaim.LineQuery{
		Source:      source,
		CompanyCode: companyCode,
		BatchNumber: batchNumber,
		Result:      result,
		Page:        readPage(r),
	}

	page, err := h.service.ListLine(r.Context(), active.Alias, query)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, LineListResponse{
		KodePerusahaan: companyCode,
		Batch:          batchNumber,
		Baris:          toLineListDTO(page.Item),
		Page:           toPageDTO(query.Page, page.Total),
		Portal:         active.Alias,
	})
}

// Export menangani GET /inbox-auto-claim/{kode}/{batch}/ekspor.
//
// Ia satu-satunya rute modul ini yang TIDAK menjawab JSON. Badan responsnya berkas CSV,
// sehingga penulis respons bersama tidak dipakai — tetapi jalur GALAT-nya tetap memakai
// penulis yang sama, supaya kegagalan mengunduh tetap berbentuk {kode, pesan} yang
// dikenali klien.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	source, ok := h.readSource(w, r)
	if !ok {
		return
	}

	companyCode, batchNumber, ok := h.readKey(w, r)
	if !ok {
		return
	}

	result, err := inboxautoclaim.ParseResult(r.URL.Query().Get("hasil"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	file, err := h.service.Export(r.Context(), active.Alias, inboxautoclaim.LineQuery{
		Source:      source,
		CompanyCode: companyCode,
		BatchNumber: batchNumber,
		Result:      result,
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	logging.From(r.Context(), h.logger).Info("berkas ekspor diterbitkan",
		slog.String("portal", active.Alias),
		slog.String("perusahaan", companyCode),
		slog.String("batch", batchNumber),
		slog.String("hasil", string(result)),
		slog.Int("baris", file.Rows),
	)

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	// Nama berkas ditulis dua kali: `filename` polos untuk peramban lama, dan
	// `filename*` berkode UTF-8 untuk yang mengikuti RFC 5987. Tanpa yang kedua, nama
	// berkas yang memuat spasi atau huruf beraksen rusak di sebagian peramban.
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment",
		map[string]string{"filename": file.FileName}))
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Content)))
	// Berkas ini memuat data nasabah. Ia tidak boleh tersimpan di cache peramban maupun
	// di proxy bersama.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(file.Content); err != nil {
		// Tidak ada yang dapat ditulis ke klien di sini — statusnya sudah terkirim.
		// Yang dapat dilakukan hanya mencatatnya supaya unduhan yang terputus terlihat.
		logging.From(r.Context(), h.logger).Warn("berkas ekspor gagal terkirim",
			slog.String("galat", err.Error()))
	}
}

// Upload menangani POST /inbox-auto-claim/unggah.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	caller, authenticated := h.caller(r.Context())
	if !authenticated {
		// Rute ini sudah berada di balik middleware Autentikasi, sehingga keadaan ini
		// berarti perakitan yang salah, bukan permintaan yang salah. Ia dijawab 500 lewat
		// penulis bersama, bukan 401 — memberi tahu pengguna "silakan masuk lagi" untuk
		// cacat perakitan hanya mengirimnya berputar-putar.
		h.writeError(w, r, errors.New("inboxautoclaim/http: konteks pemanggil tidak tersedia pada rute unggah"))
		return
	}

	// Portal diperiksa SEBELUM badan permintaan dibaca. Menarik berkas beberapa megabyte
	// dari jaringan hanya untuk menolaknya karena portalnya belum siap adalah pemborosan
	// yang terasa pengguna.
	if err := h.service.EnsurePortalReady(active.Alias); err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadRequest)
	if err := r.ParseMultipartForm(maxUploadRequest); err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Berkas tidak dapat dibaca. Pastikan ukurannya wajar dan formatnya CSV.",
		})
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			// Berkas sementara yang ditulis ParseMultipartForm ke disk dihapus. Tanpa
			// ini, setiap unggahan meninggalkan salinan data nasabah di direktori
			// sementara server.
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	source, ok := h.readSource(w, r)
	if !ok {
		return
	}

	file, header, err := r.FormFile(uploadFormField)
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: fmt.Sprintf("Berkas belum dilampirkan pada bagian %q.", uploadFormField),
		})
		return
	}
	defer func() { _ = file.Close() }()

	row, err := inboxautoclaim.ParseUpload(file)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	result, err := h.service.Upload(r.Context(), active.Alias, source, row, caller.Login)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// Nama berkas TIDAK dicatat ke log: ia dipilih pengguna dan dapat memuat nama
	// nasabah atau nomor polis. Yang dicatat hanya ukuran dan hasilnya.
	logging.From(r.Context(), h.logger).Info("unggahan auto claim tersimpan",
		slog.String("portal", active.Alias),
		slog.Int("baris", result.Rows),
		slog.Int("batch", len(result.Batch)),
		slog.Int("ditolak", len(result.Rejected)),
		slog.Int64("ukuran_berkas", header.Size),
	)

	batch := make([]UploadBatchDTO, 0, len(result.Batch))
	for _, b := range result.Batch {
		batch = append(batch, UploadBatchDTO{
			KodePerusahaan: b.CompanyCode,
			NamaPerusahaan: b.CompanyName,
			Batch:          b.BatchNumber,
			JumlahBaris:    b.Rows,
			JumlahLolos:    b.Succeeded,
			JumlahBertanda: b.Failed,
		})
	}

	// Baris yang ditolak dikirim beserta NOMOR BARISNYA di berkas.
	//
	// Tanpa nomor baris, pengguna dengan berkas 300 baris hanya tahu "ada yang gagal"
	// dan harus mencarinya sendiri. Nomor polisnya ikut karena itu yang dicarinya di
	// berkas asli — pesan saja tidak cukup untuk menemukan barisnya.
	rejected := make([]UploadRejectedDTO, 0, len(result.Rejected))
	for _, row := range result.Rejected {
		rejected = append(rejected, UploadRejectedDTO{
			Baris:      row.LineNumber,
			NomorPolis: row.PolicyNo,
			Pesan:      row.Message,
		})
	}

	h.writeResponse(w, r, http.StatusCreated, UploadResponse{
		Batch:       batch,
		JumlahBaris: result.Rows,
		Ditolak:     rejected,
		Portal:      active.Alias,
	})
}

// UploadTemplate menangani GET /inbox-auto-claim/format-unggahan.
//
// Rutenya TIDAK dipasangi pemeriksaan portal: bentuk berkas sama untuk setiap entitas,
// dan tidak satu baris data entitas pun dibacanya. Menuntut portal di sini akan membuat
// petunjuk format gagal justru saat pengguna belum memilih entitas.
func (h *Handler) UploadTemplate(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(w, r, http.StatusOK, UploadTemplateResponse{
		KolomWajib:    inboxautoclaim.RequiredUploadColumn,
		KolomOpsional: inboxautoclaim.OptionalUploadColumn,
		BatasBaris:    inboxautoclaim.MaxUploadRow,
	})
}

// readKey membaca kode perusahaan dan nomor batch dari jalur.
//
// Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readKey(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	companyCode := strings.TrimSpace(chi.URLParam(r, "kode"))
	batchNumber := strings.TrimSpace(chi.URLParam(r, "batch"))

	if companyCode == "" || batchNumber == "" {
		h.writeModuleError(w, r, inboxautoclaim.ErrBatchNotFound)
		return "", "", false
	}
	return companyCode, batchNumber, true
}

// readPage membaca nomor dan ukuran halaman dari parameter kueri.
//
// Nilai yang tidak dapat dibaca sebagai angka DIABAIKAN, bukan ditolak: PageRequest.Clean
// sudah memperbaiki nilai di luar batas, dan menolak `?halaman=abc` dengan galat hanya
// menampilkan layar rusak untuk kesalahan yang jelas maksudnya.
func readPage(r *http.Request) inboxautoclaim.PageRequest {
	return inboxautoclaim.PageRequest{
		Number: readNumber(r.URL.Query().Get("halaman"), 1),
		Size:   readNumber(r.URL.Query().Get("ukuran"), inboxautoclaim.DefaultPageSize),
	}
}

func readNumber(value string, fallback int) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return number
}

// Mount mendaftarkan rute modul Inbox Auto Claim.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// Middleware PortalAktif dipasang di sini, hanya pada rute yang menyentuh basis data
// entitas. Rute format unggahan sengaja berada di luarnya.
//
// # Urutan pendaftaran rute
//
// `/inbox-auto-claim/perusahaan` dan `/inbox-auto-claim/format-unggahan` didaftarkan
// SEBELUM `/inbox-auto-claim/{kode}/{batch}`. chi memang memilih pola yang lebih khusus
// lebih dulu, tetapi urutan penulisannya dibuat sejalan supaya orang yang membacanya
// tidak perlu mengetahui aturan itu untuk yakin rutenya benar.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// 10-API-STRATEGY.md §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Get("/inbox-auto-claim/format-unggahan", h.UploadTemplate)
	r.Get("/inbox-auto-claim/tab", h.ListTab)

	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox-auto-claim", h.ListBatch)
		perPortal.Get("/inbox-auto-claim/perusahaan", h.ListCompany)
		perPortal.Get("/inbox-auto-claim/ringkasan", h.Summarize)
		perPortal.Post("/inbox-auto-claim/unggah", h.Upload)
		perPortal.Get("/inbox-auto-claim/{kode}/{batch}", h.ListLine)
		perPortal.Get("/inbox-auto-claim/{kode}/{batch}/ekspor", h.Export)
	})
}
