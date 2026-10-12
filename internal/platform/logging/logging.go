// Package logging menyediakan log terstruktur dan ID permintaan.
//
// Satu aturan mengikat seluruh aplikasi dan ditegakkan di sini: kredensial dan token
// sesi tidak pernah masuk log, termasuk pada jalur galat dan termasuk sebagiannya.
// Paket ini karena itu tidak menyediakan cara apa pun untuk menulis nilai mentah
// keduanya — yang tersedia hanya Mask().
package logging

import (
	"context"
	"log/slog"
	"os"
	"strconv"
)

type contextKey string

const requestIDKey contextKey = "id_permintaan"

// New membuat logger terstruktur. Keluarannya JSON supaya dapat dibaca perkakas,
// bukan hanya mata manusia.
func New(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// WithRequestID menaruh ID permintaan ke dalam context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID mengambil ID permintaan dari context; kosong bila tidak ada.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// From mengembalikan logger yang sudah membawa ID permintaan, sehingga seluruh baris
// log satu permintaan dapat dirangkai kembali.
func From(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if id := RequestID(ctx); id != "" {
		return logger.With(slog.String("id_permintaan", id))
	}
	return logger
}

// Mask mengubah nilai sensitif menjadi bentuk yang tidak dapat dipakai kembali,
// tetapi masih cukup untuk membedakan satu nilai dari yang lain saat menelusuri galat.
// Nilai aslinya tidak pernah dikembalikan.
func Mask(value string) string {
	if value == "" {
		return "(kosong)"
	}
	return "(disamarkan, " + strconv.Itoa(len([]rune(value))) + " karakter)"
}
