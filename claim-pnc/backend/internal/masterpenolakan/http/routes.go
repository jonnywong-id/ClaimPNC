package masterpenolakanhttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat paling banyak tiga isian pendek; 64 KiB sudah jauh lebih
// dari cukup. Batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum memakan
// memori, bukan setelah.
const maxRequestBody = 64 << 10

// Parent menangani GET /master/penolakan-klaim/status-1.
//
// Rutenya DIPASANGI PortalAktif, berbeda dari daftar posisi klaim pada modul Master
// Status Progres: isinya dibaca dari tabel milik entitas yang bersangkutan, bukan daftar
// tetap milik aplikasi. Dua entitas punya alasan penolakan yang berbeda, dan menyajikan
// daftar satu entitas kepada entitas lain adalah kebocoran yang justru dicegah R-20.
func (h *Handler) Parent(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.ListParent(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ParentListResponse{
		Parent: toParentListDTO(list),
		Portal: active.Alias,
	})
}

// Create menangani POST /master/penolakan-klaim.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, by, request, ready := h.prepare(w, r)
	if !ready {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, by, masterpenolakan.Input{
		Name:       request.Name,
		ParentID:   request.ParentID,
		ParentName: request.ParentName,
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID, ID induk,
	// status, dan waktu pengajuan yang seluruhnya diterbitkan server, sehingga layar
	// tidak punya cara lain mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Rejection: toDTO(saved),
		Portal:    active.Alias,
	})
}

// Update menangani PUT /master/penolakan-klaim/{id}.
//
// Ia MENGEMBALIKAN baris ke antrean persetujuan; alasannya beserta buktinya ada pada doc
// comment masterpenolakan.Repo.Update. Layar menyatakannya sebelum pengguna menyimpan,
// supaya akibatnya tidak ditemukan setelah terjadi.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterpenolakan.ErrNotFound)
		return
	}

	active, by, request, ready := h.prepare(w, r)
	if !ready {
		return
	}

	saved, err := h.service.Update(r.Context(), active.Alias, by, id, masterpenolakan.Input{
		Name:       request.Name,
		ParentID:   request.ParentID,
		ParentName: request.ParentName,
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Rejection: toDTO(saved),
		Portal:    active.Alias,
	})
}

// prepare menjalankan tiga pemeriksaan yang sama pada Create dan Update: portal aktif,
// identitas pemanggil, lalu badan permintaan.
//
// Urutannya menentukan dan sengaja begini — yang paling murah dan paling sering gagal
// diperiksa lebih dulu, sebelum badan permintaan dibaca sama sekali.
//
// Nilai terakhir false bila responsnya sudah ditulis.
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (portal.Portal, string, SaveRequest, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return portal.Portal{}, "", SaveRequest{}, false
	}

	caller, known := h.caller(r.Context())
	if !known || caller.Login == "" {
		// Seharusnya mustahil: rute ini berada di balik middleware Autentikasi, yang
		// menolak permintaan tanpa sesi jauh sebelum sampai ke sini. Bila tetap terjadi,
		// ia cacat perakitan — dan menyimpan baris dengan kolom pelaku yang kosong jauh
		// lebih buruk daripada menolaknya, karena jejaknya hilang tanpa jejak.
		h.writeError(w, r, errors.New("masterpenolakan/http: identitas pemanggil tidak tersedia"))
		return portal.Portal{}, "", SaveRequest{}, false
	}

	request, parsed := h.readRequest(w, r)
	if !parsed {
		return portal.Portal{}, "", SaveRequest{}, false
	}
	return active, caller.Login, request, true
}

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest
	if !h.decode(w, r, &request) {
		return SaveRequest{}, false
	}
	return request, true
}

// decode membaca satu dokumen JSON dari badan permintaan.
//
// Satu fungsi untuk kedua tab supaya keduanya menolak hal yang sama persis — badan yang
// tidak terbaca, field yang tidak dikenal, dan badan yang memuat lebih dari satu dokumen.
func (h *Handler) decode(w http.ResponseWriter, r *http.Request, to any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(to); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan
		// badan permintaan.
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

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan jaringan,
		// cacat pemrograman — diserahkan ke penulis bersama, yang menjawab 500 dengan
		// pesan umum dan menaruh rinciannya di log saja. Rincian galat internal tidak
		// pernah dikirim ke peramban.
		h.writeError(w, r, err)
		return
	}

	if status >= http.StatusInternalServerError {
		logging.From(r.Context(), h.logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
	h.writeResponse(w, r, status, body)
}

// Mount mendaftarkan seluruh rute modul Master Penolakan Klaim — kedua tab.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// SELURUH rute dipasangi PortalAktif. Tidak ada satu pun rute di sini yang isinya milik
// aplikasi — bahkan daftar Status Penolakan 1 pun dibaca dari basis data entitas.
//
// # Yang TIDAK didaftarkan, dan kenapa
//
// Tidak ada DELETE pada satu tab pun. Seluruh export tidak memuat satu pun pernyataan
// DELETE terhadap ketiga tabel modul ini, layar lama tidak punya tombolnya, dan tidak
// satu pun tabelnya punya kolom penanda terhapus yang dapat dipakai `D-66`. Mendaftarkan
// rute yang tidak dapat berbuat apa-apa hanya memindahkan kejutannya dari layar ke API.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		// Jalur daftar Status Penolakan 1 didaftarkan LEBIH DULU daripada rute
		// ber-parameter pada prefiks yang sama. chi mencocokkan segmen statis lebih dulu,
		// sehingga urutannya sebenarnya tidak menentukan — tetapi menuliskannya berurutan
		// membuat pembaca tidak perlu mengetahui hal itu untuk yakin.
		perPortal.Get("/master/penolakan-klaim/status-1", h.Parent)

		perPortal.Get("/master/penolakan-klaim", h.List)
		perPortal.Post("/master/penolakan-klaim", h.Create)
		perPortal.Put("/master/penolakan-klaim/{id}", h.Update)

		perPortal.Get("/master/penolakan-komite", h.ListKomite)
		perPortal.Post("/master/penolakan-komite", h.CreateKomite)
		perPortal.Put("/master/penolakan-komite/{id}", h.UpdateKomite)
	})
}
