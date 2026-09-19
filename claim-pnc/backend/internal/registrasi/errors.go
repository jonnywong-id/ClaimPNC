package registrasi

import (
	"errors"
	"strings"
)

// Galat struktur alur. Ketiganya berarti definisi alur atau permintaan tidak konsisten
// — cacat pemrograman atau permintaan cacat, bukan kesalahan pengguna.
var (
	ErrUnknownStage = errors.New("registrasi: tahap tidak dikenal")
	ErrUnknownNode  = errors.New("registrasi: simpul alur tidak dikenal")
	ErrFlowLoop     = errors.New("registrasi: keputusan alur berputar tanpa ujung")
)

// Galat yang dikembalikan seam penyimpanan.
var (
	ErrClaimNotFound = errors.New("registrasi: klaim tidak ditemukan")
	ErrTaskNotFound  = errors.New("registrasi: tugas tidak ditemukan")
)

// Galat penugasan.
//
// ErrTaskAlreadyClaimed dibedakan dari ErrNotTaskOwner dengan sengaja: yang pertama
// berarti rekan kerja mendahului — pesan yang tepat adalah "sudah diambil <nama>";
// yang kedua berarti tugas memang bukan milik pemanggil sejak awal.
var (
	ErrTaskAlreadyClaimed = errors.New("registrasi: tugas sudah diambil pengguna lain")
	ErrNotTaskOwner       = errors.New("registrasi: tugas bukan milik pemanggil")
	ErrTaskAlreadyDone    = errors.New("registrasi: tugas sudah selesai")
	ErrInvalidAction      = errors.New("registrasi: tindakan tidak berlaku pada tahap ini")
	ErrStageMismatch      = errors.New("registrasi: klaim tidak sedang berada di tahap itu")
)

// ErrExchangeRateNotFound dikembalikan bila kurs mata uang pada tanggal kejadian belum
// diisi.
//
// `ADR-0015` menetapkan klaim DITOLAK, bukan dihitung dengan nilai bawaan. Memakai nilai
// bawaan berarti mencatat angka yang salah tanpa siapa pun tahu.
var ErrExchangeRateNotFound = errors.New("registrasi: kurs tanggal kejadian tidak ditemukan")

// ErrClaimNumberAlreadyIssued dikembalikan bila klaim yang sudah bernomor didaftarkan
// ulang. Nomor klaim tidak dapat ditarik kembali dan tidak diterbitkan dua kali.
var ErrClaimNumberAlreadyIssued = errors.New("registrasi: nomor klaim sudah pernah terbit")

// ViolationCode menamai satu aturan validasi.
//
// Klien membedakan pelanggaran lewat kode ini, bukan dengan mencocokkan teks pesan —
// aturan yang sama dengan yang berlaku pada galat HTTP.
type ViolationCode string

const (
	ViolationReportDateBeforeLoss    ViolationCode = "tanggal_lapor_sebelum_kejadian"
	ViolationReceivedBeforeReport    ViolationCode = "terima_dokumen_sebelum_lapor"
	ViolationLossDateInFuture        ViolationCode = "tanggal_kejadian_di_masa_depan"
	ViolationReportDateInFuture      ViolationCode = "tanggal_lapor_di_masa_depan"
	ViolationReceivedInFuture        ViolationCode = "terima_dokumen_di_masa_depan"
	ViolationLossOutsidePolicyPeriod ViolationCode = "tanggal_kejadian_di_luar_periode_polis"
	ViolationPolicyPeriodEnded       ViolationCode = "periode_polis_berakhir"
	ViolationReportedAfter7Days      ViolationCode = "lapor_lewat_7_hari"
	ViolationReceivedAfter90Days     ViolationCode = "terima_dokumen_lewat_90_hari"
	ViolationDeclarationPolicy       ViolationCode = "polis_deklarasi"
	ViolationDuplicateClaim          ViolationCode = "klaim_ganda"
	ViolationSLIKNumberEmpty         ViolationCode = "nomor_slik_kosong"
	ViolationCauseOfLossEmpty        ViolationCode = "penyebab_kerugian_kosong"
	ViolationItemWithoutCoverage     ViolationCode = "objek_tanpa_coverage"
	ViolationNoSpreading             ViolationCode = "tanpa_spreading"
	ViolationSpreadingTotalNot100    ViolationCode = "total_spreading_bukan_100"
	ViolationFacOfferIncomplete      ViolationCode = "fac_offer_tidak_lengkap"
	ViolationEstimateExceedsTSI      ViolationCode = "estimasi_melebihi_tsi"
	ViolationReporterStatusEmpty     ViolationCode = "status_pelapor_kosong"
)

// Violation adalah satu aturan yang dilanggar, beserta field yang menyebabkannya.
type Violation struct {
	Code ViolationCode

	// Field menyebut isian yang harus diperbaiki pengguna. Ia memakai nama field DTO
	// supaya layar dapat menandai kolom yang tepat tanpa kamus penerjemah.
	Field string

	Message string
}

// ValidationError mengumpulkan seluruh aturan yang dilanggar sebuah klaim.
//
// # Kenapa seluruhnya, bukan yang pertama saja
//
// Di sistem lama tiap langkah validasi diikuti transisi "exit activity", sehingga
// petugas melihat SATU pesan, memperbaikinya, menyimpan lagi, lalu melihat pesan
// berikutnya. Pada layar dengan puluhan isian itu berarti belasan kali simpan.
//
// Di sini seluruh pelanggaran dikumpulkan sekaligus. Urutannya sengaja SAMA dengan
// urutan langkah pada `Activity/InputRegister_act-Act.xml`, sehingga `Pertama()` selalu
// menghasilkan pesan yang sama dengan yang ditampilkan Pega — kesetaraan gerbang 1 tetap
// dapat diuji meskipun tampilannya berbeda.
type ValidationError struct {
	Violation []Violation
}

func (g *ValidationError) Error() string {
	if len(g.Violation) == 0 {
		return "registrasi: validasi gagal tanpa rincian"
	}
	messages := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		messages = append(messages, string(p.Code))
	}
	return "registrasi: validasi gagal: " + strings.Join(messages, ", ")
}

// First mengembalikan pelanggaran yang di sistem lama akan menghentikan penyimpanan.
func (g *ValidationError) First() (Violation, bool) {
	if len(g.Violation) == 0 {
		return Violation{}, false
	}
	return g.Violation[0], true
}

// Has menyatakan sebuah kode pelanggaran ada di dalam kumpulan.
func (g *ValidationError) Has(code ViolationCode) bool {
	for _, p := range g.Violation {
		if p.Code == code {
			return true
		}
	}
	return false
}
