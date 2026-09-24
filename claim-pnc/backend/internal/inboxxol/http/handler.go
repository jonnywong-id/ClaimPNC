package inboxxolhttp

import (
	"context"
	"encoding/csv"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxxol"
	"claim-pnc/internal/inboxxol/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk — `OperatorID.pyUserIdentifier`
	// di sistem lama, bukan NIK.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox XOL.
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

// NewHandler membentuk handler modul Inbox XOL.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// context menyiapkan portal aktif dan identitas pemanggil untuk satu permintaan.
//
// Keduanya diperiksa di SATU tempat, bukan diulang di tujuh handler: satu handler yang
// lupa memeriksa portal akan membaca basis data portal utama untuk pengguna entitas lain
// — kebocoran lintas badan hukum yang persis dicegah `R-20`, dan yang tidak menghasilkan
// satu pun pesan galat.
func (h *Handler) context(w http.ResponseWriter, r *http.Request) (string, inboxxol.Caller, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return "", inboxxol.Caller{}, false
	}

	if h.caller == nil {
		h.writeError(w, r, inboxxol.ErrCallerUnknown)
		return "", inboxxol.Caller{}, false
	}
	caller, known := h.caller(r.Context())
	if !known || strings.TrimSpace(caller.Login) == "" {
		h.writeError(w, r, inboxxol.ErrCallerUnknown)
		return "", inboxxol.Caller{}, false
	}

	return active.Alias, inboxxol.Caller{Login: caller.Login}.Clean(), true
}

// ListMasters menangani GET /inbox-xol/perjanjian.
//
// Melayani grid "PILIH MASTER XOL" pada modal tab 1 sekaligus menjadi sumber kurs bagi
// perhitungan grid utama.
func (h *Handler) ListMasters(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	masters, err := h.service.ListMasters(r.Context(), alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, MasterListResponse{Masters: toMasterDTOs(masters)})
}

// SummarizeClaims menangani GET /inbox-xol/klaim.
//
// Melayani grid "DATA XOL BASED ON DOL AND COL", pengganti defer-load `GetClaimXOL`.
func (h *Handler) SummarizeClaims(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	overview, err := h.service.SummarizeClaims(r.Context(), alias, caller,
		r.URL.Query().Get("id_master"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toClaimSummaryResponse(overview))
}

// Breakdown menangani GET /inbox-xol/klaim/rincian.
//
// Melayani `Sec_Detail_claim_XOL`, grid yang terbuka di balik satu baris grid utama.
func (h *Handler) Breakdown(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()
	rows, err := h.service.Breakdown(r.Context(), alias, caller,
		query.Get("id_master"), query.Get("tanggal_kejadian"), query.Get("sebab_kerugian"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toBreakdownResponse(rows))
}

// SearchAdvice menangani GET /inbox-xol/pla-dla.
//
// Melayani tombol "Generated DLA PLA XOL" dan "Cari Data DLA PLA XOL" sekaligus:
// keduanya menjalankan aktivitas yang sama dengan parameter yang sama
// (`Activity/BrowseDataXOLPLADLAGenerated-Act.xml`).
func (h *Handler) SearchAdvice(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	advices, err := h.service.SearchAdvice(r.Context(), alias, caller, adviceFilterFrom(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, AdviceListResponse{Advices: toAdviceDTOs(advices)})
}

// DownloadAdvice menangani GET /inbox-xol/pla-dla/unduh.
//
// # Ini pengganti sementara "Print Perhitungan", bukan padanannya
//
// Tombol lama menjalankan `DownloadPrinftPerhitunganPLADLAXOLSummary`, yang menyusun
// dokumen PLA/DLA lewat engine cetak Pega: 28 langkah, dua kueri layer dan reas, dan
// halaman JSON hasil perhitungan. `D-11` menetapkan dokumen dibuat sendiri di Go, dan
// Work Owner memutuskan 2026-09-20 bahwa tahap ini cukup mengeluarkan berkas tabel lebih
// dulu — tata letak resmi PLA/DLA menyusul.
//
// Yang diunduh karena itu ISI PERHITUNGANNYA, bukan dokumen resminya. Layar menyatakan
// pembedaan itu; berkas yang tampak seperti dokumen resmi padahal bukan adalah kekeliruan
// yang berakibat ke luar perusahaan.
//
// CSV dipilih, bukan Excel: ia dapat ditulis dengan pustaka standar, dapat dibuka Excel
// apa adanya, dan tidak menambah satu pun dependensi (`08-TECHNICAL-STRATEGY.md` §1).
func (h *Handler) DownloadAdvice(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	filter := adviceFilterFrom(r)
	advices, err := h.service.SearchAdvice(r.Context(), alias, caller, filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Header ditulis SEBELUM baris pertama, dan tidak ada satu pun galat sesudahnya:
	// setelah status 200 terkirim, kegagalan tidak lagi dapat disampaikan sebagai galat
	// HTTP — yang diterima pengguna hanyalah berkas yang terpotong.
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+adviceFileName(filter)+`"`)
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Judul kolom mengikuti grid layar, supaya berkas dan layar dapat dibandingkan
	// berdampingan tanpa menebak kolom mana yang mana.
	header := []string{
		"NO PLA / DLA", "Tipe", "Nama Insurance", "Nama Layer", "Tahun",
		"Sebab Kerugian", "Kurs (IDR)", "Share Percent", "Email", "Remark",
		"Status Persetujuan", "Tanggal Terbit",
	}
	if err := writer.Write(header); err != nil {
		h.logDownloadFailure(r, err)
		return
	}

	for _, advice := range advices {
		row := []string{
			advice.DisplayNumber(),
			string(advice.Type),
			advice.ReinsurerName,
			advice.LayerName,
			advice.Year,
			advice.CauseOfLoss,
			strconv.FormatFloat(advice.ExchangeRate, 'f', -1, 64),
			strconv.FormatFloat(advice.SharePercent, 'f', -1, 64),
			advice.Email,
			advice.Remark,
			advice.ApprovalStatus,
			advice.IssuedOn,
		}
		if err := writer.Write(row); err != nil {
			h.logDownloadFailure(r, err)
			return
		}
	}
}

// ListApprovals menangani GET /inbox-xol/persetujuan.
//
// Melayani tab "Inbox XOL Komite": dua antrean sekaligus, supaya keduanya berasal dari
// saat yang sama.
func (h *Handler) ListApprovals(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	queue, err := h.service.ListApprovals(r.Context(), alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toApprovalResponse(queue))
}

// ListCauseOfLoss menangani GET /inbox-xol/sebab-kerugian.
func (h *Handler) ListCauseOfLoss(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	causes, err := h.service.ListCauseOfLoss(r.Context(), alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toCauseOfLossResponse(causes))
}

// RejectWrite menjawab setiap aksi yang mengubah data.
//
// # Kenapa rutenya ADA padahal tidak mengerjakan apa pun
//
// Karena layar menggambar tombolnya. `Insert DOL dan COL`, `Simpan`, dan `Approval`
// tetap terlihat — menyembunyikannya akan membuat pengguna melaporkan fitur yang hilang,
// padahal ia memang belum dipindahkan.
//
// Yang dijawab bukan 404 melainkan penjelasan: kewenangan menulis tabel XOL masih ada di
// Pega selama masa paralel (`P-1`). Tanpa rute ini, penekanan tombol akan menghasilkan
// "halaman tidak ditemukan" — pesan yang menyesatkan pengguna DAN penelusur masalah.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, r, inboxxol.ErrWriteNotAvailable)
}

// adviceFilterFrom membaca penyaring pemberitahuan dari parameter query.
//
// Tipe yang tidak dikenal dibiarkan kosong, bukan dijatuhkan ke PLA: usecase menolaknya
// dengan pesan yang menyebut isiannya, dan diam-diam memilih satu tipe akan menampilkan
// daftar yang bukan diminta pengguna.
func adviceFilterFrom(r *http.Request) inboxxol.AdviceFilter {
	query := r.URL.Query()
	adviceType, _ := inboxxol.ParseAdviceType(query.Get("tipe"))
	return inboxxol.AdviceFilter{
		Year:        query.Get("tahun"),
		CauseOfLoss: query.Get("sebab_kerugian"),
		Type:        adviceType,
	}
}

// adviceFileName menyusun nama berkas unduhan.
//
// Seluruh karakter di luar huruf, angka, titik, dan garis diganti garis bawah. Nilainya
// berasal dari parameter query, dan nama berkas yang memuat kutip ganda atau baris baru
// dapat menyisipkan header sendiri ke dalam `Content-Disposition`.
func adviceFileName(filter inboxxol.AdviceFilter) string {
	parts := []string{"perhitungan", strings.ToLower(string(filter.Type)), filter.Year, filter.CauseOfLoss}
	name := strings.Join(parts, "-")

	var safe strings.Builder
	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z',
			char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9',
			char == '-', char == '.':
			safe.WriteRune(char)
		default:
			safe.WriteRune('_')
		}
	}
	return strings.Trim(safe.String(), "-_") + ".csv"
}

// logDownloadFailure mencatat kegagalan yang terjadi setelah badan respons mulai ditulis.
//
// Ia hanya dapat dicatat, tidak dapat dijawab: status 200 sudah terkirim. Yang diterima
// pengguna adalah berkas yang terpotong, dan satu-satunya jejaknya ada di log.
func (h *Handler) logDownloadFailure(r *http.Request, err error) {
	if h.logger == nil {
		return
	}
	h.logger.Error("unduhan perhitungan XOL terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()))
}
