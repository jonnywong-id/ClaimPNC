package masterbengkelhttp

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxDocumentRequest membatasi besar permintaan unggahan.
//
// Ia sedikit lebih besar dari masterbengkel.MaxDocumentBytes: satu permintaan multipart
// membawa batas bagian, nama berkas, dan tipe medianya di samping isi berkasnya. Menyamakan
// keduanya akan menolak berkas yang ukurannya tepat di batas — dan pesannya akan menyebut
// batas yang menurut pengguna belum terlampaui.
const maxDocumentRequest = masterbengkel.MaxDocumentBytes + (1 << 20)

// documentFormField adalah nama bagian multipart yang memuat berkasnya.
const documentFormField = "berkas"

// UploadDocument menangani POST /master/bengkel/{id}/dokumen.
//
// Padanan tombol "Upload Document" pada `Section/BrowseMasterHE`, yang di Pega membuka flow
// action `UploadDocument` (@baseclass 01-01-89) berisi satu pemilih berkas.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New("masterbengkel/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterbengkel.ErrNotFound)
		return
	}

	// Badan permintaan dibatasi SEBELUM diurai. Tanpa ini, satu berkas raksasa sudah
	// telanjur ditarik seluruhnya sebelum ditolak.
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentRequest)
	if err := r.ParseMultipartForm(maxDocumentRequest); err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Berkas tidak dapat dibaca. Pastikan ukurannya wajar.",
		})
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			// Berkas sementara yang ditulis ParseMultipartForm ke disk dihapus. Tanpa
			// ini, setiap unggahan meninggalkan salinannya di direktori sementara server.
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile(documentFormField)
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: fmt.Sprintf("Berkas belum dilampirkan pada bagian %q.", documentFormField),
		})
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Berkas tidak dapat dibaca sampai selesai.",
		})
		return
	}

	// Nama berkas diambil dari header multipart, lalu DIPANGKAS menjadi nama dasarnya.
	// Peramban tertentu mengirim jalur lengkap, dan jalur yang ikut tersimpan membocorkan
	// susunan direktori pengunggah — sekaligus membuka path traversal bila kelak namanya
	// dipakai menyusun jalur berkas.
	name := filepath.Base(header.Filename)

	document, err := h.service.UploadDocument(r.Context(), active.Alias,
		usecase.Actor{Login: by.Login},
		masterbengkel.UploadInput{
			WorkshopID: id,
			FileName:   name,
			Content:    content,
		})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusCreated, DocumentResponse{
		Dokumen: toDocumentDTO(document),
		Portal:  active.Alias,
	})
}

// Document menangani GET /master/bengkel/{id}/dokumen — METADATA-nya saja.
//
// Isi berkasnya diambil terpisah lewat DocumentFile. Keduanya dipisah supaya layar dapat
// menampilkan nama dan waktu unggahnya tanpa menarik berkasnya, dan supaya dokumen warisan
// Pega yang isinya kosong dapat dikenali SEBELUM pengguna menekan unduh.
func (h *Handler) Document(w http.ResponseWriter, r *http.Request) {
	document, active, ok := h.readDocument(w, r)
	if !ok {
		return
	}

	h.writeResponse(w, r, http.StatusOK, DocumentResponse{
		Dokumen: toDocumentDTO(document),
		Portal:  active,
	})
}

// readDocument menyiapkan satu dokumen untuk endpoint pembacanya.
func (h *Handler) readDocument(w http.ResponseWriter, r *http.Request) (masterbengkel.Document, string, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return masterbengkel.Document{}, "", false
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterbengkel.ErrNotFound)
		return masterbengkel.Document{}, "", false
	}

	document, err := h.service.Document(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return masterbengkel.Document{}, "", false
	}
	return document, active.Alias, true
}

// DocumentDTO adalah bentuk metadata dokumen di kawat.
//
// Nama fieldnya bahasa Indonesia — ia KONTRAK, bukan nama internal (`D-80`). Bentuknya
// disamakan dengan DocumentDTO Master Sparepart dan Master Panel supaya satu komponen layar
// dapat melayani ketiganya tanpa tiga bentuk data yang nyaris sama.
//
// URL berkasnya sengaja TIDAK ada di sini. Ia dimiliki modul dokumen penunjang beserta masa
// berlakunya, dan menyalinnya ke sini akan membuat layar menampilkan tautan yang sudah
// kedaluwarsa tanpa ada yang tahu.
type DocumentDTO struct {
	ID         string `json:"id_dokumen"`
	ImageID    string `json:"image_id"`
	Name       string `json:"nama_berkas"`
	MimeType   string `json:"tipe_media"`
	HasFile    bool   `json:"berisi"`
	UploadedBy string `json:"diunggah_oleh"`
	UploadedAt string `json:"diunggah_pada"`
}

// DocumentResponse adalah jawaban kedua endpoint metadata dokumen.
type DocumentResponse struct {
	Dokumen DocumentDTO `json:"dokumen"`
	Portal  string      `json:"portal"`
}

func toDocumentDTO(d masterbengkel.Document) DocumentDTO {
	uploadedAt := ""
	if !d.UploadedAt.IsZero() {
		uploadedAt = d.UploadedAt.Format(time.RFC3339)
	}
	return DocumentDTO{
		ID:         d.ID,
		ImageID:    d.ImageID,
		Name:       d.Name,
		MimeType:   d.MimeType,
		HasFile:    d.HasFile(),
		UploadedBy: d.UploadedBy,
		UploadedAt: uploadedAt,
	}
}
