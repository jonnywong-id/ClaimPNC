package inboxpladlahttp

import (
	"context"

	"claim-pnc/internal/inboxpladla/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: jembatan di antara keduanya dipasang
// cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Ia dicocokkan ke `POOLDATA.T_REINSURER.LOGIN`. Memakai NIK di sini akan membuat
	// layar kosong bagi SETIAP reasuradur — dan kosongnya tidak dapat dibedakan dari
	// "belum ada pekerjaan".
	Login string

	// Name adalah nama yang dibaca manusia.
	//
	// Ia tidak menyaring apa pun; satu-satunya pemakainya adalah kolom nama pembalas pada
	// balasan komunikasi. Boleh kosong.
	Name string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox PLA DLA.
type Handler struct {
	httpkit.Inbox[*usecase.Service, CallerReader]
}

// Options adalah bahan pembentuk Handler.
//
// FallbackErrorWriter menangani galat yang bukan milik modul ini.
type Options = httpkit.InboxOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul Inbox PLA DLA.
func NewHandler(o Options) *Handler { return &Handler{Inbox: httpkit.NewInbox(o, WriteError)} }
