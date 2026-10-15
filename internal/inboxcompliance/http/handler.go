package inboxcompliancehttp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Compliance.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
//
// # GetCaller hanya dipakai jalur TULIS
//
// Kedua jalur baca tidak membutuhkannya: antreannya WORKBASKET, yakni antrean bersama yang
// isinya sama bagi setiap petugas Compliance.
//
// Yang membutuhkannya adalah pengiriman ke Post Audit — bukan untuk menentukan apa yang
// boleh dikirim, melainkan untuk MENCATAT siapa yang mengirim. Tabelnya tidak punya kolom
// pengirim, sehingga log adalah satu-satunya tempat identitas itu tersimpan. Lihat catatan
// di usecase.Caller.
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

// NewHandler membentuk handler modul Inbox Compliance.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-compliance/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab dan kolomnya adalah bentuk layar, bukan data
// entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-compliance.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		inboxcompliance.QueryInput{Tab: query.Get("tab")},
		inboxcompliance.Pagination{
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

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan inboxcompliance.Pagination.Normalize
// membetulkannya menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan
// membuat layar gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman
// pertama adalah jawaban yang selalu masuk akal.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// SendPostAudit menangani POST /api/inbox-compliance/post-audit.
//
// POST, bukan PUT: ia MEMBUAT baris baru, dan pemanggilan kedua dengan badan yang sama
// membuat baris kedua — bukan menimpa yang pertama. Menandainya PUT akan menjanjikan
// idempotensi yang tidak ada (lihat catatan di usecase.SendToPostAudit).
func (h *Handler) SendPostAudit(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxcompliance.ErrCallerUnknown)
		return
	}

	var body SendPostAuditRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// 400, bukan 422: badan yang tidak dapat diurai berarti klien salah membentuk
		// permintaan, bukan pengguna salah mengisi (`10-API-STRATEGY.md` §5).
		h.writeError(w, r, errMalformedBody)
		return
	}

	sent, err := h.service.SendToPostAudit(
		r.Context(),
		active.Alias,
		usecase.Caller{Login: caller.Login},
		inboxcompliance.PostAuditInput{
			Reference: strings.TrimSpace(body.Reference),
			Remarks:   body.Remarks,
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// 201 beserta barisnya, bukan 204: layar membutuhkan nomor yang terbit untuk
	// menampilkannya pada pesan berhasil, dan mengambilnya lewat permintaan kedua berarti
	// menebak baris mana yang baru saja dibuat.
	h.writeJSON(w, r, http.StatusCreated, toSendPostAuditResponse(sent, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
//
// Hanya jalur TULIS yang memerlukannya. Kedua jalur baca tidak — antreannya workbasket,
// yang isinya sama bagi setiap petugas.
func (h *Handler) readCaller(r *http.Request) (Caller, bool) {
	if h.caller == nil {
		return Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || caller.Login == "" {
		return Caller{}, false
	}
	return caller, true
}

// OpenChecker menangani GET /api/inbox-compliance/{referensi}.
//
// # Kenapa jalurnya memuat kunci klaim, bukan parameter query
//
// Karena yang dibuka adalah SATU klaim tertentu, dan jalur yang menyebutnya dapat
// ditandai, dibagikan, dan muncul di riwayat peramban sebagai alamat tersendiri. Di Pega
// pun begitu: menekan Nomor Case membuka assignment-nya, bukan menyaring daftar.
//
// Kuncinya `PZINSKEY`, yang memuat spasi dan tanda hubung — layar WAJIB mengkodekannya.
// chi sudah mendekodekannya kembali saat dibaca lewat URLParam.
func (h *Handler) OpenChecker(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	reference := strings.TrimSpace(chi.URLParam(r, "referensi"))

	opened, err := h.service.OpenChecker(r.Context(), active.Alias, reference)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toCheckerResponse(opened, active.Alias))
}

// SubmitDecision menangani POST /api/inbox-compliance/{referensi}/keputusan.
//
// Padanan tombol "Simpan Data" pada `Section/ComplianceChecker-Section.xml`.
//
// POST, bukan PUT, meski kuerinya MERGE dan pemanggilan kedua menimpa yang pertama. Dua
// alasan: pada pilihan Bayar/PostAudit ia MENERBITKAN baris baru — akibat yang tidak
// idempoten — dan jalurnya bukan alamat sumber daya keputusan itu sendiri melainkan aksi
// atas klaimnya (`10-API-STRATEGY.md` §2, aksi bisnis dimodelkan sebagai peristiwa).
func (h *Handler) SubmitDecision(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxcompliance.ErrCallerUnknown)
		return
	}

	var body SubmitDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, r, errMalformedBody)
		return
	}

	decided, err := h.service.SubmitDecision(
		r.Context(),
		active.Alias,
		usecase.Caller{Login: caller.Login},
		inboxcompliance.DecisionInput{
			// Kunci klaim diambil dari JALUR, bukan dari badan permintaan. Badan yang
			// menyebut klaim berbeda dari jalurnya akan diam-diam memutuskan klaim yang
			// salah; dengan satu sumber, keadaan itu tidak mungkin terjadi.
			Reference: strings.TrimSpace(chi.URLParam(r, "referensi")),
			Choice:    body.Choice,
			Note:      body.Note,
			Comments:  toCommentInputs(body.Comments),
			Action:    strings.TrimSpace(body.Action),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// 200, bukan 201: yang utama adalah keputusan atas klaim yang SUDAH ada, dan
	// menyimpannya ulang menimpa yang sebelumnya. Baris Post Audit yang kadang ikut
	// terbit dibawa di dalam badan, bukan dijadikan alasan mengubah kodenya — layar tetap
	// perlu membedakan keduanya lewat isi, bukan lewat status.
	h.writeJSON(w, r, http.StatusOK, toSubmitDecisionResponse(decided, active.Alias))
}

// OpenDocument menerbitkan tautan baru untuk satu dokumen klaim.
//
// Kedua kunci — klaim dan dokumen — diambil dari JALUR, bukan dari badan permintaan.
// Alasannya sama dengan SubmitDecision: dengan satu sumber, badan yang menyebut klaim
// berbeda dari jalurnya tidak mungkin terjadi.
func (h *Handler) OpenDocument(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Pelaku WAJIB terbaca: `UserInput` ikut dikirim ke layanan penyimpanan, dan tanpa
	// itu pembukaan berkas tidak tercatat atas nama siapa pun.
	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxcompliance.ErrCallerUnknown)
		return
	}

	opened, err := h.service.OpenDocument(
		r.Context(),
		active.Alias,
		usecase.Caller{Login: caller.Login},
		strings.TrimSpace(chi.URLParam(r, "referensi")),
		strings.TrimSpace(chi.URLParam(r, "dokumen")),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, OpenDocumentResponse{
		Name:      opened.Document.Name,
		ViewerURL: opened.ViewerURL,
		Portal:    active.Alias,
	})
}

// batasUnggah membatasi ukuran satu berkas yang diunggah.
//
// # Kenapa ADA batasnya, dan kenapa angkanya ini
//
// Tanpa batas, satu permintaan dapat menghabiskan memori proses — berkasnya dibaca utuh
// sebelum di-Base64-kan. 20 MiB adalah TEBAKAN: export tidak menyebut batas apa pun, dan
// `pzMultiFilePath` tidak membawa angka.
//
// Ia sengaja ditulis sebagai konstanta bernama, bukan angka di tengah kode, supaya
// penggantinya jelas begitu batas sebenarnya diketahui.
const batasUnggah = 20 << 20

// UploadDocument menerima satu berkas dan menempelkannya ke klaim.
//
// `multipart/form-data`, bukan JSON ber-Base64: Base64 membengkakkan badan permintaan
// sepertiga, dan pembengkakan itu ditanggung dua kali — sekali di peramban, sekali lagi
// saat adapter meng-Base64-kannya untuk layanan penyimpanan. Dengan multipart, hanya
// yang kedua yang terjadi.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxcompliance.ErrCallerUnknown)
		return
	}

	// Batas dipasang pada BADAN permintaan, bukan hanya pada potongannya: tanpa ini,
	// badan yang jauh lebih besar tetap terbaca seluruhnya sebelum ditolak.
	r.Body = http.MaxBytesReader(w, r.Body, batasUnggah)
	if err := r.ParseMultipartForm(batasUnggah); err != nil {
		h.writeError(w, r, errMalformedBody)
		return
	}

	berkas, kepala, err := r.FormFile("berkas")
	if err != nil {
		h.writeError(w, r, errMalformedBody)
		return
	}
	defer berkas.Close()

	isi, err := io.ReadAll(berkas)
	if err != nil {
		h.writeError(w, r, errMalformedBody)
		return
	}

	// Ekstensi diambil dari nama berkas, bukan dari `Content-Type` yang dikirim
	// peramban — Pega pun memetakannya dari ekstensi (`InsertDokumenPNC` langkah 16).
	nama := kepala.Filename
	ekstensi := strings.TrimPrefix(filepath.Ext(nama), ".")

	tersimpan, err := h.service.UploadDocument(
		r.Context(),
		active.Alias,
		usecase.Caller{Login: caller.Login},
		usecase.UploadInput{
			Reference:   strings.TrimSpace(chi.URLParam(r, "referensi")),
			FileName:    nama,
			Extension:   ekstensi,
			Category:    strings.TrimSpace(r.FormValue("kategori")),
			SubCategory: strings.TrimSpace(r.FormValue("sub_kategori")),
			Note:        strings.TrimSpace(r.FormValue("catatan")),
			Content:     isi,
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// 201: satu lampiran BARU terbit, dan ia punya identitas sendiri.
	h.writeJSON(w, r, http.StatusCreated, toDocumentDTO(tersimpan))
}

// DeleteDocument menghapus satu lampiran klaim.
func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxcompliance.ErrCallerUnknown)
		return
	}

	if err := h.service.DeleteDocument(
		r.Context(),
		active.Alias,
		usecase.Caller{Login: caller.Login},
		strings.TrimSpace(chi.URLParam(r, "referensi")),
		strings.TrimSpace(chi.URLParam(r, "dokumen")),
	); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GenerateRejectLetter menerbitkan Surat Penolakan lalu melampirkannya pada klaim.
//
// `POST`, dan jalurnya `surat-penolakan` — bukan `dokumen/...`. Dua alasan: ia MENERBITKAN
// dokumen baru alih-alih menerima yang sudah jadi, dan isinya dibentuk server dari isian
// form, bukan diunggah peramban. Menaruhnya di bawah `dokumen` akan menyamarkannya sebagai
// unggahan biasa.
//
// Kunci klaim diambil dari JALUR, bukan dari badan permintaan — alasan yang sama dengan
// SubmitDecision: badan yang menyebut klaim berbeda dari jalurnya akan diam-diam
// menerbitkan surat penolakan atas klaim yang salah, dan surat itu keluar ke nasabah.
//
// Jawabannya 200, bukan 201: menekan tombol dua kali MENGGANTI surat yang sudah ada alih-
// alih menambah satu lagi, sehingga tidak selalu ada sumber daya baru yang tercipta.
// Mana yang terjadi dibawa di dalam badan lewat `mengganti_surat_sebelumnya`.
func (h *Handler) GenerateRejectLetter(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Pelaku WAJIB terbaca: namanya tercatat sebagai pengunggah surat, dan surat penolakan
	// adalah dokumen yang keluar ke nasabah — `D-59` menjadikan jejak audit satu-satunya
	// kontrol pengimbang karena tidak ada pemisahan tugas.
	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxcompliance.ErrCallerUnknown)
		return
	}

	var body GenerateRejectLetterRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, r, errMalformedBody)
		return
	}

	issued, err := h.service.GenerateRejectLetter(
		r.Context(),
		active.Alias,
		usecase.Caller{Login: caller.Login},
		usecase.RejectLetterInput{
			Reference:     strings.TrimSpace(chi.URLParam(r, "referensi")),
			Recipient:     body.Recipient,
			Position:      body.Position,
			PatientName:   body.PatientName,
			IncidentPlace: body.IncidentPlace,
			IncidentDate:  body.IncidentDate,
			DischargeDate: body.DischargeDate,
			PaidAmount:    body.PaidAmount,
			PaymentDate:   body.PaymentDate,
			Reasons:       body.Reasons,
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, GenerateRejectLetterResponse{
		Document: toDocumentDTOs([]inboxcompliance.Document{issued.Document})[0],
		Replaced: issued.Replaced,
		Portal:   active.Alias,
	})
}

// ListDocumentsInCategory menyerahkan lampiran klaim pada satu kategori.
//
// `GET`, dan di sini ia memang pembacaan murni — berbeda dari penerbitan tautan yang
// `POST` karena menerbitkan sesuatu yang berumur terbatas.
//
// Kategorinya parameter kueri `?kategori=`, bukan bagian jalur. Ia PENYARING atas dokumen
// klaim, bukan sumber daya tersendiri: tanpa kategori, jawabannya seluruh dokumen klaim
// itu — dan tombol "Ubah Kategori Dok" memang membutuhkan yang seluruhnya.
func (h *Handler) ListDocumentsInCategory(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	documents, err := h.service.ListDocumentsInCategory(
		r.Context(),
		active.Alias,
		strings.TrimSpace(chi.URLParam(r, "referensi")),
		strings.TrimSpace(r.URL.Query().Get("kategori")),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, ListDocumentsResponse{
		Documents: toDocumentDTOs(documents),
		Portal:    active.Alias,
	})
}

// ChangeDocumentCategory memindahkan satu lampiran ke kategori lain.
//
// `POST` atas sub-sumber daya aksi — konvensi modul ini. PATCH lebih tepat secara makna
// tetapi belum dikenal klien HTTP bersama; lihat catatan pada rutenya.
//
// Kedua kunci — klaim dan dokumen — diambil dari JALUR; hanya kategori tujuannya yang
// datang dari badan. Alasannya sama dengan SubmitDecision: badan yang menyebut klaim
// berbeda dari jalurnya akan diam-diam memindahkan dokumen milik klaim lain.
func (h *Handler) ChangeDocumentCategory(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Pelaku WAJIB terbaca: perpindahan kategori mengubah apa yang dianggap lengkap, dan
	// jejak audit adalah satu-satunya kontrol pengimbang (`D-59`).
	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxcompliance.ErrCallerUnknown)
		return
	}

	var body ChangeDocumentCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, r, errMalformedBody)
		return
	}

	if err := h.service.ChangeDocumentCategory(
		r.Context(),
		active.Alias,
		usecase.Caller{Login: caller.Login},
		strings.TrimSpace(chi.URLParam(r, "referensi")),
		strings.TrimSpace(chi.URLParam(r, "dokumen")),
		strings.TrimSpace(body.Category),
	); err != nil {
		h.writeError(w, r, err)
		return
	}

	// 204: tidak ada badan yang berguna dikembalikan, dan layar menyegarkan daftar
	// periksanya sendiri — pencacah "Total Sudah Diunggah" berubah pada DUA kategori
	// sekaligus, asal dan tujuan, sehingga mengirim balik satu baris pun tidak cukup.
	w.WriteHeader(http.StatusNoContent)
}
