package authhttp

import (
	"context"
	"net/http"
	"strings"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/usecase"
)

type kunciKonteks string

const kunciPengguna kunciKonteks = "konteks_pengguna"

// PemeriksaSesi adalah bagian usecase yang dibutuhkan middleware ini. Ia dinyatakan
// sebagai antarmuka sempit supaya middleware dapat diuji tanpa membentuk seluruh
// layanan.
type PemeriksaSesi interface {
	Periksa(ctx context.Context, token auth.Token) (usecase.Konteks, error)
}

// Autentikasi menolak permintaan yang tidak membawa sesi yang sah, dan menaruh
// identitas pemanggil ke dalam context bila sah.
//
// Identitas diteruskan lewat context, bukan lewat parameter berantai maupun variabel
// global (docs/Steering/08-TECHNICAL-STRATEGY.md §4.6).
//
// CATATAN LINGKUP: yang diperiksa di sini baru "apakah pemanggil punya sesi yang sah".
// Pemeriksaan "apakah peran pemanggil berwenang atas endpoint ini" adalah TKT-F3-005
// yang belum dikerjakan, dan ia bergantung pada tabel peran TKT-F3-004 yang masih
// terhalang artefak.
func Autentikasi(pemeriksa PemeriksaSesi, tulisGalat PenulisGalat) func(http.Handler) http.Handler {
	return func(berikutnya http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ada := TokenDariPermintaan(r)
			if !ada {
				tulisGalat(w, r, auth.ErrSesiTidakDitemukan)
				return
			}
			konteks, err := pemeriksa.Periksa(r.Context(), token)
			if err != nil {
				tulisGalat(w, r, err)
				return
			}
			berikutnya.ServeHTTP(w, r.WithContext(DenganKonteksPengguna(r.Context(), konteks)))
		})
	}
}

// DenganKonteksPengguna menaruh identitas pemanggil ke context.
func DenganKonteksPengguna(ctx context.Context, k usecase.Konteks) context.Context {
	return context.WithValue(ctx, kunciPengguna, k)
}

// KonteksPengguna mengambil identitas pemanggil dari context. Nilai kedua bernilai
// false bila permintaan tidak melewati middleware Autentikasi.
func KonteksPengguna(ctx context.Context) (usecase.Konteks, bool) {
	k, ada := ctx.Value(kunciPengguna).(usecase.Konteks)
	return k, ada
}

// TokenDariPermintaan membaca token sesi dari header Authorization dengan skema Bearer.
//
// Token hanya dibaca dari header, tidak pernah dari query string: nilai di URL ikut
// tercatat di log peramban, log proxy, dan header Referer.
func TokenDariPermintaan(r *http.Request) (auth.Token, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", false
	}
	bagian := strings.SplitN(header, " ", 2)
	if len(bagian) != 2 || !strings.EqualFold(bagian[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(bagian[1])
	if token == "" {
		return "", false
	}
	return auth.Token(token), true
}
