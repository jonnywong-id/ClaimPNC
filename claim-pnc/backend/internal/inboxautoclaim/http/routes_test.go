package inboxautoclaimhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
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
	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/inboxautoclaim/repo/memory"
	inboxautoclaimusecase "claim-pnc/internal/inboxautoclaim/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	inboxautoclaimhttp "claim-pnc/internal/inboxautoclaim/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi,
// middleware portal, dan jembatan konteks pemanggil.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang sah
// ditolak, dan bahwa unggahan menandai baris dengan login pemanggil. Menguji handler
// secara terpisah tidak dapat membuktikan ketiganya.
type testServer struct {
	server *httptest.Server
	token  string

	// asi tidak diberi isi apa pun; ia yang membuktikan pemisahan antarentitas.
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewSampleRepo()

	// ASI diberi master dan polis yang SAMA tetapi tanpa satu baris batch pun. Dengan
	// begitu, daftar batch yang kosong pada ASI membuktikan pemisahan datanya — bukan
	// sekadar membuktikan bahwa ASI belum dikonfigurasi.
	asi := memory.NewRepo(memory.SampleMaster(), memory.SamplePolicy())

	autoClaimService, err := inboxautoclaimusecase.NewService(inboxautoclaimusecase.Options{
		RepoSelector: func(alias string) (inboxautoclaim.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any) = func(
		w http.ResponseWriter, r *http.Request, status int, body any,
	) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	// Dirantai sama seperti cmd/claimpnc: galat portal dipetakan modul portal, sisanya
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan portal
	// akan lulus di sini tetapi gagal di aplikasi sungguhan.
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	handler, err := inboxautoclaimhttp.NewHandler(inboxautoclaimhttp.Options{
		Service: autoClaimService,
		Caller: func(ctx context.Context) (inboxautoclaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxautoclaimhttp.Caller{}, false
			}
			return inboxautoclaimhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    writeError,
	})
	require.NoError(t, err)

	// Hanya ASM dan ASI yang koneksinya "hidup"; SMAS ada di daftar tetapi belum siap.
	// Itulah yang membedakan "tidak ada" dari "belum tersedia".
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
				inboxautoclaimhttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server, asm: asm, asi: asi}
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

// call menjalankan satu permintaan JSON. portalAlias kosong berarti header tidak dikirim.
func (p *testServer) call(t *testing.T, method, path, portalAlias string) (*http.Response, map[string]any) {
	t.Helper()

	request, err := http.NewRequest(method, p.server.URL+path, nil)
	require.NoError(t, err)
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	return response, content
}

// upload mengirim satu berkas CSV sebagai multipart, persis seperti peramban.
func (p *testServer) upload(t *testing.T, portalAlias, csv string) (*http.Response, map[string]any) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("berkas", "auto-claim.csv")
	require.NoError(t, err)
	_, err = part.Write([]byte(csv))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request, err := http.NewRequest(http.MethodPost, p.server.URL+"/api/inbox-auto-claim/unggah?sumber=aneka", &body)
	require.NoError(t, err)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	return response, content
}

// csvJudul adalah baris judul berkas unggahan.
//
// Perhatikan tiga kolom yang TIDAK ada: inisialid, prodke, dan currency. Ketiganya hasil
// pencarian polis, bukan isian pengunggah — lihat Activity/InsertKlaimToTable_Other-Act.xml.
const csvJudul = "policyno,claimamount,dateofloss,reportdate,causeofloss,keyword,alasanklaim"

func TestDaftarBatchMenolakPermintaanTanpaSesi(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim", "ASM")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.Equal(t, "sesi_tidak_sah", content["kode"])
}

func TestDaftarBatchMenolakPermintaanTanpaPortal(t *testing.T) {
	// Tidak pernah jatuh ke portal utama sebagai cadangan: itu berarti membaca data satu
	// badan hukum di basis data badan hukum lain tanpa satu pun pesan galat (R-20).
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_disebut", content["kode"])
}

func TestDaftarBatchMenolakPortalYangBelumSiap(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim", "SMAS")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "portal_belum_siap", content["kode"])
}

func TestDaftarBatchMenjawabSembilanKolomGrid(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])

	batch, ok := content["batch"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, batch)

	pertama, ok := batch[0].(map[string]any)
	require.True(t, ok)
	for _, kolom := range []string{
		"kode_perusahaan", "nama_perusahaan", "batch", "tanggal_proses",
		"jumlah_upload", "jumlah_proses", "jumlah_berhasil", "jumlah_gagal",
		"user_upload",
	} {
		require.Contains(t, pertama, kolom, "kolom grid %q harus ada di respons", kolom)
	}
}

func TestEntitasLainTidakMelihatDataEntitasIni(t *testing.T) {
	// Pemisahannya di tingkat koneksi, bukan penyaringan baris (ADR-0030 Opsi 1). Uji ini
	// yang membuktikannya benar-benar berlaku lewat header portal.
	p := newTestServer(t)

	_, isiASM := p.call(t, http.MethodGet, "/api/inbox-auto-claim", "ASM")
	require.NotEmpty(t, isiASM["batch"])

	_, isiASI := p.call(t, http.MethodGet, "/api/inbox-auto-claim", "ASI")
	require.Empty(t, isiASI["batch"])
}

func TestPaginasiDaftarBatchDiterjemahkanKeRespons(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim?sumber=aneka&halaman=1&ukuran=2", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)

	page, ok := content["paginasi"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), page["halaman"])
	require.Equal(t, float64(2), page["ukuran"])
	require.Equal(t, float64(4), page["total"])
	require.Equal(t, float64(2), page["total_halaman"])

	require.Len(t, content["batch"], 2)
}

func TestHalamanTidakSahDiperbaikiBukanDitolak(t *testing.T) {
	// `?halaman=abc` datang dari URL yang diketik tangan. Menolaknya dengan galat hanya
	// menampilkan layar rusak untuk kesalahan yang jelas maksudnya.
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim?halaman=abc&ukuran=-3", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)

	page := content["paginasi"].(map[string]any)
	require.Equal(t, float64(1), page["halaman"])
	require.Equal(t, float64(inboxautoclaim.DefaultPageSize), page["ukuran"])
}

func TestRincianBatchMenjawabBarisnya(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim/MFIN/2?sumber=aneka", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "MFIN", content["kode_perusahaan"])
	require.Equal(t, "2", content["batch"])
	require.Len(t, content["baris"], 4)

	baris := content["baris"].([]any)[0].(map[string]any)
	for _, kolom := range []string{
		"nomor_polis", "nilai_klaim", "hasil", "tanggal_proses", "mata_uang",
	} {
		require.Contains(t, baris, kolom, "kolom rincian %q harus ada di respons", kolom)
	}

	// Mata uangnya KODE, bukan id yang tersimpan di kolom CURRENCY.
	require.Equal(t, "IDR", baris["mata_uang"])
}

func TestRincianBatchYangTidakAdaMenjawab404(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim/MFIN/999", "ASM")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

func TestPenyaringHasilTidakDikenalMenjawab400(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim/MFIN/1?hasil=sukses", "ASM")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestDaftarPerusahaanTidakTertukarDenganRinciamBatch(t *testing.T) {
	// `/inbox-auto-claim/perusahaan` harus dilayani handler daftar perusahaan, bukan
	// terbaca sebagai `{kode}` pada rute rincian. Ini yang diuji urutan pendaftaran rute.
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim/perusahaan", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, content, "perusahaan")
	require.Len(t, content["perusahaan"], 4)
}

func TestEksporMenjawabBerkasCSVBukanJSON(t *testing.T) {
	p := newTestServer(t)

	request, err := http.NewRequest(http.MethodGet,
		p.server.URL+"/api/inbox-auto-claim/MFIN/2/ekspor?sumber=aneka&hasil=berhasil", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "text/csv; charset=utf-8", response.Header.Get("Content-Type"))
	require.Contains(t, response.Header.Get("Content-Disposition"), "attachment")
	require.Contains(t, response.Header.Get("Content-Disposition"), "Laporan Hasil Klaim")
	// Berkas ini memuat data nasabah; ia tidak boleh tersimpan di cache peramban.
	require.Equal(t, "no-store", response.Header.Get("Cache-Control"))

	content, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Contains(t, string(content), "Inisial,No Polis,No Klaim")
}

func TestEksporTanpaJenisHasilDitolak(t *testing.T) {
	// Layar lama hanya punya EXPORT BERHASIL dan EXPORT GAGAL; "ekspor semua" tidak
	// pernah ada dan tidak punya judul kolom yang dapat dirujuk.
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim/MFIN/2/ekspor?sumber=aneka", "ASM")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestUnggahanSahMenjawab201DanNomorBatchBaru(t *testing.T) {
	p := newTestServer(t)

	// Kode perusahaan TIDAK ada di berkas — ia diturunkan dari nomor polisnya.
	csv := csvJudul + "\n" +
		"0100120260500,500000.00,01/04/2026,02/04/2026,12002,REF-X,catatan\n"

	response, content := p.upload(t, "ASM", csv)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, float64(1), content["jumlah_baris"])
	require.Empty(t, content["ditolak"])

	batch := content["batch"].([]any)[0].(map[string]any)
	require.Equal(t, "MFIN", batch["kode_perusahaan"], "perusahaan diturunkan dari polis")
	require.Equal(t, "Mitra Finansial Nusantara", batch["nama_perusahaan"])
	require.Equal(t, "3", batch["batch"], "MFIN sudah punya batch 1 dan 2")
	require.Equal(t, float64(1), batch["jumlah_lolos"])
	require.Equal(t, float64(0), batch["jumlah_bertanda"])
}

func TestUnggahanSatuBerkasDapatMenghasilkanBeberapaBatch(t *testing.T) {
	// Akibat langsung dari perusahaan yang diturunkan per baris: satu berkas berisi polis
	// dari dua perusahaan menghasilkan DUA batch, tanpa pengunggah menyebutnya.
	p := newTestServer(t)

	csv := csvJudul + "\n" +
		"0100120260500,500000.00,01/04/2026,02/04/2026,12002,REF-X,\n" +
		"0200120260500,700000.00,01/04/2026,02/04/2026,12002,REF-Y,\n"

	response, content := p.upload(t, "ASM", csv)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Len(t, content["batch"], 2)

	kode := map[string]bool{}
	for _, mentah := range content["batch"].([]any) {
		kode[mentah.(map[string]any)["kode_perusahaan"].(string)] = true
	}
	require.True(t, kode["MFIN"])
	require.True(t, kode["BPRC"])
}

func TestBarisYangGagalPemeriksaanTetapTersimpanBesertaPesannya(t *testing.T) {
	// Perilaku InsertKlaimToTable_Other: baris yang gagal DISISIPKAN dengan pesannya,
	// bukan ditolak. Ia harus terlihat petugas di grid dan ikut keluar di ekspor GAGAL.
	p := newTestServer(t)

	// Polis ini ada di T_GENERAL tetapi tidak ada di JSON_POLIS.
	csv := csvJudul + "\n" +
		"0100120260999,500000.00,01/04/2026,02/04/2026,12002,REF-Z,\n"

	response, content := p.upload(t, "ASM", csv)
	require.Equal(t, http.StatusCreated, response.StatusCode, "berkasnya tidak ditolak")
	require.Empty(t, content["ditolak"], "barisnya tersimpan, bukan ditolak")

	batch := content["batch"].([]any)[0].(map[string]any)
	require.Equal(t, float64(0), batch["jumlah_lolos"])
	require.Equal(t, float64(1), batch["jumlah_bertanda"])

	_, rincian := p.call(t, http.MethodGet,
		"/api/inbox-auto-claim/MFIN/"+batch["batch"].(string)+"?sumber=aneka", "ASM")
	baris := rincian["baris"].([]any)[0].(map[string]any)
	require.Equal(t, inboxautoclaim.MessagePolicyNotFound, baris["keterangan"])
	require.Equal(t, "gagal", baris["hasil"])
}

func TestBarisTanpaPerusahaanDilaporkanBesertaNomorBarisnya(t *testing.T) {
	// Satu-satunya kegagalan yang membuat baris TIDAK disisipkan. Nomor baris dan nomor
	// polisnya ikut dilaporkan — tanpa keduanya, pengguna dengan berkas ratusan baris
	// hanya tahu "ada yang gagal".
	p := newTestServer(t)

	csv := csvJudul + "\n" +
		"0900120260500,500000.00,01/04/2026,02/04/2026,12002,,\n"

	response, content := p.upload(t, "ASM", csv)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, float64(0), content["jumlah_baris"])
	require.Empty(t, content["batch"])

	ditolak := content["ditolak"].([]any)
	require.Len(t, ditolak, 1)

	baris := ditolak[0].(map[string]any)
	require.Equal(t, float64(2), baris["baris"])
	require.Equal(t, "0900120260500", baris["nomor_polis"])
	require.Equal(t, inboxautoclaim.MessageReceiverNotFound, baris["pesan"])
}

func TestUnggahanMenandaiBarisDenganLoginPengunggah(t *testing.T) {
	p := newTestServer(t)

	csv := csvJudul + "\n" +
		"0200120260500,500000.00,01/04/2026,02/04/2026,12002,REF-Y,\n"
	_, content := p.upload(t, "ASM", csv)
	batch := content["batch"].([]any)[0].(map[string]any)["batch"].(string)

	// Tab disebut di KEDUA permintaan. Unggahan menulis ke tabel tab ANEKA, jadi
	// membacanya kembali dari tab bawaan akan mencari di tabel yang berbeda.
	_, isi := p.call(t, http.MethodGet, "/api/inbox-auto-claim?sumber=aneka&perusahaan=BPRC", "ASM")
	for _, mentah := range isi["batch"].([]any) {
		baris := mentah.(map[string]any)
		if baris["batch"] == batch {
			require.Equal(t, "adminpnc", baris["user_upload"])
			return
		}
	}
	t.Fatalf("batch %q yang baru diunggah tidak muncul di daftar", batch)
}

func TestUnggahanBercacatMenjawab422BesertaSeluruhPelanggaran(t *testing.T) {
	p := newTestServer(t)

	csv := csvJudul + "\n" +
		",500000.00,01/04/2026,02/04/2026,12002,,\n" +
		"0100120260500,,01/04/2026,02/04/2026,12002,,\n"

	response, content := p.upload(t, "ASM", csv)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])

	detail, ok := content["detail"].([]any)
	require.True(t, ok)
	require.Len(t, detail, 2, "kedua pelanggaran dikirim sekaligus")

	// Kunci pelanggarannya bernama `kolom`, mengikuti masterstatusprogres.
	require.Contains(t, detail[0].(map[string]any), "kolom")
}

func TestUnggahanTanpaPortalDitolakSebelumBerkasDibaca(t *testing.T) {
	p := newTestServer(t)

	response, content := p.upload(t, "",
		csvJudul+"\n0100120260500,1.00,01/04/2026,02/04/2026,,,\n")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_disebut", content["kode"])
}

func TestUnggahanTanpaBerkasDitolak(t *testing.T) {
	p := newTestServer(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("bukanberkas", "isi"))
	require.NoError(t, writer.Close())

	request, err := http.NewRequest(http.MethodPost, p.server.URL+"/api/inbox-auto-claim/unggah?sumber=aneka", &body)
	require.NoError(t, err)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestFormatUnggahanDapatDibacaTanpaMemilihPortal(t *testing.T) {
	// Bentuk berkas sama untuk setiap entitas, dan tidak satu baris data entitas pun
	// dibacanya. Menuntut portal akan membuat petunjuk format gagal justru saat pengguna
	// belum memilih entitas.
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/inbox-auto-claim/format-unggahan", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, content["kolom_wajib"], len(inboxautoclaim.RequiredUploadColumn))
	require.Equal(t, float64(inboxautoclaim.MaxUploadRow), content["batas_baris"])
}
