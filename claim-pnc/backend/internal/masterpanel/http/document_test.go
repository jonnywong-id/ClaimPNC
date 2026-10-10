package masterpanelhttp_test

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
)

const documentRoute = "/api/master/panel/" + approvedID + "/dokumen"

// upload mengirim satu berkas sebagai multipart, persis seperti peramban mengirimkannya.
func (p *testServer) upload(
	t *testing.T, path, portalAlias, fileName, content, note string,
) (*http.Response, map[string]any) {
	t.Helper()

	var body strings.Builder
	writer := multipart.NewWriter(&stringWriter{&body})
	if fileName != "" {
		part, err := writer.CreateFormFile("berkas", fileName)
		require.NoError(t, err)
		_, err = part.Write([]byte(content))
		require.NoError(t, err)
	}
	if note != "" {
		require.NoError(t, writer.WriteField("catatan", note))
	}
	require.NoError(t, writer.Close())

	request, err := http.NewRequest(http.MethodPost, p.server.URL+path, strings.NewReader(body.String()))
	require.NoError(t, err)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		request.Header.Set("X-Portal", portalAlias)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	raw, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	content_ := map[string]any{}
	if len(raw) > 0 {
		require.NoError(t, json.Unmarshal(raw, &content_))
	}
	return response, content_
}

// stringWriter menjadikan strings.Builder dapat dipakai multipart.Writer.
type stringWriter struct{ b *strings.Builder }

func (w *stringWriter) Write(p []byte) (int, error) { return w.b.Write(p) }

func TestUnggahDokumenBerhasilDanMengembalikanDataID(t *testing.T) {
	server := newTestServer(t)

	response, body := server.upload(t, documentRoute, "ASM", "panel.pdf", "isi berkas", "Foto panel")

	require.Equal(t, http.StatusCreated, response.StatusCode)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok, "respons harus membawa objek data")
	require.NotEmpty(t, data["data_id"])
	require.NotEmpty(t, data["image_id"])
	require.Equal(t, "panel.pdf", data["nama_berkas"])
	require.Equal(t, "Foto panel", data["catatan"])
	require.Equal(t, approvedID, data["id_panel"])
	require.Equal(t, loginName, data["diunggah_oleh"],
		"INPUTOPERATOR adalah satu-satunya jejak siapa yang mengunggah")

	require.Len(t, server.uploader.Uploaded, 1)
	require.Equal(t, "ASM", server.uploader.Uploaded[0].Portal,
		"berkas harus masuk ke basis data portal yang sedang dibuka")
}

// Respons TIDAK membawa URL berkasnya. URL beserta masa berlakunya dimiliki modul dokumen
// penunjang; menyalinnya ke sini akan membuat layar memegang tautan yang masa berlakunya
// sudah lewat tanpa ada yang tahu.
func TestResponsDokumenTidakMembawaURL(t *testing.T) {
	server := newTestServer(t)

	_, body := server.upload(t, documentRoute, "ASM", "panel.pdf", "isi", "")

	data := body["data"].(map[string]any)
	require.NotContains(t, data, "url")
	require.NotContains(t, data, "berlaku_sampai")
}

func TestDokumenDapatDibacaKembali(t *testing.T) {
	server := newTestServer(t)
	_, created := server.upload(t, documentRoute, "ASM", "panel.pdf", "isi", "Catatan")

	response, body := server.call(t, http.MethodGet, documentRoute, "ASM", "")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t,
		created["data"].(map[string]any)["data_id"],
		body["data"].(map[string]any)["data_id"])
}

func TestPanelTanpaDokumenMenjawabTidakDitemukan(t *testing.T) {
	server := newTestServer(t)

	response, body := server.call(t, http.MethodGet,
		"/api/master/panel/"+pendingID+"/dokumen", "ASM", "")

	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "dokumen_belum_ada", body["kode"])
}

func TestUnggahTanpaBagianBerkasDitolak(t *testing.T) {
	server := newTestServer(t)

	response, body := server.upload(t, documentRoute, "ASM", "", "", "Catatan saja")

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "unggah_tidak_sah", body["kode"])
	require.Empty(t, server.uploader.Uploaded)
}

// Permintaan tanpa portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan.
// Jatuh ke koneksi bawaan berarti dokumen sebuah badan hukum ditulis ke basis data badan
// hukum lain (`R-20`).
func TestUnggahTanpaPortalDitolak(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.upload(t, documentRoute, "", "panel.pdf", "isi", "")

	require.GreaterOrEqual(t, response.StatusCode, http.StatusBadRequest)
	require.Empty(t, server.uploader.Uploaded,
		"tidak satu berkas pun boleh terkirim tanpa portal yang sah")
}

// Entitas lain tidak punya panel itu, sehingga unggahannya harus ditolak SEBELUM berkasnya
// terkirim ke mana pun.
func TestUnggahKePanelEntitasLainDitolak(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.upload(t, documentRoute, "ASI", "panel.pdf", "isi", "")

	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Empty(t, server.uploader.Uploaded)
}

func TestLayananPenyimpananMatiMenjawabTidakTersedia(t *testing.T) {
	server := newTestServer(t)
	server.uploader.Failure = &masterpanel.DocumentUploadError{
		Kind:    masterpanel.UploadUnavailable,
		Message: "Layanan penyimpanan dokumen sedang tidak dapat dihubungi.",
		Err:     errors.New("koneksi terputus"),
	}

	response, body := server.upload(t, documentRoute, "ASM", "panel.pdf", "isi", "")

	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "layanan_unggah_tidak_tersedia", body["kode"])
}

// Berkas yang terlalu besar dijawab 413, dan pesannya menyebut ukurannya — bukan
// "berkas tidak ditemukan" yang menyesatkan.
func TestBerkasTerlaluBesarMenjawabEmpatBelasTiga(t *testing.T) {
	server := newTestServer(t)
	server.uploader.Failure = &masterpanel.DocumentUploadError{
		Kind:    masterpanel.UploadTooLarge,
		Message: "Ukuran berkas melebihi batas 20 MB. Perkecil berkasnya lalu unggah ulang.",
	}

	response, body := server.upload(t, documentRoute, "ASM", "panel.pdf", "isi", "")

	require.Equal(t, http.StatusRequestEntityTooLarge, response.StatusCode)
	require.Equal(t, "berkas_terlalu_besar", body["kode"])
	require.Contains(t, body["pesan"], "MB")
}

// Unggahan yang separuh jalan dijawab 500, BUKAN 503. Perbedaannya menentukan: 503
// mengundang pengulangan, dan mengulang di sini menumpuk berkas ganda di layanan
// penyimpanan.
func TestUnggahSeparuhJalanBukanLimaRatusTiga(t *testing.T) {
	server := newTestServer(t)
	server.uploader.Failure = &masterpanel.DocumentUploadError{
		Kind:    masterpanel.UploadHalfDone,
		Message: "Berkas sudah terkirim tetapi catatannya gagal disimpan.",
	}

	response, body := server.upload(t, documentRoute, "ASM", "panel.pdf", "isi", "")

	require.Equal(t, http.StatusInternalServerError, response.StatusCode)
	require.Equal(t, "unggah_separuh_jalan", body["kode"])
}

// Tanpa pengunggah, modul TETAP berjalan penuh dan hanya jalur unggah yang mati —
// beserta sebabnya, dan layar dapat menanyakannya SEBELUM menggambar tombol.
func TestTanpaPengunggahSisaModulTetapBerjalan(t *testing.T) {
	server := newTestServerWithoutUploader(t)

	list, _ := server.call(t, http.MethodGet, "/api/master/panel?status=1", "ASM", "")
	require.Equal(t, http.StatusOK, list.StatusCode, "daftar harus tetap dapat dibuka")

	options, body := server.call(t, http.MethodGet, "/api/master/panel/pilihan", "", "")
	require.Equal(t, http.StatusOK, options.StatusCode)
	require.Equal(t, false, body["unggah_tersedia"],
		"layar harus dapat mengetahui jalur unggah mati sebelum pengguna memilih berkas")

	response, content := server.upload(t, documentRoute, "ASM", "panel.pdf", "isi", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "layanan_unggah_tidak_tersedia", content["kode"])
}

func TestPilihanMenyatakanUnggahTersedia(t *testing.T) {
	server := newTestServer(t)

	_, body := server.call(t, http.MethodGet, "/api/master/panel/pilihan", "", "")

	require.Equal(t, true, body["unggah_tersedia"])
}

// Unggahan berikutnya MENGGANTI yang sebelumnya — satu panel memegang satu dokumen
// (`PANEL_HE.DOKUMENID`). Ini perilaku Pega, dan layar menyatakannya sebelum pengguna
// memilih berkas.
func TestUnggahKeduaMenggantiYangPertama(t *testing.T) {
	server := newTestServer(t)

	_, first := server.upload(t, documentRoute, "ASM", "awal.pdf", "isi", "")
	_, second := server.upload(t, documentRoute, "ASM", "revisi.pdf", "isi", "")

	require.NotEqual(t,
		first["data"].(map[string]any)["data_id"],
		second["data"].(map[string]any)["data_id"])

	_, current := server.call(t, http.MethodGet, documentRoute, "ASM", "")
	require.Equal(t, "revisi.pdf", current["data"].(map[string]any)["nama_berkas"])
}
