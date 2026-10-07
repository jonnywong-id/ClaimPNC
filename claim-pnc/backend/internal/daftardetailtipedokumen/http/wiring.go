package daftardetailtipedokumenhttp

import (
	"context"
	"net/http"

	"claim-pnc/internal/daftardetailtipedokumen"
	"claim-pnc/internal/daftardetailtipedokumen/usecase"
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

// Handler melayani permintaan Daftar Detail Tipe Dokumen.
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
// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul
// dapat dipindahkan tanpa menariknya serta.
type Options = httpkit.MasterOptions[*usecase.Service, func(context.Context) (Caller, bool)]

// NewHandler membentuk handler modul Daftar Detail Tipe Dokumen.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewCrud("daftardetailtipedokumen/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /api/master/detail-tipe-dokumen.
	//
	// Menggantikan Report Definition `BrowseVLstDetTypeDoc_RD` yang mengisi grid layar
	// `ListDetTypeDocument-Harness`.
	//
	// Get menangani GET /api/master/detail-tipe-dokumen/{id}.
	//
	// Menggantikan `Activity/CNMSetDetailTypeDocument_act-Act.xml`, yang mengisi form dari
	// baris terpilih lalu memanggil `GetLbuDetType` untuk mengisi grid bisnisnya. Berbeda
	// dari List, jawaban ini MEMBAWA daftar bisnisnya — dan form memang membutuhkannya tepat
	// saat baris dibuka, bukan saat daftarnya ditampilkan.
	//
	// Create menangani POST /api/master/detail-tipe-dokumen.
	//
	// Menggantikan tombol Tambah lalu Simpan pada harness, yang memanggil
	// `CNMInsertDetailTypeDocument_act`.
	//
	// Update menangani PUT /api/master/detail-tipe-dokumen/{id}.
	//
	// PUT, bukan PATCH: seluruh isi yang boleh diubah dikirim setiap kali, sehingga
	// permintaannya menggantikan dan idempoten. Mengirim permintaan yang sama dua kali
	// menghasilkan keadaan akhir yang sama — termasuk untuk daftar bisnisnya, yang memang
	// diganti seluruhnya.
	h.Handler = crudhttp.New(crudhttp.Spec[SaveRequest]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         daftardetailtipedokumen.ErrNotFound,
		Read:             h.readRequest,
		List: crudhttp.Load(h.Service.List, func(list []daftardetailtipedokumen.DetailType, alias string) any {
			content := toListDTO(list)
			return ListResponse{
				Detail: content,
				Total:  len(content),
				Portal: alias,
			}
		}),
		Get: crudhttp.LoadOne(h.Service.Get, func(row daftardetailtipedokumen.DetailType, alias string) any {
			return SingleResponse{
				Detail: toDTO(row),
				Portal: alias,
			}
		}),
		Create: crudhttp.Save(func(r *http.Request, alias string, request SaveRequest) (daftardetailtipedokumen.DetailType, error) {
			return h.Service.Create(r.Context(), alias, toInput(request), h.identity(r))
		}, func(saved daftardetailtipedokumen.DetailType, alias string) any {
			return SingleResponse{
				Detail: toDTO(saved),
				Portal: alias,
			}
		}),
		Update: crudhttp.SaveOne(func(r *http.Request, alias, id string, request SaveRequest) (daftardetailtipedokumen.DetailType, error) {
			return h.Service.Update(r.Context(), alias, id, toInput(request), h.identity(r))
		}, func(saved daftardetailtipedokumen.DetailType, alias string) any {
			return SingleResponse{
				Detail: toDTO(saved),
				Portal: alias,
			}
		}),
	})
	return h, nil
}
