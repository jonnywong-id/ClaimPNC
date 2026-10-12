package inboxosclaimpercabanghttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxosclaimpercabang"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang sudah dibaca frontend, sama halnya
// dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeCallerUnknown = "profil_pemanggil_tidak_lengkap"
	CodeBranchUnknown = "cabang_tidak_diketahui"
	CodeClaimNotFound = "klaim_tidak_ditemukan"
	CodeInternalError = "galat_internal"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis galat
// portal dan auth, sehingga galat sesi maupun galat portal yang lolos dari middleware tetap
// dijawab dengan kode yang sudah dikenal frontend. Bila tidak ada cadangan, atau cadangannya
// pun tidak mengenalinya, jawabannya 500 dengan pesan umum dan rinciannya hanya masuk log —
// rincian galat internal tidak pernah dikirim ke peramban.
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
			logger.Error(
				"permintaan gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
		}

		writeJSON(w, r, status, body)
	}
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya pemanggil
// dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang lain" — dua
// hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	switch {
	case errors.Is(err, inboxosclaimpercabang.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca, sehingga cabang Anda tidak dapat " +
				"ditentukan. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxosclaimpercabang.ErrBranchUnknown):
		// 409, bukan 200 berisi daftar kosong — dan itu keputusan yang paling penting di
		// berkas ini.
		//
		// Daftar kosong dan "Anda belum punya cabang" TERLIHAT SAMA di layar, padahal yang
		// pertama berarti tidak ada pekerjaan dan yang kedua berarti layar tidak dapat
		// bekerja sama sekali. Sistem lama pun membedakannya: `OutstandingperCabang_PreAct`
		// langkah 2 menampilkan pesan, bukan grid kosong.
		//
		// Pesannya diambil apa adanya dari activity itu (`D-13`), beserta tindak lanjut yang
		// selama ini diketahui pengguna — menghubungi Tim IT.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeBranchUnknown,
			Message: inboxosclaimpercabang.BranchUnknownNotice,
		}, true

	case errors.Is(err, inboxosclaimpercabang.ErrClaimNotFound):
		// 404, dan pesannya SENGAJA tidak menyebut apakah nomor itu ada di cabang lain.
		//
		// Membedakan "tidak ada" dari "milik cabang lain" akan menjadikan endpoint ini alat
		// untuk memastikan sebuah nomor klaim ada di badan hukum lain, cukup dengan membaca
		// pesannya (`R-20`). Perbedaannya tetap terekam, tetapi hanya di log server.
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeClaimNotFound,
			Message: inboxosclaimpercabang.ClaimNotFoundNotice,
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
