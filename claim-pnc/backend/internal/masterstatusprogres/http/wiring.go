package masterstatusprogreshttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/masterstatusprogres/usecase"
	"claim-pnc/internal/platform/crudhttp"
)

// Handler melayani permintaan master status progres.
type Handler struct {
	*crudhttp.Handler[SaveRequest]

	service       *usecase.Service
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service
	Logger  *slog.Logger

	// TulisRespon dan TulisGalat disuntikkan dari cmd, bukan diimpor dari modul auth.
	// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul
	// dapat dipindahkan tanpa menariknya serta.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul master status progres.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterstatusprogres/http: Layanan wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("masterstatusprogres/http: TulisRespon dan TulisGalat wajib diisi")
	}
	h := &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /master/status-progres-1.
	//
	// Create menangani POST /master/status-progres-1.
	//
	// Update menangani PUT /master/status-progres-1/{id}.
	h.Handler = crudhttp.New(crudhttp.Spec[SaveRequest]{
		WriteResponse:    h.writeResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         masterstatusprogres.ErrNotFound,
		Read:             h.readRequest,
		List: crudhttp.Load(h.service.List, func(list []masterstatusprogres.ProgressStatus, alias string) any {
			return ListResponse{
				ProgressStatus: toListDTO(list),
				Portal:         alias,
			}
		}),
		Create: crudhttp.Save(func(r *http.Request, alias string, request SaveRequest) (masterstatusprogres.ProgressStatus, error) {
			return h.service.Create(r.Context(), alias, masterstatusprogres.Input{
				Name:         request.Name,
				PositionCode: request.PositionCode,
			})
		}, func(saved masterstatusprogres.ProgressStatus, alias string) any {
			return SingleResponse{
				ProgressStatus: toDTO(saved),
				Portal:         alias,
			}
		}),
		Update: crudhttp.SaveOne(func(r *http.Request, alias, id string, request SaveRequest) (masterstatusprogres.ProgressStatus, error) {
			return h.service.Update(r.Context(), alias, id, masterstatusprogres.Input{
				Name:         request.Name,
				PositionCode: request.PositionCode,
			})
		}, func(saved masterstatusprogres.ProgressStatus, alias string) any {
			return SingleResponse{
				ProgressStatus: toDTO(saved),
				Portal:         alias,
			}
		}),
	})
	return h, nil
}
