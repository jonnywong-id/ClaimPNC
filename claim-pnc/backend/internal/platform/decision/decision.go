// Package decision memuat langkah bersama keputusan persetujuan borongan pada layar master:
// merapikan daftar baris bercentang dan mencatat keputusan yang tidak menyentuh seluruhnya.
package decision

import (
	"log/slog"
	"strings"
)

// UniqueIDs memangkas setiap kunci, membuang yang kosong dan yang ganda, dengan urutan tetap.
//
// Kunci ganda dibuang: layar dapat mengirim baris yang sama dua kali bila daftarnya dimuat
// ulang saat centang masih terpasang, dan jumlah baris berubah yang dilaporkan harus
// mencerminkan baris, bukan centang.
func UniqueIDs(id []string) []string {
	wanted := make([]string, 0, len(id))
	seen := make(map[string]bool, len(id))
	for _, one := range id {
		clean := strings.TrimSpace(one)
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		wanted = append(wanted, clean)
	}
	return wanted
}

// WarnPartial mencatat keputusan yang tidak mengubah seluruh baris yang dipilih.
//
// Bukan galat: baris yang tidak berubah adalah baris yang sudah tidak ada, atau sudah
// berstatus itu. Tetapi selisihnya berarti layar menampilkan daftar yang sudah basi, dan itu
// layak terbaca. Tanpa logger, atau bila seluruhnya berubah, tidak ada yang dicatat.
func WarnPartial(logger *slog.Logger, message, portalAlias string, wanted, changed int, by string) {
	if changed == wanted || logger == nil {
		return
	}
	logger.Warn(message,
		slog.String("portal", portalAlias),
		slog.Int("dipilih", wanted),
		slog.Int("berubah", changed),
		slog.String("oleh", by))
}
