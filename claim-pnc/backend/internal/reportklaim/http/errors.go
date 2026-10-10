package reportklaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/reportklaim"
	reportklaimsql "claim-pnc/internal/reportklaim/repo/sqlstore"
)

// Kode galat modul ini.
//
// Modul memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya ke penulis
// galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama sehingga klien
// tidak menghadapi dua bentuk galat yang berbeda. Alasannya sama dengan modul Inbox
// Laporan Klaim: kontrak galat aplikasi (`TKT-F1-004`) masih terhalang keputusan Work
// Owner, dan menambah kode ke modul auth berarti menyunting modul yang sudah selesai.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeCallerIncomplete = "profil_pemanggil_tidak_lengkap"

	// CodeReportNotReady: laporannya ADA di katalog tetapi belum dapat dijalankan.
	//
	// Ia BUKAN tidak_ditemukan. Yang pertama berarti kodenya salah — layar memperbaiki
	// pemanggilannya; yang ini berarti artefak Pega-nya belum ada, dan tidak ada yang
	// dapat diperbaiki dari sisi layar maupun sisi pengguna.
	CodeReportNotReady = "laporan_belum_tersedia"

	// CodeUnknownAction: tombol yang diminta tidak ada pada panel itu.
	CodeUnknownAction = "tombol_tidak_dikenal"

	// CodeQueryNotPorted: kuerinya belum selesai dipindahkan dari export Pega.
	//
	// Dipisahkan dari CodeReportNotReady meski keduanya berarti "belum dapat dijalankan".
	// Sebab dan pemiliknya berbeda:
	//
	//	laporan_belum_tersedia   artefak Pega tidak ada          → Tim Pega
	//	kueri_belum_dipindahkan  artefaknya ada, kami yang belum → tim pengembang
	//
	// Ia juga berlaku hanya pada SEBAGIAN lini bisnis, sehingga pesannya menyebutkan itu.
	CodeQueryNotPorted = "kueri_belum_dipindahkan"

	// CodeQueryFailed: kuerinya ADA dan dijalankan, tetapi basis data menolaknya.
	//
	// Dipisahkan dari galat internal umum karena yang harus dilakukan berbeda: galat
	// internal menuntut pembacaan kode, yang ini menuntut pembacaan pesan ORA pada log
	// peladen. Tanpa pemisahan itu keduanya tampil sebagai kalimat yang sama, dan
	// kalimat itu tidak mengarahkan ke mana pun.
	CodeQueryFailed = "laporan_gagal_dijalankan"

	// CodeMitraListUnavailable: Laporan Mitra kehilangan PENYARING barisnya.
	//
	// Dipisahkan dari CodeQueryFailed meski keduanya berarti laporan tidak terbit.
	// Yang harus dilakukan berbeda, dan pemiliknya pun berbeda:
	//
	//	laporan_gagal_dijalankan  basis data menolak kueri   → baca pesan ORA di log
	//	daftar_mitra_tidak_ada    koneksi kedua belum ada    → Infra mengisi ANEKA_*
	//
	// Tanpa pemisahan ini keduanya tampil sebagai kalimat yang sama, dan kalimat itu
	// mengirim orang membaca log untuk masalah yang tidak meninggalkan jejak di log.
	CodeMitraListUnavailable = "daftar_mitra_tidak_ada"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// errorBody adalah bentuk galat yang dikirim ke klien.
type errorBody struct {
	Kode   string          `json:"kode"`
	Pesan  string          `json:"pesan"`
	Detail []violationBody `json:"detail,omitempty"`
}

type violationBody struct {
	Isian string `json:"isian"`
	Pesan string `json:"pesan"`
}

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	// Penolakan basis data ditangani lebih dulu, karena jawabannya menyertakan ID
	// permintaan — dan ID itu hanya ada di sini, bukan di mapError yang murni.
	if errors.Is(err, reportklaimsql.ErrQueryFailed) {
		h.writeQueryFailure(w, r, err)
		return
	}
	status, body, known := mapError(err)
	if !known {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, status, body)
}

// writeQueryFailure menjawab penolakan basis data: ID permintaan ke pengguna, pesan
// ORA-nya ke log.
//
// Pembagian itu disengaja. Pesan ORA memuat nama tabel dan kolom, dan membocorkannya ke
// peramban adalah celah keamanan (`11-CROSSCUTTING.md` §1.2 aturan 5). Tetapi menahan
// SELURUHNYA membuat kegagalan menjadi buntu — itulah keadaan sebelum ini. ID permintaan
// adalah jembatan yang aman: ia tidak berarti apa-apa bagi penyerang, dan ia menunjuk
// tepat satu baris log bagi yang berhak membacanya.
func (h *Handler) writeQueryFailure(w http.ResponseWriter, r *http.Request, err error) {
	id := logging.RequestID(r.Context())
	if h.logger != nil {
		h.logger.ErrorContext(r.Context(), "laporan ditolak basis data",
			slog.String("jalur", r.URL.Path),
			slog.String("id_permintaan", id),
			slog.String("galat", err.Error()))
	}
	h.writeResponse(w, r, http.StatusInternalServerError, errorBody{
		Kode: CodeQueryFailed,
		Pesan: "Laporan ini gagal dijalankan oleh basis data. Keterangan lengkapnya ada " +
			"pada log peladen dengan nomor permintaan " + id + " — sampaikan nomor itu " +
			"kepada tim pengembang.",
	})
}

// mapError memetakan galat domain ke kode HTTP beserta badannya.
func mapError(err error) (int, errorBody, bool) {
	var validation *reportklaim.ValidationError
	if errors.As(err, &validation) {
		detail := make([]violationBody, 0, len(validation.Violation))
		for _, v := range validation.Violation {
			detail = append(detail, violationBody{Isian: v.Field, Pesan: v.Message})
		}
		// 422, bukan 400: permintaannya berbentuk benar tetapi melanggar aturan bisnis.
		// Frontend menanganinya berbeda — yang satu bug frontend, yang lain kesalahan
		// pengisian (`10-API-STRATEGY.md` §5).
		return http.StatusUnprocessableEntity, errorBody{
			Kode:   CodeValidationFailed,
			Pesan:  "Ada isian yang belum benar.",
			Detail: detail,
		}, true
	}

	switch {
	case errors.Is(err, reportklaim.ErrUnknownReport):
		return http.StatusNotFound, errorBody{
			Kode:  CodeNotFound,
			Pesan: "Laporan yang diminta tidak dikenal.",
		}, true

	case errors.Is(err, reportklaim.ErrReportNotReady):
		// 409, bukan 404: laporannya ADA, keadaannya yang belum memungkinkan.
		return http.StatusConflict, errorBody{
			Kode:  CodeReportNotReady,
			Pesan: "Laporan ini belum dapat dijalankan. Keterangannya tertera pada kartunya.",
		}, true

	case errors.Is(err, reportklaim.ErrUnknownAction):
		return http.StatusBadRequest, errorBody{
			Kode:  CodeUnknownAction,
			Pesan: "Tombol yang diminta tidak ada pada laporan ini.",
		}, true

	case errors.Is(err, reportklaimsql.ErrQueryNotPorted):
		return http.StatusConflict, errorBody{
			Kode: CodeQueryNotPorted,
			Pesan: "Laporan ini belum tersedia untuk lini bisnis yang dipilih. " +
				"Lini bisnis lain pada laporan yang sama tetap dapat diunduh.",
		}, true

	case errors.Is(err, reportklaimsql.ErrMitraListUnavailable):
		// 409, bukan 500: tidak ada yang rusak. Laporan Mitra menyaring barisnya dengan
		// daftar login mitra yang hidup di koneksi KEDUA portal, dan koneksi itu belum
		// terpasang (`R-03`).
		//
		// Pesannya menyebut sebabnya, dan itu disengaja: tanpa menyebutnya, yang sampai
		// ke pengguna hanyalah "berkas tidak dapat diunduh" — kalimat yang tidak
		// mengarahkan ke mana pun, dan yang membuat orang melaporkannya sebagai cacat
		// laporan padahal yang kurang sebuah konfigurasi.
		//
		// Yang TIDAK dilakukan: menerbitkan laporannya tanpa penyaring. Berkasnya akan
		// berisi SELURUH petugas alih-alih petugas mitra — tetap wajar dilihat, dan tanpa
		// satu pun tanda bahwa isinya bukan yang diminta.
		return http.StatusConflict, errorBody{
			Kode: CodeMitraListUnavailable,
			Pesan: "Laporan Mitra belum dapat diunduh: daftar login mitra dibaca dari " +
				"koneksi kedua portal, dan koneksi itu belum dikonfigurasi. " +
				"Sampaikan ke tim Infra untuk mengisi ANEKA_<PORTAL>_* pada konfigurasi aplikasi.",
		}, true

	case errors.Is(err, reportklaim.ErrCallerUnknown):
		return http.StatusInternalServerError, errorBody{
			Kode:  CodeCallerIncomplete,
			Pesan: "Identitas pemanggil tidak dapat dibaca.",
		}, true
	}
	return 0, errorBody{}, false
}

// logExportFailure mencatat kegagalan yang terjadi SETELAH header terkirim.
//
// Sesudah header terkirim, galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke
// pengguna berupa berkas CSV separuh jadi tanpa satu pun keterangan. Satu-satunya tempat
// keadaan itu terbaca adalah log.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	if h.logger == nil {
		return
	}
	h.logger.ErrorContext(r.Context(), "ekspor laporan terputus di tengah",
		slog.String("jalur", r.URL.Path),
		slog.String("id_permintaan", logging.RequestID(r.Context())),
		slog.String("sebab", err.Error()))
}
