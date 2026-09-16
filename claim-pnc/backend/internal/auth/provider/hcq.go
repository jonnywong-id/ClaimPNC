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
const kodeBerhasilHCQ = "200"

// jenisLayananLogin adalah nilai kolom TYPESERVICE pada POOLDATA.GCNM_CONNECT_REST
// yang menandai baris alamat layanan login HCQ.
const jenisLayananLogin = "HCQ-LOGIN"

// KatalogLayanan adalah seam ke daftar alamat layanan luar.
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
type KatalogLayanan interface {
	// AlamatLayanan mengembalikan SERVICENAME untuk satu app dan jenis layanan.
	// Ia mengembalikan ErrLayananTidakTerdaftar bila barisnya tidak ada.
	AlamatLayanan(ctx context.Context, app, jenisLayanan string) (string, error)
}

// ErrLayananTidakTerdaftar dikembalikan KatalogLayanan bila baris yang diminta tidak
// ada di POOLDATA.GCNM_CONNECT_REST.
var ErrLayananTidakTerdaftar = errors.New("provider: alamat layanan tidak terdaftar")

// HCQ memverifikasi kredensial karyawan ke API internal HCC/HCQ.
type HCQ struct {
	katalog     KatalogLayanan
	klien       *http.Client
	portalAlias string
	pengguna    string
	kataSandi   string
}

// OpsiHCQ adalah bahan pembentuk provider HCQ.
type OpsiHCQ struct {
	Katalog KatalogLayanan

	// PortalAlias mengisi kolom APP saat mencari alamat layanan. Tiap portal entitas
	// boleh punya endpoint HCQ sendiri (keputusan Work Owner 2026-09-16).
	PortalAlias string

	// Pengguna dan KataSandi adalah kredensial Basic Auth ke API HCQ — kredensial
	// aplikasi, bukan kredensial pengguna yang sedang masuk.
	Pengguna  string
	KataSandi string

	BatasWaktu time.Duration
	Klien      *http.Client
}

// HCQBaru membentuk provider HCQ dan menolak bahan yang tidak lengkap.
func HCQBaru(o OpsiHCQ) (*HCQ, error) {
	switch {
	case o.Katalog == nil:
		return nil, errors.New("provider: katalog layanan wajib diisi")
	case strings.TrimSpace(o.PortalAlias) == "":
		return nil, errors.New("provider: alias portal wajib diisi")
	case o.Pengguna == "":
		return nil, errors.New("provider: HCQ_LOGIN_USER wajib diisi")
	case o.KataSandi == "":
		return nil, errors.New("provider: HCQ_LOGIN_PASSWORD wajib diisi")
	}

	klien := o.Klien
	if klien == nil {
		batas := o.BatasWaktu
		if batas <= 0 {
			batas = 15 * time.Second
		}
		// Batas waktu wajib ada. Tanpa itu, satu API yang menggantung menahan goroutine
		// permintaan sampai pengguna menyerah — dan pada jam sibuk itu menumpuk.
		klien = &http.Client{Timeout: batas}
	}

	return &HCQ{
		katalog:     o.Katalog,
		klien:       klien,
		portalAlias: strings.ToUpper(strings.TrimSpace(o.PortalAlias)),
		pengguna:    o.Pengguna,
		kataSandi:   o.KataSandi,
	}, nil
}

// permintaanHCQ adalah badan permintaan ke API HCQ.
type permintaanHCQ struct {
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
//	Placement.IsActive                 direkam; lihat catatan pada auth.Profil
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

// Verifikasi memanggil API HCQ dan menerjemahkan jawabannya menjadi profil atau galat.
//
// Pemetaan galatnya disengaja:
//   - pyErrorCode "200"            → berhasil
//   - kode lain                    → ErrKredensialSalah, karena bisnis menetapkan
//     kegagalan HCQ diteruskan ke jalur kedua
//   - jaringan/HTTP/JSON bermasalah → ErrSistemTidakTerhubung, yang dibedakan supaya
//     pengguna tahu ini bukan salah ketiknya
func (h *HCQ) Verifikasi(ctx context.Context, k auth.Kredensial) (auth.Profil, error) {
	alamat, err := h.katalog.AlamatLayanan(ctx, h.portalAlias, jenisLayananLogin)
	if err != nil {
		if errors.Is(err, ErrLayananTidakTerdaftar) {
			return auth.Profil{}, fmt.Errorf(
				"%w: baris POOLDATA.GCNM_CONNECT_REST dengan APP=%q dan TYPESERVICE=%q belum ada",
				auth.ErrSistemTidakTerhubung, h.portalAlias, jenisLayananLogin)
		}
		return auth.Profil{}, fmt.Errorf("%w: membaca alamat layanan: %v", auth.ErrSistemTidakTerhubung, err)
	}

	jawaban, err := h.panggil(ctx, alamat, k)
	if err != nil {
		return auth.Profil{}, err
	}

	if jawaban.Response.ErrorCode != kodeBerhasilHCQ {
		// Teks Result dari HCQ tidak diteruskan ke pengguna: ia dapat membedakan
		// "username tidak ditemukan" dari "password salah", dan itu membocorkan
		// keberadaan akun.
		return auth.Profil{}, auth.ErrKredensialSalah
	}

	orang := jawaban.EmpResponse.Person
	penempatan := jawaban.EmpResponse.Placement

	profil := auth.Profil{
		// NIK dan nama diambil dari Person lebih dulu; Placement dipakai sebagai
		// cadangan karena kedua blok memuat field yang sama dan pada sebagian respons
		// hanya salah satunya terisi.
		Identitas:     pilihTidakKosong(orang.NIK, penempatan.NIK),
		Nama:          pilihTidakKosong(orang.Name, penempatan.Name),
		Jenis:         auth.Karyawan,
		Login:         pilihTidakKosong(orang.Login, jawaban.Login, k.NamaPengguna),
		Email:         strings.TrimSpace(orang.Email),
		Perusahaan:    pilihTidakKosong(orang.Company, penempatan.Company),
		Cabang:        strings.TrimSpace(penempatan.BranchName),
		KodeCabang:    strings.TrimSpace(penempatan.BranchCode),
		Jabatan:       strings.TrimSpace(penempatan.PositionName),
		AktifDiSumber: penempatan.IsActive,
	}
	if err := profil.Periksa(); err != nil {
		// HCQ menjawab "berhasil" tetapi tidak menyertakan NIK atau nama. Meneruskannya
		// berarti pengguna masuk tanpa dapat dikenali data klaimnya sendiri.
		return auth.Profil{}, err
	}
	return profil, nil
}

func (h *HCQ) panggil(ctx context.Context, alamat string, k auth.Kredensial) (responsHCQ, error) {
	badan, err := json.Marshal(permintaanHCQ{Login: k.NamaPengguna, Password: k.KataSandi})
	if err != nil {
		return responsHCQ{}, fmt.Errorf("%w: menyusun permintaan: %v", auth.ErrSistemTidakTerhubung, err)
	}

	permintaan, err := http.NewRequestWithContext(ctx, http.MethodPost, alamat, bytes.NewReader(badan))
	if err != nil {
		return responsHCQ{}, fmt.Errorf("%w: menyusun permintaan: %v", auth.ErrSistemTidakTerhubung, err)
	}
	permintaan.Header.Set("Content-Type", "application/json")
	permintaan.Header.Set("Accept", "application/json")
	permintaan.SetBasicAuth(h.pengguna, h.kataSandi)

	respons, err := h.klien.Do(permintaan)
	if err != nil {
		// Galat jaringan tidak pernah dibungkus apa adanya ke pengguna: pesannya dapat
		// memuat alamat internal. Yang diteruskan hanya jenisnya.
		return responsHCQ{}, fmt.Errorf("%w: %v", auth.ErrSistemTidakTerhubung, err)
	}
	defer func() { _ = respons.Body.Close() }()

	// Batas ukuran badan: respons yang tidak wajar besar tidak boleh menghabiskan
	// memori proses yang sedang melayani pengguna lain.
	isi, err := io.ReadAll(io.LimitReader(respons.Body, 1<<20))
	if err != nil {
		return responsHCQ{}, fmt.Errorf("%w: membaca respons: %v", auth.ErrSistemTidakTerhubung, err)
	}

	// HCQ menjawab kegagalan kredensial dengan badan JSON dan — pada contoh yang
	// diberikan — status HTTP yang tidak selalu 200. Karena itu badan tetap dibaca
	// lebih dulu, dan status HTTP hanya dipakai bila badannya tidak dapat dipahami.
	var jawaban responsHCQ
	if err := json.Unmarshal(isi, &jawaban); err != nil {
		return responsHCQ{}, fmt.Errorf("%w: respons HCQ bukan JSON yang dikenali (status HTTP %d)",
			auth.ErrSistemTidakTerhubung, respons.StatusCode)
	}
	if jawaban.Response.ErrorCode == "" {
		return responsHCQ{}, fmt.Errorf("%w: respons HCQ tanpa pyErrorCode (status HTTP %d)",
			auth.ErrSistemTidakTerhubung, respons.StatusCode)
	}
	return jawaban, nil
}

func pilihTidakKosong(nilai ...string) string {
	for _, n := range nilai {
		if potong := strings.TrimSpace(n); potong != "" {
			return potong
		}
	}
	return ""
}
