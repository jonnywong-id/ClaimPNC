package inboxsurveyhttp

import (
	"net/http"
	"time"

	"claim-pnc/internal/inboxsurvey"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// systemClock membaca jam mesin.
//
// Ia satu-satunya tempat jam mesin dibaca pada modul ini, dan ia dapat DIGANTI lewat Options —
// itulah yang membuat kolom Aging dapat diuji tanpa bergantung hari saat uji dijalankan
// (`F-5`).
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

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

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
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
			Kind:     inboxsurvey.KPIKind(query.Get("jenis")),
			Category: query.Get("kategori"),
			Year:     query.Get("tahun"),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toKPIResponse(scored, active.Alias))
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
func nonNegativeNumber(raw string) int { return httpquery.NonNegative(raw) }

// jakarta mengembalikan zona WIB.
//
// Bila basis data zona waktu tidak tersedia di mesin — yang terjadi pada sebagian citra
// kontainer minimal — dipakai offset tetap +07:00. Indonesia bagian barat tidak mengenal
// daylight saving, sehingga offset tetap SETARA dan bukan penyederhanaan yang merugikan.
func jakarta() *time.Location { return clock.Jakarta() }
