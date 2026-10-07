package inboxmanageradminhttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang sudah dibaca frontend, sama
// halnya dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail = "validasi_gagal"
	CodeCallerUnknown  = "profil_pemanggil_tidak_lengkap"
	CodeTabNotAllowed  = "tab_bukan_hak_anda"
	CodeNoTabAllowed   = "tidak_ada_tab_untuk_lini_bisnis_anda"
	CodeInternalError  = "galat_internal"

	// CodeSourceColumnMissing menandai keadaan yang sudah diketahui akan terjadi dan punya
	// tindakan yang jelas: kolom sumber belum ditambahkan DBA. Layar memakainya untuk
	// membedakan "menunggu penyiapan" dari "rusak".
	CodeSourceColumnMissing = "kolom_sumber_belum_ada"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis
// galat portal dan auth, sehingga galat sesi maupun galat portal yang lolos dari
// middleware tetap dijawab dengan kode yang sudah dikenal frontend. Bila tidak ada
// cadangan, atau cadangannya pun tidak mengenalinya, jawabannya 500 dengan pesan umum dan
// rinciannya hanya masuk log — rincian galat internal tidak pernah dikirim ke peramban.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return apierror.Writer(logger, writeJSON, fallback, mapError, ErrorResponse{
		Code:    CodeInternalError,
		Message: "Terjadi kesalahan pada sistem.",
	})
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke
// yang lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *inboxmanageradmin.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
		// Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah kesalahan
		// pengguna yang harus ditandai di isiannya (`10-API-STRATEGY.md` §5).
		details := apierror.FieldErrors(validation.Violations)

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, inboxmanageradmin.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Layar ini menampilkan pekerjaan " +
				"seluruh petugas pada satu unit organisasi, sehingga pembukaannya wajib " +
				"tercatat atas nama seseorang. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxmanageradmin.ErrTabNotAllowed):
		// 403, dan ini memang soal kewenangan: tabnya ada, permintaannya berbentuk benar,
		// yang tidak cocok adalah lini bisnis pemanggil.
		return http.StatusForbidden, ErrorResponse{
			Code: CodeTabNotAllowed,
			Message: "Tab itu ada, tetapi bukan hak lini bisnis Anda. Layar ini memisahkan " +
				"antrean menurut unit organisasi admin, dan setiap lini bisnis hanya " +
				"membuka unitnya sendiri.",
		}, true

	case errors.Is(err, inboxmanageradmin.ErrNoTabAllowed):
		// 403 pula, tetapi dengan kode dan pesan yang BERBEDA.
		//
		// Membedakan keduanya penting: yang di atas berarti pengguna salah alamat, yang
		// ini berarti layarnya memang tidak punya apa pun untuknya. Menjawab keduanya
		// dengan pesan yang sama akan membuat pengguna mencoba tab lain satu per satu,
		// dan seluruhnya ditolak.
		return http.StatusForbidden, ErrorResponse{
			Code: CodeNoTabAllowed,
			Message: "Tidak ada antrean di layar ini yang menjadi hak lini bisnis Anda. " +
				"Ketiga tab dibuka oleh lini bisnis " +
				strings.Join(inboxmanageradmin.ExpectedLineBusinesses(), ", ") +
				". Bila Anda seharusnya termasuk salah satunya, laporkan ke tim " +
				"teknis — lini bisnis petugas disimpan pada master pengguna dan dapat " +
				"dilengkapi tanpa perubahan aplikasi.",
		}, true

	case errors.Is(err, inboxmanageradmin.ErrSourceColumnMissing):
		// 503, bukan 500: sistemnya tidak rusak — struktur basis datanya belum sejalan
		// dengan yang dibutuhkan layar. Statusnya berbeda supaya pemantauan tidak
		// mencampurnya dengan kegagalan tak terduga.
		//
		// # Kenapa pesannya TIDAK menyebut nama kolom
		//
		// Karena tidak ada lagi kolom yang DIKETAHUI hilang: ketiga yang sempat diminta
		// ternyata sudah ada (koreksi Work Owner 2026-09-27). Bila galat ini muncul
		// sekarang, kolomnya BELUM diketahui — dan menyebut nama yang salah lebih buruk
		// daripada tidak menyebut apa pun, karena orang akan memeriksa kolom yang baik-baik
		// saja.
		//
		// Nama kolomnya ada di pesan galat Oracle, yang masuk log. Ia tidak dikirim ke
		// peramban: rincian internal tidak pernah bocor ke klien.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeSourceColumnMissing,
			Message: "Antrean belum dapat ditampilkan: ada kolom yang dibutuhkan layar " +
				"ini tetapi belum ada di basis data. Ini BUKAN kerusakan aplikasi dan " +
				"tidak dapat diperbaiki dengan mencoba lagi. Laporkan ke tim teknis — " +
				"nama kolomnya tercatat di log server.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
