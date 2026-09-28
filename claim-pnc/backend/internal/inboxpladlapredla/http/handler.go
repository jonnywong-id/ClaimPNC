package inboxpladlapredlahttp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/inboxpladlapredla/usecase"
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
	// Ia TIDAK menyaring di layar ini — ketiga daftarnya bersama. Yang membutuhkannya
	// adalah jejak.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox PLA, DLA, Pre DLA.
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

// NewHandler membentuk handler modul Inbox PLA, DLA, Pre DLA.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-pla-dla-pre-dla/daftar.
//
// Ia GET dan tidak mengubah apa pun: daftar tab, kolom, dan selisih terencana adalah
// bentuk layar, bukan data entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK,
		toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-pla-dla-pre-dla.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	input, err := readFilter(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		caller,
		input,
		inboxpladlapredla.Pagination{
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

// Documents menangani GET /api/inbox-pla-dla-pre-dla/klaim/{kunci}.
//
// # Kenapa kunci klaim ada di JALUR, bukan di parameter query
//
// Karena ia MENUNJUK satu klaim, bukan menyaring daftar. Jalur yang menunjuk sumber daya
// tunggal membuat jawaban 404 punya arti yang jelas — klaimnya tidak ada pada entitas ini
// — sedangkan parameter query yang tidak cocok hanya menghasilkan daftar kosong.
//
// # Kuncinya WAJIB terkodekan di alamat
//
// `T_CLAIM_PNC.CLAIMID` berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx`, dan ia memuat SPASI.
// Kunci yang tidak terkodekan akan terpotong di spasi itu, dan yang sampai ke sini hanya
// `ASM-FW-GCNMFW-WORK` — kunci yang tidak pernah cocok dengan klaim mana pun. Layar
// mengodekannya; chi mengurainya kembali.
//
// # Daftar mana yang dimintakan rinciannya datang dari parameter query
//
// Berbeda dari kunci klaim, ia MENYARING: klaim yang sama punya grid rincian yang berbeda
// di tab PLA dan tab DLA — tabel yang dibaca pun berbeda. Ia bukan bagian dari identitas
// sumber dayanya, sehingga tempatnya di query, bukan di jalur.
func (h *Handler) Documents(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	claimKey := strings.TrimSpace(chi.URLParam(r, "kunci"))
	if claimKey == "" {
		h.writeError(w, r, inboxpladlapredla.ErrRowNotFound)
		return
	}

	documented, err := h.service.Documents(
		r.Context(),
		active.Alias,
		caller,
		strings.TrimSpace(r.URL.Query().Get("daftar")),
		claimKey,
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK,
		toDocumentsResponse(documented, active.Alias))
}

// Print menjawab isi panel "Print Pre DLA" satu klaim.
//
// Namanya mengikuti tombolnya, tetapi ia MEMBACA saja — tidak satu pun berkas dihasilkan,
// dan tidak satu pun baris ditulis. Lihat usecase.PrintList.
func (h *Handler) Print(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	claimKey := strings.TrimSpace(chi.URLParam(r, "kunci"))
	if claimKey == "" {
		h.writeError(w, r, inboxpladlapredla.ErrRowNotFound)
		return
	}

	printable, err := h.service.PrintList(r.Context(), active.Alias, caller, claimKey)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toPrintResponse(printable, active.Alias))
}

// RejectWrite menjawab aksi tulis yang belum tersedia.
//
// Ia sengaja BUKAN 404. Layar lama punya tiga tindakan yang belum dibangun — "Send",
// "Upload File Penunjang", dan "Print Pre DLA" — dan ketiganya tergambar sebagai tombol.
// Tindakan yang dijawab "halaman tidak ditemukan" terbaca sebagai kerusakan, sementara
// yang dibutuhkan pengguna adalah tahu ke mana ia harus pergi.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	if _, exists := portalhttp.ActivePortalFrom(r.Context()); !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	rejected := inboxpladlapredla.NewNotAvailable(r.URL.Query().Get("tindakan"))

	// Dicatat, bukan hanya ditolak. Selama masa paralel, inilah satu-satunya tanda
	// seberapa sering pengguna benar-benar membutuhkan aksi ini — dan itu yang menjadi
	// dasar memutuskan kapan ia dibangun. Tindakannya ikut dicatat, sehingga ketiga
	// tombol dapat dibedakan: yang paling sering ditekan yang paling layak dibangun
	// lebih dulu.
	if h.logger != nil {
		h.logger.Info(
			"tombol yang belum dibangun ditekan",
			slog.String("modul", "inbox-pla-dla-pre-dla"),
			slog.String("jalur", r.URL.Path),
			slog.String("tindakan", string(rejected.Action)),
			slog.String("daftar",
				strings.TrimSpace(r.URL.Query().Get("daftar"))),
		)
	}

	h.writeError(w, r, rejected)
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Urutannya penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal dijawab
// sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxpladlapredla.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxpladlapredla.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxpladlapredla.ErrCallerUnknown)
		return portal.Portal{}, inboxpladlapredla.Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil lewat jembatan yang disuntikkan cmd.
func (h *Handler) readCaller(r *http.Request) (inboxpladlapredla.Caller, bool) {
	if h.caller == nil {
		return inboxpladlapredla.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists {
		return inboxpladlapredla.Caller{}, false
	}
	return inboxpladlapredla.Caller{Login: caller.Login}, true
}

// readFilter membaca isian penyaring dari parameter query.
//
// Empat: daftar mana yang diminta, kata kunci pencarian, dan kedua batas rentang tanggal.
// Seluruhnya boleh kosong.
func readFilter(r *http.Request) (inboxpladlapredla.QueryInput, error) {
	query := r.URL.Query()

	var violations []inboxpladlapredla.Violation

	from, err := readDate(query.Get("dari"))
	if err != nil {
		violations = append(violations, inboxpladlapredla.Violation{
			Field:   inboxpladlapredla.FieldFrom,
			Message: "Tanggal \"Dari\" tidak terbaca. Bentuknya YYYY-MM-DD.",
		})
	}

	to, err := readDate(query.Get("sampai"))
	if err != nil {
		violations = append(violations, inboxpladlapredla.Violation{
			Field:   inboxpladlapredla.FieldTo,
			Message: "Tanggal \"Sampai\" tidak terbaca. Bentuknya YYYY-MM-DD.",
		})
	}

	if err := inboxpladlapredla.NewValidationError(violations); err != nil {
		return inboxpladlapredla.QueryInput{}, err
	}

	return inboxpladlapredla.QueryInput{
		Tab:    strings.TrimSpace(query.Get("daftar")),
		Search: strings.TrimSpace(query.Get("cari")),
		From:   from,
		To:     to,
	}, nil
}

// readDate mengurai satu batas rentang dari parameter query.
//
// Kosong berarti tidak menyaring, dan itu BUKAN galat — berbeda dari Pega, tempat isian
// kosong tetap dirangkai menjadi `to_date(”,'dd/mm/yyyy')` lalu ditolak Oracle.
//
// Yang DITOLAK adalah isian yang terisi tetapi tidak terbaca. Membetulkannya diam-diam
// menjadi "tidak menyaring" akan menampilkan seluruh antrean kepada pengguna yang
// sebenarnya sedang mempersempitnya — hasil yang terlihat wajar dan sepenuhnya salah.
func readDate(raw string) (*time.Time, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, nil
	}

	moment, err := time.Parse(inboxpladlapredla.DateLayout, clean)
	if err != nil {
		return nil, err
	}
	return &moment, nil
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak terbaca menghasilkan 0, yang kemudian DIBETULKAN Pagination.Normalize
// menjadi nilai bawaan — bukan ditolak. Halaman dan ukuran datang dari alamat yang mudah
// salah ketik, dan menolak seluruh permintaan karena `halaman=abc` akan membuat layar
// gagal tanpa alasan yang terbaca pengguna.
//
// Perhatikan ia berbeda perlakuannya dari tanggal di atas, dan perbedaannya disengaja:
// halaman yang salah ketik menampilkan halaman pertama — tidak ada yang tersembunyi.
// Tanggal yang salah ketik, bila diabaikan, justru MELEBARKAN hasil tanpa sepengetahuan
// pengguna.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// SendPreDLA menandai satu Pre-DLA sebagai terkirim.
//
// # Kenapa POST, dan kenapa nomornya di BADAN permintaan
//
// Ia mengubah keadaan, sehingga bukan GET. Nomor Pre-DLA dikirim di badan permintaan,
// bukan di alamat: alamat tercatat di riwayat peramban, log proxy, dan header Referer,
// dan meski nomor Pre-DLA bukan data nasabah (`D-69`), tidak ada alasan menaruh pengenal
// dokumen di tempat-tempat itu.
func (h *Handler) SendPreDLA(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	claimKey := strings.TrimSpace(chi.URLParam(r, "kunci"))
	if claimKey == "" {
		h.writeError(w, r, inboxpladlapredla.ErrRowNotFound)
		return
	}

	var badan struct {
		AdviceNo string `json:"no_advice"`
	}
	// Badan yang tidak dapat dibaca diperlakukan sebagai nomor kosong, bukan sebagai
	// galat tersendiri: keduanya berujung pada pesan yang sama dan menyebut isian yang
	// sama, dan membedakannya hanya menambah satu cabang tanpa menambah keterangan.
	_ = json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&badan)

	err := h.service.SendPreDLA(
		r.Context(), active.Alias, caller, claimKey, badan.AdviceNo)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, map[string]string{
		"pesan": "Pre-DLA ditandai terkirim.",
	})
}

// SendAdvice mengirim surat PLA/DLA ke reasuradur lalu menandai dokumennya terkirim.
//
// Ia satu-satunya rute di modul ini yang menyentuh dunia di luar basis data, dan
// satu-satunya yang akibatnya tidak dapat ditarik kembali.
func (h *Handler) SendAdvice(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	claimKey := strings.TrimSpace(chi.URLParam(r, "kunci"))
	if claimKey == "" {
		h.writeError(w, r, inboxpladlapredla.ErrRowNotFound)
		return
	}

	var badan struct {
		AdviceNo string `json:"no_advice"`
		Tab      string `json:"daftar"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&badan)

	hasil, err := h.service.SendAdvice(
		r.Context(), active.Alias, caller, badan.Tab, claimKey, badan.AdviceNo)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, SendResponse{
		Message:     "Surat terkirim dan dokumen ditandai terkirim.",
		Recipients:  len(hasil.Recipients),
		Attachments: hasil.Attachments,
	})
}
