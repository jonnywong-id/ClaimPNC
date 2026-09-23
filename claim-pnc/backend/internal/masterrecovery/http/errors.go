package masterrecoveryhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterrecovery"
	"claim-pnc/internal/platform/logging"
)

// Kode galat modul ini. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih terhalang
// keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth, dan menambah
// kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya
// ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama sehingga
// klien tidak menghadapi dua bentuk galat yang berbeda.
const (
	CodeValidationFailed   = "validasi_gagal"
	CodeMalformedRequest   = "permintaan_cacat"
	CodeBatchTaken         = "nomor_batch_sudah_dipakai"
	CodePolicyNotFound     = "polis_tidak_ditemukan"
	CodeIssuerUnreachable  = "penerbit_va_tidak_terhubung"
	CodeIssuerUnconfigured = "penerbit_va_belum_terdaftar"
	CodeIssuerRejected     = "penerbit_va_menolak"
	CodeDocumentNotSaved   = "bukti_bayar_gagal_disimpan"
	CodeClaimLineEmpty     = "berkas_klaim_kosong"
	CodeClaimLineTooBig    = "berkas_klaim_terlalu_besar"
	CodeClaimLineBad       = "berkas_klaim_tidak_terbaca"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — galat portal, kegagalan basis data,
		// kegagalan jaringan, cacat pemrograman — diserahkan ke penulis bersama. Galat
		// portal dipetakan portalhttp.WithPortalError yang membungkusnya di cmd; sisanya
		// dijawab 500 dengan pesan umum, dan rinciannya hanya masuk log.
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

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang
// lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *masterrecovery.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di kolomnya
		// (`docs/Steering/10-API-STRATEGY.md` §5).
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, v := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: v.Field, Message: v.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
			Detail:  detail,
		}, true

	case errors.Is(err, masterrecovery.ErrBatchTaken):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan penyimpanan
		// saat ini. Mencoba sekali lagi biasanya berhasil, dan pesannya mengatakan itu.
		//
		// Keadaan ini adalah yang di sistem lama TIDAK PERNAH terlihat: procedure-nya
		// melewati penyisipan tanpa satu pun pesan, sehingga petugas mengira batch-nya
		// tersimpan.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeBatchTaken,
			Message: "Nomor batch baru saja dipakai petugas lain. Simpan sekali lagi — isian Anda tidak hilang.",
		}, true

	case errors.Is(err, masterrecovery.ErrPolicyNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodePolicyNotFound,
			Message: "Nomor polis tidak ditemukan. Periksa kembali nomornya.",
		}, true

	case errors.Is(err, masterrecovery.ErrIssuerUnconfigured):
		// 500, dan pesannya menyebut apa yang harus diperbaiki. Ini bukan kesalahan
		// pengguna dan tidak dapat ditolong dengan mencoba ulang: baris layanannya belum
		// ada di katalog entitas itu.
		//
		// Nama tabelnya disebut karena ia objek milik kita sendiri — bukan rincian galat
		// driver dan bukan data nasabah — dan tanpa itu administrator tidak punya petunjuk
		// apa pun untuk menindaklanjuti.
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeIssuerUnconfigured,
			Message: "Layanan penerbit virtual account belum terdaftar untuk entitas ini pada POOLDATA.GCNM_CONNECT_REST. Hubungi administrator Claim PNC.",
		}, true

	case errors.Is(err, masterrecovery.ErrIssuerUnreachable):
		// 502: yang gagal adalah sistem di seberang, bukan permintaan ini. Dibedakan dari
		// 500 supaya pemantauan tidak membaca gangguan pihak lain sebagai cacat aplikasi.
		return http.StatusBadGateway, ErrorResponse{
			Code:    CodeIssuerUnreachable,
			Message: "Layanan penerbit virtual account sedang tidak dapat dihubungi. Coba lagi beberapa saat.",
		}, true

	case errors.Is(err, masterrecovery.ErrIssuerRejected):
		// 502 juga, tetapi sebabnya berbeda: layanan MENJAWAB dan menolak. Mencoba ulang
		// dengan isian yang sama tidak akan mengubah apa pun, dan pesannya mengatakan itu.
		return http.StatusBadGateway, ErrorResponse{
			Code:    CodeIssuerRejected,
			Message: "Layanan penerbit virtual account menolak permintaan ini. Periksa Client ID dan nama principal, lalu coba lagi.",
		}, true

	case errors.Is(err, masterrecovery.ErrDocumentNotSaved):
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeDocumentNotSaved,
			Message: "Bukti bayar gagal disimpan. Batch belum dicatat — coba unggah ulang.",
		}, true

	case errors.Is(err, masterrecovery.ErrClaimLineEmpty):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeClaimLineEmpty,
			Message: "Berkas daftar klaim tidak memuat satu baris pun yang dapat dibaca.",
		}, true

	case errors.Is(err, masterrecovery.ErrClaimLineTooMany):
		return http.StatusRequestEntityTooLarge, ErrorResponse{
			Code:    CodeClaimLineTooBig,
			Message: "Berkas daftar klaim terlalu besar. Pecah menjadi beberapa berkas lalu unggah bergantian.",
		}, true

	case errors.Is(err, masterrecovery.ErrClaimLineUnreadable):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeClaimLineBad,
			Message: "Berkas daftar klaim tidak dapat dibaca. Pakai berkas CSV dengan dua kolom: nomor polis dan nilai klaim.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
