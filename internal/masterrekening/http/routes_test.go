package masterrekeninghttp_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterrekening/cashier"
	"claim-pnc/internal/masterrekening/notification"
	"claim-pnc/internal/masterrekening/repo/memory"
	"claim-pnc/internal/masterrekening/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"

	masterrekeninghttp "claim-pnc/internal/masterrekening/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// Berkas ini lahir bersama penyelarasan lingkup portal 2026-09-19.
//
// Sebelum itu modul Master Rekening selalu membaca dan menulis basis data portal UTAMA —
// artinya rekening pembayaran SELURUH badan hukum berada di satu tempat. Rutenya kini
// dijaga middleware portal, dan yang diuji di sini adalah penjagaan itu: bukan bahwa
// handler-nya bekerja, melainkan bahwa ia MENOLAK melayani permintaan yang tidak menyebut
// entitasnya.
//
// Kegagalan yang dicegah tidak terlihat sebagai galat. Layar tampil normal, nomor
// rekeningnya masuk akal, dan yang salah hanya milik siapa data itu (`R-20`).

const routeAccount = "/api/master-rekening"

// entities memegang layanan tiap entitas supaya uji dapat memeriksa bahwa yang tersentuh
// memang entitas yang diminta.
type entities struct {
	handler http.Handler
	asm     *usecase.Service
	asi     *usecase.Service
}

func testServer(t *testing.T) *entities {
	t.Helper()

	build := func(alias string) *usecase.Service {
		return usecase.NewService(usecase.Options{
			Repo:             memory.NewRepo(),
			Bank:             memory.NewBankRepo(memory.SampleBanks()...),
			Cashier:          cashier.NewFake(),
			Notifier:         &notification.Fake{},
			Clock:            clock.FixedAt(time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC)),
			PortalAlias:      alias,
			DefaultCommittee: "KOMITE-01",
		})
	}
	asm, asi := build("ASM"), build("ASI")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}
	// Dirantai sama seperti cmd/claimpnc. Tanpa rantai ini, uji penolakan portal lulus di
	// sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(
		masterrekeninghttp.WriteError(logger),
		writeJSON,
	)

	handler := masterrekeninghttp.NewHandler(masterrekeninghttp.Options{
		ServiceSelector: func(alias string) (masterrekeninghttp.Service, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Caller: func(context.Context) (masterrekeninghttp.Caller, bool) {
			return masterrekeninghttp.Caller{
				Identity: "3171999", Name: "Petugas Contoh", Email: "petugas@example.invalid",
			}, true
		},
		Logger:     logger,
		WriteError: writeError,
	})

	// Hanya ASM dan ASI yang koneksinya "hidup"; SMAS ada di daftar tetapi belum siap.
	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		masterrekeninghttp.Mount(api, handler, portalDeps)
	})
	return &entities{handler: router, asm: asm, asi: asi}
}

// call menembak satu permintaan. Alias kosong berarti header portal tidak dikirim.
func call(t *testing.T, server *entities, method, path, portalAlias, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	rekaman := httptest.NewRecorder()
	server.handler.ServeHTTP(rekaman, request)

	content := map[string]any{}
	_ = json.Unmarshal(rekaman.Body.Bytes(), &content)
	return rekaman, content
}

// contohPengajuan adalah satu pengajuan lengkap yang lolos validasi.
const contohPengajuan = `{
	"nomor_rekening":"1234567890",
	"nama_pemilik":"BENGKEL CONTOH SEJAHTERA",
	"nama_bank":"BANK CONTOH",
	"cabang_bank":"JAKARTA PUSAT",
	"alamat_bank":"JL. CONTOH NO. 1",
	"kode_bank":"014",
	"tipe_rekening":"BIASA",
	"email":"keuangan@contoh.co.id",
	"telepon":"0211234567",
	"nik":"3171000000000000",
	"id_dokumen":"DOK-001",
	"aktif":true
}`

// PERMINTAAN TANPA PORTAL DITOLAK pada SETIAP rute, bukan dilayani portal utama.
//
// Ini uji terpenting di berkas ini. Master Rekening memuat nomor rekening, NIK, dan surel
// pihak ketiga; melayaninya dengan entitas yang salah bukan sekadar menampilkan angka yang
// keliru, melainkan membocorkan data nasabah satu badan hukum ke badan hukum lain.
func TestWithoutPortalRejected(t *testing.T) {
	server := testServer(t)

	for _, perkara := range []struct{ nama, metode, jalur, badan string }{
		{"daftar", http.MethodGet, routeAccount, ""},
		{"daftar bank", http.MethodGet, routeAccount + "/bank", ""},
		{"ambil satu", http.MethodGet, routeAccount + "/014/1234567890", ""},
		{"mengajukan", http.MethodPost, routeAccount, contohPengajuan},
		{"mengubah", http.MethodPut, routeAccount + "/014/1234567890", contohPengajuan},
		{"memutuskan", http.MethodPost, routeAccount + "/014/1234567890/keputusan", `{"status":"1","catatan":"ok"}`},
	} {
		t.Run(perkara.nama, func(t *testing.T) {
			rekaman, isi := call(t, server, perkara.metode, perkara.jalur, "", perkara.badan)
			require.Equal(t, http.StatusBadRequest, rekaman.Code)
			require.Equal(t, portalhttp.CodeNotStated, isi["kode"])
		})
	}

	// Dan tidak ada satu baris pun yang tersimpan di entitas mana pun.
	for nama, service := range map[string]*usecase.Service{"ASM": server.asm, "ASI": server.asi} {
		_, total, err := service.List(t.Context(), masterrekening.Filter{})
		require.NoError(t, err)
		require.Zerof(t, total, "entitas %s tidak boleh tersentuh", nama)
	}
}

// "Tidak ada" dibedakan dari "belum tersedia": keduanya menuntut tindak lanjut berbeda.
func TestUnknownAndNotReadyPortalAreDistinguished(t *testing.T) {
	server := testServer(t)

	rekaman, isi := call(t, server, http.MethodGet, routeAccount, "TIDAKADA", "")
	require.Equal(t, http.StatusBadRequest, rekaman.Code)
	require.Equal(t, portalhttp.CodeUnknown, isi["kode"])

	rekaman, isi = call(t, server, http.MethodGet, routeAccount, "SMAS", "")
	require.Equal(t, http.StatusServiceUnavailable, rekaman.Code)
	require.Equal(t, portalhttp.CodeNotReady, isi["kode"])
}

// Rekening yang diajukan di satu entitas TIDAK terbaca di entitas lain.
//
// Inilah cacat yang diperbaiki penyelarasan ini: sebelumnya keduanya menulis ke tempat
// yang sama, sehingga rekening ASI akan muncul di layar ASM tanpa satu pun tanda.
func TestAccountSubmittedInOnePortalNotVisibleInAnother(t *testing.T) {
	server := testServer(t)

	rekaman, _ := call(t, server, http.MethodPost, routeAccount, "ASI", contohPengajuan)
	require.Equal(t, http.StatusCreated, rekaman.Code)

	rekaman, isiASI := call(t, server, http.MethodGet, routeAccount, "ASI", "")
	require.Equal(t, http.StatusOK, rekaman.Code)
	require.Equal(t, float64(1), isiASI["jumlah"])

	rekaman, isiASM := call(t, server, http.MethodGet, routeAccount, "ASM", "")
	require.Equal(t, http.StatusOK, rekaman.Code)
	require.Equal(t, float64(0), isiASM["jumlah"],
		"rekening entitas lain TIDAK boleh terbaca di sini")
}

// Nomor rekening yang sama boleh ada di dua entitas: keduanya badan hukum berbeda, dan
// keunikan nomor rekening hanya berlaku di dalam satu entitas.
func TestSameAccountNumberAllowedInDifferentPortals(t *testing.T) {
	server := testServer(t)

	rekaman, _ := call(t, server, http.MethodPost, routeAccount, "ASM", contohPengajuan)
	require.Equal(t, http.StatusCreated, rekaman.Code)

	rekaman, _ = call(t, server, http.MethodPost, routeAccount, "ASI", contohPengajuan)
	require.Equal(t, http.StatusCreated, rekaman.Code,
		"nomor yang sama di entitas lain bukan duplikat")

	// Tetapi mengulanginya di entitas yang SAMA tetap ditolak.
	rekaman, _ = call(t, server, http.MethodPost, routeAccount, "ASM", contohPengajuan)
	require.Equal(t, http.StatusConflict, rekaman.Code)
}
