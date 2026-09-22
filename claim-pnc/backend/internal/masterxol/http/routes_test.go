package masterxolhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/masterxol/notification"
	"claim-pnc/internal/masterxol/repo/memory"
	"claim-pnc/internal/masterxol/usecase"
	"claim-pnc/internal/portal"

	masterxolhttp "claim-pnc/internal/masterxol/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// portalASM adalah entitas yang dipakai hampir seluruh uji di berkas ini. Ia disebut
// eksplisit pada setiap permintaan — persis seperti yang dituntut aplikasi sungguhan.
const portalASM = "ASM"

const basePath = "/api/master/xol"

// entities memegang penyimpanan tiap entitas supaya uji dapat memeriksa bahwa yang
// tersentuh memang entitas yang diminta, bukan entitas lain.
type entities struct {
	handler  http.Handler
	asm      *memory.Repo
	asi      *memory.Repo
	notifier *notification.Fake
}

func testServer(t *testing.T) *entities {
	t.Helper()

	asm := memory.NewSampleRepo()
	// ASI sengaja KOSONG; ia yang membuktikan pemisahan antarentitas.
	asi := memory.NewRepo()
	notifier := notification.NewFake()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterxol.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Notifier: notifier,
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}
	// Dirantai sama seperti cmd/claimpnc: galat portal dipetakan modul portal. Tanpa
	// rantai ini, uji penolakan portal lulus di sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode": "galat_internal", "pesan": err.Error(),
			})
		},
		writeJSON,
	)

	handler, err := masterxolhttp.NewHandler(masterxolhttp.Options{
		Service: service,
		Caller: func(context.Context) (masterxolhttp.Caller, bool) {
			return masterxolhttp.Caller{Identity: "JONNY", Name: "Jonny"}, true
		},
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterxolhttp.ErrorWriter(writeError),
	})
	require.NoError(t, err)

	// Hanya ASM dan ASI yang koneksinya "hidup"; SMAS ada di daftar tetapi belum siap.
	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		masterxolhttp.Mount(api, handler, portalDeps)
	})
	return &entities{handler: router, asm: asm, asi: asi, notifier: notifier}
}

// call menembak dengan portal ASM. Uji yang menguji portal lain memakai callPortal.
func call(t *testing.T, server *entities, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return callPortal(t, server, method, path, portalASM, body)
}

// callPortal menembak dengan portal tertentu. Alias kosong berarti header tidak dikirim.
func callPortal(t *testing.T, server *entities, method, path, portalAlias string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var content io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		content = bytes.NewReader(raw)
	}
	request := httptest.NewRequest(method, path, content)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	record := httptest.NewRecorder()
	server.handler.ServeHTTP(record, request)
	return record
}

func decode[T any](t *testing.T, record *httptest.ResponseRecorder) T {
	t.Helper()
	var body T
	require.NoError(t, json.Unmarshal(record.Body.Bytes(), &body))
	return body
}

// ---------- portal ----------

func TestPermintaanTanpaPortalDitolakBukanDijawabPortalUtama(t *testing.T) {
	// Menjawabnya dengan portal utama sebagai cadangan berarti menampilkan struktur
	// treaty badan hukum yang tidak diminta, tanpa satu pun pesan galat (`R-20`).
	server := testServer(t)

	record := callPortal(t, server, http.MethodGet, basePath, "", nil)
	require.Equal(t, http.StatusBadRequest, record.Code)
}

func TestPortalYangBelumSiapDijawabDenganSebabnya(t *testing.T) {
	server := testServer(t)

	record := callPortal(t, server, http.MethodGet, basePath, "SMAS", nil)
	require.Equal(t, http.StatusServiceUnavailable, record.Code)
}

func TestSetiapJawabanMenyebutEntitasYangMenjawabnya(t *testing.T) {
	server := testServer(t)

	body := decode[map[string]any](t, call(t, server, http.MethodGet, basePath, nil))
	require.Equal(t, portalASM, body["portal"])
}

func TestEntitasLainPunyaDaftarnyaSendiri(t *testing.T) {
	server := testServer(t)

	asm := decode[map[string]any](t, callPortal(t, server, http.MethodGet, basePath, "ASM", nil))
	asi := decode[map[string]any](t, callPortal(t, server, http.MethodGet, basePath, "ASI", nil))

	require.Equal(t, float64(4), asm["total"])
	require.Equal(t, float64(0), asi["total"], "ASI kosong; isinya tidak boleh datang dari ASM")
}

// ---------- daftar dan detail ----------

func TestDaftarTidakMembawaAnaknya(t *testing.T) {
	server := testServer(t)

	body := decode[struct {
		XOL []struct {
			ID    string `json:"id"`
			Layer []any  `json:"layer"`
		} `json:"xol"`
	}](t, call(t, server, http.MethodGet, basePath, nil))

	require.Len(t, body.XOL, 4)
	for _, m := range body.XOL {
		require.Empty(t, m.Layer, m.ID)
	}
}

func TestDetailMembawaSeluruhTingkatBesertaLimitRupiahnya(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodGet, basePath+"/10001", nil)
	require.Equal(t, http.StatusOK, record.Code)

	body := decode[struct {
		XOL struct {
			TipeLabel string `json:"tipe_label"`
			Bisnis    []any  `json:"bisnis"`
			Layer     []struct {
				LimitIDR int64 `json:"limit_idr"`
				Reas     []any `json:"reas"`
			} `json:"layer"`
		} `json:"xol"`
	}](t, record)

	require.Equal(t, "Property / Motor / Engineering", body.XOL.TipeLabel)
	require.Len(t, body.XOL.Bisnis, 4)
	require.Len(t, body.XOL.Layer, 3)
	require.Len(t, body.XOL.Layer[0].Reas, 6)
	require.Equal(t, int64(14_782_500_000), body.XOL.Layer[0].LimitIDR)
}

func TestIndukTidakDikenalDijawab404(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodGet, basePath+"/99999", nil)
	require.Equal(t, http.StatusNotFound, record.Code)
	require.Equal(t, "xol_tidak_ditemukan", decode[map[string]any](t, record)["kode"])
}

// ---------- simpan ----------

func TestMenambahMenjawab201DanMengajukanKeKomite(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodPost, basePath, map[string]any{
		"nama": "Section 9", "tahun": "2026", "kurs": 16000, "tipe": "1",
		"remark_pic": "pengajuan baru",
		"layer": []map[string]any{{
			"nama": "Layer 1", "limit": 1000000, "excess": 500000,
			"reas": []map[string]any{{"id": "10036322", "nama": "SWISS RE", "share": 100}},
		}},
	})
	require.Equal(t, http.StatusCreated, record.Code)

	body := decode[struct {
		XOL struct {
			ID           string `json:"id"`
			PIC          string `json:"pic"`
			StatusKomite string `json:"status_komite"`
			Layer        []struct {
				ID       string `json:"id"`
				LimitIDR int64  `json:"limit_idr"`
			} `json:"layer"`
		} `json:"xol"`
		Peringatan []string `json:"peringatan"`
	}](t, record)

	require.Equal(t, "10009", body.XOL.ID)
	require.Equal(t, "JONNY", body.XOL.PIC, "PIC diisi dari sesi, bukan dari badan permintaan")
	require.Equal(t, "0", body.XOL.StatusKomite)
	require.Equal(t, int64(16_000_000_000), body.XOL.Layer[0].LimitIDR)
	require.Empty(t, body.Peringatan)

	require.Equal(t, 1, server.notifier.Count())
}

func TestMengubahMenjawab200(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodPut, basePath+"/10001", map[string]any{
		"nama": "Section 1", "tahun": "2018", "kurs": 13500, "tipe": "1",
		"remark_pic": "kurs dikoreksi",
	})
	require.Equal(t, http.StatusOK, record.Code)
}

func TestTotalShareBelum100TersimpanDenganPeringatanBukanDitolak(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodPost, basePath, map[string]any{
		"nama": "Section 9", "tahun": "2026", "kurs": 16000, "tipe": "1",
		"layer": []map[string]any{{
			"nama": "Layer 1",
			"reas": []map[string]any{{"id": "1", "nama": "SWISS RE", "share": 60}},
		}},
	})

	require.Equal(t, http.StatusCreated, record.Code, "layar lama pun menyimpan lebih dulu")

	body := decode[struct {
		Peringatan []string `json:"peringatan"`
	}](t, record)
	require.Len(t, body.Peringatan, 1)
	require.Contains(t, body.Peringatan[0], "Layer 1")
}

func TestIsianTerlaluPanjangDijawab422BesertaBagianYangSalah(t *testing.T) {
	server := testServer(t)

	tooLong := make([]byte, masterxol.MaxNameLength+1)
	for i := range tooLong {
		tooLong[i] = 'x'
	}

	record := call(t, server, http.MethodPost, basePath, map[string]any{"nama": string(tooLong)})
	require.Equal(t, http.StatusUnprocessableEntity, record.Code)

	body := decode[struct {
		Code   string `json:"kode"`
		Detail []struct {
			Field string `json:"field"`
		} `json:"detail"`
	}](t, record)
	require.Equal(t, "validasi_gagal", body.Code)
	require.Len(t, body.Detail, 1)
	require.Equal(t, "nama", body.Detail[0].Field)
}

func TestBadanPermintaanYangMemuatKolomKomiteDitolak(t *testing.T) {
	// PIC dan status komite diisi server dari sesi. Menerimanya dari klien berarti
	// membiarkan siapa pun mengaku sebagai pengaju — dan `D-59` menjadikan jejak itu
	// satu-satunya kontrol pengimbang yang tersisa.
	server := testServer(t)

	record := call(t, server, http.MethodPost, basePath, map[string]any{
		"nama": "Section 9", "pic": "ORANGLAIN", "status_komite": "1",
	})
	require.Equal(t, http.StatusBadRequest, record.Code)
	require.Equal(t, "permintaan_cacat", decode[map[string]any](t, record)["kode"])
}

func TestLimitRupiahDariKlienDiabaikanBukanDipercaya(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodPost, basePath, map[string]any{
		"nama": "Section 9", "kurs": 10000,
		"layer": []map[string]any{{"nama": "L1", "limit": 100, "limit_idr": 999999999}},
	})
	require.Equal(t, http.StatusCreated, record.Code)

	body := decode[struct {
		XOL struct {
			Layer []struct {
				LimitIDR int64 `json:"limit_idr"`
			} `json:"layer"`
		} `json:"xol"`
	}](t, record)
	require.Equal(t, int64(1_000_000), body.XOL.Layer[0].LimitIDR)
}

// ---------- hapus ----------

func TestMenghapusIndukMenjawab204DanBerkaskade(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodDelete, basePath+"/10001", nil)
	require.Equal(t, http.StatusNoContent, record.Code)

	require.Equal(t, http.StatusNotFound,
		call(t, server, http.MethodGet, basePath+"/10001", nil).Code)
}

func TestMenghapusIndukTidakDikenalDijawab404(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodDelete, basePath+"/99999", nil)
	require.Equal(t, http.StatusNotFound, record.Code)
}

func TestMenghapusSatuBarisBisnis(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodDelete, basePath+"/10001/bisnis/10013", nil)
	require.Equal(t, http.StatusNoContent, record.Code)

	body := decode[struct {
		XOL struct {
			Bisnis []any `json:"bisnis"`
		} `json:"xol"`
	}](t, call(t, server, http.MethodGet, basePath+"/10001", nil))
	require.Len(t, body.XOL.Bisnis, 3)
}

func TestMenghapusLayerIkutMembuangReasnya(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodDelete, basePath+"/10001/layer/10001", nil)
	require.Equal(t, http.StatusNoContent, record.Code)

	body := decode[struct {
		XOL struct {
			Layer []struct {
				ID string `json:"id"`
			} `json:"layer"`
		} `json:"xol"`
	}](t, call(t, server, http.MethodGet, basePath+"/10001", nil))
	require.Len(t, body.XOL.Layer, 2)
}

func TestMenghapusLayerMilikIndukLainDijawab404(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodDelete, basePath+"/10001/layer/10011", nil)
	require.Equal(t, http.StatusNotFound, record.Code)
	require.Equal(t, "layer_xol_tidak_ditemukan", decode[map[string]any](t, record)["kode"])
}

func TestMenghapusSatuBarisReas(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodDelete, basePath+"/10001/layer/10001/reas/10036322", nil)
	require.Equal(t, http.StatusNoContent, record.Code)

	body := decode[struct {
		XOL struct {
			Layer []struct {
				Reas []any `json:"reas"`
			} `json:"layer"`
		} `json:"xol"`
	}](t, call(t, server, http.MethodGet, basePath+"/10001", nil))
	require.Len(t, body.XOL.Layer[0].Reas, 5)
}

// ---------- bekal layar ----------

func TestBekalLayarMembawaTahunDanTypeSekaligus(t *testing.T) {
	server := testServer(t)

	record := call(t, server, http.MethodGet, basePath+"/form", nil)
	require.Equal(t, http.StatusOK, record.Code)

	body := decode[struct {
		Tahun []string `json:"tahun"`
		Tipe  []struct {
			Kode  string `json:"kode"`
			Label string `json:"label"`
		} `json:"tipe"`
	}](t, record)

	require.NotEmpty(t, body.Tahun)
	require.Len(t, body.Tipe, 3)
	require.Equal(t, "1", body.Tipe[0].Kode)
}

func TestJalurFormTidakTerbacaSebagaiNomorInduk(t *testing.T) {
	// `/form` dan `/bisnis` adalah ruas statis; bila chi membacanya sebagai {id},
	// jawabannya akan 404 "master tidak ditemukan", bukan bekal layar.
	server := testServer(t)

	require.Equal(t, http.StatusOK, call(t, server, http.MethodGet, basePath+"/form", nil).Code)
	require.Equal(t, http.StatusOK, call(t, server, http.MethodGet, basePath+"/bisnis?tipe=1", nil).Code)
}

func TestPilihanBisnisMengikutiTypeXOL(t *testing.T) {
	server := testServer(t)

	body := decode[struct {
		Bisnis []struct {
			Nama string `json:"nama"`
		} `json:"bisnis"`
	}](t, call(t, server, http.MethodGet, basePath+"/bisnis?tipe=3", nil))

	nama := make([]string, 0, len(body.Bisnis))
	for _, b := range body.Bisnis {
		nama = append(nama, b.Nama)
	}
	require.Contains(t, nama, "MARINE CARGO")
	require.Contains(t, nama, "TREATY INWARD")
	require.NotContains(t, nama, "MOTOR VEHICLE")
}
