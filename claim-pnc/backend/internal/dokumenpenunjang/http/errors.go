package dokumenpenunjanghttp

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien.
//
// Klien membedakan jenis galat lewat kode ini, bukan dengan mencocokkan teks pesan.
const (
	CodeBadRequest    = "permintaan_cacat"
	CodeNotFound      = "tidak_ditemukan"
	CodeTooLarge      = "berkas_terlalu_besar"
	CodeUpstream      = "layanan_penyimpanan_gagal"
	CodeHalfDone      = "tersimpan_sebagian"
	CodeInternalError = "galat_internal"
)

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// JSONWriter menuliskan badan respons.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat menjadi respons HTTP.
//
// # Tiga jenis kegagalan, tiga jawaban berbeda — dan bedanya BUKAN kosmetik
//
//	permintaan cacat      400   perbaiki lalu ulangi
//	layanan gagal         502   ulangi apa adanya; belum ada yang tersimpan
//	metadata gagal        500   JANGAN ulangi — berkasnya sudah terkirim
//
// Yang ketiga itu sebabnya `ErrMetadataGagal` tidak digabung ke galat internal biasa.
// Pesan "coba lagi" pada keadaan itu membuat pengguna mengulang berkali-kali, dan setiap
// percobaan meninggalkan satu salinan yatim di penyimpanan yang tidak dapat ditarik
// kembali.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		switch {
		case errors.Is(err, dokumenpenunjang.ErrBerkasTerlaluBesar):
			writeJSON(w, r, http.StatusRequestEntityTooLarge, ErrorResponse{
				Code: CodeTooLarge,
				Message: fmt.Sprintf(
					"Ukuran berkas melebihi batas %d MB. Perkecil berkasnya lalu unggah ulang.",
					dokumenpenunjang.BatasUkuranBerkas>>20),
			})
			return

		case errors.Is(err, dokumenpenunjang.ErrBerkasKosong):
			writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
				Code:    CodeBadRequest,
				Message: "Berkas kosong. Pilih berkas yang berisi lalu unggah ulang.",
			})
			return

		case errors.Is(err, dokumenpenunjang.ErrNamaBerkasKosong):
			// Pesannya menyebut SEBABNYA, karena sebabnya tidak terduga: nama yang hanya
			// berisi tanda baca menjadi kosong setelah dibersihkan.
			writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
				Code: CodeBadRequest,
				Message: "Nama berkas harus memuat huruf atau angka. " +
					"Ganti namanya lalu unggah ulang.",
			})
			return

		case errors.Is(err, dokumenpenunjang.ErrPengunggahKosong):
			// 401, bukan 400: yang kurang adalah identitas pemanggil, bukan isi
			// permintaannya. Pengguna tidak dapat memperbaikinya dengan mengubah form.
			writeJSON(w, r, http.StatusUnauthorized, ErrorResponse{
				Code:    CodeBadRequest,
				Message: "Sesi Anda tidak dikenali. Masuk kembali lalu ulangi.",
			})
			return

		case errors.Is(err, dokumenpenunjang.ErrFolderAplikasiTidakAda):
			// 500: ini salah konfigurasi di sisi kita, bukan kesalahan pengguna. Pesannya
			// menyebut apa yang harus dilaporkan supaya tidak berakhir sebagai "sistem
			// error" tanpa petunjuk.
			writeJSON(w, r, http.StatusInternalServerError, ErrorResponse{
				Code: CodeInternalError,
				Message: "Folder penyimpanan dokumen belum terdaftar. " +
					"Laporkan ke administrator — unggahan tidak dapat diproses.",
			})
			return

		case errors.Is(err, dokumenpenunjang.ErrLinkTakTerjangkau):
			// 502: yang gagal database ASMD di ujung DB link, bukan aplikasi ini. Belum ada
			// berkas yang terkirim, sehingga mengulang aman.
			catatGalatHulu(logger, r, err)
			writeJSON(w, r, http.StatusBadGateway, ErrorResponse{
				Code: CodeUpstream,
				Message: "Basis data penyimpanan dokumen (DB link ASMD) sedang tidak dapat dihubungi. " +
					"Berkas belum tersimpan. Laporkan ke DBA, lalu coba lagi.",
			})
			return

		case errors.Is(err, dokumenpenunjang.ErrKonversiGagal):
			// 502, sama dengan kegagalan penyimpanan: yang gagal layanan di hulu, dan
			// belum ada apa pun yang tersimpan sehingga mengulang aman.
			//
			// Pesannya menyebut jenis berkasnya, karena itulah yang membedakan: berkas
			// selain PNG/JPG/JPEG/PDF tidak menempuh konversi sama sekali, dan pengguna
			// yang tidak tahu itu akan mengira seluruh unggah sedang mati.
			catatGalatHulu(logger, r, err)
			writeJSON(w, r, http.StatusBadGateway, ErrorResponse{
				Code: CodeUpstream,
				Message: "Layanan konversi gambar sedang tidak dapat dihubungi, sehingga " +
					"berkas PNG, JPG, dan PDF belum dapat diunggah. Berkas belum tersimpan; " +
					"silakan coba lagi.",
			})
			return

		case errors.Is(err, dokumenpenunjang.ErrUnggahGagal):
			// 502: yang gagal layanan di hulu, bukan kita. Belum ada yang tersimpan,
			// sehingga mengulang aman — dan pesannya mengatakan itu.
			catatGalatHulu(logger, r, err)
			writeJSON(w, r, http.StatusBadGateway, ErrorResponse{
				Code: CodeUpstream,
				Message: "Layanan penyimpanan dokumen sedang tidak dapat dihubungi. " +
					"Berkas belum tersimpan; silakan coba lagi.",
			})
			return

		case errors.Is(err, dokumenpenunjang.ErrMetadataGagal):
			catatGalatHulu(logger, r, err)
			writeJSON(w, r, http.StatusInternalServerError, ErrorResponse{
				Code: CodeHalfDone,
				Message: "Berkas sudah terkirim ke penyimpanan tetapi catatannya gagal disimpan, " +
					"sehingga belum muncul di daftar. JANGAN unggah ulang — laporkan ke administrator.",
			})
			return

		case errors.Is(err, dokumenpenunjang.ErrTidakDitemukan):
			writeJSON(w, r, http.StatusNotFound, ErrorResponse{
				Code:    CodeNotFound,
				Message: "Dokumen tidak ditemukan.",
			})
			return
		}

		if fallback != nil {
			fallback(w, r, err)
			return
		}

		logging.From(r.Context(), logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
		writeJSON(w, r, http.StatusInternalServerError, ErrorResponse{
			Code:    CodeInternalError,
			Message: "Terjadi kesalahan pada sistem.",
		})
	}
}

// catatGalatHulu mencatat kegagalan layanan di hulu BESERTA sebab aslinya.
//
// # Cacat yang ini perbaiki
//
// Keempat cabang hulu — DB link ASMD, konversi, penyimpanan, dan pencatatan metadata —
// `return` sebelum baris pencatatan di akhir fungsi. Akibatnya galat yang dibungkus
// **dibuang seluruhnya**: server tidak mencatat apa pun, dan yang tersisa hanyalah pesan
// umum di layar.
//
// Yang hilang bukan detail sepele. Ia memuat satu-satunya keterangan yang membedakan dua
// keadaan yang penanganannya berbeda jauh:
//
//	"menghubungi layanan penyimpanan: dial tcp ... "   layanannya TIDAK TERJANGKAU
//	"layanan menjawab 503"                             terjangkau, sedang tidak melayani
//	"respons tidak memuat URLImage (ErrorCode: ...)"   terjangkau, MENOLAK berkasnya
//
// Yang pertama urusan jaringan, yang ketiga urusan berkas atau kredensial. Tanpa baris ini
// keduanya terlihat persis sama — dan penelusurannya dimulai dari tempat yang salah.
//
// # Kenapa hanya ke log, bukan ke pengguna
//
// Pesan hulu dapat memuat apa saja, termasuk gema muatan kita sendiri — yang berisi berkas
// nasabah. `httpstorage` sudah menjaga agar isi respons tidak ikut terbawa; yang sampai ke
// sini hanyalah kode status, nama field, dan pesan layanan. Itu aman untuk log operator,
// tidak untuk peramban (`11-CROSSCUTTING` §1.2 butir 5).
func catatGalatHulu(logger *slog.Logger, r *http.Request, err error) {
	logging.From(r.Context(), logger).Error("unggah dokumen gagal di layanan hulu",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}

// writeBadRequest menjawab permintaan yang cacat bentuknya.
func writeBadRequest(writeJSON JSONWriter, w http.ResponseWriter, r *http.Request, pesan string) {
	writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
		Code:    CodeBadRequest,
		Message: pesan,
	})
}
