package masterpenolakanhttp

import (
	"context"
	"errors"
	"log/slog"

	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterpenolakan/usecase"
	"claim-pnc/internal/platform/crudhttp"
)

// Caller adalah identitas orang yang mengirim permintaan.
//
// Ia sengaja tipe milik modul ini, bukan tipe milik modul auth: modul tidak saling
// mengimpor, dan yang dibutuhkan di sini hanyalah satu field. Cara mengisinya diberikan
// saat perakitan di cmd/claimpnc lewat Options.Caller, sehingga modul ini tidak pernah
// tahu bagaimana sesi bekerja.
//
// Yang dipakai adalah LOGIN, bukan NIK. Kolom USER_INPUT pada
// POOLDATA.MST_PENOLAKAN_KLAIM_2 diisi `OperatorID.pyUserIdentifier` di sistem lama
// (`Activity/InsertMasterPenolakanNoteKlaim-Act.xml`), yaitu login operator Pega —
// sehingga baris-baris lama sudah berisi login. Mengisinya dengan NIK akan membuat satu
// kolom memuat dua jenis pengenal yang tidak dapat dibedakan sesudahnya.
type Caller struct {
	Login string
}

// Handler melayani permintaan Master Penolakan Klaim — kedua tab sekaligus.
//
// SATU handler untuk dua master, karena keduanya satu layar dan satu butir menu
// (MENU_ID 25). Yang tidak disatukan adalah layanannya: masing-masing memilih penyimpanan
// yang berbeda, dan menyatukannya berarti satu layanan yang menerima dua pemilih repo
// lalu bercabang di setiap method.
type Handler struct {
	*crudhttp.Handler[struct{}]

	service       *usecase.Service
	komite        *usecase.ServiceKomite
	caller        func(context.Context) (Caller, bool)
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	// Service melayani tab Penolakan Klaim. Wajib.
	Service *usecase.Service

	// Komite melayani tab Penolakan Komite. Wajib.
	Komite *usecase.ServiceKomite

	// Caller membaca identitas pemanggil dari context. Wajib — tanpa itu kolom
	// USER_INPUT tidak dapat diisi, dan satu-satunya jejak pertanggungjawaban yang
	// dimiliki tabel ini hilang.
	Caller func(context.Context) (Caller, bool)

	Logger *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul
	// dapat dipindahkan tanpa menariknya serta.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Master Penolakan Klaim.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil || o.Komite == nil {
		return nil, errors.New("masterpenolakan/http: Service dan Komite wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("masterpenolakan/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("masterpenolakan/http: WriteResponse dan WriteError wajib diisi")
	}
	h := &Handler{
		service:       o.Service,
		komite:        o.Komite,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /master/penolakan-klaim.
	h.Handler = crudhttp.New(crudhttp.Spec[struct{}]{
		WriteResponse:    h.writeResponse,
		WriteModuleError: h.writeModuleError,
		List: crudhttp.Load(h.service.List, func(list []masterpenolakan.RejectionStatus2, alias string) any {
			return ListResponse{
				Rejection: toListDTO(list),
				Portal:    alias,
			}
		}),
	})
	return h, nil
}
