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
	"sort"
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
//
// # Kenapa `Status` bertipe `any` dan bukan `string`
//
// Rule Pega membandingkannya sebagai TEKS (`:1408`), sehingga versi pertama paket ini
// mendeklarasikannya `string`. Layanan yang hidup ternyata mengirimkan BOOLEAN:
//
//	{"status": true, "type": "image", "image_base64": "..."}
//
// Diperiksa langsung ke layanannya pada 2026-10-13. Akibatnya `json.Unmarshal` gagal dengan
// "cannot unmarshal bool into Go struct field jawaban.status of type string", lalu Convert
// menolak respons yang sebenarnya BERHASIL — dan unggah PNG, JPG, JPEG, serta PDF ikut
// gagal seluruhnya.
//
// Seluruh uji paket ini lolos karena stub-nya menulis `{"Status":"true"}` — menyalin
// kontrak rule, bukan bentuk layanan yang hidup. Itulah sebabnya cacat ini tidak terlihat
// sampai layanannya benar-benar dihubungi.
//
// `any` menirukan `ErrorCode` pada `httpstorage.jawaban`, yang sudah lebih dulu memakainya
// untuk alasan yang sama: field yang tipe JSON-nya tidak dapat diandalkan. Menyempitkannya
// kembali menjadi `string` akan mengembalikan cacat yang sama.
type jawaban struct {
	Status       any    `json:"Status"`
	ErrorMessage string `json:"ErrorMessage"`
	ImageBase64  string `json:"image_base64"`
}

// menyatakanGagal membaca penanda kegagalan dari `Status`, apa pun tipe JSON-nya.
//
// Yang DIANGGAP gagal hanyalah nilai yang benar-benar menyatakan "false":
//
//	false     boolean, bentuk yang dikirim layanan sekarang
//	"false"   teks, bentuk yang dibandingkan rule Pega (`:1408`)
//
// Perbandingan teksnya tidak peka huruf dan dipangkas, sama seperti sebelumnya.
//
// Semua nilai lain — termasuk `Status` yang TIDAK DIKIRIM SAMA SEKALI — dianggap berhasil.
// Itu bukan kelonggaran melainkan aturan Pega apa adanya: yang dijaga hanyalah nilai
// `"false"`, dan berkas hasilnya tetap harus lolos pemeriksaan base64 di bawah.
func menyatakanGagal(status any) bool {
	switch nilai := status.(type) {
	case bool:
		return !nilai
	case string:
		return strings.EqualFold(strings.TrimSpace(nilai), "false")
	default:
		return false
	}
}

// fieldNames menyebut kunci tingkat atas respons JSON, terurut, tanpa nilainya.
//
// Hanya NAMANYA, tidak pernah nilainya: respons dapat menggemakan muatan kita sendiri, yang
// memuat berkas nasabah dalam base64. Menirukan helper senama di `httpstorage`.
func fieldNames(isi []byte) string {
	var peta map[string]json.RawMessage
	if err := json.Unmarshal(isi, &peta); err != nil {
		return "(bukan objek JSON)"
	}
	nama := make([]string, 0, len(peta))
	for k := range peta {
		nama = append(nama, k)
	}
	sort.Strings(nama)
	return strings.Join(nama, ", ")
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
		// Nama field-nya ikut disebut — bukan nilainya — supaya perubahan bentuk respons di
		// seberang terbaca langsung dari log, persis seperti pada `httpstorage`. Cacat
		// 2026-10-13 memakan waktu justru karena pesan lamanya tidak menyebutkan apa pun
		// tentang respons yang diterima.
		return nil, fmt.Errorf(
			"dokumenpenunjang/httpconverter: respons bukan JSON yang dikenali (field: %s): %w",
			fieldNames(badanRespons), err)
	}

	// Penanda gagal menurut Pega adalah `Status == "false"`
	// (`Activity/Convert_Avif-Act.xml:1408`). Layanan yang hidup mengirimkannya sebagai
	// boolean; menyatakanGagal menerima keduanya. Lihat doc `jawaban`.
	if menyatakanGagal(hasil.Status) {
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
