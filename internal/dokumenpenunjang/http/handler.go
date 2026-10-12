// Package dokumenpenunjanghttp melayani unggah dan daftar dokumen penunjang klaim.
package dokumenpenunjanghttp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/dokumenpenunjang/usecase"

	portalhttp "claim-pnc/internal/portal/http"
)

// Service adalah bagian usecase yang dipakai handler ini.
type Service interface {
	Upload(ctx context.Context, cmd usecase.UploadCommand) (dokumenpenunjang.Document, error)
	List(ctx context.Context, portalAlias, nomorKlaim string) ([]dokumenpenunjang.Document, error)
}

// Caller adalah identitas pengguna yang sedang masuk.
type Caller struct {
	Login string
}

// GetCaller membaca identitas pengguna dari konteks permintaan.
type GetCaller func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan dokumen penunjang.
type Handler struct {
	service     Service
	getCaller   GetCaller
	logger      *slog.Logger
	writeJSON   JSONWriter
	writeErrorF ErrorWriter
	location    *time.Location
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service   Service
	GetCaller GetCaller
	Logger    *slog.Logger

	WriteJSON           JSONWriter
	FallbackErrorWriter ErrorWriter

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	Location *time.Location
}

// NewHandler membentuk handler modul dokumen penunjang.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}
	return &Handler{
		service:     o.Service,
		getCaller:   o.GetCaller,
		logger:      o.Logger,
		writeJSON:   o.WriteJSON,
		writeErrorF: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
		location:    location,
	}
}

func jakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	// Basis image tanpa tzdata tidak boleh membuat modul ini gagal; zona tetap +7 sudah
	// benar untuk WIB, yang tidak mengenal daylight saving.
	return time.FixedZone("WIB", 7*60*60)
}

// List melayani GET daftar dokumen sebuah klaim.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	nomor := strings.TrimSpace(chi.URLParam(r, "nomor"))
	if nomor == "" {
		writeBadRequest(h.writeJSON, w, r, "Nomor klaim wajib diisi.")
		return
	}

	alias, ok := h.requirePortal(w, r)
	if !ok {
		return
	}

	daftar, err := h.service.List(r.Context(), alias, nomor)
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, ListResponse{
		Data: dariDaftar(daftar, h.location, time.Now()),
	})
}

// Upload melayani POST satu berkas.
//
// # Bentuknya multipart, bukan JSON base64
//
// Pega mengirim base64 karena Connect REST hanya bicara JSON. Antara peramban dan backend
// kita tidak ada batasan itu, dan multipart lebih baik pada dua hal yang nyata: muatannya
// tidak membengkak sepertiga, dan berkasnya dapat dibaca mengalir alih-alih dirakit utuh
// di memori lebih dulu.
//
// Penyandian base64 tetap terjadi — di adapter `httpstorage`, tepat sebelum dikirim ke
// layanan penyimpanan, karena di sanalah ia memang dituntut.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	nomor := strings.TrimSpace(chi.URLParam(r, "nomor"))
	if nomor == "" {
		writeBadRequest(h.writeJSON, w, r, "Nomor klaim wajib diisi.")
		return
	}

	alias, ok := h.requirePortal(w, r)
	if !ok {
		return
	}

	// Identitas yang tidak dikenali dipetakan ke ErrPengunggahKosong, bukan ke galat
	// internal: `USERINPUT` pada jejak dan `UserInput` pada muatan keduanya memuat siapa
	// yang mengunggah, dan mengosongkannya menghapus satu-satunya jejak itu (`D-59`).
	pemanggil, ada := Caller{}, false
	if h.getCaller != nil {
		pemanggil, ada = h.getCaller(r.Context())
	}
	if !ada || strings.TrimSpace(pemanggil.Login) == "" {
		h.writeErrorF(w, r, dokumenpenunjang.ErrPengunggahKosong)
		return
	}

	// Batas dipasang di DUA tempat, dan keduanya perlu.
	//
	// `MaxBytesReader` memutus koneksi ketika pengirim melampaui batas, sehingga berkas
	// 2 GB tidak sempat masuk memori sama sekali. Pemeriksaan di domain menangkap yang
	// lolos dari situ — misalnya pemanggil yang tidak lewat HTTP.
	//
	// Kelebihannya diberi ruang untuk batas multipart dan medan lain; tanpa itu, berkas
	// yang tepat sebesar batas akan ditolak karena amplopnya.
	r.Body = http.MaxBytesReader(w, r.Body, dokumenpenunjang.BatasUkuranBerkas+(1<<20))

	berkas, keterangan, err := r.FormFile("berkas")
	if err != nil {
		// `MaxBytesReader` yang memutus juga mendarat di sini. Dibedakan supaya pesannya
		// menyebut ukuran, bukan "berkas tidak ditemukan" yang menyesatkan.
		if strings.Contains(err.Error(), "request body too large") {
			h.writeErrorF(w, r, dokumenpenunjang.ErrBerkasTerlaluBesar)
			return
		}
		writeBadRequest(h.writeJSON, w, r,
			`Berkas tidak ditemukan pada permintaan. Sertakan bagian bernama "berkas".`)
		return
	}
	defer berkas.Close()

	isi, err := io.ReadAll(berkas)
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			h.writeErrorF(w, r, dokumenpenunjang.ErrBerkasTerlaluBesar)
			return
		}
		h.writeErrorF(w, r, err)
		return
	}

	dokumen, err := h.service.Upload(r.Context(), usecase.UploadCommand{
		PortalAlias: alias,
		Request: dokumenpenunjang.UploadRequest{
			ClaimNumber: nomor,
			// Nama unik seperti Pega — lihat dokumenpenunjang.NamaUnggah.
			FileName: dokumenpenunjang.NamaUnggah(time.Now(), strings.TrimSpace(r.FormValue("jenis_dokumen")),
				namaBerkasDari(keterangan.Filename)),
			DocumentType: strings.TrimSpace(r.FormValue("jenis_dokumen")),
			Content:      isi,
			By:           strings.TrimSpace(pemanggil.Login),
		},
	})
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusCreated, ItemResponse{
		Data: dariDokumen(dokumen, h.location, time.Now()),
	})
}

// requirePortal mengambil alias portal yang sedang dibuka.
//
// Ketiadaannya DITOLAK, tidak pernah dilayani portal utama sebagai cadangan — jatuh ke
// koneksi default berarti dokumen sebuah badan hukum dibaca atau ditulis lewat basis data
// badan hukum lain (`R-20`).
func (h *Handler) requirePortal(w http.ResponseWriter, r *http.Request) (string, bool) {
	aktif, ada := portalhttp.ActivePortalFrom(r.Context())
	if !ada || strings.TrimSpace(aktif.Alias) == "" {
		h.writeErrorF(w, r, errors.New("dokumenpenunjang/http: portal aktif tidak dikenali"))
		return "", false
	}
	return aktif.Alias, true
}

// namaBerkasDari membuang komponen jalur dari nama yang dikirim peramban.
//
// # Kenapa dipotong, padahal namanya nanti dibersihkan juga
//
// Sebagian peramban lama mengirim jalur penuh (`C:\Users\...\foto.pdf`). Pembersihan
// domain memang akan membuang tanda bagi jalurnya, tetapi ia juga akan menyerap seluruh
// nama folder ke dalam nama berkas — `CUsersAdministratorDokumenfotopdf`.
//
// Ia BUKAN pengaman path traversal: tidak ada satu pun jalur berkas yang dibentuk dari nama
// ini, sebab yang menentukan nama akhir adalah layanan penyimpanan. Ini soal nama yang
// terbaca.
func namaBerkasDari(nama string) string {
	rapi := strings.TrimSpace(nama)
	if i := strings.LastIndexAny(rapi, `/\`); i >= 0 {
		rapi = rapi[i+1:]
	}
	return rapi
}
