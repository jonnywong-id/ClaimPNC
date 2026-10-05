package inboxrclpuclhttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	inboxrclpuclhttp "claim-pnc/internal/inboxrclpucl/http"
)

// TestSetiapGalatModulPunyaPemetaan menjaga agar tidak satu pun galat modul ini jatuh ke
// jawaban 500 generik.
//
// # Kenapa penjaga ini ada
//
// Dua galat sudah terlanjur jatuh ke sana, dan keduanya tidak menghasilkan satu pun tanda
// selain baris log di peladen:
//
//   - `ErrTechnicalPICUnknown` — dikembalikan tombol "Kirim Ke Analyst" ketika klaim belum
//     punya PIC Teknik. Petugas membaca "Terjadi kesalahan pada sistem." dan melapor bahwa
//     aplikasinya rusak, padahal yang kurang adalah satu isian pada klaimnya.
//   - `ErrDocumentNotFound` — dikembalikan saat dokumen tidak ada atau bukan milik klaim itu.
//
// Keduanya sudah dipetakan. Penjaga ini yang membuat galat BERIKUTNYA tidak mengulanginya:
// galat baru yang lupa dipetakan gagal di sini, bukan di layar petugas.
//
// # Kenapa daftarnya ditulis tangan
//
// Go tidak dapat menelusuri variabel sebuah paket saat berjalan. Daftar ini karena itu
// disalin dari `inboxrclpucl.go`, dan menambah galat baru di sana menuntut menambahnya di
// sini pula — biaya yang disengaja: ia memaksa pertanyaan "jawaban HTTP apa untuk galat ini"
// ditanyakan saat galatnya dibuat, bukan saat petugas menemukannya.
func TestSetiapGalatModulPunyaPemetaan(t *testing.T) {
	galat := map[string]error{
		"ErrClaimNotFound":          inboxrclpucl.ErrClaimNotFound,
		"ErrDocumentNotFound":       inboxrclpucl.ErrDocumentNotFound,
		"ErrActionNotAvailable":     inboxrclpucl.ErrActionNotAvailable,
		"ErrPegaServiceUnavailable": inboxrclpucl.ErrPegaServiceUnavailable,
		"ErrTechnicalPICUnknown":    inboxrclpucl.ErrTechnicalPICUnknown,
		"ErrAlreadyWithAnalyst":     inboxrclpucl.ErrAlreadyWithAnalyst,
	}

	for nama, err := range galat {
		t.Run(nama, func(t *testing.T) {
			var terkirim inboxrclpuclhttp.ErrorResponse
			tulis := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
				w.WriteHeader(status)
				if resp, ok := body.(inboxrclpuclhttp.ErrorResponse); ok {
					terkirim = resp
				}
			}

			// Cadangannya sengaja NIL: yang diuji adalah pemetaan modul ini sendiri. Dengan
			// cadangan terpasang, galat yang tidak dikenali akan tampak tertangani padahal
			// yang menanganinya penulis galat portal — dan jawabannya tetap 500 generik.
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/inbox-rcl-pucl/x", nil)
			inboxrclpuclhttp.WriteError(nil, tulis, nil)(rec, req, err)

			require.NotEqual(t, http.StatusInternalServerError, rec.Code,
				"%s dijawab 500 generik; tambahkan pemetaannya di mapError", nama)
			require.NotEqual(t, inboxrclpuclhttp.CodeInternalError, terkirim.Code,
				"%s dijawab dengan kode galat internal; tambahkan pemetaannya di mapError",
				nama)
			require.NotEmpty(t, terkirim.Message,
				"%s dijawab tanpa kalimat apa pun untuk petugas", nama)
		})
	}
}
