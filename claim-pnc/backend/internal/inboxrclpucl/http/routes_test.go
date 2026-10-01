package inboxrclpuclhttp

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	portalhttp "claim-pnc/internal/portal/http"
)

// TestSeluruhRuteTerdaftar menjaga daftar rute modul ini.
//
// # Kenapa uji daftar rute, padahal rutenya ditulis satu baris
//
// Karena rute yang HILANG tidak menghasilkan satu pun galat saat build. Ia menghasilkan
// "Alamat API tidak dikenal" pada layar — dan itu terbaca persis seperti alamat yang salah
// ketik di sisi frontend, sehingga yang dicari orang adalah tempat yang keliru.
//
// Itu benar-benar terjadi 2026-10-01: rute `kirim-analyst` terdaftar dengan benar, tetapi yang
// berjalan adalah binary lama. Uji ini tidak dapat menangkap binary lama — tidak ada uji yang
// bisa — tetapi ia MEMPERSEMPIT penyebabnya menjadi satu kemungkinan, dan itu yang mahal saat
// sedang mencari.
func TestSeluruhRuteTerdaftar(t *testing.T) {
	r := chi.NewRouter()
	Mount(r, &Handler{}, portalhttp.ActivePortalDeps{})

	terpasang := map[string]bool{}
	require.NoError(t, chi.Walk(r, func(
		metode, rute string, _ http.Handler, _ ...func(http.Handler) http.Handler,
	) error {
		terpasang[metode+" "+rute] = true
		return nil
	}))

	for _, wajib := range []string{
		"GET /inbox-rcl-pucl",
		"GET /inbox-rcl-pucl/tab",
		"GET /inbox-rcl-pucl/ekspor",
		"GET /inbox-rcl-pucl/klaim/{referensi}",
		"GET /inbox-rcl-pucl/klaim/{referensi}/dokumen",
		"GET /inbox-rcl-pucl/klaim/{referensi}/dokumen/{dokumen}",

		// Satu-satunya rute yang MENGUBAH klaim. POST, bukan GET.
		"POST /inbox-rcl-pucl/klaim/{referensi}/tindakan/{aksi}",
	} {
		require.Truef(t, terpasang[wajib], "rute %q tidak terdaftar", wajib)
	}
}

// TestTindakanTulisBukanGET menjaga metode rute yang menimbulkan akibat.
//
// GET boleh diulang peramban, prefetch, dan perkakas pemantau. Rute yang meneruskan klaim
// tidak boleh dapat dipicu begitu.
func TestTindakanTulisBukanGET(t *testing.T) {
	r := chi.NewRouter()
	Mount(r, &Handler{}, portalhttp.ActivePortalDeps{})

	require.NoError(t, chi.Walk(r, func(
		metode, rute string, _ http.Handler, _ ...func(http.Handler) http.Handler,
	) error {
		if rute == "/inbox-rcl-pucl/klaim/{referensi}/tindakan/{aksi}" {
			require.Equal(t, "POST", metode)
		}
		return nil
	}))
}
