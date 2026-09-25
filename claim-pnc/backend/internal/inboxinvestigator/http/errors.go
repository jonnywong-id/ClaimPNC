package inboxinvestigatorhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
)

// Kode galat modul ini.
//
// # Modul ini TIDAK punya galat domain sendiri, dan itu bukan kelalaian
//
// Modul yang menulis memetakan galatnya sendiri — validasi gagal, kunci bentrok, baris sudah
// diambil orang lain — karena ketiganya lahir dari jalur TULIS. Modul ini hanya membaca
// antrean (lihat banner paket inboxinvestigator), sehingga tidak ada satu pun keadaan yang
// dapat ditolak atas dasar aturan bisnis.
//
// Antrean yang KOSONG bukan galat. Ia justru keadaan yang diharapkan pada inbox yang sudah
// dikerjakan, dan menjawabnya dengan galat akan membuat layar menampilkan pesan merah
// setiap kali pekerjaan habis.
//
// Yang tersisa hanyalah dua kelompok, dan keduanya sudah punya pemiliknya masing-masing:
//
//	galat PORTAL     dipetakan portalhttp.WithPortalError, satu pemetaan untuk seluruh modul
//	galat TEKNIS     diserahkan ke penulis galat bersama; 500 dengan pesan umum
//
// Satu-satunya kode yang didefinisikan di sini adalah CodeMalformedRequest, dan ia ada
// supaya penyaring kueri yang cacat tidak jatuh menjadi 500 — kegagalan klien tidak boleh
// terbaca sebagai kegagalan server.
const (
	CodeMalformedRequest = "permintaan_cacat"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// writeModuleError meneruskan galat ke penulis bersama, dan mencatat yang perlu dicatat.
//
// Tidak ada pemetaan di sini; lihat catatan pada blok konstanta di atas.
//
// # Galat portal TIDAK dicatat sebagai kegagalan
//
// Permintaan tanpa header portal, atau menyebut portal yang tidak dikenal, adalah kesalahan
// KLIEN — dijawab 400 oleh portalhttp.WithPortalError. Mencatatnya sebagai Error akan
// menenggelamkan kegagalan sungguhan di antara permintaan yang hanya dibuka sebelum portal
// dipilih, dan itu terjadi setiap kali seseorang membuka aplikasi.
//
// `ErrNotReady` DIKECUALIKAN dari pengecualian itu: ia dijawab 503, dan penyebabnya memang
// pekerjaan administrator — kredensial basis data entitas yang belum diisi. Ia harus
// terlihat.
//
// Rincian galat internal TIDAK pernah dikirim ke peramban; ia hanya masuk log.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	clientMistake := errors.Is(err, portal.ErrNotStated) || errors.Is(err, portal.ErrNotFound)
	if !clientMistake {
		logging.From(r.Context(), h.logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
	h.writeError(w, r, err)
}
