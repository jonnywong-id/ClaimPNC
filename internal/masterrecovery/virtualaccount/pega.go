// Package virtualaccount memenuhi seam masterrecovery.VirtualAccountIssuer.
//
// Dua pengisi: Pega di berkas ini — jalur nyata — dan Fake di fake.go, yang dipakai untuk
// pengujian serta pengembangan tanpa jaringan. Dua adapter itulah yang membuat seam ini
// benar-benar seam (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// # PENERBIT INI TERTUTUP SECARA BAKU
//
// Keputusan Work Owner 2026-09-29: jangan dibuka dulu. Yang membukanya adalah
// `VIRTUAL_ACCOUNT_ADAPTER=pega` ditambah `VIRTUAL_ACCOUNT_PENGGUNA` dan
// `VIRTUAL_ACCOUNT_SANDI` yang terisi — ketiganya sekaligus. Perakitnya ada di
// cmd/claimpnc/buildVirtualAccountIssuer.
//
// # Kontrak ini DIBACA dari rule-nya, bukan lagi disimpulkan (2026-09-29)
//
// Rule `Connect REST/VirtualAccountClaimsPNC-ConnectREST.xml` sudah diterima dari Tim
// Pega. Ia mengoreksi dua tebakan sebelumnya:
//
//  1. **Autentikasi WAJIB.** `pyUseAuthentication=true` dengan
//     `pyAuthenticationProfile=LELANG`. Adapter ini semula memakai ulang kredensial HCQ —
//     itu KELIRU, keduanya profil yang berbeda. Profil `LELANG` sendiri TIDAK ADA di
//     export; hanya namanya yang dirujuk, sehingga kredensialnya diminta terpisah.
//  2. **Badan permintaan adalah halaman `ClaimData` yang DISERIALKAN menjadi JSON.**
//     `pyMapFrom=Clipboard`, `pyMapFromKey=PNCVARECOVERY.ClaimData.City` — dan `.City`
//     bukan kota: `GeneratedVAClaimRecovery-Act.xml:1783` mengisinya dengan
//     `Param.jsonData`, hasil `Apply-DataTransform SetDataJson`
//     (`executionMode=SERIALIZE`) atas halaman itu. Jadi badannya JSON datar berisi
//     properti yang diisi activity — bukan amplop bersarang.
//
// Yang sudah benar sejak awal dan kini terkonfirmasi:
//
//   - **Respons dipetakan kembali ke halaman `ClaimData`** (`pyMapTo=JSON`,
//     `pyMapToKey=PNCVARECOVERY.ClaimData`), sehingga kunci responsnya bernama persis
//     seperti propertinya: `Status`, `Message`, `VirtualAccountNumber`.
//   - Metode POST, `Content-Type: application/json`, dan alamat `ServiceFromTable` —
//     dibaca dari POOLDATA.GCNM_CONNECT_REST (`APP=<portal>`,
//     `TYPESERVICE='GENERATEDVA'`), sehingga perpindahan endpoint tetap menjadi perubahan
//     data oleh DBA, bukan rilis ulang.
//
// Keempat properti yang diisi activity sebelum memanggil — inilah badan permintaannya:
//
//	ClaimData.SourceID      = "LELANG"        (:1109)
//	ClaimData.NoRef         = ClientID        (:1154)
//	ClaimData.CustomerName  = NamaPrincipal   (:1174)
//	ClaimData.PaymentAmount = 0               (:1200)
//
// Pembacaan respons tetap TOLERAN terhadap beberapa ejaan kunci. Ejaan yang benar kini
// diketahui dan dicoba lebih dulu; sisanya dipertahankan sebagai jaring pengaman, karena
// satu perbedaan ejaan tidak boleh membuat VA yang sudah benar-benar terbit gagal
// tercatat — dan yang di seberang layanan yang tidak kita miliki.
package virtualaccount

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/masterrecovery"
)

// DefaultServiceKind adalah nilai kolom TYPESERVICE yang dicari di
// POOLDATA.GCNM_CONNECT_REST. Diverifikasi ada pada portal ASM 2026-09-19.
const DefaultServiceKind = "GENERATEDVA"

// SourceID adalah penanda asal permintaan yang dikirim apa adanya.
//
// Nilainya "LELANG", dipatok `GeneratedVAClaimRecovery-Act.xml:1109`. Artinya tidak
// dijelaskan di mana pun, dan ia TIDAK ditebak-tebak di sini: ia diteruskan persis seperti
// sistem lama mengirimnya, karena yang di seberang mungkin bergantung padanya.
const SourceID = "LELANG"

// maxResponseBytes membatasi ukuran badan respons yang dibaca.
//
// Respons yang tidak wajar besar tidak boleh menghabiskan memori proses yang sedang
// melayani pengguna lain.
const maxResponseBytes = 1 << 20

// ServiceCatalog adalah seam ke daftar alamat layanan luar.
//
// Ia dinyatakan di sisi PEMAKAI sebagai antarmuka sempit, bukan diimpor dari modul auth,
// supaya kedua modul tetap tidak saling mengimpor. Yang tahu keduanya hanyalah berkas
// perakitan di cmd/claimpnc — pengisinya di sana adalah pembaca
// POOLDATA.GCNM_CONNECT_REST yang sama dengan yang dipakai modul lain.
type ServiceCatalog interface {
	ServiceAddress(ctx context.Context, app, serviceKind string) (string, error)
}

// Pega menerbitkan rekening virtual lewat layanan REST milik Pega.
type Pega struct {
	catalog       ServiceCatalog
	client        *http.Client
	serviceKind   string
	user          string
	password      string
	notRegistered error
}

// Options adalah bahan pembentuk Pega.
type Options struct {
	// Catalog membaca alamat layanan per portal. Wajib.
	Catalog ServiceCatalog

	// User dan Password adalah kredensial profil autentikasi `LELANG`. **Wajib** —
	// rule-nya ber-`pyUseAuthentication=true`, jadi permintaan tanpa kredensial pasti
	// ditolak. Keduanya diperiksa NewPega, bukan dibiarkan gagal di jaringan: penolakan
	// autentikasi tampak seperti gangguan layanan, dan itu menyesatkan orang yang
	// menelusurinya.
	//
	// BUKAN kredensial HCQ. Keduanya profil yang berbeda.
	User     string
	Password string

	// ServiceKind menimpa DefaultServiceKind, supaya pemindahan ke baris katalog lain
	// menjadi perubahan konfigurasi, bukan rilis ulang.
	ServiceKind string

	// NotRegistered adalah galat yang dikembalikan Catalog bila barisnya tidak ada.
	//
	// Dipasok dari luar karena galat itu milik modul auth, dan mengimpornya ke sini berarti
	// kedua modul saling mengimpor. Boleh nil; bila nil, ketiadaan baris tidak dapat
	// dibedakan dari kegagalan membaca katalog.
	NotRegistered error

	Timeout time.Duration
	Client  *http.Client
}

// NewPega membentuk adapter penerbit VA dan menolak bahan yang tidak lengkap.
func NewPega(o Options) (*Pega, error) {
	switch {
	case o.Catalog == nil:
		return nil, errors.New("masterrecovery/virtualaccount: katalog layanan wajib diisi")
	case strings.TrimSpace(o.User) == "":
		// Ditolak di sini, bukan dibiarkan gagal saat memanggil: rule-nya menuntut
		// autentikasi, sehingga permintaan tanpa kredensial pasti ditolak — dan
		// penolakannya akan terbaca sebagai gangguan layanan, bukan sebagai konfigurasi
		// yang belum lengkap.
		return nil, errors.New("masterrecovery/virtualaccount: VIRTUAL_ACCOUNT_PENGGUNA wajib diisi (profil autentikasi LELANG)")
	case strings.TrimSpace(o.Password) == "":
		return nil, errors.New("masterrecovery/virtualaccount: VIRTUAL_ACCOUNT_SANDI wajib diisi (profil autentikasi LELANG)")
	}

	client := o.Client
	if client == nil {
		limit := o.Timeout
		if limit <= 0 {
			// Lebih longgar daripada pencarian pegawai: penerbitan VA menembak layanan yang
			// sendirinya berbicara ke bank, sehingga jawabannya wajar lebih lambat.
			limit = 30 * time.Second
		}
		// Batas waktu wajib ada. Tanpa itu, satu layanan yang menggantung menahan goroutine
		// permintaan sampai pengguna menyerah — dan pada jam sibuk itu menumpuk.
		client = &http.Client{Timeout: limit}
	}

	return &Pega{
		catalog:       o.Catalog,
		client:        client,
		serviceKind:   firstNonEmpty(o.ServiceKind, DefaultServiceKind),
		user:          strings.TrimSpace(o.User),
		password:      o.Password,
		notRegistered: o.NotRegistered,
	}, nil
}

// request adalah badan permintaan penerbitan.
//
// Keempat field-nya mengikuti properti yang diisi activity lama, dengan nama yang SAMA
// PERSIS — termasuk huruf besar-kecilnya. Menyesuaikannya menjadi gaya penamaan kita
// sendiri akan menjamin layanan di seberang tidak mengenali satu pun di antaranya.
type request struct {
	SourceID      string `json:"SourceID"`
	NoRef         string `json:"NoRef"`
	CustomerName  string `json:"CustomerName"`
	PaymentAmount int64  `json:"PaymentAmount"`
}

// response adalah bagian jawaban yang benar-benar dipakai.
//
// Setiap nilai dibaca dari BEBERAPA ejaan kunci yang mungkin, karena pemetaan aslinya
// tidak dapat dibaca — lihat peringatan di kepala berkas. Ejaan pertama yang terisi yang
// dipakai.
//
// Field yang tidak dikenali diabaikan encoding/json tanpa galat, sehingga penambahan field
// di sisi layanan tidak merusak apa pun.
type response struct {
	VirtualAccountNumber string `json:"VirtualAccountNumber"`
	VirtualAccount       string `json:"VirtualAccount"`
	VANumber             string `json:"VANumber"`

	Status     string `json:"Status"`
	StatusCode string `json:"StatusCode"`

	Message      string `json:"Message"`
	ErrorMessage string `json:"ErrorMessage"`

	// ClaimData menampung bentuk bersarang, yang lazim pada layanan REST Pega karena
	// properti sumbernya memang berada di bawah halaman `ClaimData`.
	ClaimData struct {
		VirtualAccountNumber string `json:"VirtualAccountNumber"`
		Status               string `json:"Status"`
		Message              string `json:"Message"`
	} `json:"ClaimData"`
}

// Issue menerbitkan satu rekening virtual.
//
// Ia TIDAK memeriksa apakah principal-nya sudah punya VA — pemeriksaan itu milik usecase,
// dan menaruhnya juga di sini akan membuat dua tempat memutuskan hal yang sama dengan
// aturan yang dapat menyimpang.
func (p *Pega) Issue(
	ctx context.Context,
	portalAlias string,
	subject masterrecovery.VirtualAccountRequest,
) (masterrecovery.VirtualAccount, error) {
	address, err := p.catalog.ServiceAddress(ctx, strings.ToUpper(strings.TrimSpace(portalAlias)), p.serviceKind)
	if err != nil {
		if p.notRegistered != nil && errors.Is(err, p.notRegistered) {
			// Dibedakan dari gangguan jaringan: ini tidak akan pulih sendiri, dan tindak
			// lanjutnya menambahkan baris ke katalog — bukan menunggu.
			return masterrecovery.VirtualAccount{}, fmt.Errorf("%w: APP=%q TYPESERVICE=%q",
				masterrecovery.ErrIssuerUnconfigured, portalAlias, p.serviceKind)
		}
		return masterrecovery.VirtualAccount{}, fmt.Errorf("%w: membaca alamat layanan: %v",
			masterrecovery.ErrIssuerUnreachable, err)
	}

	body, err := json.Marshal(request{
		SourceID:     SourceID,
		NoRef:        strings.TrimSpace(subject.ClientID),
		CustomerName: strings.TrimSpace(subject.PrincipalName),
		// Nol, mengikuti `GeneratedVAClaimRecovery-Act.xml:1200`. Rekening virtual untuk
		// recovery menerima nominal berapa pun; ia bukan tagihan bernilai tetap.
		PaymentAmount: 0,
	})
	if err != nil {
		return masterrecovery.VirtualAccount{}, fmt.Errorf("%w: menyusun permintaan: %v",
			masterrecovery.ErrIssuerUnreachable, err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return masterrecovery.VirtualAccount{}, fmt.Errorf("%w: menyusun permintaan: %v",
			masterrecovery.ErrIssuerUnreachable, err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	// Selalu dikirim, tidak lagi bersyarat: rule-nya ber-`pyUseAuthentication=true`, dan
	// NewPega sudah menolak kredensial yang kosong.
	httpRequest.SetBasicAuth(p.user, p.password)

	httpResponse, err := p.client.Do(httpRequest)
	if err != nil {
		// Galat jaringan tidak pernah diteruskan apa adanya ke pengguna: pesannya dapat
		// memuat alamat internal. Yang diteruskan hanya jenisnya.
		return masterrecovery.VirtualAccount{}, fmt.Errorf("%w: %v", masterrecovery.ErrIssuerUnreachable, err)
	}
	defer func() { _ = httpResponse.Body.Close() }()

	content, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxResponseBytes))
	if err != nil {
		return masterrecovery.VirtualAccount{}, fmt.Errorf("%w: membaca respons: %v",
			masterrecovery.ErrIssuerUnreachable, err)
	}

	var answer response
	if err := json.Unmarshal(content, &answer); err != nil {
		return masterrecovery.VirtualAccount{}, fmt.Errorf("%w: respons bukan JSON yang dikenali (status HTTP %d)",
			masterrecovery.ErrIssuerUnreachable, httpResponse.StatusCode)
	}

	number := firstNonEmpty(
		answer.ClaimData.VirtualAccountNumber,
		answer.VirtualAccountNumber,
		answer.VirtualAccount,
		answer.VANumber,
	)
	status := firstNonEmpty(answer.ClaimData.Status, answer.Status, answer.StatusCode)
	message := firstNonEmpty(answer.ClaimData.Message, answer.Message, answer.ErrorMessage)

	// Nomor yang kosong diperlakukan sebagai PENOLAKAN, apa pun status HTTP-nya.
	//
	// Urutannya sengaja demikian: layanan Pega kerap menjawab kegagalan dengan status 200
	// beserta pesan di dalam badan, sehingga menilai dari status HTTP saja akan menerima
	// jawaban yang sebenarnya menolak — dan mencatat principal dengan VA kosong yang kelak
	// menerima dana entah ke mana.
	if number == "" {
		if message == "" {
			message = fmt.Sprintf("layanan menjawab tanpa nomor virtual account (status HTTP %d)", httpResponse.StatusCode)
		}
		return masterrecovery.VirtualAccount{}, fmt.Errorf("%w: %s", masterrecovery.ErrIssuerRejected, message)
	}

	return masterrecovery.VirtualAccount{
		Number:  strings.TrimSpace(number),
		Status:  strings.TrimSpace(status),
		Message: strings.TrimSpace(message),
		Reused:  false,
	}, nil
}

func firstNonEmpty(value ...string) string {
	for _, v := range value {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var _ masterrecovery.VirtualAccountIssuer = (*Pega)(nil)
