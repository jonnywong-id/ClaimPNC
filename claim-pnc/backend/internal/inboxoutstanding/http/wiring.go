package inboxoutstandinghttp

import (
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pengguna yang sedang masuk, sejauh yang dibutuhkan modul ini.
//
// Hanya satu field: modul ini tidak perlu tahu apa pun tentang bentuk sesi, dan modul auth
// tidak perlu tahu modul ini ada. Jembatannya dipasang di cmd/claimpnc, satu-satunya
// berkas yang memang tahu keduanya.
type Caller struct {
	Login string
}

// Handler melayani permintaan Inbox Outstanding.
type Handler struct {
	httpkit.Timed[Service, GetCaller]
}

// Options adalah bahan pembentuk Handler.
//
// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
//
// Ia parameter, bukan konstanta, supaya uji dapat menetapkannya dan tidak bergantung
// pada basis data zona waktu mesin yang menjalankan.
// Now dapat diisi uji supaya kolom umur klaim dapat diperiksa secara deterministik.
type Options = httpkit.TimedOptions[Service, GetCaller]

// NewHandler membentuk handler modul Inbox Outstanding.
func NewHandler(o Options) *Handler { return &Handler{Timed: httpkit.NewTimed(o, WriteError)} }
