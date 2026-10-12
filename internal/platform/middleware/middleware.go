// Package middleware memuat lapisan yang membungkus setiap permintaan HTTP.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"claim-pnc/internal/platform/logging"
)

// RequestID memberi setiap permintaan satu pengenal dan menaruhnya di context serta
// di header respons, sehingga satu keluhan pengguna dapat ditelusuri ke satu baris log.
func RequestID(berikutnya http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = randomID()
		}
		w.Header().Set("X-Request-Id", id)
		berikutnya.ServeHTTP(w, r.WithContext(logging.WithRequestID(r.Context(), id)))
	})
}

// Log menuliskan satu baris per permintaan.
//
// Yang dicatat adalah metode, jalur, status, dan lamanya. Header Authorization dan isi
// badan permintaan TIDAK PERNAH dicatat — di situlah kredensial dan token berada.
func Log(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(berikutnya http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mulai := time.Now()
			perekam := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			berikutnya.ServeHTTP(perekam, r)
			logging.From(r.Context(), logger).Info("permintaan selesai",
				slog.String("metode", r.Method),
				slog.String("jalur", r.URL.Path),
				slog.Int("status", perekam.status),
				slog.Duration("lama", time.Since(mulai)),
			)
		})
	}
}

// Recover menahan panic agar satu permintaan yang rusak tidak menjatuhkan proses yang
// sedang melayani permintaan lain.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(berikutnya http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if cause := recover(); cause != nil {
					logging.From(r.Context(), logger).Error("permintaan panik",
						slog.Any("sebab", cause),
						slog.String("jalur", r.URL.Path),
					)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"kode":"galat_internal","pesan":"Terjadi kesalahan pada sistem."}`))
				}
			}()
			berikutnya.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (p *statusRecorder) WriteHeader(code int) {
	p.status = code
	p.ResponseWriter.WriteHeader(code)
}

func randomID() string {
	content := make([]byte, 8)
	if _, err := rand.Read(content); err != nil {
		return "tanpa-id"
	}
	return hex.EncodeToString(content)
}
