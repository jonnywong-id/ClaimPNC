package masterbengkelhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/platform/logging"
)

// Kode galat modul ini.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih
// terhalang keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth,
// dan menambah kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan
// sisanya ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama
// sehingga klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Begitu TKT-F1-004 diputuskan, pemetaan ini pindah ke tempat bersama dan berkas ini
// tinggal memakainya. Utang itu dicatat di docs/keputusan-implementasi.md.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"

	// CodeNameTaken dibedakan dari CodeValidationFailed dengan sengaja.
	//
	// Ia konflik KEADAAN, bukan isian yang cacat: namanya sah, dan yang salah hanyalah
	// bahwa nama itu sudah dipakai bengkel lain. Perbaikannya bukan "betulkan isian"
	// melainkan "buka baris yang sudah ada, atau pakai nama lain", dan layar
	// menanganinya berbeda.
	CodeNameTaken = "nama_bengkel_sudah_ada"

	// CodeLoginTaken dibedakan dengan alasan yang sama.
	CodeLoginTaken = "login_sudah_dipakai"

	// CodeDocumentNotFound berarti bengkelnya ada tetapi belum punya lampiran, atau
	// DOKUMENID-nya menunjuk baris yang tidak ada.
	CodeDocumentNotFound = "dokumen_tidak_ditemukan"

	// CodeDocumentEmpty dibedakan dari CodeDocumentNotFound dengan sengaja.
	//
	// Ia keadaan warisan: barisnya ADA, berkasnya tidak pernah tersimpan karena jalur unggah
	// Pega tidak pernah mengisi `IMAGEID`. Membedakannya membuat layar dapat menjelaskan
	// sebabnya, alih-alih menyatakan dokumennya tidak ada padahal keterangannya terlihat di
	// baris yang sama.
	CodeDocumentEmpty = "dokumen_tanpa_isi"

	// Kode kegagalan jalur unggah.
	//
	// Keempatnya dipisahkan, bukan disatukan menjadi satu "unggah_gagal", karena yang
	// membedakannya adalah APA YANG BOLEH DILAKUKAN PENGGUNA — satu-satunya hal yang ingin
	// diketahui pengguna saat unggahan gagal.
	//
	// Nilainya SAMA PERSIS dengan Master Sparepart dan Master Panel supaya satu komponen
	// layar dapat menangani ketiganya tanpa tiga tabel kode yang nyaris sama.

	// CodeUploadInvalid: permintaannya salah — perbaiki lalu ulangi.
	CodeUploadInvalid = "unggah_tidak_sah"
	// CodeUploadTooLarge: berkasnya terlalu besar.
	CodeUploadTooLarge = "berkas_terlalu_besar"
	// CodeUploadUnavailable: layanan hulu gagal, belum ada yang tersimpan — aman diulang.
	CodeUploadUnavailable = "layanan_unggah_tidak_tersedia"
	// CodeUploadHalfDone: berkas terkirim tetapi catatannya gagal — JANGAN diulang.
	CodeUploadHalfDone = "unggah_separuh_jalan"

	// CodeUnknownStatus muncul bila status yang diminta di luar "0", "1", "2".
	CodeUnknownStatus = "status_tidak_dikenal"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan
		// jaringan, cacat pemrograman — diserahkan ke penulis bersama, yang menjawab 500
		// dengan pesan umum dan menaruh rinciannya di log saja. Rincian galat internal
		// tidak pernah dikirim ke peramban.
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

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai
// seluruh modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *masterbengkel.ValidationError

	if status, body, ok := mapUploadError(err); ok {
		return status, body, true
	}

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 berarti ada cacat di frontend, 422
		// berarti pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5).
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, p := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: p.Field, Message: p.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, masterbengkel.ErrNameTaken):
		// 409, bukan 422. Padanan pesan `Activity/ValidateMasterBengkel`:
		// "Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain."
		return http.StatusConflict, ErrorResponse{
			Code:    CodeNameTaken,
			Message: "Nama bengkel tersebut telah digunakan. Silakan ganti dengan nama yang lain.",
			Detail: []ViolationDTO{{
				Field:   "nama_bengkel",
				Message: "Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain.",
			}},
		}, true

	case errors.Is(err, masterbengkel.ErrLoginTaken):
		// Pesannya mengikuti `Activity/ValidationLoginBengkel_act` step 6, dengan satu
		// perbaikan: salah ketik "talah" pada rule aslinya tidak dibawa. Ia salah ketik,
		// bukan istilah — dan `D-13` menuntut teks layar diikuti, bukan salah ketiknya.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeLoginTaken,
			Message: "Login aplikasi tersebut telah dipakai, tolong ubah login aplikasi.",
			Detail: []ViolationDTO{{
				Field:   "login_aplikasi",
				Message: "Login aplikasi tersebut telah dipakai, tolong ubah login aplikasi.",
			}},
		}, true

	case errors.Is(err, masterbengkel.ErrUnknownStatus):
		// 422 dan menempel pada isian `status`: ia memang datang dari permintaan, dan
		// satu-satunya cara ia salah adalah klien mengirim nilai di luar ketiganya.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeUnknownStatus,
			Message: "Status persetujuan tidak dikenal.",
			Detail: []ViolationDTO{{
				Field:   "status",
				Message: `Status hanya boleh "0" menunggu, "1" disetujui, atau "2" ditolak.`,
			}},
		}, true

	case errors.Is(err, masterbengkel.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Bengkel yang dimaksud tidak ditemukan. Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	case errors.Is(err, masterbengkel.ErrDocumentNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeDocumentNotFound,
			Message: "Bengkel ini belum punya dokumen terlampir.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}

// mapUploadError memetakan kegagalan jalur unggah menurut GOLONGANNYA.
//
// Ia dipisahkan dari mapError supaya pemetaan ini terbaca utuh di satu tempat, dan supaya
// perbandingannya dengan Master Sparepart dan Master Panel — yang bentuknya sama persis —
// dapat dilakukan tanpa menelusuri satu switch panjang.
func mapUploadError(err error) (int, ErrorResponse, bool) {
	var upload *masterbengkel.DocumentUploadError
	if !errors.As(err, &upload) {
		return 0, ErrorResponse{}, false
	}

	switch upload.Kind {
	case masterbengkel.UploadInvalid:
		// 422: bentuk permintaannya benar, isinya yang salah. Menempel pada isian "berkas"
		// supaya pesannya muncul di tempat pengguna memilih berkasnya.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeUploadInvalid,
			Message: upload.Message,
			Detail:  []ViolationDTO{{Field: "berkas", Message: upload.Message}},
		}, true

	case masterbengkel.UploadTooLarge:
		// 413 adalah kode yang memang untuk ini, dan sebagian proxy sudah menjawabnya
		// sendiri sebelum permintaan sampai ke kita. Memakai kode yang sama membuat kedua
		// sumber terbaca serupa oleh layar.
		return http.StatusRequestEntityTooLarge, ErrorResponse{
			Code:    CodeUploadTooLarge,
			Message: upload.Message,
			Detail:  []ViolationDTO{{Field: "berkas", Message: upload.Message}},
		}, true

	case masterbengkel.UploadUnavailable:
		// 503, dan itu disengaja: layanan hulu yang sedang mati bukan kesalahan pengguna,
		// dan 503 menyatakan "coba lagi nanti" kepada setiap perantara yang membacanya.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:    CodeUploadUnavailable,
			Message: upload.Message,
		}, true

	case masterbengkel.UploadHalfDone:
		// 500, BUKAN 503. Perbedaannya penting: 503 mengundang pengulangan, dan mengulang
		// unggahan yang separuh berhasil menumpuk berkas ganda di layanan penyimpanan.
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeUploadHalfDone,
			Message: upload.Message,
		}, true

	case masterbengkel.UploadMisconfigured:
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
