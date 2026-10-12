package httpconverter_test

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

	"claim-pnc/internal/dokumenpenunjang/storage/httpconverter"
)

// TestMuatanSamaBentukDenganPega menjaga kontrak dengan layanan konversi.
//
// Halaman `DocAviff` hanya punya satu field (`Activity/Convert_Avif-Act.xml:694`), dan
// layanan membaca muatan menurut NAMA field-nya. Mengganti namanya tidak menghasilkan
// galat — layanan menerima permintaan itu dan menjawab tanpa gambar.
func TestMuatanSamaBentukDenganPega(t *testing.T) {
	var diterima map[string]any
	var jalur, tipeIsi string

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			jalur = r.URL.Path
			tipeIsi = r.Header.Get("Content-Type")
			isi, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(isi, &diterima)
			_, _ = w.Write([]byte(`{"Status":"true","image_base64":"` +
				base64.StdEncoding.EncodeToString([]byte("hasil avif")) + `"}`))
		}))
	defer server.Close()

	klien, err := httpconverter.NewClient(httpconverter.Options{Alamat: server.URL})
	require.NoError(t, err)

	hasil, err := klien.Convert(context.Background(), []byte("gambar asli"))
	require.NoError(t, err)

	require.Equal(t, "/convert-avif", jalur)
	require.Equal(t, "application/json", tipeIsi)
	require.Equal(t,
		base64.StdEncoding.EncodeToString([]byte("gambar asli")),
		diterima["image_base64"])
	require.Len(t, diterima, 1, "hanya satu field yang dikirim, seperti halaman DocAviff")

	require.Equal(t, []byte("hasil avif"), hasil)
}

// TestStatusFalseDianggapGagal menjaga penanda kegagalan Pega.
//
// `Activity/Convert_Avif-Act.xml:1408` memakai perbandingan TEKS `Status=="false"`, bukan
// boolean. Memperlakukannya sebagai berhasil akan meneruskan berkas kosong.
func TestStatusFalseDianggapGagal(t *testing.T) {
	for _, status := range []string{"false", "False", "FALSE", " false "} {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"Status":"` + status +
					`","ErrorMessage":"berkas rusak","image_base64":""}`))
			}))

		klien, _ := httpconverter.NewClient(httpconverter.Options{Alamat: server.URL})
		_, err := klien.Convert(context.Background(), []byte("gambar"))
		server.Close()

		require.Error(t, err, "Status %q harus dianggap gagal", status)
		require.Contains(t, err.Error(), "berkas rusak",
			"pesan layanan diteruskan supaya sebabnya terbaca")
	}
}

// TestGagalTanpaPesanTetapMenyebutkanSesuatu.
//
// Galat tanpa pesan berakhir sebagai "konversi gagal: " di log — dan itu tidak menolong
// siapa pun yang menelusurinya.
func TestGagalTanpaPesanTetapMenyebutkanSesuatu(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"Status":"false"}`))
		}))
	defer server.Close()

	klien, _ := httpconverter.NewClient(httpconverter.Options{Alamat: server.URL})
	_, err := klien.Convert(context.Background(), []byte("gambar"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "tanpa menyebut sebabnya")
}

func TestStatusSelainFalseDianggapBerhasil(t *testing.T) {
	// Layanan tidak selalu mengirim `Status`; yang dijaga Pega hanyalah nilai `"false"`.
	for _, badan := range []string{
		`{"Status":"true","image_base64":"aGFzaWw="}`,
		`{"image_base64":"aGFzaWw="}`,
		`{"Status":"OK","image_base64":"aGFzaWw="}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(badan))
			}))

		klien, _ := httpconverter.NewClient(httpconverter.Options{Alamat: server.URL})
		hasil, err := klien.Convert(context.Background(), []byte("gambar"))
		server.Close()

		require.NoError(t, err, "badan %s", badan)
		require.Equal(t, []byte("hasil"), hasil)
	}
}

func TestAlamatWajibDiisi(t *testing.T) {
	for _, alamat := range []string{"", "   "} {
		_, err := httpconverter.NewClient(httpconverter.Options{Alamat: alamat})
		require.Error(t, err, "alamat %q harus ditolak", alamat)
	}
}

func TestGarisMiringDiUjungAlamatTidakMenggandakanJalur(t *testing.T) {
	var jalur string
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			jalur = r.URL.Path
			_, _ = w.Write([]byte(`{"image_base64":"aGFzaWw="}`))
		}))
	defer server.Close()

	klien, err := httpconverter.NewClient(httpconverter.Options{Alamat: server.URL + "/"})
	require.NoError(t, err)
	_, err = klien.Convert(context.Background(), []byte("gambar"))
	require.NoError(t, err)
	require.Equal(t, "/convert-avif", jalur)
}

// TestGalatTidakMembocorkanIsiRespons menjaga `D-69`.
//
// Respons galat dapat menggemakan muatan kita sendiri — yang memuat berkas nasabah dalam
// base64.
func TestGalatTidakMembocorkanIsiRespons(t *testing.T) {
	const gema = "RAHASIA-ISI-BERKAS-NASABAH"
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(gema))
		}))
	defer server.Close()

	klien, _ := httpconverter.NewClient(httpconverter.Options{Alamat: server.URL})
	_, err := klien.Convert(context.Background(), []byte("gambar"))
	require.Error(t, err)
	require.NotContains(t, err.Error(), gema)
	require.Contains(t, err.Error(), "502")
}

// TestHasilBukanBase64DianggapGagal menutup jalan bagi isi yang rusak.
//
// Meneruskan teks mentah sebagai isi berkas akan menyimpan berkas yang tidak dapat dibuka,
// dan tidak ada satu pun galat yang menandainya.
func TestHasilBukanBase64DianggapGagal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"image_base64":"ini bukan base64 %%%"}`))
		}))
	defer server.Close()

	klien, _ := httpconverter.NewClient(httpconverter.Options{Alamat: server.URL})
	_, err := klien.Convert(context.Background(), []byte("gambar"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "base64")
}

func TestBatasWaktuDihormati(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond)
			_, _ = w.Write([]byte(`{"image_base64":"aGFzaWw="}`))
		}))
	defer server.Close()

	klien, err := httpconverter.NewClient(httpconverter.Options{
		Alamat: server.URL, BatasWaktu: 20 * time.Millisecond,
	})
	require.NoError(t, err)

	_, err = klien.Convert(context.Background(), []byte("gambar"))
	require.Error(t, err, "batas waktu wajib berlaku")
}
