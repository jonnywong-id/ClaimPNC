package sqlstore

import (
	"database/sql"

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/platform/branchlookup"
)

// BranchResolver memenuhi seam inboxlaporanklaim.BranchResolver dengan SQL.
//
// Ia terpisah dari Repo dengan sengaja: Repo membaca berkas laporan di basis data satu
// entitas, sedangkan penerjemah ini membaca HRD dan master pengguna asuransi lewat DB
// Link. Keduanya dapat gagal sendiri-sendiri, dan menyatukannya membuat kegagalan yang
// satu tampak seperti kegagalan yang lain.
type BranchResolver = branchlookup.Resolver

// branchTimeout membatasi lama penerjemahan cabang.
//
// # Kenapa ada batas waktu di sini, dan tidak di kueri lain modul ini
//
// Kueri lain membaca basis data entitas sendiri. Yang ini menembus **DB Link** ke basis
// data lain — dan `10-API-STRATEGY.md` §8.2 menetapkan batas waktu WAJIB pada setiap
// pemanggilan keluar. DB Link adalah pemanggilan keluar yang menyamar sebagai kueri
// biasa: ia melewati jaringan, dan sambungan yang mati dapat menggantung sampai batas
// waktu TCP, bukan menjawab galat.
//
// # Kenapa itu lebih buruk daripada galat
//
// Permintaan yang menggantung **tidak terbedakan dari tombol yang tidak bekerja**.
// Tombolnya tetap berbunyi "Membuat…", tidak ada pesan apa pun, dan pengguna menyerah
// sebelum jawabannya tiba. Itu kelas kegagalan yang sama dengan daftar kosong yang
// membuat modul ini harus diperbaiki pada mulanya: gagal tanpa mengatakan apa-apa.
//
// Lima detik: cukup longgar untuk DB Link yang sehat pada jaringan kantor, cukup pendek
// untuk membuat kegagalannya terasa sebagai kegagalan, bukan sebagai kelambatan.
const branchTimeout = branchlookup.Timeout

// NewBranchResolver membentuk penerjemah; db wajib sudah terhubung.
func NewBranchResolver(db *sql.DB) *BranchResolver {
	return branchlookup.New(db, getQuery("branch_of_login"), "inboxlaporanklaim/sqlstore")
}

var _ inboxlaporanklaim.BranchResolver = (*BranchResolver)(nil)
