package monitoringslinkojkhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/monitoringslinkojk/usecase"
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

// Handler melayani permintaan modul Monitoring SLINK OJK.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	logger     *slog.Logger
	writeJSON  JSONWriter
	writeError ErrorWriter

	// clock membaca waktu sekarang, dipakai mengisi `bulanlapor`.
	//
	// Fungsi, bukan pemanggilan `time.Now()` langsung: bulan lapor menentukan periode
	// laporan ke OJK, dan ia harus dapat diuji tanpa bergantung pada jam mesin penjalan
	// (`F-5`).
	clock func() time.Time
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

	// Now opsional; kosong berarti jam sistem. Lihat Handler.clock.
	Now func() time.Time
}

// NewHandler membentuk handler modul Monitoring SLINK OJK.
func NewHandler(o Options) *Handler {
	clock := o.Now
	if clock == nil {
		clock = time.Now
	}
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
		clock:      clock,
	}
}

// now membaca waktu sekarang dalam WIB.
//
// WIB, bukan UTC: `bulanlapor` berbentuk `YYYYMM` dan menentukan PERIODE laporan ke OJK.
// Pada malam tanggal 1 pukul 00.00–07.00 WIB, UTC masih menunjuk bulan sebelumnya — dan
// laporan yang masuk ke periode yang salah tidak menghasilkan satu pun galat.
func (h *Handler) now() time.Time {
	return h.clock().In(waktuIndonesiaBarat)
}

// waktuIndonesiaBarat adalah zona yang dipakai menurunkan bulan lapor.
//
// Ditetapkan sebagai offset tetap, bukan dibaca dari basis data zona waktu mesin: WIB
// tidak mengenal waktu musim panas, dan mesin penjalan di data center tidak dijamin
// membawa basis data zona waktu yang lengkap.
var waktuIndonesiaBarat = time.FixedZone("WIB", 7*60*60)

// context menyiapkan portal aktif dan identitas pemanggil untuk satu permintaan.
//
// Keduanya diperiksa di SATU tempat, bukan diulang di setiap handler: satu handler yang
// lupa memeriksa portal akan membaca basis data portal utama untuk pengguna entitas lain
// — kebocoran lintas badan hukum yang persis dicegah `R-20`, dan yang tidak menghasilkan
// satu pun pesan galat.
func (h *Handler) context(
	w http.ResponseWriter,
	r *http.Request,
) (string, monitoringslinkojk.Caller, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return "", monitoringslinkojk.Caller{}, false
	}

	if h.caller == nil {
		h.writeError(w, r, monitoringslinkojk.ErrCallerUnknown)
		return "", monitoringslinkojk.Caller{}, false
	}
	caller, known := h.caller(r.Context())
	if !known || strings.TrimSpace(caller.Login) == "" {
		h.writeError(w, r, monitoringslinkojk.ErrCallerUnknown)
		return "", monitoringslinkojk.Caller{}, false
	}

	return active.Alias, monitoringslinkojk.Caller{Login: caller.Login}.Clean(), true
}

// Describe menangani GET /monitoring-slink-ojk/keterangan.
//
// Melayani dropdown "Pilih Segmen" sekaligus kepala kedua tabel. Ia TIDAK menyentuh basis
// data — lihat usecase.Describe untuk alasannya — tetapi tetap menuntut sesi: susunan
// kolom laporan regulator bukan informasi publik.
func (h *Handler) Describe(w http.ResponseWriter, r *http.Request) {
	if _, _, ready := h.context(w, r); !ready {
		return
	}
	h.writeJSON(w, r, http.StatusOK, toDescribeResponse(h.service.Segments()))
}

// Search menangani GET /monitoring-slink-ojk/data — tombol "Cari Data".
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	request, err := readRequest(r, alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	info, err := h.service.Describe(request.Segment)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	page, err := h.service.Search(r.Context(), request)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK,
		toSearchResponse(info, page, request.Filter.Normalize()))
}

// RejectWrite menjawab ketiga tombol yang MENULIS di layar lama.
//
// Rutenya ada supaya tombolnya menjawab dengan ALASAN, bukan dengan "halaman tidak
// ditemukan". Sebab masing-masing disebut di Mount.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	if _, _, ready := h.context(w, r); !ready {
		return
	}
	h.writeError(w, r, monitoringslinkojk.ErrWriteNotAvailable)
}

// ============================================================================
// PEMBACAAN PENYARING
// ============================================================================

// readRequest membaca segmen dan penyaring dari query string.
//
// Seluruh pelanggaran bentuk dikumpulkan sekaligus, sama seperti pelanggaran aturan di
// usecase — pengguna yang salah mengisi dua isian tidak perlu mengirim dua kali (`P-5`).
func readRequest(
	r *http.Request,
	alias string,
	caller monitoringslinkojk.Caller,
) (usecase.Request, error) {
	query := r.URL.Query()

	segment, known := monitoringslinkojk.ParseSegment(query.Get("segmen"))
	if !known {
		return usecase.Request{}, monitoringslinkojk.ErrUnknownSegment
	}

	violations := make([]monitoringslinkojk.Violation, 0, 3)

	scope, valid := monitoringslinkojk.ParseBusinessScope(query.Get("business_name"))
	if !valid {
		violations = append(violations, monitoringslinkojk.Violation{
			Field:   monitoringslinkojk.FieldBusinessScope,
			Message: "Business Name harus AS. KREDIT atau SURETY BOND.",
		})
	}

	// Kedua nama parameter di bawah mempertahankan alias Pega atas permintaan Work Owner
	// (2026-09-26). `date_of_loss` adalah isian berlabel "Dari" dan
	// `date_of_request_document` adalah "Sampai"; keduanya menyaring TANGGAL REGISTRASI.
	// Lihat kepala paket monitoringslinkojk.
	from, err := readDate(query.Get("date_of_loss"))
	if err != nil {
		violations = append(violations, monitoringslinkojk.Violation{
			Field:   monitoringslinkojk.FieldDateOfLoss,
			Message: "Tanggal \"Dari\" harus berbentuk YYYY-MM-DD.",
		})
	}
	to, err := readDate(query.Get("date_of_request_document"))
	if err != nil {
		violations = append(violations, monitoringslinkojk.Violation{
			Field:   monitoringslinkojk.FieldDateOfRequestDocument,
			Message: "Tanggal \"Sampai\" harus berbentuk YYYY-MM-DD.",
		})
	}

	if err := monitoringslinkojk.NewValidationError(violations); err != nil {
		return usecase.Request{}, err
	}

	return usecase.Request{
		PortalAlias: alias,
		Caller:      caller,
		Segment:     segment,
		Filter: monitoringslinkojk.Filter{
			BusinessScope:         scope,
			GenerateType:          query.Get("tipe_generate"),
			DateOfLoss:            from,
			DateOfRequestDocument: to,
			Page:                  readInt(query.Get("halaman"), 1),
			Size:                  readInt(query.Get("ukuran"), monitoringslinkojk.DefaultPageSize),
		},
	}, nil
}

// tanggalMasukan adalah bentuk tanggal pada query string.
//
// `YYYY-MM-DD`, bukan `dd/mm/yyyy` seperti yang DITAMPILKAN. Keduanya sengaja berbeda:
// bentuk masukan tidak boleh bergantung pada kebiasaan penulisan tanggal, sedangkan
// bentuk tampilan harus mengikuti layar lama (`D-13`). Sistem lama memakai satu bentuk
// untuk keduanya, dan itulah sebab `to_date(…,'dd/mm/yyyy')` tersebar di kuerinya.
const tanggalMasukan = "2006-01-02"

// readDate membaca satu isian tanggal; kosong berarti tidak menyaring.
func readDate(raw string) (*time.Time, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation(tanggalMasukan, clean, time.UTC)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// readInt membaca satu bilangan; isian cacat jatuh ke nilai bawaan.
//
// Jatuh ke bawaan, bukan menjadi galat: nomor halaman yang cacat datang dari tautan yang
// salah salin, bukan dari kesalahan pengisian — dan menolaknya hanya membuat layar
// kosong tanpa memberi tahu apa pun yang berguna. Batas dan pembulatannya ditegakkan
// Filter.Normalize, bukan di sini.
func readInt(raw string, fallback int) int {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return fallback
	}
	value, err := strconv.Atoi(clean)
	if err != nil {
		return fallback
	}
	return value
}
