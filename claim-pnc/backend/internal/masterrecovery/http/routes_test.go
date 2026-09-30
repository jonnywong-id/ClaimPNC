package masterrecoveryhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth/provider"
	authmemory "claim-pnc/internal/auth/repo/memory"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/masterrecovery"
	"claim-pnc/internal/masterrecovery/repo/memory"
	masterrecoveryusecase "claim-pnc/internal/masterrecovery/usecase"
	"claim-pnc/internal/masterrecovery/virtualaccount"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterrecoveryhttp "claim-pnc/internal/masterrecovery/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const route = "/api/master/recovery"

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang sah
// ditolak, dan bahwa isian yang menjadi milik server tidak dapat dikirim klien. Menguji
// handler secara terpisah tidak dapat membuktikan satu pun di antaranya.
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
	asi := memory.NewRepo(nil, nil)

	recoveryService, err := masterrecoveryusecase.NewService(masterrecoveryusecase.Options{
		RepoSelector: func(alias string) (masterrecovery.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Issuer: virtualaccount.NewFake(),
		Now:    func() time.Time { return time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC) },
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

	handler, err := masterrecoveryhttp.NewHandler(masterrecoveryhttp.Options{
		Service: recoveryService,
		Caller: func(ctx context.Context) (masterrecoveryhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterrecoveryhttp.Caller{}, false
			}
			return masterrecoveryhttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
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
				masterrecoveryhttp.Mount(protected, handler, portalDeps)
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

// upload mengirim satu berkas sebagai multipart, seperti yang dilakukan peramban.
func (p *testServer) upload(t *testing.T, path, portalAlias, filename, content string) (*http.Response, map[string]any) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("berkas", filename)
	require.NoError(t, err)
	_, err = io.WriteString(part, content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request, err := http.NewRequest(http.MethodPost, p.server.URL+path, &body)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	answer := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&answer)
	return response, answer
}

// uploadWithType mengirim berkas beserta jenis isinya yang DISEBUT EKSPLISIT.
//
// `upload` di atas menyerahkan penentuan jenis kepada `CreateFormFile`, yang selalu
// menulis `application/octet-stream`. Untuk menguji perbedaan perlakuan antarjenis, jenis
// itu harus dapat ditetapkan — dan begitulah peramban mengirimnya.
func (p *testServer) uploadWithType(
	t *testing.T,
	path, portalAlias, filename, mimeType, content string,
) (*http.Response, map[string]any) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="berkas"; filename=%q`, filename))
	header.Set("Content-Type", mimeType)

	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = io.WriteString(part, content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request, err := http.NewRequest(http.MethodPost, p.server.URL+path, &body)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	answer := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&answer)
	return response, answer
}

// isianLengkap adalah badan simpan yang lolos seluruh validasi, dipakai sebagai dasar
// beberapa uji di bawah.
const isianLengkap = `{
  "nama_principal":"PT CONTOH PENJAMINAN NUSANTARA",
  "client_id":"CONTOH-PRINCIPAL-001",
  "nomor_virtual_account":"0000000000000001",
  "tahun":"2026",
  "nilai_klaim":160000,
  "pembayaran_sebelumnya":0,
  "pembayaran":5000,
  "keterangan":"pengembalian sebagian",
  "posisi_kasus":"dalam proses",
  "nomor_polis":"",
  "id_dokumen":"",
  "id_log_layanan":"",
  "baris_klaim":[]
}`

// ── Sesi dan portal ─────────────────────────────────────────────────────────────

// TestTanpaSesiDitolak memastikan rutenya berada di balik sesi. Tanpa token, permintaan
// ditolak sebelum menyentuh pemeriksaan portal maupun penyimpanan.
func TestTanpaSesiDitolak(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	for _, jalur := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, route + "/form"},
		{http.MethodGet, route + "/principal"},
		{http.MethodPost, route + "/"},
		{http.MethodPost, route + "/virtual-account"},
	} {
		response, _ := p.call(t, jalur.method, jalur.path, "ASM", "")
		require.Equal(t, http.StatusUnauthorized, response.StatusCode, jalur.path)
	}
}

// TestTanpaPortalDitolak mengunci `TKT-F6-002`: permintaan tanpa portal ditolak, TIDAK
// PERNAH dialihkan ke portal utama sebagai cadangan (`R-20`).
func TestTanpaPortalDitolak(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"/form", "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_disebut", body["kode"])
}

// TestPortalBelumSiapDibedakanDariPortalTidakDikenal memastikan pesannya menuntun ke
// tindak lanjut yang benar — keduanya menuntut hal yang berbeda.
func TestPortalBelumSiapDibedakanDariPortalTidakDikenal(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"/form", "SMAS", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, portalhttp.CodeNotReady, body["kode"])

	// Portal yang tidak ada di daftar adalah kesalahan PEMANGGIL — 400, bukan 503.
	// Perbedaannya penting: yang di atas pekerjaan tim infrastruktur, yang ini salah
	// pilih di sisi klien.
	response, body = p.call(t, http.MethodGet, route+"/form", "TIDAKADA", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, portalhttp.CodeUnknown, body["kode"])
}

// TestDataTidakBocorAntarEntitas adalah uji terpenting di berkas ini.
//
// Nomor virtual account satu badan hukum yang terbaca dari badan hukum lain berarti dana
// dapat diarahkan ke rekening yang keliru — akibat `R-20` yang paling berat di modul ini.
func TestDataTidakBocorAntarEntitas(t *testing.T) {
	p := newTestServer(t)

	_, isiASM := p.call(t, http.MethodGet, route+"/principal", "ASM", "")
	require.Equal(t, float64(2), isiASM["total"])
	require.Equal(t, "ASM", isiASM["portal"])

	_, isiASI := p.call(t, http.MethodGet, route+"/principal", "ASI", "")
	require.Equal(t, float64(0), isiASI["total"], "entitas lain tidak melihat principal ASM")
	require.Equal(t, "ASI", isiASI["portal"])
}

// ── Aturan yang dijaga transport ────────────────────────────────────────────────

// TestSimpanMengembalikanNomorBatchDariServer memastikan nomor tidak pernah datang dari
// klien.
func TestSimpanMengembalikanNomorBatchDariServer(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route+"/", "ASM", isianLengkap)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	recovery := body["recovery"].(map[string]any)
	require.Equal(t, float64(1), recovery["nomor_batch"])
	require.Equal(t, float64(155000), recovery["sisa"], "sisa dihitung server")
	require.Equal(t, "ASM", body["portal"])
}

// TestIsianMilikServerDitolakBilaDikirimKlien mengunci DisallowUnknownFields.
//
// Ketiganya bukan sekadar "field asing": masing-masing adalah keputusan yang harus tetap
// di server. Mengabaikannya diam-diam akan membuat klien yang keliru terus mengirimnya
// tanpa pernah tahu bahwa kirimannya tidak berarti apa-apa.
func TestIsianMilikServerDitolakBilaDikirimKlien(t *testing.T) {
	p := newTestServer(t)

	for _, isian := range []string{
		`{"nama_principal":"PT X","tahun":"2026","keterangan":"k","posisi_kasus":"p","nomor_batch":99}`,
		`{"nama_principal":"PT X","tahun":"2026","keterangan":"k","posisi_kasus":"p","sisa":123}`,
		`{"nama_principal":"PT X","tahun":"2026","keterangan":"k","posisi_kasus":"p","id_lini_bisnis":"01"}`,
		`{"nama_principal":"PT X","tahun":"2026","keterangan":"k","posisi_kasus":"p","dicatat_oleh":"ORANGLAIN"}`,
	} {
		response, body := p.call(t, http.MethodPost, route+"/", "ASM", isian)
		require.Equal(t, http.StatusBadRequest, response.StatusCode, isian)
		require.Equal(t, "permintaan_cacat", body["kode"])
	}
}

// TestValidasiMengembalikanSeluruhPelanggaranDenganNamaFieldnya mengunci perbaikan
// terhadap satu pesan "Wajib ISI semua field" di sistem lama.
func TestValidasiMengembalikanSeluruhPelanggaranDenganNamaFieldnya(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route+"/", "ASM", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", body["kode"])

	detail, ada := body["detail"].([]any)
	require.True(t, ada, "detail pelanggaran dikirim")
	require.GreaterOrEqual(t, len(detail), 4)

	field := map[string]bool{}
	for _, baris := range detail {
		field[baris.(map[string]any)["field"].(string)] = true
	}
	require.True(t, field["nama_principal"])
	require.True(t, field["tahun"])
	require.True(t, field["keterangan"])
	require.True(t, field["posisi_kasus"])
}

// TestDicatatOlehDiambilDariSesi mengunci pengisian kolom USERNAME.
//
// `D-59` menetapkan tidak ada pemisahan tugas formal, sehingga jejak siapa-mengerjakan-apa
// adalah satu-satunya kontrol pengimbang yang tersisa.
func TestDicatatOlehDiambilDariSesi(t *testing.T) {
	p := newTestServer(t)

	_, body := p.call(t, http.MethodPost, route+"/", "ASM", isianLengkap)
	recovery := body["recovery"].(map[string]any)
	require.NotEmpty(t, recovery["dicatat_oleh"])

	tersimpan := p.asm.Saved()
	require.Len(t, tersimpan, 1)
	require.NotEmpty(t, tersimpan[0].InputBy)
}

// TestFormMengirimNomorBatchDanDaftarTahun mengunci bekal awal layar dalam SATU permintaan
// (`10-API-STRATEGY.md` §1).
func TestFormMengirimNomorBatchDanDaftarTahun(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"/form", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(1), body["nomor_batch_perkiraan"])

	tahun := body["tahun"].([]any)
	require.NotEmpty(t, tahun)
	// Disalin apa adanya dari Data Transform GetListYear: 2015..2025, urutan menaik.
	require.Equal(t, "2015", tahun[0])
	require.Len(t, tahun, 11)
}

// TestPolisTidakDitemukanDijawab404 memastikan pencarian polis dapat dipakai layar untuk
// memperingatkan SEBELUM menyimpan.
func TestPolisTidakDitemukanDijawab404(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"/polis/POLIS-TIDAK-ADA", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "polis_tidak_ditemukan", body["kode"])

	response, body = p.call(t, http.MethodGet, route+"/polis/CONTOH-POLIS-0001", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "01", body["id_lini_bisnis"])
	require.Equal(t, "MO0001", body["id_marketing"])
}

// TestVADipakaiUlangMenjawab200SedangkanYangBaruMenjawab201 mengunci pembedaan yang
// membuat layar dapat memilih nada pemberitahuannya tanpa mencocokkan teks pesan.
func TestVADipakaiUlangMenjawab200SedangkanYangBaruMenjawab201(t *testing.T) {
	p := newTestServer(t)

	baru := `{"client_id":"PRINCIPAL-BARU","nama_principal":"PT BARU","email_inputor_va":"petugas@example.invalid"}`
	response, body := p.call(t, http.MethodPost, route+"/virtual-account", "ASM", baru)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, false, body["dipakai_ulang"])
	nomor := body["nomor_virtual_account"]
	require.NotEmpty(t, nomor)

	response, body = p.call(t, http.MethodPost, route+"/virtual-account", "ASM", baru)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, true, body["dipakai_ulang"])
	require.Equal(t, nomor, body["nomor_virtual_account"])
}

// TestPanelVAMenolakIsianTidakLengkap memastikan pengetatan terhadap layar lama benar-benar
// ditegakkan lewat HTTP, bukan hanya di fungsi pemeriksa.
func TestPanelVAMenolakIsianTidakLengkap(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route+"/virtual-account", "ASM", `{"client_id":"","nama_principal":"","email_inputor_va":""}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", body["kode"])
}

// ── Unggahan ────────────────────────────────────────────────────────────────────

// TestBuktiBayarTersimpanDanMengembalikanPenandanya mengunci alur unggah → id → simpan.
func TestBuktiBayarTersimpanDanMengembalikanPenandanya(t *testing.T) {
	p := newTestServer(t)

	response, body := p.upload(t, route+"/bukti-bayar", "ASM", "bukti-transfer.pdf", "%PDF-1.4 contoh")
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.NotEmpty(t, body["id_dokumen"])
	require.Equal(t, "bukti-transfer.pdf", body["nama_berkas"])
}

// TestBuktiBayarKosongDitolak memastikan berkas tanpa isi tidak tersimpan sebagai lampiran
// yang tampak ada tetapi tidak dapat dibuka siapa pun.
func TestBuktiBayarKosongDitolak(t *testing.T) {
	p := newTestServer(t)

	response, body := p.upload(t, route+"/bukti-bayar", "ASM", "kosong.pdf", "")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", body["kode"])
}

// TestBerkasKlaimDibacaDanDijumlahkan mengunci unggahan CSV beserta penjumlahan yang tidak
// dimiliki sistem lama.
func TestBerkasKlaimDibacaDanDijumlahkan(t *testing.T) {
	p := newTestServer(t)

	response, body := p.upload(t, route+"/baris-klaim", "ASM", "klaim.csv",
		"No Polis;Nilai Klaim\nPOL-1;1000\nPOL-2;2000\n")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(2), body["total"])
	require.Equal(t, float64(3000), body["jumlah_nilai_klaim"])
}

// TestBerkasKlaimCacatDilaporkanBarisPerBaris memastikan berkas dengan satu baris cacat
// tetap berguna, dan petugas tahu persis baris mana yang tidak ikut.
func TestBerkasKlaimCacatDilaporkanBarisPerBaris(t *testing.T) {
	p := newTestServer(t)

	_, body := p.upload(t, route+"/baris-klaim", "ASM", "klaim.csv",
		"No Polis;Nilai Klaim\nPOL-1;1000\nPOL-2;bukanangka\n")
	require.Equal(t, float64(1), body["total"])

	ditolak := body["baris_ditolak"].([]any)
	require.Len(t, ditolak, 1)
	require.Contains(t, ditolak[0].(map[string]any)["pesan"], "Baris 3")
}

// TestFormatUnggahanDiunduhSebagaiCSV memastikan tautan "Format File" menghasilkan berkas,
// bukan JSON yang tampil di peramban.
func TestFormatUnggahanDiunduhSebagaiCSV(t *testing.T) {
	p := newTestServer(t)

	request, err := http.NewRequest(http.MethodGet, p.server.URL+route+"/format-unggahan", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, response.Header.Get("Content-Type"), "text/csv")
	require.Contains(t, response.Header.Get("Content-Disposition"), "attachment")

	content, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Contains(t, string(content), "PolicyNo")
}

// ── Aksi yang sengaja TIDAK ada ─────────────────────────────────────────────────

// TestTidakAdaUbahMaupunHapus mengunci ketiadaan yang berdiri di atas BUKTI.
//
// Dahulu test ini juga menuntut GET daftar tidak ada. Tuntutan itu DICABUT 2026-09-29:
// grid Outstanding memang ada di layar lama, dan kesimpulan bahwa ia tidak ada berdiri di
// atas ketiadaan kueri di export yang ternyata tidak lengkap (`R-16`).
//
// Yang tersisa di sini bukan kesimpulan dari ketiadaan, melainkan dari isi:
// `INSERTMASTERRECOVERYKLAIM.prc` dibaca utuh dan hanya mengenal INSERT, dan tidak ada
// satu pun UPDATE maupun DELETE atas tabel ini di seluruh export.
func TestTidakAdaUbahMaupunHapus(t *testing.T) {
	p := newTestServer(t)

	for _, jalur := range []struct {
		method string
		path   string
	}{
		{http.MethodPut, route + "/1"},
		{http.MethodDelete, route + "/1"},
	} {
		response, _ := p.call(t, jalur.method, jalur.path, "ASM", "")
		require.NotEqual(t, http.StatusOK, response.StatusCode, jalur.method+" "+jalur.path)
		require.NotEqual(t, http.StatusCreated, response.StatusCode, jalur.method+" "+jalur.path)
	}
}

// TestDaftarOutstandingMengembalikanBatchTersimpan membuktikan tab Outstanding benar-benar
// membaca dari penyimpanan, bukan dari keadaan layar.
//
// Ini yang membedakannya dari daftar sesi yang digantikannya: batch yang disimpan harus
// tetap terbaca oleh permintaan BARU, sebagaimana layar Pega menampilkan batch yang
// dicatat orang lain di hari sebelumnya.
func TestDaftarOutstandingMengembalikanBatchTersimpan(t *testing.T) {
	p := newTestServer(t)

	simpan, _ := p.call(t, http.MethodPost, route+"/", "ASM", `{
		"nama_principal":"PT CONTOH PENJAMINAN NUSANTARA",
		"client_id":"CONTOH-PRINCIPAL-001",
		"nomor_virtual_account":"0000000000000001",
		"tahun":"2025",
		"nilai_klaim":160000,
		"pembayaran_sebelumnya":7000,
		"pembayaran":100000,
		"keterangan":"pengembalian sebagian",
		"posisi_kasus":"dalam proses"
	}`)
	require.Equal(t, http.StatusCreated, simpan.StatusCode)

	response, body := p.call(t, http.MethodGet, route+"/", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(1), body["total"])
	require.Equal(t, "ASM", body["portal"])

	// Baris LUAR: satu per principal.
	rows, ok := body["principal"].([]any)
	require.True(t, ok, "kunci principal harus berupa senarai")
	require.Len(t, rows, 1)

	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "PT CONTOH PENJAMINAN NUSANTARA", row["nama_principal"])
	require.Equal(t, float64(160000), row["nilai_klaim"])

	// Sisa dihitung server: pembayaran sebelumnya terisi, jadi 160.000 − 7.000.
	require.Equal(t, float64(153000), row["sisa"])

	// Baris DALAM: riwayat batch principal itu.
	batch, ok := row["batch"].([]any)
	require.True(t, ok, "baris luar harus membawa riwayat batch-nya")
	require.Len(t, batch, 1)
	require.Equal(t, float64(1), batch[0].(map[string]any)["batch"])
}

// TestDaftarOutstandingMengelompokPerPrincipal mengunci bentuk yang dibaca dari layar Pega
// yang berjalan (2026-09-29): SATU baris luar per principal, isinya seluruh batch,
// terurut dari yang paling lama, dan angka baris luar adalah angka batch TERAKHIR.
//
// Angka terakhir, bukan jumlah — itu yang membedakannya dan itu yang mudah salah. Tiga
// pembayaran di bawah berjumlah 107.000; yang harus tampil di baris luar adalah 100.000.
func TestDaftarOutstandingMengelompokPerPrincipal(t *testing.T) {
	p := newTestServer(t)

	for _, isian := range []struct{ sebelum, bayar, sisa int }{
		{0, 5000, 155000},
		{5000, 2000, 155000},
		{7000, 100000, 153000},
	} {
		simpan, _ := p.call(t, http.MethodPost, route+"/", "ASM", fmt.Sprintf(`{
			"nama_principal":"PT CONTOH PENJAMINAN NUSANTARA",
			"client_id":"CONTOH-PRINCIPAL-001",
			"nomor_virtual_account":"0000000000000001",
			"tahun":"2025",
			"nilai_klaim":160000,
			"pembayaran_sebelumnya":%d,
			"pembayaran":%d,
			"keterangan":"pengembalian sebagian",
			"posisi_kasus":"dalam proses"
		}`, isian.sebelum, isian.bayar))
		require.Equal(t, http.StatusCreated, simpan.StatusCode)
	}

	response, body := p.call(t, http.MethodGet, route+"/", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	// Tiga batch, tetapi SATU baris luar — dan totalnya menghitung principal, bukan batch.
	require.Equal(t, float64(1), body["total"])
	rows := body["principal"].([]any)
	require.Len(t, rows, 1)

	row := rows[0].(map[string]any)
	require.Equal(t, float64(7000), row["pembayaran_sebelumnya"])
	require.Equal(t, float64(100000), row["pembayaran"], "angka baris luar dari batch TERAKHIR, bukan dijumlahkan")
	require.Equal(t, float64(153000), row["sisa"])

	batch := row["batch"].([]any)
	require.Len(t, batch, 3)

	// Dari yang paling lama — mengikuti grid dalam layar lama.
	urutan := make([]float64, 0, 3)
	for _, b := range batch {
		urutan = append(urutan, b.(map[string]any)["batch"].(float64))
	}
	require.Equal(t, []float64{1, 2, 3}, urutan)

	// Kolom yang hanya ada di grid dalam ikut terbawa.
	pertama := batch[0].(map[string]any)
	require.Contains(t, pertama, "tanggal_input")
	require.Contains(t, pertama, "no_hpll")
	require.Contains(t, pertama, "id_dokumen")
}

// TestViewDocumentMengalirkanIsinya mengunci tombol View Document pada grid dalam.
//
// Yang diperiksa bukan hanya statusnya, melainkan bahwa yang keluar adalah ISI berkasnya
// apa adanya — bukan JSON ber-base64 seperti jalur lama, dan bukan berkas kosong.
func TestViewDocumentMengalirkanIsinya(t *testing.T) {
	p := newTestServer(t)

	unggah, body := p.upload(t, route+"/bukti-bayar", "ASM", "bukti-transfer.pdf", "%PDF-1.4 contoh")
	require.Equal(t, http.StatusCreated, unggah.StatusCode)
	id, ok := body["id_dokumen"].(string)
	require.True(t, ok)
	require.NotEmpty(t, id)

	request, err := http.NewRequest(http.MethodGet, p.server.URL+route+"/bukti-bayar/"+id, nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	require.Equal(t, http.StatusOK, response.StatusCode)
	// PDF ada di daftar aman, jadi ia boleh DITAMPILKAN.
	require.Contains(t, response.Header.Get("Content-Disposition"), "inline")
	require.Contains(t, response.Header.Get("Content-Disposition"), "bukti-transfer.pdf")
	require.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"))

	isi, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "%PDF-1.4 contoh", string(isi))
}

// TestBerkasYangTidakAmanDiunduhBukanDitampilkan mengunci perbaikan 2026-09-29.
//
// Versi pertama menyajikan SETIAP lampiran `inline` dengan jenis isi apa adanya dari
// basis data. Akibatnya dua, dan keduanya nyata:
//
//   - XLSX di portal ASM tergambar sebagai berhalaman-halaman karakter acak.
//   - Satu baris ber-ATTACHMIMETYPE `text/html` akan dijalankan peramban sebagai halaman
//     di origin aplikasi ini.
//
// Uji ini menjaga keduanya tertutup: yang tidak ada di daftar aman WAJIB `attachment`.
func TestBerkasYangTidakAmanDiunduhBukanDitampilkan(t *testing.T) {
	p := newTestServer(t)

	for _, berkas := range []struct {
		nama  string
		jenis string
		isi   string
	}{
		// Jenis Office — tidak dapat digambar peramban.
		{"rekap.xlsx", "application/vnd.ms-excel", "PK\x03\x04 contoh"},
		// Yang paling berbahaya: HTML yang dapat menjalankan skrip.
		{"jebakan.html", "text/html", "<script>alert(1)</script>"},
		// SVG sengaja TIDAK masuk daftar aman meski ia gambar.
		{"gambar.svg", "image/svg+xml", "<svg xmlns='http://www.w3.org/2000/svg'/>"},
	} {
		unggah, body := p.uploadWithType(t, route+"/bukti-bayar", "ASM", berkas.nama, berkas.jenis, berkas.isi)
		require.Equal(t, http.StatusCreated, unggah.StatusCode, berkas.nama)
		id := body["id_dokumen"].(string)

		request, err := http.NewRequest(http.MethodGet, p.server.URL+route+"/bukti-bayar/"+id, nil)
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer "+p.token)
		request.Header.Set(portalhttp.HeaderPortal, "ASM")

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		defer func() { _ = response.Body.Close() }()

		require.Equal(t, http.StatusOK, response.StatusCode, berkas.nama)
		require.Contains(t, response.Header.Get("Content-Disposition"), "attachment", berkas.nama)
		require.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"), berkas.nama)
	}
}

// TestViewDocumentYangTidakAdaMenjawab404 memastikan penanda yang menunjuk lampiran yang
// tidak ada tidak menghasilkan berkas kosong yang tampak berhasil diunduh.
func TestViewDocumentYangTidakAdaMenjawab404(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"/bukti-bayar/TIDAK-ADA", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "bukti_bayar_tidak_ditemukan", body["kode"])
}

// TestDaftarOutstandingMenolakPortalYangBelumSiap adalah pengaman `R-20` pada jalur BACA.
//
// Yang dibuktikan di sini bukan "daftarnya kosong", melainkan sesuatu yang lebih tegas:
// permintaan atas portal yang koneksinya belum hidup DITOLAK, dan tidak pernah dilayani
// repo portal lain sebagai cadangan. Daftar kosong masih dapat berarti "portalnya benar,
// datanya memang belum ada"; penolakan tidak bermakna ganda.
//
// Batch sengaja disimpan lebih dulu di ASM supaya ada sesuatu yang BISA bocor — tanpa itu,
// jawaban kosong akan lulus uji ini tanpa membuktikan apa pun.
func TestDaftarOutstandingMenolakPortalYangBelumSiap(t *testing.T) {
	p := newTestServer(t)

	simpan, _ := p.call(t, http.MethodPost, route+"/", "ASM", `{
		"nama_principal":"HANYA DI ASM",
		"client_id":"CONTOH-PRINCIPAL-001",
		"nomor_virtual_account":"0000000000000001",
		"tahun":"2025",
		"nilai_klaim":50000,
		"pembayaran":10000,
		"keterangan":"pengembalian sebagian",
		"posisi_kasus":"dalam proses"
	}`)
	require.Equal(t, http.StatusCreated, simpan.StatusCode)

	response, body := p.call(t, http.MethodGet, route+"/", "SIMASNET", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.NotContains(t, body, "batch", "jawaban penolakan tidak boleh membawa satu baris pun")

	// Dan yang tersimpan tetap terbaca di portal yang benar — supaya penolakan di atas
	// tidak lolos hanya karena penyimpanannya gagal diam-diam.
	lanjut, isi := p.call(t, http.MethodGet, route+"/", "ASM", "")
	require.Equal(t, http.StatusOK, lanjut.StatusCode)
	require.Equal(t, float64(1), isi["total"])
}
