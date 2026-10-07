package inboxinvestigatorhttp

import (
	"claim-pnc/internal/inboxinvestigator/usecase"
	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/platform/httpkit"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter = apierror.JSONWriter

// Handler melayani permintaan Inbox Investigator.
//
// TANPA CallerReader, dan itu bukan kelalaian melainkan akibat langsung dari bentuk layarnya:
// yang ditampilkan adalah isi **workbasket**, yaitu antrean BERSAMA yang belum bertuan
// (`D-26`). Tidak ada satu baris pun yang diturunkan dari identitas pemanggil, dan tidak ada
// perubahan yang perlu dicatat pelakunya.
//
// Itu berubah begitu layar kerjanya dibangun: mengambil pekerjaan dari antrean menuntut
// identitas pengambilnya, dan modul itulah yang akan membutuhkannya — bukan modul ini.
//
// Kewenangan membuka layar ini tetap ditegakkan — lewat middleware Autentikasi yang dipasang
// cmd, dan kelak lewat pemeriksaan peran `TKT-F3-005` yang belum ada. Di sistem lama
// pembatasnya `When/IsInvestigator-When.xml`: access group `PncInvestigator` atau
// `Administrators`, dan bukan `ViewClaimPNC`.
type Handler struct {
	httpkit.Basic[*usecase.Service]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.BasicOptions[*usecase.Service]

// NewHandler membentuk handler modul Inbox Investigator.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("inboxinvestigator/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Basic: base}, nil
}
