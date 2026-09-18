// Package kasir memenuhi seam masterrekening.Kasir dengan panggilan HTTP nyata.
//
// # Apa yang digantikan
//
// Dua Connect-REST pada sistem lama, dipanggil dari activity HitDataRekeningToKasir
// dan HitupdateDataRekeningToKasir:
//
//	InjectDataRekeningToKasir        rekening yang baru disetujui
//	UpdateSearchDataRekeningToKasir  rekening yang menggantikan rekening lama
//
// # Yang belum dapat dibuktikan
//
// Alamat dan kredensial kedua service itu TIDAK ada di dalam export rule — keduanya
// berada di konfigurasi instans Pega, yang tidak ikut terekspor. Karena itu adapter ini
// lengkap dan dapat dijalankan, tetapi BELUM PERNAH diuji terhadap sistem Kasir yang
// sesungguhnya. Bentuk badan permintaan di bawah disusun dari properti yang disalin
// activity lama ke MyServicePage; bila Kasir menuntut bentuk lain, yang berubah hanya
// berkas ini.
//
// Penghalangnya dicatat di docs/keputusan-implementasi.md dan menunggu Tim Infra.
package kasir

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/masterrekening"
)

// Konfigurasi adalah alamat dan kredensial sistem Kasir.
type Konfigurasi struct {
	// URLDaftar menerima rekening yang baru disetujui.
	URLDaftar string
	// URLPerbarui menerima rekening yang menggantikan rekening lama.
	URLPerbarui string

	Pengguna string
	Sandi    string

	// Tenggang membatasi lama menunggu jawaban Kasir. Nol berarti memakai nilai baku.
	Tenggang time.Duration
}

// Lengkap menyatakan konfigurasi ini cukup untuk menghubungi Kasir.
func (k Konfigurasi) Lengkap() bool {
	return strings.TrimSpace(k.URLDaftar) != "" && strings.TrimSpace(k.URLPerbarui) != ""
}

const tenggangBaku = 30 * time.Second

// Klien menghubungi sistem Kasir lewat HTTP.
type Klien struct {
	konf Konfigurasi
	http *http.Client
}

// KlienBaru membentuk klien Kasir.
func KlienBaru(k Konfigurasi) *Klien {
	tenggang := k.Tenggang
	if tenggang <= 0 {
		tenggang = tenggangBaku
	}
	return &Klien{konf: k, http: &http.Client{Timeout: tenggang}}
}

// permintaan adalah badan yang dikirim ke Kasir.
//
// Nama fieldnya mengikuti properti yang disalin activity lama ke MyServicePage. Ia
// sengaja DTO tersendiri, terpisah dari masterrekening.Rekening: bentuk kawat milik
// sistem lain tidak boleh menyandera bentuk domain kita.
type permintaan struct {
	NomorRekening string `json:"no_rekening"`
	NamaPemilik   string `json:"nama_rekening"`
	KodeBank      string `json:"kode_bank"`
	NamaBank      string `json:"nama_bank"`
	CabangBank    string `json:"cabang_bank"`
	AlamatBank    string `json:"alamat_bank"`
	TipeRekening  string `json:"tipe_rekening"`
	NIK           string `json:"nik"`
	Email         string `json:"email"`
	Telepon       string `json:"telepon"`
	IDDokumen     string `json:"id_dokumen"`

	// Tiga field berikut hanya terisi pada jalur pembaruan; Kasir memakainya untuk
	// menemukan rekening mana yang digantikan.
	KodeBankLama      string `json:"kode_bank_lama,omitempty"`
	NomorRekeningLama string `json:"no_rekening_lama,omitempty"`
	NamaPemilikLama   string `json:"nama_rekening_lama,omitempty"`
}

// jawaban adalah bentuk respons Kasir.
//
// ReponseCode dieja begitu — dengan huruf "s" yang hilang — karena begitulah ejaannya
// di MyServicePage.ClaimData.ServiceKasir.ReponseCode pada rule lama. Salah ketik yang
// sudah menjadi kontrak tidak dapat diperbaiki sepihak; memperbaikinya di sini berarti
// membaca field yang tidak pernah dikirim (bandingkan 03-CURRENT-ARCHITECTURE §4.7).
type jawaban struct {
	KodeRespons string `json:"ReponseCode"`
	Pesan       string `json:"ResponseMessage"`
	IDRekening  string `json:"IdRekening"`
}

// Daftarkan mendaftarkan rekening baru ke Kasir.
func (k *Klien) Daftarkan(ctx context.Context, r masterrekening.Rekening) (masterrekening.HasilKasir, error) {
	return k.kirim(ctx, k.konf.URLDaftar, badanDari(r, false))
}

// Perbarui memberi tahu Kasir bahwa sebuah rekening menggantikan rekening lama.
func (k *Klien) Perbarui(ctx context.Context, r masterrekening.Rekening) (masterrekening.HasilKasir, error) {
	return k.kirim(ctx, k.konf.URLPerbarui, badanDari(r, true))
}

func badanDari(r masterrekening.Rekening, denganYangLama bool) permintaan {
	p := permintaan{
		NomorRekening: r.NomorRekening,
		NamaPemilik:   r.NamaPemilik,
		KodeBank:      r.KodeBank,
		NamaBank:      r.NamaBank,
		CabangBank:    r.CabangBank,
		AlamatBank:    r.AlamatBank,
		TipeRekening:  r.TipeRekening,
		NIK:           r.NIK,
		Email:         r.Email,
		Telepon:       r.Telepon,
		IDDokumen:     r.IDDokumen,
	}
	if denganYangLama {
		p.KodeBankLama = r.KodeBankLama
		p.NomorRekeningLama = r.NomorRekeningLama
		p.NamaPemilikLama = r.NamaPemilikLama
	}
	return p
}

func (k *Klien) kirim(ctx context.Context, url string, badan permintaan) (masterrekening.HasilKasir, error) {
	if strings.TrimSpace(url) == "" {
		return masterrekening.HasilKasir{}, fmt.Errorf("masterrekening/kasir: alamat service belum dikonfigurasi")
	}

	isi, err := json.Marshal(badan)
	if err != nil {
		return masterrekening.HasilKasir{}, fmt.Errorf("masterrekening/kasir: menyusun permintaan: %w", err)
	}

	permintaanHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(isi))
	if err != nil {
		return masterrekening.HasilKasir{}, fmt.Errorf("masterrekening/kasir: menyusun permintaan HTTP: %w", err)
	}
	permintaanHTTP.Header.Set("Content-Type", "application/json")
	permintaanHTTP.Header.Set("Accept", "application/json")
	if k.konf.Pengguna != "" {
		permintaanHTTP.SetBasicAuth(k.konf.Pengguna, k.konf.Sandi)
	}

	respons, err := k.http.Do(permintaanHTTP)
	if err != nil {
		// Galatnya dibungkus tanpa menyertakan URL: alamat service kadang memuat
		// token di query string, dan galat berakhir di log.
		return masterrekening.HasilKasir{}, fmt.Errorf("masterrekening/kasir: menghubungi sistem Kasir: %w", err)
	}
	defer func() { _ = respons.Body.Close() }()

	var j jawaban
	if err := json.NewDecoder(respons.Body).Decode(&j); err != nil {
		return masterrekening.HasilKasir{}, fmt.Errorf("masterrekening/kasir: membaca jawaban Kasir: %w", err)
	}

	if respons.StatusCode < 200 || respons.StatusCode >= 300 {
		return masterrekening.HasilKasir{
			Berhasil: false,
			Kode:     j.KodeRespons,
			Pesan:    j.Pesan,
		}, nil
	}

	// Kode "1" adalah penanda gagal pada rule lama:
	// MyServicePage.ClaimData.ServiceKasir.ReponseCode=="1" → "kalo ada error set ke param".
	// Kode "9" juga gagal, dan tambahannya memicu surel ke PIC.
	berhasil := j.KodeRespons != "1" && j.KodeRespons != "9"

	return masterrekening.HasilKasir{
		Berhasil:   berhasil,
		IDRekening: strings.TrimSpace(j.IDRekening),
		Pesan:      strings.TrimSpace(j.Pesan),
		Kode:       strings.TrimSpace(j.KodeRespons),
	}, nil
}

var _ masterrekening.Kasir = (*Klien)(nil)
