package monitoringslinkojk

import (
	"errors"
	"strings"
)

// Kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrCallerUnknown     → identitas pemanggil tidak terbaca.
//   - ErrUnknownSegment    → segmen yang diminta bukan D01 maupun F06.
//   - ErrWriteNotAvailable → aksi tulis diminta, padahal modul ini belum menulis.
//
// ErrWriteNotAvailable ada meski tidak ada satu pun rute yang menulis, dan itu disengaja:
// ia yang dipakai menjawab pemanggil yang menekan "Proses Data Klaim", "SLIK OJK", atau
// "Upload Data Klaim", sehingga jawabannya menyebut SEBABNYA alih-alih 404 yang terbaca
// seperti salah alamat. Lihat monitoringslinkojkhttp.Mount untuk sebab masing-masing.
var (
	ErrCallerUnknown     = errors.New("monitoringslinkojk: identitas pemanggil tidak terbaca")
	ErrUnknownSegment    = errors.New("monitoringslinkojk: segmen tidak dikenal")
	ErrWriteNotAvailable = errors.New("monitoringslinkojk: modul ini belum menulis apa pun")

	// ErrSenderNotConfigured — pengiriman ke SLIK diminta, tetapi alamat layanannya
	// belum diatur.
	//
	// Ia BUKAN kegagalan jaringan melainkan kegagalan konfigurasi, dan keduanya wajib
	// dapat dibedakan: yang satu dicoba lagi, yang satu menunggu kontrak layanan
	// `Rest_SendDataClientBasedDebitur` yang **tidak ada di export** (`R-16`).
	ErrSenderNotConfigured = errors.New("monitoringslinkojk: layanan SLIK belum dikonfigurasi")

	// ErrNothingToProcess — tidak ada satu baris pun yang dapat disusun.
	//
	// Dibedakan dari "berhasil menyusun nol baris" supaya layar dapat mengatakan
	// SEBABNYA. Pelapor yang menekan tombol dan tidak melihat apa pun berubah tidak
	// punya cara membedakan "tidak ada data" dari "tombolnya rusak".
	ErrNothingToProcess = errors.New("monitoringslinkojk: tidak ada data yang dapat disusun")
)

// Field yang dapat membawa pelanggaran validasi.
//
// NAMA konstantanya berbahasa Inggris (`D-80`); NILAINYA berbahasa Indonesia karena ia
// nama field JSON — kontrak yang dibaca klien, dan termasuk pengecualian `D-80`.
//
// Nilainya SAMA PERSIS dengan nama parameter pada dto. Bila keduanya berbeda, pesannya
// tetap sampai ke layar tetapi tidak menempel pada isian mana pun.
//
// # Kedua nama tanggal sengaja mempertahankan alias Pega
//
// Keputusan Work Owner 2026-09-26. `date_of_loss` adalah isian berlabel **"Dari"** dan
// `date_of_request_document` adalah **"Sampai"**; keduanya menyaring TANGGAL REGISTRASI
// dan tidak berhubungan dengan tanggal kejadian maupun tanggal terima dokumen. Lihat
// kepala paket.
const (
	FieldSegment               = "segmen"
	FieldBusinessScope         = "business_name"
	FieldDateOfLoss            = "date_of_loss"
	FieldDateOfRequestDocument = "date_of_request_document"

	// Dua isian wajib pada baris laporan — lihat ReportEntry.Validate.
	FieldClaimID    = "no_klaim"
	FieldContractNo = "contract_no"

	// FieldFile menandai berkas unggahan itu sendiri, bukan satu isian di dalamnya.
	FieldFile = "berkas"
)

// Violation adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Violation struct {
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus.
//
// Ia sengaja bukan daftar string: transport perlu tahu isian mana yang salah untuk
// menandainya di layar, dan informasi itu hilang bila pesannya dirangkai menjadi satu
// kalimat. Mengembalikan seluruhnya sekaligus meniru perilaku sistem lama yang
// menampilkan semua pesan bersamaan (`P-5`).
type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	messages := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		messages = append(messages, v.Field+": "+v.Message)
	}
	return "monitoringslinkojk: validasi gagal — " + strings.Join(messages, "; ")
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang terbungkus
// interface — supaya `if err != nil` di pemanggil berperilaku seperti yang terbaca.
func NewValidationError(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: violations}
}
