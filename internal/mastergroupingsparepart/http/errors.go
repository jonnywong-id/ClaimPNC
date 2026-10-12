package mastergroupingspareparthttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/mastergroupingsparepart"
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
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya ke
// penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama sehingga klien
// tidak menghadapi dua bentuk galat yang berbeda.
//
// Begitu TKT-F1-004 diputuskan, pemetaan ini pindah ke tempat bersama dan berkas ini tinggal
// memakainya. Utang itu dicatat di docs/keputusan-implementasi.md.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"

	// CodeDuplicate dibedakan dari CodeValidationFailed dengan sengaja.
	//
	// Ia konflik KEADAAN, bukan isian yang cacat: keempat isian kuncinya sah, dan yang salah
	// hanyalah bahwa kombinasinya sudah dipakai baris lain. Perbaikannya bukan "betulkan
	// isian" melainkan "buka baris yang sudah ada, atau pilih kombinasi lain", dan layar
	// menanganinya berbeda.
	//
	// SATU kode untuk keempat kolomnya, bukan empat kode terpisah: kuncinya memang satu —
	// keempat kolom bersama-sama — sehingga tidak ada pilihan isian mana yang harus disorot.
	CodeDuplicate = "grouping_sudah_ada"

	// CodePartNotFound muncul bila nomor sparepart yang diketik tidak ada di Master Sparepart.
	//
	// Ia dibedakan dari CodeNotFound: yang tidak ditemukan bukan baris yang sedang dibuka,
	// melainkan acuan yang diketik pengguna — dan layar menanganinya dengan menyorot isiannya,
	// bukan dengan menutup form.
	CodePartNotFound = "sparepart_tidak_ditemukan"

	// CodeUnknownStatus muncul bila status yang diminta di luar "0", "1", "2".
	CodeUnknownStatus = "status_tidak_dikenal"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan jaringan, cacat
		// pemrograman — diserahkan ke penulis bersama, yang menjawab 500 dengan pesan umum dan
		// menaruh rinciannya di log saja. Rincian galat internal tidak pernah dikirim ke
		// peramban.
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
	var validationError *mastergroupingsparepart.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan bisnis.
		// Frontend menanganinya berbeda — 400 berarti ada cacat di frontend, 422 berarti
		// pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5).
		// `Activity/UpdateGroupingSparepartHE_act` pun menyusun dua pesannya berdampingan lalu
		// menampilkannya bersamaan.
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, p := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: p.Field, Message: p.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, mastergroupingsparepart.ErrDuplicate):
		// 409, bukan 422. Padanan pesan `local.err` — "Data sudah ada" — pada
		// `Activity/UpdateGroupingSparepartHE_act-Act.xml` langkah 15.
		return http.StatusConflict, ErrorResponse{
			Code: CodeDuplicate,
			Message: "Data sudah ada. Kombinasi nomor sparepart, nama panel, no rangka, " +
				"dan sisi ini sudah dipakai grouping lain.",
			Detail: []ViolationDTO{{
				Field: "nomor_sparepart",
				Message: "Kombinasi keempat isian ini sudah dipakai. " +
					"Ubah salah satunya, atau buka grouping yang sudah ada.",
			}},
		}, true

	case errors.Is(err, mastergroupingsparepart.ErrPartNotFound):
		// 404, dan menempel pada isiannya. Padanan pesan `local.mssg` — "Data Sparepart tidak
		// ditemukan" — pada `Activity/SetDataSparepart-Act.xml`.
		//
		// Ia dipakai jalur pencarian sparepart saja. Pada jalur simpan, keadaan yang sama
		// datang sebagai ValidationError yang menempel pada isian nomor sparepart; lihat
		// usecase.resolvePart.
		return http.StatusNotFound, ErrorResponse{
			Code:    CodePartNotFound,
			Message: "Data Sparepart tidak ditemukan. Periksa nomornya di Master Sparepart.",
			Detail: []ViolationDTO{{
				Field:   "nomor_sparepart",
				Message: "Nomor sparepart ini tidak ada di Master Sparepart.",
			}},
		}, true

	case errors.Is(err, mastergroupingsparepart.ErrGroupChassisNotFound):
		// 422 dan menempel pada isiannya. Padanan `local.err2` — "Nomor rangka dalam grouping
		// tidak ditemukan".
		//
		// Jalur simpan mengubahnya menjadi ValidationError lebih dulu; pemetaan di sini menjaga
		// jalur mana pun yang meneruskannya apa adanya tetap terbaca benar.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail: []ViolationDTO{{
				Field: "grouping_dengan_no_rangka",
				Message: "Nomor rangka dalam grouping tidak ditemukan. " +
					"Isi dengan nomor rangka yang sudah pernah didaftarkan, atau kosongkan untuk membuka grup baru.",
			}},
		}, true

	case errors.Is(err, mastergroupingsparepart.ErrUnknownStatus):
		// 422 dan menempel pada isian `status`: ia memang datang dari permintaan, dan
		// satu-satunya cara ia salah adalah klien mengirim nilai di luar ketiganya.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeUnknownStatus,
			Message: "Status persetujuan tidak dikenal.",
			Detail: []ViolationDTO{{
				Field:   "status",
				Message: `Status hanya boleh "0" menunggu, "1" disetujui, atau "2" ditolak.`,
			}},
		}, true

	case errors.Is(err, mastergroupingsparepart.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Grouping yang dimaksud tidak ditemukan. Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
