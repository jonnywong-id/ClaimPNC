package inboxosclaimpercabanghttp

import (
	"context"

	"claim-pnc/internal/inboxosclaimpercabang/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk.
	Login string

	// DetailBranchCode adalah kode cabang RINCI dari jawaban API HCQ saat masuk
	// (`EmpResponse.Placement.DetailBranchCode`), yang sudah dipetakan modul auth ke profil
	// pengguna.
	//
	// Ia BUKAN kode cabang yang dipakai klaim, dan itu sengaja dibawa apa adanya sampai ke
	// penyimpanan: terjemahannya satu pembacaan master cabang, dan menaruh terjemahan itu di
	// sini berarti lapisan transport memutuskan hal yang bukan urusannya. Lihat
	// inboxosclaimpercabang.Repo.BranchOf.
	//
	// Boleh kosong: `POOLDATA.M_LOGIN_PNC` tidak punya satu pun kolom cabang, sehingga pengguna
	// non-karyawan tidak membawanya. Yang terjadi kemudian bukan daftar kosong melainkan
	// pesan — lihat inboxosclaimpercabang.ErrBranchUnknown.
	DetailBranchCode string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox OS Claim per Cabang.
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

// NewHandler membentuk handler modul Inbox OS Claim per Cabang.
func NewHandler(o Options) *Handler { return &Handler{Inbox: httpkit.NewInbox(o, WriteError)} }
