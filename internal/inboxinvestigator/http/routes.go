package inboxinvestigatorhttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxinvestigator/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// MaxKeywordLength membatasi panjang penyaring `cari`.
//
// Ia BUKAN aturan bisnis. Batasnya dipasang karena kata kunci masuk ke dalam pola LIKE, dan
// kata kunci sepanjang megabita hanya menghasilkan pemindaian penuh yang pasti tidak
// menemukan apa pun.
//
// Seratus disamakan dengan modul lain: tidak ada nomor case, nomor polis, nama tertanggung,
// maupun nama peserta yang lebih panjang dari itu, sehingga batas ini tidak pernah menolak
// pencarian yang sungguh-sungguh.
const MaxKeywordLength = 100

// Handler melayani permintaan Inbox Investigator.
//
// TANPA CallerReader, dan itu bukan kelalaian melainkan akibat langsung dari bentuk layarnya:
// yang ditampilkan adalah isi **workbasket**, yaitu antrean BERSAMA yang belum bertuan
// (`D-26`). Tidak ada satu baris pun yang diturunkan dari identitas pemanggil, dan tidak ada
// perubahan yang perlu dicatat pelakunya.
//
// Yang berubah begitu MENGAMBIL pekerjaan dari antrean dibangun: tindakan itu menuntut
// identitas pengambilnya. Ia bagian modul Penugasan, bukan modul ini.
//
// Menyimpan hasil investigasi sudah ada di sini dan MEMANG menuntut identitas — pelakunya
// masuk ke `DIBUAT_OLEH`/`DIUBAH_OLEH` — tetapi ia datang dari konteks permintaan, bukan
// dari pembaca pemanggil pada daftar.
//
// Kewenangan membuka layar ini tetap ditegakkan — lewat middleware Autentikasi yang dipasang
// cmd, dan kelak lewat pemeriksaan peran `TKT-F3-005` yang belum ada. Di sistem lama
// pembatasnya `When/IsInvestigator-When.xml`: access group `PncInvestigator` atau
// `Administrators`, dan bukan `ViewClaimPNC`.
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

// NewHandler membentuk handler modul Inbox Investigator.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("inboxinvestigator/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New(
			"inboxinvestigator/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /inbox/investigator.
//
// # Penyaringnya
//
//	?cari=...  mempersempit pada nomor case, no polis, nama tertanggung, nama peserta,
//	           nama bisnis, nama cabang, dan nama admin
//
// TANPA penyaring workbasket: nilainya konstanta, dan menjadikannya parameter berarti
// menyediakan cara membaca antrean peran lain lewat endpoint ini. Lihat
// inboxinvestigator.Workbasket.
//
// TANPA penyaring rentang tanggal, meski layar lama punya isian "Dari" dan "Sampai".
// Keduanya milik tombol Export Data Investigation dan menyaring kolom yang tidak dibaca grid
// sama sekali; lihat inboxinvestigator.Filter.
//
// TANPA paginasi. Work Owner memilih penyaringan dan paginasi dikerjakan peramban
// (2026-09-23), sehingga satu permintaan mengembalikan seluruh antrean sampai batas
// `inboxinvestigator.MaxRows` — dan menyatakan bila terpotong.
//
// # Ia GET, dan memang aman diulang
//
// Tidak ada keadaan yang berubah karenanya. Membuka layar ini berkali-kali tidak memakai
// jatah apa pun dan tidak mengambil pekerjaan siapa pun — berbeda dari View History Claim,
// yang membukanya memakai satu jatah sehingga harus POST.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	keyword := strings.TrimSpace(r.URL.Query().Get("cari"))
	if len(keyword) > MaxKeywordLength {
		// 400, bukan daftar kosong. Permintaan yang tidak dapat dipenuhi ditolak dengan
		// sebabnya — daftar kosong akan terbaca sebagai "antreannya memang kosong", dan
		// pengguna tidak punya cara membedakan keduanya (`10-API-STRATEGY.md` §4).
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Kata pencarian terlalu panjang.",
		})
		return
	}

	page, err := h.service.List(r.Context(), active.Alias, keyword)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, toListResponse(page, active.Alias))
}

// Mount mendaftarkan rute modul Inbox Investigator.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// Antrean pekerjaan ada di basis data SETIAP entitas, dan isinya memuat nama tertanggung
// serta nama peserta — data nasabah satu badan hukum. Tidak ada satu pun rute di modul ini
// yang isinya milik aplikasi, sehingga tidak ada yang dikecualikan.
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
// # Kenapa jalurnya `/inbox/investigator`, bukan `/inbox-investigator`
//
// Karena INBOX adalah kelompok menu tersendiri di sistem lama — `MENU_ID 2`, induk dari 30
// butir. Menempatkan modul inbox pertama di bawah satu awalan membuat yang berikutnya punya
// tempat yang sudah jelas, sama seperti `/master/...` yang sudah berlaku.
//
// # SATU rute yang mengubah keadaan, dan hanya satu
//
// `POST /inbox/investigator/{referensi}/investigasi` — menyimpan hasil investigasi dan
// memindahkan klaim ke Analyst. Padanannya `SetStatusInvestigator_Act`.
//
// **Tidak ada PUT maupun DELETE.** Hasil investigasi tidak pernah dihapus (`D-66`), dan
// koreksinya menempuh POST yang sama — formulir yang dibuka ulang memperbarui barisnya.
//
// Mengambil pekerjaan dari antrean (`openAssignment`) TETAP tidak ada: antrean ini
// workbasket, dan pengambilannya bagian dari modul Penugasan yang belum dibangun.
//
// # Rute ekspor IKUT didaftarkan
//
// `GET /inbox/investigator/ekspor` — tombol "Export Data Investigation" beserta ketiga
// kendalinya. Ia GET, sama seperti daftar, karena menerbitkan berkas tidak mengubah satu
// baris pun.
//
// Ia dipasangi pemeriksaan portal yang SAMA. Berkasnya memuat alamat rumah sakit dan nomor
// rekam medis — data medis milik satu badan hukum, dan justru kelompok yang `FR-R2` batasi
// aksesnya. Mengecualikannya dari pemeriksaan portal akan menjadikan tombol unduh ini
// jalur kebocoran yang paling tidak terlihat (`R-20`).
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox/investigator", h.List)
		perPortal.Get("/inbox/investigator/ekspor", h.Export)

		// Formulir kerja Investigator — padanan Flow Action `InputInvestigator`.
		//
		// Keduanya berada DI BAWAH awalan modul ini, bukan di bawah `/registrasi/...`.
		// Itu disengaja: formulirnya milik layar ini, dan menaruhnya di modul lain akan
		// membuat dua modul memiliki satu jalur tulis yang sama.
		perPortal.Get("/inbox/investigator/{referensi}/investigasi", h.Investigation)
		perPortal.Post("/inbox/investigator/{referensi}/investigasi", h.SubmitInvestigation)
	})
}
