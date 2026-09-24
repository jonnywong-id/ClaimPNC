package inboxkomunikasicabanghttp

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/inboxkomunikasicabang/usecase"
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
	// Ia yang diterjemahkan menjadi kode cabang, dan karena itu MENENTUKAN apa yang
	// terlihat — bukan sekadar mengisi jejak log seperti di sebagian modul inbox lain.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Komunikasi Cabang.
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

// NewHandler membentuk handler modul Inbox Komunikasi Cabang.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-komunikasi-cabang/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab, kolom, dan selisih terencana adalah bentuk
// layar, bukan data entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-komunikasi-cabang.
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
		readFilter(query),
		inboxkomunikasicabang.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
}

// Detail menangani GET /api/inbox-komunikasi-cabang/komunikasi/{komunikasi}.
//
// Ia layar "Detail Komunikasi" — yang di Pega terbuka lewat flow action
// `DETAILKOMUNIKASICABANG_11` saat tombolnya ditekan.
//
// Nomornya diambil dari JALUR, bukan dari parameter query, karena ia mengidentifikasi sumber
// daya — bukan menyaringnya (`10-API-STRATEGY.md` §2). Ia di-decode chi lebih dulu, sehingga
// nomor yang memuat karakter khusus sampai utuh.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	detailed, err := h.service.Detail(
		r.Context(),
		active.Alias,
		caller,
		inboxkomunikasicabang.DetailInput{ID: chi.URLParam(r, "komunikasi")},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Batas cabang datang BERSAMA hasilnya, bukan diminta ulang.
	//
	// Layar menggambar keterangan batasnya dari sini, dan keterangan itu menyatakan batas
	// PEMANGGIL — bukan asal percakapan yang kebetulan terbuka. Keduanya berbeda: percakapan
	// dari pusat ke cabang punya asal "PUSAT" meski yang membukanya petugas cabang.
	//
	// Meminta ulang akan menembus DB Link dua kali untuk satu permintaan; lihat
	// `usecase.Detailed`.
	h.writeJSON(w, r, http.StatusOK,
		toConversationDetailResponse(detailed.Detail, detailed.Branch, active.Alias))
}

// RejectWrite menjawab aksi tulis yang belum tersedia.
//
// Ia sengaja BUKAN 404. Layar lama punya EMPAT tindakan yang menulis — "Kirim Pesan",
// "Balas", "Selesai Komunikasi", dan "Tambah" — dan tindakan yang dijawab "halaman tidak
// ditemukan" terbaca sebagai kerusakan, sementara yang dibutuhkan pengguna adalah tahu ke
// mana ia harus pergi.
//
// Salah satunya bahkan TIDAK DAPAT dibangun sekalipun diputuskan: activity di balik tombol
// "Balas" (`PNCReplyMessageCabang`) tidak ada di export mana pun, sehingga tidak ada yang
// dapat dibaca untuk ditulis ulang.
//
// Portal tetap diperiksa lebih dulu meski permintaannya pasti ditolak: jawaban yang menyebut
// portal aktif untuk permintaan yang tidak menyebut portal akan membuat layar mengira ia
// sudah berada di portal yang benar.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	if _, exists := portalhttp.ActivePortalFrom(r.Context()); !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Dicatat, bukan hanya ditolak. Selama masa paralel, inilah satu-satunya tanda seberapa
	// sering pengguna benar-benar membutuhkan aksi ini — dan itu yang menjadi dasar
	// memutuskan kapan kepemilikan tabelnya dipindahkan (`P-1`).
	if h.logger != nil {
		h.logger.Info(
			"aksi tulis diminta pada modul yang belum menulis",
			slog.String("modul", "inbox-komunikasi-cabang"),
			slog.String("jalur", r.URL.Path),
			slog.String("tindakan", strings.TrimSpace(r.URL.Query().Get("tindakan"))),
		)
	}

	h.writeError(w, r, inboxkomunikasicabang.ErrWriteNotAvailable)
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Urutannya penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal dijawab
// sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxkomunikasicabang.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxkomunikasicabang.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxkomunikasicabang.ErrCallerUnknown)
		return portal.Portal{}, inboxkomunikasicabang.Caller{}, false
	}

	return active, caller, true
}

// readFilter membaca isian penyaring grid dari parameter query.
//
// Hanya satu: tab mana yang diminta. Kotak "Filter" layar lama TIDAK dibaca di sini, dan itu
// bukan kelalaian — kedua isiannya disalin ke variabel lokal lalu tidak pernah dipakai lagi,
// sehingga penyaringnya memang tidak berfungsi. Lihat catatan pada `inboxkomunikasicabang.Query`.
//
// Ia tetap dikumpulkan sebagai fungsi tersendiri supaya daftar dan ekspor membaca parameter
// yang SAMA PERSIS. Ekspor yang membaca tab dengan cara berbeda akan menghasilkan berkas
// yang isinya tidak dapat dicocokkan dengan apa pun di layar.
func readFilter(query url.Values) inboxkomunikasicabang.QueryInput {
	return inboxkomunikasicabang.QueryInput{Tab: query.Get("tab")}
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxkomunikasicabang.Caller, bool) {
	if h.caller == nil {
		return inboxkomunikasicabang.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return inboxkomunikasicabang.Caller{}, false
	}
	return inboxkomunikasicabang.Caller{Login: caller.Login}, true
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
