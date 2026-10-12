package registrasi

import "context"

// UnitOfWork adalah seam ke batas transaksi.
//
// # Kenapa ia seam, bukan detail penyimpanan
//
// `TKT-B02-001` menuntut penyimpanan klaim gagal di langkah mana pun **tidak
// meninggalkan satu baris pun**. Satu penyimpanan registrasi menyentuh empat hal
// sekaligus: klaim beserta pohon objek–coverage–spreading, tugas yang ditutup, tugas
// berikutnya yang lahir, dan jejak audit. Keempatnya harus jadi atau tidak sama sekali.
//
// Batas itu adalah keputusan ATURAN BISNIS — usecase yang tahu di mana satu pekerjaan
// utuh dimulai dan berakhir, bukan repo. Karena itu ia dinyatakan di sini, dan siapa pun
// yang membaca usecase melihat batasnya tanpa membuka lapisan penyimpanan.
//
// Pengisinya: `repo/sqlstore` membuka transaksi basis data; `repo/memori` mengunci
// seluruh penyimpanan. Keduanya memenuhi janji yang sama.
type UnitOfWork interface {
	// Run menjalankan kerja di dalam satu batas transaksi.
	//
	// Context yang diterima kerja adalah context yang SUDAH membawa transaksi; repo
	// yang dipanggil di dalamnya wajib memakai context itu, bukan context luar.
	// Mengembalikan galat membatalkan seluruhnya.
	Run(ctx context.Context, work func(context.Context) error) error
}
