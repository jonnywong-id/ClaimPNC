package inboxreceivetkahttp

import (
	"claim-pnc/internal/inboxreceivetka/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan Inbox Receive TKA.
//
// # TANPA CallerReader, dan ketiadaannya adalah utang yang disadari
//
// Modul ini MENULIS — ia mengubah tanggal pada data klaim — sehingga siapa pelakunya adalah
// keterangan yang seharusnya tercatat. Modul Jejak Audit (`S-5`) yang akan menyimpannya
// belum ada, dan daftar peristiwa wajib auditnya masih ditunggu dari Compliance
// (`ADR-0026`). Sampai itu tiba, pelakunya hanya tercatat di log aplikasi lewat middleware
// permintaan, bukan sebagai jejak audit yang tidak dapat diubah.
//
// Itu bukan keadaan yang boleh dibiarkan diam: `D-59` menetapkan satuan izin adalah MENU dan
// tidak ada pemisahan tugas, sehingga jejak audit adalah satu-satunya kontrol pengimbang
// yang tersisa. Dicatat di docs/keputusan-implementasi.md.
//
// Kewenangan membuka layar ini ditegakkan lewat middleware Autentikasi yang dipasang cmd,
// dan kelak lewat pemeriksaan peran `TKT-F3-005` yang belum ada. Sistem lama tidak memberi
// petunjuk apa pun tentang peran mana yang berhak: tidak ada When rule yang menjaga
// `MENU_ID 49`, dan `pyPrivilegeName` terisi pada 1 dari 902 activity.
type Handler struct {
	httpkit.Basic[*usecase.Service]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.BasicOptions[*usecase.Service]

// NewHandler membentuk handler modul Inbox Receive TKA.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("inboxreceivetka/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Basic: base}, nil
}
