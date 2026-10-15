package masterbengkelhttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Rute pembuka dokumen — `GET /master/bengkel/{id}/dokumen/tautan`.
//
// Yang diuji di sini adalah PEMASANGANNYA, bukan penerbitan alamatnya. Keduanya kelas
// cacat yang berbeda, dan yang satu tidak tertangkap oleh uji yang lain: jalur yang salah
// pasang tetap lulus seluruh uji usecase, sebab usecase-nya memang tidak pernah dipanggil.
//
// Pemasangannya rawan karena rutenya BERSARANG di bawah rute yang sudah ada
// (`.../dokumen`), dan rute yang tertelan tetangganya menjawab 404 tanpa satu pun tanda
// bahwa sebabnya adalah pendaftaran, bukan datanya.

// firstApprovedID mengambil satu ID bengkel yang benar-benar ada di contoh.
func firstApprovedID(t *testing.T, p *testServer) string {
	t.Helper()

	response, content := p.call(t, http.MethodGet, "/api/master/bengkel?status=1", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	list := rows(t, content)
	require.NotEmpty(t, list, "contoh harus punya sekurangnya satu bengkel disetujui")

	first, ok := list[0].(map[string]any)
	require.True(t, ok)
	id, ok := first["id_bengkel"].(string)
	require.True(t, ok)
	require.NotEmpty(t, id)
	return id
}

// Bengkel yang belum pernah dilampiri dijawab "dokumen tidak ditemukan" — dan justru
// itulah buktinya rutenya sampai ke handler yang benar.
//
// Rute yang TIDAK terpasang juga menjawab 404, sehingga yang membedakan keduanya adalah
// KODE di badan respons, bukan status HTTP-nya.
func TestDocumentLinkRouteIsMounted(t *testing.T) {
	p := newTestServer(t)
	id := firstApprovedID(t, p)

	response, content := p.call(t, http.MethodGet,
		"/api/master/bengkel/"+id+"/dokumen/tautan", "ASM", "")

	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "dokumen_tidak_ditemukan", content["kode"],
		"404 tanpa kode modul berarti rutenya tidak terpasang, bukan dokumennya yang tidak ada")
}

// Rutenya menuntut portal seperti seluruh rute modul ini.
//
// Tanpa portal, permintaan tidak boleh jatuh ke koneksi bawaan — itu persis kebocoran
// antar badan hukum yang `R-20` cegah.
func TestDocumentLinkRouteRequiresPortal(t *testing.T) {
	p := newTestServer(t)
	id := firstApprovedID(t, p)

	response, _ := p.call(t, http.MethodGet,
		"/api/master/bengkel/"+id+"/dokumen/tautan", "", "")

	require.NotEqual(t, http.StatusOK, response.StatusCode)
	require.NotEqual(t, http.StatusNotFound, response.StatusCode,
		"penolakan portal harus terbaca sebagai penolakan portal, bukan sebagai rute yang tidak ada")
}

// Bengkel yang tidak ada dibedakan dari bengkel yang belum punya dokumen.
func TestDocumentLinkRouteRejectsUnknownWorkshop(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet,
		"/api/master/bengkel/TIDAK-ADA-9999/dokumen/tautan", "ASM", "")

	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}
