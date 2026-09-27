package inboxkomunikasicabanghttp

import (
	"context"
	"encoding/json"
	"io"
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

	// Name adalah nama pengguna yang terbaca manusia.
	//
	// Ia dibutuhkan SEJAK 2026-09-24, ketika modul ini mulai menulis: balasan menyimpannya
	// di `REPLYFROMNAME`, dan itulah yang digambar kolom "Penjawab(Dari)". Ia disimpan
	// bersama balasannya, bukan diambil lewat join saat dibaca — jejak yang namanya diambil
	// lewat join berubah ketika orangnya berganti nama, dan jejak yang dapat berubah bukan
	// jejak.
	//
	// Pada rute BACA ia tidak dipakai sama sekali, sehingga rute baca tetap dilayani meski
	// nama tidak terbaca. Yang menolak adalah NewReplyCommand, di lapisan domain.
	Name string
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

// maxReplyBodyBytes membatasi badan permintaan balasan yang dibaca.
//
// Ia dinyatakan dalam BITA, sementara batas panjang balasan dinyatakan dalam rune
// (`maxReplyLength`, 4.000). Keduanya bukan aturan yang sama dan tidak boleh disamakan:
// yang ini menjaga MEMORI server terhadap kiriman yang tidak berniat baik, yang itu
// menjaga isian agar masuk akal bagi manusia dan basis data.
//
// Nilainya longgar dengan sengaja — 4.000 rune UTF-8 dapat memakan sampai 16.000 bita, dan
// batas yang lebih ketat akan menolak balasan yang panjangnya SAH dengan pesan yang
// menyesatkan ("badan permintaan tidak dapat dibaca") alih-alih pesan yang menyebut
// panjangnya.
const maxReplyBodyBytes = 64 << 10

// Reply menangani POST /api/inbox-komunikasi-cabang/komunikasi/{komunikasi}/balas.
//
// Ia menulis. Yang terjadi di balik satu permintaan ini ada dua: `M_KOMUNIKASI_PNC` diperbarui
// dan satu baris riwayat masuk ke `M_KOMUNIKASI_CABANG` — keduanya dalam SATU transaksi, yang
// di sistem lama tidak demikian. Lihat catatan pada repo/sqlstore.
//
// POST, bukan PUT. Balasan bukan penggantian sumber daya yang sudah ada melainkan peristiwa
// yang ditambahkan padanya, dan setiap pemanggilan menambah satu baris riwayat lagi —
// sehingga ia memang tidak idempoten (`10-API-STRATEGY.md` §2).
func (h *Handler) Reply(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	var body ReplyRequest

	decoder := json.NewDecoder(io.LimitReader(r.Body, maxReplyBodyBytes))

	// Isian yang tidak dikenal DITOLAK, bukan diabaikan diam-diam. Layar yang salah menamai
	// isiannya akan mengirim balasan kosong tanpa satu pun tanda, dan balasan kosong yang
	// tersimpan memindahkan percakapan ke tab "Sudah Dijawab" — terbaca sudah dijawab padahal
	// tidak ada jawabannya.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		h.writeError(w, r, inboxkomunikasicabang.NewValidationError(
			[]inboxkomunikasicabang.Violation{{
				Field:   inboxkomunikasicabang.FieldReplyMessage,
				Message: "Balasan tidak dapat dibaca dari permintaan.",
			}},
		))
		return
	}

	err := h.service.Reply(r.Context(), active.Alias, caller, inboxkomunikasicabang.ReplyInput{
		ID:      chi.URLParam(r, "komunikasi"),
		Message: body.Message,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// 200, bukan 201. Tidak ada sumber daya baru yang punya alamat sendiri — yang berubah
	// adalah percakapan yang alamatnya sudah ada.
	h.writeJSON(w, r, http.StatusOK, ActionResponse{
		ID: strings.TrimSpace(chi.URLParam(r, "komunikasi")),
		Message: "Balasan tersimpan. Percakapan ini berpindah ke tab \"Sudah Dijawab\", " +
			"dan cabang tujuan melihatnya sebagai jawaban terakhir.",
		Portal: active.Alias,
	})
}

// Finish menangani POST /api/inbox-komunikasi-cabang/komunikasi/{komunikasi}/selesai.
//
// Ia tombol "Selesai Komunikasi", dan akibatnya TIDAK DAPAT DIBATALKAN dari layar mana pun:
// barisnya hilang dari kedua tab, dan sistem lama tidak punya satu pun tindakan yang
// membukanya kembali.
//
// Tidak ada badan permintaan. Nomornya di jalur, pelakunya dari sesi, dan tidak ada isian
// ketiga — menerima badan permintaan yang isinya tidak dipakai hanya membuka pertanyaan
// tentang apa yang terjadi bila ia diisi.
func (h *Handler) Finish(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	id := chi.URLParam(r, "komunikasi")

	err := h.service.Finish(r.Context(), active.Alias, caller,
		inboxkomunikasicabang.DetailInput{ID: id})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, ActionResponse{
		ID: strings.TrimSpace(id),
		Message: "Percakapan ditutup. Ia tidak lagi tampil di kedua tab, dan tidak dapat " +
			"dibuka kembali dari layar ini.",
		Portal: active.Alias,
	})
}

// CATATAN. Handler.RejectWrite DIHAPUS pada 2026-09-24.
//
// Ia menjawab keempat tindakan tulis layar lama dengan alasan, dan keempatnya kini benar-benar
// bekerja: "Balas", "Selesai Komunikasi", "Kirim Pesan", dan "Tambah" — yang terakhir ternyata
// tombol yang hanya MEMBUKA form, tidak menyentuh peladen sama sekali.
//
// Jalur penolakan yang tidak lagi dicapai siapa pun lebih buruk daripada tidak ada: ia tetap
// dipelihara, tetap diuji, dan tetap menyatakan kepada pembacanya bahwa ada sesuatu yang belum
// tersedia. Bersamanya ikut dicabut rute `/tindakan` dan galat ErrWriteNotAvailable.

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

	// Nama BOLEH kosong di sini, dan itu disengaja: seluruh rute baca hanya membutuhkan
	// login, dan menolak permintaan baca karena nama tidak terbaca akan mematikan layar
	// untuk keadaan yang tidak menghalangi apa pun. Yang menuntut nama hanyalah balasan,
	// dan penolakannya terjadi di NewReplyCommand — satu tempat, bukan dua.
	return inboxkomunikasicabang.Caller{Login: caller.Login, Name: caller.Name}, true
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
