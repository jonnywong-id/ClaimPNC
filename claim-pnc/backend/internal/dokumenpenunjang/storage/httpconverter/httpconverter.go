// Package httpconverter memenuhi seam dokumenpenunjang.Converter terhadap layanan konversi
// gambar internal.
//
// # Bentuk yang ditiru
//
// `Connect REST/KonversiAvif-ConnectREST.xml`:
//
//	POST   <alamat>/convert-avif
//	Header Content-Type: application/json
//	Auth   pyUseAuthentication = false
//	Body   DocAviff_JSON.JSON  ← hasil @GetPageJSONString() atas halaman DocAviff
//
// Halaman `DocAviff` hanya punya satu field (`Activity/Convert_Avif-Act.xml:694`), sehingga
// muatannya `{"image_base64": "..."}`.
//
// Responsnya dibaca pada tiga field: `Status`, `ErrorMessage`, dan `image_base64`
// (`:1242`, `:1408`, `:1502`).
//
// # Alamatnya `http://`, bukan `https://` — dan itu dicatat, bukan diperbaiki diam-diam
//
// `pyBaseURL` pada rule itu memakai HTTP polos. Artinya isi berkas — termasuk dokumen
// nasabah — melintas jaringan **tanpa enkripsi**, dan tanpa otentikasi pula.
//
// Tidak diubah di sini karena mengubahnya akan membuat permintaannya gagal bila layanan
// belum melayani TLS, dan itu mematikan unggahan sama sekali. Yang dilakukan: alamatnya
// dibuat **dapat diganti lewat konfigurasi**, sehingga Tim Infra dapat mengarahkannya ke
// HTTPS tanpa rilis ulang. Ia satu golongan dengan 18 dari 21 Connect REST
// ber-`pyUseAuthentication=false` yang `D-73` catat.
package httpconverter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// JalurKonversi adalah jalur sumber daya pada layanan konversi.
//
// Disalin dari `pyResourcePath`, yang satu-satunya parameter jalurnya `pyMapFrom=CONSTANT`.
const JalurKonversi = "/convert-avif"

// Client mengirim berkas ke layanan konversi.
type Client struct {
	alamat string
	http   *http.Client
}

// Options adalah bahan pembentuk Client.
type Options struct {
	// Alamat adalah pangkal layanan, TANPA JalurKonversi. Wajib.
	Alamat string

	// BatasWaktu membatasi satu konversi. Bawaannya 60 detik — sama dengan unggah, dengan
	// alasan yang sama: muatannya membawa berkas.
	BatasWaktu time.Duration

	// HTTPClient boleh diisi untuk pengujian.
	HTTPClient *http.Client
}

// NewClient membentuk Client.
func NewClient(o Options) (*Client, error) {
	alamat := strings.TrimRight(strings.TrimSpace(o.Alamat), "/")
	if alamat == "" {
		return nil, errors.New(
			"dokumenpenunjang/httpconverter: alamat layanan konversi wajib diisi")
	}

	klien := o.HTTPClient
	if klien == nil {
		batas := o.BatasWaktu
		if batas <= 0 {
			batas = 60 * time.Second
		}
		klien = &http.Client{Timeout: batas}
	}
	return &Client{alamat: alamat, http: klien}, nil
}

// muatan adalah halaman `DocAviff` sebagaimana Pega men-serialisasinya.
type muatan struct {
	ImageBase64 string `json:"image_base64"`
}

// jawaban adalah bentuk respons layanan.
//
// `Hasil` sengaja TIDAK dibaca. Pega mengurainya menjadi halaman bersarang lewat satu step
// Java (`Activity/Convert_Avif-Act.xml:1178`), tetapi tidak satu pun nilainya dipakai
// sesudahnya — yang dipakai hanya `Status` dan `image_base64`. Menguraikannya di sini akan
// menambah bentuk yang tidak dibaca siapa pun.
type jawaban struct {
	Status       string `json:"Status"`
	ErrorMessage string `json:"ErrorMessage"`
	ImageBase64  string `json:"image_base64"`
}

// Convert memenuhi dokumenpenunjang.Converter.
func (c *Client) Convert(ctx context.Context, isi []byte) ([]byte, error) {
	badan, err := json.Marshal(muatan{
		ImageBase64: base64.StdEncoding.EncodeToString(isi),
	})
	if err != nil {
		return nil, fmt.Errorf("dokumenpenunjang/httpconverter: menyusun muatan: %w", err)
	}

	permintaan, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.alamat+JalurKonversi, bytes.NewReader(badan))
	if err != nil {
		return nil, fmt.Errorf(
			"dokumenpenunjang/httpconverter: menyusun permintaan: %w", err)
	}
	permintaan.Header.Set("Content-Type", "application/json")

	respons, err := c.http.Do(permintaan)
	if err != nil {
		return nil, fmt.Errorf(
			"dokumenpenunjang/httpconverter: menghubungi layanan konversi: %w", err)
	}
	defer respons.Body.Close()

	// Batasnya lebih besar daripada pada unggah: responsnya MEMBAWA berkas hasil dalam
	// base64, bukan sekadar beberapa field. Batas 20 MiB berkas menjadi ±27 MiB base64;
	// 64 MiB memberi ruang tanpa membuka pintu bagi respons yang tidak wajar.
	badanRespons, err := io.ReadAll(io.LimitReader(respons.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("dokumenpenunjang/httpconverter: membaca respons: %w", err)
	}

	if respons.StatusCode < 200 || respons.StatusCode > 299 {
		// Isi respons TIDAK ikut dikembalikan: ia dapat menggemakan muatan kita sendiri,
		// yang memuat berkas nasabah dalam base64.
		return nil, fmt.Errorf(
			"dokumenpenunjang/httpconverter: layanan menjawab %d", respons.StatusCode)
	}

	var hasil jawaban
	if err := json.Unmarshal(badanRespons, &hasil); err != nil {
		return nil, fmt.Errorf(
			"dokumenpenunjang/httpconverter: respons bukan JSON yang dikenali: %w", err)
	}

	// `Status == "false"` adalah penanda gagal menurut Pega
	// (`Activity/Convert_Avif-Act.xml:1408`). Perbandingannya TEKS, bukan boolean — itulah
	// yang dikirim layanan, dan mengubahnya menjadi boolean akan membuat `"false"` terbaca
	// sebagai nilai yang tidak dikenal.
	//
	// Perbandingannya tidak peka huruf: `"False"` dan `"FALSE"` menyatakan hal yang sama,
	// dan memperlakukannya sebagai berhasil akan meneruskan berkas kosong.
	if strings.EqualFold(strings.TrimSpace(hasil.Status), "false") {
		pesan := strings.TrimSpace(hasil.ErrorMessage)
		if pesan == "" {
			pesan = "layanan konversi menolak berkas tanpa menyebut sebabnya"
		}
		return nil, fmt.Errorf("dokumenpenunjang/httpconverter: %s", pesan)
	}

	isiHasil, err := base64.StdEncoding.DecodeString(strings.TrimSpace(hasil.ImageBase64))
	if err != nil {
		return nil, fmt.Errorf(
			"dokumenpenunjang/httpconverter: isi hasil konversi bukan base64 yang sah: %w", err)
	}
	return isiHasil, nil
}
