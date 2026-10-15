// Package pegaslik memenuhi seam monitoringslinkojk.Sender dengan memanggil layanan REST
// `ASMRequestServiceCreateClient` pada aplikasi Pega `ASMFWInternalWork`.
//
// # Apa yang sebenarnya dipanggil
//
// Bukan OJK. Layanan ini mendaftarkan klien/CIF di aplikasi Pega lain. Tombol di layar
// bernama "SLIK OJK" dan namanya dipertahankan apa adanya (`D-13`), tetapi yang terjadi di
// baliknya adalah pendaftaran klien.
//
// # Ketergantungan yang TIDAK hilang setelah Pega Claim PNC dimatikan
//
// `ASMFWInternalWork` berada di luar lingkup migrasi (`D-03`). Jalur ini karena itu tetap
// memanggil Pega bahkan ketika seluruh modul lain sudah berdiri sendiri — satu-satunya di
// modul ini yang begitu. Bila Pega mati, tombol ini gagal; itu bukan cacat, melainkan sifat
// yang perlu diketahui sebelum cutover.
//
// # Kontraknya dibaca dari mana
//
//	Connect REST/Rest_SendDataClientBasedDebitur-ConnectREST.xml   metode, batas waktu
//	Service REST/ASMRequestServiceCreateClient-REST.xml            jalur, field jawaban
//	Activity/ASMRequestServiceCreateClient_Act -Act.xml            cabang dan galatnya
package pegaslik

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/monitoringslinkojk"
)

// resourceCreateClient adalah `pyResourcePath` layanannya.
//
// Pola jalur layanan REST Pega: `/prweb/PRRestService/{paket}/{kelas}/{nama layanan}`.
// Kelima layanan di export mengikutinya, dan ruas terakhir selalu NAMA LAYANAN — bukan nama
// paket. Dicatat karena alamat yang diberikan Work Owner sempat mengulang nama paket di
// ruas itu, dan salah satu dari keduanya pasti keliru.
const resourceCreateClient = "prweb/PRRestService/ASMFWInternalWork/Data-Portal/ASMRequestServiceCreateClient"

// defaultTimeout mengikuti `pyResponseTimeout` pada Connect REST-nya: 30.000 ms.
const defaultTimeout = 30 * time.Second

// Client memanggil layanan pendaftaran klien.
type Client struct {
	// baseURL KOSONG berarti alamatnya belum disediakan untuk portal ini — keadaan yang
	// SAH, bukan kekeliruan tatanan. Lihat NewClient.
	baseURL string
	path    string

	// user dan password mengisi Basic Auth. Keduanya kosong mengirim permintaan TANPA
	// otentikasi, dan itu memang keadaan layanan ini hari ini
	// (`pyUseAuthentication=false`). Keduanya tetap disediakan supaya ketika Keamanan
	// Informasi menetapkan sebaliknya, yang berubah satu nilai konfigurasi — bukan kode.
	user     string
	password string

	http *http.Client
}

// Config adalah tatanan pemanggil, seluruhnya dari konfigurasi per portal.
type Config struct {
	// BaseURL misalnya `https://clouduniapp.sinarmas.co.id`. Kosong berarti portal itu
	// belum punya alamat, dan tombolnya menolak dengan sebab yang terbaca.
	BaseURL string

	// Path menimpa resourceCreateClient bila diisi.
	//
	// Ia konfigurasi karena jalurnya ditentukan Tim Pega, bukan kami — dan karena ruas
	// terakhirnya masih menunggu konfirmasi.
	Path string

	User     string
	Password string

	// Timeout wajib; kosong memakai defaultTimeout.
	Timeout time.Duration
}

// NewClient membentuk pemanggil.
//
// BaseURL kosong TIDAK menghasilkan galat di sini, melainkan pemanggil yang setiap
// pengirimannya mengembalikan ErrSenderNotConfigured. Itu disengaja: aplikasi tetap menyala,
// layar pemantauan tetap terbaca, dan yang gagal hanyalah tindakan yang memang belum dapat
// dijalankan untuk portal itu.
//
// Pada rancangan empat portal (`D-75`), ini yang mencegah portal tanpa alamat diam-diam
// menembak alamat entitas lain — jalur kebocoran yang `R-20` jaga.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	path := strings.Trim(strings.TrimSpace(cfg.Path), "/")
	if path == "" {
		path = resourceCreateClient
	}
	return &Client{
		baseURL:  strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		path:     path,
		user:     strings.TrimSpace(cfg.User),
		password: cfg.Password,
		http:     &http.Client{Timeout: timeout},
	}
}

// ============================================================================
// BENTUK PERMINTAAN — nama isiannya SENGAJA mengikuti properti Pega
// ============================================================================
//
// `pyFirstName`, `ASMIDCard`, `TelFaxCode` — ejaannya aneh dan tidak seragam, dan itu
// dipertahankan apa adanya. Ia kontrak dengan sistem lain: menamainya ulang berarti
// menambah satu terjemahan yang dapat salah, dan salahnya tidak menghasilkan galat
// melainkan field yang diam-diam kosong di sisi penerima.
//
// Catat `TelFaxCode` dengan F besar berdampingan `TelfaxNumber` dengan f kecil. Itu ada di
// Pega, bukan salah ketik di sini.

type createClientRequest struct {
	RequestService requestService `json:"RequestService"`
}

type requestService struct {
	NBWorkPage nbWorkPage `json:"NBWorkPage"`
}

type nbWorkPage struct {
	Policy policyPage `json:"Policy"`

	// CustomerP dan CustomerC keduanya `omitempty`: penerimanya memilih salah satu
	// menurut `CustomerType`, dan mengirim keduanya hanya menambah muatan yang diabaikan.
	CustomerP *personPage  `json:"Customer_P,omitempty"`
	CustomerC *companyPage `json:"Customer_C,omitempty"`

	AddressList []addressPage `json:"AddressList,omitempty"`
}

type policyPage struct {
	CustomerType  string `json:"CustomerType"`
	IdTransaction string `json:"IdTransaction"`
}

type personPage struct {
	FirstName      string `json:"pyFirstName"`
	LastName       string `json:"pyLastName,omitempty"`
	ASMIDCard      string `json:"ASMIDCard,omitempty"`
	ASMGender      string `json:"ASMGender,omitempty"`
	ASMDateOfBirth string `json:"ASMDateOfBirth,omitempty"`
	ASMNPWP        string `json:"ASMNPWP,omitempty"`
	ASMMotherName  string `json:"ASMMotherName,omitempty"`
}

type companyPage struct {
	Company  string `json:"pyCompany"`
	ASMComID string `json:"ASMComID,omitempty"`
	ASMNPWP  string `json:"ASMNPWP,omitempty"`
}

type addressPage struct {
	ASMAddress  string      `json:"ASMAddress,omitempty"`
	ASMDistrict string      `json:"ASMDistrict,omitempty"`
	ASMCity     string      `json:"ASMCity,omitempty"`
	ASMZipCode  string      `json:"ASMZipCode,omitempty"`
	IsPrimary   string      `json:"IsPrimary,omitempty"`
	ASMTelfax   []phonePage `json:"ASMTelfax,omitempty"`
}

type phonePage struct {
	TelfaxType   string `json:"TelfaxType,omitempty"`
	TelFaxCode   string `json:"TelFaxCode,omitempty"`
	TelfaxNumber string `json:"TelfaxNumber,omitempty"`
}

// createClientResponse adalah jawaban layanannya.
//
// Ketiga isiannya dibaca dari elemen `pyName` pada Service REST-nya — bukan ditebak.
// `AutoAcceptMessageProtection` diterima tetapi tidak dipakai modul ini; ia dicantumkan
// supaya terlihat bahwa ia memang ada, bukan terlewat.
type createClientResponse struct {
	ClientID string `json:"ClientID"`
	ErrMsg   string `json:"ErrMsg"`

	AutoAcceptMessageProtection string `json:"AutoAcceptMessageProtection"`
}

// Send mengirim satu debitur dan mengembalikan bukti kirimnya.
//
// # Tiga kelas kegagalan yang SENGAJA dibedakan
//
//	belum tersedia / tidak terhubung -> ErrSenderNotConfigured  -> 503
//	sampai dan DITOLAK layanannya    -> galat biasa beserta ErrMsg
//	muatan pasti ditolak sejak awal  -> ErrEmptyClient / ErrEmptyCompany
//
// Yang pertama menyuruh orang menunggu Infra; yang kedua menyuruh memeriksa datanya. Satu
// galat untuk keduanya akan menyuruh orang menunggu pihak yang tidak sedang bermasalah.
func (c *Client) Send(
	ctx context.Context,
	submission monitoringslinkojk.Submission,
) (monitoringslinkojk.SubmissionResult, error) {
	if c.baseURL == "" {
		return monitoringslinkojk.SubmissionResult{},
			monitoringslinkojk.ErrSenderNotConfigured
	}

	debtor := submission.Debtor
	if debtor.RequiredFieldMissing() {
		return monitoringslinkojk.SubmissionResult{}, missingFieldError(debtor)
	}

	body, err := json.Marshal(buildRequest(submission))
	if err != nil {
		return monitoringslinkojk.SubmissionResult{},
			fmt.Errorf("menyusun permintaan pendaftaran klien: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/"+c.path, bytes.NewReader(body))
	if err != nil {
		return monitoringslinkojk.SubmissionResult{},
			fmt.Errorf("menyusun permintaan HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.user != "" {
		req.SetBasicAuth(c.user, c.password)
	}

	res, err := c.http.Do(req)
	if err != nil {
		// Gagal menghubungi DISATUKAN dengan "belum disediakan": bagi pelapor keduanya
		// berarti hal yang sama — pengirimannya belum dapat dijalankan dari sini. Yang
		// membedakan ada di log, bukan di layar.
		return monitoringslinkojk.SubmissionResult{},
			fmt.Errorf("%w: %v", monitoringslinkojk.ErrSenderNotConfigured, err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 500 {
		return monitoringslinkojk.SubmissionResult{},
			fmt.Errorf("%w: layanan menjawab %d",
				monitoringslinkojk.ErrSenderNotConfigured, res.StatusCode)
	}
	if res.StatusCode >= 400 {
		// 4xx BUKAN ketidaktersediaan: permintaannya sampai dan ditolak.
		return monitoringslinkojk.SubmissionResult{},
			fmt.Errorf("layanan pendaftaran klien menolak klaim %s: %d",
				submission.ClaimID, res.StatusCode)
	}

	var reply createClientResponse
	if err := json.NewDecoder(res.Body).Decode(&reply); err != nil {
		return monitoringslinkojk.SubmissionResult{},
			fmt.Errorf("membaca jawaban pendaftaran klien: %w", err)
	}

	// `ErrMsg` terisi berarti DITOLAK meski HTTP-nya 200.
	//
	// Layanan Pega menjawab 200 untuk penolakan bisnis; yang membedakan hanya isi
	// `ErrMsg`. Tanpa pemeriksaan ini, penolakan tercatat sebagai pengiriman berhasil —
	// dan pada laporan regulator itu kebohongan yang tidak terlihat oleh siapa pun.
	if msg := strings.TrimSpace(reply.ErrMsg); msg != "" {
		return monitoringslinkojk.SubmissionResult{},
			fmt.Errorf("layanan pendaftaran klien menolak klaim %s: %s",
				submission.ClaimID, msg)
	}

	return monitoringslinkojk.SubmissionResult{
		ClientID: strings.TrimSpace(reply.ClientID),

		// TransactionID yang disimpan adalah yang KITA kirim, bukan yang dikembalikan.
		//
		// Layanannya tidak mengembalikan nomor transaksi — ia memakai yang kita kirim
		// sebagai kunci idempotensi. Menyimpan nilai itu kembali membuat baris pengiriman
		// dapat dicocokkan dengan log di sisi penerima.
		TransactionID: submission.Debtor.TransactionID,
	}, nil
}

// buildRequest menyusun badan permintaan dari satu pengiriman.
//
// Hanya SATU cabang pelanggan yang diisi — mengikuti penerimanya, yang membaca
// `Customer_P` atau `Customer_C` menurut `CustomerType`, tidak keduanya.
func buildRequest(submission monitoringslinkojk.Submission) createClientRequest {
	debtor := submission.Debtor

	page := nbWorkPage{
		Policy: policyPage{
			CustomerType:  customerTypeOf(debtor),
			IdTransaction: debtor.TransactionID,
		},
		AddressList: addressPages(debtor.Addresses),
	}

	if customerTypeOf(debtor) == monitoringslinkojk.CustomerCompany {
		page.CustomerC = &companyPage{
			Company:  debtor.CompanyName,
			ASMComID: debtor.CompanyID,
			ASMNPWP:  debtor.NPWP,
		}
	} else {
		page.CustomerP = &personPage{
			FirstName:      debtor.FirstName,
			LastName:       debtor.LastName,
			ASMIDCard:      debtor.IDCardNo,
			ASMGender:      debtor.Gender,
			ASMDateOfBirth: debtor.DateOfBirth,
			ASMNPWP:        debtor.NPWP,
			ASMMotherName:  debtor.MotherName,
		}
	}

	return createClientRequest{RequestService: requestService{NBWorkPage: page}}
}

// customerTypeOf menjatuhkan nilai kosong ke perorangan.
//
// Bukan ke kosong: `CustomerType` kosong membuat penerimanya tidak mengambil cabang mana
// pun, dan permintaannya gagal tanpa sebab yang terbaca. Perorangan adalah jalur segmen
// F06 — "debitur individu" — sehingga ia bawaan yang benar bila tipenya tidak terbaca.
func customerTypeOf(debtor monitoringslinkojk.Debtor) string {
	if strings.TrimSpace(debtor.CustomerType) == monitoringslinkojk.CustomerCompany {
		return monitoringslinkojk.CustomerCompany
	}
	return monitoringslinkojk.CustomerPerson
}

func addressPages(addresses []monitoringslinkojk.DebtorAddress) []addressPage {
	if len(addresses) == 0 {
		return nil
	}
	out := make([]addressPage, 0, len(addresses))
	for _, address := range addresses {
		page := addressPage{
			ASMAddress:  address.Street,
			ASMDistrict: address.District,
			ASMCity:     address.City,
			ASMZipCode:  address.PostalCode,
		}
		if address.Primary {
			page.IsPrimary = "true"
		}
		for _, phone := range address.Phones {
			page.ASMTelfax = append(page.ASMTelfax, phonePage{
				TelfaxType:   phone.Type,
				TelFaxCode:   phone.AreaCode,
				TelfaxNumber: phone.Number,
			})
		}
		out = append(out, page)
	}
	return out
}

// missingFieldError memilih galat yang sepadan dengan cabangnya.
//
// Teks pesannya disalin APA ADANYA dari `ASMRequestServiceCreateClient_Act`, supaya yang
// terbaca pelapor di layar sama dengan yang akan dikatakan layanannya sendiri.
func missingFieldError(debtor monitoringslinkojk.Debtor) error {
	if customerTypeOf(debtor) == monitoringslinkojk.CustomerCompany {
		return monitoringslinkojk.ErrEmptyCompany
	}
	return monitoringslinkojk.ErrEmptyClient
}
