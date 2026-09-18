package statusprogreshttp_test

import (
	"bytes"
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
	authmemori "claim-pnc/internal/auth/repo/memori"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/platform/waktu"
	"claim-pnc/internal/portal"
	"claim-pnc/internal/statusprogres"
	"claim-pnc/internal/statusprogres/repo/memori"
	statusprogresusecase "claim-pnc/internal/statusprogres/usecase"

	authhttp "claim-pnc/internal/auth/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemori "claim-pnc/internal/portal/repo/memori"
	statusprogreshttp "claim-pnc/internal/statusprogres/http"
)

// peladen merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa permintaan tanpa portal
// yang sah ditolak. Menguji handler secara terpisah tidak dapat membuktikan keduanya.
type peladen struct {
	server *httptest.Server
	token  string

	// asi tidak diberi isi apa pun; ia yang membuktikan pemisahan antarentitas.
	asm *memori.Repo
	asi *memori.Repo
}

func siapkanPeladen(t *testing.T) *peladen {
	t.Helper()

	sistemIdentitas, err := provider.TiruanBaru(false, nil)
	require.NoError(t, err)

	layananAuth, err := usecase.LayananBaru(usecase.Opsi{
		Identitas:       sistemIdentitas,
		PenggunaRepo:    authmemori.PenggunaRepoBaru(),
		SesiRepo:        authmemori.SesiRepoBaru(),
		Jam:             waktu.JamTetapPada(time.Date(2026, 9, 17, 3, 0, 0, 0, time.UTC)),
		MasaBerlakuSesi: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memori.RepoBaru(memori.DaftarContoh()...)
	asi := memori.RepoBaru()

	layananStatusProgres, err := statusprogresusecase.LayananBaru(statusprogresusecase.Opsi{
		PemilihRepo: func(alias string) (statusprogres.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrBelumSiap
			}
		},
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var tulisRespon func(w http.ResponseWriter, r *http.Request, status int, badan any) = func(
		w http.ResponseWriter, r *http.Request, status int, badan any,
	) {
		authhttp.TulisJSON(w, r, status, badan, logger)
	}
	// Dirantai sama seperti cmd/claimpnc: galat portal dipetakan modul portal, sisanya
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan
	// portal akan lulus di sini tetapi gagal di aplikasi sungguhan.
	var tulisGalat func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.DenganGalatPortal(
		authhttp.TulisGalat(logger), tulisRespon,
	)

	handler, err := statusprogreshttp.HandlerBaru(statusprogreshttp.Opsi{
		Layanan:     layananStatusProgres,
		Logger:      logger,
		TulisRespon: tulisRespon,
		TulisGalat:  tulisGalat,
	})
	require.NoError(t, err)

	// Hanya ASM dan ASI yang koneksinya "hidup"; SMAS ada di daftar tetapi belum siap.
	// Itulah yang membedakan "tidak ada" dari "belum tersedia".
	bahanPortal := portalhttp.BahanPortalAktif{
		Repo:       portalmemori.RepoBaru(portalmemori.DaftarContoh()...),
		AliasSiap:  func() []string { return []string{"ASM", "ASI"} },
		Logger:     logger,
		TulisGalat: tulisGalat,
	}

	server := httptest.NewServer(httpserver.Router(httpserver.Bahan{
		Logger: logger,
		PasangAPI: func(api chi.Router) {
			authhttp.Pasang(api, authhttp.HandlerBaru(layananAuth, logger), layananAuth, logger)
			api.Group(func(terlindungi chi.Router) {
				terlindungi.Use(authhttp.Autentikasi(layananAuth, tulisGalat))
				statusprogreshttp.Pasang(terlindungi, handler, bahanPortal)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &peladen{server: server, asm: asm, asi: asi}
	p.token = p.masuk(t)
	return p
}

func (p *peladen) masuk(t *testing.T) string {
	t.Helper()

	badan := strings.NewReader(`{"nama_pengguna":"adminpnc","kata_sandi":"rahasia123"}`)
	respons, err := http.Post(p.server.URL+"/api/masuk", "application/json", badan)
	require.NoError(t, err)
	defer func() { _ = respons.Body.Close() }()
	require.Equal(t, http.StatusOK, respons.StatusCode)

	var isi struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(respons.Body).Decode(&isi))
	require.NotEmpty(t, isi.Token)
	return isi.Token
}

// panggil menjalankan satu permintaan. portalAlias kosong berarti header tidak dikirim.
func (p *peladen) panggil(t *testing.T, metode, jalur, portalAlias, badan string) (*http.Response, map[string]any) {
	t.Helper()

	var pembaca *strings.Reader
	if badan == "" {
		pembaca = strings.NewReader("")
	} else {
		pembaca = strings.NewReader(badan)
	}

	permintaan, err := http.NewRequest(metode, p.server.URL+jalur, pembaca)
	require.NoError(t, err)
	if p.token != "" {
		permintaan.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		permintaan.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}
	if badan != "" {
		permintaan.Header.Set("Content-Type", "application/json")
	}

	respons, err := http.DefaultClient.Do(permintaan)
	require.NoError(t, err)
	t.Cleanup(func() { _ = respons.Body.Close() })

	isi := map[string]any{}
	_ = json.NewDecoder(respons.Body).Decode(&isi)
	return respons, isi
}

// Rutenya berada di balik sesi. Tanpa token, permintaan ditolak sebelum menyentuh
// pemeriksaan portal maupun basis data.
func TestTanpaSesiDitolak(t *testing.T) {
	p := siapkanPeladen(t)
	tokenAsli := p.token
	p.token = ""

	respons, isi := p.panggil(t, http.MethodGet, "/api/master/status-progres-1", "ASM", "")
	require.Equal(t, http.StatusUnauthorized, respons.StatusCode)
	require.Equal(t, authhttp.KodeSesiTidakSah, isi["kode"])

	p.token = tokenAsli
}

// PERMINTAAN TANPA PORTAL DITOLAK, bukan dilayani portal utama (TKT-F6-002, R-20).
//
// Ini uji terpenting di berkas ini. Bila permintaan tanpa portal jatuh ke koneksi
// default, data satu badan hukum akan terbaca atau tertulis di basis data badan hukum
// lain — dan layarnya tampak normal, karena angkanya masuk akal. Yang salah hanya
// milik siapa data itu.
func TestTanpaPortalDitolak(t *testing.T) {
	p := siapkanPeladen(t)

	t.Run("membaca daftar", func(t *testing.T) {
		respons, isi := p.panggil(t, http.MethodGet, "/api/master/status-progres-1", "", "")
		require.Equal(t, http.StatusBadRequest, respons.StatusCode)
		require.Equal(t, portalhttp.KodeTidakDisebut, isi["kode"])
	})

	t.Run("menambah", func(t *testing.T) {
		respons, isi := p.panggil(t, http.MethodPost, "/api/master/status-progres-1", "",
			`{"nama":"UJI","kode_posisi":"002"}`)
		require.Equal(t, http.StatusBadRequest, respons.StatusCode)
		require.Equal(t, portalhttp.KodeTidakDisebut, isi["kode"])
	})

	t.Run("mengubah", func(t *testing.T) {
		respons, isi := p.panggil(t, http.MethodPut, "/api/master/status-progres-1/01", "",
			`{"nama":"UJI","kode_posisi":"002"}`)
		require.Equal(t, http.StatusBadRequest, respons.StatusCode)
		require.Equal(t, portalhttp.KodeTidakDisebut, isi["kode"])
	})

	// Dan tidak ada satu baris pun yang berubah di entitas mana pun.
	daftarASM, err := p.asm.Daftar(t.Context())
	require.NoError(t, err)
	require.Len(t, daftarASM, 6)
	daftarASI, err := p.asi.Daftar(t.Context())
	require.NoError(t, err)
	require.Empty(t, daftarASI)
}

// "Tidak ada" dibedakan dari "belum tersedia": keduanya menuntut tindak lanjut berbeda.
func TestPortalTidakDikenalDanBelumSiapDibedakan(t *testing.T) {
	p := siapkanPeladen(t)

	t.Run("tidak ada di daftar", func(t *testing.T) {
		respons, isi := p.panggil(t, http.MethodGet, "/api/master/status-progres-1", "TIDAKADA", "")
		require.Equal(t, http.StatusBadRequest, respons.StatusCode)
		require.Equal(t, portalhttp.KodeTidakDikenal, isi["kode"])
	})

	t.Run("ada tetapi koneksinya belum hidup", func(t *testing.T) {
		respons, isi := p.panggil(t, http.MethodGet, "/api/master/status-progres-1", "SMAS", "")
		require.Equal(t, http.StatusServiceUnavailable, respons.StatusCode)
		require.Equal(t, portalhttp.KodeBelumSiap, isi["kode"])
	})
}

func TestDaftarMenyebutkanPortalYangMenjawab(t *testing.T) {
	p := siapkanPeladen(t)

	respons, isi := p.panggil(t, http.MethodGet, "/api/master/status-progres-1", "ASM", "")
	require.Equal(t, http.StatusOK, respons.StatusCode)
	require.Equal(t, "ASM", isi["portal"], "layar harus dapat memastikan data ini milik entitas yang dipilih")

	baris, ok := isi["status_progres"].([]any)
	require.True(t, ok)
	require.Len(t, baris, 6)

	pertama, ok := baris[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "01", pertama["id"])
	require.Equal(t, "DOKUMEN DITERIMA", pertama["nama"])
	require.Equal(t, "002", pertama["kode_posisi"])
	require.Equal(t, "REGISTER", pertama["nama_posisi"], "label dikirim bersama kodenya")
}

// Tabel kosong terkirim sebagai [] dan bukan null: layar yang menerima null harus
// menjaganya sendiri, dan satu layar yang lupa akan gagal justru saat tabelnya kosong.
func TestDaftarKosongTerkirimSebagaiArrayKosong(t *testing.T) {
	p := siapkanPeladen(t)

	respons, _ := p.panggil(t, http.MethodGet, "/api/master/status-progres-1", "ASI", "")
	require.Equal(t, http.StatusOK, respons.StatusCode)

	permintaan, err := http.NewRequest(http.MethodGet, p.server.URL+"/api/master/status-progres-1", nil)
	require.NoError(t, err)
	permintaan.Header.Set("Authorization", "Bearer "+p.token)
	permintaan.Header.Set(portalhttp.HeaderPortal, "ASI")
	mentah, err := http.DefaultClient.Do(permintaan)
	require.NoError(t, err)
	defer func() { _ = mentah.Body.Close() }()

	badan := make([]byte, 512)
	n, _ := mentah.Body.Read(badan)
	require.Contains(t, string(badan[:n]), `"status_progres":[]`)
}

func TestTambahMengembalikanBarisTersimpan(t *testing.T) {
	p := siapkanPeladen(t)

	respons, isi := p.panggil(t, http.MethodPost, "/api/master/status-progres-1", "ASI",
		`{"nama":"MENUNGGU BERKAS","kode_posisi":"004"}`)
	require.Equal(t, http.StatusCreated, respons.StatusCode)

	baris, ok := isi["status_progres"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "01", baris["id"], "ID diterbitkan server; layar tidak punya cara lain mengetahuinya")
	require.Equal(t, "MENUNGGU BERKAS", baris["nama"])
	require.Equal(t, "SURVEY", baris["nama_posisi"])

	// Tersimpan di ASI, dan HANYA di ASI.
	diASI, err := p.asi.Daftar(t.Context())
	require.NoError(t, err)
	require.Len(t, diASI, 1)
	diASM, err := p.asm.Daftar(t.Context())
	require.NoError(t, err)
	require.Len(t, diASM, 6)
}

// Seluruh pelanggaran isian dikirim sekaligus, dan status 422 — bukan 400.
func TestValidasiGagalMengirimSeluruhPelanggaran(t *testing.T) {
	p := siapkanPeladen(t)

	respons, isi := p.panggil(t, http.MethodPost, "/api/master/status-progres-1", "ASM",
		`{"nama":"","kode_posisi":"999"}`)
	require.Equal(t, http.StatusUnprocessableEntity, respons.StatusCode)
	require.Equal(t, statusprogreshttp.KodeValidasiGagal, isi["kode"])

	detail, ok := isi["detail"].([]any)
	require.True(t, ok, "detail per isian wajib ada supaya layar dapat menyorot isian yang salah")
	require.Len(t, detail, 2)

	kolom := map[string]bool{}
	for _, d := range detail {
		baris, ok := d.(map[string]any)
		require.True(t, ok)
		kolom[baris["kolom"].(string)] = true
	}
	require.True(t, kolom["nama"])
	require.True(t, kolom["kode_posisi"])
}

func TestUbahBarisYangAda(t *testing.T) {
	p := siapkanPeladen(t)

	respons, isi := p.panggil(t, http.MethodPut, "/api/master/status-progres-1/03", "ASM",
		`{"nama":"SURVEI DIJADWALKAN","kode_posisi":"006"}`)
	require.Equal(t, http.StatusOK, respons.StatusCode)

	baris, ok := isi["status_progres"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "03", baris["id"])
	require.Equal(t, "SURVEI DIJADWALKAN", baris["nama"])
	require.Equal(t, "KOMITE", baris["nama_posisi"])
}

func TestUbahBarisYangTidakAda(t *testing.T) {
	p := siapkanPeladen(t)

	respons, isi := p.panggil(t, http.MethodPut, "/api/master/status-progres-1/99", "ASM",
		`{"nama":"APA SAJA","kode_posisi":"002"}`)
	require.Equal(t, http.StatusNotFound, respons.StatusCode)
	require.Equal(t, statusprogreshttp.KodeTidakDitemukan, isi["kode"])
}

// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa tanda.
func TestFieldTidakDikenalDitolak(t *testing.T) {
	p := siapkanPeladen(t)

	respons, isi := p.panggil(t, http.MethodPost, "/api/master/status-progres-1", "ASM",
		`{"nama":"UJI","kode_posisi":"002","namaa":"salah ketik"}`)
	require.Equal(t, http.StatusBadRequest, respons.StatusCode)
	require.Equal(t, statusprogreshttp.KodePermintaanCacat, isi["kode"])
}

func TestBadanCacatDitolak(t *testing.T) {
	p := siapkanPeladen(t)

	respons, isi := p.panggil(t, http.MethodPost, "/api/master/status-progres-1", "ASM", `{bukan json`)
	require.Equal(t, http.StatusBadRequest, respons.StatusCode)
	require.Equal(t, statusprogreshttp.KodePermintaanCacat, isi["kode"])
}

// Daftar posisi TIDAK menuntut portal: ia daftar milik aplikasi, bukan isi basis data
// entitas mana pun. Menuntut portal di sini akan membuat dropdown gagal justru saat
// pengguna belum memilih portal.
func TestDaftarPosisiTidakMenuntutPortal(t *testing.T) {
	p := siapkanPeladen(t)

	respons, isi := p.panggil(t, http.MethodGet, "/api/master/posisi-klaim", "", "")
	require.Equal(t, http.StatusOK, respons.StatusCode)

	posisi, ok := isi["posisi"].([]any)
	require.True(t, ok)
	require.Len(t, posisi, 4)

	pertama, ok := posisi[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "002", pertama["kode"])
	require.Equal(t, "REGISTER", pertama["nama"])
}

// Tetapi ia tetap berada di balik sesi.
func TestDaftarPosisiTetapMenuntutSesi(t *testing.T) {
	p := siapkanPeladen(t)
	p.token = ""

	respons, _ := p.panggil(t, http.MethodGet, "/api/master/posisi-klaim", "", "")
	require.Equal(t, http.StatusUnauthorized, respons.StatusCode)
}
