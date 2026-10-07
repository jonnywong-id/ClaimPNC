package inboxacceptopenprotectionhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat yang dikenali klien.
//
// Klien membedakan jenis galat lewat kode ini, bukan dengan mencocokkan teks pesan.
const (
	CodeBadRequest    = "permintaan_cacat"
	CodeNotFound      = "tidak_ditemukan"
	CodeConflict      = "konflik"
	CodeInternalError = "galat_internal"

	// CodeForbidden menandai pemanggil sudah masuk tetapi tidak berwenang.
	//
	// Dibedakan dari CodeNotFound dengan sengaja — lihat komentar ErrForbidden.
	CodeForbidden = "tidak_berwenang"
)

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// JSONWriter menuliskan badan respons.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// WriteError memetakan galat menjadi respons HTTP.
//
// # Kenapa "sudah diakseptasi" dijawab 409, dan pesannya berbunyi demikian
//
// Layar ini antrean BERSAMA (`Flow/CreateProtection_Flow.xml` menempatkannya di workbasket
// `ProtectionPNC`). Petugas kedua yang menekan tombol atas baris yang sama TIDAK sedang
// melakukan kesalahan — ia hanya kalah cepat. Pesan yang menyatakan keputusan sudah diambil
// membuatnya menutup form; pesan galat teknis membuatnya mencoba lagi.
//
// Rincian galat internal TIDAK PERNAH dikirim ke peramban (`11-CROSSCUTTING` §1.2 butir 5).
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return apierror.ContextWriter(logger, writeJSON, fallback, mapError, ErrorResponse{
		Code:    CodeInternalError,
		Message: "Terjadi kesalahan pada sistem.",
	})
}

// mapError memetakan galat yang dikenali modul ini menjadi status dan badan respons; nilai
// ketiga false bila galatnya bukan milik modul ini.
func mapError(err error) (int, ErrorResponse, bool) {
	switch {
	case errors.Is(err, inboxacceptopenprotection.ErrForbidden):
		// Pesannya menyebut SEBABNYA — access group — supaya petugas tahu apa yang
		// harus diminta, bukan sekadar bahwa ia ditolak. Nama group yang dimilikinya
		// TIDAK disebut: itu memberi tahu penyerang peta kewenangan aplikasi.
		return http.StatusForbidden, ErrorResponse{
			Code:    CodeForbidden,
			Message: "Access group Anda tidak berwenang atas layar akseptasi proteksi. Hubungi administrator bila seharusnya berhak.",
		}, true

	case errors.Is(err, inboxacceptopenprotection.ErrClaimNotSynced):
		// 409, bukan 500: tidak ada yang rusak — klaimnya belum ada di daftar klaim.
		// Pesannya menyebut apa yang harus dibereskan dan menegaskan keputusannya TIDAK
		// tersimpan, supaya petugas tidak mengira persetujuannya sudah berlaku.
		return http.StatusConflict, ErrorResponse{
			Code: CodeConflict,
			Message: "Keputusan tidak disimpan: klaim yang ditaut belum ada di daftar klaim, " +
				"sehingga perubahan Tanggal Kejadian tidak dapat diterapkan. Laporkan nomor klaimnya ke administrator.",
		}, true

	case errors.Is(err, inboxacceptopenprotection.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Permintaan proteksi tidak ditemukan.",
		}, true

	case errors.Is(err, inboxacceptopenprotection.ErrAlreadyDecided):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeConflict,
			Message: "Permintaan proteksi ini sudah diakseptasi. Muat ulang daftar untuk melihat keputusannya.",
		}, true

	case errors.Is(err, inboxacceptopenprotection.ErrIncomplete):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeConflict,
			Message: "Permintaan proteksi belum tertaut ke klaim, sehingga belum dapat diakseptasi.",
		}, true

	case errors.Is(err, inboxacceptopenprotection.ErrUnknownDecision):
		// 400, bukan 409: bentuk permintaannya yang salah, bukan keadaan datanya.
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeBadRequest,
			Message: `Keputusan harus "setuju" atau "tolak".`,
		}, true
	}
	return 0, ErrorResponse{}, false
}

// writeBadRequest menjawab permintaan yang cacat bentuknya.
func writeBadRequest(writeJSON JSONWriter, w http.ResponseWriter, r *http.Request, message string) {
	writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
		Code:    CodeBadRequest,
		Message: message,
	})
}
