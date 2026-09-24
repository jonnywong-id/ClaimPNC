package inboxoutstandinghttp

import (
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxoutstanding/usecase"
	"claim-pnc/internal/platform/logging"

	portalhttp "claim-pnc/internal/portal/http"
)

// Service adalah bagian usecase yang dipakai handler ini.
//
// Dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor sebagai tipe konkret,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, q usecase.Query) (usecase.Result, error)
}

// Caller adalah identitas pengguna yang sedang masuk, sejauh yang dibutuhkan modul ini.
//
// Hanya satu field: modul ini tidak perlu tahu apa pun tentang bentuk sesi, dan modul auth
// tidak perlu tahu modul ini ada. Jembatannya dipasang di cmd/claimpnc, satu-satunya
// berkas yang memang tahu keduanya.
type Caller struct {
	Login string
}

// GetCaller membaca identitas pengguna dari konteks permintaan.
type GetCaller func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan Inbox Outstanding.
type Handler struct {
	service     Service
	getCaller   GetCaller
	logger      *slog.Logger
	writeJSON   JSONWriter
	writeErrorF ErrorWriter
	location    *time.Location
	now         func() time.Time
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service   Service
	GetCaller GetCaller
	Logger    *slog.Logger

	WriteJSON           JSONWriter
	FallbackErrorWriter ErrorWriter

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	//
	// Ia parameter, bukan konstanta, supaya uji dapat menetapkannya dan tidak bergantung
	// pada basis data zona waktu mesin yang menjalankan.
	Location *time.Location

	// Now dapat diisi uji supaya kolom umur klaim dapat diperiksa secara deterministik.
	Now func() time.Time
}

// NewHandler membentuk handler modul Inbox Outstanding.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}

	return &Handler{
		service:     o.Service,
		getCaller:   o.GetCaller,
		logger:      o.Logger,
		writeJSON:   o.WriteJSON,
		writeErrorF: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
		location:    location,
		now:         now,
	}
}

// jakarta mengembalikan zona WIB.
//
// Bila basis data zona waktu tidak tersedia di mesin — yang terjadi pada sebagian citra
// kontainer minimal — dipakai offset tetap +07:00. Indonesia bagian barat tidak mengenal
// daylight saving, sehingga offset tetap SETARA dan bukan penyederhanaan yang merugikan.
func jakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// List menangani GET /api/inbox-outstanding.
//
// Menggantikan `RDB List/BrowseInboxOutstanding1-SQL.xml` beserta
// `Activity/InboxOutstanding_Act-Act.xml`. Yang di sana menjadi penyisipan potongan SQL
// per jabatan, di sini menjadi satu kueri berparameter dengan batas data yang diturunkan
// di server.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q, err := h.queryFrom(r)
	if err != nil {
		writeBadRequest(h.writeJSON, w, r, err.Error())
		return
	}

	result, err := h.service.List(r.Context(), q)
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}
	h.logLineLookupFailure(r, result)

	now := h.now()
	claims := make([]claimDTO, 0, len(result.Page.Claims))
	for _, c := range result.Page.Claims {
		claims = append(claims, toClaimDTO(c, now, h.location))
	}

	h.writeJSON(w, r, http.StatusOK, listResponse{
		Klaim: claims,
		Total: result.Page.Total,
		BatasLini: scopeDTO{
			TanpaBatas: result.Scope.Unrestricted,
			GroupPanel: result.Scope.GroupPanels,
		},
	})
}

// exportBatchSize adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Export ditulis SAMBIL dibaca, sekumpulan baris pada satu waktu, sehingga memori tetap
// datar berapa pun jumlah barisnya (`15-NFR` §3.2 butir 6). Memuat seluruh hasil lebih
// dulu akan membuat satu permintaan export menahan memori sebesar hasilnya.
const exportBatchSize = 500

// exportMaxRows membatasi banyaknya baris yang diekspor.
//
// # Kenapa ada batas, dan kenapa angkanya bukan warisan
//
// Sistem lama membatasi 500 baris lewat `pyMaxRecords` pada 54 dari 56 laporannya,
// sehingga kebutuhan export bervolume besar BELUM PERNAH benar-benar dilayani dan tidak
// ada data historis yang sahih untuk menentukan angkanya (`15-NFR` §3.2).
//
// Angka di bawah karena itu adalah batas pengaman yang dipilih sadar, bukan peniruan:
// cukup longgar untuk melampaui batas 500 yang selama ini membatasi, cukup ketat untuk
// menolak permintaan yang akan menahan koneksi basis data terlalu lama.
//
// Berapa baris maksimum yang WAJIB dilayani adalah pertanyaan terbuka `ADR-0011`.
const exportMaxRows = 10000

// Export menangani GET /api/inbox-outstanding/unduh.
//
// # Kenapa CSV, bukan XLSX
//
// Tombol di layar lama berbunyi "Export Excel", tetapi yang dipanggilnya adalah
// `pxConvertResultsToCSV` (`Activity/ExportOutstanding_Act-Act.xml`, tiga kemunculan) —
// keluarannya CSV. Memakai `encoding/csv` dari pustaka standar karena itu SETARA dengan
// sistem lama, bukan penyederhanaan, dan tidak menambah satu pun dependensi.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	q, err := h.queryFrom(r)
	if err != nil {
		writeBadRequest(h.writeJSON, w, r, err.Error())
		return
	}

	// Halaman diabaikan saat mengekspor: yang diminta adalah seluruh hasil yang cocok,
	// bukan halaman yang sedang dilihat.
	q.Offset = 0
	q.Limit = exportBatchSize

	first, err := h.service.List(r.Context(), q)
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}
	h.logLineLookupFailure(r, first)

	// Header ditulis SEBELUM baris pertama dikirim. Setelah badan respons mulai mengalir,
	// status HTTP tidak dapat diubah lagi — sehingga galat yang terjadi di tengah tidak
	// dapat dijawab dengan 500. Itu diterima, dan alasannya ada di bawah.
	filename := "inbox-outstanding-" + h.now().In(h.location).Format("20060102-150405") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(exportHeader()); err != nil {
		h.logExportInterrupted(r, err)
		return
	}

	now := h.now()
	written := 0
	page := first

	for {
		for _, c := range page.Page.Claims {
			if written >= exportMaxRows {
				return
			}
			if err := writer.Write(exportRow(toClaimDTO(c, now, h.location))); err != nil {
				// Sambungan putus di tengah unduhan adalah kejadian biasa — pengguna
				// menutup tab. Ia dicatat sebagai peringatan, bukan galat, dan tidak dapat
				// diberitahukan ke klien karena badan respons sudah mengalir.
				h.logExportInterrupted(r, err)
				return
			}
			written++
		}

		// Baris yang diterima lebih sedikit dari yang diminta berarti sudah habis.
		if len(page.Page.Claims) < q.Limit || written >= page.Page.Total || written >= exportMaxRows {
			return
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			h.logExportInterrupted(r, err)
			return
		}

		q.Offset = written
		page, err = h.service.List(r.Context(), q)
		if err != nil {
			h.logExportInterrupted(r, err)
			return
		}
	}
}

// exportHeader adalah judul kolom CSV.
//
// Urutannya mengikuti kolom layar, supaya berkas yang diunduh terbaca sebagai salinan apa
// yang dilihat pengguna — bukan susunan lain yang harus dicocokkan sendiri.
// exportHeader adalah judul kolom CSV.
//
// Judulnya SAMA PERSIS dengan kolom layar, dan urutannya pun sama — berkas yang diunduh
// harus terbaca sebagai salinan apa yang dilihat pengguna, bukan susunan lain yang harus
// dicocokkan sendiri. Teksnya berbahasa Inggris karena begitulah layar lama menulisnya
// (`D-13`).
func exportHeader() []string {
	return []string{
		"Claim no", "Policy no", "Insured name", "Business Name", "Business source",
		"Branch name", "Admin name", "Register Date", "Date of loss",
		"Total Aging", "Aging", "Claim status", "Status ASM", "ASM PIC",
	}
}

func exportRow(c claimDTO) []string {
	return []string{
		c.NomorKlaim, c.NomorPolis, c.Tertanggung, c.NamaBisnis, c.SumberBisnis,
		c.NamaCabang, c.AdminPNC, c.TanggalPendaftaran, c.TanggalKejadian,
		strconv.Itoa(c.UmurHari), optionalInt(c.AgingHari),
		c.StatusTampil, c.StatusKlaim, c.PICTeknik,
	}
}

// optionalInt menuliskan angka, atau sel KOSONG bila nilainya belum terisi.
//
// Sel kosong, bukan "0": pada berkas yang dibuka di pengolah angka, nol adalah angka yang
// ikut terhitung dalam rata-rata dan penjumlahan. "Belum diisi" bukan nol.
func optionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

// queryFrom membaca penyaring dan paginasi dari query string.
//
// Identitas pemanggil diambil dari KONTEKS, tidak pernah dari badan permintaan maupun
// query string — batas data yang dapat diminta klien bukan batas data.
func (h *Handler) queryFrom(r *http.Request) (usecase.Query, error) {
	caller, found := h.getCaller(r.Context())
	if !found || strings.TrimSpace(caller.Login) == "" {
		return usecase.Query{}, fmt.Errorf("identitas pemanggil tidak dikenali")
	}

	// Portal aktif sudah diperiksa middleware; ketiadaannya di sini berarti rute dipasang
	// di luar middleware itu — cacat perakitan, bukan kesalahan pengguna. Ia ditolak,
	// tidak pernah dilayani portal utama sebagai cadangan (`R-20`).
	activePortal, portalFound := portalhttp.ActivePortalFrom(r.Context())
	if !portalFound {
		return usecase.Query{}, fmt.Errorf("portal aktif tidak dikenali")
	}

	q := r.URL.Query()

	limit, err := positiveInt(q.Get("batas"), inboxoutstanding.DefaultLimit)
	if err != nil {
		return usecase.Query{}, fmt.Errorf("parameter batas tidak sah")
	}
	if limit > inboxoutstanding.MaxLimit {
		// Ditolak, bukan dipangkas diam-diam (`10-API-STRATEGY.md` §4): klien yang meminta
		// seribu baris lalu menerima seratus tanpa diberi tahu akan menampilkan daftar yang
		// ia kira lengkap.
		return usecase.Query{}, fmt.Errorf(
			"parameter batas melebihi maksimum %d", inboxoutstanding.MaxLimit)
	}

	offset, err := positiveInt(q.Get("lewati"), 0)
	if err != nil {
		return usecase.Query{}, fmt.Errorf("parameter lewati tidak sah")
	}

	return usecase.Query{
		LoginID:     caller.Login,
		PortalAlias: activePortal.Alias,
		Search:      strings.TrimSpace(q.Get("cari")),
		Stage:       strings.TrimSpace(q.Get("tahap")),
		BranchCode:  strings.TrimSpace(q.Get("cabang")),
		Limit:       limit,
		Offset:      offset,
	}, nil
}

// positiveInt membaca bilangan bulat tak negatif; kosong berarti nilai baku.
func positiveInt(raw string, fallback int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("bukan bilangan bulat tak negatif")
	}
	return value, nil
}

// logLineLookupFailure mencatat kegagalan membaca lini bisnis.
//
// Ia WAJIB dipanggil pada setiap jalur yang memakai Result. Kegagalan itu menghasilkan
// batas data "tanpa batas" — sama dengan pengguna yang memang belum punya lini — sehingga
// tanpa catatan ini, kegagalan basis data menjadi tidak terlihat oleh siapa pun.
func (h *Handler) logLineLookupFailure(r *http.Request, result usecase.Result) {
	if result.LineLookupError == nil {
		return
	}
	logging.From(r.Context(), h.logger).Warn("lini bisnis tidak dapat dibaca; batas data tidak berlaku",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", result.LineLookupError.Error()),
	)
}

// logExportInterrupted mencatat export yang berhenti di tengah.
func (h *Handler) logExportInterrupted(r *http.Request, err error) {
	logging.From(r.Context(), h.logger).Warn("export berhenti sebelum selesai",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
