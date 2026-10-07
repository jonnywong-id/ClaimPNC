package inboxmanagerreceivepuclhttp

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Metadata menangani GET /api/inbox-manager-receive-pucl/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab, kolom, dan selisih terencana adalah bentuk
// layar, bukan data entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toMetadataResponse(h.Service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-manager-receive-pucl.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()

	listed, err := h.Service.List(
		r.Context(),
		active.Alias,
		caller,
		readFilter(query),
		inboxmanagerreceivepucl.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
}

// Document menangani GET /api/inbox-manager-receive-pucl/dokumen/{referensi}.
//
// Ia LAYAR KERJA penerimaan dokumen — yang di Pega terbuka lewat Open Assignment ketika
// nomor case pada grid Receive diklik, dan flow action yang menunggu di sana adalah
// `InputReceiveDocument`.
//
// Kuncinya diambil dari JALUR, bukan parameter query: ia mengidentifikasi sumber daya, bukan
// menyaringnya (`10-API-STRATEGY.md` §2).
func (h *Handler) Document(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	// `pzInsKey` memuat SPASI — `ASM-FW-GCNMFW-WORK RCV-900001` — sehingga ia dikirim layar
	// dalam bentuk terkodekan. chi sudah menguraikannya; yang tersisa hanyalah memangkas
	// spasi di ujung, supaya kunci yang tidak sengaja berspasi tidak menjadi "tidak
	// ditemukan" yang tidak dapat dijelaskan siapa pun.
	reference := strings.TrimSpace(chi.URLParam(r, "referensi"))

	doc, err := h.Service.Document(r.Context(), active.Alias, caller, reference)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toDocumentResponse(doc, active.Alias))
}

// RejectWrite menjawab aksi tulis yang belum tersedia.
//
// Ia sengaja BUKAN 404. Layar lama punya tindakan yang menulis — antara lain mencetak surat
// PUCL/RCL, yang mengisi `TANGGALCETAKDOKUMENPUCL_1` pada objek kerja klaim. Tindakan yang
// dijawab "halaman tidak ditemukan" terbaca sebagai kerusakan, sementara yang dibutuhkan
// pengguna adalah tahu ke mana ia harus pergi.
//
// Portal tetap diperiksa lebih dulu meski permintaannya pasti ditolak: jawaban yang
// menyebut portal aktif untuk permintaan yang tidak menyebut portal akan membuat layar
// mengira ia sudah berada di portal yang benar.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	if _, exists := portalhttp.ActivePortalFrom(r.Context()); !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	// Dicatat, bukan hanya ditolak. Selama masa paralel, inilah satu-satunya tanda seberapa
	// sering pengguna benar-benar membutuhkan aksi ini — dan itu yang menjadi dasar
	// memutuskan kapan kepemilikan tabelnya dipindahkan (`P-1`).
	//
	// Kode tindakannya ikut dicatat, dan itu bukan kelengkapan: layar kerja penerimaan
	// dokumen punya DELAPAN tombol yang pemiliknya berbeda-beda — `B-2` untuk "Register
	// Klaim", `S-1` untuk unggah dokumen, `S-3` untuk komunikasi, `S-4` untuk transfer ke
	// ASM. Jejak yang hanya menyebut jalurnya tidak dapat menjawab pertanyaan yang justru
	// ingin dijawabnya: modul mana yang paling mendesak dibangun.
	if h.Logger != nil {
		h.Logger.Info(
			"aksi tulis diminta pada modul yang belum menulis",
			slog.String("modul", "inbox-manager-receive-pucl"),
			slog.String("jalur", r.URL.Path),
			slog.String("tindakan", r.URL.Query().Get("tindakan")),
			slog.String("berkas", r.URL.Query().Get("berkas")),
		)
	}

	h.WriteError(w, r, inboxmanagerreceivepucl.ErrWriteNotAvailable)
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Ia dikumpulkan karena List dan Export menuntut keduanya dengan urutan yang sama, dan
// urutan itu penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal dijawab
// sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxmanagerreceivepucl.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxmanagerreceivepucl.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxmanagerreceivepucl.ErrCallerUnknown)
		return portal.Portal{}, inboxmanagerreceivepucl.Caller{}, false
	}

	return active, caller, true
}

// readFilter membaca isian penyaring dari parameter query.
//
// Hanya satu: tab mana yang diminta. Layar lama tidak punya satu pun kotak cari maupun
// checkbox — lihat catatan di inboxmanagerreceivepucl.Query.
//
// Ia tetap dikumpulkan sebagai fungsi tersendiri supaya daftar dan ekspor membaca parameter
// yang SAMA PERSIS. Ekspor yang membaca tab dengan cara berbeda akan menghasilkan berkas
// yang isinya tidak dapat dicocokkan dengan apa pun di layar.
func readFilter(query map[string][]string) inboxmanagerreceivepucl.QueryInput {
	values := query["tab"]
	if len(values) == 0 {
		return inboxmanagerreceivepucl.QueryInput{}
	}
	return inboxmanagerreceivepucl.QueryInput{Tab: values[0]}
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxmanagerreceivepucl.Caller, bool) {
	if h.Caller == nil {
		return inboxmanagerreceivepucl.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return inboxmanagerreceivepucl.Caller{}, false
	}
	return inboxmanagerreceivepucl.Caller{Login: caller.Login}, true
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan Pagination.Normalize membetulkannya
// menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan membuat layar
// gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman pertama adalah
// jawaban yang selalu masuk akal.
func positiveNumber(raw string) int { return httpquery.NonNegative(raw) }
