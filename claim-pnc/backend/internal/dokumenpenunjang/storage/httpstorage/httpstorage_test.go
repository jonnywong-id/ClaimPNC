package httpstorage_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/dokumenpenunjang/storage/httpstorage"
)

func perintah() dokumenpenunjang.PerintahUnggah {
	return dokumenpenunjang.PerintahUnggah{
		NamaAplikasi: "klaimpnc",
		Pengunggah:   "PETUGAS01",
		NomorKlaim:   "PNC-1865",
		KodeAkses:    "KODE-1",
		Folder:       "Doc/2026/09/",
		NamaBerkas:   "FotoKerugianpdf",
		TipeMedia:    "application/pdf",
		Isi:          []byte("%PDF-1.4 isi"),
	}
}

// TestMuatanSamaBentukDenganPega adalah uji terpenting berkas ini.
//
// Layanan penyimpanan membaca muatan menurut NAMA FIELD-nya. Mengganti satu nama tidak
// menghasilkan galat — layanan menerima permintaan itu dan menjawab berhasil, hanya dengan
// field yang bersangkutan kosong. Uji ini menahan setiap penggantian nama.
func TestMuatanSamaBentukDenganPega(t *testing.T) {
	var diterima map[string]any
	var jalur, tipeIsi string

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			jalur = r.URL.Path
			tipeIsi = r.Header.Get("Content-Type")
			isi, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(isi, &diterima)
			_, _ = w.Write([]byte(`{"ImageID":"IMG-1","URLImage":"https://penyimpanan.contoh/a","exp":"2026-12-31T23:59:59Z"}`))
		}))
	defer server.Close()

	klien, err := httpstorage.NewClient(httpstorage.Options{Alamat: server.URL})
	require.NoError(t, err)

	hasil, err := klien.Upload(context.Background(), perintah())
	require.NoError(t, err)

	require.Equal(t, "/api/v1/upload", jalur)
	require.Equal(t, "application/json", tipeIsi)

	// Kesembilan nama field, persis seperti halaman DocAPI di Pega — termasuk KodeString dari
	// GenerateTokenPNCDokumen, yang ditolak layanan bila kosong.
	require.Equal(t, "KODE-1", diterima["KodeString"])
	require.Equal(t, "PETUGAS01", diterima["UserInput"])
	require.Equal(t, "PNC-1865", diterima["NoClaim"])
	require.Equal(t, "klaimpnc", diterima["App"])
	require.Equal(t, "Doc/2026/09/", diterima["Folder"])
	require.Equal(t, "FotoKerugianpdf", diterima["NamaFile"])
	require.Equal(t, "application/pdf", diterima["MimeType"])
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("%PDF-1.4 isi")),
		diterima["Image"])
	require.Len(t, diterima, 9, "tidak boleh ada field kesepuluh yang ikut terkirim")

	require.Equal(t, "IMG-1", hasil.ImageID)
	require.NotNil(t, hasil.ExpiresAt)
}

// TestDurasiTerkirimSebagaiAngkaBukanTeks menjaga hasil akhir dua langkah Pega.
//
// Pega merakit `"Durasi":"0"` lalu MEMBUANG tanda kutipnya dengan `@replaceAll`. Yang
// benar-benar dikirim karena itu angka, bukan teks — dan uji ini menegaskan yang dikirim,
// bukan cara merakitnya.
func TestDurasiTerkirimSebagaiAngkaBukanTeks(t *testing.T) {
	var mentah []byte
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			mentah, _ = io.ReadAll(r.Body)
			_, _ = w.Write([]byte(`{"ImageID":"IMG-1","URLImage":"https://penyimpanan.contoh/a"}`))
		}))
	defer server.Close()

	klien, err := httpstorage.NewClient(httpstorage.Options{Alamat: server.URL})
	require.NoError(t, err)
	_, err = klien.Upload(context.Background(), perintah())
	require.NoError(t, err)

	require.Contains(t, string(mentah), `"Durasi":0`)
	require.NotContains(t, string(mentah), `"Durasi":"0"`)
}

// TestAlamatWajibDiisi menjaga larangan alamat tertanam di kode.
//
// `12-CROSSCUTTING.md` §3.4 dan `R-18`: endpoint per lingkungan adalah konfigurasi. Nilai
// bawaan akan melanggar itu sekaligus menuliskan hostname produksi di kode (`D-69`).
func TestAlamatWajibDiisi(t *testing.T) {
	for _, alamat := range []string{"", "   "} {
		_, err := httpstorage.NewClient(httpstorage.Options{Alamat: alamat})
		require.Error(t, err, "alamat %q harus ditolak", alamat)
	}
}

func TestGarisMiringDiUjungAlamatTidakMenggandakanJalur(t *testing.T) {
	var jalur string
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			jalur = r.URL.Path
			_, _ = w.Write([]byte(`{"ImageID":"IMG-1","URLImage":"https://penyimpanan.contoh/a"}`))
		}))
	defer server.Close()

	klien, err := httpstorage.NewClient(httpstorage.Options{Alamat: server.URL + "/"})
	require.NoError(t, err)
	_, err = klien.Upload(context.Background(), perintah())
	require.NoError(t, err)
	require.Equal(t, "/api/v1/upload", jalur)
}

// TestResponsTanpaURLImageDianggapGagal menutup keberhasilan semu.
//
// Pega menilai unggahan dari URLImage (precondition InsertDataPNCStorage). Tanpanya berkas
// tidak tersimpan; pesan layanan harus ikut supaya penolakannya dapat ditelusuri.
func TestResponsTanpaURLImageDianggapGagal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		}))
	defer server.Close()

	klien, _ := httpstorage.NewClient(httpstorage.Options{Alamat: server.URL})
	_, err := klien.Upload(context.Background(), perintah())
	require.Error(t, err)
	require.Contains(t, err.Error(), "URLImage")
	require.Contains(t, err.Error(), `"ok"`, "pesan layanan ikut ke galat")
}

// TestGalatTidakMembocorkanIsiRespons menjaga `D-69`.
//
// Respons galat dapat menggemakan muatan kita sendiri — yang memuat berkas nasabah dalam
// base64. Yang boleh naik ke pemanggil hanyalah kode statusnya.
func TestGalatTidakMembocorkanIsiRespons(t *testing.T) {
	const gema = "RAHASIA-ISI-BERKAS-NASABAH"
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(gema))
		}))
	defer server.Close()

	klien, _ := httpstorage.NewClient(httpstorage.Options{Alamat: server.URL})
	_, err := klien.Upload(context.Background(), perintah())
	require.Error(t, err)
	require.NotContains(t, err.Error(), gema)
	require.Contains(t, err.Error(), "500")
}

// TestMasaBerlakuYangTidakTerbacaTidakMenggagalkanUnggahan.
//
// Masa berlaku yang gagal diurai membuat URL diperlakukan tetap berlaku — jauh lebih baik
// daripada membatalkan unggahan yang sebenarnya berhasil.
func TestMasaBerlakuYangTidakTerbacaTidakMenggagalkanUnggahan(t *testing.T) {
	for _, exp := range []string{"", "entah kapan", "2026-12-31T23:59:59Z"} {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"ImageID":"IMG-1","URLImage":"https://penyimpanan.contoh/a","exp":"` + exp + `"}`))
			}))

		klien, _ := httpstorage.NewClient(httpstorage.Options{Alamat: server.URL})
		hasil, err := klien.Upload(context.Background(), perintah())
		server.Close()

		require.NoError(t, err, "exp %q tidak boleh menggagalkan unggahan", exp)
		require.Equal(t, "IMG-1", hasil.ImageID)
	}
}

func TestBatasWaktuDihormati(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond)
			_, _ = w.Write([]byte(`{"ImageID":"IMG-1","URLImage":"https://penyimpanan.contoh/a"}`))
		}))
	defer server.Close()

	klien, err := httpstorage.NewClient(httpstorage.Options{
		Alamat: server.URL, BatasWaktu: 20 * time.Millisecond,
	})
	require.NoError(t, err)

	_, err = klien.Upload(context.Background(), perintah())
	require.Error(t, err, "batas waktu wajib berlaku (09-API-STRATEGY.md §8.2)")
}
