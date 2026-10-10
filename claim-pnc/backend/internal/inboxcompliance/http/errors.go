package inboxcompliancehttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxcompliance"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang dibaca frontend, sama halnya
// dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail  = "validasi_gagal"
	CodeTabNotReady     = "tab_belum_tersedia"
	CodeMalformedBody   = "permintaan_tidak_terbaca"
	CodeClaimNotInQueue = "klaim_tidak_di_antrean"
	CodeCallerUnknown   = "profil_pemanggil_tidak_lengkap"
	CodeStoreMissing    = "tabel_keputusan_belum_ada"

	CodeDocumentNotFound       = "dokumen_tidak_ditemukan"
	CodeDocumentServiceMissing = "layanan_dokumen_belum_ada"
	CodeLetterRendererMissing  = "pembentuk_surat_belum_ada"

	CodeInternalError = "galat_internal"
)

// errMalformedBody berarti badan permintaan tidak dapat diurai sebagai JSON.
//
// Ia galat milik lapisan transport, bukan domain — domain tidak tahu apa pun tentang bentuk
// kawatnya — sehingga ia tinggal di berkas ini, bukan di errors.go modul.
var errMalformedBody = errors.New("inboxcompliancehttp: badan permintaan tidak dapat dibaca")

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

		if status >= http.StatusInternalServerError {
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
	var (
		validation *inboxcompliance.ValidationError
		notReady   *inboxcompliance.TabNotReadyError
	)

	switch {
	case errors.Is(err, errMalformedBody):
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedBody,
			Message: "Permintaan tidak dapat dibaca. Muat ulang halaman lalu coba lagi.",
		}, true

	case errors.Is(err, inboxcompliance.ErrClaimNotInQueue):
		// 409, bukan 404. Klaimnya boleh jadi ada dan sehat — yang tidak ada adalah
		// posisinya di antrean ini, dan keadaan itu berubah tanpa pengguna melakukan
		// apa pun. Menjawab 404 akan membuat petugas mencari klaimnya, padahal yang perlu
		// dilakukan hanyalah menyegarkan daftar.
		return http.StatusConflict, ErrorResponse{
			Code: CodeClaimNotInQueue,
			Message: "Klaim ini sudah tidak ada di antrean Compliance. Mungkin sudah " +
				"dikirim petugas lain. Segarkan daftar lalu periksa kembali.",
		}, true

	case errors.Is(err, inboxcompliance.ErrDecisionStoreMissing):
		// 503, bukan 500. Permintaannya benar dan kodenya benar — yang belum ada adalah
		// tabel yang hanya DBA dapat membuatnya (`D-63`). Keadaan itu akan berubah tanpa
		// pengguna melakukan apa pun, persis seperti ErrTabNotReady.
		//
		// Pesannya menyebut NOMOR MIGRASINYA. Tanpa itu, petugas melaporkan "aplikasi
		// rusak", pengembang membuka log, lalu menemukan ORA-00942 yang sudah diketahui
		// sejak migrasinya ditulis — satu putaran penuh untuk informasi yang sudah ada.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeStoreMissing,
			Message: "Form Compliance Checker belum dapat dibuka: tabel penyimpan " +
				"keputusannya belum ada di basis data. DBA perlu membuat " +
				"POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE di basis data setiap entitas — " +
				"skripnya ada di claim-pnc/docs/permintaan-dba-compliance.sql.",
		}, true

	// Cabang `ErrSaveNotAllowed` DICABUT 2026-10-07 bersama galatnya: ia menolak klaim
	// Travel atas premis yang terbukti salah. Lihat errors.go pada paket domain.

	case errors.Is(err, inboxcompliance.ErrDocumentNotFound):
		// 404. Dokumen yang tidak ada dan dokumen milik klaim LAIN dijawab sama —
		// membedakannya akan memberi tahu pemanggil bahwa sebuah id itu sah.
		return http.StatusNotFound, ErrorResponse{
			Code: CodeDocumentNotFound,
			Message: "Dokumen tidak ditemukan pada klaim ini. Segarkan halaman — " +
				"dokumennya mungkin sudah dihapus.",
		}, true

	case errors.Is(err, inboxcompliance.ErrDocumentServiceMissing):
		// 503, bukan 500: tidak ada yang rusak — layanan dokumennya memang belum
		// dikonfigurasi untuk portal ini. Pesannya menyebut apa yang kurang supaya
		// petugas tidak melaporkannya sebagai kerusakan.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeDocumentServiceMissing,
			Message: "Berkas belum dapat dibuka: layanan dokumen belum dikonfigurasi " +
				"untuk portal ini. Keputusan atas klaim tetap dapat disimpan.",
		}, true

	case errors.Is(err, inboxcompliance.ErrLetterRendererMissing):
		// 500, BUKAN 503 — berbeda dari ErrDocumentServiceMissing di atas meski keduanya
		// "sesuatu belum dipasang".
		//
		// Yang di atas adalah layanan di seberang jaringan yang memang boleh belum
		// dikonfigurasi per portal; keadaan itu normal dan akan berubah sendiri. Yang ini
		// adalah paket DI DALAM binary yang sama: bila ia tidak terpasang, perakitan
		// modulnya yang keliru, dan tidak ada yang akan berubah sampai seseorang
		// memperbaiki kodenya. Menjawabnya 503 akan membuat petugas menunggu sesuatu yang
		// tidak akan datang.
		return http.StatusInternalServerError, ErrorResponse{
			Code: CodeLetterRendererMissing,
			Message: "Surat penolakan belum dapat dibuat karena pembentuk suratnya belum " +
				"terpasang di aplikasi. Laporkan ke tim pengembang — ini bukan kesalahan " +
				"pengisian Anda.",
		}, true

	case errors.Is(err, inboxcompliance.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca, sehingga pengiriman ini tidak dapat " +
				"dicatat atas nama siapa pun. Masuk ulang lalu coba lagi.",
		}, true

	case errors.As(err, &notReady):
		// 503, bukan 404 maupun 422. Tabnya ADA dan permintaannya benar — yang belum ada
		// adalah artefak dari pihak lain, dan keadaan itu akan berubah tanpa pengguna
		// melakukan apa pun. 404 akan membuat pengguna mengira tabnya tidak pernah ada;
		// 422 akan membuatnya mencari isian yang salah pada layar tanpa isian.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeTabNotReady,
			Message: "Tab " + notReady.Tab.Name + " belum dapat menampilkan data. " +
				notReady.Blocker,
		}, true

	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
		// Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah kesalahan
		// pengguna yang harus ditandai di isiannya (`10-API-STRATEGY.md` §5).
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
