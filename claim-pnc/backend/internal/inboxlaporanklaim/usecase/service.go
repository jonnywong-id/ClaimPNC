// Package usecase mengorkestrasi modul Inbox Laporan Klaim.
//
// Tugasnya empat, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// memeriksa penyaring yang diminta, memberlakukan batas data cabang, lalu memanggil
// Repo. Ia tidak tahu apa pun tentang HTTP maupun SQL.
package usecase

import (
	"errors"
	"time"

	"claim-pnc/internal/inboxlaporanklaim"
)

// Service adalah pintu masuk seluruh perkara Inbox Laporan Klaim.
type Service struct {
	repoSelector inboxlaporanklaim.RepoSelector
	clock        inboxlaporanklaim.Clock
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector inboxlaporanklaim.RepoSelector

	// Clock menyediakan waktu. Wajib.
	//
	// Ia tidak diberi nilai bawaan jam sistem dengan sengaja: nilai bawaan yang diam-diam
	// benar di produksi adalah nilai bawaan yang diam-diam salah di pengujian, dan umur
	// berkas di layar ini seluruhnya dihitung darinya.
	Clock inboxlaporanklaim.Clock
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxlaporanklaim/usecase: RepoSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("inboxlaporanklaim/usecase: Clock wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, clock: o.Clock}, nil
}

// Now mengembalikan waktu acuan yang dipakai layanan ini.
//
// Ia dibuka supaya lapisan transport dapat menghitung umur berkas dengan acuan YANG SAMA
// dengan yang dipakai saat membuat berkas — dua pemanggilan jam yang berbeda pada satu
// permintaan dapat jatuh di dua hari yang berbeda tepat di tengah malam.
func (s *Service) Now() time.Time { return s.clock.Now() }
