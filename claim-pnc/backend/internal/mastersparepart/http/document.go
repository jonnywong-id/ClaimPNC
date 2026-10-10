package masterspareparthttp

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/mastersparepart"
	usecase "claim-pnc/internal/mastersparepart/usecase"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxUploadBody membatasi besar badan permintaan unggah.
//
// Ia TIDAK memakai batas badan JSON modul ini: batas itu untuk amplop JSON, dan satu berkas
// jelas melampauinya. Nilainya mengikuti batas modul dokumen penunjang — satu angka untuk
// seluruh aplikasi — ditambah satu megabyte untuk amplop multipart dan medan lain. Tanpa
// kelebihan itu, berkas yang tepat sebesar batas akan ditolak karena amplopnya, dan pesannya
// akan menyebut ukuran berkas yang sebenarnya sah.
const maxUploadBody = dokumenpenunjang.BatasUkuranBerkas + (1 << 20)

// Kode galat jalur unggah.
//
// Keempatnya dipisahkan, bukan disatukan menjadi satu "unggah_gagal", karena yang
// membedakannya adalah APA YANG BOLEH DILAKUKAN PENGGUNA — dan itulah satu-satunya hal yang
// ingin diketahui pengguna saat unggahan gagal.
//
// Nilainya SAMA PERSIS dengan Master Panel supaya satu komponen layar dapat menangani
// keduanya tanpa dua tabel kode yang nyaris sama.
const (
	// CodeUploadInvalid: permintaannya salah — perbaiki lalu ulangi.
	CodeUploadInvalid = "unggah_tidak_sah"
	// CodeUploadTooLarge: berkasnya terlalu besar.
	CodeUploadTooLarge = "berkas_terlalu_besar"
	// CodeUploadUnavailable: layanan hulu gagal, belum ada yang tersimpan — aman diulang.
	CodeUploadUnavailable = "layanan_unggah_tidak_tersedia"
	// CodeUploadHalfDone: berkas terkirim tetapi catatannya gagal — JANGAN diulang.
	CodeUploadHalfDone = "unggah_separuh_jalan"
	// CodeDocumentMissing: sparepart ini belum punya dokumen.
	CodeDocumentMissing = "dokumen_belum_ada"
)

// UploadDocument menerima satu berkas dan menautkannya ke sparepart-nya.
//
// # Bentuknya multipart, bukan JSON base64
//
// Pega mengirim base64 karena Connect REST hanya bicara JSON. Antara peramban dan backend
// kita tidak ada batasan itu, dan multipart lebih hemat: base64 membengkakkan muatan sekitar
// sepertiga. Penyandian base64 tetap terjadi — di adapter `httpstorage` modul dokumen
// penunjang, tepat sebelum berkasnya dikirim ke layanan penyimpanan.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists || strings.TrimSpace(active.Alias) == "" {
		h.writeModuleError(w, r, errors.New("mastersparepart/http: portal aktif tidak dikenali"))
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeModuleError(w, r, mastersparepart.OneViolation("id", "Sparepart tidak dikenali."))
		return
	}

	// Batas dipasang di dua tempat, dan keduanya perlu. MaxBytesReader memutus koneksi
	// sebelum berkas raksasa masuk memori; pemeriksaan di domain menangkap yang lolos dari
	// situ — misalnya pemanggil yang tidak lewat HTTP.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBody)

	file, header, err := r.FormFile("berkas")
	if err != nil {
		// MaxBytesReader yang memutus juga mendarat di sini. Dibedakan supaya pesannya
		// menyebut ukuran, bukan "berkas tidak ditemukan" yang menyesatkan.
		if strings.Contains(err.Error(), "request body too large") {
			h.writeModuleError(w, r, tooLarge(err))
			return
		}
		h.writeModuleError(w, r, &mastersparepart.DocumentUploadError{
			Kind:    mastersparepart.UploadInvalid,
			Message: `Berkas tidak ditemukan pada permintaan. Sertakan bagian bernama "berkas".`,
			Err:     err,
		})
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			h.writeModuleError(w, r, tooLarge(err))
			return
		}
		h.writeModuleError(w, r, err)
		return
	}

	by, known := h.actor(r)
	if !known {
		h.writeError(w, r, errors.New("mastersparepart/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	doc, err := h.service.UploadDocument(r.Context(), usecase.UploadCommand{
		PortalAlias: active.Alias,
		SparepartID: id,
		FileName:    baseName(header.Filename),
		Content:     content,
		Note:        strings.TrimSpace(r.FormValue("catatan")),
		By:          by,
	}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusCreated, DocumentResponse{Data: toDocumentDTO(doc)})
}

// GetDocument mengembalikan dokumen sebuah sparepart.
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists || strings.TrimSpace(active.Alias) == "" {
		h.writeModuleError(w, r, errors.New("mastersparepart/http: portal aktif tidak dikenali"))
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeModuleError(w, r, mastersparepart.OneViolation("id", "Sparepart tidak dikenali."))
		return
	}

	doc, err := h.service.DocumentOf(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, DocumentResponse{Data: toDocumentDTO(doc)})
}

// tooLarge membungkus pemutusan MaxBytesReader menjadi galat berukuran.
func tooLarge(err error) error {
	return &mastersparepart.DocumentUploadError{
		Kind: mastersparepart.UploadTooLarge,
		Message: "Ukuran berkas melebihi batas " +
			strconv.Itoa(dokumenpenunjang.BatasUkuranBerkas>>20) +
			" MB. Perkecil berkasnya lalu unggah ulang.",
		Err: err,
	}
}

// baseName membuang komponen jalur dari nama yang dikirim peramban.
//
// Sebagian peramban lama — dan sebagian klien bukan peramban — mengirim nama lengkap berikut
// jalurnya. Nama itu kelak dibersihkan modul dokumen penunjang, tetapi pembersihan itu
// membuang pemisah jalur TANPA membuang bagian jalurnya, sehingga `C:\Data\Foto.png` akan
// menjadi `CDataFotopng`. Dipotong lebih dulu supaya yang tersimpan tetap nama berkasnya.
func baseName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, `\`, "/")
	return filepath.Base(name)
}

// mapUploadError memetakan galat unggah menjadi status dan badan respons.
//
// Dipanggil dari mapError sebagai cabang terakhirnya.
func mapUploadError(err error) (int, ErrorResponse, bool) {
	var upload *mastersparepart.DocumentUploadError
	if !errors.As(err, &upload) {
		if errors.Is(err, mastersparepart.ErrDocumentMissing) {
			return http.StatusNotFound, ErrorResponse{
				Code:    CodeDocumentMissing,
				Message: "Sparepart ini belum punya dokumen.",
			}, true
		}
		return 0, ErrorResponse{}, false
	}

	switch upload.Kind {
	case mastersparepart.UploadInvalid:
		// 422: bentuk permintaannya benar, isinya yang salah. Menempel pada isian "berkas"
		// supaya pesannya muncul di tempat pengguna memilih berkasnya.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeUploadInvalid,
			Message: upload.Message,
			Detail:  []ViolationDTO{{Field: "berkas", Message: upload.Message}},
		}, true

	case mastersparepart.UploadTooLarge:
		// 413 adalah kode yang memang untuk ini, dan sebagian proxy sudah menjawabnya
		// sendiri sebelum permintaan sampai ke kita. Memakai kode yang sama membuat kedua
		// sumber terbaca serupa oleh layar.
		return http.StatusRequestEntityTooLarge, ErrorResponse{
			Code:    CodeUploadTooLarge,
			Message: upload.Message,
			Detail:  []ViolationDTO{{Field: "berkas", Message: upload.Message}},
		}, true

	case mastersparepart.UploadUnavailable:
		// 503, dan itu disengaja: layanan hulu yang sedang mati bukan kesalahan pengguna,
		// dan 503 menyatakan "coba lagi nanti" kepada setiap perantara yang membacanya.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:    CodeUploadUnavailable,
			Message: upload.Message,
		}, true

	case mastersparepart.UploadHalfDone:
		// 500, BUKAN 503. Perbedaannya penting: 503 mengundang pengulangan, dan mengulang
		// unggahan yang separuh berhasil menumpuk berkas ganda di layanan penyimpanan.
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeUploadHalfDone,
			Message: upload.Message,
		}, true

	case mastersparepart.UploadMisconfigured:
		// 503: salah konfigurasi di sisi kita, dan pengguna tidak dapat berbuat apa pun.
		// Mengulang tidak membantu, tetapi menyatakannya 500 akan membuatnya terbaca sebagai
		// cacat program — padahal yang kurang adalah pemasangan.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:    CodeUploadUnavailable,
			Message: upload.Message,
		}, true
	}
	return 0, ErrorResponse{}, false
}

// actor mengambil pengguna yang sedang memanggil.
//
// Ketiadaannya TIDAK dilayani sebagai anonim: `INPUTOPERATOR` pada baris lampiran adalah
// satu-satunya jejak siapa yang mengunggah, dan `D-59` menjadikan jejak audit kontrol
// pengimbang satu-satunya karena tidak ada pemisahan tugas. Baris tanpa pengunggah menghapus
// kontrol itu tanpa ada yang menyadarinya.
func (h *Handler) actor(r *http.Request) (usecase.Actor, bool) {
	by, known := h.caller(r.Context())
	if !known {
		return usecase.Actor{}, false
	}
	return usecase.Actor{Login: strings.TrimSpace(by.Login)}, true
}

// toDocumentDTO memindahkan dokumen domain ke bentuk kawat.
func toDocumentDTO(doc mastersparepart.SparepartDocument) DocumentDTO {
	dto := DocumentDTO{
		DataID:      doc.DataID,
		ImageID:     doc.ImageID,
		Name:        doc.Name,
		MimeType:    doc.MimeType,
		Note:        doc.Note,
		UploadedBy:  doc.UploadedBy,
		SparepartID: doc.SparepartID,
	}
	if doc.UploadedAt != nil {
		dto.UploadedAt = doc.UploadedAt.Format(time.RFC3339)
	}
	return dto
}
