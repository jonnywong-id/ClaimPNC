package dashboardclaimhttp_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/repo/memory"
	"claim-pnc/internal/dashboardclaim/usecase"

	dashboardhttp "claim-pnc/internal/dashboardclaim/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// wib dipakai agar uji tidak bergantung pada basis data zona waktu mesin penjalan.
var wib = time.FixedZone("WIB", 7*60*60)

// Dua entitas dipakai di berkas ini, dan keduanya harus ada di daftar portal contoh SERTA
// dinyatakan siap — middleware membedakan "portal tidak dikenal" dari "portal belum siap".
const (
	portalUtama = "ASM"
	portalLain  = "ASI"
)

// testServer membentuk server dengan isi yang SAMA untuk setiap portal.
//
// Dipakai uji yang menguji bentuk respons dan penyaring, bukan pemisahan entitas.
func testServer(t *testing.T) http.Handler {
	t.Helper()

	repo := memory.NewRepo(memory.SampleOutstanding(), memory.SampleSurveys())
	closed := memory.NewClosedReader(memory.SampleClosed())

	return buildServer(t,
		func(string) (dashboardclaim.Repo, error) { return repo, nil },
		closed,
	)
}

// buildServer merakit rantai yang SAMA PERSIS dengan cmd/claimpnc.
//
// Rantai penulis galat portal ikut ditiru, bukan disederhanakan. Bila ia tidak ditiru, uji
// penolakan portal akan lulus di sini tetapi gagal di aplikasi sungguhan — galat portal
// dijawab 500 alih-alih 400, dan frontend tidak mengenalinya.
func buildServer(
	t *testing.T,
	selector dashboardclaim.RepoSelector,
	closed dashboardclaim.ClosedClaimReader,
) http.Handler {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: selector,
		ClosedClaim:  closed,
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}

	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode":  "galat_internal",
				"pesan": err.Error(),
			})
		},
		writeJSON,
	)

	handler, err := dashboardhttp.NewHandler(dashboardhttp.Options{
		Service:       service,
		Logger:        logger,
		Location:      wib,
		WriteResponse: writeJSON,
		WriteError:    dashboardhttp.WriteError(logger, writeJSON, dashboardhttp.ErrorWriter(writeError)),
	})
	require.NoError(t, err)

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{portalUtama, portalLain} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		dashboardhttp.Mount(api, handler, portalDeps)
	})
	return router
}

// get mengirim permintaan BESERTA header portal.
func get(t *testing.T, server http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	return getAs(t, server, path, portalUtama)
}

// getAs mengirim permintaan atas nama entitas tertentu.
func getAs(t *testing.T, server http.Handler, path, portal string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, path, nil)
	if portal != "" {
		request.Header.Set("X-Portal", portal)
	}

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

// decode membaca badan respons menjadi map.
func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

// total membaca jumlah seluruh baris dari keterangan halaman.
//
// Ia dipisahkan menjadi helper supaya uji tidak mengulang penelusuran dua tingkat, dan
// supaya bentuk keterangan halaman hanya diketahui di SATU tempat di berkas ini.
func total(t *testing.T, body map[string]any) float64 {
	t.Helper()

	halaman, ok := body["halaman"].(map[string]any)
	require.True(t, ok, "respons telusur wajib memuat keterangan halaman")
	return halaman["total"].(float64)
}

// TestRingkasanMengembalikanEmpatKartuBerurutan menjaga urutan kartu.
//
// Urutannya bukan selera tata letak melainkan urutan yang sudah dikenal pengguna —
// `param.tipe` 0…3 pada sistem lama (`D-13`). Karena itu ia ditentukan SERVER dan dijaga
// uji, bukan diserahkan pada frontend.
func TestRingkasanMengembalikanEmpatKartuBerurutan(t *testing.T) {
	server := testServer(t)

	recorder := get(t, server, "/api/dashboard-claim/ringkasan")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	cards, ok := body["kartu"].([]any)
	require.True(t, ok, "respons harus memuat larik kartu")
	require.Len(t, cards, 4)

	expected := []string{"outstanding", "close-claim", "loss-adjuster", "internal-surveyor"}
	for index, want := range expected {
		card := cards[index].(map[string]any)
		require.Equalf(t, want, card["tile"], "kartu ke-%d tidak pada urutannya", index+1)
	}

	require.Equal(t, portalUtama, body["portal"], "setiap respons wajib menyebut entitasnya")
}

// TestRingkasanMenghitungDariContohYangSama memastikan keempat angka benar-benar dihitung,
// bukan dikembalikan nol karena seam-nya tidak terpasang.
func TestRingkasanMenghitungDariContohYangSama(t *testing.T) {
	server := testServer(t)

	body := decode(t, get(t, server, "/api/dashboard-claim/ringkasan"))
	cards := body["kartu"].([]any)

	counts := map[string]float64{}
	for _, raw := range cards {
		card := raw.(map[string]any)
		counts[card["tile"].(string)] = card["jumlah"].(float64)
	}

	require.Equal(t, float64(len(memory.SampleOutstanding())), counts["outstanding"])
	require.Equal(t, float64(len(memory.SampleClosed())), counts["close-claim"],
		"tile Close Claim dibaca lewat seam — nol berarti seam-nya tidak terpasang")
	require.Equal(t, float64(2), counts["loss-adjuster"])
	require.Equal(t, float64(2), counts["internal-surveyor"])
}

// TestRingkasanMenyebutkanSelisihTerencana menjaga janji `D-54`.
//
// Selisih yang disengaja terhadap sistem lama WAJIB dinyatakan lebih dulu. Yang ditemukan
// tanpa dinyatakan akan dilaporkan sebagai cacat pada gerbang 1, dan menjelaskannya
// belakangan jauh lebih mahal.
func TestRingkasanMenyebutkanSelisihTerencana(t *testing.T) {
	server := testServer(t)

	body := decode(t, get(t, server, "/api/dashboard-claim/ringkasan"))
	differences, ok := body["selisih_terencana"].([]any)

	require.True(t, ok, "respons ringkasan wajib menyebutkan selisih terencana")
	require.NotEmpty(t, differences)
}

// TestRingkasanMemisahkanSelisihDariCatatanWarisan menjaga pemisahan yang menentukan.
//
// `selisih_terencana` berisi hal yang BERBEDA dari Pega dan menuntut persetujuan;
// `catatan_warisan` berisi hal yang SAMA dengan Pega dan hanya menuntut penjelasan.
//
// Menggabungkan keduanya akan membuat penguji gerbang 1 mencari selisih yang tidak ada —
// dan tiga butir teratas justru perilaku yang sengaja dipertahankan (keputusan Work Owner
// 2026-09-26).
func TestRingkasanMemisahkanSelisihDariCatatanWarisan(t *testing.T) {
	server := testServer(t)

	body := decode(t, get(t, server, "/api/dashboard-claim/ringkasan"))

	selisih, ok := body["selisih_terencana"].([]any)
	require.True(t, ok, "respons ringkasan wajib menyebutkan selisih terencana")

	warisan, ok := body["catatan_warisan"].([]any)
	require.True(t, ok, "respons ringkasan wajib menyebutkan catatan warisan")
	require.NotEmpty(t, warisan,
		"perbedaan angka kartu terhadap telusurnya wajib dinyatakan, bukan dibiarkan tampak sebagai cacat")

	// Kedua angka kartu survei TIDAK boleh lagi muncul sebagai selisih: keduanya kini
	// mengikuti Pega apa adanya.
	for _, raw := range selisih {
		require.NotContainsf(t, raw.(string), "BARIS SURVEI",
			"penyeragaman angka kartu sudah dicabut — ia tidak boleh lagi diumumkan sebagai selisih")
	}
}

// TestPenyaringLiniBisnisMenyaringKeempatKartu memastikan penyaring berlaku menyeluruh.
//
// Bila ia hanya berlaku pada sebagian kartu, pengguna membaca empat angka yang berasal dari
// populasi berbeda — dan tidak ada yang menandainya.
func TestPenyaringLiniBisnisMenyaringKeempatKartu(t *testing.T) {
	server := testServer(t)

	body := decode(t, get(t, server, "/api/dashboard-claim/ringkasan?lini_bisnis=PA"))
	require.Equal(t, "PA", body["lini_bisnis"])

	counts := map[string]float64{}
	for _, raw := range body["kartu"].([]any) {
		card := raw.(map[string]any)
		counts[card["tile"].(string)] = card["jumlah"].(float64)
	}

	// Data contoh memuat tepat satu klaim berjalan, satu klaim tutup, dan satu survei
	// adjuster ber-Group Panel '002'.
	require.Equal(t, float64(1), counts["outstanding"])
	require.Equal(t, float64(1), counts["close-claim"])
	require.Equal(t, float64(1), counts["loss-adjuster"])
	require.Equal(t, float64(0), counts["internal-surveyor"])
}

// TestGroupPanel009TidakTermasukNonMBU menjaga perilaku sistem lama yang mudah terlewat.
//
// `Activity/SetDashboardClaim-Act.xml` hanya menyebut '003', '004', dan '006' pada cabang
// NONMBU. Klaim ber-Group Panel '009' karena itu TIDAK tampil pada pilihan Non-MBU, meski
// `CONTEXT.md` mendaftarnya sebagai varian Aneka.
//
// Perilaku itu direplikasi sesuai `P-5`. Uji ini yang menjaganya tidak "diperbaiki" tanpa
// keputusan — dan yang membuat perbaikannya kelak menjadi perubahan yang terlihat.
func TestGroupPanel009TidakTermasukNonMBU(t *testing.T) {
	server := testServer(t)

	nonMBU := decode(t, get(t, server, "/api/dashboard-claim/outstanding?lini_bisnis=NONMBU"))
	semua := decode(t, get(t, server, "/api/dashboard-claim/outstanding?lini_bisnis=ALL"))

	require.Less(t, total(t, nonMBU), total(t, semua),
		"Group Panel 009 seharusnya tidak ikut pada pilihan Non-MBU")

	for _, raw := range nonMBU["klaim"].([]any) {
		claim := raw.(map[string]any)
		require.NotEqual(t, "PNCN.26.0005", claim["nomor_klaim"],
			"klaim ber-Group Panel 009 tidak boleh tampil pada pilihan Non-MBU")
	}
}

// TestTelusurKlaimMemakaiBentukKlaim memastikan tile bertipe klaim menggambar kolom klaim.
func TestTelusurKlaimMemakaiBentukKlaim(t *testing.T) {
	server := testServer(t)

	for _, tile := range []string{"outstanding", "close-claim"} {
		body := decode(t, get(t, server, "/api/dashboard-claim/"+tile))

		require.Equal(t, "klaim", body["bentuk"], "tile %s seharusnya bertipe klaim", tile)
		require.NotEmpty(t, body["klaim"], "tile %s seharusnya mengisi larik klaim", tile)
		require.Empty(t, body["survei"], "tile %s tidak boleh mengisi larik survei", tile)
	}
}

// TestTelusurSurveiMemakaiBentukSurvei memastikan tile bertipe survei menggambar kolom survei.
func TestTelusurSurveiMemakaiBentukSurvei(t *testing.T) {
	server := testServer(t)

	for _, tile := range []string{"loss-adjuster", "internal-surveyor"} {
		body := decode(t, get(t, server, "/api/dashboard-claim/"+tile))

		require.Equal(t, "survei", body["bentuk"], "tile %s seharusnya bertipe survei", tile)
		require.NotEmpty(t, body["survei"], "tile %s seharusnya mengisi larik survei", tile)
		require.Empty(t, body["klaim"], "tile %s tidak boleh mengisi larik klaim", tile)
	}
}

// TestKeduaLarikSelaluAdaMeskiKosong menjaga bentuk respons.
//
// Larik yang nil akan tersandi menjadi `null`, dan frontend yang memetakannya tanpa
// memeriksa null lebih dulu akan gagal di peramban — bukan saat diuji.
func TestKeduaLarikSelaluAdaMeskiKosong(t *testing.T) {
	server := testServer(t)

	body := decode(t, get(t, server, "/api/dashboard-claim/outstanding?cari=TIDAK-ADA-SAMA-SEKALI"))

	require.NotNil(t, body["klaim"], "larik klaim tidak boleh null")
	require.NotNil(t, body["survei"], "larik survei tidak boleh null")
	require.Equal(t, float64(0), total(t, body))
}

// TestTileTidakDikenalDijawab404 memastikan jalur yang salah ketik terlihat.
//
// 404, bukan 422: yang salah bukan isian melainkan jalurnya. Dan bukan pula diam-diam
// menampilkan tile lain.
func TestTileTidakDikenalDijawab404(t *testing.T) {
	server := testServer(t)

	recorder := get(t, server, "/api/dashboard-claim/tile-yang-tidak-ada")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, "tile_tidak_dikenal", decode(t, recorder)["kode"])
}

// TestJalurStatisTidakTerbacaSebagaiTile memastikan /penyaring dan /ringkasan tidak pernah
// jatuh ke rute telusur.
//
// Keduanya cocok dengan pola `/{tile}`, dan yang membedakannya hanyalah urutan pendaftaran
// di Mount. Yang diperiksa adalah `bentuk` — kunci yang HANYA dimiliki respons telusur.
// Memeriksa kunci `tile` tidak berguna: respons metadata memang memuat daftar tile, dan
// versi pertama uji ini gagal karena itu. Yang diperbaiki ujinya, bukan kodenya.
func TestJalurStatisTidakTerbacaSebagaiTile(t *testing.T) {
	server := testServer(t)

	expected := map[string]string{
		"penyaring": "lini_bisnis",
		"ringkasan": "kartu",
	}

	for path, marker := range expected {
		recorder := get(t, server, "/api/dashboard-claim/"+path)
		require.Equalf(t, http.StatusOK, recorder.Code, "jalur %s seharusnya dilayani rutenya sendiri", path)

		body := decode(t, recorder)
		require.Containsf(t, body, marker, "jalur %s tidak menjawab dengan bentuknya sendiri", path)
		require.NotContainsf(t, body, "bentuk", "jalur %s terbaca sebagai telusur tile", path)
	}
}

// TestLiniBisnisTidakDikenalDijawab422 memastikan isian yang salah ditolak, bukan diam-diam
// diperlakukan sebagai ALL.
//
// Memperlakukannya sebagai ALL akan menampilkan SELURUH pekerjaan saat pengguna mengira ia
// sedang menyaring — kegagalan yang tampak seperti data.
func TestLiniBisnisTidakDikenalDijawab422(t *testing.T) {
	server := testServer(t)

	recorder := get(t, server, "/api/dashboard-claim/ringkasan?lini_bisnis=TIDAK-ADA")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, "validasi_gagal", body["kode"])
	require.NotEmpty(t, body["detail"], "respons validasi wajib menyebut isian yang salah")
}

// TestSeluruhPelanggaranDikumpulkan menjaga `12-CROSSCUTTING` §1.2 butir 1.
//
// Mengembalikan satu pesan per percobaan akan menyiksa pengguna pada isian yang panjang, dan
// itu juga perilaku yang berbeda dari layar lama.
func TestSeluruhPelanggaranDikumpulkan(t *testing.T) {
	server := testServer(t)

	recorder := get(t, server, "/api/dashboard-claim/ringkasan?lini_bisnis=SALAH&halaman=banyak&ukuran=-5")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	detail := decode(t, recorder)["detail"].([]any)
	require.Len(t, detail, 3, "ketiga pelanggaran wajib dikembalikan sekaligus")
}

// TestPaginasiDibatasi memastikan permintaan halaman yang terlalu besar DITOLAK nilainya,
// bukan dipenuhi (`10-API-STRATEGY.md` §4).
func TestPaginasiDibatasi(t *testing.T) {
	server := testServer(t)

	body := decode(t, get(t, server, "/api/dashboard-claim/outstanding?ukuran=9999"))
	halaman := body["halaman"].(map[string]any)
	require.Equal(t, float64(dashboardclaim.MaxLimit), halaman["ukuran"])
}

// TestHalamanKeduaMelewatiHalamanPertama memastikan konversi halaman ke offset benar.
//
// Halaman dihitung mulai 1, sehingga halaman 1 tidak melewati satu baris pun. Salah satu
// off-by-one di sini akan MELEWATKAN baris pertama setiap kali — kegagalan yang tidak
// terlihat sebagai galat, hanya sebagai data yang kurang.
func TestHalamanKeduaMelewatiHalamanPertama(t *testing.T) {
	server := testServer(t)

	pertama := decode(t, get(t, server, "/api/dashboard-claim/outstanding?halaman=1&ukuran=2"))
	kedua := decode(t, get(t, server, "/api/dashboard-claim/outstanding?halaman=2&ukuran=2"))

	require.Equal(t, float64(1), pertama["halaman"].(map[string]any)["halaman"])
	require.Equal(t, float64(2), kedua["halaman"].(map[string]any)["halaman"])

	// Lima klaim contoh dengan ukuran 2 menghasilkan tiga halaman — dibulatkan KE ATAS.
	require.Equal(t, float64(3), pertama["halaman"].(map[string]any)["total_halaman"])

	awal := pertama["klaim"].([]any)
	lanjut := kedua["klaim"].([]any)
	require.Len(t, awal, 2)
	require.Len(t, lanjut, 2)

	require.NotEqual(t, awal[0].(map[string]any)["nomor_klaim"], lanjut[0].(map[string]any)["nomor_klaim"],
		"halaman kedua mengulang baris halaman pertama")
}

// TestHalamanKosongDiperlakukanSebagaiHalamanPertama memastikan layar yang baru dibuka tidak
// ditolak hanya karena belum menyebut halaman.
func TestHalamanKosongDiperlakukanSebagaiHalamanPertama(t *testing.T) {
	server := testServer(t)

	body := decode(t, get(t, server, "/api/dashboard-claim/outstanding"))
	require.Equal(t, float64(1), body["halaman"].(map[string]any)["halaman"])
}

// TestPermintaanTanpaPortalDitolak menjaga `R-20`.
//
// Permintaan tanpa header portal TIDAK PERNAH dilayani portal utama sebagai cadangan. Pada
// layar ini yang bocor bukan satu baris melainkan seluruh angka ringkasan satu badan hukum.
func TestPermintaanTanpaPortalDitolak(t *testing.T) {
	server := testServer(t)

	recorder := getAs(t, server, "/api/dashboard-claim/ringkasan", "")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "portal_tidak_disebut", decode(t, recorder)["kode"])
}

// TestEntitasTerpisah adalah uji yang paling menentukan di berkas ini.
//
// Dua entitas diberi isi yang BERBEDA, dan masing-masing hanya boleh melihat miliknya
// sendiri. Kegagalan di sini tidak tampak sebagai galat — layar tampil normal, angkanya
// masuk akal, dan yang salah hanya *milik siapa* data itu (`R-20`).
func TestEntitasTerpisah(t *testing.T) {
	berisi := memory.NewRepo(memory.SampleOutstanding(), memory.SampleSurveys())
	kosong := memory.NewRepo(nil, nil)

	server := buildServer(t,
		func(alias string) (dashboardclaim.Repo, error) {
			if alias == portalUtama {
				return berisi, nil
			}
			return kosong, nil
		},
		memory.NewClosedReader(nil),
	)

	utama := decode(t, getAs(t, server, "/api/dashboard-claim/outstanding", portalUtama))
	require.Greater(t, total(t, utama), float64(0))
	require.Equal(t, portalUtama, utama["portal"])

	lain := decode(t, getAs(t, server, "/api/dashboard-claim/outstanding", portalLain))
	require.Equal(t, float64(0), total(t, lain),
		"entitas lain tidak boleh melihat satu baris pun milik entitas pertama")
	require.Equal(t, portalLain, lain["portal"])
}

// TestGalatPemilihPortalTidakJatuhKeCadangan memastikan portal yang repo-nya bermasalah
// menghasilkan galat, BUKAN data portal lain.
func TestGalatPemilihPortalTidakJatuhKeCadangan(t *testing.T) {
	server := buildServer(t,
		func(string) (dashboardclaim.Repo, error) {
			return nil, fmt.Errorf("koneksi entitas belum siap")
		},
		memory.NewClosedReader(memory.SampleClosed()),
	)

	recorder := get(t, server, "/api/dashboard-claim/outstanding")
	require.Equal(t, http.StatusInternalServerError, recorder.Code,
		"galat pemilih portal wajib dikembalikan, bukan dijawab dengan data mana pun")
}

// TestCloseClaimTidakMenyentuhPemilihRepo memastikan tile yang dilayani seam tetap dapat
// dibaca meski repo modul ini sendiri bermasalah.
//
// Keduanya memang sumber yang berbeda, dan memilih repo lebih dulu akan menolak permintaan
// yang sebenarnya tidak membacanya.
func TestCloseClaimTidakMenyentuhPemilihRepo(t *testing.T) {
	server := buildServer(t,
		func(string) (dashboardclaim.Repo, error) {
			return nil, fmt.Errorf("koneksi entitas belum siap")
		},
		memory.NewClosedReader(memory.SampleClosed()),
	)

	recorder := get(t, server, "/api/dashboard-claim/close-claim")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotEmpty(t, decode(t, recorder)["klaim"])
}

// TestPenyaringMembentukLayarTanpaMenyentuhBasisData memastikan metadata tetap dijawab meski
// penyimpanan bermasalah.
//
// Isinya bentuk layar, bukan data — dan layar yang tidak dapat menggambar dropdown-nya
// menjadi tidak dapat dipakai sama sekali, bahkan untuk melihat galatnya.
func TestPenyaringMembentukLayarTanpaMenyentuhBasisData(t *testing.T) {
	server := buildServer(t,
		func(string) (dashboardclaim.Repo, error) {
			return nil, fmt.Errorf("koneksi entitas belum siap")
		},
		memory.NewClosedReader(nil),
	)

	recorder := get(t, server, "/api/dashboard-claim/penyaring")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Len(t, body["lini_bisnis"], 5)
	require.Len(t, body["tile"], 4)
}

// TestTanggalDiformatWIB memastikan konversi zona waktu terjadi SATU KALI di lapisan ini.
//
// Waktu disimpan UTC (`F-5`). Contoh registrasi pukul 03:00 UTC menjadi 10:00 WIB pada hari
// yang sama, sehingga **jamnya** yang membuktikan konversinya terjadi — bukan tanggalnya.
//
// Sejak 2026-10-07 kolom ini membawa jam, karena layar Pega menggambarnya
// (`24 Jan 20 14:54:24`). Uji ini dulu justru mematok bentuk "tanggal saja" sebagai yang
// benar; sekarang ia mematok jamnya, dan dengan begitu ia memeriksa hal yang lebih kuat:
// konversi yang meleset beberapa jam dulu lolos selama ia tidak menyeberangi tengah malam.
//
// `tanggal_kejadian` diperiksa berdampingan karena ia SENGAJA tetap tanggal saja — ia
// tanggal, bukan stempel waktu. Tanpa baris itu, "semua kolom tanggal diberi jam" akan
// lolos sebagai perubahan yang tidak disengaja.
func TestTanggalDiformatWIB(t *testing.T) {
	server := testServer(t)

	body := decode(t, get(t, server, "/api/dashboard-claim/outstanding"))
	first := body["klaim"].([]any)[0].(map[string]any)

	require.Equal(t, "2026-08-05 10:00:00", first["tanggal_pendaftaran"],
		"03:00 UTC harus tampil sebagai 10:00 WIB — jamnya yang membuktikan konversinya")

	require.Regexp(t, `^(\d{4}-\d{2}-\d{2})?$`, first["tanggal_kejadian"],
		"tanggal kejadian tetap tanggal saja, bukan stempel waktu")
}

// Rincian klaim dibaca lewat rutenya sendiri, bukan ikut pada daftar.
//
// Di Pega ia popup yang datanya diambil SAAT diklik (`setDataViewKlaim_Act` →
// `ViewTempDetailClaim`, `pyTarget: popup`). Menyertakan dokumen JSON pada setiap baris
// daftar akan membawa puluhan dokumen klaim pada setiap pemuatan halaman.
func TestClaimDetailRoute(t *testing.T) {
	server := testServer(t)

	body := decode(t, get(t, server, "/api/dashboard-claim/klaim/CONTOH-BERJALAN-1"))

	require.Equal(t, "CONTOH-BERJALAN-1", body["klaim_id"])
	require.NotNil(t, body["dokumen"], "dokumen null memaksa layar menjaga satu keadaan lagi")
}

// Rute rincian TIDAK boleh tertangkap pola "/{tile}".
//
// chi memilih menurut urutan pendaftaran. Bila "/klaim/{id}" dipasang sesudah "/{tile}",
// permintaannya dijawab sebagai tile bernama "klaim" — dan jawabannya 404 tile tidak dikenal,
// galat yang menyesatkan karena menyebut hal yang tidak diminta siapa pun.
func TestClaimDetailRouteIsNotSwallowedByTile(t *testing.T) {
	server := testServer(t)

	recorder := get(t, server, "/api/dashboard-claim/klaim/CONTOH-BERJALAN-1")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.NotContains(t, body, "tile", "jawabannya rincian klaim, bukan daftar tile")
}

// Klaim yang tidak ada dijawab 404, bukan 500.
func TestClaimDetailMissingClaim(t *testing.T) {
	server := testServer(t)

	recorder := get(t, server, "/api/dashboard-claim/klaim/TIDAK-ADA")
	require.Equal(t, http.StatusNotFound, recorder.Code)
}
