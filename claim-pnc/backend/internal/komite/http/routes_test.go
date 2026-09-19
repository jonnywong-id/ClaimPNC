package komitehttp_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/komite/repo/memory"
	"claim-pnc/internal/komite/usecase"
	"claim-pnc/internal/platform/money"

	komitehttp "claim-pnc/internal/komite/http"
)

func serverUji(t *testing.T) http.Handler {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{Repo: memory.NewSampleRepo()})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tulisJSON := func(w http.ResponseWriter, _ *http.Request, status int, badan any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if badan != nil {
			_ = json.NewEncoder(w).Encode(badan)
		}
	}

	handler := komitehttp.NewHandler(komitehttp.Options{
		Service:       service,
		Logger:        logger,
		WriteResponse: tulisJSON,
	})

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		komitehttp.Mount(api, handler)
	})
	return router
}

func panggil(t *testing.T, server http.Handler, jalur string) *httptest.ResponseRecorder {
	t.Helper()

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, httptest.NewRequest(http.MethodGet, jalur, nil))
	return rekaman
}

func bacaPenjenjangan(t *testing.T, rekaman *httptest.ResponseRecorder) komitehttp.TieringResponse {
	t.Helper()
	var respons komitehttp.TieringResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	return respons
}

func TestDaftarAmbangMengembalikanSeluruhBarisBesertaRingkasannya(t *testing.T) {
	rekaman := panggil(t, serverUji(t), "/api/master/ambang-komite")
	require.Equal(t, http.StatusOK, rekaman.Code)

	var respons komitehttp.ThresholdListResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))

	require.Equal(t, len(respons.Thresholds), respons.Total, "total datang dari server")
	require.Less(t, respons.TotalTiers, respons.Total,
		"sebagian baris bukan jenjang persetujuan — yang tidak aktif dan yang hanya untuk registrasi")

	// Daftar lini datang DARI DATA, bukan dari daftar tetap di dalam kode.
	require.Equal(t,
		[]string{"BONDING", "NONMBU", "NONMBUAB", "NONMBUC", "PA", "TRAVEL"},
		respons.BusinessLine)

	// Batas pita dikirim supaya pengguna dapat melihatnya, bukan menghafalnya.
	require.Len(t, respons.BandPolicies, 1)
	require.Equal(t, "NONMBU", respons.BandPolicies[0].BusinessLine)
	require.Equal(t, "100000000.00", respons.BandPolicies[0].Boundary)
}

// Nilai uang dikirim sebagai TEKS desimal kanonik, bukan angka JSON.
//
// Angka JSON adalah floating point ganda di hampir seluruh peramban, dan mengirim nilai
// uang lewatnya berarti menyerahkan ketepatannya kepada pembulatan biner — persis yang
// `I-12` larang.
func TestNilaiUangDikirimSebagaiTeksKanonik(t *testing.T) {
	rekaman := panggil(t, serverUji(t), "/api/master/ambang-komite")

	var mentah struct {
		Ambang []map[string]any `json:"ambang"`
	}
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &mentah))
	require.NotEmpty(t, mentah.Thresholds)

	for _, baris := range mentah.Thresholds {
		require.IsType(t, "", baris["batas_bawah"], "batas_bawah harus teks, bukan angka JSON")
		require.IsType(t, "", baris["batas_atas"], "batas_atas harus teks, bukan angka JSON")
	}
}

// Alamat surel tidak boleh muncul di respons sama sekali.
//
// Ia tidak dibaca dari basis data, sehingga uji ini sebenarnya menguji rantai yang lebih
// panjang: kueri, scanner, DTO, dan handler — bila salah satunya kelak menambahkannya,
// inilah yang gagal lebih dulu. `D-67` menetapkan alamat pribadi pada master lama tidak
// dibawa ke sistem baru.
func TestResponsTidakPernahMemuatAlamatSurel(t *testing.T) {
	for _, jalur := range []string{
		"/api/master/ambang-komite",
		"/api/master/ambang-komite/integritas",
		"/api/komite/penjenjangan?nilai=75000000&lini=PA",
	} {
		t.Run(jalur, func(t *testing.T) {
			badan := panggil(t, serverUji(t), jalur).Body.String()
			require.NotContains(t, badan, "@", "respons memuat sesuatu yang menyerupai alamat surel")
		})
	}
}

// Ketujuh kasus pada spec.md diuji ULANG lewat HTTP, bukan hanya di domain.
//
// Pengulangan ini disengaja: yang diuji di sini bukan aturannya — itu sudah dibuktikan
// jenjang_test — melainkan bahwa aturan yang benar itu benar-benar SAMPAI ke peramban
// tanpa berubah di jalan.
func TestPenjenjanganLewatHTTPSesuaiSpec(t *testing.T) {
	server := serverUji(t)

	kasus := []struct {
		nama  string
		jalur string
		mau   int
	}{
		{"PA Rp 5.000.000", "/api/komite/penjenjangan?nilai=5000000&lini=PA", 1},
		{"PA Rp 75.000.000", "/api/komite/penjenjangan?nilai=75000000&lini=PA", 3},
		{"PA Rp 150.000.000", "/api/komite/penjenjangan?nilai=150000000&lini=PA", 4},
		{"Travel Rp 150.000.000", "/api/komite/penjenjangan?nilai=150000000&lini=TRAVEL", 3},
		{"Non-MBU Rp 80.000.000", "/api/komite/penjenjangan?nilai=80000000&lini=NONMBU", 2},
		{"Non-MBU Rp 750.000.000", "/api/komite/penjenjangan?nilai=750000000&lini=NONMBU", 2},
		{"Non-MBU Rp 2.000.000.000", "/api/komite/penjenjangan?nilai=2000000000&lini=NONMBU", 3},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			rekaman := panggil(t, server, k.jalur)
			require.Equal(t, http.StatusOK, rekaman.Code)

			respons := bacaPenjenjangan(t, rekaman)
			require.Equal(t, k.mau, respons.TierCount)
			require.Len(t, respons.Approvers, k.mau)
			require.False(t, respons.NoApprovers)
		})
	}
}

func TestPenjenjanganMelaporkanPitaDanKeraguanUrutan(t *testing.T) {
	server := serverUji(t)

	t.Run("Non-MBU berpita dan urutannya tidak pasti", func(t *testing.T) {
		respons := bacaPenjenjangan(t, panggil(t, server,
			"/api/komite/penjenjangan?nilai=80000000&lini=NONMBU"))

		require.True(t, respons.UsesBand)
		require.Equal(t, "1", respons.Band)
		require.True(t, respons.AmbiguousOrder,
			"dua baris ber-DEGREE 1 harus dilaporkan supaya tidak terbaca sebagai cacat")
	})

	t.Run("PA tidak berpita", func(t *testing.T) {
		respons := bacaPenjenjangan(t, panggil(t, server,
			"/api/komite/penjenjangan?nilai=150000000&lini=PA"))

		require.False(t, respons.UsesBand)
		require.Empty(t, respons.Band)
		require.False(t, respons.AmbiguousOrder)
	})
}

// Urutan dan alasan setiap penyetuju ikut dikirim, supaya layar dapat menjelaskan
// KENAPA seseorang masuk daftar — bukan hanya menampilkan hasilnya.
func TestPenyetujuMembawaUrutanAlasanDanAsalBarisnya(t *testing.T) {
	respons := bacaPenjenjangan(t, panggil(t, serverUji(t),
		"/api/komite/penjenjangan?nilai=75000000&lini=PA"))

	require.Len(t, respons.Approvers, 3)
	for i, p := range respons.Approvers {
		require.Equal(t, i+1, p.Order)
		require.NotEmpty(t, p.Name)
		require.NotEmpty(t, p.OperatorID)
		require.NotEmpty(t, p.ThresholdID, "asal barisnya harus dapat ditelusuri")
		require.NotEmpty(t, p.LowerBound)
	}
	require.Equal(t, "0.00", respons.Approvers[0].LowerBound)
	require.Equal(t, "10000001.00", respons.Approvers[1].LowerBound)
	require.Equal(t, "50000001.00", respons.Approvers[2].LowerBound)
}

// "Lini tidak ada di master" dan "ada tetapi tidak ada jenjang yang cocok" dijawab
// BERBEDA, dan pembedaannya disengaja.
func TestLiniTidakDikenalDibedakanDariTanpaPenyetuju(t *testing.T) {
	server := serverUji(t)

	t.Run("lini tidak ada menjadi 404", func(t *testing.T) {
		rekaman := panggil(t, server, "/api/komite/penjenjangan?nilai=1000000&lini=MARINE")
		require.Equal(t, http.StatusNotFound, rekaman.Code)

		var galat komitehttp.ErrorResponse
		require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &galat))
		require.Equal(t, komitehttp.CodeUnknownLine, galat.Code)
	})

	// Lini yang ADA di master selalu dijawab 200, termasuk pada nilai nol.
	//
	// Catatan atas master yang berlaku: tidak ada satu pun kombinasi nilai dan lini yang
	// benar-benar menghasilkan nol penyetuju, karena setiap tangga punya anak tangga
	// berambang bawah nol. Keadaan "tanpa penyetuju" karena itu diuji di tingkat domain
	// dengan master buatan — lihat TestLiniTidakDikenalDibedakanDariTanpaPenyetuju pada
	// jenjang_test.
	t.Run("lini yang ada dijawab 200", func(t *testing.T) {
		rekaman := panggil(t, server, "/api/komite/penjenjangan?nilai=0&lini=NONMBU")
		require.Equal(t, http.StatusOK, rekaman.Code)

		respons := bacaPenjenjangan(t, rekaman)
		require.Equal(t, 1, respons.TierCount,
			"anak tangga berambang nol tetap ikut pada nilai nol")
	})
}

func TestNilaiCacatDijawab400DanLiniKosongDijawab422(t *testing.T) {
	server := serverUji(t)

	t.Run("nilai bukan angka", func(t *testing.T) {
		rekaman := panggil(t, server, "/api/komite/penjenjangan?nilai=lima&lini=PA")
		require.Equal(t, http.StatusBadRequest, rekaman.Code,
			"bentuk permintaannya yang salah, bukan isinya")

		var galat komitehttp.ErrorResponse
		require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &galat))
		require.Equal(t, komitehttp.CodeInvalidValue, galat.Code)
	})

	// Pemisah ribuan ditolak dengan sengaja — artinya berbeda antar bahasa.
	t.Run("nilai berpemisah ribuan", func(t *testing.T) {
		rekaman := panggil(t, server, "/api/komite/penjenjangan?nilai=50.000.000&lini=PA")
		require.Equal(t, http.StatusBadRequest, rekaman.Code)
	})

	t.Run("lini kosong", func(t *testing.T) {
		rekaman := panggil(t, server, "/api/komite/penjenjangan?nilai=1000000&lini=")
		require.Equal(t, http.StatusUnprocessableEntity, rekaman.Code,
			"permintaannya berbentuk benar, isinya yang melanggar aturan")

		var galat komitehttp.ErrorResponse
		require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &galat))
		require.Equal(t, komitehttp.CodeValidationFailed, galat.Code)
		require.NotEmpty(t, galat.Details, "field yang salah harus disebut supaya layar dapat menandainya")
	})
}

// Cacat pada master dijawab 200, bukan galat: ia TEMUAN yang dilaporkan endpoint ini,
// dan menjawabnya dengan galat akan membuat layar menampilkan halaman gagal justru pada
// saat ia paling perlu menampilkan isinya.
func TestIntegritasMelaporkanTemuanDenganStatus200(t *testing.T) {
	rekaman := panggil(t, serverUji(t), "/api/master/ambang-komite/integritas")
	require.Equal(t, http.StatusOK, rekaman.Code)

	var respons komitehttp.IntegrityResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))

	// Master yang berlaku bersih dari cacat.
	require.Zero(t, respons.DefectCount)

	// Tetapi peringatannya ada, dan itulah nilainya: atap tangga PA dan Travel, serta
	// dua baris Non-MBU ber-DEGREE sama.
	require.NotZero(t, respons.WarningCount)
	require.Len(t, respons.Findings, respons.DefectCount+respons.WarningCount)

	adaAtapPA := false
	for _, t2 := range respons.Findings {
		if t2.BusinessLine == "PA" && t2.Kind == "atap_tangga" {
			adaAtapPA = true
			require.Equal(t, "peringatan", t2.Severity)
			require.NotEmpty(t, t2.ThresholdIDs, "temuan harus dapat ditelusuri ke barisnya")
		}
	}
	require.True(t, adaAtapPA, "atap tangga PA di Rp 200.000.000 harus dilaporkan")
}

// Penjenjangan TIDAK mengubah apa pun, sehingga GET adalah metode yang benar dan
// hasilnya dapat ditautkan. Metode yang mengubah data tidak boleh tersedia sama sekali.
func TestHanyaGetYangTersedia(t *testing.T) {
	server := serverUji(t)

	for _, metode := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(metode, func(t *testing.T) {
			rekaman := httptest.NewRecorder()
			server.ServeHTTP(rekaman, httptest.NewRequest(metode, "/api/master/ambang-komite", nil))
			require.Equal(t, http.StatusMethodNotAllowed, rekaman.Code,
				"master ambang komite dibaca saja selama masa paralel")
		})
	}
}

// serverSimasnet membentuk peladen dengan kebijakan portal Simasnet: satu penyetuju,
// diacak, penginput dikecualikan.
//
// Data ambangnya SENGAJA dibuat: baris Simasnet hidup di POOLDATA milik server Simasnet,
// sedangkan `Database/emailkomite.csv` berasal dari POOLDATA server ASM — lihat catatan
// pada kolamSimasnet di komite/mode_test.go.
func serverSimasnet(t *testing.T) http.Handler {
	t.Helper()

	baris := func(id, nama, operator string) komite.Threshold {
		return komite.Threshold{
			ID:            id,
			Name:          nama,
			OperatorID:    operator,
			BusinessLine:  "SIMASNET",
			CommitteeType: "1",
			LowerBound:    money.FromRupiah(0),
			UpperBound:    money.FromRupiah(3_000_000),
			Tier:          1,
			Active:        true,
			ForAdjustment: true,
		}
	}

	kebijakan := komite.SimasnetPolicy()
	service, err := usecase.NewService(usecase.Options{
		Repo: memory.NewRepo(
			baris("101", "Petugas A", "USERA"),
			baris("102", "Petugas B", "USERB"),
			baris("103", "Petugas C", "USERC"),
		),
		Policy: &kebijakan,
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

	handler := komitehttp.NewHandler(komitehttp.Options{
		Service: service, Logger: logger, WriteResponse: tulisJSON,
	})

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) { komitehttp.Mount(api, handler) })
	return router
}

// Mode dikirim ke layar, bukan dibiarkan disimpulkan dari panjang daftar penyetuju.
func TestModeDikirimPadaKeduaRespons(t *testing.T) {
	daftar := panggil(t, serverUji(t), "/api/master/ambang-komite")
	var master komitehttp.ThresholdListResponse
	require.NoError(t, json.Unmarshal(daftar.Body.Bytes(), &master))
	require.Equal(t, "kumulatif", master.Mode)

	hasil := bacaPenjenjangan(t, panggil(t, serverUji(t),
		"/api/komite/penjenjangan?nilai=75000000&lini=PA"))
	require.Equal(t, "kumulatif", hasil.Mode)
}

// Jalur Simasnet memilih TEPAT SATU penyetuju dan melaporkan seluruh kandidatnya.
//
// Kandidat dilaporkan karena yang dapat diperiksa pada mode ini bukan SIAPA yang terpilih
// — itu acak — melainkan apakah KUMPULAN yang layak sudah benar.
func TestJalurSimasnetMemilihSatuDanMelaporkanKandidat(t *testing.T) {
	hasil := bacaPenjenjangan(t, panggil(t, serverSimasnet(t),
		"/api/komite/penjenjangan?nilai=1000000&lini=SIMASNET"))

	require.Equal(t, "satu-penyetuju", hasil.Mode)
	require.Equal(t, 1, hasil.TierCount)
	require.Len(t, hasil.Candidates, 3)
	require.Empty(t, hasil.ExcludedApplicant)
}

// Operator yang mengajukan dikecualikan, dan pengecualiannya DILAPORKAN — bukan terjadi
// diam-diam. Inilah aturan yang mencegah seseorang menyetujui pengajuannya sendiri.
func TestPenginputDikecualikanDanDilaporkan(t *testing.T) {
	hasil := bacaPenjenjangan(t, panggil(t, serverSimasnet(t),
		"/api/komite/penjenjangan?nilai=1000000&lini=SIMASNET&penginput=USERA"))

	require.Equal(t, "USERA", hasil.ExcludedApplicant)
	require.Len(t, hasil.Candidates, 2)
	for _, k := range hasil.Candidates {
		require.NotEqual(t, "USERA", k.OperatorID)
	}
	require.NotEqual(t, "USERA", hasil.Approvers[0].OperatorID)
}

// Pada mode kumulatif, penginput JUGA dikecualikan — Work Owner, 2026-09-18.
//
// Akibatnya jumlah penyetuju berkurang satu bila penginputnya kebetulan anggota komite,
// dan yang tersingkir dilaporkan supaya sebabnya terlihat.
func TestPenginputJugaDikecualikanPadaModeKumulatif(t *testing.T) {
	hasil := bacaPenjenjangan(t, panggil(t, serverUji(t),
		"/api/komite/penjenjangan?nilai=80000000&lini=NONMBU&penginput=ELLENSUPRIYATI"))

	require.Equal(t, 1, hasil.TierCount, "dari dua penyetuju, satu tersingkir")
	require.Equal(t, "INDRAGUNAWAN", hasil.Approvers[0].OperatorID)
	require.Equal(t, "ELLENSUPRIYATI", hasil.ExcludedApplicant)
	require.Len(t, hasil.Excluded, 1)
	require.Empty(t, hasil.Candidates, "kandidat hanya bermakna pada mode satu-penyetuju")
}

// Keadaan yang paling perlu terlihat: seluruh penyetuju tersingkir karena penginputnya
// satu-satunya yang berwenang.
//
// Dijawab 200 beserta penandanya, bukan galat — ia TEMUAN tentang isi master yang harus
// dilihat Work Owner, dan pada master yang berlaku setiap lini punya jenjang terendah
// yang diisi satu orang saja.
func TestSeluruhPenyetujuDapatTersingkirDanSebabnyaTerlihat(t *testing.T) {
	rekaman := panggil(t, serverUji(t),
		"/api/komite/penjenjangan?nilai=20000000&lini=NONMBU&penginput=ELLENSUPRIYATI")
	require.Equal(t, http.StatusOK, rekaman.Code)

	hasil := bacaPenjenjangan(t, rekaman)
	require.True(t, hasil.NoApprovers)
	require.Len(t, hasil.Excluded, 1,
		"sebabnya harus terlihat, bukan tampak seperti master yang berlubang")
	require.Equal(t, "ELLENSUPRIYATI", hasil.Excluded[0].OperatorID)
}
