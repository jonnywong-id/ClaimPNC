package inboxpladlahttp_test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	inboxpladlahttp "claim-pnc/internal/inboxpladla/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// TestSeluruhRuteTerdaftar menjaga daftar rute modul ini.
//
// # Kenapa uji daftar rute, padahal rutenya ditulis satu baris
//
// Karena rute yang HILANG tidak menghasilkan satu pun galat saat build. Ia menghasilkan
// "Alamat API tidak dikenal" pada layar — kalimat yang terbaca persis seperti salah ketik
// di sisi frontend, sehingga yang dicari orang adalah tempat yang keliru.
//
// Presedennya ada di `inboxrclpucl`, dan sebabnya di sana ternyata bukan rutenya sama
// sekali: rutenya terdaftar dengan benar, yang berjalan adalah binary lama. Hal yang sama
// terulang di modul ini pada 2026-10-09 dengan rute `ringkas-daftar`.
//
// Uji ini TIDAK dapat menangkap binary lama — tidak ada uji yang bisa. Yang ia lakukan
// adalah mempersempit penyebabnya menjadi satu kemungkinan: bila uji ini hijau sementara
// layar menjawab "Alamat API tidak dikenal", maka yang berjalan bukan kode ini.
func TestSeluruhRuteTerdaftar(t *testing.T) {
	r := chi.NewRouter()
	inboxpladlahttp.Mount(r, &inboxpladlahttp.Handler{}, portalhttp.ActivePortalDeps{})

	terpasang := map[string]bool{}
	require.NoError(t, chi.Walk(r, func(
		metode, rute string, _ http.Handler, _ ...func(http.Handler) http.Handler,
	) error {
		terpasang[metode+" "+rute] = true
		return nil
	}))

	for _, wajib := range []string{
		"GET /inbox-pla-dla",
		"GET /inbox-pla-dla/daftar",

		// Tabel ringkas per STATUS KLAIM di dalam satu daftar. Tidak digambar layar sejak
		// 2026-10-09, tetapi rutenya dipertahankan — lihat catatan §156.5.
		"GET /inbox-pla-dla/ringkas",

		// Tabel "Status / Jumlah" per DAFTAR — navigasi layar ini, dan satu-satunya yang
		// digambar Pega. Rute inilah yang 404 pada 2026-10-09.
		"GET /inbox-pla-dla/ringkas-daftar",

		"GET /inbox-pla-dla/ekspor",

		// Layar rincian. Segmen `klaim` ada supaya rute tetap di atasnya tidak tertangkap
		// sebagai kunci klaim.
		"GET /inbox-pla-dla/klaim/{kunci}",
		"GET /inbox-pla-dla/klaim/{kunci}/dokumen",
		"GET /inbox-pla-dla/klaim/{kunci}/dokumen/{dokumen}",
	} {
		require.Truef(t, terpasang[wajib], "rute %q tidak terdaftar", wajib)
	}
}

// TestRuteTetapTidakTertangkapSebagaiKunciKlaim menjaga segmen `klaim` tetap ada.
//
// Tanpa segmen itu, `/inbox-pla-dla/{kunci}` akan menangkap `daftar`, `ringkas`,
// `ringkas-daftar`, dan `ekspor` sebagai nomor klaim pada urutan pendaftaran
// tertentu — kelas kerusakan yang baru muncul ketika rutenya bertambah, dan yang paling
// mudah terlewat karena jawabannya 200, bukan 404.
func TestRuteTetapTidakTertangkapSebagaiKunciKlaim(t *testing.T) {
	r := chi.NewRouter()
	inboxpladlahttp.Mount(r, &inboxpladlahttp.Handler{}, portalhttp.ActivePortalDeps{})

	require.NoError(t, chi.Walk(r, func(
		_, rute string, _ http.Handler, _ ...func(http.Handler) http.Handler,
	) error {
		require.NotEqual(t, "/inbox-pla-dla/{kunci}", rute,
			"kunci klaim wajib berada di bawah segmen `klaim`")
		return nil
	}))
}
