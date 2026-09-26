package daftartipedokumenbisnishttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth/provider"
	authmemory "claim-pnc/internal/auth/repo/memory"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/daftartipedokumenbisnis"
	"claim-pnc/internal/daftartipedokumenbisnis/repo/memory"
	daftartipedokumenbisnisusecase "claim-pnc/internal/daftartipedokumenbisnis/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	daftartipedokumenbisnishttp "claim-pnc/internal/daftartipedokumenbisnis/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa data satu entitas tidak
// pernah terbaca lewat entitas lain. Menguji handler secara terpisah tidak dapat
// membuktikan keduanya.
type testServer struct {
	server *httptest.Server
	token  string

	// position adalah pyPosition yang dilaporkan jembatan Caller. Ia menentukan apakah
	// tombol "Pilih semua" ditampilkan — meniru `pyVisible` layar lama.
	position *string

	// asi sengaja dibiarkan kosong; ia yang membuktikan pemisahan antarentitas.
	asm *memory.Repo
	asi *memory.Repo
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	authService, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        authmemory.NewUserRepo(),
		SessionRepo:     authmemory.NewSessionRepo(),
		Clock:           clock.FixedAt(time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewRepo(memory.SampleList()...)
	asi := memory.NewRepo()
	business := memory.NewBusinessRepo(memory.SampleBusinessList()...)
	documentType := memory.NewReferenceRepo(memory.SampleDocumentTypeList()...)
	detailTypeDoc := memory.NewReferenceRepo(memory.SampleDetailTypeDocList()...)
	objectDoc := memory.NewReferenceRepo(memory.SampleObjectDocList()...)

	perPortal := func(alias string) error {
		if alias != "ASM" && alias != "ASI" {
			return portal.ErrNotReady
		}
		return nil
	}

	service, err := daftartipedokumenbisnisusecase.NewService(daftartipedokumenbisnisusecase.Options{
		RepoSelector: func(alias string) (daftartipedokumenbisnis.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		BusinessSelector: func(alias string) (daftartipedokumenbisnis.BusinessRepo, error) {
			if err := perPortal(alias); err != nil {
				return nil, err
			}
			return business, nil
		},
		DocumentTypeSelector: func(alias string) (daftartipedokumenbisnis.DocumentTypeRepo, error) {
			if err := perPortal(alias); err != nil {
				return nil, err
			}
			return documentType, nil
		},
		DetailTypeDocSelector: func(alias string) (daftartipedokumenbisnis.DetailTypeDocRepo, error) {
			if err := perPortal(alias); err != nil {
				return nil, err
			}
			return detailTypeDoc, nil
		},
		ObjectDocSelector: func(alias string) (daftartipedokumenbisnis.ObjectDocRepo, error) {
			if err := perPortal(alias); err != nil {
				return nil, err
			}
			return objectDoc, nil
		},
		Clock: clock.FixedAt(time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)),
		// Satu dari kelima kode lini MBU yang dilewati "Pilih semua" di Pega.
		BulkSelectExcludedBusinesses: []string{"10028"},
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Keduanya dideklarasikan sebagai tipe fungsi TANPA NAMA, bukan dibiarkan mengambil
	// tipe bernama milik salah satu modul. Setiap modul mendeklarasikan tipe penulisnya
	// sendiri — persis supaya modul tidak saling mengimpor.
	var writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any) = func(
		w http.ResponseWriter, r *http.Request, status int, body any,
	) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	position := "NONMBU"

	handler, err := daftartipedokumenbisnishttp.NewHandler(daftartipedokumenbisnishttp.Options{
		Service: service,
		Logger:  logger,
		Caller: func(_ context.Context) (daftartipedokumenbisnishttp.Caller, bool) {
			return daftartipedokumenbisnishttp.Caller{
				Identity: "90000001",
				Position: position,
			}, true
		},
		WriteResponse: writeResponse,
		WriteError:    daftartipedokumenbisnishttp.ErrorWriter(writeError),
	})
	require.NoError(t, err)

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   writeError,
	}

	server := httptest.NewServer(httpserver.Router(httpserver.Deps{
		Logger: logger,
		MountAPI: func(api chi.Router) {
			authhttp.Mount(api, authhttp.NewHandler(authService, logger), authService, logger)
			api.Group(func(protected chi.Router) {
				protected.Use(authhttp.Authenticate(authService, writeError))
				daftartipedokumenbisnishttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server, asm: asm, asi: asi, position: &position}
	p.token = p.login(t)
	return p
}

func (p *testServer) login(t *testing.T) string {
	t.Helper()

	body := strings.NewReader(`{"nama_pengguna":"adminpnc","kata_sandi":"rahasia123"}`)
	response, err := http.Post(p.server.URL+"/api/masuk", "application/json", body)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusOK, response.StatusCode)

	var content struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&content))
	require.NotEmpty(t, content.Token)
	return content.Token
}

// call menjalankan satu permintaan. portalAlias kosong berarti header tidak dikirim.
func (p *testServer) call(t *testing.T, method, path, portalAlias, body string) (*http.Response, map[string]any) {
	t.Helper()

	request, err := http.NewRequest(method, p.server.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	return response, content
}

const route = "/api/master/tipe-dokumen-bisnis"

func TestDaftarBisnisMenyebutEntitasYangMenjawabnya(t *testing.T) {
	// Pada aplikasi yang melayani empat badan hukum, "data siapa ini" tidak boleh hanya
	// diandaikan — server menyebutkannya (`R-20`).
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])
	require.NotZero(t, content["total"])
}

func TestEntitasLainTidakMelihatDataEntitasIni(t *testing.T) {
	// Inilah uji yang membenarkan seluruh mekanisme pemilihan portal.
	server := newTestServer(t)

	_, asm := server.call(t, http.MethodGet, route, "ASM", "")
	require.NotZero(t, asm["total"])

	response, asi := server.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(0), asi["total"])
}

func TestPermintaanTanpaPortalDitolak(t *testing.T) {
	// Menolak, BUKAN jatuh ke portal utama. Jatuh ke default berarti menulis data satu
	// badan hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodGet, route, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestPortalYangBelumSiapDitolak(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodGet, route, "SYARIAH", "")
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

func TestRuteMenuntutSesi(t *testing.T) {
	server := newTestServer(t)
	server.token = ""

	response, _ := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// Grid tingkat kedua: aturan milik satu lini bisnis.
func TestDaftarPerBisnisHanyaMemuatBisnisItu(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/bisnis/001", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	rows, _ := content["tipe_dokumen_bisnis"].([]any)
	require.NotEmpty(t, rows)
	for _, row := range rows {
		require.Equal(t, "001", row.(map[string]any)["id_bisnis"])
	}
}

// Lini bisnis yang belum punya aturan menjawab daftar KOSONG, bukan 404. Layar Tambah
// justru menuju ke sana — menjawab 404 akan menutup jalan menuju satu-satunya layar yang
// dapat memperbaikinya.
func TestBisnisTanpaAturanMenjawabDaftarKosongBukan404(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/bisnis/005", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(0), content["total"])
}

// Penambahan menghasilkan PERKALIAN bisnis kali baris dokumen — perilaku yang tertulis
// pada komentar langkah `InsertDetailTypeDocumentBusiness_act` sendiri.
func TestPenambahanMenghasilkanPerkalianBisnisKaliDokumen(t *testing.T) {
	server := newTestServer(t)

	body := `{"bisnis":["001","004","005"],"dokumen":[
		{"id_tipe_dokumen":"20001","id_object_dokumen":"","id_detail_dokumen":"40001",
		 "detail_dokumen":"Laporan Kerugian","status_wajib":true,"minimum_dokumen":1},
		{"id_tipe_dokumen":"20003","id_object_dokumen":"","id_detail_dokumen":"40003",
		 "detail_dokumen":"Berita Acara Survei","status_wajib":false,"minimum_dokumen":0}]}`

	response, content := server.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, float64(6), content["total"])
}

// Satu-satunya aturan isian yang ditiru dari layar lama
// (`InsertDetailTypeDocumentBusiness_act:209`), termasuk ejaannya.
func TestPenambahanTanpaBisnisDitolakDenganKalimatLayarLama(t *testing.T) {
	server := newTestServer(t)

	body := `{"bisnis":[],"dokumen":[{"id_tipe_dokumen":"20001","id_object_dokumen":"",
		"id_detail_dokumen":"40001","detail_dokumen":"Laporan","status_wajib":true,
		"minimum_dokumen":1}]}`

	response, content := server.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "nama_bisnis_belum_diisi", content["kode"])
	require.Equal(t, "Nama Bisnis belum di isi.", content["pesan"])
}

// Penambahan TANPA baris dokumen DITERIMA, dan tidak menyimpan apa pun.
//
// Meniru Pega: perulangan atas baris dokumen berputar nol kali lalu layar tertutup, tanpa
// pesan apa pun. Penolakan yang sempat ada di sini adalah penambahan saya sendiri, dicabut
// atas keputusan Work Owner 2026-09-23.
func TestPenambahanTanpaBarisDokumenDiterimaTanpaMenyimpan(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route, "ASM", `{"bisnis":["001"],"dokumen":[]}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, float64(0), content["total"])
}

// Jejak simpan tidak dapat dikirim klien: tanpa itu USER_EDIT — kolom yang justru dipakai
// menelusuri siapa mengubah apa — dapat diaku-aku.
func TestBadanPermintaanYangMenyelundupkanJejakSimpanDitolak(t *testing.T) {
	server := newTestServer(t)

	body := `{"bisnis":["001"],"user_edit":"ORANGLAIN","dokumen":[]}`
	response, content := server.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestPenyuntinganMengubahBarisDanMengembalikannya(t *testing.T) {
	server := newTestServer(t)

	body := `{"id_tipe_dokumen":"20004","id_object_dokumen":"30003","id_detail_dokumen":"40005",
		"detail_dokumen":"Kuitansi Pembayaran","status_wajib":false,"minimum_dokumen":2}`

	response, content := server.call(t, http.MethodPut, route+"/10001", "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	rule := content["tipe_dokumen_bisnis"].(map[string]any)
	require.Equal(t, "Kuitansi Pembayaran", rule["detail_dokumen"])
	require.Equal(t, false, rule["status_wajib"])
	require.Equal(t, float64(2), rule["minimum_dokumen"])
}

// BUSINESSID tidak ikut berubah — procedure lama pun tidak mengubahnya
// (`PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:39-41`). Sebuah aturan tidak dapat dipindahkan ke
// lini bisnis lain; ia disalin.
func TestPenyuntinganTidakMemindahkanBarisKeBisnisLain(t *testing.T) {
	server := newTestServer(t)

	body := `{"id_tipe_dokumen":"20001","id_object_dokumen":"","id_detail_dokumen":"40001",
		"detail_dokumen":"Laporan Kerugian","status_wajib":true,"minimum_dokumen":1}`

	_, content := server.call(t, http.MethodPut, route+"/10001", "ASM", body)
	rule := content["tipe_dokumen_bisnis"].(map[string]any)
	require.Equal(t, "001", rule["id_bisnis"])
}

// Badan permintaan penyuntingan TIDAK menerima `bisnis`. Menerimanya lalu mengabaikannya
// akan membuat klien mengira perpindahan berhasil.
func TestPenyuntinganMenolakBadanYangMenyertakanBisnis(t *testing.T) {
	server := newTestServer(t)

	body := `{"bisnis":["004"],"id_tipe_dokumen":"20001","id_object_dokumen":"",
		"id_detail_dokumen":"40001","detail_dokumen":"Laporan","status_wajib":true,
		"minimum_dokumen":1}`

	response, _ := server.call(t, http.MethodPut, route+"/10001", "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestPenyuntinganBarisYangTidakAdaMenjawab404(t *testing.T) {
	server := newTestServer(t)

	body := `{"id_tipe_dokumen":"20001","id_object_dokumen":"","id_detail_dokumen":"40001",
		"detail_dokumen":"Laporan","status_wajib":true,"minimum_dokumen":1}`

	response, content := server.call(t, http.MethodPut, route+"/99999", "ASM", body)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tipe_dokumen_bisnis_tidak_ditemukan", content["kode"])
}

// Daftar TIDAK membawa jaminan; pengambilan satu baris membawanya.
//
// Uji ini menjaga sebuah KEPUTUSAN, bukan sekadar menjalankan kode: bila kelak seseorang
// "merapikan" keduanya menjadi sama, layar akan mengira setiap baris tidak punya jaminan —
// dan itu perbedaan antara dokumen yang wajib dan yang tidak.
func TestDaftarTidakMembawaJaminanTetapiPengambilanSatuBarisMembawanya(t *testing.T) {
	server := newTestServer(t)

	_, list := server.call(t, http.MethodGet, route+"/bisnis/001", "ASM", "")
	rows := list["tipe_dokumen_bisnis"].([]any)
	for _, row := range rows {
		require.Empty(t, row.(map[string]any)["jenis_klaim"])
	}

	_, single := server.call(t, http.MethodGet, route+"/10001", "ASM", "")
	rule := single["tipe_dokumen_bisnis"].(map[string]any)
	require.Len(t, rule["jenis_klaim"], 1)
}

// Jaminan DITAMBAHKAN, tidak menggantikan daftarnya. Meniru
// `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:55` — tidak ada satu pun DELETE terhadap tabel itu
// di seluruh export.
func TestPenambahanJaminanTidakMenggantikanYangSudahAda(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route+"/10001/jenis-klaim", "ASM",
		`{"id_jenis_klaim":"10015"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	rule := content["tipe_dokumen_bisnis"].(map[string]any)
	require.Len(t, rule["jenis_klaim"], 2)
}

// Jaminan yang sudah ada DIBIARKAN, bukan ditolak sebagai galat — procedure lama pun
// hanya menyisipkan bila belum ada.
func TestPenambahanJaminanYangSudahAdaDiamSaja(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route+"/10001/jenis-klaim", "ASM",
		`{"id_jenis_klaim":"10009"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	rule := content["tipe_dokumen_bisnis"].(map[string]any)
	require.Len(t, rule["jenis_klaim"], 1)
}

// Jaminan kosong DILEWATI, bukan ditolak.
//
// Meniru `InsertDetailTypeDocumentBusiness_act:3547`, yang precondition langkahnya
// berbunyi `.ContentNote==""` lalu KELUAR dari iterasi — baris itu sekadar tidak disimpan.
// Penolakan yang sempat ada di sini adalah penambahan saya sendiri, dicabut atas keputusan
// Work Owner 2026-09-23.
func TestPenambahanJaminanKosongDilewatiBukanDitolak(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route+"/10001/jenis-klaim", "ASM",
		`{"id_jenis_klaim":"  "}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	// Daftarnya tidak bertambah: yang kosong memang tidak disimpan.
	rule := content["tipe_dokumen_bisnis"].(map[string]any)
	require.Len(t, rule["jenis_klaim"], 1)
}

// Keempat daftar pilihan ikut menuntut portal: keempat masternya hidup di basis data
// setiap entitas.
func TestKeempatDaftarPilihanMenuntutPortal(t *testing.T) {
	server := newTestServer(t)

	for _, path := range []string{
		"/api/master/bisnis-pilihan",
		"/api/master/tipe-dokumen-pilihan",
		"/api/master/detail-dokumen-pilihan",
		"/api/master/objek-dokumen-pilihan",
	} {
		response, _ := server.call(t, http.MethodGet, path, "", "")
		require.Equal(t, http.StatusBadRequest, response.StatusCode, path)

		response, content := server.call(t, http.MethodGet, path, "ASM", "")
		require.Equal(t, http.StatusOK, response.StatusCode, path)
		require.Equal(t, "ASM", content["portal"], path)
	}
}

// Daftar pilihan bisnis memuat SELURUH bisnis, bukan hanya yang sudah punya aturan.
// Bisnis yang belum punya satu baris pun justru yang paling sering dituju layar Tambah.
func TestDaftarPilihanBisnisMemuatBisnisYangBelumPunyaAturan(t *testing.T) {
	server := newTestServer(t)

	_, punya := server.call(t, http.MethodGet, route, "ASM", "")
	_, semua := server.call(t, http.MethodGet, "/api/master/bisnis-pilihan", "ASM", "")

	require.Greater(t, semua["total"], punya["total"])
}

// Kelima kode lini MBU dilewati "Pilih semua" (`SetAllBusiness-Act.xml:984`), tetapi tetap
// muncul di daftarnya — pengecualiannya hanya berlaku pada tombolnya, bukan pada daftar.
func TestBisnisYangDikecualikanTetapMunculDiDaftarnya(t *testing.T) {
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet, "/api/master/bisnis-pilihan", "ASM", "")
	list := content["bisnis"].([]any)

	var excluded int
	var foundExcludedCode bool
	for _, row := range list {
		entry := row.(map[string]any)
		if entry["dikecualikan_pilih_semua"] == true {
			excluded++
			if entry["id"] == "10028" {
				foundExcludedCode = true
			}
		}
	}

	require.NotEmpty(t, list, "daftar bisnis kosong; uji ini kehilangan gunanya")
	require.Equal(t, 1, excluded, "hanya kode yang dikonfigurasi yang boleh ditandai")
	require.True(t, foundExcludedCode, "yang ditandai harus kode yang dikonfigurasi, bukan yang lain")
}

// Tombol "Pilih semua" hanya TERLIHAT oleh pyPosition NONMBU, meniru `pyVisible` layar
// lama. Ia penyembunyian tampilan, bukan kewenangan: seluruh rute tetap dapat dipanggil.
func TestPenandaPilihSemuaMengikutiPosisiPemanggil(t *testing.T) {
	server := newTestServer(t)

	_, allowed := server.call(t, http.MethodGet, "/api/master/bisnis-pilihan", "ASM", "")
	require.Equal(t, true, allowed["boleh_pilih_semua"])

	*server.position = "MBU"
	_, denied := server.call(t, http.MethodGet, "/api/master/bisnis-pilihan", "ASM", "")
	require.Equal(t, false, denied["boleh_pilih_semua"])

	// Daftarnya tetap terkirim utuh: yang disembunyikan tombolnya, bukan datanya.
	require.Equal(t, allowed["total"], denied["total"])
}

// Rincian dokumen membawa tahap pemiliknya, sehingga layar dapat menyempitkan pilihannya
// mengikuti Tipe Dokumen yang sudah dipilih — persis parameter `idDocument` pada
// `BrowseVLstDetTypeDoc_RD`.
func TestDaftarPilihanDetailDokumenMembawaTahapPemiliknya(t *testing.T) {
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet, "/api/master/detail-dokumen-pilihan", "ASM", "")
	items := content["pilihan"].([]any)
	require.NotEmpty(t, items)
	for _, item := range items {
		require.NotEmpty(t, item.(map[string]any)["id_induk"])
	}
}
