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

type kunciKonteks string

const kunciPortalAktif kunciKonteks = "konteks_portal_aktif"

// PenulisGalat menuliskan galat dalam bentuk respons HTTP. Bentuknya sama dengan yang
// dipakai modul lain supaya klien tidak perlu mengenali dua bentuk galat.
type PenulisGalat func(w http.ResponseWriter, r *http.Request, err error)

// PenulisRespon menuliskan badan respons.
type PenulisRespon func(w http.ResponseWriter, r *http.Request, status int, badan any)

// Kode galat portal. Klien membedakan jenis galat lewat kode ini, tidak pernah dengan
// mencocokkan teks pesan.
//
// Ketiganya tinggal di modul portal, bukan di modul bisnis yang memakainya. Setiap
// modul bisnis yang menyentuh basis data entitas akan menemui ketiga galat yang sama,
// dan memetakannya sendiri-sendiri berarti empat modul berikutnya dapat menjawab tiga
// kode berbeda untuk keadaan yang sama.
const (
	KodeTidakDisebut = "portal_tidak_disebut"
	KodeTidakDikenal = "portal_tidak_dikenal"
	KodeBelumSiap    = "portal_belum_siap"
)

// ResponsGalat adalah bentuk galat portal, sama dengan bentuk galat modul lain.
type ResponsGalat struct {
	Kode  string `json:"kode"`
	Pesan string `json:"pesan"`
}

// DenganGalatPortal membungkus penulis galat supaya galat portal terpetakan benar.
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
func DenganGalatPortal(
	lanjut func(w http.ResponseWriter, r *http.Request, err error),
	tulisRespon func(w http.ResponseWriter, r *http.Request, status int, badan any),
) PenulisGalat {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, badan, dikenali := petakanGalatPortal(err)
		if !dikenali {
			lanjut(w, r, err)
			return
		}
		tulisRespon(w, r, status, badan)
	}
}

func petakanGalatPortal(err error) (int, ResponsGalat, bool) {
	switch {
	case errors.Is(err, portal.ErrTidakDisebut):
		return http.StatusBadRequest, ResponsGalat{
			Kode:  KodeTidakDisebut,
			Pesan: "Portal entitas belum dipilih. Pilih portal lebih dulu sebelum membuka data entitas.",
		}, true

	case errors.Is(err, portal.ErrTidakAda):
		return http.StatusBadRequest, ResponsGalat{
			Kode:  KodeTidakDikenal,
			Pesan: "Portal entitas tidak dikenal.",
		}, true

	case errors.Is(err, portal.ErrBelumSiap):
		// 503, bukan 400: tidak ada yang salah pada permintaannya. Entitasnya memang
		// belum dilayani karena kredensial basis datanya belum diisi, dan itu pekerjaan
		// tim infrastruktur — bukan sesuatu yang dapat diperbaiki pengguna.
		return http.StatusServiceUnavailable, ResponsGalat{
			Kode:  KodeBelumSiap,
			Pesan: "Basis data portal entitas ini belum tersedia. Hubungi administrator Claim PNC.",
		}, true

	default:
		return 0, ResponsGalat{}, false
	}
}

// BahanPortalAktif adalah yang dibutuhkan middleware PortalAktif.
type BahanPortalAktif struct {
	// Repo membaca daftar portal. Ia dibaca pada setiap permintaan, bukan sekali saat
	// start: menambah entitas berarti menambah baris tabel, dan baris baru itu harus
	// berlaku tanpa merilis ulang aplikasi (ADR-0030).
	Repo portal.Repo

	// AliasSiap menyebut portal yang koneksinya hidup.
	AliasSiap func() []string

	Logger     *slog.Logger
	TulisGalat PenulisGalat
}

// PortalAktif menolak permintaan yang tidak menyebut portal yang sah, dan menaruh
// portal terpilih ke dalam context bila sah.
//
// Ia dipasang pada rute modul bisnis yang menyentuh basis data entitas — BUKAN pada
// rute daftar portal itu sendiri, yang justru dibutuhkan pengguna untuk memilih.
//
// Urutannya sesudah Autentikasi: pertanyaan "siapa pemanggil ini" harus terjawab lebih
// dulu daripada "entitas mana yang ia minta".
func PortalAktif(b BahanPortalAktif) func(http.Handler) http.Handler {
	return func(berikutnya http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			daftar, err := b.Repo.Daftar(r.Context())
			if err != nil {
				b.TulisGalat(w, r, err)
				return
			}

			var siap []string
			if b.AliasSiap != nil {
				siap = b.AliasSiap()
			}

			dipilih, err := portal.PilihAktif(daftar, siap, r.Header.Get(HeaderPortal))
			if err != nil {
				// Penolakan portal dicatat pada tingkat Warn, bukan Info: ia dapat
				// berarti antarmuka yang keliru, tetapi dapat juga berarti permintaan
				// yang disusun tangan ke entitas yang bukan haknya. Nilai header dicatat
				// karena ia alias entitas — bukan data nasabah dan bukan kredensial.
				logging.Dari(r.Context(), b.Logger).Warn("permintaan portal ditolak",
					slog.String("jalur", r.URL.Path),
					slog.String("portal_diminta", r.Header.Get(HeaderPortal)),
					slog.String("sebab", err.Error()),
				)
				b.TulisGalat(w, r, err)
				return
			}

			berikutnya.ServeHTTP(w, r.WithContext(DenganPortalAktif(r.Context(), dipilih)))
		})
	}
}

// DenganPortalAktif menaruh portal yang aktif ke context.
func DenganPortalAktif(ctx context.Context, p portal.Portal) context.Context {
	return context.WithValue(ctx, kunciPortalAktif, p)
}

// PortalAktifDari mengambil portal yang aktif dari context.
//
// Nilai kedua false bila permintaan tidak melewati middleware PortalAktif. Handler modul
// bisnis WAJIB memeriksanya dan menolak bila false — bukan melanjutkan dengan portal
// utama sebagai cadangan.
func PortalAktifDari(ctx context.Context) (portal.Portal, bool) {
	p, ada := ctx.Value(kunciPortalAktif).(portal.Portal)
	return p, ada
}
