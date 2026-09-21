package masterrekeninghttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterrekening/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas orang yang mengirim permintaan.
//
// Ia sengaja tipe milik modul ini, bukan tipe milik modul auth: modul tidak saling
// mengimpor, dan yang dibutuhkan di sini hanyalah tiga field. Cara mengisinya diberikan
// saat perakitan di cmd/claimpnc lewat Options.Caller, sehingga modul ini tidak pernah
// tahu bagaimana sesi bekerja.
type Caller struct {
	Identity string
	Name     string
	Email    string
}

// Service adalah bagian usecase yang dibutuhkan handler ini.
//
// Dinyatakan sebagai antarmuka di paket yang MEMAKAInya, bukan di paket yang
// mengisinya (docs/Steering/08-TECHNICAL-STRATEGY.md §2) — itulah yang membuat handler
// dapat diuji tanpa membentuk seluruh layanan beserta repo dan seam Kasirnya.
type Service interface {
	List(ctx context.Context, f masterrekening.Filter) ([]masterrekening.Account, int, error)
	Get(ctx context.Context, k masterrekening.Key) (masterrekening.Account, error)
	ListBanks(ctx context.Context) ([]masterrekening.Bank, error)
	Submit(ctx context.Context, p usecase.Submission, oleh usecase.Submitter) (masterrekening.Account, error)
	Update(ctx context.Context, k masterrekening.Key, p usecase.Submission, oleh usecase.Submitter) (masterrekening.Account, error)
	Decide(ctx context.Context, k masterrekening.Key, decision usecase.Decision, oleh usecase.Committee, logger *slog.Logger) (masterrekening.Account, error)
}

// ServiceSelector memilih layanan milik satu portal entitas.
//
// # Kenapa satu layanan per portal, bukan satu layanan yang menerima alias
//
// Berbeda dari modul master lain, Service di sini memegang lebih dari sekadar
// penyimpanan: ada seam Kasir, seam Notifier, jam, dan `portalAlias` yang menentukan
// apakah rekening yang disetujui didaftarkan ke Kasir. Menambahkan parameter alias pada
// setiap method berarti setiap method harus merakit ulang keputusan itu.
//
// Satu layanan per portal membuat `portalAlias` tetap menjadi apa adanya — sifat layanan
// itu, bukan parameter yang dapat tertukar.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat; tidak
// pernah dialihkan ke portal utama sebagai cadangan (`R-20`).
type ServiceSelector func(portalAlias string) (Service, error)

// Handler melayani permintaan master rekening.
type Handler struct {
	serviceSelector ServiceSelector
	caller          func(context.Context) (Caller, bool)
	logger          *slog.Logger

	writeError func(w http.ResponseWriter, r *http.Request, err error)
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	// ServiceSelector memilih layanan milik satu portal entitas. Wajib.
	ServiceSelector ServiceSelector

	// Caller membaca identitas pemanggil dari context. Diisi saat perakitan
	// dengan pembaca konteks milik modul auth.
	Caller func(context.Context) (Caller, bool)

	Logger     *slog.Logger
	WriteError func(w http.ResponseWriter, r *http.Request, err error)
}

// NewHandler membentuk handler modul master rekening.
func NewHandler(o Options) *Handler {
	write := o.WriteError
	if write == nil {
		write = WriteError(o.Logger)
	}
	return &Handler{
		serviceSelector: o.ServiceSelector,
		caller:          o.Caller,
		logger:          o.Logger,
		writeError:      write,
	}
}

// serviceFor mengembalikan layanan milik entitas yang sedang aktif.
//
// Nilai kedua false berarti jawabannya sudah ditulis dan pemanggil harus berhenti. Dua
// sebab yang dibedakan: permintaan tidak melewati middleware portal sama sekali, dan
// portalnya disebut tetapi tidak dapat dilayani.
func (h *Handler) serviceFor(w http.ResponseWriter, r *http.Request) (Service, string, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return nil, "", false
	}

	service, err := h.serviceSelector(active.Alias)
	if err != nil {
		h.writeError(w, r, err)
		return nil, "", false
	}
	return service, active.Alias, true
}

// clientMaxLimit menahan permintaan halaman yang tidak masuk akal dari peramban.
const clientMaxLimit = 200

// List menangani GET /master-rekening.
//
// Lima tab layar lama — Cari Data, Komite Approval, Waiting Approval, Approve, dan
// Reject — seluruhnya dilayani endpoint ini dengan saringan yang berbeda. Membuat lima
// endpoint untuk lima tab berarti lima tempat yang harus diubah setiap kali kolomnya
// bertambah.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.serviceFor(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()

	status := masterrekening.ApprovalStatus(strings.TrimSpace(q.Get("status")))
	if status != "" && !status.Known() {
		h.writeError(w, r, masterrekening.ErrUnknownStatus)
		return
	}

	f := masterrekening.Filter{
		Status:    status,
		Number:    strings.TrimSpace(q.Get("nomor_rekening")),
		OwnerName: strings.TrimSpace(q.Get("nama_pemilik")),
		BankName:  strings.TrimSpace(q.Get("nama_bank")),
		Limit:     angka(q.Get("batas"), 50, clientMaxLimit),
		Offset:    angka(q.Get("lewati"), 0, 0),
	}

	// Tab "Komite Approval" hanya menampilkan yang menunggu keputusan komite yang
	// sedang masuk. Identitas komitenya diambil dari sesi, TIDAK dari query string —
	// kalau dari query string, siapa pun dapat melihat antrean komite mana pun.
	if q.Get("komite_saya") == "1" {
		caller, existing := h.identity(r)
		if !existing {
			h.writeError(w, r, errors.New("masterrekening/http: konteks pemanggil tidak ada"))
			return
		}
		f.MyCommitteeOnly = true
		f.CommitteeIdentity = caller.Identity
	}

	rows, total, err := service.List(r.Context(), f)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	content := make([]AccountDTO, 0, len(rows))
	for _, b := range rows {
		content = append(content, FromAccount(b))
	}
	WriteJSON(w, r, http.StatusOK, ListResponse{
		Account: content,
		Count:   total,
		Limit:   f.Limit,
		Offset:  f.Offset,
	}, h.logger)
}

// Get menangani GET /master-rekening/{kodeBank}/{nomorRekening}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.serviceFor(w, r)
	if !ok {
		return
	}

	acct, err := service.Get(r.Context(), keyFromPath(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	WriteJSON(w, r, http.StatusOK, FromAccount(acct), h.logger)
}

// ListBanks menangani GET /master-rekening/bank.
func (h *Handler) ListBanks(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.serviceFor(w, r)
	if !ok {
		return
	}

	bank, err := service.ListBanks(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	content := make([]BankDTO, 0, len(bank))
	for _, b := range bank {
		content = append(content, BankDTO{Code: b.Code, Name: b.Name})
	}
	WriteJSON(w, r, http.StatusOK, BankListResponse{Bank: content}, h.logger)
}

// Submit menangani POST /master-rekening.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.serviceFor(w, r)
	if !ok {
		return
	}

	var body SaveRequest
	if !h.readBody(w, r, &body) {
		return
	}
	caller, existing := h.identity(r)
	if !existing {
		h.writeError(w, r, errors.New("masterrekening/http: konteks pemanggil tidak ada"))
		return
	}

	acct, err := service.Submit(r.Context(), submissionFrom(body), usecase.Submitter{
		Identity: caller.Identity,
		Name:     caller.Name,
		Email:    caller.Email,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	WriteJSON(w, r, http.StatusCreated, FromAccount(acct), h.logger)
}

// Update menangani PUT /master-rekening/{kodeBank}/{nomorRekening}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.serviceFor(w, r)
	if !ok {
		return
	}

	var body SaveRequest
	if !h.readBody(w, r, &body) {
		return
	}
	caller, existing := h.identity(r)
	if !existing {
		h.writeError(w, r, errors.New("masterrekening/http: konteks pemanggil tidak ada"))
		return
	}

	acct, err := service.Update(r.Context(), keyFromPath(r), submissionFrom(body), usecase.Submitter{
		Identity: caller.Identity,
		Name:     caller.Name,
		Email:    caller.Email,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	WriteJSON(w, r, http.StatusOK, FromAccount(acct), h.logger)
}

// Decide menangani POST /master-rekening/{kodeBank}/{nomorRekening}/keputusan.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.serviceFor(w, r)
	if !ok {
		return
	}

	var body DecideRequest
	if !h.readBody(w, r, &body) {
		return
	}
	caller, existing := h.identity(r)
	if !existing {
		h.writeError(w, r, errors.New("masterrekening/http: konteks pemanggil tidak ada"))
		return
	}

	acct, err := service.Decide(r.Context(), keyFromPath(r), usecase.Decision{
		Status:     masterrekening.ApprovalStatus(strings.TrimSpace(body.Status)),
		Note:       body.Note,
		DocumentID: body.DocumentID,
	}, usecase.Committee{
		Identity: caller.Identity,
		Name:     caller.Name,
		Email:    caller.Email,
	}, h.logger)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	WriteJSON(w, r, http.StatusOK, FromAccount(acct), h.logger)
}

func (h *Handler) identity(r *http.Request) (Caller, bool) {
	if h.caller == nil {
		return Caller{}, false
	}
	return h.caller(r.Context())
}

func (h *Handler) readBody(w http.ResponseWriter, r *http.Request, to any) bool {
	reader := json.NewDecoder(r.Body)
	// Field yang tidak dikenal ditolak, bukan diabaikan diam-diam: klien yang salah
	// mengeja nama field harus tahu isiannya tidak sampai, bukan menemukan kolomnya
	// kosong berminggu-minggu kemudian.
	reader.DisallowUnknownFields()

	if err := reader.Decode(to); err != nil {
		WriteJSON(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		}, h.logger)
		return false
	}
	return true
}

func submissionFrom(b SaveRequest) usecase.Submission {
	return usecase.Submission{
		Number:            b.Number,
		OwnerName:         b.OwnerName,
		BankName:          b.BankName,
		BankBranch:        b.BankBranch,
		BankAddress:       b.BankAddress,
		BankCode:          b.BankCode,
		AccountType:       b.AccountType,
		Email:             b.Email,
		Phone:             b.Phone,
		NIK:               b.NIK,
		DocumentID:        b.DocumentID,
		Note:              b.Note,
		Active:            b.Active,
		PreviousBankCode:  b.PreviousBankCode,
		PreviousNumber:    b.PreviousNumber,
		PreviousOwnerName: b.PreviousOwnerName,
	}
}

func keyFromPath(r *http.Request) masterrekening.Key {
	return masterrekening.Key{
		BankCode: strings.TrimSpace(chi.URLParam(r, "kodeBank")),
		Number:   strings.TrimSpace(chi.URLParam(r, "nomorRekening")),
	}
}

// angka membaca bilangan dari query string dengan nilai baku dan batas atas.
//
// Masukan yang tidak dapat dibaca jatuh ke nilai baku alih-alih menjadi galat: sebuah
// nomor halaman yang salah ketik tidak sebanding dengan menolak seluruh permintaan.
func angka(text string, fallback, tertinggi int) int {
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || value < 0 {
		return fallback
	}
	if tertinggi > 0 && value > tertinggi {
		return tertinggi
	}
	return value
}
