package inboxrclpuclhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxrclpucl"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang sudah dibaca frontend, sama
// halnya dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail     = "validasi_gagal"
	CodeCallerUnknown      = "profil_pemanggil_tidak_lengkap"
	CodeWriteNotAvailable  = "belum_tersedia"
	CodeReportNotAvailable = "laporan_tidak_tersedia"
	CodeClaimNotFound      = "klaim_tidak_ditemukan"
	CodeInternalError      = "galat_internal"
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
	var validation *inboxrclpucl.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
		// Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah kesalahan
		// pengguna yang harus ditandai di isiannya (`10-API-STRATEGY.md` §5).
		//
		// Di layar ini ia yang menyampaikan kedua tanggal laporan yang belum diisi, dan
		// keduanya dikirim SEKALIGUS — bukan satu, lalu satu lagi (`P-5`).
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, inboxrclpucl.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Antrean RCL/PUCL adalah antrean " +
				"bersama, sehingga pembukaannya wajib tercatat atas nama seseorang. " +
				"Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxrclpucl.ErrPegaServiceUnavailable):
		// 503, bukan 500. Keduanya "gagal" bagi mesin, tetapi menuntut orang yang BERBEDA:
		// 500 berarti tim pengembang harus memperbaiki kode, 503 berarti layanan Pega belum
		// tersedia dan yang bertindak adalah Tim Pega serta Infra.
		//
		// Pesannya menyebutkan itu, karena petugas yang membacanya tidak punya cara lain
		// mengetahui ke mana laporannya harus pergi.
		//
		// Pesannya menyebut APA yang ditunggu, bukan sekadar "belum tersambung". Petugas yang
		// menekan tombol ini sudah menanyakannya berkali-kali, dan jawaban yang tidak menyebut
		// penghalangnya terbaca seperti kerusakan yang seharusnya sudah diperbaiki.
		//
		// Nomor case TIDAK lagi disebut di kalimat ini. Dulu ia berbunyi "salin nomor case-nya
		// dari layar ini", padahal nomornya hanya ada di kaki layar — kalimat yang benar
		// berujung pada gulir mencari. Layar kini menggambar nomornya tepat di bawah pesan ini
		// (`CopyCaseNumber`), sehingga menyebutnya di sini hanya mengulang apa yang terlihat.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: "layanan_pega_belum_tersedia",
			Message: "Tindakan ini dijalankan oleh Pega, dan layanannya belum dibangun — " +
				"yang ditunggu rule Service REST `ActionClaimPUCL` dari Tim Pega. " +
				"Sementara itu tindakan ini dikerjakan di Pega.",
		}, true

	case errors.Is(err, inboxrclpucl.ErrActionNotAvailable):
		// 409, bukan 403. Bukan soal kewenangan pemanggil melainkan KEADAAN klaimnya:
		// tombolnya memang tidak digambar untuk jalur dan lini bisnis klaim ini.
		return http.StatusConflict, ErrorResponse{
			Code: "tindakan_tidak_tersedia",
			Message: "Tindakan ini tidak berlaku untuk klaim ini. Tombolnya hanya muncul " +
				"pada jalur PUCL lini Personal Accident.",
		}, true

	case errors.Is(err, inboxrclpucl.ErrAlreadyWithAnalyst):
		// 409: permintaannya sah, keadaan klaimnya yang sudah berubah. Pesannya menyebut
		// DI MANA klaimnya sekarang, karena itu pertanyaan yang akan muncul berikutnya —
		// dan tanpa jawabannya petugas menekan tombolnya lagi.
		return http.StatusConflict, ErrorResponse{
			Code: "sudah_di_analyst",
			Message: "Klaim ini sudah berada di tahap Send To Analis dan tidak lagi " +
				"menjadi pekerjaan RCL/PUCL. Tidak ada yang dipindahkan.",
		}, true

	case errors.Is(err, inboxrclpucl.ErrTechnicalPICUnknown):
		// 409: permintaannya sah, DATA klaimnya yang belum lengkap.
		//
		// # Kenapa pemetaan ini ada, dan apa akibatnya ketika belum ada
		//
		// Galat ini sudah dikembalikan repo sejak tahap Send To Analis dibangun, dan sudah
		// diteruskan usecase apa adanya — tetapi tidak pernah dipetakan di sini. Akibatnya ia
		// jatuh ke cadangan, yang juga tidak mengenalinya, lalu menjadi **"Terjadi kesalahan
		// pada sistem."** Petugas membaca kalimat yang menyatakan sistemnya rusak, padahal
		// yang kurang adalah satu isian pada klaimnya — dan satu-satunya keterangan yang
		// menyebut sebabnya hanya masuk log peladen.
		//
		// Pesannya menyebut DI MANA nilainya tinggal, bukan sekadar bahwa ia kosong: tanpa
		// itu tidak ada yang dapat dikerjakan petugas selain menekan tombolnya lagi.
		return http.StatusConflict, ErrorResponse{
			Code: "pic_teknik_belum_ada",
			Message: "Klaim ini belum punya PIC Teknik, sehingga tidak ada yang dapat " +
				"menerimanya di tahap Send To Analis. Tetapkan PIC Teknik klaim lebih " +
				"dulu dari layar registrasi, lalu ulangi. Klaimnya tetap berada di " +
				"antrean RCL/PUCL dan tidak ada yang berubah.",
		}, true

	case errors.Is(err, inboxrclpucl.ErrDocumentNotFound):
		// 404, dan sebabnya TIDAK dirinci — lihat catatan pada ErrDocumentNotFound: dokumen
		// yang tidak ada dan dokumen milik klaim lain sengaja dijawab sama, supaya jawaban
		// ini tidak memberi tahu bahwa sebuah id dokumen memang ada.
		//
		// Ia pun sebelumnya jatuh ke 500 generik. Dokumen yang hilang dari penyimpanan lalu
		// dilaporkan sebagai "sistem rusak" membuat petugas melapor ke tim yang salah.
		return http.StatusNotFound, ErrorResponse{
			Code: "dokumen_tidak_ditemukan",
			Message: "Dokumen tidak ditemukan pada klaim ini. Daftar dokumennya mungkin " +
				"sudah berubah — muat ulang daftarnya, lalu coba lagi.",
		}, true

	case errors.Is(err, inboxrclpucl.ErrClaimNotFound):
		// 404, dan pesannya menyebut PORTAL.
		//
		// Penyebab paling mungkin bukan klaim yang benar-benar tidak ada, melainkan kunci
		// yang benar dibuka pada portal yang salah — dan itu keadaan yang tidak
		// menghasilkan satu pun tanda lain (`R-20`). Pesan "tidak ditemukan" tanpa
		// menyebut portal akan membuat pengguna menyimpulkan datanya hilang.
		return http.StatusNotFound, ErrorResponse{
			Code: CodeClaimNotFound,
			Message: "Klaim tidak ditemukan pada entitas yang sedang dipilih. " +
				"Periksa pilihan portal di bilah atas — kunci klaim milik entitas lain " +
				"tidak dapat dibuka dari sini.",
		}, true

	case errors.Is(err, inboxrclpucl.ErrReportNotAvailable):
		// 404, bukan 422. Yang salah bukan isian pengguna melainkan alamat yang diminta:
		// hanya tab "Cetak Surat" yang punya laporan rentang tanggal, dan layar tidak
		// pernah menawarkannya di tab lain. Permintaan seperti ini datang dari alamat yang
		// diketik sendiri atau dari tautan lama.
		return http.StatusNotFound, ErrorResponse{
			Code: CodeReportNotAvailable,
			Message: "Laporan rentang tanggal hanya tersedia pada tab \"Cetak Surat\". " +
				"Tab lain mengunduh isi tabelnya sendiri.",
		}, true

	case errors.Is(err, inboxrclpucl.ErrWriteNotAvailable):
		// 501, bukan 403 maupun 404.
		//
		// 403 akan menyatakan pengguna tidak berwenang — padahal ia berwenang, dan
		// wewenangnya bukan yang menghalangi. 404 akan menyatakan alamatnya tidak ada,
		// sehingga tombolnya terbaca sebagai kerusakan. 501 menyatakan yang sebenarnya:
		// alamatnya ada, permintaannya sah, kemampuannya yang belum dibangun.
		return http.StatusNotImplemented, ErrorResponse{
			Code: CodeWriteNotAvailable,
			Message: "Tindakan \"Cetak Surat\" dan \"Reminder PUCL\" belum tersedia di " +
				"sistem baru. Keduanya menulis ke objek kerja klaim, dan selama Pega dan " +
				"sistem baru berjalan berdampingan tabel itu hanya boleh ditulis satu " +
				"sistem — hari ini Pega. Kerjakan lewat Pega.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
