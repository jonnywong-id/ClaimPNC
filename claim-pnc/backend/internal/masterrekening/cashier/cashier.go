// Package cashier memenuhi seam masterrekening.Cashier dengan panggilan HTTP nyata.
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
package cashier

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

// Config adalah alamat dan kredensial sistem Kasir.
type Config struct {
	// RegisterURL menerima rekening yang baru disetujui.
	RegisterURL string
	// UpdateURL menerima rekening yang menggantikan rekening lama.
	UpdateURL string

	User     string
	Password string

	// Timeout membatasi lama menunggu jawaban Kasir. Nol berarti memakai nilai baku.
	Timeout time.Duration
}

// Complete menyatakan konfigurasi ini cukup untuk menghubungi Kasir.
func (k Config) Complete() bool {
	return strings.TrimSpace(k.RegisterURL) != "" && strings.TrimSpace(k.UpdateURL) != ""
}

const defaultTimeout = 30 * time.Second

// Client menghubungi sistem Kasir lewat HTTP.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient membentuk klien Kasir.
func NewClient(k Config) *Client {
	timeout := k.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{cfg: k, http: &http.Client{Timeout: timeout}}
}

// request adalah badan yang dikirim ke Kasir.
//
// Nama fieldnya mengikuti properti yang disalin activity lama ke MyServicePage. Ia
// sengaja DTO tersendiri, terpisah dari masterrekening.Account: bentuk kawat milik
// sistem lain tidak boleh menyandera bentuk domain kita.
type request struct {
	Number      string `json:"no_rekening"`
	OwnerName   string `json:"nama_rekening"`
	BankCode    string `json:"kode_bank"`
	BankName    string `json:"nama_bank"`
	BankBranch  string `json:"cabang_bank"`
	BankAddress string `json:"alamat_bank"`
	AccountType string `json:"tipe_rekening"`
	NIK         string `json:"nik"`
	Email       string `json:"email"`
	Phone       string `json:"telepon"`
	DocumentID  string `json:"id_dokumen"`

	// Tiga field berikut hanya terisi pada jalur pembaruan; Kasir memakainya untuk
	// menemukan rekening mana yang digantikan.
	PreviousBankCode  string `json:"kode_bank_lama,omitempty"`
	PreviousNumber    string `json:"no_rekening_lama,omitempty"`
	PreviousOwnerName string `json:"nama_rekening_lama,omitempty"`
}

// response adalah bentuk respons Kasir.
//
// ReponseCode dieja begitu — dengan huruf "s" yang hilang — karena begitulah ejaannya
// di MyServicePage.ClaimData.ServiceKasir.ReponseCode pada rule lama. Salah ketik yang
// sudah menjadi kontrak tidak dapat diperbaiki sepihak; memperbaikinya di sini berarti
// membaca field yang tidak pernah dikirim (bandingkan 03-CURRENT-ARCHITECTURE §4.7).
type response struct {
	ResponseCode string `json:"ReponseCode"`
	Message      string `json:"ResponseMessage"`
	AccountID    string `json:"IdRekening"`
}

// Register mendaftarkan rekening baru ke Kasir.
func (k *Client) Register(ctx context.Context, r masterrekening.Account) (masterrekening.CashierResult, error) {
	return k.send(ctx, k.cfg.RegisterURL, bodyFrom(r, false))
}

// Update memberi tahu Kasir bahwa sebuah rekening menggantikan rekening lama.
func (k *Client) Update(ctx context.Context, r masterrekening.Account) (masterrekening.CashierResult, error) {
	return k.send(ctx, k.cfg.UpdateURL, bodyFrom(r, true))
}

func bodyFrom(r masterrekening.Account, withPrevious bool) request {
	p := request{
		Number:      r.Number,
		OwnerName:   r.OwnerName,
		BankCode:    r.BankCode,
		BankName:    r.BankName,
		BankBranch:  r.BankBranch,
		BankAddress: r.BankAddress,
		AccountType: r.AccountType,
		NIK:         r.NIK,
		Email:       r.Email,
		Phone:       r.Phone,
		DocumentID:  r.DocumentID,
	}
	if withPrevious {
		p.PreviousBankCode = r.PreviousBankCode
		p.PreviousNumber = r.PreviousNumber
		p.PreviousOwnerName = r.PreviousOwnerName
	}
	return p
}

func (k *Client) send(ctx context.Context, url string, body request) (masterrekening.CashierResult, error) {
	if strings.TrimSpace(url) == "" {
		return masterrekening.CashierResult{}, fmt.Errorf("masterrekening/cashier: alamat service belum dikonfigurasi")
	}

	content, err := json.Marshal(body)
	if err != nil {
		return masterrekening.CashierResult{}, fmt.Errorf("masterrekening/cashier: menyusun permintaan: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(content))
	if err != nil {
		return masterrekening.CashierResult{}, fmt.Errorf("masterrekening/cashier: menyusun permintaan HTTP: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	if k.cfg.User != "" {
		httpRequest.SetBasicAuth(k.cfg.User, k.cfg.Password)
	}

	respons, err := k.http.Do(httpRequest)
	if err != nil {
		// Galatnya dibungkus tanpa menyertakan URL: alamat service kadang memuat
		// token di query string, dan galat berakhir di log.
		return masterrekening.CashierResult{}, fmt.Errorf("masterrekening/cashier: menghubungi sistem Kasir: %w", err)
	}
	defer func() { _ = respons.Body.Close() }()

	var j response
	if err := json.NewDecoder(respons.Body).Decode(&j); err != nil {
		return masterrekening.CashierResult{}, fmt.Errorf("masterrekening/cashier: membaca jawaban Cashier: %w", err)
	}

	if respons.StatusCode < 200 || respons.StatusCode >= 300 {
		return masterrekening.CashierResult{
			Succeeded: false,
			Code:      j.ResponseCode,
			Message:   j.Message,
		}, nil
	}

	// Kode "1" adalah penanda gagal pada rule lama:
	// MyServicePage.ClaimData.ServiceKasir.ReponseCode=="1" → "kalo ada error set ke param".
	// Kode "9" juga gagal, dan tambahannya memicu surel ke PIC.
	ok := j.ResponseCode != "1" && j.ResponseCode != "9"

	return masterrekening.CashierResult{
		Succeeded: ok,
		AccountID: strings.TrimSpace(j.AccountID),
		Message:   strings.TrimSpace(j.Message),
		Code:      strings.TrimSpace(j.ResponseCode),
	}, nil
}

var _ masterrekening.Cashier = (*Client)(nil)
