package inboxsalvagehttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/usecase"
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
	// Berbeda dari modul inbox lain, ia MENYARING di sini: daftar "Request Balai Lelang"
	// menampilkan pengajuan yang PIC-nya pemanggil sendiri.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Salvage.
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

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan
	// diimpor dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa
	// menariknya serta.
	GetCaller CallerReader

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
	// galat portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox Salvage.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-salvage/daftar.
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

// List menangani GET /api/inbox-salvage.
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
		readFilter(query.Get("daftar"), query.Get("cari")),
		inboxsalvage.Pagination{
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

// Counts menangani GET /api/inbox-salvage/ringkas.
//
// Ia terpisah dari List karena isinya TIDAK berubah saat pengguna berpindah daftar, dan
// layar dapat menyimpannya lebih lama. Menggabungkannya akan menjalankan keempat belas
// hitungannya setiap kali pengguna membuka tab lain.
func (h *Handler) Counts(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	counts, err := h.service.Counts(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toCountsResponse(counts, active.Alias))
}

// maxCreateBody membatasi ukuran badan permintaan simpan.
//
// Badan terbesar yang wajar adalah form berisi 500 baris Detail Item Salvage, dan itu jauh
// di bawah batas ini. Batasnya ada supaya badan permintaan sepanjang beberapa megabita
// tidak pernah dibaca seluruhnya ke memori sebelum ditolak.
const maxCreateBody = 2 << 20 // 2 MiB

// Create menangani POST /api/inbox-salvage.
//
// Ia MENULIS — satu-satunya di modul ini, dan satu-satunya di antara modul inbox yang sudah
// dibangun. Tabel yang ditulisnya dimiliki modul ini selama masa paralel (`P-1`).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	var request CreateRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCreateBody))

	// Isian yang TIDAK dikenal ditolak, bukan diabaikan.
	//
	// Form ini punya tujuh belas isian dengan nama yang mirip-mirip — `minimum_salvage`,
	// `nilai_penawaran`, `share_tertanggung` — dan satu salah ketik yang diabaikan diam-
	// diam akan menyimpan pengajuan dengan nilai uang yang kosong tanpa satu pun tanda.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		h.writeError(w, r, inboxsalvage.NewValidationError([]inboxsalvage.Violation{{
			Field: inboxsalvage.FieldFormClaimNo,
			Message: "Data yang dikirim tidak terbaca. Muat ulang halaman lalu isi " +
				"ulang formulirnya.",
		}}))
		return
	}

	created, err := h.service.Create(r.Context(), active.Alias, caller, request.toInput())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusCreated, CreateResponse{
		SalvageID: created.SalvageID,
		ItemCount: created.ItemCount,
		Message: "Pengajuan salvage tersimpan dan masuk antrean Checker. " +
			"Yang BELUM terjadi: pengajuan ini tidak dikirim ke balai lelang, tidak ada " +
			"email yang terkirim, dan berkas lampiran belum tersimpan. Ketiganya belum " +
			"dibangun di sistem baru.",
		Portal: active.Alias,
	})
}

// maxUploadBody membatasi ukuran berkas "Upload Detail Salvage".
const maxUploadBody = 2 << 20 // 2 MiB

// Upload menangani POST /api/inbox-salvage/unggah-detail.
//
// # Ia TIDAK menyimpan apa pun
//
// Itu yang paling mudah disalahpahami tentang tombolnya, dan penelusuran ke
// `Activity/UploadDetailSalvage-Act.xml` membuktikannya: ketiga langkahnya hanya menyalin
// isi berkas ke grid di dalam form. Tidak ada satu pun `RDB-List`, `Obj-Save`, maupun
// `Commit`.
//
// Jawabannya karena itu 200, bukan 201: tidak ada sumber daya yang tercipta.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if _, _, ready := h.prepare(w, r); !ready {
		return
	}

	// Batas ukuran dipasang pada BADAN permintaan, sebelum berkasnya dibaca. Memasangnya
	// setelah `FormFile` tidak menolong: pada saat itu badan permintaan sudah terbaca
	// seluruhnya ke memori atau ke berkas sementara.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBody)

	file, _, err := r.FormFile("berkas")
	if err != nil {
		h.writeError(w, r, inboxsalvage.NewValidationError([]inboxsalvage.Violation{{
			Field:   inboxsalvage.FieldFormFile,
			Message: "Berkas belum dipilih, atau ukurannya melebihi batas.",
		}}))
		return
	}
	defer file.Close()

	// Yang diurai adalah BERKASNYA, bukan `r.Body`. Badan permintaan sudah habis dibaca
	// `FormFile` di atas, dan menguraikannya lagi akan selalu menghasilkan berkas kosong.
	items, err := inboxsalvage.ParseUpload(file)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	rows := make([]DetailItemRequest, 0, len(items))
	for _, item := range items {
		rows = append(rows, DetailItemRequest{
			Name:     item.Name,
			Quantity: item.Quantity,
			Unit:     item.Unit,
			Remarks:  item.Remarks,
		})
	}

	h.writeJSON(w, r, http.StatusOK, UploadResponse{
		Items: rows,
		Message: "Berkas terbaca dan isinya dimasukkan ke tabel Detail Item Salvage. " +
			"Belum ada yang tersimpan — tekan Submit untuk menyimpannya.",
	})
}

// RejectWrite menjawab aksi tulis yang belum tersedia.
//
// Ia sengaja BUKAN 404. Layar lama punya tiga tindakan yang belum dibangun — Approve,
// Reject, dan Send To BalaiLelang — dan ketiganya tergambar sebagai tombol pada grid
// Checker. Tindakan yang dijawab "halaman tidak ditemukan" terbaca sebagai kerusakan,
// sementara yang dibutuhkan pengguna adalah tahu ke mana ia harus pergi.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	if _, exists := portalhttp.ActivePortalFrom(r.Context()); !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Dicatat, bukan hanya ditolak. Selama masa paralel, inilah satu-satunya tanda seberapa
	// sering pengguna benar-benar membutuhkan aksi ini — dan itu yang menjadi dasar
	// memutuskan kapan ia dibangun.
	if h.logger != nil {
		h.logger.Info(
			"aksi tulis diminta pada tindakan yang belum dibangun",
			slog.String("modul", "inbox-salvage"),
			slog.String("jalur", r.URL.Path),
			slog.String("tindakan", strings.TrimSpace(r.URL.Query().Get("tindakan"))),
		)
	}

	h.writeError(w, r, inboxsalvage.ErrWriteNotAvailable)
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Urutannya penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal dijawab
// sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxsalvage.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxsalvage.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxsalvage.ErrCallerUnknown)
		return portal.Portal{}, inboxsalvage.Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil lewat jembatan yang disuntikkan cmd.
func (h *Handler) readCaller(r *http.Request) (inboxsalvage.Caller, bool) {
	if h.caller == nil {
		return inboxsalvage.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists {
		return inboxsalvage.Caller{}, false
	}
	return inboxsalvage.Caller{Login: caller.Login}, true
}

// readFilter membaca isian penyaring dari parameter query.
//
// Dua: daftar mana yang diminta, dan kata kunci pencariannya. Keduanya boleh kosong.
func readFilter(tab, search string) inboxsalvage.QueryInput {
	return inboxsalvage.QueryInput{
		Tab:    strings.TrimSpace(tab),
		Search: strings.TrimSpace(search),
	}
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak terbaca menghasilkan 0, yang kemudian DIBETULKAN Pagination.Normalize
// menjadi nilai bawaan — bukan ditolak. Halaman dan ukuran datang dari alamat yang mudah
// salah ketik, dan menolak seluruh permintaan karena `halaman=abc` akan membuat layar gagal
// tanpa alasan yang terbaca pengguna.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// Detail menangani GET /api/inbox-salvage/pengajuan/{id}.
//
// Ia melayani panel "Detail Salvage" — layar yang di Pega dibuka lewat
// `SetDataDetailSalvage_act`, dan yang isinya berasal dari DUA kueri: satu baris kepala dan
// satu daftar barang.
//
// # Kenapa ID-nya di jalur, bukan di parameter query
//
// Karena ia MENUNJUK satu pengajuan, bukan menyaring daftar. Jalur yang menunjuk sumber daya
// tunggal membuat jawaban 404 punya arti yang jelas — pengajuannya tidak ada — sedangkan
// parameter query yang tidak cocok hanya menghasilkan daftar kosong.
//
// # Yang TIDAK diperiksa di sini
//
// Apakah pemanggil berhak melihat pengajuan INI. SELURUH daftar yang ditawarkan layar
// memang bersama, sehingga pemeriksaan per baris tidak punya dasar di Pega.
//
// Satu-satunya daftar yang pernah menyaring menurut PIC — "Request Balai Lelang" — kini
// tidak ditawarkan. Akibatnya modul ini TIDAK punya satu pun pembatas berbasis pengguna,
// dan jejak audit menjadi satu-satunya kontrol yang tersisa (`D-59`).
//
// Itu perilaku Pega pula: `SetDataDetailSalvage_act` tidak memeriksa pemanggil sama sekali.
// Ia dipertahankan (`P-5`), dan yang mengimbanginya adalah pencatatan setiap pembukaan
// (`D-59`). Bila kelak `TKT-F3-004` menetapkan pemeriksaan per baris, tempatnya di usecase,
// bukan di sini.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	h.detail(w, r, active, caller, inboxsalvage.DetailKeySubmission,
		chi.URLParam(r, "id"))
}

// DetailByClaim menangani GET /api/inbox-salvage/klaim/{no}.
//
// Ia melayani keenam daftar yang barisnya KLAIM. Barisnya tidak membawa ID pengajuan sama
// sekali — kueri yang memasoknya membaca `T_CLAIM_PNC` dan tidak mengambil satu pun kolom
// dari `PNC_SALVAGE` — sehingga pengajuannya dicari dari klaimnya.
//
// Rute TERSENDIRI, bukan satu rute dengan penanda jenis di parameter query. Alasannya:
// nomor klaim dan ID pengajuan adalah dua ruang nilai yang berbeda, dan satu rute yang
// menerima keduanya akan menjawab "tidak ditemukan" untuk nilai yang sebenarnya sah pada
// ruang yang lain — pesan yang menyesatkan justru di layar tempat nomor klaim dan ID
// pengajuan sudah sering tertukar oleh alias Pega yang menyesatkan.
func (h *Handler) DetailByClaim(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	h.detail(w, r, active, caller, inboxsalvage.DetailKeyClaim,
		chi.URLParam(r, "no"))
}

// detail adalah badan bersama kedua rute rincian.
func (h *Handler) detail(
	w http.ResponseWriter,
	r *http.Request,
	active portal.Portal,
	caller inboxsalvage.Caller,
	key inboxsalvage.DetailKey,
	reference string,
) {
	clean := strings.TrimSpace(reference)
	if clean == "" {
		h.writeError(w, r, inboxsalvage.ErrRowNotFound)
		return
	}

	detail, err := h.service.Detail(r.Context(), active.Alias, caller, key, clean)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDetailResponse(detail, active.Alias))
}
