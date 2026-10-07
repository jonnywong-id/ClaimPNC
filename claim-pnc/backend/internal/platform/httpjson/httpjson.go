// Package httpjson mengurai badan permintaan JSON dengan aturan yang sama di setiap modul.
//
// Sebelumnya setiap handler menyalin langkah yang sama: batasi ukuran badan, tolak field yang
// tidak dikenal, tolak data sesudah objek pertama, dan jawab 400 dengan badan galat modul.
// Badan galatnya tetap milik modul — yang dipusatkan di sini hanya langkahnya.
package httpjson

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"claim-pnc/internal/platform/apierror"
)

// Decode mengurai badan ke target secara KETAT: ukurannya dibatasi maxBytes, field yang tidak
// dikenal ditolak, dan badan yang memuat lebih dari satu dokumen JSON juga ditolak.
// Kegagalan dijawab 400 dengan badan malformed; nilai balik false berarti responsnya sudah
// ditulis.
//
// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan badan permintaan.
func Decode(w http.ResponseWriter, r *http.Request, maxBytes int64, target any,
	writeJSON apierror.JSONWriter, malformed any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, r, http.StatusBadRequest, malformed)
		return false
	}
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		writeJSON(w, r, http.StatusBadRequest, malformed)
		return false
	}
	return true
}

// DecodeLoose mengurai badan ke target dengan batas ukuran saja: field yang tidak dikenal
// diabaikan dan data sesudah objek pertama tidak diperiksa. Dipakai modul yang sejak awal
// menerima badan seperti itu.
func DecodeLoose(w http.ResponseWriter, r *http.Request, maxBytes int64, target any,
	writeJSON apierror.JSONWriter, malformed any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeJSON(w, r, http.StatusBadRequest, malformed)
		return false
	}
	return true
}
