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

	// Pemilih mengabaikan alias: yang diuji di berkas ini adalah lapisan transport, bukan
	// pemilihan basis data per entitas. Pemilihan itu diuji di usecase.
	service, err := usecase.NewService(func(string) (inboxoutstanding.Repo, error) { return repo, nil })
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
		ClaimID:       id,
		ClaimNumber:   "PNCN.26." + id,
		PolicyNumber:  "POL-" + id,
		InsuredName:   "PT Contoh " + id,
		BusinessName:  "Fire",
		BranchName:    "JKT",
		GroupPanel:    panel,
		RegisteredAt:  time.Date(2026, time.September, 15, 20, 0, 0, 0, time.UTC),
		ReportDate:    &reportDate,
		ProcessStatus: "New",
		// Seluruh klaim contoh dimiliki pemanggil uji: daftar TANPA pemilik tidak lagi
		// mungkin sejak layar ini menjadi My Inbox.
		CurrentHolder:  "ADMINPNC",
		ProgressStatus: "On Progress",
		TechnicalPIC:   "BUDISANTOSO",
		RecordedBy:     "ADMINPNC",
		CurrentStage:   "Komite",
	}
}

func TestDaftarDikembalikanBesertaTotalDanPemiliknya(t *testing.T) {
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
		Total   int    `json:"total"`
		Pemilik string `json:"pemilik"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))

	require.Equal(t, 2, body.Total)
	require.Len(t, body.Klaim, 2)
	require.Equal(t, "ADMINPNC", body.Pemilik, "layar menyatakan pekerjaan siapa yang ditampilkan")
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
}

// Daftar hanya memuat pekerjaan pemanggil — janji terpenting layar ini.
//
// Tanpa penyaring ini, layar bernama "My Inbox" menampilkan pekerjaan seluruh operator:
// terisi, tampak wajar, dan salah tanpa satu pun galat.
func TestDaftarHanyaMemuatPekerjaanPemanggil(t *testing.T) {
	milikOrangLain := sampleClaim("0009", "002")
	milikOrangLain.CurrentHolder = "SITIRAHAYU"

	server := testServer(t, "ADMINPNC", sampleClaim("0001", "006"), milikOrangLain)

	res := get(t, server, "/api/inbox-outstanding")
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Klaim []struct {
			NomorKlaim string `json:"nomor_klaim"`
		} `json:"klaim"`
		Total   int    `json:"total"`
		Pemilik string `json:"pemilik"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))

	require.Equal(t, 1, body.Total, "pekerjaan operator lain tidak boleh ikut")
	require.Equal(t, "PNCN.26.0001", body.Klaim[0].NomorKlaim)
	require.Equal(t, "ADMINPNC", body.Pemilik)
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

// Unduhan TIDAK terikat pada pemilik pekerjaan — berbeda dari daftar.
//
// # Uji ini sempat menyatakan kebalikannya
//
// Sebelumnya ia menuntut unduhan berisi "satu baris pekerjaan pemanggil saja", dengan
// alasan yang terdengar meyakinkan: tanpa itu seorang petugas dapat membaca pekerjaan
// operator lain lewat tombol unduh. Alasan itu masuk akal, tetapi TIDAK COCOK dengan
// sistem yang sedang dimigrasikan.
//
// `RDB List/ExportDataDetailKlaim-SQL.xml` tidak punya satu pun penyaring operator. Yang
// membatasi unduhan di sana adalah cakupan lini bisnis, dan sengaja demikian: unduhan itu
// memang berkas pemantauan satu lini, bukan salinan inbox pribadi.
//
// Uji lama lulus karena export memanggil ulang daftar, sehingga mewarisi penyaring
// pemiliknya tanpa satu baris kode pun yang menyatakannya — dan akibatnya petugas yang
// inbox-nya kosong mengunduh berkas kosong. Lihat TestUnduhTetapBerisiSaatInboxPemanggilKosong.
func TestUnduhTidakDisaringPemilikPekerjaan(t *testing.T) {
	milikOrangLain := sampleClaim("0002", "006")
	milikOrangLain.CurrentHolder = "SITIRAHAYU"

	server := testServer(t, "ADMINPNC", sampleClaim("0001", "006"), milikOrangLain)

	res := get(t, server, "/api/inbox-outstanding/unduh")
	require.Equal(t, http.StatusOK, res.Code)

	records, err := csv.NewReader(strings.NewReader(res.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3, "judul + kedua klaim, termasuk milik operator lain")

	nomor := []string{records[1][0], records[2][0]}
	require.Contains(t, nomor, "PNCN.26.0001")
	require.Contains(t, nomor, "PNCN.26.0002", "pekerjaan operator lain ikut terunduh")
}

// Unduhan tetap berisi meski inbox pemanggil kosong.
//
// Inilah kegagalan yang ditemukan Work Owner: export dulu memanggil ulang daftar, sehingga
// petugas yang tidak sedang memegang satu pun tugas menerima berkas berisi judul kolom
// saja — padahal di Pega berkasnya berisi seluruh klaim dalam cakupan lininya.
func TestUnduhTetapBerisiSaatInboxPemanggilKosong(t *testing.T) {
	milikOrangLain := sampleClaim("0002", "006")
	milikOrangLain.CurrentHolder = "SITIRAHAYU"

	// Pemanggil tidak memegang satu pun tugas.
	server := testServer(t, "ADMINPNC", milikOrangLain)

	// Daftar memang kosong — itu benar dan tidak diubah.
	daftar := get(t, server, "/api/inbox-outstanding")
	require.Equal(t, http.StatusOK, daftar.Code)
	var body struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(daftar.Body.Bytes(), &body))
	require.Equal(t, 0, body.Total, "inbox pemanggil memang kosong")

	// Unduhan tetap berisi.
	res := get(t, server, "/api/inbox-outstanding/unduh")
	require.Equal(t, http.StatusOK, res.Code)

	records, err := csv.NewReader(strings.NewReader(res.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2, "judul + satu baris, meski inbox pemanggil kosong")
	require.Equal(t, "PNCN.26.0002", records[1][0])
}

// Rentang tanggal yang tidak sah ditolak sebelum satu baris pun dibaca.
func TestUnduhMenolakRentangTanggalYangTidakSah(t *testing.T) {
	server := testServer(t, "ADMINPNC", sampleClaim("0001", "006"))

	for _, path := range []string{
		"/api/inbox-outstanding/unduh?dari=16-09-2026",
		"/api/inbox-outstanding/unduh?sampai=bukan-tanggal",
		"/api/inbox-outstanding/unduh?dari=2026-09-20&sampai=2026-09-10",
	} {
		res := get(t, server, path)
		require.Equal(t, http.StatusBadRequest, res.Code, path)
	}
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
