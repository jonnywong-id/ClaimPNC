package mastergroupingspareparthttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastergroupingsparepart"
	"claim-pnc/internal/mastergroupingsparepart/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat delapan isian pendek dan tidak punya daftar anak sama sekali.
// 32 KiB sudah jauh lebih dari cukup, dan batasnya ada supaya permintaan bertubuh raksasa
// ditolak sebelum memakan memori.
const maxRequestBody = 32 << 10

// maxDecisionRows membatasi banyaknya baris pada satu keputusan borongan.
//
// Sistem lama tidak membatasinya. Batas di sini bukan aturan bisnis melainkan penjaga sumber
// daya — setiap baris menjadi satu pernyataan UPDATE di dalam satu transaksi, dan transaksi
// yang menahan ribuan kunci baris menghalangi Pega yang sedang melayani produksi pada tabel
// yang sama (D-21).
const maxDecisionRows = 200

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field, dan berbeda dari Master Sparepart ia TIDAK TERSIMPAN: kedua tabel modul ini
// tidak punya kolom pencatat pelaku. Ia dibawa hanya untuk dicatat di log — satu-satunya
// tempat yang tersedia sampai `S-5` Jejak Audit dibangun.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa menyeret
// satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan master grouping sparepart.
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

// NewHandler membentuk handler modul master grouping sparepart.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("mastergroupingsparepart/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("mastergroupingsparepart/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New(
			"mastergroupingsparepart/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/grouping-sparepart.
//
// # Penyaringnya
//
//	?status=0|1|2  wajib dikenal; bila tidak disebut dianggap "1"
//	?cari=...      mempersempit pada nomor sparepart, nama sparepart, nama panel, dan no rangka
//
// Yang pertama adalah cerminan ketiga tab `Section/PNCMasterGroupingSparepartHE-Section.xml`.
//
// Yang kedua DITAMBAHKAN sebagai penyaring sisi server. Layar lama punya kotak pencarian, dan
// ia menyaring di klipboard — atas daftar yang sudah terlanjur dimuat seluruhnya.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	status := mastergroupingsparepart.StatusApproved
	if raw := r.URL.Query().Get("status"); raw != "" {
		status = mastergroupingsparepart.ApprovalStatus(raw)
	}

	list, err := h.service.List(r.Context(), active.Alias, status, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Grouping: toListDTO(list),
		Status:   string(status),
		Portal:   active.Alias,
	})
}

// Get menangani GET /master/grouping-sparepart/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, mastergroupingsparepart.ErrNotFound)
		return
	}

	found, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Grouping: toDTO(found),
		Portal:   active.Alias,
	})
}

// Create menangani POST /master/grouping-sparepart.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras, bukan diam-diam mencatat log tanpa pelaku.
		h.writeError(w, r, errors.New(
			"mastergroupingsparepart/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, request.toInput(),
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID, nomor grup,
	// status, dan kelima isian turunan yang seluruhnya diterbitkan server, sehingga layar tidak
	// punya cara lain mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Grouping: toDTO(saved),
		Portal:   active.Alias,
	})
}

// Save menangani PUT /master/grouping-sparepart/{id}.
//
// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan.
// Keputusan komite menempuh Decide; lihat usecase.Service.Save untuk selisihnya terhadap
// sistem lama, yang menerima statusnya sebagai parameter.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New(
			"mastergroupingsparepart/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, mastergroupingsparepart.ErrNotFound)
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Save(r.Context(), active.Alias, id, request.toInput(),
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Grouping: toDTO(saved),
		Portal:   active.Alias,
	})
}

// Decide menangani POST /master/grouping-sparepart/keputusan.
//
// # Kenapa POST ke sub-sumber daya, bukan PATCH pada tiap baris
//
// Karena yang terjadi adalah SATU peristiwa bisnis — sebuah keputusan atas sekumpulan
// pengajuan — dan `10-API-STRATEGY.md` §2 menetapkan aksi bisnis dimodelkan sebagai peristiwa,
// bukan sebagai pembaruan field. Memecahnya menjadi sederet PATCH juga akan membuat keputusan
// yang di Pega utuh menjadi sesuatu yang dapat gagal separuh jalan.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New(
			"mastergroupingsparepart/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request DecisionRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	if len(request.ID) > maxDecisionRows {
		h.writeModuleError(w, r, mastergroupingsparepart.OneViolation("id_grouping",
			"Terlalu banyak grouping dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."))
		return
	}

	status := mastergroupingsparepart.ApprovalStatus(request.Status)
	changed, err := h.service.Decide(r.Context(), active.Alias, request.ID, status,
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, DecisionResponse{
		Changed:     changed,
		Status:      string(status),
		StatusLabel: status.Label(),
		Portal:      active.Alias,
	})
}

// Options menangani GET /master/grouping-sparepart/pilihan.
//
// Isinya DIBACA DARI BASIS DATA entitas yang sedang dibuka — panel dan tipe kendaraan ada di
// basis data setiap entitas, dan isinya berbeda. Karena itu rutenya ikut dipasangi pemeriksaan
// portal; menyajikan daftar panel satu entitas kepada entitas lain adalah kebocoran yang
// justru dicegah `R-20`.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	set, err := h.service.Options(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK,
		toOptionsDTO(set.Panel, set.VehicleType, active.Alias))
}

// Sides menangani GET /master/grouping-sparepart/sisi.
//
// # Kenapa endpoint tersendiri, bukan bagian dari /pilihan
//
// Karena isinya bergantung pada panel yang dipilih, dan di Pega pun ia berjalan belakangan:
// `Activity/GetSisiPanel-Act.xml` dipanggil setelah Nama Panel dipilih, bukan saat layar
// dibuka. Memuat seluruh sisi setiap panel di muka berarti mengirim daftar yang 99 persen
// isinya tidak akan pernah dipakai.
//
// # Kenapa KEDUA parameternya diminta
//
//	?id_panel=...    ID_PANEL, diisi layar dari pilihan Nama Panel
//	?nama_panel=...  nilai yang diketik pada isian Nama Panel
//
// Keduanya dipakai bersama oleh `RDB List/GetDataSisiPanel-SQL.xml`, dan ditiru apa adanya.
// Lihat mastergroupingsparepart.LookupRepo.ListSides untuk ketidakpastian soal kolom `NAMA`
// yang keduanya saring.
func (h *Handler) Sides(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	side, err := h.service.Sides(r.Context(), active.Alias, mastergroupingsparepart.SideKey{
		PanelID:   r.URL.Query().Get("id_panel"),
		PanelName: r.URL.Query().Get("nama_panel"),
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, toSideDTO(side, active.Alias))
}

// Part menangani GET /master/grouping-sparepart/sparepart.
//
// Ia padanan `Activity/SetDataSparepart-Act.xml`: kelima isian turunan yang muncul begitu
// Nomor Sparepart selesai diketik.
//
//	?nomor=...  NO_SPART yang diketik pengguna
//
// Ia dipanggil layar SEBELUM menyimpan, persis seperti di Pega — pengguna melihat nama
// sparepartnya muncul dan tahu nomornya sudah benar. Penyimpanan tetap mencarinya sekali lagi;
// lihat usecase.resolvePart.
func (h *Handler) Part(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	part, err := h.service.LookupPart(r.Context(), active.Alias, r.URL.Query().Get("nomor"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, toPartDTO(part, active.Alias))
}

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
// ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field akan
	// terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun tanda
	// bahwa ada yang salah.
	//
	// Pada modul ini ia sekaligus menjaga kelima isian turunan tetap milik server: klien yang
	// mengirim `nama_sparepart` akan DITOLAK, bukan diabaikan — sehingga cacat pada klien
	// terlihat saat pertama dicoba alih-alih menjadi kebiasaan yang tidak berakibat.
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

// Mount mendaftarkan rute modul master grouping sparepart.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang merakit
// urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// Termasuk keempat rute acuan. Seluruh isinya dibaca dari basis data entitas —
// `POOLDATA.PANEL_HE`, `POOLDATA.LOKASI_PANEL_HE`, `POOLDATA.SPAREPART_HE`, dan `branddetail`
// ada di basis data setiap entitas, dan isinya berbeda. Tidak ada satu pun rutenya yang isinya
// konstanta milik aplikasi.
//
// # Urutan pendaftaran
//
// Keempat rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter `{id}` pada
// prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya sebenarnya
// tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu mengetahui hal
// itu untuk yakin bahwa `/sisi` tidak pernah terbaca sebagai sebuah ID grouping.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
//
// # Yang TIDAK didaftarkan
//
// Tidak ada DELETE terhadap grouping. Sistem lama tidak punya satu pun terhadap kedua tabel
// ini — `UpdateGroupingSparepartHE_act` tidak memuat satu pun langkah hapus — dan `D-66`
// melarang penghapusan fisik data bernilai bisnis. Grouping yang tidak lagi berlaku ditolak
// lewat keputusan, bukan dibuang.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/grouping-sparepart/pilihan", h.Options)
		perPortal.Get("/master/grouping-sparepart/sisi", h.Sides)
		perPortal.Get("/master/grouping-sparepart/sparepart", h.Part)
		perPortal.Post("/master/grouping-sparepart/keputusan", h.Decide)

		perPortal.Get("/master/grouping-sparepart", h.List)
		perPortal.Post("/master/grouping-sparepart", h.Create)
		perPortal.Get("/master/grouping-sparepart/{id}", h.Get)
		perPortal.Put("/master/grouping-sparepart/{id}", h.Save)
	})
}
