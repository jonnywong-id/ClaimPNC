package monitoringslinkojkhttp_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/monitoringslinkojk/repo/memory"
	"claim-pnc/internal/monitoringslinkojk/usecase"

	slinkhttp "claim-pnc/internal/monitoringslinkojk/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testPortal adalah entitas yang dipakai seluruh uji di berkas ini.
//
// Ia harus ada di daftar portal contoh DAN dinyatakan siap, karena middleware membedakan
// "portal tidak dikenal" dari "portal belum siap".
const testPortal = "ASM"

func testServer(t *testing.T, login string) http.Handler {
	t.Helper()
	handler, _ := testServerWithRepo(t, login)
	return handler
}

// testServerWithRepo mengembalikan repo-nya juga, untuk uji yang perlu memeriksa APA yang
// tersimpan — bukan sekadar berapa banyak.
func testServerWithRepo(t *testing.T, login string) (http.Handler, *memory.Repo) {
	t.Helper()

	repo := memory.NewSampleRepo()

	// Pemilih mengabaikan alias: yang diuji di berkas ini adalah lapisan transport, bukan
	// pemilihan basis data per entitas. Pemilihan itu diuji di usecase.
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (monitoringslinkojk.Repo, error) { return repo, nil },
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

	handler := slinkhttp.NewHandler(slinkhttp.Options{
		Service: service,
		GetCaller: func(context.Context) (slinkhttp.Caller, bool) {
			return slinkhttp.Caller{Login: login}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: slinkhttp.ErrorWriterFrom(writeError),
	})

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		slinkhttp.Mount(api, handler, portalDeps)
	})
	return router, repo
}

// get mengirim permintaan BESERTA header portal.
//
// Portalnya disertakan di sini, bukan di tiap uji, supaya uji yang sengaja
// menghilangkannya terlihat jelas sebagai pengecualian.
func get(t *testing.T, server http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Portal", testPortal)

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

// ============================================================================
// KETERANGAN LAYAR
// ============================================================================

func TestDescribeReturnsBothSegments(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"), "/api/monitoring-slink-ojk/keterangan")
	require.Equal(t, http.StatusOK, response.Code)

	var body struct {
		Segments []struct {
			Code            string `json:"kode"`
			Label           string `json:"label"`
			Column          int    `json:"jumlah_kolom"`
			AvailableColumn int    `json:"jumlah_kolom_tersedia"`
			Columns         []struct {
				Key       string `json:"kunci"`
				Header    string `json:"judul"`
				Available bool   `json:"tersedia"`
			} `json:"kolom"`
		} `json:"segmen"`
		Scopes []struct {
			Value string `json:"nilai"`
			Note  string `json:"keterangan"`
		} `json:"business_name"`
		RowKey string `json:"kunci_baris"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))

	require.Len(t, body.Segments, 2)
	require.Equal(t, "D01", body.Segments[0].Code)
	require.Equal(t, 20, body.Segments[0].Column)
	require.Equal(t, "F06", body.Segments[1].Code)
	require.Equal(t, 20, body.Segments[1].Column)
	require.Equal(t, 20, body.Segments[1].AvailableColumn)

	require.Equal(t, monitoringslinkojk.RowKeyColumn, body.RowKey)

	// Pilihan "SURETY BOND" WAJIB membawa keterangan: ia meniadakan Asuransi Kredit
	// alih-alih memilih Surety Bond, dan tanpa keterangan pengguna akan salah membaca
	// hasilnya sebagai daftar Surety Bond.
	var suretyNote string
	for _, scope := range body.Scopes {
		if scope.Value == string(monitoringslinkojk.ScopeSuretyBond) {
			suretyNote = scope.Note
		}
	}
	require.NotEmpty(t, suretyNote)
}

// ============================================================================
// PENCARIAN
// ============================================================================

func TestSearchDefaultsToSegmentD01(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"), "/api/monitoring-slink-ojk/data")
	require.Equal(t, http.StatusOK, response.Code)

	var body struct {
		Segment struct {
			Code string `json:"kode"`
		} `json:"segmen"`
		Rows []map[string]string `json:"baris"`
		Meta struct {
			Total     int `json:"total"`
			TotalPage int `json:"total_halaman"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))

	require.Equal(t, "D01", body.Segment.Code)
	require.Equal(t, 4, body.Meta.Total)
	require.Equal(t, 1, body.Meta.TotalPage)
	require.Len(t, body.Rows, 4)
}

// Setiap baris memuat SELURUH kunci kolom katalog, termasuk yang belum bersumber.
//
// Tanpa itu, tabel yang menggambar 38 kolom dari katalog menampilkan `undefined` pada
// kunci yang tidak ada — dan `undefined` di sel laporan regulator terbaca seperti cacat
// sistem alih-alih seperti kolom yang memang belum bersumber.
func TestSearchRowsCarryEveryCatalogKey(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"), "/api/monitoring-slink-ojk/data?segmen=F06")
	require.Equal(t, http.StatusOK, response.Code)

	var body struct {
		Rows []map[string]string `json:"baris"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.NotEmpty(t, body.Rows)

	row := body.Rows[0]
	for _, column := range monitoringslinkojk.Columns(monitoringslinkojk.SegmentF06) {
		_, exists := row[column.Key]
		require.Truef(t, exists, "kolom %q hilang dari baris", column.Key)
	}
	require.Contains(t, row, monitoringslinkojk.RowKeyColumn)
	require.Equal(t, "", row["nama_lengkap"],
		"kolom tanpa sumber dikirim sebagai teks kosong, bukan dihilangkan")
}

func TestSearchRejectsUnknownSegment(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"), "/api/monitoring-slink-ojk/data?segmen=D99")
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "segmen_tidak_dikenal")
}

// Isian tanggal yang cacat dijawab 422 beserta nama isiannya, bukan 400 tanpa keterangan.
func TestSearchRejectsMalformedDate(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/data?date_of_loss=01-03-2026")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)

	var body struct {
		Code   string `json:"kode"`
		Detail []struct {
			Field string `json:"field"`
		} `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, "validasi_gagal", body.Code)
	require.Equal(t, monitoringslinkojk.FieldDateOfLoss, body.Detail[0].Field)
}

func TestSearchRejectsReversedRange(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/data?date_of_loss=2026-03-31&date_of_request_document=2026-03-01")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
}

// Permintaan TANPA header portal ditolak, tidak pernah dilayani portal utama sebagai
// cadangan (`R-20`).
func TestRequestWithoutPortalIsRejected(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/monitoring-slink-ojk/data", nil)
	recorder := httptest.NewRecorder()
	testServer(t, "PELAPOR").ServeHTTP(recorder, request)

	require.NotEqual(t, http.StatusOK, recorder.Code)
}

func TestRequestWithoutCallerIsRejected(t *testing.T) {
	response := get(t, testServer(t, "   "), "/api/monitoring-slink-ojk/data")
	require.Equal(t, http.StatusConflict, response.Code)
	require.Contains(t, response.Body.String(), "profil_pemanggil_tidak_lengkap")
}

// ============================================================================
// EKSPOR
// ============================================================================

func TestExportWritesCSVWithEveryColumn(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"), "/api/monitoring-slink-ojk/ekspor")
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Header().Get("Content-Type"), "text/csv")
	// Nama berkasnya memang menyebut F06 meski isinya segmen D01 — ketertukaran itu
	// direplikasi dari Pega atas keputusan Work Owner 2026-09-26.
	require.Contains(t, response.Header().Get("Content-Disposition"), "Laporan F06 SLIK OJK")
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))

	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	require.NoError(t, err)

	expected := monitoringslinkojk.ExportHeaders(monitoringslinkojk.SegmentD01)
	require.Equal(t, expected, records[0])
	require.Len(t, records, 5, "satu baris kepala ditambah empat baris data contoh")

	// Segmen D01 SEJAJAR — 27 judul, 27 sel. Yang misalign hanya F06.
	for _, record := range records[1:] {
		require.Len(t, record, len(expected))
	}
}

// Berkas ekspor F06 SENGAJA tidak sejajar — 38 judul, 34 sel per baris.
//
// Ini replikasi `ExportDataSlinkFOG` apa adanya, diputuskan Work Owner 2026-09-26 sesudah
// selisihnya disampaikan beserta akibatnya. Uji ini membuktikan berkas yang dihasilkan
// benar-benar berbentuk demikian — bukan hanya katalognya.
func TestExportF06ReplicatesPegaMisalignment(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"), "/api/monitoring-slink-ojk/ekspor?segmen=F06")
	require.Equal(t, http.StatusOK, response.Code)

	// Nama berkasnya menyebut D01 meski isinya segmen F06 — tertukar, dan direplikasi.
	require.Contains(t, response.Header().Get("Content-Disposition"), "Laporan SLIK OJK D01")

	// FieldsPerRecord dimatikan: pembaca CSV baku MENOLAK baris yang panjangnya berbeda
	// dari baris kepala, dan di sini perbedaan itu justru yang sedang diuji.
	reader := csv.NewReader(strings.NewReader(response.Body.String()))
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	require.NoError(t, err)
	require.Len(t, records[0], 38, "baris kepala tetap 38 judul")

	for _, record := range records[1:] {
		require.Len(t, record, 34, "baris data hanya 34 sel — persis seperti Pega")
	}

	// Pergeserannya dinyatakan sebagai NILAI, bukan hanya sebagai panjang baris: isi
	// Alamat berada di sel ketujuh, yaitu di bawah judul "Jenis Kelamin".
	//
	// Diperiksa pada baris KEDUA, bukan pertama. Barisnya terurut tanggal registrasi
	// menurun, dan yang teratas adalah baris contoh yang alamatnya SENGAJA kosong —
	// memakainya akan membuat uji ini lulus meski pergeserannya hilang.
	require.Equal(t, "Jenis Kelamin", records[0][6])
	require.Equal(t, "Jalan Contoh Nomor 1", records[2][6])

	// Dan tidak ada nilai Alamat di bawah judul "Alamat" yang sebenarnya (sel ke-11) —
	// itulah akibat pergeserannya.
	require.Equal(t, "Alamat", records[0][10])
	require.Empty(t, records[2][10])
}

// Nol baris tetap menghasilkan berkas berisi kepala kolom, bukan 204.
//
// Pelapor yang menekan Export dan tidak menerima apa pun tidak punya cara membedakan
// "tidak ada data" dari "tombolnya rusak".
func TestExportWithoutMatchesStillWritesHeader(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/ekspor?date_of_loss=1990-01-01&date_of_request_document=1990-01-02")
	require.Equal(t, http.StatusOK, response.Code)

	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1)
}

// Penyaring cacat ditolak sebagai JSON — sebelum satu byte CSV pun terkirim.
func TestExportRejectsBadFilterAsJSON(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/ekspor?date_of_loss=bukan-tanggal")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Contains(t, response.Header().Get("Content-Type"), "application/json")
}

// ============================================================================
// FORMAT FILE
// ============================================================================

func TestTemplateFileIsAligned(t *testing.T) {
	response := get(t, testServer(t, "PELAPOR"), "/api/monitoring-slink-ojk/format")
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Header().Get("Content-Disposition"),
		monitoringslinkojk.TemplateFileName)

	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, monitoringslinkojk.TemplateColumns, records[0])
	require.Len(t, records[0], 25, "25 judul, sama dengan CSVPropHeaders di Pega")
	require.Len(t, records[1], len(records[0]),
		"baris contoh wajib sejajar dengan judulnya, seperti di Pega")
}

// ============================================================================
// TIGA TOMBOL YANG MENULIS
// ============================================================================

// post mengirim permintaan tulis beserta header portal.
func post(
	t *testing.T,
	server http.Handler,
	path string,
	body io.Reader,
	contentType string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("X-Portal", testPortal)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

// "Proses Data Klaim" menyusun laporan dari data klaim sumber.
//
// ============================================================================
// DUA FASILITAS PADA SATU KLAIM: yang kedua bertanda 'U', BUKAN 'C'
// ============================================================================
//
// Empat baris contoh disusun, tetapi hasilnya **3 baru + 1 diperbarui** — bukan 4 baru.
// Sebabnya ada di pencacahnya sendiri:
//
//	GetCountTClaimSlikOJK:  select count(1) ... where claimid = ?
//
// Ia menghitung per **claimid saja**, tanpa `contractno`. Data contoh memuat satu klaim
// dengan DUA fasilitas kredit (`PNCN.26.0101`, kontrak `KTR-2026-0101` dan `KTR-2026-0102`),
// sehingga begitu fasilitas pertama tersusun, fasilitas KEDUA dari klaim yang sama sudah
// terhitung "pernah ada" dan ditandai `'U'`.
//
// Perilaku itu ada di Pega dan DIREPLIKASI — menambahkan `contractno` ke pencacah akan
// mengubah isi kolom `operasidata` pada laporan regulator atas dasar tebakan.
//
// Angka 3 dan 1 dikunci di sini supaya perilakunya tidak berubah tanpa ada yang
// menyadarinya; versi pertama uji ini menuntut 4 dan gagal, dan kegagalannya yang
// menemukan perilaku ini.
func TestProcessCreatesReportRows(t *testing.T) {
	response := post(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/proses", nil, "")
	require.Equal(t, http.StatusOK, response.Code)

	var body struct {
		Created int `json:"baru"`
		Updated int `json:"diperbarui"`
		Total   int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))

	require.Equal(t, 3, body.Created,
		"tiga klaim berbeda; fasilitas kedua dari klaim yang sama tidak terhitung baru")
	require.Equal(t, 1, body.Updated,
		"fasilitas kedua klaim PNCN.26.0101 — pencacah menghitung per claimid saja")
	require.Equal(t, 4, body.Total, "keempat barisnya tetap tersusun")
}

// Penyusunan KEDUA menandai barisnya `'U'`, bukan `'C'`.
//
// Ini uji terpenting di antara aksi tulis: tanpa pencacah, klaim yang sudah pernah
// dilaporkan akan masuk laporan sebagai klaim BARU — dan OJK menerima satu klaim
// terhitung dua kali tanpa satu pun tanda.
func TestProcessMarksRepeatAsUpdate(t *testing.T) {
	server := testServer(t, "PELAPOR")

	require.Equal(t, http.StatusOK,
		post(t, server, "/api/monitoring-slink-ojk/proses", nil, "").Code)

	response := post(t, server, "/api/monitoring-slink-ojk/proses", nil, "")
	require.Equal(t, http.StatusOK, response.Code)

	var body struct {
		Created int `json:"baru"`
		Updated int `json:"diperbarui"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))

	require.Zero(t, body.Created, "tidak ada klaim yang benar-benar baru pada putaran kedua")
	require.Equal(t, 4, body.Updated)
}

// Penyaring yang tidak menemukan apa pun dijawab 422 BESERTA sebabnya.
//
// Dibedakan dari "berhasil menyusun nol baris": pelapor yang menekan tombol dan tidak
// melihat apa pun berubah tidak punya cara membedakan "tidak ada data" dari "tombolnya
// rusak".
func TestProcessWithoutSourceRowsIsRejected(t *testing.T) {
	response := post(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/proses?date_of_loss=1990-01-01&date_of_request_document=1990-01-02", nil, "")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Contains(t, response.Body.String(), "tidak_ada_data")
}

// ============================================================================
// UNGGAH BERKAS
// ============================================================================

// uploadBody menyusun badan permintaan multipart berisi satu berkas CSV.
func uploadBody(t *testing.T, isi string) (io.Reader, string) {
	t.Helper()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile("berkas", "unggahan.csv")
	require.NoError(t, err)
	_, err = part.Write([]byte(isi))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return &buffer, writer.FormDataContentType()
}

// Kepala kolom dibaca MENURUT NAMANYA, bukan menurut posisi.
//
// Berkas di bawah sengaja menaruh ContractNo SEBELUM ClaimID — urutan yang berbeda dari
// berkas "Format File". Bila nilainya dibaca menurut posisi, keduanya tertukar dan
// laporan regulator memuat nomor kontrak di kolom nomor klaim.
func TestUploadReadsHeadersByName(t *testing.T) {
	csvBody := "ContractNo,ClaimID,TUNGGAKAN,RecoveryClaim\n" +
		"KTR-9001,PNCN.26.9001,5000,1000\n"

	body, contentType := uploadBody(t, csvBody)
	response := post(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/unggah", body, contentType)
	require.Equal(t, http.StatusOK, response.Code)

	var result struct {
		Created int `json:"baru"`
		Total   int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.Equal(t, 1, result.Created)
	require.Equal(t, 1, result.Total)
}

// Ketiga kolom yang sempat terlewat benar-benar terisi dari berkas.
//
// `PNCUploadAutoClaimSlikOJK` menetapkan 27 properti; tiga di antaranya —
// `JumlahHariTunggakan`, `NoKTP`, `NPWPPerusahaan` — semula tidak ikut dibaca di sini,
// sehingga tiga kolom `T_CLAIM_SLIK_OJK` masuk kosong padahal berkasnya memuatnya.
//
// Uji ini juga mengunci kebalikannya: `nopolis` TETAP kosong pada jalur unggah, karena
// Pega memang tidak menetapkannya di sini.
func TestUploadFillsArrearsDaysAndIdentityColumns(t *testing.T) {
	csvBody := "ClaimID,ContractNo,JumlahHariTunggakan,NoKTP,NPWPPerusahaan,PolicyNo\n" +
		"PNCN.26.9001,KTR-9001,93,3201234567890001,012345678901000,POL-77\n"

	body, contentType := uploadBody(t, csvBody)
	server, repo := testServerWithRepo(t, "PELAPOR")
	response := post(t, server, "/api/monitoring-slink-ojk/unggah", body, contentType)
	require.Equal(t, http.StatusOK, response.Code)

	reports := repo.Reports()
	require.Len(t, reports, 1)

	entry := reports[0]
	require.Equal(t, "93", entry.ArrearsDays)
	require.Equal(t, "3201234567890001", entry.IDCardNo)
	require.Equal(t, "012345678901000", entry.CompanyNPWP)
	require.Empty(t, entry.PolicyNo, "jalur unggah Pega tidak menetapkan nopolis")
}

// Seluruh baris berkas ditulis, termasuk yang No Klaim-nya kosong.
//
// Penyaringan baris dicabut atas permintaan Work Owner (2026-09-27) supaya jalur ini
// sama dengan Pega, yang memang tidak memeriksa apa pun.
func TestUploadWritesEveryRow(t *testing.T) {
	csvBody := "ClaimID,ContractNo\n" +
		"PNCN.26.9001,KTR-9001\n" +
		",KTR-9002\n" + // No Klaim kosong — TETAP ditulis
		"PNCN.26.9003,KTR-9003\n"

	body, contentType := uploadBody(t, csvBody)
	response := post(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/unggah", body, contentType)
	require.Equal(t, http.StatusOK, response.Code)

	var result struct {
		Total   int `json:"total"`
		Skipped []struct {
			Line   int    `json:"baris"`
			Reason string `json:"alasan"`
		} `json:"ditolak"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))

	require.Equal(t, 3, result.Total)
	require.Empty(t, result.Skipped)
}

// Berkas yang kepala kolomnya tidak dikenali ditolak dengan petunjuk yang dapat dikerjakan.
func TestUploadRejectsUnknownHeader(t *testing.T) {
	body, contentType := uploadBody(t, "kolom_a,kolom_b\n1,2\n")
	response := post(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/unggah", body, contentType)

	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Contains(t, response.Body.String(), "Format File")
}

func TestUploadWithoutFileIsRejected(t *testing.T) {
	body, contentType := uploadBody(t, "")
	// Nama field sengaja BENAR tetapi isinya kosong; yang diuji adalah berkas tanpa
	// satu baris pun di bawah kepala kolomnya.
	response := post(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/unggah", body, contentType)
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
}

// ============================================================================
// PENGIRIMAN KE SLIK
// ============================================================================

// Tanpa seam pengirim, tombolnya menolak dengan 503 BESERTA sebabnya.
//
// 503, bukan 500: layanannya memang belum tersedia, dan itu keadaan yang diketahui —
// kontrak `Rest_SendDataClientBasedDebitur` tidak ada di export (`R-16`).
func TestSubmitWithoutSenderIsRejected(t *testing.T) {
	body := strings.NewReader(`{"no_klaim":"PNCN.26.0101","contract_no":"KTR-2026-0101"}`)
	response := post(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/kirim", body, "application/json")

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Contains(t, response.Body.String(), "layanan_slik_belum_siap")
}

// Badan KOSONG berarti "kirim seluruh yang sedang tampil", bukan permintaan cacat.
//
// Itulah bentuk yang dipakai tombol di layar. Ia tetap ditolak 503 selama seam pengirim
// belum dikonfigurasi — tetapi lewat jalur yang berbeda, dan uji ini yang memastikan
// jalurnya memang yang dimaksud.
func TestSubmitWithEmptyBodyTargetsFilteredRows(t *testing.T) {
	for _, body := range []io.Reader{
		nil,
		strings.NewReader(`{"no_klaim":"","contract_no":""}`),
	} {
		response := post(t, testServer(t, "PELAPOR"),
			"/api/monitoring-slink-ojk/kirim", body, "application/json")

		require.Equal(t, http.StatusServiceUnavailable, response.Code)
		require.Contains(t, response.Body.String(), "layanan_slik_belum_siap")
	}
}

// Satu isian terisi dan satunya kosong adalah permintaan CACAT, bukan permintaan
// "seluruhnya".
//
// Pembedaannya penting: pemanggil yang mengirim nomor klaim tanpa nomor kontrak sedang
// mencoba mengirim SATU klaim dan lupa satu isian — menafsirkannya sebagai "kirim
// seluruhnya" akan mengirim jauh lebih banyak daripada yang ia maksud.
func TestSubmitRejectsPartialClaim(t *testing.T) {
	body := strings.NewReader(`{"no_klaim":"PNCN.26.0101","contract_no":""}`)
	response := post(t, testServer(t, "PELAPOR"),
		"/api/monitoring-slink-ojk/kirim", body, "application/json")

	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Contains(t, response.Body.String(), "validasi_gagal")
	require.Contains(t, response.Body.String(), monitoringslinkojk.FieldContractNo)
}
