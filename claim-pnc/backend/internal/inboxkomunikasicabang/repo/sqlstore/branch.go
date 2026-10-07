package sqlstore

import (
	"database/sql"

	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/platform/branchlookup"
)

// BranchResolver memenuhi seam inboxkomunikasicabang.BranchResolver dengan SQL.
//
// Ia terpisah dari Repo dengan sengaja: Repo membaca percakapan di basis data satu entitas,
// sedangkan penerjemah ini membaca HRD dan master pengguna asuransi lewat DB Link. Keduanya
// dapat gagal sendiri-sendiri, dan menyatukannya membuat kegagalan yang satu tampak seperti
// kegagalan yang lain.
//
// Di layar ini pembedaan itu menentukan APA YANG DILIHAT petugas: sumber cabang yang mati
// menutup layarnya, sementara petugas yang sekadar tidak terdaftar tetap dilayani sebagai
// kantor pusat (`P-5`).
type BranchResolver = branchlookup.Resolver

// branchTimeout membatasi lama penerjemahan cabang.
//
// # Kenapa ada batas waktu di sini, dan tidak di kueri lain modul ini
//
// Kueri lain membaca basis data entitas sendiri. Yang ini menembus DB LINK ke basis data
// lain — dan `10-API-STRATEGY.md` §8.2 menetapkan batas waktu WAJIB pada setiap pemanggilan
// keluar. DB Link adalah pemanggilan keluar yang menyamar sebagai kueri biasa: ia melewati
// jaringan, dan sambungan yang mati dapat menggantung sampai batas waktu TCP alih-alih
// menjawab galat.
//
// # Kenapa menggantung lebih buruk daripada galat
//
// Permintaan yang menggantung tidak terbedakan dari layar yang tidak bekerja. Tidak ada
// pesan apa pun, dan pengguna menyerah sebelum jawabannya tiba.
//
// Lima detik: cukup longgar untuk DB Link yang sehat pada jaringan kantor, cukup pendek
// untuk membuat kegagalannya terasa sebagai kegagalan, bukan sebagai kelambatan. Angkanya
// sama dengan modul Inbox Laporan Klaim, dan kesamaannya disengaja — keduanya menembus
// sambungan yang sama.
const branchTimeout = branchlookup.Timeout

// NewBranchResolver membentuk penerjemah; db wajib sudah terhubung.
func NewBranchResolver(db *sql.DB) *BranchResolver {
	return branchlookup.New(db, getQuery("branch_of_login"), "inboxkomunikasicabang/sqlstore")
}

var _ inboxkomunikasicabang.BranchResolver = (*BranchResolver)(nil)
