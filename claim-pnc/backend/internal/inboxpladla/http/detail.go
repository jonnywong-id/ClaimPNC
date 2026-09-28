package inboxpladlahttp

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxpladla"
)

// Berkas ini melayani layar RINCIAN satu klaim — tombol **"Detail Claim"**.
//
// # Kunci klaimnya MEMUAT SPASI
//
// Bentuknya `ASM-FW-GCNMFW-WORK PNC-xxxx`, dan spasi itu wajib terkodekan oleh layar
// (`encodeURIComponent`). Chi menyerahkan segmen alamat sudah ter-decode, sehingga yang
// dibaca handler adalah kuncinya apa adanya — termasuk spasinya.
//
// Preseden yang sama sudah ada pada modul `inboxpladlapredla`.

// maxReplyBody membatasi ukuran badan permintaan balasan.
//
// Ia bukan pengganti batas panjang pesan — yang itu ditegakkan domain dan menjawab dengan
// pesan yang dapat dibaca pengguna. Yang ini menutup badan permintaan yang jelas
// menyerang: tanpa batas, satu permintaan dapat memaksa peladen membaca berapa pun bita ke
// dalam memori sebelum satu pun validasi berjalan.
//
// Angkanya longgar terhadap batas domain (4.000 rune) supaya pesan sah yang seluruhnya
// huruf non-ASCII tetap masuk, dan tetap jauh di bawah ukuran yang membahayakan.
const maxReplyBody = 64 << 10

// Detail menangani GET /api/inbox-pla-dla/klaim/{kunci}.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	detail, err := h.service.Detail(r.Context(), active.Alias, caller, claimKeyOf(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDetailResponse(detail, active.Alias))
}

// Documents menangani GET /api/inbox-pla-dla/klaim/{kunci}/dokumen.
//
// Ia meniru tombol **"Dokumen"** pada baris PLA atau DLA, dan kedua penyaringnya datang
// dari baris itu: nomor pemberitahuan dan jenisnya.
func (h *Handler) Documents(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()

	kind, known := inboxpladla.ParseAdviceKind(query.Get("jenis"))
	if !known {
		h.writeError(w, r, inboxpladla.ErrAdviceKindUnknown)
		return
	}

	rows, err := h.service.Documents(
		r.Context(), active.Alias, caller, claimKeyOf(r),
		strings.TrimSpace(query.Get("nomor")), kind,
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, DocumentsResponse{
		Rows:    toDocumentDTOs(rows),
		Columns: toColumnDTOs(inboxpladla.DocumentColumns()),
		Portal:  active.Alias,
	})
}

// DocumentContent menangani GET /api/inbox-pla-dla/klaim/{kunci}/dokumen/{id}.
//
// # Ia menulis BERKAS, bukan JSON
//
// Karena itulah yang diminta tombol "View Document": pengguna menekan dan berharap
// berkasnya terbuka. Membungkusnya base64 di dalam JSON — seperti `GetAttachmentFromDB_Sql`
// — memaksa layar merakit ulang berkasnya sendiri dan membesarkan muatan sepertiga tanpa
// satu pun manfaat.
func (h *Handler) DocumentContent(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	document, err := h.service.DocumentContent(
		r.Context(), active.Alias, caller, claimKeyOf(r),
		strings.TrimSpace(chi.URLParam(r, "dokumen")),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", contentTypeOf(document))
	w.Header().Set("Content-Disposition", attachmentHeader(document.Name))

	// Dokumen milik pihak luar TIDAK boleh disimpan perantara mana pun.
	w.Header().Set("Cache-Control", "no-store")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document.Content)
}

// Reply menangani POST /api/inbox-pla-dla/klaim/{kunci}/komunikasi/balas.
//
// **Satu-satunya rute yang MENULIS di modul ini**, dan pelakunya pihak luar.
func (h *Handler) Reply(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	var body ReplyRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxReplyBody))
	if err := decoder.Decode(&body); err != nil {
		// Badan yang tidak terbaca adalah kesalahan BENTUK, bukan kesalahan isi —
		// sehingga ia 400 lewat ValidationError, bukan 422. Lihat mapError.
		h.writeError(w, r, inboxpladla.NewValidationError([]inboxpladla.Violation{{
			Field:   inboxpladla.FieldReply,
			Message: "Badan permintaan tidak terbaca sebagai JSON.",
		}}))
		return
	}

	err := h.service.Reply(
		r.Context(), active.Alias, caller, claimKeyOf(r),
		inboxpladla.ReplyInput{
			ConversationID: body.ConversationID,
			Message:        body.Message,
		},
		time.Now(),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, ReplyResponse{
		Message: "Balasan Anda tersimpan dan sudah terlihat oleh petugas Asuransi " +
			"Sinar Mas.",
		Portal: active.Alias,
	})
}

// claimKeyOf membaca kunci klaim dari alamat.
func claimKeyOf(r *http.Request) string {
	return strings.TrimSpace(chi.URLParam(r, "kunci"))
}

// contentTypeOf memilih jenis isi yang dinyatakan ke klien.
//
// Yang tersimpan di `ATTACHMIMETYPE` tidak selalu terisi, dan nilai kosong akan membuat
// peramban menebak sendiri. Bila kosong, jenisnya diturunkan dari akhiran namanya; bila
// itu pun gagal, `application/octet-stream` — yang berarti "unduh saja", bukan "tampilkan".
func contentTypeOf(document inboxpladla.DocumentContent) string {
	if declared := strings.TrimSpace(document.MimeType); declared != "" {
		return declared
	}
	if guessed := mime.TypeByExtension(
		strings.ToLower(filepath.Ext(document.Name)),
	); guessed != "" {
		return guessed
	}
	return "application/octet-stream"
}

// attachmentHeader menyusun header Content-Disposition.
//
// # Nama berkasnya datang dari BASIS DATA, bukan dari kode
//
// Ia dapat memuat tanda kutip, koma, baris baru, maupun huruf non-ASCII — dan ketiganya
// merusak header bila disisipkan apa adanya. `mime.FormatMediaType` melakukan pengutipan
// yang benar; bila namanya kosong, dipakai nama cadangan supaya unduhan tetap punya nama.
func attachmentHeader(name string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		clean = "dokumen"
	}

	formatted := mime.FormatMediaType("attachment", map[string]string{
		"filename": clean,
	})
	if formatted == "" {
		// FormatMediaType menyerah pada nama yang tidak dapat diwakili; sebagai
		// gantinya dipakai bentuk RFC 5987 yang selalu sah.
		return "attachment; filename*=UTF-8''" + url.PathEscape(clean)
	}
	return formatted
}

// mapDetailError menerjemahkan galat layar rincian menjadi status dan badan HTTP.
//
// Ia dipisahkan dari mapError supaya berkas galat layar induk tidak tumbuh menjadi daftar
// panjang yang harus dibaca seluruhnya untuk menemukan satu baris.
func mapDetailError(err error) (int, ErrorResponse, bool) {
	var notAvailable *inboxpladla.NotAvailableError

	switch {
	case errors.As(err, &notAvailable):
		// 501 — alamatnya ada, permintaannya sah, kemampuannya yang belum dibangun.
		return http.StatusNotImplemented, ErrorResponse{
			Code:    CodeWriteNotAvailable,
			Message: notAvailable.Reason(),
		}, true

	case errors.Is(err, inboxpladla.ErrNotAClaimList):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code: CodeNotAClaimList,
			Message: "Tampilan \"DATA PLA DLA XOL KLAIM\" tidak menampilkan klaim satu " +
				"per satu, sehingga ia tidak punya daftar maupun tabel ringkas. " +
				"Bukalah tampilan itu lewat bilah tab.",
		}, true

	case errors.Is(err, inboxpladla.ErrAdviceKindUnknown):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeAdviceKindUnknown,
			Message: "Jenis pemberitahuan harus PLA atau DLA.",
		}, true

	case errors.Is(err, inboxpladla.ErrRowNotFound):
		// 404, dan pesannya sengaja TIDAK membedakan "tidak ada" dari "bukan milik Anda".
		// Membedakannya memberi tahu penanya bahwa klaimnya ADA.
		return http.StatusNotFound, ErrorResponse{
			Code: CodeClaimNotFound,
			Message: "Klaim ini tidak ditemukan pada entitas yang sedang Anda buka, " +
				"atau tidak satu pun pemberitahuannya dikirimkan kepada Anda.",
		}, true

	case errors.Is(err, inboxpladla.ErrDocumentNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code: CodeDocumentNotFound,
			Message: "Dokumen ini tidak ditemukan, atau bukan bagian dari pemberitahuan " +
				"yang dikirimkan kepada Anda.",
		}, true

	case errors.Is(err, inboxpladla.ErrConversationNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code: CodeConversationNotFound,
			Message: "Percakapan ini tidak ditemukan pada klaim tersebut, atau tidak " +
				"menyangkut Anda.",
		}, true

	case errors.Is(err, inboxpladla.ErrConversationAlreadyAnswered):
		// 409, bukan 422: permintaannya sah seluruhnya — keadaan percakapannya yang
		// sudah berubah. Pesannya menyatakan pekerjaannya SUDAH selesai, bukan gagal,
		// supaya pengguna tidak mencoba lagi.
		return http.StatusConflict, ErrorResponse{
			Code: CodeConversationAnswered,
			Message: "Percakapan ini sudah dijawab. Setiap percakapan hanya menyimpan " +
				"satu balasan, sehingga balasan baru tidak dapat ditambahkan — muat " +
				"ulang rinciannya untuk membaca balasan yang sudah ada.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
