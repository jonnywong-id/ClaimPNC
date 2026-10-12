// Package directory memenuhi seam masterpicteknik.EmployeeDirectory dengan memanggil API
// internal HCC/HCQ.
//
// # Kenapa modul ini memanggil HCQ sendiri, bukan memakai ulang provider auth
//
// Keduanya menembak endpoint yang sama, tetapi menanyakan hal yang berbeda:
//
//	auth/provider.HCQ.Verify   "benarkah sandi ini milik orang ini?"
//	directory.HCQ.Lookup       "siapa orang dengan identitas ini, dan siapa atasannya?"
//
// `Verify` menolak dengan ErrWrongCredential begitu `pyErrorCode` bukan "200", dan
// membuang seluruh blok EmpResponse. Untuk pencarian nama itu justru bagian yang
// dibutuhkan. Memakai ulang `Verify` karena itu mustahil tanpa menyuntingnya — dan modul
// Login sudah dinyatakan selesai, sehingga menyentuhnya bukan pilihan.
//
// Yang DIPAKAI ULANG adalah katalog layanannya: alamat endpoint dibaca dari baris
// POOLDATA.GCNM_CONNECT_REST yang sama persis, lewat antarmuka sempit ServiceCatalog di
// bawah. Tidak ada alamat yang ditulis dua kali.
//
// # Sandi pengisi
//
// Permintaan ke endpoint ini menuntut sepasang Login dan Password. Untuk pencarian nama
// tidak ada sandi yang dapat diberikan — yang dicari justru orang lain, bukan pemanggil.
// `SetMstUserTeknisMstUser_act` langkah 6 mengisinya dengan `"1"`, dan nilai itu dipakai
// di sini sebagai bawaan supaya perilakunya setara.
//
// Karena itu jawaban `pyErrorCode` TIDAK dipakai sebagai penentu. Yang menentukan adalah
// ada-tidaknya nama di dalam blok EmpResponse: bila ada, pegawainya dikenal; bila tidak,
// ia dianggap tidak terdaftar. Membaca verdict sandi akan menolak setiap pencarian, karena
// sandi pengisi memang tidak pernah benar.
package directory

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

	"claim-pnc/internal/masterpicteknik"
)

// DefaultServiceKind adalah nilai kolom TYPESERVICE yang dicari di
// POOLDATA.GCNM_CONNECT_REST.
//
// Ditetapkan Work Owner 2026-09-19. Sistem lama memakai `"GetEmployee"` untuk pencarian
// ini dan `"HCQ-LOGIN"` untuk masuk; keputusannya menyatukan keduanya ke satu baris
// layanan, dan itu baris yang sudah pasti ada.
const DefaultServiceKind = "HCQ-LOGIN"

// DefaultLookupPassword adalah sandi pengisi permintaan pencarian.
//
// Nilainya "1", sama dengan yang diisi `SetMstUserTeknisMstUser_act` langkah 6 ke
// `InputEmployee.pyCachingData.pyPwdCurrent`. Ia BUKAN rahasia dan bukan kredensial siapa
// pun — sekadar mengisi field yang bentuk permintaannya tuntut.
const DefaultLookupPassword = "1"

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
// POOLDATA.GCNM_CONNECT_REST yang sama dengan yang dipakai saat masuk.
type ServiceCatalog interface {
	// ServiceAddress mengembalikan SERVICENAME untuk satu app dan jenis layanan.
	ServiceAddress(ctx context.Context, app, serviceKind string) (string, error)
}

// HCQ mencari pegawai ke API internal HCC/HCQ.
type HCQ struct {
	catalog        ServiceCatalog
	client         *http.Client
	serviceKind    string
	lookupPassword string
	user           string
	password       string
	notRegistered  error
}

// Options adalah bahan pembentuk HCQ.
type Options struct {
	// Catalog membaca alamat layanan. Wajib.
	Catalog ServiceCatalog

	// User dan Password adalah kredensial Basic Auth ke API HCQ — kredensial APLIKASI,
	// bukan kredensial pengguna yang sedang masuk. Wajib.
	User     string
	Password string

	// ServiceKind menimpa DefaultServiceKind. Ia dapat dikonfigurasi supaya pemindahan ke
	// layanan pencarian tersendiri — bila kelak didaftarkan DBA — menjadi perubahan
	// konfigurasi, bukan rilis ulang.
	ServiceKind string

	// LookupPassword menimpa DefaultLookupPassword.
	LookupPassword string

	// NotRegistered adalah galat yang dikembalikan Catalog bila barisnya tidak ada.
	//
	// Ia dipasok dari luar karena galat itu milik modul auth, dan mengimpornya ke sini
	// berarti kedua modul saling mengimpor. Boleh nil; bila nil, ketiadaan baris tidak
	// dapat dibedakan dari kegagalan membaca katalog.
	NotRegistered error

	Timeout time.Duration
	Client  *http.Client
}

// New membentuk adapter direktori dan menolak bahan yang tidak lengkap.
func New(o Options) (*HCQ, error) {
	switch {
	case o.Catalog == nil:
		return nil, errors.New("masterpicteknik/directory: katalog layanan wajib diisi")
	case strings.TrimSpace(o.User) == "":
		return nil, errors.New("masterpicteknik/directory: HCQ_LOGIN_USER wajib diisi")
	case strings.TrimSpace(o.Password) == "":
		return nil, errors.New("masterpicteknik/directory: HCQ_LOGIN_PASSWORD wajib diisi")
	}

	client := o.Client
	if client == nil {
		limit := o.Timeout
		if limit <= 0 {
			limit = 15 * time.Second
		}
		// Batas waktu wajib ada. Tanpa itu, satu API yang menggantung menahan goroutine
		// permintaan sampai pengguna menyerah — dan pada jam sibuk itu menumpuk.
		client = &http.Client{Timeout: limit}
	}

	return &HCQ{
		catalog:        o.Catalog,
		client:         client,
		serviceKind:    firstNonEmpty(o.ServiceKind, DefaultServiceKind),
		lookupPassword: firstNonEmpty(o.LookupPassword, DefaultLookupPassword),
		user:           o.User,
		password:       o.Password,
		notRegistered:  o.NotRegistered,
	}, nil
}

// request adalah badan permintaan ke API HCQ.
type request struct {
	Login    string `json:"Login"`
	Password string `json:"Password"`
}

// response adalah bagian respons HCQ yang benar-benar dipakai modul ini.
//
// Respons lengkapnya memuat lebih dari empat puluh field. Yang didaftarkan di sini hanya
// yang dibutuhkan untuk mengisi dua isian di form — nama dan atasan — mengikuti arahan
// Work Owner "ambil data yang perlu saja" (2026-09-16).
//
// Field yang tidak dikenali diabaikan encoding/json tanpa galat, sehingga penambahan field
// di sisi HCQ tidak merusak apa pun.
type response struct {
	Response struct {
		ErrorCode string `json:"pyErrorCode"`
	} `json:"Response"`

	EmpResponse struct {
		Person struct {
			NIK   string `json:"NIK"`
			Login string `json:"Login"`
			Name  string `json:"Name"`
			Email string `json:"pyEmail1"`
		} `json:"Person"`
		Placement struct {
			NIK  string `json:"NIK"`
			Name string `json:"Name"`
		} `json:"Placement"`
	} `json:"EmpResponse"`

	// EmpLeader adalah atasan pegawai menurut struktur organisasi. Blok inilah yang
	// membuat satu pencarian menjawab isian Nama DAN isian Atasan sekaligus.
	//
	// Ia dapat KOSONG dan itu wajar — pucuk pimpinan tidak punya atasan.
	EmpLeader struct {
		Person struct {
			NIK   string `json:"NIK"`
			Login string `json:"Login"`
			Name  string `json:"Name"`
		} `json:"Person"`
	} `json:"EmpLeader"`

	Login string `json:"Login"`
}

// Lookup mencari seorang pegawai berdasarkan identitasnya.
//
// # Apa yang dipakai sebagai identitas
//
// Yang dikirim adalah `Login` — nama pengguna yang dipakai orang untuk masuk, sesuai
// keputusan Work Owner 2026-09-19. Bentuknya paling dekat dengan OPERATOR_ID yang selama
// ini diisi di master, dan itu pula yang dikirim `SetMstUserTeknisMstUser_act` setelah
// diubah menjadi huruf besar (`@toUpperCase(TempDcol.OPERATOR_ID)`).
func (h *HCQ) Lookup(ctx context.Context, portalAlias, operatorID string) (masterpicteknik.Employee, error) {
	operatorID = strings.TrimSpace(operatorID)
	if operatorID == "" {
		return masterpicteknik.Employee{}, masterpicteknik.ErrEmployeeUnknown
	}

	address, err := h.catalog.ServiceAddress(ctx, strings.ToUpper(strings.TrimSpace(portalAlias)), h.serviceKind)
	if err != nil {
		if h.notRegistered != nil && errors.Is(err, h.notRegistered) {
			// Dibedakan dari gangguan jaringan: ini tidak akan pulih sendiri, dan tindak
			// lanjutnya menambahkan baris ke katalog — bukan menunggu.
			return masterpicteknik.Employee{}, fmt.Errorf("%w: APP=%q TYPESERVICE=%q",
				masterpicteknik.ErrDirectoryNotConfigured, portalAlias, h.serviceKind)
		}
		return masterpicteknik.Employee{}, fmt.Errorf("%w: membaca alamat layanan: %v",
			masterpicteknik.ErrDirectoryUnreachable, err)
	}

	// Huruf besar mengikuti `@toUpperCase(TempDcol.OPERATOR_ID)` di activity lamanya.
	answer, err := h.call(ctx, address, strings.ToUpper(operatorID))
	if err != nil {
		return masterpicteknik.Employee{}, err
	}

	person := answer.EmpResponse.Person
	placement := answer.EmpResponse.Placement
	leader := answer.EmpLeader.Person

	name := firstNonEmpty(person.Name, placement.Name)
	if name == "" {
		// Tidak ada nama berarti pegawainya tidak dikenal. Inilah keadaan yang di sistem
		// lama membuat `TempDcol.MCL_NAME` tetap kosong, lalu ditolak
		// `CNMInsertMstUserTeknis_act` langkah 2 dengan keterangan "set error kalau tidak
		// ditemukan di service".
		return masterpicteknik.Employee{}, masterpicteknik.ErrEmployeeUnknown
	}

	return masterpicteknik.Employee{
		// Identitas dikembalikan sebagaimana dikenal HCQ, bukan sebagaimana diketik
		// pengguna: itu yang membuat besar-kecil huruf di master seragam dengan sumbernya.
		OperatorID:     firstNonEmpty(person.Login, answer.Login, operatorID),
		Name:           name,
		Email:          strings.TrimSpace(person.Email),
		SupervisorID:   firstNonEmpty(leader.Login, leader.NIK),
		SupervisorName: strings.TrimSpace(leader.Name),
	}, nil
}

func (h *HCQ) call(ctx context.Context, address, login string) (response, error) {
	body, err := json.Marshal(request{Login: login, Password: h.lookupPassword})
	if err != nil {
		return response{}, fmt.Errorf("%w: menyusun permintaan: %v", masterpicteknik.ErrDirectoryUnreachable, err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return response{}, fmt.Errorf("%w: menyusun permintaan: %v", masterpicteknik.ErrDirectoryUnreachable, err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.SetBasicAuth(h.user, h.password)

	httpResponse, err := h.client.Do(httpRequest)
	if err != nil {
		// Galat jaringan tidak pernah dibungkus apa adanya ke pengguna: pesannya dapat
		// memuat alamat internal. Yang diteruskan hanya jenisnya.
		return response{}, fmt.Errorf("%w: %v", masterpicteknik.ErrDirectoryUnreachable, err)
	}
	defer func() { _ = httpResponse.Body.Close() }()

	content, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxResponseBytes))
	if err != nil {
		return response{}, fmt.Errorf("%w: membaca respons: %v", masterpicteknik.ErrDirectoryUnreachable, err)
	}

	// Badan dibaca lebih dulu, dan status HTTP hanya dipakai bila badannya tidak dapat
	// dipahami: HCQ menjawab kegagalan dengan badan JSON dan status yang tidak selalu 200.
	var answer response
	if err := json.Unmarshal(content, &answer); err != nil {
		return response{}, fmt.Errorf("%w: respons HCQ bukan JSON yang dikenali (status HTTP %d)",
			masterpicteknik.ErrDirectoryUnreachable, httpResponse.StatusCode)
	}
	return answer, nil
}

func firstNonEmpty(value ...string) string {
	for _, v := range value {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var _ masterpicteknik.EmployeeDirectory = (*HCQ)(nil)
