// Package logging menyediakan log terstruktur dan ID permintaan.
//
// Satu aturan mengikat seluruh aplikasi dan ditegakkan di sini: kredensial dan token
// sesi tidak pernah masuk log, termasuk pada jalur galat dan termasuk sebagiannya.
// Paket ini karena itu tidak menyediakan cara apa pun untuk menulis nilai mentah
// keduanya — yang tersedia hanya Samarkan().
package logging

import (
	"context"
	"log/slog"
	"os"
	"strconv"
)

type kunciKonteks string

const kunciIDPermintaan kunciKonteks = "id_permintaan"

// Baru membuat logger terstruktur. Keluarannya JSON supaya dapat dibaca perkakas,
// bukan hanya mata manusia.
func Baru(taraf slog.Level) *slog.Logger {
	opsi := &slog.HandlerOptions{Level: taraf}
	return slog.New(slog.NewJSONHandler(os.Stdout, opsi))
}

// DenganIDPermintaan menaruh ID permintaan ke dalam context.
func DenganIDPermintaan(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, kunciIDPermintaan, id)
}

// IDPermintaan mengambil ID permintaan dari context; kosong bila tidak ada.
func IDPermintaan(ctx context.Context) string {
	id, _ := ctx.Value(kunciIDPermintaan).(string)
	return id
}

// Dari mengembalikan logger yang sudah membawa ID permintaan, sehingga seluruh baris
// log satu permintaan dapat dirangkai kembali.
func Dari(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if id := IDPermintaan(ctx); id != "" {
		return logger.With(slog.String("id_permintaan", id))
	}
	return logger
}

// Samarkan mengubah nilai sensitif menjadi bentuk yang tidak dapat dipakai kembali,
// tetapi masih cukup untuk membedakan satu nilai dari yang lain saat menelusuri galat.
// Nilai aslinya tidak pernah dikembalikan.
func Samarkan(nilai string) string {
	if nilai == "" {
		return "(kosong)"
	}
	return "(disamarkan, " + strconv.Itoa(len([]rune(nilai))) + " karakter)"
}
