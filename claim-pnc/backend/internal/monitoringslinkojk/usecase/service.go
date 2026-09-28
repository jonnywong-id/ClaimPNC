// Package usecase mengorkestrasi modul Monitoring SLINK OJK.
//
// Empat operasi, seluruhnya MEMBACA:
//
//	Columns   katalog kolom satu segmen — dipakai grid dan berkas CSV
//	Search    satu halaman grid           — tombol "Cari Data"
//	Export    seluruh baris, dialirkan    — tombol "Export Data"
//	Template  berkas contoh unggahan      — tombol "Format File"
//
// # Kenapa tidak ada operasi yang menulis
//
// Tiga tombol di layar lama memang menulis, dan ketiganya TIDAK dibangun di sini karena
// artefaknya tidak ada di export — bukan karena lingkupnya dipersempit. Rinciannya
// disebut satu per satu di monitoringslinkojkhttp.Mount, beserta artefak yang hilang.
//
// # Di mana validasi terjadi, dan kenapa di sini
//
// Rentang tanggal terbalik ditolak di lapisan ini, bukan di transport dan bukan di SQL.
// Di transport ia akan terlewat oleh pemanggil lain — ekspor, uji, perkakas; di SQL ia
// tidak menghasilkan galat sama sekali melainkan **nol baris yang tampak wajar**, dan
// pada layar pemantauan laporan regulator nol baris terbaca sebagai "tidak ada yang perlu
// dilaporkan".
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/monitoringslinkojk"
)

// Service melayani modul Monitoring SLINK OJK.
type Service struct {
	repoSelector monitoringslinkojk.RepoSelector

	// sender mengirim data debitur ke sistem SLIK. Boleh nil.
	//
	// Nil berarti layanannya belum dikonfigurasi, dan tombol "SLIK OJK" menolak dengan
	// sebab yang terbaca — bukan diam-diam berhasil. Kontrak layanannya
	// (`Rest_SendDataClientBasedDebitur`) memang tidak ada di export (`R-16`), sehingga
	// nil adalah keadaan yang WAJAR hari ini, bukan cacat perakitan.
	sender monitoringslinkojk.Sender
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector monitoringslinkojk.RepoSelector

	// Sender opsional; lihat Service.sender.
	Sender monitoringslinkojk.Sender
}

// NewService membentuk layanan modul Monitoring SLINK OJK.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("monitoringslinkojk/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, sender: o.Sender}, nil
}

// repoFor memilih penyimpanan milik satu portal.
//
// Galat pemilihan portal diteruskan APA ADANYA, tidak dibungkus galat modul ini:
// transport mengenali `portal.ErrNotReady` untuk menjawab dengan kode yang sudah dikenal
// frontend, dan membungkusnya akan memutus pengenalan itu.
func (s *Service) repoFor(portalAlias string) (monitoringslinkojk.Repo, error) {
	return s.repoSelector(portalAlias)
}

// Request adalah satu permintaan baca, lengkap dengan konteksnya.
//
// Ketiganya dibawa bersama karena ketiganya SELALU dibutuhkan bersama, dan memisahkannya
// menjadi tiga parameter membuat satu pemanggil dapat lupa membawa salah satunya —
// paling mungkin portal, yang justru paling mahal bila terlewat (`R-20`).
type Request struct {
	PortalAlias string
	Caller      monitoringslinkojk.Caller
	Segment     monitoringslinkojk.Segment
	Filter      monitoringslinkojk.Filter
}

// SegmentInfo adalah keterangan satu segmen beserta katalog kolomnya.
type SegmentInfo struct {
	Segment monitoringslinkojk.Segment
	Label   string
	Columns []monitoringslinkojk.Column

	// AvailableCount adalah banyaknya kolom yang benar-benar terisi.
	//
	// Ia dikirim ke layar supaya keterangan "30 dari 38 kolom belum punya sumber" dapat
	// ditampilkan SEKALI di atas tabel, alih-alih memaksa pengguna menyimpulkannya dari
	// 38 sel kosong.
	AvailableCount int
}

// Describe mengembalikan katalog kolom satu segmen.
//
// Tidak menyentuh basis data dan tidak menuntut portal: katalog kolom adalah pengetahuan
// modul, bukan data entitas. Memaksanya lewat portal hanya akan membuat layar gagal
// menggambar kepala tabel ketika basis datanya sedang tidak siap — padahal justru saat
// itulah pengguna perlu melihat bahwa layarnya utuh.
func (s *Service) Describe(segment monitoringslinkojk.Segment) (SegmentInfo, error) {
	if !segment.Valid() {
		return SegmentInfo{}, monitoringslinkojk.ErrUnknownSegment
	}
	columns := monitoringslinkojk.Columns(segment)
	return SegmentInfo{
		Segment:        segment,
		Label:          segment.Label(),
		Columns:        columns,
		AvailableCount: len(monitoringslinkojk.AvailableColumns(segment)),
	}, nil
}

// Segments mengembalikan katalog KEDUA segmen — isi dropdown "Pilih Segmen".
func (s *Service) Segments() []SegmentInfo {
	all := []monitoringslinkojk.Segment{
		monitoringslinkojk.SegmentD01,
		monitoringslinkojk.SegmentF06,
	}
	out := make([]SegmentInfo, 0, len(all))
	for _, segment := range all {
		// Galatnya diabaikan dengan sengaja: kedua segmen di atas konstanta, dan
		// Describe hanya menolak segmen yang tidak dikenal.
		info, _ := s.Describe(segment)
		out = append(out, info)
	}
	return out
}

// Search mengembalikan satu halaman grid — tombol "Cari Data".
func (s *Service) Search(ctx context.Context, request Request) (monitoringslinkojk.Page, error) {
	filter, err := s.prepare(request)
	if err != nil {
		return monitoringslinkojk.Page{}, err
	}

	repo, err := s.repoFor(request.PortalAlias)
	if err != nil {
		return monitoringslinkojk.Page{}, err
	}

	page, err := repo.Search(ctx, request.Segment, filter)
	if err != nil {
		return monitoringslinkojk.Page{}, fmt.Errorf(
			"monitoringslinkojk/usecase: pencarian segmen %s: %w", request.Segment, err)
	}
	return page, nil
}

// Export mengalirkan SELURUH baris yang cocok — tombol "Export Data".
//
// Barisnya diserahkan satu per satu lewat `emit`, tidak dikumpulkan lebih dulu: berkas
// ekspor berjalan atas data historis puluhan juta baris (`D-10`), dan menampungnya di
// memori membuat pemakaian memori tumbuh sebanding dengan hasil
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
//
// Paginasi pada Filter DIABAIKAN di sini — ekspor tidak mengenal halaman. Itu ditegaskan
// dengan mengosongkannya, bukan diserahkan pada kedisiplinan pemanggil.
func (s *Service) Export(
	ctx context.Context,
	request Request,
	emit func(monitoringslinkojk.Row) error,
) error {
	if emit == nil {
		return errors.New("monitoringslinkojk/usecase: penerima baris wajib diisi")
	}

	filter, err := s.prepare(request)
	if err != nil {
		return err
	}
	filter.Page = 0
	filter.Size = 0

	repo, err := s.repoFor(request.PortalAlias)
	if err != nil {
		return err
	}

	if err := repo.Stream(ctx, request.Segment, filter, emit); err != nil {
		return fmt.Errorf(
			"monitoringslinkojk/usecase: ekspor segmen %s: %w", request.Segment, err)
	}
	return nil
}

// prepare memeriksa satu permintaan dan mengembalikan penyaring yang sudah rapi.
//
// Seluruh pelanggaran dikumpulkan sekaligus, tidak berhenti pada yang pertama (`P-5`).
func (s *Service) prepare(request Request) (monitoringslinkojk.Filter, error) {
	if strings.TrimSpace(request.Caller.Login) == "" {
		return monitoringslinkojk.Filter{}, monitoringslinkojk.ErrCallerUnknown
	}
	if !request.Segment.Valid() {
		return monitoringslinkojk.Filter{}, monitoringslinkojk.ErrUnknownSegment
	}

	filter := request.Filter.Normalize()

	violations := make([]monitoringslinkojk.Violation, 0, 2)

	if !filter.BusinessScope.Valid() {
		violations = append(violations, monitoringslinkojk.Violation{
			Field:   monitoringslinkojk.FieldBusinessScope,
			Message: "Business Name harus AS. KREDIT atau SURETY BOND.",
		})
	}

	if filter.DateRangeReversed() {
		// Ditempelkan pada isian "Dari", bukan pada keduanya: pengguna yang melihat dua
		// isian bertanda merah tidak tahu mana yang harus diubah.
		violations = append(violations, monitoringslinkojk.Violation{
			Field:   monitoringslinkojk.FieldDateOfLoss,
			Message: "Tanggal \"Dari\" tidak boleh melewati tanggal \"Sampai\".",
		})
	}

	if err := monitoringslinkojk.NewValidationError(violations); err != nil {
		return monitoringslinkojk.Filter{}, err
	}

	// GenerateType TIDAK ikut menyaring. Lihat Filter.GenerateType: isiannya ada di layar
	// lama tetapi tidak satu pun kueri membacanya, dan daftar pilihannya hilang dari
	// export (`R-16`). Ia dibiarkan lewat apa adanya supaya layar tetap dapat
	// mengirimkannya, dan supaya penyaringnya dapat dipasang di SATU tempat begitu
	// aturannya diterima.

	return filter, nil
}
