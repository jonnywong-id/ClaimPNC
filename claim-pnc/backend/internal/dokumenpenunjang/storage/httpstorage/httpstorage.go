// Package httpstorage memenuhi seam dokumenpenunjang.Storage terhadap layanan penyimpanan
// internal.
//
// # Bentuk yang ditiru
//
// `Connect REST/UploadDokumenPNC-ConnectREST.xml`:
//
//	POST   <alamat>/api/v1/upload
//	Header Content-Type: application/json
//	Auth   pyUseAuthentication = false
//	Body   DocAPI_JSON.JSON  ← hasil @GetPageJSONString() atas halaman DocAPI
//
// Muatannya karena itu adalah halaman `DocAPI` apa adanya, dengan delapan field yang
// disusun `Activity/InsertDokumenPNC-Act.xml`.
//
// # `pyUseAuthentication = false`, dan itu DIBAWA apa adanya
//
// Tidak ada header otentikasi, tidak ada token di muatan. Token yang dibuat
// `GENERAL.GET_TOKEN_STORAGE` hanya disisipkan ke tabel jejak dan tidak pernah dikirim —
// diperiksa menyeluruh pada rule Connect REST-nya.
//
// Ini kelemahan nyata dan bukan temuan modul ini: ia satu golongan dengan 18 dari 21
// Connect REST yang ber-`pyUseAuthentication=false` (`D-73`). Menambah otentikasi di sini
// akan membuat permintaannya ditolak layanan yang belum menuntutnya, jadi yang dilakukan
// adalah **membawanya apa adanya sambil mencatatnya**, bukan memperbaikinya diam-diam.
package httpstorage

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

	"claim-pnc/internal/dokumenpenunjang"
)

// JalurUnggah adalah jalur sumber daya pada layanan penyimpanan.
//
// Disalin dari `pyResourcePath` beserta ketiga `pyResourcePathParameters`-nya — `api`,
// `v1`, `upload` — yang seluruhnya `pyMapFrom=CONSTANT`. Jalurnya tetap; yang berubah antar
// lingkungan hanyalah alamat pangkalnya.
const JalurUnggah = "/api/v1/upload"

// Client mengirim berkas ke layanan penyimpanan.
type Client struct {
	alamat string
	http   *http.Client
}

// Options adalah bahan pembentuk Client.
type Options struct {
	// Alamat adalah pangkal layanan, TANPA JalurUnggah. Ia WAJIB dan tidak punya nilai
	// bawaan.
	//
	// # Kenapa tidak ada bawaan, padahal alamatnya diketahui
	//
	// Dua alasan yang berdiri sendiri. Pertama, `12-CROSSCUTTING.md` §3.4 melarang perilaku
	// yang bergantung pada alamat yang tertanam di kode — endpoint per lingkungan adalah
	// konfigurasi (`R-18`, yang lahir justru dari dua Connect REST yang menunjuk host
	// sandbox di ruleset produksi). Kedua, `D-69` melarang hostname produksi tertulis di
	// artefak yang di-commit.
	//
	// Bawaan yang "memudahkan" akan melanggar keduanya sekaligus, dan diam-diam mengirim
	// berkas nasabah ke alamat yang tidak seorang pun pilih.
	Alamat string

	// BatasWaktu membatasi satu unggahan. Bawaannya 60 detik — lebih panjang dari 30 detik
	// yang `09-API-STRATEGY.md` §8.2 tetapkan untuk pemanggilan biasa, karena muatannya
	// memuat berkas dan melintas jaringan internal.
	BatasWaktu time.Duration

	// HTTPClient boleh diisi untuk pengujian.
	HTTPClient *http.Client
}

// NewClient membentuk Client.
func NewClient(o Options) (*Client, error) {
	alamat := strings.TrimRight(strings.TrimSpace(o.Alamat), "/")
	if alamat == "" {
		return nil, errors.New(
			"dokumenpenunjang/httpstorage: alamat layanan penyimpanan wajib diisi")
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

// muatan adalah halaman `DocAPI` sebagaimana Pega men-serialisasinya.
//
// Nama field-nya HARUS sama persis dengan nama properti Pega — layanan penyimpanan membaca
// nama itu, bukan urutannya. Mengganti `NoClaim` menjadi `nomor_klaim` akan membuat
// layanannya menerima permintaan tanpa nomor klaim, dan tetap menjawab berhasil.
//
// `Durasi` bertipe angka, bukan teks. Pega merakit JSON-nya lewat `@GetPageJSONString()`
// yang menghasilkan `"Durasi":"0"`, lalu MEMBUANG tanda kutipnya dengan
// `@replaceAll(JSON, "\"0\"", 0)` (`Activity/InsertDokumenPNC-Act.xml:4032`). Dua langkah
// itu hanya diperlukan karena Pega tidak punya tipe angka pada halaman; di sini cukup satu.
type muatan struct {
	UserInput string `json:"UserInput"`
	NoClaim   string `json:"NoClaim"`
	App       string `json:"App"`
	Folder    string `json:"Folder"`
	NamaFile  string `json:"NamaFile"`
	Image     string `json:"Image"`
	MimeType  string `json:"MimeType"`
	Durasi    int    `json:"Durasi"`
}

// jawaban adalah bentuk respons layanan.
//
// Hanya dua field yang Pega baca: `ImageID` dan `exp`
// (`Activity/InsertDokumenPNC-Act.xml:4448`, `:4492`). Field lain diabaikan — dan itu
// disengaja: layanan boleh menambah field tanpa membuat modul ini gagal mengurai.
type jawaban struct {
	ImageID string `json:"ImageID"`
	Exp     string `json:"exp"`
	URL     string `json:"URLImage"`
	Folder  string `json:"appfolder"`
	Pesan   string `json:"message"`
}

// Upload memenuhi dokumenpenunjang.Storage.
func (c *Client) Upload(
	ctx context.Context,
	perintah dokumenpenunjang.PerintahUnggah,
) (dokumenpenunjang.HasilUnggah, error) {
	badan, err := json.Marshal(muatan{
		UserInput: perintah.Pengunggah,
		NoClaim:   perintah.NomorKlaim,
		App:       perintah.NamaAplikasi,
		Folder:    perintah.Folder,
		NamaFile:  perintah.NamaBerkas,
		Image:     base64.StdEncoding.EncodeToString(perintah.Isi),
		MimeType:  perintah.TipeMedia,
		Durasi:    0,
	})
	if err != nil {
		return dokumenpenunjang.HasilUnggah{}, fmt.Errorf(
			"dokumenpenunjang/httpstorage: menyusun muatan: %w", err)
	}

	permintaan, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.alamat+JalurUnggah, bytes.NewReader(badan))
	if err != nil {
		return dokumenpenunjang.HasilUnggah{}, fmt.Errorf(
			"dokumenpenunjang/httpstorage: menyusun permintaan: %w", err)
	}
	permintaan.Header.Set("Content-Type", "application/json")

	respons, err := c.http.Do(permintaan)
	if err != nil {
		return dokumenpenunjang.HasilUnggah{}, fmt.Errorf(
			"dokumenpenunjang/httpstorage: menghubungi layanan penyimpanan: %w", err)
	}
	defer respons.Body.Close()

	// Batas baca supaya respons yang tidak wajar tidak menghabiskan memori. 1 MiB jauh di
	// atas respons terbesar yang masuk akal untuk lima field.
	isi, err := io.ReadAll(io.LimitReader(respons.Body, 1<<20))
	if err != nil {
		return dokumenpenunjang.HasilUnggah{}, fmt.Errorf(
			"dokumenpenunjang/httpstorage: membaca respons: %w", err)
	}

	if respons.StatusCode < 200 || respons.StatusCode > 299 {
		// Isi respons TIDAK ikut dikembalikan ke pemanggil: ia dapat memuat apa saja,
		// termasuk gema muatan kita sendiri — yang berisi berkas nasabah. Yang dibawa
		// hanyalah kode statusnya; selebihnya urusan log.
		return dokumenpenunjang.HasilUnggah{}, fmt.Errorf(
			"dokumenpenunjang/httpstorage: layanan menjawab %d", respons.StatusCode)
	}

	var hasil jawaban
	if err := json.Unmarshal(isi, &hasil); err != nil {
		return dokumenpenunjang.HasilUnggah{}, fmt.Errorf(
			"dokumenpenunjang/httpstorage: respons bukan JSON yang dikenali: %w", err)
	}
	if strings.TrimSpace(hasil.ImageID) == "" {
		return dokumenpenunjang.HasilUnggah{}, errors.New(
			"dokumenpenunjang/httpstorage: respons tidak memuat ImageID")
	}

	return dokumenpenunjang.HasilUnggah{
		ImageID:   strings.TrimSpace(hasil.ImageID),
		URL:       strings.TrimSpace(hasil.URL),
		Folder:    strings.TrimSpace(hasil.Folder),
		ExpiresAt: uraiKedaluwarsa(hasil.Exp),
	}, nil
}

// uraiKedaluwarsa membaca `exp` dari respons.
//
// # Bentuknya tidak dijamin, jadi beberapa dicoba
//
// Pega tidak mengurainya sebagai waktu melainkan MENGOLAHNYA SEBAGAI TEKS
// (`Activity/InsertDokumenPNC-Act.xml:4495`):
//
//	@substring(@pxReplaceAllViaRegex(exp, "[-:]", ""), 0, 19) + " GMT"
//
// Membuang tanda hubung dan titik dua lalu memotong 19 karakter hanya masuk akal bila
// bentuknya menyerupai ISO-8601. Karena itu bentuk ISO dicoba lebih dulu.
//
// Gagal mengurai TIDAK menggagalkan unggahan: masa berlaku yang tidak terbaca membuat URL
// diperlakukan tetap berlaku (lihat Document.Kedaluwarsa), dan itu jauh lebih baik daripada
// membatalkan unggahan yang sebenarnya berhasil.
func uraiKedaluwarsa(nilai string) *time.Time {
	rapi := strings.TrimSpace(nilai)
	if rapi == "" {
		return nil
	}
	for _, bentuk := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"02/01/2006 15:04:05",
		"2006-01-02",
	} {
		if waktu, err := time.Parse(bentuk, rapi); err == nil {
			return &waktu
		}
	}
	return nil
}
