package masterreashttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterreas/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat empat isian pendek. 32 KiB sudah jauh lebih dari cukup —
// angkanya disamakan dengan modul master lain alih-alih diperkecil, supaya tidak ada satu
// modul yang diam-diam menolak permintaan yang diterima modul tetangganya.
const maxRequestBody = 32 << 10

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Dipakai HANYA untuk mengisi log: `POOLDATA.T_REINSURER` tidak punya satu pun kolom
// pencatat pelaku maupun stempel waktu. Itu bukan pengganti jejak audit `S-5`, dan tidak
// diklaim demikian.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

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
	caller        CallerReader
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service
	Caller  CallerReader
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
	if o.Caller == nil {
		return nil, errors.New("masterreas/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New(
			"masterreas/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
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

// Save menangani PUT /master/reas.
//
// # Jalurnya TANPA parameter, dan itu disengaja
//
// Kunci alaminya tiga kolom, salah satunya boleh kosong; lihat SaveRequest. Seluruh kuncinya
// dikirim di badan permintaan.
//
// # Hanya `email` yang berubah
//
// `Database/UPDATEREAS.prc` pada baris yang sudah ada hanya menyentuh kolom `EMAIl`. Baris
// yang tidak ada **ditolak 404**, tidak disisipkan diam-diam seperti yang dilakukan prosedur
// itu — penyisipan milik alur PLA/DLA, bukan milik petugas yang menekan Simpan di sini.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras alih-alih menghasilkan log tanpa pelaku — dan pada tabel ini
		// log adalah SATU-SATUNYA tempat "siapa yang mengubah surel ini" terekam.
		h.writeError(w, r, errors.New(
			"masterreas/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	if err := h.service.Save(r.Context(), active.Alias, request.toKey(), request.toInput(),
		usecase.Actor{Login: by.Login}, h.logger); err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// Badannya memuat baris sebagaimana ia tersimpan sekarang. Dibentuk dari permintaan,
	// bukan dibaca ulang dari basis data: yang berubah hanya surel, dan ketiga kolom kunci
	// beserta sisanya tidak tersentuh.
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Member: MemberDTO{
			ReinsurerID:   strings.TrimSpace(request.ReinsurerID),
			ReinsurerName: strings.TrimSpace(request.ReinsurerName),
			Type:          strings.TrimSpace(request.Type),
			Email:         strings.TrimSpace(request.Email),
		},
		Portal: active.Alias,
	})
}

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
// ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam. Pada modul ini itu lebih
	// dari sekadar menangkap salah ketik: `login`, `negara`, dan `cadangan` DIKIRIM pada
	// setiap jawaban tetapi tidak dapat dikirim balik, dan klien yang mengembalikan seluruh
	// objek apa adanya harus mengetahuinya saat pertama dicoba — bukan menemukan bahwa
	// ketiganya diam-diam tidak berpengaruh.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan badan
		// permintaan.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	return true
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
// # Yang didaftarkan, dan yang TIDAK
//
// **`PUT` ada, dan ia hanya mengubah surel.** Layar lamanya punya kolom **Aksi** berisi
// tombol Ubah — diberitahukan Work Owner 2026-10-05, dan tidak dapat dibaca dari export
// karena section gridnya hilang (`R-16`). Yang dapat diubah hanya `EMAIL`, karena
// `Database/UPDATEREAS.prc` pada baris yang sudah ada memang hanya menyentuh kolom itu.
//
// Jalurnya **tanpa parameter** — kunci alaminya tiga kolom dan salah satunya boleh kosong,
// sehingga seluruhnya dikirim di badan permintaan; lihat SaveRequest.
//
// **Tidak ada POST.** Layar lamanya tidak punya tombol Tambah, dan itu terkalibrasi: indeks
// rule `Harness/MasterLoginSurvey-Harness.xml` — layar yang terbukti punya dua tombol —
// menyebut `PYBUTTONLABEL!REFRESH` **dan** `!TAMBAH`; `DataMemberReas` hanya `REFRESH`.
// Baris baru lahir dari alur PLA/DLA.
//
// **Tidak ada DELETE.** Tidak satu pun rule di export menghapus baris tabel ini, dan `D-66`
// melarang penghapusan fisik data bernilai bisnis.
//
// **Tidak ada GET satu baris.** Form ubah dimuat dari baris yang sudah ada di daftar, sama
// seperti layar lamanya memuat grid lebih dulu.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/reas", h.List)
		perPortal.Put("/master/reas", h.Save)
	})
}
