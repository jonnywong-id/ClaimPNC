package inboxmanagerhttp

import (
	"context"

	"claim-pnc/internal/inboxmanager/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Setiap keputusan yang ditulis modul ini dicatat atas namanya, dan pada dua antrean ia
	// ikut tersimpan di kolom basis data. Tanpa login, tidak satu pun keputusan boleh
	// ditulis.
	Login string

	// OrgUnit adalah unit organisasi pengguna — padanan `OperatorID.pyOrgUnit`.
	//
	// Satu nilai punya arti khusus: `Development` membuka tab yang dibatasi lini bisnis.
	OrgUnit string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Manager.
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

// NewHandler membentuk handler modul Inbox Manager.
func NewHandler(o Options) *Handler { return &Handler{Inbox: httpkit.NewInbox(o, WriteError)} }
