package inboxxolhttp

import (
	"claim-pnc/internal/inboxxol/usecase"
	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/platform/httpkit"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
//
// Login adalah nama pengguna yang DIKETIK saat masuk — `OperatorID.pyUserIdentifier`
// di sistem lama, bukan NIK.
type Caller = httpkit.Caller

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan modul Inbox XOL.
type Handler struct {
	httpkit.Inbox[*usecase.Service, CallerReader]
}

// Options adalah bahan pembentuk Handler.
//
// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan
// diimpor dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa
// menariknya serta.
// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
// galat portal.
type Options = httpkit.InboxOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul Inbox XOL.
func NewHandler(o Options) *Handler { return &Handler{Inbox: httpkit.NewInbox(o, WriteError)} }
