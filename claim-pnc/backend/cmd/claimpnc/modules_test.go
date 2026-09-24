package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/logging"

	portalhttp "claim-pnc/internal/portal/http"
)

// Sepuluh modul yang dirakit modules.go BENAR-BENAR terpasang rutenya.
//
// # Kenapa uji ini ada
//
// Kesepuluh modul itu pernah hidup berbulan-bulan dengan usecase, penyimpanan SQL,
// penyimpanan memori, dan ujinya sendiri LENGKAP dan LULUS — tetapi tidak satu pun rutenya
// terdaftar, karena perakitannya hilang pada penggabungan cabang. Setiap uji modul lulus,
// `go build` lulus, `go vet` lulus, dan layarnya tetap menjawab 404.
//
// Uji modul tidak dapat menangkap itu: ia menguji modulnya, bukan apakah modulnya dipakai.
// Uji inilah yang menutup celah tersebut — ia menembak router yang benar-benar dirakit
// aplikasi, dan gagal begitu sebuah modul lepas dari perakitannya lagi.
//
// Jalur yang diperiksa diambil dari apa yang BENAR-BENAR dipanggil frontend, bukan dari
// daftar yang ditulis ulang di sini — itulah yang membuatnya berarti.
func TestExtraModulesMounted(t *testing.T) {
	assembly, err := build(devConfig(), logging.New(0))
	require.NoError(t, err)
	t.Cleanup(assembly.close)

	router := chi.NewRouter()
	err = mountExtra(
		router,
		assembly.extra,
		portalhttp.ActivePortalDeps{
			Repo:         assembly.portal,
			ReadyAliases: assembly.readyAliases,
			Logger:       logging.New(0),
			WriteError:   func(http.ResponseWriter, *http.Request, error) {},
		},
		func(http.ResponseWriter, *http.Request, int, any) {},
		func(http.ResponseWriter, *http.Request, error) {},
		logging.New(0),
	)
	require.NoError(t, err)

	registered := map[string]bool{}
	require.NoError(t, chi.Walk(router, func(
		method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler,
	) error {
		// chi menuliskan rute yang berakhir pada "/" untuk kelompok; ujungnya dipangkas
		// supaya pencocokannya tidak bergantung pada bentuk itu.
		registered[method+" "+strings.TrimSuffix(route, "/")] = true
		return nil
	}))

	// Satu jalur per modul, dipilih yang PALING DIPAKAI layarnya — daftar utama. Bila
	// modulnya lepas dari perakitan, jalur inilah yang pertama hilang.
	for _, want := range []string{
		"GET /inbox-admin",
		"GET /inbox-compliance",
		"GET /master/auto-claim",
		"GET /master/bengkel",
		"GET /master/panel",
		"GET /master/pasal-kerugian",
		"GET /master/penolakan-klaim",
		"GET /master/sparepart",
		"GET /master/supplier",
		// Laporan Klaim TIDAK disebut di sini: modulnya dinamai ulang menjadi
		// inboxlaporanklaim dan perakitannya pindah ke main.go, sehingga rutenya tidak
		// lagi lewat mountExtra.
		"GET /riwayat-klaim",
		"POST /registrasi/klaim",
	} {
		require.True(t, registered[want],
			"rute %q tidak terdaftar — modulnya lepas dari perakitan di modules.go", want)
	}
}

// Pemilih penyimpanan kesepuluh modul menolak portal selain portal utama.
//
// Tanpa basis data, godaannya adalah melayani alias apa pun dari satu penyimpanan. Itu
// membuat berpindah entitas TAMPAK berhasil padahal datanya itu-itu juga — dan justru
// menyembunyikan kelas kesalahan yang `R-20` peringatkan, tepat di lingkungan tempat
// pelanggarannya paling mudah lolos.
func TestExtraSelectorsRejectOtherPortals(t *testing.T) {
	var store storage
	setExtraMemorySelectors("ASM", &store)

	for _, alias := range []string{"ASI", "SMAS", "TIDAKADA", ""} {
		_, err := store.extra.inboxAdmin(alias)
		require.Error(t, err, "inbox admin tidak boleh melayani portal %q", alias)

		_, err = store.extra.autoClaim(alias)
		require.Error(t, err, "master auto claim tidak boleh melayani portal %q", alias)

		_, err = store.extra.workshop(alias)
		require.Error(t, err, "master bengkel tidak boleh melayani portal %q", alias)

		_, err = store.extra.panel(alias)
		require.Error(t, err, "master panel tidak boleh melayani portal %q", alias)

		_, err = store.extra.clause(alias)
		require.Error(t, err, "master pasal kerugian tidak boleh melayani portal %q", alias)

		_, err = store.extra.rejection(alias)
		require.Error(t, err, "master penolakan klaim tidak boleh melayani portal %q", alias)

		_, err = store.extra.rejectionKomite(alias)
		require.Error(t, err, "master penolakan komite tidak boleh melayani portal %q", alias)

		_, err = store.extra.sparepart(alias)
		require.Error(t, err, "master sparepart tidak boleh melayani portal %q", alias)

		_, err = store.extra.supplier(alias)
		require.Error(t, err, "master supplier tidak boleh melayani portal %q", alias)

		_, err = store.extra.claimHistory(alias)
		require.Error(t, err, "riwayat klaim tidak boleh melayani portal %q", alias)

		_, err = store.extra.protection(alias)
		require.Error(t, err, "proteksi data tidak boleh melayani portal %q", alias)
	}

	// Portal utama tetap dilayani — penolakan di atas tidak boleh menutup semuanya.
	_, err := store.extra.inboxAdmin("asm")
	require.NoError(t, err, "portal utama harus dilayani, termasuk bila hurufnya kecil")
}

// Penyimpanan di memori dipakai KEMBALI antarpermintaan.
//
// Bila dibuat ulang setiap kali dipilih, baris yang baru disimpan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab yang terlihat.
func TestExtraMemoryStoresReused(t *testing.T) {
	var store storage
	setExtraMemorySelectors("ASM", &store)

	first, err := store.extra.supplier("ASM")
	require.NoError(t, err)
	second, err := store.extra.supplier("ASM")
	require.NoError(t, err)
	require.Same(t, first, second, "penyimpanan master supplier dibuat ulang setiap dipilih")
}
