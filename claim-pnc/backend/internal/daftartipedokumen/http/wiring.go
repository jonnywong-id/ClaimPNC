package daftartipedokumenhttp

import (
	"context"
	"net/http"

	"claim-pnc/internal/daftartipedokumen"
	"claim-pnc/internal/daftartipedokumen/usecase"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas orang yang mengirim permintaan.
//
// Ia sengaja tipe milik modul ini, bukan tipe milik modul auth: modul tidak saling
// mengimpor, dan yang dibutuhkan di sini hanyalah satu field. Cara mengisinya diberikan
// saat perakitan di cmd/claimpnc lewat Options.Caller, sehingga modul ini tidak pernah
// tahu bagaimana sesi bekerja.
//
// Isinya mengisi kolom USER_EDIT, meniru `OperatorID.pyUserIdentifier` di Pega.
type Caller struct {
	Identity string
}

// Handler melayani permintaan Daftar Tipe Dokumen.
type Handler struct {
	httpkit.Master[*usecase.Service, func(context.Context) (Caller, bool)]
	*crudhttp.Handler[SaveRequest]
}

// Options adalah bahan pembentuk Handler.
//
// Caller membaca identitas pemanggil dari context. Diisi saat perakitan di
// cmd/claimpnc; ia jembatan satu arah dari modul auth yang tidak membuat kedua modul
// saling mengimpor.
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul dapat
// dipindahkan tanpa menariknya serta.
type Options = httpkit.MasterOptions[*usecase.Service, func(context.Context) (Caller, bool)]

// NewHandler membentuk handler modul Daftar Tipe Dokumen.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewCrud("daftartipedokumen/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /api/master/tipe-dokumen.
	//
	// Menggantikan Report Definition `BrowseLstDocType_RD` yang mengisi grid layar
	// `ListDocumentTypeInbox`.
	//
	// Create menangani POST /api/master/tipe-dokumen.
	//
	// Menggantikan tombol Tambah lalu Simpan pada harness, yang mengirim sentinel `"UnknownID"`
	// supaya procedure memilih cabang INSERT. Di sini ID tidak pernah ikut di badan permintaan
	// sama sekali — sentinel itu tidak dibawa.
	//
	// Update menangani PUT /api/master/tipe-dokumen/{id}.
	//
	// PUT, bukan PATCH: seluruh isi yang boleh diubah dikirim setiap kali, sehingga
	// permintaannya menggantikan dan idempoten. Mengirim permintaan yang sama dua kali
	// menghasilkan keadaan akhir yang sama.
	h.Handler = crudhttp.New(crudhttp.Spec[SaveRequest]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         daftartipedokumen.ErrNotFound,
		Read:             h.readRequest,
		List: crudhttp.Load(h.Service.List, func(list []daftartipedokumen.DocumentType, alias string) any {
			content := toListDTO(list)
			return ListResponse{
				DocumentType: content,
				Total:        len(content),
				Portal:       alias,
			}
		}),
		Create: crudhttp.Save(func(r *http.Request, alias string, request SaveRequest) (daftartipedokumen.DocumentType, error) {
			return h.Service.Create(r.Context(), alias, daftartipedokumen.Input{
				Type:          request.Type,
				ProcessStatus: request.ProcessStatus,
			}, h.identity(r))
		}, func(saved daftartipedokumen.DocumentType, alias string) any {
			return SingleResponse{
				DocumentType: toDTO(saved),
				Portal:       alias,
			}
		}),
		Update: crudhttp.SaveOne(func(r *http.Request, alias, id string, request SaveRequest) (daftartipedokumen.DocumentType, error) {
			return h.Service.Update(r.Context(), alias, id, daftartipedokumen.Input{
				Type:          request.Type,
				ProcessStatus: request.ProcessStatus,
			}, h.identity(r))
		}, func(saved daftartipedokumen.DocumentType, alias string) any {
			return SingleResponse{
				DocumentType: toDTO(saved),
				Portal:       alias,
			}
		}),
	})
	return h, nil
}
