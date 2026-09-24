// Package usecase mengorkestrasi modul Inbox Receive TKA.
//
// Ia yang mengetahui urutan langkah; bentuk datanya ada di paket domain, dan cara membacanya
// dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
//
// # Tiga berkas, dan pembagiannya mengikuti apa yang dikerjakan
//
//	service.go  perakitan dan pemilihan portal   ← berkas ini
//	browse.go   membaca daftar
//	submit.go   mengisi tanggal kelengkapan dokumen
//
// Modul Inbox Investigator cukup dengan satu berkas karena ia hanya membaca. Modul ini
// menulis, dan jalur tulisnya membawa urutan langkah yang jauh lebih panjang daripada jalur
// bacanya — menyatukan keduanya akan menyembunyikan yang panjang di balik yang pendek.
package usecase

import (
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxreceivetka"
)

// Service adalah pintu masuk seluruh perkara Inbox Receive TKA.
type Service struct {
	repoSelector inboxreceivetka.RepoSelector
	notifier     inboxreceivetka.Notifier
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
//
// TANPA Clock, dan ketiadaannya disengaja. Satu-satunya tanggal yang ditulis modul ini
// datang dari pengguna — ia MENGETIKKAN tanggal kelengkapan dokumen, dan sistem lama pun
// memakai `Param.Tanggal` apa adanya tanpa menyentuh jam dinding. Tidak ada nilai di modul
// ini yang bergantung pada "sekarang", sehingga seam Clock tidak punya apa pun untuk
// divariasikan.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector inboxreceivetka.RepoSelector

	// Notifier mengirim pemberitahuan kelengkapan dokumen.
	//
	// BOLEH nil, dan itu keadaan yang wajar — bukan rakitan yang setengah jadi. Penerima
	// surel datang dari konfigurasi `SMTP_*`, dan lingkungan pengembangan sering tidak
	// memilikinya. Nil berarti pemberitahuan dilewati dengan catatan, bukan pekerjaan
	// ditolak; lihat Complete.
	Notifier inboxreceivetka.Notifier

	// Logger mencatat peristiwa bisnis dan kegagalan pemberitahuan.
	//
	// BOLEH nil; bila nil, pencatatannya dibuang. Lihat NewService.
	Logger *slog.Logger
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang: rakitan
// yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
//
// # Logger yang nil DIGANTI di sini, bukan diperiksa di setiap tempat pemakaian
//
// `logging.From` mengembalikan logger yang diterimanya apa adanya ketika permintaan tidak
// membawa ID — termasuk ketika yang diterimanya nil — sehingga pemanggilan berikutnya
// menembus pointer nil. Ia paket platform yang dipakai modul-modul yang sudah dinyatakan
// selesai, jadi yang disesuaikan adalah modul ini.
//
// Penggantinya dipasang SEKALI di sini, bukan sebagai pemeriksaan nil di setiap tempat
// pencatatan. Pemeriksaan yang tersebar hanya perlu terlewat satu kali untuk membuat jalur
// yang jarang dilalui — misalnya kegagalan kirim surel — menjatuhkan seluruh permintaan.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxreceivetka/usecase: RepoSelector wajib diisi")
	}

	logger := o.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return &Service{
		repoSelector: o.RepoSelector,
		notifier:     o.Notifier,
		logger:       logger,
	}, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum kueri dijalankan.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("inboxreceivetka/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}

// NotificationEnabled menyatakan pemberitahuan dapat dikirim.
//
// Dipakai transport untuk menerangkan kepada layar bahwa penyimpanan berhasil sementara
// pemberitahuannya memang tidak pernah dicoba — berbeda dari dicoba lalu gagal.
func (s *Service) NotificationEnabled() bool { return s.notifier != nil }
