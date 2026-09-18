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

// RekeningDTO adalah bentuk rekening yang dikirim ke peramban.
//
// Terpisah dari masterrekening.Rekening supaya perubahan internal tidak bocor ke
// klien, dan supaya field yang tidak perlu dilihat peramban tidak ikut terkirim.
type RekeningDTO struct {
	NomorRekening string `json:"nomor_rekening"`
	NamaPemilik   string `json:"nama_pemilik"`
	NamaBank      string `json:"nama_bank"`
	CabangBank    string `json:"cabang_bank"`
	AlamatBank    string `json:"alamat_bank"`
	KodeBank      string `json:"kode_bank"`
	TipeRekening  string `json:"tipe_rekening"`
	Aktif         bool   `json:"aktif"`

	Email       string `json:"email"`
	Telepon     string `json:"telepon"`
	NIK         string `json:"nik"`
	Catatan     string `json:"catatan"`
	Dokumen     string `json:"id_dokumen"`
	DiinputOleh string `json:"diinput_oleh"`

	Status      string `json:"status"`
	StatusLabel string `json:"status_label"`
	Komite      string `json:"komite_approval"`

	// Waktu dikirim dalam RFC 3339 UTC. Peramban yang menampilkannya dalam WIB
	// melakukannya di satu tempat, bukan dengan menambah tujuh jam di sini.
	DiinputPada    time.Time  `json:"diinput_pada"`
	DiputuskanPada *time.Time `json:"diputuskan_pada,omitempty"`

	StatusLayanan   string `json:"status_layanan"`
	IDRekeningKasir string `json:"id_rekening_kasir"`
	ResponsKasir    string `json:"respons_kasir"`

	// DapatDipakai menjawab pertanyaan yang sesungguhnya ditanyakan layar: boleh
	// tidak rekening ini menerima pembayaran klaim. Ia dihitung di server supaya
	// dua syaratnya — disetujui DAN aktif — tidak perlu diulang di setiap layar.
	DapatDipakai bool `json:"dapat_dipakai"`
}

// DariRekening mengubah rekening domain menjadi DTO.
func DariRekening(r masterrekening.Rekening) RekeningDTO {
	return RekeningDTO{
		NomorRekening:   r.NomorRekening,
		NamaPemilik:     r.NamaPemilik,
		NamaBank:        r.NamaBank,
		CabangBank:      r.CabangBank,
		AlamatBank:      r.AlamatBank,
		KodeBank:        r.KodeBank,
		TipeRekening:    r.TipeRekening,
		Aktif:           r.Aktif,
		Email:           r.Email,
		Telepon:         r.Telepon,
		NIK:             r.NIK,
		Catatan:         r.Catatan,
		Dokumen:         r.IDDokumen,
		DiinputOleh:     r.DiinputOleh,
		Status:          string(r.Status),
		StatusLabel:     r.Status.Label(),
		Komite:          r.KomiteApproval,
		DiinputPada:     r.DiinputPada,
		DiputuskanPada:  r.DiputuskanPada,
		StatusLayanan:   r.StatusLayanan,
		IDRekeningKasir: r.IDRekeningKasir,
		ResponsKasir:    r.ResponsKasir,
		DapatDipakai:    r.DapatDipakai(),
	}
}

// BankDTO adalah satu bank pada daftar pilihan.
type BankDTO struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

// ResponsDaftar adalah jawaban GET /api/master-rekening.
type ResponsDaftar struct {
	Rekening []RekeningDTO `json:"rekening"`

	// Jumlah adalah banyaknya baris yang cocok dengan saringan SEBELUM dipotong
	// paginasi. Tanpa angka ini, layar tidak dapat menampilkan jumlah halaman.
	Jumlah int `json:"jumlah"`
	Batas  int `json:"batas"`
	Lewati int `json:"lewati"`
}

// ResponsDaftarBank adalah jawaban GET /api/master-rekening/bank.
type ResponsDaftarBank struct {
	Bank []BankDTO `json:"bank"`
}

// PermintaanSimpan adalah badan POST dan PUT master rekening.
//
// Yang TIDAK ada di sini dan itu disengaja: status persetujuan, komite, waktu
// keputusan, dan seluruh jejak Kasir. Semuanya ditetapkan server — bila peramban dapat
// mengirimnya, siapa pun yang dapat membuka layar ini dapat menerbitkan rekening yang
// langsung berstatus disetujui.
type PermintaanSimpan struct {
	NomorRekening string `json:"nomor_rekening"`
	NamaPemilik   string `json:"nama_pemilik"`
	NamaBank      string `json:"nama_bank"`
	CabangBank    string `json:"cabang_bank"`
	AlamatBank    string `json:"alamat_bank"`
	KodeBank      string `json:"kode_bank"`
	TipeRekening  string `json:"tipe_rekening"`
	Email         string `json:"email"`
	Telepon       string `json:"telepon"`
	NIK           string `json:"nik"`
	IDDokumen     string `json:"id_dokumen"`
	Catatan       string `json:"catatan"`
	Aktif         bool   `json:"aktif"`

	KodeBankLama      string `json:"kode_bank_lama"`
	NomorRekeningLama string `json:"nomor_rekening_lama"`
	NamaPemilikLama   string `json:"nama_pemilik_lama"`
}

// PermintaanPutuskan adalah badan POST keputusan komite.
type PermintaanPutuskan struct {
	// Status wajib "1" (setuju) atau "2" (tolak).
	Status    string `json:"status"`
	Catatan   string `json:"catatan"`
	IDDokumen string `json:"id_dokumen"`
}

// ResponsGalat adalah bentuk galat yang dikirim ke klien.
//
// Bentuknya sama dengan modul auth supaya klien menangani galat seluruh aplikasi
// dengan satu jalur. Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004,
// yang masih terhalang keputusan Work Owner; sampai itu ada, bentuk ini yang dipakai.
type ResponsGalat struct {
	Kode  string `json:"kode"`
	Pesan string `json:"pesan"`

	// Field menyebut kesalahan per kolom pada galat validasi, supaya layar dapat
	// menaruh pesannya di kolom yang benar alih-alih menumpuknya di atas formulir
	// (TKT-U2-002).
	Field map[string]string `json:"field,omitempty"`
}
