package portalhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
)

// HeaderPortal adalah nama header tempat peramban menyebut portal yang sedang aktif.
//
// # Kenapa header, bukan bagian jalur atau parameter kueri
//
// Portal bukan bagian dari identitas sumber daya. `/api/master/status-progres-1` adalah
// sumber daya yang sama di setiap entitas; yang berbeda adalah basis data mana yang
// menjawabnya. Menaruhnya di jalur (`/api/ASM/master/...`) akan menggandakan setiap
// rute sebanyak jumlah entitas, dan menaruhnya di kueri membuatnya ikut tercatat di log
// peramban dan log proxy bersama seluruh URL.
//
// Awalan `X-` dipakai karena ini header khusus aplikasi, bukan header standar.
const HeaderPortal = "X-Portal"

type contextKey string

const activePortalKey contextKey = "konteks_portal_aktif"

// ErrorWriter menuliskan galat dalam bentuk respons HTTP. Bentuknya sama dengan yang
// dipakai modul lain supaya klien tidak perlu mengenali dua bentuk galat.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// Kode galat portal. Klien membedakan jenis galat lewat kode ini, tidak pernah dengan
// mencocokkan teks pesan.
//
// Ketiganya tinggal di modul portal, bukan di modul bisnis yang memakainya. Setiap
// modul bisnis yang menyentuh basis data entitas akan menemui ketiga galat yang sama,
// dan memetakannya sendiri-sendiri berarti empat modul berikutnya dapat menjawab tiga
// kode berbeda untuk keadaan yang sama.
const (
	CodeNotStated = "portal_tidak_disebut"
	CodeUnknown   = "portal_tidak_dikenal"
	CodeNotReady  = "portal_belum_siap"
)

// ErrorResponse adalah bentuk galat portal, sama dengan bentuk galat modul lain.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// WithPortalError membungkus penulis galat supaya galat portal terpetakan benar.
//
// Galat selain galat portal diteruskan ke `lanjut` apa adanya. Pola membungkus dipakai
// alih-alih menyunting penulis galat modul auth: kontrak galat yang mengikat seluruh
// aplikasi adalah TKT-F1-004 dan ia masih terhalang, sehingga menambah kode ke modul
// auth berarti menyunting modul yang sudah dinyatakan selesai.
//
// Kedua parameternya bertipe fungsi TANPA NAMA dengan sengaja. Setiap modul menamai
// tipe penulisnya sendiri, dan Go tidak mengizinkan nilai bertipe bernama dipakai
// sebagai tipe bernama lain walau tanda tangannya sama — parameter tanpa nama menerima
// semuanya, sehingga modul tetap tidak perlu saling mengimpor tipe.
func WithPortalError(
	next func(w http.ResponseWriter, r *http.Request, err error),
	writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any),
) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body, known := mapPortalError(err)
		if !known {
			next(w, r, err)
			return
		}
		writeResponse(w, r, status, body)
	}
}

func mapPortalError(err error) (int, ErrorResponse, bool) {
	switch {
	case errors.Is(err, portal.ErrNotStated):
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeNotStated,
			Message: "Portal entitas belum dipilih. Pilih portal lebih dulu sebelum membuka data entitas.",
		}, true

	case errors.Is(err, portal.ErrNotFound):
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeUnknown,
			Message: "Portal entitas tidak dikenal.",
		}, true

	case errors.Is(err, portal.ErrNotReady):
		// 503, bukan 400: tidak ada yang salah pada permintaannya. Entitasnya memang
		// belum dilayani karena kredensial basis datanya belum diisi, dan itu pekerjaan
		// tim infrastruktur — bukan sesuatu yang dapat diperbaiki pengguna.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:    CodeNotReady,
			Message: "Basis data portal entitas ini belum tersedia. Hubungi administrator Claim PNC.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}

// ActivePortalDeps adalah yang dibutuhkan middleware ActivePortal.
type ActivePortalDeps struct {
	// Repo membaca daftar portal. Ia dibaca pada setiap permintaan, bukan sekali saat
	// start: menambah entitas berarti menambah baris tabel, dan baris baru itu harus
	// berlaku tanpa merilis ulang aplikasi (ADR-0030).
	Repo portal.Repo

	// ReadyAliases menyebut portal yang koneksinya hidup.
	ReadyAliases func() []string

	Logger     *slog.Logger
	WriteError ErrorWriter
}

// ActivePortal menolak permintaan yang tidak menyebut portal yang sah, dan menaruh
// portal terpilih ke dalam context bila sah.
//
// Ia dipasang pada rute modul bisnis yang menyentuh basis data entitas — BUKAN pada
// rute daftar portal itu sendiri, yang justru dibutuhkan pengguna untuk memilih.
//
// Urutannya sesudah Autentikasi: pertanyaan "siapa pemanggil ini" harus terjawab lebih
// dulu daripada "entitas mana yang ia minta".
func ActivePortal(b ActivePortalDeps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			list, err := b.Repo.List(r.Context())
			if err != nil {
				b.WriteError(w, r, err)
				return
			}

			var ready []string
			if b.ReadyAliases != nil {
				ready = b.ReadyAliases()
			}

			selected, err := portal.SelectActive(list, ready, r.Header.Get(HeaderPortal))
			if err != nil {
				// Penolakan portal dicatat pada tingkat Warn, bukan Info: ia dapat
				// berarti antarmuka yang keliru, tetapi dapat juga berarti permintaan
				// yang disusun tangan ke entitas yang bukan haknya. Nilai header dicatat
				// karena ia alias entitas — bukan data nasabah dan bukan kredensial.
				logging.From(r.Context(), b.Logger).Warn("permintaan portal ditolak",
					slog.String("jalur", r.URL.Path),
					slog.String("portal_diminta", r.Header.Get(HeaderPortal)),
					slog.String("sebab", err.Error()),
				)
				b.WriteError(w, r, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithActivePortal(r.Context(), selected)))
		})
	}
}

// WithActivePortal menaruh portal yang aktif ke context.
func WithActivePortal(ctx context.Context, p portal.Portal) context.Context {
	return context.WithValue(ctx, activePortalKey, p)
}

// ActivePortalFrom mengambil portal yang aktif dari context.
//
// Nilai kedua false bila permintaan tidak melewati middleware ActivePortal. Handler modul
// bisnis WAJIB memeriksanya dan menolak bila false — bukan melanjutkan dengan portal
// utama sebagai cadangan.
func ActivePortalFrom(ctx context.Context) (portal.Portal, bool) {
	p, exists := ctx.Value(activePortalKey).(portal.Portal)
	return p, exists
}
