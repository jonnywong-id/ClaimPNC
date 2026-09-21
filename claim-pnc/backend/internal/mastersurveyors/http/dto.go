// Package mastersurveyorshttp adalah lapisan transport modul Master Surveyors: bentuk
// permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// portalhttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `mastersurveyorshttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package mastersurveyorshttp

import (
	"sort"
	"time"

	"claim-pnc/internal/mastersurveyors"
)

// SurveyorDTO adalah bentuk satu surveyor yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari mastersurveyors.Surveyor. Memakai tipe modul langsung
// sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`docs/Steering/08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya Indonesia karena ia KONTRAK, bukan nama internal (`D-80`).
type SurveyorDTO struct {
	ID string `json:"id"`

	TypeCode string `json:"kode_tipe"`
	// TypeDescription ikut dikirim supaya grid tidak perlu memanggil master tipe secara
	// terpisah untuk setiap baris. Satu layar sebaiknya dilayani satu permintaan
	// (`docs/Steering/10-API-STRATEGY.md` §1).
	TypeDescription string `json:"nama_tipe"`

	Name         string `json:"nama"`
	Address      string `json:"alamat"`
	PostalCode   string `json:"kode_pos"`
	State        string `json:"provinsi"`
	Phone        string `json:"telepon"`
	Fax          string `json:"faksimile"`
	Email        string `json:"email"`
	OtherContact string `json:"kontak_lain"`
	BranchCode   string `json:"kode_cabang"`
	BranchName   string `json:"nama_cabang"`
	AppLogin     string `json:"login_aplikasi"`
	DocumentID   string `json:"id_dokumen"`

	Status string `json:"status"`
	// StatusLabel dikirim dari server supaya empat layar yang menampilkannya tidak dapat
	// menerjemahkan kode yang sama menjadi dua sebutan berbeda.
	StatusLabel string `json:"status_label"`

	Committee string `json:"komite"`
	Note      string `json:"catatan"`

	// DecidedAt nil selama komite belum memutuskan. Dikirim sebagai penunjuk supaya
	// "belum diputuskan" terbedakan dari "diputuskan pada waktu nol".
	DecidedAt *time.Time `json:"tanggal_keputusan"`

	// RequiresAppLogin dihitung server dari kode tipenya. Layar memakainya untuk menandai
	// kolom login sebagai wajib — dan aturannya tetap DITEGAKKAN di server, karena apa pun
	// yang hanya ditegakkan di peramban bukan aturan (`D-59`).
	RequiresAppLogin bool `json:"wajib_login_aplikasi"`
}

// toDTO menyalin nilai domain menjadi bentuk kontrak.
func toDTO(s mastersurveyors.Surveyor) SurveyorDTO {
	return SurveyorDTO{
		ID:               s.ID,
		TypeCode:         s.TypeCode,
		TypeDescription:  s.TypeDescription,
		Name:             s.Name,
		Address:          s.Address,
		PostalCode:       s.PostalCode,
		State:            s.State,
		Phone:            s.Phone,
		Fax:              s.Fax,
		Email:            s.Email,
		OtherContact:     s.OtherContact,
		BranchCode:       s.BranchCode,
		BranchName:       s.BranchName,
		AppLogin:         s.AppLogin,
		DocumentID:       s.DocumentID,
		Status:           string(s.Status),
		StatusLabel:      s.Status.Label(),
		Committee:        s.Committee,
		Note:             s.Note,
		DecidedAt:        s.DecidedAt,
		RequiresAppLogin: s.RequiresAppLogin(),
	}
}

// ListResponse adalah jawaban GET /api/master/surveyor.
type ListResponse struct {
	Surveyor []SurveyorDTO `json:"surveyor"`

	// Total adalah jumlah seluruh baris yang cocok SEBELUM dipotong paginasi — bukan
	// panjang senarai di atas. Tanpa angka ini layar tidak dapat tahu masih ada halaman
	// berikutnya atau tidak.
	Total int `json:"total"`

	// Portal menyebut entitas yang BENAR-BENAR menjawab permintaan ini.
	//
	// Ia dikirim pada setiap jawaban, bukan diandaikan sama dengan yang diminta: satu
	// aplikasi melayani empat badan hukum dengan basis data terpisah (`ADR-0030`), dan
	// "data siapa ini" tidak boleh hanya ditebak dari keadaan layar (`R-20`).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban untuk satu surveyor: ambil, ajukan, ubah, dan putuskan.
type SingleResponse struct {
	Surveyor SurveyorDTO `json:"surveyor"`
	Portal   string      `json:"portal"`
}

// SaveRequest adalah isian form ajukan dan ubah.
//
// Empat field yang dimiliki sistem — status, komite, tanggal keputusan, catatan komite —
// SENGAJA TIDAK ADA di sini. Menerimanya dari badan permintaan akan membuat siapa pun
// yang boleh menyunting surveyor dapat menyetujuinya sendiri dalam satu permintaan.
//
// ID juga tidak diterima: pada pengajuan ia dibuat penyimpanan, dan pada perubahan ia
// diambil dari jalur URL.
type SaveRequest struct {
	TypeCode     string `json:"kode_tipe"`
	Name         string `json:"nama"`
	Address      string `json:"alamat"`
	PostalCode   string `json:"kode_pos"`
	State        string `json:"provinsi"`
	Phone        string `json:"telepon"`
	Fax          string `json:"faksimile"`
	Email        string `json:"email"`
	OtherContact string `json:"kontak_lain"`
	BranchCode   string `json:"kode_cabang"`
	BranchName   string `json:"nama_cabang"`
	AppLogin     string `json:"login_aplikasi"`
	DocumentID   string `json:"id_dokumen"`
}

// DecisionRequest adalah keputusan komite atas satu surveyor.
type DecisionRequest struct {
	// Status wajib "1" (setuju) atau "2" (tolak). Nilai lain ditolak — termasuk "0",
	// karena "belum memutuskan" bukan keputusan yang dapat dikirim.
	Status string `json:"status"`
	Note   string `json:"catatan"`
}

// ViolationDTO adalah satu pelanggaran aturan isian.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
//
// Bentuknya sama dengan modul lain supaya peramban punya SATU cara menangani galat, bukan
// satu cara per modul (`docs/Steering/10-API-STRATEGY.md` §3).
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// violationsOf mengubah peta pelanggaran menjadi senarai yang urutannya tetap.
//
// Diurutkan menurut nama field, dan itu bukan kerapian: peta Go tidak punya urutan,
// sehingga tanpa pengurutan pesan galat yang sama akan tiba dalam urutan berbeda setiap
// kali — menyulitkan pengujian dan membuat layar memindahkan penanda kesalahan tanpa
// sebab.
func violationsOf(field map[string]string) []ViolationDTO {
	names := make([]string, 0, len(field))
	for name := range field {
		names = append(names, name)
	}
	sort.Strings(names)

	detail := make([]ViolationDTO, 0, len(names))
	for _, name := range names {
		detail = append(detail, ViolationDTO{Field: name, Message: field[name]})
	}
	return detail
}
