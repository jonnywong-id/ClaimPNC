package inboxcompliancehttp

import (
	"claim-pnc/internal/inboxcompliance/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
//
// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
type Caller = httpkit.Caller

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan modul Inbox Compliance.
type Handler struct {
	httpkit.Inbox[*usecase.Service, CallerReader]
}

// Options adalah bahan pembentuk Handler.
//
// # GetCaller hanya dipakai jalur TULIS
//
// Kedua jalur baca tidak membutuhkannya: antreannya WORKBASKET, yakni antrean bersama yang
// isinya sama bagi setiap petugas Compliance.
//
// Yang membutuhkannya adalah pengiriman ke Post Audit — bukan untuk menentukan apa yang
// boleh dikirim, melainkan untuk MENCATAT siapa yang mengirim. Tabelnya tidak punya kolom
// pengirim, sehingga log adalah satu-satunya tempat identitas itu tersimpan. Lihat catatan
// di usecase.Caller.
//
// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan
// diimpor dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa
// menariknya serta.
// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
// galat portal.
type Options = httpkit.InboxOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul Inbox Compliance.
func NewHandler(o Options) *Handler { return &Handler{Inbox: httpkit.NewInbox(o, WriteError)} }
