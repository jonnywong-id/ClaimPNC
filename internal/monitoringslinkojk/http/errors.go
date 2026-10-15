package monitoringslinkojkhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/monitoringslinkojk"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang sudah dibaca frontend, sama
// halnya dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail  = "validasi_gagal"
	CodeUnknownSegment  = "segmen_tidak_dikenal"
	CodeCallerUnknown   = "profil_pemanggil_tidak_lengkap"
	CodeWriteNotAllowed = "aksi_belum_tersedia"

	// CodeSenderNotReady — layanan SLIK belum dikonfigurasi. Dibedakan dari
	// CodeWriteNotAllowed: yang ini menunggu ALAMAT layanan, bukan menunggu kewenangan.
	CodeSenderNotReady = "layanan_slik_belum_siap"

	// CodeNothingToProcess — tidak ada data yang dapat disusun.
	CodeNothingToProcess = "tidak_ada_data"

	CodeInternalError = "galat_internal"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis
// galat portal dan auth, sehingga galat sesi maupun galat portal yang lolos dari
// middleware tetap dijawab dengan kode yang sudah dikenal frontend. Bila tidak ada
// cadangan, atau cadangannya pun tidak mengenalinya, jawabannya 500 dengan pesan umum dan
// rinciannya hanya masuk log — rincian galat internal tidak pernah dikirim ke peramban.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body, recognized := mapError(err)

		if !recognized {
			if fallback != nil {
				fallback(w, r, err)
				return
			}
			status, body = http.StatusInternalServerError, ErrorResponse{
				Code:    CodeInternalError,
				Message: "Terjadi kesalahan pada sistem.",
			}
		}

		if status >= http.StatusInternalServerError && logger != nil {
			logger.Error("permintaan gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()))
		}

		writeJSON(w, r, status, body)
	}
}

// ErrorWriterFrom membungkus penulis galat modul lain menjadi cadangan modul ini.
func ErrorWriterFrom(writer func(http.ResponseWriter, *http.Request, error)) ErrorWriter {
	if writer == nil {
		return nil
	}
	return ErrorWriter(writer)
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini — dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke
// yang lain", dua hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *monitoringslinkojk.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikembalikan sekaligus, meniru sistem lama yang
		// menampilkan semua pesan bersamaan (`P-5`).
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, monitoringslinkojk.ErrUnknownSegment):
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeUnknownSegment,
			Message: "Segmen yang diminta tidak dikenal. Pilih Segment Slik D01 atau Segment Slik F06.",
		}, true

	case errors.Is(err, monitoringslinkojk.ErrCallerUnknown):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, monitoringslinkojk.ErrSenderNotConfigured):
		// 503, bukan 500: layanannya memang belum tersedia, dan itu keadaan yang
		// diketahui — bukan kegagalan tak terduga. Pesannya menyebut sebabnya supaya
		// pelapor tahu ini bukan gangguan sesaat yang akan hilang bila dicoba lagi.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeSenderNotReady,
			Message: "Pengiriman ke SLIK belum dapat dilakukan dari aplikasi ini: " +
				"alamat layanannya belum diatur. Sementara ini pengiriman masih " +
				"dilakukan lewat aplikasi Pega.",
		}, true

	case errors.Is(err, monitoringslinkojk.ErrNothingToProcess):
		// 422, bukan 404: permintaannya sah dan penyaringnya benar; yang tidak ada
		// adalah datanya. Dibedakan dari "berhasil menyusun nol baris" supaya pelapor
		// tahu SEBABNYA — lihat ErrNothingToProcess.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code: CodeNothingToProcess,
			Message: "Tidak ada data klaim yang cocok dengan penyaring ini. " +
				"Tekan \"Cari Data\" lebih dulu untuk melihat apa yang akan disusun.",
		}, true

	case errors.Is(err, monitoringslinkojk.ErrWriteNotAvailable):
		// 409, bukan 404 maupun 405. Jalurnya ada dan metodenya benar; yang belum ada
		// adalah ARTEFAKNYA — lihat Mount untuk sebab masing-masing tombol. Jawaban 404
		// akan terbaca seperti salah alamat, dan penyebab sesungguhnya tidak akan pernah
		// sampai ke pengguna maupun ke penelusur masalah.
		return http.StatusConflict, ErrorResponse{
			Code: CodeWriteNotAllowed,
			Message: "Aksi ini belum tersedia di aplikasi baru. Selama masa paralel, " +
				"penyusunan dan pengiriman data SLIK OJK masih dilakukan lewat aplikasi Pega.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
