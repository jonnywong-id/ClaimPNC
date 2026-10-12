package registrasihttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/oraerror"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Kode galat modul registrasi.
//
// Ia melanjutkan daftar milik modul auth, bukan menggantikannya: klien memakai satu
// mekanisme pembeda untuk seluruh aplikasi. Kontrak galat yang mengikat seluruh aplikasi
// adalah `TKT-F1-004`, yang masih terhalang keputusan Work Owner — sampai itu selesai,
// bentuk ini mengikuti bentuk yang sudah berjalan.
const (
	CodeValidationFailed     = "validasi_gagal"
	CodePolicyNotFound       = "polis_tidak_ditemukan"
	CodeClaimNotFound        = "klaim_tidak_ditemukan"
	CodeTaskNotFound         = "tugas_tidak_ditemukan"
	CodeTaskAlreadyClaimed   = "tugas_sudah_diambil"
	CodeNotTaskOwner         = "bukan_pemilik_tugas"
	CodeTaskAlreadyDone      = "tugas_sudah_selesai"
	CodeStageMismatch        = "tahap_tidak_bersesuai"
	CodeNotAvailableAtStage  = "tidak_tersedia_di_tahap"
	CodeInvalidAction        = "tindakan_tidak_sah"
	CodeExchangeRateNotFound = "kurs_tidak_ditemukan"
	CodeAccountNotFound      = "rekening_tidak_ditemukan"
	CodeCommitteeNotFound    = "komite_tidak_ditemukan"
	CodeNotCommitteeTurn     = "bukan_giliran_komite"
	CodeReportRegistered     = "laporan_sudah_diregistrasi"
	CodeDocumentUpload       = "unggah_dokumen_gagal"
	CodeAttachmentNotFound   = "lampiran_tidak_ditemukan"
	CodeDocumentLink         = "tautan_dokumen_tidak_tersedia"
	CodeDocumentDelete       = "hapus_dokumen_gagal"
	CodeMalformedRequest     = "permintaan_cacat"
	CodeInternalError        = "galat_internal"
	// CodeDatabaseError: penyimpanan/pembacaan tabel ditolak Oracle; pesannya memuat galat
	// Oracle apa adanya (Work Owner 2026-10-10).
	CodeDatabaseError           = "galat_basis_data"
	CodePremiumUnavailable      = "status_premi_tidak_terbaca"
	CodeCashierUnavailable      = "kasir_tidak_terhubung"
	CodePLASendUnavailable      = "kirim_pla_tidak_tersedia"
	CodeSurveyUnavailable       = "survey_tidak_tersedia"
	CodeSurveyCommitteeAwaiting = "komite_survey_menunggu"
	CodeCashierNoReply          = "kasir_tidak_menjawab"
	CodeCashierRejected         = "kasir_menolak"
)

// mapError memilih status HTTP dan badan respons untuk sebuah galat.
func mapError(err error) (int, ErrorResponse) {
	var validation *registrasi.ValidationError
	var registered *registrasi.ReportAlreadyRegisteredError
	var upload *registrasi.DocumentUploadError
	var cashierRejected *usecase.CashierRejectedError
	var linkExpired *registrasi.DocumentLinkExpiredError

	switch {
	case errors.As(err, &upload):
		status := map[registrasi.UploadFailure]int{
			registrasi.UploadInvalid:       http.StatusBadRequest,
			registrasi.UploadTooLarge:      http.StatusRequestEntityTooLarge,
			registrasi.UploadUnavailable:   http.StatusBadGateway,
			registrasi.UploadHalfDone:      http.StatusInternalServerError,
			registrasi.UploadMisconfigured: http.StatusInternalServerError,
		}[upload.Kind]
		if status == 0 {
			status = http.StatusInternalServerError
		}
		return status, ErrorResponse{Code: CodeDocumentUpload, Message: upload.Message}

	case errors.Is(err, registrasi.ErrAttachmentNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code: CodeAttachmentNotFound, Message: "This file is not attached to this claim.",
		}

	case errors.As(err, &linkExpired):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code: CodeDocumentLink,
			Message: "The storage service returned a link that already expired at " +
				linkExpired.ExpiresAt.In(clock.ZoneWIB).Format("02/01/2006 15:04") + " WIB. Try again.",
		}

	case errors.Is(err, registrasi.ErrAttachmentDeleteNotAllowed):
		return http.StatusForbidden, ErrorResponse{
			Code:    CodeDocumentDelete,
			Message: "Only the user who uploaded this file can delete it.",
		}

	case errors.Is(err, registrasi.ErrDocumentDeleteHalfDone):
		return http.StatusInternalServerError, ErrorResponse{
			Code: CodeDocumentDelete,
			Message: "The file was deleted from storage, but the attachment record could not be removed. " +
				"Report this to the administrator.",
		}

	case errors.Is(err, registrasi.ErrDocumentDeleteFailed):
		return http.StatusBadGateway, ErrorResponse{
			Code:    CodeDocumentDelete,
			Message: "The storage service did not delete this file. Nothing was changed; try again later.",
		}

	case errors.Is(err, registrasi.ErrDocumentLinkRenewFailed):
		return http.StatusBadGateway, ErrorResponse{
			Code:    CodeDocumentLink,
			Message: "The storage service could not renew the link to this file. Try again later.",
		}

	case errors.Is(err, registrasi.ErrDocumentLinkEmpty):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeDocumentLink,
			Message: "The storage service has not recorded a link for this file yet. Try again in a moment.",
		}

	case errors.Is(err, registrasi.ErrDocumentLinkUnavailable):
		return http.StatusBadGateway, ErrorResponse{
			Code:    CodeDocumentLink,
			Message: "The document storage metadata cannot be reached. Try again later.",
		}

	case errors.Is(err, registrasi.ErrDocumentTypeUnknown):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeDocumentUpload,
			Message: "This document type is not in the checklist of this claim's line of business.",
		}

	case errors.Is(err, registrasi.ErrDocumentFileEmpty):
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeDocumentUpload,
			Message: "Berkas kosong. Pilih berkas yang berisi lalu unggah ulang.",
		}

	case errors.As(err, &registered):
		// 409: berkasnya sudah menjadi klaim; menekan Register Klaim lagi tidak sah.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeReportRegistered,
			Message: "This Receive Document is already registered as claim " + registered.ClaimNumber + ".",
		}

	case errors.As(err, &validation):
		// 422, bukan 400: badan permintaan terbaca dengan benar dan bentuknya sah —
		// yang ditolak adalah ISINYA menurut aturan bisnis. Membedakan keduanya
		// menentukan apakah layar menandai kolom atau melaporkan cacat pemrograman.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:      CodeValidationFailed,
			Message:   shortMessage(validation),
			Violation: violationsDTO(validation),
		}

	case errors.Is(err, registrasi.ErrPolicyNotFound):
		// 422, bukan 404: yang tidak ditemukan bukan alamat yang diminta melainkan ISI
		// permintaannya, dan petugas dapat memperbaikinya sendiri. Pesannya menyebut
		// tindakan yang mungkin, bukan sekadar menyatakan kegagalan.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code: CodePolicyNotFound,
			Message: "Nomor Polis tidak ditemukan. Periksa kembali nomornya, " +
				"atau pastikan polisnya sudah terbit di sistem polis.",
		}

	case errors.Is(err, registrasi.ErrUnknownAreaLevel):
		// Tingkat wilayah adalah bagian alamat yang disusun layar, bukan isian petugas;
		// nilai di luar kelima tingkat adalah permintaan cacat.
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Tingkat wilayah tidak dikenal.",
		}

	case errors.Is(err, registrasi.ErrAccountNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeAccountNotFound,
			Message: "Account number is not registered in Master Rekening",
		}

	case errors.Is(err, registrasi.ErrCommitteeNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeCommitteeNotFound,
			Message: "Committee case not found.",
		}

	case errors.Is(err, registrasi.ErrNotCommitteeTurn):
		// 403: yang memutuskan hanyalah anggota jenjang yang sedang ditunggu.
		return http.StatusForbidden, ErrorResponse{
			Code:    CodeNotCommitteeTurn,
			Message: "This committee decision is not waiting for you.",
		}

	case errors.Is(err, registrasi.ErrClaimNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeClaimNotFound,
			Message: "Klaim tidak ditemukan.",
		}

	case errors.Is(err, registrasi.ErrTaskNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeTaskNotFound,
			Message: "Tugas tidak ditemukan.",
		}

	case errors.Is(err, registrasi.ErrTaskAlreadyClaimed):
		// 409: rekan kerja mendahului. Mengulang permintaan tidak akan menolong, dan
		// ini bukan kesalahan pemanggil.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeTaskAlreadyClaimed,
			Message: "Tugas ini sudah diambil pengguna lain.",
		}

	case errors.Is(err, registrasi.ErrNotTaskOwner):
		return http.StatusForbidden, ErrorResponse{
			Code:    CodeNotTaskOwner,
			Message: "Tugas ini bukan milik Anda.",
		}

	case errors.Is(err, registrasi.ErrTaskAlreadyDone):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeTaskAlreadyDone,
			Message: "Tugas ini sudah selesai dikerjakan.",
		}

	case errors.Is(err, registrasi.ErrStageMismatch):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeStageMismatch,
			Message: "Klaim sudah berpindah tahap. Muat ulang layar sebelum menyimpan.",
		}

	case errors.Is(err, registrasi.ErrNotAvailableAtStage):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeNotAvailableAtStage,
			Message: "Fitur ini tidak tersedia pada tahap klaim saat ini.",
		}

	case errors.Is(err, registrasi.ErrInvalidAction):
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeInvalidAction,
			Message: "Tindakan itu tidak berlaku pada tahap ini.",
		}

	case errors.Is(err, usecase.ErrCashierNoReply):
		// Kasir menerima permintaan tetapi tidak menjawab: pembayaran MUNGKIN sudah diproses.
		// Baris tidak ditandai terkirim; petugas diminta memeriksa Kasir sebelum mengulang.
		return http.StatusGatewayTimeout, ErrorResponse{
			Code: CodeCashierNoReply,
			Message: "The cashier system did not reply in time. The transfer may already have been received by the cashier — " +
				"check it in the cashier system before trying again, so the payment is not sent twice.",
		}

	case errors.Is(err, registrasi.ErrSurveyUnavailable):
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:    CodeSurveyUnavailable,
			Message: "Survey storage is not configured on this server. Report it to the administrator.",
		}

	case errors.Is(err, registrasi.ErrSurveyCommitteeAwaiting):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeSurveyCommitteeAwaiting,
			Message: "Survey committee cases are decided through the Pega email-link service (KomiteAcceptSurvey), which is not available yet.",
		}

	case errors.Is(err, usecase.ErrSurveyNotInProgress):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeInvalidAction,
			Message: "Only a survey that is in progress can be cancelled.",
		}

	case errors.Is(err, usecase.ErrPLASendUnavailable):
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:    CodePLASendUnavailable,
			Message: "PLA email sending is not configured on this server. Report it to the administrator.",
		}

	case errors.Is(err, usecase.ErrCashierUnavailable):
		// Kasir tidak menjawab atau alamatnya belum terdaftar: tidak ada yang ditandai terkirim.
		return http.StatusBadGateway, ErrorResponse{
			Code:    CodeCashierUnavailable,
			Message: "The cashier system could not be reached, so nothing was transferred. Try again, or report it to the administrator.",
		}

	case errors.As(err, &cashierRejected):
		// Jawaban Kasir tanpa CaseIDCashier: pesannya ditampilkan apa adanya, seperti Pega.
		return http.StatusUnprocessableEntity, ErrorResponse{Code: CodeCashierRejected, Message: cashierRejected.Message}

	case errors.Is(err, usecase.ErrPremiumUnavailable):
		// Status premi yang tidak terbaca MENAHAN akseptasi (Pega tidak menangani
		// kegagalan layanannya). Rinciannya masuk log.
		return http.StatusBadGateway, ErrorResponse{
			Code:    CodePremiumUnavailable,
			Message: "The premium status could not be checked, so the acceptance was not saved. Try again, or report it to the administrator.",
		}

	case errors.Is(err, registrasi.ErrExchangeRateNotFound):
		// Kurs yang belum diisi MENOLAK klaim; ia tidak diganti nilai bawaan
		// (`ADR-0015`). Pesannya menyebut apa yang harus dilakukan orang, bukan apa
		// yang gagal di dalam sistem.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeExchangeRateNotFound,
			Message: "Kurs mata uang pada tanggal kejadian belum tersedia. Hubungi bagian yang mengisi kurs.",
		}

	default:
		// Galat Oracle ditampilkan apa adanya beserta tabelnya, bukan pesan umum — supaya
		// petugas dapat melaporkannya (Work Owner 2026-10-10). Lihat platform/oraerror.
		if message, ok := oraerror.Describe(err); ok {
			return http.StatusInternalServerError, ErrorResponse{Code: CodeDatabaseError, Message: message}
		}
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeInternalError,
			Message: "Terjadi kesalahan pada sistem.",
		}
	}
}

// shortMessage mengambil pelanggaran pertama sebagai ringkasan.
//
// Yang pertama bukan sembarang pertama: urutannya sama dengan urutan langkah pada
// `Activity/InputRegister_act-Act.xml`, sehingga pesan ringkas ini selalu sama dengan
// satu-satunya pesan yang ditampilkan Pega.
func shortMessage(g *registrasi.ValidationError) string {
	if p, ok := g.First(); ok {
		return p.Message
	}
	return "Data klaim belum memenuhi ketentuan."
}

func violationsDTO(g *registrasi.ValidationError) []ViolationDTO {
	result := make([]ViolationDTO, 0, len(g.Violation))
	for _, p := range g.Violation {
		result = append(result, ViolationDTO{
			Code:    string(p.Code),
			Field:   p.Field,
			Message: p.Message,
		})
	}
	return result
}
