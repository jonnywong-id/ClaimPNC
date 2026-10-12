package inboxsurveyhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"claim-pnc/internal/inboxsurvey"
	"claim-pnc/internal/inboxsurvey/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Itulah yang dicocokkan ke `POOLDATA.MST_LOGIN_SURVEYOR.LOGIN` — pemetaan yang
	// `RDB List/GetLoginLeaderSurveyor-SQL.xml` pakai persis begitu. Memakai NIK di sini akan
	// membuat setiap pengguna terbaca sebagai bukan surveyor.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul My Work.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	clock      inboxsurvey.Clock
	location   *time.Location
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// systemClock membaca jam mesin.
//
// Ia satu-satunya tempat jam mesin dibaca pada modul ini, dan ia dapat DIGANTI lewat Options —
// itulah yang membuat kolom Aging dapat diuji tanpa bergantung hari saat uji dijalankan
// (`F-5`).
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan diimpor
	// dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa menariknya
	// serta.
	GetCaller CallerReader

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	Location *time.Location

	// Clock adalah seam ke waktu. Kosong berarti jam mesin.
	//
	// Ia dibutuhkan kolom Aging, yang DIHITUNG dari tanggal janji survei dicatat terhadap
	// tanggal WIB hari ini — bukan dibaca dari kolom.
	Clock inboxsurvey.Clock

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan galat
	// portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul My Work.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}

	clock := o.Clock
	if clock == nil {
		clock = systemClock{}
	}

	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		clock:      clock,
		location:   location,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-survey/keterangan.
//
// Ia GET dan tidak mengubah apa pun: judul kolom, judul tab, selisih terencana, dan
// keterbatasan adalah bentuk layar, bukan data entitas.
//
// Ia TIDAK menuntut identitas surveyor. Bentuk layar sama bagi siapa pun, dan menolaknya bagi
// pengguna yang belum terdaftar akan membuat layar tidak dapat menampilkan pesan yang
// menjelaskan KENAPA ia belum terdaftar.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Keterangan layar dibaca PER PORTAL sejak 2026-10-07: ketersediaan tab bergantung pada
	// kolom yang ada dan terisi di basis data portal itu, bukan pada konstanta di kode.
	meta := h.service.Metadata(r.Context(), active.Alias)

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(meta, active.Alias))
}

// List menangani GET /api/inbox-survey.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ok := h.begin(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		caller,
		inboxsurvey.Filter{
			Tab:    inboxsurvey.Tab(query.Get("tab")),
			Search: query.Get("cari"),
			Offset: nonNegativeNumber(query.Get("lewati")),
			Limit:  nonNegativeNumber(query.Get("batas")),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK,
		toListResponse(listed, active.Alias, h.clock.Now(), h.location))
}

// Counts menangani GET /api/inbox-survey/jumlah-tab.
//
// Rute tersendiri, bukan bagian jawaban daftar: bilah tab tidak berubah saat pengguna
// berpindah halaman, sehingga menempelkannya pada setiap permintaan daftar akan menjalankan
// tujuh penjumlahan setiap kali tombol halaman ditekan.
func (h *Handler) Counts(w http.ResponseWriter, r *http.Request) {
	active, caller, ok := h.begin(w, r)
	if !ok {
		return
	}

	counted, err := h.service.Counts(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toCountResponse(counted, active.Alias))
}

// KPI menangani GET /api/inbox-survey/kpi.
func (h *Handler) KPI(w http.ResponseWriter, r *http.Request) {
	active, caller, ok := h.begin(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	scored, err := h.service.KPI(
		r.Context(),
		active.Alias,
		caller,
		inboxsurvey.KPIFilter{
			Status:  inboxsurvey.SurveyStatus(query.Get("status_survei")),
			Report:  inboxsurvey.ReportType(query.Get("tipe_report")),
			Quarter: query.Get("kuartal"),
			Year:    query.Get("tahun"),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toKPIResponse(scored, active.Alias))
}

// KPIYears menangani GET /api/inbox-survey/kpi/tahun — isi dropdown "Tahun Kuartal".
//
// Rute TERSENDIRI, bukan bagian /keterangan: isinya bergantung pada identitas pemanggil,
// sedangkan /keterangan tidak menyentuh basis data sama sekali dan di-cache selamanya oleh
// layar. Menyatukannya akan membuat daftar tahun milik pengguna sebelumnya ikut terbawa.
func (h *Handler) KPIYears(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.begin(w, r)
	if !ready {
		return
	}

	years, err := h.service.KPIYears(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, TahunKPIResponse{Portal: active.Alias, Tahun: years})
}

// begin menjalankan dua pemeriksaan yang sama bagi ketiga rute berdata.
//
// Ia mengembalikan false bila sudah menuliskan galatnya sendiri, sehingga pemanggil cukup
// berhenti. Bentuk itu dipilih supaya urutan pemeriksaan TIDAK dapat berbeda antar rute:
// portal lebih dulu, baru identitas. Urutan yang berbeda menghasilkan galat berbeda untuk
// keadaan yang sama, dan layar tidak dapat menjelaskan apa pun kepada penggunanya.
func (h *Handler) begin(
	w http.ResponseWriter,
	r *http.Request,
) (portal.Portal, inboxsurvey.Caller, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxsurvey.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxsurvey.ErrCallerUnknown)
		return portal.Portal{}, inboxsurvey.Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxsurvey.Caller, bool) {
	if h.caller == nil {
		return inboxsurvey.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || caller.Login == "" {
		return inboxsurvey.Caller{}, false
	}
	return inboxsurvey.Caller{Login: caller.Login}, true
}

// nonNegativeNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan Filter.Normalize membetulkannya menjadi
// nilai bawaan. Menolak seluruh permintaan karena `batas=abc` akan membuat layar gagal tanpa
// alasan yang terbaca pengguna — sementara menampilkan halaman pertama adalah jawaban yang
// selalu masuk akal.
func nonNegativeNumber(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
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
