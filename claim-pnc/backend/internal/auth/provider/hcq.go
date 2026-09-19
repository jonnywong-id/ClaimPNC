package provider

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

	"claim-pnc/internal/auth"
)

// Kode galat yang dikembalikan HCC/HCQ di `Response.pyErrorCode`.
//
// Hanya "200" yang berarti berhasil. Kode lain — termasuk "403" untuk kredensial
// kosong atau salah — berarti gagal, dan bisnis menetapkan kegagalan itu diteruskan ke
// jalur kedua (POOLDATA.M_LOGIN_PNC), bukan langsung ditolak.
const hcqSuccessCode = "200"

// loginServiceKind adalah nilai kolom TYPESERVICE pada POOLDATA.GCNM_CONNECT_REST
// yang menandai baris alamat layanan login HCQ.
const loginServiceKind = "HCQ-LOGIN"

// ServiceCatalog adalah seam ke daftar alamat layanan luar.
//
// Alamat endpoint HCQ tidak ditulis di konfigurasi aplikasi melainkan dibaca dari
// POOLDATA.GCNM_CONNECT_REST, sehingga perpindahan endpoint menjadi perubahan data yang
// dilakukan DBA — bukan rilis ulang aplikasi.
//
// Ini juga mengganti cara sistem lama memilih alamat. `RDB List/BrowseServiceName_sql-SQL.xml`
// menyaring dengan `APPLICATIONIP like '%{ASIS:TempError.source}%'` — membandingkan nama
// server, dan merangkai nilai ke dalam teks SQL. Penggantinya menyaring dengan alias
// portal lewat parameter binding: perilakunya menjadi eksplisit (ADR-0030) dan celah
// injeksi pola `{ASIS:...}` ikut tertutup.
type ServiceCatalog interface {
	// ServiceAddress mengembalikan SERVICENAME untuk satu app dan jenis layanan.
	// Ia mengembalikan ErrServiceNotRegistered bila barisnya tidak ada.
	ServiceAddress(ctx context.Context, app, serviceKind string) (string, error)
}

// ErrServiceNotRegistered dikembalikan ServiceCatalog bila baris yang diminta tidak
// ada di POOLDATA.GCNM_CONNECT_REST.
var ErrServiceNotRegistered = errors.New("provider: alamat layanan tidak terdaftar")

// HCQ memverifikasi kredensial karyawan ke API internal HCC/HCQ.
type HCQ struct {
	katalog     ServiceCatalog
	klien       *http.Client
	portalAlias string
	user        string
	password    string
}

// HCQOptions adalah bahan pembentuk provider HCQ.
type HCQOptions struct {
	Katalog ServiceCatalog

	// PortalAlias mengisi kolom APP saat mencari alamat layanan. Tiap portal entitas
	// boleh punya endpoint HCQ sendiri (keputusan Work Owner 2026-09-16).
	PortalAlias string

	// User dan Password adalah kredensial Basic Auth ke API HCQ — kredensial
	// aplikasi, bukan kredensial pengguna yang sedang masuk.
	User     string
	Password string

	Timeout time.Duration
	Klien   *http.Client
}

// NewHCQ membentuk provider HCQ dan menolak bahan yang tidak lengkap.
func NewHCQ(o HCQOptions) (*HCQ, error) {
	switch {
	case o.Katalog == nil:
		return nil, errors.New("provider: katalog layanan wajib diisi")
	case strings.TrimSpace(o.PortalAlias) == "":
		return nil, errors.New("provider: alias portal wajib diisi")
	case o.User == "":
		return nil, errors.New("provider: HCQ_LOGIN_USER wajib diisi")
	case o.Password == "":
		return nil, errors.New("provider: HCQ_LOGIN_PASSWORD wajib diisi")
	}

	klien := o.Klien
	if klien == nil {
		limit := o.Timeout
		if limit <= 0 {
			limit = 15 * time.Second
		}
		// Batas waktu wajib ada. Tanpa itu, satu API yang menggantung menahan goroutine
		// permintaan sampai pengguna menyerah — dan pada jam sibuk itu menumpuk.
		klien = &http.Client{Timeout: limit}
	}

	return &HCQ{
		katalog:     o.Katalog,
		klien:       klien,
		portalAlias: strings.ToUpper(strings.TrimSpace(o.PortalAlias)),
		user:        o.User,
		password:    o.Password,
	}, nil
}

// hcqRequest adalah badan permintaan ke API HCQ.
type hcqRequest struct {
	Login    string `json:"Login"`
	Password string `json:"Password"`
}

// responsHCQ adalah badan respons API HCQ.
//
// # Kenapa hanya sebagian field yang didaftarkan
//
// Respons lengkapnya memuat lebih dari empat puluh field, termasuk data atasan
// (`EmpLeader`), grade, tanggal bergabung, dan susunan organisasi. Yang didaftarkan di
// sini hanya yang **benar-benar dipakai**, atas arahan Work Owner 2026-09-16 ("ambil
// data yang perlu saja").
//
// Menyalin seluruhnya bukan sekadar berlebihan: setiap field yang ikut masuk menjadi
// data pegawai yang tersimpan dan harus dijaga, padahal tidak satu pun aturan bisnis
// yang membutuhkannya. Field yang tidak dikenali diabaikan encoding/json tanpa galat,
// sehingga penambahan field di sisi HCQ tidak merusak apa pun.
//
// Alasan tiap field yang diambil:
//
//	Person.NIK       kunci alami pengguna, dipakai seluruh data klaim
//	Person.Name      nama yang ditampilkan di aplikasi
//	Person.Login     yang diketik pengguna, untuk penelusuran
//	Person.pyEmail1  tujuan pemberitahuan (S-3)
//	Person.pyCompany entitas pemilik pegawai
//	Placement.BranchName + BranchCode  batas data per cabang (11-SECURITY §3.2)
//	Placement.PositionName             jabatan, ditampilkan dan berguna untuk penelusuran
//	Placement.IsActive                 direkam; lihat catatan pada auth.Profile
type responsHCQ struct {
	Response struct {
		StatusMessage string `json:"pyStatusMessage"`
		Alias         string `json:"Alias"`
		ErrorCode     string `json:"pyErrorCode"`
		Result        string `json:"Result"`
	} `json:"Response"`
	EmpResponse struct {
		Person struct {
			NIK     string `json:"NIK"`
			Company string `json:"pyCompany"`
			Login   string `json:"Login"`
			Email   string `json:"pyEmail1"`
			Name    string `json:"Name"`
		} `json:"Person"`
		Placement struct {
			BranchName   string `json:"BranchName"`
			BranchCode   string `json:"BranchCode"`
			PositionName string `json:"PositionName"`
			Company      string `json:"pyCompany"`
			Name         string `json:"Name"`
			NIK          string `json:"NIK"`
			IsActive     *bool  `json:"IsActive"`
		} `json:"Placement"`
	} `json:"EmpResponse"`
	Login string `json:"Login"`
}

// Verify memanggil API HCQ dan menerjemahkan jawabannya menjadi profil atau galat.
//
// Pemetaan galatnya disengaja:
//   - pyErrorCode "200"            → berhasil
//   - kode lain                    → ErrWrongCredential, karena bisnis menetapkan
//     kegagalan HCQ diteruskan ke jalur kedua
//   - jaringan/HTTP/JSON bermasalah → ErrIdentitySystemUnreachable, yang dibedakan supaya
//     pengguna tahu ini bukan salah ketiknya
func (h *HCQ) Verify(ctx context.Context, k auth.Credential) (auth.Profile, error) {
	address, err := h.katalog.ServiceAddress(ctx, h.portalAlias, loginServiceKind)
	if err != nil {
		if errors.Is(err, ErrServiceNotRegistered) {
			return auth.Profile{}, fmt.Errorf(
				"%w: baris POOLDATA.GCNM_CONNECT_REST dengan APP=%q dan TYPESERVICE=%q belum ada",
				auth.ErrIdentitySystemUnreachable, h.portalAlias, loginServiceKind)
		}
		return auth.Profile{}, fmt.Errorf("%w: membaca alamat layanan: %v", auth.ErrIdentitySystemUnreachable, err)
	}

	response, err := h.panggil(ctx, address, k)
	if err != nil {
		return auth.Profile{}, err
	}

	if response.Response.ErrorCode != hcqSuccessCode {
		// Teks Result dari HCQ tidak diteruskan ke pengguna: ia dapat membedakan
		// "username tidak ditemukan" dari "password salah", dan itu membocorkan
		// keberadaan akun.
		return auth.Profile{}, auth.ErrWrongCredential
	}

	orang := response.EmpResponse.Person
	penempatan := response.EmpResponse.Placement

	profile := auth.Profile{
		// NIK dan nama diambil dari Person lebih dulu; Placement dipakai sebagai
		// cadangan karena kedua blok memuat field yang sama dan pada sebagian respons
		// hanya salah satunya terisi.
		Identity:       firstNonEmpty(orang.NIK, penempatan.NIK),
		Name:           firstNonEmpty(orang.Name, penempatan.Name),
		Kind:           auth.Employee,
		Login:          firstNonEmpty(orang.Login, response.Login, k.Username),
		Email:          strings.TrimSpace(orang.Email),
		Company:        firstNonEmpty(orang.Company, penempatan.Company),
		Branch:         strings.TrimSpace(penempatan.BranchName),
		BranchCode:     strings.TrimSpace(penempatan.BranchCode),
		Position:       strings.TrimSpace(penempatan.PositionName),
		ActiveAtSource: penempatan.IsActive,
	}
	if err := profile.Check(); err != nil {
		// HCQ menjawab "berhasil" tetapi tidak menyertakan NIK atau nama. Meneruskannya
		// berarti pengguna masuk tanpa dapat dikenali data klaimnya sendiri.
		return auth.Profile{}, err
	}
	return profile, nil
}

func (h *HCQ) panggil(ctx context.Context, address string, k auth.Credential) (responsHCQ, error) {
	body, err := json.Marshal(hcqRequest{Login: k.Username, Password: k.Password})
	if err != nil {
		return responsHCQ{}, fmt.Errorf("%w: menyusun permintaan: %v", auth.ErrIdentitySystemUnreachable, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return responsHCQ{}, fmt.Errorf("%w: menyusun permintaan: %v", auth.ErrIdentitySystemUnreachable, err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.SetBasicAuth(h.user, h.password)

	respons, err := h.klien.Do(request)
	if err != nil {
		// Galat jaringan tidak pernah dibungkus apa adanya ke pengguna: pesannya dapat
		// memuat alamat internal. Yang diteruskan hanya jenisnya.
		return responsHCQ{}, fmt.Errorf("%w: %v", auth.ErrIdentitySystemUnreachable, err)
	}
	defer func() { _ = respons.Body.Close() }()

	// Batas ukuran badan: respons yang tidak wajar besar tidak boleh menghabiskan
	// memori proses yang sedang melayani pengguna lain.
	content, err := io.ReadAll(io.LimitReader(respons.Body, 1<<20))
	if err != nil {
		return responsHCQ{}, fmt.Errorf("%w: membaca respons: %v", auth.ErrIdentitySystemUnreachable, err)
	}

	// HCQ menjawab kegagalan kredensial dengan badan JSON dan — pada contoh yang
	// diberikan — status HTTP yang tidak selalu 200. Karena itu badan tetap dibaca
	// lebih dulu, dan status HTTP hanya dipakai bila badannya tidak dapat dipahami.
	var response responsHCQ
	if err := json.Unmarshal(content, &response); err != nil {
		return responsHCQ{}, fmt.Errorf("%w: respons HCQ bukan JSON yang dikenali (status HTTP %d)",
			auth.ErrIdentitySystemUnreachable, respons.StatusCode)
	}
	if response.Response.ErrorCode == "" {
		return responsHCQ{}, fmt.Errorf("%w: respons HCQ tanpa pyErrorCode (status HTTP %d)",
			auth.ErrIdentitySystemUnreachable, respons.StatusCode)
	}
	return response, nil
}

func firstNonEmpty(value ...string) string {
	for _, n := range value {
		if trimmed := strings.TrimSpace(n); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
