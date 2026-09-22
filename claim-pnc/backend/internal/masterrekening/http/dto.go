// Package masterrekeninghttp adalah lapisan transport modul Master Rekening.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola yang sama dengan
// auth/http dan portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama
// paketnya `masterrekeninghttp` supaya tidak menutupi `net/http`.
package masterrekeninghttp

import (
	"time"

	"claim-pnc/internal/masterrekening"
)

// AccountDTO adalah bentuk rekening yang dikirim ke peramban.
//
// Terpisah dari masterrekening.Account supaya perubahan internal tidak bocor ke
// klien, dan supaya field yang tidak perlu dilihat peramban tidak ikut terkirim.
type AccountDTO struct {
	Number      string `json:"nomor_rekening"`
	OwnerName   string `json:"nama_pemilik"`
	BankName    string `json:"nama_bank"`
	BankBranch  string `json:"cabang_bank"`
	BankAddress string `json:"alamat_bank"`
	BankCode    string `json:"kode_bank"`
	AccountType string `json:"tipe_rekening"`
	Active      bool   `json:"aktif"`

	Email     string `json:"email"`
	Phone     string `json:"telepon"`
	NIK       string `json:"nik"`
	Note      string `json:"catatan"`
	Document  string `json:"id_dokumen"`
	CreatedBy string `json:"diinput_oleh"`

	Status      string `json:"status"`
	StatusLabel string `json:"status_label"`
	Committee   string `json:"komite_approval"`

	// Waktu dikirim dalam RFC 3339 UTC. Peramban yang menampilkannya dalam WIB
	// melakukannya di satu tempat, bukan dengan menambah tujuh jam di sini.
	CreatedAt time.Time  `json:"diinput_pada"`
	DecidedAt *time.Time `json:"diputuskan_pada,omitempty"`

	ServiceStatus    string `json:"status_layanan"`
	CashierAccountID string `json:"id_rekening_kasir"`
	CashierResponse  string `json:"respons_kasir"`

	// Usable menjawab pertanyaan yang sesungguhnya ditanyakan layar: boleh
	// tidak rekening ini menerima pembayaran klaim. Ia dihitung di server supaya
	// dua syaratnya — disetujui DAN aktif — tidak perlu diulang di setiap layar.
	Usable bool `json:"dapat_dipakai"`
}

// FromAccount mengubah rekening domain menjadi DTO.
func FromAccount(r masterrekening.Account) AccountDTO {
	return AccountDTO{
		Number:           r.Number,
		OwnerName:        r.OwnerName,
		BankName:         r.BankName,
		BankBranch:       r.BankBranch,
		BankAddress:      r.BankAddress,
		BankCode:         r.BankCode,
		AccountType:      r.AccountType,
		Active:           r.Active,
		Email:            r.Email,
		Phone:            r.Phone,
		NIK:              r.NIK,
		Note:             r.Note,
		Document:         r.DocumentID,
		CreatedBy:        r.CreatedBy,
		Status:           string(r.Status),
		StatusLabel:      r.Status.Label(),
		Committee:        r.CommitteeApproval,
		CreatedAt:        r.CreatedAt,
		DecidedAt:        r.DecidedAt,
		ServiceStatus:    r.ServiceStatus,
		CashierAccountID: r.CashierAccountID,
		CashierResponse:  r.CashierResponse,
		Usable:           r.Usable(),
	}
}

// BankDTO adalah satu bank pada daftar pilihan.
type BankDTO struct {
	Code string `json:"kode"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master-rekening.
type ListResponse struct {
	Account []AccountDTO `json:"rekening"`

	// Count adalah banyaknya baris yang cocok dengan saringan SEBELUM dipotong
	// paginasi. Tanpa angka ini, layar tidak dapat menampilkan jumlah halaman.
	Count  int `json:"jumlah"`
	Limit  int `json:"batas"`
	Offset int `json:"lewati"`
}

// BankListResponse adalah jawaban GET /api/master-rekening/bank.
type BankListResponse struct {
	Bank []BankDTO `json:"bank"`
}

// SaveRequest adalah badan POST dan PUT master rekening.
//
// Yang TIDAK ada di sini dan itu disengaja: status persetujuan, komite, waktu
// keputusan, dan seluruh jejak Kasir. Semuanya ditetapkan server — bila peramban dapat
// mengirimnya, siapa pun yang dapat membuka layar ini dapat menerbitkan rekening yang
// langsung berstatus disetujui.
type SaveRequest struct {
	Number      string `json:"nomor_rekening"`
	OwnerName   string `json:"nama_pemilik"`
	BankName    string `json:"nama_bank"`
	BankBranch  string `json:"cabang_bank"`
	BankAddress string `json:"alamat_bank"`
	BankCode    string `json:"kode_bank"`
	AccountType string `json:"tipe_rekening"`
	Email       string `json:"email"`
	Phone       string `json:"telepon"`
	NIK         string `json:"nik"`
	DocumentID  string `json:"id_dokumen"`
	Note        string `json:"catatan"`
	Active      bool   `json:"aktif"`

	PreviousBankCode  string `json:"kode_bank_lama"`
	PreviousNumber    string `json:"nomor_rekening_lama"`
	PreviousOwnerName string `json:"nama_pemilik_lama"`
}

// DecideRequest adalah badan POST keputusan komite.
type DecideRequest struct {
	// Status wajib "1" (setuju) atau "2" (tolak).
	Status     string `json:"status"`
	Note       string `json:"catatan"`
	DocumentID string `json:"id_dokumen"`
}

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
//
// Bentuknya sama dengan modul auth supaya klien menangani galat seluruh aplikasi
// dengan satu jalur. Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004,
// yang masih terhalang keputusan Work Owner; sampai itu ada, bentuk ini yang dipakai.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Field menyebut kesalahan per kolom pada galat validasi, supaya layar dapat
	// menaruh pesannya di kolom yang benar alih-alih menumpuknya di atas formulir
	// (TKT-U2-002).
	//
	// Ia bentuk LAMA dan dipertahankan supaya klien yang masih membacanya tidak putus.
	// Yang dikirim sekarang adalah Detail di bawah.
	Field map[string]string `json:"field,omitempty"`

	// Detail menyebut kesalahan per kolom sebagai SENARAI TERURUT.
	//
	// Ia menggantikan Field karena urutan peta Go sengaja acak, sehingga dua permintaan
	// yang sama dapat menghasilkan badan respons yang berbeda — uji kontrak menjadi
	// rapuh dan log sulit dibandingkan. Penyusunnya ada di errors.go (`violationsFrom`).
	//
	// Kedua bentuk dikenali klien: `APIError.violations()` menggabungkan `field` dan
	// `detail` menjadi satu peta, sehingga layar tidak perlu memilih di antara keduanya.
	// Penyeragaman kontraknya masuk TKT-F1-004.
	Detail []PelanggaranDTO `json:"detail,omitempty"`
}

// PelanggaranDTO adalah satu aturan yang dilanggar beserta kolom yang melanggarnya.
//
// Nama kuncinya `field`, sama dengan modul `masterstatus`. Modul `masterstatusprogres`
// memakai `kolom` untuk hal yang sama — perbedaan yang belum diseragamkan, dan yang
// ditelan `APIError.violations()` di sisi klien sampai TKT-F1-004 menyelesaikannya.
type PelanggaranDTO struct {
	Field string `json:"field"`
	Pesan string `json:"pesan"`
}
