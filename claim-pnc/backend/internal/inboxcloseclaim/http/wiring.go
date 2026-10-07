package inboxcloseclaimhttp

import (
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pengguna yang sedang masuk, sejauh yang dibutuhkan modul ini.
//
// Dua field, bukan satu seperti modul yang hanya membaca: jejak permintaan menyimpan NAMA
// pemohon bersama login-nya, supaya jejak itu tetap terbaca utuh tanpa join ke tabel
// pengguna — jejak yang namanya diambil lewat join akan berubah ketika orangnya berganti
// nama.
type Caller struct {
	Login string
	Name  string
}

// Handler melayani permintaan Inbox Close Claim.
type Handler struct {
	httpkit.Timed[Service, GetCaller]
}

// Options adalah bahan pembentuk Handler.
//
// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
//
// Ia parameter, bukan konstanta, supaya uji dapat menetapkannya dan tidak bergantung
// pada basis data zona waktu mesin yang menjalankan.
// Now dapat diisi uji supaya kolom Lama Waktu Klaim dapat diperiksa secara
// deterministik.
type Options = httpkit.TimedOptions[Service, GetCaller]

// NewHandler membentuk handler modul Inbox Close Claim.
func NewHandler(o Options) *Handler { return &Handler{Timed: httpkit.NewTimed(o, WriteError)} }
