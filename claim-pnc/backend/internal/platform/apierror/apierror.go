// Package apierror memuat pola baku penulisan galat HTTP yang dipakai setiap modul.
//
// Setiap modul tetap memiliki pemetaan galatnya sendiri (mapError): galat domain mana menjadi
// status dan kode apa, dengan pesan yang dilihat pengguna. Yang tinggal di sini hanyalah
// langkah di sekelilingnya, yang dulu tersalin di setiap berkas errors.go:
//
//  1. petakan galat lewat pemetaan modul;
//  2. galat yang tidak dikenali diserahkan ke penulis cadangan;
//  3. status 5xx dicatat — rincian galat internal TIDAK pernah dikirim ke peramban, hanya
//     masuk log;
//  4. badan respons ditulis sebagai JSON.
package apierror

import (
	"log/slog"
	"net/http"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/platform/validation"
)

// JSONWriter menuliskan badan respons beserta statusnya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// Mapper memetakan galat ke status dan badan respons; known bernilai false bila galatnya
// bukan galat yang dikenal modul.
type Mapper[B any] func(err error) (status int, body B, known bool)

// Write menuliskan satu galat dengan pemetaan modul — bentuk method writeModuleError pada
// Handler. Galat yang tidak dikenali diteruskan apa adanya ke fallback; galat 5xx dicatat
// lewat logger yang membawa ID permintaan.
func Write[B any](w http.ResponseWriter, r *http.Request, err error, mapErr Mapper[B],
	logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) {
	status, body, known := mapErr(err)
	if !known {
		fallback(w, r, err)
		return
	}
	if status >= http.StatusInternalServerError {
		logFailure(logging.From(r.Context(), logger), r, err)
	}
	writeJSON(w, r, status, body)
}

// Writer membangun ErrorWriter dengan pemetaan modul — bentuk fungsi WriteError yang
// dipasang saat aplikasi dirakit.
//
// Galat yang tidak dikenali diserahkan ke fallback bila ada; bila tidak, dijawab 500 dengan
// badan internal milik modul. Galat 5xx dicatat bila logger tersedia.
func Writer[B any](logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter,
	mapErr Mapper[B], internal B) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body, known := mapErr(err)
		if !known {
			if fallback != nil {
				fallback(w, r, err)
				return
			}
			status, body = http.StatusInternalServerError, internal
		}
		if status >= http.StatusInternalServerError && logger != nil {
			logFailure(logger, r, err)
		}
		writeJSON(w, r, status, body)
	}
}

func logFailure(logger *slog.Logger, r *http.Request, err error) {
	logger.Error("permintaan gagal",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}

// ContextWriter sama dengan Writer, dengan satu beda: pencatatan galat 5xx memakai logger
// yang diperkaya konteks permintaan (logging.From), sehingga barisnya membawa id permintaan,
// dan dilakukan meski logger kosong.
func ContextWriter[B any](logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter,
	mapErr Mapper[B], internal B) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body, known := mapErr(err)
		if !known {
			if fallback != nil {
				fallback(w, r, err)
				return
			}
			status, body = http.StatusInternalServerError, internal
		}
		if status >= http.StatusInternalServerError {
			logFailure(logging.From(r.Context(), logger), r, err)
		}
		writeJSON(w, r, status, body)
	}
}

// Details menyalin seluruh pelanggaran validasi ke bentuk jawaban modul — SELURUHNYA
// sekaligus, bukan yang pertama saja (P-5).
func Details[D any](list []validation.Violation, build func(field, message string) D) []D {
	out := make([]D, 0, len(list))
	for _, v := range list {
		out = append(out, build(v.Field, v.Message))
	}
	return out
}
