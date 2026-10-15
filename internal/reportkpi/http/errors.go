package reportkpihttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/reportkpi"
	reportkpisql "claim-pnc/internal/reportkpi/repo/sqlstore"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang dibaca frontend, sama halnya
// dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail    = "validasi_gagal"
	CodeCallerUnknown     = "profil_pemanggil_tidak_lengkap"
	CodeWriteNotAvailable = "belum_tersedia"
	CodeInternalError     = "galat_internal"

	// CodeQueryFailed: kuerinya dijalankan, basis data yang menolaknya.
	//
	// Dipisahkan dari galat internal umum karena tindak lanjutnya berbeda: yang satu
	// menuntut pembacaan kode, yang ini menuntut pembacaan pesan ORA pada log peladen.
	CodeQueryFailed = "penilaian_gagal_diambil"

	// CodeSecondaryConnectionMissing: sambungan ANEKA_<PORTAL>_* belum dipasang.
	//
	// Ia BUKAN kerusakan, dan karena itu tidak boleh berbagi kode dengan kerusakan:
	// yang dibutuhkan adalah lima baris konfigurasi, bukan perbaikan kode.
	CodeSecondaryConnectionMissing = "koneksi_kedua_belum_terpasang"
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
		// Penolakan basis data ditangani lebih dulu, karena jawabannya menyertakan ID
		// permintaan — dan ID itu hanya ada di sini, bukan di mapError yang murni.
		//
		// Pesan ORA-nya TIDAK ikut dikirim: ia memuat nama tabel dan kolom
		// (`11-CROSSCUTTING.md` §1.2 aturan 5). Yang dikirim adalah ID permintaan, yang
		// menunjuk tepat satu baris log bagi yang berhak membacanya dan tidak berarti
		// apa-apa bagi yang tidak.
		if errors.Is(err, reportkpisql.ErrQueryFailed) {
			id := logging.RequestID(r.Context())
			if logger != nil {
				logger.ErrorContext(r.Context(), "kueri KPI ditolak basis data",
					slog.String("jalur", r.URL.Path),
					slog.String("id_permintaan", id),
					slog.String("galat", err.Error()))
			}
			writeJSON(w, r, http.StatusInternalServerError, ErrorResponse{
				Code: CodeQueryFailed,
				Message: "Penilaian gagal diambil dari basis data. Keterangan lengkapnya " +
					"ada pada log peladen dengan nomor permintaan " + id + " — sampaikan " +
					"nomor itu kepada tim pengembang.",
			})
			return
		}

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
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang
// lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *reportkpi.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
		// Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah kesalahan
		// pengguna yang harus ditandai di isiannya (`10-API-STRATEGY.md` §5).
		//
		// Di layar ini ia yang menyampaikan tipe report yang belum dipilih dan kedua
		// tanggal periode yang belum diisi — dan ketiganya dikirim SEKALIGUS, bukan satu
		// lalu satu lagi (`P-5`).
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, reportkpi.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Laporan ini memuat penilaian kinerja " +
				"adjuster, sehingga pembukaannya wajib tercatat atas nama seseorang. " +
				"Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, reportkpi.ErrWriteNotAvailable):
		// 501, bukan 403 maupun 404.
		//
		// 403 akan menyatakan pengguna tidak berwenang — padahal ia berwenang, dan
		// wewenangnya bukan yang menghalangi. 404 akan menyatakan alamatnya tidak ada,
		// sehingga tombolnya terbaca sebagai kerusakan. 501 menyatakan yang sebenarnya:
		// alamatnya ada, permintaannya sah, kemampuannya yang belum dibangun.
		return http.StatusNotImplemented, ErrorResponse{
			Code: CodeWriteNotAvailable,
			Message: "Menghitung ulang penilaian KPI belum tersedia di sistem baru. " +
				"Perhitungannya MENULIS ke " + reportkpi.SourceTable + ", dan selama Pega " +
				"dan sistem baru berjalan berdampingan tabel itu hanya boleh ditulis satu " +
				"sistem — hari ini Pega. Jalankan perhitungannya lewat Pega; hasilnya " +
				"langsung terbaca di sini.",
		}, true

	case errors.Is(err, reportkpisql.ErrHolidayCalendarUnavailable):
		// 503, bukan 500. Tidak ada yang rusak — sambungan yang dibutuhkannya belum
		// dipasang, dan itu keadaan yang dapat diperbaiki tanpa menyentuh satu baris kode.
		//
		// Pesannya menyebut nama variabelnya, karena pesan yang hanya berkata "koneksi
		// kedua tidak tersedia" memaksa pembacanya mencari tahu koneksi yang mana.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeSecondaryConnectionMissing,
			Message: "Penilaian tidak dapat dihitung karena kalender hari libur belum " +
				"dapat dibaca. Kalender itu berada di basis data ASMD, dan sambungannya " +
				"diatur lewat ANEKA_<PORTAL>_HOST, _PORT, _SERVICE, _PENGGUNA, dan _SANDI " +
				"pada berkas konfigurasi. Tanpa kalender itu, setiap rentang yang memuat " +
				"hari libur akan terhitung lebih panjang — sehingga penilaiannya sengaja " +
				"tidak ditampilkan, bukan ditampilkan keliru.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
