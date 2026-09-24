package inboxoutstandinghttp_test

import (
	"context"
	"encoding/csv"
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

	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxoutstanding/repo/memory"
	"claim-pnc/internal/inboxoutstanding/usecase"

	outstandinghttp "claim-pnc/internal/inboxoutstanding/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testNow tetap, supaya kolom umur klaim dapat diperiksa dengan angka pasti.
var testNow = time.Date(2026, time.September, 20, 3, 0, 0, 0, time.UTC)

// wib dipakai agar uji tidak bergantung pada basis data zona waktu mesin penjalan.
var wib = time.FixedZone("WIB", 7*60*60)

// testPortal adalah entitas yang dipakai seluruh uji di berkas ini.
//
// Ia harus ada di daftar portal contoh DAN dinyatakan siap, karena middleware membedakan
// "portal tidak dikenal" dari "portal belum siap".
const testPortal = "ASM"

func testServer(t *testing.T, login string, claims ...inboxoutstanding.OutstandingClaim) http.Handler {
	t.Helper()

	repo := memory.NewRepo()
	repo.Add(claims...)
	repo.SetLineBusiness("DEWILESTARI", inboxoutstanding.LinePA)

	// Pemilih mengabaikan alias: yang diuji di berkas ini adalah lapisan transport, bukan
	// pemilihan basis data per entitas. Pemilihan itu diuji di usecase.
	service, err := usecase.NewService(
		func(string) (inboxoutstanding.Repo, error) { return repo, nil },
		repo,
	)
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}

	// Galat portal dipetakan modul portal, persis seperti rantai di cmd/claimpnc. Bila
	// rantai ini tidak ditiru, uji penolakan portal akan lulus di sini tetapi gagal di
	// aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode":  "galat_internal",
				"pesan": err.Error(),
			})
		},
		writeJSON,
	)

	handler := outstandinghttp.NewHandler(outstandinghttp.Options{
		Service: service,
		GetCaller: func(context.Context) (outstandinghttp.Caller, bool) {
			return outstandinghttp.Caller{Login: login}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: outstandinghttp.ErrorWriter(writeError),
		Location:            wib,
		Now:                 func() time.Time { return testNow },
	})

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		outstandinghttp.Mount(api, handler, portalDeps)
	})
	return router
}

// get mengirim permintaan BESERTA header portal.
//
// Portalnya disertakan di sini, bukan di tiap uji, supaya uji yang sengaja
// menghilangkannya — TestPermintaanTanpaPortalDitolak — terlihat jelas sebagai
// pengecualian.
func get(t *testing.T, server http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Portal", testPortal)

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func sampleClaim(id, panel string) inboxoutstanding.OutstandingClaim {
	reportDate := time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC)
	return inboxoutstanding.OutstandingClaim{
		ClaimID:        id,
		ClaimNumber:    "PNCN.26." + id,
		PolicyNumber:   "POL-" + id,
		InsuredName:    "PT Contoh " + id,
		BusinessName:   "Fire",
		BranchName:     "JKT",
		GroupPanel:     panel,
		RegisteredAt:   time.Date(2026, time.September, 15, 20, 0, 0, 0, time.UTC),
		ReportDate:     &reportDate,
		ProcessStatus:  "BERJALAN",
		ProgressStatus: "On Progress",
		TechnicalPIC:   "BUDISANTOSO",
		RecordedBy:     "ADMINPNC",
		CurrentStage:   "Komite",
		CurrentHolder:  "BUDISANTOSO",
	}
}

func TestDaftarDikembalikanBesertaTotalDanBatasLini(t *testing.T) {
	server := testServer(t, "ADMINPNC", sampleClaim("0001", "006"), sampleClaim("0002", "002"))

	res := get(t, server, "/api/inbox-outstanding")
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Klaim []struct {
			NomorKlaim         string `json:"nomor_klaim"`
			TanggalPendaftaran string `json:"tanggal_pendaftaran"`
			TanggalLapor       string `json:"tanggal_lapor"`
			StatusTampil       string `json:"status_tampil"`
			UmurHari           int    `json:"umur_hari"`
		} `json:"klaim"`
		Total     int `json:"total"`
		BatasLini struct {
			TanpaBatas bool     `json:"tanpa_batas"`
			GroupPanel []string `json:"group_panel"`
		} `json:"batas_lini"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))

	require.Equal(t, 2, body.Total)
	require.Len(t, body.Klaim, 2)
	require.True(t, body.BatasLini.TanpaBatas, "ADMINPNC tidak punya lini di data uji")
}

// Tanggal dikirim sebagai tanggal WIB, bukan timestamp UTC.
//
// Klaim di bawah didaftarkan 15 September 20.00 UTC, yaitu 16 September 03.00 WIB.
// Mengirim tanggal UTC akan menampilkannya sebagai 15 September — bergeser satu hari,
// tanpa satu pun galat (`R-12`).
func TestTanggalDikirimSebagaiTanggalWIB(t *testing.T) {
	server := testServer(t, "ADMINPNC", sampleClaim("0001", "006"))

	res := get(t, server, "/api/inbox-outstanding")
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Klaim []struct {
			TanggalPendaftaran string `json:"tanggal_pendaftaran"`
			TanggalLapor       string `json:"tanggal_lapor"`
			UmurHari           int    `json:"umur_hari"`
		} `json:"klaim"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))

	require.Equal(t, "2026-09-16", body.Klaim[0].TanggalPendaftaran)
	require.Equal(t, "2026-09-10", body.Klaim[0].TanggalLapor)

	// testNow = 20 Sept 03.00 UTC = 20 Sept 10.00 WIB. Daftar 16 Sept WIB -> 4 hari.
	require.Equal(t, 4, body.Klaim[0].UmurHari)
}

func TestBatasLiniMenyaringBarisDanDinyatakanDiRespons(t *testing.T) {
	server := testServer(t, "DEWILESTARI", sampleClaim("0001", "006"), sampleClaim("0002", "002"))

	res := get(t, server, "/api/inbox-outstanding")
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Klaim []struct {
			GroupPanel string `json:"group_panel"`
		} `json:"klaim"`
		Total     int `json:"total"`
		BatasLini struct {
			TanpaBatas bool     `json:"tanpa_batas"`
			GroupPanel []string `json:"group_panel"`
		} `json:"batas_lini"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))

	require.Equal(t, 1, body.Total, "hanya klaim PA yang terlihat")
	require.Equal(t, "002", body.Klaim[0].GroupPanel)
	require.False(t, body.BatasLini.TanpaBatas)
	require.Equal(t, []string{"002"}, body.BatasLini.GroupPanel)
}

func TestPencarianDanPenyaringDiteruskanKeServer(t *testing.T) {
	server := testServer(t, "ADMINPNC", sampleClaim("0001", "006"), sampleClaim("0002", "006"))

	res := get(t, server, "/api/inbox-outstanding?cari=POL-0002")
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, 1, body.Total)
}

// Batas di atas maksimum DITOLAK, bukan dipangkas diam-diam.
//
// Klien yang meminta seribu baris lalu menerima seratus tanpa diberi tahu akan
// menampilkan daftar yang ia kira lengkap.
func TestBatasDiAtasMaksimumDitolak(t *testing.T) {
	server := testServer(t, "ADMINPNC")

	res := get(t, server, "/api/inbox-outstanding?batas=1000")
	require.Equal(t, http.StatusBadRequest, res.Code)

	var body struct {
		Code string `json:"kode"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, "permintaan_cacat", body.Code)
}

func TestParameterPaginasiYangBukanBilanganDitolak(t *testing.T) {
	server := testServer(t, "ADMINPNC")

	for _, path := range []string{
		"/api/inbox-outstanding?batas=banyak",
		"/api/inbox-outstanding?lewati=-3",
	} {
		res := get(t, server, path)
		require.Equal(t, http.StatusBadRequest, res.Code, path)
	}
}

func TestUnduhMengirimCSVBesertaNamaBerkas(t *testing.T) {
	server := testServer(t, "ADMINPNC", sampleClaim("0001", "006"), sampleClaim("0002", "002"))

	res := get(t, server, "/api/inbox-outstanding/unduh")
	require.Equal(t, http.StatusOK, res.Code)
	require.Contains(t, res.Header().Get("Content-Type"), "text/csv")
	require.Contains(t, res.Header().Get("Content-Disposition"), "attachment")
	require.Contains(t, res.Header().Get("Content-Disposition"), "inbox-outstanding-")

	records, err := csv.NewReader(strings.NewReader(res.Body.String())).ReadAll()
	require.NoError(t, err)

	require.Len(t, records, 3, "satu baris judul dan dua baris data")

	// Judulnya sama persis dengan kolom layar, dan berbahasa Inggris karena begitulah
	// `Section/InboxRegister_Section-Section.xml` menulisnya (`D-13`).
	require.Equal(t, []string{
		"Claim no", "Policy no", "Insured name", "Business Name", "Business source",
		"Branch name", "Admin name", "Register Date", "Date of loss",
		"Total Aging", "Aging", "Claim status", "Status ASM", "ASM PIC",
	}, records[0])
}

// Kolom "Aging" yang belum terisi ditulis sebagai sel KOSONG, bukan nol.
//
// Pada berkas yang dibuka di pengolah angka, nol ikut terhitung dalam rata-rata dan
// penjumlahan. "Belum diisi" bukan nol.
func TestAgingYangBelumTerisiDitulisSebagaiSelKosong(t *testing.T) {
	nol := 0
	terisi := sampleClaim("0001", "006")
	terisi.AgingDays = &nol
	kosong := sampleClaim("0002", "006") // AgingDays nil

	server := testServer(t, "ADMINPNC", terisi, kosong)

	res := get(t, server, "/api/inbox-outstanding/unduh")
	require.Equal(t, http.StatusOK, res.Code)

	records, err := csv.NewReader(strings.NewReader(res.Body.String())).ReadAll()
	require.NoError(t, err)

	agingColumn := -1
	for i, title := range records[0] {
		if title == "Aging" {
			agingColumn = i
		}
	}
	require.NotEqual(t, -1, agingColumn, "kolom Aging harus ada")

	nilai := []string{records[1][agingColumn], records[2][agingColumn]}
	require.Contains(t, nilai, "0", "yang terisi nol ditulis sebagai 0")
	require.Contains(t, nilai, "", "yang belum terisi ditulis kosong")
}

// Unduhan mengikuti batas data yang sama dengan daftar.
//
// Bila tidak, seorang petugas dapat melewati batas linianya hanya dengan menekan tombol
// unduh — dan tidak ada galat yang muncul.
func TestUnduhTundukPadaBatasLiniYangSama(t *testing.T) {
	server := testServer(t, "DEWILESTARI", sampleClaim("0001", "006"), sampleClaim("0002", "002"))

	res := get(t, server, "/api/inbox-outstanding/unduh")
	require.Equal(t, http.StatusOK, res.Code)

	records, err := csv.NewReader(strings.NewReader(res.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2, "judul + satu baris klaim PA saja")
	require.Equal(t, "PNCN.26.0002", records[1][0])
}

// Unduhan mengabaikan halaman: yang diminta adalah seluruh hasil, bukan halaman yang
// sedang dilihat.
func TestUnduhMengabaikanParameterHalaman(t *testing.T) {
	var claims []inboxoutstanding.OutstandingClaim
	for i := 0; i < 5; i++ {
		claims = append(claims, sampleClaim(string(rune('a'+i)), "006"))
	}
	server := testServer(t, "ADMINPNC", claims...)

	res := get(t, server, "/api/inbox-outstanding/unduh?batas=2&lewati=4")
	require.Equal(t, http.StatusOK, res.Code)

	records, err := csv.NewReader(strings.NewReader(res.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 6, "judul + kelima klaim")
}

// Modul ini hanya MEMBACA. Rute tulis yang tidak didaftarkan dijawab 405, sehingga
// penambahannya kelak menjadi keputusan sadar.
func TestRuteTulisTidakTersedia(t *testing.T) {
	server := testServer(t, "ADMINPNC")

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/api/inbox-outstanding/", nil)
		request.Header.Set("X-Portal", testPortal)
		server.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusMethodNotAllowed, recorder.Code, method)
	}
}

// Permintaan tanpa portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan.
//
// Inilah yang mencegah klaim satu badan hukum tampil di layar badan hukum lain tanpa satu
// pun galat (`R-20`, `TKT-F6-002`). Kedua rute diuji: melewatkan unduhan berarti batas
// entitas dapat dilewati hanya dengan menekan tombol unduh.
func TestPermintaanTanpaPortalDitolak(t *testing.T) {
	server := testServer(t, "ADMINPNC", sampleClaim("0001", "006"))

	for _, path := range []string{
		"/api/inbox-outstanding",
		"/api/inbox-outstanding/unduh",
	} {
		recorder := httptest.NewRecorder()
		// Sengaja TANPA header X-Portal.
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

		require.Equal(t, http.StatusBadRequest, recorder.Code, path)

		var body struct {
			Code string `json:"kode"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		require.Equal(t, "portal_tidak_disebut", body.Code, path)
	}
}

// Portal yang tidak ada di daftar entitas ditolak sebagai TIDAK DIKENAL, bukan diam-diam
// dilayani.
func TestPortalTidakDikenalDitolak(t *testing.T) {
	server := testServer(t, "ADMINPNC", sampleClaim("0001", "006"))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding", nil)
	request.Header.Set("X-Portal", "ENTAH-APA")
	server.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)

	var body struct {
		Code string `json:"kode"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "portal_tidak_dikenal", body.Code)
}
