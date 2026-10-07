package mastertipespareparthttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat modul ini.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih terhalang
// keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth, dan
// menambah kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan
// sisanya ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama
// sehingga klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Begitu TKT-F1-004 diputuskan, pemetaan ini pindah ke tempat bersama dan berkas ini
// tinggal memakainya. Utang itu dicatat di docs/keputusan-implementasi.md.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"

	// CodeNameTaken dibedakan dari CodeValidationFailed dengan sengaja.
	//
	// Ia konflik KEADAAN, bukan isian yang cacat: namanya sah, dan yang salah hanyalah bahwa
	// nama itu sudah dipakai tipe lain. Perbaikannya bukan "betulkan isian" melainkan "buka
	// baris yang sudah ada, atau pakai nama lain", dan layar menanganinya berbeda.
	//
	// Namanya `kunci_tipe_sparepart_sudah_ada` supaya sebentuk dengan
	// `kunci_kategori_sparepart_sudah_ada` dan `kunci_sparepart_sudah_ada` — ketiganya
	// menyatakan hal yang sama, yaitu kunci alami yang bentrok.
	CodeNameTaken = "kunci_tipe_sparepart_sudah_ada"

	// CodeCategoryNotFound muncul bila kategori yang dipilih tidak ada atau belum disetujui.
	//
	// Dibedakan dari CodeValidationFailed karena penyebabnya BUKAN isian yang salah ketik:
	// pengguna memilihnya dari dropdown, dan dropdown itu berisi apa yang sah saat form
	// dibuka. Yang paling mungkin terjadi adalah kategorinya ditolak petugas lain sementara
	// form terbuka — dan perbaikannya adalah memuat ulang daftar pilihannya, bukan
	// membetulkan ketikan.
	CodeCategoryNotFound = "kategori_sparepart_tidak_ditemukan"

	// CodeUnknownStatus muncul bila status yang diminta di luar "0", "1", "2".
	CodeUnknownStatus = "status_tidak_dikenal"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter = apierror.JSONWriter

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	apierror.Write(w, r, err, mapError, h.Logger, h.WriteResponse, h.WriteError)
}

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *mastertipesparepart.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan bisnis.
		// Frontend menanganinya berbeda — 400 berarti ada cacat di frontend, 422 berarti
		// pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5). Pada modul
		// berisi dua isian wajib, bentuk itu benar-benar terpakai.
		detail := apierror.ColumnErrors(validationError.Violation)
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, mastertipesparepart.ErrNameTaken):
		// 409, bukan 422 — ia konflik keadaan; lihat CodeNameTaken.
		//
		// Keterangan pada `detail` menyebut DUA kemungkinan yang paling mengejutkan, dan
		// keduanya memang tidak terlihat dari tab yang sedang dibuka:
		//
		//  1. nama itu dipakai baris yang sudah DITOLAK — pemeriksaannya tidak menyaring
		//     APPROVAL sama sekali;
		//  2. nama itu dipakai tipe di kategori LAIN — pemeriksaannya juga tidak menyaring
		//     kategori.
		//
		// Keduanya perilaku sistem lama yang ditiru apa adanya (`P-5`), dan tanpa keterangan
		// ini penolakannya tidak dapat dijelaskan dari layar.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeNameTaken,
			Message: "Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain.",
			Detail: []ViolationDTO{{
				Field: "nama_tipe_sparepart",
				Message: "Nama ini sudah dipakai tipe lain — termasuk tipe di kategori yang " +
					"berbeda, dan termasuk tipe yang sudah ditolak. Periksa juga tab Reject.",
			}},
		}, true

	case errors.Is(err, mastertipesparepart.ErrCategoryNotFound):
		// 409, bukan 422 — sama seperti ErrNameTaken, ia konflik keadaan dan bukan isian yang
		// cacat. Pengguna memilihnya dari dropdown; yang berubah adalah dunia di luar formnya.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCategoryNotFound,
			Message: "Kategori sparepart yang dipilih sudah tidak tersedia. " +
				"Muat ulang daftar pilihannya, lalu pilih kembali.",
			Detail: []ViolationDTO{{
				Field: "id_kategori_sparepart",
				Message: "Kategori ini tidak ada, atau persetujuannya dicabut sementara form " +
					"terbuka. Hanya kategori berstatus Approve yang dapat dipilih.",
			}},
		}, true

	case errors.Is(err, mastertipesparepart.ErrUnknownStatus):
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

	case errors.Is(err, mastertipesparepart.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code: CodeNotFound,
			Message: "Tipe sparepart yang dimaksud tidak ditemukan. " +
				"Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
