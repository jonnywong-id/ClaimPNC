// Package masterloginhttp adalah lapisan transport modul Master Login.
//
// Namanya mengikuti `D-81`: folder modul memakai nama modul bisnis apa adanya
// ("Master Login"), dan paket transportnya menambahkan akhiran `http` tanpa tanda hubung
// karena Go tidak mengizinkannya. Pola yang sama dipakai `masterspareparthttp`,
// `masterpanelhttp`, dan `masterbengkelhttp`.
//
// Lapisan Transport — ia boleh tahu Domain, dan dilarang tahu SQL maupun nama tabel.
package masterloginhttp

import "claim-pnc/internal/masterlogin"

// SurveyorLoginDTO adalah satu login surveyor sebagaimana dilihat klien.
//
// # Kenapa nama field JSON-nya bahasa Indonesia
//
// Ia KONTRAK, bukan nama internal (`D-80`). Nama tipe, field Go, dan variabel di modul ini
// seluruhnya bahasa Inggris; yang tetap Indonesia hanyalah yang dipakai di luar kode — dan
// nama field JSON termasuk di dalamnya, karena mengubahnya adalah perubahan yang merusak
// klien, bukan penggantian nama.
//
// Namanya mengikuti LABEL DI LAYAR, bukan nama kolom: `nama`, `login`, `email`, `telp`, dan
// `alamat` adalah kelima caption pada `Section/BrowseLoginSurveyor-Section.xml`. Dua yang
// terakhir memakai nama kolomnya karena keduanya tidak punya label — lihat di bawah.
type SurveyorLoginDTO struct {
	// Name adalah kolom NAMA — caption "Nama".
	Name string `json:"nama"`

	// Login adalah kolom LOGIN — caption "Login", dan KUNCI baris ini.
	//
	// Ia yang dipakai pada jalur URL `GET`/`PUT`, bukan sebuah ID terpisah: tabelnya tidak
	// punya kolom kunci lain, dan setiap pernyataan simpannya menyaring `where login = ...`.
	Login string `json:"login"`

	// Email adalah kolom EMAIL — caption "Email".
	Email string `json:"email"`

	// Phone adalah kolom TELP — caption "Telp".
	Phone string `json:"telp"`

	// Address adalah kolom ALAMAT — caption "Alamat".
	Address string `json:"alamat"`

	// LoginStatus adalah kolom STSLOGIN.
	//
	// TIDAK ADA labelnya di layar Pega — kolom ini tidak digambar sama sekali. Ia tetap
	// dikirim karena layar baru MENAMPILKANNYA sebagai keterangan baca-saja pada form:
	// nilainya menentukan peran seseorang, dan menyembunyikan hal yang tersimpan tidak
	// membuatnya tidak tersimpan.
	//
	// Ia tidak dapat dikirim balik; lihat SaveRequest.
	LoginStatus string `json:"status_login"`

	// LeaderLogin adalah kolom LOGINLEADER — login orang yang menjadi leader baris ini.
	//
	// TIDAK ADA labelnya di layar Pega, dengan alasan yang sama seperti LoginStatus. Ia
	// dikirim karena tanpa itu tidak ada cara apa pun mengetahui sebuah baris bertaut ke tim
	// siapa — dan baris ber-LOGINLEADER kosong tidak dapat dibedakan dari yang bertim.
	//
	// Ia tidak dapat dikirim balik; lihat SaveRequest.
	LeaderLogin string `json:"login_leader"`
}

// ListResponse adalah jawaban GET /master/login.
type ListResponse struct {
	// Login selalu berupa array, tidak pernah null — layar tidak perlu menjaga dua bentuk
	// kosong yang berbeda. Lihat toListDTO.
	Login []SurveyorLoginDTO `json:"login_surveyor"`

	// Portal adalah entitas yang menjawab. Ia dikirim pada SETIAP jawaban, dan itu bukan
	// hiasan: satu aplikasi melayani empat badan hukum dengan basis data terpisah, dan
	// "data siapa ini" tidak boleh hanya diandaikan (ADR-0030, R-20).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban satu baris — dipakai Get, Create, dan Save.
type SingleResponse struct {
	Login  SurveyorLoginDTO `json:"login_surveyor"`
	Portal string           `json:"portal"`
}

// SaveRequest adalah badan permintaan tambah dan simpan.
//
// LIMA isian pada layar, EMPAT di sini. Yang TIDAK ada, dan ketiadaannya disengaja:
//
//	login        diturunkan server dari nama; lihat masterlogin.DeriveLogin
//	status_login selalu "Member" pada penambahan, dipertahankan pada penyuntingan
//	login_leader diturunkan dari leader milik pengguna yang menyimpan
//
// Ketiganya dikirim KELUAR pada setiap jawaban tetapi tidak dapat dikirim MASUK. Menerimanya
// berarti membuka jalan menetapkan kunci baris, peran, dan induk tim lewat permintaan HTTP
// biasa — tiga hal yang di sistem lama pun tidak pernah berada di tangan pengguna, karena
// ketiganya tidak digambar di layar.
//
// Server memasang `DisallowUnknownFields`, sehingga mengirim salah satunya DITOLAK sebagai
// permintaan cacat — bukan diabaikan diam-diam. Cacat pada klien terlihat saat pertama
// dicoba.
type SaveRequest struct {
	Name    string `json:"nama"`
	Email   string `json:"email"`
	Phone   string `json:"telp"`
	Address string `json:"alamat"`
}

// toInput mengubah badan permintaan menjadi nilai domain.
func (r SaveRequest) toInput() masterlogin.Input {
	return masterlogin.Input{
		Name:    r.Name,
		Email:   r.Email,
		Phone:   r.Phone,
		Address: r.Address,
	}
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}` — ditambah `detail` untuk pelanggaran
// per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan mencocokkan
// teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
//
// Nama kuncinya `kolom`, mengikuti masterbengkel, masterpanel, mastersparepart,
// masterstatusprogres, masterautoclaim, dan masterkategorisparepart. Penyeragamannya dengan
// masterstatus — yang memakai `field` — adalah TKT-F1-004 yang masih terhalang. Frontend
// sudah menampung keduanya lewat `APIError.violations()`.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah satu baris domain menjadi bentuk yang dikirim ke klien.
func toDTO(one masterlogin.SurveyorLogin) SurveyorLoginDTO {
	return SurveyorLoginDTO{
		Name:        one.Name,
		Login:       one.Login,
		Email:       one.Email,
		Phone:       one.Phone,
		Address:     one.Address,
		LoginStatus: one.LoginStatus,
		LeaderLogin: one.LeaderLogin,
	}
}

// toListDTO mengubah daftar domain menjadi daftar DTO.
//
// Selalu mengembalikan slice yang TIDAK nil, sehingga JSON-nya `[]` dan bukan `null`. Tanpa
// ini, layar harus menjaga dua bentuk kosong yang berbeda — dan satu layar yang lupa akan
// gagal menggambar tabel kosong.
func toListDTO(list []masterlogin.SurveyorLogin) []SurveyorLoginDTO {
	result := make([]SurveyorLoginDTO, 0, len(list))
	for _, one := range list {
		result = append(result, toDTO(one))
	}
	return result
}
