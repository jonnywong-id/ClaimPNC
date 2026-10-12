package inboxinvestigatorhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	portalhttp "claim-pnc/internal/portal/http"
)

// formRoute adalah jalur formulir kerja Investigator.
func formRoute(reference string) string {
	return route + "/" + reference + "/investigasi"
}

// callJSON menjalankan satu permintaan ber-badan JSON.
func callJSON(
	t *testing.T,
	p *testServer,
	method, path, portalAlias string,
	body any,
) (*http.Response, map[string]any) {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		require.NoError(t, err)
	}

	request, err := http.NewRequest(method, p.server.URL+path, bytes.NewReader(payload))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
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

// filledForm adalah badan permintaan formulir yang terisi minimal dan sah.
func filledForm() map[string]any {
	return map[string]any{
		"dapat_diinvestigasi": "1",
		"tempat_kejadian":     "1",
		"nama_rumah_sakit":    "RS Contoh",
		"hasil_investigasi":   "Keterangan contoh.",
	}
}

// Membuka formulir mengisi Tanggal Investigasi, dan menyebut keadaan tampilnya.
func TestOpenFormPresetsTheInvestigationDate(t *testing.T) {
	p := newTestServer(t)
	response, content := callJSON(t, p, http.MethodGet, formRoute("ref-1"), "ASM", nil)

	require.Equal(t, http.StatusOK, response.StatusCode)

	one, ok := content["investigasi"].(map[string]any)
	require.True(t, ok, "jawaban tidak memuat investigasi: %v", content)
	require.NotEmpty(t, one["tanggal_investigasi"])
	require.Equal(t, "ref-1", one["referensi"])

	shown, ok := content["tampil"].(map[string]any)
	require.True(t, ok, "jawaban tidak memuat keadaan tampil")
	require.Contains(t, shown, "nama_rumah_sakit")
}

// Keadaan tampil datang dari SERVER, dan mengikuti isian yang sudah tersimpan.
//
// Keenam syaratnya adalah aturan yang dibaca dari `pyCondition` section lama. Aturan yang
// hidup di dua tempat akan berbeda pada perubahan berikutnya.
func TestVisibilityFollowsTheStoredAnswers(t *testing.T) {
	p := newTestServer(t)

	body := filledForm()
	body["tempat_kejadian"] = "0" // bukan rumah sakit
	response, _ := callJSON(t, p, http.MethodPost, formRoute("ref-2"), "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	_, content := callJSON(t, p, http.MethodGet, formRoute("ref-2"), "ASM", nil)
	shown := content["tampil"].(map[string]any)

	require.Equal(t, false, shown["nama_rumah_sakit"])
	require.Equal(t, true, shown["nama_tempat_lainnya"])
	require.Equal(t, false, shown["alamat_rs_klinik"])
}

// Menyimpan formulir memindahkan klaim ke Analyst, dan jawabannya MENYEBUTKANNYA.
//
// Pekerjaan hilang dari antrean setelah ini; tanpa keterangan itu ia terbaca seperti data
// yang lenyap.
func TestSubmitMovesTheClaimAndSaysSo(t *testing.T) {
	p := newTestServer(t)
	response, content := callJSON(
		t, p, http.MethodPost, formRoute("ref-3"), "ASM", filledForm())

	require.Equal(t, http.StatusOK, response.StatusCode)

	move, ok := content["pindah"].(map[string]any)
	require.True(t, ok, "jawaban tidak memuat perpindahan: %v", content)
	require.Equal(t, "ref-3", move["referensi"])
	require.Equal(t, "5", move["status_survei"])
	require.Equal(t, "5", move["status_pnc"])
	require.Equal(t, "1151", move["status_klaim"])
	require.NotEmpty(t, move["pada"])
}

// Isian yang disimpan dapat dibaca kembali apa adanya.
func TestSubmittedAnswersCanBeReadBack(t *testing.T) {
	p := newTestServer(t)

	body := filledForm()
	body["nomor_rekam_medik"] = "RM-CONTOH-1"
	response, _ := callJSON(t, p, http.MethodPost, formRoute("ref-4"), "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	_, content := callJSON(t, p, http.MethodGet, formRoute("ref-4"), "ASM", nil)
	one := content["investigasi"].(map[string]any)

	require.Equal(t, "RM-CONTOH-1", one["nomor_rekam_medik"])
	require.Equal(t, "RS Contoh", one["nama_rumah_sakit"])
	require.Equal(t, "1", one["dapat_diinvestigasi"])
}

// Referensi diambil dari JALUR, bukan dari badan permintaan.
//
// Keduanya dapat berbeda, dan yang menang harus yang sudah melewati pemeriksaan portal —
// kalau tidak, satu permintaan dapat menyimpan hasil investigasi atas pekerjaan milik
// entitas lain (`R-20`).
func TestReferenceComesFromThePathNotTheBody(t *testing.T) {
	p := newTestServer(t)

	body := filledForm()
	body["referensi"] = "ref-milik-orang-lain"
	response, content := callJSON(
		t, p, http.MethodPost, formRoute("ref-5"), "ASM", body)

	require.Equal(t, http.StatusOK, response.StatusCode)
	move := content["pindah"].(map[string]any)
	require.Equal(t, "ref-5", move["referensi"])
}

// Formulir tanpa pilihan "Dapat Diinvestigasi" dijawab 400 beserta nama isiannya.
func TestSubmitRejectsAFormWithoutTheInvestigationChoice(t *testing.T) {
	p := newTestServer(t)

	body := filledForm()
	delete(body, "dapat_diinvestigasi")
	response, content := callJSON(
		t, p, http.MethodPost, formRoute("ref-6"), "ASM", body)

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
	require.Contains(t, content["pesan"], "Dapat Diinvestigasi")
}

// Tanggal yang tidak dapat dibaca dijawab 400 beserta nama isiannya — bukan disimpan
// sebagai tanggal nol.
func TestSubmitRejectsAnUnreadableDate(t *testing.T) {
	p := newTestServer(t)

	body := filledForm()
	body["tanggal_lahir"] = "01/09/2026"
	response, content := callJSON(
		t, p, http.MethodPost, formRoute("ref-7"), "ASM", body)

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Contains(t, content["pesan"], "Tanggal Lahir")
}

// Badan permintaan yang bukan JSON dijawab 400, bukan 500.
func TestSubmitRejectsAMalformedBody(t *testing.T) {
	p := newTestServer(t)

	request, err := http.NewRequest(http.MethodPost,
		p.server.URL+formRoute("ref-8"), bytes.NewReader([]byte("bukan json")))
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Formulir TIDAK dapat dibuka maupun disimpan tanpa menyebut portal.
//
// Isinya memuat data medis milik satu badan hukum (`FR-R2`, `R-20`).
func TestFormRequiresPortal(t *testing.T) {
	p := newTestServer(t)

	open, _ := callJSON(t, p, http.MethodGet, formRoute("ref-9"), "", nil)
	require.Equal(t, http.StatusBadRequest, open.StatusCode)

	submit, _ := callJSON(t, p, http.MethodPost, formRoute("ref-9"), "", filledForm())
	require.Equal(t, http.StatusBadRequest, submit.StatusCode)
}

// Formulir berada di balik middleware sesi, sama seperti daftarnya.
func TestFormRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	open, _ := callJSON(t, p, http.MethodGet, formRoute("ref-10"), "ASM", nil)
	require.Equal(t, http.StatusUnauthorized, open.StatusCode)

	submit, _ := callJSON(t, p, http.MethodPost, formRoute("ref-10"), "ASM", filledForm())
	require.Equal(t, http.StatusUnauthorized, submit.StatusCode)
}

// Hasil investigasi satu entitas TIDAK terlihat dari entitas lain.
//
// Ini uji kebocoran antarentitas pada jalur TULIS. Di sana akibatnya lebih berat daripada
// di daftar: yang tersimpan memuat nomor rekam medis (`R-20`, `FR-R2`).
func TestOneEntityNeverSeesAnothersInvestigation(t *testing.T) {
	p := newTestServer(t)

	body := filledForm()
	body["nomor_rekam_medik"] = "RM-MILIK-ASM"
	response, _ := callJSON(t, p, http.MethodPost, formRoute("ref-11"), "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	_, content := callJSON(t, p, http.MethodGet, formRoute("ref-11"), "ASI", nil)
	one := content["investigasi"].(map[string]any)
	require.Empty(t, one["nomor_rekam_medik"],
		"hasil investigasi ASM terbaca dari portal ASI")
}
