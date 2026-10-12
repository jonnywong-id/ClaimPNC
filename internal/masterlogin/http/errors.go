package masterloginhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/platform/logging"
)

// Kode galat modul ini.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih terhalang
// keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth, dan menambah
// kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya
// ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama sehingga
// klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Begitu TKT-F1-004 diputuskan, pemetaan ini pindah ke tempat bersama dan berkas ini
// tinggal memakainya. Utang itu dicatat di docs/keputusan-implementasi.md.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"

	// CodeLoginTaken dibedakan dari CodeValidationFailed dengan sengaja.
	//
	// Ia konflik KEADAAN, bukan isian yang cacat: Namanya sah, dan yang salah hanyalah
	// bahwa Login yang diturunkan darinya sudah dipakai baris lain. Perbaikannya bukan
	// "betulkan isian" melainkan "buka baris yang sudah ada, atau pakai nama lain", dan
	// layar menanganinya berbeda.
	//
	// Namanya `kunci_login_surveyor_sudah_ada` supaya ia sebentuk dengan
	// `kunci_kategori_sparepart_sudah_ada` dan `kunci_sparepart_sudah_ada` — ketiganya
	// menyatakan hal yang sama, yaitu kunci alami yang bentrok.
	CodeLoginTaken = "kunci_login_surveyor_sudah_ada"

	// CodeNameLocked muncul bila Nama yang dikirim berbeda dari Nama yang tersimpan.
	//
	// Ia BUKAN galat validasi: isiannya tidak cacat, dan tidak ada yang perlu dibetulkan —
	// yang terjadi adalah permintaan menyentuh isian yang memang tidak dapat disunting.
	// Layar menanganinya dengan memuat ulang baris, bukan dengan menyorot isian.
	//
	// Ia juga penanda bahwa permintaannya kemungkinan TIDAK datang dari layar ini: layar
	// mengunci isiannya, sehingga nilai yang berbeda berarti pengunciannya dilewati.
	CodeNameLocked = "nama_login_surveyor_terkunci"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan jaringan,
		// cacat pemrograman — diserahkan ke penulis bersama, yang menjawab 500 dengan pesan
		// umum dan menaruh rinciannya di log saja. Rincian galat internal tidak pernah
		// dikirim ke peramban.
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
	var validationError *masterlogin.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan bisnis.
		// Frontend menanganinya berbeda — 400 berarti ada cacat di frontend, 422 berarti
		// pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5).
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, p := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: p.Field, Message: p.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, masterlogin.ErrLoginTaken):
		// 409, bukan 422 — ia konflik keadaan; lihat CodeLoginTaken.
		//
		// Pesannya mengikuti `local.msg` pada `Activity/SetLoginSurveyor_act` apa adanya:
		// "Login sudah terdaftar dengan nama yang sama".
		//
		// Yang DITAMBAHKAN hanyalah keterangan pada `detail`, yang menyebut hal yang paling
		// mengejutkan: yang bentrok bukan Nama melainkan LOGIN yang DITURUNKAN darinya,
		// sehingga dua nama yang terlihat berbeda — "Budi Hartono" dan "Budi.Hartono" —
		// menghasilkan login yang sama persis. Tanpa keterangan itu, penolakannya tidak
		// dapat dijelaskan dari layar.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeLoginTaken,
			Message: "Login sudah terdaftar dengan nama yang sama",
			Detail: []ViolationDTO{{
				Field: "nama",
				Message: "Login dibentuk dari Nama dengan membuang spasi, titik, koma, dan " +
					"tanda hubung — sehingga dua nama yang berbeda dapat menghasilkan " +
					"login yang sama. Cari login itu di daftar, atau pakai nama lain.",
			}},
		}, true

	case errors.Is(err, masterlogin.ErrNameLocked):
		// 409, sama seperti login ganda: keduanya konflik keadaan, bukan isian yang cacat.
		//
		// Ia tidak seharusnya terlihat pengguna sama sekali — layar mengunci isiannya. Bila
		// ia muncul, permintaannya tidak datang dari layar ini, atau layar dan server
		// membaca baris yang berbeda karena daftarnya sudah basi.
		return http.StatusConflict, ErrorResponse{
			Code: CodeNameLocked,
			Message: "Nama tidak dapat diubah. Login dibentuk dari Nama, dan mengubahnya " +
				"akan memindahkan kunci baris ini.",
			Detail: []ViolationDTO{{
				Field: "nama",
				Message: "Tutup form ini dan muat ulang daftarnya. Bila namanya memang harus " +
					"berbeda, tambahkan login baru — baris lama tidak dapat dinamai ulang.",
			}},
		}, true

	case errors.Is(err, masterlogin.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code: CodeNotFound,
			Message: "Login surveyor yang dimaksud tidak ditemukan. " +
				"Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
