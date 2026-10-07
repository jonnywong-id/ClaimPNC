package inboxkomunikasicabanghttp

import (
	"context"

	"claim-pnc/internal/inboxkomunikasicabang/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Ia yang diterjemahkan menjadi kode cabang, dan karena itu MENENTUKAN apa yang
	// terlihat — bukan sekadar mengisi jejak log seperti di sebagian modul inbox lain.
	Login string

	// Name adalah nama pengguna yang terbaca manusia.
	//
	// Ia dibutuhkan SEJAK 2026-09-24, ketika modul ini mulai menulis: balasan menyimpannya
	// di `REPLYFROMNAME`, dan itulah yang digambar kolom "Penjawab(Dari)". Ia disimpan
	// bersama balasannya, bukan diambil lewat join saat dibaca — jejak yang namanya diambil
	// lewat join berubah ketika orangnya berganti nama, dan jejak yang dapat berubah bukan
	// jejak.
	//
	// Pada rute BACA ia tidak dipakai sama sekali, sehingga rute baca tetap dilayani meski
	// nama tidak terbaca. Yang menolak adalah NewReplyCommand, di lapisan domain.
	Name string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Komunikasi Cabang.
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

// NewHandler membentuk handler modul Inbox Komunikasi Cabang.
func NewHandler(o Options) *Handler { return &Handler{Inbox: httpkit.NewInbox(o, WriteError)} }
