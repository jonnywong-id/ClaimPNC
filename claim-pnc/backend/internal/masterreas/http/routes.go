package masterreashttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterreas/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// MaxKeywordLength membatasi panjang penyaring `cari`.
//
// Ia BUKAN aturan bisnis — sistem lama tidak punya kotak pencarian sama sekali di layar ini.
// Batasnya dipasang karena kata kunci masuk ke dalam pola LIKE, dan kata kunci sepanjang
// megabita hanya menghasilkan pemindaian penuh yang pasti tidak menemukan apa pun.
//
// Seratus dipilih agar sama dengan batas nama pada modul master lain: tidak ada nama
// perusahaan reasuransi, login, maupun surel yang lebih panjang dari itu, sehingga batas ini
// tidak pernah menolak pencarian yang sungguh-sungguh.
const MaxKeywordLength = 100

// Handler melayani permintaan master reas.
//
// TANPA CallerReader, berbeda dari modul master lain. Dua sebabnya, dan keduanya berasal
// dari modul ini yang hanya membaca: tidak ada baris yang diturunkan dari identitas
// pemanggil, dan tidak ada perubahan yang perlu dicatat pelakunya.
//
// Kewenangan membuka layar ini tetap ditegakkan — lewat middleware Autentikasi yang dipasang
// cmd, dan kelak lewat pemeriksaan peran `TKT-F3-005` yang belum ada.
type Handler struct {
	service       *usecase.Service
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service
	Logger  *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul master reas.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterreas/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New(
			"masterreas/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/reas.
//
// # Penyaringnya
//
//	?cari=...  mempersempit pada kode reas, nama reas, login, dan email
//
// TANPA penyaring status: tabelnya tidak punya kolom persetujuan, dan layar lamanya tidak
// bertab — harness-nya memuat satu grid dan satu tombol Refresh.
//
// TANPA penyaring TYPE, meski kolomnya ada. Sistem lama menyaringnya hanya di alur PLA/DLA,
// dengan nilai yang diambil dari nomor dokumen yang sedang dikirim — bukan dari pilihan
// pengguna. Menyediakannya di sini berarti menawarkan penyaring yang nilainya sahnya sendiri
// tidak diketahui (`R-16`).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	keyword := strings.TrimSpace(r.URL.Query().Get("cari"))
	if len(keyword) > MaxKeywordLength {
		// 400, bukan daftar kosong. Permintaan yang tidak dapat dipenuhi ditolak dengan
		// sebabnya — daftar kosong akan terbaca sebagai "tidak ada datanya", dan pengguna
		// tidak punya cara membedakan keduanya (`10-API-STRATEGY.md` §4).
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Kata pencarian terlalu panjang.",
		})
		return
	}

	list, err := h.service.List(r.Context(), active.Alias, keyword)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Member: toListDTO(list),
		Portal: active.Alias,
	})
}

// Mount mendaftarkan rute modul master reas.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// `POOLDATA.T_REINSURER` ada di basis data SETIAP entitas dan isinya berbeda — ia menentukan
// mitra reasuransi mana yang menerima pemberitahuan klaim badan hukum itu, beserta surel
// tujuannya. Tidak ada satu pun rute di modul ini yang isinya milik aplikasi, sehingga tidak
// ada yang dikecualikan.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan (`R-20`, `TKT-F6-002`).
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
//
// # Yang TIDAK didaftarkan, dan itu keputusan berdasar bukti
//
// **Tidak ada POST, PUT, maupun DELETE.** Layar lamanya tidak punya jalur tulis: harness
// `DataMemberReas` memuat satu grid dan satu tombol Refresh, dan satu-satunya penulis
// `T_REINSURER` di sistem lama adalah alur PLA/DLA lewat `Database/UPDATEREAS.prc` —
// dipanggil `UpdateDetailPLA2` dan `UpdateDetailDLA2`, bukan layar ini.
//
// **Tidak ada GET satu baris.** Tidak ada layar detail di sistem lama, seluruh kolomnya muat
// di dalam grid, dan kunci alaminya tiga kolom sehingga harus dipaksakan ke jalur URL.
//
// Bila kelak terbukti layar lamanya punya tombol simpan — section gridnya memang tidak ada
// di export (`R-16`) — yang perlu ditambahkan adalah Repo.Insert/Update beserta rutenya.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/reas", h.List)
	})
}
