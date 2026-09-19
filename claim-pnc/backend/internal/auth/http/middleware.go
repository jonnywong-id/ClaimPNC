package authhttp

import (
	"context"
	"net/http"
	"strings"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/usecase"
)

type contextKey string

const callerKey contextKey = "konteks_pengguna"

// SessionChecker adalah bagian usecase yang dibutuhkan middleware ini. Ia dinyatakan
// sebagai antarmuka sempit supaya middleware dapat diuji tanpa membentuk seluruh
// layanan.
type SessionChecker interface {
	Check(ctx context.Context, token auth.Token) (usecase.Caller, error)
}

// Authenticate menolak permintaan yang tidak membawa sesi yang sah, dan menaruh
// identitas pemanggil ke dalam context bila sah.
//
// Identitas diteruskan lewat context, bukan lewat parameter berantai maupun variabel
// global (docs/Steering/08-TECHNICAL-STRATEGY.md §4.6).
//
// CATATAN LINGKUP: yang diperiksa di sini baru "apakah pemanggil punya sesi yang sah".
// Pemeriksaan "apakah peran pemanggil berwenang atas endpoint ini" adalah TKT-F3-005
// yang belum dikerjakan, dan ia bergantung pada tabel peran TKT-F3-004 yang masih
// terhalang artefak.
func Authenticate(pemeriksa SessionChecker, writeError ErrorWriter) func(http.Handler) http.Handler {
	return func(berikutnya http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, existing := TokenFromRequest(r)
			if !existing {
				writeError(w, r, auth.ErrSessionNotFound)
				return
			}
			baseCtx, err := pemeriksa.Check(r.Context(), token)
			if err != nil {
				writeError(w, r, err)
				return
			}
			berikutnya.ServeHTTP(w, r.WithContext(WithCaller(r.Context(), baseCtx)))
		})
	}
}

// WithCaller menaruh identitas pemanggil ke context.
func WithCaller(ctx context.Context, k usecase.Caller) context.Context {
	return context.WithValue(ctx, callerKey, k)
}

// CallerFromContext mengambil identitas pemanggil dari context. Nilai kedua bernilai
// false bila permintaan tidak melewati middleware Authenticate.
func CallerFromContext(ctx context.Context) (usecase.Caller, bool) {
	k, existing := ctx.Value(callerKey).(usecase.Caller)
	return k, existing
}

// TokenFromRequest membaca token sesi dari header Authorization dengan skema Bearer.
//
// Token hanya dibaca dari header, tidak pernah dari query string: nilai di URL ikut
// tercatat di log peramban, log proxy, dan header Referer.
func TokenFromRequest(r *http.Request) (auth.Token, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", false
	}
	part := strings.SplitN(header, " ", 2)
	if len(part) != 2 || !strings.EqualFold(part[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(part[1])
	if token == "" {
		return "", false
	}
	return auth.Token(token), true
}
