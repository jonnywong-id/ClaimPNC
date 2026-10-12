// Package detailpenyebabhttp adalah lapisan transport modul Detail Penyebab Kerugian.
//
// Ia memetakan permintaan HTTP menjadi perkara domain, dan galat domain menjadi kode HTTP.
// Domain tidak tahu apa pun tentang HTTP, dan paket ini tidak tahu apa pun tentang SQL.
package detailpenyebabhttp

import "claim-pnc/internal/detailpenyebab"

// # Kenapa nama field JSON berbahasa Indonesia
//
// Ia **kontrak**, bukan nama internal (`D-80`). Mengubahnya adalah perubahan yang merusak
// klien, bukan penggantian nama — sehingga ia tidak ikut dialihkan ke bahasa Inggris
// bersama identifier di dalam kode.
//
// # Nama kolom basis data TIDAK dipakai sebagai nama field
//
// `D_COL_ID`, `OLD_D_COL_ID`, dan `M_COL_ID` tidak menyebutkan isinya sama sekali, dan
// `LOSS_CODE` menyebutkan hal yang berbeda dari isinya pada kueri yang berbeda — lihat
// banner paket detailpenyebab. Yang dipakai di sini adalah **label yang tertulis di atas
// isiannya di layar**, karena itulah nama yang dikenal petugas dan yang tertulis di tiket.

// BusinessDTO adalah satu lini bisnis pada badan permintaan maupun jawaban.
type BusinessDTO struct {
	// ID adalah kode lini bisnis, POOLDATA.BUSINESS.ID.
	ID string `json:"id"`

	// Nama adalah sebutan lini bisnis, POOLDATA.BUSINESS.NOTE.
	Nama string `json:"nama"`
}

// DetailDTO adalah satu baris Detail Penyebab Kerugian pada jawaban.
type DetailDTO struct {
	// ID adalah D_COL_ID. Diterbitkan server; klien tidak pernah mengirimnya.
	ID string `json:"id"`

	// IDLama adalah OLD_D_COL_ID — ID baris ini pada sistem sebelum Pega.
	//
	// Ia DIKIRIM pada setiap jawaban meski tidak digambar di layar, supaya klien dapat
	// mengembalikannya saat menyimpan. Bila klien tidak mengembalikannya, server memakai
	// nilai yang tersimpan — lihat usecase.Service.Save.
	IDLama string `json:"id_lama"`

	// IDMaster adalah M_COL_ID — kunci induk di V_M_CAUSE_OF_LOSS.
	IDMaster string `json:"id_master"`

	// NamaMaster adalah COL_DESC — sebutan induk.
	//
	// Ia TIDAK tersimpan di baris ini melainkan dibaca dari induknya, dan karena itu
	// **tidak dapat dikirim balik** oleh klien. Lihat SaveRequest.
	//
	// Kosong berarti induknya tidak ditemukan — baris yatim, yang mungkin ada karena tidak
	// ada foreign key yang diketahui (`R-08`).
	NamaMaster string `json:"nama_master"`

	// DeskripsiKerugian adalah DESCRIPTION, berlabel "Deskripsi Kerugian" di layar.
	DeskripsiKerugian string `json:"deskripsi_kerugian"`

	// KodeKehilangan adalah LOSS_CODE, berlabel "Kode Kehilangan" di layar.
	KodeKehilangan string `json:"kode_kehilangan"`

	// StatusAktif adalah STS_AKTIF — "1" atau "0"; lihat detailpenyebab.ActiveYes.
	StatusAktif string `json:"status_aktif"`

	// LabelStatusAktif adalah sebutan status yang siap ditampilkan.
	//
	// Ia DITURUNKAN server, bukan disimpan. Alasannya: nilai kosong dan nilai "0"
	// sama-sama berarti tidak aktif bagi penyaring tetapi berbeda artinya bagi petugas —
	// dan menurunkannya di frontend berarti aturan itu hidup di dua tempat.
	LabelStatusAktif string `json:"label_status_aktif"`

	// Bisnis adalah lini bisnis tempat detail ini berlaku.
	//
	// Ia KOSONG pada jawaban daftar dan terisi pada jawaban satu baris — grid layar lama
	// pun tidak menampilkannya. Lihat detailpenyebab.Repo.List.
	Bisnis []BusinessDTO `json:"bisnis"`
}

// ListResponse adalah jawaban GET daftar.
type ListResponse struct {
	Detail []DetailDTO `json:"detail"`

	// Portal ikut dikirim supaya klien dapat memastikan jawaban yang diterimanya memang
	// milik portal yang sedang dibuka. Tanpa itu, jawaban basi dari portal sebelumnya
	// tidak dapat dibedakan dari jawaban yang benar (`R-20`).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban GET satu baris, POST, dan PUT.
type SingleResponse struct {
	Detail DetailDTO `json:"detail"`
	Portal string    `json:"portal"`
}

// MasterOptionDTO adalah satu pilihan pada isian "ID Master Kerugian".
type MasterOptionDTO struct {
	ID    string `json:"id"`
	Nama  string `json:"nama"`
	Label string `json:"label"`
}

// LookupResponse adalah jawaban pencarian daftar pilihan.
//
// Kedua daftarnya berada di satu bentuk jawaban, tetapi hanya SATU yang terisi pada setiap
// permintaan — yang mana, ditentukan jalurnya. Menyatukannya membuat klien punya satu
// bentuk jawaban untuk diurai, bukan dua.
type LookupResponse struct {
	Master []MasterOptionDTO `json:"master"`
	Bisnis []BusinessDTO     `json:"bisnis"`
	Portal string            `json:"portal"`
}

// ActiveOptionDTO adalah satu pilihan pada dropdown Status Aktif.
type ActiveOptionDTO struct {
	Kode  string `json:"kode"`
	Label string `json:"label"`
}

// OptionsResponse adalah jawaban daftar pilihan yang tetap.
//
// Ia dilayani server, bukan ditulis tangan di frontend, supaya kedua sisi tidak pernah
// berbeda pendapat tentang nilai apa yang sah — dan supaya penambahan pilihan kelak
// menyentuh satu tempat.
type OptionsResponse struct {
	StatusAktif []ActiveOptionDTO `json:"status_aktif"`
}

// SaveRequest adalah badan permintaan POST dan PUT.
//
// # Yang TIDAK ada di sini, dan itu disengaja
//
//	id            diterbitkan server saat menambah; datang dari jalur URL saat mengubah
//	nama_master   milik induk; menerimanya berarti sebutan tersimpan dapat berbeda dari induknya
//	label_status_aktif  turunan dari status_aktif
//
// Ketiganya DIKIRIM pada setiap jawaban tetapi tidak dapat dikirim balik. Klien yang
// mengembalikan seluruh objek apa adanya akan ditolak dengan `permintaan_cacat`, bukan
// diabaikan diam-diam — lihat Handler.readRequest.
type SaveRequest struct {
	IDLama            string        `json:"id_lama"`
	IDMaster          string        `json:"id_master"`
	DeskripsiKerugian string        `json:"deskripsi_kerugian"`
	KodeKehilangan    string        `json:"kode_kehilangan"`
	StatusAktif       string        `json:"status_aktif"`
	Bisnis            []BusinessDTO `json:"bisnis"`
}

// toInput mengubah badan permintaan menjadi nilai domain.
func (r SaveRequest) toInput() detailpenyebab.Input {
	business := make([]detailpenyebab.Business, 0, len(r.Bisnis))
	for _, line := range r.Bisnis {
		business = append(business, detailpenyebab.Business{ID: line.ID, Name: line.Nama})
	}

	return detailpenyebab.Input{
		LegacyID:    r.IDLama,
		MasterID:    r.IDMaster,
		Description: r.DeskripsiKerugian,
		LossCode:    r.KodeKehilangan,
		Active:      r.StatusAktif,
		Business:    business,
	}
}

// toDTO mengubah satu baris domain menjadi bentuk jawaban.
func toDTO(one detailpenyebab.CauseOfLossDetail) DetailDTO {
	business := make([]BusinessDTO, 0, len(one.Business))
	for _, line := range one.Business {
		business = append(business, BusinessDTO{ID: line.ID, Nama: line.Name})
	}

	return DetailDTO{
		ID:                one.ID,
		IDLama:            one.LegacyID,
		IDMaster:          one.MasterID,
		NamaMaster:        one.MasterLabel,
		DeskripsiKerugian: one.Description,
		KodeKehilangan:    one.LossCode,
		StatusAktif:       one.Active,
		LabelStatusAktif:  detailpenyebab.ActiveLabel(one.Active),
		Bisnis:            business,
	}
}

// toListDTO mengubah daftar domain menjadi bentuk jawaban.
//
// Slice-nya selalu dibuat, tidak pernah nil, supaya jawaban kosong terbaca `[]` dan bukan
// `null` — klien yang memetakannya langsung tidak perlu memeriksa nil lebih dulu.
func toListDTO(list []detailpenyebab.CauseOfLossDetail) []DetailDTO {
	result := make([]DetailDTO, 0, len(list))
	for _, one := range list {
		result = append(result, toDTO(one))
	}
	return result
}

// toMasterDTO mengubah daftar pilihan induk menjadi bentuk jawaban.
//
// `label` menggabungkan kode dan sebutannya, karena itulah yang ditampilkan daftar turun —
// dan menyusunnya di server membuat kedua sisi tidak pernah menampilkannya berbeda.
func toMasterDTO(list []detailpenyebab.MasterOption) []MasterOptionDTO {
	result := make([]MasterOptionDTO, 0, len(list))
	for _, one := range list {
		label := one.Label
		if one.ID != "" && label != "" {
			label = one.ID + " — " + one.Label
		} else if label == "" {
			label = one.ID
		}
		result = append(result, MasterOptionDTO{ID: one.ID, Nama: one.Label, Label: label})
	}
	return result
}

// toBusinessDTO mengubah daftar pilihan lini bisnis menjadi bentuk jawaban.
func toBusinessDTO(list []detailpenyebab.Business) []BusinessDTO {
	result := make([]BusinessDTO, 0, len(list))
	for _, one := range list {
		result = append(result, BusinessDTO{ID: one.ID, Nama: one.Name})
	}
	return result
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama persis dengan modul lain — `{kode, pesan, detail}` — sehingga klien tidak
// menghadapi dua bentuk galat yang berbeda.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}
