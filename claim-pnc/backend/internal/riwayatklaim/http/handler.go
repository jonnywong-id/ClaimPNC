package riwayatklaimhttp

import (
	"net/http"

	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
	"claim-pnc/internal/riwayatklaim"
)

// Open menangani POST /riwayat-klaim/buka.
//
// # Kenapa POST, bukan GET
//
// Karena ia MENGUBAH keadaan: satu jatah pencarian terpakai setiap kali layar dibuka,
// persis sistem lama. Menjadikannya GET berarti berjanji permintaannya aman diulang —
// janji yang tidak dipenuhi, dan yang akan dilanggar oleh hal-hal yang tidak terlihat
// seperti prefetch peramban atau percobaan ulang otomatis.
func (h *Handler) Open(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, riwayatklaim.ErrCallerUnknown)
		return
	}

	opened, err := h.Service.Open(r.Context(), active.Alias, caller)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, OpenResponse{
		SearchTypes: toSearchTypeListDTO(opened.SearchTypes),
		Access:      toAccessDTO(opened.Access),
		Portal:      active.Alias,
	})
}

// Search menangani GET /riwayat-klaim.
//
// Ia GET meski mencatat jejak audit: yang dicatat adalah LOG, bukan keadaan yang dilihat
// pengguna, dan jatah tidak berkurang karenanya. Mengulang permintaan yang sama
// menghasilkan hasil yang sama.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, riwayatklaim.ErrCallerUnknown)
		return
	}

	query := r.URL.Query()

	searchDate, readable := parseDate(query.Get("tanggal_pencarian"))
	if !readable {
		h.writeBadDate(w, r, riwayatklaim.FieldSearchDate)
		return
	}

	birthDate, readable := parseDate(query.Get("tanggal_lahir"))
	if !readable {
		h.writeBadDate(w, r, riwayatklaim.FieldBirthDate)
		return
	}

	found, err := h.Service.Search(
		r.Context(),
		active.Alias,
		caller,
		riwayatklaim.CriteriaInput{
			Type:       query.Get("tipe"),
			Text:       query.Get("nilai"),
			SearchDate: searchDate,
			BirthDate:  birthDate,
		},
		riwayatklaim.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toSearchResponse(found, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (riwayatklaim.Caller, bool) {
	if h.Caller == nil {
		return riwayatklaim.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
	if !exists || caller.Login == "" {
		return riwayatklaim.Caller{}, false
	}
	return riwayatklaim.Caller{Login: caller.Login}, true
}

// writeBadDate menjawab tanggal yang tidak dapat dibaca.
//
// 422 dengan penunjuk isian, bukan 400: bentuk permintaannya benar — parameternya ada dan
// bertipe teks — hanya isinya yang tidak dapat dibaca sebagai tanggal. Layar dapat
// menandai isian yang salah alih-alih menampilkan galat umum.
func (h *Handler) writeBadDate(w http.ResponseWriter, r *http.Request, field string) {
	h.WriteError(w, r, riwayatklaim.NewValidationError([]riwayatklaim.Violation{{
		Field:   field,
		Message: "Tanggal tidak dapat dibaca. Bentuknya YYYY-MM-DD.",
	}}))
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan riwayatklaim.Pagination.Normalize
// membetulkannya menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc`
// akan membuat layar gagal tanpa alasan yang terbaca pengguna — sementara menampilkan
// halaman pertama adalah jawaban yang selalu masuk akal.
func positiveNumber(raw string) int { return httpquery.NonNegative(raw) }
