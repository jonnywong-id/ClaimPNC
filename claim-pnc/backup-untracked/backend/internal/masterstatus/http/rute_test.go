package masterstatushttp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus/repo/memori"
	"claim-pnc/internal/masterstatus/usecase"

	masterstatushttp "claim-pnc/internal/masterstatus/http"
)

func serverUji(t *testing.T) http.Handler {
	t.Helper()

	repo := memori.RepoBaru(memori.DaftarContoh()...)
	layanan, err := usecase.LayananBaru(usecase.Opsi{Repo: repo})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tulisJSON := func(w http.ResponseWriter, _ *http.Request, status int, badan any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if badan != nil {
			_ = json.NewEncoder(w).Encode(badan)
		}
	}

	handler := masterstatushttp.HandlerBaru(masterstatushttp.Opsi{
		Layanan:     layanan,
		Logger:      logger,
		TulisRespon: tulisJSON,
	})

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		masterstatushttp.Pasang(api, handler)
	})
	return router
}

func panggil(t *testing.T, server http.Handler, metode, jalur string, badan any) *httptest.ResponseRecorder {
	t.Helper()

	var isi io.Reader
	if badan != nil {
		mentah, err := json.Marshal(badan)
		require.NoError(t, err)
		isi = bytes.NewReader(mentah)
	}
	permintaan := httptest.NewRequest(metode, jalur, isi)
	if badan != nil {
		permintaan.Header.Set("Content-Type", "application/json")
	}

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, permintaan)
	return rekaman
}

func TestDaftarMengembalikan33StatusDanTotalnya(t *testing.T) {
	rekaman := panggil(t, serverUji(t), http.MethodGet, "/api/master/status-klaim", nil)
	require.Equal(t, http.StatusOK, rekaman.Code)

	var respons masterstatushttp.ResponsDaftar
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))

	require.Len(t, respons.StatusKlaim, 33)
	require.Equal(t, 33, respons.Total, "total datang dari server, bukan dihitung klien")
	require.Equal(t, "1134", respons.StatusKlaim[0].Kode)
	require.Equal(t, "01", respons.StatusKlaim[0].KodeLama)
	require.Empty(t, respons.StatusKlaim[32].KodeLama)
}

func TestAmbilSatuStatus(t *testing.T) {
	rekaman := panggil(t, serverUji(t), http.MethodGet, "/api/master/status-klaim/1149", nil)
	require.Equal(t, http.StatusOK, rekaman.Code)

	var respons masterstatushttp.ResponsSatu
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	require.Equal(t, "Claim Committee", respons.StatusKlaim.Label)
}

func TestAmbilStatusTidakAdaDijawab404(t *testing.T) {
	rekaman := panggil(t, serverUji(t), http.MethodGet, "/api/master/status-klaim/9999", nil)
	require.Equal(t, http.StatusNotFound, rekaman.Code)

	var galat masterstatushttp.ResponsGalat
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &galat))
	require.Equal(t, masterstatushttp.KodeTidakDitemukan, galat.Kode)
}

func TestTambahDijawab201DenganKodeBaru(t *testing.T) {
	server := serverUji(t)

	rekaman := panggil(t, server, http.MethodPost, "/api/master/status-klaim",
		masterstatushttp.PermintaanSimpan{Label: "Status Percobaan"})
	require.Equal(t, http.StatusCreated, rekaman.Code, "sumber daya baru terbentuk")

	var respons masterstatushttp.ResponsSatu
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	require.Equal(t, "1167", respons.StatusKlaim.Kode)

	daftar := panggil(t, server, http.MethodGet, "/api/master/status-klaim", nil)
	var isi masterstatushttp.ResponsDaftar
	require.NoError(t, json.Unmarshal(daftar.Body.Bytes(), &isi))
	require.Equal(t, 34, isi.Total)
}

// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan bisnis.
// Frontend menanganinya berbeda.
func TestLabelKosongDijawab422DenganDetailPerField(t *testing.T) {
	rekaman := panggil(t, serverUji(t), http.MethodPost, "/api/master/status-klaim",
		masterstatushttp.PermintaanSimpan{Label: "  "})
	require.Equal(t, http.StatusUnprocessableEntity, rekaman.Code)

	var galat masterstatushttp.ResponsGalat
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &galat))
	require.Equal(t, masterstatushttp.KodeValidasiGagal, galat.Kode)
	require.Len(t, galat.Detail, 1, "layar perlu tahu kolom mana yang salah")
	require.Equal(t, "label", galat.Detail[0].Field)
	require.NotEmpty(t, galat.Detail[0].Pesan)
}

// 409, bukan 422: isian penggunanya sah tetapi bentrok dengan keadaan penyimpanan.
func TestLabelGandaDijawab409(t *testing.T) {
	rekaman := panggil(t, serverUji(t), http.MethodPost, "/api/master/status-klaim",
		masterstatushttp.PermintaanSimpan{Label: "Paid"})
	require.Equal(t, http.StatusConflict, rekaman.Code)

	var galat masterstatushttp.ResponsGalat
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &galat))
	require.Equal(t, masterstatushttp.KodeLabelSudahAda, galat.Kode)
	require.Empty(t, galat.Detail, "konflik bukan galat per-field")
}

func TestUbahDijawab200(t *testing.T) {
	server := serverUji(t)

	rekaman := panggil(t, server, http.MethodPut, "/api/master/status-klaim/1163",
		masterstatushttp.PermintaanSimpan{Label: "Sudah Dibayar"})
	require.Equal(t, http.StatusOK, rekaman.Code)

	var respons masterstatushttp.ResponsSatu
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	require.Equal(t, "1163", respons.StatusKlaim.Kode)
	require.Equal(t, "Sudah Dibayar", respons.StatusKlaim.Label)
}

// PUT mengirim seluruh isi yang boleh diubah, sehingga mengirimnya dua kali menghasilkan
// keadaan akhir yang sama.
func TestUbahBersifatIdempoten(t *testing.T) {
	server := serverUji(t)
	permintaan := masterstatushttp.PermintaanSimpan{Label: "Sudah Dibayar"}

	pertama := panggil(t, server, http.MethodPut, "/api/master/status-klaim/1163", permintaan)
	require.Equal(t, http.StatusOK, pertama.Code)

	kedua := panggil(t, server, http.MethodPut, "/api/master/status-klaim/1163", permintaan)
	require.Equal(t, http.StatusOK, kedua.Code)
	require.JSONEq(t, pertama.Body.String(), kedua.Body.String())
}

func TestUbahStatusTidakAdaDijawab404(t *testing.T) {
	rekaman := panggil(t, serverUji(t), http.MethodPut, "/api/master/status-klaim/9999",
		masterstatushttp.PermintaanSimpan{Label: "Apa Saja"})
	require.Equal(t, http.StatusNotFound, rekaman.Code)
}

func TestBadanPermintaanCacatDijawab400(t *testing.T) {
	server := serverUji(t)

	permintaan := httptest.NewRequest(http.MethodPost, "/api/master/status-klaim",
		bytes.NewReader([]byte("{bukan json")))
	permintaan.Header.Set("Content-Type", "application/json")

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, permintaan)

	require.Equal(t, http.StatusBadRequest, rekaman.Code, "400 berarti klien salah membentuk permintaan")

	var galat masterstatushttp.ResponsGalat
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &galat))
	require.Equal(t, masterstatushttp.KodePermintaanCacat, galat.Kode)
}

// Kode tidak pernah datang dari klien. Mengirimnya di badan permintaan tidak boleh
// memindahkan status ke kode lain.
func TestKodeDiBadanPermintaanDiabaikan(t *testing.T) {
	server := serverUji(t)

	mentah := []byte(`{"kode":"7777","label":"Status Percobaan"}`)
	permintaan := httptest.NewRequest(http.MethodPost, "/api/master/status-klaim", bytes.NewReader(mentah))
	permintaan.Header.Set("Content-Type", "application/json")

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, permintaan)
	require.Equal(t, http.StatusCreated, rekaman.Code)

	var respons masterstatushttp.ResponsSatu
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	require.Equal(t, "1167", respons.StatusKlaim.Kode, "kode tetap dibuat sistem, bukan diambil dari klien")
}

// Tidak ada rute hapus. Sama seperti layar Pega yang tidak punya tombol hapus.
func TestRuteHapusTidakAda(t *testing.T) {
	rekaman := panggil(t, serverUji(t), http.MethodDelete, "/api/master/status-klaim/1163", nil)
	require.Equal(t, http.StatusMethodNotAllowed, rekaman.Code)
}

// Badan permintaan yang sangat besar ditolak, bukan dibaca seluruhnya ke memori.
func TestBadanPermintaanTerlaluBesarDitolak(t *testing.T) {
	server := serverUji(t)

	besar := make([]byte, 64<<10)
	for i := range besar {
		besar[i] = 'a'
	}
	mentah := append(append([]byte(`{"label":"`), besar...), []byte(`"}`)...)

	permintaan := httptest.NewRequest(http.MethodPost, "/api/master/status-klaim", bytes.NewReader(mentah))
	permintaan.Header.Set("Content-Type", "application/json")

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, permintaan)
	require.Equal(t, http.StatusBadRequest, rekaman.Code)
}
