package inboxreceivetkahttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/platform/logging"
)

// Kode galat modul ini.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah `TKT-F1-004`, dan ia masih terhalang
// keputusan Work Owner. Modul ini karena itu memetakan galat yang DIKENALINYA sendiri, lalu
// menyerahkan sisanya ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}`
// tetap sama sehingga klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Berbeda dari Inbox Investigator yang hanya membaca dan karena itu tidak punya galat domain
// sama sekali, modul ini MENULIS — dan setiap keadaan yang dapat menolak penulisannya perlu
// dapat dibedakan oleh layar, karena ketiganya menuntut tindakan yang berbeda dari pengguna.
const (
	// CodeMalformedRequest: bentuk permintaannya sendiri yang salah.
	CodeMalformedRequest = "permintaan_cacat"

	// CodeDateRequired: tanggal kelengkapan dokumen kosong.
	//
	// Padanan langsung `Page-Set-Messages` pada langkah ke-10
	// `Activity/SubmitTanggalLengkapTKA-Act.xml`.
	CodeDateRequired = "tanggal_dokumen_wajib_diisi"

	// CodeTaskNotFound: barisnya sudah tidak ada di inbox.
	//
	// Penyebabnya dua dan sengaja tidak dibedakan: tidak pernah ada, atau sudah diisi orang
	// lain lebih dulu. Keduanya berarti hal yang sama bagi pengguna — tidak ada lagi yang
	// perlu ia kerjakan pada baris itu — dan tindakannya pun sama, yaitu menyegarkan daftar.
	CodeTaskNotFound = "pekerjaan_tidak_ditemukan"

	// CodeClaimMissing: barisnya ada di inbox, tetapi klaimnya tidak ada di data klaim
	// utama.
	//
	// Ia DIBEDAKAN dari CodeTaskNotFound karena tindakannya berbeda. Menyegarkan daftar
	// tidak akan menolong — barisnya akan muncul lagi, dan Submit atasnya akan gagal lagi.
	// Yang perlu terjadi adalah perbaikan data, dan layar harus mengatakannya begitu
	// alih-alih menyuruh pengguna mencoba lagi.
	CodeClaimMissing = "klaim_tidak_ditemukan"

	// CodeClaimAmbiguous: satu nomor klaim menunjuk lebih dari satu baris.
	//
	// Keadaan ini mungkin terjadi karena tidak ada satu pun constraint keunikan pada kedua
	// tabel (`R-08`). Penulisan dihentikan, bukan diteruskan atas salah satu barisnya.
	CodeClaimAmbiguous = "nomor_klaim_ganda"
)

// errMalformedDate menandai tanggal yang tidak dapat diurai.
//
// Ia dipisahkan dari inboxreceivetka.ErrDateRequired dengan sengaja: tanggal KOSONG adalah
// kesalahan pengguna yang dijawab 422, sedangkan tanggal yang bentuknya rusak berarti layar
// mengirim sesuatu yang tidak sesuai kontrak — cacat frontend, dijawab 400
// (`10-API-STRATEGY.md` §5).
//
// Ia hidup di lapisan transport, bukan di domain, karena "tidak dapat diurai" adalah perkara
// bentuk wire — dan domain tidak boleh tahu apa pun tentang bentuk wire.
var errMalformedDate = errors.New("inboxreceivetkahttp: tanggal tidak dapat diurai")

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan jaringan,
		// galat portal, cacat pemrograman — diserahkan ke penulis bersama. Rincian galat
		// internal tidak pernah dikirim ke peramban.
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

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	switch {
	case errors.Is(err, errMalformedDate):
		// 400: bentuk permintaannya yang salah, bukan isinya.
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Tanggal tidak dapat dibaca. Bentuk yang diharapkan YYYY-MM-DD.",
		}, true

	case errors.Is(err, inboxreceivetka.ErrClaimNumberRequired):
		// 400, bukan 422: layar selalu mengirim nomor klaim baris yang diklik pengguna,
		// sehingga ketiadaannya berarti permintaannya tidak datang dari layar ini.
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak menyebut klaim mana yang diisi.",
		}, true

	case errors.Is(err, inboxreceivetka.ErrDateRequired):
		// 422: bentuk permintaannya benar, isinya yang melanggar aturan bisnis.
		//
		// Pesannya mengikuti `Local.ErrMessages` pada activity lama APA ADANYA —
		// "Silahkan isi tanggal terlebih dahulu" — termasuk ejaan "Silahkan", supaya
		// pengguna membaca kalimat yang sama seperti selama ini (`D-13`).
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeDateRequired,
			Message: "Silahkan isi tanggal terlebih dahulu",
		}, true

	case errors.Is(err, inboxreceivetka.ErrTaskNotFound):
		// 409, bukan 404: pengguna sedang menatap barisnya, dan yang terjadi adalah
		// keadaannya berubah sejak daftar dimuat — bukan alamat yang salah
		// (`10-API-STRATEGY.md` §5).
		return http.StatusConflict, ErrorResponse{
			Code: CodeTaskNotFound,
			Message: "Klaim ini sudah tidak ada di Inbox Receive TKA. " +
				"Kemungkinan tanggalnya sudah diisi orang lain. Segarkan daftar.",
		}, true

	case errors.Is(err, inboxreceivetka.ErrClaimMissing):
		return http.StatusConflict, ErrorResponse{
			Code: CodeClaimMissing,
			Message: "Klaim ini tidak ditemukan pada data klaim utama, sehingga tanggalnya " +
				"tidak dapat disimpan. Laporkan nomor klaimnya ke administrator Claim PNC.",
		}, true

	case errors.Is(err, inboxreceivetka.ErrClaimAmbiguous):
		return http.StatusConflict, ErrorResponse{
			Code: CodeClaimAmbiguous,
			Message: "Nomor klaim ini menunjuk lebih dari satu baris data, sehingga tidak " +
				"jelas mana yang harus diperbarui. Tidak ada yang diubah. " +
				"Laporkan nomor klaimnya ke administrator Claim PNC.",
		}, true
	}

	return 0, ErrorResponse{}, false
}
