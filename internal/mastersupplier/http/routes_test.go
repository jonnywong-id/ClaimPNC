package mastersupplierhttp_test

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
	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersupplier/repo/memory"
	mastersupplierusecase "claim-pnc/internal/mastersupplier/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	mastersupplierhttp "claim-pnc/internal/mastersupplier/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Ketiga baris contoh pada memory.SampleList.
const (
	activeHeavyID = "0100000000001" // aktif, rekanan, Heavy Equipment
	activeOtherID = "0100000000002" // aktif, bukan HE
	inactiveID    = "0100000000003" // TIDAK aktif
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang sah
// ditolak, dan bahwa satu entitas tidak pernah melihat data entitas lain. Menguji handler
// secara terpisah tidak dapat membuktikan ketiganya.
type testServer struct {
	server *httptest.Server
	token  string

	// asi tidak diberi satu baris master pun; ia yang membuktikan pemisahan
	// antarentitas (ADR-0030, R-20).
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewSampleRepo()
	asi := memory.NewRepo(memory.Options{Banks: memory.SampleBanks()})

	supplierService, err := mastersupplierusecase.NewService(mastersupplierusecase.Options{
		RepoSelector: func(alias string) (mastersupplier.Store, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Clock: clock.FixedAt(time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	writeResponse := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	// Dirantai sama seperti cmd/claimpnc: galat portal dipetakan modul portal, sisanya
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan portal
	// akan lulus di sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(authhttp.WriteError(logger), writeResponse)

	handler, err := mastersupplierhttp.NewHandler(mastersupplierhttp.Options{
		Service: supplierService,
		Caller: func(ctx context.Context) (mastersupplierhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return mastersupplierhttp.Caller{}, false
			}
			return mastersupplierhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    mastersupplierhttp.ErrorWriter(writeError),
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
				protected.Use(authhttp.Authenticate(authService, authhttp.ErrorWriter(writeError)))
				mastersupplierhttp.Mount(protected, handler, portalDeps)
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

	body := strings.NewReader(`{"nama_pengguna":"` + loginName + `","kata_sandi":"rahasia123"}`)
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

// rows mengambil senarai supplier dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["supplier"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai supplier: %v", content)
	return list
}

// one mengambil satu supplier dari badan respons.
func one(t *testing.T, content map[string]any) map[string]any {
	t.Helper()

	item, ok := content["supplier"].(map[string]any)
	require.Truef(t, ok, "badan respons tidak memuat satu supplier: %v", content)
	return item
}

// validBody adalah badan permintaan yang lolos seluruh pemeriksaan.
func validBody(name string) string {
	return `{
		"nama":"` + name + `",
		"alamat":"Jalan Contoh Nomor 9",
		"kota":"Jakarta Pusat",
		"nama_cabang":"Cabang Contoh Pusat",
		"kode_pos":"10110",
		"negara":"Indonesia",
		"telepon":"021-0000009",
		"fax":"",
		"email":"",
		"npwp":"",
		"contact_person":"Narahubung Uji",
		"status_rekanan":"1",
		"status_supply":"1",
		"term_of_payment":"30",
		"term_of_delivery":"7",
		"keterangan":"Pengajuan baru.",
		"bank":"Bank Contoh Satu",
		"no_account":"9000000009",
		"account_name":"",
		"bank_branch":"",
		"jenis_supplier":"1",
		"status_aktif":"1",
		"status_autopayment":"0"
	}`
}

// ============================================================================
// Sesi dan portal
// ============================================================================

// Seluruh rute berada di balik sesi. Permintaan tanpa token ditolak 401 — termasuk kelima
// rute lookup, yang isinya pun dibaca dari basis data entitas.
func TestEveryRouteRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/master/supplier"},
		{http.MethodGet, "/api/master/supplier/" + activeHeavyID},
		{http.MethodPost, "/api/master/supplier"},
		{http.MethodPut, "/api/master/supplier/" + activeHeavyID},
		{http.MethodGet, "/api/master/supplier/cabang"},
		{http.MethodGet, "/api/master/supplier/kota?cari=jak"},
		{http.MethodGet, "/api/master/supplier/negara"},
		{http.MethodGet, "/api/master/supplier/bank"},
		{http.MethodGet, "/api/master/supplier/sandi"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			response, _ := p.call(t, route.method, route.path, "ASM", "")
			require.Equal(t, http.StatusUnauthorized, response.StatusCode)
		})
	}
}

// Permintaan TANPA portal ditolak, bukan dijawab dengan data portal utama.
//
// Jatuh ke portal bawaan berarti menyajikan data satu badan hukum kepada permintaan yang
// tidak menyebut badan hukum mana pun — persis kebocoran yang dicegah R-20.
func TestPortalIsMandatory(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/supplier", "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, portalhttp.CodeNotStated, content["kode"])
}

// Portal yang tidak dikenal dijawab 400 dengan kode yang BERBEDA dari portal yang tidak
// disebut. Layar membedakan keduanya: yang pertama berarti pilihannya salah, yang kedua
// berarti pengguna belum memilih sama sekali.
func TestUnknownPortalIsRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/supplier", "ENTAH", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, portalhttp.CodeUnknown, content["kode"])
}

// Dua entitas melihat data yang berbeda. ASI tidak diberi satu baris master pun.
func TestEntitiesAreSeparated(t *testing.T) {
	p := newTestServer(t)

	_, asm := p.call(t, http.MethodGet, "/api/master/supplier", "ASM", "")
	require.Len(t, rows(t, asm), 3)
	require.Equal(t, "ASM", asm["portal"])

	_, asi := p.call(t, http.MethodGet, "/api/master/supplier", "ASI", "")
	require.Empty(t, rows(t, asi))
	require.Equal(t, "ASI", asi["portal"])
}

// Supplier yang ditambahkan di satu entitas tidak pernah muncul di entitas lain.
func TestCreateStaysWithinItsEntity(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodPost, "/api/master/supplier", "ASI", validBody("Supplier ASI"))
	require.Equal(t, http.StatusCreated, response.StatusCode)

	_, asm := p.call(t, http.MethodGet, "/api/master/supplier", "ASM", "")
	require.Len(t, rows(t, asm), 3, "entitas lain tidak boleh ikut bertambah")

	_, asi := p.call(t, http.MethodGet, "/api/master/supplier", "ASI", "")
	require.Len(t, rows(t, asi), 1)
}

// ============================================================================
// Daftar dan pengambilan
// ============================================================================

func TestListReturnsRows(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/supplier", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, rows(t, content), 3)
}

func TestListFiltersByKeyword(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, "/api/master/supplier?cari=bandung", "ASM", "")
	require.Len(t, rows(t, content), 1)
}

func TestGetReturnsSingleRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/supplier/"+activeHeavyID, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	item := one(t, content)
	require.Equal(t, activeHeavyID, item["id_supplier"])
	require.Equal(t, "Supplier Contoh Utama", item["nama"])
	require.Equal(t, true, item["heavy_equipment"])
	require.Equal(t, true, item["aktif"])
}

func TestGetUnknownRowIs404(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/supplier/tidak-ada", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// ============================================================================
// Penambahan
// ============================================================================

// Supplier baru lahir TIDAK aktif, berapa pun yang dipilih di layar — dan responsnya
// menyebutkan keduanya supaya layar dapat menjelaskan kenapa.
func TestCreateStartsInactive(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/supplier", "ASM",
		validBody("Supplier Contoh Baru"))
	require.Equal(t, http.StatusCreated, response.StatusCode)

	item := one(t, content)
	require.Equal(t, "1", item["status_aktif"], "yang DIMINTA")
	require.Equal(t, "0", item["status_aktif_berlaku"], "yang BERLAKU")
	require.Equal(t, false, item["aktif"])
	require.NotEmpty(t, item["id_supplier"])
}

// Jejak pemanggil benar-benar tersimpan; ia kunci USERKLAIMID di dalam dokumen.
func TestCreateRecordsCaller(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodPost, "/api/master/supplier", "ASM",
		validBody("Supplier Contoh Baru"))

	item := one(t, content)
	require.Equal(t, loginName, item["diubah_oleh"])
	require.Equal(t, "20/09/2026", item["diubah_pada"])
}

// Penambahan menyisipkan satu baris ke antrean persetujuan.
func TestCreateRequestsApproval(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodPost, "/api/master/supplier", "ASM",
		validBody("Supplier Contoh Baru"))

	queue := p.asm.Approval()
	require.Len(t, queue, 1)
	require.Equal(t, one(t, content)["id_supplier"], queue[0].SupplierID)
	require.Equal(t, mastersupplier.PositionRequested, queue[0].Position)
	require.Equal(t, loginName, queue[0].RequestedBy)
	require.Equal(t, "Pengajuan baru.", queue[0].Reason,
		"ALASAN_REQ diisi KETERANGAN supplier")
	require.Contains(t, queue[0].ID, queue[0].SupplierID,
		"PROTEKSI_ID menyebut supplier-nya; lihat ComposeApprovalID")
}

// Isian yang tidak lengkap dijawab 422 dengan SELURUH pelanggarannya sekaligus.
func TestCreateRejectsIncompleteInput(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/supplier", "ASM", `{
		"nama":"","alamat":"","kota":"","nama_cabang":"","kode_pos":"","negara":"",
		"telepon":"","fax":"","email":"","npwp":"","contact_person":"",
		"status_rekanan":"","status_supply":"","term_of_payment":"","term_of_delivery":"",
		"keterangan":"","bank":"","no_account":"","account_name":"","bank_branch":"",
		"jenis_supplier":"","status_aktif":"","status_autopayment":""
	}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])

	detail, ok := content["detail"].([]any)
	require.True(t, ok)
	require.Len(t, detail, 15, "kelima belas isian wajib dilaporkan sekaligus")

	// Nama fieldnya WAJIB `kolom`, bukan nama lain yang kebetulan lebih tepat artinya.
	//
	// Klien bersama `api/client.ts` membaca nama isian dari `field` atau `kolom` saja.
	// Nama lain membuat APIError.violations() mengembalikan peta kosong, sehingga layar
	// menampilkan satu kotak galat umum alih-alih menyorot isian yang salah — pada form
	// lima belas isian wajib, itu berarti pengguna tidak diberi tahu yang mana.
	first, ok := detail[0].(map[string]any)
	require.True(t, ok)
	require.Contains(t, first, "kolom")
	require.Contains(t, first, "pesan")
	require.NotEmpty(t, first["kolom"])
}

func TestCreateRejectsDuplicateName(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/supplier", "ASM",
		validBody("Supplier Contoh Utama"))

	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "nama_supplier_sudah_ada", content["kode"])
}

// Field yang tidak dikenal ditolak, bukan diabaikan diam-diam.
//
// Salah ketik nama field akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai
// kosong tanpa satu pun tanda bahwa ada yang salah.
func TestCreateRejectsUnknownField(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/supplier", "ASM",
		`{"nama":"X","isian_yang_tidak_ada":"y"}`)

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// SUPPLIER_HE tidak diterima dari klien — ia turunan, dan server yang menurunkannya.
func TestCreateRejectsDerivedField(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodPost, "/api/master/supplier", "ASM",
		`{"nama":"X","supplier_he":"1"}`)

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// ============================================================================
// Penyimpanan
// ============================================================================

// Nama TIDAK dapat diubah setelah tersimpan.
//
// Layar Pega menegakkannya dengan mengunci isiannya; server ikut memeriksanya karena
// permintaan yang tidak datang dari layar tidak tersentuh penguncian itu.
func TestSaveRejectsRename(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/supplier/"+activeHeavyID, "ASM",
		validBody("Nama Yang Diganti"))

	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "nama_supplier_terkunci", content["kode"])
}

func TestSaveAcceptsSameName(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/supplier/"+activeHeavyID, "ASM",
		validBody("Supplier Contoh Utama"))

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "Supplier Contoh Utama", one(t, content)["nama"])
}

// Pada penyimpanan, status aktif yang BERLAKU menyusul yang diminta — berbeda dari
// penambahan yang selalu tidak aktif.
func TestSaveAppliesRequestedActiveStatus(t *testing.T) {
	p := newTestServer(t)

	// Baris contohnya TIDAK aktif; badan permintaan memintanya menjadi aktif.
	response, content := p.call(t, http.MethodPut, "/api/master/supplier/"+inactiveID, "ASM",
		validBody("Supplier Contoh Nonaktif"))
	require.Equal(t, http.StatusOK, response.StatusCode)

	item := one(t, content)
	require.Equal(t, "1", item["status_aktif_berlaku"])
	require.Equal(t, true, item["aktif"])
}

// Menonaktifkan supplier berlaku SEKETIKA, tanpa persetujuan siapa pun.
//
// Ia satu-satunya jalur di modul ini yang mengubah keadaan tanpa melewati antrean mana
// pun — perilaku sistem lama yang ditiru apa adanya
// (`EditMasterSupplier_post` step 12).
func TestSaveSkipsApprovalWhenDeactivating(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody("Supplier Contoh Utama"),
		`"status_aktif":"1"`, `"status_aktif":"0"`, 1)

	response, _ := p.call(t, http.MethodPut, "/api/master/supplier/"+activeHeavyID, "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, p.asm.Approval())
}

func TestSaveRequestsApprovalWhenActivating(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodPut, "/api/master/supplier/"+inactiveID, "ASM",
		validBody("Supplier Contoh Nonaktif"))

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, p.asm.Approval(), 1)
}

// Mengubah Status Supply benar-benar mengubah SUPPLIER_HE ke kedua arah.
//
// Sistem lama hanya punya cabang "bila" tanpa cabang "selain itu", sehingga supplier HE
// yang diubah menjadi bukan-HE tetap tersimpan sebagai HE dan perubahannya hilang tanpa
// satu pun tanda. SELISIH YANG DIRENCANAKAN.
func TestSaveTurnsOffHeavyEquipment(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody("Supplier Contoh Utama"),
		`"status_supply":"1"`, `"status_supply":"0"`, 1)

	_, content := p.call(t, http.MethodPut, "/api/master/supplier/"+activeHeavyID, "ASM", body)

	item := one(t, content)
	require.Equal(t, "0", item["supplier_he"])
	require.Equal(t, false, item["heavy_equipment"])
}

func TestSaveUnknownRowIs404(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodPut, "/api/master/supplier/tidak-ada", "ASM",
		validBody("Supplier Apa Pun"))
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

// ============================================================================
// Lookup
// ============================================================================

func TestLookupRoutesAreServed(t *testing.T) {
	p := newTestServer(t)

	for _, route := range []struct {
		path string
		key  string
	}{
		{"/api/master/supplier/cabang", "cabang"},
		{"/api/master/supplier/negara", "negara"},
		{"/api/master/supplier/bank", "bank"},
		{"/api/master/supplier/kota?cari=jak", "kota"},
	} {
		t.Run(route.key, func(t *testing.T) {
			response, content := p.call(t, http.MethodGet, route.path, "ASM", "")
			require.Equal(t, http.StatusOK, response.StatusCode)
			list, ok := content[route.key].([]any)
			require.Truef(t, ok, "badan respons tidak memuat senarai %q: %v", route.key, content)
			require.NotEmpty(t, list)
		})
	}
}

// Kata kunci yang terlalu pendek dijawab daftar kosong, BUKAN galat: pengguna yang baru
// mengetik satu huruf belum melakukan kesalahan apa pun.
func TestCityLookupIgnoresShortKeyword(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/supplier/kota?cari=j", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, content["kota"])
}

// Kelima daftar sandi dikirim sekaligus, dan nilai yang artinya terbukti selalu ada —
// termasuk pada entitas yang belum punya satu baris pun.
func TestCodeRouteMergesKnownAndStored(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/supplier/sandi", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	for _, key := range []string{
		"status_rekanan", "status_supply", "jenis_supplier", "status_aktif", "status_autopayment",
	} {
		_, exists := content[key]
		require.Truef(t, exists, "daftar sandi %q tidak dikirim", key)
	}

	supply, ok := content["status_supply"].([]any)
	require.True(t, ok)
	require.Len(t, supply, 2)

	_, empty := p.call(t, http.MethodGet, "/api/master/supplier/sandi", "ASI", "")
	emptySupply, ok := empty["status_supply"].([]any)
	require.True(t, ok)
	require.Len(t, emptySupply, 2,
		"entitas kosong pun harus menawarkan sandi yang artinya terbukti")
}

// ============================================================================
// Yang sengaja TIDAK ada
// ============================================================================

// Tidak ada DELETE. Sistem lama tidak punya satu pun — layarnya bahkan tidak punya
// tombolnya — dan D-66 melarang penghapusan fisik data bernilai bisnis.
func TestDeleteIsNotRouted(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, "/api/master/supplier/"+activeHeavyID, "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}

// Tidak ada endpoint keputusan persetujuan: sisi pemutus antrean proteksi_klaimmbu TIDAK
// ADA di export sama sekali (R-16), dan membangunnya berarti mengarang aturan yang
// menentukan supplier mana yang boleh dipakai.
func TestApprovalDecisionIsNotRouted(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodPost, "/api/master/supplier/keputusan", "ASM", `{}`)
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}
