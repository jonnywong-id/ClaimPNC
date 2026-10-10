package inboxosclaimpercabanghttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/inboxosclaimpercabang/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk.
	Login string

	// DetailBranchCode adalah kode cabang RINCI dari jawaban API HCQ saat masuk
	// (`EmpResponse.Placement.DetailBranchCode`), yang sudah dipetakan modul auth ke profil
	// pengguna.
	//
	// Ia BUKAN kode cabang yang dipakai klaim, dan itu sengaja dibawa apa adanya sampai ke
	// penyimpanan: terjemahannya satu pembacaan master cabang, dan menaruh terjemahan itu di
	// sini berarti lapisan transport memutuskan hal yang bukan urusannya. Lihat
	// inboxosclaimpercabang.Repo.BranchOf.
	//
	// Boleh kosong: `POOLDATA.M_LOGIN_PNC` tidak punya satu pun kolom cabang, sehingga pengguna
	// non-karyawan tidak membawanya. Yang terjadi kemudian bukan daftar kosong melainkan
	// pesan — lihat inboxosclaimpercabang.ErrBranchUnknown.
	DetailBranchCode string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox OS Claim per Cabang.
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

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan diimpor
	// dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa menariknya
	// serta.
	GetCaller CallerReader

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan galat
	// portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox OS Claim per Cabang.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// List menangani GET /api/inbox-os-claim-per-cabang.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		caller,
		inboxosclaimpercabang.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
		// Satu-satunya masukan pengguna yang sampai ke kueri layar ini. Ia dipangkas di
		// sini, di-escape di repositori, dan diikat sebagai parameter — tidak pernah
		// dirangkai ke dalam teks SQL.
		strings.TrimSpace(query.Get("cari")),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Ia dikumpulkan karena List dan Export menuntut keduanya dengan urutan yang sama, dan urutan
// itu penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal dijawab sebagai
// permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxosclaimpercabang.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxosclaimpercabang.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxosclaimpercabang.ErrCallerUnknown)
		return portal.Portal{}, inboxosclaimpercabang.Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
//
// Cabang yang kosong TIDAK membuat pemanggil dianggap tidak terbaca. Keduanya keadaan yang
// berbeda dan punya jawaban yang berbeda: yang pertama menyuruh pengguna masuk ulang, yang
// kedua menyuruhnya menghubungi Tim IT.
func (h *Handler) readCaller(r *http.Request) (inboxosclaimpercabang.Caller, bool) {
	if h.caller == nil {
		return inboxosclaimpercabang.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return inboxosclaimpercabang.Caller{}, false
	}
	return inboxosclaimpercabang.Caller{
		Login:            caller.Login,
		DetailBranchCode: caller.DetailBranchCode,
	}, true
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan Pagination.Normalize membetulkannya
// menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan membuat layar
// gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman pertama adalah
// jawaban yang selalu masuk akal.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}
