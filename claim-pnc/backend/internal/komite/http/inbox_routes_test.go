package komitehttp_test

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

	"claim-pnc/internal/komite/repo/memory"
	"claim-pnc/internal/komite/usecase"
	"claim-pnc/internal/platform/clock"

	komitehttp "claim-pnc/internal/komite/http"
)

// sekarangHTTP dibekukan supaya Aging pada respons dapat diperiksa dengan angka pasti.
//
// Nilainya sama dengan tanggal acuan SampleCases, sehingga umur kasus contoh di sini
// persis seperti yang dimaksud saat contohnya disusun.
var sekarangHTTP = time.Date(2026, 9, 20, 2, 0, 0, 0, time.UTC)

// serverInbox membangun server berisi rute Inbox Komite dengan pemanggil yang sudah
// dikenali sebagai SampleOperator.
//
// Pemanggil dipasang lewat jembatan yang sama dengan yang dipakai cmd/claimpnc, sehingga
// yang diuji di sini adalah jalur yang benar-benar berjalan di produksi — bukan jalan
// pintas yang hanya ada di pengujian.
func serverInbox(t *testing.T, login string) http.Handler {
	t.Helper()

	store := memory.NewSampleInboxStore()
	service, err := usecase.NewInboxService(usecase.InboxOptions{
		Cases:     store,
		Decisions: store,
		IDs:       memory.IDGenerator{},
		Clock:     clock.FixedAt(sekarangHTTP),
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tulisJSON := func(w http.ResponseWriter, _ *http.Request, status int, badan any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if badan != nil {
			_ = json.NewEncoder(w).Encode(badan)
		}
	}

	master, err := usecase.NewService(usecase.Options{Repo: memory.NewSampleRepo()})
	require.NoError(t, err)

	inbox := komitehttp.NewInboxHandler(komitehttp.InboxHandlerOptions{
		Service: service,
		Caller: func(context.Context) (komitehttp.InboxCaller, bool) {
			if login == "" {
				return komitehttp.InboxCaller{}, false
			}
			return komitehttp.InboxCaller{Login: login, Name: "Penguji"}, true
		},
		Logger:        logger,
		WriteResponse: tulisJSON,
	})

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		komitehttp.Mount(api, komitehttp.NewHandler(komitehttp.Options{
			Service:       master,
			Logger:        logger,
			WriteResponse: tulisJSON,
		}), inbox)
	})
	return router
}

func kirim(t *testing.T, server http.Handler, metode, jalur, badan string) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if badan != "" {
		body = strings.NewReader(badan)
	}
	permintaan := httptest.NewRequest(metode, jalur, body)
	if badan != "" {
		permintaan.Header.Set("Content-Type", "application/json")
	}

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, permintaan)
	return rekaman
}

func bacaDaftarInbox(t *testing.T, rekaman *httptest.ResponseRecorder) komitehttp.InboxListResponse {
	t.Helper()
	var respons komitehttp.InboxListResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	return respons
}

func bacaKasus(t *testing.T, rekaman *httptest.ResponseRecorder) komitehttp.CommitteeCaseResponse {
	t.Helper()
	var respons komitehttp.CommitteeCaseResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	return respons
}

func bacaGalat(t *testing.T, rekaman *httptest.ResponseRecorder) komitehttp.ErrorResponse {
	t.Helper()
	var respons komitehttp.ErrorResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	return respons
}

func TestDaftarInboxMengembalikanPekerjaanMilikPemanggil(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	rekaman := kirim(t, server, http.MethodGet, "/api/komite/inbox", "")
	require.Equal(t, http.StatusOK, rekaman.Code)

	respons := bacaDaftarInbox(t, rekaman)
	require.Equal(t, "outstanding", respons.Kind, "kotak bawaan adalah yang berisi pekerjaan")
	require.Equal(t, memory.SampleOperator, respons.Operator)
	require.Equal(t, 3, respons.Total)
	require.Equal(t, 3, respons.Summary.Outstanding)
	require.NotEmpty(t, respons.Now)

	// Nilai uang dikirim sebagai TEKS desimal kanonik, bukan angka JSON.
	require.Equal(t, "45000000.00", respons.Cases[0].ClaimValue)
}

// Kotak yang tidak dikenali JATUH ke Outstanding, dan pantulan `kotak` pada respons
// memberi tahu layar bahwa permintaannya diperlakukan berbeda.
func TestKotakTidakDikenalJatuhKeOutstandingDanDipantulkan(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	respons := bacaDaftarInbox(t, kirim(t, server, http.MethodGet, "/api/komite/inbox?kotak=entahlah", ""))
	require.Equal(t, "outstanding", respons.Kind)
}

// Tanggal yang tidak dapat diurai DITOLAK sebagai validasi, bukan diabaikan.
//
// Penyaring yang gagal terurai lalu dianggap kosong akan menampilkan SELURUH riwayat
// kepada seseorang yang mengira ia sedang melihat satu minggu.
func TestTanggalCacatDitolakSebagai422(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	rekaman := kirim(t, server, http.MethodGet, "/api/komite/inbox?dari=20-09-2026", "")
	require.Equal(t, http.StatusUnprocessableEntity, rekaman.Code)
	require.Equal(t, komitehttp.CodeValidationFailed, bacaGalat(t, rekaman).Code)
}

// Paginasi yang cacat TIDAK ditolak — ia jatuh ke halaman pertama.
//
// Bedanya dari tanggal: `lewati=abc` tidak mengubah APA yang ditampilkan, hanya dari mana
// halamannya dimulai.
func TestPaginasiCacatJatuhKeHalamanPertama(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	rekaman := kirim(t, server, http.MethodGet, "/api/komite/inbox?lewati=abc&batas=-3", "")
	require.Equal(t, http.StatusOK, rekaman.Code)

	respons := bacaDaftarInbox(t, rekaman)
	require.Zero(t, respons.Offset)
	require.Equal(t, 25, respons.Limit, "batas cacat memakai ukuran halaman bawaan")
}

// Permintaan yang batasnya melebihi MaxPageSize DIPANGKAS, dan batas yang benar-benar
// dipakai dipantulkan.
func TestBatasHalamanDipangkasDanDipantulkan(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	respons := bacaDaftarInbox(t, kirim(t, server, http.MethodGet, "/api/komite/inbox?batas=9999", ""))
	require.Equal(t, 100, respons.Limit)
}

// Sesi tanpa identitas tidak melihat apa pun, dan jawabannya TIDAK membocorkan bahwa
// inbox orang lain ada.
func TestPemanggilTanpaIdentitasTidakMelihatApaPun(t *testing.T) {
	server := serverInbox(t, "")

	rekaman := kirim(t, server, http.MethodGet, "/api/komite/inbox", "")
	require.Equal(t, http.StatusNotFound, rekaman.Code)
	require.Equal(t, komitehttp.CodeCaseNotFound, bacaGalat(t, rekaman).Code)
}

// Kasus milik orang lain dijawab 404 YANG SAMA dengan kasus yang tidak ada.
//
// Membedakannya akan mengubah endpoint ini menjadi alat untuk menebak nomor case.
func TestKasusOrangLainDanKasusTidakAdaDijawabSama(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	milikOrangLain := kirim(t, server, http.MethodGet, "/api/komite/inbox/K-2606", "")
	tidakAda := kirim(t, server, http.MethodGet, "/api/komite/inbox/K-9999", "")

	require.Equal(t, http.StatusNotFound, milikOrangLain.Code)
	require.Equal(t, http.StatusNotFound, tidakAda.Code)
	require.Equal(t, bacaGalat(t, tidakAda).Code, bacaGalat(t, milikOrangLain).Code)
	require.Equal(t, bacaGalat(t, tidakAda).Message, bacaGalat(t, milikOrangLain).Message)
}

func TestDetailKasusMengembalikanPenjenjangan(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	rekaman := kirim(t, server, http.MethodGet, "/api/komite/inbox/K-2601", "")
	require.Equal(t, http.StatusOK, rekaman.Code)

	kasus := bacaKasus(t, rekaman).Case
	require.Equal(t, "K-2601", kasus.CaseID)
	require.Equal(t, "menunggu", kasus.Progress.Outcome)
	require.True(t, kasus.Progress.TierCountUnknown)
	require.False(t, kasus.Progress.DecidedByMe)
	require.Empty(t, kasus.LegacyOutcome, "Pega belum memutuskan apa pun pada kasus ini")
}

func TestKeputusanTercatatDanDikembalikan(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	rekaman := kirim(t, server, http.MethodPost, "/api/komite/inbox/K-2601/keputusan",
		`{"keputusan":"setuju","catatan":"Sesuai hasil survei."}`)
	require.Equal(t, http.StatusOK, rekaman.Code)

	kasus := bacaKasus(t, rekaman).Case
	require.True(t, kasus.Progress.DecidedByMe)
	require.Len(t, kasus.Progress.Decisions, 1)
	require.Equal(t, "setuju", kasus.Progress.Decisions[0].Kind)
	require.Equal(t, 1, kasus.Progress.Decisions[0].Tier)
	require.Equal(t, memory.SampleOperator, kasus.Progress.Decisions[0].ActorLogin)
}

// Keputusan kedua dijawab 409, bukan 422: yang berubah adalah KEADAAN kasusnya, bukan
// isian permintaannya. Layar menanganinya dengan memuat ulang.
func TestKeputusanKeduaDijawab409(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)
	badan := `{"keputusan":"setuju"}`

	require.Equal(t, http.StatusOK,
		kirim(t, server, http.MethodPost, "/api/komite/inbox/K-2601/keputusan", badan).Code)

	kedua := kirim(t, server, http.MethodPost, "/api/komite/inbox/K-2601/keputusan", badan)
	require.Equal(t, http.StatusConflict, kedua.Code)
	require.Equal(t, komitehttp.CodeDecisionClosed, bacaGalat(t, kedua).Code)
}

func TestPenolakanTanpaCatatanDitolak422(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	rekaman := kirim(t, server, http.MethodPost, "/api/komite/inbox/K-2601/keputusan",
		`{"keputusan":"tolak"}`)
	require.Equal(t, http.StatusUnprocessableEntity, rekaman.Code)

	galat := bacaGalat(t, rekaman)
	require.Equal(t, komitehttp.CodeValidationFailed, galat.Code)
	require.Len(t, galat.Details, 1)
	require.Equal(t, "catatan", galat.Details[0].Field)
}

// Field yang tidak dikenali DITOLAK, bukan diabaikan.
//
// Klien yang mengirim `jenjang` harus tahu bahwa ia tidak dipakai — mengabaikannya
// diam-diam akan membuatnya mengira ia menentukan sesuatu yang sebenarnya milik server.
func TestFieldTidakDikenalPadaBadanDitolak(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	rekaman := kirim(t, server, http.MethodPost, "/api/komite/inbox/K-2601/keputusan",
		`{"keputusan":"setuju","jenjang":4}`)
	require.Equal(t, http.StatusBadRequest, rekaman.Code)
	require.Equal(t, komitehttp.CodeMalformedBody, bacaGalat(t, rekaman).Code)
}

func TestBadanYangBukanJSONDitolak400(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	rekaman := kirim(t, server, http.MethodPost, "/api/komite/inbox/K-2601/keputusan", "bukan json")
	require.Equal(t, http.StatusBadRequest, rekaman.Code)
}

// Keputusan atas kasus orang lain dijawab 404, sama dengan kasus yang tidak ada.
func TestKeputusanAtasKasusOrangLainDijawab404(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	rekaman := kirim(t, server, http.MethodPost, "/api/komite/inbox/K-2606/keputusan",
		`{"keputusan":"setuju"}`)
	require.Equal(t, http.StatusNotFound, rekaman.Code)
}

// Waktu yang tidak terisi dikirim KOSONG, bukan sebagai tanggal tahun 1.
//
// `0001-01-01T00:00:00Z` akan tergambar di layar sebagai tanggal yang terbaca seperti
// data rusak, padahal artinya "belum ada".
func TestWaktuYangTidakTerisiDikirimKosong(t *testing.T) {
	server := serverInbox(t, memory.SampleOperator)

	kasus := bacaKasus(t, kirim(t, server, http.MethodGet, "/api/komite/inbox/K-2603", "")).Case
	require.False(t, kasus.HasAIAssessment)
	require.Empty(t, kasus.AIAssessedAt)
	require.NotEmpty(t, kasus.CommitteeDate)
}
