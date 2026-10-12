package inboxanalystdoctorhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"claim-pnc/internal/inboxanalystdoctor"
	"claim-pnc/internal/inboxanalystdoctor/usecase"
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
	// Itulah yang dicocokkan ke `PC_ASSIGN_WORKLIST.PXASSIGNEDOPERATORID`. Memakai NIK di
	// sini akan membuat antrean tampak kosong bagi setiap pengguna.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Analyst Doctor.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	clock      inboxanalystdoctor.Clock
	location   *time.Location
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan diimpor
	// dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa menariknya
	// serta.
	GetCaller CallerReader

	// Clock adalah seam waktu (`F-5`). Wajib: kolom "Lama Waktu Klaim" dihitung darinya.
	Clock inboxanalystdoctor.Clock

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	Location *time.Location

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan galat
	// portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox Analyst Doctor.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}

	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		clock:      o.Clock,
		location:   location,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-analyst-doctor/keterangan.
//
// Ia GET dan tidak mengubah apa pun: judul kolom, selisih terencana, dan keterbatasan adalah
// bentuk layar, bukan data entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-analyst-doctor.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxanalystdoctor.ErrCallerUnknown)
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		caller,
		inboxanalystdoctor.Filter{
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
		toListResponse(listed, active.Alias, h.now(), h.location))
}

// now membaca waktu lewat seam, bukan dari jam sistem.
//
// Cadangan `time.Now()` ada supaya perakit yang lupa mengisi Clock tidak menjatuhkan seluruh
// layar. Ia TIDAK menyembunyikan kelalaian itu: kolom "Lama Waktu Klaim" tetap benar, dan
// yang hilang hanyalah kemampuan mengujinya secara deterministik.
func (h *Handler) now() time.Time {
	if h.clock == nil {
		return time.Now()
	}
	return h.clock.Now()
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxanalystdoctor.Caller, bool) {
	if h.caller == nil {
		return inboxanalystdoctor.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || caller.Login == "" {
		return inboxanalystdoctor.Caller{}, false
	}
	return inboxanalystdoctor.Caller{Login: caller.Login}, true
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
