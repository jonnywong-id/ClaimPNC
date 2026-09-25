// Package usecase mengorkestrasi modul Report Klaim.
//
// Tugasnya empat, dan tidak lebih: memilih Repo milik portal yang sedang aktif, mencari
// laporan yang diminta di katalog, memeriksa penyaringnya, lalu mengalirkan barisnya ke
// pemanggil. Ia tidak tahu apa pun tentang HTTP maupun SQL.
package usecase

import (
	"context"
	"errors"
	"time"

	"claim-pnc/internal/reportklaim"
)

// Service adalah pintu masuk seluruh perkara Report Klaim.
type Service struct {
	repoSelector reportklaim.RepoSelector
	clock        reportklaim.Clock
	audit        reportklaim.AuditSink
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector reportklaim.RepoSelector

	// Clock menyediakan waktu. Wajib.
	//
	// Ia tidak diberi nilai bawaan jam sistem dengan sengaja: nilai bawaan yang diam-diam
	// benar di produksi adalah nilai bawaan yang diam-diam salah di pengujian.
	Clock reportklaim.Clock

	// Audit mencatat siapa mengunduh laporan apa. OPSIONAL.
	//
	// Ketiadaannya berarti tidak ada yang dicatat — dan itu keadaan yang harus disadari,
	// bukan keadaan yang boleh dibiarkan di produksi. `D-59` menjadikan jejak audit
	// satu-satunya kontrol pengimbang karena tidak ada pemisahan tugas, dan berkas dari
	// modul ini memuat data nasabah LINTAS CABANG.
	Audit reportklaim.AuditSink
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("reportklaim/usecase: RepoSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("reportklaim/usecase: Clock wajib diisi")
	}
	return &Service{
		repoSelector: o.RepoSelector,
		clock:        o.Clock,
		audit:        o.Audit,
	}, nil
}

// Now mengembalikan waktu acuan yang dipakai layanan ini.
func (s *Service) Now() time.Time { return s.clock.Now() }

// Catalog mengembalikan ke-28 panel beserta keadaannya.
//
// Ia TIDAK menyaring panel yang terhalang. Panel seperti itu tetap tampil di layar
// bertanda sebabnya — sama seperti butir menu yang belum punya layar tetap tampil.
// Menghilangkannya membuat pengguna melaporkan laporan yang "hilang", dan membuat
// kemajuan migrasi tidak terbaca dari layar.
func (s *Service) Catalog() []reportklaim.Report { return reportklaim.Catalog() }

// BusinessOptions mengembalikan isi autocomplete "Bisnis" milik satu portal.
func (s *Service) BusinessOptions(ctx context.Context, portalAlias string) ([]reportklaim.BusinessOption, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListBusinessOptions(ctx)
}

// ExportRequest adalah satu permintaan unduh berkas.
type ExportRequest struct {
	// PortalAlias menentukan basis data mana yang menjawab. Wajib.
	PortalAlias string

	// Caller adalah petugas yang menekan tombolnya — dipakai jejak audit.
	Caller reportklaim.Caller

	// Code adalah laporan yang diminta.
	Code reportklaim.Code

	// Action adalah tombol yang ditekan. Kosong pada panel bertombol tunggal.
	Action string

	// Filter adalah isian penyaring dari layar.
	//
	// Filter.FixedParam SENGAJA diabaikan di sini dan diisi ulang dari Action — nilai itu
	// milik tombol, bukan milik pengguna. Membiarkannya datang dari luar berarti
	// membiarkan pemanggil memilih antara komite yang MENYETUJUI dan yang MENOLAK.
	Filter reportklaim.Filter
}

// Export menjalankan satu laporan dan memanggil emit sekali per baris.
//
// Ia mengembalikan laporannya beserta jumlah baris yang benar-benar terkirim, supaya
// pemanggil dapat menyusun nama berkas dan mencatat jejak auditnya tanpa menghitung ulang.
//
// # Urutan pemeriksaan, dan kenapa begitu
//
// Katalog diperiksa LEBIH DULU, sebelum penyaring. Laporan yang tidak ada atau belum
// dapat dijalankan harus dijawab begitu — bukan dijawab "tanggal wajib diisi", yang
// membuat pengguna mengisi tanggal berkali-kali untuk laporan yang memang belum siap.
func (s *Service) Export(
	ctx context.Context,
	request ExportRequest,
	emit func(reportklaim.Row) error,
) (reportklaim.Report, int, error) {
	report, err := reportklaim.Lookup(request.Code)
	if err != nil {
		return reportklaim.Report{}, 0, err
	}

	action, known := report.Action(request.Action)
	if !known {
		return report, 0, reportklaim.ErrUnknownAction
	}

	filter := request.Filter
	filter.FixedParam = action.FixedParam

	if violation := report.Validate(filter); len(violation) > 0 {
		return report, 0, &reportklaim.ValidationError{Violation: violation}
	}

	repo, err := s.repoSelector(request.PortalAlias)
	if err != nil {
		return report, 0, err
	}

	rows := 0
	err = repo.Stream(ctx, report, filter, func(row reportklaim.Row) error {
		if err := emit(row); err != nil {
			return err
		}
		rows++
		return nil
	})
	if err != nil {
		return report, rows, err
	}

	// Dicatat SETELAH seluruh baris terkirim. Unduhan yang gagal di tengah bukan unduhan,
	// dan mencatatnya sebagai berhasil membuat jejak audit berbohong ke arah yang paling
	// merugikan.
	if s.audit != nil {
		s.audit.ReportDownloaded(ctx, request.Caller, request.PortalAlias, report, filter, rows)
	}
	return report, rows, nil
}
